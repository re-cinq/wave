package adapter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSkillSource(t *testing.T, dir, name, body string) string {
	t.Helper()
	src := filepath.Join(dir, name+"-src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: test\n---\n\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return src
}

func TestProvisionSkillsCopiesAndDropsSentinel(t *testing.T) {
	tmp := t.TempDir()
	workspace := filepath.Join(tmp, "ws")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}

	src := writeSkillSource(t, tmp, "alpha", "alpha body")
	refs := []SkillRef{{Name: "alpha", SourcePath: src, Description: "test"}}

	if err := ProvisionSkills(workspace, ".agents/skills", refs); err != nil {
		t.Fatalf("ProvisionSkills: %v", err)
	}

	skillMd := filepath.Join(workspace, ".agents/skills/alpha/SKILL.md")
	if _, err := os.Stat(skillMd); err != nil {
		t.Errorf("expected SKILL.md provisioned: %v", err)
	}
	sentinel := filepath.Join(workspace, ".agents/skills/alpha", SentinelFile)
	if _, err := os.Stat(sentinel); err != nil {
		t.Errorf("expected sentinel file: %v", err)
	}
}

func TestProvisionSkillsPreservesUserCommittedSkills(t *testing.T) {
	tmp := t.TempDir()
	workspace := filepath.Join(tmp, "ws")
	skillsDir := filepath.Join(workspace, ".agents/skills")
	userSkill := filepath.Join(skillsDir, "user-committed")
	if err := os.MkdirAll(userSkill, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userSkill, "SKILL.md"), []byte("user content"), 0o644); err != nil {
		t.Fatal(err)
	}
	// no sentinel — must be preserved

	src := writeSkillSource(t, tmp, "wave-managed", "body")
	refs := []SkillRef{{Name: "wave-managed", SourcePath: src}}

	if err := ProvisionSkills(workspace, ".agents/skills", refs); err != nil {
		t.Fatalf("ProvisionSkills: %v", err)
	}

	if _, err := os.Stat(filepath.Join(userSkill, "SKILL.md")); err != nil {
		t.Errorf("user-committed skill should be preserved: %v", err)
	}
	if _, err := os.Stat(filepath.Join(skillsDir, "wave-managed", SentinelFile)); err != nil {
		t.Errorf("wave-managed skill should be provisioned: %v", err)
	}
}

func TestProvisionSkillsRemovesStaleWaveManaged(t *testing.T) {
	tmp := t.TempDir()
	workspace := filepath.Join(tmp, "ws")
	skillsDir := filepath.Join(workspace, ".agents/skills")
	stale := filepath.Join(skillsDir, "stale")
	if err := os.MkdirAll(stale, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stale, SentinelFile), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := ProvisionSkills(workspace, ".agents/skills", nil); err != nil {
		t.Fatalf("ProvisionSkills: %v", err)
	}

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale wave-managed dir should be removed, stat err: %v", err)
	}
}

func TestProvisionSkillsPanicsOnTraversal(t *testing.T) {
	tmp := t.TempDir()
	workspace := filepath.Join(tmp, "ws")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on workspace-escape")
		}
		if !strings.Contains(r.(string), "refusing skill provisioning outside workspace") {
			t.Errorf("unexpected panic message: %v", r)
		}
	}()

	_ = ProvisionSkills(workspace, "../../../etc/skills", nil)
}

func TestProvisionSkillsRejectsAbsoluteSubdir(t *testing.T) {
	tmp := t.TempDir()
	if err := ProvisionSkills(tmp, "/etc/skills", nil); err == nil {
		t.Fatal("expected error for absolute targetSubdir")
	}
}

func TestProvisionSkillsEmptyWorkspace(t *testing.T) {
	if err := ProvisionSkills("", ".agents/skills", nil); err == nil {
		t.Fatal("expected error for empty workspace")
	}
}
