package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestProvisionFromStore_Success(t *testing.T) {
	// Set up a skill store with a skill that has resources
	storeDir := t.TempDir()
	skillSrc := filepath.Join(storeDir, "test-skill")
	if err := os.MkdirAll(filepath.Join(skillSrc, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}

	skillMD := `---
name: test-skill
description: A test skill
---
# Test Skill

Body content here.
`
	if err := os.WriteFile(filepath.Join(skillSrc, "SKILL.md"), []byte(skillMD), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillSrc, "scripts", "helper.sh"), []byte("#!/bin/bash\necho hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := NewDirectoryStore(SkillSource{Root: storeDir, Precedence: 0})
	workspace := t.TempDir()

	infos, err := ProvisionFromStore(store, workspace, []string{"test-skill"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(infos) != 1 {
		t.Fatalf("expected 1 info, got %d", len(infos))
	}
	if infos[0].Name != "test-skill" {
		t.Errorf("expected name %q, got %q", "test-skill", infos[0].Name)
	}
	if infos[0].Description != "A test skill" {
		t.Errorf("expected description %q, got %q", "A test skill", infos[0].Description)
	}

	// Verify SKILL.md was written
	skillMDPath := filepath.Join(workspace, ".wave", "skills", "test-skill", "SKILL.md")
	data, err := os.ReadFile(skillMDPath)
	if err != nil {
		t.Fatalf("SKILL.md not found: %v", err)
	}
	if string(data) == "" {
		t.Error("SKILL.md is empty")
	}

	// Verify resource was copied
	scriptPath := filepath.Join(workspace, ".wave", "skills", "test-skill", "scripts", "helper.sh")
	if _, err := os.Stat(scriptPath); err != nil {
		t.Errorf("resource file not copied: %v", err)
	}
}

func TestProvisionFromStore_MissingSkillSkipsWithWarning(t *testing.T) {
	storeDir := t.TempDir()
	store := NewDirectoryStore(SkillSource{Root: storeDir, Precedence: 0})
	workspace := t.TempDir()

	infos, err := ProvisionFromStore(store, workspace, []string{"nonexistent"})
	if err != nil {
		t.Fatalf("expected no error for missing skill (should warn and skip), got: %v", err)
	}
	if len(infos) != 0 {
		t.Errorf("expected empty infos for missing skill, got %d", len(infos))
	}
}

func TestProvisionFromStore_EmptyList(t *testing.T) {
	storeDir := t.TempDir()
	store := NewDirectoryStore(SkillSource{Root: storeDir, Precedence: 0})
	workspace := t.TempDir()

	infos, err := ProvisionFromStore(store, workspace, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if infos != nil {
		t.Errorf("expected nil infos, got %v", infos)
	}

	infos, err = ProvisionFromStore(store, workspace, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if infos != nil {
		t.Errorf("expected nil infos, got %v", infos)
	}
}

func TestProvisionFromStore_PathTraversal(t *testing.T) {
	// Create a skill with a resource path that escapes the skill directory
	storeDir := t.TempDir()
	skillSrc := filepath.Join(storeDir, "evil-skill")
	if err := os.MkdirAll(skillSrc, 0o755); err != nil {
		t.Fatal(err)
	}

	skillMD := `---
name: evil-skill
description: A skill with bad resources
---
# Evil Skill

Body content.
`
	if err := os.WriteFile(filepath.Join(skillSrc, "SKILL.md"), []byte(skillMD), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a mock store that returns a skill with a traversal path
	store := &mockStoreForTraversal{
		skill: Skill{
			Name:          "evil-skill",
			Description:   "A skill with bad resources",
			Body:          "# Evil Skill\n\nBody content.\n",
			SourcePath:    skillSrc,
			ResourcePaths: []string{"../../../etc/passwd"},
		},
	}

	workspace := t.TempDir()
	_, err := ProvisionFromStore(store, workspace, []string{"evil-skill"})
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
}

func TestProvisionFromStore_MultipleSkills(t *testing.T) {
	storeDir := t.TempDir()

	for _, name := range []string{"alpha", "beta"} {
		skillSrc := filepath.Join(storeDir, name)
		if err := os.MkdirAll(skillSrc, 0o755); err != nil {
			t.Fatal(err)
		}
		skillMD := "---\nname: " + name + "\ndescription: " + name + " skill\n---\n# " + name + "\n\nBody.\n"
		if err := os.WriteFile(filepath.Join(skillSrc, "SKILL.md"), []byte(skillMD), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	store := NewDirectoryStore(SkillSource{Root: storeDir, Precedence: 0})
	workspace := t.TempDir()

	infos, err := ProvisionFromStore(store, workspace, []string{"alpha", "beta"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(infos) != 2 {
		t.Fatalf("expected 2 infos, got %d", len(infos))
	}

	// Verify both SKILL.md files were written
	for _, name := range []string{"alpha", "beta"} {
		skillMDPath := filepath.Join(workspace, ".wave", "skills", name, "SKILL.md")
		if _, err := os.Stat(skillMDPath); err != nil {
			t.Errorf("SKILL.md for %q not found: %v", name, err)
		}
	}
}

// mockStoreForTraversal is a minimal Store implementation that returns
// a skill with controlled ResourcePaths for testing path traversal.
type mockStoreForTraversal struct {
	skill Skill
}

func (m *mockStoreForTraversal) Read(name string) (Skill, error) {
	if name == m.skill.Name {
		return m.skill, nil
	}
	return Skill{}, fmt.Errorf("skill %q not found", name)
}

func (m *mockStoreForTraversal) Write(skill Skill) error  { return nil }
func (m *mockStoreForTraversal) List() ([]Skill, error)   { return nil, nil }
func (m *mockStoreForTraversal) Delete(name string) error { return nil }

// --- T009: TestProvisionFromStore_AllResources — US4-1: all resource dirs provisioned ---

func TestProvisionFromStore_AllResources(t *testing.T) {
	storeDir := t.TempDir()
	skillSrc := filepath.Join(storeDir, "full-skill")

	for _, sub := range []string{"scripts", "references", "assets"} {
		if err := os.MkdirAll(filepath.Join(skillSrc, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	skillMD := "---\nname: full-skill\ndescription: Skill with all resources\n---\n# Full Skill\n\nBody.\n"
	if err := os.WriteFile(filepath.Join(skillSrc, "SKILL.md"), []byte(skillMD), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillSrc, "scripts", "setup.sh"), []byte("#!/bin/bash\necho setup"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillSrc, "references", "api.json"), []byte(`{"version":"1"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillSrc, "assets", "logo.txt"), []byte("LOGO"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := NewDirectoryStore(SkillSource{Root: storeDir, Precedence: 0})
	workspace := t.TempDir()

	infos, err := ProvisionFromStore(store, workspace, []string{"full-skill"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(infos) != 1 {
		t.Fatalf("expected 1 info, got %d", len(infos))
	}

	// Verify SKILL.md exists
	skillMDPath := filepath.Join(workspace, ".wave", "skills", "full-skill", "SKILL.md")
	if _, err := os.Stat(skillMDPath); err != nil {
		t.Errorf("SKILL.md not found: %v", err)
	}

	// Verify all resource files exist at correct paths under .wave/skills/<name>/
	for _, path := range []string{
		filepath.Join(workspace, ".wave", "skills", "full-skill", "scripts", "setup.sh"),
		filepath.Join(workspace, ".wave", "skills", "full-skill", "references", "api.json"),
		filepath.Join(workspace, ".wave", "skills", "full-skill", "assets", "logo.txt"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("resource file not found: %s: %v", path, err)
		}
	}
}

// --- T010: TestProvisionFromStore_ContentMatch — US6-2: body content matches ---

func TestProvisionFromStore_ContentMatch(t *testing.T) {
	storeDir := t.TempDir()
	skillSrc := filepath.Join(storeDir, "content-skill")
	if err := os.MkdirAll(skillSrc, 0o755); err != nil {
		t.Fatal(err)
	}

	bodyContent := "# Content Skill\n\nThis is the expected body content.\nWith multiple lines.\n"
	skillMD := "---\nname: content-skill\ndescription: Content match test\n---\n" + bodyContent
	if err := os.WriteFile(filepath.Join(skillSrc, "SKILL.md"), []byte(skillMD), 0o644); err != nil {
		t.Fatal(err)
	}

	store := NewDirectoryStore(SkillSource{Root: storeDir, Precedence: 0})
	workspace := t.TempDir()

	_, err := ProvisionFromStore(store, workspace, []string{"content-skill"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// ProvisionFromStore writes only the Body (not full SKILL.md with frontmatter) to workspace
	data, err := os.ReadFile(filepath.Join(workspace, ".wave", "skills", "content-skill", "SKILL.md"))
	if err != nil {
		t.Fatalf("failed to read provisioned SKILL.md: %v", err)
	}
	if string(data) != bodyContent {
		t.Errorf("provisioned SKILL.md body mismatch:\ngot:  %q\nwant: %q", string(data), bodyContent)
	}
}

// --- T011: TestProvisionFromStore_IsolatedDirs — US4-4: multi-skill isolation ---

func TestProvisionFromStore_IsolatedDirs(t *testing.T) {
	storeDir := t.TempDir()

	for _, name := range []string{"alpha", "beta", "gamma"} {
		skillSrc := filepath.Join(storeDir, name)
		if err := os.MkdirAll(skillSrc, 0o755); err != nil {
			t.Fatal(err)
		}
		skillMD := "---\nname: " + name + "\ndescription: " + name + " skill\n---\n# " + name + "\n"
		if err := os.WriteFile(filepath.Join(skillSrc, "SKILL.md"), []byte(skillMD), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	store := NewDirectoryStore(SkillSource{Root: storeDir, Precedence: 0})
	workspace := t.TempDir()

	infos, err := ProvisionFromStore(store, workspace, []string{"alpha", "beta", "gamma"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(infos) != 3 {
		t.Fatalf("expected 3 infos, got %d", len(infos))
	}

	// Verify each skill has its own isolated directory
	for _, name := range []string{"alpha", "beta", "gamma"} {
		skillMDPath := filepath.Join(workspace, ".wave", "skills", name, "SKILL.md")
		if _, err := os.Stat(skillMDPath); err != nil {
			t.Errorf("SKILL.md for %q not found: %v", name, err)
		}
	}

	// Verify no cross-contamination: each dir should only have SKILL.md
	for _, name := range []string{"alpha", "beta", "gamma"} {
		skillDir := filepath.Join(workspace, ".wave", "skills", name)
		entries, err := os.ReadDir(skillDir)
		if err != nil {
			t.Fatalf("failed to read dir for %q: %v", name, err)
		}
		if len(entries) != 1 {
			names := make([]string, len(entries))
			for i, e := range entries {
				names[i] = e.Name()
			}
			t.Errorf("skill %q dir has %d entries (expected 1): %v", name, len(entries), names)
		}
	}
}
