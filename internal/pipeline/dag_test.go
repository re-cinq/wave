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

func TestValidateDAG_ArtifactRefStepAndPipelineMutuallyExclusive(t *testing.T) {
	tests := []struct {
		name    string
		refs    []ArtifactRef
		wantErr bool
	}{
		{
			name: "step only is valid",
			refs: []ArtifactRef{
				{Step: "analyze", Artifact: "report", As: "input"},
			},
			wantErr: false,
		},
		{
			name: "pipeline only is valid",
			refs: []ArtifactRef{
				{Pipeline: "other-pipeline", Artifact: "report", As: "input"},
			},
			wantErr: false,
		},
		{
			name:    "neither step nor pipeline is valid",
			refs:    []ArtifactRef{{Artifact: "report", As: "input"}},
			wantErr: false,
		},
		{
			name: "both step and pipeline is invalid",
			refs: []ArtifactRef{
				{Step: "analyze", Pipeline: "other-pipeline", Artifact: "report", As: "input"},
			},
			wantErr: true,
		},
		{
			name: "second ref has both step and pipeline",
			refs: []ArtifactRef{
				{Step: "analyze", Artifact: "report", As: "input"},
				{Step: "build", Pipeline: "other", Artifact: "output", As: "build-output"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline := &Pipeline{
				Steps: []Step{
					{
						ID:      "step1",
						Persona: "agent1",
						Memory: MemoryConfig{
							Strategy:        "fresh",
							InjectArtifacts: tt.refs,
						},
					},
				},
			}

			validator := &DAGValidator{}
			err := validator.ValidateDAG(pipeline)
			if tt.wantErr && err == nil {
				t.Error("expected error for mutually exclusive step and pipeline, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
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

// ============================================================================
// Rework DAG Validation Tests
// ============================================================================

func TestValidateDAG_ReworkTargetExists(t *testing.T) {
	pipeline := &Pipeline{
		Steps: []Step{
			{
				ID:      "step1",
				Persona: "agent1",
				Retry: RetryConfig{
					OnFailure:  "rework",
					ReworkStep: "nonexistent",
				},
			},
		},
	}

	validator := &DAGValidator{}
	err := validator.ValidateDAG(pipeline)
	if err == nil {
		t.Fatal("Expected error for rework target referencing nonexistent step, got nil")
	}
	if got := err.Error(); !contains(got, "does not exist") {
		t.Errorf("Expected error about nonexistent step, got: %s", got)
	}
}

func TestValidateDAG_ReworkTargetNotUpstream(t *testing.T) {
	// step2 depends on step1, so step1 is an upstream dep of step2.
	// step2 cannot rework to step1 because it's upstream.
	pipeline := &Pipeline{
		Steps: []Step{
			{ID: "step1", Persona: "agent1"},
			{
				ID:           "step2",
				Persona:      "agent2",
				Dependencies: []string{"step1"},
				Retry: RetryConfig{
					OnFailure:  "rework",
					ReworkStep: "step1",
				},
			},
		},
	}

	validator := &DAGValidator{}
	err := validator.ValidateDAG(pipeline)
	if err == nil {
		t.Fatal("Expected error for rework target being upstream dependency, got nil")
	}
	if got := err.Error(); !contains(got, "upstream dependency") {
		t.Errorf("Expected error about upstream dependency, got: %s", got)
	}
}

func TestValidateDAG_ReworkTargetDependsOnFailingStep(t *testing.T) {
	// step1 has rework_step: step2, but step2 depends on step1 → cycle
	pipeline := &Pipeline{
		Steps: []Step{
			{
				ID:      "step1",
				Persona: "agent1",
				Retry: RetryConfig{
					OnFailure:  "rework",
					ReworkStep: "step2",
				},
			},
			{
				ID:           "step2",
				Persona:      "agent2",
				Dependencies: []string{"step1"},
			},
		},
	}

	validator := &DAGValidator{}
	err := validator.ValidateDAG(pipeline)
	if err == nil {
		t.Fatal("Expected error for rework target depending on the failing step, got nil")
	}
	if got := err.Error(); !contains(got, "depends on step") {
		t.Errorf("Expected error about dependency cycle, got: %s", got)
	}
}

func TestValidateDAG_ReworkTargetSelfReference(t *testing.T) {
	pipeline := &Pipeline{
		Steps: []Step{
			{
				ID:      "step1",
				Persona: "agent1",
				Retry: RetryConfig{
					OnFailure:  "rework",
					ReworkStep: "step1",
				},
			},
		},
	}

	validator := &DAGValidator{}
	err := validator.ValidateDAG(pipeline)
	if err == nil {
		t.Fatal("Expected error for rework target being self, got nil")
	}
	if got := err.Error(); !contains(got, "cannot rework to itself") {
		t.Errorf("Expected error about self-rework, got: %s", got)
	}
}

func TestValidateDAG_ValidReworkTarget(t *testing.T) {
	// step1 reworks to fallback, which is an independent step — valid.
	pipeline := &Pipeline{
		Steps: []Step{
			{
				ID:      "step1",
				Persona: "agent1",
				Retry: RetryConfig{
					OnFailure:  "rework",
					ReworkStep: "fallback",
				},
			},
			{ID: "fallback", Persona: "agent2"},
		},
	}

	validator := &DAGValidator{}
	err := validator.ValidateDAG(pipeline)
	if err != nil {
		t.Errorf("Expected valid rework config, got: %v", err)
	}
}

func TestValidateDAG_RetryConfigValidation(t *testing.T) {
	// rework_step set without on_failure: rework should fail
	pipeline := &Pipeline{
		Steps: []Step{
			{
				ID:      "step1",
				Persona: "agent1",
				Retry: RetryConfig{
					OnFailure:  "fail",
					ReworkStep: "fallback",
				},
			},
			{ID: "fallback", Persona: "agent2"},
		},
	}

	validator := &DAGValidator{}
	err := validator.ValidateDAG(pipeline)
	if err == nil {
		t.Fatal("Expected error for rework_step set without on_failure: rework, got nil")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
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

func TestValidateDAG_DuplicateReworkTarget(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{ID: "step-1", Persona: "nav", Exec: ExecConfig{Source: "a"},
				Retry: RetryConfig{OnFailure: "rework", ReworkStep: "rework-step"}},
			{ID: "step-2", Persona: "nav", Exec: ExecConfig{Source: "b"},
				Retry: RetryConfig{OnFailure: "rework", ReworkStep: "rework-step"}},
			{ID: "rework-step", Persona: "nav", Exec: ExecConfig{Source: "fix"}},
		},
	}

	validator := &DAGValidator{}
	err := validator.ValidateDAG(p)
	if err == nil {
		t.Error("Expected error for duplicate rework target, got nil")
	}
	if err != nil && !contains(err.Error(), "is used by both") {
		t.Errorf("Expected 'is used by both' error, got: %v", err)
	}
}

func TestValidateDAG_InvalidOnFailureValue(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{ID: "step-1", Persona: "nav", Exec: ExecConfig{Source: "a"},
				Retry: RetryConfig{OnFailure: "invalid_value"}},
		},
	}

	validator := &DAGValidator{}
	err := validator.ValidateDAG(p)
	if err == nil {
		t.Error("Expected error for invalid on_failure value, got nil")
	}
	if err != nil && !contains(err.Error(), "invalid on_failure") {
		t.Errorf("Expected 'invalid on_failure' error, got: %v", err)
	}
}
