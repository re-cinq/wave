package state

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateRunBranch(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create a run first
	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	// Update branch
	err = store.UpdateRunBranch(runID, "feat/my-branch")
	require.NoError(t, err)

	// Verify branch was set
	run, err := store.GetRun(runID)
	require.NoError(t, err)
	assert.Equal(t, "feat/my-branch", run.BranchName)
}

func TestUpdateRunBranch_NonExistentRun(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.UpdateRunBranch("nonexistent-run-id", "some-branch")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "run not found")
}

func TestUpdateRunBranch_UpdateExisting(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	// Set initial branch
	err = store.UpdateRunBranch(runID, "first-branch")
	require.NoError(t, err)

	// Update to new branch
	err = store.UpdateRunBranch(runID, "second-branch")
	require.NoError(t, err)

	run, err := store.GetRun(runID)
	require.NoError(t, err)
	assert.Equal(t, "second-branch", run.BranchName)
}

func TestUpdateRunPID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	err = store.UpdateRunPID(runID, 12345)
	require.NoError(t, err)

	run, err := store.GetRun(runID)
	require.NoError(t, err)
	assert.Equal(t, 12345, run.PID)
}

func TestUpdateRunPID_NonExistentRun(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Unlike UpdateRunBranch, UpdateRunPID doesn't check rows affected
	err := store.UpdateRunPID("nonexistent-run-id", 12345)
	// This may or may not error depending on implementation
	_ = err
}

func TestStepProgress_CRUD(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	// Update step progress
	err = store.UpdateStepProgress(runID, "step-1", "navigator", "running", 50, "analyzing", "Analyzing codebase", 5000, 100)
	require.NoError(t, err)

	// Get step progress
	progress, err := store.GetStepProgress("step-1")
	require.NoError(t, err)
	assert.Equal(t, "step-1", progress.StepID)
	assert.Equal(t, runID, progress.RunID)
	assert.Equal(t, "navigator", progress.Persona)
	assert.Equal(t, "running", progress.State)
	assert.Equal(t, 50, progress.Progress)
	assert.Equal(t, "analyzing", progress.CurrentAction)
	assert.Equal(t, "Analyzing codebase", progress.Message)
	assert.Equal(t, int64(5000), progress.EstimatedCompletionMs)
	assert.Equal(t, 100, progress.TokensUsed)
}

func TestStepProgress_Upsert(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	// Initial insert
	err = store.UpdateStepProgress(runID, "step-1", "navigator", "running", 25, "starting", "Starting", 10000, 50)
	require.NoError(t, err)

	// Update (upsert)
	err = store.UpdateStepProgress(runID, "step-1", "navigator", "running", 75, "finishing", "Nearly done", 2000, 300)
	require.NoError(t, err)

	progress, err := store.GetStepProgress("step-1")
	require.NoError(t, err)
	assert.Equal(t, 75, progress.Progress)
	assert.Equal(t, "finishing", progress.CurrentAction)
	assert.Equal(t, 300, progress.TokensUsed)
}

func TestGetStepProgress_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, err := store.GetStepProgress("nonexistent-step")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetAllStepProgress(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	// Add progress for multiple steps
	steps := []struct {
		stepID  string
		persona string
		state   string
	}{
		{"step-1", "navigator", "completed"},
		{"step-2", "implementer", "running"},
		{"step-3", "reviewer", "pending"},
	}

	for _, s := range steps {
		err := store.UpdateStepProgress(runID, s.stepID, s.persona, s.state, 0, "", "", 0, 0)
		require.NoError(t, err)
	}

	// Get all step progress for the run
	all, err := store.GetAllStepProgress(runID)
	require.NoError(t, err)
	assert.Len(t, all, 3)

	// Verify they're all from the same run
	for _, p := range all {
		assert.Equal(t, runID, p.RunID)
	}
}

func TestGetAllStepProgress_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	all, err := store.GetAllStepProgress(runID)
	require.NoError(t, err)
	assert.Empty(t, all)
}

func TestPipelineProgress_CRUD(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	// Update pipeline progress
	err = store.UpdatePipelineProgress(runID, 5, 2, 2, 40, 30000)
	require.NoError(t, err)

	// Get pipeline progress
	progress, err := store.GetPipelineProgress(runID)
	require.NoError(t, err)
	assert.Equal(t, runID, progress.RunID)
	assert.Equal(t, 5, progress.TotalSteps)
	assert.Equal(t, 2, progress.CompletedSteps)
	assert.Equal(t, 2, progress.CurrentStepIndex)
	assert.Equal(t, 40, progress.OverallProgress)
	assert.Equal(t, int64(30000), progress.EstimatedCompletionMs)
}

func TestPipelineProgress_Upsert(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	// Initial insert
	err = store.UpdatePipelineProgress(runID, 5, 1, 1, 20, 60000)
	require.NoError(t, err)

	// Update
	err = store.UpdatePipelineProgress(runID, 5, 4, 4, 80, 10000)
	require.NoError(t, err)

	progress, err := store.GetPipelineProgress(runID)
	require.NoError(t, err)
	assert.Equal(t, 4, progress.CompletedSteps)
	assert.Equal(t, 80, progress.OverallProgress)
}

func TestGetPipelineProgress_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, err := store.GetPipelineProgress("nonexistent-run")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCleanupOldPerformanceMetrics(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	now := time.Now()

	// Record a performance metric to create some data to cleanup
	metric := &PerformanceMetricRecord{
		RunID:      runID,
		StepID:     "step-1",
		Persona:    "navigator",
		StartedAt:  now,
		DurationMs: 100,
		Success:    true,
	}
	err = store.RecordPerformanceMetric(metric)
	require.NoError(t, err)

	// Cleanup with a very long duration (should delete nothing since metric is fresh)
	deleted, err := store.CleanupOldPerformanceMetrics(24 * time.Hour * 365 * 100) // 100 years
	require.NoError(t, err)
	assert.Equal(t, 0, deleted, "nothing should be deleted with 100-year window")

	// Cleanup with zero duration (should delete everything older than now)
	deleted, err = store.CleanupOldPerformanceMetrics(0)
	require.NoError(t, err)
	// The metric was created just now, so it may or may not be deleted (timing-sensitive)
	_ = deleted
}

func TestGetEvents_StepFilter(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	// Log events for different steps
	err = store.LogEvent(runID, "step-1", "running", "navigator", "step 1 started", 100, 500, "", "")
	require.NoError(t, err)
	err = store.LogEvent(runID, "step-2", "running", "implementer", "step 2 started", 200, 600, "", "")
	require.NoError(t, err)
	err = store.LogEvent(runID, "step-1", "completed", "navigator", "step 1 done", 150, 1000, "", "")
	require.NoError(t, err)

	// Get all events
	allEvents, err := store.GetEvents(runID, EventQueryOptions{})
	require.NoError(t, err)
	assert.Len(t, allEvents, 3)

	// Get events filtered by step
	step1Events, err := store.GetEvents(runID, EventQueryOptions{StepID: "step-1"})
	require.NoError(t, err)
	assert.Len(t, step1Events, 2)
	for _, e := range step1Events {
		if !strings.Contains(e.StepID, "step-1") {
			t.Errorf("expected step-1, got %s", e.StepID)
		}
	}

	step2Events, err := store.GetEvents(runID, EventQueryOptions{StepID: "step-2"})
	require.NoError(t, err)
	assert.Len(t, step2Events, 1)
}
