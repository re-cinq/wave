//go:build integration
// +build integration

// Integration tests for prototype dummy phase.
// Run with: go test -tags=integration ./internal/pipeline/...

package pipeline

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/event"
)

func TestPrototypeDummyPhaseInitialization(t *testing.T) {
	// Test that dummy phase can be initialized and executed
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "dummy phase with completed docs phase",
			input:       "Build a web application for team collaboration (dummy phase test)",
			expectError: false,
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

			// Trim to dummy step only
			dummyStep := findStepByID(pipeline, "dummy")
			if dummyStep == nil {
				t.Fatal("Dummy step not found in pipeline")
			}
			dummyOnly := *dummyStep
			dummyOnly.Dependencies = nil
			pipeline.Steps = []Step{dummyOnly}

			// Change to project root for schema file access during execution
			originalWd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Chdir("../.."); err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(originalWd)

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

			err = executor.Execute(ctx, pipeline, testManifest, tt.input)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for input %q, but got none", tt.input)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for input %q: %v", tt.input, err)
			}
		})
	}
}

func TestPrototypeDummyPhaseConfiguration(t *testing.T) {
	// Verify dummy phase has correct dependencies and persona
	pipeline, err := loadTestPrototypePipeline()
	if err != nil {
		t.Fatalf("Failed to load prototype pipeline: %v", err)
	}

	dummyStep := findStepByID(pipeline, "dummy")
	if dummyStep == nil {
		t.Fatal("Dummy step not found in pipeline")
	}

	// Verify dependency on docs phase
	expectedDependencies := []string{"docs"}
	if len(dummyStep.Dependencies) != len(expectedDependencies) {
		t.Errorf("Expected %d dependencies, got %d", len(expectedDependencies), len(dummyStep.Dependencies))
	}

	// Verify persona is correct
	if dummyStep.Persona != "craftsman" {
		t.Errorf("Expected craftsman persona, got %s", dummyStep.Persona)
	}
}

func TestPrototypeDummyPhaseArtifactInjection(t *testing.T) {
	// Test that dummy phase properly injects artifacts from docs and spec phases
	_ = createPrototypeTestManifest()
	pipeline, err := loadTestPrototypePipeline()
	if err != nil {
		t.Fatalf("Failed to load prototype pipeline: %v", err)
	}

	// Find dummy step
	dummyStep := findStepByID(pipeline, "dummy")
	if dummyStep == nil {
		t.Fatal("Dummy step not found in pipeline")
	}

	// Verify artifact injection configuration
	if len(dummyStep.Memory.InjectArtifacts) == 0 {
		t.Fatal("No artifact injection configured for dummy phase")
	}

	// Verify docs artifact injection
	foundDocsArtifact := false
	foundSpecArtifact := false
	for _, injection := range dummyStep.Memory.InjectArtifacts {
		if injection.Step == "docs" && injection.Artifact == "feature-docs" && injection.As == "feature-docs.md" {
			foundDocsArtifact = true
		}
		if injection.Step == "spec" && injection.Artifact == "spec" && injection.As == "spec.md" {
			foundSpecArtifact = true
		}
	}

	if !foundDocsArtifact {
		t.Error("Docs artifact injection not properly configured")
	}

	if !foundSpecArtifact {
		t.Error("Spec artifact injection not properly configured")
	}

	// Verify output artifacts configuration
	expectedArtifacts := map[string]string{
		"prototype":            "prototype/",
		"interface-definitions": "interfaces.md",
		"contract_data":        "artifact.json",
	}

	if len(dummyStep.OutputArtifacts) != len(expectedArtifacts) {
		t.Errorf("Expected %d output artifacts, got %d", len(expectedArtifacts), len(dummyStep.OutputArtifacts))
	}

	for _, artifact := range dummyStep.OutputArtifacts {
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

func TestPrototypeDummyPhaseContractValidation(t *testing.T) {
	// Test that dummy phase has proper contract validation configured
	pipeline, err := loadTestPrototypePipeline()
	if err != nil {
		t.Fatalf("Failed to load prototype pipeline: %v", err)
	}

	dummyStep := findStepByID(pipeline, "dummy")
	if dummyStep == nil {
		t.Fatal("Dummy step not found in pipeline")
	}

	// Verify contract configuration
	if dummyStep.Handover.Contract.Type != "json_schema" {
		t.Errorf("Expected json_schema contract, got %s", dummyStep.Handover.Contract.Type)
	}

	expectedSchemaPath := ".wave/contracts/dummy-phase.schema.json"
	if dummyStep.Handover.Contract.SchemaPath != expectedSchemaPath {
		t.Errorf("Expected schema path %s, got %s", expectedSchemaPath, dummyStep.Handover.Contract.SchemaPath)
	}

	if !dummyStep.Handover.Contract.MustPass {
		t.Error("Contract validation should be required (must_pass: true)")
	}

	if dummyStep.Handover.Contract.MaxRetries <= 0 {
		t.Error("Contract should allow retries for robustness")
	}
}

func TestPrototypeDummyPhaseWorkspaceConfiguration(t *testing.T) {
	// Test that dummy phase has proper workspace configuration
	pipeline, err := loadTestPrototypePipeline()
	if err != nil {
		t.Fatalf("Failed to load prototype pipeline: %v", err)
	}

	dummyStep := findStepByID(pipeline, "dummy")
	if dummyStep == nil {
		t.Fatal("Dummy step not found in pipeline")
	}

	// Verify workspace mount configuration
	if len(dummyStep.Workspace.Mount) == 0 {
		t.Error("Dummy step has no workspace mounts configured")
	}

	// Verify readwrite access for prototype code generation
	hasReadWriteMount := false
	for _, mount := range dummyStep.Workspace.Mount {
		if mount.Mode == "readwrite" {
			hasReadWriteMount = true
			break
		}
	}

	if !hasReadWriteMount {
		t.Error("Dummy step requires readwrite workspace mount for code generation")
	}
}

func TestPrototypeDummyPhasePrototypeGeneration(t *testing.T) {
	// Test prototype generation expectations
	pipeline, err := loadTestPrototypePipeline()
	if err != nil {
		t.Fatalf("Failed to load prototype pipeline: %v", err)
	}

	dummyStep := findStepByID(pipeline, "dummy")
	if dummyStep == nil {
		t.Fatal("Dummy step not found in pipeline")
	}

	// Verify that exec source mentions prototype generation
	if dummyStep.Exec.Source == "" {
		t.Error("Dummy step has no execution source defined")
	}

	// Check that prompt mentions key prototype requirements
	promptContent := dummyStep.Exec.Source
	requiredMentions := []string{
		"prototype",
		"interfaces",
		"stub",
		"working",
		"runnable",
	}

	for _, mention := range requiredMentions {
		if !containsIgnoreCase(promptContent, mention) {
			t.Errorf("Dummy step prompt should mention '%s' for proper prototype generation", mention)
		}
	}
}

// Helper function for case-insensitive string checking
func containsIgnoreCase(text, substr string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(substr))
}

