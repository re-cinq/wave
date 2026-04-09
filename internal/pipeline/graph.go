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

	// Dependency-deferred steps: when an edge targets a step whose
	// dependencies are not yet satisfied, the target is queued here
	// and the walker executes unmet dependencies first.
	pending []string // FIFO queue of deferred step IDs

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
	for k, v := range initialVisitCounts {
		gw.visitCounts[k] = v
		gw.totalVisits += v
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

		// Enforce dependency constraints: if the target step has
		// dependencies that haven't been visited yet, defer it and
		// route to an unsatisfied dependency instead.
		nextStepID, err = gw.resolveDeps(nextStepID)
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
//
// IMPORTANT: This returns only the first dependent found in declaration order.
// ValidateGraph rejects pipelines where an edge-less step has multiple dependents,
// so this single-return behavior is safe. If fan-out is needed, the step must
// define explicit edges to route to multiple successors.
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

// resolveDeps enforces dependency constraints before routing to the next step.
//
// After each step completes and evaluateEdges proposes a nextStepID, this
// method checks two things:
//
//  1. Are there any unvisited steps whose dependencies are NOW all satisfied?
//     These "dependency-gated" steps must execute before the edge target,
//     because edges alone would bypass them.
//
//  2. Does the proposed target itself have unsatisfied dependencies?
//     If so, defer it and route to a ready dependency instead.
//
// This ensures steps with `dependencies:` constraints are never skipped
// even when edges route around them.
func (gw *GraphWalker) resolveDeps(nextStepID string) (string, error) {
	gw.mu.Lock()
	defer gw.mu.Unlock()

	// Check for dependency-gated steps that have become ready and would
	// be bypassed by direct edge routing.
	if gated := gw.findGatedReady(nextStepID); gated != "" {
		// Defer the original edge target so we come back to it later.
		if nextStepID != "" {
			gw.pending = append(gw.pending, nextStepID)
		}
		return gated, nil
	}

	// Terminal — check pending queue.
	if nextStepID == "" {
		if id := gw.drainPending(); id != "" {
			return id, nil
		}
		return "", nil
	}

	target, ok := gw.stepMap[nextStepID]
	if !ok {
		return nextStepID, nil // will be caught later
	}

	if gw.depsReady(target) {
		// If this step was previously deferred, remove it from pending.
		gw.removePending(nextStepID)
		return nextStepID, nil
	}

	// Dependencies not met — defer the target and route to a ready dep.
	gw.pending = append(gw.pending, nextStepID)

	if readyDep := gw.findReadyDep(target); readyDep != "" {
		return readyDep, nil
	}

	// No individual dep is ready — check pending queue for anything
	// else that has become unblocked.
	if id := gw.drainPending(); id != "" {
		return id, nil
	}

	return "", fmt.Errorf("step %q has unsatisfied dependencies and no ready step available", nextStepID)
}

// findGatedReady scans all pipeline steps for one that:
//   - has not been visited yet
//   - has dependencies (i.e., is dependency-gated)
//   - has all dependencies now satisfied
//   - is NOT the proposed nextStepID (that one is handled separately)
//   - is not already in the pending queue
//   - is NOT the target of any edge in the pipeline (edge targets are
//     reached via normal edge routing; only "orphaned" gated steps need
//     injection)
//
// This catches steps that edges would otherwise bypass entirely.
func (gw *GraphWalker) findGatedReady(nextStepID string) string {
	// Build set of all edge targets — these steps are reachable via edges
	// and should not be force-injected.
	edgeTargets := gw.edgeTargetSet()

	inPending := make(map[string]bool, len(gw.pending))
	for _, id := range gw.pending {
		inPending[id] = true
	}

	for i := range gw.pipeline.Steps {
		s := &gw.pipeline.Steps[i]
		if s.ID == nextStepID {
			continue // handled by the caller
		}
		if gw.visitCounts[s.ID] > 0 {
			continue // already visited
		}
		if len(s.Dependencies) == 0 {
			continue // no dependency gate
		}
		if inPending[s.ID] {
			continue // already queued
		}
		if edgeTargets[s.ID] {
			continue // reachable via edges — don't force-inject
		}
		if gw.depsReady(s) {
			return s.ID
		}
	}
	return ""
}

// edgeTargetSet returns the set of step IDs that are the target of at
// least one edge in the pipeline.
func (gw *GraphWalker) edgeTargetSet() map[string]bool {
	targets := make(map[string]bool)
	for i := range gw.pipeline.Steps {
		for _, edge := range gw.pipeline.Steps[i].Edges {
			targets[edge.Target] = true
		}
	}
	return targets
}

// depsReady returns true if all of the step's declared dependencies have been
// visited at least once. It must be called with gw.mu held (or from a
// context where visitCounts is not concurrently modified).
func (gw *GraphWalker) depsReady(step *Step) bool {
	for _, dep := range step.Dependencies {
		if gw.visitCounts[dep] == 0 {
			return false
		}
	}
	return true
}

// removePending removes a specific step from the pending queue if present.
func (gw *GraphWalker) removePending(stepID string) {
	for i, id := range gw.pending {
		if id == stepID {
			gw.pending = append(gw.pending[:i], gw.pending[i+1:]...)
			return
		}
	}
}

// drainPending returns the first step in the pending queue whose
// dependencies are now satisfied, removing it from the queue.
// Returns empty string if nothing is ready.
func (gw *GraphWalker) drainPending() string {
	for i, id := range gw.pending {
		step := gw.stepMap[id]
		if step != nil && gw.depsReady(step) {
			gw.pending = append(gw.pending[:i], gw.pending[i+1:]...)
			return id
		}
	}
	return ""
}

// findReadyDep walks the dependency tree of target to find a step that
// is unvisited but whose own dependencies are all satisfied (i.e., ready
// to execute). This traversal is recursive so transitive deps are resolved.
func (gw *GraphWalker) findReadyDep(target *Step) string {
	return gw.findReadyDepVisited(target, make(map[string]bool))
}

func (gw *GraphWalker) findReadyDepVisited(target *Step, seen map[string]bool) string {
	for _, dep := range target.Dependencies {
		if gw.visitCounts[dep] > 0 {
			continue // already visited
		}
		if seen[dep] {
			continue // avoid cycles
		}
		seen[dep] = true
		depStep := gw.stepMap[dep]
		if depStep == nil {
			continue
		}
		if gw.depsReady(depStep) {
			return dep
		}
		// dep itself isn't ready — try resolving ITS deps first
		if deeper := gw.findReadyDepVisited(depStep, seen); deeper != "" {
			return deeper
		}
	}
	return ""
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
