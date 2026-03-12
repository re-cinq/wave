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

	// Validate rework configurations early
	for i := range pipeline.Steps {
		if pipeline.Steps[i].Retry.Rework != nil {
			if err := pipeline.Steps[i].Retry.Rework.Validate(); err != nil {
				return nil, fmt.Errorf("step %q: %w", pipeline.Steps[i].ID, err)
			}
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
	}

	// Validate rework targets
	for _, step := range p.Steps {
		if step.Retry.OnFailure == "rework" && step.Retry.Rework != nil {
			if err := step.Retry.Rework.Validate(); err != nil {
				return fmt.Errorf("step %q: %w", step.ID, err)
			}
			if step.Retry.Rework.TargetStep != "" {
				if _, exists := stepMap[step.Retry.Rework.TargetStep]; !exists {
					return fmt.Errorf("step %q rework target_step %q does not exist in pipeline",
						step.ID, step.Retry.Rework.TargetStep)
				}
			}
		}
	}

	// Detect rework cycles
	if err := v.detectReworkCycles(p.Steps, stepMap); err != nil {
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

// detectReworkCycles checks for cycles in rework target references.
// e.g., step A reworks to B, step B reworks to A.
func (v *DAGValidator) detectReworkCycles(steps []Step, stepMap map[string]*Step) error {
	// Build rework adjacency: stepID -> rework target step ID
	reworkTargets := make(map[string]string)
	for _, step := range steps {
		if step.Retry.OnFailure == "rework" && step.Retry.Rework != nil && step.Retry.Rework.TargetStep != "" {
			reworkTargets[step.ID] = step.Retry.Rework.TargetStep
		}
	}

	// Walk each chain checking for cycles
	for startID := range reworkTargets {
		visited := map[string]bool{startID: true}
		current := reworkTargets[startID]
		for current != "" {
			if visited[current] {
				return fmt.Errorf("rework cycle detected: step %q rework chain leads back to %q", startID, current)
			}
			visited[current] = true
			current = reworkTargets[current]
		}
	}
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

