package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"

	"docker-container-healthchecker/appjson"
)

type CheckCommand struct {
	command.Meta

	appJSONFile string
	headers     []string
	checkType   string
	networkName string
	port        int
	processType string
}

func (c *CheckCommand) Name() string {
	return "check"
}

func (c *CheckCommand) Synopsis() string {
	return "Checks the health status of one or more containers"
}

func (c *CheckCommand) Help() string {
	return command.CommandHelp(c)
}

func (c *CheckCommand) Examples() map[string]string {
	appName := os.Getenv("CLI_APP_NAME")
	return map[string]string{
		"Check the web process": fmt.Sprintf("%s %s dokku.web.1", appName, c.Name()),
	}
}

func (c *CheckCommand) Arguments() []command.Argument {
	args := []command.Argument{}
	args = append(args, command.Argument{
		Name:        "container-id",
		Description: "ID or Name of container to check",
		Optional:    false,
		Type:        command.ArgumentString,
	})
	return args
}

func (c *CheckCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *CheckCommand) ParsedArguments(args []string) (map[string]command.Argument, error) {
	return command.ParseArguments(args, c.Arguments())
}

func (c *CheckCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.Name(), command.FlagSetClient)
	f.IntVar(&c.port, "port", 5000, "container port to check")
	f.StringSliceVar(&c.headers, "header", []string{}, "one or more headers in 'curl -H' format to specify for path requests")
	f.StringVar(&c.appJSONFile, "app-json", "app.json", "full path to app.json file")
	f.StringVar(&c.checkType, "type", "startup", "check to interpret")
	f.StringVar(&c.networkName, "network", "bridge", "container network to use for http 'path' checks")
	f.StringVar(&c.processType, "process-type", "web", "process type to check")
	return f
}

func (c *CheckCommand) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(command.FlagSetClient),
		complete.Flags{
			"--app-json":     complete.PredictAnything,
			"--header":       complete.PredictAnything,
			"--network":      complete.PredictAnything,
			"--port":         complete.PredictAnything,
			"--process-type": complete.PredictAnything,
			"--type":         complete.PredictSet("liveness", "readiness", "startup"),
		},
	)
}

func (c *CheckCommand) Run(args []string) int {
	flags := c.FlagSet()
	flags.Usage = func() { c.Ui.Output(c.Help()) }
	if err := flags.Parse(args); err != nil {
		c.Ui.Error(err.Error())
		c.Ui.Error(command.CommandErrorText(c))
		return 1
	}

	arguments, err := c.ParsedArguments(flags.Args())
	if err != nil {
		c.Ui.Error(err.Error())
		c.Ui.Error(command.CommandErrorText(c))
		return 1
	}

	logger, ok := c.Ui.(*command.ZerologUi)
	if !ok {
		c.Ui.Error("Unable to fetch logger from cli")
		return 1
	}

	b, err := os.ReadFile(c.appJSONFile)
	if err != nil {
		logger.Error(err.Error())
		return 1
	}

	var appJSON appjson.AppJSON
	if err := json.Unmarshal(b, &appJSON); err != nil {
		logger.Error(err.Error())
		return 1
	}

	containerIDorName := arguments["container-id"].StringValue()
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		logger.Error(err.Error())
		return 1
	}

	container, err := cli.ContainerInspect(context.Background(), containerIDorName)
	if err != nil {
		logger.Error(err.Error())
		return 1
	}

	if !container.State.Running {
		logger.Error(fmt.Sprintf("Container state: %s", container.State.Status))
		return 1
	}

	var healthchecks []appjson.Healthcheck

	// collect the healthchecks
	for processType, data := range appJSON.Healthchecks {
		if processType != c.processType {
			continue
		}

		for _, healthcheck := range data {
			if healthcheck.Type == "" {
				logger.Error(fmt.Sprintf("Missing type field on healthcheck name='%s'", healthcheck.GetName()))
			}

			if healthcheck.Type != c.checkType {
				continue
			}

			healthchecks = append(healthchecks, healthcheck)
		}
	}

	if len(healthchecks) == 0 {
		healthchecks = append(healthchecks, appjson.Healthcheck{
			Name:   "autogenerated",
			Type:   c.checkType,
			Uptime: 10,
		})
	}

	logger.LogHeader2(fmt.Sprintf("Executing %d healthchecks", len(healthchecks)))
	var wg sync.WaitGroup
	responseChan := make(chan HealthcheckResponse)
	for _, healthcheck := range healthchecks {
		wg.Add(1)
		go func(h appjson.Healthcheck) {
			defer wg.Done()
			responseChan <- c.processHealthcheck(h, container, logger)
		}(healthcheck)
	}

	go func() {
		wg.Wait()
		close(responseChan)
	}()

	errorCount := 0
	for resp := range responseChan {
		errorCount += len(resp.Errors)
		if len(resp.Errors) > 0 {
			err := resp.Errors[len(resp.Errors)-1]
			logger.Error(fmt.Sprintf("Failure in name='%s': %s", resp.HealthcheckName, err.Error()))
		} else {
			logger.Info(fmt.Sprintf("Healthcheck succeeded name='%s'", resp.HealthcheckName))
		}
	}

	return errorCount
}

type HealthcheckResponse struct {
	HealthcheckName string
	Errors          []error
}

func (c *CheckCommand) processHealthcheck(healthcheck appjson.Healthcheck, container types.ContainerJSON, logger *command.ZerologUi) HealthcheckResponse {
	tt, err := time.Parse(time.RFC3339, container.State.StartedAt)
	if err != nil {
		return HealthcheckResponse{
			HealthcheckName: healthcheck.GetName(),
			Errors:          []error{err},
		}
	}

	delay := 0
	if time.Since(tt).Seconds() < float64(healthcheck.GetInitialDelay()) {
		delay = int(time.Since(tt).Seconds() - float64(healthcheck.GetInitialDelay()))
	}

	switch healthcheck.GetCheckType() {
	case "command":
		logger.Info(fmt.Sprintf("Running healthcheck name='%s' attempts=%d command='%s' timeout=%d type='command' wait=%d", healthcheck.GetName(), healthcheck.GetAttempts(), healthcheck.Command, healthcheck.GetTimeout(), healthcheck.GetWait()))
	case "path":
		logger.Info(fmt.Sprintf("Running healthcheck name='%s' delay=%d path='%s' retries=%d timeout=%d type='path'", healthcheck.GetName(), healthcheck.GetInitialDelay(), healthcheck.GetPath(), healthcheck.GetRetries(), healthcheck.GetTimeout()))
	case "uptime":
		logger.Info(fmt.Sprintf("Running healthcheck name='%s' type='uptime' uptime=%d", healthcheck.GetName(), healthcheck.Uptime))
	}

	if delay > 0 {
		time.Sleep(time.Duration(delay) * time.Second)
	}

	ctx := appjson.HealthcheckContext{
		Headers: c.headers,
		Network: c.networkName,
		Port:    c.port,
	}

	b, errs := healthcheck.Execute(container, ctx)
	if len(errs) > 0 {
		if len(b) > 0 {
			logger.Error(fmt.Sprintf("Error for healthcheck name='%s', output: %s", healthcheck.GetName(), strings.TrimSpace(string(b))))
		}
		if err := healthcheck.HandleFailure(errs); err != nil {
			logger.Error(fmt.Sprintf("Error in HandleFailure: %s", err))
		}

		return HealthcheckResponse{
			HealthcheckName: healthcheck.GetName(),
			Errors:          errs,
		}
	}

	return HealthcheckResponse{
		HealthcheckName: healthcheck.GetName(),
		Errors:          []error{},
	}
}
