package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/event"
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

func TestResumeManager_LoadResumeState_HashSuffixedRunDirs(t *testing.T) {
	executor := NewDefaultPipelineExecutor(adapter.NewMockAdapter())
	manager := NewResumeManager(executor)

	pipeline := &Pipeline{
		Metadata: PipelineMetadata{Name: "gh-refresh"},
		Steps: []Step{
			{
				ID: "gather-context",
				Workspace: WorkspaceConfig{Type: "worktree"},
				OutputArtifacts: []ArtifactDef{
					{Name: "issue_context", Path: "artifact.json"},
				},
			},
			{
				ID:           "draft-update",
				Dependencies: []string{"gather-context"},
				Workspace:    WorkspaceConfig{Type: "worktree"},
				OutputArtifacts: []ArtifactDef{
					{Name: "update_draft", Path: "artifact.json"},
				},
			},
			{
				ID:           "apply-update",
				Dependencies: []string{"draft-update"},
				Workspace:    WorkspaceConfig{Type: "worktree"},
			},
		},
	}

	tests := []struct {
		name              string
		fromStep          string
		setupWorkspace    func(t *testing.T, tempDir string)
		expectedCompleted []string
		expectedArtifacts int
		checkArtifactKey  string // verify a specific artifact path is set
	}{
		{
			name:     "finds artifacts in hash-suffixed __wt_ dir",
			fromStep: "draft-update",
			setupWorkspace: func(t *testing.T, tempDir string) {
				// Simulate a previous run with hash-suffixed dir and __wt_ worktree
				wtDir := filepath.Join(tempDir, ".wave/workspaces/gh-refresh-20260219-142150-deb8/__wt_gh-refresh-20260219-142150-deb8")
				if err := os.MkdirAll(wtDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(wtDir, "artifact.json"), []byte(`{"issue":{}}`), 0644); err != nil {
					t.Fatal(err)
				}
			},
			expectedCompleted: []string{"gather-context"},
			expectedArtifacts: 1,
			checkArtifactKey:  "gather-context:issue_context",
		},
		{
			name:     "finds artifacts across multiple hash-suffixed runs (picks most recent)",
			fromStep: "apply-update",
			setupWorkspace: func(t *testing.T, tempDir string) {
				// Old run with gather-context
				oldWt := filepath.Join(tempDir, ".wave/workspaces/gh-refresh-20260219-100000-aaaa/__wt_some-branch")
				if err := os.MkdirAll(oldWt, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(oldWt, "artifact.json"), []byte(`{"old":true}`), 0644); err != nil {
					t.Fatal(err)
				}

				// Newer run with both gather-context and draft-update artifacts
				newWt := filepath.Join(tempDir, ".wave/workspaces/gh-refresh-20260219-200000-bbbb/__wt_other-branch")
				if err := os.MkdirAll(newWt, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(newWt, "artifact.json"), []byte(`{"new":true}`), 0644); err != nil {
					t.Fatal(err)
				}
			},
			expectedCompleted: []string{"gather-context", "draft-update"},
			expectedArtifacts: 2,
			checkArtifactKey:  "draft-update:update_draft",
		},
		{
			name:     "finds artifacts in old-style step-named dirs under hash-suffixed run",
			fromStep: "draft-update",
			setupWorkspace: func(t *testing.T, tempDir string) {
				// Old-style: step ID as directory name
				stepDir := filepath.Join(tempDir, ".wave/workspaces/gh-refresh-20260219-142150-deb8/gather-context")
				if err := os.MkdirAll(stepDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(stepDir, "artifact.json"), []byte(`{}`), 0644); err != nil {
					t.Fatal(err)
				}
			},
			expectedCompleted: []string{"gather-context"},
			expectedArtifacts: 1,
			checkArtifactKey:  "gather-context:issue_context",
		},
		{
			name:     "mount-type steps found in hash-suffixed run dirs",
			fromStep: "apply-update",
			setupWorkspace: func(t *testing.T, tempDir string) {
				// gather-context in a __wt_ dir
				wtDir := filepath.Join(tempDir, ".wave/workspaces/gh-refresh-20260219-142150-deb8/__wt_some-branch")
				if err := os.MkdirAll(wtDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(wtDir, "artifact.json"), []byte(`{}`), 0644); err != nil {
					t.Fatal(err)
				}
				// draft-update in an old-style step dir from a different run
				stepDir := filepath.Join(tempDir, ".wave/workspaces/gh-refresh-20260219-144614-3336/draft-update")
				if err := os.MkdirAll(stepDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(stepDir, "artifact.json"), []byte(`{}`), 0644); err != nil {
					t.Fatal(err)
				}
			},
			expectedCompleted: []string{"gather-context", "draft-update"},
			expectedArtifacts: 2,
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

			if err := os.Chdir(tempDir); err != nil {
				t.Fatal(err)
			}

			tt.setupWorkspace(t, tempDir)

			state, err := manager.loadResumeState(pipeline, tt.fromStep)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(state.CompletedSteps) != len(tt.expectedCompleted) {
				t.Errorf("Expected %d completed steps, got %d: %v", len(tt.expectedCompleted), len(state.CompletedSteps), state.CompletedSteps)
			}

			for i, expected := range tt.expectedCompleted {
				if i < len(state.CompletedSteps) && state.CompletedSteps[i] != expected {
					t.Errorf("Expected completed step %d to be %s, got %s", i, expected, state.CompletedSteps[i])
				}
			}

			if len(state.ArtifactPaths) != tt.expectedArtifacts {
				t.Errorf("Expected %d artifact paths, got %d: %v", tt.expectedArtifacts, len(state.ArtifactPaths), state.ArtifactPaths)
			}

			if tt.checkArtifactKey != "" {
				if path, ok := state.ArtifactPaths[tt.checkArtifactKey]; !ok {
					t.Errorf("Expected artifact key %q to be set, but it wasn't. Keys: %v", tt.checkArtifactKey, state.ArtifactPaths)
				} else if _, err := os.Stat(path); err != nil {
					t.Errorf("Artifact path %q for key %q does not exist: %v", path, tt.checkArtifactKey, err)
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

	// Create required docs artifacts for contract validation
	err = os.WriteFile(filepath.Join(docsWorkspace, "artifact.json"), []byte("{}"), 0644)
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

	// Create dummy workspace (older than docs re-run scenario)
	dummyWorkspace := filepath.Join(tempDir, ".wave/workspaces/prototype/dummy")
	err = os.MkdirAll(dummyWorkspace, 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(dummyWorkspace, "artifact.json"), []byte("{}"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	// Set dummy workspace time to be older than docs
	dummyFile := filepath.Join(dummyWorkspace, "artifact.json")
	err = os.Chtimes(dummyFile, baseTime.Add(-30*time.Minute), baseTime.Add(-30*time.Minute))
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

	// Verify stale reason mentions the upstream change (dummy depends on docs)
	found := false
	for _, reason := range staleReasons {
		if strings.Contains(reason, "upstream phase 'docs'") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected stale reason to mention upstream docs phase, got: %v", staleReasons)
	}
}

func TestCreateResumeSubpipelineStripsPriorDependencies(t *testing.T) {
	executor := NewDefaultPipelineExecutor(adapter.NewMockAdapter())
	manager := NewResumeManager(executor)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "issue-research"},
		Steps: []Step{
			{ID: "fetch-issue", Persona: "github-analyst"},
			{ID: "analyze-topics", Persona: "researcher", Dependencies: []string{"fetch-issue"}},
			{ID: "research-topics", Persona: "researcher", Dependencies: []string{"analyze-topics"}},
		},
	}

	sub := manager.createResumeSubpipeline(p, "analyze-topics")

	if len(sub.Steps) != 2 {
		t.Fatalf("expected 2 steps in subpipeline, got %d", len(sub.Steps))
	}

	// analyze-topics should have its dependency on fetch-issue stripped
	if len(sub.Steps[0].Dependencies) != 0 {
		t.Errorf("expected analyze-topics to have 0 dependencies, got %v", sub.Steps[0].Dependencies)
	}

	// research-topics should still depend on analyze-topics (it's in the subpipeline)
	if len(sub.Steps[1].Dependencies) != 1 || sub.Steps[1].Dependencies[0] != "analyze-topics" {
		t.Errorf("expected research-topics to depend on analyze-topics, got %v", sub.Steps[1].Dependencies)
	}

	// Verify DAG validates successfully
	validator := &DAGValidator{}
	if err := validator.ValidateDAG(sub); err != nil {
		t.Errorf("subpipeline DAG should be valid, got: %v", err)
	}
}

func TestResumeFromStepWithForceSkipsValidation(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(500),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter)
	manager := NewResumeManager(executor)

	m := &manifest.Manifest{
		Metadata: manifest.Metadata{Name: "test-project"},
		Adapters: map[string]manifest.Adapter{
			"claude": {Binary: "claude", Mode: "headless"},
		},
		Personas: map[string]manifest.Persona{
			"researcher": {
				Adapter:     "claude",
				Temperature: 0.1,
			},
		},
		Runtime: manifest.Runtime{
			WorkspaceRoot:     tmpDir,
			DefaultTimeoutMin: 5,
		},
	}

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "prototype"},
		Steps: []Step{
			{ID: "spec", Persona: "researcher", Exec: ExecConfig{Source: "generate spec"}},
			{ID: "docs", Persona: "researcher", Dependencies: []string{"spec"}, Exec: ExecConfig{Source: "generate docs"}},
		},
	}

	ctx := context.Background()

	// Without force, this should fail because spec workspace doesn't exist
	err := manager.ResumeFromStep(ctx, p, m, "test", "docs", false)
	if err == nil {
		t.Error("expected error without force when prerequisites are missing")
	}
	if err != nil && !strings.Contains(err.Error(), "prerequisite phase") {
		t.Errorf("expected prerequisite phase error, got: %v", err)
	}

	// With force, validation should be skipped. Execution may fail for other reasons
	// (mock adapter, missing workspace, etc.) but NOT due to phase validation.
	err = manager.ResumeFromStep(ctx, p, m, "test", "docs", true)
	if err != nil && strings.Contains(err.Error(), "prerequisite phase") {
		t.Errorf("force should skip phase validation, got: %v", err)
	}
}

// testEmitter captures events for test assertions.
type testEmitter struct {
	mu     sync.Mutex
	events []event.Event
}

func (te *testEmitter) Emit(evt event.Event) {
	te.mu.Lock()
	defer te.mu.Unlock()
	te.events = append(te.events, evt)
}

func TestResumeFromStep_EmitsSyntheticCompletionEvents(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Create workspace dirs for step 1 ("gather") with an artifact file
	// so that loadResumeState discovers it as completed.
	gatherWs := filepath.Join(tmpDir, ".wave/workspaces/test-pipeline/gather")
	if err := os.MkdirAll(gatherWs, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gatherWs, "artifact.json"), []byte(`{"gathered": true}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Define a 3-step pipeline: gather -> analyze -> report
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "test-pipeline"},
		Steps: []Step{
			{
				ID:      "gather",
				Persona: "researcher",
				Exec:    ExecConfig{Source: "gather data"},
				OutputArtifacts: []ArtifactDef{
					{Name: "gathered_data", Path: "artifact.json"},
				},
			},
			{
				ID:           "analyze",
				Persona:      "analyst",
				Dependencies: []string{"gather"},
				Exec:         ExecConfig{Source: "analyze data"},
			},
			{
				ID:           "report",
				Persona:      "writer",
				Dependencies: []string{"analyze"},
				Exec:         ExecConfig{Source: "write report"},
			},
		},
	}

	m := &manifest.Manifest{
		Metadata: manifest.Metadata{Name: "test-project"},
		Adapters: map[string]manifest.Adapter{
			"claude": {Binary: "claude", Mode: "headless"},
		},
		Personas: map[string]manifest.Persona{
			"researcher": {Adapter: "claude", Temperature: 0.1},
			"analyst":    {Adapter: "claude", Temperature: 0.1},
			"writer":     {Adapter: "claude", Temperature: 0.1},
		},
		Runtime: manifest.Runtime{
			WorkspaceRoot:     tmpDir,
			DefaultTimeoutMin: 5,
		},
	}

	// Create executor with a mock adapter and attach testEmitter
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(500),
	)
	emitter := &testEmitter{}
	executor := NewDefaultPipelineExecutor(mockAdapter, WithEmitter(emitter))
	manager := NewResumeManager(executor)

	ctx := context.Background()

	// Call ResumeFromStep with force=true to skip phase validation.
	// Execution will likely fail (mock adapter + workspace issues), but
	// synthetic completion events are emitted BEFORE execution begins.
	_ = manager.ResumeFromStep(ctx, p, m, "test-input", "analyze", true)

	// Filter captured events for synthetic completion events
	emitter.mu.Lock()
	defer emitter.mu.Unlock()

	var syntheticEvents []event.Event
	for _, evt := range emitter.events {
		if evt.State == "completed" && evt.Message == "completed in prior run" {
			syntheticEvents = append(syntheticEvents, evt)
		}
	}

	if len(syntheticEvents) == 0 {
		t.Fatal("expected at least one synthetic completion event for prior steps, got none")
	}

	// Verify that step "gather" received a synthetic completion event
	foundGather := false
	for _, evt := range syntheticEvents {
		if evt.StepID == "gather" {
			foundGather = true
			if evt.Persona != "researcher" {
				t.Errorf("expected Persona %q for gather step event, got %q", "researcher", evt.Persona)
			}
			if evt.Message != "completed in prior run" {
				t.Errorf("expected Message %q, got %q", "completed in prior run", evt.Message)
			}
			break
		}
	}

	if !foundGather {
		t.Errorf("expected synthetic completion event with StepID='gather', got events: %v", syntheticEvents)
	}
}
