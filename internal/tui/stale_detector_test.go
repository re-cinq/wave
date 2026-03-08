package tui

import (
	"os"
	"testing"

	"github.com/recinq/wave/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsProcessAlive_CurrentProcess(t *testing.T) {
	// Current process is definitely alive
	assert.True(t, IsProcessAlive(os.Getpid()))
}

func TestIsProcessAlive_InvalidPID(t *testing.T) {
	assert.False(t, IsProcessAlive(0))
	assert.False(t, IsProcessAlive(-1))
}

func TestIsProcessAlive_DeadPID(t *testing.T) {
	// Use a very high PID that's almost certainly not in use
	assert.False(t, IsProcessAlive(4194304))
}

func TestStaleRunDetector_DetectsStaleRuns(t *testing.T) {
	store, err := state.NewStateStore(":memory:")
	require.NoError(t, err)
	defer store.Close()

	// Create a run with a dead PID
	runID, err := store.CreateRun("test-pipeline", "test-input")
	require.NoError(t, err)
	err = store.UpdateRunStatus(runID, "running", "", 0)
	require.NoError(t, err)
	// Use a PID that's definitely not alive
	err = store.UpdateRunPID(runID, 4194304)
	require.NoError(t, err)

	detector := NewStaleRunDetector(store)
	staleIDs, err := detector.DetectStaleRuns()
	require.NoError(t, err)
	assert.Contains(t, staleIDs, runID)

	// Verify the run was transitioned to "failed"
	run, err := store.GetRun(runID)
	require.NoError(t, err)
	assert.Equal(t, "failed", run.Status)
	// UpdateRunStatus stores error info in CurrentStep (not ErrorMessage)
	assert.Contains(t, run.CurrentStep, "stale")
}

func TestStaleRunDetector_SkipsRunsWithZeroPID(t *testing.T) {
	store, err := state.NewStateStore(":memory:")
	require.NoError(t, err)
	defer store.Close()

	// Create a run with PID=0 (in-process)
	runID, err := store.CreateRun("test-pipeline", "test-input")
	require.NoError(t, err)
	err = store.UpdateRunStatus(runID, "running", "", 0)
	require.NoError(t, err)

	detector := NewStaleRunDetector(store)
	staleIDs, err := detector.DetectStaleRuns()
	require.NoError(t, err)
	assert.Empty(t, staleIDs)

	// Run should still be "running"
	run, err := store.GetRun(runID)
	require.NoError(t, err)
	assert.Equal(t, "running", run.Status)
}

func TestStaleRunDetector_KeepsLiveRuns(t *testing.T) {
	store, err := state.NewStateStore(":memory:")
	require.NoError(t, err)
	defer store.Close()

	// Create a run with the current process PID (definitely alive)
	runID, err := store.CreateRun("test-pipeline", "test-input")
	require.NoError(t, err)
	err = store.UpdateRunStatus(runID, "running", "", 0)
	require.NoError(t, err)
	err = store.UpdateRunPID(runID, os.Getpid())
	require.NoError(t, err)

	detector := NewStaleRunDetector(store)
	staleIDs, err := detector.DetectStaleRuns()
	require.NoError(t, err)
	assert.Empty(t, staleIDs)

	// Run should still be "running"
	run, err := store.GetRun(runID)
	require.NoError(t, err)
	assert.Equal(t, "running", run.Status)
}
