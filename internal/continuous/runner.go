package continuous

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/recinq/wave/internal/event"
)

// ExecutorFunc executes a pipeline for a single work item.
// It receives a context and returns the run ID and any error.
type ExecutorFunc func(ctx context.Context, input string) (runID string, err error)

// Runner manages the continuous pipeline execution loop.
type Runner struct {
	Source          WorkItemSource
	PipelineName    string
	OnFailure       FailurePolicy
	MaxIterations   int
	Delay           time.Duration
	Emitter         event.EventEmitter
	ExecutorFactory func(input string) ExecutorFunc
}

// Summary holds the aggregate results of a continuous run.
type Summary struct {
	Total     int
	Succeeded int
	Failed    int
	Skipped   int
	Duration  time.Duration
}

// String returns a human-readable summary.
func (s *Summary) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Continuous run complete: %d iterations", s.Total)
	if s.Succeeded > 0 {
		fmt.Fprintf(&b, ", %d succeeded", s.Succeeded)
	}
	if s.Failed > 0 {
		fmt.Fprintf(&b, ", %d failed", s.Failed)
	}
	if s.Skipped > 0 {
		fmt.Fprintf(&b, ", %d skipped", s.Skipped)
	}
	fmt.Fprintf(&b, " (%s)", s.Duration.Truncate(time.Millisecond))
	return b.String()
}

// HasFailures returns true if any iteration failed.
func (s *Summary) HasFailures() bool {
	return s.Failed > 0
}

// Run executes the continuous pipeline loop.
func (r *Runner) Run(ctx context.Context) (*Summary, error) {
	startTime := time.Now()
	summary := &Summary{}
	processedIDs := make(map[string]bool)

	// Emit loop start
	if r.Emitter != nil {
		r.Emitter.Emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: r.PipelineName,
			State:      event.StateLoopStart,
			Message:    fmt.Sprintf("Starting continuous run from source: %s", r.Source.Name()),
		})
	}

	iteration := 0
	for ctx.Err() == nil {
		// Check max iterations
		if r.MaxIterations > 0 && iteration >= r.MaxIterations {
			break
		}

		// Get next work item
		item, err := r.Source.Next(ctx)
		if err != nil {
			// Source error — treat as fatal
			summary.Duration = time.Since(startTime)
			return summary, fmt.Errorf("source error: %w", err)
		}
		if item == nil {
			// Source exhausted
			break
		}

		// Dedup check
		if processedIDs[item.ID] {
			summary.Skipped++
			summary.Total++
			iteration++
			continue
		}
		processedIDs[item.ID] = true

		iteration++
		iterStart := time.Now()

		// Emit iteration start
		if r.Emitter != nil {
			r.Emitter.Emit(event.Event{
				Timestamp:      time.Now(),
				PipelineID:     r.PipelineName,
				State:          event.StateLoopIterationStart,
				Message:        fmt.Sprintf("Starting iteration %d: %s", iteration, item.ID),
				Iteration:      iteration,
				TotalProcessed: summary.Total,
				WorkItemID:     item.ID,
			})
		}

		// Execute pipeline
		executor := r.ExecutorFactory(item.Input)
		_, execErr := executor(ctx, item.Input)
		iterDuration := time.Since(iterStart)

		if execErr != nil {
			summary.Failed++

			if r.Emitter != nil {
				r.Emitter.Emit(event.Event{
					Timestamp:      time.Now(),
					PipelineID:     r.PipelineName,
					State:          event.StateLoopIterationFailed,
					Message:        fmt.Sprintf("Iteration %d failed: %v", iteration, execErr),
					Iteration:      iteration,
					TotalProcessed: summary.Total + 1,
					WorkItemID:     item.ID,
					DurationMs:     iterDuration.Milliseconds(),
				})
			}

			// Check failure policy
			if r.OnFailure == FailurePolicyHalt {
				summary.Total++
				break
			}
		} else {
			summary.Succeeded++

			if r.Emitter != nil {
				r.Emitter.Emit(event.Event{
					Timestamp:      time.Now(),
					PipelineID:     r.PipelineName,
					State:          event.StateLoopIterationComplete,
					Message:        fmt.Sprintf("Iteration %d completed successfully", iteration),
					Iteration:      iteration,
					TotalProcessed: summary.Total + 1,
					WorkItemID:     item.ID,
					DurationMs:     iterDuration.Milliseconds(),
				})
			}
		}

		summary.Total++

		// Delay between iterations (skip if context is cancelled)
		if r.Delay > 0 {
			select {
			case <-ctx.Done():
			case <-time.After(r.Delay):
			}
		}
	}

	summary.Duration = time.Since(startTime)

	// Emit loop summary
	if r.Emitter != nil {
		r.Emitter.Emit(event.Event{
			Timestamp:      time.Now(),
			PipelineID:     r.PipelineName,
			State:          event.StateLoopSummary,
			Message:        summary.String(),
			TotalProcessed: summary.Total,
		})
	}

	return summary, nil
}
