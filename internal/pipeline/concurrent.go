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
	"golang.org/x/sync/errgroup"
)

// ConcurrentResult holds the result of a single concurrent agent execution.
type ConcurrentResult struct {
	AgentIndex    int
	WorkspacePath string
	Output        map[string]interface{}
	Error         error
}

// ConcurrentExecutor handles parallel agent execution for steps with concurrency > 1.
type ConcurrentExecutor struct {
	executor *DefaultPipelineExecutor
}

// NewConcurrentExecutor creates a new ConcurrentExecutor.
func NewConcurrentExecutor(executor *DefaultPipelineExecutor) *ConcurrentExecutor {
	return &ConcurrentExecutor{executor: executor}
}

// Execute runs N parallel agents for a step, each in an isolated workspace.
// It uses errgroup for fail-fast semantics: if any agent fails, the context
// is cancelled and remaining agents are abandoned.
func (c *ConcurrentExecutor) Execute(ctx context.Context, execution *PipelineExecution, step *Step) error {
	maxStepConcurrency := execution.Manifest.Runtime.GetMaxStepConcurrency()
	n := step.EffectiveConcurrency(maxStepConcurrency)
	pipelineID := execution.Status.ID

	c.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      "concurrent_start",
		Message:    fmt.Sprintf("Starting concurrent execution with %d agents", n),
	})

	// Create errgroup with concurrency limit
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(n)

	// Channel to collect results
	resultsChan := make(chan ConcurrentResult, n)

	// Spawn N agents
	for i := 0; i < n; i++ {
		agentIndex := i
		g.Go(func() error {
			result := c.executeAgent(gctx, execution, step, agentIndex)
			resultsChan <- result
			if result.Error != nil {
				return fmt.Errorf("agent %d failed: %w", agentIndex, result.Error)
			}
			return nil
		})
	}

	// Wait for all agents to complete (or first failure)
	err := g.Wait()
	close(resultsChan)

	// Collect all results
	results := make([]ConcurrentResult, 0, n)
	for result := range resultsChan {
		results = append(results, result)
	}

	// Aggregate results into execution
	c.aggregateResults(execution, step, results)

	if err != nil {
		// Count failures for reporting
		failCount := 0
		for _, r := range results {
			if r.Error != nil {
				failCount++
			}
		}
		c.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "concurrent_failed",
			Message:    fmt.Sprintf("Concurrent execution failed: %d/%d agents failed", failCount, n),
		})
		return fmt.Errorf("concurrent execution failed: %w", err)
	}

	c.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      "concurrent_complete",
		Message:    fmt.Sprintf("All %d agents completed successfully", n),
	})

	return nil
}

// executeAgent runs a single agent in an isolated workspace.
func (c *ConcurrentExecutor) executeAgent(ctx context.Context, execution *PipelineExecution, step *Step, agentIndex int) ConcurrentResult {
	result := ConcurrentResult{
		AgentIndex: agentIndex,
	}

	pipelineID := execution.Status.ID
	agentStepID := fmt.Sprintf("%s_agent_%d", step.ID, agentIndex)

	// Emit per-agent start event
	c.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      "concurrent_agent_start",
		Message:    fmt.Sprintf("Agent %d starting", agentIndex),
	})

	// Track per-agent state
	if c.executor.store != nil {
		c.executor.store.SaveStepState(pipelineID, agentStepID, "running", "")
	}

	// Create isolated workspace for this agent
	workspacePath, err := c.createAgentWorkspace(execution, step, agentIndex)
	if err != nil {
		result.Error = fmt.Errorf("failed to create agent workspace: %w", err)
		if c.executor.store != nil {
			c.executor.store.SaveStepState(pipelineID, agentStepID, "failed", result.Error.Error())
		}
		return result
	}
	result.WorkspacePath = workspacePath

	// Create a per-agent execution to avoid data races on shared maps
	agentExecution := &PipelineExecution{
		Pipeline:        execution.Pipeline,
		Manifest:        execution.Manifest,
		States:          make(map[string]string),
		Results:         make(map[string]map[string]interface{}),
		ArtifactPaths:   make(map[string]string),
		WorkspacePaths:  map[string]string{step.ID: workspacePath},
		WorktreePaths:   make(map[string]*WorktreeInfo),
		Input:           execution.Input,
		Status:          execution.Status,
		Context:         execution.Context,
		AttemptContexts: make(map[string]*AttemptContext),
	}

	// Copy artifact paths from parent execution
	execution.mu.Lock()
	for k, v := range execution.ArtifactPaths {
		agentExecution.ArtifactPaths[k] = v
	}
	execution.mu.Unlock()

	// Run the step execution in the agent workspace
	err = c.executor.runStepExecution(ctx, agentExecution, step)
	if err != nil {
		result.Error = err
		c.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "concurrent_agent_failed",
			Message:    fmt.Sprintf("Agent %d failed: %v", agentIndex, err),
		})
		if c.executor.store != nil {
			c.executor.store.SaveStepState(pipelineID, agentStepID, "failed", err.Error())
		}
		return result
	}

	// Collect output
	if stepResult, ok := agentExecution.Results[step.ID]; ok {
		result.Output = stepResult
	}

	c.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      "concurrent_agent_complete",
		Message:    fmt.Sprintf("Agent %d completed", agentIndex),
	})
	if c.executor.store != nil {
		c.executor.store.SaveStepState(pipelineID, agentStepID, "completed", "")
	}

	return result
}

// createAgentWorkspace creates an isolated workspace for a concurrent agent.
func (c *ConcurrentExecutor) createAgentWorkspace(execution *PipelineExecution, step *Step, agentIndex int) (string, error) {
	pipelineID := execution.Status.ID
	wsRoot := execution.Manifest.Runtime.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = ".wave/workspaces"
	}

	// Create agent-specific workspace: .wave/workspaces/<pipeline>/<step>_agent_<N>/
	wsPath := filepath.Join(wsRoot, pipelineID, fmt.Sprintf("%s_agent_%d", step.ID, agentIndex))
	if err := os.MkdirAll(wsPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create agent workspace directory: %w", err)
	}

	return wsPath, nil
}

// aggregateResults combines all agent results into the parent execution state.
func (c *ConcurrentExecutor) aggregateResults(execution *PipelineExecution, step *Step, results []ConcurrentResult) {
	agentResults := make([]map[string]interface{}, 0, len(results))
	agentWorkspaces := make([]string, 0, len(results))

	successCount := 0
	failCount := 0
	for _, result := range results {
		if result.Output != nil {
			agentResults = append(agentResults, result.Output)
		}
		agentWorkspaces = append(agentWorkspaces, result.WorkspacePath)
		if result.Error == nil {
			successCount++
		} else {
			failCount++
		}
	}

	aggregated := map[string]interface{}{
		"agent_results":    agentResults,
		"agent_workspaces": agentWorkspaces,
		"total_agents":     len(results),
		"success_count":    successCount,
		"fail_count":       failCount,
	}

	execution.mu.Lock()
	execution.Results[step.ID] = aggregated
	execution.mu.Unlock()

	// Merge output artifacts from all successful agents
	c.mergeOutputArtifacts(execution, step, results)
}

// mergeOutputArtifacts collects per-agent output artifacts and writes merged versions.
func (c *ConcurrentExecutor) mergeOutputArtifacts(execution *PipelineExecution, step *Step, results []ConcurrentResult) {
	if len(step.OutputArtifacts) == 0 {
		return
	}

	// Use first agent's workspace as the primary workspace for merged artifacts
	var primaryWorkspace string
	for _, r := range results {
		if r.Error == nil && r.WorkspacePath != "" {
			primaryWorkspace = r.WorkspacePath
			break
		}
	}
	if primaryWorkspace == "" {
		return
	}

	execution.mu.Lock()
	execution.WorkspacePaths[step.ID] = primaryWorkspace
	execution.mu.Unlock()

	for _, art := range step.OutputArtifacts {
		c.mergeArtifact(execution, step, art, results, primaryWorkspace)
	}
}

// mergeArtifact merges a single artifact from all agents into the primary workspace.
func (c *ConcurrentExecutor) mergeArtifact(execution *PipelineExecution, step *Step, art ArtifactDef, results []ConcurrentResult, primaryWorkspace string) {
	var contents [][]byte

	for _, r := range results {
		if r.Error != nil || r.WorkspacePath == "" {
			continue
		}

		// Try to read the artifact from the agent's workspace
		resolvedPath := art.Path
		if resolvedPath == "" {
			resolvedPath = filepath.Join(".wave", "output", art.Name)
		}
		artPath := filepath.Join(r.WorkspacePath, resolvedPath)
		data, err := os.ReadFile(artPath)
		if err != nil {
			continue
		}
		contents = append(contents, data)
	}

	if len(contents) == 0 {
		return
	}

	// Merge based on artifact type
	var merged []byte
	if art.Type == "json" {
		merged = mergeJSONArtifacts(contents)
	} else {
		merged = mergeTextArtifacts(contents)
	}

	// Write merged artifact to primary workspace
	resolvedPath := art.Path
	if resolvedPath == "" {
		resolvedPath = filepath.Join(".wave", "output", art.Name)
	}
	destPath := filepath.Join(primaryWorkspace, resolvedPath)
	os.MkdirAll(filepath.Dir(destPath), 0755)
	os.WriteFile(destPath, merged, 0644)

	// Register the merged artifact path
	key := step.ID + ":" + art.Name
	execution.mu.Lock()
	execution.ArtifactPaths[key] = destPath
	execution.mu.Unlock()
}

// mergeJSONArtifacts wraps all JSON contents in an array.
func mergeJSONArtifacts(contents [][]byte) []byte {
	var items []json.RawMessage
	for _, data := range contents {
		// Validate it's valid JSON before adding
		var raw json.RawMessage
		if json.Unmarshal(data, &raw) == nil {
			items = append(items, raw)
		} else {
			// Wrap non-JSON content as a string
			quoted, _ := json.Marshal(string(data))
			items = append(items, quoted)
		}
	}
	result, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		// Fallback: concatenate
		return mergeTextArtifacts(contents)
	}
	return result
}

// mergeTextArtifacts concatenates text artifacts with agent index headers.
func mergeTextArtifacts(contents [][]byte) []byte {
	var buf strings.Builder
	for i, data := range contents {
		if i > 0 {
			buf.WriteString("\n\n")
		}
		fmt.Fprintf(&buf, "--- Agent %d ---\n", i)
		buf.Write(data)
	}
	return []byte(buf.String())
}

// emit sends an event through the executor's emitter.
func (c *ConcurrentExecutor) emit(ev event.Event) {
	if c.executor.emitter != nil {
		c.executor.emitter.Emit(ev)
	}
}

