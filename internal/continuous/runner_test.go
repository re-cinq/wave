package continuous

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/recinq/wave/internal/event"
)

// mockSource is a test WorkItemSource.
type mockSource struct {
	items []*WorkItem
	index int
}

func (s *mockSource) Next(_ context.Context) (*WorkItem, error) {
	if s.index >= len(s.items) {
		return nil, nil
	}
	item := s.items[s.index]
	s.index++
	return item, nil
}

func (s *mockSource) Name() string { return "mock" }

// mockEmitter collects emitted events for assertion.
type mockEmitter struct {
	mu     sync.Mutex
	events []event.Event
}

func (e *mockEmitter) Emit(ev event.Event) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, ev)
}

func (e *mockEmitter) getEvents() []event.Event {
	e.mu.Lock()
	defer e.mu.Unlock()
	copied := make([]event.Event, len(e.events))
	copy(copied, e.events)
	return copied
}

func TestRunnerNormalCompletion(t *testing.T) {
	source := &mockSource{
		items: []*WorkItem{
			{ID: "1", Input: "https://github.com/org/repo/issues/1"},
			{ID: "2", Input: "https://github.com/org/repo/issues/2"},
			{ID: "3", Input: "https://github.com/org/repo/issues/3"},
		},
	}

	emitter := &mockEmitter{}
	executionCount := 0

	runner := &Runner{
		Source:       source,
		PipelineName: "test-pipeline",
		OnFailure:    FailurePolicyHalt,
		Emitter:      emitter,
		ExecutorFactory: func(input string) ExecutorFunc {
			return func(ctx context.Context, input string) (string, error) {
				executionCount++
				return fmt.Sprintf("run-%d", executionCount), nil
			}
		},
	}

	summary, err := runner.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if summary.Total != 3 {
		t.Errorf("Total = %d, want 3", summary.Total)
	}
	if summary.Succeeded != 3 {
		t.Errorf("Succeeded = %d, want 3", summary.Succeeded)
	}
	if summary.Failed != 0 {
		t.Errorf("Failed = %d, want 0", summary.Failed)
	}
	if executionCount != 3 {
		t.Errorf("executionCount = %d, want 3", executionCount)
	}
	if summary.HasFailures() {
		t.Error("HasFailures() = true, want false")
	}
}

func TestRunnerEmptySource(t *testing.T) {
	source := &mockSource{items: nil}
	emitter := &mockEmitter{}

	runner := &Runner{
		Source:       source,
		PipelineName: "test-pipeline",
		Emitter:      emitter,
		ExecutorFactory: func(input string) ExecutorFunc {
			return func(ctx context.Context, input string) (string, error) {
				t.Fatal("executor should not be called")
				return "", nil
			}
		},
	}

	summary, err := runner.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if summary.Total != 0 {
		t.Errorf("Total = %d, want 0", summary.Total)
	}
}

func TestRunnerMaxIterations(t *testing.T) {
	source := &mockSource{
		items: []*WorkItem{
			{ID: "1", Input: "issue/1"},
			{ID: "2", Input: "issue/2"},
			{ID: "3", Input: "issue/3"},
		},
	}

	runner := &Runner{
		Source:        source,
		PipelineName:  "test-pipeline",
		MaxIterations: 2,
		Emitter:       &mockEmitter{},
		ExecutorFactory: func(input string) ExecutorFunc {
			return func(ctx context.Context, input string) (string, error) {
				return "run-id", nil
			}
		},
	}

	summary, err := runner.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if summary.Total != 2 {
		t.Errorf("Total = %d, want 2", summary.Total)
	}
}

func TestRunnerContextCancellation(t *testing.T) {
	source := &mockSource{
		items: []*WorkItem{
			{ID: "1", Input: "issue/1"},
			{ID: "2", Input: "issue/2"},
			{ID: "3", Input: "issue/3"},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0

	runner := &Runner{
		Source:       source,
		PipelineName: "test-pipeline",
		Emitter:      &mockEmitter{},
		ExecutorFactory: func(input string) ExecutorFunc {
			return func(ctx context.Context, input string) (string, error) {
				callCount++
				if callCount == 1 {
					cancel() // Cancel after first iteration
				}
				return "run-id", nil
			}
		},
	}

	summary, err := runner.Run(ctx)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Should have completed the first iteration, then stopped
	if summary.Total != 1 {
		t.Errorf("Total = %d, want 1", summary.Total)
	}
	if summary.Succeeded != 1 {
		t.Errorf("Succeeded = %d, want 1", summary.Succeeded)
	}
}

func TestRunnerDedup(t *testing.T) {
	source := &mockSource{
		items: []*WorkItem{
			{ID: "1", Input: "issue/1"},
			{ID: "1", Input: "issue/1"}, // duplicate
			{ID: "2", Input: "issue/2"},
		},
	}

	executionCount := 0
	runner := &Runner{
		Source:       source,
		PipelineName: "test-pipeline",
		Emitter:      &mockEmitter{},
		ExecutorFactory: func(input string) ExecutorFunc {
			return func(ctx context.Context, input string) (string, error) {
				executionCount++
				return "run-id", nil
			}
		},
	}

	summary, err := runner.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if executionCount != 2 {
		t.Errorf("executionCount = %d, want 2 (dedup should skip one)", executionCount)
	}
	if summary.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", summary.Skipped)
	}
	if summary.Succeeded != 2 {
		t.Errorf("Succeeded = %d, want 2", summary.Succeeded)
	}
	if summary.Total != 3 {
		t.Errorf("Total = %d, want 3", summary.Total)
	}
}

func TestRunnerFailurePolicyHalt(t *testing.T) {
	source := &mockSource{
		items: []*WorkItem{
			{ID: "1", Input: "issue/1"},
			{ID: "2", Input: "issue/2"},
			{ID: "3", Input: "issue/3"},
		},
	}

	callCount := 0
	runner := &Runner{
		Source:       source,
		PipelineName: "test-pipeline",
		OnFailure:    FailurePolicyHalt,
		Emitter:      &mockEmitter{},
		ExecutorFactory: func(input string) ExecutorFunc {
			return func(ctx context.Context, input string) (string, error) {
				callCount++
				if callCount == 2 {
					return "run-2", fmt.Errorf("step failed")
				}
				return fmt.Sprintf("run-%d", callCount), nil
			}
		},
	}

	summary, err := runner.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if summary.Total != 2 {
		t.Errorf("Total = %d, want 2 (halt after failure)", summary.Total)
	}
	if summary.Succeeded != 1 {
		t.Errorf("Succeeded = %d, want 1", summary.Succeeded)
	}
	if summary.Failed != 1 {
		t.Errorf("Failed = %d, want 1", summary.Failed)
	}
	if callCount != 2 {
		t.Errorf("callCount = %d, want 2", callCount)
	}
	if !summary.HasFailures() {
		t.Error("HasFailures() = false, want true")
	}
}

func TestRunnerFailurePolicySkip(t *testing.T) {
	source := &mockSource{
		items: []*WorkItem{
			{ID: "1", Input: "issue/1"},
			{ID: "2", Input: "issue/2"},
			{ID: "3", Input: "issue/3"},
		},
	}

	callCount := 0
	runner := &Runner{
		Source:       source,
		PipelineName: "test-pipeline",
		OnFailure:    FailurePolicySkip,
		Emitter:      &mockEmitter{},
		ExecutorFactory: func(input string) ExecutorFunc {
			return func(ctx context.Context, input string) (string, error) {
				callCount++
				if callCount == 2 {
					return "run-2", fmt.Errorf("step failed")
				}
				return fmt.Sprintf("run-%d", callCount), nil
			}
		},
	}

	summary, err := runner.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if summary.Total != 3 {
		t.Errorf("Total = %d, want 3 (skip continues)", summary.Total)
	}
	if summary.Succeeded != 2 {
		t.Errorf("Succeeded = %d, want 2", summary.Succeeded)
	}
	if summary.Failed != 1 {
		t.Errorf("Failed = %d, want 1", summary.Failed)
	}
	if callCount != 3 {
		t.Errorf("callCount = %d, want 3", callCount)
	}
}

func TestRunnerEventEmission(t *testing.T) {
	source := &mockSource{
		items: []*WorkItem{
			{ID: "1", Input: "issue/1"},
		},
	}

	emitter := &mockEmitter{}
	runner := &Runner{
		Source:       source,
		PipelineName: "test-pipeline",
		Emitter:      emitter,
		ExecutorFactory: func(input string) ExecutorFunc {
			return func(ctx context.Context, input string) (string, error) {
				return "run-1", nil
			}
		},
	}

	_, err := runner.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	events := emitter.getEvents()
	// Expect: loop_start, loop_iteration_start, loop_iteration_complete, loop_summary
	states := make([]string, len(events))
	for i, ev := range events {
		states[i] = ev.State
	}

	expected := []string{
		event.StateLoopStart,
		event.StateLoopIterationStart,
		event.StateLoopIterationComplete,
		event.StateLoopSummary,
	}
	if len(states) != len(expected) {
		t.Fatalf("got %d events %v, want %d events %v", len(states), states, len(expected), expected)
	}
	for i, want := range expected {
		if states[i] != want {
			t.Errorf("event[%d].State = %q, want %q", i, states[i], want)
		}
	}

	// Verify iteration metadata
	iterStartEvent := events[1]
	if iterStartEvent.Iteration != 1 {
		t.Errorf("iteration_start.Iteration = %d, want 1", iterStartEvent.Iteration)
	}
	if iterStartEvent.WorkItemID != "1" {
		t.Errorf("iteration_start.WorkItemID = %q, want %q", iterStartEvent.WorkItemID, "1")
	}
}

func TestSummaryString(t *testing.T) {
	s := &Summary{
		Total:     5,
		Succeeded: 3,
		Failed:    1,
		Skipped:   1,
		Duration:  2*time.Second + 500*time.Millisecond,
	}
	got := s.String()
	if got == "" {
		t.Error("String() returned empty")
	}
	// Should contain key counts
	for _, want := range []string{"5 iterations", "3 succeeded", "1 failed", "1 skipped"} {
		if !strings.Contains(got, want) {
			t.Errorf("String() = %q, missing %q", got, want)
		}
	}
}
