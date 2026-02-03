package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/manifest"
)

func TestResumeManager_ValidateResumePoint(t *testing.T) {
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

	tests := []struct {
		name            string
		fromStep        string
		setupWorkspace  func(t *testing.T, tempDir string)
		expectError     bool
		errorContains   string
	}{
		{
			name:     "valid step exists",
			fromStep: "docs",
			setupWorkspace: func(t *testing.T, tempDir string) {
				// Create completed spec phase
				specWorkspace := filepath.Join(tempDir, ".wave/workspaces/prototype/spec")
				err := os.MkdirAll(specWorkspace, 0755)
				if err != nil {
					t.Fatal(err)
				}
				err = os.WriteFile(filepath.Join(specWorkspace, "artifact.json"), []byte("{}"), 0644)
				if err != nil {
					t.Fatal(err)
				}
				err = os.WriteFile(filepath.Join(specWorkspace, "spec.md"), []byte("# Test spec"), 0644)
				if err != nil {
					t.Fatal(err)
				}
			},
			expectError: false,
		},
		{
			name:     "step does not exist",
			fromStep: "nonexistent",
			setupWorkspace: func(t *testing.T, tempDir string) {},
			expectError: true,
			errorContains: "step 'nonexistent' not found",
		},
		{
			name:     "missing prerequisite",
			fromStep: "dummy",
			setupWorkspace: func(t *testing.T, tempDir string) {
				// Create spec but not docs
				specWorkspace := filepath.Join(tempDir, ".wave/workspaces/prototype/spec")
				err := os.MkdirAll(specWorkspace, 0755)
				if err != nil {
					t.Fatal(err)
				}
				err = os.WriteFile(filepath.Join(specWorkspace, "artifact.json"), []byte("{}"), 0644)
				if err != nil {
					t.Fatal(err)
				}
				err = os.WriteFile(filepath.Join(specWorkspace, "spec.md"), []byte("# Test spec"), 0644)
				if err != nil {
					t.Fatal(err)
				}
			},
			expectError: true,
			errorContains: "prerequisite phase 'docs' not completed",
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

			err = manager.ValidateResumePoint(pipeline, tt.fromStep)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tt.name)
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.name, err)
				}
			}
		})
	}
}

func TestResumeManager_LoadResumeState(t *testing.T) {
	executor := NewDefaultPipelineExecutor(adapter.NewMockAdapter())
	manager := NewResumeManager(executor)

	pipeline := &Pipeline{
		Metadata: PipelineMetadata{Name: "prototype"},
		Steps: []Step{
			{
				ID: "spec",
				OutputArtifacts: []ArtifactDef{
					{Name: "spec", Path: "spec.md"},
					{Name: "contract_data", Path: "artifact.json"},
				},
			},
			{
				ID: "docs",
				Dependencies: []string{"spec"},
				OutputArtifacts: []ArtifactDef{
					{Name: "feature-docs", Path: "feature-docs.md"},
					{Name: "contract_data", Path: "artifact.json"},
				},
			},
			{
				ID: "dummy",
				Dependencies: []string{"docs"},
				OutputArtifacts: []ArtifactDef{
					{Name: "prototype", Path: "prototype/"},
					{Name: "interface-definitions", Path: "interfaces.md"},
					{Name: "contract_data", Path: "artifact.json"},
				},
			},
		},
	}

	tests := []struct {
		name               string
		fromStep           string
		setupWorkspace     func(t *testing.T, tempDir string)
		expectedCompleted  []string
		expectedArtifactDefs  int
	}{
		{
			name:     "resume from docs - spec completed",
			fromStep: "docs",
			setupWorkspace: func(t *testing.T, tempDir string) {
				// Create completed spec workspace
				specWorkspace := filepath.Join(tempDir, ".wave/workspaces/prototype/spec")
				err := os.MkdirAll(specWorkspace, 0755)
				if err != nil {
					t.Fatal(err)
				}
				err = os.WriteFile(filepath.Join(specWorkspace, "spec.md"), []byte("content"), 0644)
				if err != nil {
					t.Fatal(err)
				}
				err = os.WriteFile(filepath.Join(specWorkspace, "artifact.json"), []byte("{}"), 0644)
				if err != nil {
					t.Fatal(err)
				}
			},
			expectedCompleted: []string{"spec"},
			expectedArtifactDefs: 2, // spec:spec + spec:contract_data
		},
		{
			name:     "resume from dummy - spec and docs completed",
			fromStep: "dummy",
			setupWorkspace: func(t *testing.T, tempDir string) {
				// Create completed spec and docs workspaces
				phases := []struct {
					phase string
					files []string
				}{
					{"spec", []string{"spec.md", "artifact.json"}},
					{"docs", []string{"feature-docs.md", "artifact.json"}},
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
			expectedCompleted: []string{"spec", "docs"},
			expectedArtifactDefs: 4, // 2 artifacts per phase * 2 phases
		},
		{
			name:     "resume from beginning - no completed phases",
			fromStep: "spec",
			setupWorkspace: func(t *testing.T, tempDir string) {
				// Don't create any workspaces
			},
			expectedCompleted: []string{},
			expectedArtifactDefs: 0,
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

			state, err := manager.loadResumeState(pipeline, tt.fromStep)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(state.CompletedSteps) != len(tt.expectedCompleted) {
				t.Errorf("Expected %d completed steps, got %d", len(tt.expectedCompleted), len(state.CompletedSteps))
			}

			for i, expected := range tt.expectedCompleted {
				if i < len(state.CompletedSteps) && state.CompletedSteps[i] != expected {
					t.Errorf("Expected completed step %d to be %s, got %s", i, expected, state.CompletedSteps[i])
				}
			}

			if len(state.ArtifactPaths) != tt.expectedArtifactDefs {
				t.Errorf("Expected %d artifact paths, got %d", tt.expectedArtifactDefs, len(state.ArtifactPaths))
			}

			// Verify completed steps are marked with correct state
			for _, step := range tt.expectedCompleted {
				if state.States[step] != StateCompleted {
					t.Errorf("Expected step %s to have state %s, got %s", step, StateCompleted, state.States[step])
				}
			}
		})
	}
}

func TestResumeManager_CreateResumeSubpipeline(t *testing.T) {
	executor := NewDefaultPipelineExecutor(adapter.NewMockAdapter())
	manager := NewResumeManager(executor)

	pipeline := &Pipeline{
		Kind: "WavePipeline",
		Metadata: PipelineMetadata{
			Name:        "prototype",
			Description: "Test pipeline",
		},
		Steps: []Step{
			{ID: "spec"},
			{ID: "docs", Dependencies: []string{"spec"}},
			{ID: "dummy", Dependencies: []string{"docs"}},
			{ID: "implement", Dependencies: []string{"dummy"}},
		},
	}

	tests := []struct {
		name           string
		fromStep       string
		expectedSteps  []string
	}{
		{
			name:          "resume from beginning",
			fromStep:      "spec",
			expectedSteps: []string{"spec", "docs", "dummy", "implement"},
		},
		{
			name:          "resume from docs",
			fromStep:      "docs",
			expectedSteps: []string{"docs", "dummy", "implement"},
		},
		{
			name:          "resume from dummy",
			fromStep:      "dummy",
			expectedSteps: []string{"dummy", "implement"},
		},
		{
			name:          "resume from implement",
			fromStep:      "implement",
			expectedSteps: []string{"implement"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subpipeline := manager.createResumeSubpipeline(pipeline, tt.fromStep)

			if len(subpipeline.Steps) != len(tt.expectedSteps) {
				t.Errorf("Expected %d steps, got %d", len(tt.expectedSteps), len(subpipeline.Steps))
			}

			for i, expectedStep := range tt.expectedSteps {
				if i < len(subpipeline.Steps) && subpipeline.Steps[i].ID != expectedStep {
					t.Errorf("Expected step %d to be %s, got %s", i, expectedStep, subpipeline.Steps[i].ID)
				}
			}

			// Verify metadata is preserved
			if subpipeline.Metadata.Name != pipeline.Metadata.Name {
				t.Errorf("Expected metadata name to be preserved")
			}

			if subpipeline.Kind != pipeline.Kind {
				t.Errorf("Expected kind to be preserved")
			}
		})
	}
}

func TestResumeManager_GetRecommendedResumePoint(t *testing.T) {
	executor := NewDefaultPipelineExecutor(adapter.NewMockAdapter())
	manager := NewResumeManager(executor)

	pipeline := &Pipeline{
		Metadata: PipelineMetadata{Name: "prototype"},
		Steps: []Step{
			{ID: "spec"},
			{ID: "docs"},
			{ID: "dummy"},
			{ID: "implement"},
		},
	}

	tests := []struct {
		name            string
		setupWorkspace  func(t *testing.T, tempDir string)
		expectedPoint   string
		expectError     bool
	}{
		{
			name: "no phases completed",
			setupWorkspace: func(t *testing.T, tempDir string) {
				// Don't create any workspaces
			},
			expectedPoint: "spec",
			expectError:   false,
		},
		{
			name: "spec completed, docs incomplete",
			setupWorkspace: func(t *testing.T, tempDir string) {
				// Create completed spec workspace
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
			},
			expectedPoint: "docs",
			expectError:   false,
		},
		{
			name: "spec and docs completed, dummy incomplete",
			setupWorkspace: func(t *testing.T, tempDir string) {
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
			expectedPoint: "dummy",
			expectError:   false,
		},
		{
			name: "all phases completed",
			setupWorkspace: func(t *testing.T, tempDir string) {
				phases := []struct {
					phase string
					files []string
					dirs  []string
				}{
					{"spec", []string{"artifact.json", "spec.md"}, []string{}},
					{"docs", []string{"artifact.json", "feature-docs.md"}, []string{}},
					{"dummy", []string{"artifact.json", "interfaces.md"}, []string{"prototype"}},
					{"implement", []string{"artifact.json", "implementation-plan.md"}, []string{}},
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

					for _, dir := range phase.dirs {
						err = os.MkdirAll(filepath.Join(phaseWorkspace, dir), 0755)
						if err != nil {
							t.Fatal(err)
						}
					}
				}
			},
			expectedPoint: "implement", // Suggest implement for additional work
			expectError:   false,
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

			point, err := manager.GetRecommendedResumePoint(pipeline)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.name, err)
				}

				if point != tt.expectedPoint {
					t.Errorf("Expected recommended point %s, got %s", tt.expectedPoint, point)
				}
			}
		})
	}
}

func TestResumeManager_IntegrationWithStaleDetection(t *testing.T) {
	// Test integration between resume functionality and stale artifact detection
	executor := NewDefaultPipelineExecutor(adapter.NewMockAdapter())
	manager := NewResumeManager(executor)

	pipeline := &Pipeline{
		Metadata: PipelineMetadata{Name: "prototype"},
		Steps: []Step{
			{ID: "spec"},
			{ID: "docs", Dependencies: []string{"spec"}},
			{ID: "dummy", Dependencies: []string{"docs"}},
		},
	}

	_ = &manifest.Manifest{
		APIVersion: "v1",
		Kind:       "WaveManifest",
		Metadata: manifest.Metadata{
			Name: "test-manifest",
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

	// Setup workspace with stale artifacts scenario
	baseTime := time.Now().Add(-1 * time.Hour)

	// Create docs workspace (older)
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

	// Create spec workspace (newer - simulating re-run after docs)
	specWorkspace := filepath.Join(tempDir, ".wave/workspaces/prototype/spec")
	err = os.MkdirAll(specWorkspace, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create required spec artifacts
	err = os.WriteFile(filepath.Join(specWorkspace, "artifact.json"), []byte("{}"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	specFile := filepath.Join(specWorkspace, "spec.md")
	err = os.WriteFile(specFile, []byte("updated spec content"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.Chtimes(specFile, baseTime.Add(30*time.Minute), baseTime.Add(30*time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	// Validate resume point - should succeed despite stale artifacts
	err = manager.ValidateResumePoint(pipeline, "dummy")
	if err != nil {
		t.Errorf("Resume point validation should succeed despite stale artifacts: %v", err)
	}

	// Test that stale artifact detection is triggered during resume
	// This would normally be part of ResumeFromStep, but we're testing the detection separately
	detector := manager.detector
	staleReasons, err := detector.DetectStaleArtifacts(pipeline, "dummy")
	if err != nil {
		t.Fatalf("Stale detection failed: %v", err)
	}

	if len(staleReasons) == 0 {
		t.Error("Expected stale artifacts to be detected")
	}

	// Verify stale reason mentions the upstream change
	found := false
	for _, reason := range staleReasons {
		if strings.Contains(reason, "upstream phase 'spec'") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected stale reason to mention upstream spec phase, got: %v", staleReasons)
	}
}