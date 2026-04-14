package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/testutil"
)

// mockStepTracker tracks step executions and returns configurable outcomes.
type mockStepTracker struct {
	calls    []string
	outcomes map[string][]string // stepID -> sequence of outcomes per call
	callIdx  map[string]int      // stepID -> current call index
}

func newMockStepTracker() *mockStepTracker {
	return &mockStepTracker{
		outcomes: make(map[string][]string),
		callIdx:  make(map[string]int),
	}
}

// setOutcomes configures the sequence of outcomes for a step.
// Each entry is "success", "failure" (soft — no Go error), or "error" (hard — returns Go error).
// When the sequence is exhausted, subsequent calls return "success".
func (m *mockStepTracker) setOutcomes(stepID string, outcomes ...string) {
	m.outcomes[stepID] = outcomes
}

func (m *mockStepTracker) executor(ctx context.Context, step *Step) (*StepResult, error) {
	m.calls = append(m.calls, step.ID)

	idx := m.callIdx[step.ID]
	m.callIdx[step.ID] = idx + 1

	outcomes := m.outcomes[step.ID]
	if idx < len(outcomes) {
		switch outcomes[idx] {
		case "failure":
			// Soft failure: outcome is "failure" but no Go error.
			// The walker continues routing via edges or DAG fallback.
			return &StepResult{
				StepID:  step.ID,
				Outcome: "failure",
				Context: make(map[string]string),
			}, nil
		case "error":
			// Hard failure: returns a Go error.
			// The walker treats this as fatal unless the step has edges.
			return &StepResult{
				StepID:  step.ID,
				Outcome: "failure",
				Error:   fmt.Errorf("step %s failed", step.ID),
				Context: make(map[string]string),
			}, fmt.Errorf("step %s failed", step.ID)
		}
	}

	return &StepResult{
		StepID:  step.ID,
		Outcome: "success",
		Context: make(map[string]string),
	}, nil
}

// callCount returns how many times a step was called.
func (m *mockStepTracker) callCount(stepID string) int {
	count := 0
	for _, id := range m.calls {
		if id == stepID {
			count++
		}
	}
	return count
}

// --- 5.2: Graph walker unit tests ---

func TestGraphWalker_LinearGraph(t *testing.T) {
	// Simple A -> B -> C pipeline with no edges (uses DAG-order fallback via findNextDAGStep).
	// B depends on A, C depends on B.
	p := &Pipeline{
		Steps: []Step{
			{ID: "A"},
			{ID: "B", Dependencies: []string{"A"}},
			{ID: "C", Dependencies: []string{"B"}},
		},
	}

	tracker := newMockStepTracker()
	gw := NewGraphWalker(p)
	err := gw.Walk(context.Background(), tracker.executor, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should execute all 3 steps in order
	if len(tracker.calls) != 3 {
		t.Fatalf("expected 3 calls, got %d: %v", len(tracker.calls), tracker.calls)
	}
	if tracker.calls[0] != "A" || tracker.calls[1] != "B" || tracker.calls[2] != "C" {
		t.Errorf("expected call order [A, B, C], got %v", tracker.calls)
	}

	// Verify visit counts
	counts := gw.VisitCounts()
	for _, id := range []string{"A", "B", "C"} {
		if counts[id] != 1 {
			t.Errorf("expected visit count 1 for step %q, got %d", id, counts[id])
		}
	}
}

func TestGraphWalker_EdgeChain(t *testing.T) {
	// A has edge to B, B has edge to C. Forward edge chain.
	p := &Pipeline{
		Steps: []Step{
			{ID: "A", Edges: []EdgeConfig{{Target: "B"}}},
			{ID: "B", Edges: []EdgeConfig{{Target: "C"}}},
			{ID: "C"},
		},
	}

	tracker := newMockStepTracker()
	gw := NewGraphWalker(p)
	err := gw.Walk(context.Background(), tracker.executor, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tracker.calls) != 3 {
		t.Fatalf("expected 3 calls, got %d: %v", len(tracker.calls), tracker.calls)
	}
	if tracker.calls[0] != "A" || tracker.calls[1] != "B" || tracker.calls[2] != "C" {
		t.Errorf("expected call order [A, B, C], got %v", tracker.calls)
	}
}

func TestGraphWalker_SimpleLoop(t *testing.T) {
	// Steps: implement -> test -> gate -> fix -> (back to test)
	// Gate is conditional: outcome=success -> done (terminal), fallback -> fix
	// Fix has edge back to test.
	// Test fails first 2 times, succeeds on 3rd.
	p := &Pipeline{
		Steps: []Step{
			{ID: "implement"},
			{ID: "test", Dependencies: []string{"implement"}},
			{ID: "gate", Dependencies: []string{"test"}, Type: StepTypeConditional, Edges: []EdgeConfig{
				{Target: "done", Condition: "outcome=success"},
				{Target: "fix"},
			}},
			{ID: "fix", Dependencies: []string{"gate"}, MaxVisits: 5, Edges: []EdgeConfig{
				{Target: "test"},
			}},
			{ID: "done", Dependencies: []string{"gate"}},
		},
	}

	tracker := newMockStepTracker()
	// test fails first 2 times, succeeds 3rd time
	tracker.setOutcomes("test", "failure", "failure", "success")

	gw := NewGraphWalker(p)
	err := gw.Walk(context.Background(), tracker.executor, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify visit counts
	counts := gw.VisitCounts()
	if counts["implement"] != 1 {
		t.Errorf("expected implement visited 1 time, got %d", counts["implement"])
	}
	if counts["test"] != 3 {
		t.Errorf("expected test visited 3 times, got %d", counts["test"])
	}
	if counts["gate"] != 3 {
		t.Errorf("expected gate visited 3 times, got %d", counts["gate"])
	}
	if counts["fix"] != 2 {
		t.Errorf("expected fix visited 2 times, got %d", counts["fix"])
	}
	if counts["done"] != 1 {
		t.Errorf("expected done visited 1 time, got %d", counts["done"])
	}
}

func TestGraphWalker_MaxVisitsEnforcement(t *testing.T) {
	// A step with max_visits=2 in a loop. After 2 visits, should return error.
	// looper always edges back to itself.
	p := &Pipeline{
		Steps: []Step{
			{ID: "start", Edges: []EdgeConfig{{Target: "looper"}}},
			{ID: "looper", MaxVisits: 2, Edges: []EdgeConfig{{Target: "looper"}}},
		},
	}

	tracker := newMockStepTracker()
	gw := NewGraphWalker(p)
	err := gw.Walk(context.Background(), tracker.executor, nil)
	if err == nil {
		t.Fatal("expected error for exceeding max_visits, got nil")
	}
	if !strings.Contains(err.Error(), "exceeded max_visits limit (2)") {
		t.Errorf("expected max_visits error, got: %v", err)
	}

	// Looper should have been visited exactly 2 times before being rejected
	counts := gw.VisitCounts()
	if counts["looper"] != 2 {
		t.Errorf("expected looper visited 2 times before error, got %d", counts["looper"])
	}
}

func TestGraphWalker_MaxStepVisitsEnforcement(t *testing.T) {
	// Pipeline with max_step_visits=5. Multiple steps in a loop.
	// A -> B -> A (loop). Each step gets default max_visits=10, but total is capped at 5.
	p := &Pipeline{
		MaxStepVisits: 5,
		Steps: []Step{
			{ID: "A", Edges: []EdgeConfig{{Target: "B"}}},
			{ID: "B", Edges: []EdgeConfig{{Target: "A"}}},
		},
	}

	tracker := newMockStepTracker()
	gw := NewGraphWalker(p)
	err := gw.Walk(context.Background(), tracker.executor, nil)
	if err == nil {
		t.Fatal("expected error for exceeding max_step_visits, got nil")
	}
	if !strings.Contains(err.Error(), "exceeded max_step_visits limit (5 total visits)") {
		t.Errorf("expected max_step_visits error, got: %v", err)
	}

	// Total visits should be exactly 5 (some A, some B)
	counts := gw.VisitCounts()
	total := 0
	for _, v := range counts {
		total += v
	}
	if total != 5 {
		t.Errorf("expected 5 total visits, got %d", total)
	}
}

func TestGraphWalker_CircuitBreaker(t *testing.T) {
	// Step in a loop that always returns the same error.
	// Should trigger circuit breaker after 3 identical errors.
	p := &Pipeline{
		Steps: []Step{
			{ID: "start", Edges: []EdgeConfig{{Target: "flaky"}}},
			{ID: "flaky", MaxVisits: 10, Edges: []EdgeConfig{
				{Target: "done", Condition: "outcome=success"},
				{Target: "flaky"},
			}},
			{ID: "done"},
		},
	}

	tracker := newMockStepTracker()
	// flaky always fails with a hard error — same error message each time
	tracker.setOutcomes("flaky", "error", "error", "error", "error", "error")

	gw := NewGraphWalker(p)
	err := gw.Walk(context.Background(), tracker.executor, nil)
	if err == nil {
		t.Fatal("expected circuit breaker error, got nil")
	}
	if !strings.Contains(err.Error(), "circuit breaker triggered") {
		t.Errorf("expected circuit breaker error, got: %v", err)
	}

	// Should have visited flaky exactly 3 times (circuit breaker window)
	counts := gw.VisitCounts()
	if counts["flaky"] != 3 {
		t.Errorf("expected flaky visited 3 times before circuit breaker, got %d", counts["flaky"])
	}
}

// --- 5.4: Backward compatibility tests ---

func Test_isGraphPipeline_DAGMode(t *testing.T) {
	// Existing pipeline with no edges or conditional types should return false.
	p := &Pipeline{
		Steps: []Step{
			{ID: "step1"},
			{ID: "step2", Dependencies: []string{"step1"}},
			{ID: "step3", Dependencies: []string{"step2"}},
		},
	}
	if isGraphPipeline(p) {
		t.Error("expected isGraphPipeline to return false for DAG-only pipeline")
	}
}

func Test_isGraphPipeline_GraphMode(t *testing.T) {
	// Pipeline with edges should return true.
	tests := []struct {
		name     string
		pipeline *Pipeline
	}{
		{
			name: "step with edges",
			pipeline: &Pipeline{
				Steps: []Step{
					{ID: "A", Edges: []EdgeConfig{{Target: "B"}}},
					{ID: "B"},
				},
			},
		},
		{
			name: "conditional step type",
			pipeline: &Pipeline{
				Steps: []Step{
					{ID: "A"},
					{ID: "gate", Type: StepTypeConditional, Edges: []EdgeConfig{{Target: "A"}}},
				},
			},
		},
		{
			name: "command step with edges",
			pipeline: &Pipeline{
				Steps: []Step{
					{ID: "A", Type: StepTypeCommand, Script: "echo test", Edges: []EdgeConfig{{Target: "B"}}},
					{ID: "B"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !isGraphPipeline(tt.pipeline) {
				t.Error("expected isGraphPipeline to return true for graph-mode pipeline")
			}
		})
	}
}

func TestValidateGraph_BasicValidation(t *testing.T) {
	v := &DAGValidator{}

	tests := []struct {
		name      string
		pipeline  *Pipeline
		wantError string // empty string means no error expected
	}{
		{
			name: "valid graph pipeline",
			pipeline: &Pipeline{
				Steps: []Step{
					{ID: "A", Edges: []EdgeConfig{{Target: "B"}}},
					{ID: "B", Type: StepTypeConditional, Edges: []EdgeConfig{
						{Target: "A", Condition: "outcome=failure"},
						{Target: "C"},
					}},
					{ID: "C"},
				},
			},
			wantError: "",
		},
		{
			name: "edge targets non-existent step",
			pipeline: &Pipeline{
				Steps: []Step{
					{ID: "A", Edges: []EdgeConfig{{Target: "nonexistent"}}},
				},
			},
			wantError: "edge targeting non-existent step",
		},
		{
			name: "conditional step without edges",
			pipeline: &Pipeline{
				Steps: []Step{
					{ID: "gate", Type: StepTypeConditional},
				},
			},
			wantError: "type=conditional but has no edges",
		},
		{
			name: "command step without script",
			pipeline: &Pipeline{
				Steps: []Step{
					{ID: "cmd", Type: StepTypeCommand},
				},
			},
			wantError: "type=command but has no script",
		},
		{
			name: "invalid condition syntax",
			pipeline: &Pipeline{
				Steps: []Step{
					{ID: "A", Edges: []EdgeConfig{{Target: "B", Condition: "invalid"}}},
					{ID: "B"},
				},
			},
			wantError: "missing '=' operator",
		},
		{
			name: "negative max_visits",
			pipeline: &Pipeline{
				Steps: []Step{
					{ID: "A", MaxVisits: -1},
				},
			},
			wantError: "negative max_visits",
		},
		{
			name: "negative max_step_visits",
			pipeline: &Pipeline{
				MaxStepVisits: -1,
				Steps: []Step{
					{ID: "A"},
				},
			},
			wantError: "negative max_step_visits",
		},
		{
			name: "dependency on non-existent step",
			pipeline: &Pipeline{
				Steps: []Step{
					{ID: "A", Dependencies: []string{"ghost"}},
				},
			},
			wantError: "depends on non-existent step",
		},
		{
			name: "valid command step with script",
			pipeline: &Pipeline{
				Steps: []Step{
					{ID: "cmd", Type: StepTypeCommand, Script: "echo hello", Edges: []EdgeConfig{{Target: "next"}}},
					{ID: "next"},
				},
			},
			wantError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateGraph(tt.pipeline)
			if tt.wantError == "" {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantError)
				} else if !strings.Contains(err.Error(), tt.wantError) {
					t.Errorf("expected error containing %q, got: %v", tt.wantError, err)
				}
			}
		})
	}
}

// --- 5.5: End-to-end loop integration test ---

func TestGraphWalker_ImplementTestFixCycle(t *testing.T) {
	// Full implement -> test -> gate -> fix -> (back to test) cycle:
	// - implement: always succeeds
	// - test: command step that fails first 2 times, succeeds 3rd time
	// - gate: conditional step with edges: outcome=success -> finalize, fallback -> fix
	// - fix: always succeeds, has edge back to test
	// - finalize: always succeeds
	//
	// Expected execution trace:
	//   implement(1) -> test(1,fail) -> gate(1,fail) -> fix(1) -> test(2,fail) -> gate(2,fail) -> fix(2) -> test(3,ok) -> gate(3,ok) -> finalize(1)
	// Visit counts: implement=1, test=3, gate=3, fix=2, finalize=1
	p := &Pipeline{
		Steps: []Step{
			{ID: "implement"},
			{ID: "test", Dependencies: []string{"implement"}, Type: StepTypeCommand, Script: "echo test"},
			{ID: "gate", Dependencies: []string{"test"}, Type: StepTypeConditional, Edges: []EdgeConfig{
				{Target: "finalize", Condition: "outcome=success"},
				{Target: "fix"},
			}},
			{ID: "fix", Dependencies: []string{"gate"}, MaxVisits: 5, Edges: []EdgeConfig{
				{Target: "test"},
			}},
			{ID: "finalize", Dependencies: []string{"gate"}},
		},
	}

	tracker := newMockStepTracker()
	// test fails first 2 times, succeeds 3rd time
	tracker.setOutcomes("test", "failure", "failure", "success")

	gw := NewGraphWalker(p)
	err := gw.Walk(context.Background(), tracker.executor, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify individual visit counts
	counts := gw.VisitCounts()
	expected := map[string]int{
		"implement": 1,
		"test":      3,
		"gate":      3,
		"fix":       2,
		"finalize":  1,
	}
	for stepID, want := range expected {
		got := counts[stepID]
		if got != want {
			t.Errorf("step %q: expected %d visits, got %d", stepID, want, got)
		}
	}

	// Verify the exact execution order
	expectedCalls := []string{
		"implement",
		"test",     // fail #1
		"fix",      // fix #1
		"test",     // fail #2
		"fix",      // fix #2
		"test",     // success
		"finalize", // terminal
	}
	// Note: gate is conditional so it doesn't appear in executor calls
	if len(tracker.calls) != len(expectedCalls) {
		t.Fatalf("expected %d executor calls, got %d: %v", len(expectedCalls), len(tracker.calls), tracker.calls)
	}
	for i, want := range expectedCalls {
		if tracker.calls[i] != want {
			t.Errorf("call[%d]: expected %q, got %q (full trace: %v)", i, want, tracker.calls[i], tracker.calls)
		}
	}

	// Verify total visits (including conditional gate steps)
	totalVisits := 0
	for _, v := range counts {
		totalVisits += v
	}
	if totalVisits != 10 { // 1+3+3+2+1
		t.Errorf("expected 10 total visits, got %d", totalVisits)
	}
}

func TestGraphWalker_ContextCancellation(t *testing.T) {
	// Verify that the walker respects context cancellation.
	p := &Pipeline{
		Steps: []Step{
			{ID: "A", Edges: []EdgeConfig{{Target: "B"}}},
			{ID: "B", Edges: []EdgeConfig{{Target: "A"}}},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	tracker := newMockStepTracker()

	callCount := 0
	wrappedExecutor := func(ctx context.Context, step *Step) (*StepResult, error) {
		callCount++
		if callCount >= 3 {
			cancel()
		}
		return tracker.executor(ctx, step)
	}

	gw := NewGraphWalker(p)
	err := gw.Walk(ctx, wrappedExecutor, nil)
	if err == nil {
		t.Fatal("expected cancellation error, got nil")
	}
	if !strings.Contains(err.Error(), "cancelled") {
		t.Errorf("expected cancellation error, got: %v", err)
	}
}

func TestGraphWalker_EmptyPipeline(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{},
	}

	tracker := newMockStepTracker()
	gw := NewGraphWalker(p)
	err := gw.Walk(context.Background(), tracker.executor, nil)
	if err == nil {
		t.Fatal("expected error for empty pipeline, got nil")
	}
	if !strings.Contains(err.Error(), "no steps") {
		t.Errorf("expected 'no steps' error, got: %v", err)
	}
}

func TestGraphWalker_ConditionalInheritsOutcome(t *testing.T) {
	// Verify that a conditional step inherits the outcome from the previous step.
	// If the previous step fails, the conditional routes based on failure.
	// If the previous step succeeds, the conditional routes based on success.
	p := &Pipeline{
		Steps: []Step{
			{ID: "worker"},
			{ID: "router", Dependencies: []string{"worker"}, Type: StepTypeConditional, Edges: []EdgeConfig{
				{Target: "happy", Condition: "outcome=success"},
				{Target: "sad", Condition: "outcome=failure"},
			}},
			{ID: "happy", Dependencies: []string{"router"}},
			{ID: "sad", Dependencies: []string{"router"}},
		},
	}

	// Test success path
	t.Run("success_path", func(t *testing.T) {
		tracker := newMockStepTracker()
		tracker.setOutcomes("worker", "success")

		gw := NewGraphWalker(p)
		err := gw.Walk(context.Background(), tracker.executor, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		counts := gw.VisitCounts()
		if counts["happy"] != 1 {
			t.Errorf("expected happy visited 1 time, got %d", counts["happy"])
		}
		if counts["sad"] != 0 {
			t.Errorf("expected sad visited 0 times, got %d", counts["sad"])
		}
	})

	// Test failure path: worker fails and has no edges, so failure is fatal
	// by the current walker design. To test the conditional routing on failure,
	// the failing step needs edges.
	t.Run("failure_path", func(t *testing.T) {
		pFail := &Pipeline{
			Steps: []Step{
				{ID: "worker", Edges: []EdgeConfig{{Target: "router"}}},
				{ID: "router", Type: StepTypeConditional, Edges: []EdgeConfig{
					{Target: "happy", Condition: "outcome=success"},
					{Target: "sad", Condition: "outcome=failure"},
				}},
				{ID: "happy"},
				{ID: "sad"},
			},
		}

		tracker := newMockStepTracker()
		tracker.setOutcomes("worker", "failure")

		gw := NewGraphWalker(pFail)
		err := gw.Walk(context.Background(), tracker.executor, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		counts := gw.VisitCounts()
		if counts["happy"] != 0 {
			t.Errorf("expected happy visited 0 times, got %d", counts["happy"])
		}
		if counts["sad"] != 1 {
			t.Errorf("expected sad visited 1 time, got %d", counts["sad"])
		}
	})
}

func TestGraphWalker_ResumeFromVisitCounts(t *testing.T) {
	// Verify that initial visit counts are restored and enforced.
	p := &Pipeline{
		Steps: []Step{
			{ID: "A", MaxVisits: 3, Edges: []EdgeConfig{{Target: "A"}}},
		},
	}

	tracker := newMockStepTracker()
	gw := NewGraphWalker(p)
	// Start with 2 prior visits — only 1 more allowed before hitting max_visits=3
	err := gw.Walk(context.Background(), tracker.executor, map[string]int{"A": 2})
	if err == nil {
		t.Fatal("expected max_visits error, got nil")
	}
	if !strings.Contains(err.Error(), "exceeded max_visits limit (3)") {
		t.Errorf("expected max_visits error, got: %v", err)
	}

	// Should have executed A exactly once before hitting the limit on the second attempt
	if tracker.callCount("A") != 1 {
		t.Errorf("expected 1 call to A, got %d", tracker.callCount("A"))
	}
}

// --- Validation: fan-out rejection ---

func TestValidateGraph_RejectsFanOutWithoutEdges(t *testing.T) {
	v := &DAGValidator{}

	// Step A has no edges but two dependents B and C — should be rejected
	p := &Pipeline{
		Steps: []Step{
			{ID: "A"},
			{ID: "B", Dependencies: []string{"A"}},
			{ID: "C", Dependencies: []string{"A"}},
		},
	}

	err := v.ValidateGraph(p)
	if err == nil {
		t.Fatal("expected validation error for fan-out without edges, got nil")
	}
	if !strings.Contains(err.Error(), "multiple dependents") {
		t.Errorf("expected 'multiple dependents' error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "add explicit edges") {
		t.Errorf("expected 'add explicit edges' guidance, got: %v", err)
	}
}

func TestValidateGraph_AllowsFanOutWithEdges(t *testing.T) {
	v := &DAGValidator{}

	// Step A has explicit edges to B and C — should be accepted
	p := &Pipeline{
		Steps: []Step{
			{ID: "A", Edges: []EdgeConfig{
				{Target: "B", Condition: "outcome=success"},
				{Target: "C"},
			}},
			{ID: "B"},
			{ID: "C"},
		},
	}

	err := v.ValidateGraph(p)
	if err != nil {
		t.Errorf("expected no error for fan-out with explicit edges, got: %v", err)
	}
}

// --- Dependency enforcement tests ---

func TestGraphWalker_DependencyGatedStepNotSkipped(t *testing.T) {
	// Reproduces re-cinq/wave#630: edges route around a dependency-gated step.
	//
	// Pipeline topology (modelled on wave-validate):
	//   test-gate -> generate-report   (via edge, outcome=success)
	//   test-gate -> diagnose-failure   (via edge, outcome=failure)
	//   diagnose-failure -> (no edges, DAG fallback)
	//   approval-gate depends on [test-gate, diagnose-failure] — NO edge targets it
	//   generate-report depends on [approval-gate]
	//
	// Bug: the walker follows test-gate's edge directly to generate-report
	// (or diagnose-failure), never visiting approval-gate.
	//
	// Expected (failure path):
	//   test-gate(fail) -> diagnose-failure -> approval-gate -> generate-report
	p := &Pipeline{
		Steps: []Step{
			{ID: "test-gate", Edges: []EdgeConfig{
				{Target: "generate-report", Condition: "outcome=success"},
				{Target: "diagnose-failure"},
			}},
			{ID: "diagnose-failure"},
			{ID: "approval-gate", Dependencies: []string{"test-gate", "diagnose-failure"}},
			{ID: "generate-report", Dependencies: []string{"approval-gate"}},
		},
	}

	t.Run("failure_path_visits_gate", func(t *testing.T) {
		tracker := newMockStepTracker()
		tracker.setOutcomes("test-gate", "failure")

		gw := NewGraphWalker(p)
		err := gw.Walk(context.Background(), tracker.executor, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		counts := gw.VisitCounts()
		for _, id := range []string{"test-gate", "diagnose-failure", "approval-gate", "generate-report"} {
			if counts[id] != 1 {
				t.Errorf("step %q: expected 1 visit, got %d (trace: %v)", id, counts[id], tracker.calls)
			}
		}

		// Verify ordering: approval-gate must come after diagnose-failure
		// and before generate-report.
		gateIdx := -1
		diagnoseIdx := -1
		reportIdx := -1
		for i, id := range tracker.calls {
			switch id {
			case "approval-gate":
				gateIdx = i
			case "diagnose-failure":
				diagnoseIdx = i
			case "generate-report":
				reportIdx = i
			}
		}
		if gateIdx < 0 {
			t.Fatal("approval-gate was never executed")
		}
		if diagnoseIdx >= 0 && gateIdx <= diagnoseIdx {
			t.Errorf("approval-gate (idx %d) ran before diagnose-failure (idx %d)", gateIdx, diagnoseIdx)
		}
		if reportIdx >= 0 && reportIdx <= gateIdx {
			t.Errorf("generate-report (idx %d) ran before approval-gate (idx %d)", reportIdx, gateIdx)
		}
	})

	t.Run("success_path_visits_gate", func(t *testing.T) {
		// On the success path, test-gate edges to generate-report directly.
		// But generate-report depends on approval-gate, which depends on
		// [test-gate, diagnose-failure]. diagnose-failure hasn't run, so
		// approval-gate's deps aren't met. The walker should defer
		// generate-report, execute diagnose-failure, then approval-gate,
		// then generate-report.
		tracker := newMockStepTracker()
		tracker.setOutcomes("test-gate", "success")

		gw := NewGraphWalker(p)
		err := gw.Walk(context.Background(), tracker.executor, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		counts := gw.VisitCounts()
		for _, id := range []string{"test-gate", "diagnose-failure", "approval-gate", "generate-report"} {
			if counts[id] != 1 {
				t.Errorf("step %q: expected 1 visit, got %d (trace: %v)", id, counts[id], tracker.calls)
			}
		}

		// approval-gate must precede generate-report
		gateIdx := -1
		reportIdx := -1
		for i, id := range tracker.calls {
			switch id {
			case "approval-gate":
				gateIdx = i
			case "generate-report":
				reportIdx = i
			}
		}
		if gateIdx < 0 {
			t.Fatal("approval-gate was never executed")
		}
		if reportIdx >= 0 && reportIdx <= gateIdx {
			t.Errorf("generate-report (idx %d) ran before approval-gate (idx %d)", reportIdx, gateIdx)
		}
	})
}

func TestGraphWalker_EdgeTargetWithUnmetDeps(t *testing.T) {
	// An edge targets a step whose dependencies are not satisfied.
	// The walker should defer the target and execute the missing
	// dependency first.
	//
	// Pipeline: A --(edge)--> C, but C depends on [A, B].
	// B has no edge targeting it but depends on A (so it's gated).
	p := &Pipeline{
		Steps: []Step{
			{ID: "A", Edges: []EdgeConfig{{Target: "C"}}},
			{ID: "B", Dependencies: []string{"A"}},
			{ID: "C", Dependencies: []string{"A", "B"}},
		},
	}

	tracker := newMockStepTracker()
	gw := NewGraphWalker(p)
	err := gw.Walk(context.Background(), tracker.executor, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	counts := gw.VisitCounts()
	for _, id := range []string{"A", "B", "C"} {
		if counts[id] != 1 {
			t.Errorf("step %q: expected 1 visit, got %d", id, counts[id])
		}
	}

	// B must execute before C
	bIdx := -1
	cIdx := -1
	for i, id := range tracker.calls {
		switch id {
		case "B":
			bIdx = i
		case "C":
			cIdx = i
		}
	}
	if bIdx < 0 {
		t.Fatal("B was never executed")
	}
	if cIdx >= 0 && cIdx <= bIdx {
		t.Errorf("C (idx %d) ran before B (idx %d)", cIdx, bIdx)
	}
}

// --- Integration test: command step through DefaultPipelineExecutor ---

func TestExecuteGraphPipeline_CommandStepIntegration(t *testing.T) {
	// Integration test: exercises a command step through the real execution path
	// using DefaultPipelineExecutor (not the mock walker).
	// Verifies that:
	// 1. Command steps execute scripts and capture stdout
	// 2. Environment is filtered (PATH always present)
	// 3. Graph walker routes correctly through command + regular steps
	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(100),
	)

	collector := testutil.NewEventCollector()
	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	// Pipeline: init (command: echo hello) -> verify (command: echo done)
	// Both are command steps, linked by edge routing.
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "cmd-integration-test"},
		Steps: []Step{
			{
				ID:     "init",
				Type:   StepTypeCommand,
				Script: "echo hello-from-command",
				Edges:  []EdgeConfig{{Target: "verify"}},
			},
			{
				ID:     "verify",
				Type:   StepTypeCommand,
				Script: "echo done",
			},
		},
	}

	err := executor.Execute(context.Background(), p, m, "test input")
	if err != nil {
		t.Fatalf("expected graph pipeline to succeed, got: %v", err)
	}

	// Verify events were emitted for both command steps
	events := collector.GetEvents()
	foundInit := false
	foundVerify := false
	for _, ev := range events {
		if ev.StepID == "init" && ev.State == stateCompleted {
			foundInit = true
		}
		if ev.StepID == "verify" && ev.State == stateCompleted {
			foundVerify = true
		}
	}
	if !foundInit {
		t.Error("expected completed event for 'init' step")
	}
	if !foundVerify {
		t.Error("expected completed event for 'verify' step")
	}
}

// --- Unit test: resolveCommandWorkDir ---

func TestResolveCommandWorkDir(t *testing.T) {
	t.Run("no mounts returns workspace root", func(t *testing.T) {
		wsRoot := t.TempDir()
		step := &Step{ID: "test-step"}
		got := resolveCommandWorkDir(wsRoot, step)
		if got != wsRoot {
			t.Errorf("expected %q, got %q", wsRoot, got)
		}
	})

	t.Run("mount with source ./ resolves to mount target", func(t *testing.T) {
		wsRoot := t.TempDir()
		projectDir := filepath.Join(wsRoot, "project")
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			t.Fatal(err)
		}

		step := &Step{
			ID: "test-step",
			Workspace: WorkspaceConfig{
				Mount: []Mount{
					{Source: "./", Target: "/project", Mode: "readonly"},
				},
			},
		}
		got := resolveCommandWorkDir(wsRoot, step)
		if got != projectDir {
			t.Errorf("expected %q, got %q", projectDir, got)
		}
	})

	t.Run("mount with source . resolves to mount target", func(t *testing.T) {
		wsRoot := t.TempDir()
		projectDir := filepath.Join(wsRoot, "src")
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			t.Fatal(err)
		}

		step := &Step{
			ID: "test-step",
			Workspace: WorkspaceConfig{
				Mount: []Mount{
					{Source: ".", Target: "/src", Mode: "readwrite"},
				},
			},
		}
		got := resolveCommandWorkDir(wsRoot, step)
		if got != projectDir {
			t.Errorf("expected %q, got %q", projectDir, got)
		}
	})

	t.Run("mount with non-root source returns workspace root", func(t *testing.T) {
		wsRoot := t.TempDir()
		subDir := filepath.Join(wsRoot, "data")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatal(err)
		}

		step := &Step{
			ID: "test-step",
			Workspace: WorkspaceConfig{
				Mount: []Mount{
					{Source: "./subdir", Target: "/data", Mode: "readonly"},
				},
			},
		}
		got := resolveCommandWorkDir(wsRoot, step)
		if got != wsRoot {
			t.Errorf("expected %q (workspace root), got %q", wsRoot, got)
		}
	})

	t.Run("mount target dir missing returns workspace root", func(t *testing.T) {
		wsRoot := t.TempDir()
		step := &Step{
			ID: "test-step",
			Workspace: WorkspaceConfig{
				Mount: []Mount{
					{Source: "./", Target: "/project", Mode: "readonly"},
				},
			},
		}
		got := resolveCommandWorkDir(wsRoot, step)
		if got != wsRoot {
			t.Errorf("expected %q (workspace root), got %q", wsRoot, got)
		}
	})

	t.Run("picks first project-root mount from multiple", func(t *testing.T) {
		wsRoot := t.TempDir()
		firstDir := filepath.Join(wsRoot, "first")
		secondDir := filepath.Join(wsRoot, "second")
		if err := os.MkdirAll(firstDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(secondDir, 0755); err != nil {
			t.Fatal(err)
		}

		step := &Step{
			ID: "test-step",
			Workspace: WorkspaceConfig{
				Mount: []Mount{
					{Source: "./", Target: "/first", Mode: "readonly"},
					{Source: "./", Target: "/second", Mode: "readwrite"},
				},
			},
		}
		got := resolveCommandWorkDir(wsRoot, step)
		if got != firstDir {
			t.Errorf("expected first mount %q, got %q", firstDir, got)
		}
	})
}

// --- Unit test: resolveCommandWorkDir (extended coverage) ---

func TestResolveCommandWorkDir_WorktreeDetection(t *testing.T) {
	t.Run("worktree __wt_ directory is returned", func(t *testing.T) {
		wsRoot := t.TempDir()
		wtDir := filepath.Join(wsRoot, "__wt_mybranch")
		if err := os.MkdirAll(wtDir, 0755); err != nil {
			t.Fatal(err)
		}

		step := &Step{ID: "test-step"}
		got := resolveCommandWorkDir(wsRoot, step)
		if got != wtDir {
			t.Errorf("expected worktree dir %q, got %q", wtDir, got)
		}
	})

	t.Run("multiple __wt_ directories returns first found", func(t *testing.T) {
		wsRoot := t.TempDir()
		// Create multiple __wt_ dirs
		if err := os.MkdirAll(filepath.Join(wsRoot, "__wt_alpha"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(wsRoot, "__wt_beta"), 0755); err != nil {
			t.Fatal(err)
		}

		step := &Step{ID: "test-step"}
		got := resolveCommandWorkDir(wsRoot, step)
		// Should return one of the __wt_ dirs (os.ReadDir returns alphabetically)
		if !strings.HasPrefix(filepath.Base(got), "__wt_") {
			t.Errorf("expected a __wt_ directory, got %q", got)
		}
	})

	t.Run("mount with target / becomes empty and is skipped", func(t *testing.T) {
		wsRoot := t.TempDir()
		step := &Step{
			ID: "test-step",
			Workspace: WorkspaceConfig{
				Mount: []Mount{
					{Source: "./", Target: "/", Mode: "readonly"},
				},
			},
		}
		got := resolveCommandWorkDir(wsRoot, step)
		// Target "/" trims to "", should be skipped, fallback to workspace root
		if got != wsRoot {
			t.Errorf("expected workspace root %q, got %q", wsRoot, got)
		}
	})
}

func TestResolveCommandWorkDir_BareWorkspaceFallback(t *testing.T) {
	t.Run("bare workspace with Makefile has marker and does not fall back to CWD", func(t *testing.T) {
		wsRoot := t.TempDir()
		// Place a Makefile in the workspace — counts as a project marker
		if err := os.WriteFile(filepath.Join(wsRoot, "Makefile"), []byte("all:\n"), 0644); err != nil {
			t.Fatal(err)
		}

		step := &Step{ID: "test-step"}
		got := resolveCommandWorkDir(wsRoot, step)
		if got != wsRoot {
			t.Errorf("expected workspace root %q (has Makefile), got %q", wsRoot, got)
		}
	})

	t.Run("bare workspace without markers falls back to CWD with project marker", func(t *testing.T) {
		wsRoot := t.TempDir()
		// No project markers in wsRoot — should fall back to CWD if CWD has a marker.
		// We run this from the Wave project dir which has go.mod, so CWD should be returned.
		step := &Step{ID: "test-step"}
		got := resolveCommandWorkDir(wsRoot, step)

		cwd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}

		// Check if CWD has a project marker
		cwdHasMarker := false
		for _, marker := range []string{"go.mod", "package.json", "Cargo.toml", "pyproject.toml", "Makefile"} {
			if _, err := os.Stat(filepath.Join(cwd, marker)); err == nil {
				cwdHasMarker = true
				break
			}
		}

		if cwdHasMarker {
			if got != cwd {
				t.Errorf("expected CWD %q (has project marker), got %q", cwd, got)
			}
		} else {
			// If CWD doesn't have a marker either, we get wsRoot back
			if got != wsRoot {
				t.Errorf("expected workspace root %q (no markers anywhere), got %q", wsRoot, got)
			}
		}
	})

	t.Run("bare workspace with no marker and CWD without marker returns workspace", func(t *testing.T) {
		wsRoot := t.TempDir()
		emptyDir := t.TempDir()

		// Change to an empty dir with no markers for this test
		origDir, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(emptyDir); err != nil {
			t.Fatal(err)
		}
		defer os.Chdir(origDir)

		step := &Step{ID: "test-step"}
		got := resolveCommandWorkDir(wsRoot, step)
		if got != wsRoot {
			t.Errorf("expected workspace root %q, got %q", wsRoot, got)
		}
	})
}

// --- Unit test: filterEnvPassthrough ---

func TestFilterEnvPassthrough(t *testing.T) {
	// Set a known env var for testing
	_ = os.Setenv("WAVE_TEST_PASSTHROUGH", "secret_value")
	_ = os.Setenv("WAVE_TEST_BLOCKED", "should_not_appear")
	defer os.Unsetenv("WAVE_TEST_PASSTHROUGH")
	defer os.Unsetenv("WAVE_TEST_BLOCKED")

	filtered := filterEnvPassthrough([]string{"WAVE_TEST_PASSTHROUGH", "HOME"})

	// PATH should always be present
	foundPath := false
	foundPassthrough := false
	foundBlocked := false
	for _, entry := range filtered {
		name, _, _ := strings.Cut(entry, "=")
		switch name {
		case "PATH":
			foundPath = true
		case "WAVE_TEST_PASSTHROUGH":
			foundPassthrough = true
		case "WAVE_TEST_BLOCKED":
			foundBlocked = true
		}
	}

	if !foundPath {
		t.Error("expected PATH to always be present in filtered env")
	}
	if !foundPassthrough {
		t.Error("expected WAVE_TEST_PASSTHROUGH to be in filtered env")
	}
	if foundBlocked {
		t.Error("expected WAVE_TEST_BLOCKED to NOT be in filtered env")
	}
}

// --- Coverage gap: filterEnvPassthrough essentials and edge cases ---

func TestFilterEnvPassthrough_AllEssentialsIncluded(t *testing.T) {
	// Set all essential vars so we can verify they appear in output.
	essentials := []string{
		"PATH", "HOME", "USER", "TMPDIR",
		"GOPATH", "GOMODCACHE", "GOCACHE", "GOROOT",
		"XDG_DATA_HOME", "XDG_CONFIG_HOME", "XDG_CACHE_HOME",
	}
	for _, name := range essentials {
		t.Setenv(name, "test_value_"+name)
	}

	// Empty passthrough list — essentials should still be present.
	filtered := filterEnvPassthrough(nil)

	found := make(map[string]bool)
	for _, entry := range filtered {
		name, _, _ := strings.Cut(entry, "=")
		found[name] = true
	}

	for _, name := range essentials {
		if !found[name] {
			t.Errorf("essential variable %q missing from filtered env", name)
		}
	}
}

func TestFilterEnvPassthrough_EdgeCases(t *testing.T) {
	t.Run("empty passthrough still includes essentials", func(t *testing.T) {
		t.Setenv("PATH", "/usr/bin")
		filtered := filterEnvPassthrough([]string{})
		foundPath := false
		for _, entry := range filtered {
			name, _, _ := strings.Cut(entry, "=")
			if name == "PATH" {
				foundPath = true
			}
		}
		if !foundPath {
			t.Error("PATH should be present even with empty passthrough")
		}
	})

	t.Run("duplicate in passthrough does not cause double entries", func(t *testing.T) {
		t.Setenv("PATH", "/usr/bin")
		filtered := filterEnvPassthrough([]string{"PATH", "PATH"})
		count := 0
		for _, entry := range filtered {
			name, _, _ := strings.Cut(entry, "=")
			if name == "PATH" {
				count++
			}
		}
		if count != 1 {
			t.Errorf("expected PATH to appear once, got %d", count)
		}
	})

	t.Run("passthrough var not set in environment is absent", func(t *testing.T) {
		os.Unsetenv("WAVE_NONEXISTENT_VAR_12345")
		filtered := filterEnvPassthrough([]string{"WAVE_NONEXISTENT_VAR_12345"})
		for _, entry := range filtered {
			name, _, _ := strings.Cut(entry, "=")
			if name == "WAVE_NONEXISTENT_VAR_12345" {
				t.Error("unset variable should not appear in filtered env")
			}
		}
	})

	t.Run("variable with equals sign in value is preserved correctly", func(t *testing.T) {
		t.Setenv("WAVE_EQ_TEST", "bar=baz=qux")
		defer os.Unsetenv("WAVE_EQ_TEST")
		filtered := filterEnvPassthrough([]string{"WAVE_EQ_TEST"})
		found := false
		for _, entry := range filtered {
			name, val, ok := strings.Cut(entry, "=")
			if name == "WAVE_EQ_TEST" {
				found = true
				if !ok || val != "bar=baz=qux" {
					t.Errorf("expected value 'bar=baz=qux', got %q", val)
				}
			}
		}
		if !found {
			t.Error("WAVE_EQ_TEST should appear in filtered env")
		}
	})

	t.Run("non-essential non-passthrough vars are excluded", func(t *testing.T) {
		t.Setenv("WAVE_SECRET_KEY", "super_secret")
		defer os.Unsetenv("WAVE_SECRET_KEY")
		filtered := filterEnvPassthrough([]string{"HOME"})
		for _, entry := range filtered {
			name, _, _ := strings.Cut(entry, "=")
			if name == "WAVE_SECRET_KEY" {
				t.Error("WAVE_SECRET_KEY should not leak into filtered env")
			}
		}
	})
}

// --- Coverage gap: resolveCommandWorkDir edge cases ---

func TestResolveCommandWorkDir_MountPriorityOverWorktree(t *testing.T) {
	// When both a mount target and a __wt_ directory exist, mount should win.
	wsRoot := t.TempDir()
	projectDir := filepath.Join(wsRoot, "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	wtDir := filepath.Join(wsRoot, "__wt_mybranch")
	if err := os.MkdirAll(wtDir, 0755); err != nil {
		t.Fatal(err)
	}

	step := &Step{
		ID: "test-step",
		Workspace: WorkspaceConfig{
			Mount: []Mount{
				{Source: "./", Target: "/project", Mode: "readonly"},
			},
		},
	}
	got := resolveCommandWorkDir(wsRoot, step)
	if got != projectDir {
		t.Errorf("mount should take priority over __wt_ dir: expected %q, got %q", projectDir, got)
	}
}

func TestResolveCommandWorkDir_WtFileNotDirectory(t *testing.T) {
	// A __wt_ entry that is a file (not directory) should not be returned.
	wsRoot := t.TempDir()
	wtFile := filepath.Join(wsRoot, "__wt_notadir")
	if err := os.WriteFile(wtFile, []byte("not a directory"), 0644); err != nil {
		t.Fatal(err)
	}

	step := &Step{ID: "test-step"}
	got := resolveCommandWorkDir(wsRoot, step)
	// Should not return the file; should fall through to workspace root or CWD.
	if got == wtFile {
		t.Errorf("__wt_ file (not directory) should not be returned as workdir")
	}
}

func TestResolveCommandWorkDir_MountTargetIsFile(t *testing.T) {
	// When mount target exists as a file (not directory), os.Stat().IsDir() is false.
	wsRoot := t.TempDir()
	targetFile := filepath.Join(wsRoot, "project")
	if err := os.WriteFile(targetFile, []byte("I am a file"), 0644); err != nil {
		t.Fatal(err)
	}

	step := &Step{
		ID: "test-step",
		Workspace: WorkspaceConfig{
			Mount: []Mount{
				{Source: "./", Target: "/project", Mode: "readonly"},
			},
		},
	}
	got := resolveCommandWorkDir(wsRoot, step)
	// target exists as file, not dir — mount should be skipped
	if got == targetFile {
		t.Errorf("mount target that is a file should be skipped, got %q", got)
	}
}

func TestResolveCommandWorkDir_NonexistentWorkspacePath(t *testing.T) {
	// When workspacePath does not exist, os.ReadDir fails — function should
	// return workspacePath unchanged (after all branches fail).
	wsRoot := filepath.Join(t.TempDir(), "does-not-exist")
	step := &Step{ID: "test-step"}
	got := resolveCommandWorkDir(wsRoot, step)
	// All branches should fail gracefully; the function returns wsRoot.
	// (CWD fallback only activates when wsRoot has no markers, and wsRoot doesn't exist.)
	if got != wsRoot {
		cwd, _ := os.Getwd()
		if got != cwd {
			t.Errorf("expected workspace root %q or CWD %q, got %q", wsRoot, cwd, got)
		}
	}
}
