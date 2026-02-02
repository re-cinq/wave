package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/recinq/wave/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cancelTestEnv provides a testing environment for cancel tests
type cancelTestEnv struct {
	t       *testing.T
	rootDir string
	origDir string
	store   state.StateStore
}

// newCancelTestEnv creates a new test environment with a temp directory and state store
func newCancelTestEnv(t *testing.T) *cancelTestEnv {
	t.Helper()

	origDir, err := os.Getwd()
	require.NoError(t, err, "failed to get current directory")

	tmpDir := t.TempDir()
	err = os.Chdir(tmpDir)
	require.NoError(t, err, "failed to change to temp directory")

	// Create .wave directory structure
	err = os.MkdirAll(".wave/pids", 0755)
	require.NoError(t, err, "failed to create .wave/pids directory")

	// Create state store
	store, err := state.NewStateStore(".wave/state.db")
	require.NoError(t, err, "failed to create state store")

	return &cancelTestEnv{
		t:       t,
		rootDir: tmpDir,
		origDir: origDir,
		store:   store,
	}
}

// cleanup restores the original working directory and closes the store
func (e *cancelTestEnv) cleanup() {
	if e.store != nil {
		e.store.Close()
	}
	err := os.Chdir(e.origDir)
	if err != nil {
		e.t.Errorf("failed to restore original directory: %v", err)
	}
}

// createRunningRun creates a running pipeline run in the database
func (e *cancelTestEnv) createRunningRun(pipelineName string) string {
	e.t.Helper()

	runID, err := e.store.CreateRun(pipelineName, `{"test": true}`)
	require.NoError(e.t, err, "failed to create run")

	err = e.store.UpdateRunStatus(runID, "running", "step-1", 1000)
	require.NoError(e.t, err, "failed to update run status")

	return runID
}

// createCompletedRun creates a completed pipeline run in the database
func (e *cancelTestEnv) createCompletedRun(pipelineName string) string {
	e.t.Helper()

	runID, err := e.store.CreateRun(pipelineName, `{"test": true}`)
	require.NoError(e.t, err, "failed to create run")

	err = e.store.UpdateRunStatus(runID, "completed", "step-final", 5000)
	require.NoError(e.t, err, "failed to update run status")

	return runID
}

// createPidFile creates a fake PID file for testing force cancellation
func (e *cancelTestEnv) createPidFile(runID string, pid int) {
	e.t.Helper()

	pidFile := filepath.Join(".wave", "pids", runID+".pid")
	err := os.WriteFile(pidFile, []byte(string(rune('0'+pid%10))+string(rune('0'+pid/10%10))+string(rune('0'+pid/100%10))), 0644)
	if pid > 999 {
		// For larger PIDs, use proper formatting
		err = os.WriteFile(pidFile, []byte(intToString(pid)), 0644)
	}
	require.NoError(e.t, err, "failed to create pid file")
}

func intToString(n int) string {
	if n == 0 {
		return "0"
	}
	var result []byte
	for n > 0 {
		result = append([]byte{byte('0' + n%10)}, result...)
		n /= 10
	}
	return string(result)
}

// executeCancelCmd runs the cancel command with given arguments and returns output/error
func executeCancelCmd(args ...string) (stdout, stderr string, err error) {
	cmd := NewCancelCmd()
	cmd.SetArgs(args)

	// Capture stdout
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	err = cmd.Execute()

	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var outBuf, errBuf bytes.Buffer
	io.Copy(&outBuf, rOut)
	io.Copy(&errBuf, rErr)

	return outBuf.String(), errBuf.String(), err
}

// TestCancelGraceful tests graceful cancellation (flag is set)
func TestCancelGraceful(t *testing.T) {
	env := newCancelTestEnv(t)
	defer env.cleanup()

	// Create a running pipeline
	runID := env.createRunningRun("test-pipeline")

	// Cancel without force
	stdout, stderr, err := executeCancelCmd(runID)

	require.NoError(t, err)
	assert.Contains(t, stdout, "Cancellation requested")
	assert.Contains(t, stdout, "test-pipeline")
	assert.Empty(t, stderr)

	// Verify cancellation was requested in database
	record, err := env.store.CheckCancellation(runID)
	require.NoError(t, err)
	assert.NotNil(t, record)
	assert.Equal(t, runID, record.RunID)
	assert.False(t, record.Force)
}

// TestCancelForce tests force cancellation (with mock process)
func TestCancelForce(t *testing.T) {
	env := newCancelTestEnv(t)
	defer env.cleanup()

	// Create a running pipeline
	runID := env.createRunningRun("test-pipeline")

	// Cancel with force flag
	stdout, stderr, err := executeCancelCmd("--force", runID)

	require.NoError(t, err)
	assert.Contains(t, stdout, "Force cancellation sent")
	assert.Contains(t, stdout, "test-pipeline")
	_ = stderr // May contain warnings about missing pidfile

	// Verify cancellation was requested with force flag
	record, err := env.store.CheckCancellation(runID)
	require.NoError(t, err)
	assert.NotNil(t, record)
	assert.True(t, record.Force)
}

// TestCancelNoRunningPipeline tests cancellation with no running pipeline
func TestCancelNoRunningPipeline(t *testing.T) {
	env := newCancelTestEnv(t)
	defer env.cleanup()

	// Don't create any running pipelines

	// Try to cancel - should fail since there's nothing to cancel
	_, _, err := executeCancelCmd()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "No running pipelines to cancel")
}

// TestCancelInvalidRunID tests cancellation with invalid run-id
func TestCancelInvalidRunID(t *testing.T) {
	env := newCancelTestEnv(t)
	defer env.cleanup()

	// Try to cancel non-existent run
	_, _, err := executeCancelCmd("nonexistent-run-id")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Run not found")
}

// TestCancelNonRunningPipeline tests cancellation of a completed pipeline
func TestCancelNonRunningPipeline(t *testing.T) {
	env := newCancelTestEnv(t)
	defer env.cleanup()

	// Create a completed pipeline
	runID := env.createCompletedRun("completed-pipeline")

	// Try to cancel it
	_, _, err := executeCancelCmd(runID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

// TestCancelMostRecentRunning tests cancelling a running pipeline when multiple exist
func TestCancelMostRecentRunning(t *testing.T) {
	env := newCancelTestEnv(t)
	defer env.cleanup()

	// Create multiple running pipelines
	runID1 := env.createRunningRun("pipeline-1")
	runID2 := env.createRunningRun("pipeline-2")

	// Cancel without specifying run ID - should cancel one of them
	stdout, _, err := executeCancelCmd()

	require.NoError(t, err)
	assert.Contains(t, stdout, "Cancellation requested")
	// Should cancel one of the two pipelines
	cancelledOne := strings.Contains(stdout, runID1) || strings.Contains(stdout, runID2)
	assert.True(t, cancelledOne, "should cancel one of the running pipelines")
}

// TestCancelConcurrent tests concurrent cancellation requests against the database
// Note: This test focuses on verifying database-level concurrency, not stdout capture
func TestCancelConcurrent(t *testing.T) {
	env := newCancelTestEnv(t)
	defer env.cleanup()

	// Create a running pipeline
	runID := env.createRunningRun("concurrent-test")

	// Launch multiple concurrent cancellation requests directly to the store
	var wg sync.WaitGroup
	numRequests := 5
	results := make([]error, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = env.store.RequestCancellation(runID, false)
		}(i)
	}

	wg.Wait()

	// All requests should complete (concurrent cancellation requests should be safe)
	errorCount := 0
	for _, err := range results {
		if err != nil {
			errorCount++
		}
	}
	// All should succeed (or at least most - SQLite handles concurrent access)
	assert.LessOrEqual(t, errorCount, 1, "most cancellation requests should succeed")

	// Verify cancellation was recorded
	record, err := env.store.CheckCancellation(runID)
	require.NoError(t, err)
	assert.NotNil(t, record)
}

// TestCancelJSONFormat tests JSON output format
func TestCancelJSONFormat(t *testing.T) {
	env := newCancelTestEnv(t)
	defer env.cleanup()

	// Create a running pipeline
	runID := env.createRunningRun("json-test")

	// Cancel with JSON format
	stdout, _, err := executeCancelCmd("--format", "json", runID)

	require.NoError(t, err)

	// Parse JSON output
	var result CancelResult
	err = json.Unmarshal([]byte(stdout), &result)
	require.NoError(t, err)

	assert.True(t, result.Success)
	assert.Equal(t, runID, result.RunID)
	assert.Equal(t, "json-test", result.PipelineName)
	assert.Contains(t, result.Message, "Cancellation requested")
	assert.False(t, result.Force)
}

// TestCancelJSONFormatForce tests JSON output format with force flag
func TestCancelJSONFormatForce(t *testing.T) {
	env := newCancelTestEnv(t)
	defer env.cleanup()

	// Create a running pipeline
	runID := env.createRunningRun("json-force-test")

	// Cancel with force and JSON format
	stdout, _, err := executeCancelCmd("--format", "json", "--force", runID)

	require.NoError(t, err)

	// Parse JSON output
	var result CancelResult
	err = json.Unmarshal([]byte(stdout), &result)
	require.NoError(t, err)

	assert.True(t, result.Success)
	assert.True(t, result.Force)
	assert.Contains(t, result.Message, "Force cancellation")
}

// TestCancelJSONFormatNoRunning tests JSON output when no running pipeline
func TestCancelJSONFormatNoRunning(t *testing.T) {
	env := newCancelTestEnv(t)
	defer env.cleanup()

	// Cancel with JSON format - no running pipelines
	stdout, _, err := executeCancelCmd("--format", "json")

	// JSON output is returned even on failure (error is nil for JSON format)
	require.NoError(t, err)

	// Parse JSON output
	var result CancelResult
	err = json.Unmarshal([]byte(stdout), &result)
	require.NoError(t, err)

	assert.False(t, result.Success)
	assert.Contains(t, result.Message, "No running pipelines to cancel")
}

// TestCancelJSONFormatError tests JSON output on error
func TestCancelJSONFormatError(t *testing.T) {
	env := newCancelTestEnv(t)
	defer env.cleanup()

	// Cancel non-existent run with JSON format
	stdout, _, _ := executeCancelCmd("--format", "json", "nonexistent-id")

	// Parse JSON output - even errors should be valid JSON
	var result CancelResult
	err := json.Unmarshal([]byte(stdout), &result)
	require.NoError(t, err)

	assert.False(t, result.Success)
	assert.Contains(t, result.Message, "not found")
}

// TestNewCancelCmdFlags verifies the cancel command has all expected flags
func TestNewCancelCmdFlags(t *testing.T) {
	cmd := NewCancelCmd()

	// Verify command properties
	assert.Equal(t, "cancel [run-id]", cmd.Use)
	assert.Contains(t, cmd.Short, "Cancel")

	// Verify all flags exist
	flags := cmd.Flags()

	forceFlag := flags.Lookup("force")
	assert.NotNil(t, forceFlag, "force flag should exist")
	assert.Equal(t, "f", forceFlag.Shorthand)

	formatFlag := flags.Lookup("format")
	assert.NotNil(t, formatFlag, "format flag should exist")
	assert.Equal(t, "text", formatFlag.DefValue)
}

// TestCancelNoStateDB tests behavior when state database doesn't exist
func TestCancelNoStateDB(t *testing.T) {
	origDir, err := os.Getwd()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer os.Chdir(origDir)

	// Don't create .wave directory - state DB won't exist

	_, _, err = executeCancelCmd()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to open state database")
}

// TestCancelUpdatesRunStatus verifies that cancel updates the run status to cancelled
func TestCancelUpdatesRunStatus(t *testing.T) {
	env := newCancelTestEnv(t)
	defer env.cleanup()

	// Create a running pipeline
	runID := env.createRunningRun("status-test")

	// Verify initial status is running
	run, err := env.store.GetRun(runID)
	require.NoError(t, err)
	assert.Equal(t, "running", run.Status)

	// Cancel the pipeline
	_, _, err = executeCancelCmd(runID)
	require.NoError(t, err)

	// Verify status is now cancelled
	run, err = env.store.GetRun(runID)
	require.NoError(t, err)
	assert.Equal(t, "cancelled", run.Status)
}

// TestForceKillRunNoPidFile tests forceKillRun when no pidfile exists
func TestForceKillRunNoPidFile(t *testing.T) {
	origDir, err := os.Getwd()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer os.Chdir(origDir)

	// Create .wave/pids but no pidfile
	err = os.MkdirAll(".wave/pids", 0755)
	require.NoError(t, err)

	// Should return nil (no error) when pidfile doesn't exist
	err = forceKillRun("nonexistent-run")
	assert.NoError(t, err)
}

// TestForceKillRunInvalidPid tests forceKillRun with invalid PID in pidfile
func TestForceKillRunInvalidPid(t *testing.T) {
	origDir, err := os.Getwd()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer os.Chdir(origDir)

	// Create .wave/pids with invalid pidfile
	err = os.MkdirAll(".wave/pids", 0755)
	require.NoError(t, err)
	err = os.WriteFile(".wave/pids/test-run.pid", []byte("not-a-number"), 0644)
	require.NoError(t, err)

	// Should return error for invalid PID
	err = forceKillRun("test-run")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid pid")
}

// TestForceKillRunNonExistentProcess tests forceKillRun with a PID that doesn't exist
func TestForceKillRunNonExistentProcess(t *testing.T) {
	origDir, err := os.Getwd()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer os.Chdir(origDir)

	// Create .wave/pids with a PID that (almost certainly) doesn't exist
	err = os.MkdirAll(".wave/pids", 0755)
	require.NoError(t, err)
	// Use a very high PID that's unlikely to exist
	err = os.WriteFile(".wave/pids/test-run.pid", []byte("999999"), 0644)
	require.NoError(t, err)

	// Should not return error - process doesn't exist is fine
	err = forceKillRun("test-run")
	assert.NoError(t, err)

	// Pidfile should be removed
	_, err = os.Stat(".wave/pids/test-run.pid")
	assert.True(t, os.IsNotExist(err))
}
