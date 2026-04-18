package main

import (
	"context"
	"fmt"
	"os"

	"docker-container-healthchecker/commands"

	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/mitchellh/cli"
)

// The name of the cli tool
var AppName = "docker-container-healthchecker"

// Holds the version
var Version string

func main() {
	os.Exit(Run(os.Args[1:]))
}

// Executes the specified subcommand
func Run(args []string) int {
	ctx := context.Background()
	commandMeta := command.SetupRun(ctx, AppName, Version, args)
	commandMeta.Ui = command.HumanZerologUiWithFields(commandMeta.Ui, make(map[string]interface{}, 0))

	cliArgs := os.Args[1:]
	// When invoked by `docker <plugin-name> ...`, Docker CLI prepends the
	// plugin name as argv[1] and sets DOCKER_CLI_PLUGIN_ORIGINAL_CLI_COMMAND.
	// Strip the prepended name only in that case so direct invocations keep
	// their args intact.
	if os.Getenv("DOCKER_CLI_PLUGIN_ORIGINAL_CLI_COMMAND") != "" &&
		len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		cliArgs = os.Args[2:]
	}

	c := cli.NewCLI(AppName, Version)
	c.Args = cliArgs
	c.Commands = command.Commands(ctx, commandMeta, Commands)
	c.HiddenCommands = []string{"docker-cli-plugin-metadata"}
	exitCode, err := c.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing CLI: %s\n", err.Error())
		return 1
	}

	return exitCode
}

// Returns a list of implemented commands
func Commands(ctx context.Context, meta command.Meta) map[string]cli.CommandFactory {
	return map[string]cli.CommandFactory{
		"add": func() (cli.Command, error) {
			return &commands.AddCommand{Meta: meta}, nil
		},
		"check": func() (cli.Command, error) {
			return &commands.CheckCommand{Meta: meta}, nil
		},
		"convert": func() (cli.Command, error) {
			return &commands.ConvertCommand{Meta: meta}, nil
		},
		"docker-cli-plugin-metadata": func() (cli.Command, error) {
			return &commands.DockerCliPluginMetadataCommand{Meta: meta, Version: Version}, nil
		},
		"exists": func() (cli.Command, error) {
			return &commands.ExistsCommand{Meta: meta}, nil
		},
		"version": func() (cli.Command, error) {
			return &command.VersionCommand{Meta: meta}, nil
		},
	}
}
