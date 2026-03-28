package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/defaults"
)

// executeSkillCmd runs the skill command with given arguments and captures output.
func executeSkillCmd(args ...string) (stdout string, err error) {
	cmd := NewSkillCmd()

	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(args)

	err = cmd.Execute()
	return outBuf.String(), err
}

func TestSkillListShowsAllTemplates(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	if err := os.MkdirAll(".wave/skills", 0755); err != nil {
		t.Fatal(err)
	}

	out, err := executeSkillCmd("list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All shipped templates should appear
	expected := defaults.SkillTemplateNames()
	for _, name := range expected {
		if !strings.Contains(out, name) {
			t.Errorf("expected template %q in list output, got: %s", name, out)
		}
	}
}

func TestSkillListJSON(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	if err := os.MkdirAll(".wave/skills", 0755); err != nil {
		t.Fatal(err)
	}

	out, err := executeSkillCmd("list", "--format", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result SkillTemplateListOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nraw: %s", err, out)
	}

	expectedCount := len(defaults.GetSkillTemplates())
	if len(result.Templates) != expectedCount {
		t.Errorf("expected %d templates, got %d", expectedCount, len(result.Templates))
	}

	// Verify each template has a name and description
	for _, tmpl := range result.Templates {
		if tmpl.Name == "" {
			t.Error("template has empty name")
		}
		if tmpl.Description == "" {
			t.Errorf("template %q has empty description", tmpl.Name)
		}
	}
}

func TestSkillListShowsInstalledStatus(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	// Install one template manually
	env.createSkill("gh-cli", "GitHub CLI operations")

	out, err := executeSkillCmd("list", "--format", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result SkillTemplateListOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nraw: %s", err, out)
	}

	for _, tmpl := range result.Templates {
		if tmpl.Name == "gh-cli" {
			if !tmpl.Installed {
				t.Error("expected gh-cli to show as installed")
			}
		} else {
			if tmpl.Installed {
				t.Errorf("expected %q to show as not installed", tmpl.Name)
			}
		}
	}
}

func TestSkillInstallCopiesTemplate(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	if err := os.MkdirAll(".wave/skills", 0755); err != nil {
		t.Fatal(err)
	}

	out, err := executeSkillCmd("install", "gh-cli")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "gh-cli") {
		t.Errorf("expected skill name in output, got: %s", out)
	}

	// Verify file was created
	destFile := filepath.Join(".wave", "skills", "gh-cli", "SKILL.md")
	data, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("SKILL.md not created: %v", err)
	}

	// Verify content matches the embedded template
	templates := defaults.GetSkillTemplates()
	if !bytes.Equal(data, templates["gh-cli"]) {
		t.Error("installed SKILL.md content does not match embedded template")
	}
}

func TestSkillInstallJSON(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	if err := os.MkdirAll(".wave/skills", 0755); err != nil {
		t.Fatal(err)
	}

	out, err := executeSkillCmd("install", "docker", "--format", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result SkillTemplateInstallOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nraw: %s", err, out)
	}

	if result.Name != "docker" {
		t.Errorf("expected name 'docker', got %q", result.Name)
	}
	if result.Destination == "" {
		t.Error("expected non-empty destination")
	}
}

func TestSkillInstallErrorIfAlreadyExists(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	env.createSkill("gh-cli", "GitHub CLI operations")

	_, err := executeSkillCmd("install", "gh-cli")
	if err == nil {
		t.Fatal("expected error for already installed skill")
	}

	cliErr, ok := err.(*CLIError)
	if !ok {
		t.Fatalf("expected *CLIError, got %T: %v", err, err)
	}
	if cliErr.Code != CodeSkillAlreadyExists {
		t.Errorf("expected code %q, got %q", CodeSkillAlreadyExists, cliErr.Code)
	}
}

func TestSkillInstallErrorIfNotFound(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	if err := os.MkdirAll(".wave/skills", 0755); err != nil {
		t.Fatal(err)
	}

	_, err := executeSkillCmd("install", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent template")
	}

	cliErr, ok := err.(*CLIError)
	if !ok {
		t.Fatalf("expected *CLIError, got %T: %v", err, err)
	}
	if cliErr.Code != CodeSkillNotFound {
		t.Errorf("expected code %q, got %q", CodeSkillNotFound, cliErr.Code)
	}
	// Should list available templates
	if !strings.Contains(cliErr.Suggestion, "gh-cli") {
		t.Errorf("expected available templates in suggestion, got: %s", cliErr.Suggestion)
	}
}

func TestSkillInstallRequiresArg(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	_, err := executeSkillCmd("install")
	if err == nil {
		t.Fatal("expected error for missing argument")
	}
}
