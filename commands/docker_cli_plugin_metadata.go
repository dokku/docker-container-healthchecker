package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

// DockerCliPluginMetadataCommand is the command for the plugin metadata
type DockerCliPluginMetadataCommand struct {
	command.Meta
	// Version is the version of the plugin
	Version string
}

// PluginMetadata is the metadata for the plugin
type PluginMetadata struct {
	// SchemaVersion is the schema version of the plugin
	SchemaVersion string `json:"SchemaVersion"`
	// Vendor is the vendor of the plugin
	Vendor string `json:"Vendor"`
	// Version is the version of the plugin
	Version string `json:"Version"`
	// ShortDescription is the short description of the plugin
	ShortDescription string `json:"ShortDescription"`
}

func (c *DockerCliPluginMetadataCommand) Name() string {
	return "docker-cli-plugin-metadata"
}

func (c *DockerCliPluginMetadataCommand) Synopsis() string {
	return "Prints the metadata for the Docker CLI plugin"
}

func (c *DockerCliPluginMetadataCommand) Help() string {
	return command.CommandHelp(c)
}

func (c *DockerCliPluginMetadataCommand) Examples() map[string]string {
	appName := os.Getenv("CLI_APP_NAME")
	return map[string]string{
		"Prints the metadata for the Docker CLI plugin": fmt.Sprintf("%s %s", appName, c.Name()),
	}
}

func (c *DockerCliPluginMetadataCommand) Arguments() []command.Argument {
	return []command.Argument{}
}

func (c *DockerCliPluginMetadataCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *DockerCliPluginMetadataCommand) ParsedArguments(args []string) (map[string]command.Argument, error) {
	return command.ParseArguments(args, c.Arguments())
}

func (c *DockerCliPluginMetadataCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.Name(), command.FlagSetClient)
	return f
}

func (c *DockerCliPluginMetadataCommand) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(command.FlagSetClient),
		complete.Flags{},
	)
}

func (c *DockerCliPluginMetadataCommand) Run(args []string) int {
	flags := c.FlagSet()
	flags.Usage = func() { c.Ui.Output(c.Help()) }
	if err := flags.Parse(args); err != nil {
		c.Ui.Error(err.Error())
		c.Ui.Error(command.CommandErrorText(c))
		return 1
	}

	metadata := PluginMetadata{
		SchemaVersion:    "0.1.0",
		Vendor:           "Jose Diaz-Gonzalez",
		Version:          c.Version,
		ShortDescription: "Runs healthchecks against local docker containers",
	}

	jsonData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling metadata: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(jsonData))

	return 0
}
