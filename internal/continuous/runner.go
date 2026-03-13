package continuous

import (
	"context"
	"fmt"
	"time"

	"github.com/recinq/wave/internal/event"
)

// ProcessedItemTracker is the subset of state.StateStore needed for continuous mode.
type ProcessedItemTracker interface {
	MarkItemProcessed(pipelineName, itemKey, runID string) error
	IsItemProcessed(pipelineName, itemKey string) (bool, error)
}

// RunnerConfig configures the continuous execution runner.
type RunnerConfig struct {
	Provider        WorkItemProvider
	PipelineFactory func(input string) error
	Emitter         event.EventEmitter
	Store           ProcessedItemTracker
	PipelineName    string
	Delay           time.Duration
	HaltOnError     bool
	MaxIterations   int // 0 = unlimited
}

// Runner orchestrates continuous pipeline execution over work items.
type Runner struct {
	cfg RunnerConfig
}

// NewRunner creates a continuous execution runner.
func NewRunner(cfg RunnerConfig) *Runner {
	return &Runner{cfg: cfg}
}

// Run executes the continuous loop: fetch items, run pipelines, track progress.
// Returns nil on clean exhaustion or graceful shutdown.
func (r *Runner) Run(ctx context.Context) error {
	r.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: r.cfg.PipelineName,
		State:      event.StateContinuousStarted,
		Message:    fmt.Sprintf("Continuous mode started for pipeline %s", r.cfg.PipelineName),
	})

	iteration := 0
	for {
		// Check for context cancellation before fetching next item
		if ctx.Err() != nil {
			r.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: r.cfg.PipelineName,
				State:      event.StateContinuousStopped,
				Message:    fmt.Sprintf("Continuous mode stopped after %d iterations", iteration),
			})
			return nil
		}

		// Check max iterations limit
		if r.cfg.MaxIterations > 0 && iteration >= r.cfg.MaxIterations {
			r.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: r.cfg.PipelineName,
				State:      event.StateContinuousExhausted,
				Message:    fmt.Sprintf("Max iterations reached (%d)", r.cfg.MaxIterations),
			})
			return nil
		}

		// Fetch next work item
		item, err := r.cfg.Provider.Next(ctx)
		if err != nil {
			if ctx.Err() != nil {
				r.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: r.cfg.PipelineName,
					State:      event.StateContinuousStopped,
					Message:    "Continuous mode stopped during item fetch",
				})
				return nil
			}
			return fmt.Errorf("failed to fetch next work item: %w", err)
		}

		// No more items — we're done
		if item == nil {
			r.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: r.cfg.PipelineName,
				State:      event.StateContinuousExhausted,
				Message:    fmt.Sprintf("No more work items after %d iterations", iteration),
			})
			return nil
		}

		iteration++

		r.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: r.cfg.PipelineName,
			State:      event.StateContinuousIterationStarted,
			Message:    fmt.Sprintf("Starting iteration %d: %s", iteration, item.Key),
		})

		// Execute the pipeline for this item
		execErr := r.cfg.PipelineFactory(item.Input)
		if execErr != nil {
			r.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: r.cfg.PipelineName,
				State:      event.StateContinuousIterationFailed,
				Message:    fmt.Sprintf("Iteration %d failed (%s): %v", iteration, item.Key, execErr),
			})

			if r.cfg.HaltOnError {
				return fmt.Errorf("continuous mode halted on iteration %d (%s): %w", iteration, item.Key, execErr)
			}
			// Skip and continue — still mark as processed to avoid retrying
			if r.cfg.Store != nil {
				r.cfg.Store.MarkItemProcessed(r.cfg.PipelineName, item.Key, "")
			}
		} else {
			// Mark as processed on success
			if r.cfg.Store != nil {
				r.cfg.Store.MarkItemProcessed(r.cfg.PipelineName, item.Key, "")
			}

			r.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: r.cfg.PipelineName,
				State:      event.StateContinuousIterationCompleted,
				Message:    fmt.Sprintf("Iteration %d completed: %s", iteration, item.Key),
			})
		}

		// Delay between iterations (respect context cancellation)
		if r.cfg.Delay > 0 {
			select {
			case <-ctx.Done():
				r.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: r.cfg.PipelineName,
					State:      event.StateContinuousStopped,
					Message:    fmt.Sprintf("Continuous mode stopped during delay after %d iterations", iteration),
				})
				return nil
			case <-time.After(r.cfg.Delay):
			}
		}
	}
}

func (r *Runner) emit(ev event.Event) {
	if r.cfg.Emitter != nil {
		r.cfg.Emitter.Emit(ev)
	}
}
