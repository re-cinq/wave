package webui

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupGitRepo creates a temporary git repository with a main branch,
// a feature branch, and sample changes for diff testing.
func setupGitRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	// Initialize repo
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command %v failed: %v\n%s", args, err, out)
		}
	}

	run("git", "init", "-b", "main")
	run("git", "config", "user.email", "test@test.com")
	run("git", "config", "user.name", "Test")
	run("git", "config", "commit.gpgsign", "false")
	run("git", "config", "tag.gpgsign", "false")

	// Create initial files on main
	if err := os.WriteFile(filepath.Join(dir, "existing.go"), []byte("package main\n\nfunc hello() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "toDelete.go"), []byte("package main\n\nfunc remove() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Create a binary file (NUL bytes trigger git's binary detection)
	if err := os.WriteFile(filepath.Join(dir, "image.png"), []byte{0x89, 0x50, 0x4E, 0x47, 0x00, 0x00, 0x00, 0x0D}, 0o644); err != nil {
		t.Fatal(err)
	}

	run("git", "add", "-A")
	run("git", "commit", "-m", "initial commit")

	// Create feature branch with changes
	run("git", "checkout", "-b", "feature-branch")

	// Add a new file
	if err := os.WriteFile(filepath.Join(dir, "new_file.go"), []byte("package main\n\nfunc newFunc() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Modify existing file
	if err := os.WriteFile(filepath.Join(dir, "existing.go"), []byte("package main\n\nfunc hello() {\n\tprintln(\"hello\")\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Delete a file
	_ = os.Remove(filepath.Join(dir, "toDelete.go"))

	// Modify binary file
	if err := os.WriteFile(filepath.Join(dir, "image.png"), []byte{0x89, 0x50, 0x4E, 0x47, 0x00, 0x00, 0x00, 0x0D, 0xFF}, 0o644); err != nil {
		t.Fatal(err)
	}

	run("git", "add", "-A")
	run("git", "commit", "-m", "feature changes")

	// Go back to main so base branch resolution works
	run("git", "checkout", "main")

	return dir
}

func TestResolveBaseBranch(t *testing.T) {
	dir := setupGitRepo(t)
	ctx := context.Background()

	branch, err := resolveBaseBranch(ctx, dir)
	if err != nil {
		t.Fatalf("resolveBaseBranch() error: %v", err)
	}
	if branch != "main" {
		t.Errorf("expected main, got %q", branch)
	}
}

func TestResolveBaseBranch_NoMainBranch(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	// Create a repo with only a "master" branch
	run := func(args ...string) {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command %v failed: %v\n%s", args, err, out)
		}
	}

	run("git", "init", "-b", "master")
	run("git", "config", "user.email", "test@test.com")
	run("git", "config", "user.name", "Test")
	run("git", "config", "commit.gpgsign", "false")
	run("git", "config", "tag.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", "-A")
	run("git", "commit", "-m", "init")

	branch, err := resolveBaseBranch(ctx, dir)
	if err != nil {
		t.Fatalf("resolveBaseBranch() error: %v", err)
	}
	if branch != "master" {
		t.Errorf("expected master, got %q", branch)
	}
}

func TestComputeDiffSummary(t *testing.T) {
	dir := setupGitRepo(t)
	ctx := context.Background()

	summary := computeDiffSummary(ctx, dir, "main", "feature-branch")

	if !summary.Available {
		t.Fatal("expected Available=true")
	}
	if summary.TotalFiles < 3 {
		t.Errorf("expected at least 3 changed files, got %d", summary.TotalFiles)
	}

	// Check file statuses
	fileMap := make(map[string]FileSummary)
	for _, f := range summary.Files {
		fileMap[f.Path] = f
	}

	if f, ok := fileMap["new_file.go"]; ok {
		if f.Status != "added" {
			t.Errorf("new_file.go: expected status 'added', got %q", f.Status)
		}
	} else {
		t.Error("new_file.go not found in diff summary")
	}

	if f, ok := fileMap["existing.go"]; ok {
		if f.Status != "modified" {
			t.Errorf("existing.go: expected status 'modified', got %q", f.Status)
		}
	} else {
		t.Error("existing.go not found in diff summary")
	}

	if f, ok := fileMap["toDelete.go"]; ok {
		if f.Status != "deleted" {
			t.Errorf("toDelete.go: expected status 'deleted', got %q", f.Status)
		}
	} else {
		t.Error("toDelete.go not found in diff summary")
	}

	if f, ok := fileMap["image.png"]; ok {
		if !f.Binary {
			t.Error("image.png: expected Binary=true")
		}
	} else {
		t.Error("image.png not found in diff summary")
	}

	// Verify sorted alphabetically
	for i := 1; i < len(summary.Files); i++ {
		if summary.Files[i].Path < summary.Files[i-1].Path {
			t.Errorf("files not sorted: %q before %q", summary.Files[i-1].Path, summary.Files[i].Path)
		}
	}
}

func TestComputeDiffSummary_NonexistentBranch(t *testing.T) {
	dir := setupGitRepo(t)
	ctx := context.Background()

	summary := computeDiffSummary(ctx, dir, "main", "nonexistent-branch")
	if summary.Available {
		t.Error("expected Available=false for nonexistent branch")
	}
	if summary.Message == "" {
		t.Error("expected a message for unavailable diff")
	}
}

func TestComputeFileDiff(t *testing.T) {
	dir := setupGitRepo(t)
	ctx := context.Background()

	diff, err := computeFileDiff(ctx, dir, "main", "feature-branch", "existing.go")
	if err != nil {
		t.Fatalf("computeFileDiff() error: %v", err)
	}

	if diff.Path != "existing.go" {
		t.Errorf("expected path 'existing.go', got %q", diff.Path)
	}
	if diff.Status != "modified" {
		t.Errorf("expected status 'modified', got %q", diff.Status)
	}
	if diff.Content == "" {
		t.Error("expected non-empty diff content")
	}
	if diff.Additions == 0 {
		t.Error("expected at least one addition")
	}
}

func TestComputeFileDiff_BinaryFile(t *testing.T) {
	dir := setupGitRepo(t)
	ctx := context.Background()

	diff, err := computeFileDiff(ctx, dir, "main", "feature-branch", "image.png")
	if err != nil {
		t.Fatalf("computeFileDiff() error: %v", err)
	}

	if !diff.Binary {
		t.Error("expected Binary=true for image.png")
	}
	if diff.Content != "" {
		t.Error("expected empty content for binary file")
	}
}

func TestComputeFileDiff_PathTraversal(t *testing.T) {
	dir := setupGitRepo(t)
	ctx := context.Background()

	_, err := computeFileDiff(ctx, dir, "main", "feature-branch", "../../../etc/passwd")
	if err == nil {
		t.Error("expected error for path traversal")
	}
	if !strings.Contains(err.Error(), "invalid file path") {
		t.Errorf("expected 'invalid file path' error, got %q", err.Error())
	}
}

func TestComputeFileDiff_AbsolutePath(t *testing.T) {
	dir := setupGitRepo(t)
	ctx := context.Background()

	_, err := computeFileDiff(ctx, dir, "main", "feature-branch", "/etc/passwd")
	if err == nil {
		t.Error("expected error for absolute path")
	}
}

func TestComputeFileDiff_Truncation(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	run := func(args ...string) {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command %v failed: %v\n%s", args, err, out)
		}
	}

	run("git", "init", "-b", "main")
	run("git", "config", "user.email", "test@test.com")
	run("git", "config", "user.name", "Test")
	run("git", "config", "commit.gpgsign", "false")
	run("git", "config", "tag.gpgsign", "false")

	// Create empty file on main
	if err := os.WriteFile(filepath.Join(dir, "large.txt"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", "-A")
	run("git", "commit", "-m", "init")

	// Create a large file on feature branch
	run("git", "checkout", "-b", "feature")
	largeContent := strings.Repeat("This is a line of content for testing truncation.\n", 3000)
	if err := os.WriteFile(filepath.Join(dir, "large.txt"), []byte(largeContent), 0o644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", "-A")
	run("git", "commit", "-m", "large file")
	run("git", "checkout", "main")

	diff, err := computeFileDiff(ctx, dir, "main", "feature", "large.txt")
	if err != nil {
		t.Fatalf("computeFileDiff() error: %v", err)
	}

	if !diff.Truncated {
		t.Error("expected Truncated=true for large diff")
	}
	if diff.Size <= maxDiffSize {
		t.Errorf("expected Size > %d, got %d", maxDiffSize, diff.Size)
	}
}
