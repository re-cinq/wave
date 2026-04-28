package pipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/manifest"
)

// TestMaterialiseMountSubset covers the workspace subset mount mode
// added for issue #1453.
func TestMaterialiseMountSubset(t *testing.T) {
	tmp := t.TempDir()

	// Project tree with three files; we'll subset to two of them.
	source := filepath.Join(tmp, "project")
	for _, p := range []string{"keep/a.go", "keep/b.go", "drop/c.go"} {
		full := filepath.Join(source, p)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("package x\n// "+p), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// pr-context.json artifact listing only the keep/* files.
	artifact := filepath.Join(tmp, "pr-context.json")
	body, _ := json.Marshal(map[string]any{
		"changed_files": []string{"keep/a.go", "keep/b.go"},
	})
	if err := os.WriteFile(artifact, body, 0644); err != nil {
		t.Fatal(err)
	}

	// chdir into tmp so the subset root resolves under tmp/.agents/workspaces/_subsets/.
	origWD, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	exec := &PipelineExecution{
		Pipeline: &Pipeline{
			Steps: []Step{{ID: "fetch-pr"}, {ID: "audit", Dependencies: []string{"fetch-pr"}}},
		},
		Manifest:       &manifest.Manifest{},
		ArtifactPaths:  map[string]string{"fetch-pr:pr-context": artifact},
		WorkspacePaths: map[string]string{},
		Status:         &PipelineStatus{ID: "test-run"},
		Context:        NewPipelineContext("test-run", "test", "audit"),
	}

	executor := NewDefaultPipelineExecutor(adapter.NewMockAdapter())

	mount := Mount{
		Source:     source,
		Target:     "/project",
		Mode:       "readonly",
		SubsetFrom: "fetch-pr.pr-context.changed_files",
	}

	subsetDir, err := executor.materialiseMountSubset(exec, "audit", 0, mount)
	if err != nil {
		t.Fatalf("materialise: %v", err)
	}

	if _, err := os.Stat(filepath.Join(subsetDir, "keep/a.go")); err != nil {
		t.Errorf("keep/a.go missing in subset: %v", err)
	}
	if _, err := os.Stat(filepath.Join(subsetDir, "keep/b.go")); err != nil {
		t.Errorf("keep/b.go missing in subset: %v", err)
	}
	if _, err := os.Stat(filepath.Join(subsetDir, "drop/c.go")); err == nil {
		t.Errorf("drop/c.go must NOT be in subset")
	}
}

// TestMaterialiseMountSubset_PathTraversalRejected verifies that
// listed paths escaping the source root are silently dropped.
func TestMaterialiseMountSubset_PathTraversalRejected(t *testing.T) {
	tmp := t.TempDir()
	source := filepath.Join(tmp, "project")
	if err := os.MkdirAll(source, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "ok.go"), []byte("package x"), 0644); err != nil {
		t.Fatal(err)
	}

	artifact := filepath.Join(tmp, "pr-context.json")
	body, _ := json.Marshal(map[string]any{
		"changed_files": []string{"ok.go", "../../etc/passwd", "/etc/passwd"},
	})
	if err := os.WriteFile(artifact, body, 0644); err != nil {
		t.Fatal(err)
	}

	origWD, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	exec := &PipelineExecution{
		Pipeline:       &Pipeline{Steps: []Step{{ID: "fetch-pr"}}},
		Manifest:       &manifest.Manifest{},
		ArtifactPaths:  map[string]string{"fetch-pr:pr-context": artifact},
		WorkspacePaths: map[string]string{},
		Status:         &PipelineStatus{ID: "test-run"},
		Context:        NewPipelineContext("test-run", "test", "audit"),
	}
	executor := NewDefaultPipelineExecutor(adapter.NewMockAdapter())

	subsetDir, err := executor.materialiseMountSubset(exec, "audit", 0, Mount{
		Source:     source,
		Target:     "/project",
		SubsetFrom: "fetch-pr.pr-context.changed_files",
	})
	if err != nil {
		t.Fatalf("materialise: %v", err)
	}

	if _, err := os.Stat(filepath.Join(subsetDir, "ok.go")); err != nil {
		t.Errorf("ok.go missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(subsetDir, "../../etc/passwd")); err == nil {
		t.Errorf("path traversal entry must be dropped")
	}
}

// TestMaterialiseMountSubset_SymlinkRejected verifies that a symlink
// inside Source pointing outside is dropped rather than dereferenced.
func TestMaterialiseMountSubset_SymlinkRejected(t *testing.T) {
	tmp := t.TempDir()
	source := filepath.Join(tmp, "project")
	secret := filepath.Join(tmp, "secret.txt")
	if err := os.MkdirAll(source, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secret, []byte("CONFIDENTIAL"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(source, "leak")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "ok.go"), []byte("package x"), 0644); err != nil {
		t.Fatal(err)
	}

	artifact := filepath.Join(tmp, "pr-context.json")
	body, _ := json.Marshal(map[string]any{"changed_files": []string{"ok.go", "leak"}})
	if err := os.WriteFile(artifact, body, 0644); err != nil {
		t.Fatal(err)
	}

	origWD, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	exec := &PipelineExecution{
		Pipeline:       &Pipeline{Steps: []Step{{ID: "fetch-pr"}}},
		Manifest:       &manifest.Manifest{},
		ArtifactPaths:  map[string]string{"fetch-pr:pr-context": artifact},
		WorkspacePaths: map[string]string{},
		Status:         &PipelineStatus{ID: "test-run"},
		Context:        NewPipelineContext("test-run", "test", "audit"),
	}
	executor := NewDefaultPipelineExecutor(adapter.NewMockAdapter())

	subsetDir, err := executor.materialiseMountSubset(exec, "audit", 0, Mount{
		Source:     source,
		Target:     "/project",
		SubsetFrom: "fetch-pr.pr-context.changed_files",
	})
	if err != nil {
		t.Fatalf("materialise: %v", err)
	}

	if _, err := os.Stat(filepath.Join(subsetDir, "ok.go")); err != nil {
		t.Errorf("ok.go missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(subsetDir, "leak")); err == nil {
		t.Errorf("symlink leaking outside source must be dropped")
	}
}
