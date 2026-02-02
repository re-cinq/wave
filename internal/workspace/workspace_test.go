package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestWorkspaceManager(t *testing.T) (WorkspaceManager, string) {
	tmpDir, err := os.MkdirTemp("", "wave-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	wm, err := NewWorkspaceManager(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
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

	expectedArtifact := filepath.Join(workspacePath, "artifacts", "step-1_output")
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
