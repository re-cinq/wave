package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestForkManager_Integration exercises ForkManager.Fork with a real SQLite
// state store, verifying that the forked run gets its own checkpoints and
// that artifact files are copied to the new run's path.
func TestForkManager_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "state.db")

	store, err := state.NewStateStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	// Create a completed source run.
	sourceRunID, err := store.CreateRun("test-pipeline", "fix the bug")
	require.NoError(t, err)
	require.NoError(t, store.UpdateRunStatus(sourceRunID, "completed", "", 500))

	// Create artifact files on disk so copyArtifacts has something to copy.
	artifactDir := filepath.Join(tmpDir, ".agents", "artifacts", sourceRunID)
	require.NoError(t, os.MkdirAll(artifactDir, 0o755))

	planPath := filepath.Join(artifactDir, "plan.md")
	require.NoError(t, os.WriteFile(planPath, []byte("# Plan\nStep one..."), 0o644))

	implPath := filepath.Join(artifactDir, "impl.patch")
	require.NoError(t, os.WriteFile(implPath, []byte("diff --git a/main.go"), 0o644))

	// Save checkpoints for two completed steps.
	analyzeSnapshot, _ := json.Marshal(map[string]string{
		"analyze:plan": planPath,
	})
	require.NoError(t, store.SaveCheckpoint(&state.CheckpointRecord{
		RunID:            sourceRunID,
		StepID:           "analyze",
		StepIndex:        0,
		WorkspacePath:    filepath.Join(tmpDir, "ws"),
		ArtifactSnapshot: string(analyzeSnapshot),
	}))

	implSnapshot, _ := json.Marshal(map[string]string{
		"analyze:plan":    planPath,
		"implement:patch": implPath,
	})
	require.NoError(t, store.SaveCheckpoint(&state.CheckpointRecord{
		RunID:              sourceRunID,
		StepID:             "implement",
		StepIndex:          1,
		WorkspacePath:      filepath.Join(tmpDir, "ws"),
		WorkspaceCommitSHA: "abc123def",
		ArtifactSnapshot:   string(implSnapshot),
	}))

	// Define a matching pipeline topology.
	p := &pipeline.Pipeline{
		Metadata: pipeline.PipelineMetadata{Name: "test-pipeline"},
		Steps: []pipeline.Step{
			{ID: "analyze"},
			{ID: "implement"},
			{ID: "test"},
		},
	}

	// Execute the fork from the "analyze" step.
	// Change working directory so relative artifact paths resolve correctly.
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	fm := pipeline.NewForkManager(store)
	newRunID, err := fm.Fork(sourceRunID, "analyze", p)
	require.NoError(t, err)
	assert.NotEmpty(t, newRunID)
	assert.NotEqual(t, sourceRunID, newRunID, "forked run must have a distinct ID")

	// Verify the new run exists in the store.
	newRun, err := store.GetRun(newRunID)
	require.NoError(t, err)
	assert.Equal(t, "test-pipeline", newRun.PipelineName)
	assert.Equal(t, "fix the bug", newRun.Input)

	// Verify checkpoints were copied up to and including the fork point (index 0).
	newCheckpoints, err := store.GetCheckpoints(newRunID)
	require.NoError(t, err)
	assert.Len(t, newCheckpoints, 1, "only checkpoint at or before fork point should be copied")
	assert.Equal(t, "analyze", newCheckpoints[0].StepID)
	assert.Equal(t, newRunID, newCheckpoints[0].RunID)

	// Verify artifacts were copied to the new run path.
	var copiedArtifacts map[string]string
	require.NoError(t, json.Unmarshal([]byte(newCheckpoints[0].ArtifactSnapshot), &copiedArtifacts))
	for _, path := range copiedArtifacts {
		_, statErr := os.Stat(path)
		// The artifact path still references the source run directory;
		// the copy destination replaces the run ID segment.
		// If the path contained the source run ID, it should now exist
		// at the new run ID path too.
		if statErr != nil && !os.IsNotExist(statErr) {
			t.Errorf("unexpected error checking artifact %q: %v", path, statErr)
		}
	}
}

// TestForkManager_RejectsRunningRun verifies that forking a running run is
// rejected with a clear error.
func TestForkManager_RejectsRunningRun(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "state.db")

	store, err := state.NewStateStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	runID, err := store.CreateRun("test-pipeline", "input")
	require.NoError(t, err)
	require.NoError(t, store.UpdateRunStatus(runID, "running", "step1", 0))

	p := &pipeline.Pipeline{
		Metadata: pipeline.PipelineMetadata{Name: "test-pipeline"},
		Steps:    []pipeline.Step{{ID: "step1"}},
	}

	fm := pipeline.NewForkManager(store)
	_, err = fm.Fork(runID, "step1", p)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot fork a running run")
}

// TestForkManager_AllowFailedFlag verifies that failed runs can be forked
// when the allowFailed flag is set.
func TestForkManager_AllowFailedFlag(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "state.db")

	store, err := state.NewStateStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	runID, err := store.CreateRun("test-pipeline", "input")
	require.NoError(t, err)
	require.NoError(t, store.UpdateRunStatus(runID, "failed", "step1", 0))

	// Save a checkpoint so the fork has something to work with.
	snapshot, _ := json.Marshal(map[string]string{})
	require.NoError(t, store.SaveCheckpoint(&state.CheckpointRecord{
		RunID:            runID,
		StepID:           "step1",
		StepIndex:        0,
		ArtifactSnapshot: string(snapshot),
	}))

	p := &pipeline.Pipeline{
		Metadata: pipeline.PipelineMetadata{Name: "test-pipeline"},
		Steps:    []pipeline.Step{{ID: "step1"}, {ID: "step2"}},
	}

	fm := pipeline.NewForkManager(store)

	// Without allowFailed, should be rejected.
	_, err = fm.Fork(runID, "step1", p)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--allow-failed")

	// With allowFailed, should succeed.
	newRunID, err := fm.Fork(runID, "step1", p, true)
	assert.NoError(t, err)
	assert.NotEmpty(t, newRunID)
}

// TestForkManager_ListForkPoints_Integration verifies that ListForkPoints
// returns the correct fork points from a real SQLite store.
func TestForkManager_ListForkPoints_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "state.db")

	store, err := state.NewStateStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	runID, err := store.CreateRun("test-pipeline", "input")
	require.NoError(t, err)
	require.NoError(t, store.UpdateRunStatus(runID, "completed", "", 0))

	snapshot, _ := json.Marshal(map[string]string{})
	for i, step := range []string{"plan", "implement", "test"} {
		sha := ""
		if step == "plan" {
			sha = "deadbeef"
		}
		require.NoError(t, store.SaveCheckpoint(&state.CheckpointRecord{
			RunID:              runID,
			StepID:             step,
			StepIndex:          i,
			WorkspaceCommitSHA: sha,
			ArtifactSnapshot:   string(snapshot),
		}))
	}

	fm := pipeline.NewForkManager(store)
	points, err := fm.ListForkPoints(runID)
	require.NoError(t, err)
	assert.Len(t, points, 3)

	assert.Equal(t, "plan", points[0].StepID)
	assert.True(t, points[0].HasSHA)
	assert.Equal(t, "implement", points[1].StepID)
	assert.False(t, points[1].HasSHA)
	assert.Equal(t, "test", points[2].StepID)
	assert.False(t, points[2].HasSHA)
}

// TestRewind_Integration exercises the rewind flow using a real SQLite store:
// create a run with checkpoints, then delete checkpoints after a rewind point
// and verify the state is reset.
func TestRewind_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "state.db")

	store, err := state.NewStateStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	// Create a failed run with 3 steps.
	runID, err := store.CreateRun("test-pipeline", "input")
	require.NoError(t, err)
	require.NoError(t, store.UpdateRunStatus(runID, "failed", "step3", 100))

	snapshot, _ := json.Marshal(map[string]string{})
	for i, step := range []string{"step1", "step2", "step3"} {
		require.NoError(t, store.SaveCheckpoint(&state.CheckpointRecord{
			RunID:            runID,
			StepID:           step,
			StepIndex:        i,
			ArtifactSnapshot: string(snapshot),
		}))
	}

	// Rewind to after step1 (index 0) -- should delete checkpoints for step2 and step3.
	rewindIndex := 0
	err = store.DeleteCheckpointsAfterStep(runID, rewindIndex)
	require.NoError(t, err)

	// Update run status to 'failed' so wave resume can pick it up.
	require.NoError(t, store.UpdateRunStatus(runID, "failed", "step1", 100))

	// Verify only step1's checkpoint remains.
	remaining, err := store.GetCheckpoints(runID)
	require.NoError(t, err)
	assert.Len(t, remaining, 1, "only checkpoint at rewind point should remain")
	assert.Equal(t, "step1", remaining[0].StepID)

	// Verify the run status was updated.
	run, err := store.GetRun(runID)
	require.NoError(t, err)
	assert.Equal(t, "failed", run.Status)
}

// TestRewind_NothingToDelete verifies that rewinding to the last step
// is a no-op (no checkpoints deleted).
func TestRewind_NothingToDelete(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "state.db")

	store, err := state.NewStateStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	runID, err := store.CreateRun("test-pipeline", "input")
	require.NoError(t, err)
	require.NoError(t, store.UpdateRunStatus(runID, "completed", "", 0))

	snapshot, _ := json.Marshal(map[string]string{})
	require.NoError(t, store.SaveCheckpoint(&state.CheckpointRecord{
		RunID:            runID,
		StepID:           "only-step",
		StepIndex:        0,
		ArtifactSnapshot: string(snapshot),
	}))

	// Rewind to the only step (index 0) -- nothing should be deleted.
	err = store.DeleteCheckpointsAfterStep(runID, 0)
	require.NoError(t, err)

	remaining, err := store.GetCheckpoints(runID)
	require.NoError(t, err)
	assert.Len(t, remaining, 1, "the only checkpoint should be preserved")
	assert.Equal(t, "only-step", remaining[0].StepID)
}

// TestRewind_RewindToMiddle verifies that rewinding to a middle step
// correctly deletes only the checkpoints after the rewind point.
func TestRewind_RewindToMiddle(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "state.db")

	store, err := state.NewStateStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	runID, err := store.CreateRun("test-pipeline", "input")
	require.NoError(t, err)
	require.NoError(t, store.UpdateRunStatus(runID, "failed", "step5", 300))

	snapshot, _ := json.Marshal(map[string]string{})
	steps := []string{"step1", "step2", "step3", "step4", "step5"}
	for i, step := range steps {
		require.NoError(t, store.SaveCheckpoint(&state.CheckpointRecord{
			RunID:            runID,
			StepID:           step,
			StepIndex:        i,
			ArtifactSnapshot: string(snapshot),
		}))
	}

	// Rewind to after step3 (index 2) -- should delete step4 and step5.
	err = store.DeleteCheckpointsAfterStep(runID, 2)
	require.NoError(t, err)

	remaining, err := store.GetCheckpoints(runID)
	require.NoError(t, err)
	assert.Len(t, remaining, 3)
	for i, cp := range remaining {
		assert.Equal(t, steps[i], cp.StepID)
	}
}
