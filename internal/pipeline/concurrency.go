package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/workspace"
	"golang.org/x/sync/errgroup"
)

// ConcurrencyExecutor handles spawning multiple identical agents for a single step.
// Unlike MatrixExecutor which fans out over different items, ConcurrencyExecutor
// runs N copies of the same step in parallel with isolated workspaces.
type ConcurrencyExecutor struct {
	executor *DefaultPipelineExecutor
}

// NewConcurrencyExecutor creates a new ConcurrencyExecutor.
func NewConcurrencyExecutor(executor *DefaultPipelineExecutor) *ConcurrencyExecutor {
	return &ConcurrencyExecutor{executor: executor}
}

// Execute runs N concurrent workers for the step, where N is step.Concurrency.
// It uses fail-fast semantics: the errgroup cancels remaining workers on first failure.
// The concurrency is capped by the manifest's runtime.max_concurrent_workers setting.
func (c *ConcurrencyExecutor) Execute(ctx context.Context, execution *PipelineExecution, step *Step) error {
	if step.Concurrency <= 1 {
		return fmt.Errorf("step %q has concurrency %d, which does not require concurrent execution", step.ID, step.Concurrency)
	}

	pipelineID := execution.Status.ID
	workerCount := step.Concurrency

	// Cap by runtime.max_concurrent_workers if set
	maxWorkers := execution.Manifest.Runtime.MaxConcurrentWorkers
	if maxWorkers <= 0 {
		maxWorkers = 10 // Default cap
	}
	if workerCount > maxWorkers {
		workerCount = maxWorkers
	}

	// Emit concurrent start event
	c.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      "concurrent_start",
		Message:    fmt.Sprintf("Starting %d concurrent workers (requested=%d, cap=%d)", workerCount, step.Concurrency, maxWorkers),
	})

	// Create errgroup with concurrency limit
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(workerCount)

	// Channel to collect results
	resultsChan := make(chan MatrixResult, workerCount)

	// Spawn workers
	for i := 0; i < workerCount; i++ {
		workerIndex := i

		g.Go(func() error {
			result := c.executeWorker(gctx, execution, step, workerIndex)
			resultsChan <- result
			if result.Error != nil {
				return result.Error
			}
			return nil
		})
	}

	// Wait for all workers to complete
	err := g.Wait()
	close(resultsChan)

	// Collect all results
	results := make([]MatrixResult, 0, workerCount)
	for result := range resultsChan {
		results = append(results, result)
	}

	// Aggregate results into execution
	c.aggregateResults(execution, step, results)

	if err != nil {
		// Collect failure information
		failedWorkers := c.collectFailedWorkers(results)
		successCount := len(results) - len(failedWorkers)

		failureMsg := fmt.Sprintf("Concurrent execution failed: %d/%d workers failed, %d succeeded",
			len(failedWorkers), workerCount, successCount)

		c.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "concurrent_failed",
			Message:    failureMsg,
		})

		return fmt.Errorf("concurrent execution failed: %d/%d workers failed. %s",
			len(failedWorkers), workerCount, c.formatFailedWorkerErrors(failedWorkers))
	}

	c.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      "concurrent_complete",
		Message:    fmt.Sprintf("All %d workers completed successfully", workerCount),
	})

	return nil
}

// executeWorker runs a single concurrent worker in an isolated workspace.
func (c *ConcurrencyExecutor) executeWorker(ctx context.Context, execution *PipelineExecution, step *Step, workerIndex int) MatrixResult {
	result := MatrixResult{
		ItemIndex: workerIndex,
	}

	pipelineID := execution.Status.ID

	// Create isolated workspace for this worker
	workspacePath, err := c.createWorkerWorkspace(execution, step, workerIndex)
	if err != nil {
		result.Error = fmt.Errorf("failed to create worker workspace: %w", err)
		return result
	}
	result.WorkspacePath = workspacePath

	c.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      "concurrent_worker_start",
		Message:    fmt.Sprintf("Worker %d starting in %s", workerIndex, workspacePath),
	})

	// Create a worker execution context (isolated from other workers)
	workerExecution := &PipelineExecution{
		Pipeline:       execution.Pipeline,
		Manifest:       execution.Manifest,
		States:         make(map[string]string),
		Results:        make(map[string]map[string]interface{}),
		ArtifactPaths:  make(map[string]string),
		WorkspacePaths: map[string]string{step.ID: workspacePath},
		Input:          execution.Input,
		Status:         execution.Status,
		Context:        execution.Context,
	}

	// Copy artifact paths from parent execution
	for k, v := range execution.ArtifactPaths {
		workerExecution.ArtifactPaths[k] = v
	}

	// Run the step execution
	err = c.executor.runStepExecution(ctx, workerExecution, step)
	if err != nil {
		result.Error = err
		c.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "concurrent_worker_failed",
			Message:    fmt.Sprintf("Worker %d failed: %v", workerIndex, err),
		})
		return result
	}

	// Collect output
	if stepResult, ok := workerExecution.Results[step.ID]; ok {
		result.Output = stepResult
	}

	c.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      "concurrent_worker_complete",
		Message:    fmt.Sprintf("Worker %d completed", workerIndex),
	})

	return result
}

// createWorkerWorkspace creates an isolated workspace for a concurrent worker.
func (c *ConcurrencyExecutor) createWorkerWorkspace(execution *PipelineExecution, step *Step, workerIndex int) (string, error) {
	pipelineID := execution.Status.ID
	wsRoot := execution.Manifest.Runtime.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = ".wave/workspaces"
	}

	// Create worker-specific workspace under .wave/workspaces/<pipeline>/<step>/worker_<index>/
	wsPath := filepath.Join(wsRoot, pipelineID, step.ID, fmt.Sprintf("worker_%d", workerIndex))
	if err := os.MkdirAll(wsPath, 0755); err != nil {
		return "", err
	}

	// If step has mounts, use workspace manager
	if c.executor.wsManager != nil && len(step.Workspace.Mount) > 0 {
		templateVars := map[string]string{
			"pipeline_id":  pipelineID,
			"step_id":      step.ID,
			"worker_index": fmt.Sprintf("%d", workerIndex),
		}
		return c.executor.wsManager.Create(workspace.WorkspaceConfig{
			Root:  wsRoot,
			Mount: toWorkspaceMounts(step.Workspace.Mount),
		}, templateVars)
	}

	return wsPath, nil
}

// aggregateResults combines all worker results into the execution state.
func (c *ConcurrencyExecutor) aggregateResults(execution *PipelineExecution, step *Step, results []MatrixResult) {
	aggregated := make(map[string]interface{})
	workerResults := make([]map[string]interface{}, 0, len(results))
	workerPaths := make([]string, 0, len(results))

	for _, result := range results {
		if result.Output != nil {
			workerResults = append(workerResults, result.Output)
		}
		workerPaths = append(workerPaths, result.WorkspacePath)
	}

	aggregated["worker_results"] = workerResults
	aggregated["worker_workspaces"] = workerPaths
	aggregated["total_workers"] = len(results)

	// Count successes and failures
	successCount := 0
	failCount := 0
	for _, result := range results {
		if result.Error == nil {
			successCount++
		} else {
			failCount++
		}
	}
	aggregated["success_count"] = successCount
	aggregated["fail_count"] = failCount

	execution.Results[step.ID] = aggregated

	// Register the first worker's workspace as the step workspace
	if len(workerPaths) > 0 {
		execution.WorkspacePaths[step.ID] = filepath.Dir(workerPaths[0])
	}
}

// collectFailedWorkers extracts failed worker information from results.
func (c *ConcurrencyExecutor) collectFailedWorkers(results []MatrixResult) []FailedWorkerInfo {
	var failed []FailedWorkerInfo
	for _, result := range results {
		if result.Error != nil {
			failed = append(failed, FailedWorkerInfo{
				Index: result.ItemIndex,
				Error: result.Error,
			})
		}
	}
	return failed
}

// formatFailedWorkerErrors formats the errors from failed workers.
func (c *ConcurrencyExecutor) formatFailedWorkerErrors(failed []FailedWorkerInfo) string {
	if len(failed) == 0 {
		return ""
	}

	maxErrors := 3
	if len(failed) < maxErrors {
		maxErrors = len(failed)
	}

	var errorStrs []string
	for i := 0; i < maxErrors; i++ {
		errorStrs = append(errorStrs, fmt.Sprintf("worker[%d]: %v", failed[i].Index, failed[i].Error))
	}

	result := strings.Join(errorStrs, "; ")
	if len(failed) > maxErrors {
		result += fmt.Sprintf(" (and %d more)", len(failed)-maxErrors)
	}
	return result
}

// emit sends an event through the executor's emitter.
func (c *ConcurrencyExecutor) emit(ev event.Event) {
	if c.executor.emitter != nil {
		c.executor.emitter.Emit(ev)
	}
}
