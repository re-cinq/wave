package pipeline

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
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
	if err := yaml.Unmarshal(data, &pipeline); err != nil {
		return nil, fmt.Errorf("failed to parse pipeline YAML: %w", err)
	}

	if pipeline.Kind == "" {
		pipeline.Kind = "WavePipeline"
	}

	// Default memory strategy to "fresh" (constitutional requirement)
	for i := range pipeline.Steps {
		if pipeline.Steps[i].Memory.Strategy == "" {
			pipeline.Steps[i].Memory.Strategy = "fresh"
		}
	}

	return &pipeline, nil
}

type DAGValidator struct{}

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
		if step.Retry.OnFailure == "rework" {
			if err := v.validateReworkTarget(step.ID, step.Retry.ReworkStep, stepMap); err != nil {
				return err
			}
		}
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

