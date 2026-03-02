package commands

import (
	"testing"

	"github.com/recinq/wave/internal/tui"
	"github.com/stretchr/testify/assert"
)

func TestNewRunCmd_ProposeFlag(t *testing.T) {
	cmd := NewRunCmd()
	flag := cmd.Flags().Lookup("propose")
	assert.NotNil(t, flag, "--propose flag should be registered")
	assert.Equal(t, "false", flag.DefValue, "--propose should default to false")
}

func TestNewRunCmd_ProposeNonTTY(t *testing.T) {
	// When --propose is set but not on a TTY, the command should fall through
	// to normal pipeline resolution (not invoke proposal selector).
	t.Setenv("WAVE_FORCE_TTY", "0")
	cmd := NewRunCmd()
	cmd.Root().PersistentFlags().String("output", "auto", "")
	cmd.Root().PersistentFlags().Bool("verbose", false, "")
	cmd.Root().PersistentFlags().Bool("debug", false, "")
	cmd.SetArgs([]string{"--propose", "--mock"})
	err := cmd.Execute()
	// Should get "pipeline name is required" since --propose is a no-op without TTY
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline name is required")
}

func TestNewRunCmd_ProposeWithPipeline(t *testing.T) {
	// When both --propose and a pipeline name are given, the pipeline name
	// takes precedence (--propose is only for interactive selection).
	t.Setenv("WAVE_FORCE_TTY", "0")
	cmd := NewRunCmd()
	cmd.Root().PersistentFlags().String("output", "auto", "")
	cmd.Root().PersistentFlags().Bool("verbose", false, "")
	cmd.Root().PersistentFlags().Bool("debug", false, "")
	cmd.SetArgs([]string{"nonexistent-pipeline", "--propose"})
	err := cmd.Execute()
	// Should fail at pipeline loading, not at proposal selector
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "pipeline name is required")
}

func TestMockProposalProvider_ReturnsValidProposals(t *testing.T) {
	proposals := tui.MockProposalProvider()
	assert.NotEmpty(t, proposals, "mock proposals should not be empty")

	err := tui.ValidateProposals(proposals)
	assert.NoError(t, err, "mock proposals should pass validation")

	// Verify the mock includes parallel groups
	hasParallelGroup := false
	for _, p := range proposals {
		if p.ParallelGroup != "" {
			hasParallelGroup = true
			break
		}
	}
	assert.True(t, hasParallelGroup, "mock proposals should include at least one parallel group")

	// Verify the mock includes dependencies
	hasDeps := false
	for _, p := range proposals {
		if len(p.Dependencies) > 0 {
			hasDeps = true
			break
		}
	}
	assert.True(t, hasDeps, "mock proposals should include at least one proposal with dependencies")
}
