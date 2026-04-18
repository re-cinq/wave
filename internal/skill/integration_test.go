package skill

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// T017: TestSkillLifecycle_FileAdapter — SC-003, US6-1, US6-2
// End-to-end test: create temp dir with valid SKILL.md, install via FileAdapter
// into real DirectoryStore, List to verify appears, ProvisionFromStore to workspace,
// verify SKILL.md body content, Delete, verify gone.
func TestSkillLifecycle_FileAdapter(t *testing.T) {
	// Create a source skill directory with valid SKILL.md
	srcDir := t.TempDir()
	skillDir := filepath.Join(srcDir, "lifecycle-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}

	bodyContent := "# Lifecycle Skill\n\nThis is a lifecycle test.\n"
	skillMD := "---\nname: lifecycle-skill\ndescription: Lifecycle test skill\n---\n" + bodyContent
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a real DirectoryStore
	storeDir := t.TempDir()
	store := NewDirectoryStore(SkillSource{Root: storeDir, Precedence: 1})

	// Install via FileAdapter
	adapter := NewFileAdapter(srcDir)
	result, err := adapter.Install(context.TODO(), skillDir, store)
	if err != nil {
		t.Fatalf("FileAdapter.Install() error = %v", err)
	}
	if len(result.Skills) != 1 {
		t.Fatalf("expected 1 installed skill, got %d", len(result.Skills))
	}
	if result.Skills[0].Name != "lifecycle-skill" {
		t.Errorf("installed skill name = %q, want %q", result.Skills[0].Name, "lifecycle-skill")
	}

	// List to verify appears
	skills, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill in list, got %d", len(skills))
	}
	if skills[0].Name != "lifecycle-skill" {
		t.Errorf("listed skill name = %q, want %q", skills[0].Name, "lifecycle-skill")
	}

	// ProvisionFromStore to workspace
	workspace := t.TempDir()
	infos, err := ProvisionFromStore(store, workspace, []string{"lifecycle-skill"})
	if err != nil {
		t.Fatalf("ProvisionFromStore() error = %v", err)
	}
	if len(infos) != 1 {
		t.Fatalf("expected 1 info, got %d", len(infos))
	}

	// Verify SKILL.md body content matches original (ProvisionFromStore writes Body only)
	data, err := os.ReadFile(filepath.Join(workspace, ".agents", "skills", "lifecycle-skill", "SKILL.md"))
	if err != nil {
		t.Fatalf("failed to read provisioned SKILL.md: %v", err)
	}
	if string(data) != bodyContent {
		t.Errorf("provisioned SKILL.md content mismatch:\ngot:  %q\nwant: %q", string(data), bodyContent)
	}

	// Delete
	if err := store.Delete("lifecycle-skill"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify gone
	skills, err = store.List()
	if err != nil {
		t.Fatalf("List() after delete error = %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("expected 0 skills after delete, got %d", len(skills))
	}
}

// T018: TestSkillLifecycle_MultiSource — US6-3
// Install two skills from different file:// paths into a 2-source DirectoryStore,
// List, verify both appear with correct metadata and source paths.
func TestSkillLifecycle_MultiSource(t *testing.T) {
	// Create two source skill directories
	srcDir1 := t.TempDir()
	skill1Dir := filepath.Join(srcDir1, "skill-one")
	if err := os.MkdirAll(skill1Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skill1Dir, "SKILL.md"),
		[]byte("---\nname: skill-one\ndescription: First skill\n---\n# One\n"),
		0o644); err != nil {
		t.Fatal(err)
	}

	srcDir2 := t.TempDir()
	skill2Dir := filepath.Join(srcDir2, "skill-two")
	if err := os.MkdirAll(skill2Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skill2Dir, "SKILL.md"),
		[]byte("---\nname: skill-two\ndescription: Second skill\n---\n# Two\n"),
		0o644); err != nil {
		t.Fatal(err)
	}

	// Create a 2-source store
	storeDir1 := t.TempDir()
	storeDir2 := t.TempDir()
	store := NewDirectoryStore(
		SkillSource{Root: storeDir1, Precedence: 2},
		SkillSource{Root: storeDir2, Precedence: 1},
	)

	// Install first skill (goes to highest precedence = storeDir1)
	adapter1 := NewFileAdapter(srcDir1)
	result1, err := adapter1.Install(context.TODO(), skill1Dir, store)
	if err != nil {
		t.Fatalf("FileAdapter.Install(skill-one) error = %v", err)
	}
	if len(result1.Skills) != 1 {
		t.Fatalf("expected 1 skill from source 1, got %d", len(result1.Skills))
	}

	// Install second skill
	adapter2 := NewFileAdapter(srcDir2)
	result2, err := adapter2.Install(context.TODO(), skill2Dir, store)
	if err != nil {
		t.Fatalf("FileAdapter.Install(skill-two) error = %v", err)
	}
	if len(result2.Skills) != 1 {
		t.Fatalf("expected 1 skill from source 2, got %d", len(result2.Skills))
	}

	// List and verify both appear
	skills, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(skills))
	}

	nameSet := make(map[string]bool)
	for _, s := range skills {
		nameSet[s.Name] = true
		if s.Description == "" {
			t.Errorf("skill %q has empty description", s.Name)
		}
	}
	if !nameSet["skill-one"] {
		t.Error("skill-one not found in list")
	}
	if !nameSet["skill-two"] {
		t.Error("skill-two not found in list")
	}
}
