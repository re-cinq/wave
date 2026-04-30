package pipeline

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"strings"

	"github.com/recinq/wave/internal/adapter/adaptertest"
	"github.com/recinq/wave/internal/contract"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestSQLiteStore returns a file-backed SQLite state store for tests that
// need to exercise the EvolutionStore surface (in-memory mode does not work
// with the executor's WAL connection pool semantics).
func newTestSQLiteStore(t *testing.T) state.StateStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "state.db")
	store, err := state.NewStateStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	return store
}

// TestPipelineEvalHook_AllSuccessRun verifies that a pipeline with two
// successful steps writes one pipeline_eval row with FailureClass empty,
// ContractPass true, and DurationMs > 0.
func TestPipelineEvalHook_AllSuccessRun(t *testing.T) {
	store := newTestSQLiteStore(t)
	mock := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status":"ok"}`),
		adaptertest.WithTokensUsed(100),
	)
	executor := NewDefaultPipelineExecutor(mock,
		WithStateStore(store),
		WithEmitter(testutil.NewEventCollector()),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "eval-success-test"},
		Steps: []Step{
			{ID: "step-a", Persona: "navigator", Exec: ExecConfig{Source: "A"}},
			{ID: "step-b", Persona: "navigator", Dependencies: []string{"step-a"}, Exec: ExecConfig{Source: "B"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	require.NoError(t, executor.Execute(ctx, p, m, "test-input"))

	rows, err := store.GetEvalsForPipeline("eval-success-test", 0)
	require.NoError(t, err)
	require.Len(t, rows, 1, "expected exactly one pipeline_eval row for the run")

	row := rows[0]
	assert.NotEmpty(t, row.RunID, "RunID should be set")
	assert.Equal(t, "eval-success-test", row.PipelineName)
	assert.Empty(t, row.FailureClass, "all-success run should have empty FailureClass")
	require.NotNil(t, row.ContractPass)
	assert.True(t, *row.ContractPass, "all-success run should have ContractPass=true")
	require.NotNil(t, row.DurationMs)
	assert.Greater(t, *row.DurationMs, int64(0), "DurationMs should be positive")
	assert.Nil(t, row.RetryCount, "no retries occurred → RetryCount nil")
}

// TestPipelineEvalHook_FailedRun verifies that a pipeline with a failing step
// writes a pipeline_eval row with FailureClass="failure" and ContractPass=false.
func TestPipelineEvalHook_FailedRun(t *testing.T) {
	store := newTestSQLiteStore(t)

	// Always fails — exhausts retries.
	failAdapter := newCountingFailAdapter(10, errors.New("step blew up"))

	executor := NewDefaultPipelineExecutor(failAdapter,
		WithStateStore(store),
		WithEmitter(testutil.NewEventCollector()),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "eval-failed-test"},
		Steps: []Step{
			{
				ID:      "step-1",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "do work"},
				Retry: RetryConfig{
					MaxAttempts: 1, // fail fast
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := executor.Execute(ctx, p, m, "test-input")
	require.Error(t, err, "pipeline should fail")

	rows, err := store.GetEvalsForPipeline("eval-failed-test", 0)
	require.NoError(t, err)
	require.Len(t, rows, 1, "failed run should still record one pipeline_eval row")

	row := rows[0]
	assert.Equal(t, "failure", row.FailureClass, "non-contract step failure → FailureClass=failure")
	require.NotNil(t, row.ContractPass)
	assert.False(t, *row.ContractPass, "failure run should have ContractPass=false")
	require.NotNil(t, row.DurationMs)
	assert.Greater(t, *row.DurationMs, int64(0))
}

// TestPipelineEvalHook_RetryCount verifies that retry attempts are counted
// into RetryCount on the pipeline_eval row.
func TestPipelineEvalHook_RetryCount(t *testing.T) {
	store := newTestSQLiteStore(t)

	// Fails twice then succeeds — three attempts total = 2 retries.
	failAdapter := newCountingFailAdapter(2, errors.New("transient blip"))

	executor := NewDefaultPipelineExecutor(failAdapter,
		WithStateStore(store),
		WithEmitter(testutil.NewEventCollector()),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "eval-retry-test"},
		Steps: []Step{
			{
				ID:      "step-1",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "do work"},
				Retry: RetryConfig{
					MaxAttempts: 3,
					BaseDelay:   "1ms",
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	require.NoError(t, executor.Execute(ctx, p, m, "test-input"))

	rows, err := store.GetEvalsForPipeline("eval-retry-test", 0)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	require.NotNil(t, rows[0].RetryCount, "retries fired → RetryCount populated")
	assert.GreaterOrEqual(t, *rows[0].RetryCount, 2, "expected at least 2 retries recorded")
}

// errOnRecordEvalStore wraps a real StateStore but returns an error from
// RecordEval. Used to verify the hook is non-fatal.
type errOnRecordEvalStore struct {
	state.StateStore
	recordEvalErr error
	calls         int
}

func (s *errOnRecordEvalStore) RecordEval(rec state.PipelineEvalRecord) error {
	s.calls++
	if s.recordEvalErr != nil {
		return s.recordEvalErr
	}
	return s.StateStore.RecordEval(rec)
}

// TestPipelineEvalHook_StoreErrorIsNonFatal verifies that a RecordEval error
// does not propagate up — the pipeline still returns nil.
func TestPipelineEvalHook_StoreErrorIsNonFatal(t *testing.T) {
	inner := newTestSQLiteStore(t)
	store := &errOnRecordEvalStore{
		StateStore:    inner,
		recordEvalErr: errors.New("synthetic eval write failure"),
	}

	mock := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status":"ok"}`),
	)
	executor := NewDefaultPipelineExecutor(mock,
		WithStateStore(store),
		WithEmitter(testutil.NewEventCollector()),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "eval-error-test"},
		Steps: []Step{
			{ID: "step-a", Persona: "navigator", Exec: ExecConfig{Source: "A"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := executor.Execute(ctx, p, m, "test-input")
	require.NoError(t, err, "RecordEval failure must not bubble up to pipeline result")

	assert.GreaterOrEqual(t, store.calls, 1, "RecordEval should have been attempted")
}

// stubEvolutionTrigger is a deterministic EvolutionTrigger for testing
// the executor emission path.
type stubEvolutionTrigger struct {
	fire   bool
	reason string
	err    error
	calls  int
}

func (s *stubEvolutionTrigger) ShouldEvolve(_ string) (bool, string, error) {
	s.calls++
	return s.fire, s.reason, s.err
}

// TestEvolutionTrigger_FireEmitsEvent verifies recordPipelineEval emits
// "evolution_proposed" with the trigger's reason in Message.
func TestEvolutionTrigger_FireEmitsEvent(t *testing.T) {
	store := newTestSQLiteStore(t)
	collector := testutil.NewEventCollector()
	trigger := &stubEvolutionTrigger{fire: true, reason: "drift: -20%"}

	mock := adaptertest.NewMockAdapter(adaptertest.WithStdoutJSON(`{"status":"ok"}`))
	executor := NewDefaultPipelineExecutor(mock,
		WithStateStore(store),
		WithEmitter(collector),
		WithEvolutionTrigger(trigger),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "evol-fire-test"},
		Steps: []Step{
			{ID: "step-a", Persona: "navigator", Exec: ExecConfig{Source: "A"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	require.NoError(t, executor.Execute(ctx, p, m, "in"))

	assert.GreaterOrEqual(t, trigger.calls, 1, "trigger should be consulted")
	assert.True(t, collector.HasEventWithState("evolution_proposed"),
		"firing trigger should emit evolution_proposed event")

	// Verify reason is carried in Message.
	var found bool
	for _, e := range collector.GetEvents() {
		if e.State == "evolution_proposed" {
			found = true
			assert.Contains(t, e.Message, "evol-fire-test")
			assert.Contains(t, e.Message, "drift: -20%")
		}
	}
	require.True(t, found, "evolution_proposed event must exist")
}

// TestEvolutionTrigger_NoFireNoEvent: trigger returns (false, ...) →
// no evolution_proposed event.
func TestEvolutionTrigger_NoFireNoEvent(t *testing.T) {
	store := newTestSQLiteStore(t)
	collector := testutil.NewEventCollector()
	trigger := &stubEvolutionTrigger{fire: false}

	mock := adaptertest.NewMockAdapter(adaptertest.WithStdoutJSON(`{"status":"ok"}`))
	executor := NewDefaultPipelineExecutor(mock,
		WithStateStore(store),
		WithEmitter(collector),
		WithEvolutionTrigger(trigger),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "evol-nofire-test"},
		Steps: []Step{
			{ID: "step-a", Persona: "navigator", Exec: ExecConfig{Source: "A"}},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	require.NoError(t, executor.Execute(ctx, p, m, "in"))

	assert.GreaterOrEqual(t, trigger.calls, 1, "trigger should be consulted")
	assert.False(t, collector.HasEventWithState("evolution_proposed"),
		"non-firing trigger must not emit evolution_proposed")
}

// TestEvolutionTrigger_ErrorWarnsButDoesNotEmit: trigger returns error →
// warning event, no evolution_proposed.
func TestEvolutionTrigger_ErrorWarnsButDoesNotEmit(t *testing.T) {
	store := newTestSQLiteStore(t)
	collector := testutil.NewEventCollector()
	trigger := &stubEvolutionTrigger{err: errors.New("trigger boom")}

	mock := adaptertest.NewMockAdapter(adaptertest.WithStdoutJSON(`{"status":"ok"}`))
	executor := NewDefaultPipelineExecutor(mock,
		WithStateStore(store),
		WithEmitter(collector),
		WithEvolutionTrigger(trigger),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "evol-err-test"},
		Steps: []Step{
			{ID: "step-a", Persona: "navigator", Exec: ExecConfig{Source: "A"}},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	require.NoError(t, executor.Execute(ctx, p, m, "in"))

	assert.False(t, collector.HasEventWithState("evolution_proposed"),
		"erroring trigger must not emit evolution_proposed")

	// Confirm a warning carrying the error message was emitted.
	var sawWarn bool
	for _, e := range collector.GetEvents() {
		if e.State == "warning" && strings.Contains(e.Message, "evolution trigger") {
			sawWarn = true
		}
	}
	assert.True(t, sawWarn, "trigger error should produce a warning event")
}

// TestEvolutionTrigger_NilNoPanic: nil trigger → no emission, no panic.
func TestEvolutionTrigger_NilNoPanic(t *testing.T) {
	store := newTestSQLiteStore(t)
	collector := testutil.NewEventCollector()

	mock := adaptertest.NewMockAdapter(adaptertest.WithStdoutJSON(`{"status":"ok"}`))
	executor := NewDefaultPipelineExecutor(mock,
		WithStateStore(store),
		WithEmitter(collector),
		// no WithEvolutionTrigger
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "evol-nil-test"},
		Steps: []Step{
			{ID: "step-a", Persona: "navigator", Exec: ExecConfig{Source: "A"}},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	require.NoError(t, executor.Execute(ctx, p, m, "in"))

	assert.False(t, collector.HasEventWithState("evolution_proposed"),
		"nil trigger must not emit evolution_proposed")
}

// TestPipelineEvalHook_DuplicateInsertSwallowed verifies that resume safety:
// firing the hook twice for the same (pipeline_name, run_id) does not error.
func TestPipelineEvalHook_DuplicateInsertSwallowed(t *testing.T) {
	store := newTestSQLiteStore(t)

	mock := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status":"ok"}`),
	)
	executor := NewDefaultPipelineExecutor(mock,
		WithStateStore(store),
		WithEmitter(testutil.NewEventCollector()),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "eval-dup-test"},
		Steps: []Step{
			{ID: "step-a", Persona: "navigator", Exec: ExecConfig{Source: "A"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	require.NoError(t, executor.Execute(ctx, p, m, "test-input"))

	// Find the run we just wrote and re-fire the hook directly.
	rows, err := store.GetEvalsForPipeline("eval-dup-test", 0)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	runID := rows[0].RunID

	// Build a synthetic execution and SignalSet for the duplicate attempt.
	exec := &PipelineExecution{
		Pipeline: p,
		Status: &PipelineStatus{
			ID:           runID,
			PipelineName: "eval-dup-test",
			StartedAt:    time.Now().Add(-time.Second),
		},
	}
	signalSet := executor.signalSetFor(runID)
	signalSet.Add(contract.Signal{Kind: contract.SignalSuccess, StepID: "step-a"})

	// recordPipelineEval must not panic and must not write a second row.
	executor.recordPipelineEval(exec)

	rows, err = store.GetEvalsForPipeline("eval-dup-test", 0)
	require.NoError(t, err)
	assert.Len(t, rows, 1, "duplicate insert should be swallowed — still one row")
}
