package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/recinq/wave/internal/event"
	"golang.org/x/sync/errgroup"
)

// ConcurrencyResult holds the result of a single concurrent agent execution.
type ConcurrencyResult struct {
	AgentIndex    int
	WorkspacePath string
	Output        map[string]interface{}
	Error         error
}

// ConcurrencyExecutor handles spawning N identical agents for a step.
type ConcurrencyExecutor struct {
	executor *DefaultPipelineExecutor
}

// NewConcurrencyExecutor creates a new ConcurrencyExecutor.
func NewConcurrencyExecutor(executor *DefaultPipelineExecutor) *ConcurrencyExecutor {
	return &ConcurrencyExecutor{executor: executor}
}

// Execute runs N concurrent identical agents for a step, using errgroup for
// fail-fast semantics and per-agent workspace isolation.
func (c *ConcurrencyExecutor) Execute(ctx context.Context, execution *PipelineExecution, step *Step) error {
	if step.Concurrency <= 1 {
		return fmt.Errorf("step %q concurrency is %d, expected > 1", step.ID, step.Concurrency)
	}

	pipelineID := execution.Status.ID
	concurrency := step.Concurrency

	// Cap at MaxConcurrency from runtime config
	maxConcurrency := execution.Manifest.Runtime.GetMaxConcurrency()
	if concurrency > maxConcurrency {
		concurrency = maxConcurrency
	}

	c.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      "concurrency_start",
		Message:    fmt.Sprintf("Starting %d concurrent agents (max: %d)", concurrency, maxConcurrency),
	})

	// Create errgroup with concurrency limit for fail-fast behavior
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)

	// Channel to collect results
	resultsChan := make(chan ConcurrencyResult, concurrency)

	// Spawn agents
	for i := 0; i < concurrency; i++ {
		agentIndex := i
		g.Go(func() error {
			result := c.executeAgent(gctx, execution, step, agentIndex)
			resultsChan <- result
			if result.Error != nil {
				return result.Error
			}
			return nil
		})
	}

	// Wait for all agents to complete (or first failure)
	err := g.Wait()
	close(resultsChan)

	// Collect all results
	results := make([]ConcurrencyResult, 0, concurrency)
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
			State:      "concurrency_failed",
			Message:    fmt.Sprintf("Concurrent execution failed: %d/%d agents failed", failCount, concurrency),
		})
		return fmt.Errorf("concurrent execution failed: %w", err)
	}

	c.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      "concurrency_complete",
		Message:    fmt.Sprintf("All %d agents completed successfully", concurrency),
	})

	return nil
}

// executeAgent runs a single agent in an isolated workspace.
func (c *ConcurrencyExecutor) executeAgent(ctx context.Context, execution *PipelineExecution, step *Step, agentIndex int) ConcurrencyResult {
	result := ConcurrencyResult{
		AgentIndex: agentIndex,
	}

	pipelineID := execution.Status.ID

	// Create isolated workspace for this agent
	workspacePath, err := c.createAgentWorkspace(execution, step, agentIndex)
	if err != nil {
		result.Error = fmt.Errorf("failed to create agent workspace: %w", err)
		return result
	}
	result.WorkspacePath = workspacePath

	c.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      "concurrency_agent_start",
		Message:    fmt.Sprintf("Agent %d starting in %s", agentIndex, workspacePath),
	})

	// Create a worker execution context with isolated state
	agentExecution := &PipelineExecution{
		Pipeline:       execution.Pipeline,
		Manifest:       execution.Manifest,
		States:         make(map[string]string),
		Results:        make(map[string]map[string]interface{}),
		ArtifactPaths:  make(map[string]string),
		WorkspacePaths: map[string]string{step.ID: workspacePath},
		WorktreePaths:  make(map[string]*WorktreeInfo),
		Input:          execution.Input,
		Status:         execution.Status,
		Context:        execution.Context,
	}

	// Copy artifact paths from parent execution
	execution.mu.Lock()
	for k, v := range execution.ArtifactPaths {
		agentExecution.ArtifactPaths[k] = v
	}
	execution.mu.Unlock()

	// Run the step execution
	err = c.executor.runStepExecution(ctx, agentExecution, step)
	if err != nil {
		result.Error = err
		c.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "concurrency_agent_failed",
			Message:    fmt.Sprintf("Agent %d failed: %v", agentIndex, err),
		})
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
		State:      "concurrency_agent_complete",
		Message:    fmt.Sprintf("Agent %d completed", agentIndex),
	})

	return result
}

// createAgentWorkspace creates an isolated workspace for a concurrent agent.
func (c *ConcurrencyExecutor) createAgentWorkspace(execution *PipelineExecution, step *Step, agentIndex int) (string, error) {
	pipelineID := execution.Status.ID
	wsRoot := execution.Manifest.Runtime.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = ".wave/workspaces"
	}

	// Create agent-specific workspace under .wave/workspaces/<pipeline>/<step>/agent_<index>/
	wsPath := filepath.Join(wsRoot, pipelineID, step.ID, fmt.Sprintf("agent_%d", agentIndex))
	if err := os.MkdirAll(wsPath, 0755); err != nil {
		return "", err
	}

	return wsPath, nil
}

// aggregateResults combines all agent results into the execution state.
func (c *ConcurrencyExecutor) aggregateResults(execution *PipelineExecution, step *Step, results []ConcurrencyResult) {
	aggregated := make(map[string]interface{})
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

	aggregated["agent_results"] = agentResults
	aggregated["agent_workspaces"] = agentWorkspaces
	aggregated["total_agents"] = len(results)
	aggregated["success_count"] = successCount
	aggregated["fail_count"] = failCount

	execution.mu.Lock()
	execution.Results[step.ID] = aggregated
	// Register first agent's workspace as the step workspace
	if len(agentWorkspaces) > 0 {
		execution.WorkspacePaths[step.ID] = filepath.Dir(agentWorkspaces[0])
	}
	execution.mu.Unlock()
}

// emit sends an event through the executor's emitter.
func (c *ConcurrencyExecutor) emit(ev event.Event) {
	if c.executor.emitter != nil {
		c.executor.emitter.Emit(ev)
	}
}
