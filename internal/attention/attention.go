// Package attention provides an attention classifier that monitors pipeline
// events and determines whether operator attention is needed. It powers the
// "Clippy" notification widget in the Wave dashboard.
package attention

import (
	"sync"
	"time"

	"github.com/recinq/wave/internal/event"
)

// State represents the attention level required for a run.
type State string

const (
	// Autonomous means the run is progressing without issues.
	Autonomous State = "autonomous"
	// NeedsReview means a gate or review step is waiting for operator input.
	NeedsReview State = "needs_review"
	// Blocked means the run is stalled or cancelled.
	Blocked State = "blocked"
	// Failed means the run has failed.
	Failed State = "failed"
)

// severity returns a numeric severity for comparison (higher = more urgent).
func (s State) severity() int {
	switch s {
	case Failed:
		return 3
	case Blocked:
		return 2
	case NeedsReview:
		return 1
	default:
		return 0
	}
}

// RunAttention holds the attention state for a single run.
type RunAttention struct {
	RunID        string    `json:"run_id"`
	PipelineName string    `json:"pipeline_name"`
	State        State     `json:"state"`
	StepID       string    `json:"step_id,omitempty"`
	Message      string    `json:"message,omitempty"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Summary is the aggregate attention state across all active runs.
type Summary struct {
	WorstState    State          `json:"worst_state"`
	TotalRuns     int            `json:"total_runs"`
	Autonomous    int            `json:"autonomous"`
	NeedsReview   int            `json:"needs_review"`
	Blocked       int            `json:"blocked"`
	Failed        int            `json:"failed"`
	Runs          []RunAttention `json:"runs,omitempty"`
}

// Classify maps a pipeline event to an attention state.
// Returns the state and true if the event is relevant, or ("", false) if the
// event should be ignored (e.g., progress ticks).
func Classify(ev event.Event) (State, bool) {
	switch ev.State {
	case "started", "running", "retrying", "reworking",
		"step_progress", "eta_updated", "stream_activity",
		"contract_validating", "compaction_progress",
		"sequence_started", "iteration_started", "branch_evaluated",
		"hook_started", "hook_passed":
		return Autonomous, true

	case "completed":
		return Autonomous, true

	case "gate_waiting", "review_pending":
		return NeedsReview, true

	case "stalled", "cancelled":
		return Blocked, true

	case "failed", "hook_failed":
		return Failed, true

	default:
		return "", false
	}
}

// Broker aggregates attention state across all active runs and notifies
// subscribers when the summary changes.
type Broker struct {
	mu          sync.RWMutex
	runs        map[string]*RunAttention
	subscribers map[chan Summary]struct{}
}

// NewBroker creates a new attention broker.
func NewBroker() *Broker {
	return &Broker{
		runs:        make(map[string]*RunAttention),
		subscribers: make(map[chan Summary]struct{}),
	}
}

// Update processes a pipeline event and updates the attention state.
// If the state changes, all subscribers are notified.
func (b *Broker) Update(ev event.Event) {
	state, relevant := Classify(ev)
	if !relevant {
		return
	}

	b.mu.Lock()

	runID := ev.PipelineID
	ra, exists := b.runs[runID]
	if !exists {
		ra = &RunAttention{RunID: runID}
		b.runs[runID] = ra
	}

	// Completed runs are removed from tracking.
	if ev.State == "completed" && ev.StepID == "" {
		delete(b.runs, runID)
		b.mu.Unlock()
		b.notify()
		return
	}

	ra.State = state
	ra.StepID = ev.StepID
	ra.Message = ev.Message
	ra.UpdatedAt = ev.Timestamp
	if ra.PipelineName == "" {
		ra.PipelineName = runID
	}

	b.mu.Unlock()
	b.notify()
}

// Summary returns the current aggregate attention state.
func (b *Broker) Summary() Summary {
	b.mu.RLock()
	defer b.mu.RUnlock()

	s := Summary{WorstState: Autonomous}
	for _, ra := range b.runs {
		s.TotalRuns++
		switch ra.State {
		case Autonomous:
			s.Autonomous++
		case NeedsReview:
			s.NeedsReview++
		case Blocked:
			s.Blocked++
		case Failed:
			s.Failed++
		}
		if ra.State.severity() > s.WorstState.severity() {
			s.WorstState = ra.State
		}
		s.Runs = append(s.Runs, *ra)
	}
	return s
}

// Subscribe returns a channel that receives attention summaries on every change.
func (b *Broker) Subscribe() chan Summary {
	ch := make(chan Summary, 16)
	b.mu.Lock()
	b.subscribers[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber channel.
func (b *Broker) Unsubscribe(ch chan Summary) {
	b.mu.Lock()
	delete(b.subscribers, ch)
	b.mu.Unlock()
	close(ch)
}

func (b *Broker) notify() {
	s := b.Summary()
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subscribers {
		select {
		case ch <- s:
		default:
			// Drop if subscriber is slow — they'll get the next one.
		}
	}
}
