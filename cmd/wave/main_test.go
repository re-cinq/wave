package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoTUIFlag_IsPersistent(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("no-tui")
	assert.NotNil(t, flag, "--no-tui should be registered as a persistent flag")
	assert.Equal(t, "false", flag.DefValue)
}

func TestAllSubcommands_ShowPersistentFlags(t *testing.T) {
	expectedFlags := []string{
		"--json",
		"--quiet",
		"--no-color",
		"--debug",
		"--verbose",
		"--no-tui",
		"--output",
	}

	// Get all registered subcommands
	subcommands := rootCmd.Commands()
	assert.NotEmpty(t, subcommands, "root command should have subcommands")

	for _, subcmd := range subcommands {
		t.Run(subcmd.Name(), func(t *testing.T) {
			// Get the help text
			help := subcmd.UsageString()

			for _, flag := range expectedFlags {
				assert.True(t, strings.Contains(help, flag),
					"subcommand %q help should contain %q", subcmd.Name(), flag)
			}
		})
	}
}
