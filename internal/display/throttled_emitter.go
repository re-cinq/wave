package display

import (
	"sync"
	"time"

	"github.com/recinq/wave/internal/event"
)

// ThrottledProgressEmitter wraps a ProgressEmitter and throttles stream_activity
// events to at most 1 per throttleInterval (default 1 second). All other event
// types pass through immediately. This prevents display flooding during rapid
// tool call bursts while maintaining real-time feedback for lifecycle events.
type ThrottledProgressEmitter struct {
	inner                  event.ProgressEmitter
	mu                     sync.Mutex
	lastStreamActivityTime time.Time
	pendingStreamActivity  *event.Event
	throttleInterval       time.Duration
}

// NewThrottledProgressEmitter creates a ThrottledProgressEmitter with the default
// 1-second throttle interval.
func NewThrottledProgressEmitter(inner event.ProgressEmitter) *ThrottledProgressEmitter {
	return &ThrottledProgressEmitter{
		inner:            inner,
		throttleInterval: 1 * time.Second,
	}
}

// NewThrottledProgressEmitterWithInterval creates a ThrottledProgressEmitter with
// a custom throttle interval. Useful for testing with short intervals.
func NewThrottledProgressEmitterWithInterval(inner event.ProgressEmitter, interval time.Duration) *ThrottledProgressEmitter {
	return &ThrottledProgressEmitter{
		inner:            inner,
		throttleInterval: interval,
	}
}

// EmitProgress implements the ProgressEmitter interface.
func (t *ThrottledProgressEmitter) EmitProgress(evt event.Event) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if evt.State != event.StateStreamActivity {
		// Non-stream-activity events: flush any pending stream event first,
		// then forward immediately.
		if t.pendingStreamActivity != nil {
			pending := t.pendingStreamActivity
			t.pendingStreamActivity = nil
			if t.inner != nil {
				t.inner.EmitProgress(*pending)
			}
		}
		if t.inner != nil {
			return t.inner.EmitProgress(evt)
		}
		return nil
	}

	// stream_activity event handling
	now := time.Now()

	if t.lastStreamActivityTime.IsZero() || now.Sub(t.lastStreamActivityTime) >= t.throttleInterval {
		// First event ever, or throttle window has passed: forward immediately
		t.lastStreamActivityTime = now
		t.pendingStreamActivity = nil
		if t.inner != nil {
			return t.inner.EmitProgress(evt)
		}
		return nil
	}

	// Within throttle window: store as pending (most-recent-wins coalescing)
	evtCopy := evt
	t.pendingStreamActivity = &evtCopy
	return nil
}
