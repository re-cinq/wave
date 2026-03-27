package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVisitCount_SaveAndGet(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	pipelineID := "test-pipeline-vc"
	stepID := "step-1"

	// Save pipeline state (required for foreign key)
	err := store.SavePipelineState(pipelineID, "running", "test input")
	require.NoError(t, err)

	// Save step state (creates the step row)
	err = store.SaveStepState(pipelineID, stepID, StateRunning, "")
	require.NoError(t, err)

	// Save visit count
	err = store.SaveStepVisitCount(pipelineID, stepID, 3)
	require.NoError(t, err)

	// Retrieve visit count
	count, err := store.GetStepVisitCount(pipelineID, stepID)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestVisitCount_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Get visit count for a step that does not exist — should return 0
	count, err := store.GetStepVisitCount("nonexistent-pipeline", "nonexistent-step")
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestVisitCount_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	pipelineID := "test-pipeline-update"
	stepID := "step-loop"

	// Save pipeline and step state
	err := store.SavePipelineState(pipelineID, "running", "")
	require.NoError(t, err)
	err = store.SaveStepState(pipelineID, stepID, StateRunning, "")
	require.NoError(t, err)

	// Save initial visit count
	err = store.SaveStepVisitCount(pipelineID, stepID, 1)
	require.NoError(t, err)

	count, err := store.GetStepVisitCount(pipelineID, stepID)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Update visit count to a higher value
	err = store.SaveStepVisitCount(pipelineID, stepID, 5)
	require.NoError(t, err)

	count, err = store.GetStepVisitCount(pipelineID, stepID)
	require.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestVisitCount_IncludedInStepStates(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	pipelineID := "test-pipeline-states"
	stepID := "step-a"

	// Save pipeline and step state
	err := store.SavePipelineState(pipelineID, "running", "")
	require.NoError(t, err)
	err = store.SaveStepState(pipelineID, stepID, StateCompleted, "")
	require.NoError(t, err)

	// Save a visit count
	err = store.SaveStepVisitCount(pipelineID, stepID, 7)
	require.NoError(t, err)

	// Retrieve via GetStepStates and verify VisitCount is populated
	states, err := store.GetStepStates(pipelineID)
	require.NoError(t, err)
	require.Len(t, states, 1)
	assert.Equal(t, stepID, states[0].StepID)
	assert.Equal(t, pipelineID, states[0].PipelineID)
	assert.Equal(t, StepState("completed"), states[0].State)
	assert.Equal(t, 7, states[0].VisitCount)
}

func TestVisitCount_SaveCreatesStepIfMissing(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	pipelineID := "test-pipeline-auto"
	stepID := "step-new"

	// Save pipeline state (required for foreign key)
	err := store.SavePipelineState(pipelineID, "running", "")
	require.NoError(t, err)

	// Do NOT create step state first — SaveStepVisitCount should auto-insert
	err = store.SaveStepVisitCount(pipelineID, stepID, 2)
	require.NoError(t, err)

	count, err := store.GetStepVisitCount(pipelineID, stepID)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// Verify the step row was created with pending state
	states, err := store.GetStepStates(pipelineID)
	require.NoError(t, err)
	require.Len(t, states, 1)
	assert.Equal(t, StatePending, states[0].State)
	assert.Equal(t, 2, states[0].VisitCount)
}
