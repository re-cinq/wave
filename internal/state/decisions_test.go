package state

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordDecision(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create a run first (decisions have FK to pipeline_run)
	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	record := &DecisionRecord{
		RunID:     runID,
		StepID:    "investigate",
		Timestamp: time.Now(),
		Category:  "model_routing",
		Decision:  "selected model claude-opus for step investigate",
		Rationale: "per-persona model configuration",
		Context:   `{"model":"claude-opus","persona":"investigator"}`,
	}

	err = store.RecordDecision(record)
	require.NoError(t, err)
	assert.NotZero(t, record.ID, "ID should be set after insert")
}

func TestRecordDecision_DefaultTimestamp(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	record := &DecisionRecord{
		RunID:    runID,
		StepID:   "plan",
		Category: "retry",
		Decision: "retrying step plan",
	}

	err = store.RecordDecision(record)
	require.NoError(t, err)
	assert.NotZero(t, record.ID)
}

func TestRecordDecision_EmptyContext(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	record := &DecisionRecord{
		RunID:     runID,
		StepID:    "implement",
		Timestamp: time.Now(),
		Category:  "contract",
		Decision:  "contract passed",
		Rationale: "test_suite validated successfully",
	}

	err = store.RecordDecision(record)
	require.NoError(t, err)

	// Retrieve and verify context defaults to "{}"
	decisions, err := store.GetDecisions(runID)
	require.NoError(t, err)
	require.Len(t, decisions, 1)
	assert.Equal(t, "{}", decisions[0].Context)
}

func TestGetDecisions(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	now := time.Now()

	// Insert multiple decisions
	records := []*DecisionRecord{
		{
			RunID:     runID,
			StepID:    "investigate",
			Timestamp: now.Add(-3 * time.Minute),
			Category:  "model_routing",
			Decision:  "selected model A",
			Rationale: "per-step pinning",
			Context:   `{"model":"A"}`,
		},
		{
			RunID:     runID,
			StepID:    "plan",
			Timestamp: now.Add(-2 * time.Minute),
			Category:  "model_routing",
			Decision:  "selected model B",
			Rationale: "per-persona config",
			Context:   `{"model":"B"}`,
		},
		{
			RunID:     runID,
			StepID:    "implement",
			Timestamp: now.Add(-1 * time.Minute),
			Category:  "contract",
			Decision:  "contract passed",
			Rationale: "test_suite OK",
		},
	}

	for _, r := range records {
		err := store.RecordDecision(r)
		require.NoError(t, err)
	}

	decisions, err := store.GetDecisions(runID)
	require.NoError(t, err)
	require.Len(t, decisions, 3)

	// Verify ordering (by timestamp ASC)
	assert.Equal(t, "selected model A", decisions[0].Decision)
	assert.Equal(t, "selected model B", decisions[1].Decision)
	assert.Equal(t, "contract passed", decisions[2].Decision)

	// Verify fields are populated
	assert.Equal(t, "investigate", decisions[0].StepID)
	assert.Equal(t, "model_routing", decisions[0].Category)
	assert.Equal(t, "per-step pinning", decisions[0].Rationale)
	assert.Equal(t, `{"model":"A"}`, decisions[0].Context)
}

func TestGetDecisions_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	decisions, err := store.GetDecisions(runID)
	require.NoError(t, err)
	assert.Empty(t, decisions)
}

func TestGetDecisions_IsolatedByRunID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID1, err := store.CreateRun("pipeline-a", "input a")
	require.NoError(t, err)
	runID2, err := store.CreateRun("pipeline-b", "input b")
	require.NoError(t, err)

	err = store.RecordDecision(&DecisionRecord{
		RunID:     runID1,
		StepID:    "step1",
		Timestamp: time.Now(),
		Category:  "model_routing",
		Decision:  "run1 decision",
	})
	require.NoError(t, err)

	err = store.RecordDecision(&DecisionRecord{
		RunID:     runID2,
		StepID:    "step1",
		Timestamp: time.Now(),
		Category:  "model_routing",
		Decision:  "run2 decision",
	})
	require.NoError(t, err)

	decisions1, err := store.GetDecisions(runID1)
	require.NoError(t, err)
	require.Len(t, decisions1, 1)
	assert.Equal(t, "run1 decision", decisions1[0].Decision)

	decisions2, err := store.GetDecisions(runID2)
	require.NoError(t, err)
	require.Len(t, decisions2, 1)
	assert.Equal(t, "run2 decision", decisions2[0].Decision)
}

func TestGetDecisionsByStep(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	now := time.Now()

	// Insert decisions for different steps
	for _, step := range []string{"investigate", "plan", "implement", "investigate"} {
		err := store.RecordDecision(&DecisionRecord{
			RunID:     runID,
			StepID:    step,
			Timestamp: now,
			Category:  "model_routing",
			Decision:  "decision for " + step,
		})
		require.NoError(t, err)
		now = now.Add(time.Second)
	}

	// Filter by step
	decisions, err := store.GetDecisionsByStep(runID, "investigate")
	require.NoError(t, err)
	require.Len(t, decisions, 2)
	assert.Equal(t, "decision for investigate", decisions[0].Decision)
	assert.Equal(t, "decision for investigate", decisions[1].Decision)

	// Filter by another step
	decisions, err = store.GetDecisionsByStep(runID, "plan")
	require.NoError(t, err)
	require.Len(t, decisions, 1)
	assert.Equal(t, "decision for plan", decisions[0].Decision)
}

func TestGetDecisionsByStep_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	decisions, err := store.GetDecisionsByStep(runID, "nonexistent")
	require.NoError(t, err)
	assert.Empty(t, decisions)
}

func TestRecordDecision_AllCategories(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	categories := []string{"model_routing", "retry", "contract", "budget", "composition"}
	for _, cat := range categories {
		err := store.RecordDecision(&DecisionRecord{
			RunID:     runID,
			StepID:    "step1",
			Timestamp: time.Now(),
			Category:  cat,
			Decision:  cat + " decision",
		})
		require.NoError(t, err, "should record %s decision", cat)
	}

	decisions, err := store.GetDecisions(runID)
	require.NoError(t, err)
	assert.Len(t, decisions, 5)
}

func TestRecordDecision_AppendOnly(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	runID, err := store.CreateRun("test-pipeline", "test input")
	require.NoError(t, err)

	// Record multiple decisions for the same step — should all be preserved
	for i := 0; i < 5; i++ {
		err := store.RecordDecision(&DecisionRecord{
			RunID:     runID,
			StepID:    "step1",
			Timestamp: time.Now(),
			Category:  "retry",
			Decision:  "attempt decision",
		})
		require.NoError(t, err)
	}

	decisions, err := store.GetDecisions(runID)
	require.NoError(t, err)
	assert.Len(t, decisions, 5, "all decisions should be preserved (append-only)")
}
