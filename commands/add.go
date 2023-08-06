package commands

import (
	"docker-container-healthchecker/appjson"
	"fmt"
	"os"

	"github.com/Jeffail/gabs/v2"
	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

type AddCommand struct {
	command.Meta

	appJSONFile string
	checkType   string
	ifEmpty     bool
	inPlace     bool
	prettyPrint bool
	uptime      int
}

func (c *AddCommand) Name() string {
	return "add"
}

func (c *AddCommand) Synopsis() string {
	return "Adds a healthcheck to a process type"
}

func (c *AddCommand) Help() string {
	return command.CommandHelp(c)
}

func (c *AddCommand) Examples() map[string]string {
	appName := os.Getenv("CLI_APP_NAME")
	return map[string]string{
		"Add the default healthcheck to the web process type": fmt.Sprintf("%s %s web --if-empty", appName, c.Name()),
	}
}

func (c *AddCommand) Arguments() []command.Argument {
	args := []command.Argument{}
	args = append(args, command.Argument{
		Name:        "process-type",
		Description: "process type to add a check to",
		Optional:    true,
		Type:        command.ArgumentString,
	})
	return args
}

func (c *AddCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *AddCommand) ParsedArguments(args []string) (map[string]command.Argument, error) {
	return command.ParseArguments(args, c.Arguments())
}

func (c *AddCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.Name(), command.FlagSetClient)
	f.BoolVar(&c.prettyPrint, "pretty", false, "pretty print json output")
	f.BoolVar(&c.ifEmpty, "if-empty", false, "only add if there are no healthchecks for the process")
	f.BoolVar(&c.inPlace, "in-place", false, "modify any app.json file in place")
	f.StringVar(&c.appJSONFile, "app-json", "", "full path to app.json file to update")
	f.StringVar(&c.checkType, "type", "startup", "check to interpret")
	f.IntVar(&c.uptime, "uptime", 1, "amount of time the container should be running for at minimum")
	return f
}

func (c *AddCommand) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(command.FlagSetClient),
		complete.Flags{
			"--app-json": complete.PredictAnything,
			"--if-empty": complete.PredictNothing,
			"--in-place": complete.PredictNothing,
			"--pretty":   complete.PredictNothing,
			"--type":     complete.PredictSet("liveness", "readiness", "startup"),
			"--uptime":   complete.PredictAnything,
		},
	)
}

func (c *AddCommand) Run(args []string) int {
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

	contents := []byte("{}")
	if c.appJSONFile != "" {
		if _, err := os.Stat(c.appJSONFile); err != nil {
			c.Ui.Error(err.Error())
			return 1
		}

		contents, err = os.ReadFile(c.appJSONFile)
		if err != nil {
			c.Ui.Error(err.Error())
			return 1
		}
	}

	parsed, err := gabs.ParseJSON(contents)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	processType := arguments["process-type"].StringValue()
	if processType == "" {
		processType = "web"
	}
	path := fmt.Sprintf("healthchecks.%s", processType)
	exists := parsed.ExistsP(path)
	length := len(parsed.S(path).Children())
	if c.ifEmpty && exists && length > 0 {
		return 0
	}

	healthcheck := appjson.Healthcheck{
		Name:   "default",
		Type:   c.checkType,
		Uptime: c.uptime,
	}

	if exists && length > 0 {
		parsed.ArrayAppend(healthcheck, path)
	} else {
		parsed.SetP([]appjson.Healthcheck{healthcheck}, path)
	}

	var b []byte
	if c.prettyPrint {
		b = parsed.BytesIndent("", "  ")
	} else {
		b = parsed.Bytes()
	}

	if c.inPlace {
		if err := os.WriteFile(c.appJSONFile, b, 0644); err != nil {
			c.Ui.Error(err.Error())
			return 1
		}

		return 0
	}

	fmt.Println(string(b))

	return 0
}