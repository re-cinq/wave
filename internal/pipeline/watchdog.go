package pipeline

import (
	"context"
	"time"
)

// StallWatchdog detects when a pipeline step has stalled by monitoring
// activity signals. If no activity is received within the configured
// timeout, it cancels the derived context.
type StallWatchdog struct {
	timeout  time.Duration
	activity chan struct{}
	cancel   context.CancelFunc
	done     chan struct{}
}

// NewStallWatchdog creates a new watchdog with the given stall timeout.
// It panics if timeout is zero or negative.
func NewStallWatchdog(timeout time.Duration) *StallWatchdog {
	if timeout <= 0 {
		panic("pipeline: stall watchdog timeout must be positive")
	}
	return &StallWatchdog{
		timeout:  timeout,
		activity: make(chan struct{}, 1),
		done:     make(chan struct{}),
	}
}

// Start returns a derived context that will be canceled if no activity
// is received for the configured timeout duration. The caller should
// call Stop when the watchdog is no longer needed.
func (w *StallWatchdog) Start(ctx context.Context) context.Context {
	ctx, w.cancel = context.WithCancel(ctx)

	go func() {
		defer close(w.done)

		timer := time.NewTimer(w.timeout)
		defer timer.Stop()

		for {
			select {
			case <-w.activity:
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(w.timeout)
			case <-timer.C:
				w.cancel()
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return ctx
}

// NotifyActivity signals that the step is still making progress.
// It is safe to call from any goroutine. The call never blocks.
func (w *StallWatchdog) NotifyActivity() {
	select {
	case w.activity <- struct{}{}:
	default:
	}
}

// Stop cancels the watchdog context and waits for the background
// goroutine to exit. It uses a short timeout to avoid hanging
// indefinitely if the goroutine is stuck.
func (w *StallWatchdog) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
	select {
	case <-w.done:
	case <-time.After(1 * time.Second):
	}
}
