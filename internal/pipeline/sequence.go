package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/state"
)

// SequenceResult holds the outcome of a sequence execution.
type SequenceResult struct {
	PipelineResults []PipelineResult
	TotalTokens     int
}

// PipelineResult holds the outcome of a single pipeline within a sequence.
type PipelineResult struct {
	PipelineName string
	RunID        string
	Status       string // "completed", "failed"
	Error        error
	TokensUsed   int
	Duration     time.Duration
}

// SequenceExecutor runs a list of pipelines in order.
type SequenceExecutor struct {
	newExecutor func(opts ...ExecutorOption) *DefaultPipelineExecutor
	baseOpts    []ExecutorOption
	emitter     event.EventEmitter
	store       state.StateStore
}

// NewSequenceExecutor creates a new sequence executor.
//
// newExecutor is a factory function that creates a fresh executor for each
// pipeline in the sequence. baseOpts are applied to every executor, with
// per-pipeline overrides (run ID, emitter, store) appended on top.
func NewSequenceExecutor(
	newExecutor func(opts ...ExecutorOption) *DefaultPipelineExecutor,
	baseOpts []ExecutorOption,
	emitter event.EventEmitter,
	store state.StateStore,
) *SequenceExecutor {
	return &SequenceExecutor{
		newExecutor: newExecutor,
		baseOpts:    baseOpts,
		emitter:     emitter,
		store:       store,
	}
}

// Execute runs the given pipelines in sequence. Each pipeline gets the same
// input string. If any pipeline fails, execution stops and an error is
// returned along with the partial result.
func (s *SequenceExecutor) Execute(ctx context.Context, pipelines []*Pipeline, m *manifest.Manifest, input string) (*SequenceResult, error) {
	result := &SequenceResult{}

	if len(pipelines) == 0 {
		return result, nil
	}

	s.emit(event.Event{
		Timestamp: time.Now(),
		State:     event.StateSequenceStarted,
		Message:   fmt.Sprintf("Starting sequence of %d pipelines", len(pipelines)),
	})

	for i, p := range pipelines {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		pipelineName := p.Metadata.Name

		s.emit(event.Event{
			Timestamp:  time.Now(),
			State:      event.StateSequenceProgress,
			PipelineID: pipelineName,
			Message:    fmt.Sprintf("Pipeline %d/%d: %s", i+1, len(pipelines), pipelineName),
		})

		// Build per-pipeline options: start from shared base, then add
		// pipeline-specific run ID, emitter, and store.
		opts := make([]ExecutorOption, len(s.baseOpts))
		copy(opts, s.baseOpts)

		runID := GenerateRunID(pipelineName, 8)
		if s.store != nil {
			if storeRunID, err := s.store.CreateRun(pipelineName, input); err == nil {
				runID = storeRunID
			}
		}
		opts = append(opts, WithRunID(runID))
		if s.emitter != nil {
			opts = append(opts, WithEmitter(s.emitter))
		}
		if s.store != nil {
			opts = append(opts, WithStateStore(s.store))
		}

		executor := s.newExecutor(opts...)

		startTime := time.Now()
		execErr := executor.Execute(ctx, p, m, input)
		duration := time.Since(startTime)

		pr := PipelineResult{
			PipelineName: pipelineName,
			RunID:        runID,
			TokensUsed:   executor.GetTotalTokens(),
			Duration:     duration,
		}

		if execErr != nil {
			pr.Status = "failed"
			pr.Error = execErr
			result.PipelineResults = append(result.PipelineResults, pr)
			result.TotalTokens += pr.TokensUsed

			s.emit(event.Event{
				Timestamp:     time.Now(),
				State:         event.StateSequenceFailed,
				PipelineID:    pipelineName,
				Message:       fmt.Sprintf("Sequence stopped at pipeline %d/%d (%s): %s", i+1, len(pipelines), pipelineName, execErr.Error()),
				FailureReason: "pipeline_failed",
			})

			return result, fmt.Errorf("sequence failed at pipeline %d/%d (%s): %w", i+1, len(pipelines), pipelineName, execErr)
		}

		pr.Status = "completed"
		result.PipelineResults = append(result.PipelineResults, pr)
		result.TotalTokens += pr.TokensUsed
	}

	s.emit(event.Event{
		Timestamp: time.Now(),
		State:     event.StateSequenceCompleted,
		Message:   fmt.Sprintf("Sequence completed: %d pipelines, %d total tokens", len(pipelines), result.TotalTokens),
	})

	return result, nil
}

func (s *SequenceExecutor) emit(ev event.Event) {
	if s.emitter != nil {
		s.emitter.Emit(ev)
	}
}
