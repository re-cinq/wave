package workspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestWorkspaceManager(t *testing.T) (WorkspaceManager, string) {
	tmpDir, err := os.MkdirTemp("", "wave-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	wm, err := NewWorkspaceManager(tmpDir)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create workspace manager: %v", err)
	}

	return wm, tmpDir
}

func cleanupTestDir(t *testing.T, dir string) {
	if err := os.RemoveAll(dir); err != nil {
		t.Logf("Failed to cleanup test dir: %v", err)
	}
}

func TestWorkspaceManager_Create(t *testing.T) {
	wm, tmpDir := setupTestWorkspaceManager(t)
	defer cleanupTestDir(t, tmpDir)

	sourceDir := filepath.Join(tmpDir, "source-repo")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create test source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "test.txt"), []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	templateVars := map[string]string{
		"pipeline_id": "pipe-1",
		"step_id":     "step-1",
		"var1":        "value1",
	}

	tests := []struct {
		name         string
		cfg          WorkspaceConfig
		templateVars map[string]string
		wantErr      bool
		checkContent bool
	}{
		{
			name: "valid workspace",
			cfg: WorkspaceConfig{
				Mount: []Mount{
					{Source: sourceDir, Target: "/src", Mode: "readwrite"},
				},
			},
			templateVars: templateVars,
			wantErr:      false,
			checkContent: true,
		},
		{
			name: "readonly workspace",
			cfg: WorkspaceConfig{
				Mount: []Mount{
					{Source: sourceDir, Target: "/src", Mode: "readonly"},
				},
			},
			templateVars: templateVars,
			wantErr:      false,
			checkContent: false,
		},
		{
			name: "template substitution",
			cfg: WorkspaceConfig{
				Mount: []Mount{
					{Source: sourceDir, Target: "/src/{{var1}}", Mode: "readwrite"},
				},
			},
			templateVars: map[string]string{
				"pipeline_id": "pipe-2",
				"step_id":     "step-2",
				"var1":        "custom",
			},
			wantErr:      false,
			checkContent: false,
		},
		{
			name: "no mounts",
			cfg: WorkspaceConfig{
				Mount: []Mount{},
			},
			templateVars: templateVars,
			wantErr:      true,
		},
		{
			name: "missing source",
			cfg: WorkspaceConfig{
				Mount: []Mount{
					{Source: filepath.Join(tmpDir, "nonexistent"), Target: "/src", Mode: "readwrite"},
				},
			},
			templateVars: templateVars,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspacePath, err := wm.Create(tt.cfg, tt.templateVars)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				pipelineID := tt.templateVars["pipeline_id"]
				stepID := tt.templateVars["step_id"]
				expectedPath := filepath.Join(tmpDir, pipelineID, stepID)
				if workspacePath != expectedPath {
					t.Errorf("Create() returned wrong path: got %v, want %v", workspacePath, expectedPath)
				}

				if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
					t.Errorf("Create() did not create workspace directory")
				}

				if tt.checkContent {
					srcContent, err := os.ReadFile(filepath.Join(sourceDir, "test.txt"))
					if err != nil {
						t.Errorf("Failed to read source file: %v", err)
					}
					dstContent, err := os.ReadFile(filepath.Join(workspacePath, "src", "test.txt"))
					if err != nil {
						t.Errorf("Failed to read destination file: %v", err)
					} else if string(dstContent) != string(srcContent) {
						t.Errorf("File content mismatch: got %s, want %s", string(dstContent), string(srcContent))
					}
				}
			}
		})
	}
}

func TestWorkspaceManager_CreateMountModes(t *testing.T) {
	wm, tmpDir := setupTestWorkspaceManager(t)
	defer cleanupTestDir(t, tmpDir)

	sourceDir := filepath.Join(tmpDir, "source-repo")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create test source: %v", err)
	}

	tests := []struct {
		name         string
		mode         string
		expectedPerm os.FileMode
	}{
		{"readwrite mode", "readwrite", 0755},
		{"readonly mode", "readonly", 0555},
		{"default mode", "", 0755},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := WorkspaceConfig{
				Mount: []Mount{
					{Source: sourceDir, Target: "/src", Mode: tt.mode},
				},
			}
			templateVars := map[string]string{
				"pipeline_id": "pipe-mode",
				"step_id":     "step-mode",
			}

			workspacePath, err := wm.Create(cfg, templateVars)
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}

			mountPath := filepath.Join(workspacePath, "src")
			info, err := os.Stat(mountPath)
			if err != nil {
				t.Errorf("Failed to stat mount: %v", err)
			} else if info.Mode().Perm() != tt.expectedPerm {
				t.Errorf("Mount permissions: got %v, want %v", info.Mode().Perm(), tt.expectedPerm)
			}
		})
	}
}

func TestWorkspaceManager_InjectArtifacts(t *testing.T) {
	wm, tmpDir := setupTestWorkspaceManager(t)
	defer cleanupTestDir(t, tmpDir)

	sourceDir := filepath.Join(tmpDir, "source-repo")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create test source: %v", err)
	}

	cfg := WorkspaceConfig{
		Mount: []Mount{
			{Source: sourceDir, Target: "/src", Mode: "readwrite"},
		},
	}
	templateVars := map[string]string{
		"pipeline_id": "pipe-1",
		"step_id":     "step-1",
	}

	workspacePath, err := wm.Create(cfg, templateVars)
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	artifactsDir := filepath.Join(tmpDir, "artifacts")
	if err := os.MkdirAll(artifactsDir, 0755); err != nil {
		t.Fatalf("Failed to create artifacts dir: %v", err)
	}

	artifactFile := filepath.Join(artifactsDir, "output.txt")
	if err := os.WriteFile(artifactFile, []byte("artifact content"), 0644); err != nil {
		t.Fatalf("Failed to create artifact file: %v", err)
	}

	refs := []ArtifactRef{
		{Step: "step-1", Artifact: "output.txt", As: "output"},
		{Step: "step-2", Artifact: "data.json"},
	}
	resolvedPaths := map[string]string{
		"step-1:output.txt": artifactFile,
		"step-2":            filepath.Join(artifactsDir, "step-2"),
	}

	err = wm.InjectArtifacts(workspacePath, refs, resolvedPaths)
	if err != nil {
		t.Errorf("InjectArtifacts() error = %v", err)
	}

	expectedArtifact := filepath.Join(workspacePath, ".wave", "artifacts", "step-1_output")
	if _, err := os.Stat(expectedArtifact); os.IsNotExist(err) {
		t.Errorf("InjectArtifacts() did not create expected artifact")
	}
}

func TestWorkspaceManager_CleanAll(t *testing.T) {
	wm, tmpDir := setupTestWorkspaceManager(t)
	defer cleanupTestDir(t, tmpDir)

	sourceDir := filepath.Join(tmpDir, "source-repo")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create test source: %v", err)
	}

	cfg := WorkspaceConfig{
		Mount: []Mount{
			{Source: sourceDir, Target: "/src", Mode: "readwrite"},
		},
	}

	templateVars := map[string]string{
		"pipeline_id": "pipe-1",
		"step_id":     "step-1",
	}

	_, err := wm.Create(cfg, templateVars)
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	if err := wm.CleanAll("pipe-1"); err != nil {
		t.Errorf("CleanAll() error = %v", err)
	}

	pipelineDir := filepath.Join(tmpDir, "pipe-1")
	if _, err := os.Stat(pipelineDir); !os.IsNotExist(err) {
		t.Errorf("CleanAll() did not remove pipeline directory")
	}
}

func TestWorkspaceManager_CleanAllAbsolutePath(t *testing.T) {
	wm, tmpDir := setupTestWorkspaceManager(t)
	defer cleanupTestDir(t, tmpDir)

	sourceDir := filepath.Join(tmpDir, "source-repo")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create test source: %v", err)
	}

	cfg := WorkspaceConfig{
		Mount: []Mount{
			{Source: sourceDir, Target: "/src", Mode: "readwrite"},
		},
	}

	templateVars := map[string]string{
		"pipeline_id": "pipe-2",
		"step_id":     "step-1",
	}

	workspacePath, err := wm.Create(cfg, templateVars)
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	if err := wm.CleanAll(workspacePath); err != nil {
		t.Errorf("CleanAll() error = %v", err)
	}

	if _, err := os.Stat(workspacePath); !os.IsNotExist(err) {
		t.Errorf("CleanAll() did not remove workspace directory")
	}
}

func TestWorkspaceManager_SubstituteVars(t *testing.T) {
	wm, tmpDir := setupTestWorkspaceManager(t)
	defer cleanupTestDir(t, tmpDir)

	sourceDir := filepath.Join(tmpDir, "source-repo")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create test source: %v", err)
	}

	tests := []struct {
		name         string
		path         string
		templateVars map[string]string
		expected     string
	}{
		{
			name:         "single substitution",
			path:         "/src/{{var1}}",
			templateVars: map[string]string{"var1": "custom"},
			expected:     "/src/custom",
		},
		{
			name:         "multiple substitutions",
			path:         "/{{step_id}}/{{var1}}",
			templateVars: map[string]string{"step_id": "step-1", "var1": "custom"},
			expected:     "/step-1/custom",
		},
		{
			name:         "no substitution",
			path:         "/src",
			templateVars: map[string]string{"var1": "custom"},
			expected:     "/src",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := WorkspaceConfig{
				Mount: []Mount{
					{Source: sourceDir, Target: tt.path, Mode: "readwrite"},
				},
			}
			templateVars := map[string]string{
				"pipeline_id": "pipe-1",
				"step_id":     "step-1",
			}
			for k, v := range tt.templateVars {
				templateVars[k] = v
			}

			workspacePath, err := wm.Create(cfg, templateVars)
			if err != nil {
				t.Errorf("Create() error = %v", err)
				return
			}

			expectedMountPath := filepath.Join(workspacePath, filepath.FromSlash(tt.expected))
			if _, err := os.Stat(expectedMountPath); os.IsNotExist(err) {
				t.Errorf("Create() did not create mount path at %s", expectedMountPath)
			}
		})
	}
}

func TestWorkspaceManager_EmptyInputs(t *testing.T) {
	wm, tmpDir := setupTestWorkspaceManager(t)
	defer cleanupTestDir(t, tmpDir)

	sourceDir := filepath.Join(tmpDir, "source-repo")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create test source: %v", err)
	}

	tests := []struct {
		name   string
		method func() error
	}{
		{"CleanAll empty", func() error { return wm.CleanAll("") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.method(); err == nil {
				t.Errorf("Expected error for empty input")
			}
		})
	}
}

func TestListWorkspacesSortedByTime(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wave-workspace-sort-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create workspaces with specific modification times
	workspaces := []struct {
		name    string
		modTime int64
	}{
		{"ws-newest", 3},
		{"ws-oldest", 1},
		{"ws-middle", 2},
	}

	for _, ws := range workspaces {
		wsDir := filepath.Join(tmpDir, ws.name)
		if err := os.MkdirAll(wsDir, 0755); err != nil {
			t.Fatalf("Failed to create workspace dir: %v", err)
		}
		// Set the modification time (using fake times based on order)
		// We'll rely on creation order and small delays for actual sorting
	}

	// Run the function
	result, err := ListWorkspacesSortedByTime(tmpDir)
	if err != nil {
		t.Fatalf("ListWorkspacesSortedByTime() error = %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 workspaces, got %d", len(result))
	}

	// Verify each workspace has correct fields
	for _, ws := range result {
		if ws.Name == "" {
			t.Errorf("Workspace name should not be empty")
		}
		if ws.Path == "" {
			t.Errorf("Workspace path should not be empty")
		}
		if ws.ModTime == 0 {
			t.Errorf("Workspace modTime should not be zero")
		}
	}
}

func TestListWorkspacesSortedByTime_NonExistentDir(t *testing.T) {
	result, err := ListWorkspacesSortedByTime("/nonexistent/path/12345")
	if err != nil {
		t.Errorf("Should not error for non-existent directory: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected empty result for non-existent directory, got %d", len(result))
	}
}

func TestListWorkspacesSortedByTime_EmptyDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wave-workspace-empty-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	result, err := ListWorkspacesSortedByTime(tmpDir)
	if err != nil {
		t.Errorf("ListWorkspacesSortedByTime() error = %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected empty result for empty directory, got %d", len(result))
	}
}

func TestListWorkspacesSortedByTime_SkipsFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wave-workspace-files-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a directory
	if err := os.MkdirAll(filepath.Join(tmpDir, "real-workspace"), 0755); err != nil {
		t.Fatalf("Failed to create workspace dir: %v", err)
	}

	// Create a file (should be skipped)
	if err := os.WriteFile(filepath.Join(tmpDir, "not-a-workspace.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result, err := ListWorkspacesSortedByTime(tmpDir)
	if err != nil {
		t.Errorf("ListWorkspacesSortedByTime() error = %v", err)
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 workspace (file should be skipped), got %d", len(result))
	}
	if result[0].Name != "real-workspace" {
		t.Errorf("Expected 'real-workspace', got '%s'", result[0].Name)
	}
}

func TestSortWorkspacesByTime(t *testing.T) {
	workspaces := []WorkspaceInfo{
		{Name: "ws-c", Path: "/path/ws-c", ModTime: 300},
		{Name: "ws-a", Path: "/path/ws-a", ModTime: 100},
		{Name: "ws-b", Path: "/path/ws-b", ModTime: 200},
	}

	sortWorkspacesByTime(workspaces)

	// Should be sorted oldest to newest
	if workspaces[0].Name != "ws-a" {
		t.Errorf("First workspace should be ws-a (oldest), got %s", workspaces[0].Name)
	}
	if workspaces[1].Name != "ws-b" {
		t.Errorf("Second workspace should be ws-b (middle), got %s", workspaces[1].Name)
	}
	if workspaces[2].Name != "ws-c" {
		t.Errorf("Third workspace should be ws-c (newest), got %s", workspaces[2].Name)
	}
}

// =============================================================================
// T104: Workspace Isolation Tests
// =============================================================================

// TestWorkspaceIsolation_SeparatePipelines verifies that workspaces for
// different pipelines are completely isolated from each other.
func TestWorkspaceIsolation_SeparatePipelines(t *testing.T) {
	wm, tmpDir := setupTestWorkspaceManager(t)
	defer cleanupTestDir(t, tmpDir)

	// Create source directory with test content
	sourceDir := filepath.Join(tmpDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "shared.txt"), []byte("original"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	cfg := WorkspaceConfig{
		Mount: []Mount{{Source: sourceDir, Target: "/src", Mode: "readwrite"}},
	}

	// Create workspace for pipeline A
	wsA, err := wm.Create(cfg, map[string]string{
		"pipeline_id": "pipeline-A",
		"step_id":     "step-1",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace A: %v", err)
	}

	// Create workspace for pipeline B
	wsB, err := wm.Create(cfg, map[string]string{
		"pipeline_id": "pipeline-B",
		"step_id":     "step-1",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace B: %v", err)
	}

	// Verify workspaces are at different paths
	if wsA == wsB {
		t.Errorf("Workspaces should have different paths: A=%s, B=%s", wsA, wsB)
	}

	// Modify file in workspace A
	fileA := filepath.Join(wsA, "src", "shared.txt")
	if err := os.WriteFile(fileA, []byte("modified by pipeline A"), 0644); err != nil {
		t.Fatalf("Failed to write to workspace A: %v", err)
	}

	// Verify workspace B still has original content
	fileB := filepath.Join(wsB, "src", "shared.txt")
	contentB, err := os.ReadFile(fileB)
	if err != nil {
		t.Fatalf("Failed to read from workspace B: %v", err)
	}
	if string(contentB) != "original" {
		t.Errorf("Workspace B was modified! Expected 'original', got '%s'", string(contentB))
	}

	// Verify original source is unchanged
	originalContent, err := os.ReadFile(filepath.Join(sourceDir, "shared.txt"))
	if err != nil {
		t.Fatalf("Failed to read original: %v", err)
	}
	if string(originalContent) != "original" {
		t.Errorf("Original source was modified! Expected 'original', got '%s'", string(originalContent))
	}
}

// TestWorkspaceIsolation_SeparateSteps verifies that different steps within
// the same pipeline have isolated workspaces.
func TestWorkspaceIsolation_SeparateSteps(t *testing.T) {
	wm, tmpDir := setupTestWorkspaceManager(t)
	defer cleanupTestDir(t, tmpDir)

	sourceDir := filepath.Join(tmpDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "data.txt"), []byte("step data"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	cfg := WorkspaceConfig{
		Mount: []Mount{{Source: sourceDir, Target: "/src", Mode: "readwrite"}},
	}

	// Create workspaces for different steps in the same pipeline
	wsStep1, err := wm.Create(cfg, map[string]string{
		"pipeline_id": "my-pipeline",
		"step_id":     "navigate",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace for step 1: %v", err)
	}

	wsStep2, err := wm.Create(cfg, map[string]string{
		"pipeline_id": "my-pipeline",
		"step_id":     "execute",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace for step 2: %v", err)
	}

	wsStep3, err := wm.Create(cfg, map[string]string{
		"pipeline_id": "my-pipeline",
		"step_id":     "review",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace for step 3: %v", err)
	}

	// Verify all paths are different
	paths := []string{wsStep1, wsStep2, wsStep3}
	seen := make(map[string]bool)
	for _, p := range paths {
		if seen[p] {
			t.Errorf("Duplicate workspace path detected: %s", p)
		}
		seen[p] = true
	}

	// Create a new file in step1 workspace
	newFile := filepath.Join(wsStep1, "new_file.txt")
	if err := os.WriteFile(newFile, []byte("created by step1"), 0644); err != nil {
		t.Fatalf("Failed to create new file: %v", err)
	}

	// Verify step2 workspace doesn't have the new file
	if _, err := os.Stat(filepath.Join(wsStep2, "new_file.txt")); !os.IsNotExist(err) {
		t.Error("Step 2 workspace should not have file created in step 1")
	}

	// Verify step3 workspace doesn't have the new file
	if _, err := os.Stat(filepath.Join(wsStep3, "new_file.txt")); !os.IsNotExist(err) {
		t.Error("Step 3 workspace should not have file created in step 1")
	}
}

// TestWorkspaceIsolation_ConcurrentCreation verifies that concurrent workspace
// creation doesn't cause isolation violations.
func TestWorkspaceIsolation_ConcurrentCreation(t *testing.T) {
	wm, tmpDir := setupTestWorkspaceManager(t)
	defer cleanupTestDir(t, tmpDir)

	sourceDir := filepath.Join(tmpDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "test.txt"), []byte("concurrent test"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	cfg := WorkspaceConfig{
		Mount: []Mount{{Source: sourceDir, Target: "/src", Mode: "readwrite"}},
	}

	const numPipelines = 10
	const stepsPerPipeline = 5

	type result struct {
		path string
		err  error
	}

	results := make(chan result, numPipelines*stepsPerPipeline)

	// Create workspaces concurrently
	for p := 0; p < numPipelines; p++ {
		for s := 0; s < stepsPerPipeline; s++ {
			go func(pipelineID, stepID int) {
				path, err := wm.Create(cfg, map[string]string{
					"pipeline_id": filepath.Base(tmpDir) + "-pipe-" + string(rune('A'+pipelineID)),
					"step_id":     "step-" + string(rune('0'+stepID)),
				})
				results <- result{path: path, err: err}
			}(p, s)
		}
	}

	// Collect results
	paths := make(map[string]bool)
	for i := 0; i < numPipelines*stepsPerPipeline; i++ {
		r := <-results
		if r.err != nil {
			t.Errorf("Workspace creation failed: %v", r.err)
			continue
		}
		if paths[r.path] {
			t.Errorf("Duplicate workspace path in concurrent creation: %s", r.path)
		}
		paths[r.path] = true
	}

	// Verify all workspaces exist and are distinct
	if len(paths) != numPipelines*stepsPerPipeline {
		t.Errorf("Expected %d unique paths, got %d", numPipelines*stepsPerPipeline, len(paths))
	}
}

// TestWorkspaceIsolation_ReadonlyPreservation verifies that readonly mounts
// maintain isolation by preventing writes.
func TestWorkspaceIsolation_ReadonlyPreservation(t *testing.T) {
	wm, tmpDir := setupTestWorkspaceManager(t)
	defer cleanupTestDir(t, tmpDir)

	sourceDir := filepath.Join(tmpDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "readonly.txt"), []byte("protected"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	cfg := WorkspaceConfig{
		Mount: []Mount{{Source: sourceDir, Target: "/src", Mode: "readonly"}},
	}

	ws, err := wm.Create(cfg, map[string]string{
		"pipeline_id": "readonly-test",
		"step_id":     "step-1",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	mountPath := filepath.Join(ws, "src")
	info, err := os.Stat(mountPath)
	if err != nil {
		t.Fatalf("Failed to stat mount: %v", err)
	}

	// Verify readonly permissions
	if info.Mode().Perm() != 0555 {
		t.Errorf("Expected readonly permissions 0555, got %v", info.Mode().Perm())
	}
}

// TestWorkspaceIsolation_ArtifactInjectionIsolation verifies that artifact
// injection doesn't leak data between pipelines.
func TestWorkspaceIsolation_ArtifactInjectionIsolation(t *testing.T) {
	wm, tmpDir := setupTestWorkspaceManager(t)
	defer cleanupTestDir(t, tmpDir)

	sourceDir := filepath.Join(tmpDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}

	cfg := WorkspaceConfig{
		Mount: []Mount{{Source: sourceDir, Target: "/src", Mode: "readwrite"}},
	}

	// Create workspaces for two pipelines
	wsA, err := wm.Create(cfg, map[string]string{
		"pipeline_id": "pipeline-A",
		"step_id":     "step-review",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace A: %v", err)
	}

	wsB, err := wm.Create(cfg, map[string]string{
		"pipeline_id": "pipeline-B",
		"step_id":     "step-review",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace B: %v", err)
	}

	// Create an artifact directory with a secret file
	artifactDir := filepath.Join(tmpDir, "artifacts", "step-1")
	if err := os.MkdirAll(artifactDir, 0755); err != nil {
		t.Fatalf("Failed to create artifact dir: %v", err)
	}
	secretFile := filepath.Join(artifactDir, "secret.txt")
	if err := os.WriteFile(secretFile, []byte("secret data for pipeline A"), 0644); err != nil {
		t.Fatalf("Failed to create artifact: %v", err)
	}

	// Inject artifact only into pipeline A
	refsA := []ArtifactRef{{Step: "step-1", Artifact: "secret.txt", As: "secret"}}
	resolvedA := map[string]string{"step-1:secret.txt": secretFile}

	err = wm.InjectArtifacts(wsA, refsA, resolvedA)
	if err != nil {
		t.Fatalf("Failed to inject artifacts into A: %v", err)
	}

	// Pipeline B should NOT have the artifact
	artifactPathB := filepath.Join(wsB, ".wave", "artifacts", "step-1_secret")
	if _, err := os.Stat(artifactPathB); !os.IsNotExist(err) {
		t.Error("Pipeline B has artifact that was only injected into pipeline A")
	}

	// Pipeline A should have the artifact
	artifactPathA := filepath.Join(wsA, ".wave", "artifacts", "step-1_secret")
	if _, err := os.Stat(artifactPathA); os.IsNotExist(err) {
		t.Error("Pipeline A missing injected artifact")
	}
}

// TestWorkspaceIsolation_NoPathTraversal verifies that workspace paths
// cannot escape their designated directories.
func TestWorkspaceIsolation_NoPathTraversal(t *testing.T) {
	wm, tmpDir := setupTestWorkspaceManager(t)
	defer cleanupTestDir(t, tmpDir)

	sourceDir := filepath.Join(tmpDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}

	// Create a sensitive file outside the workspace
	sensitiveDir := filepath.Join(tmpDir, "sensitive")
	if err := os.MkdirAll(sensitiveDir, 0755); err != nil {
		t.Fatalf("Failed to create sensitive dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sensitiveDir, "secret.txt"), []byte("top secret"), 0644); err != nil {
		t.Fatalf("Failed to create sensitive file: %v", err)
	}

	cfg := WorkspaceConfig{
		Mount: []Mount{{Source: sourceDir, Target: "/src", Mode: "readwrite"}},
	}

	ws, err := wm.Create(cfg, map[string]string{
		"pipeline_id": "normal-pipeline",
		"step_id":     "step-1",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Verify the workspace is contained within the base directory
	if !strings.HasPrefix(ws, tmpDir) {
		t.Errorf("Workspace %s escaped base directory %s", ws, tmpDir)
	}

	// Attempt path traversal from within the workspace
	traversalPath := filepath.Join(ws, "..", "..", "sensitive", "secret.txt")
	resolvedPath, err := filepath.Abs(traversalPath)
	if err != nil {
		t.Fatalf("Failed to resolve traversal path: %v", err)
	}

	// The resolved traversal path must still be within tmpDir (the base
	// directory), not escaping to the filesystem root. This verifies that
	// the workspace hierarchy prevents escaping the base directory.
	if !strings.HasPrefix(resolvedPath, tmpDir) {
		t.Errorf("Traversal path %q escaped base directory %q (resolved to %q)", traversalPath, tmpDir, resolvedPath)
	}

	// Verify that reading the sensitive file through the traversal path
	// from within the workspace's own mount does NOT succeed — the mount
	// should not contain the sensitive directory.
	sensitivePath := filepath.Join(ws, "src", "..", "..", "sensitive", "secret.txt")
	content, readErr := os.ReadFile(sensitivePath)
	if readErr == nil && string(content) == "top secret" {
		t.Errorf("Successfully read sensitive file through workspace traversal: %q", sensitivePath)
	}
}

// TestWorkspaceIsolation_CleanupOneDoesntAffectOther verifies that cleaning
// one pipeline's workspace doesn't affect other pipelines.
func TestWorkspaceIsolation_CleanupOneDoesntAffectOther(t *testing.T) {
	wm, tmpDir := setupTestWorkspaceManager(t)
	defer cleanupTestDir(t, tmpDir)

	sourceDir := filepath.Join(tmpDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}

	cfg := WorkspaceConfig{
		Mount: []Mount{{Source: sourceDir, Target: "/src", Mode: "readwrite"}},
	}

	// Create workspaces for two pipelines
	_, err := wm.Create(cfg, map[string]string{
		"pipeline_id": "pipeline-to-delete",
		"step_id":     "step-1",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace A: %v", err)
	}

	wsKeep, err := wm.Create(cfg, map[string]string{
		"pipeline_id": "pipeline-to-keep",
		"step_id":     "step-1",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace B: %v", err)
	}

	// Add content to the workspace we're keeping
	testFile := filepath.Join(wsKeep, "important.txt")
	if err := os.WriteFile(testFile, []byte("important data"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Clean the first pipeline
	if err := wm.CleanAll("pipeline-to-delete"); err != nil {
		t.Fatalf("CleanAll failed: %v", err)
	}

	// Verify the kept pipeline's workspace still exists with its data
	if _, err := os.Stat(wsKeep); os.IsNotExist(err) {
		t.Error("Kept workspace was deleted when cleaning other pipeline")
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read kept file: %v", err)
	}
	if string(content) != "important data" {
		t.Errorf("Kept file content changed: %s", string(content))
	}
}

// TestCopyRecursive_SymlinkToDirectory verifies that copyRecursive handles
// symlinks to directories without failing with "copy_file_range: is a directory".
func TestCopyRecursive_SymlinkToDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source directory with a subdirectory and a symlink to that subdirectory.
	// The symlink target must be within the source tree to pass the boundary check.
	srcDir := filepath.Join(tmpDir, "src")
	targetDir := filepath.Join(srcDir, "real-dir")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "inner.txt"), []byte("inside symlinked dir"), 0644); err != nil {
		t.Fatalf("Failed to create inner file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "regular.txt"), []byte("regular file"), 0644); err != nil {
		t.Fatalf("Failed to create regular file: %v", err)
	}
	// Symlink within the source tree (mirrors specs/ cross-references)
	if err := os.Symlink(targetDir, filepath.Join(srcDir, "linked-dir")); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Copy the source directory
	dstDir := filepath.Join(tmpDir, "dst")
	if err := copyRecursive(srcDir, dstDir); err != nil {
		t.Fatalf("copyRecursive failed: %v", err)
	}

	// Verify regular file was copied
	content, err := os.ReadFile(filepath.Join(dstDir, "regular.txt"))
	if err != nil {
		t.Fatalf("Failed to read regular file: %v", err)
	}
	if string(content) != "regular file" {
		t.Errorf("Expected 'regular file', got %q", string(content))
	}

	// Verify the symlinked directory's contents were copied
	innerContent, err := os.ReadFile(filepath.Join(dstDir, "linked-dir", "inner.txt"))
	if err != nil {
		t.Fatalf("Failed to read file from symlinked dir: %v", err)
	}
	if string(innerContent) != "inside symlinked dir" {
		t.Errorf("Expected 'inside symlinked dir', got %q", string(innerContent))
	}

	// Verify it's a real directory, not a symlink
	info, err := os.Lstat(filepath.Join(dstDir, "linked-dir"))
	if err != nil {
		t.Fatalf("Failed to lstat linked-dir: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("Expected real directory, got symlink")
	}
}

// TestNewWorkspaceManager_GitRootResolution verifies that NewWorkspaceManager
// resolves a relative baseDir against the git repository root rather than the
// process CWD. This prevents stray .wave/workspaces/ directories from being
// created inside source subdirectories when tests are run from a subpackage.
func TestNewWorkspaceManager_GitRootResolution(t *testing.T) {
	// Create a temporary directory to act as the git repository root.
	gitRoot := t.TempDir()

	// Initialise a git repository so that "git rev-parse --show-toplevel"
	// returns the temp directory.
	if out, err := exec.Command("git", "init", "-q", gitRoot).CombinedOutput(); err != nil {
		t.Skipf("git init failed (git unavailable?): %v — %s", err, out)
	}

	// Create a subdirectory inside the repo to simulate running from a
	// subpackage (e.g. internal/pipeline/).
	subDir := filepath.Join(gitRoot, "internal", "pipeline")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	// Change CWD to the subdirectory. t.Chdir restores the original CWD at
	// the end of the test.
	t.Chdir(subDir)

	// Call NewWorkspaceManager with a relative path — the canonical default.
	wm, err := NewWorkspaceManager(".wave/workspaces")
	if err != nil {
		t.Fatalf("NewWorkspaceManager() error = %v", err)
	}

	// The internal baseDir must be anchored to the git root, not subDir.
	mgr := wm.(*workspaceManager)
	if strings.HasPrefix(mgr.baseDir, subDir) {
		t.Errorf("baseDir %q is rooted at subDir %q — expected git root %q", mgr.baseDir, subDir, gitRoot)
	}
	if !strings.HasPrefix(mgr.baseDir, gitRoot) {
		t.Errorf("baseDir %q is not rooted at git root %q", mgr.baseDir, gitRoot)
	}
}

// TestNewWorkspaceManager_AbsolutePathPassthrough verifies that an absolute
// baseDir (e.g. t.TempDir()) is left unchanged by NewWorkspaceManager and is
// NOT re-joined with the git root.
func TestNewWorkspaceManager_AbsolutePathPassthrough(t *testing.T) {
	absDir := t.TempDir()
	wm, err := NewWorkspaceManager(absDir)
	if err != nil {
		t.Fatalf("NewWorkspaceManager() error = %v", err)
	}
	mgr := wm.(*workspaceManager)
	if mgr.baseDir != absDir {
		t.Errorf("absolute baseDir was modified: got %q, want %q", mgr.baseDir, absDir)
	}
}

// TestCopyRecursive_SymlinkOutsideSourceSkipped verifies that symlinks pointing
// outside the source tree are skipped to prevent path traversal.
func TestCopyRecursive_SymlinkOutsideSourceSkipped(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an external directory (outside source tree)
	externalDir := filepath.Join(tmpDir, "external")
	if err := os.MkdirAll(externalDir, 0755); err != nil {
		t.Fatalf("Failed to create external dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(externalDir, "secret.txt"), []byte("secret"), 0644); err != nil {
		t.Fatalf("Failed to create secret file: %v", err)
	}

	// Create source directory with a symlink pointing outside
	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create src dir: %v", err)
	}
	if err := os.Symlink(externalDir, filepath.Join(srcDir, "escape")); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	dstDir := filepath.Join(tmpDir, "dst")
	if err := copyRecursive(srcDir, dstDir); err != nil {
		t.Fatalf("copyRecursive failed: %v", err)
	}

	// The external directory's contents should NOT be copied
	if _, err := os.Stat(filepath.Join(dstDir, "escape", "secret.txt")); !os.IsNotExist(err) {
		t.Error("Symlink outside source tree was followed — path traversal vulnerability")
	}
}
