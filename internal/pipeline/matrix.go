package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/workspace"
	"golang.org/x/sync/errgroup"
)

// MatrixResult holds the result of a single matrix worker execution.
type MatrixResult struct {
	ItemIndex     int
	Item          interface{}
	WorkspacePath string
	Output        map[string]interface{}
	ModifiedFiles []string
	Error         error
}

// MatrixExecutor handles fan-out execution for matrix strategy steps.
type MatrixExecutor struct {
	executor *DefaultPipelineExecutor
}

// NewMatrixExecutor creates a new MatrixExecutor.
func NewMatrixExecutor(executor *DefaultPipelineExecutor) *MatrixExecutor {
	return &MatrixExecutor{executor: executor}
}

// Execute runs the matrix strategy for a step.
// It reads the items_source JSON file, spawns goroutines for each item,
// and collects results from all workers.
func (m *MatrixExecutor) Execute(ctx context.Context, execution *PipelineExecution, step *Step) error {
	if step.Strategy == nil || step.Strategy.Type != "matrix" {
		return fmt.Errorf("step %q does not have a matrix strategy", step.ID)
	}

	strategy := step.Strategy
	pipelineID := execution.Status.ID

	// Emit matrix start event
	m.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      "matrix_start",
		Message:    fmt.Sprintf("Starting matrix execution with items_source=%s", strategy.ItemsSource),
	})

	// Read items from source file
	items, err := m.readItemsSource(execution, strategy)
	if err != nil {
		return fmt.Errorf("failed to read items_source: %w", err)
	}

	// T083: Handle zero tasks gracefully
	if len(items) == 0 {
		m.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "matrix_complete",
			Message:    "No items to process",
		})
		// Initialize empty results so downstream steps can check
		execution.Results[step.ID] = map[string]interface{}{
			"worker_results":    []map[string]interface{}{},
			"worker_workspaces": []string{},
			"total_workers":     0,
			"success_count":     0,
			"fail_count":        0,
		}
		return nil
	}

	m.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      "matrix_items_loaded",
		Message:    fmt.Sprintf("Loaded %d items for parallel execution", len(items)),
	})

	// Determine max concurrency
	maxConcurrency := strategy.MaxConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = len(items) // No limit, run all in parallel
	}

	// Create errgroup with concurrency limit
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrency)

	// Channel to collect results
	resultsChan := make(chan MatrixResult, len(items))

	// Spawn workers
	for i, item := range items {
		itemIndex := i
		itemValue := item

		g.Go(func() error {
			result := m.executeWorker(gctx, execution, step, itemIndex, itemValue)
			resultsChan <- result
			if result.Error != nil {
				return result.Error
			}
			return nil
		})
	}

	// Wait for all workers to complete
	err = g.Wait()
	close(resultsChan)

	// Collect all results
	results := make([]MatrixResult, 0, len(items))
	for result := range resultsChan {
		results = append(results, result)
	}

	// Check for file conflicts
	if conflictErr := m.detectFileConflicts(results); conflictErr != nil {
		m.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "matrix_conflict",
			Message:    conflictErr.Error(),
		})
		return conflictErr
	}

	// Aggregate results into execution
	m.aggregateResults(execution, step, results)

	// T082: Improved partial failure reporting
	if err != nil {
		// Collect detailed failure information
		failedWorkers := m.collectFailedWorkers(results)
		successCount := len(results) - len(failedWorkers)

		// Build detailed failure message
		failureMsg := m.buildPartialFailureMessage(failedWorkers, successCount, len(items))

		m.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "matrix_failed",
			Message:    failureMsg,
		})

		// Return error with detailed failure information
		return fmt.Errorf("matrix execution partially failed: %d/%d workers failed. %s", len(failedWorkers), len(items), m.formatFailedWorkerErrors(failedWorkers))
	}

	m.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      "matrix_complete",
		Message:    fmt.Sprintf("Successfully processed %d items", len(items)),
	})

	return nil
}

// FailedWorkerInfo holds information about a failed worker for reporting.
type FailedWorkerInfo struct {
	Index int
	Item  interface{}
	Error error
}

// collectFailedWorkers extracts failed worker information from results.
func (m *MatrixExecutor) collectFailedWorkers(results []MatrixResult) []FailedWorkerInfo {
	var failed []FailedWorkerInfo
	for _, result := range results {
		if result.Error != nil {
			failed = append(failed, FailedWorkerInfo{
				Index: result.ItemIndex,
				Item:  result.Item,
				Error: result.Error,
			})
		}
	}
	return failed
}

// buildPartialFailureMessage creates a human-readable failure summary.
func (m *MatrixExecutor) buildPartialFailureMessage(failed []FailedWorkerInfo, successCount, totalItems int) string {
	if len(failed) == totalItems {
		return fmt.Sprintf("All %d workers failed", totalItems)
	}
	return fmt.Sprintf("Partial failure: %d/%d workers failed, %d succeeded. Failed workers: %v",
		len(failed), totalItems, successCount, m.extractFailedIndices(failed))
}

// extractFailedIndices returns a slice of failed worker indices.
func (m *MatrixExecutor) extractFailedIndices(failed []FailedWorkerInfo) []int {
	indices := make([]int, len(failed))
	for i, f := range failed {
		indices[i] = f.Index
	}
	return indices
}

// formatFailedWorkerErrors formats the errors from failed workers for the error message.
func (m *MatrixExecutor) formatFailedWorkerErrors(failed []FailedWorkerInfo) string {
	if len(failed) == 0 {
		return ""
	}

	// Limit to first 3 errors to avoid overly long messages
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

// readItemsSource reads and parses the items_source JSON file.
// The path format is "<step_id>/<artifact_path>" or an absolute path.
func (m *MatrixExecutor) readItemsSource(execution *PipelineExecution, strategy *MatrixStrategy) ([]interface{}, error) {
	itemsSourcePath := strategy.ItemsSource

	// Check if it's a reference to a previous step's artifact (format: "step_id/artifact_path")
	if !filepath.IsAbs(itemsSourcePath) && strings.Contains(itemsSourcePath, "/") {
		parts := strings.SplitN(itemsSourcePath, "/", 2)
		if len(parts) == 2 {
			stepID := parts[0]
			artifactPath := parts[1]

			// Look for the artifact in the previous step's workspace
			if wsPath, ok := execution.WorkspacePaths[stepID]; ok {
				itemsSourcePath = filepath.Join(wsPath, artifactPath)
			} else {
				// Try to find via artifact paths
				key := stepID + ":" + artifactPath
				if artPath, ok := execution.ArtifactPaths[key]; ok {
					itemsSourcePath = artPath
				}
			}
		}
	}

	// Read the JSON file
	data, err := os.ReadFile(itemsSourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read items_source file %q: %w", itemsSourcePath, err)
	}

	// Parse JSON
	var rawData interface{}
	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse items_source JSON: %w", err)
	}

	// Extract array using item_key if specified
	if strategy.ItemKey != "" {
		rawData, err = m.extractByKey(rawData, strategy.ItemKey)
		if err != nil {
			return nil, err
		}
	}

	// Ensure we have an array
	items, ok := rawData.([]interface{})
	if !ok {
		return nil, fmt.Errorf("items_source must be a JSON array, got %T", rawData)
	}

	return items, nil
}

// extractByKey extracts a nested value from data using a dot-separated key path.
func (m *MatrixExecutor) extractByKey(data interface{}, key string) (interface{}, error) {
	if key == "" {
		return data, nil
	}

	parts := strings.Split(key, ".")
	current := data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			val, ok := v[part]
			if !ok {
				return nil, fmt.Errorf("key %q not found in object", part)
			}
			current = val
		default:
			return nil, fmt.Errorf("cannot extract key %q from non-object type %T", part, current)
		}
	}

	return current, nil
}

// executeWorker runs a single matrix item in an isolated workspace.
func (m *MatrixExecutor) executeWorker(ctx context.Context, execution *PipelineExecution, step *Step, itemIndex int, item interface{}) MatrixResult {
	result := MatrixResult{
		ItemIndex: itemIndex,
		Item:      item,
	}

	pipelineID := execution.Status.ID

	// Create isolated workspace for this worker
	workspacePath, err := m.createWorkerWorkspace(execution, step, itemIndex)
	if err != nil {
		result.Error = fmt.Errorf("failed to create worker workspace: %w", err)
		return result
	}
	result.WorkspacePath = workspacePath

	m.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      "matrix_worker_start",
		Message:    fmt.Sprintf("Worker %d starting in %s", itemIndex, workspacePath),
	})

	// Create a modified step with the task template variable replaced
	workerStep := m.createWorkerStep(step, item)

	// Inject artifacts from dependencies into worker workspace
	if err := m.executor.injectArtifacts(execution, workerStep, workspacePath); err != nil {
		result.Error = fmt.Errorf("failed to inject artifacts: %w", err)
		return result
	}

	// Execute the step using the executor's run method
	workerExecution := &PipelineExecution{
		Pipeline:       execution.Pipeline,
		Manifest:       execution.Manifest,
		States:         make(map[string]string),
		Results:        make(map[string]map[string]interface{}),
		ArtifactPaths:  make(map[string]string),
		WorkspacePaths: map[string]string{step.ID: workspacePath},
		Input:          execution.Input,
		Status:         execution.Status,
		Context:        execution.Context, // Fix: Copy context to prevent nil pointer dereference
	}

	// Copy artifact paths from parent execution
	for k, v := range execution.ArtifactPaths {
		workerExecution.ArtifactPaths[k] = v
	}

	// Run the step execution
	err = m.executor.runStepExecution(ctx, workerExecution, workerStep)
	if err != nil {
		result.Error = err
		m.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "matrix_worker_failed",
			Message:    fmt.Sprintf("Worker %d failed: %v", itemIndex, err),
		})
		return result
	}

	// Collect output
	if stepResult, ok := workerExecution.Results[step.ID]; ok {
		result.Output = stepResult
	}

	// Scan for modified files
	result.ModifiedFiles, _ = m.scanModifiedFiles(workspacePath)

	m.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      "matrix_worker_complete",
		Message:    fmt.Sprintf("Worker %d completed", itemIndex),
	})

	return result
}

// createWorkerWorkspace creates an isolated workspace for a matrix worker.
func (m *MatrixExecutor) createWorkerWorkspace(execution *PipelineExecution, step *Step, itemIndex int) (string, error) {
	pipelineID := execution.Status.ID
	wsRoot := execution.Manifest.Runtime.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = ".wave/workspaces"
	}

	// Create worker-specific workspace under .wave/workspaces/<pipeline>/<step>/worker_<index>/
	wsPath := filepath.Join(wsRoot, pipelineID, step.ID, fmt.Sprintf("worker_%d", itemIndex))
	if err := os.MkdirAll(wsPath, 0755); err != nil {
		return "", err
	}

	// If step has mounts, copy them to worker workspace
	if m.executor.wsManager != nil && len(step.Workspace.Mount) > 0 {
		templateVars := map[string]string{
			"pipeline_id":  pipelineID,
			"step_id":      step.ID,
			"worker_index": fmt.Sprintf("%d", itemIndex),
		}
		return m.executor.wsManager.Create(workspace.WorkspaceConfig{
			Root:  wsRoot,
			Mount: toWorkspaceMounts(step.Workspace.Mount),
		}, templateVars)
	}

	return wsPath, nil
}

// createWorkerStep creates a modified step with the {{ task }} template variable replaced.
func (m *MatrixExecutor) createWorkerStep(step *Step, item interface{}) *Step {
	// Deep copy the step
	workerStep := *step

	// Serialize the item to JSON for template replacement
	itemJSON, err := json.Marshal(item)
	if err != nil {
		// Fall back to string representation
		itemJSON = []byte(fmt.Sprintf("%v", item))
	}
	itemStr := string(itemJSON)

	// Replace {{ task }} template variable in the exec source
	prompt := workerStep.Exec.Source
	for _, pattern := range []string{"{{ task }}", "{{task}}", "{{ task}}", "{{task }}"} {
		for idx := indexOf(prompt, pattern); idx != -1; idx = indexOf(prompt, pattern) {
			prompt = prompt[:idx] + itemStr + prompt[idx+len(pattern):]
		}
	}
	workerStep.Exec.Source = prompt

	return &workerStep
}

// scanModifiedFiles scans the workspace for files that were created or modified.
func (m *MatrixExecutor) scanModifiedFiles(workspacePath string) ([]string, error) {
	var files []string

	err := filepath.Walk(workspacePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Ignore errors
		}
		if info.IsDir() {
			return nil
		}

		// Get relative path from workspace
		relPath, err := filepath.Rel(workspacePath, path)
		if err != nil {
			return nil
		}

		files = append(files, relPath)
		return nil
	})

	return files, err
}

// detectFileConflicts checks if multiple workers modified the same file.
func (m *MatrixExecutor) detectFileConflicts(results []MatrixResult) error {
	fileWriters := make(map[string][]int) // file path -> list of worker indices

	for _, result := range results {
		if result.Error != nil {
			continue
		}
		for _, file := range result.ModifiedFiles {
			fileWriters[file] = append(fileWriters[file], result.ItemIndex)
		}
	}

	var conflicts []string
	for file, writers := range fileWriters {
		if len(writers) > 1 {
			conflicts = append(conflicts, fmt.Sprintf("%s (workers: %v)", file, writers))
		}
	}

	if len(conflicts) > 0 {
		return fmt.Errorf("file conflicts detected: %v", conflicts)
	}

	return nil
}

// aggregateResults combines all worker results into the execution state.
func (m *MatrixExecutor) aggregateResults(execution *PipelineExecution, step *Step, results []MatrixResult) {
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

// emit sends an event through the executor's emitter.
func (m *MatrixExecutor) emit(ev event.Event) {
	if m.executor.emitter != nil {
		m.executor.emitter.Emit(ev)
	}
}

// MatrixWorkerContext holds context information for a matrix worker.
type MatrixWorkerContext struct {
	Index         int
	Item          interface{}
	WorkspacePath string
}

// Ensure workspace package is used (for linter)
var _ = workspace.WorkspaceConfig{}
