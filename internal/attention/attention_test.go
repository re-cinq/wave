package attention

import (
	"testing"
	"time"

	"github.com/recinq/wave/internal/event"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		state    string
		want     State
		relevant bool
	}{
		{"started", Autonomous, true},
		{"running", Autonomous, true},
		{"completed", Autonomous, true},
		{"retrying", Autonomous, true},
		{"gate_waiting", NeedsReview, true},
		{"review_pending", NeedsReview, true},
		{"stalled", Blocked, true},
		{"cancelled", Blocked, true},
		{"failed", Failed, true},
		{"hook_failed", Failed, true},
		{"unknown_state", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			got, relevant := Classify(event.Event{State: tt.state})
			if relevant != tt.relevant {
				t.Fatalf("Classify(%q) relevant = %v, want %v", tt.state, relevant, tt.relevant)
			}
			if got != tt.want {
				t.Fatalf("Classify(%q) = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}

func TestBrokerUpdateAndSummary(t *testing.T) {
	b := NewBroker()

	// Start a run
	b.Update(event.Event{
		PipelineID: "run-1",
		State:      "started",
		Timestamp:  time.Now(),
	})

	s := b.Summary()
	if s.TotalRuns != 1 {
		t.Fatalf("expected 1 run, got %d", s.TotalRuns)
	}
	if s.WorstState != Autonomous {
		t.Fatalf("expected autonomous, got %s", s.WorstState)
	}

	// Gate waiting
	b.Update(event.Event{
		PipelineID: "run-1",
		StepID:     "review",
		State:      "gate_waiting",
		Timestamp:  time.Now(),
	})

	s = b.Summary()
	if s.WorstState != NeedsReview {
		t.Fatalf("expected needs_review, got %s", s.WorstState)
	}
	if s.NeedsReview != 1 {
		t.Fatalf("expected 1 needs_review, got %d", s.NeedsReview)
	}

	// Second run fails
	b.Update(event.Event{
		PipelineID: "run-2",
		State:      "failed",
		Timestamp:  time.Now(),
	})

	s = b.Summary()
	if s.TotalRuns != 2 {
		t.Fatalf("expected 2 runs, got %d", s.TotalRuns)
	}
	if s.WorstState != Failed {
		t.Fatalf("expected failed, got %s", s.WorstState)
	}

	// Complete run-1 (pipeline-level completed = no step ID)
	b.Update(event.Event{
		PipelineID: "run-1",
		State:      "completed",
		Timestamp:  time.Now(),
	})

	s = b.Summary()
	if s.TotalRuns != 1 {
		t.Fatalf("expected 1 run after completion, got %d", s.TotalRuns)
	}
}

func TestBrokerSubscribe(t *testing.T) {
	b := NewBroker()
	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	b.Update(event.Event{
		PipelineID: "run-1",
		State:      "started",
		Timestamp:  time.Now(),
	})

	select {
	case s := <-ch:
		if s.TotalRuns != 1 {
			t.Fatalf("expected 1 run in notification, got %d", s.TotalRuns)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for subscriber notification")
	}
}

func TestStateSeverity(t *testing.T) {
	if Autonomous.severity() >= NeedsReview.severity() {
		t.Error("autonomous should be less severe than needs_review")
	}
	if NeedsReview.severity() >= Blocked.severity() {
		t.Error("needs_review should be less severe than blocked")
	}
	if Blocked.severity() >= Failed.severity() {
		t.Error("blocked should be less severe than failed")
	}
}
