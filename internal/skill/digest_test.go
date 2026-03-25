package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestComputeDigest(t *testing.T) {
	dir := t.TempDir()

	// Create SKILL.md
	content := "---\nname: test\ndescription: test skill\n---\nBody content.\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	s := Skill{
		Name:       "test",
		SourcePath: dir,
	}

	digest, err := ComputeDigest(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(digest, "sha256:") {
		t.Errorf("digest should have sha256: prefix, got %q", digest)
	}

	// Should be deterministic
	digest2, err := ComputeDigest(s)
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	if digest != digest2 {
		t.Errorf("digests should be identical: %q != %q", digest, digest2)
	}
}

func TestComputeDigestWithResources(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: test\ndescription: test\n---\nbody\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create resource files
	scriptsDir := filepath.Join(dir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scriptsDir, "helper.sh"), []byte("#!/bin/bash\necho hello\n"), 0644); err != nil {
		t.Fatal(err)
	}

	sNoRes := Skill{Name: "test", SourcePath: dir}
	sWithRes := Skill{Name: "test", SourcePath: dir, ResourcePaths: []string{"scripts/helper.sh"}}

	d1, err := ComputeDigest(sNoRes)
	if err != nil {
		t.Fatal(err)
	}
	d2, err := ComputeDigest(sWithRes)
	if err != nil {
		t.Fatal(err)
	}

	if d1 == d2 {
		t.Error("digests should differ when resource files are included")
	}
}

func TestComputeDigestResourceSortOrder(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: test\ndescription: test\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}
	scriptsDir := filepath.Join(dir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scriptsDir, "a.sh"), []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scriptsDir, "b.sh"), []byte("b"), 0644); err != nil {
		t.Fatal(err)
	}

	s1 := Skill{Name: "test", SourcePath: dir, ResourcePaths: []string{"scripts/a.sh", "scripts/b.sh"}}
	s2 := Skill{Name: "test", SourcePath: dir, ResourcePaths: []string{"scripts/b.sh", "scripts/a.sh"}}

	d1, _ := ComputeDigest(s1)
	d2, _ := ComputeDigest(s2)

	if d1 != d2 {
		t.Error("digests should be identical regardless of resource path order")
	}
}

func TestComputeDigestDifferentContent(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir1, "SKILL.md"), []byte("content A"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir2, "SKILL.md"), []byte("content B"), 0644); err != nil {
		t.Fatal(err)
	}

	d1, _ := ComputeDigest(Skill{SourcePath: dir1})
	d2, _ := ComputeDigest(Skill{SourcePath: dir2})

	if d1 == d2 {
		t.Error("different content should produce different digests")
	}
}

func TestComputeDigestMissingSKILLMd(t *testing.T) {
	dir := t.TempDir()

	_, err := ComputeDigest(Skill{SourcePath: dir})
	if err == nil {
		t.Error("expected error for missing SKILL.md")
	}
}
