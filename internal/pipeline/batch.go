package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/recinq/wave/internal/event"
	"golang.org/x/sync/errgroup"
)

// PipelineBatchExecutor orchestrates concurrent execution of multiple independent
// pipeline DAGs. It uses Kahn's algorithm for dependency-tier computation and
// errgroup with SetLimit for concurrency control. This follows the composition
// pattern of MatrixExecutor — delegating to DefaultPipelineExecutor instances
// rather than modifying the single-pipeline execution flow.
type PipelineBatchExecutor struct {
	executor *DefaultPipelineExecutor
	debug    bool
}

// NewPipelineBatchExecutor creates a batch executor that delegates pipeline
// execution to child executors derived from the given parent executor.
func NewPipelineBatchExecutor(executor *DefaultPipelineExecutor) *PipelineBatchExecutor {
	return &PipelineBatchExecutor{
		executor: executor,
		debug:    executor.debug,
	}
}

// BatchArtifactRegistry provides thread-safe storage of artifact paths produced
// by pipelines in a batch. Keys follow the format "pipelineName:stepID:artifactName".
type BatchArtifactRegistry struct {
	mu    sync.RWMutex
	paths map[string]string
}

// newBatchArtifactRegistry creates an empty artifact registry.
func newBatchArtifactRegistry() *BatchArtifactRegistry {
	return &BatchArtifactRegistry{
		paths: make(map[string]string),
	}
}

// Register adds an artifact path to the registry.
func (r *BatchArtifactRegistry) Register(pipelineName, stepID, artifactName, path string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := fmt.Sprintf("%s:%s:%s", pipelineName, stepID, artifactName)
	r.paths[key] = path
}

// Get retrieves an artifact path from the registry.
func (r *BatchArtifactRegistry) Get(pipelineName, stepID, artifactName string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := fmt.Sprintf("%s:%s:%s", pipelineName, stepID, artifactName)
	path, ok := r.paths[key]
	return path, ok
}

// GetAllForPipeline returns all artifact paths registered for a given pipeline name.
func (r *BatchArtifactRegistry) GetAllForPipeline(pipelineName string) map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]string)
	prefix := pipelineName + ":"
	for key, path := range r.paths {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			result[key[len(prefix):]] = path
		}
	}
	return result
}

// batchEventInterceptor wraps an EventEmitter to capture per-pipeline token counts.
type batchEventInterceptor struct {
	inner  event.EventEmitter
	mu     sync.Mutex
	tokens map[string]int
}

func newBatchEventInterceptor(inner event.EventEmitter) *batchEventInterceptor {
	return &batchEventInterceptor{
		inner:  inner,
		tokens: make(map[string]int),
	}
}

func (i *batchEventInterceptor) Emit(ev event.Event) {
	if ev.TokensUsed > 0 && ev.PipelineID != "" {
		i.mu.Lock()
		i.tokens[ev.PipelineID] += ev.TokensUsed
		i.mu.Unlock()
	}
	if i.inner != nil {
		i.inner.Emit(ev)
	}
}

func (i *batchEventInterceptor) getTotalTokens() int {
	i.mu.Lock()
	defer i.mu.Unlock()
	total := 0
	for _, t := range i.tokens {
		total += t
	}
	return total
}

// ExecuteBatch orchestrates concurrent execution of multiple pipelines according
// to their dependency ordering and the configured error policy.
func (b *PipelineBatchExecutor) ExecuteBatch(ctx context.Context, config *PipelineBatchConfig) (*PipelineBatchResult, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid batch config: %w", err)
	}

	batchStart := time.Now()

	// Build name-to-entry lookup
	entryByName := make(map[string]*PipelineBatchEntry, len(config.Pipelines))
	for i := range config.Pipelines {
		entryByName[config.Pipelines[i].Name] = &config.Pipelines[i]
	}

	// Build name set for tier computation
	names := make(map[string]bool, len(config.Pipelines))
	for _, entry := range config.Pipelines {
		names[entry.Name] = true
	}

	deps := config.Dependencies
	if deps == nil {
		deps = make(map[string][]string)
	}

	tiers, err := computePipelineTiers(names, deps)
	if err != nil {
		return nil, fmt.Errorf("failed to compute pipeline tiers: %w", err)
	}

	// Emit batch_started event
	b.emit(event.Event{
		Timestamp: time.Now(),
		State:     event.StateBatchStarted,
		Message:   fmt.Sprintf("Starting batch execution: %d pipelines in %d tiers", len(config.Pipelines), len(tiers)),
	})

	// Track results and failures
	var resultsMu sync.Mutex
	results := make([]PipelineRunResult, 0, len(config.Pipelines))
	failed := make(map[string]bool)
	artifactRegistry := newBatchArtifactRegistry()

	// Determine max concurrency
	maxConcurrency := config.MaxConcurrentPipelines
	if maxConcurrency <= 0 {
		maxConcurrency = len(config.Pipelines)
	}

	// Execute tiers sequentially
	for tierIdx, tier := range tiers {
		if ctx.Err() != nil {
			for _, name := range tier {
				resultsMu.Lock()
				results = append(results, PipelineRunResult{
					Name:       name,
					Status:     RunStatusSkipped,
					SkipReason: "batch context cancelled",
				})
				resultsMu.Unlock()
			}
			continue
		}

		// Filter out pipelines whose dependencies failed
		var runnablePipelines []string
		for _, name := range tier {
			if shouldSkip, reason := b.shouldSkipPipeline(name, deps, failed); shouldSkip {
				resultsMu.Lock()
				results = append(results, PipelineRunResult{
					Name:       name,
					Status:     RunStatusSkipped,
					SkipReason: reason,
				})
				resultsMu.Unlock()
				continue
			}
			runnablePipelines = append(runnablePipelines, name)
		}

		if len(runnablePipelines) == 0 {
			continue
		}

		var tierErr error
		switch config.OnFailure {
		case OnFailureAbortAll:
			tierErr = b.executeTierAbortAll(ctx, tierIdx, runnablePipelines, entryByName, maxConcurrency, &resultsMu, &results, failed, artifactRegistry)
		default:
			tierErr = b.executeTierContinue(ctx, tierIdx, runnablePipelines, entryByName, maxConcurrency, &resultsMu, &results, failed, artifactRegistry)
		}

		if tierErr != nil && config.OnFailure == OnFailureAbortAll {
			for remainingTier := tierIdx + 1; remainingTier < len(tiers); remainingTier++ {
				for _, name := range tiers[remainingTier] {
					resultsMu.Lock()
					results = append(results, PipelineRunResult{
						Name:       name,
						Status:     RunStatusSkipped,
						SkipReason: "batch aborted due to pipeline failure",
					})
					resultsMu.Unlock()
				}
			}
			break
		}
	}

	// Aggregate results
	batchResult := &PipelineBatchResult{
		Results:       results,
		TotalDuration: time.Since(batchStart),
	}
	for _, r := range results {
		batchResult.TotalTokens += r.TokensUsed
		switch r.Status {
		case RunStatusCompleted:
			batchResult.CompletedCount++
		case RunStatusFailed:
			batchResult.FailedCount++
		case RunStatusSkipped:
			batchResult.SkippedCount++
		}
	}

	b.emit(event.Event{
		Timestamp:  time.Now(),
		State:      event.StateBatchCompleted,
		DurationMs: time.Since(batchStart).Milliseconds(),
		Message:    fmt.Sprintf("Batch completed: %d succeeded, %d failed, %d skipped", batchResult.CompletedCount, batchResult.FailedCount, batchResult.SkippedCount),
		TokensUsed: batchResult.TotalTokens,
	})

	return batchResult, nil
}

// executeTierContinue runs pipelines in a tier with the "continue" error policy.
// Failures are recorded but do not cancel sibling pipelines.
func (b *PipelineBatchExecutor) executeTierContinue(
	ctx context.Context,
	tierIdx int,
	pipelines []string,
	entries map[string]*PipelineBatchEntry,
	maxConcurrency int,
	resultsMu *sync.Mutex,
	results *[]PipelineRunResult,
	failed map[string]bool,
	registry *BatchArtifactRegistry,
) error {
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrency)

	for _, name := range pipelines {
		name := name
		entry := entries[name]

		g.Go(func() error {
			result := b.executeSinglePipeline(gctx, tierIdx, name, entry, registry)

			resultsMu.Lock()
			*results = append(*results, result)
			if result.Status == RunStatusFailed {
				failed[name] = true
			}
			resultsMu.Unlock()

			return nil
		})
	}

	return g.Wait()
}

// executeTierAbortAll runs pipelines in a tier with the "abort-all" error policy.
// The first failure cancels all running pipelines via context cancellation.
func (b *PipelineBatchExecutor) executeTierAbortAll(
	ctx context.Context,
	tierIdx int,
	pipelines []string,
	entries map[string]*PipelineBatchEntry,
	maxConcurrency int,
	resultsMu *sync.Mutex,
	results *[]PipelineRunResult,
	failed map[string]bool,
	registry *BatchArtifactRegistry,
) error {
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrency)

	var firstErr error
	var errOnce sync.Once

	for _, name := range pipelines {
		name := name
		entry := entries[name]

		g.Go(func() error {
			result := b.executeSinglePipeline(gctx, tierIdx, name, entry, registry)

			resultsMu.Lock()
			*results = append(*results, result)
			if result.Status == RunStatusFailed {
				failed[name] = true
			}
			resultsMu.Unlock()

			if result.Status == RunStatusFailed {
				errOnce.Do(func() {
					firstErr = result.Error
				})
				return result.Error
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return firstErr
	}
	return nil
}

// executeSinglePipeline runs one pipeline and returns its result.
func (b *PipelineBatchExecutor) executeSinglePipeline(
	ctx context.Context,
	tierIdx int,
	name string,
	entry *PipelineBatchEntry,
	registry *BatchArtifactRegistry,
) PipelineRunResult {
	result := PipelineRunResult{
		Name:          name,
		ArtifactPaths: make(map[string]string),
	}

	start := time.Now()

	b.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: name,
		State:      event.StateBatchPipelineStarted,
		Message:    fmt.Sprintf("Pipeline %q starting in tier %d", name, tierIdx),
	})

	// Create a child executor with a token-tracking event interceptor
	childExecutor := b.executor.NewChildExecutor()
	interceptor := newBatchEventInterceptor(childExecutor.emitter)
	childExecutor.emitter = interceptor

	// Execute the pipeline
	err := childExecutor.Execute(ctx, entry.Pipeline, entry.Manifest, entry.Input)
	result.Duration = time.Since(start)
	result.TokensUsed = interceptor.getTotalTokens()

	if err != nil {
		result.Status = RunStatusFailed
		result.Error = err

		b.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: name,
			State:      event.StateBatchPipelineFailed,
			Message:    fmt.Sprintf("Pipeline %q failed: %v", name, err),
			DurationMs: result.Duration.Milliseconds(),
		})
	} else {
		result.Status = RunStatusCompleted

		// Register output artifacts in the batch registry
		b.registerPipelineArtifacts(name, entry, result.ArtifactPaths, registry)

		b.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: name,
			State:      event.StateBatchPipelineCompleted,
			Message:    fmt.Sprintf("Pipeline %q completed", name),
			DurationMs: result.Duration.Milliseconds(),
			TokensUsed: result.TokensUsed,
		})
	}

	return result
}

// registerPipelineArtifacts records output artifact paths from a completed pipeline's
// step definitions into both the per-pipeline result and the shared batch registry.
func (b *PipelineBatchExecutor) registerPipelineArtifacts(
	pipelineName string,
	entry *PipelineBatchEntry,
	resultArtifacts map[string]string,
	registry *BatchArtifactRegistry,
) {
	for _, step := range entry.Pipeline.Steps {
		for _, artifact := range step.OutputArtifacts {
			if artifact.Path == "" {
				continue
			}
			key := step.ID + ":" + artifact.Name
			resultArtifacts[key] = artifact.Path
			registry.Register(pipelineName, step.ID, artifact.Name, artifact.Path)
		}
	}
}

// shouldSkipPipeline checks if a pipeline should be skipped because one of its
// dependencies failed.
func (b *PipelineBatchExecutor) shouldSkipPipeline(name string, deps map[string][]string, failed map[string]bool) (bool, string) {
	for _, dep := range deps[name] {
		if failed[dep] {
			return true, fmt.Sprintf("dependency %q failed", dep)
		}
	}
	return false, ""
}

// emit sends an event through the parent executor's event emitter.
func (b *PipelineBatchExecutor) emit(ev event.Event) {
	if b.executor.emitter != nil {
		b.executor.emitter.Emit(ev)
	}
}
