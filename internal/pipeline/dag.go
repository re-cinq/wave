package pipeline

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/recinq/wave/internal/skill"
	"gopkg.in/yaml.v3"
)

type PipelineLoader interface {
	Load(path string) (*Pipeline, error)
}

type YAMLPipelineLoader struct{}

func (l *YAMLPipelineLoader) Load(path string) (*Pipeline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read pipeline file: %w", err)
	}
	return l.Unmarshal(data)
}

func (l *YAMLPipelineLoader) Unmarshal(data []byte) (*Pipeline, error) {
	var pipeline Pipeline
	// Use strict decoder that rejects unknown YAML fields.
	// This catches hallucinated fields like allow_recovery, recovery_level, etc.
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&pipeline); err != nil {
		return nil, fmt.Errorf("failed to parse pipeline YAML: %w", err)
	}

	applyPipelineDefaults(&pipeline)

	// Type-check I/O protocol declarations (input.type, pipeline_outputs[*].type,
	// step input_ref) against the shared schema registry. Catches misspelled
	// type names before any step runs. See docs/adr/010-pipeline-io-protocol.md.
	if err := ValidatePipelineIOTypes(&pipeline); err != nil {
		return nil, err
	}

	// Cross-pipeline typed-wiring check: when a sub-pipeline step consumes a
	// sibling output via input_ref.from, verify the source's declared output
	// type matches the child pipeline's declared input type. The loader is
	// nil here (no recursive child loading at top-level Unmarshal); the check
	// only enforces shape and intra-pipeline rules. Cross-pipeline typing is
	// enforced by SequenceExecutor when it actually loads the children.
	if err := TypedWiringCheck(&pipeline, nil, ""); err != nil {
		return nil, err
	}

	// Enforce Wave Lego Protocol (ADR-011) at load time. Rules 3 and 5 are
	// hard errors: pipeline_outputs must declare types, and contract
	// on_failure must be one of fail/skip/continue/rework/warn. Shipped
	// pipelines have been migrated; fail fast on any drift.
	if errs := CollectWLPLoadErrors(&pipeline); len(errs) > 0 {
		return nil, fmt.Errorf("WLP validation failed: %s", strings.Join(errs, "; "))
	}

	return &pipeline, nil
}

// applyPipelineDefaults sets Kind and Memory.Strategy defaults on a parsed Pipeline.
func applyPipelineDefaults(p *Pipeline) {
	if p.Kind == "" {
		p.Kind = "WavePipeline"
	}

	// Default memory strategy to "fresh" (constitutional requirement)
	for i := range p.Steps {
		if p.Steps[i].Memory.Strategy == "" {
			p.Steps[i].Memory.Strategy = "fresh"
		}
	}
}

type DAGValidator struct {
	Warnings []string // Non-fatal validation warnings (e.g., mixed-persona thread groups)
}

func (v *DAGValidator) ValidateDAG(p *Pipeline) error {
	stepMap := make(map[string]*Step)
	for i := range p.Steps {
		stepMap[p.Steps[i].ID] = &p.Steps[i]
	}

	for _, step := range p.Steps {
		for _, depID := range step.Dependencies {
			if _, exists := stepMap[depID]; !exists {
				return fmt.Errorf("step %q depends on non-existent step %q", step.ID, depID)
			}
		}

		// Validate artifact refs for mutual exclusivity of Step and Pipeline
		for i, ref := range step.Memory.InjectArtifacts {
			if err := ref.Validate(step.ID, i); err != nil {
				return err
			}
		}

		// Validate RetryConfig
		if err := step.Retry.Validate(); err != nil {
			return fmt.Errorf("step %q: %w", step.ID, err)
		}

		// Validate rework targets
		if step.Retry.OnFailure == OnFailureRework {
			if err := v.validateReworkTarget(step.ID, step.Retry.ReworkStep, stepMap); err != nil {
				return err
			}
		}

		// Validate agent_review contract fields: self-review prevention and contract-level rework_step
		contracts := step.Handover.EffectiveContracts()
		for _, c := range contracts {
			if c.Type != "agent_review" {
				continue
			}
			// Self-review prevention: reviewer persona must differ from step persona
			if c.Persona != "" && c.Persona == step.Persona {
				return fmt.Errorf("step %q: agent_review contract persona %q must differ from step persona (self-review not allowed)",
					step.ID, c.Persona)
			}
			// Validate contract-level rework_step target
			if c.OnFailure == OnFailureRework && c.ReworkStep != "" {
				if err := v.validateReworkTarget(step.ID, c.ReworkStep, stepMap); err != nil {
					return fmt.Errorf("step %q: agent_review contract rework_step: %w", step.ID, err)
				}
			}
			// Validate contract-level on_failure enum
			if c.OnFailure != "" {
				switch c.OnFailure {
				case OnFailureFail, OnFailureSkip, OnFailureContinue, OnFailureRework, OnFailureWarn:
					// valid
				default:
					return fmt.Errorf("step %q: agent_review contract has invalid on_failure value %q (must be fail, skip, continue, rework, or warn)",
						step.ID, c.OnFailure)
				}
			}
		}
	}

	// Validate that each rework target is unique (prevent race on concurrent rework)
	reworkTargets := make(map[string]string) // target -> source step
	for _, step := range p.Steps {
		if step.Retry.OnFailure == OnFailureRework {
			target := step.Retry.ReworkStep
			if existing, ok := reworkTargets[target]; ok {
				return fmt.Errorf("rework target %q is used by both step %q and step %q (each target must be unique)", target, existing, step.ID)
			}
			reworkTargets[target] = step.ID
		}
	}

	// Validate on_failure enum values
	for _, step := range p.Steps {
		if step.Retry.OnFailure != "" {
			switch step.Retry.OnFailure {
			case OnFailureFail, OnFailureSkip, OnFailureContinue, OnFailureRework, OnFailureWarn:
				// valid
			default:
				return fmt.Errorf("step %q has invalid on_failure value %q (must be fail, skip, continue, rework, or warn)", step.ID, step.Retry.OnFailure)
			}
		}
		// Validate concurrency and matrix strategy are mutually exclusive
		if step.Concurrency > 1 && step.Strategy != nil && step.Strategy.Type == "matrix" {
			return fmt.Errorf("step %q sets both concurrency (%d) and matrix strategy — they are mutually exclusive", step.ID, step.Concurrency)
		}
	}

	// Validate thread group constraints
	if err := v.validateThreadGroups(p, stepMap); err != nil {
		return err
	}

	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for _, step := range p.Steps {
		if !visited[step.ID] {
			if err := v.detectCycle(step.ID, stepMap, visited, recStack); err != nil {
				return err
			}
		}
	}

	return nil
}

// validFidelityValues enumerates acceptable fidelity field values.
var validFidelityValues = map[string]bool{
	FidelityFull: true, FidelityCompact: true, FidelitySummary: true, FidelityFresh: true, "": true,
}

// validateThreadGroups validates thread group constraints:
// 1. Steps in the same thread group must form a dependency chain (cannot be concurrent)
// 2. Mixed-persona thread groups emit a warning
// 3. Fidelity field values must be valid
func (v *DAGValidator) validateThreadGroups(p *Pipeline, stepMap map[string]*Step) error {
	// Validate fidelity field values on all steps
	for _, step := range p.Steps {
		if !validFidelityValues[step.Fidelity] {
			return fmt.Errorf("step %q has invalid fidelity value %q (must be full, compact, summary, or fresh)", step.ID, step.Fidelity)
		}
		if step.Fidelity != "" && step.Thread == "" {
			v.Warnings = append(v.Warnings, fmt.Sprintf("step %q has fidelity %q but no thread — fidelity has no effect without a thread group", step.ID, step.Fidelity))
		}
	}

	// Group steps by thread (skip rework_only steps — they are triggered by the
	// rework mechanism, not the normal DAG scheduler, so they don't participate
	// in thread concurrency ordering)
	threadGroups := make(map[string][]Step) // threadGroup -> ordered steps
	for _, step := range p.Steps {
		if step.Thread != "" && !step.ReworkOnly {
			threadGroups[step.Thread] = append(threadGroups[step.Thread], step)
		}
	}

	for threadName, steps := range threadGroups {
		if len(steps) < 2 {
			continue
		}

		// Check that consecutive steps in the same thread have a dependency chain.
		// Each step (except the first) must directly or transitively depend on
		// at least one earlier step in the same thread group.
		for i := 1; i < len(steps); i++ {
			hasDep := false
			for j := 0; j < i; j++ {
				if v.isTransitiveDep(steps[i].ID, steps[j].ID, stepMap) || v.directlyDependsOn(steps[i], steps[j].ID) {
					hasDep = true
					break
				}
			}
			if !hasDep {
				return fmt.Errorf("step %q in thread %q has no dependency on prior thread step %q — steps in the same thread must form a dependency chain to prevent concurrent execution",
					steps[i].ID, threadName, steps[i-1].ID)
			}
		}

		// Warn on mixed-persona thread groups
		personas := make(map[string]bool)
		for _, step := range steps {
			personas[step.Persona] = true
		}
		if len(personas) > 1 {
			v.Warnings = append(v.Warnings, fmt.Sprintf("thread %q has steps with different personas — persona isolation is still enforced but conversation context may cross persona boundaries", threadName))
		}
	}

	return nil
}

// directlyDependsOn returns true if step directly lists depID in its Dependencies.
func (v *DAGValidator) directlyDependsOn(step Step, depID string) bool {
	for _, dep := range step.Dependencies {
		if dep == depID {
			return true
		}
	}
	return false
}

// validateReworkTarget checks that a rework_step reference is valid:
// 1. The target step exists in the pipeline
// 2. The target is not a direct or transitive dependency of the failing step
// 3. The failing step is not a dependency of the rework target (would create cycle)
func (v *DAGValidator) validateReworkTarget(stepID, reworkStepID string, stepMap map[string]*Step) error {
	if _, exists := stepMap[reworkStepID]; !exists {
		return fmt.Errorf("step %q has rework_step %q which does not exist in the pipeline", stepID, reworkStepID)
	}

	if stepID == reworkStepID {
		return fmt.Errorf("step %q cannot rework to itself", stepID)
	}

	// Rework target must be marked as rework_only to prevent double execution
	if target := stepMap[reworkStepID]; !target.ReworkOnly {
		return fmt.Errorf("rework target %q must have rework_only: true (referenced by step %q)", reworkStepID, stepID)
	}

	// Check that rework target is not an upstream dependency of the failing step
	if v.isTransitiveDep(stepID, reworkStepID, stepMap) {
		return fmt.Errorf("step %q has rework_step %q which is an upstream dependency (would create cycle)", stepID, reworkStepID)
	}

	// Check that the failing step is not a dependency of the rework target
	if v.isTransitiveDep(reworkStepID, stepID, stepMap) {
		return fmt.Errorf("step %q has rework_step %q which depends on step %q (would create cycle)", stepID, reworkStepID, stepID)
	}

	return nil
}

// isTransitiveDep returns true if targetID is a direct or transitive dependency of stepID.
func (v *DAGValidator) isTransitiveDep(stepID, targetID string, stepMap map[string]*Step) bool {
	visited := make(map[string]bool)
	return v.reachable(stepID, targetID, stepMap, visited)
}

// reachable checks if targetID is reachable from stepID via dependencies.
func (v *DAGValidator) reachable(stepID, targetID string, stepMap map[string]*Step, visited map[string]bool) bool {
	if visited[stepID] {
		return false
	}
	visited[stepID] = true

	step, exists := stepMap[stepID]
	if !exists {
		return false
	}

	for _, dep := range step.Dependencies {
		if dep == targetID {
			return true
		}
		if v.reachable(dep, targetID, stepMap, visited) {
			return true
		}
	}
	return false
}

func (v *DAGValidator) detectCycle(stepID string, stepMap map[string]*Step, visited, recStack map[string]bool) error {
	visited[stepID] = true
	recStack[stepID] = true

	step, exists := stepMap[stepID]
	if !exists {
		return nil
	}

	for _, depID := range step.Dependencies {
		if !visited[depID] {
			if err := v.detectCycle(depID, stepMap, visited, recStack); err != nil {
				return err
			}
		} else if recStack[depID] {
			return fmt.Errorf("cycle detected: step %q depends on %q, creating a circular dependency", stepID, depID)
		}
	}

	recStack[stepID] = false
	return nil
}

func (v *DAGValidator) TopologicalSort(p *Pipeline) ([]*Step, error) {
	if err := v.ValidateDAG(p); err != nil {
		return nil, err
	}

	stepMap := make(map[string]*Step)
	for i := range p.Steps {
		stepMap[p.Steps[i].ID] = &p.Steps[i]
	}

	visited := make(map[string]bool)
	result := make([]*Step, 0, len(p.Steps))

	var visit func(stepID string) error
	visit = func(stepID string) error {
		if visited[stepID] {
			return nil
		}
		visited[stepID] = true

		step := stepMap[stepID]
		for _, depID := range step.Dependencies {
			if err := visit(depID); err != nil {
				return err
			}
		}

		result = append(result, step)
		return nil
	}

	for _, step := range p.Steps {
		if err := visit(step.ID); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// validatePipelineSkills validates skill references declared in the pipeline's Skills field.
// This is called from the executor when a skill store is available.
func validatePipelineSkills(p *Pipeline, store skill.Store) []error {
	if len(p.Skills) == 0 || store == nil {
		return nil
	}
	return skill.ValidateSkillRefs(p.Skills, "pipeline:"+p.Metadata.Name, store)
}

// detectSubPipelineCycles checks for circular sub-pipeline references across pipelines.
// It loads referenced sub-pipelines from disk and walks the reference graph.
// Returns an error if pipeline A references B which transitively references A.
func detectSubPipelineCycles(p *Pipeline, pipelinesDir string) error {
	visited := map[string]bool{}
	recStack := map[string]bool{}
	loader := &YAMLPipelineLoader{}

	return detectSubPipelineCyclesDFS(p.Metadata.Name, pipelinesDir, loader, visited, recStack)
}

func detectSubPipelineCyclesDFS(name, pipelinesDir string, loader *YAMLPipelineLoader, visited, recStack map[string]bool) error {
	if recStack[name] {
		return fmt.Errorf("circular sub-pipeline reference detected: pipeline %q references itself transitively", name)
	}
	if visited[name] {
		return nil
	}

	visited[name] = true
	recStack[name] = true

	// Try to load the pipeline to find its sub-pipeline references
	path := pipelinesDir + "/" + name + ".yaml"
	p, err := loader.Load(path)
	if err != nil {
		// Pipeline file not found — can't trace further, not an error
		recStack[name] = false
		return nil
	}

	for _, step := range p.Steps {
		if step.SubPipeline != "" {
			if err := detectSubPipelineCyclesDFS(step.SubPipeline, pipelinesDir, loader, visited, recStack); err != nil {
				return err
			}
		}
	}

	recStack[name] = false
	return nil
}

// isGraphPipeline returns true if any step defines edges or uses a conditional type,
// indicating the pipeline should be executed in graph mode rather than DAG mode.
func isGraphPipeline(p *Pipeline) bool {
	for _, step := range p.Steps {
		if len(step.Edges) > 0 || step.Type == StepTypeConditional {
			return true
		}
	}
	return false
}

// ValidateGraph validates a graph-mode pipeline. Unlike ValidateDAG, it allows
// backward edges (cycles) but enforces safety limits and structural requirements.
func (v *DAGValidator) ValidateGraph(p *Pipeline) error {
	stepMap := make(map[string]*Step)
	for i := range p.Steps {
		stepMap[p.Steps[i].ID] = &p.Steps[i]
	}

	// Validate basic step structure (dependencies exist, retry config, etc.)
	for _, step := range p.Steps {
		for _, depID := range step.Dependencies {
			if _, exists := stepMap[depID]; !exists {
				return fmt.Errorf("step %q depends on non-existent step %q", step.ID, depID)
			}
		}
		if err := step.Retry.Validate(); err != nil {
			return fmt.Errorf("step %q: %w", step.ID, err)
		}
	}

	// Validate edge targets exist (allow _complete sentinel for pipeline termination)
	for _, step := range p.Steps {
		for _, edge := range step.Edges {
			if edge.Target != EdgeTargetComplete {
				if _, exists := stepMap[edge.Target]; !exists {
					return fmt.Errorf("step %q has edge targeting non-existent step %q", step.ID, edge.Target)
				}
			}
			// Validate condition syntax
			if edge.Condition != "" {
				if _, err := ParseCondition(edge.Condition); err != nil {
					return fmt.Errorf("step %q edge to %q: %w", step.ID, edge.Target, err)
				}
			}
		}
	}

	// Validate conditional steps have edges
	for _, step := range p.Steps {
		if step.Type == StepTypeConditional && len(step.Edges) == 0 {
			return fmt.Errorf("step %q is type=conditional but has no edges defined", step.ID)
		}
	}

	// Validate command steps have scripts
	for _, step := range p.Steps {
		if step.Type == StepTypeCommand && step.Script == "" {
			return fmt.Errorf("step %q is type=command but has no script defined", step.ID)
		}
	}

	// Validate max_visits is positive when set
	for _, step := range p.Steps {
		if step.MaxVisits < 0 {
			return fmt.Errorf("step %q has negative max_visits (%d)", step.ID, step.MaxVisits)
		}
	}

	// Validate max_step_visits is positive when set
	if p.MaxStepVisits < 0 {
		return fmt.Errorf("pipeline has negative max_step_visits (%d)", p.MaxStepVisits)
	}

	// Validate that no step without edges has multiple dependents (fan-out).
	// findNextDAGStep returns the first dependent found in declaration order,
	// silently dropping any additional dependents. Reject at validation time
	// so pipeline authors get a clear error instead of surprising behavior.
	for _, step := range p.Steps {
		if len(step.Edges) > 0 {
			continue // steps with explicit edges use edge routing, not DAG fallback
		}
		var dependents []string
		for _, candidate := range p.Steps {
			for _, dep := range candidate.Dependencies {
				if dep == step.ID {
					dependents = append(dependents, candidate.ID)
				}
			}
		}
		if len(dependents) > 1 {
			return fmt.Errorf("step %q has no edges but multiple dependents %v; add explicit edges to control fan-out routing", step.ID, dependents)
		}
	}

	return nil
}
