package skill

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/recinq/wave/internal/manifest"
)

func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create .claude/commands/ with skill command files
	commandsDir := filepath.Join(dir, ".claude", "commands")
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test skill command files
	files := map[string]string{
		"speckit.specify.md": "# Specify\nRun specification workflow",
		"speckit.plan.md":    "# Plan\nRun planning workflow",
		"bmad.init.md":       "# BMAD Init\nInitialize BMAD",
		"other.cmd.md":       "# Other\nSome other command",
	}

	for name, content := range files {
		path := filepath.Join(commandsDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	return dir
}

func TestProvision_CopiesMatchingFiles(t *testing.T) {
	repoRoot := setupTestRepo(t)
	workspace := t.TempDir()

	skills := map[string]manifest.SkillConfig{
		"speckit": {
			Check: "true",
		},
	}

	p := NewProvisioner(skills, repoRoot)
	if err := p.Provision(workspace, []string{"speckit"}); err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	// Verify speckit commands were copied
	commandsDir := filepath.Join(workspace, ".claude", "commands")
	entries, err := os.ReadDir(commandsDir)
	if err != nil {
		t.Fatalf("failed to read commands dir: %v", err)
	}

	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}

	if len(names) != 2 {
		t.Fatalf("expected 2 speckit files, got %d: %v", len(names), names)
	}

	// Verify content was preserved
	data, err := os.ReadFile(filepath.Join(commandsDir, "speckit.specify.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "# Specify\nRun specification workflow" {
		t.Errorf("unexpected content: %s", string(data))
	}
}

func TestProvision_UndeclaredSkill(t *testing.T) {
	repoRoot := setupTestRepo(t)
	workspace := t.TempDir()

	skills := map[string]manifest.SkillConfig{} // No skills declared

	p := NewProvisioner(skills, repoRoot)
	if err := p.Provision(workspace, []string{"speckit"}); err != nil {
		t.Fatalf("Provision should not fail for undeclared skills: %v", err)
	}

	// No files should have been copied
	commandsDir := filepath.Join(workspace, ".claude", "commands")
	if _, err := os.Stat(commandsDir); !os.IsNotExist(err) {
		entries, _ := os.ReadDir(commandsDir)
		if len(entries) > 0 {
			t.Error("expected no files to be copied for undeclared skill")
		}
	}
}

func TestProvision_CustomGlob(t *testing.T) {
	repoRoot := setupTestRepo(t)
	workspace := t.TempDir()

	skills := map[string]manifest.SkillConfig{
		"bmad": {
			Check:        "true",
			CommandsGlob: filepath.Join(".claude", "commands", "bmad.*.md"),
		},
	}

	p := NewProvisioner(skills, repoRoot)
	if err := p.Provision(workspace, []string{"bmad"}); err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	commandsDir := filepath.Join(workspace, ".claude", "commands")
	entries, err := os.ReadDir(commandsDir)
	if err != nil {
		t.Fatalf("failed to read commands dir: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 bmad file, got %d", len(entries))
	}
	if entries[0].Name() != "bmad.init.md" {
		t.Errorf("expected bmad.init.md, got %s", entries[0].Name())
	}
}

func TestProvision_Empty(t *testing.T) {
	p := NewProvisioner(nil, t.TempDir())
	if err := p.Provision(t.TempDir(), nil); err != nil {
		t.Fatalf("Provision of empty list should not fail: %v", err)
	}
}

func TestProvisionAll(t *testing.T) {
	repoRoot := setupTestRepo(t)
	workspace := t.TempDir()

	skills := map[string]manifest.SkillConfig{
		"speckit": {Check: "true"},
		"bmad":    {Check: "true"},
	}

	p := NewProvisioner(skills, repoRoot)
	if err := p.ProvisionAll(workspace); err != nil {
		t.Fatalf("ProvisionAll failed: %v", err)
	}

	commandsDir := filepath.Join(workspace, ".claude", "commands")
	entries, err := os.ReadDir(commandsDir)
	if err != nil {
		t.Fatalf("failed to read commands dir: %v", err)
	}

	// Should have speckit.specify.md, speckit.plan.md, and bmad.init.md
	if len(entries) != 3 {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Fatalf("expected 3 files, got %d: %v", len(entries), names)
	}
}

func TestDiscoverCommands(t *testing.T) {
	repoRoot := setupTestRepo(t)

	skills := map[string]manifest.SkillConfig{
		"speckit": {Check: "true"},
		"bmad":    {Check: "true"},
	}

	p := NewProvisioner(skills, repoRoot)
	commands, err := p.DiscoverCommands([]string{"speckit", "bmad"})
	if err != nil {
		t.Fatalf("DiscoverCommands failed: %v", err)
	}

	if len(commands["speckit"]) != 2 {
		t.Errorf("expected 2 speckit commands, got %d", len(commands["speckit"]))
	}
	if len(commands["bmad"]) != 1 {
		t.Errorf("expected 1 bmad command, got %d", len(commands["bmad"]))
	}
}

func TestFormatSkillCommandPrompt(t *testing.T) {
	tests := []struct {
		command  string
		args     string
		expected string
	}{
		{"speckit.specify", "add auth", "Run `/speckit.specify` with: add auth"},
		{"/speckit.plan", "", "Run `/speckit.plan`"},
		{"bmad.init", "my-project", "Run `/bmad.init` with: my-project"},
	}

	for _, tt := range tests {
		result := FormatSkillCommandPrompt(tt.command, tt.args)
		if result != tt.expected {
			t.Errorf("FormatSkillCommandPrompt(%q, %q) = %q, want %q", tt.command, tt.args, result, tt.expected)
		}
	}
}
