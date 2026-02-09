package display

import (
	"sync"
	"testing"
	"time"

	"github.com/recinq/wave/internal/event"
)

// mockProgressEmitter records all events passed to EmitProgress for assertion.
type mockProgressEmitter struct {
	mu     sync.Mutex
	events []event.Event
}

func (m *mockProgressEmitter) EmitProgress(evt event.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, evt)
	return nil
}

func (m *mockProgressEmitter) getEvents() []event.Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]event.Event, len(m.events))
	copy(result, m.events)
	return result
}

func TestThrottledEmitter_FirstStreamActivityPassesThrough(t *testing.T) {
	mock := &mockProgressEmitter{}
	emitter := NewThrottledProgressEmitterWithInterval(mock, 100*time.Millisecond)

	evt := event.Event{
		State:    event.StateStreamActivity,
		ToolName: "Read",
	}
	if err := emitter.EmitProgress(evt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events := mock.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event in mock, got %d", len(events))
	}
	if events[0].State != event.StateStreamActivity {
		t.Errorf("expected state %q, got %q", event.StateStreamActivity, events[0].State)
	}
	if events[0].ToolName != "Read" {
		t.Errorf("expected tool_name %q, got %q", "Read", events[0].ToolName)
	}
}

func TestThrottledEmitter_EventsWithinWindowCoalesced(t *testing.T) {
	mock := &mockProgressEmitter{}
	emitter := NewThrottledProgressEmitterWithInterval(mock, 100*time.Millisecond)

	// Emit 5 stream_activity events rapidly — no sleep between them.
	for i := 0; i < 5; i++ {
		evt := event.Event{
			State:    event.StateStreamActivity,
			ToolName: "Bash",
			Message:  "rapid-burst",
		}
		if err := emitter.EmitProgress(evt); err != nil {
			t.Fatalf("unexpected error on emit %d: %v", i, err)
		}
	}

	// Only the first event should have passed through; the rest are coalesced.
	events := mock.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event after rapid burst, got %d", len(events))
	}

	// Wait for the throttle window to expire.
	time.Sleep(150 * time.Millisecond)

	// Emit one more stream_activity — should pass through because window expired.
	lastEvt := event.Event{
		State:    event.StateStreamActivity,
		ToolName: "Write",
		Message:  "after-window",
	}
	if err := emitter.EmitProgress(lastEvt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events = mock.getEvents()
	if len(events) != 2 {
		t.Fatalf("expected 2 events after window expiry, got %d", len(events))
	}

	// The second event in mock should be the most-recently emitted one.
	if events[1].ToolName != "Write" {
		t.Errorf("expected second event tool_name %q, got %q", "Write", events[1].ToolName)
	}
	if events[1].Message != "after-window" {
		t.Errorf("expected second event message %q, got %q", "after-window", events[1].Message)
	}
}

func TestThrottledEmitter_NonStreamActivityPassesImmediately(t *testing.T) {
	mock := &mockProgressEmitter{}
	emitter := NewThrottledProgressEmitterWithInterval(mock, 100*time.Millisecond)

	// Emit a stream_activity (passes through as first).
	if err := emitter.EmitProgress(event.Event{
		State:    event.StateStreamActivity,
		ToolName: "Read",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Emit a "running" event immediately — should pass through.
	if err := emitter.EmitProgress(event.Event{
		State:   event.StateRunning,
		Message: "step running",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Emit a "completed" event immediately — should pass through.
	if err := emitter.EmitProgress(event.Event{
		State:   event.StateCompleted,
		Message: "step completed",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events := mock.getEvents()
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[0].State != event.StateStreamActivity {
		t.Errorf("event[0]: expected state %q, got %q", event.StateStreamActivity, events[0].State)
	}
	if events[1].State != event.StateRunning {
		t.Errorf("event[1]: expected state %q, got %q", event.StateRunning, events[1].State)
	}
	if events[2].State != event.StateCompleted {
		t.Errorf("event[2]: expected state %q, got %q", event.StateCompleted, events[2].State)
	}
}

func TestThrottledEmitter_PendingFlushedOnNonStreamEvent(t *testing.T) {
	mock := &mockProgressEmitter{}
	emitter := NewThrottledProgressEmitterWithInterval(mock, 100*time.Millisecond)

	// First stream_activity passes through.
	if err := emitter.EmitProgress(event.Event{
		State:    event.StateStreamActivity,
		ToolName: "Read",
		Message:  "first",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Quickly emit 3 more stream_activity events (all within throttle window).
	// These get stored as pending (most-recent-wins).
	for i := 0; i < 3; i++ {
		if err := emitter.EmitProgress(event.Event{
			State:    event.StateStreamActivity,
			ToolName: "Bash",
			Message:  "pending",
		}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// At this point mock should have exactly 1 event (the first pass-through).
	events := mock.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event before flush, got %d", len(events))
	}

	// Now emit a "completed" event. This should:
	//   1. Flush the pending stream_activity (the last one stored)
	//   2. Forward the completed event itself
	if err := emitter.EmitProgress(event.Event{
		State:   event.StateCompleted,
		Message: "done",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events = mock.getEvents()
	if len(events) != 3 {
		t.Fatalf("expected 3 events after flush, got %d", len(events))
	}

	// Event 0: first stream_activity that passed through.
	if events[0].State != event.StateStreamActivity {
		t.Errorf("event[0]: expected state %q, got %q", event.StateStreamActivity, events[0].State)
	}
	if events[0].Message != "first" {
		t.Errorf("event[0]: expected message %q, got %q", "first", events[0].Message)
	}

	// Event 1: the flushed pending stream_activity (last one wins).
	if events[1].State != event.StateStreamActivity {
		t.Errorf("event[1]: expected state %q, got %q", event.StateStreamActivity, events[1].State)
	}
	if events[1].ToolName != "Bash" {
		t.Errorf("event[1]: expected tool_name %q, got %q", "Bash", events[1].ToolName)
	}

	// Event 2: the completed event.
	if events[2].State != event.StateCompleted {
		t.Errorf("event[2]: expected state %q, got %q", event.StateCompleted, events[2].State)
	}
	if events[2].Message != "done" {
		t.Errorf("event[2]: expected message %q, got %q", "done", events[2].Message)
	}
}

func TestThrottledEmitter_ConcurrentAccess(t *testing.T) {
	mock := &mockProgressEmitter{}
	emitter := NewThrottledProgressEmitterWithInterval(mock, 10*time.Millisecond)

	var wg sync.WaitGroup
	const goroutines = 10
	const eventsPerGoroutine = 100

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < eventsPerGoroutine; i++ {
				state := event.StateStreamActivity
				if i%2 == 0 {
					state = event.StateRunning
				}
				_ = emitter.EmitProgress(event.Event{
					State:   state,
					Message: "concurrent",
				})
			}
		}(g)
	}

	wg.Wait()

	events := mock.getEvents()
	if len(events) == 0 {
		t.Fatal("expected at least some events to pass through, got 0")
	}
	// With 10 goroutines x 100 events, many should pass through (at least the
	// non-stream-activity "running" events always pass immediately).
	t.Logf("received %d events from %d total emitted", len(events), goroutines*eventsPerGoroutine)
}

func TestThrottledEmitter_ConfigurableInterval(t *testing.T) {
	mock := &mockProgressEmitter{}
	emitter := NewThrottledProgressEmitterWithInterval(mock, 10*time.Millisecond)

	// First stream_activity passes through.
	if err := emitter.EmitProgress(event.Event{
		State:    event.StateStreamActivity,
		ToolName: "Read",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Sleep past the 10ms interval.
	time.Sleep(15 * time.Millisecond)

	// Second stream_activity should also pass through because the window expired.
	if err := emitter.EmitProgress(event.Event{
		State:    event.StateStreamActivity,
		ToolName: "Write",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events := mock.getEvents()
	if len(events) != 2 {
		t.Fatalf("expected 2 events with short interval, got %d", len(events))
	}
	if events[0].ToolName != "Read" {
		t.Errorf("event[0]: expected tool_name %q, got %q", "Read", events[0].ToolName)
	}
	if events[1].ToolName != "Write" {
		t.Errorf("event[1]: expected tool_name %q, got %q", "Write", events[1].ToolName)
	}
}

func TestThrottledEmitter_NilInnerEmitter(t *testing.T) {
	// Creating with nil inner should not panic on any operation.
	emitter := NewThrottledProgressEmitter(nil)

	// Emit stream_activity — should not panic.
	if err := emitter.EmitProgress(event.Event{
		State:    event.StateStreamActivity,
		ToolName: "Read",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Emit within throttle window to create a pending event.
	if err := emitter.EmitProgress(event.Event{
		State:    event.StateStreamActivity,
		ToolName: "Write",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Emit non-stream event to trigger the flush path with nil inner.
	if err := emitter.EmitProgress(event.Event{
		State:   event.StateCompleted,
		Message: "done",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// If we reach here without panicking, the test passes.
}
