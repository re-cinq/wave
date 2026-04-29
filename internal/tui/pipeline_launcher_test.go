package tui

import (
	"testing"

	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"github.com/stretchr/testify/assert"
)

func TestNewPipelineLauncher_InitializesFields(t *testing.T) {
	launcher := NewPipelineLauncher(LaunchDependencies{})
	assert.NotNil(t, launcher)
	assert.Nil(t, launcher.program)
}

func TestPipelineLauncher_Cancel_NilStore_IsNoOp(t *testing.T) {
	launcher := NewPipelineLauncher(LaunchDependencies{})
	// Should not panic even with no store
	launcher.Cancel("nonexistent-run-id")
}

func TestPipelineLauncher_Cancel_CallsRequestCancellation(t *testing.T) {
	store := &cancelMockStore{}
	launcher := NewPipelineLauncher(LaunchDependencies{Store: store})

	launcher.Cancel("test-run-1")

	assert.Equal(t, "test-run-1", store.cancelledRunID, "should call RequestCancellation via store")
}

// cancelMockStore records RequestCancellation calls.
type cancelMockStore struct {
	baseStateStore
	cancelledRunID string
}

func (c *cancelMockStore) RequestCancellation(runID string, force bool) error {
	c.cancelledRunID = runID
	return nil
}

func TestPipelineLauncher_CancelAll_IsNoOp(t *testing.T) {
	launcher := NewPipelineLauncher(LaunchDependencies{})
	// CancelAll is a no-op for detached pipelines — should not panic
	assert.NotPanics(t, func() {
		launcher.CancelAll()
	})
}

func TestPipelineLauncher_Cleanup_IsNoOp(t *testing.T) {
	launcher := NewPipelineLauncher(LaunchDependencies{})
	// Cleanup is a no-op for detached pipelines — should not panic
	assert.NotPanics(t, func() {
		launcher.Cleanup("nonexistent-run-id")
	})
}

func TestPipelineLauncher_Launch_MissingPipelineDir_ReturnsError(t *testing.T) {
	launcher := NewPipelineLauncher(LaunchDependencies{
		PipelinesDir: "/nonexistent/dir",
	})

	cmd := launcher.Launch(LaunchConfig{PipelineName: "nonexistent"})
	assert.NotNil(t, cmd)

	msg := cmd()
	errMsg, ok := msg.(LaunchErrorMsg)
	assert.True(t, ok, "should return LaunchErrorMsg")
	assert.Contains(t, errMsg.Err.Error(), "loading pipeline")
}

func TestPipelineLauncher_SetProgram(t *testing.T) {
	launcher := NewPipelineLauncher(LaunchDependencies{})
	assert.Nil(t, launcher.program)

	// SetProgram with nil should not panic
	launcher.SetProgram(nil)
	assert.Nil(t, launcher.program)
}

func TestTUIProgressEmitter_EmitProgress_NilProgram(t *testing.T) {
	emitter := &TUIProgressEmitter{program: nil, runID: "run-1"}
	// Should not panic with nil program
	err := emitter.EmitProgress(event.Event{State: event.StateStarted})
	assert.NoError(t, err)
}

func TestLaunchConfigToOptions_TranslatesKnownFlags(t *testing.T) {
	cfg := LaunchConfig{
		PipelineName:  "impl-issue",
		Input:         "fix bug",
		ModelOverride: "haiku",
		Adapter:       "claude",
		Timeout:       30,
		FromStep:      "implement",
		Steps:         "plan,implement",
		Exclude:       "create-pr",
		OnFailure:     "skip",
		Flags: []string{
			"--verbose",
			"--debug",
			"--dry-run",
			"--mock",
			"--detach",
			"--output text",
		},
	}

	opts := launchConfigToOptions(cfg, "run-xyz")

	assert.Equal(t, "impl-issue", opts.Pipeline)
	assert.Equal(t, "fix bug", opts.Input)
	assert.Equal(t, "run-xyz", opts.RunID)
	assert.Equal(t, "haiku", opts.Model)
	assert.Equal(t, "claude", opts.Adapter)
	assert.Equal(t, 30, opts.Timeout)
	assert.Equal(t, "implement", opts.FromStep)
	assert.Equal(t, "plan,implement", opts.Steps)
	assert.Equal(t, "create-pr", opts.Exclude)
	assert.Equal(t, "skip", opts.OnFailure)
	assert.True(t, opts.Output.Verbose, "--verbose should map to Output.Verbose")
	assert.True(t, opts.Output.Debug, "--debug should map to Output.Debug")
	assert.True(t, opts.DryRun, "--dry-run should map to DryRun")
	assert.True(t, opts.Mock, "--mock should map to Mock")
	assert.Equal(t, "text", opts.Output.Format, "--output text should map to Output.Format")
	// --detach is intentionally a no-op; the runner is producing the
	// detached child, recursing would be wrong.
	assert.False(t, opts.Detach, "--detach must not propagate into config.RuntimeConfig.Detach")
}

func TestLaunchConfigToOptions_EmptyFlagsLeaveDefaults(t *testing.T) {
	cfg := LaunchConfig{PipelineName: "p"}
	opts := launchConfigToOptions(cfg, "rid")

	assert.Equal(t, "p", opts.Pipeline)
	assert.Equal(t, "rid", opts.RunID)
	assert.False(t, opts.Output.Verbose)
	assert.False(t, opts.Output.Debug)
	assert.False(t, opts.DryRun)
	assert.False(t, opts.Mock)
	assert.Equal(t, "", opts.Output.Format)
}

func TestManifestEnvPassthrough_NilManifest(t *testing.T) {
	out := manifestEnvPassthrough(nil)
	assert.Nil(t, out)
}

func TestManifestEnvPassthrough_NoPassthrough(t *testing.T) {
	m := &manifest.Manifest{}
	out := manifestEnvPassthrough(m)
	assert.Nil(t, out)
}

func TestManifestEnvPassthrough_ForwardsListedKeys(t *testing.T) {
	m := &manifest.Manifest{
		Runtime: manifest.Runtime{
			Sandbox: manifest.RuntimeSandbox{
				EnvPassthrough: []string{"GH_TOKEN", "GITHUB_TOKEN"},
			},
		},
	}

	out := manifestEnvPassthrough(m)
	assert.Equal(t, []string{"GH_TOKEN", "GITHUB_TOKEN"}, out)
}
