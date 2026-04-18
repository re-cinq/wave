package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestBuildPassthroughEnv_MinimalEnv(t *testing.T) {
	deps := LaunchDependencies{}
	env := buildPassthroughEnv(deps)

	// Should always include HOME and PATH
	assert.Contains(t, env, "HOME="+os.Getenv("HOME"))
	assert.Contains(t, env, "PATH="+os.Getenv("PATH"))
	assert.Len(t, env, 2)
}

func TestBuildPassthroughEnv_WithManifestPassthrough(t *testing.T) {
	// Set a test env var to verify passthrough
	t.Setenv("WAVE_TEST_VAR", "test-value")

	deps := LaunchDependencies{
		Manifest: &manifest.Manifest{
			Runtime: manifest.Runtime{
				Sandbox: manifest.RuntimeSandbox{
					EnvPassthrough: []string{"WAVE_TEST_VAR", "NONEXISTENT_VAR"},
				},
			},
		},
	}
	env := buildPassthroughEnv(deps)

	// Should include HOME, PATH, and the passthrough var
	assert.Contains(t, env, "HOME="+os.Getenv("HOME"))
	assert.Contains(t, env, "PATH="+os.Getenv("PATH"))
	assert.Contains(t, env, "WAVE_TEST_VAR=test-value")
	// NONEXISTENT_VAR should not be included since it's not set
	assert.Len(t, env, 3)
}

func TestBuildPassthroughEnv_NilManifest(t *testing.T) {
	deps := LaunchDependencies{
		Manifest: nil,
	}
	env := buildPassthroughEnv(deps)
	assert.Len(t, env, 2, "should only include HOME and PATH")
}

func TestOpenRunLog_CreatesDirectoryAndFile(t *testing.T) {
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	f, err := openRunLog("test-run-123")
	require.NoError(t, err)
	defer f.Close()

	// Verify the directory was created
	_, err = os.Stat(filepath.Join(".agents", "logs"))
	assert.NoError(t, err)

	// Verify the file was created
	_, err = os.Stat(filepath.Join(".agents", "logs", "test-run-123.log"))
	assert.NoError(t, err)

	// Verify it's writable
	_, err = f.WriteString("test output\n")
	assert.NoError(t, err)
}

func TestOpenRunLog_AppendsToExisting(t *testing.T) {
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	// First write
	f1, err := openRunLog("append-run")
	require.NoError(t, err)
	_, _ = f1.WriteString("first\n")
	f1.Close()

	// Second write (append)
	f2, err := openRunLog("append-run")
	require.NoError(t, err)
	_, _ = f2.WriteString("second\n")
	f2.Close()

	content, err := os.ReadFile(filepath.Join(".agents", "logs", "append-run.log"))
	require.NoError(t, err)
	assert.Equal(t, "first\nsecond\n", string(content))
}
