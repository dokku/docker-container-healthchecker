package commands

import (
	"docker-container-healthchecker/appjson"
	"docker-container-healthchecker/convert"
	"encoding/json"
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
	f.BoolVar(&c.prettyPrint, "pretty", false, "pretty print json output")
	f.StringVar(&c.appJSONFile, "app-json", "", "full path to app.json file to update")
	return f
}

func (c *ConvertCommand) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(command.FlagSetClient),
		complete.Flags{
			"--app-json": complete.PredictAnything,
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

	healthchecks := checkFile.ToHealthchecks()
	if c.appJSONFile == "" {
		output := appjson.AppJSON{
			Healthchecks: map[string][]appjson.Healthcheck{
				"web": healthchecks,
			},
		}

		var b []byte
		if c.prettyPrint {
			b, err = json.MarshalIndent(output, "", "    ")
		} else {
			b, err = json.Marshal(output)
		}

		if err != nil {
			c.Ui.Error(err.Error())
			return 1
		}

		fmt.Println(string(b))
	} else {
		contents := []byte("{}")
		if _, err := os.Stat(c.appJSONFile); err == nil {
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

		parsed.SetP(healthchecks, "healthchecks.web")

		if c.prettyPrint {
			err = os.WriteFile(c.appJSONFile, parsed.BytesIndent("", "  "), 0644)
		} else {
			err = os.WriteFile(c.appJSONFile, parsed.Bytes(), 0644)
		}

		if err != nil {
			c.Ui.Error(err.Error())
			return 1
		}
	}

	return 0
}
