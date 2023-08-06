package commands

import (
	"docker-container-healthchecker/convert"
	"fmt"
	"os"

	"github.com/Jeffail/gabs/v2"
	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

type ConvertCommand struct {
	command.Meta

	appJSONFile string
	inPlace     bool
	prettyPrint bool
}

func (c *ConvertCommand) Name() string {
	return "convert"
}

func (c *ConvertCommand) Synopsis() string {
	return "Eats one or more lollipops"
}

func (c *ConvertCommand) Help() string {
	return command.CommandHelp(c)
}

func (c *ConvertCommand) Examples() map[string]string {
	appName := os.Getenv("CLI_APP_NAME")
	return map[string]string{
		"Convert a file": fmt.Sprintf("%s %s CHECKS", appName, c.Name()),
	}
}

func (c *ConvertCommand) Arguments() []command.Argument {
	args := []command.Argument{}
	args = append(args, command.Argument{
		Name:        "check-file",
		Description: "path to check file",
		Optional:    false,
		Type:        command.ArgumentString,
	})
	return args
}

func (c *ConvertCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *ConvertCommand) ParsedArguments(args []string) (map[string]command.Argument, error) {
	return command.ParseArguments(args, c.Arguments())
}

func (c *ConvertCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.Name(), command.FlagSetClient)
	f.BoolVar(&c.inPlace, "in-place", false, "modify any app.json file in place")
	f.BoolVar(&c.prettyPrint, "pretty", false, "pretty print json output")
	f.StringVar(&c.appJSONFile, "app-json", "app.json", "full path to app.json file")
	return f
}

func (c *ConvertCommand) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(command.FlagSetClient),
		complete.Flags{
			"--app-json": complete.PredictAnything,
			"--in-place": complete.PredictAnything,
			"--pretty":   complete.PredictNothing,
		},
	)
}

func (c *ConvertCommand) Run(args []string) int {
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

	data, err := os.ReadFile(arguments["check-file"].StringValue())
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	checkFile := convert.New(convert.WithData(data))
	if err := checkFile.Parse(); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	contents := []byte("{}")
	if c.appJSONFile == "" {
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

	parsed.SetP(checkFile.ToHealthchecks(), "healthchecks.web")

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
