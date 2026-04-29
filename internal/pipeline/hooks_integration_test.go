package pipeline

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter/adaptertest"
	"github.com/recinq/wave/internal/hooks"
	"github.com/recinq/wave/internal/ontology"
	"github.com/recinq/wave/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockHookRunner records all RunHooks calls and optionally returns errors for specific event types.
type mockHookRunner struct {
	mu     sync.Mutex
	calls  []hooks.HookEvent
	failOn map[hooks.EventType]error
}

func (m *mockHookRunner) RunHooks(_ context.Context, evt hooks.HookEvent) ([]hooks.HookResult, error) {
	m.mu.Lock()
	m.calls = append(m.calls, evt)
	m.mu.Unlock()
	if err, ok := m.failOn[evt.Type]; ok {
		return nil, err
	}
	return nil, nil
}

// getCalls returns a snapshot of recorded calls.
func (m *mockHookRunner) getCalls() []hooks.HookEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]hooks.HookEvent, len(m.calls))
	copy(out, m.calls)
	return out
}

// TestHooksFireAtCorrectLifecyclePoints verifies that the executor fires lifecycle
// hook events in the correct order: run_start, then per-step (step_start,
// workspace_created, step_completed), and finally run_completed.
func TestHooksFireAtCorrectLifecyclePoints(t *testing.T) {
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
		adaptertest.WithTokensUsed(500),
	)

	hr := &mockHookRunner{failOn: map[hooks.EventType]error{}}

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
		withHookRunner(hr),
		WithOntologyService(ontology.NoOp{}),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "hooks-lifecycle-test"},
		Steps: []Step{
			{ID: "alpha", Persona: "navigator", Dependencies: []string{}, Exec: ExecConfig{Source: "A"}},
			{ID: "beta", Persona: "navigator", Dependencies: []string{"alpha"}, Exec: ExecConfig{Source: "B"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test-input")
	require.NoError(t, err)

	calls := hr.getCalls()

	// Collect counts per event type
	counts := make(map[hooks.EventType]int)
	for _, c := range calls {
		counts[c.Type]++
	}

	// run_start fires exactly once
	assert.Equal(t, 1, counts[hooks.EventRunStart], "run_start should fire once")
	// step_start fires once per step
	assert.Equal(t, 2, counts[hooks.EventStepStart], "step_start should fire once per step")
	// workspace_created fires once per step
	assert.Equal(t, 2, counts[hooks.EventWorkspaceCreated], "workspace_created should fire once per step")
	// step_completed fires once per step
	assert.Equal(t, 2, counts[hooks.EventStepCompleted], "step_completed should fire once per step")
	// run_completed fires exactly once
	assert.Equal(t, 1, counts[hooks.EventRunCompleted], "run_completed should fire once")

	// Verify ordering: run_start is first, run_completed is last
	require.NotEmpty(t, calls, "should have at least one hook call")
	assert.Equal(t, hooks.EventRunStart, calls[0].Type, "first hook should be run_start")
	assert.Equal(t, hooks.EventRunCompleted, calls[len(calls)-1].Type, "last hook should be run_completed")

	// Verify per-step ordering: for each step, step_start comes before workspace_created
	// which comes before step_completed.
	for _, stepID := range []string{"alpha", "beta"} {
		var stepStart, wsCreated, stepCompleted int
		for i, c := range calls {
			if c.StepID != stepID {
				continue
			}
			switch c.Type {
			case hooks.EventStepStart:
				stepStart = i
			case hooks.EventWorkspaceCreated:
				wsCreated = i
			case hooks.EventStepCompleted:
				stepCompleted = i
			}
		}
		assert.True(t, stepStart < wsCreated,
			"step_start should come before workspace_created for %s", stepID)
		assert.True(t, wsCreated < stepCompleted,
			"workspace_created should come before step_completed for %s", stepID)
	}

	// Verify pipeline ID is consistent across all calls
	pipelineID := calls[0].PipelineID
	require.NotEmpty(t, pipelineID)
	for _, c := range calls {
		assert.Equal(t, pipelineID, c.PipelineID, "all hooks should share the same pipeline ID")
	}
}

// TestBlockingStepStartHookAbortsPipeline verifies that a blocking hook error
// on step_start causes the pipeline to fail.
func TestBlockingStepStartHookAbortsPipeline(t *testing.T) {
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
		adaptertest.WithTokensUsed(500),
	)

	hookErr := errors.New("step_start hook rejected")
	hr := &mockHookRunner{
		failOn: map[hooks.EventType]error{
			hooks.EventStepStart: hookErr,
		},
	}

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
		withHookRunner(hr),
		WithOntologyService(ontology.NoOp{}),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "hooks-blocking-test"},
		Steps: []Step{
			{ID: "only-step", Persona: "navigator", Dependencies: []string{}, Exec: ExecConfig{Source: "A"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test-input")
	require.Error(t, err, "pipeline should fail when step_start hook returns an error")
	assert.Contains(t, err.Error(), "step_start hook failed",
		"error message should indicate step_start hook failure")
}

// TestNonBlockingHooksContinue verifies that an error from a non-blocking hook
// (run_completed) does not cause the pipeline to fail.
func TestNonBlockingHooksContinue(t *testing.T) {
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
		adaptertest.WithTokensUsed(500),
	)

	hr := &mockHookRunner{
		failOn: map[hooks.EventType]error{
			hooks.EventRunCompleted: errors.New("run_completed hook failed"),
		},
	}

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
		withHookRunner(hr),
		WithOntologyService(ontology.NoOp{}),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "hooks-nonblocking-test"},
		Steps: []Step{
			{ID: "step-one", Persona: "navigator", Dependencies: []string{}, Exec: ExecConfig{Source: "A"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test-input")
	require.NoError(t, err, "pipeline should succeed even when non-blocking run_completed hook errors")

	// Verify all lifecycle events still fired
	calls := hr.getCalls()
	types := make(map[hooks.EventType]bool)
	for _, c := range calls {
		types[c.Type] = true
	}
	assert.True(t, types[hooks.EventRunStart], "run_start should have fired")
	assert.True(t, types[hooks.EventStepStart], "step_start should have fired")
	assert.True(t, types[hooks.EventStepCompleted], "step_completed should have fired")
	assert.True(t, types[hooks.EventRunCompleted], "run_completed should have fired")
}
