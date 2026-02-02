package pipeline

import (
	"testing"
)

func TestValidateDAG_ValidPipeline(t *testing.T) {
	pipeline := &Pipeline{
		Steps: []Step{
			{ID: "step1", Persona: "agent1", Dependencies: []string{}},
			{ID: "step2", Persona: "agent2", Dependencies: []string{"step1"}},
			{ID: "step3", Persona: "agent3", Dependencies: []string{"step2"}},
		},
	}

	validator := &DAGValidator{}
	err := validator.ValidateDAG(pipeline)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestValidateDAG_MissingDependency(t *testing.T) {
	pipeline := &Pipeline{
		Steps: []Step{
			{ID: "step1", Persona: "agent1", Dependencies: []string{"nonexistent"}},
		},
	}

	validator := &DAGValidator{}
	err := validator.ValidateDAG(pipeline)
	if err == nil {
		t.Error("Expected error for missing dependency, got nil")
	}
}

func TestValidateDAG_CycleDetection(t *testing.T) {
	pipeline := &Pipeline{
		Steps: []Step{
			{ID: "step1", Persona: "agent1", Dependencies: []string{"step3"}},
			{ID: "step2", Persona: "agent2", Dependencies: []string{"step1"}},
			{ID: "step3", Persona: "agent3", Dependencies: []string{"step2"}},
		},
	}

	validator := &DAGValidator{}
	err := validator.ValidateDAG(pipeline)
	if err == nil {
		t.Error("Expected error for cycle, got nil")
	}
}

func TestValidateDAG_SelfReference(t *testing.T) {
	pipeline := &Pipeline{
		Steps: []Step{
			{ID: "step1", Persona: "agent1", Dependencies: []string{"step1"}},
		},
	}

	validator := &DAGValidator{}
	err := validator.ValidateDAG(pipeline)
	if err == nil {
		t.Error("Expected error for self-reference, got nil")
	}
}

func TestTopologicalSort_SimplePipeline(t *testing.T) {
	pipeline := &Pipeline{
		Steps: []Step{
			{ID: "step1", Persona: "agent1"},
			{ID: "step2", Persona: "agent2", Dependencies: []string{"step1"}},
			{ID: "step3", Persona: "agent3", Dependencies: []string{"step2"}},
		},
	}

	validator := &DAGValidator{}
	result, err := validator.TopologicalSort(pipeline)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 steps, got %d", len(result))
	}

	order := make(map[string]int)
	for i, step := range result {
		order[step.ID] = i
	}

	if order["step1"] >= order["step2"] {
		t.Error("step1 should come before step2")
	}
	if order["step2"] >= order["step3"] {
		t.Error("step2 should come before step3")
	}
}

func TestTopologicalSort_ComplexPipeline(t *testing.T) {
	pipeline := &Pipeline{
		Steps: []Step{
			{ID: "a", Persona: "agent1"},
			{ID: "b", Persona: "agent2"},
			{ID: "c", Persona: "agent3", Dependencies: []string{"a", "b"}},
			{ID: "d", Persona: "agent4", Dependencies: []string{"a"}},
			{ID: "e", Persona: "agent5", Dependencies: []string{"c", "d"}},
		},
	}

	validator := &DAGValidator{}
	result, err := validator.TopologicalSort(pipeline)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	order := make(map[string]int)
	for i, step := range result {
		order[step.ID] = i
	}

	if order["a"] >= order["c"] || order["b"] >= order["c"] {
		t.Error("a and b should come before c")
	}
	if order["a"] >= order["d"] {
		t.Error("a should come before d")
	}
	if order["c"] >= order["e"] || order["d"] >= order["e"] {
		t.Error("c and d should come before e")
	}
}

func TestTopologicalSort_IndependentSteps(t *testing.T) {
	pipeline := &Pipeline{
		Steps: []Step{
			{ID: "step1", Persona: "agent1"},
			{ID: "step2", Persona: "agent2"},
			{ID: "step3", Persona: "agent3"},
		},
	}

	validator := &DAGValidator{}
	result, err := validator.TopologicalSort(pipeline)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 steps, got %d", len(result))
	}
}

func TestTopologicalSort_WithCycle(t *testing.T) {
	pipeline := &Pipeline{
		Steps: []Step{
			{ID: "step1", Persona: "agent1", Dependencies: []string{"step2"}},
			{ID: "step2", Persona: "agent2", Dependencies: []string{"step1"}},
		},
	}

	validator := &DAGValidator{}
	_, err := validator.TopologicalSort(pipeline)
	if err == nil {
		t.Error("Expected error for cycle, got nil")
	}
}

func TestYAMLPipelineLoader_ValidYAML(t *testing.T) {
	yamlContent := []byte(`
kind: WavePipeline
metadata:
  name: test-pipeline
input:
  source: cli
steps:
  - id: step1
    persona: agent1
    memory:
      strategy: fresh
    workspace:
      root: ./
    exec:
      type: prompt
      source: test prompt
`)

	loader := &YAMLPipelineLoader{}
	pipeline, err := loader.Unmarshal(yamlContent)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if pipeline.Metadata.Name != "test-pipeline" {
		t.Errorf("Expected name 'test-pipeline', got '%s'", pipeline.Metadata.Name)
	}

	if len(pipeline.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(pipeline.Steps))
	}
}

func TestYAMLPipelineLoader_InvalidYAML(t *testing.T) {
	yamlContent := []byte(`
invalid: yaml: content:
  - unclosed [ bracket
`)

	loader := &YAMLPipelineLoader{}
	_, err := loader.Unmarshal(yamlContent)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}
