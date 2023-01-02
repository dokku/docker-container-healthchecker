package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	checkType   string
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
	f.StringVar(&c.appJSONFile, "app-json", "app.json", "full path to app.json file")
	f.StringVar(&c.checkType, "check-type", "startup", "check to interpret")
	f.StringVar(&c.processType, "process-type", "web", "process type to check")
	return f
}

func (c *CheckCommand) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(command.FlagSetClient),
		complete.Flags{
			"--app-json":     complete.PredictAnything,
			"--check-type":   complete.PredictSet("liveness", "readiness", "startup"),
			"--port":         complete.PredictAnything,
			"--process-type": complete.PredictAnything,
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

	logger.LogHeader2(fmt.Sprintf("Reading app.json file from %s", c.appJSONFile))
	b, err := ioutil.ReadFile(c.appJSONFile)
	if err != nil {
		logger.Error(err.Error())
		return 1
	}

	logger.Info("Parsing app.json data")
	var appJSON appjson.AppJSON
	if err := json.Unmarshal(b, &appJSON); err != nil {
		logger.Error(err.Error())
		return 1
	}

	containerIDorName := arguments["container-id"].StringValue()
	logger.Info(fmt.Sprintf("Fetching container %s", containerIDorName))
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
			if healthcheck.Type != c.checkType {
				continue
			}

			healthchecks = append(healthchecks, healthcheck)
		}
	}

	var wg sync.WaitGroup
	wgDone := make(chan bool)
	errChannel := make(chan ErrContainer)
	for _, healthcheck := range healthchecks {
		wg.Add(1)
		healthcheck := healthcheck
		go func() {
			defer wg.Done()
			processHealthcheck(healthcheck, container, c.port, logger, errChannel)
		}()
	}

	go func() {
		wg.Wait()
		close(wgDone)
	}()

	hasErrors := false
	select {
	case <-wgDone:
		break
	case container := <-errChannel:
		close(errChannel)
		if len(container.Errors) > 0 {
			hasErrors = true
			err := container.Errors[len(container.Errors)-1]
			logger.Error(fmt.Sprintf("Failure in name='%s': %s", container.HealthcheckName, err.Error()))
		}
	}

	if hasErrors {
		return 1
	}

	return 0
}

type ErrContainer struct {
	HealthcheckName string
	Errors          []error
}

func processHealthcheck(healthcheck appjson.Healthcheck, container types.ContainerJSON, containerPort int, logger *command.ZerologUi, errChannel chan<- ErrContainer) {
	tt, err := time.Parse(time.RFC3339, container.State.StartedAt)
	if err != nil {
		errChannel <- ErrContainer{
			HealthcheckName: healthcheck.GetName(),
			Errors:          []error{err},
		}
		return
	}

	delay := 0
	if time.Since(tt).Seconds() < float64(healthcheck.GetInitialDelay()) {
		delay = int(time.Since(tt).Seconds() - float64(healthcheck.GetInitialDelay()))
	}

	logger.Info(fmt.Sprintf("Running healthcheck name='%s' attempts=%d delay=%d wait=%d", healthcheck.GetName(), healthcheck.Attempts, healthcheck.GetInitialDelay(), healthcheck.GetWait()))
	if delay > 0 {
		time.Sleep(time.Duration(delay) * time.Second)
	}

	b, errs := healthcheck.Execute(container, containerPort)
	if errs != nil {
		if len(b) > 0 {
			logger.Error(fmt.Sprintf("Error for healthcheck name='%s', output: %s", healthcheck.GetName(), strings.TrimSpace(string(b))))
		}

		errChannel <- ErrContainer{
			HealthcheckName: healthcheck.GetName(),
			Errors:          errs,
		}
		return
	}
	errChannel <- ErrContainer{
		HealthcheckName: healthcheck.GetName(),
		Errors:          []error{},
	}
}
