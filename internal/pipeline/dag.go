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

