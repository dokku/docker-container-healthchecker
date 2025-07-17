package commands

import (
	"fmt"
	"os"

	"github.com/Jeffail/gabs/v2"
	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

type ExistsCommand struct {
	command.Meta

	appJSONFile string
}

func (c *ExistsCommand) Name() string {
	return "exists"
}

func (c *ExistsCommand) Synopsis() string {
	return "Checks if a process type has any healthchecks defined"
}

func (c *ExistsCommand) Help() string {
	return command.CommandHelp(c)
}

func (c *ExistsCommand) Examples() map[string]string {
	appName := os.Getenv("CLI_APP_NAME")
	return map[string]string{
		"Check if there are any web healthchecks": fmt.Sprintf("%s %s web", appName, c.Name()),
	}
}

func (c *ExistsCommand) Arguments() []command.Argument {
	args := []command.Argument{}
	args = append(args, command.Argument{
		Name:        "process-type",
		Description: "process type to add a check to",
		Optional:    false,
		Type:        command.ArgumentString,
	})
	return args
}

func (c *ExistsCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *ExistsCommand) ParsedArguments(args []string) (map[string]command.Argument, error) {
	return command.ParseArguments(args, c.Arguments())
}

func (c *ExistsCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.Name(), command.FlagSetClient)
	f.StringVar(&c.appJSONFile, "app-json", "app.json", "full path to app.json file")
	return f
}

func (c *ExistsCommand) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(command.FlagSetClient),
		complete.Flags{
			"--app-json": complete.PredictAnything,
		},
	)
}

func (c *ExistsCommand) Run(args []string) int {
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
		if _, err := os.Stat(c.appJSONFile); err == nil {
			contents, err = os.ReadFile(c.appJSONFile)
			if err != nil {
				c.Ui.Error(err.Error())
				return 1
			}
		}
	}

	parsed, err := gabs.ParseJSON(contents)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	processType := arguments["process-type"].StringValue()
	path := fmt.Sprintf("healthchecks.%s", processType)
	exists := parsed.ExistsP(path)
	length := len(parsed.Path(path).Children())
	if exists && length > 0 {
		return 0
	}

	c.Ui.Error(fmt.Sprintf("No healthchecks found in app.json for %s process type", processType))
	return 1
}
