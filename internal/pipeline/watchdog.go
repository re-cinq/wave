package pipeline

import (
	"context"
	"errors"
	"time"
)

// readOnlyTools lists tools that indicate activity but not progress.
// If only these tools fire, the agent may be stuck in a read loop.
var readOnlyTools = map[string]bool{
	"Read": true, "Glob": true, "Grep": true,
	"WebSearch": true, "WebFetch": true,
	"ToolSearch": true, "TaskList": true, "TaskGet": true,
}

// StallWatchdog detects when a pipeline step has stalled by monitoring
// activity signals. It tracks two timers:
//   - activity timer: resets on ANY tool use (detects complete silence)
//   - progress timer: resets only on write tools (detects read-only loops)
//
// The step is cancelled if either timer expires.
type StallWatchdog struct {
	timeout         time.Duration
	progressTimeout time.Duration
	activity        chan struct{}
	progress        chan struct{}
	cancel          context.CancelFunc
	done            chan struct{}
}

// NewStallWatchdog creates a new watchdog with the given stall timeout.
// The progress timeout defaults to 3x the activity timeout — an agent
// can read for up to 3x longer than it can be completely silent.
// Returns an error if timeout is zero or negative.
func NewStallWatchdog(timeout time.Duration) (*StallWatchdog, error) {
	if timeout <= 0 {
		return nil, errors.New("pipeline: stall watchdog timeout must be positive")
	}
	return &StallWatchdog{
		timeout:         timeout,
		progressTimeout: timeout * 3,
		activity:        make(chan struct{}, 1),
		progress:        make(chan struct{}, 1),
		done:            make(chan struct{}),
	}, nil
}

// Start returns a derived context that will be canceled if either the
// activity timer (complete silence) or progress timer (read-only loop)
// expires. The caller should call Stop when the watchdog is no longer needed.
func (w *StallWatchdog) Start(ctx context.Context) context.Context {
	ctx, w.cancel = context.WithCancel(ctx)

	go func() {
		defer close(w.done)

		activityTimer := time.NewTimer(w.timeout)
		progressTimer := time.NewTimer(w.progressTimeout)
		defer activityTimer.Stop()
		defer progressTimer.Stop()

		for {
			select {
			case <-w.activity:
				if !activityTimer.Stop() {
					select {
					case <-activityTimer.C:
					default:
					}
				}
				activityTimer.Reset(w.timeout)
			case <-w.progress:
				if !progressTimer.Stop() {
					select {
					case <-progressTimer.C:
					default:
					}
				}
				progressTimer.Reset(w.progressTimeout)
			case <-activityTimer.C:
				w.cancel()
				return
			case <-progressTimer.C:
				w.cancel()
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return ctx
}

// NotifyActivity signals that the step has tool activity (any tool).
// It is safe to call from any goroutine. The call never blocks.
func (w *StallWatchdog) NotifyActivity() {
	select {
	case w.activity <- struct{}{}:
	default:
	}
}

// NotifyProgress signals that the step made write-progress (file writes,
// edits, commits — not just reads). Resets the progress timer.
// It is safe to call from any goroutine. The call never blocks.
func (w *StallWatchdog) NotifyProgress() {
	select {
	case w.progress <- struct{}{}:
	default:
	}
}

// IsProgressTool returns true if the tool name indicates write-progress
// rather than read-only activity.
func IsProgressTool(toolName string) bool {
	return !readOnlyTools[toolName]
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
