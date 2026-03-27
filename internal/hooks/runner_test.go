package hooks

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/recinq/wave/internal/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testEmitter is a minimal event emitter that records emitted events.
type testEmitter struct {
	events []event.Event
}

func (e *testEmitter) Emit(evt event.Event) {
	e.events = append(e.events, evt)
}

func newTestEmitter() *testEmitter {
	return &testEmitter{}
}

func TestRunnerEventTypeFiltering(t *testing.T) {
	hooks := []LifecycleHookDef{
		{
			Name:    "step-start-hook",
			Event:   EventStepStart,
			Type:    HookTypeCommand,
			Command: "true",
		},
		{
			Name:    "run-start-hook",
			Event:   EventRunStart,
			Type:    HookTypeCommand,
			Command: "true",
		},
	}

	emitter := newTestEmitter()
	runner, err := NewHookRunner(hooks, emitter)
	require.NoError(t, err)

	// Send a step_start event — only the step-start-hook should fire
	results, err := runner.RunHooks(context.Background(), HookEvent{
		Type:       EventStepStart,
		PipelineID: "test",
		StepID:     "step-1",
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "step-start-hook", results[0].HookName)
	assert.Equal(t, DecisionProceed, results[0].Decision)
}

func TestRunnerMatcherFiltering(t *testing.T) {
	hooks := []LifecycleHookDef{
		{
			Name:    "implement-only-hook",
			Event:   EventStepStart,
			Type:    HookTypeCommand,
			Command: "true",
			Matcher: "^implement$",
		},
		{
			Name:    "any-step-hook",
			Event:   EventStepStart,
			Type:    HookTypeCommand,
			Command: "true",
		},
	}

	emitter := newTestEmitter()
	runner, err := NewHookRunner(hooks, emitter)
	require.NoError(t, err)

	// Step "implement" should match both hooks
	results, err := runner.RunHooks(context.Background(), HookEvent{
		Type:       EventStepStart,
		PipelineID: "test",
		StepID:     "implement",
	})
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "implement-only-hook", results[0].HookName)
	assert.Equal(t, "any-step-hook", results[1].HookName)

	// Step "deploy" should only match the any-step hook
	results, err = runner.RunHooks(context.Background(), HookEvent{
		Type:       EventStepStart,
		PipelineID: "test",
		StepID:     "deploy",
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "any-step-hook", results[0].HookName)
}

func TestRunnerSequentialExecutionOrder(t *testing.T) {
	// Use commands that write to a shared file to verify order
	tmpDir := t.TempDir()
	orderFile := tmpDir + "/order.txt"

	hooks := []LifecycleHookDef{
		{
			Name:    "first",
			Event:   EventStepStart,
			Type:    HookTypeCommand,
			Command: "echo -n 'A' >> " + orderFile,
		},
		{
			Name:    "second",
			Event:   EventStepStart,
			Type:    HookTypeCommand,
			Command: "echo -n 'B' >> " + orderFile,
		},
		{
			Name:    "third",
			Event:   EventStepStart,
			Type:    HookTypeCommand,
			Command: "echo -n 'C' >> " + orderFile,
		},
	}

	emitter := newTestEmitter()
	runner, err := NewHookRunner(hooks, emitter)
	require.NoError(t, err)

	results, err := runner.RunHooks(context.Background(), HookEvent{
		Type:       EventStepStart,
		PipelineID: "test",
		StepID:     "step-1",
	})
	require.NoError(t, err)
	require.Len(t, results, 3)

	// Verify order by reading the file
	content, err := readFileContent(orderFile)
	require.NoError(t, err)
	assert.Equal(t, "ABC", content, "hooks should execute in definition order")
}

func TestRunnerBlockingHookFailureHaltsExecution(t *testing.T) {
	hooks := []LifecycleHookDef{
		{
			Name:    "succeeds",
			Event:   EventStepStart,
			Type:    HookTypeCommand,
			Command: "true",
		},
		{
			Name:     "blocks",
			Event:    EventStepStart,
			Type:     HookTypeCommand,
			Command:  "false",
			Blocking: boolPtr(true),
		},
		{
			Name:    "never-runs",
			Event:   EventStepStart,
			Type:    HookTypeCommand,
			Command: "true",
		},
	}

	emitter := newTestEmitter()
	runner, err := NewHookRunner(hooks, emitter)
	require.NoError(t, err)

	results, err := runner.RunHooks(context.Background(), HookEvent{
		Type:       EventStepStart,
		PipelineID: "test",
		StepID:     "step-1",
	})

	// Should return an error from the blocking hook
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "blocks")

	// Only 2 results: the first succeeds, the second blocks (third never runs)
	require.Len(t, results, 2)
	assert.Equal(t, "succeeds", results[0].HookName)
	assert.Equal(t, DecisionProceed, results[0].Decision)
	assert.Equal(t, "blocks", results[1].HookName)
	assert.Equal(t, DecisionBlock, results[1].Decision)
}

func TestRunnerNonBlockingHookFailureContinues(t *testing.T) {
	hooks := []LifecycleHookDef{
		{
			Name:     "non-blocking-fail",
			Event:    EventRunCompleted,
			Type:     HookTypeCommand,
			Command:  "false",
			Blocking: boolPtr(false),
		},
		{
			Name:     "after-fail",
			Event:    EventRunCompleted,
			Type:     HookTypeCommand,
			Command:  "true",
			Blocking: boolPtr(false),
		},
	}

	emitter := newTestEmitter()
	runner, err := NewHookRunner(hooks, emitter)
	require.NoError(t, err)

	results, err := runner.RunHooks(context.Background(), HookEvent{
		Type:       EventRunCompleted,
		PipelineID: "test",
	})

	// No error because non-blocking failures don't halt
	assert.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, DecisionBlock, results[0].Decision)
	assert.Equal(t, DecisionProceed, results[1].Decision)
}

func TestRunnerFailOpenBlockingHookContinues(t *testing.T) {
	// A blocking hook with fail_open=true should continue when there's an execution error.
	// We simulate this by using an HTTP hook that fails to connect (produces an Err),
	// but with fail_open=true.
	hooks := []LifecycleHookDef{
		{
			Name:     "fail-open-hook",
			Event:    EventStepStart,
			Type:     HookTypeHTTP,
			URL:      "http://127.0.0.1:1", // Will fail to connect
			Blocking: boolPtr(true),
			FailOpen: boolPtr(true),
			Timeout:  "1s",
		},
		{
			Name:    "after-fail-open",
			Event:   EventStepStart,
			Type:    HookTypeCommand,
			Command: "true",
		},
	}

	emitter := newTestEmitter()
	runner, err := NewHookRunner(hooks, emitter)
	require.NoError(t, err)

	results, err := runner.RunHooks(context.Background(), HookEvent{
		Type:       EventStepStart,
		PipelineID: "test",
		StepID:     "step-1",
	})

	// No error because fail-open allows continuation
	assert.NoError(t, err)
	// Both hooks should have results
	require.Len(t, results, 2)
	// The fail-open hook still records its block decision
	assert.Equal(t, DecisionBlock, results[0].Decision)
	assert.NotNil(t, results[0].Err)
	// But the next hook runs
	assert.Equal(t, DecisionProceed, results[1].Decision)
}

func TestRunnerEmptyHooksList(t *testing.T) {
	emitter := newTestEmitter()
	runner, err := NewHookRunner(nil, emitter)
	require.NoError(t, err)

	results, err := runner.RunHooks(context.Background(), HookEvent{
		Type:       EventStepStart,
		PipelineID: "test",
		StepID:     "step-1",
	})

	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestRunnerNilEmitter(t *testing.T) {
	hooks := []LifecycleHookDef{
		{
			Name:    "test-hook",
			Event:   EventStepStart,
			Type:    HookTypeCommand,
			Command: "true",
		},
	}

	// nil emitter should not panic
	runner, err := NewHookRunner(hooks, nil)
	require.NoError(t, err)

	results, err := runner.RunHooks(context.Background(), HookEvent{
		Type:       EventStepStart,
		PipelineID: "test",
		StepID:     "step-1",
	})

	assert.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, DecisionProceed, results[0].Decision)
}

func TestRunnerInvalidMatcher(t *testing.T) {
	hooks := []LifecycleHookDef{
		{
			Name:    "bad-matcher",
			Event:   EventStepStart,
			Type:    HookTypeCommand,
			Command: "true",
			Matcher: "[", // invalid regex
		},
	}

	_, err := NewHookRunner(hooks, newTestEmitter())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bad-matcher")
	assert.Contains(t, err.Error(), "invalid matcher")
}

func TestRunnerDurationIsRecorded(t *testing.T) {
	hooks := []LifecycleHookDef{
		{
			Name:    "slow-hook",
			Event:   EventStepStart,
			Type:    HookTypeCommand,
			Command: "sleep 0.05",
			Timeout: "5s",
		},
	}

	emitter := newTestEmitter()
	runner, err := NewHookRunner(hooks, emitter)
	require.NoError(t, err)

	results, err := runner.RunHooks(context.Background(), HookEvent{
		Type:       EventStepStart,
		PipelineID: "test",
		StepID:     "step-1",
	})

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.True(t, results[0].Duration >= 40*time.Millisecond,
		"duration should be at least ~50ms, got %v", results[0].Duration)
}

func TestRunnerEmitsHookEvents(t *testing.T) {
	hooks := []LifecycleHookDef{
		{
			Name:    "emit-test",
			Event:   EventStepStart,
			Type:    HookTypeCommand,
			Command: "true",
		},
	}

	emitter := newTestEmitter()
	runner, err := NewHookRunner(hooks, emitter)
	require.NoError(t, err)

	_, err = runner.RunHooks(context.Background(), HookEvent{
		Type:       EventStepStart,
		PipelineID: "test-pipeline",
		StepID:     "test-step",
	})
	require.NoError(t, err)

	// Should have emitted hook_started and hook_passed events
	require.GreaterOrEqual(t, len(emitter.events), 2)

	var states []string
	for _, e := range emitter.events {
		states = append(states, e.State)
	}
	assert.Contains(t, states, event.StateHookStarted)
	assert.Contains(t, states, event.StateHookPassed)
}

func TestRunnerRunStartEventNoStepFilter(t *testing.T) {
	// Run-level events (empty stepID) should still match hooks with matchers
	hooks := []LifecycleHookDef{
		{
			Name:    "run-hook-with-matcher",
			Event:   EventRunStart,
			Type:    HookTypeCommand,
			Command: "true",
			Matcher: "implement",
		},
	}

	emitter := newTestEmitter()
	runner, err := NewHookRunner(hooks, emitter)
	require.NoError(t, err)

	// Empty StepID means matcher check is skipped (run-level event)
	results, err := runner.RunHooks(context.Background(), HookEvent{
		Type:       EventRunStart,
		PipelineID: "test",
		StepID:     "", // empty = run-level event
	})
	require.NoError(t, err)
	// Matcher is only applied when stepID != "", so this hook should run
	require.Len(t, results, 1)
	assert.Equal(t, DecisionProceed, results[0].Decision)
}

// readFileContent reads a file and returns its content as a string.
func readFileContent(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
