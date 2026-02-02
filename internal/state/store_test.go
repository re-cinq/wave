package state

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestStore creates a new in-memory StateStore for testing.
// Returns the store and a cleanup function.
func setupTestStore(t *testing.T) (StateStore, func()) {
	t.Helper()

	store, err := NewStateStore(":memory:")
	require.NoError(t, err, "failed to create test store")

	cleanup := func() {
		if err := store.Close(); err != nil {
			t.Errorf("failed to close test store: %v", err)
		}
	}

	return store, cleanup
}

// setupTestStoreWithFile creates a file-based StateStore for concurrent testing.
// SQLite in-memory databases don't support true concurrent access across connections.
// Returns the store and a cleanup function.
func setupTestStoreWithFile(t *testing.T) (StateStore, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "wave-state-test-*")
	require.NoError(t, err, "failed to create temp dir")

	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := NewStateStore(dbPath)
	require.NoError(t, err, "failed to create test store")

	cleanup := func() {
		if err := store.Close(); err != nil {
			t.Errorf("failed to close test store: %v", err)
		}
		os.RemoveAll(tmpDir)
	}

	return store, cleanup
}

// TestNewStateStore verifies that creating a new store works correctly.
func TestNewStateStore(t *testing.T) {
	t.Run("creates in-memory store successfully", func(t *testing.T) {
		store, err := NewStateStore(":memory:")
		require.NoError(t, err)
		assert.NotNil(t, store)

		err = store.Close()
		assert.NoError(t, err)
	})

	t.Run("fails with invalid path", func(t *testing.T) {
		// Try to create a store in a non-existent directory
		store, err := NewStateStore("/nonexistent/path/to/db.sqlite")
		assert.Error(t, err)
		assert.Nil(t, store)
	})
}

// TestSavePipelineState tests the SavePipelineState method.
func TestSavePipelineState(t *testing.T) {
	testCases := []struct {
		name   string
		id     string
		status string
		input  string
	}{
		{
			name:   "save new pipeline with running status",
			id:     "pipeline-001",
			status: "running",
			input:  `{"key": "value"}`,
		},
		{
			name:   "save pipeline with completed status",
			id:     "pipeline-002",
			status: "completed",
			input:  `{"foo": "bar"}`,
		},
		{
			name:   "save pipeline with empty input",
			id:     "pipeline-003",
			status: "pending",
			input:  "",
		},
		{
			name:   "save pipeline with failed status and error details",
			id:     "pipeline-004",
			status: "failed",
			input:  `{"error": "something went wrong"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store, cleanup := setupTestStore(t)
			defer cleanup()

			err := store.SavePipelineState(tc.id, tc.status, tc.input)
			assert.NoError(t, err)

			// Verify the state was saved correctly
			record, err := store.GetPipelineState(tc.id)
			require.NoError(t, err)
			assert.Equal(t, tc.id, record.PipelineID)
			assert.Equal(t, tc.status, record.Status)
			assert.Equal(t, tc.input, record.Input)
			assert.False(t, record.CreatedAt.IsZero())
			assert.False(t, record.UpdatedAt.IsZero())
		})
	}

	t.Run("update existing pipeline state", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		pipelineID := "pipeline-update-test"

		// Save initial state
		err := store.SavePipelineState(pipelineID, "pending", `{"initial": true}`)
		require.NoError(t, err)

		initial, err := store.GetPipelineState(pipelineID)
		require.NoError(t, err)

		// Update to running
		err = store.SavePipelineState(pipelineID, "running", `{"initial": false}`)
		require.NoError(t, err)

		updated, err := store.GetPipelineState(pipelineID)
		require.NoError(t, err)

		assert.Equal(t, "running", updated.Status)
		assert.Equal(t, `{"initial": false}`, updated.Input)
		assert.Equal(t, initial.CreatedAt, updated.CreatedAt, "created_at should not change on update")
		assert.True(t, updated.UpdatedAt.After(initial.CreatedAt) || updated.UpdatedAt.Equal(initial.UpdatedAt),
			"updated_at should be >= created_at")
	})
}

// TestGetPipelineState tests the GetPipelineState method.
func TestGetPipelineState(t *testing.T) {
	t.Run("get existing pipeline state", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		pipelineID := "test-pipeline-get"
		status := "completed"
		input := `{"data": "test"}`

		err := store.SavePipelineState(pipelineID, status, input)
		require.NoError(t, err)

		record, err := store.GetPipelineState(pipelineID)
		require.NoError(t, err)
		assert.NotNil(t, record)
		assert.Equal(t, pipelineID, record.PipelineID)
		assert.Equal(t, pipelineID, record.Name)
		assert.Equal(t, status, record.Status)
		assert.Equal(t, input, record.Input)
	})

	t.Run("get non-existent pipeline returns error", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		record, err := store.GetPipelineState("nonexistent-id")
		assert.Error(t, err)
		assert.Nil(t, record)
		assert.Contains(t, err.Error(), "pipeline state not found")
	})

	t.Run("get pipeline with special characters in ID", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		pipelineID := "pipeline-with-special-chars_123-abc"
		err := store.SavePipelineState(pipelineID, "running", "")
		require.NoError(t, err)

		record, err := store.GetPipelineState(pipelineID)
		require.NoError(t, err)
		assert.Equal(t, pipelineID, record.PipelineID)
	})
}

// TestSaveStepState tests the SaveStepState method.
func TestSaveStepState(t *testing.T) {
	testCases := []struct {
		name       string
		pipelineID string
		stepID     string
		state      StepState
		errMsg     string
	}{
		{
			name:       "save step with pending state",
			pipelineID: "pipeline-1",
			stepID:     "step-1",
			state:      StatePending,
			errMsg:     "",
		},
		{
			name:       "save step with running state",
			pipelineID: "pipeline-1",
			stepID:     "step-2",
			state:      StateRunning,
			errMsg:     "",
		},
		{
			name:       "save step with completed state",
			pipelineID: "pipeline-1",
			stepID:     "step-3",
			state:      StateCompleted,
			errMsg:     "",
		},
		{
			name:       "save step with failed state and error message",
			pipelineID: "pipeline-1",
			stepID:     "step-4",
			state:      StateFailed,
			errMsg:     "execution timeout exceeded",
		},
		{
			name:       "save step with retrying state",
			pipelineID: "pipeline-1",
			stepID:     "step-5",
			state:      StateRetrying,
			errMsg:     "temporary failure, retrying",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store, cleanup := setupTestStore(t)
			defer cleanup()

			// First create the pipeline
			err := store.SavePipelineState(tc.pipelineID, "running", "")
			require.NoError(t, err)

			// Save the step state
			err = store.SaveStepState(tc.pipelineID, tc.stepID, tc.state, tc.errMsg)
			assert.NoError(t, err)

			// Verify by retrieving
			steps, err := store.GetStepStates(tc.pipelineID)
			require.NoError(t, err)
			require.Len(t, steps, 1)

			assert.Equal(t, tc.stepID, steps[0].StepID)
			assert.Equal(t, tc.pipelineID, steps[0].PipelineID)
			assert.Equal(t, tc.state, steps[0].State)
			assert.Equal(t, tc.errMsg, steps[0].ErrorMessage)
		})
	}

	t.Run("update step state preserves started_at", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		pipelineID := "pipeline-update"
		stepID := "step-update"

		err := store.SavePipelineState(pipelineID, "running", "")
		require.NoError(t, err)

		// Start running
		err = store.SaveStepState(pipelineID, stepID, StateRunning, "")
		require.NoError(t, err)

		steps, err := store.GetStepStates(pipelineID)
		require.NoError(t, err)
		require.Len(t, steps, 1)
		initialStartedAt := steps[0].StartedAt
		assert.NotNil(t, initialStartedAt, "started_at should be set for running state")

		// Complete the step
		err = store.SaveStepState(pipelineID, stepID, StateCompleted, "")
		require.NoError(t, err)

		steps, err = store.GetStepStates(pipelineID)
		require.NoError(t, err)
		require.Len(t, steps, 1)

		assert.Equal(t, StateCompleted, steps[0].State)
		assert.NotNil(t, steps[0].StartedAt, "started_at should still be set")
		assert.NotNil(t, steps[0].CompletedAt, "completed_at should be set for completed state")
	})
}

// TestGetStepStates tests the GetStepStates method.
func TestGetStepStates(t *testing.T) {
	t.Run("get steps for pipeline with multiple steps", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		pipelineID := "pipeline-multi-step"

		err := store.SavePipelineState(pipelineID, "running", "")
		require.NoError(t, err)

		// Save multiple steps
		steps := []struct {
			stepID string
			state  StepState
		}{
			{"step-a", StateCompleted},
			{"step-b", StateRunning},
			{"step-c", StatePending},
		}

		for _, s := range steps {
			err := store.SaveStepState(pipelineID, s.stepID, s.state, "")
			require.NoError(t, err)
		}

		// Retrieve all steps
		retrieved, err := store.GetStepStates(pipelineID)
		require.NoError(t, err)
		assert.Len(t, retrieved, 3)

		// Steps should be ordered by step_id
		assert.Equal(t, "step-a", retrieved[0].StepID)
		assert.Equal(t, "step-b", retrieved[1].StepID)
		assert.Equal(t, "step-c", retrieved[2].StepID)
	})

	t.Run("get steps for empty pipeline returns empty slice", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		pipelineID := "pipeline-no-steps"

		err := store.SavePipelineState(pipelineID, "running", "")
		require.NoError(t, err)

		steps, err := store.GetStepStates(pipelineID)
		require.NoError(t, err)
		assert.Empty(t, steps)
	})

	t.Run("get steps for non-existent pipeline returns empty slice", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		steps, err := store.GetStepStates("nonexistent-pipeline")
		require.NoError(t, err)
		assert.Empty(t, steps)
	})

	t.Run("get steps only returns steps for specified pipeline", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		// Create two pipelines
		err := store.SavePipelineState("pipeline-1", "running", "")
		require.NoError(t, err)
		err = store.SavePipelineState("pipeline-2", "running", "")
		require.NoError(t, err)

		// Add steps to both
		err = store.SaveStepState("pipeline-1", "step-p1-1", StateRunning, "")
		require.NoError(t, err)
		err = store.SaveStepState("pipeline-1", "step-p1-2", StateRunning, "")
		require.NoError(t, err)
		err = store.SaveStepState("pipeline-2", "step-p2-1", StateRunning, "")
		require.NoError(t, err)

		// Get steps for pipeline-1
		steps, err := store.GetStepStates("pipeline-1")
		require.NoError(t, err)
		assert.Len(t, steps, 2)

		for _, s := range steps {
			assert.Equal(t, "pipeline-1", s.PipelineID)
		}
	})
}

// TestConcurrentAccess tests concurrent access from multiple goroutines
// simulating matrix workers accessing the state store simultaneously.
func TestConcurrentAccess(t *testing.T) {
	t.Run("concurrent pipeline state updates", func(t *testing.T) {
		store, cleanup := setupTestStoreWithFile(t)
		defer cleanup()

		numWorkers := 10
		numUpdates := 50
		var wg sync.WaitGroup

		// Create pipelines first
		for i := 0; i < numWorkers; i++ {
			pipelineID := pipelineIDFromIndex(i)
			err := store.SavePipelineState(pipelineID, "pending", "")
			require.NoError(t, err)
		}

		// Launch concurrent workers
		wg.Add(numWorkers)
		for i := 0; i < numWorkers; i++ {
			go func(workerID int) {
				defer wg.Done()
				pipelineID := pipelineIDFromIndex(workerID)

				for j := 0; j < numUpdates; j++ {
					status := "running"
					if j == numUpdates-1 {
						status = "completed"
					}
					err := store.SavePipelineState(pipelineID, status, "")
					assert.NoError(t, err)
				}
			}(i)
		}

		wg.Wait()

		// Verify all pipelines have final state
		for i := 0; i < numWorkers; i++ {
			pipelineID := pipelineIDFromIndex(i)
			record, err := store.GetPipelineState(pipelineID)
			require.NoError(t, err)
			assert.Equal(t, "completed", record.Status)
		}
	})

	t.Run("concurrent step state updates", func(t *testing.T) {
		store, cleanup := setupTestStoreWithFile(t)
		defer cleanup()

		pipelineID := "concurrent-pipeline"
		numWorkers := 10
		var wg sync.WaitGroup

		err := store.SavePipelineState(pipelineID, "running", "")
		require.NoError(t, err)

		// Each worker updates a different step
		wg.Add(numWorkers)
		for i := 0; i < numWorkers; i++ {
			go func(workerID int) {
				defer wg.Done()
				stepID := stepIDFromIndex(workerID)

				// Simulate step lifecycle
				states := []StepState{StatePending, StateRunning, StateCompleted}
				for _, state := range states {
					err := store.SaveStepState(pipelineID, stepID, state, "")
					assert.NoError(t, err)
				}
			}(i)
		}

		wg.Wait()

		// Verify all steps completed
		steps, err := store.GetStepStates(pipelineID)
		require.NoError(t, err)
		assert.Len(t, steps, numWorkers)

		for _, step := range steps {
			assert.Equal(t, StateCompleted, step.State)
		}
	})

	t.Run("concurrent reads and writes", func(t *testing.T) {
		store, cleanup := setupTestStoreWithFile(t)
		defer cleanup()

		pipelineID := "read-write-pipeline"
		err := store.SavePipelineState(pipelineID, "running", "")
		require.NoError(t, err)

		numWriters := 5
		numReaders := 5
		numOperations := 20
		var wg sync.WaitGroup

		// Writers update step states
		wg.Add(numWriters)
		for i := 0; i < numWriters; i++ {
			go func(writerID int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					stepID := stepIDFromIndices(writerID, j)
					err := store.SaveStepState(pipelineID, stepID, StateRunning, "")
					assert.NoError(t, err)
				}
			}(i)
		}

		// Readers query step states
		wg.Add(numReaders)
		for i := 0; i < numReaders; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					_, err := store.GetStepStates(pipelineID)
					assert.NoError(t, err)
				}
			}()
		}

		wg.Wait()
	})
}

// TestStateTransitions tests valid state transitions for steps.
func TestStateTransitions(t *testing.T) {
	t.Run("pending to running to completed", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		pipelineID := "transition-test-1"
		stepID := "step-1"

		err := store.SavePipelineState(pipelineID, "running", "")
		require.NoError(t, err)

		// Pending
		err = store.SaveStepState(pipelineID, stepID, StatePending, "")
		require.NoError(t, err)
		verifyStepState(t, store, pipelineID, stepID, StatePending, 0)

		// Running
		err = store.SaveStepState(pipelineID, stepID, StateRunning, "")
		require.NoError(t, err)
		verifyStepState(t, store, pipelineID, stepID, StateRunning, 0)

		// Completed
		err = store.SaveStepState(pipelineID, stepID, StateCompleted, "")
		require.NoError(t, err)
		verifyStepState(t, store, pipelineID, stepID, StateCompleted, 0)
	})

	t.Run("pending to running to failed", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		pipelineID := "transition-test-2"
		stepID := "step-1"

		err := store.SavePipelineState(pipelineID, "running", "")
		require.NoError(t, err)

		// Pending -> Running -> Failed
		err = store.SaveStepState(pipelineID, stepID, StatePending, "")
		require.NoError(t, err)

		err = store.SaveStepState(pipelineID, stepID, StateRunning, "")
		require.NoError(t, err)

		err = store.SaveStepState(pipelineID, stepID, StateFailed, "process exited with code 1")
		require.NoError(t, err)

		verifyStepState(t, store, pipelineID, stepID, StateFailed, 0)

		steps, err := store.GetStepStates(pipelineID)
		require.NoError(t, err)
		require.Len(t, steps, 1)
		assert.Equal(t, "process exited with code 1", steps[0].ErrorMessage)
	})

	t.Run("running to retrying increments retry count", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		pipelineID := "transition-test-3"
		stepID := "step-1"

		err := store.SavePipelineState(pipelineID, "running", "")
		require.NoError(t, err)

		// Start running
		err = store.SaveStepState(pipelineID, stepID, StateRunning, "")
		require.NoError(t, err)
		verifyStepState(t, store, pipelineID, stepID, StateRunning, 0)

		// First retry
		err = store.SaveStepState(pipelineID, stepID, StateRetrying, "temporary failure")
		require.NoError(t, err)
		verifyStepState(t, store, pipelineID, stepID, StateRetrying, 1)

		// Second retry
		err = store.SaveStepState(pipelineID, stepID, StateRetrying, "temporary failure")
		require.NoError(t, err)
		verifyStepState(t, store, pipelineID, stepID, StateRetrying, 2)

		// Third retry
		err = store.SaveStepState(pipelineID, stepID, StateRetrying, "temporary failure")
		require.NoError(t, err)
		verifyStepState(t, store, pipelineID, stepID, StateRetrying, 3)

		// Finally complete
		err = store.SaveStepState(pipelineID, stepID, StateCompleted, "")
		require.NoError(t, err)
		verifyStepState(t, store, pipelineID, stepID, StateCompleted, 3)
	})

	t.Run("retrying to running to completed", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		pipelineID := "transition-test-4"
		stepID := "step-1"

		err := store.SavePipelineState(pipelineID, "running", "")
		require.NoError(t, err)

		// Initial run fails and retries
		err = store.SaveStepState(pipelineID, stepID, StateRunning, "")
		require.NoError(t, err)

		err = store.SaveStepState(pipelineID, stepID, StateRetrying, "network error")
		require.NoError(t, err)

		// Retry run
		err = store.SaveStepState(pipelineID, stepID, StateRunning, "")
		require.NoError(t, err)

		// Final completion
		err = store.SaveStepState(pipelineID, stepID, StateCompleted, "")
		require.NoError(t, err)

		verifyStepState(t, store, pipelineID, stepID, StateCompleted, 1)
	})

	t.Run("full lifecycle with timestamps", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		pipelineID := "transition-test-5"
		stepID := "step-1"

		err := store.SavePipelineState(pipelineID, "running", "")
		require.NoError(t, err)

		// Pending - no timestamps
		err = store.SaveStepState(pipelineID, stepID, StatePending, "")
		require.NoError(t, err)

		steps, err := store.GetStepStates(pipelineID)
		require.NoError(t, err)
		require.Len(t, steps, 1)
		assert.Nil(t, steps[0].StartedAt, "pending step should not have started_at")
		assert.Nil(t, steps[0].CompletedAt, "pending step should not have completed_at")

		// Running - started_at set
		err = store.SaveStepState(pipelineID, stepID, StateRunning, "")
		require.NoError(t, err)

		steps, err = store.GetStepStates(pipelineID)
		require.NoError(t, err)
		require.Len(t, steps, 1)
		assert.NotNil(t, steps[0].StartedAt, "running step should have started_at")
		assert.Nil(t, steps[0].CompletedAt, "running step should not have completed_at")

		startedAt := steps[0].StartedAt

		// Completed - completed_at set, started_at preserved
		err = store.SaveStepState(pipelineID, stepID, StateCompleted, "")
		require.NoError(t, err)

		steps, err = store.GetStepStates(pipelineID)
		require.NoError(t, err)
		require.Len(t, steps, 1)
		assert.NotNil(t, steps[0].StartedAt, "completed step should have started_at")
		assert.NotNil(t, steps[0].CompletedAt, "completed step should have completed_at")
		assert.Equal(t, startedAt, steps[0].StartedAt, "started_at should be preserved")
	})
}

// TestStoreClose tests the Close method.
func TestStoreClose(t *testing.T) {
	t.Run("close releases resources", func(t *testing.T) {
		store, err := NewStateStore(":memory:")
		require.NoError(t, err)

		err = store.Close()
		assert.NoError(t, err)
	})

	t.Run("operations after close fail", func(t *testing.T) {
		store, err := NewStateStore(":memory:")
		require.NoError(t, err)

		err = store.Close()
		require.NoError(t, err)

		// Attempt operations after close - should fail
		err = store.SavePipelineState("test", "running", "")
		assert.Error(t, err)
	})
}

// TestEdgeCases tests various edge cases.
func TestEdgeCases(t *testing.T) {
	t.Run("very long input data", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		longInput := make([]byte, 100000)
		for i := range longInput {
			longInput[i] = 'x'
		}

		err := store.SavePipelineState("long-input-pipeline", "running", string(longInput))
		require.NoError(t, err)

		record, err := store.GetPipelineState("long-input-pipeline")
		require.NoError(t, err)
		assert.Len(t, record.Input, 100000)
	})

	t.Run("unicode in pipeline data", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		unicodeInput := `{"message": "Hello, \u4e16\u754c! \u3053\u3093\u306b\u3061\u306f"}`
		err := store.SavePipelineState("unicode-pipeline", "running", unicodeInput)
		require.NoError(t, err)

		record, err := store.GetPipelineState("unicode-pipeline")
		require.NoError(t, err)
		assert.Equal(t, unicodeInput, record.Input)
	})

	t.Run("step error message with special characters", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		pipelineID := "special-error-pipeline"
		err := store.SavePipelineState(pipelineID, "running", "")
		require.NoError(t, err)

		specialError := "Error: file not found\n\tat line 42\n\t'quoted string'\n\t\"double quoted\""
		err = store.SaveStepState(pipelineID, "step-1", StateFailed, specialError)
		require.NoError(t, err)

		steps, err := store.GetStepStates(pipelineID)
		require.NoError(t, err)
		require.Len(t, steps, 1)
		assert.Equal(t, specialError, steps[0].ErrorMessage)
	})

	t.Run("multiple steps same state", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		pipelineID := "multi-same-state"
		err := store.SavePipelineState(pipelineID, "running", "")
		require.NoError(t, err)

		// Create multiple steps all in running state
		for i := 0; i < 5; i++ {
			stepID := stepIDFromIndex(i)
			err := store.SaveStepState(pipelineID, stepID, StateRunning, "")
			require.NoError(t, err)
		}

		steps, err := store.GetStepStates(pipelineID)
		require.NoError(t, err)
		assert.Len(t, steps, 5)

		for _, step := range steps {
			assert.Equal(t, StateRunning, step.State)
		}
	})
}

// TestListRecentPipelines tests the ListRecentPipelines method.
func TestListRecentPipelines(t *testing.T) {
	t.Run("returns empty list when no pipelines exist", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		records, err := store.ListRecentPipelines(10)
		require.NoError(t, err)
		assert.Empty(t, records)
	})

	t.Run("returns pipelines in order by updated_at DESC", func(t *testing.T) {
		store, cleanup := setupTestStoreWithFile(t)
		defer cleanup()

		// Create pipelines
		err := store.SavePipelineState("oldest", "completed", "")
		require.NoError(t, err)
		err = store.SavePipelineState("middle", "running", "")
		require.NoError(t, err)
		err = store.SavePipelineState("newest", "pending", "")
		require.NoError(t, err)

		// Update the oldest to make it the most recent
		time.Sleep(1100 * time.Millisecond) // Ensure different second
		err = store.SavePipelineState("oldest", "updated", "")
		require.NoError(t, err)

		records, err := store.ListRecentPipelines(10)
		require.NoError(t, err)
		require.Len(t, records, 3)

		// "oldest" was updated last, so should be first
		assert.Equal(t, "oldest", records[0].PipelineID)
		assert.Equal(t, "updated", records[0].Status)
	})

	t.Run("respects limit parameter", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		for i := 0; i < 10; i++ {
			pid := pipelineIDFromIndex(i)
			err := store.SavePipelineState(pid, "running", "")
			require.NoError(t, err)
		}

		// Request only 5
		records, err := store.ListRecentPipelines(5)
		require.NoError(t, err)
		assert.Len(t, records, 5)
	})

	t.Run("returns all fields correctly", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		inputJSON := `{"key": "value", "test": true}`
		err := store.SavePipelineState("test-pipeline", "running", inputJSON)
		require.NoError(t, err)

		records, err := store.ListRecentPipelines(1)
		require.NoError(t, err)
		require.Len(t, records, 1)

		record := records[0]
		assert.Equal(t, "test-pipeline", record.PipelineID)
		assert.Equal(t, "test-pipeline", record.Name)
		assert.Equal(t, "running", record.Status)
		assert.Equal(t, inputJSON, record.Input)
		assert.False(t, record.CreatedAt.IsZero())
		assert.False(t, record.UpdatedAt.IsZero())
	})

	t.Run("limit of zero returns empty list", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		err := store.SavePipelineState("test", "running", "")
		require.NoError(t, err)

		records, err := store.ListRecentPipelines(0)
		require.NoError(t, err)
		assert.Empty(t, records)
	})
}

// Helper functions

func pipelineIDFromIndex(i int) string {
	return "pipeline-" + string(rune('0'+i))
}

func stepIDFromIndex(i int) string {
	return "step-" + string(rune('0'+i))
}

func stepIDFromIndices(i, j int) string {
	return "step-" + string(rune('0'+i)) + "-" + string(rune('0'+j%10))
}

func verifyStepState(t *testing.T, store StateStore, pipelineID, stepID string, expectedState StepState, expectedRetryCount int) {
	t.Helper()

	steps, err := store.GetStepStates(pipelineID)
	require.NoError(t, err)

	var found *StepStateRecord
	for i := range steps {
		if steps[i].StepID == stepID {
			found = &steps[i]
			break
		}
	}

	require.NotNil(t, found, "step %s not found in pipeline %s", stepID, pipelineID)
	assert.Equal(t, expectedState, found.State, "unexpected state for step %s", stepID)
	assert.Equal(t, expectedRetryCount, found.RetryCount, "unexpected retry count for step %s", stepID)
}

// =============================================================================
// Ops Commands Tests (spec 016)
// =============================================================================

// TestCreateRun tests run creation and ID generation.
func TestCreateRun(t *testing.T) {
	testCases := []struct {
		name         string
		pipelineName string
		input        string
	}{
		{
			name:         "create run with input",
			pipelineName: "debug",
			input:        `{"task": "test"}`,
		},
		{
			name:         "create run without input",
			pipelineName: "build",
			input:        "",
		},
		{
			name:         "create run with special characters in pipeline name",
			pipelineName: "my-pipeline_v2",
			input:        "test input",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store, cleanup := setupTestStore(t)
			defer cleanup()

			runID, err := store.CreateRun(tc.pipelineName, tc.input)
			require.NoError(t, err)
			assert.NotEmpty(t, runID)
			assert.Contains(t, runID, tc.pipelineName)

			// Verify the run was created
			run, err := store.GetRun(runID)
			require.NoError(t, err)
			assert.Equal(t, runID, run.RunID)
			assert.Equal(t, tc.pipelineName, run.PipelineName)
			assert.Equal(t, "pending", run.Status)
			assert.Equal(t, tc.input, run.Input)
			assert.False(t, run.StartedAt.IsZero())
			assert.Nil(t, run.CompletedAt)
		})
	}

	t.Run("run ID format follows pattern", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		runID, err := store.CreateRun("debug", "")
		require.NoError(t, err)

		// Should match pattern: {pipeline}-{YYYYMMDD}-{HHMMSS}-{suffix}
		assert.Regexp(t, `^debug-\d{8}-\d{6}-\d{4}$`, runID)
	})
}

// TestUpdateRunStatus tests run status updates.
func TestUpdateRunStatus(t *testing.T) {
	testCases := []struct {
		name           string
		status         string
		currentStep    string
		tokens         int
		expectComplete bool
		expectCancel   bool
	}{
		{
			name:           "update to running",
			status:         "running",
			currentStep:    "step-1",
			tokens:         100,
			expectComplete: false,
		},
		{
			name:           "update to completed",
			status:         "completed",
			currentStep:    "step-final",
			tokens:         500,
			expectComplete: true,
		},
		{
			name:           "update to failed",
			status:         "failed",
			currentStep:    "step-error",
			tokens:         250,
			expectComplete: true,
		},
		{
			name:           "update to cancelled",
			status:         "cancelled",
			currentStep:    "step-mid",
			tokens:         150,
			expectComplete: true,
			expectCancel:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store, cleanup := setupTestStore(t)
			defer cleanup()

			runID, err := store.CreateRun("test", "")
			require.NoError(t, err)

			err = store.UpdateRunStatus(runID, tc.status, tc.currentStep, tc.tokens)
			require.NoError(t, err)

			run, err := store.GetRun(runID)
			require.NoError(t, err)

			assert.Equal(t, tc.status, run.Status)
			assert.Equal(t, tc.currentStep, run.CurrentStep)
			assert.Equal(t, tc.tokens, run.TotalTokens)

			if tc.expectComplete {
				assert.NotNil(t, run.CompletedAt)
			} else {
				assert.Nil(t, run.CompletedAt)
			}

			if tc.expectCancel {
				assert.NotNil(t, run.CancelledAt)
			} else {
				assert.Nil(t, run.CancelledAt)
			}
		})
	}

	t.Run("update non-existent run returns error", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		err := store.UpdateRunStatus("nonexistent", "running", "step-1", 100)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "run not found")
	})
}

// TestGetRun tests retrieving a single run.
func TestGetRun(t *testing.T) {
	t.Run("get existing run", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		runID, err := store.CreateRun("test-pipeline", `{"key": "value"}`)
		require.NoError(t, err)

		err = store.UpdateRunStatus(runID, "running", "step-1", 100)
		require.NoError(t, err)

		run, err := store.GetRun(runID)
		require.NoError(t, err)
		assert.Equal(t, runID, run.RunID)
		assert.Equal(t, "test-pipeline", run.PipelineName)
		assert.Equal(t, "running", run.Status)
		assert.Equal(t, `{"key": "value"}`, run.Input)
		assert.Equal(t, "step-1", run.CurrentStep)
		assert.Equal(t, 100, run.TotalTokens)
	})

	t.Run("get non-existent run returns error", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		run, err := store.GetRun("nonexistent")
		assert.Error(t, err)
		assert.Nil(t, run)
		assert.Contains(t, err.Error(), "run not found")
	})
}

// TestGetRunningRuns tests retrieving all running pipelines.
func TestGetRunningRuns(t *testing.T) {
	t.Run("returns only running runs", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		// Create runs with different statuses
		run1, err := store.CreateRun("pipeline-1", "")
		require.NoError(t, err)
		err = store.UpdateRunStatus(run1, "running", "step-1", 100)
		require.NoError(t, err)

		run2, err := store.CreateRun("pipeline-2", "")
		require.NoError(t, err)
		err = store.UpdateRunStatus(run2, "completed", "final", 200)
		require.NoError(t, err)

		run3, err := store.CreateRun("pipeline-3", "")
		require.NoError(t, err)
		err = store.UpdateRunStatus(run3, "running", "step-2", 150)
		require.NoError(t, err)

		runs, err := store.GetRunningRuns()
		require.NoError(t, err)
		assert.Len(t, runs, 2)

		for _, r := range runs {
			assert.Equal(t, "running", r.Status)
		}
	})

	t.Run("returns empty slice when no running runs", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		runs, err := store.GetRunningRuns()
		require.NoError(t, err)
		assert.Empty(t, runs)
	})
}

// TestListRuns tests listing runs with various filters.
func TestListRuns(t *testing.T) {
	t.Run("list all runs", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		for i := 0; i < 5; i++ {
			_, err := store.CreateRun("test-pipeline", "")
			require.NoError(t, err)
		}

		runs, err := store.ListRuns(ListRunsOptions{})
		require.NoError(t, err)
		assert.Len(t, runs, 5)
	})

	t.Run("filter by pipeline name", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		_, err := store.CreateRun("pipeline-a", "")
		require.NoError(t, err)
		_, err = store.CreateRun("pipeline-b", "")
		require.NoError(t, err)
		_, err = store.CreateRun("pipeline-a", "")
		require.NoError(t, err)

		runs, err := store.ListRuns(ListRunsOptions{PipelineName: "pipeline-a"})
		require.NoError(t, err)
		assert.Len(t, runs, 2)

		for _, r := range runs {
			assert.Equal(t, "pipeline-a", r.PipelineName)
		}
	})

	t.Run("filter by status", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		run1, err := store.CreateRun("test", "")
		require.NoError(t, err)
		err = store.UpdateRunStatus(run1, "running", "", 0)
		require.NoError(t, err)

		run2, err := store.CreateRun("test", "")
		require.NoError(t, err)
		err = store.UpdateRunStatus(run2, "completed", "", 0)
		require.NoError(t, err)

		runs, err := store.ListRuns(ListRunsOptions{Status: "running"})
		require.NoError(t, err)
		assert.Len(t, runs, 1)
		assert.Equal(t, "running", runs[0].Status)
	})

	t.Run("limit results", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		for i := 0; i < 10; i++ {
			_, err := store.CreateRun("test", "")
			require.NoError(t, err)
		}

		runs, err := store.ListRuns(ListRunsOptions{Limit: 3})
		require.NoError(t, err)
		assert.Len(t, runs, 3)
	})

	t.Run("results ordered by started_at DESC", func(t *testing.T) {
		store, cleanup := setupTestStoreWithFile(t)
		defer cleanup()

		_, err := store.CreateRun("first", "")
		require.NoError(t, err)

		time.Sleep(1100 * time.Millisecond)

		_, err = store.CreateRun("second", "")
		require.NoError(t, err)

		runs, err := store.ListRuns(ListRunsOptions{})
		require.NoError(t, err)
		require.Len(t, runs, 2)

		// Second run should be first (most recent)
		assert.Equal(t, "second", runs[0].PipelineName)
		assert.Equal(t, "first", runs[1].PipelineName)
	})
}

// TestDeleteRun tests run deletion.
func TestDeleteRun(t *testing.T) {
	t.Run("delete existing run", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		runID, err := store.CreateRun("test", "")
		require.NoError(t, err)

		err = store.DeleteRun(runID)
		require.NoError(t, err)

		// Verify run is deleted
		run, err := store.GetRun(runID)
		assert.Error(t, err)
		assert.Nil(t, run)
	})

	t.Run("delete run with associated data", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		runID, err := store.CreateRun("test", "")
		require.NoError(t, err)

		// Add associated data
		err = store.LogEvent(runID, "step-1", "running", "dev", "started", 100, 500)
		require.NoError(t, err)
		err = store.RegisterArtifact(runID, "step-1", "output.txt", "/path/output.txt", "file", 1024)
		require.NoError(t, err)
		err = store.RequestCancellation(runID, false)
		require.NoError(t, err)

		// Delete run (should cascade)
		err = store.DeleteRun(runID)
		require.NoError(t, err)

		// Verify run and associated data are deleted
		run, err := store.GetRun(runID)
		assert.Error(t, err)
		assert.Nil(t, run)

		events, err := store.GetEvents(runID, EventQueryOptions{})
		require.NoError(t, err)
		assert.Empty(t, events)

		artifacts, err := store.GetArtifacts(runID, "")
		require.NoError(t, err)
		assert.Empty(t, artifacts)

		cancel, err := store.CheckCancellation(runID)
		require.NoError(t, err)
		assert.Nil(t, cancel)
	})

	t.Run("delete non-existent run returns error", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		err := store.DeleteRun("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "run not found")
	})
}

// TestLogEvent tests event logging.
func TestLogEvent(t *testing.T) {
	testCases := []struct {
		name       string
		stepID     string
		state      string
		persona    string
		message    string
		tokens     int
		durationMs int64
	}{
		{
			name:       "log step started event",
			stepID:     "step-1",
			state:      "running",
			persona:    "developer",
			message:    "Step started",
			tokens:     0,
			durationMs: 0,
		},
		{
			name:       "log step completed event",
			stepID:     "step-1",
			state:      "completed",
			persona:    "developer",
			message:    "Step completed successfully",
			tokens:     150,
			durationMs: 5000,
		},
		{
			name:       "log step failed event",
			stepID:     "step-2",
			state:      "failed",
			persona:    "reviewer",
			message:    "Error: timeout exceeded",
			tokens:     50,
			durationMs: 30000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store, cleanup := setupTestStore(t)
			defer cleanup()

			runID, err := store.CreateRun("test", "")
			require.NoError(t, err)

			err = store.LogEvent(runID, tc.stepID, tc.state, tc.persona, tc.message, tc.tokens, tc.durationMs)
			require.NoError(t, err)

			events, err := store.GetEvents(runID, EventQueryOptions{})
			require.NoError(t, err)
			require.Len(t, events, 1)

			event := events[0]
			assert.Equal(t, runID, event.RunID)
			assert.Equal(t, tc.stepID, event.StepID)
			assert.Equal(t, tc.state, event.State)
			assert.Equal(t, tc.persona, event.Persona)
			assert.Equal(t, tc.message, event.Message)
			assert.Equal(t, tc.tokens, event.TokensUsed)
			assert.Equal(t, tc.durationMs, event.DurationMs)
			assert.False(t, event.Timestamp.IsZero())
		})
	}
}

// TestGetEvents tests event retrieval with filters.
func TestGetEvents(t *testing.T) {
	t.Run("get all events for run", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		runID, err := store.CreateRun("test", "")
		require.NoError(t, err)

		for i := 0; i < 5; i++ {
			stepID := stepIDFromIndex(i)
			err = store.LogEvent(runID, stepID, "running", "dev", "message", 10, 100)
			require.NoError(t, err)
		}

		events, err := store.GetEvents(runID, EventQueryOptions{})
		require.NoError(t, err)
		assert.Len(t, events, 5)
	})

	t.Run("filter by step ID", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		runID, err := store.CreateRun("test", "")
		require.NoError(t, err)

		err = store.LogEvent(runID, "step-1", "running", "dev", "msg1", 10, 100)
		require.NoError(t, err)
		err = store.LogEvent(runID, "step-2", "running", "dev", "msg2", 10, 100)
		require.NoError(t, err)
		err = store.LogEvent(runID, "step-1", "completed", "dev", "msg3", 10, 100)
		require.NoError(t, err)

		events, err := store.GetEvents(runID, EventQueryOptions{StepID: "step-1"})
		require.NoError(t, err)
		assert.Len(t, events, 2)

		for _, e := range events {
			assert.Equal(t, "step-1", e.StepID)
		}
	})

	t.Run("filter errors only", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		runID, err := store.CreateRun("test", "")
		require.NoError(t, err)

		err = store.LogEvent(runID, "step-1", "running", "dev", "started", 10, 100)
		require.NoError(t, err)
		err = store.LogEvent(runID, "step-1", "failed", "dev", "error occurred", 10, 100)
		require.NoError(t, err)
		err = store.LogEvent(runID, "step-2", "completed", "dev", "done", 10, 100)
		require.NoError(t, err)

		events, err := store.GetEvents(runID, EventQueryOptions{ErrorsOnly: true})
		require.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, "failed", events[0].State)
	})

	t.Run("limit and offset", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		runID, err := store.CreateRun("test", "")
		require.NoError(t, err)

		for i := 0; i < 10; i++ {
			err = store.LogEvent(runID, "step-1", "running", "dev", "msg", i, 100)
			require.NoError(t, err)
		}

		// Get first 3
		events, err := store.GetEvents(runID, EventQueryOptions{Limit: 3})
		require.NoError(t, err)
		assert.Len(t, events, 3)

		// Get next 3 with offset
		events, err = store.GetEvents(runID, EventQueryOptions{Limit: 3, Offset: 3})
		require.NoError(t, err)
		assert.Len(t, events, 3)
	})

	t.Run("events ordered by timestamp ASC", func(t *testing.T) {
		store, cleanup := setupTestStoreWithFile(t)
		defer cleanup()

		runID, err := store.CreateRun("test", "")
		require.NoError(t, err)

		err = store.LogEvent(runID, "step-1", "running", "dev", "first", 10, 100)
		require.NoError(t, err)

		time.Sleep(1100 * time.Millisecond)

		err = store.LogEvent(runID, "step-2", "running", "dev", "second", 20, 200)
		require.NoError(t, err)

		events, err := store.GetEvents(runID, EventQueryOptions{})
		require.NoError(t, err)
		require.Len(t, events, 2)

		// First event should be first (oldest)
		assert.Equal(t, "first", events[0].Message)
		assert.Equal(t, "second", events[1].Message)
	})
}

// TestRegisterArtifact tests artifact registration.
func TestRegisterArtifact(t *testing.T) {
	testCases := []struct {
		name         string
		stepID       string
		artifactName string
		path         string
		artifactType string
		sizeBytes    int64
	}{
		{
			name:         "register file artifact",
			stepID:       "step-1",
			artifactName: "output.txt",
			path:         "/workspace/run-123/output.txt",
			artifactType: "file",
			sizeBytes:    1024,
		},
		{
			name:         "register directory artifact",
			stepID:       "step-2",
			artifactName: "results",
			path:         "/workspace/run-123/results",
			artifactType: "directory",
			sizeBytes:    0,
		},
		{
			name:         "register large artifact",
			stepID:       "step-3",
			artifactName: "model.bin",
			path:         "/workspace/run-123/model.bin",
			artifactType: "binary",
			sizeBytes:    1073741824, // 1GB
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store, cleanup := setupTestStore(t)
			defer cleanup()

			runID, err := store.CreateRun("test", "")
			require.NoError(t, err)

			err = store.RegisterArtifact(runID, tc.stepID, tc.artifactName, tc.path, tc.artifactType, tc.sizeBytes)
			require.NoError(t, err)

			artifacts, err := store.GetArtifacts(runID, "")
			require.NoError(t, err)
			require.Len(t, artifacts, 1)

			artifact := artifacts[0]
			assert.Equal(t, runID, artifact.RunID)
			assert.Equal(t, tc.stepID, artifact.StepID)
			assert.Equal(t, tc.artifactName, artifact.Name)
			assert.Equal(t, tc.path, artifact.Path)
			assert.Equal(t, tc.artifactType, artifact.Type)
			assert.Equal(t, tc.sizeBytes, artifact.SizeBytes)
			assert.False(t, artifact.CreatedAt.IsZero())
		})
	}
}

// TestGetArtifacts tests artifact retrieval.
func TestGetArtifacts(t *testing.T) {
	t.Run("get all artifacts for run", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		runID, err := store.CreateRun("test", "")
		require.NoError(t, err)

		for i := 0; i < 3; i++ {
			stepID := stepIDFromIndex(i)
			name := "artifact-" + stepID
			err = store.RegisterArtifact(runID, stepID, name, "/path/"+name, "file", 1024)
			require.NoError(t, err)
		}

		artifacts, err := store.GetArtifacts(runID, "")
		require.NoError(t, err)
		assert.Len(t, artifacts, 3)
	})

	t.Run("filter by step ID", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		runID, err := store.CreateRun("test", "")
		require.NoError(t, err)

		err = store.RegisterArtifact(runID, "step-1", "artifact-1", "/path/1", "file", 100)
		require.NoError(t, err)
		err = store.RegisterArtifact(runID, "step-2", "artifact-2", "/path/2", "file", 200)
		require.NoError(t, err)
		err = store.RegisterArtifact(runID, "step-1", "artifact-3", "/path/3", "file", 300)
		require.NoError(t, err)

		artifacts, err := store.GetArtifacts(runID, "step-1")
		require.NoError(t, err)
		assert.Len(t, artifacts, 2)

		for _, a := range artifacts {
			assert.Equal(t, "step-1", a.StepID)
		}
	})

	t.Run("get artifacts for run with no artifacts", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		runID, err := store.CreateRun("test", "")
		require.NoError(t, err)

		artifacts, err := store.GetArtifacts(runID, "")
		require.NoError(t, err)
		assert.Empty(t, artifacts)
	})
}

// TestCancellation tests cancellation flag operations.
func TestCancellation(t *testing.T) {
	t.Run("request cancellation", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		runID, err := store.CreateRun("test", "")
		require.NoError(t, err)

		err = store.RequestCancellation(runID, false)
		require.NoError(t, err)

		cancel, err := store.CheckCancellation(runID)
		require.NoError(t, err)
		require.NotNil(t, cancel)
		assert.Equal(t, runID, cancel.RunID)
		assert.False(t, cancel.Force)
		assert.False(t, cancel.RequestedAt.IsZero())
	})

	t.Run("request force cancellation", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		runID, err := store.CreateRun("test", "")
		require.NoError(t, err)

		err = store.RequestCancellation(runID, true)
		require.NoError(t, err)

		cancel, err := store.CheckCancellation(runID)
		require.NoError(t, err)
		require.NotNil(t, cancel)
		assert.True(t, cancel.Force)
	})

	t.Run("check non-existent cancellation returns nil", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		runID, err := store.CreateRun("test", "")
		require.NoError(t, err)

		cancel, err := store.CheckCancellation(runID)
		require.NoError(t, err)
		assert.Nil(t, cancel)
	})

	t.Run("clear cancellation", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		runID, err := store.CreateRun("test", "")
		require.NoError(t, err)

		err = store.RequestCancellation(runID, false)
		require.NoError(t, err)

		// Verify it exists
		cancel, err := store.CheckCancellation(runID)
		require.NoError(t, err)
		require.NotNil(t, cancel)

		// Clear it
		err = store.ClearCancellation(runID)
		require.NoError(t, err)

		// Verify it's gone
		cancel, err = store.CheckCancellation(runID)
		require.NoError(t, err)
		assert.Nil(t, cancel)
	})

	t.Run("update cancellation overwrites previous", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		runID, err := store.CreateRun("test", "")
		require.NoError(t, err)

		// Request non-force cancellation
		err = store.RequestCancellation(runID, false)
		require.NoError(t, err)

		cancel, err := store.CheckCancellation(runID)
		require.NoError(t, err)
		assert.False(t, cancel.Force)

		// Update to force cancellation
		err = store.RequestCancellation(runID, true)
		require.NoError(t, err)

		cancel, err = store.CheckCancellation(runID)
		require.NoError(t, err)
		assert.True(t, cancel.Force)
	})
}

// TestConcurrentOpsAccess tests concurrent access for ops operations.
func TestConcurrentOpsAccess(t *testing.T) {
	t.Run("concurrent event logging", func(t *testing.T) {
		store, cleanup := setupTestStoreWithFile(t)
		defer cleanup()

		runID, err := store.CreateRun("concurrent-test", "")
		require.NoError(t, err)

		numWorkers := 10
		numEvents := 20
		var wg sync.WaitGroup

		wg.Add(numWorkers)
		for i := 0; i < numWorkers; i++ {
			go func(workerID int) {
				defer wg.Done()
				for j := 0; j < numEvents; j++ {
					stepID := stepIDFromIndices(workerID, j)
					err := store.LogEvent(runID, stepID, "running", "dev", "msg", j, int64(j*100))
					assert.NoError(t, err)
				}
			}(i)
		}

		wg.Wait()

		events, err := store.GetEvents(runID, EventQueryOptions{})
		require.NoError(t, err)
		assert.Len(t, events, numWorkers*numEvents)
	})

	t.Run("concurrent artifact registration", func(t *testing.T) {
		store, cleanup := setupTestStoreWithFile(t)
		defer cleanup()

		runID, err := store.CreateRun("concurrent-test", "")
		require.NoError(t, err)

		numWorkers := 5
		numArtifacts := 10
		var wg sync.WaitGroup

		wg.Add(numWorkers)
		for i := 0; i < numWorkers; i++ {
			go func(workerID int) {
				defer wg.Done()
				for j := 0; j < numArtifacts; j++ {
					stepID := stepIDFromIndices(workerID, j)
					name := "artifact-" + stepID
					err := store.RegisterArtifact(runID, stepID, name, "/path/"+name, "file", int64(j*100))
					assert.NoError(t, err)
				}
			}(i)
		}

		wg.Wait()

		artifacts, err := store.GetArtifacts(runID, "")
		require.NoError(t, err)
		assert.Len(t, artifacts, numWorkers*numArtifacts)
	})

	t.Run("concurrent cancellation checks", func(t *testing.T) {
		store, cleanup := setupTestStoreWithFile(t)
		defer cleanup()

		runID, err := store.CreateRun("concurrent-test", "")
		require.NoError(t, err)

		err = store.RequestCancellation(runID, false)
		require.NoError(t, err)

		numReaders := 20
		var wg sync.WaitGroup

		wg.Add(numReaders)
		for i := 0; i < numReaders; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < 50; j++ {
					cancel, err := store.CheckCancellation(runID)
					assert.NoError(t, err)
					assert.NotNil(t, cancel)
				}
			}()
		}

		wg.Wait()
	})
}

// TestOldRunsCleanup tests filtering runs older than a duration.
func TestOldRunsCleanup(t *testing.T) {
	t.Run("filter runs older than duration", func(t *testing.T) {
		store, cleanup := setupTestStoreWithFile(t)
		defer cleanup()

		// Create an old run
		oldRunID, err := store.CreateRun("old-pipeline", "")
		require.NoError(t, err)

		// Wait to create time gap
		time.Sleep(2 * time.Second)

		// Create a new run
		_, err = store.CreateRun("new-pipeline", "")
		require.NoError(t, err)

		// List runs older than 1 second (should include old run)
		runs, err := store.ListRuns(ListRunsOptions{OlderThan: 1 * time.Second})
		require.NoError(t, err)
		assert.Len(t, runs, 1)
		assert.Equal(t, oldRunID, runs[0].RunID)
	})
}
