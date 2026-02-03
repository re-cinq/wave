//go:build integration
// +build integration

// Integration tests for prototype implement phase.
// Run with: go test -tags=integration ./internal/pipeline/...

package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/event"
)

func TestPrototypeImplementPhaseInitialization(t *testing.T) {
	// Test that implement phase can be initialized and executed with dummy dependency
	tests := []struct {
		name          string
		hasDummyPhase bool
		expectError   bool
	}{
		{
			name:          "implement phase with completed dummy phase",
			hasDummyPhase: true,
			expectError:   false,
		},
		{
			name:          "implement phase without dummy phase",
			hasDummyPhase: false,
			expectError:   true, // Should fail due to missing dependency
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test manifest and load pipeline BEFORE changing directory
			testManifest := createPrototypeTestManifest()
			pipeline, err := loadTestPrototypePipeline()
			if err != nil {
				t.Fatalf("Failed to load prototype pipeline: %v", err)
			}

			// Change to project root for schema file access during execution
			originalWd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Chdir("../.."); err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(originalWd)

			// Setup test environment
			tempDir := t.TempDir()

			// Create mock adapter for testing
			mockAdapter := adapter.NewMockAdapter()

			// Create test executor
			emitter := event.NewNDJSONEmitter()
			executor := NewDefaultPipelineExecutor(mockAdapter,
				WithEmitter(emitter),
				WithDebug(true),
			)

			// Create test context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if tt.hasDummyPhase {
				// Create mock dummy phase artifacts in workspace
				dummyWorkspace := filepath.Join(tempDir, "dummy")
				err = os.MkdirAll(dummyWorkspace, 0755)
				if err != nil {
					t.Fatalf("Failed to create dummy workspace: %v", err)
				}

				// Create mock prototype directory
				prototypeDir := filepath.Join(dummyWorkspace, "prototype")
				err = os.MkdirAll(prototypeDir, 0755)
				if err != nil {
					t.Fatalf("Failed to create prototype directory: %v", err)
				}

				// Create a simple prototype file
				prototypeCode := `// Test prototype code
package main

import "fmt"

func main() {
    fmt.Println("Test feature prototype")
}
`
				err = os.WriteFile(filepath.Join(prototypeDir, "main.go"), []byte(prototypeCode), 0644)
				if err != nil {
					t.Fatalf("Failed to create prototype code: %v", err)
				}

				// Create mock interfaces.md file
				interfacesContent := `# Interface Definitions

## CLI Interface
- create: Create new items
- list: List all items
- get: Get item by ID
- delete: Delete item by ID

## Data Structures
- Item: {id, name, description, status}
`
				err = os.WriteFile(filepath.Join(dummyWorkspace, "interfaces.md"), []byte(interfacesContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create interfaces.md: %v", err)
				}

				// Create mock artifact.json for dummy phase
				dummyArtifact := `{
  "phase": "dummy",
  "artifacts": {
    "prototype": {"path": "prototype/", "exists": true, "content_type": "code"},
    "interface_definitions": {"path": "interfaces.md", "exists": true, "content_type": "markdown"}
  },
  "validation": {
    "runnable": true,
    "prototype_quality": "excellent"
  },
  "metadata": {
    "timestamp": "2026-02-03T12:00:00Z",
    "source_docs_path": "artifacts/feature-docs.md"
  }
}`
				err = os.WriteFile(filepath.Join(dummyWorkspace, "artifact.json"), []byte(dummyArtifact), 0644)
				if err != nil {
					t.Fatalf("Failed to create dummy artifact.json: %v", err)
				}
			}

			// Execute implement phase only (fourth step) - would need dummy phase first in real execution
			input := "Build a web application for team collaboration (implement phase test)"
			err = executor.Execute(ctx, pipeline, testManifest, input)

			if tt.expectError && err == nil {
				t.Error("Expected error for implement phase without dummy phase, but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for implement phase: %v", err)
			}

			// If no error expected and dummy phase exists, verify implement phase configuration
			if !tt.expectError && tt.hasDummyPhase {
				implementStep := findStepByID(pipeline, "implement")
				if implementStep == nil {
					t.Fatal("Implement step not found in pipeline")
				}

				// Verify dependency on dummy phase
				expectedDependencies := []string{"dummy"}
				if len(implementStep.Dependencies) != len(expectedDependencies) {
					t.Errorf("Expected %d dependencies, got %d", len(expectedDependencies), len(implementStep.Dependencies))
				}

				// Verify persona is correct
				if implementStep.Persona != "craftsman" {
					t.Errorf("Expected craftsman persona, got %s", implementStep.Persona)
				}
			}
		})
	}
}

func TestPrototypeImplementPhaseArtifactInjection(t *testing.T) {
	// Test that implement phase properly injects artifacts from all previous phases
	_ = createPrototypeTestManifest()
	pipeline, err := loadTestPrototypePipeline()
	if err != nil {
		t.Fatalf("Failed to load prototype pipeline: %v", err)
	}

	// Find implement step
	implementStep := findStepByID(pipeline, "implement")
	if implementStep == nil {
		t.Fatal("Implement step not found in pipeline")
	}

	// Verify artifact injection configuration
	if len(implementStep.Memory.InjectArtifacts) == 0 {
		t.Fatal("No artifact injection configured for implement phase")
	}

	// Verify artifacts from all previous phases are injected
	foundSpecArtifact := false
	foundDocsArtifact := false
	foundDummyArtifact := false

	for _, injection := range implementStep.Memory.InjectArtifacts {
		switch injection.Step {
		case "spec":
			if injection.Artifact == "spec" && injection.As == "spec.md" {
				foundSpecArtifact = true
			}
		case "docs":
			if injection.Artifact == "feature-docs" && injection.As == "feature-docs.md" {
				foundDocsArtifact = true
			}
		case "dummy":
			if injection.Artifact == "prototype" && injection.As == "prototype/" {
				foundDummyArtifact = true
			}
		}
	}

	if !foundSpecArtifact {
		t.Error("Spec artifact injection not properly configured")
	}

	if !foundDocsArtifact {
		t.Error("Docs artifact injection not properly configured")
	}

	if !foundDummyArtifact {
		t.Error("Dummy artifact injection not properly configured")
	}

	// Verify output artifacts configuration
	expectedArtifacts := map[string]string{
		"implementation-plan":   "implementation-plan.md",
		"progress-checklist":    "implementation-checklist.md",
		"contract_data":         "artifact.json",
	}

	if len(implementStep.OutputArtifacts) != len(expectedArtifacts) {
		t.Errorf("Expected %d output artifacts, got %d", len(expectedArtifacts), len(implementStep.OutputArtifacts))
	}

	for _, artifact := range implementStep.OutputArtifacts {
		expectedPath, exists := expectedArtifacts[artifact.Name]
		if !exists {
			t.Errorf("Unexpected artifact: %s", artifact.Name)
			continue
		}

		if artifact.Path != expectedPath {
			t.Errorf("Expected path %s for artifact %s, got %s", expectedPath, artifact.Name, artifact.Path)
		}
	}
}

func TestPrototypeImplementPhaseContractValidation(t *testing.T) {
	// Test that implement phase has proper contract validation configured
	pipeline, err := loadTestPrototypePipeline()
	if err != nil {
		t.Fatalf("Failed to load prototype pipeline: %v", err)
	}

	implementStep := findStepByID(pipeline, "implement")
	if implementStep == nil {
		t.Fatal("Implement step not found in pipeline")
	}

	// Verify contract configuration
	if implementStep.Handover.Contract.Type != "json_schema" {
		t.Errorf("Expected json_schema contract, got %s", implementStep.Handover.Contract.Type)
	}

	expectedSchemaPath := ".wave/contracts/implement-phase.schema.json"
	if implementStep.Handover.Contract.SchemaPath != expectedSchemaPath {
		t.Errorf("Expected schema path %s, got %s", expectedSchemaPath, implementStep.Handover.Contract.SchemaPath)
	}

	if !implementStep.Handover.Contract.MustPass {
		t.Error("Contract validation should be required (must_pass: true)")
	}

	if implementStep.Handover.Contract.MaxRetries <= 0 {
		t.Error("Contract should allow retries for robustness")
	}
}

func TestPrototypeImplementPhaseWorkspaceConfiguration(t *testing.T) {
	// Test that implement phase has proper workspace configuration
	pipeline, err := loadTestPrototypePipeline()
	if err != nil {
		t.Fatalf("Failed to load prototype pipeline: %v", err)
	}

	implementStep := findStepByID(pipeline, "implement")
	if implementStep == nil {
		t.Fatal("Implement step not found in pipeline")
	}

	// Verify workspace mount configuration
	if len(implementStep.Workspace.Mount) == 0 {
		t.Error("Implement step has no workspace mounts configured")
	}

	// Verify readwrite access for implementation
	hasReadWriteMount := false
	for _, mount := range implementStep.Workspace.Mount {
		if mount.Mode == "readwrite" {
			hasReadWriteMount = true
			break
		}
	}

	if !hasReadWriteMount {
		t.Error("Implement step requires readwrite workspace mount for implementation")
	}
}

func TestPrototypeImplementPhaseImplementationGoals(t *testing.T) {
	// Test implementation phase goals and requirements
	pipeline, err := loadTestPrototypePipeline()
	if err != nil {
		t.Fatalf("Failed to load prototype pipeline: %v", err)
	}

	implementStep := findStepByID(pipeline, "implement")
	if implementStep == nil {
		t.Fatal("Implement step not found in pipeline")
	}

	// Verify that exec source mentions implementation goals
	if implementStep.Exec.Source == "" {
		t.Error("Implement step has no execution source defined")
	}

	// Check that prompt mentions key implementation requirements
	promptContent := implementStep.Exec.Source
	requiredMentions := []string{
		"implementation",
		"production",
		"test",
		"plan",
		"checklist",
	}

	for _, mention := range requiredMentions {
		if !containsIgnoreCase(promptContent, mention) {
			t.Errorf("Implement step prompt should mention '%s' for proper implementation guidance", mention)
		}
	}

	// Verify that all previous phase artifacts are referenced
	artifactReferences := []string{
		"spec.md",
		"feature-docs.md",
		"prototype",
	}

	for _, reference := range artifactReferences {
		if !containsIgnoreCase(promptContent, reference) {
			t.Errorf("Implement step prompt should reference '%s' artifact", reference)
		}
	}
}

func TestPrototypeImplementPhaseEndToEnd(t *testing.T) {
	// End-to-end test for the implement phase configuration
	pipeline, err := loadTestPrototypePipeline()
	if err != nil {
		t.Fatalf("Failed to load prototype pipeline: %v", err)
	}

	implementStep := findStepByID(pipeline, "implement")
	if implementStep == nil {
		t.Fatal("Implement step not found in pipeline")
	}

	// Verify the implement step is the fourth step in the sequence
	stepIndex := -1
	for i, step := range pipeline.Steps {
		if step.ID == "implement" {
			stepIndex = i
			break
		}
	}

	if stepIndex == -1 {
		t.Fatal("Implement step not found in pipeline steps")
	}

	// Verify proper sequencing (after spec, docs, dummy)
	expectedPreviousSteps := []string{"spec", "docs", "dummy"}
	if stepIndex < len(expectedPreviousSteps) {
		t.Errorf("Implement step is at index %d, but should come after %v", stepIndex, expectedPreviousSteps)
	}

	// Verify that spec, docs, and dummy steps exist before implement
	for i := 0; i < stepIndex; i++ {
		stepID := pipeline.Steps[i].ID
		found := false
		for _, expectedStep := range expectedPreviousSteps {
			if stepID == expectedStep {
				found = true
				break
			}
		}
		if !found && i < len(expectedPreviousSteps) {
			t.Errorf("Expected step %s at position %d, found %s", expectedPreviousSteps[i], i, stepID)
		}
	}
}