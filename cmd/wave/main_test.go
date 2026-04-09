package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldLaunchTUI_NoTUIFlag(t *testing.T) {
	cmd := rootCmd
	err := cmd.PersistentFlags().Set("no-tui", "true")
	assert.NoError(t, err)
	defer cmd.PersistentFlags().Set("no-tui", "false") //nolint:errcheck // test cleanup, flag always exists

	result := shouldLaunchTUI(cmd)
	assert.False(t, result)
}

func TestShouldLaunchTUI_ForceTTYEnabled(t *testing.T) {
	t.Setenv("WAVE_FORCE_TTY", "1")

	cmd := rootCmd
	result := shouldLaunchTUI(cmd)
	assert.True(t, result)
}

func TestShouldLaunchTUI_ForceTTYTrue(t *testing.T) {
	t.Setenv("WAVE_FORCE_TTY", "true")

	cmd := rootCmd
	result := shouldLaunchTUI(cmd)
	assert.True(t, result)
}

func TestShouldLaunchTUI_ForceTTYDisabled(t *testing.T) {
	t.Setenv("WAVE_FORCE_TTY", "0")

	cmd := rootCmd
	result := shouldLaunchTUI(cmd)
	assert.False(t, result)
}

func TestShouldLaunchTUI_ForceTTYFalse(t *testing.T) {
	t.Setenv("WAVE_FORCE_TTY", "false")

	cmd := rootCmd
	result := shouldLaunchTUI(cmd)
	assert.False(t, result)
}

func TestShouldLaunchTUI_TermDumb(t *testing.T) {
	t.Setenv("TERM", "dumb")

	cmd := rootCmd
	result := shouldLaunchTUI(cmd)
	assert.False(t, result)
}

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
