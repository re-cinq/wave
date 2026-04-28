package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPhaseSkipValidator(t *testing.T) {
	validator := NewPhaseSkipValidator()

	// Create a test prototype pipeline
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
			name:           "start from beginning",
			fromStep:       "",
			setupWorkspace: func(t *testing.T, tempDir string) {},
			expectError:    false,
		},
		{
			name:           "resume from spec with no prerequisites",
			fromStep:       "spec",
			setupWorkspace: func(t *testing.T, tempDir string) {},
			expectError:    false,
		},
		{
			name:     "resume from docs with completed spec",
			fromStep: "docs",
			setupWorkspace: func(t *testing.T, tempDir string) {
				// Create completed spec phase workspace
				specWorkspace := filepath.Join(tempDir, ".agents/workspaces/prototype/spec")
				err := os.MkdirAll(specWorkspace, 0755)
				if err != nil {
					t.Fatal(err)
				}

				// Create required artifacts
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
			name:           "skip to docs without completed spec",
			fromStep:       "docs",
			setupWorkspace: func(t *testing.T, tempDir string) {},
			expectError:    true,
			errorContains:  "prerequisite phase 'spec' not completed",
		},
		{
			name:     "skip to dummy without completed docs",
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
		{
			name:     "resume from implement with all prerequisites",
			fromStep: "implement",
			setupWorkspace: func(t *testing.T, tempDir string) {
				phases := []struct {
					phase string
					files []string
				}{
					{"spec", []string{"artifact.json", "spec.md"}},
					{"docs", []string{"artifact.json", "feature-docs.md"}},
					{"dummy", []string{"artifact.json", "interfaces.md"}},
				}

				for _, phase := range phases {
					phaseWorkspace := filepath.Join(tempDir, ".agents/workspaces/prototype", phase.phase)
					err := os.MkdirAll(phaseWorkspace, 0755)
					if err != nil {
						t.Fatal(err)
					}

					for _, file := range phase.files {
						if file == "prototype/" {
							err = os.MkdirAll(filepath.Join(phaseWorkspace, "prototype"), 0755)
						} else {
							err = os.WriteFile(filepath.Join(phaseWorkspace, file), []byte("test content"), 0644)
						}
						if err != nil {
							t.Fatal(err)
						}
					}

					// Create prototype directory for dummy phase
					if phase.phase == "dummy" {
						err = os.MkdirAll(filepath.Join(phaseWorkspace, "prototype"), 0755)
						if err != nil {
							t.Fatal(err)
						}
					}
				}
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup temporary directory
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

			err = validator.ValidatePhaseSequence(pipeline, tt.fromStep)

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

func TestStaleArtifactDetector(t *testing.T) {
	detector := NewStaleArtifactDetector()

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
		currentStep     string
		setupWorkspace  func(t *testing.T, tempDir string)
		expectStale     bool
		expectedReasons int
	}{
		{
			name:        "no upstream changes",
			currentStep: "docs",
			setupWorkspace: func(t *testing.T, tempDir string) {
				baseTime := time.Now().Add(-1 * time.Hour)

				// Create spec workspace (older)
				specWorkspace := filepath.Join(tempDir, ".agents/workspaces/prototype/spec")
				err := os.MkdirAll(specWorkspace, 0755)
				if err != nil {
					t.Fatal(err)
				}
				specFile := filepath.Join(specWorkspace, "spec.md")
				err = os.WriteFile(specFile, []byte("spec content"), 0644)
				if err != nil {
					t.Fatal(err)
				}
				err = os.Chtimes(specFile, baseTime, baseTime)
				if err != nil {
					t.Fatal(err)
				}

				// Create docs workspace (newer)
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
				err = os.Chtimes(docsFile, baseTime.Add(30*time.Minute), baseTime.Add(30*time.Minute))
				if err != nil {
					t.Fatal(err)
				}
			},
			expectStale:     false,
			expectedReasons: 0,
		},
		{
			name:        "upstream spec phase re-run",
			currentStep: "docs",
			setupWorkspace: func(t *testing.T, tempDir string) {
				baseTime := time.Now().Add(-1 * time.Hour)

				// Create docs workspace (older)
				docsWorkspace := filepath.Join(tempDir, ".agents/workspaces/prototype/docs")
				err := os.MkdirAll(docsWorkspace, 0755)
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

				// Create spec workspace (newer - re-run after docs)
				specWorkspace := filepath.Join(tempDir, ".agents/workspaces/prototype/spec")
				err = os.MkdirAll(specWorkspace, 0755)
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
			},
			expectStale:     true,
			expectedReasons: 1,
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

			reasons, err := detector.DetectStaleArtifacts(pipeline, tt.currentStep)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.expectStale {
				if len(reasons) != tt.expectedReasons {
					t.Errorf("Expected %d stale reasons, got %d: %v", tt.expectedReasons, len(reasons), reasons)
				}
			} else {
				if len(reasons) > 0 {
					t.Errorf("Expected no stale artifacts, but got: %v", reasons)
				}
			}
		})
	}
}

func TestErrorMessageProvider(t *testing.T) {
	provider := NewErrorMessageProvider()

	tests := []struct {
		name             string
		phase            string
		originalError    error
		expectedContains []string
	}{
		{
			name:          "spec phase failure",
			phase:         "spec",
			originalError: fmt.Errorf("failed to create spec.md"),
			expectedContains: []string{
				"Phase 'spec' failed",
				"craftsman persona has write permissions",
				"wave run prototype --from-step spec",
				"Workspace: .agents/workspaces/prototype/spec",
			},
		},
		{
			name:          "docs phase failure",
			phase:         "docs",
			originalError: fmt.Errorf("artifact injection failed"),
			expectedContains: []string{
				"Phase 'docs' failed",
				".agents/artifacts/input-spec.md is accessible",
				"wave run prototype --from-step docs",
			},
		},
		{
			name:          "dummy phase failure",
			phase:         "dummy",
			originalError: fmt.Errorf("prototype generation failed"),
			expectedContains: []string{
				"Phase 'dummy' failed",
				"prototype/ directory is created",
				"interfaces.md documents all interfaces",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.FormatPhaseFailureError(tt.phase, tt.originalError)
			errorMsg := err.Error()

			for _, expected := range tt.expectedContains {
				if !strings.Contains(errorMsg, expected) {
					t.Errorf("Error message should contain %q, but got:\n%s", expected, errorMsg)
				}
			}
		})
	}
}

func TestConcurrencyValidator(t *testing.T) {
	validator := NewConcurrencyValidator()

	tests := []struct {
		name          string
		action        func() error
		expectError   bool
		errorContains string
	}{
		{
			name: "acquire first lock",
			action: func() error {
				return validator.AcquireWorkspaceLock("pipeline1", "workspace1")
			},
			expectError: false,
		},
		{
			name: "acquire same workspace again",
			action: func() error {
				_ = validator.AcquireWorkspaceLock("pipeline1", "workspace1")
				return validator.AcquireWorkspaceLock("pipeline2", "workspace1")
			},
			expectError:   true,
			errorContains: "workspace 'workspace1' is already in use",
		},
		{
			name: "acquire same pipeline again",
			action: func() error {
				_ = validator.AcquireWorkspaceLock("pipeline1", "workspace1")
				return validator.AcquireWorkspaceLock("pipeline1", "workspace2")
			},
			expectError:   true,
			errorContains: "pipeline 'pipeline1' is already running",
		},
		{
			name: "acquire after release",
			action: func() error {
				_ = validator.AcquireWorkspaceLock("pipeline1", "workspace1")
				validator.ReleaseWorkspaceLock("pipeline1")
				return validator.AcquireWorkspaceLock("pipeline2", "workspace1")
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset validator for each test
			validator = NewConcurrencyValidator()

			err := tt.action()

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

func TestConcurrencyValidator_IsWorkspaceInUse(t *testing.T) {
	validator := NewConcurrencyValidator()

	// Initially no workspace should be in use
	if validator.IsWorkspaceInUse("workspace1") {
		t.Error("Expected workspace1 to not be in use initially")
	}

	// Acquire lock
	err := validator.AcquireWorkspaceLock("pipeline1", "workspace1")
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	// Now workspace should be in use
	if !validator.IsWorkspaceInUse("workspace1") {
		t.Error("Expected workspace1 to be in use after acquiring lock")
	}

	// Different workspace should not be in use
	if validator.IsWorkspaceInUse("workspace2") {
		t.Error("Expected workspace2 to not be in use")
	}

	// Release lock
	validator.ReleaseWorkspaceLock("pipeline1")

	// Now workspace should not be in use
	if validator.IsWorkspaceInUse("workspace1") {
		t.Error("Expected workspace1 to not be in use after releasing lock")
	}
}

func TestConcurrencyValidator_GetRunningPipelines(t *testing.T) {
	validator := NewConcurrencyValidator()

	// Initially no pipelines running
	running := validator.GetRunningPipelines()
	if len(running) != 0 {
		t.Errorf("Expected no running pipelines, got %v", running)
	}

	// Acquire some locks
	_ = validator.AcquireWorkspaceLock("pipeline1", "workspace1")
	_ = validator.AcquireWorkspaceLock("pipeline2", "workspace2")

	// Check running pipelines
	running = validator.GetRunningPipelines()
	if len(running) != 2 {
		t.Errorf("Expected 2 running pipelines, got %d", len(running))
	}

	if running["pipeline1"] != "workspace1" {
		t.Errorf("Expected pipeline1 to use workspace1, got %s", running["pipeline1"])
	}

	if running["pipeline2"] != "workspace2" {
		t.Errorf("Expected pipeline2 to use workspace2, got %s", running["pipeline2"])
	}

	// Release one lock
	validator.ReleaseWorkspaceLock("pipeline1")

	// Check running pipelines again
	running = validator.GetRunningPipelines()
	if len(running) != 1 {
		t.Errorf("Expected 1 running pipeline, got %d", len(running))
	}

	if _, exists := running["pipeline1"]; exists {
		t.Error("Expected pipeline1 to not be in running pipelines after release")
	}
}

func TestValidateThreadFields(t *testing.T) {
	tests := []struct {
		name        string
		steps       []Step
		expectErrs  int
		errContains string
	}{
		{
			name: "valid thread with default fidelity",
			steps: []Step{
				{ID: "impl", Thread: "impl"},
				{ID: "fix", Thread: "impl"},
			},
			expectErrs: 0,
		},
		{
			name: "valid thread with explicit fidelity",
			steps: []Step{
				{ID: "impl", Thread: "impl", Fidelity: "full"},
				{ID: "fix", Thread: "impl", Fidelity: "compact"},
			},
			expectErrs: 0,
		},
		{
			name: "no thread no fidelity is valid",
			steps: []Step{
				{ID: "review"},
			},
			expectErrs: 0,
		},
		{
			name: "unknown fidelity value",
			steps: []Step{
				{ID: "impl", Thread: "impl", Fidelity: "invalid"},
			},
			expectErrs:  1,
			errContains: "unknown fidelity value",
		},
		{
			name: "fidelity without thread",
			steps: []Step{
				{ID: "review", Fidelity: "full"},
			},
			expectErrs:  1,
			errContains: "fidelity has no effect without a thread group",
		},
		{
			name: "multiple errors",
			steps: []Step{
				{ID: "step-a", Thread: "impl", Fidelity: "bogus"},
				{ID: "step-b", Fidelity: "compact"},
			},
			expectErrs: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pipeline{
				Metadata: PipelineMetadata{Name: "test"},
				Steps:    tt.steps,
			}
			errs := ValidateThreadFields(p)
			if len(errs) != tt.expectErrs {
				t.Errorf("expected %d errors, got %d: %v", tt.expectErrs, len(errs), errs)
			}
			if tt.errContains != "" && len(errs) > 0 {
				found := false
				for _, err := range errs {
					if strings.Contains(err.Error(), tt.errContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error containing %q, got %v", tt.errContains, errs)
				}
			}
		})
	}
}

// TestValidateGenericStepSequence_CompositionAndWorktree is a regression test
// for #1434. Composition steps (iterate, sub_pipeline, etc.) and worktree
// steps don't leave a step-named directory under the run workspace —
// composition steps put their state in child runs + registered artifacts;
// worktree steps prune the worktree on success. The validator must not
// reject resume just because there's no on-disk dir for those step kinds.
func TestValidateGenericStepSequence_CompositionAndWorktree(t *testing.T) {
	tmpDir := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	pipeline := &Pipeline{
		Metadata: PipelineMetadata{Name: "audit-issue"},
		Steps: []Step{
			{ID: "fetch-issue"},
			{ID: "parallel-evidence", Iterate: &IterateConfig{Over: "{{ axes }}", Mode: "parallel"}},
			{ID: "synthesize", Workspace: WorkspaceConfig{Type: "worktree"}},
			{ID: "create-pr"},
		},
	}

	// Create only the fetch-issue workspace — none for composition or worktree
	// steps, mirroring the real audit-issue failure shape.
	runDir := filepath.Join(tmpDir, ".agents/workspaces/audit-issue-20260428-111246-6a7d")
	_ = os.MkdirAll(filepath.Join(runDir, "fetch-issue"), 0755)

	v := NewPhaseSkipValidator()
	if err := v.ValidatePhaseSequence(pipeline, "create-pr"); err != nil {
		t.Fatalf("expected resume to validate (composition + worktree priors should be skipped), got: %v", err)
	}
}
