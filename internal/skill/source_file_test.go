package skill

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileAdapterPrefix(t *testing.T) {
	a := NewFileAdapter("/tmp")
	if a.Prefix() != "file" {
		t.Errorf("Prefix() = %q, want %q", a.Prefix(), "file")
	}
}

func TestFileAdapterRelativePath(t *testing.T) {
	root := t.TempDir()
	makeTestSkillDir(t, root, "custom-skill", "Custom skill")

	a := NewFileAdapter(root)
	store := newMemoryStore()

	result, err := a.Install(context.Background(), "./custom-skill", store)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if len(result.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(result.Skills))
	}
	if result.Skills[0].Name != "custom-skill" {
		t.Errorf("Name = %q, want %q", result.Skills[0].Name, "custom-skill")
	}
	if store.writes != 1 {
		t.Errorf("expected 1 write, got %d", store.writes)
	}
}

func TestFileAdapterAbsolutePath(t *testing.T) {
	root := t.TempDir()
	makeTestSkillDir(t, root, "abs-skill", "Absolute path skill")

	skillDir := filepath.Join(root, "abs-skill")
	a := NewFileAdapter(root)
	store := newMemoryStore()

	result, err := a.Install(context.Background(), skillDir, store)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if len(result.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(result.Skills))
	}
	if result.Skills[0].Name != "abs-skill" {
		t.Errorf("Name = %q, want %q", result.Skills[0].Name, "abs-skill")
	}
}

func TestFileAdapterPathNotFound(t *testing.T) {
	root := t.TempDir()
	a := NewFileAdapter(root)
	store := newMemoryStore()

	_, err := a.Install(context.Background(), "./nonexistent", store)
	if err == nil {
		t.Fatal("expected error for non-existent path")
	}
	if !strings.Contains(err.Error(), "path not found") {
		t.Errorf("error should contain 'path not found': %v", err)
	}
}

func TestFileAdapterSymlinkRejected(t *testing.T) {
	root := t.TempDir()
	target := t.TempDir()
	makeTestSkillDir(t, target, "real-skill", "Real skill")

	// Create symlink inside root pointing to target
	symlink := filepath.Join(root, "linked-skill")
	if err := os.Symlink(filepath.Join(target, "real-skill"), symlink); err != nil {
		t.Skip("symlinks not supported")
	}

	a := NewFileAdapter(root)
	store := newMemoryStore()

	_, err := a.Install(context.Background(), "./linked-skill", store)
	if err == nil {
		t.Fatal("expected error for symlink")
	}
	if !strings.Contains(err.Error(), "symlink rejected") {
		t.Errorf("error should contain 'symlink rejected': %v", err)
	}
}

func TestFileAdapterPathTraversal(t *testing.T) {
	root := t.TempDir()
	a := NewFileAdapter(root)
	store := newMemoryStore()

	_, err := a.Install(context.Background(), "../../etc/passwd", store)
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
	// The error should be either "path not found" or "path traversal detected"
	errStr := err.Error()
	if !strings.Contains(errStr, "path") {
		t.Errorf("error should be about path issue: %v", err)
	}
}

func TestFileAdapterNoSkillMD(t *testing.T) {
	root := t.TempDir()
	// Create a directory without SKILL.md
	emptyDir := filepath.Join(root, "empty-dir")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatal(err)
	}

	a := NewFileAdapter(root)
	store := newMemoryStore()

	_, err := a.Install(context.Background(), "./empty-dir", store)
	if err == nil {
		t.Fatal("expected error for missing SKILL.md")
	}
	if !strings.Contains(err.Error(), "SKILL.md") {
		t.Errorf("error should mention SKILL.md: %v", err)
	}
}

func TestFileAdapterInvalidSkillMD(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "bad-skill")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("not valid frontmatter"), 0644); err != nil {
		t.Fatal(err)
	}

	a := NewFileAdapter(root)
	store := newMemoryStore()

	_, err := a.Install(context.Background(), "./bad-skill", store)
	if err == nil {
		t.Fatal("expected error for invalid SKILL.md")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("error should mention parse failure: %v", err)
	}
}
