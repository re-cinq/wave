package pipeline

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestCheckpointRecorder_Record(t *testing.T) {
	t.Run("records checkpoint with artifact snapshot", func(t *testing.T) {
		store := testutil.NewMockStateStore()

		execution := &PipelineExecution{
			Status: &PipelineStatus{ID: "run-123"},
			ArtifactPaths: map[string]string{
				"step1:plan": "/tmp/artifacts/plan.md",
				"step2:code": "/tmp/artifacts/code.go",
			},
			WorkspacePaths: map[string]string{
				"step1": "/tmp/ws/step1",
				"step2": "/tmp/ws/step2",
			},
			Pipeline: &Pipeline{
				Steps: []Step{{ID: "step1"}, {ID: "step2"}},
			},
		}

		step := &Step{ID: "step2"}
		recorder := &CheckpointRecorder{store: store}

		// Should not panic — the default mock SaveCheckpoint is a no-op.
		recorder.Record(execution, step, 1)
	})

	t.Run("nil store is no-op", func(t *testing.T) {
		execution := &PipelineExecution{
			Status:         &PipelineStatus{ID: "run-123"},
			ArtifactPaths:  map[string]string{},
			WorkspacePaths: map[string]string{},
		}
		step := &Step{ID: "step1"}
		recorder := &CheckpointRecorder{store: nil}

		// Must not panic when store is nil.
		recorder.Record(execution, step, 0)
	})

	t.Run("empty artifact paths records empty JSON object", func(t *testing.T) {
		var capturedRecord *state.CheckpointRecord
		var mu sync.Mutex

		store := &checkpointCapturingStore{
			MockStateStore: testutil.NewMockStateStore(),
			onSave: func(r *state.CheckpointRecord) {
				mu.Lock()
				capturedRecord = r
				mu.Unlock()
			},
		}

		execution := &PipelineExecution{
			Status:         &PipelineStatus{ID: "run-empty"},
			ArtifactPaths:  map[string]string{},
			WorkspacePaths: map[string]string{},
			Pipeline: &Pipeline{
				Steps: []Step{{ID: "step1"}},
			},
		}

		step := &Step{ID: "step1"}
		recorder := &CheckpointRecorder{store: store}
		recorder.Record(execution, step, 0)

		mu.Lock()
		defer mu.Unlock()

		assert.NotNil(t, capturedRecord)
		assert.Equal(t, "run-empty", capturedRecord.RunID)
		assert.Equal(t, "step1", capturedRecord.StepID)
		assert.Equal(t, 0, capturedRecord.StepIndex)

		// Empty artifact map should still produce valid JSON.
		var snapshot map[string]string
		err := json.Unmarshal([]byte(capturedRecord.ArtifactSnapshot), &snapshot)
		assert.NoError(t, err)
		assert.Empty(t, snapshot)
	})

	t.Run("captures artifact snapshot as JSON", func(t *testing.T) {
		var capturedRecord *state.CheckpointRecord
		var mu sync.Mutex

		store := &checkpointCapturingStore{
			MockStateStore: testutil.NewMockStateStore(),
			onSave: func(r *state.CheckpointRecord) {
				mu.Lock()
				capturedRecord = r
				mu.Unlock()
			},
		}

		artifacts := map[string]string{
			"plan:output":    "/artifacts/plan.md",
			"implement:code": "/artifacts/code.go",
		}

		execution := &PipelineExecution{
			Status:         &PipelineStatus{ID: "test-run-abc"},
			ArtifactPaths:  artifacts,
			WorkspacePaths: map[string]string{"implement": "/ws/implement"},
			Pipeline: &Pipeline{
				Steps: []Step{{ID: "plan"}, {ID: "implement"}},
			},
		}

		step := &Step{ID: "implement"}
		recorder := &CheckpointRecorder{store: store}
		recorder.Record(execution, step, 1)

		mu.Lock()
		defer mu.Unlock()

		assert.NotNil(t, capturedRecord)
		assert.Equal(t, "test-run-abc", capturedRecord.RunID)
		assert.Equal(t, "implement", capturedRecord.StepID)
		assert.Equal(t, 1, capturedRecord.StepIndex)
		assert.Equal(t, "/ws/implement", capturedRecord.WorkspacePath)

		// Verify artifact snapshot is valid JSON with expected content.
		var snapshot map[string]string
		err := json.Unmarshal([]byte(capturedRecord.ArtifactSnapshot), &snapshot)
		assert.NoError(t, err)
		assert.Equal(t, "/artifacts/plan.md", snapshot["plan:output"])
		assert.Equal(t, "/artifacts/code.go", snapshot["implement:code"])
	})

	t.Run("workspace path resolved from step ID", func(t *testing.T) {
		var capturedRecord *state.CheckpointRecord
		var mu sync.Mutex

		store := &checkpointCapturingStore{
			MockStateStore: testutil.NewMockStateStore(),
			onSave: func(r *state.CheckpointRecord) {
				mu.Lock()
				capturedRecord = r
				mu.Unlock()
			},
		}

		execution := &PipelineExecution{
			Status:        &PipelineStatus{ID: "run-ws"},
			ArtifactPaths: map[string]string{},
			WorkspacePaths: map[string]string{
				"plan":      "/ws/plan",
				"implement": "/ws/implement",
			},
			Pipeline: &Pipeline{
				Steps: []Step{{ID: "plan"}, {ID: "implement"}},
			},
		}

		step := &Step{ID: "plan"}
		recorder := &CheckpointRecorder{store: store}
		recorder.Record(execution, step, 0)

		mu.Lock()
		defer mu.Unlock()

		assert.NotNil(t, capturedRecord)
		assert.Equal(t, "/ws/plan", capturedRecord.WorkspacePath)
	})
}

// checkpointCapturingStore wraps MockStateStore to capture SaveCheckpoint calls.
type checkpointCapturingStore struct {
	*testutil.MockStateStore
	onSave func(*state.CheckpointRecord)
}

func (s *checkpointCapturingStore) SaveCheckpoint(record *state.CheckpointRecord) error {
	if s.onSave != nil {
		s.onSave(record)
	}
	return nil
}
