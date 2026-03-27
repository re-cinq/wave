package pipeline

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// GraphWalker implements edge-following execution for graph-mode pipelines.
// It replaces the topological sort loop when a pipeline contains edges or
// conditional steps.
type GraphWalker struct {
	pipeline *Pipeline
	stepMap  map[string]*Step

	// Visit tracking
	visitCounts map[string]int // stepID -> visit count
	totalVisits int            // total visits across all steps
	mu          sync.Mutex

	// Circuit breaker: track last 3 errors per step
	errorHistory map[string][]string // stepID -> last N error messages
}

const circuitBreakerWindow = 3

// NewGraphWalker creates a new GraphWalker for the given pipeline.
func NewGraphWalker(p *Pipeline) *GraphWalker {
	stepMap := make(map[string]*Step, len(p.Steps))
	for i := range p.Steps {
		stepMap[p.Steps[i].ID] = &p.Steps[i]
	}
	return &GraphWalker{
		pipeline:     p,
		stepMap:      stepMap,
		visitCounts:  make(map[string]int),
		errorHistory: make(map[string][]string),
	}
}

// StepResult holds the result of executing a single step.
type StepResult struct {
	StepID  string
	Outcome string            // "success" or "failure"
	Error   error             // non-nil if step failed
	Context map[string]string // context values set by the step
	Stdout  string            // captured stdout (for command steps)
}

// StepExecutor is a callback function that the GraphWalker calls to execute each step.
// The walker handles routing; the executor handles the actual step execution.
type StepExecutor func(ctx context.Context, step *Step) (*StepResult, error)

// Walk executes the pipeline by following edges from the first step.
// The stepExecutor callback is called for each step that needs to run.
// initialVisitCounts can be provided to resume from a previous state.
func (gw *GraphWalker) Walk(ctx context.Context, stepExecutor StepExecutor, initialVisitCounts map[string]int) error {
	// Restore visit counts from previous state (resume support)
	if initialVisitCounts != nil {
		for k, v := range initialVisitCounts {
			gw.visitCounts[k] = v
			gw.totalVisits += v
		}
	}

	// Find the entry point: first step in the pipeline
	if len(gw.pipeline.Steps) == 0 {
		return fmt.Errorf("graph pipeline has no steps")
	}

	currentStepID := gw.pipeline.Steps[0].ID
	var lastResult *StepResult

	for currentStepID != "" {
		// Check context cancellation
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("graph execution cancelled: %w", err)
		}

		step, ok := gw.stepMap[currentStepID]
		if !ok {
			return fmt.Errorf("step %q not found in pipeline", currentStepID)
		}

		// Check per-step visit limit
		gw.mu.Lock()
		visits := gw.visitCounts[currentStepID]
		gw.mu.Unlock()

		maxVisits := step.EffectiveMaxVisits()
		if visits >= maxVisits {
			return fmt.Errorf("step %q exceeded max_visits limit (%d)", currentStepID, maxVisits)
		}

		// Check graph-level total visit limit
		maxTotal := gw.pipeline.EffectiveMaxStepVisits()
		gw.mu.Lock()
		if gw.totalVisits >= maxTotal {
			gw.mu.Unlock()
			return fmt.Errorf("pipeline exceeded max_step_visits limit (%d total visits)", maxTotal)
		}
		gw.mu.Unlock()

		// Increment visit count
		gw.mu.Lock()
		gw.visitCounts[currentStepID]++
		gw.totalVisits++
		gw.mu.Unlock()

		// Execute the step (or skip for conditional steps)
		var result *StepResult
		if step.Type == StepTypeConditional {
			// Conditional steps don't execute — they just evaluate edges.
			// Inherit outcome from the previous step for routing decisions.
			result = &StepResult{
				StepID:  currentStepID,
				Outcome: "success",
				Context: make(map[string]string),
			}
			if lastResult != nil {
				result.Outcome = lastResult.Outcome
				for k, v := range lastResult.Context {
					result.Context[k] = v
				}
			}
		} else {
			var err error
			result, err = stepExecutor(ctx, step)
			if err != nil {
				// Record error for circuit breaker
				if gw.checkCircuitBreaker(currentStepID, err.Error()) {
					return fmt.Errorf("circuit breaker triggered for step %q: same error repeated %d times: %w",
						currentStepID, circuitBreakerWindow, err)
				}

				// If the step has edges, route based on failure
				if len(step.Edges) > 0 {
					result = &StepResult{
						StepID:  currentStepID,
						Outcome: "failure",
						Error:   err,
						Context: make(map[string]string),
					}
				} else {
					// No edges — failure is fatal
					return fmt.Errorf("step %q failed: %w", currentStepID, err)
				}
			}
		}

		// Track last result for conditional step inheritance
		lastResult = result

		// Determine next step by evaluating edges
		nextStepID, err := gw.evaluateEdges(step, result)
		if err != nil {
			return err
		}

		currentStepID = nextStepID
	}

	return nil
}

// evaluateEdges picks the next step by evaluating edges in order.
// First matching condition wins. An edge with no condition is an unconditional fallback.
// Returns empty string when there are no edges (terminal step).
func (gw *GraphWalker) evaluateEdges(step *Step, result *StepResult) (string, error) {
	if len(step.Edges) == 0 {
		// No edges — this is a terminal step. Find the next step in DAG order.
		return gw.findNextDAGStep(step)
	}

	stepCtx := &StepContext{
		Outcome: result.Outcome,
		Context: result.Context,
	}
	if stepCtx.Context == nil {
		stepCtx.Context = make(map[string]string)
	}

	for _, edge := range step.Edges {
		cond, err := ParseCondition(edge.Condition)
		if err != nil {
			return "", fmt.Errorf("step %q edge to %q: %w", step.ID, edge.Target, err)
		}

		if EvaluateCondition(cond, stepCtx) {
			// Validate target exists
			if _, ok := gw.stepMap[edge.Target]; !ok {
				return "", fmt.Errorf("step %q edge targets non-existent step %q", step.ID, edge.Target)
			}
			return edge.Target, nil
		}
	}

	// No edge matched — check if there's a fallback (unconditional) edge
	// that we might have missed (shouldn't happen since unconditional always matches)
	return "", fmt.Errorf("step %q: no matching edge found (outcome=%q)", step.ID, result.Outcome)
}

// findNextDAGStep returns the next step in declaration order that depends on the current step.
// Returns empty string if there's no dependent step (terminal).
func (gw *GraphWalker) findNextDAGStep(step *Step) (string, error) {
	for i := range gw.pipeline.Steps {
		s := &gw.pipeline.Steps[i]
		for _, dep := range s.Dependencies {
			if dep == step.ID {
				return s.ID, nil
			}
		}
	}
	return "", nil // terminal step
}

// checkCircuitBreaker records an error and returns true if the circuit breaker
// should trip (same error repeated circuitBreakerWindow times).
func (gw *GraphWalker) checkCircuitBreaker(stepID string, errMsg string) bool {
	gw.mu.Lock()
	defer gw.mu.Unlock()

	normalized := normalizeErrorMessage(errMsg)
	history := gw.errorHistory[stepID]
	history = append(history, normalized)

	// Keep only last N errors
	if len(history) > circuitBreakerWindow {
		history = history[len(history)-circuitBreakerWindow:]
	}
	gw.errorHistory[stepID] = history

	// Check if all errors in the window are identical
	if len(history) < circuitBreakerWindow {
		return false
	}
	first := history[0]
	for _, msg := range history[1:] {
		if msg != first {
			return false
		}
	}
	return true
}

// normalizeErrorMessage strips variable parts (timestamps, line numbers)
// to compare error signatures.
func normalizeErrorMessage(msg string) string {
	// Strip common variable patterns for comparison
	lines := strings.Split(msg, "\n")
	var normalized []string
	for _, line := range lines {
		// Keep the line but strip leading whitespace
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		normalized = append(normalized, line)
	}
	result := strings.Join(normalized, "\n")
	// Truncate very long messages for comparison
	if len(result) > 500 {
		result = result[:500]
	}
	return result
}

// VisitCounts returns a copy of the current visit counts for state persistence.
func (gw *GraphWalker) VisitCounts() map[string]int {
	gw.mu.Lock()
	defer gw.mu.Unlock()
	counts := make(map[string]int, len(gw.visitCounts))
	for k, v := range gw.visitCounts {
		counts[k] = v
	}
	return counts
}
