package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/fileutil"
)

func TestCopyRecursive_LargeFileSkipped(t *testing.T) {
	tmpDir := t.TempDir()

	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	// Create a small file (should be copied)
	smallFile := filepath.Join(srcDir, "small.txt")
	if err := os.WriteFile(smallFile, []byte("small content"), 0644); err != nil {
		t.Fatalf("failed to create small file: %v", err)
	}

	// Create a file >10MB (should be skipped)
	largeFile := filepath.Join(srcDir, "large.bin")
	data := make([]byte, 10*1024*1024+1) // 10MB + 1 byte
	if err := os.WriteFile(largeFile, data, 0644); err != nil {
		t.Fatalf("failed to create large file: %v", err)
	}

	dstDir := filepath.Join(tmpDir, "dst")
	if err := copyRecursive(srcDir, dstDir); err != nil {
		t.Fatalf("copyRecursive failed: %v", err)
	}

	// Small file should be copied
	if _, err := os.Stat(filepath.Join(dstDir, "small.txt")); os.IsNotExist(err) {
		t.Error("small file should have been copied")
	}

	// Large file should be skipped
	if _, err := os.Stat(filepath.Join(dstDir, "large.bin")); !os.IsNotExist(err) {
		t.Error("large file (>10MB) should have been skipped")
	}
}

func TestCopyRecursive_SkipDirs(t *testing.T) {
	tmpDir := t.TempDir()

	srcDir := filepath.Join(tmpDir, "src")

	// Create directories that should be skipped
	skippedDirs := []string{
		"node_modules",
		".git",
		".wave",
		".claude",
		"vendor",
		"__pycache__",
		".venv",
		"dist",
		"build",
		".next",
		".cache",
	}

	for _, dir := range skippedDirs {
		dirPath := filepath.Join(srcDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatalf("failed to create dir %s: %v", dir, err)
		}
		// Put a file inside to verify the directory is actually skipped
		if err := os.WriteFile(filepath.Join(dirPath, "marker.txt"), []byte("should not be copied"), 0644); err != nil {
			t.Fatalf("failed to create marker in %s: %v", dir, err)
		}
	}

	// Create a regular directory that should be copied
	regularDir := filepath.Join(srcDir, "src-code")
	if err := os.MkdirAll(regularDir, 0755); err != nil {
		t.Fatalf("failed to create regular dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(regularDir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to create main.go: %v", err)
	}

	dstDir := filepath.Join(tmpDir, "dst")
	if err := copyRecursive(srcDir, dstDir); err != nil {
		t.Fatalf("copyRecursive failed: %v", err)
	}

	// Verify skipped directories
	for _, dir := range skippedDirs {
		markerPath := filepath.Join(dstDir, dir, "marker.txt")
		if _, err := os.Stat(markerPath); !os.IsNotExist(err) {
			t.Errorf("directory %q should have been skipped, but marker file exists", dir)
		}
	}

	// Verify regular directory was copied
	mainGo := filepath.Join(dstDir, "src-code", "main.go")
	content, err := os.ReadFile(mainGo)
	if err != nil {
		t.Errorf("regular directory should have been copied, can't read main.go: %v", err)
	} else if string(content) != "package main" {
		t.Errorf("copied content mismatch: got %q", string(content))
	}
}

func TestCopyRecursive_BrokenSymlinkSkipped(t *testing.T) {
	tmpDir := t.TempDir()

	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	// Create a regular file
	if err := os.WriteFile(filepath.Join(srcDir, "real.txt"), []byte("real"), 0644); err != nil {
		t.Fatalf("failed to create real file: %v", err)
	}

	// Create a broken symlink (target doesn't exist)
	brokenLink := filepath.Join(srcDir, "broken-link")
	if err := os.Symlink(filepath.Join(srcDir, "nonexistent-target"), brokenLink); err != nil {
		t.Fatalf("failed to create broken symlink: %v", err)
	}

	dstDir := filepath.Join(tmpDir, "dst")
	if err := copyRecursive(srcDir, dstDir); err != nil {
		t.Fatalf("copyRecursive should not fail on broken symlinks, got: %v", err)
	}

	// Real file should be copied
	if _, err := os.Stat(filepath.Join(dstDir, "real.txt")); os.IsNotExist(err) {
		t.Error("real file should have been copied")
	}
}

func TestCopyPath_SingleFile(t *testing.T) {
	tmpDir := t.TempDir()

	srcFile := filepath.Join(tmpDir, "source.txt")
	content := "test file content with special chars: àéîõü"
	if err := os.WriteFile(srcFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	dstFile := filepath.Join(tmpDir, "dest.txt")
	if err := fileutil.CopyPath(srcFile, dstFile); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	got, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("failed to read dest file: %v", err)
	}
	if string(got) != content {
		t.Errorf("content mismatch: got %q, want %q", string(got), content)
	}
}

func TestCopyPath_SourceNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	err := fileutil.CopyPath(filepath.Join(tmpDir, "nonexistent"), filepath.Join(tmpDir, "dest"))
	if err == nil {
		t.Error("expected error for non-existent source file")
	}
}

func TestCopyRecursive_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	dstDir := filepath.Join(tmpDir, "dst")
	if err := copyRecursive(srcDir, dstDir); err != nil {
		t.Fatalf("copyRecursive should handle empty directory: %v", err)
	}

	info, err := os.Stat(dstDir)
	if err != nil {
		t.Fatalf("dst dir should exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("dst should be a directory")
	}
}

func TestCopyRecursive_NestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	srcDir := filepath.Join(tmpDir, "src")
	nestedPath := filepath.Join(srcDir, "a", "b", "c")
	if err := os.MkdirAll(nestedPath, 0755); err != nil {
		t.Fatalf("failed to create nested dirs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nestedPath, "deep.txt"), []byte("deep content"), 0644); err != nil {
		t.Fatalf("failed to create deep file: %v", err)
	}

	dstDir := filepath.Join(tmpDir, "dst")
	if err := copyRecursive(srcDir, dstDir); err != nil {
		t.Fatalf("copyRecursive failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dstDir, "a", "b", "c", "deep.txt"))
	if err != nil {
		t.Fatalf("failed to read deep file: %v", err)
	}
	if string(content) != "deep content" {
		t.Errorf("content mismatch: got %q", string(content))
	}
}

func TestCopyRecursive_FileAtExact10MB(t *testing.T) {
	tmpDir := t.TempDir()

	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	// Exactly 10MB should be copied (the check is > 10MB, not >=)
	exactFile := filepath.Join(srcDir, "exact.bin")
	data := make([]byte, 10*1024*1024) // Exactly 10MB
	if err := os.WriteFile(exactFile, data, 0644); err != nil {
		t.Fatalf("failed to create exact 10MB file: %v", err)
	}

	dstDir := filepath.Join(tmpDir, "dst")
	if err := copyRecursive(srcDir, dstDir); err != nil {
		t.Fatalf("copyRecursive failed: %v", err)
	}

	// Exactly 10MB should be copied (not skipped)
	if _, err := os.Stat(filepath.Join(dstDir, "exact.bin")); os.IsNotExist(err) {
		t.Error("file at exactly 10MB should be copied (threshold is >10MB)")
	}
}

func TestCopyRecursive_SkipDirsDoNotApplyToRoot(t *testing.T) {
	tmpDir := t.TempDir()

	// If the source directory itself is named "vendor", it should still be copied
	// (skipDirs only applies to subdirectories, checked via relPath != ".")
	srcDir := filepath.Join(tmpDir, "vendor")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "lib.go"), []byte("package vendor"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	dstDir := filepath.Join(tmpDir, "dst")
	if err := copyRecursive(srcDir, dstDir); err != nil {
		t.Fatalf("copyRecursive failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dstDir, "lib.go"))
	if err != nil {
		t.Fatalf("root directory named 'vendor' should still be copied: %v", err)
	}
	if string(content) != "package vendor" {
		t.Errorf("content mismatch: got %q", string(content))
	}
}

func TestCopyRecursive_SourceNotFound(t *testing.T) {
	err := copyRecursive("/nonexistent/path/12345", t.TempDir())
	if err == nil {
		t.Error("expected error for non-existent source")
	}
}

func TestInjectArtifacts_EmptyWorkspacePath(t *testing.T) {
	tmpDir := t.TempDir()
	wm, err := NewWorkspaceManager(tmpDir)
	if err != nil {
		t.Fatalf("failed to create workspace manager: %v", err)
	}

	err = wm.InjectArtifacts("", nil, nil)
	if err == nil {
		t.Error("expected error for empty workspace path")
	}
	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("error message should mention empty path, got: %v", err)
	}
}

func TestInjectArtifacts_SkipsEmptyRefs(t *testing.T) {
	tmpDir := t.TempDir()
	wm, err := NewWorkspaceManager(tmpDir)
	if err != nil {
		t.Fatalf("failed to create workspace manager: %v", err)
	}

	workspacePath := filepath.Join(tmpDir, "ws")
	_ = os.MkdirAll(workspacePath, 0755)

	// Refs with empty fields should be skipped
	refs := []ArtifactRef{
		{Step: "", Artifact: "output.txt"},
		{Step: "step-1", Artifact: ""},
	}

	err = wm.InjectArtifacts(workspacePath, refs, map[string]string{})
	if err != nil {
		t.Errorf("empty refs should be skipped without error: %v", err)
	}
}

func TestCreateMount_EmptySourceOrTarget(t *testing.T) {
	tmpDir := t.TempDir()
	wm, err := NewWorkspaceManager(tmpDir)
	if err != nil {
		t.Fatalf("failed to create workspace manager: %v", err)
	}

	tests := []struct {
		name   string
		source string
		target string
	}{
		{"empty source", "", "/dst"},
		{"empty target", tmpDir, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := WorkspaceConfig{
				Mount: []Mount{{Source: tt.source, Target: tt.target, Mode: "readwrite"}},
			}
			_, err := wm.Create(cfg, map[string]string{
				"pipeline_id": "test",
				"step_id":     "step",
			})
			if err == nil {
				t.Error("expected error for empty source or target")
			}
			if !strings.Contains(err.Error(), "cannot be empty") {
				t.Errorf("error should mention empty source/target, got: %v", err)
			}
		})
	}
}
