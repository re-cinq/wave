package pipeline

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/state"
	"golang.org/x/sync/errgroup"
)

// errParallelStagePartialFailure indicates that one or more pipelines in a
// parallel stage failed while other pipelines completed successfully.
var errParallelStagePartialFailure = errors.New("parallel stage partial failure")

// SequenceResult holds the outcome of a sequence execution.
type SequenceResult struct {
	PipelineResults []PipelineResult
	TotalTokens     int
}

// PipelineResult holds the outcome of a single pipeline within a sequence.
type PipelineResult struct {
	PipelineName string
	RunID        string
	Status       string // stateCompleted, stateFailed
	Error        error
	TokensUsed   int
	Duration     time.Duration
}

// SequenceExecutor runs a list of pipelines in order.
type SequenceExecutor struct {
	emitterMixin
	newExecutor     func(opts ...ExecutorOption) *DefaultPipelineExecutor
	baseOpts        []ExecutorOption
	store           state.StateStore
	mu              sync.Mutex                   // protects pipelineOutputs
	pipelineOutputs map[string]map[string][]byte // pipelineName -> artifactName -> data
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
		emitterMixin:    emitterMixin{emitter: emitter},
		newExecutor:     newExecutor,
		baseOpts:        baseOpts,
		store:           store,
		pipelineOutputs: make(map[string]map[string][]byte),
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
		if len(s.pipelineOutputs) > 0 {
			opts = append(opts, WithCrossPipelineArtifacts(s.pipelineOutputs))
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
			pr.Status = stateFailed
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

		pr.Status = stateCompleted
		result.PipelineResults = append(result.PipelineResults, pr)
		result.TotalTokens += pr.TokensUsed

		s.recordPipelineOutputs(p, runID, ".agents/workspaces")
	}

	s.emit(event.Event{
		Timestamp: time.Now(),
		State:     event.StateSequenceCompleted,
		Message:   fmt.Sprintf("Sequence completed: %d pipelines, %d total tokens", len(pipelines), result.TotalTokens),
	})

	return result, nil
}

// Stage groups pipelines that can run together.
type Stage struct {
	Pipelines []*Pipeline
	Parallel  bool
}

// ExecutionPlan is an ordered list of stages.
type ExecutionPlan struct {
	Stages        []Stage
	FailFast      bool // stop on first failure (default true)
	MaxConcurrent int  // max concurrent pipelines per parallel stage (0 = unlimited)
}

// ExecutePlan runs an execution plan with support for parallel stages.
// Sequential stages run pipelines one after another (same as Execute).
// Parallel stages run all pipelines concurrently using errgroup.
func (s *SequenceExecutor) ExecutePlan(ctx context.Context, plan ExecutionPlan, m *manifest.Manifest, input string) (*SequenceResult, error) {
	result := &SequenceResult{}

	totalPipelines := 0
	for _, stage := range plan.Stages {
		totalPipelines += len(stage.Pipelines)
	}
	if totalPipelines == 0 {
		return result, nil
	}

	s.emit(event.Event{
		Timestamp: time.Now(),
		State:     event.StateSequenceStarted,
		Message:   fmt.Sprintf("Starting execution plan: %d stages, %d pipelines", len(plan.Stages), totalPipelines),
	})

	var hadFailures bool

	for stageIdx, stage := range plan.Stages {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		if stage.Parallel && len(stage.Pipelines) > 1 {
			stageResults, err := s.executeParallelStage(ctx, stage, stageIdx, m, input, plan.FailFast, plan.MaxConcurrent)
			result.PipelineResults = append(result.PipelineResults, stageResults...)
			for _, pr := range stageResults {
				result.TotalTokens += pr.TokensUsed
			}
			if err != nil {
				if !plan.FailFast && errors.Is(err, errParallelStagePartialFailure) {
					hadFailures = true
				} else {
					return result, err
				}
			}
		} else {
			for _, p := range stage.Pipelines {
				pr, err := s.executeSinglePipeline(ctx, p, m, input)
				result.PipelineResults = append(result.PipelineResults, pr)
				result.TotalTokens += pr.TokensUsed
				if err != nil {
					if plan.FailFast {
						s.emit(event.Event{
							Timestamp:     time.Now(),
							State:         event.StateSequenceFailed,
							PipelineID:    p.Metadata.Name,
							Message:       fmt.Sprintf("Plan stopped: %s failed", p.Metadata.Name),
							FailureReason: "pipeline_failed",
						})
						return result, err
					}
					hadFailures = true
				}
			}
		}
	}

	if hadFailures {
		s.emit(event.Event{
			Timestamp:     time.Now(),
			State:         event.StateSequenceFailed,
			Message:       fmt.Sprintf("Plan completed with failures: %d pipelines, %d total tokens", len(result.PipelineResults), result.TotalTokens),
			FailureReason: "partial_failure",
		})
		// Collect all failed pipeline names for the aggregate error
		var failedNames []string
		for _, pr := range result.PipelineResults {
			if pr.Status == stateFailed {
				failedNames = append(failedNames, pr.PipelineName)
			}
		}
		return result, fmt.Errorf("%w: %v", errParallelStagePartialFailure, failedNames)
	}

	s.emit(event.Event{
		Timestamp: time.Now(),
		State:     event.StateSequenceCompleted,
		Message:   fmt.Sprintf("Plan completed: %d pipelines, %d total tokens", len(result.PipelineResults), result.TotalTokens),
	})

	return result, nil
}

// executeParallelStage runs all pipelines in a stage concurrently.
func (s *SequenceExecutor) executeParallelStage(ctx context.Context, stage Stage, stageIdx int, m *manifest.Manifest, input string, failFast bool, maxConcurrent int) ([]PipelineResult, error) {
	names := make([]string, len(stage.Pipelines))
	for i, p := range stage.Pipelines {
		names[i] = p.Metadata.Name
	}

	s.emit(event.Event{
		Timestamp: time.Now(),
		State:     event.StateParallelStageStarted,
		Message:   fmt.Sprintf("Parallel stage %d: %d pipelines", stageIdx+1, len(stage.Pipelines)),
	})

	var mu sync.Mutex
	results := make([]PipelineResult, len(stage.Pipelines))

	g, gctx := errgroup.WithContext(ctx)
	if maxConcurrent > 0 {
		g.SetLimit(maxConcurrent)
	}
	for i, p := range stage.Pipelines {
		g.Go(func() error {
			pr, err := s.executeSinglePipeline(gctx, p, m, input)
			mu.Lock()
			results[i] = pr
			mu.Unlock()
			if err != nil && failFast {
				return err
			}
			return nil
		})
	}

	err := g.Wait()

	// In non-fail-fast mode, errgroup returns nil (goroutines return nil),
	// but individual pipelines may have failed. Scan results for failures.
	if err == nil && !failFast {
		var failedNames []string
		for _, pr := range results {
			if pr.Status == stateFailed {
				failedNames = append(failedNames, pr.PipelineName)
			}
		}
		if len(failedNames) > 0 {
			err = fmt.Errorf("%w: %v", errParallelStagePartialFailure, failedNames)
		}
	}

	state := event.StateParallelStageCompleted
	if err != nil {
		state = event.StateParallelStageFailed
	}
	s.emit(event.Event{
		Timestamp: time.Now(),
		State:     state,
		Message:   fmt.Sprintf("Parallel stage %d completed", stageIdx+1),
	})

	return results, err
}

// executeSinglePipeline runs one pipeline and returns its result.
func (s *SequenceExecutor) executeSinglePipeline(ctx context.Context, p *Pipeline, m *manifest.Manifest, input string) (PipelineResult, error) {
	pipelineName := p.Metadata.Name

	s.emit(event.Event{
		Timestamp:  time.Now(),
		State:      event.StateSequenceProgress,
		PipelineID: pipelineName,
		Message:    fmt.Sprintf("Starting pipeline: %s", pipelineName),
	})

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
	s.mu.Lock()
	if len(s.pipelineOutputs) > 0 {
		opts = append(opts, WithCrossPipelineArtifacts(s.pipelineOutputs))
	}
	s.mu.Unlock()

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
		pr.Status = stateFailed
		pr.Error = execErr
		return pr, execErr
	}

	pr.Status = stateCompleted
	s.recordPipelineOutputs(p, runID, ".agents/workspaces")
	return pr, nil
}

// recordPipelineOutputs captures output artifacts from a completed pipeline
// for use by subsequent pipelines in the sequence.
func (s *SequenceExecutor) recordPipelineOutputs(p *Pipeline, runID string, wsRoot string) {
	if len(p.Steps) == 0 {
		return
	}

	pipelineName := p.Metadata.Name
	outputs := make(map[string][]byte)
	terminalStep := p.Steps[len(p.Steps)-1]
	for _, art := range terminalStep.OutputArtifacts {
		data, err := LoadStepArtifact(wsRoot, runID, terminalStep.ID, art.Name)
		if err == nil {
			outputs[art.Name] = data
		} else {
			s.emit(event.Event{
				Timestamp:  time.Now(),
				State:      "warning",
				PipelineID: pipelineName,
				StepID:     terminalStep.ID,
				Message:    fmt.Sprintf("Failed to load output artifact %q from step %q: %v", art.Name, terminalStep.ID, err),
			})
		}
	}

	// Also check pipeline_outputs aliases
	for name, po := range p.PipelineOutputs {
		for _, step := range p.Steps {
			if step.ID == po.Step {
				for _, art := range step.OutputArtifacts {
					if art.Name == po.Artifact {
						data, err := LoadStepArtifact(wsRoot, runID, step.ID, art.Name)
						if err == nil {
							if po.Field != "" {
								val, extractErr := ExtractJSONPath(data, "."+po.Field)
								if extractErr == nil {
									outputs[name] = []byte(val)
								} else {
									s.emit(event.Event{
										Timestamp:  time.Now(),
										State:      "warning",
										PipelineID: pipelineName,
										StepID:     step.ID,
										Message:    fmt.Sprintf("Failed to extract field %q from artifact %q in step %q: %v", po.Field, art.Name, step.ID, extractErr),
									})
								}
							} else {
								outputs[name] = data
							}
						} else {
							s.emit(event.Event{
								Timestamp:  time.Now(),
								State:      "warning",
								PipelineID: pipelineName,
								StepID:     step.ID,
								Message:    fmt.Sprintf("Failed to load pipeline output artifact %q from step %q: %v", art.Name, step.ID, err),
							})
						}
					}
				}
			}
		}
	}

	if len(outputs) > 0 {
		s.mu.Lock()
		s.pipelineOutputs[pipelineName] = outputs
		s.mu.Unlock()
	}
}

// GetPipelineOutputs returns the captured outputs for cross-pipeline handoff.
func (s *SequenceExecutor) GetPipelineOutputs() map[string]map[string][]byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pipelineOutputs
}
