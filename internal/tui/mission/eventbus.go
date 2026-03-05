package mission

import (
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/recinq/wave/internal/event"
)

// RunEvent carries an event from a specific pipeline run.
type RunEvent struct {
	RunID string
	Event event.Event
}

// RunEventMsg is a BubbleTea message carrying a run event.
type RunEventMsg struct {
	RunEvent
}

// EventBus is a fan-in channel collecting events from all concurrent pipeline runs.
type EventBus struct {
	ch     chan RunEvent
	closed bool
	mu     sync.Mutex
}

// NewEventBus creates a new EventBus with a buffered channel.
func NewEventBus() *EventBus {
	return &EventBus{
		ch: make(chan RunEvent, 256),
	}
}

// Send sends an event to the bus. Safe to call from any goroutine.
// Returns false if the bus is closed or the channel is full.
func (b *EventBus) Send(evt RunEvent) bool {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return false
	}
	b.mu.Unlock()

	select {
	case b.ch <- evt:
		return true
	default:
		// Channel full — drop event rather than blocking the executor
		return false
	}
}

// Close closes the event bus channel.
func (b *EventBus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.closed {
		b.closed = true
		close(b.ch)
	}
}

// WaitForRunEvent returns a tea.Cmd that blocks until the next event arrives
// on the bus. When the bus is closed it returns nil (no more events).
func WaitForRunEvent(bus *EventBus) tea.Cmd {
	return func() tea.Msg {
		evt, ok := <-bus.ch
		if !ok {
			return nil
		}
		return RunEventMsg{evt}
	}
}

// BusEmitter implements event.ProgressEmitter, forwarding events to the EventBus
// tagged with a specific run ID.
type BusEmitter struct {
	bus   *EventBus
	runID string
}

// NewBusEmitter creates a BusEmitter for a specific run.
func NewBusEmitter(bus *EventBus, runID string) *BusEmitter {
	return &BusEmitter{bus: bus, runID: runID}
}

// EmitProgress implements event.ProgressEmitter.
func (e *BusEmitter) EmitProgress(evt event.Event) error {
	e.bus.Send(RunEvent{RunID: e.runID, Event: evt})
	return nil
}
