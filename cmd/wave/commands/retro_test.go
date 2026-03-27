package commands

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRetroCmd_HasSubcommands(t *testing.T) {
	cmd := NewRetroCmd()

	assert.Equal(t, "retro", cmd.Use)
	assert.Contains(t, cmd.Short, "retrospective")

	subcommands := cmd.Commands()
	names := make(map[string]bool, len(subcommands))
	for _, sub := range subcommands {
		names[sub.Name()] = true
	}

	assert.True(t, names["view"], "should have view subcommand")
	assert.True(t, names["list"], "should have list subcommand")
	assert.True(t, names["stats"], "should have stats subcommand")
}

func TestRetroViewCmd_RequiresExactlyOneArg(t *testing.T) {
	cmd := NewRetroCmd()

	// Find the view subcommand
	var viewCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Name() == "view" {
			viewCmd = sub
			break
		}
	}
	require.NotNil(t, viewCmd, "view subcommand must exist")

	// Verify the Args constraint is ExactArgs(1) by checking the validator
	err := viewCmd.Args(viewCmd, []string{})
	assert.Error(t, err, "zero args should be rejected")

	err = viewCmd.Args(viewCmd, []string{"run1"})
	assert.NoError(t, err, "exactly one arg should be accepted")

	err = viewCmd.Args(viewCmd, []string{"run1", "run2"})
	assert.Error(t, err, "two args should be rejected")
}

func TestRetroViewCmd_HasJSONFlag(t *testing.T) {
	cmd := NewRetroCmd()

	var viewCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Name() == "view" {
			viewCmd = sub
			break
		}
	}
	require.NotNil(t, viewCmd)

	jsonFlag := viewCmd.Flags().Lookup("json")
	assert.NotNil(t, jsonFlag, "view should have --json flag")
	assert.Equal(t, "false", jsonFlag.DefValue)
}

func TestRetroListCmd_HasFlags(t *testing.T) {
	cmd := NewRetroCmd()

	var listCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Name() == "list" {
			listCmd = sub
			break
		}
	}
	require.NotNil(t, listCmd)

	pipelineFlag := listCmd.Flags().Lookup("pipeline")
	assert.NotNil(t, pipelineFlag, "list should have --pipeline flag")

	sinceFlag := listCmd.Flags().Lookup("since")
	assert.NotNil(t, sinceFlag, "list should have --since flag")

	limitFlag := listCmd.Flags().Lookup("limit")
	assert.NotNil(t, limitFlag, "list should have --limit flag")
	assert.Equal(t, "20", limitFlag.DefValue)

	jsonFlag := listCmd.Flags().Lookup("json")
	assert.NotNil(t, jsonFlag, "list should have --json flag")
}

func TestRetroStatsCmd_HasFlags(t *testing.T) {
	cmd := NewRetroCmd()

	var statsCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Name() == "stats" {
			statsCmd = sub
			break
		}
	}
	require.NotNil(t, statsCmd)

	jsonFlag := statsCmd.Flags().Lookup("json")
	assert.NotNil(t, jsonFlag, "stats should have --json flag")

	pipelineFlag := statsCmd.Flags().Lookup("pipeline")
	assert.NotNil(t, pipelineFlag, "stats should have --pipeline flag")

	sinceFlag := statsCmd.Flags().Lookup("since")
	assert.NotNil(t, sinceFlag, "stats should have --since flag")
}

func TestCapitalize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"bumpy", "Bumpy"},
		{"smooth", "Smooth"},
		{"Effortless", "Effortless"},
		{"a", "A"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, capitalize(tt.input))
		})
	}
}

func TestComputeRetroStats_Empty(t *testing.T) {
	stats := computeRetroStats(nil)
	assert.Equal(t, 0, stats.TotalRuns)
	assert.Empty(t, stats.SmoothnessDistribution)
	assert.Empty(t, stats.FrictionFrequency)
	assert.Empty(t, stats.PipelineStats)
}
