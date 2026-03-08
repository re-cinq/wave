package tui

import (
	"os"
	"testing"

	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/state"
	"github.com/stretchr/testify/assert"
)

func TestNewPipelineLauncher_InitializesDefaults(t *testing.T) {
	launcher := NewPipelineLauncher(LaunchDependencies{})
	assert.NotNil(t, launcher)
	assert.Nil(t, launcher.program)
}

func TestPipelineLauncher_Cancel_NoStore_IsNoOp(t *testing.T) {
	launcher := NewPipelineLauncher(LaunchDependencies{})
	// Should not panic when no store is set
	launcher.Cancel("nonexistent-run-id")
}

func TestPipelineLauncher_Cancel_WithStore_RequestsCancellation(t *testing.T) {
	store, cleanup := setupTestStateStore(t)
	defer cleanup()

	launcher := NewPipelineLauncher(LaunchDependencies{Store: store})

	// Create a run first
	runID, err := store.CreateRun("test-pipeline", "test-input")
	assert.NoError(t, err)

	launcher.Cancel(runID)

	// Verify cancellation was requested in the store
	cancel, err := store.CheckCancellation(runID)
	assert.NoError(t, err)
	assert.NotNil(t, cancel)
	assert.Equal(t, runID, cancel.RunID)
}

func TestPipelineLauncher_CancelAll_IsNoOp(t *testing.T) {
	launcher := NewPipelineLauncher(LaunchDependencies{})
	// Should not panic — CancelAll is a no-op for detached subprocesses
	launcher.CancelAll()
}

func TestPipelineLauncher_Cleanup_IsNoOp(t *testing.T) {
	launcher := NewPipelineLauncher(LaunchDependencies{})
	// Should not panic
	launcher.Cleanup("nonexistent-run-id")
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

func TestBuildPassthroughEnv_IncludesHomeAndPath(t *testing.T) {
	deps := LaunchDependencies{}
	env := buildPassthroughEnv(deps)

	// Should include HOME and PATH at minimum
	hasHome := false
	hasPath := false
	for _, v := range env {
		if len(v) > 5 && v[:5] == "HOME=" {
			hasHome = true
		}
		if len(v) > 5 && v[:5] == "PATH=" {
			hasPath = true
		}
	}
	if os.Getenv("HOME") != "" {
		assert.True(t, hasHome, "should include HOME")
	}
	if os.Getenv("PATH") != "" {
		assert.True(t, hasPath, "should include PATH")
	}
}

// setupTestStateStore creates a real in-memory state store for testing.
func setupTestStateStore(t *testing.T) (state.StateStore, func()) {
	t.Helper()
	store, err := state.NewStateStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	cleanup := func() { store.Close() }
	return store, cleanup
}
