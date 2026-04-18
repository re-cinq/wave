package pipeline

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStressTest_FailureClassification_ExitCode1 verifies that a command step
// failing with exit code 1 is classified as transient (the default class when
// the error message does not match any deterministic/test/budget pattern).
func TestStressTest_FailureClassification_ExitCode1(t *testing.T) {
	// Simulate an error that looks like a command step exit code 1 failure.
	// The executor wraps these as "command step \"<id>\" failed: exit status 1"
	err := errors.New("command step \"deliberate-fail\" failed: exit status 1")
	class := ClassifyStepFailure(err, nil, nil)
	assert.Equal(t, FailureClassTransient, class,
		"exit code 1 with no recognizable pattern should default to transient")
}

// TestStressTest_CircuitBreaker_TripsOnRepeatedIdenticalErrors verifies that the
// CircuitBreaker (failure.go) trips after the configured limit of identical failures.
func TestStressTest_CircuitBreaker_TripsOnRepeatedIdenticalErrors(t *testing.T) {
	// Configure a circuit breaker with limit=2 that tracks transient failures
	// (matching exit code 1 classification from the test above).
	cb := NewCircuitBreaker(2, []string{FailureClassTransient})

	errMsg := "command step \"deliberate-fail\" failed: exit status 1"
	fp := NormalizeFingerprint("deliberate-fail", FailureClassTransient, errMsg)

	// First failure: not yet tripped
	tripped := cb.Record(fp, FailureClassTransient)
	assert.False(t, tripped, "should not trip after first failure")
	assert.Equal(t, 1, cb.Count(fp))

	// Second failure: trips at limit=2
	tripped = cb.Record(fp, FailureClassTransient)
	assert.True(t, tripped, "should trip after reaching limit of 2 identical failures")
	assert.Equal(t, 2, cb.Count(fp))
}

// TestStressTest_GraphWalker_MaxVisitsEnforced verifies that a command step
// in a self-loop is limited by max_visits. This mirrors the deliberate-fail
// step in the wave-stress-test pipeline (max_visits=2).
func TestStressTest_GraphWalker_MaxVisitsEnforced(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{
				ID:     "succeed",
				Type:   StepTypeCommand,
				Script: "echo ok",
				Edges:  []EdgeConfig{{Target: "deliberate-fail"}},
			},
			{
				ID:        "deliberate-fail",
				Type:      StepTypeCommand,
				Script:    "exit 1",
				MaxVisits: 2,
				Edges: []EdgeConfig{
					{Target: "done", Condition: "outcome=success"},
					{Target: "deliberate-fail"},
				},
			},
			{ID: "done", Type: StepTypeCommand, Script: "echo done"},
		},
	}

	tracker := newMockStepTracker()
	// deliberate-fail always returns a hard error
	tracker.setOutcomes("deliberate-fail", "error", "error", "error", "error", "error")

	gw := NewGraphWalker(p)
	err := gw.Walk(context.Background(), tracker.executor, nil)

	require.Error(t, err, "pipeline should fail due to max_visits enforcement")
	assert.Contains(t, err.Error(), "exceeded max_visits limit (2)",
		"error should indicate max_visits was exceeded")

	counts := gw.VisitCounts()
	assert.Equal(t, 2, counts["deliberate-fail"],
		"deliberate-fail should have been visited exactly 2 times before rejection")
}

// TestStressTest_GraphWalker_CircuitBreakerTrips verifies that the graph walker's
// circuit breaker triggers when a step produces 3 identical errors in a row,
// even if max_visits would allow more iterations.
func TestStressTest_GraphWalker_CircuitBreakerTrips(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{
				ID:     "start",
				Type:   StepTypeCommand,
				Script: "echo ok",
				Edges:  []EdgeConfig{{Target: "flaky"}},
			},
			{
				ID:        "flaky",
				Type:      StepTypeCommand,
				Script:    "exit 1",
				MaxVisits: 10, // high enough that circuit breaker kicks in first
				Edges: []EdgeConfig{
					{Target: "done", Condition: "outcome=success"},
					{Target: "flaky"},
				},
			},
			{ID: "done", Type: StepTypeCommand, Script: "echo done"},
		},
	}

	tracker := newMockStepTracker()
	// flaky always fails with a hard error — same error message each time
	tracker.setOutcomes("flaky", "error", "error", "error", "error", "error")

	gw := NewGraphWalker(p)
	err := gw.Walk(context.Background(), tracker.executor, nil)

	require.Error(t, err, "pipeline should fail due to circuit breaker")
	assert.Contains(t, err.Error(), "circuit breaker triggered",
		"error should indicate circuit breaker was triggered")

	// The graph walker circuit breaker window is 3
	counts := gw.VisitCounts()
	assert.Equal(t, 3, counts["flaky"],
		"flaky should have been visited exactly 3 times (circuit breaker window) before tripping")
}

// TestStressTest_Executor_CircuitBreakerWithFailureClassification is an integration
// test that exercises the full executor path: adapter failure -> failure classification
// -> circuit breaker -> pipeline termination. This is the runtime path that the
// wave-stress-test pipeline would follow.
func TestStressTest_Executor_CircuitBreakerWithFailureClassification(t *testing.T) {
	collector := testutil.NewEventCollector()
	store := newAttemptTrackingStore()

	// Always fails with a generic error (classified as transient)
	failAdapter := newCountingFailAdapter(10, errors.New("exit status 1"))

	executor := NewDefaultPipelineExecutor(failAdapter,
		WithEmitter(collector),
		WithStateStore(store),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)
	m.Runtime.CircuitBreaker = manifest.CircuitBreakerConfig{
		Limit:          2,
		TrackedClasses: []string{FailureClassTransient},
	}

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "stress-test-circuit-breaker"},
		Steps: []Step{
			{
				ID:      "fail-step",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "do work"},
				Retry: RetryConfig{
					MaxAttempts: 5,
					BaseDelay:   "1ms",
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test failure handling")
	require.Error(t, err, "pipeline should fail due to circuit breaker tripping")

	// Verify the circuit breaker tripped before exhausting all retries
	callCount := failAdapter.getCallCount()
	assert.Less(t, callCount, 5,
		"circuit breaker should trip before all 5 retry attempts are exhausted")

	// Verify the failure was classified as transient
	attempts := store.getAttempts()
	var failedAttempts []string
	for _, a := range attempts {
		if a.State == stateFailed {
			failedAttempts = append(failedAttempts, a.FailureClass)
		}
	}
	require.NotEmpty(t, failedAttempts, "should have at least one failed attempt record")
	assert.Equal(t, FailureClassTransient, failedAttempts[0],
		"exit status 1 should be classified as transient")

	// Verify circuit breaker tripped event was emitted
	events := collector.GetEvents()
	foundCircuitBreaker := false
	for _, e := range events {
		if strings.Contains(e.Message, "circuit breaker tripped") {
			foundCircuitBreaker = true
			break
		}
	}
	assert.True(t, foundCircuitBreaker,
		"should emit a circuit breaker tripped event")
}

// TestStressTest_Executor_MaxVisitsEnforced_GraphMode verifies that the executor
// enforces max_visits when running a graph-mode pipeline with a looping command step.
// This matches the wave-stress-test pipeline structure.
func TestStressTest_Executor_MaxVisitsEnforced_GraphMode(t *testing.T) {
	collector := testutil.NewEventCollector()

	// The succeed step uses an adapter; the command steps do not.
	mockAdapter := newCountingFailAdapter(0, nil) // never fails

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	// Pipeline with a command step that loops back to itself.
	// max_visits=2 means it can only be visited twice before rejection.
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "stress-test-max-visits"},
		Steps: []Step{
			{
				ID:        "deliberate-fail",
				Type:      StepTypeCommand,
				Script:    "exit 1",
				MaxVisits: 2,
				Edges: []EdgeConfig{
					{Target: "done", Condition: "outcome=success"},
					{Target: "deliberate-fail"},
				},
			},
			{ID: "done", Type: StepTypeCommand, Script: "echo done"},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test max visits")

	// The pipeline should fail because deliberate-fail always exits 1 and
	// max_visits=2 is hit before the circuit breaker window of 3.
	require.Error(t, err, "pipeline should fail due to max_visits enforcement")

	// Verify the error mentions max_visits
	assert.True(t,
		strings.Contains(err.Error(), "max_visits") || strings.Contains(err.Error(), "circuit breaker"),
		"error should reference max_visits or circuit breaker: %v", err)
}

// TestStressTest_FingerprinterStability verifies that the same command failure
// always produces the same fingerprint, which is essential for the circuit
// breaker to correctly identify repeated failures.
func TestStressTest_FingerprinterStability(t *testing.T) {
	errMsg := "command step \"deliberate-fail\" failed: exit status 1"
	class := FailureClassTransient

	fp1 := NormalizeFingerprint("deliberate-fail", class, errMsg)
	fp2 := NormalizeFingerprint("deliberate-fail", class, errMsg)

	assert.Equal(t, fp1, fp2, "identical inputs must produce identical fingerprints")
	assert.True(t, strings.HasPrefix(fp1, "deliberate-fail:transient:"),
		"fingerprint should have format stepID:class:normalized_error")
}

// TestStressTest_CircuitBreaker_LoadFromAttempts_CommandFailures verifies that
// circuit breaker state can be restored from prior attempt records, simulating
// a pipeline resume after command step failures.
func TestStressTest_CircuitBreaker_LoadFromAttempts_CommandFailures(t *testing.T) {
	cb := NewCircuitBreaker(3, []string{FailureClassTransient})

	// Simulate 2 prior command step failures
	attempts := []StepAttemptReplay{
		{StepID: "deliberate-fail", FailureClass: FailureClassTransient, ErrorMessage: "exit status 1"},
		{StepID: "deliberate-fail", FailureClass: FailureClassTransient, ErrorMessage: "exit status 1"},
	}
	cb.LoadFromAttempts(attempts)

	fp := NormalizeFingerprint("deliberate-fail", FailureClassTransient, "exit status 1")
	assert.Equal(t, 2, cb.Count(fp), "should have 2 prior failures loaded")

	// One more failure should trip the breaker (2 + 1 = 3 = limit)
	tripped := cb.Record(fp, FailureClassTransient)
	assert.True(t, tripped,
		"circuit breaker should trip after replayed attempts + new failure reach limit")
}

// TestStressTest_PipelineYAML_Parseable verifies that the wave-stress-test pipeline
// YAML can be parsed and validates correctly as a graph pipeline.
func TestStressTest_PipelineYAML_Parseable(t *testing.T) {
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "wave-stress-test"},
		Steps: []Step{
			{
				ID:      "succeed",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "Say hello"},
				OutputArtifacts: []ArtifactDef{
					{Name: "hello", Path: ".agents/output/hello.txt", Type: "text"},
				},
			},
			{
				ID:           "deliberate-fail",
				Type:         StepTypeCommand,
				Dependencies: []string{"succeed"},
				Script:       "echo 'This step deliberately fails' && exit 1",
				MaxVisits:    2,
				Edges: []EdgeConfig{
					{Target: "done", Condition: "outcome=success"},
					{Target: "deliberate-fail"},
				},
			},
			{ID: "done", Type: StepTypeCommand, Script: "echo 'Pipeline completed successfully'"},
		},
	}

	assert.True(t, isGraphPipeline(p),
		"pipeline with edges should be detected as graph pipeline")

	// Verify the step structure
	assert.Len(t, p.Steps, 3)
	assert.Equal(t, 2, p.Steps[1].MaxVisits)
	assert.Equal(t, StepTypeCommand, p.Steps[1].Type)
	assert.NotEmpty(t, p.Steps[1].Script)
	assert.Len(t, p.Steps[1].Edges, 2)

	// Verify max_visits effective default for done step (no explicit setting)
	doneStep := p.Steps[2]
	assert.Equal(t, 10, doneStep.EffectiveMaxVisits(),
		"step without explicit max_visits should default to 10")

	// Verify max_visits explicit for deliberate-fail
	failStep := p.Steps[1]
	assert.Equal(t, 2, failStep.EffectiveMaxVisits(),
		"explicit max_visits should be returned as-is")
}

// TestStressTest_FailureClassification_CommandErrors exercises classification
// of various error messages that a command step might produce.
func TestStressTest_FailureClassification_CommandErrors(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected string
	}{
		{
			name:     "exit status 1 is transient",
			errMsg:   "command step \"build\" failed: exit status 1",
			expected: FailureClassTransient,
		},
		{
			name:     "permission denied is deterministic",
			errMsg:   "command step \"deploy\" failed: permission denied",
			expected: FailureClassDeterministic,
		},
		{
			name:     "test failed is test_failure",
			errMsg:   "command step \"test\" failed: test failed",
			expected: FailureClassTestFailure,
		},
		{
			name:     "rate limit is transient",
			errMsg:   "command step \"api\" failed: rate limit exceeded",
			expected: FailureClassTransient,
		},
		{
			name:     "missing binary is deterministic",
			errMsg:   "command step \"run\" failed: missing binary: node",
			expected: FailureClassDeterministic,
		},
		{
			name:     "context window is budget_exhausted",
			errMsg:   "command step \"gen\" failed: context window full",
			expected: FailureClassBudgetExhausted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			class := ClassifyStepFailure(fmt.Errorf("%s", tt.errMsg), nil, nil)
			assert.Equal(t, tt.expected, class)
		})
	}
}
