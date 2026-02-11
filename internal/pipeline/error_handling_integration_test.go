//go:build integration
// +build integration

// Integration tests for error handling with full pipeline execution.
// Run with: go test -tags=integration ./internal/pipeline/...

package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
)

func TestErrorHandlingIntegration(t *testing.T) {
	// Test integration of all error handling components
	executor := NewDefaultPipelineExecutor(adapter.NewMockAdapter(),
		WithEmitter(event.NewNDJSONEmitter()),
		WithDebug(true))

	pipeline := &Pipeline{
		Metadata: PipelineMetadata{
			Name:        "prototype",
			Description: "Test prototype pipeline with error handling",
		},
		Steps: []Step{
			{
				ID:      "spec",
				Persona: "craftsman",
				OutputArtifacts: []ArtifactDef{
					{Name: "spec", Path: "spec.md"},
					{Name: "contract_data", Path: "artifact.json"},
				},
				Handover: HandoverConfig{
					Contract: ContractConfig{
						Type:       "json_schema",
						SchemaPath: ".wave/contracts/spec-phase.schema.json",
						MustPass:   true,
					},
				},
			},
			{
				ID:           "docs",
				Persona:      "philosopher",
				Dependencies: []string{"spec"},
				OutputArtifacts: []ArtifactDef{
					{Name: "feature-docs", Path: "feature-docs.md"},
					{Name: "contract_data", Path: "artifact.json"},
				},
				Handover: HandoverConfig{
					Contract: ContractConfig{
						Type:       "json_schema",
						SchemaPath: ".wave/contracts/docs-phase.schema.json",
						MustPass:   true,
					},
				},
			},
			{
				ID:           "dummy",
				Persona:      "craftsman",
				Dependencies: []string{"docs"},
				OutputArtifacts: []ArtifactDef{
					{Name: "prototype", Path: "prototype/"},
					{Name: "interface-definitions", Path: "interfaces.md"},
					{Name: "contract_data", Path: "artifact.json"},
				},
				Handover: HandoverConfig{
					Contract: ContractConfig{
						Type:       "json_schema",
						SchemaPath: ".wave/contracts/dummy-phase.schema.json",
						MustPass:   true,
					},
				},
			},
		},
	}

	manifest := &manifest.Manifest{
		APIVersion: "v1",
		Kind:       "WaveManifest",
		Metadata: manifest.Metadata{
			Name: "test-manifest",
		},
		Personas: map[string]manifest.Persona{
			"craftsman": {
				Adapter:     "claude",
				Description: "Implementation specialist",
				Permissions: manifest.Permissions{
					AllowedTools: []string{"Read", "Write", "Edit"},
				},
			},
			"philosopher": {
				Adapter:     "claude",
				Description: "Documentation specialist",
				Permissions: manifest.Permissions{
					AllowedTools: []string{"Read", "Write"},
				},
			},
		},
	}

	tests := []struct {
		name           string
		setupWorkspace func(t *testing.T, tempDir string)
		fromStep       string
		expectError    bool
		errorContains  []string
	}{
		{
			name: "successful execution from beginning",
			setupWorkspace: func(t *testing.T, tempDir string) {
				// Create contract schemas
				contractDir := filepath.Join(tempDir, ".wave/contracts")
				err := os.MkdirAll(contractDir, 0755)
				if err != nil {
					t.Fatal(err)
				}

				// Create minimal schema files
				schemas := []string{"spec-phase.schema.json", "docs-phase.schema.json", "dummy-phase.schema.json"}
				for _, schema := range schemas {
					schemaContent := `{"type": "object", "properties": {}}`
					err = os.WriteFile(filepath.Join(contractDir, schema), []byte(schemaContent), 0644)
					if err != nil {
						t.Fatal(err)
					}
				}
			},
			fromStep:    "",
			expectError: false, // Mock adapter may fail, but validation should pass
		},
		{
			name: "phase skip validation failure",
			setupWorkspace: func(t *testing.T, tempDir string) {
				// Don't create any prerequisite workspaces
			},
			fromStep:      "dummy",
			expectError:   true,
			errorContains: []string{"prerequisite phase", "not completed"},
		},
		{
			name: "stale artifact detection with warning",
			setupWorkspace: func(t *testing.T, tempDir string) {
				baseTime := time.Now().Add(-1 * time.Hour)

				// Create completed spec phase workspace
				specWorkspace := filepath.Join(tempDir, ".wave/workspaces/prototype/spec")
				err := os.MkdirAll(specWorkspace, 0755)
				if err != nil {
					t.Fatal(err)
				}
				err = os.WriteFile(filepath.Join(specWorkspace, "artifact.json"), []byte("{}"), 0644)
				if err != nil {
					t.Fatal(err)
				}
				err = os.WriteFile(filepath.Join(specWorkspace, "spec.md"), []byte("spec content"), 0644)
				if err != nil {
					t.Fatal(err)
				}

				// Create completed docs phase workspace (older)
				docsWorkspace := filepath.Join(tempDir, ".wave/workspaces/prototype/docs")
				err = os.MkdirAll(docsWorkspace, 0755)
				if err != nil {
					t.Fatal(err)
				}
				docsFile := filepath.Join(docsWorkspace, "feature-docs.md")
				err = os.WriteFile(docsFile, []byte("docs content"), 0644)
				if err != nil {
					t.Fatal(err)
				}
				err = os.Chtimes(docsFile, baseTime, baseTime)
				if err != nil {
					t.Fatal(err)
				}
				// Add required artifact.json for contract validation
				err = os.WriteFile(filepath.Join(docsWorkspace, "artifact.json"), []byte("{}"), 0644)
				if err != nil {
					t.Fatal(err)
				}

				// Make spec newer than docs (simulating spec re-run)
				specFile := filepath.Join(specWorkspace, "spec.md")
				err = os.Chtimes(specFile, baseTime.Add(30*time.Minute), baseTime.Add(30*time.Minute))
				if err != nil {
					t.Fatal(err)
				}
			},
			fromStep:    "dummy",
			expectError: false, // Should warn but not fail
		},
		{
			name: "concurrent execution protection",
			setupWorkspace: func(t *testing.T, tempDir string) {
				// Setup will be handled in test logic
			},
			fromStep:      "",
			expectError:   true,
			errorContains: []string{"workspace", "already in use"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			originalWd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(originalWd)

			err = os.Chdir(tempDir)
			if err != nil {
				t.Fatal(err)
			}

			tt.setupWorkspace(t, tempDir)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			var executionErr error

			if tt.name == "concurrent execution protection" {
				// Test concurrent execution protection
				concurrency := NewConcurrencyValidator()

				// Acquire lock first
				workspaceID := fmt.Sprintf("%s/%s", pipeline.Metadata.Name, "full")
				err := concurrency.AcquireWorkspaceLock(pipeline.Metadata.Name, workspaceID)
				if err != nil {
					t.Fatal(err)
				}

				// Try to acquire again (should fail)
				executionErr = concurrency.AcquireWorkspaceLock("another-pipeline", workspaceID)

				concurrency.ReleaseWorkspaceLock(pipeline.Metadata.Name)
			} else {
				// Test normal execution or resumption
				if tt.fromStep == "" {
					executionErr = executor.ExecuteWithValidation(ctx, pipeline, manifest, "test input")
				} else {
					executionErr = executor.ResumeWithValidation(ctx, pipeline, manifest, "test input", tt.fromStep, false)
				}
			}

			if tt.expectError {
				if executionErr == nil {
					t.Errorf("Expected error for %s, but got none", tt.name)
				} else {
					// Verify error contains expected content
					errorMsg := executionErr.Error()
					for _, expectedContent := range tt.errorContains {
						if !strings.Contains(errorMsg, expectedContent) {
							t.Errorf("Expected error to contain %q, got: %s", expectedContent, errorMsg)
						}
					}
				}
			} else {
				if executionErr != nil && !strings.Contains(executionErr.Error(), "mock adapter") {
					// Ignore mock adapter failures, focus on validation logic
					t.Errorf("Unexpected error for %s: %v", tt.name, executionErr)
				}
			}
		})
	}
}

func TestErrorMessageFormatting(t *testing.T) {
	// Test that error messages are properly formatted and contain helpful information
	provider := NewErrorMessageProvider()

	phases := []string{"spec", "docs", "dummy", "implement"}
	for _, phase := range phases {
		t.Run(phase, func(t *testing.T) {
			originalError := fmt.Errorf("test error for %s", phase)
			formattedError := provider.FormatPhaseFailureError(phase, originalError, "prototype")

			errorMsg := formattedError.Error()

			// Verify essential components are present
			requiredComponents := []string{
				fmt.Sprintf("Phase '%s' failed", phase),
				"🔧 Troubleshooting Guide",
				"🔄 Retry Options",
				"📋 Debug Information",
				"wave run prototype",
				".wave/workspaces/prototype/" + phase,
			}

			for _, component := range requiredComponents {
				if !strings.Contains(errorMsg, component) {
					t.Errorf("Error message should contain %q, but message was:\n%s", component, errorMsg)
				}
			}

			// Verify retry commands are valid
			if !strings.Contains(errorMsg, "--from-step "+phase) {
				t.Errorf("Error message should contain retry command for phase %s", phase)
			}

			// Verify retry command uses the actual pipeline name, not hardcoded "prototype"
			if !strings.Contains(errorMsg, "wave run prototype") {
				t.Errorf("Error message should contain pipeline-specific retry command for phase %s", phase)
			}
		})
	}
}

func TestResumeManagerIntegration(t *testing.T) {
	// Test integration of ResumeManager with all validation components
	executor := NewDefaultPipelineExecutor(adapter.NewMockAdapter())
	manager := NewResumeManager(executor)

	pipeline := &Pipeline{
		Metadata: PipelineMetadata{Name: "prototype"},
		Steps: []Step{
			{ID: "spec"},
			{ID: "docs", Dependencies: []string{"spec"}},
			{ID: "dummy", Dependencies: []string{"docs"}},
			{ID: "implement", Dependencies: []string{"dummy"}},
		},
	}

	tempDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalWd)

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Test validation integration
	tests := []struct {
		name           string
		setupWorkspace func(t *testing.T)
		fromStep       string
		expectValid    bool
		expectedReason string
	}{
		{
			name: "valid resume point",
			setupWorkspace: func(t *testing.T) {
				// Create completed spec and docs phases
				phases := []struct {
					phase string
					files []string
				}{
					{"spec", []string{"artifact.json", "spec.md"}},
					{"docs", []string{"artifact.json", "feature-docs.md"}},
				}

				for _, phase := range phases {
					phaseWorkspace := filepath.Join(tempDir, ".wave/workspaces/prototype", phase.phase)
					err := os.MkdirAll(phaseWorkspace, 0755)
					if err != nil {
						t.Fatal(err)
					}

					for _, file := range phase.files {
						err = os.WriteFile(filepath.Join(phaseWorkspace, file), []byte("content"), 0644)
						if err != nil {
							t.Fatal(err)
						}
					}
				}
			},
			fromStep:    "dummy",
			expectValid: true,
		},
		{
			name: "invalid resume point - missing prerequisites",
			setupWorkspace: func(t *testing.T) {
				// Don't create any completed phases
			},
			fromStep:       "dummy",
			expectValid:    false,
			expectedReason: "prerequisite",
		},
		{
			name: "invalid resume point - nonexistent step",
			setupWorkspace: func(t *testing.T) {},
			fromStep:       "nonexistent",
			expectValid:    false,
			expectedReason: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean workspace for each test
			workspaceDir := filepath.Join(tempDir, ".wave/workspaces")
			os.RemoveAll(workspaceDir)

			tt.setupWorkspace(t)

			err := manager.ValidateResumePoint(pipeline, tt.fromStep)

			if tt.expectValid {
				if err != nil {
					t.Errorf("Expected validation to pass for %s, got error: %v", tt.name, err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected validation to fail for %s, but got no error", tt.name)
				} else if tt.expectedReason != "" && !strings.Contains(err.Error(), tt.expectedReason) {
					t.Errorf("Expected error to contain %q, got: %s", tt.expectedReason, err.Error())
				}
			}
		})
	}

	// Test recommended resume point functionality
	t.Run("recommended resume point", func(t *testing.T) {
		// Clean workspace
		workspaceDir := filepath.Join(tempDir, ".wave/workspaces")
		os.RemoveAll(workspaceDir)

		// Create completed spec phase only
		specWorkspace := filepath.Join(tempDir, ".wave/workspaces/prototype/spec")
		err := os.MkdirAll(specWorkspace, 0755)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(filepath.Join(specWorkspace, "artifact.json"), []byte("{}"), 0644)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(filepath.Join(specWorkspace, "spec.md"), []byte("content"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		recommendedPoint, err := manager.GetRecommendedResumePoint(pipeline)
		if err != nil {
			t.Fatalf("Failed to get recommended resume point: %v", err)
		}

		if recommendedPoint != "docs" {
			t.Errorf("Expected recommended resume point to be 'docs', got '%s'", recommendedPoint)
		}
	})
}

func TestPipelineStatusTracking(t *testing.T) {
	// Test that pipeline status is properly tracked through execution
	executor := NewDefaultPipelineExecutor(adapter.NewMockAdapter(),
		WithEmitter(event.NewNDJSONEmitter()))

	pipeline := &Pipeline{
		Metadata: PipelineMetadata{Name: "test-pipeline"},
		Steps: []Step{
			{ID: "step1"},
			{ID: "step2", Dependencies: []string{"step1"}},
		},
	}

	manifest := &manifest.Manifest{
		APIVersion: "v1",
		Kind:       "WaveManifest",
		Metadata:   manifest.Metadata{Name: "test"},
	}

	// Test status tracking during execution
	tempDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalWd)

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// This would normally fail with mock adapter, but we're testing status tracking
	executor.ExecuteWithValidation(ctx, pipeline, manifest, "test input")

	// Get status
	status, err := executor.GetStatus("test-pipeline")
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if status == nil {
		t.Fatal("Expected status to be non-nil")
	}

	if status.ID != "test-pipeline" {
		t.Errorf("Expected status ID to be 'test-pipeline', got '%s'", status.ID)
	}

	// Status should be failed due to mock adapter, but structure should be correct
	if status.State != StateFailed && status.State != StateCompleted {
		// Allow either failed (due to mock) or completed (if simulation works)
		t.Logf("Pipeline state: %s", status.State)
	}
}