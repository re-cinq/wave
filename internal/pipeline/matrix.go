package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
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
	Skipped    bool
	SkipReason string
	ItemID       string
	OutputBranch string
}

// TierContext tracks accumulated branch state during stacked tier execution.
// Maps item IDs to their output branch names for downstream tier resolution.
type TierContext struct {
	outputBranches map[string]string // itemID -> output branch name
}

// NewTierContext creates a new TierContext.
func NewTierContext() *TierContext {
	return &TierContext{
		outputBranches: make(map[string]string),
	}
}

// SetBranch records the output branch for a completed item.
func (tc *TierContext) SetBranch(itemID, branch string) {
	tc.outputBranches[itemID] = branch
}

// GetBranch returns the output branch for an item, if available.
func (tc *TierContext) GetBranch(itemID string) (string, bool) {
	branch, ok := tc.outputBranches[itemID]
	return branch, ok
}

// ResolveBranch returns the parent branches for a given list of dependency item IDs.
func (tc *TierContext) ResolveBranch(deps []string) ([]string, error) {
	var branches []string
	for _, dep := range deps {
		branch, ok := tc.outputBranches[dep]
		if !ok {
			continue // Parent has no output branch — will use default base
		}
		if branch != "" {
			branches = append(branches, branch)
		}
	}
	return branches, nil
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
		execution.mu.Lock()
		execution.Results[step.ID] = map[string]interface{}{
			"worker_results":    []map[string]interface{}{},
			"worker_workspaces": []string{},
			"total_workers":     0,
			"success_count":     0,
			"fail_count":        0,
		}
		execution.mu.Unlock()
		return nil
	}

	m.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      "matrix_items_loaded",
		Message:    fmt.Sprintf("Loaded %d items for parallel execution", len(items)),
	})

	// Resolve worker function: child pipeline or direct step execution
	worker := matrixWorkerFunc(m.executeWorker)
	if strategy.ChildPipeline != "" {
		childPipeline, err := m.loadChildPipeline(strategy.ChildPipeline)
		if err != nil {
			return err
		}
		m.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "matrix_child_pipeline_loaded",
			Message:    fmt.Sprintf("Loaded child pipeline %q with %d steps", childPipeline.PipelineName(), len(childPipeline.Steps)),
		})
		worker = m.childPipelineWorker(childPipeline)
	}

	// Branch to tiered execution when dependency_key is configured
	if strategy.DependencyKey != "" {
		return m.tieredExecution(ctx, execution, step, items, worker)
	}

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
			result := worker(gctx, execution, step, itemIndex, itemValue)
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
			execution.mu.Lock()
			wsPath, wsOk := execution.WorkspacePaths[stepID]
			artKey := stepID + ":" + artifactPath
			artPath, artOk := execution.ArtifactPaths[artKey]
			execution.mu.Unlock()

			if wsOk {
				itemsSourcePath = filepath.Join(wsPath, artifactPath)
			} else if artOk {
				itemsSourcePath = artPath
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
		WorktreePaths:  make(map[string]*WorktreeInfo),
		Input:          execution.Input,
		Status:         execution.Status,
		Context:        execution.Context, // Fix: Copy context to prevent nil pointer dereference
	}

	// Copy artifact paths from parent execution
	execution.mu.Lock()
	for k, v := range execution.ArtifactPaths {
		workerExecution.ArtifactPaths[k] = v
	}
	execution.mu.Unlock()

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

	execution.mu.Lock()
	execution.Results[step.ID] = aggregated
	// Register the first worker's workspace as the step workspace
	if len(workerPaths) > 0 {
		execution.WorkspacePaths[step.ID] = filepath.Dir(workerPaths[0])
	}
	execution.mu.Unlock()
}

// emit sends an event through the executor's emitter.
func (m *MatrixExecutor) emit(ev event.Event) {
	if m.executor.emitter != nil {
		m.executor.emitter.Emit(ev)
	}
}

// matrixWorkerFunc is the signature for a function that executes a single matrix item.
type matrixWorkerFunc func(ctx context.Context, execution *PipelineExecution, step *Step, itemIndex int, item interface{}) MatrixResult

// tieredExecution runs matrix items in dependency-ordered tiers.
// Items within a tier run in parallel; tiers execute sequentially.
// If an item fails, items that depend on it are skipped.
func (m *MatrixExecutor) tieredExecution(ctx context.Context, execution *PipelineExecution, step *Step, items []interface{}, worker matrixWorkerFunc) error {
	strategy := step.Strategy
	pipelineID := execution.Status.ID

	// Build ID → item index mapping
	idToIndex := make(map[string]int, len(items))
	indexToID := make(map[int]string, len(items))
	for i, item := range items {
		id, err := m.extractItemID(item, strategy.ItemIDKey)
		if err != nil {
			return fmt.Errorf("failed to extract item_id_key %q from item %d: %w", strategy.ItemIDKey, i, err)
		}
		if _, exists := idToIndex[id]; exists {
			return fmt.Errorf("duplicate item ID %q at index %d", id, i)
		}
		idToIndex[id] = i
		indexToID[i] = id
	}

	// Build dependency graph
	deps := make(map[string][]string, len(items))
	for i, item := range items {
		id := indexToID[i]
		itemDeps, err := m.extractDependencies(item, strategy.DependencyKey)
		if err != nil {
			return fmt.Errorf("failed to extract dependency_key %q from item %d: %w", strategy.DependencyKey, i, err)
		}
		for _, depID := range itemDeps {
			if _, ok := idToIndex[depID]; !ok {
				return fmt.Errorf("item %q depends on %q which does not exist", id, depID)
			}
		}
		deps[id] = itemDeps
	}

	// Compute tiers using Kahn's algorithm
	tiers, err := m.computeTiers(idToIndex, deps)
	if err != nil {
		return err
	}

	m.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      "matrix_tiers_computed",
		Message:    fmt.Sprintf("Computed %d execution tiers for %d items", len(tiers), len(items)),
	})

	maxConcurrency := strategy.MaxConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = len(items)
	}

	allResults := make([]MatrixResult, 0, len(items))
	failed := make(map[string]bool)     // IDs of items that failed
	succeeded := make(map[string]bool)   // IDs of items that succeeded

	// Initialize stacking context when stacked mode is active with dependency tiers
	var tierCtx *TierContext
	var cleanupBranches []string
	stacked := strategy.Stacked && strategy.DependencyKey != ""
	if stacked {
		tierCtx = NewTierContext()
	}

	// Schedule cleanup of integration branches on exit
	if stacked {
		defer func() {
			if len(cleanupBranches) > 0 {
				// Find repo root from execution context
				repoRoot := "."
				execution.mu.Lock()
				for _, info := range execution.WorktreePaths {
					if info.RepoRoot != "" {
						repoRoot = info.RepoRoot
						break
					}
				}
				execution.mu.Unlock()
				m.cleanupIntegrationBranches(repoRoot, cleanupBranches)
			}
		}()
	}

	for tierIdx, tier := range tiers {
		m.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "matrix_tier_start",
			Message:    fmt.Sprintf("Starting tier %d with %d items", tierIdx, len(tier)),
		})

		// Partition tier into runnable and skipped items
		var runnable []string
		for _, id := range tier {
			shouldSkip, reason := m.shouldSkipItem(id, deps[id], failed)
			if shouldSkip {
				idx := idToIndex[id]
				allResults = append(allResults, MatrixResult{
					ItemIndex:  idx,
					Item:       items[idx],
					ItemID:     id,
					Skipped:    true,
					SkipReason: reason,
				})
				failed[id] = true // propagate failure downstream
				m.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      "matrix_item_skipped",
					Message:    fmt.Sprintf("Skipping item %q: %s", id, reason),
				})
				continue
			}
			runnable = append(runnable, id)
		}

		if len(runnable) == 0 {
			continue
		}

		// Resolve stacked base branches for this tier's items
		stackedBases := make(map[string]string) // itemID -> resolved base branch
		if stacked && tierIdx > 0 {
			for _, id := range runnable {
				itemDeps := deps[id]
				parentBranches, _ := tierCtx.ResolveBranch(itemDeps)

				switch {
				case len(parentBranches) == 1:
					stackedBases[id] = parentBranches[0]
					m.emit(event.Event{
						Timestamp:  time.Now(),
						PipelineID: pipelineID,
						StepID:     step.ID,
						State:      "matrix_stacked_branch_resolved",
						Message:    fmt.Sprintf("Item %q will branch from parent branch %s", id, parentBranches[0]),
					})
				case len(parentBranches) > 1:
					// Multi-parent: create integration branch
					repoRoot := "."
					execution.mu.Lock()
					for _, info := range execution.WorktreePaths {
						if info.RepoRoot != "" {
							repoRoot = info.RepoRoot
							break
						}
					}
					execution.mu.Unlock()
					integrationBranch, err := m.createIntegrationBranch(repoRoot, pipelineID, id, parentBranches)
					if err != nil {
						// Mark this item as failed due to merge conflict
						idx := idToIndex[id]
						allResults = append(allResults, MatrixResult{
							ItemIndex: idx,
							Item:      items[idx],
							ItemID:    id,
							Error:     fmt.Errorf("failed to create integration branch for item %q: %w", id, err),
						})
						failed[id] = true
						continue
					}
					stackedBases[id] = integrationBranch
					cleanupBranches = append(cleanupBranches, integrationBranch)
					m.emit(event.Event{
						Timestamp:  time.Now(),
						PipelineID: pipelineID,
						StepID:     step.ID,
						State:      "matrix_integration_branch_created",
						Message:    fmt.Sprintf("Created integration branch %s for item %q from %d parents", integrationBranch, id, len(parentBranches)),
					})
				case len(parentBranches) == 0 && len(itemDeps) > 0:
					// Parent had no output branch — use default base (graceful fallback)
					m.emit(event.Event{
						Timestamp:  time.Now(),
						PipelineID: pipelineID,
						StepID:     step.ID,
						State:      "matrix_stacked_branch_resolved",
						Message:    fmt.Sprintf("Item %q: parent(s) have no output branch, using default base", id),
					})
				}
			}

			// Remove items that failed integration branch creation from runnable
			var stillRunnable []string
			for _, id := range runnable {
				if !failed[id] {
					stillRunnable = append(stillRunnable, id)
				}
			}
			runnable = stillRunnable

			if len(runnable) == 0 {
				continue
			}
		}

		// Execute runnable items in parallel with concurrency limit
		g, gctx := errgroup.WithContext(ctx)
		concurrency := maxConcurrency
		if concurrency > len(runnable) {
			concurrency = len(runnable)
		}
		g.SetLimit(concurrency)

		var mu sync.Mutex
		tierResults := make([]MatrixResult, 0, len(runnable))

		for _, id := range runnable {
			itemID := id
			itemIndex := idToIndex[itemID]
			itemValue := items[itemIndex]

			// Wrap worker to inject stacked base branch if available
			effectiveWorker := worker
			if baseBranch, ok := stackedBases[itemID]; ok && baseBranch != "" {
				effectiveWorker = m.wrapWorkerWithBaseBranch(worker, baseBranch)
			}

			g.Go(func() error {
				result := effectiveWorker(gctx, execution, step, itemIndex, itemValue)
				result.ItemID = itemID

				mu.Lock()
				tierResults = append(tierResults, result)
				mu.Unlock()

				// Don't return error — we want all items in this tier to execute
				return nil
			})
		}

		_ = g.Wait()

		// Record results and track failures; update tier context with output branches
		for _, result := range tierResults {
			allResults = append(allResults, result)
			if result.Error != nil {
				failed[result.ItemID] = true
			} else {
				succeeded[result.ItemID] = true
				if stacked && result.OutputBranch != "" {
					tierCtx.SetBranch(result.ItemID, result.OutputBranch)
				}
			}
		}
	}

	// Aggregate all results
	m.aggregateResults(execution, step, allResults)

	// Count outcomes
	skipCount := 0
	failCount := 0
	successCount := 0
	for _, r := range allResults {
		switch {
		case r.Skipped:
			skipCount++
		case r.Error != nil:
			failCount++
		default:
			successCount++
		}
	}

	// Override counts in aggregated results — aggregateResults does not distinguish skipped items
	execution.mu.Lock()
	if results, ok := execution.Results[step.ID]; ok {
		results["skip_count"] = skipCount
		results["success_count"] = successCount
		results["fail_count"] = failCount
	}
	execution.mu.Unlock()

	if failCount > 0 {
		failedWorkers := m.collectFailedWorkers(allResults)
		failureMsg := m.buildPartialFailureMessage(failedWorkers, successCount, len(items))
		m.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "matrix_failed",
			Message:    fmt.Sprintf("%s (skipped: %d)", failureMsg, skipCount),
		})
		return fmt.Errorf("matrix execution partially failed: %d/%d workers failed, %d skipped. %s",
			failCount, len(items), skipCount, m.formatFailedWorkerErrors(failedWorkers))
	}

	m.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      "matrix_complete",
		Message:    fmt.Sprintf("Successfully processed %d items (%d skipped)", successCount, skipCount),
	})

	return nil
}

// extractItemID extracts a string ID from an item using a dot-path key.
func (m *MatrixExecutor) extractItemID(item interface{}, key string) (string, error) {
	val, err := m.extractByKey(item, key)
	if err != nil {
		return "", err
	}
	switch v := val.(type) {
	case string:
		return v, nil
	case float64:
		// JSON numbers are float64 — format as integer if possible
		if v == float64(int(v)) {
			return fmt.Sprintf("%d", int(v)), nil
		}
		return fmt.Sprintf("%g", v), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

// extractDependencies extracts a string slice of dependency IDs from an item.
// Returns an empty slice if the key points to null or an empty array.
func (m *MatrixExecutor) extractDependencies(item interface{}, key string) ([]string, error) {
	val, err := m.extractByKey(item, key)
	if err != nil {
		// Key not found means no dependencies
		return nil, nil
	}
	if val == nil {
		return nil, nil
	}
	arr, ok := val.([]interface{})
	if !ok {
		return nil, fmt.Errorf("dependency_key value must be an array, got %T", val)
	}
	result := make([]string, 0, len(arr))
	for _, v := range arr {
		switch dv := v.(type) {
		case string:
			result = append(result, dv)
		case float64:
			if dv == float64(int(dv)) {
				result = append(result, fmt.Sprintf("%d", int(dv)))
			} else {
				result = append(result, fmt.Sprintf("%g", dv))
			}
		default:
			result = append(result, fmt.Sprintf("%v", v))
		}
	}
	return result, nil
}

// computeTiers uses Kahn's algorithm (BFS topological sort) to group items
// into execution tiers. Returns an error if the graph has cycles.
func (m *MatrixExecutor) computeTiers(idToIndex map[string]int, deps map[string][]string) ([][]string, error) {
	// Build in-degree map
	inDegree := make(map[string]int, len(idToIndex))
	for id := range idToIndex {
		inDegree[id] = 0
	}
	for id, depList := range deps {
		inDegree[id] = len(depList)
	}

	// Build reverse adjacency: dep → items that depend on it
	reverse := make(map[string][]string, len(idToIndex))
	for id, depList := range deps {
		for _, dep := range depList {
			reverse[dep] = append(reverse[dep], id)
		}
	}

	var tiers [][]string
	remaining := len(idToIndex)

	for remaining > 0 {
		// Collect items with in-degree 0
		var tier []string
		for id, deg := range inDegree {
			if deg == 0 {
				tier = append(tier, id)
			}
		}

		if len(tier) == 0 {
			return nil, fmt.Errorf("dependency cycle detected among remaining %d items", remaining)
		}

		// Sort tier for deterministic ordering
		sort.Strings(tier)

		// Remove items from graph
		for _, id := range tier {
			delete(inDegree, id)
			for _, dependent := range reverse[id] {
				inDegree[dependent]--
			}
		}

		tiers = append(tiers, tier)
		remaining -= len(tier)
	}

	return tiers, nil
}

// shouldSkipItem checks if an item should be skipped because a dependency failed.
func (m *MatrixExecutor) shouldSkipItem(id string, itemDeps []string, failed map[string]bool) (bool, string) {
	for _, dep := range itemDeps {
		if failed[dep] {
			return true, fmt.Sprintf("dependency %q failed", dep)
		}
	}
	return false, ""
}

// loadChildPipeline loads a pipeline by name or path for use in child pipeline execution.
func (m *MatrixExecutor) loadChildPipeline(name string) (*Pipeline, error) {
	path := m.resolveChildPipelinePath(name)
	loader := &YAMLPipelineLoader{}
	pipeline, err := loader.Load(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load child pipeline %q from %s: %w", name, path, err)
	}
	return pipeline, nil
}

// resolveChildPipelinePath converts a pipeline name to its filesystem path.
// If the name already ends in .yaml or .yml, it's treated as a direct path.
// Otherwise it resolves to .wave/pipelines/<name>.yaml.
func (m *MatrixExecutor) resolveChildPipelinePath(name string) string {
	if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
		return name
	}
	return filepath.Join(".wave", "pipelines", name+".yaml")
}

// childPipelineWorker returns a matrixWorkerFunc that executes a full child
// pipeline for each matrix item, using a fresh executor per item.
func (m *MatrixExecutor) childPipelineWorker(childPipeline *Pipeline) matrixWorkerFunc {
	return func(ctx context.Context, execution *PipelineExecution, step *Step, itemIndex int, item interface{}) MatrixResult {
		result := MatrixResult{
			ItemIndex: itemIndex,
			Item:      item,
		}

		pipelineID := execution.Status.ID

		// Render input from template
		input, err := m.renderInputTemplate(step.Strategy.InputTemplate, item)
		if err != nil {
			result.Error = fmt.Errorf("failed to render input template: %w", err)
			return result
		}

		m.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "matrix_child_pipeline_start",
			Message:    fmt.Sprintf("Starting child pipeline %q for item %d", childPipeline.PipelineName(), itemIndex),
		})

		// Create child executor with independent state
		childExecutor := m.executor.NewChildExecutor()

		// Apply stacked base branch override from context (set by wrapWorkerWithBaseBranch)
		if baseBranch := StackedBaseBranchFromContext(ctx); baseBranch != "" {
			childExecutor.stackedBaseBranch = baseBranch
		}

		// Execute the child pipeline
		if err := childExecutor.Execute(ctx, childPipeline, execution.Manifest, input); err != nil {
			result.Error = err
			m.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      "matrix_child_pipeline_failed",
				Message:    fmt.Sprintf("Child pipeline for item %d failed: %v", itemIndex, err),
			})
			return result
		}

		// Capture output branch from child executor's worktree paths (FR-007)
		if exec := childExecutor.LastExecution(); exec != nil {
			exec.mu.Lock()
			for branch := range exec.WorktreePaths {
				result.OutputBranch = branch
				break // Use the first worktree branch found
			}
			exec.mu.Unlock()
		}

		m.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "matrix_child_pipeline_complete",
			Message:    fmt.Sprintf("Child pipeline for item %d completed", itemIndex),
		})

		return result
	}
}

// stackedBaseBranchKey is the context key for stacked matrix base branch override.
type stackedBaseBranchKey struct{}

// wrapWorkerWithBaseBranch wraps a matrixWorkerFunc to inject the stacked base branch
// into the context. The child executor reads this via context.Value during worktree setup.
func (m *MatrixExecutor) wrapWorkerWithBaseBranch(worker matrixWorkerFunc, baseBranch string) matrixWorkerFunc {
	return func(ctx context.Context, execution *PipelineExecution, step *Step, itemIndex int, item interface{}) MatrixResult {
		ctx = context.WithValue(ctx, stackedBaseBranchKey{}, baseBranch)
		return worker(ctx, execution, step, itemIndex, item)
	}
}

// StackedBaseBranchFromContext extracts the stacked base branch override from context.
func StackedBaseBranchFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(stackedBaseBranchKey{}).(string); ok {
		return v
	}
	return ""
}

// createIntegrationBranch creates a temporary branch that merges multiple parent branches.
// Returns the integration branch name and an error if any merge conflicts occur.
func (m *MatrixExecutor) createIntegrationBranch(repoRoot, pipelineID, itemID string, parentBranches []string) (string, error) {
	if len(parentBranches) == 0 {
		return "", fmt.Errorf("no parent branches to merge")
	}

	branchName := fmt.Sprintf("integration/%s/%s", pipelineID, itemID)

	// Create branch from first parent
	cmd := exec.Command("git", "checkout", "-b", branchName, parentBranches[0])
	cmd.Dir = repoRoot
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to create integration branch from %s: %s: %w", parentBranches[0], string(output), err)
	}

	// Merge each additional parent
	for i := 1; i < len(parentBranches); i++ {
		cmd := exec.Command("git", "merge", parentBranches[i], "--no-edit")
		cmd.Dir = repoRoot
		if output, err := cmd.CombinedOutput(); err != nil {
			// Abort the failed merge
			abortCmd := exec.Command("git", "merge", "--abort")
			abortCmd.Dir = repoRoot
			_ = abortCmd.Run()

			// Clean up the integration branch
			checkoutCmd := exec.Command("git", "checkout", parentBranches[0])
			checkoutCmd.Dir = repoRoot
			_ = checkoutCmd.Run()

			deleteCmd := exec.Command("git", "branch", "-D", branchName)
			deleteCmd.Dir = repoRoot
			_ = deleteCmd.Run()

			return "", fmt.Errorf("merge conflict between %s and %s: %s: %w", parentBranches[i-1], parentBranches[i], string(output), err)
		}
	}

	// Switch back to original branch so we don't leave the repo on the integration branch
	cmd = exec.Command("git", "checkout", "-")
	cmd.Dir = repoRoot
	_ = cmd.Run()

	return branchName, nil
}

// cleanupIntegrationBranches removes temporary integration branches.
func (m *MatrixExecutor) cleanupIntegrationBranches(repoRoot string, branches []string) {
	for _, branch := range branches {
		cmd := exec.Command("git", "branch", "-D", branch)
		cmd.Dir = repoRoot
		if output, err := cmd.CombinedOutput(); err != nil {
			m.emit(event.Event{
				Timestamp: time.Now(),
				State:     "matrix_cleanup_warning",
				Message:   fmt.Sprintf("Failed to cleanup integration branch %s: %s: %v", branch, string(output), err),
			})
		}
	}
}

// renderInputTemplate renders a Go text/template with the matrix item as data context.
// If the template string is empty, the item is serialized to JSON as the default input.
func (m *MatrixExecutor) renderInputTemplate(tmplStr string, item interface{}) (string, error) {
	if tmplStr == "" {
		data, err := json.Marshal(item)
		if err != nil {
			return "", fmt.Errorf("failed to serialize item: %w", err)
		}
		return string(data), nil
	}

	tmpl, err := template.New("input").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse input template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, item); err != nil {
		return "", fmt.Errorf("failed to execute input template: %w", err)
	}

	return buf.String(), nil
}
