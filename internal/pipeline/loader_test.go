package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// =============================================================================
// T036-T038: Named Pipeline Integration Tests
// Tests for loading and validating pipeline YAML files
// =============================================================================

// TestFailureModesValidationPipelineLoads verifies that failure-modes-validation.yaml
// can be loaded from YAML without errors and has valid step dependencies.
func TestFailureModesValidationPipelineLoads(t *testing.T) {
	pipelinePath := findPipelineFile(t, "failure-modes-validation.yaml")

	loader := &YAMLPipelineLoader{}
	p, err := loader.Load(pipelinePath)
	if err != nil {
		t.Fatalf("Failed to load failure-modes-validation.yaml: %v", err)
	}

	// Verify pipeline metadata
	if p.Metadata.Name != "failure-modes-validation" {
		t.Errorf("Expected pipeline name 'failure-modes-validation', got '%s'", p.Metadata.Name)
	}

	// Verify we have the expected steps
	expectedSteps := []string{"contract-validation", "timeout-test", "artifact-handling"}
	if len(p.Steps) != len(expectedSteps) {
		t.Errorf("Expected %d steps, got %d", len(expectedSteps), len(p.Steps))
	}

	stepIDs := make(map[string]bool)
	for _, step := range p.Steps {
		stepIDs[step.ID] = true
	}

	for _, expected := range expectedSteps {
		if !stepIDs[expected] {
			t.Errorf("Missing expected step '%s'", expected)
		}
	}

	// Validate DAG - dependencies should be valid
	validator := &DAGValidator{}
	if err := validator.ValidateDAG(p); err != nil {
		t.Errorf("DAG validation failed: %v", err)
	}

	// Verify topological sort works
	sorted, err := validator.TopologicalSort(p)
	if err != nil {
		t.Errorf("Topological sort failed: %v", err)
	}

	// Verify execution order: contract-validation must come before timeout-test
	// and timeout-test must come before artifact-handling
	order := make(map[string]int)
	for i, step := range sorted {
		order[step.ID] = i
	}

	if order["contract-validation"] >= order["timeout-test"] {
		t.Error("contract-validation should execute before timeout-test")
	}
	if order["timeout-test"] >= order["artifact-handling"] {
		t.Error("timeout-test should execute before artifact-handling")
	}
}

// TestContractValidationTestPipelineLoads verifies that contract-validation-test.yaml
// can be loaded and is designed to fail contract validation.
func TestContractValidationTestPipelineLoads(t *testing.T) {
	pipelinePath := findPipelineFile(t, "contract-validation-test.yaml")

	loader := &YAMLPipelineLoader{}
	p, err := loader.Load(pipelinePath)
	if err != nil {
		t.Fatalf("Failed to load contract-validation-test.yaml: %v", err)
	}

	// Verify pipeline metadata
	if p.Metadata.Name != "contract-validation-test" {
		t.Errorf("Expected pipeline name 'contract-validation-test', got '%s'", p.Metadata.Name)
	}

	// This pipeline should have exactly one step
	if len(p.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(p.Steps))
	}

	// Verify the step has a contract configuration
	step := p.Steps[0]
	if step.ID != "produce-invalid-json" {
		t.Errorf("Expected step ID 'produce-invalid-json', got '%s'", step.ID)
	}

	// Verify contract is configured
	if step.Handover.Contract.Type != "json_schema" {
		t.Errorf("Expected contract type 'json_schema', got '%s'", step.Handover.Contract.Type)
	}

	// Verify on_failure is set to fail (not retry) for predictable failure
	if step.Handover.Contract.OnFailure != "fail" {
		t.Errorf("Expected on_failure 'fail', got '%s'", step.Handover.Contract.OnFailure)
	}

	// Verify max_retries is 0 to ensure immediate failure
	if step.Handover.Contract.MaxRetries != 0 {
		t.Errorf("Expected max_retries 0, got %d", step.Handover.Contract.MaxRetries)
	}
}

// TestAllPipelinesLoadCorrectly verifies that all pipeline YAML files in the
// .wave/pipelines directory can be loaded without errors.
func TestAllPipelinesLoadCorrectly(t *testing.T) {
	pipelinesDir := findPipelinesDir(t)

	entries, err := os.ReadDir(pipelinesDir)
	if err != nil {
		t.Fatalf("Failed to read pipelines directory: %v", err)
	}

	loader := &YAMLPipelineLoader{}
	validator := &DAGValidator{}

	var loadedCount int
	var failedPipelines []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}

		pipelinePath := filepath.Join(pipelinesDir, entry.Name())
		t.Run(entry.Name(), func(t *testing.T) {
			p, err := loader.Load(pipelinePath)
			if err != nil {
				t.Errorf("Failed to load pipeline %s: %v", entry.Name(), err)
				failedPipelines = append(failedPipelines, entry.Name())
				return
			}

			// Verify basic structure
			if p.Metadata.Name == "" {
				t.Errorf("Pipeline %s has empty metadata.name", entry.Name())
			}

			if len(p.Steps) == 0 {
				t.Errorf("Pipeline %s has no steps", entry.Name())
			}

			// Validate DAG
			if err := validator.ValidateDAG(p); err != nil {
				t.Errorf("Pipeline %s has invalid DAG: %v", entry.Name(), err)
				failedPipelines = append(failedPipelines, entry.Name())
				return
			}

			loadedCount++
		})
	}

	if len(failedPipelines) > 0 {
		t.Errorf("Failed to load %d pipelines: %v", len(failedPipelines), failedPipelines)
	}

	t.Logf("Successfully loaded and validated %d pipelines", loadedCount)
}

// TestPipelineStepDependenciesAreValid tests that all step dependencies in
// pipelines reference existing steps.
func TestPipelineStepDependenciesAreValid(t *testing.T) {
	testCases := []struct {
		name          string
		pipelineName  string
		expectedDeps  map[string][]string // step ID -> expected dependencies
	}{
		{
			name:         "failure-modes-validation dependencies",
			pipelineName: "failure-modes-validation.yaml",
			expectedDeps: map[string][]string{
				"contract-validation": {},
				"timeout-test":        {"contract-validation"},
				"artifact-handling":   {"timeout-test"},
			},
		},
		{
			name:         "contract-validation-test dependencies",
			pipelineName: "contract-validation-test.yaml",
			expectedDeps: map[string][]string{
				"produce-invalid-json": {},
			},
		},
	}

	loader := &YAMLPipelineLoader{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pipelinePath := findPipelineFile(t, tc.pipelineName)
			p, err := loader.Load(pipelinePath)
			if err != nil {
				t.Fatalf("Failed to load pipeline: %v", err)
			}

			for _, step := range p.Steps {
				expected, ok := tc.expectedDeps[step.ID]
				if !ok {
					t.Errorf("Unexpected step ID '%s'", step.ID)
					continue
				}

				if len(step.Dependencies) != len(expected) {
					t.Errorf("Step %s: expected %d dependencies, got %d",
						step.ID, len(expected), len(step.Dependencies))
					continue
				}

				for i, dep := range step.Dependencies {
					if dep != expected[i] {
						t.Errorf("Step %s: expected dependency '%s', got '%s'",
							step.ID, expected[i], dep)
					}
				}
			}
		})
	}
}

// TestPipelineLoaderFailsOnInvalidYAML verifies that the loader returns
// appropriate errors for invalid YAML content.
func TestPipelineLoaderFailsOnInvalidYAML(t *testing.T) {
	testCases := []struct {
		name        string
		yamlContent string
		expectError bool
	}{
		{
			name: "valid minimal pipeline",
			yamlContent: `
kind: WavePipeline
metadata:
  name: test
steps:
  - id: step1
    persona: agent1
    exec:
      type: prompt
      source: test
`,
			expectError: false,
		},
		{
			name: "invalid yaml syntax",
			yamlContent: `
kind: WavePipeline
metadata:
  name: test
steps:
  - id: [unclosed bracket
`,
			expectError: true,
		},
		{
			name: "malformed indentation",
			yamlContent: `
kind: WavePipeline
metadata:
name: test
  description: wrong indent
`,
			expectError: true,
		},
	}

	loader := &YAMLPipelineLoader{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := loader.Unmarshal([]byte(tc.yamlContent))

			if tc.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestPipelineWithInvalidDependenciesFailsValidation verifies that pipelines
// with invalid dependencies are caught during DAG validation.
func TestPipelineWithInvalidDependenciesFailsValidation(t *testing.T) {
	testCases := []struct {
		name           string
		steps          []Step
		expectError    bool
		errorSubstring string
	}{
		{
			name: "valid dependencies",
			steps: []Step{
				{ID: "a", Persona: "agent1", Dependencies: []string{}},
				{ID: "b", Persona: "agent2", Dependencies: []string{"a"}},
			},
			expectError: false,
		},
		{
			name: "missing dependency",
			steps: []Step{
				{ID: "a", Persona: "agent1", Dependencies: []string{"nonexistent"}},
			},
			expectError:    true,
			errorSubstring: "non-existent",
		},
		{
			name: "circular dependency",
			steps: []Step{
				{ID: "a", Persona: "agent1", Dependencies: []string{"b"}},
				{ID: "b", Persona: "agent2", Dependencies: []string{"a"}},
			},
			expectError:    true,
			errorSubstring: "cycle",
		},
		{
			name: "self-reference",
			steps: []Step{
				{ID: "a", Persona: "agent1", Dependencies: []string{"a"}},
			},
			expectError:    true,
			errorSubstring: "cycle",
		},
	}

	validator := &DAGValidator{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := &Pipeline{
				Metadata: PipelineMetadata{Name: "test-pipeline"},
				Steps:    tc.steps,
			}

			err := validator.ValidateDAG(p)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
					return
				}
				if tc.errorSubstring != "" && !strings.Contains(err.Error(), tc.errorSubstring) {
					t.Errorf("Error should contain '%s', got: %v", tc.errorSubstring, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestNewPipelinesHaveValidContracts verifies that the new failure mode test
// pipelines have properly configured contracts.
func TestNewPipelinesHaveValidContracts(t *testing.T) {
	testCases := []struct {
		pipelineName     string
		stepID           string
		expectedContract ContractConfig
	}{
		{
			pipelineName: "failure-modes-validation.yaml",
			stepID:       "contract-validation",
			expectedContract: ContractConfig{
				Type:       "json_schema",
				OnFailure:  "fail",
				MaxRetries: 1,
			},
		},
		{
			pipelineName: "failure-modes-validation.yaml",
			stepID:       "artifact-handling",
			expectedContract: ContractConfig{
				Type:       "json_schema",
				OnFailure:  "fail",
				MaxRetries: 1,
			},
		},
		{
			pipelineName: "contract-validation-test.yaml",
			stepID:       "produce-invalid-json",
			expectedContract: ContractConfig{
				Type:       "json_schema",
				OnFailure:  "fail",
				MaxRetries: 0,
			},
		},
	}

	loader := &YAMLPipelineLoader{}

	for _, tc := range testCases {
		t.Run(tc.pipelineName+"/"+tc.stepID, func(t *testing.T) {
			pipelinePath := findPipelineFile(t, tc.pipelineName)
			p, err := loader.Load(pipelinePath)
			if err != nil {
				t.Fatalf("Failed to load pipeline: %v", err)
			}

			var step *Step
			for i := range p.Steps {
				if p.Steps[i].ID == tc.stepID {
					step = &p.Steps[i]
					break
				}
			}

			if step == nil {
				t.Fatalf("Step '%s' not found in pipeline", tc.stepID)
			}

			contract := step.Handover.Contract

			if contract.Type != tc.expectedContract.Type {
				t.Errorf("Expected contract type '%s', got '%s'",
					tc.expectedContract.Type, contract.Type)
			}

			if contract.OnFailure != tc.expectedContract.OnFailure {
				t.Errorf("Expected on_failure '%s', got '%s'",
					tc.expectedContract.OnFailure, contract.OnFailure)
			}

			if contract.MaxRetries != tc.expectedContract.MaxRetries {
				t.Errorf("Expected max_retries %d, got %d",
					tc.expectedContract.MaxRetries, contract.MaxRetries)
			}
		})
	}
}

// =============================================================================
// Helper functions
// =============================================================================

// findPipelinesDir locates the .wave/pipelines directory by checking common locations.
func findPipelinesDir(t *testing.T) string {
	t.Helper()

	candidates := []string{
		".wave/pipelines",
		"../.wave/pipelines",
		"../../.wave/pipelines",
		"../../../.wave/pipelines",
		"../../../../.wave/pipelines",
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			absPath, err := filepath.Abs(candidate)
			if err != nil {
				continue
			}
			return absPath
		}
	}

	t.Skip("Could not find .wave/pipelines directory")
	return ""
}

// findPipelineFile locates a specific pipeline YAML file.
func findPipelineFile(t *testing.T, name string) string {
	t.Helper()

	pipelinesDir := findPipelinesDir(t)
	pipelinePath := filepath.Join(pipelinesDir, name)

	if _, err := os.Stat(pipelinePath); err != nil {
		t.Skipf("Pipeline file %s not found: %v", name, err)
	}

	return pipelinePath
}
