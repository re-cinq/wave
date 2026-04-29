package pipeline

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/testutil"
)

func TestGateExecutor_Approval_Auto(t *testing.T) {
	emitter := testutil.NewEventCollector()
	gate := NewGateExecutor(emitter, nil, nil)

	ctx := context.Background()
	err := gate.Execute(ctx, &GateConfig{Type: "approval", Auto: true}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !emitter.HasEventWithState(event.StateGateResolved) {
		t.Error("expected gate_resolved event")
	}
}

func TestGateExecutor_Timer(t *testing.T) {
	emitter := testutil.NewEventCollector()
	gate := NewGateExecutor(emitter, nil, nil)

	ctx := context.Background()
	start := time.Now()
	err := gate.Execute(ctx, &GateConfig{Type: "timer", Timeout: "100ms"}, nil)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed < 90*time.Millisecond {
		t.Errorf("timer resolved too quickly: %v", elapsed)
	}

	if !emitter.HasEventWithState(event.StateGateResolved) {
		t.Error("expected gate_resolved event")
	}
}

func TestGateExecutor_Timer_MissingTimeout(t *testing.T) {
	gate := NewGateExecutor(nil, nil, nil)

	ctx := context.Background()
	err := gate.Execute(ctx, &GateConfig{Type: "timer"}, nil)
	if err == nil {
		t.Fatal("expected error for timer without timeout")
	}
}

func TestGateExecutor_Approval_Timeout(t *testing.T) {
	gate := NewGateExecutor(nil, nil, nil)

	ctx := context.Background()
	err := gate.Execute(ctx, &GateConfig{Type: "approval", Timeout: "50ms"}, nil)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestGateExecutor_Approval_ContextCancel(t *testing.T) {
	gate := NewGateExecutor(nil, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := gate.Execute(ctx, &GateConfig{Type: "approval"}, nil)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestGateExecutor_PollGate_Auto(t *testing.T) {
	emitter := testutil.NewEventCollector()
	gate := NewGateExecutor(emitter, nil, nil)

	ctx := context.Background()
	err := gate.Execute(ctx, &GateConfig{Type: "pr_merge", Auto: true}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !emitter.HasEventWithState(event.StateGateResolved) {
		t.Error("expected gate_resolved event")
	}
}

func TestGateExecutor_UnknownType(t *testing.T) {
	gate := NewGateExecutor(nil, nil, nil)

	ctx := context.Background()
	err := gate.Execute(ctx, &GateConfig{Type: "unknown"}, nil)
	if err == nil {
		t.Fatal("expected error for unknown gate type")
	}
}

func TestGateExecutor_NilConfig(t *testing.T) {
	gate := NewGateExecutor(nil, nil, nil)

	ctx := context.Background()
	err := gate.Execute(ctx, nil, nil)
	if err == nil {
		t.Fatal("expected error for nil config")
	}
}

// newTestGateExecutor builds a GateExecutor with an injected commandRunner for testing.
func newTestGateExecutor(emitter event.EventEmitter, runner commandRunner) *GateExecutor {
	g := NewGateExecutor(emitter, nil, nil)
	g.runner = runner
	return g
}

// -- pr_merge gate tests --

func TestGateExecutor_PRMerge_Auto(t *testing.T) {
	emitter := testutil.NewEventCollector()
	g := newTestGateExecutor(emitter, nil) // runner never called for Auto

	err := g.Execute(context.Background(), &GateConfig{
		Type: "pr_merge", Auto: true, PRNumber: 42, Repo: "owner/repo",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !emitter.HasEventWithState(event.StateGateResolved) {
		t.Error("expected gate_resolved event")
	}
}

func TestGateExecutor_PRMerge_MissingPRNumber(t *testing.T) {
	g := newTestGateExecutor(nil, func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return nil, fmt.Errorf("should not be called")
	})

	err := g.Execute(context.Background(), &GateConfig{
		Type: "pr_merge", Repo: "owner/repo", Interval: "10ms", Timeout: "50ms",
	}, nil)
	if err == nil {
		t.Fatal("expected error for missing pr_number")
	}
	if !strings.Contains(err.Error(), "pr_number") {
		t.Errorf("expected error mentioning pr_number, got: %v", err)
	}
}

func TestGateExecutor_PRMerge_Merged(t *testing.T) {
	emitter := testutil.NewEventCollector()

	callCount := 0
	runner := func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		callCount++
		return []byte(`{"merged":true,"state":"closed"}`), nil
	}

	g := newTestGateExecutor(emitter, runner)
	err := g.Execute(context.Background(), &GateConfig{
		Type: "pr_merge", PRNumber: 42, Repo: "owner/repo",
		Interval: "10ms", Timeout: "2s",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !emitter.HasEventWithState(event.StateGateResolved) {
		t.Error("expected gate_resolved event")
	}
	if callCount == 0 {
		t.Error("expected at least one CLI call")
	}
}

func TestGateExecutor_PRMerge_ClosedWithoutMerge(t *testing.T) {
	runner := func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte(`{"merged":false,"state":"closed"}`), nil
	}

	g := newTestGateExecutor(nil, runner)
	err := g.Execute(context.Background(), &GateConfig{
		Type: "pr_merge", PRNumber: 7, Repo: "owner/repo",
		Interval: "10ms", Timeout: "2s",
	}, nil)
	if err == nil {
		t.Fatal("expected failure when PR closed without merge")
	}
	if !strings.Contains(err.Error(), "closed without merging") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGateExecutor_PRMerge_StillOpen_ThenMerged(t *testing.T) {
	emitter := testutil.NewEventCollector()
	callCount := 0

	runner := func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		callCount++
		if callCount < 3 {
			return []byte(`{"merged":false,"state":"open"}`), nil
		}
		return []byte(`{"merged":true,"state":"closed"}`), nil
	}

	g := newTestGateExecutor(emitter, runner)
	err := g.Execute(context.Background(), &GateConfig{
		Type: "pr_merge", PRNumber: 99, Repo: "owner/repo",
		Interval: "10ms", Timeout: "2s",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount < 3 {
		t.Errorf("expected at least 3 calls, got %d", callCount)
	}
	if !emitter.HasEventWithState(event.StateGateResolved) {
		t.Error("expected gate_resolved event")
	}
}

func TestGateExecutor_PRMerge_Timeout(t *testing.T) {
	runner := func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte(`{"merged":false,"state":"open"}`), nil
	}

	g := newTestGateExecutor(nil, runner)
	err := g.Execute(context.Background(), &GateConfig{
		Type: "pr_merge", PRNumber: 1, Repo: "owner/repo",
		Interval: "10ms", Timeout: "80ms",
	}, nil)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

func TestGateExecutor_PRMerge_ContextCancel(t *testing.T) {
	runner := func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte(`{"merged":false,"state":"open"}`), nil
	}

	g := newTestGateExecutor(nil, runner)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(40 * time.Millisecond)
		cancel()
	}()

	err := g.Execute(ctx, &GateConfig{
		Type: "pr_merge", PRNumber: 1, Repo: "owner/repo",
		Interval: "10ms", Timeout: "5s",
	}, nil)
	if err == nil {
		t.Fatal("expected context cancellation")
	}
}

func TestGateExecutor_PRMerge_CLIError_Retries(t *testing.T) {
	emitter := testutil.NewEventCollector()
	callCount := 0

	runner := func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		callCount++
		if callCount < 3 {
			return nil, fmt.Errorf("transient CLI error")
		}
		return []byte(`{"merged":true,"state":"closed"}`), nil
	}

	g := newTestGateExecutor(emitter, runner)
	err := g.Execute(context.Background(), &GateConfig{
		Type: "pr_merge", PRNumber: 5, Repo: "owner/repo",
		Interval: "10ms", Timeout: "2s",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !emitter.HasEventWithState(event.StateGateResolved) {
		t.Error("expected gate_resolved event")
	}
}

func TestGateExecutor_PRMerge_InvalidInterval(t *testing.T) {
	g := NewGateExecutor(nil, nil, nil)
	err := g.Execute(context.Background(), &GateConfig{
		Type: "pr_merge", PRNumber: 1, Repo: "owner/repo",
		Interval: "not-a-duration",
	}, nil)
	if err == nil {
		t.Fatal("expected error for invalid interval")
	}
}

func TestGateExecutor_PRMerge_InvalidTimeout(t *testing.T) {
	g := NewGateExecutor(nil, nil, nil)
	err := g.Execute(context.Background(), &GateConfig{
		Type: "pr_merge", PRNumber: 1, Repo: "owner/repo",
		Timeout: "not-a-duration",
	}, nil)
	if err == nil {
		t.Fatal("expected error for invalid timeout")
	}
}

// -- ci_pass gate tests --

func TestGateExecutor_CIPass_Auto(t *testing.T) {
	emitter := testutil.NewEventCollector()
	g := newTestGateExecutor(emitter, nil)

	err := g.Execute(context.Background(), &GateConfig{
		Type: "ci_pass", Auto: true, Repo: "owner/repo", Branch: "main",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !emitter.HasEventWithState(event.StateGateResolved) {
		t.Error("expected gate_resolved event")
	}
}

func TestGateExecutor_CIPass_Success(t *testing.T) {
	emitter := testutil.NewEventCollector()

	runner := func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte(`[{"status":"completed","conclusion":"success"}]`), nil
	}

	g := newTestGateExecutor(emitter, runner)
	err := g.Execute(context.Background(), &GateConfig{
		Type: "ci_pass", Branch: "feat/my-feature", Repo: "owner/repo",
		Interval: "10ms", Timeout: "2s",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !emitter.HasEventWithState(event.StateGateResolved) {
		t.Error("expected gate_resolved event")
	}
}

func TestGateExecutor_CIPass_Failure(t *testing.T) {
	runner := func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte(`[{"status":"completed","conclusion":"failure"}]`), nil
	}

	g := newTestGateExecutor(nil, runner)
	err := g.Execute(context.Background(), &GateConfig{
		Type: "ci_pass", Branch: "feat/my-feature", Repo: "owner/repo",
		Interval: "10ms", Timeout: "2s",
	}, nil)
	if err == nil {
		t.Fatal("expected failure when CI fails")
	}
	if !strings.Contains(err.Error(), "failure") {
		t.Errorf("expected failure message, got: %v", err)
	}
}

func TestGateExecutor_CIPass_Cancelled(t *testing.T) {
	runner := func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte(`[{"status":"completed","conclusion":"cancelled"}]`), nil
	}

	g := newTestGateExecutor(nil, runner)
	err := g.Execute(context.Background(), &GateConfig{
		Type: "ci_pass", Branch: "feat/my-feature", Repo: "owner/repo",
		Interval: "10ms", Timeout: "2s",
	}, nil)
	if err == nil {
		t.Fatal("expected failure for cancelled CI")
	}
}

func TestGateExecutor_CIPass_Skipped_TreatedAsPass(t *testing.T) {
	emitter := testutil.NewEventCollector()

	runner := func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte(`[{"status":"completed","conclusion":"skipped"}]`), nil
	}

	g := newTestGateExecutor(emitter, runner)
	err := g.Execute(context.Background(), &GateConfig{
		Type: "ci_pass", Branch: "main", Repo: "owner/repo",
		Interval: "10ms", Timeout: "2s",
	}, nil)
	if err != nil {
		t.Fatalf("skipped conclusion should be treated as pass, got error: %v", err)
	}
	if !emitter.HasEventWithState(event.StateGateResolved) {
		t.Error("expected gate_resolved event")
	}
}

func TestGateExecutor_CIPass_InProgress_ThenSuccess(t *testing.T) {
	emitter := testutil.NewEventCollector()
	callCount := 0

	runner := func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		callCount++
		if callCount < 3 {
			return []byte(`[{"status":"in_progress","conclusion":""}]`), nil
		}
		return []byte(`[{"status":"completed","conclusion":"success"}]`), nil
	}

	g := newTestGateExecutor(emitter, runner)
	err := g.Execute(context.Background(), &GateConfig{
		Type: "ci_pass", Branch: "main", Repo: "owner/repo",
		Interval: "10ms", Timeout: "2s",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount < 3 {
		t.Errorf("expected at least 3 calls, got %d", callCount)
	}
	if !emitter.HasEventWithState(event.StateGateResolved) {
		t.Error("expected gate_resolved event")
	}
}

func TestGateExecutor_CIPass_NoRuns_ThenSuccess(t *testing.T) {
	emitter := testutil.NewEventCollector()
	callCount := 0

	runner := func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		callCount++
		if callCount < 2 {
			return []byte(`[]`), nil
		}
		return []byte(`[{"status":"completed","conclusion":"success"}]`), nil
	}

	g := newTestGateExecutor(emitter, runner)
	err := g.Execute(context.Background(), &GateConfig{
		Type: "ci_pass", Branch: "main", Repo: "owner/repo",
		Interval: "10ms", Timeout: "2s",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !emitter.HasEventWithState(event.StateGateResolved) {
		t.Error("expected gate_resolved event")
	}
}

func TestGateExecutor_CIPass_Timeout(t *testing.T) {
	runner := func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte(`[{"status":"in_progress","conclusion":""}]`), nil
	}

	g := newTestGateExecutor(nil, runner)
	err := g.Execute(context.Background(), &GateConfig{
		Type: "ci_pass", Branch: "main", Repo: "owner/repo",
		Interval: "10ms", Timeout: "80ms",
	}, nil)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

func TestGateExecutor_CIPass_ContextCancel(t *testing.T) {
	runner := func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte(`[{"status":"in_progress","conclusion":""}]`), nil
	}

	g := newTestGateExecutor(nil, runner)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(40 * time.Millisecond)
		cancel()
	}()

	err := g.Execute(ctx, &GateConfig{
		Type: "ci_pass", Branch: "main", Repo: "owner/repo",
		Interval: "10ms", Timeout: "5s",
	}, nil)
	if err == nil {
		t.Fatal("expected context cancellation")
	}
}

func TestGateExecutor_CIPass_CLIError_Retries(t *testing.T) {
	emitter := testutil.NewEventCollector()
	callCount := 0

	runner := func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		callCount++
		if callCount < 3 {
			return nil, fmt.Errorf("transient error")
		}
		return []byte(`[{"status":"completed","conclusion":"success"}]`), nil
	}

	g := newTestGateExecutor(emitter, runner)
	err := g.Execute(context.Background(), &GateConfig{
		Type: "ci_pass", Branch: "main", Repo: "owner/repo",
		Interval: "10ms", Timeout: "2s",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !emitter.HasEventWithState(event.StateGateResolved) {
		t.Error("expected gate_resolved event")
	}
}

func TestGateExecutor_CIPass_InvalidJSON_Retries(t *testing.T) {
	emitter := testutil.NewEventCollector()
	callCount := 0

	runner := func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		callCount++
		if callCount < 2 {
			return []byte(`not json`), nil
		}
		return []byte(`[{"status":"completed","conclusion":"success"}]`), nil
	}

	g := newTestGateExecutor(emitter, runner)
	err := g.Execute(context.Background(), &GateConfig{
		Type: "ci_pass", Branch: "main", Repo: "owner/repo",
		Interval: "10ms", Timeout: "2s",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// -- parsePollGateTiming tests --

func TestParsePollGateTiming_Defaults(t *testing.T) {
	interval, timeout, err := parsePollGateTiming(&GateConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if interval != 30*time.Second {
		t.Errorf("expected 30s interval, got %v", interval)
	}
	if timeout != 60*time.Minute {
		t.Errorf("expected 60m timeout, got %v", timeout)
	}
}

func TestParsePollGateTiming_Custom(t *testing.T) {
	interval, timeout, err := parsePollGateTiming(&GateConfig{
		Interval: "1m",
		Timeout:  "2h",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if interval != time.Minute {
		t.Errorf("expected 1m interval, got %v", interval)
	}
	if timeout != 2*time.Hour {
		t.Errorf("expected 2h timeout, got %v", timeout)
	}
}

func TestParsePollGateTiming_InvalidInterval(t *testing.T) {
	_, _, err := parsePollGateTiming(&GateConfig{Interval: "bad"})
	if err == nil {
		t.Fatal("expected error for invalid interval")
	}
}

func TestParsePollGateTiming_InvalidTimeout(t *testing.T) {
	_, _, err := parsePollGateTiming(&GateConfig{Timeout: "bad"})
	if err == nil {
		t.Fatal("expected error for invalid timeout")
	}
}

// -- Choice-based approval gate tests --

func TestGateExecutor_Approval_WithChoices_AutoApproveHandler(t *testing.T) {
	emitter := testutil.NewEventCollector()
	handler := &AutoApproveHandler{}
	gate := NewGateExecutorWithHandler(emitter, nil, nil, handler)

	config := &GateConfig{
		Type: "approval",
		Choices: []GateChoice{
			{Label: "Approve", Key: "a", Target: "implement"},
			{Label: "Revise", Key: "r", Target: "plan"},
			{Label: "Abort", Key: "q", Target: "_fail"},
		},
		Default: "a",
	}

	decision, err := gate.ExecuteWithDecision(context.Background(), config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision == nil {
		t.Fatal("expected a decision")
	}
	if decision.Choice != "a" {
		t.Errorf("expected choice 'a', got %q", decision.Choice)
	}
	if decision.Label != "Approve" {
		t.Errorf("expected label 'Approve', got %q", decision.Label)
	}
	if decision.Target != "implement" {
		t.Errorf("expected target 'implement', got %q", decision.Target)
	}
	if !emitter.HasEventWithState(event.StateGateResolved) {
		t.Error("expected gate_resolved event")
	}
}

func TestGateExecutor_Approval_WithChoices_NoHandler_TimeoutDefault(t *testing.T) {
	emitter := testutil.NewEventCollector()
	// No handler, but has choices with default — should use default on timeout
	gate := NewGateExecutor(emitter, nil, nil)

	config := &GateConfig{
		Type:    "approval",
		Timeout: "50ms",
		Choices: []GateChoice{
			{Label: "Approve", Key: "a", Target: "implement"},
		},
		Default: "a",
	}

	decision, err := gate.ExecuteWithDecision(context.Background(), config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision == nil {
		t.Fatal("expected a decision on timeout with default")
	}
	if decision.Choice != "a" {
		t.Errorf("expected choice 'a', got %q", decision.Choice)
	}
	if !emitter.HasEventWithState(event.StateGateResolved) {
		t.Error("expected gate_resolved event")
	}
}

func TestGateExecutor_Approval_LegacyAutoApprove_StillWorks(t *testing.T) {
	// Legacy gate: Auto=true, no choices — should auto-approve as before
	emitter := testutil.NewEventCollector()
	gate := NewGateExecutor(emitter, nil, nil)

	decision, err := gate.ExecuteWithDecision(context.Background(), &GateConfig{
		Type: "approval",
		Auto: true,
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision != nil {
		t.Error("legacy auto-approve should return nil decision")
	}
	if !emitter.HasEventWithState(event.StateGateResolved) {
		t.Error("expected gate_resolved event")
	}
}

// testGateHandler is a mock handler for testing.
type testGateHandler struct {
	decision *GateDecision
	err      error
}

func (h *testGateHandler) Prompt(_ context.Context, _ *GateConfig) (*GateDecision, error) {
	return h.decision, h.err
}

func TestGateExecutor_Approval_WithChoices_CustomHandler(t *testing.T) {
	emitter := testutil.NewEventCollector()
	handler := &testGateHandler{
		decision: &GateDecision{
			Choice: "r",
			Label:  "Revise",
			Target: "plan",
			Text:   "Please add more tests",
		},
	}
	gate := NewGateExecutorWithHandler(emitter, nil, nil, handler)

	config := &GateConfig{
		Type: "approval",
		Choices: []GateChoice{
			{Label: "Approve", Key: "a", Target: "implement"},
			{Label: "Revise", Key: "r", Target: "plan"},
		},
		Freeform: true,
	}

	decision, err := gate.ExecuteWithDecision(context.Background(), config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Choice != "r" {
		t.Errorf("expected choice 'r', got %q", decision.Choice)
	}
	if decision.Text != "Please add more tests" {
		t.Errorf("expected freeform text, got %q", decision.Text)
	}
	if decision.Target != "plan" {
		t.Errorf("expected target 'plan', got %q", decision.Target)
	}
}

func TestGateExecutor_Approval_HandlerError(t *testing.T) {
	handler := &testGateHandler{
		err: fmt.Errorf("user cancelled"),
	}
	gate := NewGateExecutorWithHandler(nil, nil, nil, handler)

	config := &GateConfig{
		Type: "approval",
		Choices: []GateChoice{
			{Label: "Approve", Key: "a"},
		},
	}

	_, err := gate.ExecuteWithDecision(context.Background(), config, nil)
	if err == nil {
		t.Fatal("expected error from handler")
	}
	if !strings.Contains(err.Error(), "user cancelled") {
		t.Errorf("expected 'user cancelled' in error, got: %v", err)
	}
}
