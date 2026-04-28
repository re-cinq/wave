package pipeline

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter/adaptertest"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/testutil"
)

func TestResumeManager_ValidateResumePoint(t *testing.T) {
	executor := NewDefaultPipelineExecutor(adaptertest.NewMockAdapter())
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
		name           string
		fromStep       string
		setupWorkspace func(t *testing.T, tempDir string)
		expectError    bool
		errorContains  string
	}{
		{
			name:     "valid step exists",
			fromStep: "docs",
			setupWorkspace: func(t *testing.T, tempDir string) {
				// Create completed spec phase
				specWorkspace := filepath.Join(tempDir, ".agents/workspaces/prototype/spec")
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
			name:           "step does not exist",
			fromStep:       "nonexistent",
			setupWorkspace: func(t *testing.T, tempDir string) {},
			expectError:    true,
			errorContains:  "step 'nonexistent' not found",
		},
		{
			name:     "missing prerequisite",
			fromStep: "dummy",
			setupWorkspace: func(t *testing.T, tempDir string) {
				// Create spec but not docs
				specWorkspace := filepath.Join(tempDir, ".agents/workspaces/prototype/spec")
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
			expectError:   true,
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
			defer func() { _ = os.Chdir(originalWd) }()

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
	executor := NewDefaultPipelineExecutor(adaptertest.NewMockAdapter())
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
				ID:           "docs",
				Dependencies: []string{"spec"},
				OutputArtifacts: []ArtifactDef{
					{Name: "feature-docs", Path: "feature-docs.md"},
					{Name: "contract_data", Path: "artifact.json"},
				},
			},
			{
				ID:           "dummy",
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
		name                 string
		fromStep             string
		setupWorkspace       func(t *testing.T, tempDir string)
		expectedCompleted    []string
		expectedArtifactDefs int
	}{
		{
			name:     "resume from docs - spec completed",
			fromStep: "docs",
			setupWorkspace: func(t *testing.T, tempDir string) {
				// Create completed spec workspace
				specWorkspace := filepath.Join(tempDir, ".agents/workspaces/prototype/spec")
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
			expectedCompleted:    []string{"spec"},
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
					phaseWorkspace := filepath.Join(tempDir, ".agents/workspaces/prototype", phase.phase)
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
			expectedCompleted:    []string{"spec", "docs"},
			expectedArtifactDefs: 4, // 2 artifacts per phase * 2 phases
		},
		{
			name:     "resume from beginning - no completed phases",
			fromStep: "spec",
			setupWorkspace: func(t *testing.T, tempDir string) {
				// Don't create any workspaces
			},
			expectedCompleted:    []string{},
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
			defer func() { _ = os.Chdir(originalWd) }()

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
				if state.States[step] != stateCompleted {
					t.Errorf("Expected step %s to have state %s, got %s", step, stateCompleted, state.States[step])
				}
			}
		})
	}
}

func TestResumeManager_LoadResumeState_HashSuffixedRunDirs(t *testing.T) {
	executor := NewDefaultPipelineExecutor(adaptertest.NewMockAdapter())
	manager := NewResumeManager(executor)

	pipeline := &Pipeline{
		Metadata: PipelineMetadata{Name: "gh-refresh"},
		Steps: []Step{
			{
				ID:        "gather-context",
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
				wtDir := filepath.Join(tempDir, ".agents/workspaces/gh-refresh-20260219-142150-deb8/__wt_gh-refresh-20260219-142150-deb8")
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
				oldWt := filepath.Join(tempDir, ".agents/workspaces/gh-refresh-20260219-100000-aaaa/__wt_some-branch")
				if err := os.MkdirAll(oldWt, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(oldWt, "artifact.json"), []byte(`{"old":true}`), 0644); err != nil {
					t.Fatal(err)
				}

				// Newer run with both gather-context and draft-update artifacts
				newWt := filepath.Join(tempDir, ".agents/workspaces/gh-refresh-20260219-200000-bbbb/__wt_other-branch")
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
				stepDir := filepath.Join(tempDir, ".agents/workspaces/gh-refresh-20260219-142150-deb8/gather-context")
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
				wtDir := filepath.Join(tempDir, ".agents/workspaces/gh-refresh-20260219-142150-deb8/__wt_some-branch")
				if err := os.MkdirAll(wtDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(wtDir, "artifact.json"), []byte(`{}`), 0644); err != nil {
					t.Fatal(err)
				}
				// draft-update in an old-style step dir from a different run
				stepDir := filepath.Join(tempDir, ".agents/workspaces/gh-refresh-20260219-144614-3336/draft-update")
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
			defer func() { _ = os.Chdir(originalWd) }()

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
	executor := NewDefaultPipelineExecutor(adaptertest.NewMockAdapter())
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
		name          string
		fromStep      string
		expectedSteps []string
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
	executor := NewDefaultPipelineExecutor(adaptertest.NewMockAdapter())
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
		name           string
		setupWorkspace func(t *testing.T, tempDir string)
		expectedPoint  string
		expectError    bool
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
				specWorkspace := filepath.Join(tempDir, ".agents/workspaces/prototype/spec")
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
					phaseWorkspace := filepath.Join(tempDir, ".agents/workspaces/prototype", phase.phase)
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
					phaseWorkspace := filepath.Join(tempDir, ".agents/workspaces/prototype", phase.phase)
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
			defer func() { _ = os.Chdir(originalWd) }()

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
	executor := NewDefaultPipelineExecutor(adaptertest.NewMockAdapter())
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
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Setup workspace with stale artifacts scenario
	baseTime := time.Now().Add(-1 * time.Hour)

	// Create docs workspace (older)
	docsWorkspace := filepath.Join(tempDir, ".agents/workspaces/prototype/docs")
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
	specWorkspace := filepath.Join(tempDir, ".agents/workspaces/prototype/spec")
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
	dummyWorkspace := filepath.Join(tempDir, ".agents/workspaces/prototype/dummy")
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
	executor := NewDefaultPipelineExecutor(adaptertest.NewMockAdapter())
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

func TestLoadResumeState_WithPriorRunID(t *testing.T) {
	executor := NewDefaultPipelineExecutor(adaptertest.NewMockAdapter())
	manager := NewResumeManager(executor)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "speckit-flow"},
		Steps: []Step{
			{
				ID:        "specify",
				Workspace: WorkspaceConfig{Type: "worktree"},
				OutputArtifacts: []ArtifactDef{
					{Name: "spec-status", Path: ".agents/output/specify-status.json"},
				},
			},
			{
				ID:           "clarify",
				Dependencies: []string{"specify"},
				Workspace:    WorkspaceConfig{Type: "worktree"},
				OutputArtifacts: []ArtifactDef{
					{Name: "clarify-status", Path: ".agents/output/clarify-status.json"},
				},
			},
			{
				ID:           "checklist",
				Dependencies: []string{"clarify"},
				Workspace:    WorkspaceConfig{Type: "worktree"},
			},
		},
	}

	tests := []struct {
		name              string
		priorRunID        string
		fromStep          string
		setupWorkspace    func(t *testing.T, tmpDir string)
		expectedCompleted []string
		expectedArtifacts int
		checkArtifactPath string // verify a specific artifact path contains the run ID
	}{
		{
			name:       "resolves artifacts from specified run ID",
			priorRunID: "speckit-flow-20260306-084028-bd46",
			fromStep:   "checklist",
			setupWorkspace: func(t *testing.T, tmpDir string) {
				// Create the specific run's worktree workspace with artifacts
				wtDir := filepath.Join(tmpDir, ".agents/workspaces/speckit-flow-20260306-084028-bd46/__wt_speckit-flow-20260306-084028-bd46")
				if err := os.MkdirAll(filepath.Join(wtDir, ".agents/output"), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(wtDir, ".agents/output/specify-status.json"), []byte(`{"status":"done"}`), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(wtDir, ".agents/output/clarify-status.json"), []byte(`{"status":"done"}`), 0644); err != nil {
					t.Fatal(err)
				}
			},
			expectedCompleted: []string{"specify", "clarify"},
			expectedArtifacts: 2,
			checkArtifactPath: "speckit-flow-20260306-084028-bd46",
		},
		{
			name:       "prefers specified run over newer run",
			priorRunID: "speckit-flow-20260306-084028-old1",
			fromStep:   "clarify",
			setupWorkspace: func(t *testing.T, tmpDir string) {
				// Create the specified (older) run's workspace
				oldDir := filepath.Join(tmpDir, ".agents/workspaces/speckit-flow-20260306-084028-old1/__wt_branch-old")
				if err := os.MkdirAll(filepath.Join(oldDir, ".agents/output"), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(oldDir, ".agents/output/specify-status.json"), []byte(`{"status":"old"}`), 0644); err != nil {
					t.Fatal(err)
				}

				// Create a newer run's workspace that should NOT be used
				newDir := filepath.Join(tmpDir, ".agents/workspaces/speckit-flow-20260306-200000-new2/__wt_branch-new")
				if err := os.MkdirAll(filepath.Join(newDir, ".agents/output"), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(newDir, ".agents/output/specify-status.json"), []byte(`{"status":"new"}`), 0644); err != nil {
					t.Fatal(err)
				}
			},
			expectedCompleted: []string{"specify"},
			expectedArtifacts: 1,
			checkArtifactPath: "speckit-flow-20260306-084028-old1",
		},
		{
			name:       "falls back to glob scan when run ID dir does not exist",
			priorRunID: "speckit-flow-nonexistent",
			fromStep:   "clarify",
			setupWorkspace: func(t *testing.T, tmpDir string) {
				// Only the glob-matched run exists
				wtDir := filepath.Join(tmpDir, ".agents/workspaces/speckit-flow-20260306-100000-abcd/__wt_some-branch")
				if err := os.MkdirAll(filepath.Join(wtDir, ".agents/output"), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(wtDir, ".agents/output/specify-status.json"), []byte(`{"status":"fallback"}`), 0644); err != nil {
					t.Fatal(err)
				}
			},
			expectedCompleted: []string{"specify"},
			expectedArtifacts: 1,
			checkArtifactPath: "speckit-flow-20260306-100000-abcd",
		},
		{
			name:       "no run ID uses default glob behavior",
			priorRunID: "",
			fromStep:   "clarify",
			setupWorkspace: func(t *testing.T, tmpDir string) {
				wtDir := filepath.Join(tmpDir, ".agents/workspaces/speckit-flow-20260306-100000-abcd/__wt_some-branch")
				if err := os.MkdirAll(filepath.Join(wtDir, ".agents/output"), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(wtDir, ".agents/output/specify-status.json"), []byte(`{"status":"glob"}`), 0644); err != nil {
					t.Fatal(err)
				}
			},
			expectedCompleted: []string{"specify"},
			expectedArtifacts: 1,
			checkArtifactPath: "speckit-flow-20260306-100000-abcd",
		},
		{
			name:       "resolves non-worktree step artifacts from specified run",
			priorRunID: "speckit-flow-20260306-084028-bd46",
			fromStep:   "clarify",
			setupWorkspace: func(t *testing.T, tmpDir string) {
				// Create a basic (non-worktree) step workspace under the specified run
				stepDir := filepath.Join(tmpDir, ".agents/workspaces/speckit-flow-20260306-084028-bd46/specify")
				if err := os.MkdirAll(filepath.Join(stepDir, ".agents/output"), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(stepDir, ".agents/output/specify-status.json"), []byte(`{"status":"basic"}`), 0644); err != nil {
					t.Fatal(err)
				}
			},
			expectedCompleted: []string{"specify"},
			expectedArtifacts: 1,
			checkArtifactPath: "speckit-flow-20260306-084028-bd46",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			origDir, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = os.Chdir(origDir) }()

			if err := os.Chdir(tmpDir); err != nil {
				t.Fatal(err)
			}

			tt.setupWorkspace(t, tmpDir)

			var state *ResumeState
			if tt.priorRunID != "" {
				state, err = manager.loadResumeState(p, tt.fromStep, tt.priorRunID)
			} else {
				state, err = manager.loadResumeState(p, tt.fromStep)
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(state.CompletedSteps) != len(tt.expectedCompleted) {
				t.Errorf("Expected %d completed steps, got %d: %v",
					len(tt.expectedCompleted), len(state.CompletedSteps), state.CompletedSteps)
			}

			for i, expected := range tt.expectedCompleted {
				if i < len(state.CompletedSteps) && state.CompletedSteps[i] != expected {
					t.Errorf("Expected completed step %d to be %s, got %s",
						i, expected, state.CompletedSteps[i])
				}
			}

			if len(state.ArtifactPaths) != tt.expectedArtifacts {
				t.Errorf("Expected %d artifact paths, got %d: %v",
					tt.expectedArtifacts, len(state.ArtifactPaths), state.ArtifactPaths)
			}

			// Verify artifact paths contain the expected run ID
			if tt.checkArtifactPath != "" {
				for key, path := range state.ArtifactPaths {
					if !strings.Contains(path, tt.checkArtifactPath) {
						t.Errorf("Artifact path for %q should contain %q, got %q",
							key, tt.checkArtifactPath, path)
					}
				}
			}
		})
	}
}

func TestLoadResumeState_LoadsFailureContext(t *testing.T) {
	mockAdapter := adaptertest.NewMockAdapter()
	executor := NewDefaultPipelineExecutor(mockAdapter)

	// Wire a mock store with step attempt data
	store := &resumeMockStore{
		attempts: map[string][]state.StepAttemptRecord{
			"prior-run:implement": {
				{
					RunID:        "prior-run",
					StepID:       "implement",
					Attempt:      1,
					State:        "failed",
					ErrorMessage: "contract validation failed: missing field 'status'",
					FailureClass: "contract_failure",
					StdoutTail:   "wrote file.go\nran tests\nfailed at validation",
				},
			},
		},
	}
	executor.store = store

	manager := NewResumeManager(executor)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "speckit-flow"},
		Steps: []Step{
			{ID: "specify"},
			{ID: "implement", Dependencies: []string{"specify"}},
		},
	}

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	// Create workspace for specify step
	specDir := filepath.Join(tmpDir, ".agents/workspaces/prior-run/specify")
	_ = os.MkdirAll(specDir, 0755)

	rs, err := manager.loadResumeState(p, "implement", "prior-run")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify failure context was loaded
	ctx, ok := rs.FailureContexts["implement"]
	if !ok {
		t.Fatal("Expected failure context for 'implement' step, got none")
	}
	if ctx.Attempt != 1 {
		t.Errorf("Expected attempt 1, got %d", ctx.Attempt)
	}
	if ctx.PriorError != "contract validation failed: missing field 'status'" {
		t.Errorf("Expected prior error, got %q", ctx.PriorError)
	}
	if ctx.FailureClass != "contract_failure" {
		t.Errorf("Expected failure class 'contract_failure', got %q", ctx.FailureClass)
	}
	if ctx.PriorStdout != "wrote file.go\nran tests\nfailed at validation" {
		t.Errorf("Expected prior stdout, got %q", ctx.PriorStdout)
	}
}

func TestLoadResumeState_NoFailureContextWithoutStore(t *testing.T) {
	mockAdapter := adaptertest.NewMockAdapter()
	executor := NewDefaultPipelineExecutor(mockAdapter)
	// No store set — executor.store is nil
	manager := NewResumeManager(executor)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "test-pipeline"},
		Steps: []Step{
			{ID: "step1"},
			{ID: "step2", Dependencies: []string{"step1"}},
		},
	}

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	rs, err := manager.loadResumeState(p, "step2", "some-run")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(rs.FailureContexts) != 0 {
		t.Errorf("Expected empty failure contexts without store, got %d", len(rs.FailureContexts))
	}
}

func TestLoadResumeState_NoFailureContextWhenStepSucceeded(t *testing.T) {
	mockAdapter := adaptertest.NewMockAdapter()
	executor := NewDefaultPipelineExecutor(mockAdapter)

	store := &resumeMockStore{
		attempts: map[string][]state.StepAttemptRecord{
			"prior-run:implement": {
				{
					RunID:   "prior-run",
					StepID:  "implement",
					Attempt: 1,
					State:   "succeeded",
				},
			},
		},
	}
	executor.store = store
	manager := NewResumeManager(executor)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "test-pipeline"},
		Steps: []Step{
			{ID: "step1"},
			{ID: "implement", Dependencies: []string{"step1"}},
		},
	}

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	rs, err := manager.loadResumeState(p, "implement", "prior-run")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if _, ok := rs.FailureContexts["implement"]; ok {
		t.Error("Should not load failure context when last attempt succeeded")
	}
}

// resumeMockStore implements state.StateStore for resume failure context tests.
type resumeMockStore struct {
	testutil.MockStateStore
	attempts map[string][]state.StepAttemptRecord // key: "runID:stepID"
}

func (s *resumeMockStore) GetStepAttempts(runID string, stepID string) ([]state.StepAttemptRecord, error) {
	key := runID + ":" + stepID
	return s.attempts[key], nil
}

func TestResumeFromStepWithForceSkipsValidation(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
		adaptertest.WithTokensUsed(500),
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

// TestResumeWithExcludeFilter verifies --from-step + -x combo works correctly.
// When resuming from "step-b" with -x "step-c", only step-b should execute.
func TestResumeWithExcludeFilter(t *testing.T) {
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
		adaptertest.WithTokensUsed(100),
	)

	collector := testutil.NewEventCollector()
	filter := &StepFilter{Exclude: []string{"step-c"}}
	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
		WithStepFilter(filter),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "resume-exclude-test"},
		Steps: []Step{
			{ID: "step-a", Persona: "navigator", Exec: ExecConfig{Source: "A"}},
			{ID: "step-b", Persona: "navigator", Dependencies: []string{"step-a"}, Exec: ExecConfig{Source: "B"}},
			{ID: "step-c", Persona: "navigator", Dependencies: []string{"step-b"}, Exec: ExecConfig{Source: "C"}},
		},
	}

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Create workspace for step-a to simulate prior completion
	stepAWs := filepath.Join(tmpDir, ".agents/workspaces/resume-exclude-test/step-a")
	if err := os.MkdirAll(stepAWs, 0755); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	manager := NewResumeManager(executor)
	_ = manager.ResumeFromStep(ctx, p, m, "test", "step-b", true)
	// The execution itself may fail (mock adapter etc.) but the filter should
	// have removed step-c from the execution plan
	order := collector.GetStepExecutionOrder()
	for _, stepID := range order {
		if stepID == "step-c" {
			t.Error("step-c should have been excluded by the filter")
		}
	}
}

// TestExecuteResumedPipeline_ReturnsStepError verifies that errors from
// executeResumedPipeline are wrapped as *StepExecutionError so the CLI can extract
// the step ID via errors.As() for recovery hints.
func TestExecuteResumedPipeline_ReturnsStepExecutionError(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	// Create a mock adapter that always fails
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithFailure(fmt.Errorf("adapter crashed")),
	)

	collector := testutil.NewEventCollector()
	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)
	manager := NewResumeManager(executor)

	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "test-resume-steperr"},
		Steps: []Step{
			{ID: "step-a", Persona: "navigator", Exec: ExecConfig{Source: "A"}},
			{ID: "step-b", Persona: "navigator", Dependencies: []string{"step-a"}, Exec: ExecConfig{Source: "B"}},
		},
	}

	// Create workspace for step-a to simulate prior completion
	stepAWs := filepath.Join(tmpDir, ".agents/workspaces/test-resume-steperr/step-a")
	if err := os.MkdirAll(stepAWs, 0755); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := manager.ResumeFromStep(ctx, p, m, "test input", "step-b", true)
	if err == nil {
		t.Fatal("expected error from resumed pipeline, got nil")
	}

	// Verify the error is a *StepExecutionError
	var stepErr *StepExecutionError
	if !errors.As(err, &stepErr) {
		t.Fatalf("expected error to be *StepExecutionError, got %T: %v", err, err)
	}

	if stepErr.StepID != "step-b" {
		t.Errorf("expected StepExecutionError.StepID = %q, got %q", "step-b", stepErr.StepID)
	}

	// Verify the original error is preserved
	if !strings.Contains(stepErr.Err.Error(), "adapter crashed") {
		t.Errorf("expected original error to contain 'adapter crashed', got %q", stepErr.Err.Error())
	}

	// Verify a "failed" event was emitted
	foundFailed := false
	for _, ev := range collector.GetEvents() {
		if ev.StepID == "step-b" && ev.State == "failed" {
			foundFailed = true
			break
		}
	}
	if !foundFailed {
		t.Error("expected a 'failed' event for step-b to be emitted")
	}
}

// TestResumeNonPrototypePipeline verifies that PhaseSkipValidator does not
// reject non-prototype pipelines when valid prior state exists.
func TestResumeNonPrototypePipeline(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
		adaptertest.WithTokensUsed(500),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter)
	manager := NewResumeManager(executor)

	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "impl-issue"},
		Steps: []Step{
			{ID: "fetch-assess", Persona: "navigator", Exec: ExecConfig{Source: "assess"}},
			{ID: "plan", Persona: "navigator", Dependencies: []string{"fetch-assess"}, Exec: ExecConfig{Source: "plan"}},
			{ID: "implement", Persona: "craftsman", Dependencies: []string{"plan"}, Exec: ExecConfig{Source: "implement"}},
		},
	}

	// Create workspace for fetch-assess and plan to simulate prior completion
	for _, stepID := range []string{"fetch-assess", "plan"} {
		wsDir := filepath.Join(tmpDir, ".agents/workspaces/impl-issue-20260316-000000-abcd", stepID)
		if err := os.MkdirAll(wsDir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Without force: should pass validation (non-prototype pipelines now validate
	// workspace existence)
	err := manager.ResumeFromStep(ctx, p, m, "test", "implement", false)
	// The execution itself may fail (mock adapter), but the phase validation
	// should NOT reject it
	if err != nil && strings.Contains(err.Error(), "prerequisite phase") {
		t.Errorf("non-prototype pipeline should not get prototype phase validation error, got: %v", err)
	}
	if err != nil && strings.Contains(err.Error(), "no prior run state") {
		t.Errorf("should find prior run state, got: %v", err)
	}
}

// TestResumeNonPrototype_NoRunStateFails verifies that resuming a non-prototype
// pipeline without any prior run state fails validation (unless --force is used).
func TestResumeNonPrototype_NoRunStateFails(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	executor := NewDefaultPipelineExecutor(adaptertest.NewMockAdapter())
	manager := NewResumeManager(executor)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "impl-issue"},
		Steps: []Step{
			{ID: "fetch-assess"},
			{ID: "plan", Dependencies: []string{"fetch-assess"}},
			{ID: "implement", Dependencies: []string{"plan"}},
		},
	}

	validator := NewPhaseSkipValidator()

	// No workspaces exist — should fail validation
	err := validator.ValidatePhaseSequence(p, "implement")
	if err == nil {
		t.Error("expected validation error when no prior run state exists")
	}
	if err != nil && !strings.Contains(err.Error(), "no prior run state") {
		t.Errorf("expected 'no prior run state' error, got: %v", err)
	}

	// Starting from first step should pass (no prior work needed)
	err = validator.ValidatePhaseSequence(p, "fetch-assess")
	if err != nil {
		t.Errorf("starting from first step should not require prior state, got: %v", err)
	}

	// Force should skip validation entirely (tested via ResumeFromStep)
	_ = manager // prevent unused
}

// TestGetRecommendedResumePoint_NonPrototype verifies that GetRecommendedResumePoint
// works for non-prototype pipelines.
func TestGetRecommendedResumePoint_NonPrototype(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	executor := NewDefaultPipelineExecutor(adaptertest.NewMockAdapter())
	manager := NewResumeManager(executor)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "impl-issue"},
		Steps: []Step{
			{ID: "fetch-assess"},
			{ID: "plan", Dependencies: []string{"fetch-assess"}},
			{ID: "implement", Dependencies: []string{"plan"}},
		},
	}

	// No workspaces — should recommend first step
	point, err := manager.GetRecommendedResumePoint(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if point != "fetch-assess" {
		t.Errorf("expected first step with no state, got %q", point)
	}

	// Create workspace for fetch-assess only
	runDir := filepath.Join(tmpDir, ".agents/workspaces/impl-issue-20260316-000000-abcd")
	if err := os.MkdirAll(filepath.Join(runDir, "fetch-assess"), 0755); err != nil {
		t.Fatal(err)
	}

	point, err = manager.GetRecommendedResumePoint(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if point != "plan" {
		t.Errorf("expected 'plan' with only fetch-assess complete, got %q", point)
	}
}

// capturingMockStore wraps MockStateStore and captures RecordStepAttempt calls.
type capturingMockStore struct {
	*testutil.MockStateStore
	mu       sync.Mutex
	attempts []*state.StepAttemptRecord
}

func (s *capturingMockStore) RecordStepAttempt(record *state.StepAttemptRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attempts = append(s.attempts, record)
	return nil
}

func (s *capturingMockStore) getAttempts() []*state.StepAttemptRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]*state.StepAttemptRecord, len(s.attempts))
	copy(cp, s.attempts)
	return cp
}

// TestFailureClassRecordedOnStepAttempt verifies that the executor populates
// FailureClass on StepAttemptRecord when recording a failed step.
func TestFailureClassRecordedOnStepAttempt(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	// Use a failing adapter
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithFailure(fmt.Errorf("runtime failure: process exited with code 1")),
	)

	store := &capturingMockStore{MockStateStore: testutil.NewMockStateStore()}
	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithStateStore(store),
		WithEmitter(testutil.NewEventCollector()),
	)

	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "test-failure-class"},
		Steps: []Step{
			{ID: "step-a", Persona: "navigator", Exec: ExecConfig{Source: "do something"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_ = executor.Execute(ctx, p, m, "test input")

	// Check that at least one attempt was recorded with a FailureClass
	attempts := store.getAttempts()
	if len(attempts) == 0 {
		t.Fatal("expected at least one recorded step attempt, got none")
	}

	// The adaptertest.WithFailure returns a generic error (not contract/security/preflight),
	// so it should be classified as "unknown" by recovery.ClassifyError()
	lastAttempt := attempts[len(attempts)-1]
	if lastAttempt.FailureClass == "" {
		t.Error("expected FailureClass to be set on failed step attempt, got empty string")
	}
	if lastAttempt.StepID != "step-a" {
		t.Errorf("expected step ID 'step-a', got %q", lastAttempt.StepID)
	}
	if lastAttempt.State != "failed" {
		t.Errorf("expected state 'failed', got %q", lastAttempt.State)
	}
}

// TestResumeFromStep_ReusesExecutorRunID verifies that ResumeFromStep does not call
// store.CreateRun when executor.runID is already set via WithRunID.
//
// This is the unit-level guard for the second fix in issue #700: before the fix,
// ResumeFromStep always called createRunID() which called store.CreateRun(), creating
// a third phantom run record even though the executor already had a pre-assigned run ID.
func TestResumeFromStep_ReusesExecutorRunID(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	var mu sync.Mutex
	createCount := 0

	store := testutil.NewMockStateStore(
		testutil.WithCreateRun(func(pipelineName, input string) (string, error) {
			mu.Lock()
			createCount++
			mu.Unlock()
			return "should-not-be-called", nil
		}),
	)

	executor := NewDefaultPipelineExecutor(
		adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		),
		WithRunID("preset-run-id"),
		WithStateStore(store),
	)
	manager := NewResumeManager(executor)

	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "phantom-run-test"},
		Steps: []Step{
			{ID: "step1", Persona: "navigator", Exec: ExecConfig{Source: "step 1 work"}},
			{ID: "step2", Persona: "navigator", Dependencies: []string{"step1"}, Exec: ExecConfig{Source: "step 2 work"}},
		},
	}

	// Create workspace for step1 to simulate prior completion
	step1Ws := filepath.Join(tmpDir, ".agents/workspaces/phantom-run-test/step1")
	if err := os.MkdirAll(step1Ws, 0755); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// force=true skips phase validation; we only care that CreateRun is not called.
	// The execution may fail (mock adapter / missing files) but that is expected.
	_ = manager.ResumeFromStep(ctx, p, m, "test input", "step2", true)

	mu.Lock()
	count := createCount
	mu.Unlock()

	if count != 0 {
		t.Errorf("expected 0 CreateRun calls when executor.runID is pre-set via WithRunID, got %d", count)
	}
}

// initGitRepo initializes a minimal git repo with a GitHub remote for forge detection.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"git", "init"},
		{"git", "remote", "add", "origin", "https://github.com/test-owner/test-repo.git"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git init failed: %s: %v", out, err)
		}
	}
}

func TestResumeFromStep_InjectsForgeVariables(t *testing.T) {
	executor := NewDefaultPipelineExecutor(adaptertest.NewMockAdapter())
	manager := NewResumeManager(executor)

	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(tempDir)
	defer func() { _ = os.Chdir(origDir) }()

	initGitRepo(t, tempDir)

	p := &Pipeline{
		Kind:     "WavePipeline",
		Metadata: PipelineMetadata{Name: "test-forge-resume"},
		Steps: []Step{
			{ID: "implement"},
			{
				ID:           "create-pr",
				Persona:      "{{ forge.type }}-commenter",
				Dependencies: []string{"implement"},
			},
		},
	}

	m := &manifest.Manifest{}
	wsDir := filepath.Join(tempDir, ".agents", "workspaces", "test-forge-resume", "implement")
	_ = os.MkdirAll(wsDir, 0755)
	_ = os.WriteFile(filepath.Join(wsDir, "artifact.json"), []byte("{}"), 0644)

	ctx := context.Background()
	_ = manager.ResumeFromStep(ctx, p, m, "test-input", "create-pr", true)

	executor.mu.RLock()
	defer executor.mu.RUnlock()

	var execution *PipelineExecution
	for _, exec := range executor.pipelines {
		if exec.Pipeline.Metadata.Name == "test-forge-resume" {
			execution = exec
			break
		}
	}
	if execution == nil {
		t.Fatal("expected execution to be stored in executor.pipelines")
	}

	forgeType := execution.Context.CustomVariables["forge.type"]
	if forgeType == "" {
		t.Error("forge.type not injected into resume context")
	}

	resolved := execution.Context.ResolvePlaceholders("{{ forge.type }}-commenter")
	if strings.Contains(resolved, "{{") {
		t.Errorf("persona template not resolved: got %q", resolved)
	}
}

func TestResumeFromStep_ReusesWorktree(t *testing.T) {
	executor := NewDefaultPipelineExecutor(adaptertest.NewMockAdapter())
	manager := NewResumeManager(executor)

	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(tempDir)
	defer func() { _ = os.Chdir(origDir) }()

	initGitRepo(t, tempDir)

	p := &Pipeline{
		Kind:     "WavePipeline",
		Metadata: PipelineMetadata{Name: "test-wt-resume"},
		Steps: []Step{
			{
				ID:        "implement",
				Workspace: WorkspaceConfig{Type: "worktree", Branch: "{{ pipeline_id }}"},
				OutputArtifacts: []ArtifactDef{
					{Name: "result", Path: ".agents/output/result.json"},
				},
			},
			{
				ID:           "create-pr",
				Workspace:    WorkspaceConfig{Type: "worktree", Branch: "{{ pipeline_id }}"},
				Dependencies: []string{"implement"},
			},
		},
	}

	m := &manifest.Manifest{}

	priorRunID := "test-wt-resume-20260412-195527-41ee"
	wtDir := filepath.Join(tempDir, ".agents", "workspaces", priorRunID, "__wt_"+priorRunID)
	_ = os.MkdirAll(filepath.Join(wtDir, ".agents", "output"), 0755)
	_ = os.WriteFile(filepath.Join(wtDir, ".agents", "output", "result.json"), []byte(`{"ok":true}`), 0644)

	ctx := context.Background()
	executor.runID = "test-wt-resume-20260413-new-run"
	_ = manager.ResumeFromStep(ctx, p, m, "test-input", "create-pr", true, priorRunID)

	executor.mu.RLock()
	defer executor.mu.RUnlock()

	var execution *PipelineExecution
	for _, exec := range executor.pipelines {
		if exec.Pipeline.Metadata.Name == "test-wt-resume" {
			execution = exec
			break
		}
	}
	if execution == nil {
		t.Fatal("expected execution to be stored in executor.pipelines")
	}

	expectedBranch := "test-wt-resume-20260413-new-run"
	wtInfo, exists := execution.WorktreePaths[expectedBranch]
	if !exists {
		t.Fatalf("WorktreePaths[%q] not seeded from prior run; keys: %v",
			expectedBranch, func() []string {
				keys := make([]string, 0, len(execution.WorktreePaths))
				for k := range execution.WorktreePaths {
					keys = append(keys, k)
				}
				return keys
			}())
	}

	absWt, _ := filepath.Abs(wtDir)
	if wtInfo.AbsPath != absWt {
		t.Errorf("WorktreePaths AbsPath = %q, want %q", wtInfo.AbsPath, absWt)
	}
}
