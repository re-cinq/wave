package pipeline

import (
	"fmt"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/skill"
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

func TestValidateDAG_ConcurrencyAndMatrixMutuallyExclusive(t *testing.T) {
	tests := []struct {
		name        string
		concurrency int
		strategy    *MatrixStrategy
		wantErr     bool
	}{
		{
			name:        "concurrency only is valid",
			concurrency: 3,
			strategy:    nil,
			wantErr:     false,
		},
		{
			name:        "matrix only is valid",
			concurrency: 0,
			strategy:    &MatrixStrategy{Type: "matrix", ItemsSource: "items.json", ItemKey: "items"},
			wantErr:     false,
		},
		{
			name:        "both concurrency and matrix is invalid",
			concurrency: 3,
			strategy:    &MatrixStrategy{Type: "matrix", ItemsSource: "items.json", ItemKey: "items"},
			wantErr:     true,
		},
		{
			name:        "concurrency=1 with matrix is valid (concurrency disabled)",
			concurrency: 1,
			strategy:    &MatrixStrategy{Type: "matrix", ItemsSource: "items.json", ItemKey: "items"},
			wantErr:     false,
		},
		{
			name:        "concurrency=0 with matrix is valid (concurrency disabled)",
			concurrency: 0,
			strategy:    &MatrixStrategy{Type: "matrix", ItemsSource: "items.json", ItemKey: "items"},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline := &Pipeline{
				Steps: []Step{
					{
						ID:          "step1",
						Persona:     "agent1",
						Concurrency: tt.concurrency,
						Strategy:    tt.strategy,
					},
				},
			}

			validator := &DAGValidator{}
			err := validator.ValidateDAG(pipeline)
			if tt.wantErr && err == nil {
				t.Error("expected error for concurrent + matrix, got nil")
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
			{ID: "step1", Persona: "agent1", ReworkOnly: true},
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
				ReworkOnly:   true,
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
	// step1 reworks to fallback, which is an independent rework-only step — valid.
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
			{ID: "fallback", Persona: "agent2", ReworkOnly: true},
		},
	}

	validator := &DAGValidator{}
	err := validator.ValidateDAG(pipeline)
	if err != nil {
		t.Errorf("Expected valid rework config, got: %v", err)
	}
}

func TestValidateDAG_ReworkTargetNotReworkOnly(t *testing.T) {
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
			{ID: "step2", Persona: "agent2"}, // not rework_only
		},
	}

	validator := &DAGValidator{}
	err := validator.ValidateDAG(pipeline)
	if err == nil {
		t.Fatal("Expected error for rework target without rework_only, got nil")
	}
	if got := err.Error(); !contains(got, "rework_only") {
		t.Errorf("Expected error about rework_only requirement, got: %s", got)
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
			{ID: "rework-step", Persona: "nav", ReworkOnly: true, Exec: ExecConfig{Source: "fix"}},
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

// mockSkillStore implements skill.Store for testing pipeline skill validation.
type mockSkillStore struct {
	skills map[string]skill.Skill
}

func (m *mockSkillStore) Read(name string) (skill.Skill, error) {
	if s, ok := m.skills[name]; ok {
		return s, nil
	}
	return skill.Skill{}, fmt.Errorf("%w: %s", skill.ErrNotFound, name)
}

func (m *mockSkillStore) Write(_ skill.Skill) error    { return nil }
func (m *mockSkillStore) List() ([]skill.Skill, error) { return nil, nil }
func (m *mockSkillStore) Delete(_ string) error        { return nil }

func Test_validatePipelineSkills(t *testing.T) {
	t.Run("valid skills pass validation", func(t *testing.T) {
		store := &mockSkillStore{skills: map[string]skill.Skill{
			"speckit": {Name: "speckit", Description: "Speckit skill"},
			"golang":  {Name: "golang", Description: "Go skill"},
		}}
		p := &Pipeline{
			Metadata: PipelineMetadata{Name: "test-pipeline"},
			Skills:   []string{"speckit", "golang"},
		}

		errs := validatePipelineSkills(p, store)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got: %v", errs)
		}
	})

	t.Run("invalid skill format produces error", func(t *testing.T) {
		store := &mockSkillStore{skills: map[string]skill.Skill{}}
		p := &Pipeline{
			Metadata: PipelineMetadata{Name: "test-pipeline"},
			Skills:   []string{"INVALID_NAME"},
		}

		errs := validatePipelineSkills(p, store)
		if len(errs) == 0 {
			t.Fatal("expected error for invalid skill name format, got none")
		}
		errMsg := errs[0].Error()
		if !strings.Contains(errMsg, "INVALID_NAME") {
			t.Errorf("expected error to mention invalid name, got: %s", errMsg)
		}
		if !strings.Contains(errMsg, "pipeline:test-pipeline") {
			t.Errorf("expected error to contain pipeline scope, got: %s", errMsg)
		}
	})

	t.Run("nonexistent skill produces scoped error", func(t *testing.T) {
		store := &mockSkillStore{skills: map[string]skill.Skill{
			"speckit": {Name: "speckit", Description: "Speckit skill"},
		}}
		p := &Pipeline{
			Metadata: PipelineMetadata{Name: "my-pipeline"},
			Skills:   []string{"nonexistent-skill"},
		}

		errs := validatePipelineSkills(p, store)
		if len(errs) == 0 {
			t.Fatal("expected error for nonexistent skill, got none")
		}
		errMsg := errs[0].Error()
		if !strings.Contains(errMsg, "nonexistent-skill") {
			t.Errorf("expected error to mention skill name, got: %s", errMsg)
		}
		if !strings.Contains(errMsg, "pipeline:my-pipeline") {
			t.Errorf("expected error to contain pipeline scope, got: %s", errMsg)
		}
	})

	t.Run("no skills field produces no error", func(t *testing.T) {
		store := &mockSkillStore{skills: map[string]skill.Skill{}}
		p := &Pipeline{
			Metadata: PipelineMetadata{Name: "test-pipeline"},
		}

		errs := validatePipelineSkills(p, store)
		if len(errs) != 0 {
			t.Errorf("expected no errors for pipeline without skills, got: %v", errs)
		}
	})
}

func TestValidateDAG_ThreadGroupWithDependencyChain(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{ID: "implement", Persona: "craftsman", Thread: "impl"},
			{ID: "fix", Persona: "craftsman", Thread: "impl", Dependencies: []string{"implement"}},
		},
	}

	v := &DAGValidator{}
	err := v.ValidateDAG(p)
	if err != nil {
		t.Errorf("Expected no error for valid thread group, got: %v", err)
	}
}

func TestValidateDAG_ThreadGroupWithoutDependencyChain(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{ID: "step-a", Persona: "craftsman", Thread: "impl"},
			{ID: "step-b", Persona: "craftsman", Thread: "impl"}, // no dependency on step-a
		},
	}

	v := &DAGValidator{}
	err := v.ValidateDAG(p)
	if err == nil {
		t.Error("Expected error for concurrent thread steps, got nil")
	}
	if err != nil && !contains(err.Error(), "no dependency on prior thread step") {
		t.Errorf("Expected 'no dependency' error, got: %v", err)
	}
}

func TestValidateDAG_InvalidFidelityValue(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{ID: "step-a", Persona: "craftsman", Thread: "impl", Fidelity: "invalid"},
		},
	}

	v := &DAGValidator{}
	err := v.ValidateDAG(p)
	if err == nil {
		t.Error("Expected error for invalid fidelity value, got nil")
	}
	if err != nil && !contains(err.Error(), "invalid fidelity value") {
		t.Errorf("Expected 'invalid fidelity' error, got: %v", err)
	}
}

func TestValidateDAG_ValidFidelityValues(t *testing.T) {
	for _, fidelity := range []string{"full", "compact", "summary", "fresh", ""} {
		p := &Pipeline{
			Steps: []Step{
				{ID: "step-a", Persona: "craftsman", Thread: "impl", Fidelity: fidelity},
			},
		}
		v := &DAGValidator{}
		err := v.ValidateDAG(p)
		if err != nil {
			t.Errorf("Fidelity %q should be valid, got error: %v", fidelity, err)
		}
	}
}

func TestValidateDAG_FidelityWithoutThread_Warning(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{ID: "step-a", Persona: "craftsman", Fidelity: "full"}, // no thread
		},
	}

	v := &DAGValidator{}
	err := v.ValidateDAG(p)
	if err != nil {
		t.Errorf("Fidelity without thread should be a warning, not an error, got: %v", err)
	}
	if len(v.Warnings) == 0 {
		t.Error("Expected a warning for fidelity without thread")
	}
	found := false
	for _, w := range v.Warnings {
		if contains(w, "fidelity has no effect") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'fidelity has no effect' warning, got: %v", v.Warnings)
	}
}

func TestValidateDAG_MixedPersonaThread_Warning(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{ID: "implement", Persona: "craftsman", Thread: "impl"},
			{ID: "review", Persona: "navigator", Thread: "impl", Dependencies: []string{"implement"}},
		},
	}

	v := &DAGValidator{}
	err := v.ValidateDAG(p)
	if err != nil {
		t.Errorf("Mixed-persona thread should not error, got: %v", err)
	}
	if len(v.Warnings) == 0 {
		t.Error("Expected a warning for mixed-persona thread group")
	}
	found := false
	for _, w := range v.Warnings {
		if contains(w, "different personas") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'different personas' warning, got: %v", v.Warnings)
	}
}

func TestValidateDAG_ThreadGroupThreeSteps(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{ID: "step-a", Persona: "craftsman", Thread: "impl"},
			{ID: "step-b", Persona: "craftsman", Thread: "impl", Dependencies: []string{"step-a"}},
			{ID: "step-c", Persona: "craftsman", Thread: "impl", Dependencies: []string{"step-b"}},
		},
	}

	v := &DAGValidator{}
	err := v.ValidateDAG(p)
	if err != nil {
		t.Errorf("Expected no error for valid 3-step thread chain, got: %v", err)
	}
}

func TestValidateDAG_StepWithoutThread_NoValidation(t *testing.T) {
	// Steps without thread fields should not trigger thread validation
	p := &Pipeline{
		Steps: []Step{
			{ID: "step-a", Persona: "craftsman"},
			{ID: "step-b", Persona: "navigator"}, // no dependency, no thread — fine
		},
	}

	v := &DAGValidator{}
	err := v.ValidateDAG(p)
	if err != nil {
		t.Errorf("Expected no error for steps without threads, got: %v", err)
	}
}

func TestValidateDAG_AgentReviewSelfPreventionAndReworkStep(t *testing.T) {
	t.Run("same persona rejected (self-review)", func(t *testing.T) {
		p := &Pipeline{
			Steps: []Step{
				{
					ID:      "impl",
					Persona: "craftsman",
					Handover: HandoverConfig{
						Contract: ContractConfig{
							Type:    "agent_review",
							Persona: "craftsman", // same as step persona
						},
					},
				},
				{ID: "rework", Persona: "craftsman", ReworkOnly: true},
			},
		}
		v := &DAGValidator{}
		err := v.ValidateDAG(p)
		if err == nil {
			t.Fatal("expected error for self-review, got nil")
		}
		if !strings.Contains(err.Error(), "self-review") {
			t.Errorf("error should mention self-review: %v", err)
		}
	})

	t.Run("different persona accepted", func(t *testing.T) {
		p := &Pipeline{
			Steps: []Step{
				{
					ID:      "impl",
					Persona: "craftsman",
					Handover: HandoverConfig{
						Contract: ContractConfig{
							Type:    "agent_review",
							Persona: "navigator",
						},
					},
				},
			},
		}
		v := &DAGValidator{}
		err := v.ValidateDAG(p)
		if err != nil {
			t.Errorf("expected no error for valid reviewer, got: %v", err)
		}
	})

	t.Run("non-agent_review contracts skipped for self-review check", func(t *testing.T) {
		p := &Pipeline{
			Steps: []Step{
				{
					ID:      "impl",
					Persona: "craftsman",
					Handover: HandoverConfig{
						Contract: ContractConfig{
							Type: "test_suite", // not agent_review
						},
					},
				},
			},
		}
		v := &DAGValidator{}
		err := v.ValidateDAG(p)
		if err != nil {
			t.Errorf("expected no error for non-agent_review contract, got: %v", err)
		}
	})
}
