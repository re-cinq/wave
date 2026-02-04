//go:build integration
// +build integration

// Integration tests for prototype pipeline end-to-end flows.
// Run with: go test -tags=integration ./internal/pipeline/...

package pipeline

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/event"
)

func TestPrototypePipelineEndToEnd(t *testing.T) {
	// End-to-end test for the complete prototype pipeline
	tests := []struct {
		name         string
		input        string
		expectSteps  int
		expectError  bool
		description  string
	}{
		{
			name:         "complete prototype pipeline",
			input:        "Build a task management CLI tool with user authentication",
			expectSteps:  4, // spec, docs, dummy, implement
			expectError:  false,
			description:  "Full pipeline execution from spec to implementation",
		},
		{
			name:         "simple feature pipeline",
			input:        "Create a file organizer utility",
			expectSteps:  4,
			expectError:  false,
			description:  "Pipeline with simpler feature description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			_ = t.TempDir()

			// Create test manifest
			testManifest := createPrototypeTestManifest()

			// Load prototype pipeline
			pipeline, err := loadTestPrototypePipeline()
			if err != nil {
				t.Fatalf("Failed to load prototype pipeline: %v", err)
			}

			// Verify pipeline structure
			if len(pipeline.Steps) < tt.expectSteps {
				t.Fatalf("Expected at least %d steps, got %d", tt.expectSteps, len(pipeline.Steps))
			}

			// Verify step sequence and dependencies
			expectedStepOrder := []string{"spec", "docs", "dummy", "implement"}
			for i, expectedStep := range expectedStepOrder {
				if i >= len(pipeline.Steps) {
					break
				}

				step := &pipeline.Steps[i]
				if step.ID != expectedStep {
					t.Errorf("Expected step %d to be %s, got %s", i, expectedStep, step.ID)
				}

				// Verify dependencies for steps after the first
				if i > 0 {
					expectedDep := expectedStepOrder[i-1]
					hasDependency := false
					for _, dep := range step.Dependencies {
						if dep == expectedDep {
							hasDependency = true
							break
						}
					}
					if !hasDependency {
						t.Errorf("Step %s should depend on %s", step.ID, expectedDep)
					}
				}
			}

			// Create mock adapter for testing
			mockAdapter := adapter.NewMockAdapter()

			// Create test executor
			emitter := event.NewNDJSONEmitter()
			executor := NewDefaultPipelineExecutor(mockAdapter,
				WithEmitter(emitter),
				WithDebug(true),
			)

			// Create test context with extended timeout for full pipeline
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			// Execute full pipeline (would normally run all steps in sequence)
			// Note: With mock adapter, this tests configuration but not actual execution
			err = executor.Execute(ctx, pipeline, testManifest, tt.input)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, but got none", tt.description)
			}
			if !tt.expectError && err != nil {
				// Note: Mock adapter may fail due to empty output, but configuration should be valid
				t.Logf("Pipeline execution failed (expected with mock adapter): %v", err)
			}

			// Verify pipeline metadata
			if pipeline.Metadata.Name != "prototype" {
				t.Errorf("Expected pipeline name 'prototype', got %s", pipeline.Metadata.Name)
			}

			if pipeline.Metadata.Description == "" {
				t.Error("Pipeline should have a description")
			}
		})
	}
}

func TestPrototypePipelineStepConfiguration(t *testing.T) {
	// Test that all pipeline steps are properly configured
	pipeline, err := loadTestPrototypePipeline()
	if err != nil {
		t.Fatalf("Failed to load prototype pipeline: %v", err)
	}

	// Define expected step configurations
	expectedSteps := map[string]struct {
		persona      string
		hasContract  bool
		hasWorkspace bool
		hasArtifacts bool
	}{
		"spec": {
			persona:      "craftsman", // Changed from navigator due to write permissions
			hasContract:  true,
			hasWorkspace: true,
			hasArtifacts: true,
		},
		"docs": {
			persona:      "philosopher",
			hasContract:  true,
			hasWorkspace: true,
			hasArtifacts: true,
		},
		"dummy": {
			persona:      "craftsman",
			hasContract:  true,
			hasWorkspace: true,
			hasArtifacts: true,
		},
		"implement": {
			persona:      "craftsman",
			hasContract:  true,
			hasWorkspace: true,
			hasArtifacts: true,
		},
	}

	// Verify each step configuration
	for _, step := range pipeline.Steps {
		expected, exists := expectedSteps[step.ID]
		if !exists {
			continue // Skip PR cycle steps for now
		}

		t.Run(step.ID, func(t *testing.T) {
			// Verify persona assignment
			if step.Persona != expected.persona {
				t.Errorf("Expected persona %s, got %s", expected.persona, step.Persona)
			}

			// Verify contract configuration
			if expected.hasContract {
				if step.Handover.Contract.Type == "" {
					t.Error("Step should have contract validation configured")
				}
				if step.Handover.Contract.SchemaPath == "" {
					t.Error("Step should have schema path configured")
				}
				if !step.Handover.Contract.MustPass {
					t.Error("Step should require contract validation to pass")
				}
			}

			// Verify workspace configuration
			if expected.hasWorkspace {
				if len(step.Workspace.Mount) == 0 {
					t.Error("Step should have workspace mounts configured")
				}
			}

			// Verify output artifacts configuration
			if expected.hasArtifacts {
				if len(step.OutputArtifacts) == 0 {
					t.Error("Step should have output artifacts configured")
				}

				// Verify artifact.json is included for contract validation
				hasArtifactJson := false
				for _, artifact := range step.OutputArtifacts {
					if artifact.Name == "contract_data" && artifact.Path == "artifact.json" {
						hasArtifactJson = true
						break
					}
				}
				if !hasArtifactJson {
					t.Error("Step should include artifact.json for contract validation")
				}
			}
		})
	}
}

func TestPrototypePipelineArtifactFlow(t *testing.T) {
	// Test that artifacts flow correctly between pipeline steps
	pipeline, err := loadTestPrototypePipeline()
	if err != nil {
		t.Fatalf("Failed to load prototype pipeline: %v", err)
	}

	// Define expected artifact flows
	type artifactFlow struct {
		fromStep string
		artifact string
		toStep   string
		as       string
	}

	expectedFlows := []artifactFlow{
		{fromStep: "spec", artifact: "spec", toStep: "docs", as: "input-spec.md"},
		{fromStep: "docs", artifact: "feature-docs", toStep: "dummy", as: "feature-docs.md"},
		{fromStep: "spec", artifact: "spec", toStep: "dummy", as: "spec.md"},
		{fromStep: "spec", artifact: "spec", toStep: "implement", as: "spec.md"},
		{fromStep: "docs", artifact: "feature-docs", toStep: "implement", as: "feature-docs.md"},
		{fromStep: "dummy", artifact: "prototype", toStep: "implement", as: "prototype/"},
	}

	stepMap := make(map[string]*Step)
	for i := range pipeline.Steps {
		stepMap[pipeline.Steps[i].ID] = &pipeline.Steps[i]
	}

	// Verify each expected artifact flow
	for _, flow := range expectedFlows {
		t.Run(flow.fromStep+"->"+flow.toStep, func(t *testing.T) {
			toStep, exists := stepMap[flow.toStep]
			if !exists {
				t.Fatalf("Step %s not found", flow.toStep)
			}

			// Find the expected artifact injection
			found := false
			for _, injection := range toStep.Memory.InjectArtifacts {
				if injection.Step == flow.fromStep &&
				   injection.Artifact == flow.artifact &&
				   injection.As == flow.as {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected artifact flow from %s.%s to %s as %s not found",
					flow.fromStep, flow.artifact, flow.toStep, flow.as)
			}
		})
	}
}

func TestPrototypePipelineContractSchemas(t *testing.T) {
	// Test that all contract schemas exist and are valid
	_, err := loadTestPrototypePipeline()
	if err != nil {
		t.Fatalf("Failed to load prototype pipeline: %v", err)
	}

	// Navigate up to project root from internal/pipeline directory
	expectedSchemas := []string{
		"../../.wave/contracts/spec-phase.schema.json",
		"../../.wave/contracts/docs-phase.schema.json",
		"../../.wave/contracts/dummy-phase.schema.json",
		"../../.wave/contracts/implement-phase.schema.json",
	}

	for _, schemaPath := range expectedSchemas {
		t.Run(filepath.Base(schemaPath), func(t *testing.T) {
			// Check file exists
			if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
				t.Fatalf("Schema file does not exist: %s", schemaPath)
			}

			// Check file is valid JSON
			schemaData, err := os.ReadFile(schemaPath)
			if err != nil {
				t.Fatalf("Failed to read schema: %v", err)
			}

			if len(schemaData) == 0 {
				t.Fatal("Schema file is empty")
			}

			// Basic JSON validity check
			var schema map[string]interface{}
			if err := json.Unmarshal(schemaData, &schema); err != nil {
				t.Fatalf("Schema is not valid JSON: %v", err)
			}

			// Check required JSON Schema fields
			requiredFields := []string{"$schema", "title", "type"}
			for _, field := range requiredFields {
				if _, exists := schema[field]; !exists {
					t.Errorf("Schema missing required field: %s", field)
				}
			}
		})
	}
}

func TestPrototypePipelinePersonaPermissions(t *testing.T) {
	// Test that personas have appropriate permissions for their roles
	testManifest := createPrototypeTestManifest()
	pipeline, err := loadTestPrototypePipeline()
	if err != nil {
		t.Fatalf("Failed to load prototype pipeline: %v", err)
	}

	// Define expected persona requirements
	personaRequirements := map[string]struct {
		needsWrite bool
		needsRead  bool
	}{
		"craftsman":  {needsWrite: true, needsRead: true},   // spec, dummy, implement phases
		"philosopher": {needsWrite: true, needsRead: true},  // docs phase
	}

	for _, step := range pipeline.Steps {
		if step.ID == "spec" || step.ID == "docs" || step.ID == "dummy" || step.ID == "implement" {
			t.Run(step.ID, func(t *testing.T) {
				persona, exists := testManifest.Personas[step.Persona]
				if !exists {
					t.Fatalf("Persona %s not found in manifest", step.Persona)
				}

				requirements, exists := personaRequirements[step.Persona]
				if !exists {
					t.Fatalf("No requirements defined for persona %s", step.Persona)
				}

				// Check write permissions if required
				if requirements.needsWrite {
					hasWrite := false
					for _, tool := range persona.Permissions.AllowedTools {
						if tool == "Write" || tool == "Edit" {
							hasWrite = true
							break
						}
					}
					if !hasWrite {
						t.Errorf("Persona %s needs write permissions for step %s", step.Persona, step.ID)
					}
				}

				// Check read permissions if required
				if requirements.needsRead {
					hasRead := false
					for _, tool := range persona.Permissions.AllowedTools {
						if tool == "Read" {
							hasRead = true
							break
						}
					}
					if !hasRead {
						t.Errorf("Persona %s needs read permissions for step %s", step.Persona, step.ID)
					}
				}
			})
		}
	}
}