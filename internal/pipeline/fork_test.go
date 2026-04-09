package pipeline

import (
	"fmt"
	"testing"

	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestResolveStepID(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{ID: "plan"},
			{ID: "implement"},
			{ID: "test"},
		},
	}

	tests := []struct {
		name     string
		ref      string
		expected string
	}{
		{"by name", "plan", "plan"},
		{"by name middle", "implement", "implement"},
		{"by name last", "test", "test"},
		{"by index 0", "0", "plan"},
		{"by index 1", "1", "implement"},
		{"by index 2", "2", "test"},
		{"not found name", "nonexistent", ""},
		{"index out of range", "5", ""},
		{"negative index", "-1", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveStepID(tt.ref, p)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []string
	}{
		{"simple relative", "a/b/c", []string{"a", "b", "c"}},
		{"single component", "file", []string{"file"}},
		{"with trailing slash cleaned", "a/b/c/", []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitPath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReplaceRunIDInPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		oldRunID string
		newRunID string
		expected string
	}{
		{
			"replaces run ID segment",
			".wave/artifacts/run-001/plan.md",
			"run-001",
			"fork-001",
			".wave/artifacts/fork-001/plan.md",
		},
		{
			"no match returns original",
			"/tmp/artifacts/plan.md",
			"run-001",
			"fork-001",
			"/tmp/artifacts/plan.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceRunIDInPath(tt.path, tt.oldRunID, tt.newRunID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestForkManager_ListForkPoints(t *testing.T) {
	t.Run("returns fork points from checkpoints", func(t *testing.T) {
		store := &forkTestStore{
			MockStateStore: testutil.NewMockStateStore(),
			checkpoints: []state.CheckpointRecord{
				{StepID: "plan", StepIndex: 0, WorkspaceCommitSHA: "abc123"},
				{StepID: "implement", StepIndex: 1},
			},
		}
		fm := NewForkManager(store)
		points, err := fm.ListForkPoints("run-123")
		assert.NoError(t, err)
		assert.Len(t, points, 2)
		assert.Equal(t, "plan", points[0].StepID)
		assert.Equal(t, 0, points[0].StepIndex)
		assert.True(t, points[0].HasSHA)
		assert.Equal(t, "implement", points[1].StepID)
		assert.Equal(t, 1, points[1].StepIndex)
		assert.False(t, points[1].HasSHA)
	})

	t.Run("returns empty for no checkpoints", func(t *testing.T) {
		store := &forkTestStore{MockStateStore: testutil.NewMockStateStore()}
		fm := NewForkManager(store)
		points, err := fm.ListForkPoints("run-123")
		assert.NoError(t, err)
		assert.Empty(t, points)
	})

	t.Run("propagates store error", func(t *testing.T) {
		store := &forkTestStore{
			MockStateStore: testutil.NewMockStateStore(),
			checkpointsErr: fmt.Errorf("db connection lost"),
		}
		fm := NewForkManager(store)
		_, err := fm.ListForkPoints("run-123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "db connection lost")
	})
}

func TestForkManager_Fork_Errors(t *testing.T) {
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "test-pipeline"},
		Steps:    []Step{{ID: "plan"}, {ID: "implement"}},
	}

	t.Run("rejects running run", func(t *testing.T) {
		store := &forkTestStore{
			MockStateStore: testutil.NewMockStateStore(),
			run:            &state.RunRecord{Status: "running", PipelineName: "test"},
		}
		fm := NewForkManager(store)
		_, err := fm.Fork("run-123", "plan", p)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot fork a running run")
	})

	t.Run("rejects failed run without allow-failed", func(t *testing.T) {
		store := &forkTestStore{
			MockStateStore: testutil.NewMockStateStore(),
			run:            &state.RunRecord{Status: "failed", PipelineName: "test"},
		}
		fm := NewForkManager(store)
		_, err := fm.Fork("run-123", "plan", p)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot fork failed run")
		assert.Contains(t, err.Error(), "--allow-failed")
	})

	t.Run("allows failed run with allow-failed flag", func(t *testing.T) {
		store := &forkTestStore{
			MockStateStore: testutil.NewMockStateStore(),
			run:            &state.RunRecord{Status: "failed", PipelineName: "test"},
			// No checkpoint — will fail at checkpoint lookup, proving the status check passed.
		}
		fm := NewForkManager(store)
		_, err := fm.Fork("run-123", "plan", p, true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no checkpoint found")
	})

	t.Run("rejects unknown step", func(t *testing.T) {
		store := &forkTestStore{
			MockStateStore: testutil.NewMockStateStore(),
			run:            &state.RunRecord{Status: "completed", PipelineName: "test"},
		}
		fm := NewForkManager(store)
		_, err := fm.Fork("run-123", "nonexistent", p)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found in pipeline")
	})

	t.Run("rejects missing source run", func(t *testing.T) {
		store := &forkTestStore{
			MockStateStore: testutil.NewMockStateStore(),
			runErr:         fmt.Errorf("run not found"),
		}
		fm := NewForkManager(store)
		_, err := fm.Fork("nonexistent-run", "plan", p)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("rejects step with no checkpoint", func(t *testing.T) {
		store := &forkTestStore{
			MockStateStore: testutil.NewMockStateStore(),
			run:            &state.RunRecord{Status: "completed", PipelineName: "test"},
			// No checkpoint set — GetCheckpoint will return error.
		}
		fm := NewForkManager(store)
		_, err := fm.Fork("run-123", "plan", p)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no checkpoint found")
	})

	t.Run("resolves step by numeric index", func(t *testing.T) {
		store := &forkTestStore{
			MockStateStore: testutil.NewMockStateStore(),
			run:            &state.RunRecord{Status: "completed", PipelineName: "test"},
			// No checkpoint for "implement" — will fail at checkpoint lookup.
		}
		fm := NewForkManager(store)
		_, err := fm.Fork("run-123", "1", p)
		assert.Error(t, err)
		// The important thing: it resolved "1" to "implement" and tried to get its checkpoint.
		assert.Contains(t, err.Error(), "no checkpoint found")
	})
}

// forkTestStore wraps MockStateStore with overridable checkpoint and run methods.
type forkTestStore struct {
	*testutil.MockStateStore
	checkpoints    []state.CheckpointRecord
	checkpointsErr error
	checkpoint     *state.CheckpointRecord
	run            *state.RunRecord
	runErr         error
	savedCPs       []state.CheckpointRecord
}

func (s *forkTestStore) GetRun(runID string) (*state.RunRecord, error) {
	if s.runErr != nil {
		return nil, s.runErr
	}
	if s.run != nil {
		return s.run, nil
	}
	return nil, fmt.Errorf("run not found: %s", runID)
}

func (s *forkTestStore) GetCheckpoints(runID string) ([]state.CheckpointRecord, error) {
	if s.checkpointsErr != nil {
		return nil, s.checkpointsErr
	}
	return s.checkpoints, nil
}

func (s *forkTestStore) GetCheckpoint(runID, stepID string) (*state.CheckpointRecord, error) {
	if s.checkpoint != nil {
		return s.checkpoint, nil
	}
	for i, cp := range s.checkpoints {
		if cp.StepID == stepID {
			return &s.checkpoints[i], nil
		}
	}
	return nil, fmt.Errorf("checkpoint not found for run %s step %s", runID, stepID)
}

func (s *forkTestStore) CreateRunWithFork(pipelineName, input, forkedFrom string) (string, error) {
	return "fork-run-001", nil
}

func (s *forkTestStore) SaveCheckpoint(record *state.CheckpointRecord) error {
	s.savedCPs = append(s.savedCPs, *record)
	return nil
}
