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

// =============================================================================
// Phase 10: Edge Case Tests (T034)
// =============================================================================

// TestCircularDependencyDetectedAtLoadTime (T034) verifies that cycles are detected
// at pipeline load/validation time, not during execution. The error should clearly
// identify the cycle.
func TestCircularDependencyDetectedAtLoadTime(t *testing.T) {
	testCases := []struct {
		name          string
		steps         []Step
		expectError   bool
		errorContains string
	}{
		{
			name: "direct cycle A->B->A",
			steps: []Step{
				{ID: "a", Persona: "agent1", Dependencies: []string{"b"}},
				{ID: "b", Persona: "agent2", Dependencies: []string{"a"}},
			},
			expectError:   true,
			errorContains: "cycle",
		},
		{
			name: "triangle cycle A->B->C->A",
			steps: []Step{
				{ID: "a", Persona: "agent1", Dependencies: []string{"c"}},
				{ID: "b", Persona: "agent2", Dependencies: []string{"a"}},
				{ID: "c", Persona: "agent3", Dependencies: []string{"b"}},
			},
			expectError:   true,
			errorContains: "cycle",
		},
		{
			name: "self-referencing step",
			steps: []Step{
				{ID: "a", Persona: "agent1", Dependencies: []string{"a"}},
			},
			expectError:   true,
			errorContains: "cycle",
		},
		{
			name: "complex cycle with multiple paths",
			steps: []Step{
				{ID: "a", Persona: "agent1", Dependencies: []string{}},
				{ID: "b", Persona: "agent2", Dependencies: []string{"a"}},
				{ID: "c", Persona: "agent3", Dependencies: []string{"a"}},
				{ID: "d", Persona: "agent4", Dependencies: []string{"b", "e"}}, // depends on e, which depends on d
				{ID: "e", Persona: "agent5", Dependencies: []string{"c", "d"}}, // cycle: d -> e -> d
			},
			expectError:   true,
			errorContains: "cycle",
		},
		{
			name: "no cycle - valid DAG",
			steps: []Step{
				{ID: "a", Persona: "agent1", Dependencies: []string{}},
				{ID: "b", Persona: "agent2", Dependencies: []string{"a"}},
				{ID: "c", Persona: "agent3", Dependencies: []string{"a"}},
				{ID: "d", Persona: "agent4", Dependencies: []string{"b", "c"}},
			},
			expectError: false,
		},
		{
			name: "no cycle - diamond pattern",
			steps: []Step{
				{ID: "a", Persona: "agent1", Dependencies: []string{}},
				{ID: "b", Persona: "agent2", Dependencies: []string{"a"}},
				{ID: "c", Persona: "agent3", Dependencies: []string{"a"}},
				{ID: "d", Persona: "agent4", Dependencies: []string{"b", "c"}},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pipeline := &Pipeline{
				Metadata: PipelineMetadata{Name: "cycle-detection-test"},
				Steps:    tc.steps,
			}

			validator := &DAGValidator{}

			// Test ValidateDAG - this should detect cycles at validation time
			err := validator.ValidateDAG(pipeline)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, got nil", tc.name)
					return
				}
				// Verify the error message contains relevant information about the cycle
				if tc.errorContains != "" && !containsString(err.Error(), tc.errorContains) {
					t.Errorf("Error should contain %q, got: %s", tc.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for %s, got: %v", tc.name, err)
				}
			}

			// Also test TopologicalSort - it should fail for cyclic graphs
			_, sortErr := validator.TopologicalSort(pipeline)
			if tc.expectError {
				if sortErr == nil {
					t.Errorf("TopologicalSort should fail for cyclic graph: %s", tc.name)
				}
			} else {
				if sortErr != nil {
					t.Errorf("TopologicalSort should succeed for valid DAG: %s, got: %v", tc.name, sortErr)
				}
			}
		})
	}
}

// TestCycleErrorIdentifiesSteps verifies that cycle detection errors identify
// the steps involved in the cycle, making debugging easier.
func TestCycleErrorIdentifiesSteps(t *testing.T) {
	pipeline := &Pipeline{
		Metadata: PipelineMetadata{Name: "cycle-identification-test"},
		Steps: []Step{
			{ID: "alpha", Persona: "agent1", Dependencies: []string{"gamma"}},
			{ID: "beta", Persona: "agent2", Dependencies: []string{"alpha"}},
			{ID: "gamma", Persona: "agent3", Dependencies: []string{"beta"}},
		},
	}

	validator := &DAGValidator{}
	err := validator.ValidateDAG(pipeline)

	if err == nil {
		t.Fatal("Expected cycle detection error")
	}

	errorMsg := err.Error()

	// The error should mention "cycle" and ideally some of the step IDs involved
	if !containsString(errorMsg, "cycle") {
		t.Errorf("Error should mention 'cycle', got: %s", errorMsg)
	}

	// Check that at least one step ID is mentioned in the error
	stepsMentioned := containsString(errorMsg, "alpha") ||
		containsString(errorMsg, "beta") ||
		containsString(errorMsg, "gamma")

	if !stepsMentioned {
		t.Errorf("Error should mention at least one step ID involved in cycle, got: %s", errorMsg)
	}
}

// TestMissingDependencyDetectedAtLoadTime verifies that missing dependencies
// are also detected at validation time, not during execution.
func TestMissingDependencyDetectedAtLoadTime(t *testing.T) {
	pipeline := &Pipeline{
		Metadata: PipelineMetadata{Name: "missing-dep-test"},
		Steps: []Step{
			{ID: "a", Persona: "agent1", Dependencies: []string{"nonexistent-step"}},
			{ID: "b", Persona: "agent2", Dependencies: []string{"a", "also-missing"}},
		},
	}

	validator := &DAGValidator{}
	err := validator.ValidateDAG(pipeline)

	if err == nil {
		t.Fatal("Expected error for missing dependencies")
	}

	// Error should mention the missing step
	if !containsString(err.Error(), "nonexistent") {
		t.Errorf("Error should mention 'nonexistent-step', got: %s", err.Error())
	}
}

// containsString is a helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
