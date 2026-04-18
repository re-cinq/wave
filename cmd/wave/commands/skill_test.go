package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

	if err := os.MkdirAll(".agents/skills", 0755); err != nil {
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

	if err := os.MkdirAll(".agents/skills", 0755); err != nil {
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

	if err := os.MkdirAll(".agents/skills", 0755); err != nil {
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
	destFile := filepath.Join(".agents", "skills", "gh-cli", "SKILL.md")
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

	if err := os.MkdirAll(".agents/skills", 0755); err != nil {
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

	if err := os.MkdirAll(".agents/skills", 0755); err != nil {
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

// --- Remote source detection tests ---

func TestIsRemoteSource(t *testing.T) {
	tests := []struct {
		source string
		want   bool
	}{
		// Remote sources
		{"github:owner/repo", true},
		{"github:re-cinq/wave-skills/golang", true},
		{"tessl:spec-kit", true},
		{"tessl:github/spec-kit", true},
		{"https://example.com/skill.tar.gz", true},
		{"file:./my-skill", true},
		{"bmad:install", true},
		{"openspec:init", true},
		{"speckit:init", true},

		// Bare names — not remote
		{"gh-cli", false},
		{"docker", false},
		{"testing", false},
		{"my-custom-skill", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			got := isRemoteSource(tt.source)
			if got != tt.want {
				t.Errorf("isRemoteSource(%q) = %v, want %v", tt.source, got, tt.want)
			}
		})
	}
}

func TestSkillInstallDispatchesBundledForBareName(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	if err := os.MkdirAll(".agents/skills", 0755); err != nil {
		t.Fatal(err)
	}

	// Bare name should use bundled template install
	out, err := executeSkillCmd("install", "gh-cli")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "gh-cli") {
		t.Errorf("expected skill name in output, got: %s", out)
	}

	// Verify the file exists at the bundled template path
	destFile := filepath.Join(".agents", "skills", "gh-cli", "SKILL.md")
	if _, err := os.Stat(destFile); err != nil {
		t.Fatalf("expected SKILL.md to exist: %v", err)
	}
}

func TestSkillInstallNotFoundSuggestsRemoteSources(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	if err := os.MkdirAll(".agents/skills", 0755); err != nil {
		t.Fatal(err)
	}

	_, err := executeSkillCmd("install", "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}

	cliErr, ok := err.(*CLIError)
	if !ok {
		t.Fatalf("expected *CLIError, got %T: %v", err, err)
	}

	// Should mention remote sources in suggestion
	if !strings.Contains(cliErr.Suggestion, "github:") {
		t.Errorf("expected github: in suggestion, got: %s", cliErr.Suggestion)
	}
	if !strings.Contains(cliErr.Suggestion, "tessl:") {
		t.Errorf("expected tessl: in suggestion, got: %s", cliErr.Suggestion)
	}
}

func TestSkillInstallRemoteFileSource(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	// Create source skill directory
	srcDir := filepath.Join(env.rootDir, "my-remote-skill")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: my-remote-skill\ndescription: A test remote skill\n---\n# My Skill\n"
	if err := os.WriteFile(filepath.Join(srcDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create target store directory
	if err := os.MkdirAll(".agents/skills", 0755); err != nil {
		t.Fatal(err)
	}

	out, err := executeSkillCmd("install", "file:"+srcDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "my-remote-skill") {
		t.Errorf("expected skill name in output, got: %s", out)
	}

	// Verify skill was installed
	destFile := filepath.Join(".agents", "skills", "my-remote-skill", "SKILL.md")
	if _, err := os.Stat(destFile); err != nil {
		t.Fatalf("expected SKILL.md to be installed: %v", err)
	}
}

func TestSkillInstallRemoteFileSourceJSON(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	// Create source skill directory
	srcDir := filepath.Join(env.rootDir, "json-skill")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: json-skill\ndescription: A JSON test skill\n---\n# Skill\n"
	if err := os.WriteFile(filepath.Join(srcDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(".agents/skills", 0755); err != nil {
		t.Fatal(err)
	}

	out, err := executeSkillCmd("install", "file:"+srcDir, "--format", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result SkillInstallOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\nraw: %s", err, out)
	}
	if len(result.InstalledSkills) == 0 {
		t.Error("expected at least one installed skill")
	}
	if result.InstalledSkills[0] != "json-skill" {
		t.Errorf("expected 'json-skill', got %q", result.InstalledSkills[0])
	}
	if result.Source != "file:"+srcDir {
		t.Errorf("expected source to be %q, got %q", "file:"+srcDir, result.Source)
	}
}

func TestSkillInstallRemoteGitHubMissingGit(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	if err := os.MkdirAll(".agents/skills", 0755); err != nil {
		t.Fatal(err)
	}

	// Empty PATH so git is not found
	t.Setenv("PATH", "")

	_, err := executeSkillCmd("install", "github:owner/repo")
	if err == nil {
		t.Fatal("expected error for missing git")
	}
	cliErr, ok := err.(*CLIError)
	if !ok {
		t.Fatalf("expected *CLIError, got %T: %v", err, err)
	}
	if cliErr.Code != CodeSkillDependencyMissing {
		t.Errorf("expected code %q, got %q", CodeSkillDependencyMissing, cliErr.Code)
	}
}

func TestSkillInstallRemoteTesslMissingCLI(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	if err := os.MkdirAll(".agents/skills", 0755); err != nil {
		t.Fatal(err)
	}

	// Empty PATH so tessl is not found
	t.Setenv("PATH", "")

	_, err := executeSkillCmd("install", "tessl:spec-kit")
	if err == nil {
		t.Fatal("expected error for missing tessl")
	}
	cliErr, ok := err.(*CLIError)
	if !ok {
		t.Fatalf("expected *CLIError, got %T: %v", err, err)
	}
	if cliErr.Code != CodeSkillDependencyMissing {
		t.Errorf("expected code %q, got %q", CodeSkillDependencyMissing, cliErr.Code)
	}
}

// --- List --remote flag tests ---

func TestSkillListRemoteFlag(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	if err := os.MkdirAll(".agents/skills", 0755); err != nil {
		t.Fatal(err)
	}

	out, err := executeSkillCmd("list", "--remote")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should include remote source hints
	if !strings.Contains(out, "github:") {
		t.Errorf("expected github: hint in --remote output, got: %s", out)
	}
	if !strings.Contains(out, "tessl:") {
		t.Errorf("expected tessl: hint in --remote output, got: %s", out)
	}
	if !strings.Contains(out, "https://") {
		t.Errorf("expected https:// hint in --remote output, got: %s", out)
	}
}

func TestSkillListRemoteFlagStillShowsTemplates(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	if err := os.MkdirAll(".agents/skills", 0755); err != nil {
		t.Fatal(err)
	}

	out, err := executeSkillCmd("list", "--remote")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should still include bundled templates
	expected := defaults.SkillTemplateNames()
	for _, name := range expected {
		if !strings.Contains(out, name) {
			t.Errorf("expected template %q in --remote list output, got: %s", name, out)
		}
	}
}

func TestSkillListWithoutRemoteFlagNoHints(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	if err := os.MkdirAll(".agents/skills", 0755); err != nil {
		t.Fatal(err)
	}

	out, err := executeSkillCmd("list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Without --remote, should NOT include remote source hints
	if strings.Contains(out, "Remote sources also supported") {
		t.Errorf("did not expect remote hints without --remote flag, got: %s", out)
	}
}

// --- HTTP-based tests using httptest.NewServer ---

func TestSkillInstallHTTPSURLNotFound(t *testing.T) {
	// Use httptest to simulate a 404 response
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, "Not Found")
	}))
	defer server.Close()

	env := newSkillTestEnv(t)
	defer env.cleanup()

	if err := os.MkdirAll(".agents/skills", 0755); err != nil {
		t.Fatal(err)
	}

	// The URL adapter requires https:// and validates against SSRF (loopback).
	// httptest.NewTLSServer uses localhost which will be rejected by SSRF checks.
	// We test the dispatch logic by confirming it reaches the URL adapter.
	_, err := executeSkillCmd("install", "https://example.invalid/skill.tar.gz")
	if err == nil {
		t.Fatal("expected error for unreachable URL")
	}
	// Should be classified as a skill source error
	cliErr, ok := err.(*CLIError)
	if !ok {
		t.Fatalf("expected *CLIError, got %T: %v", err, err)
	}
	if cliErr.Code != CodeSkillSourceError {
		t.Errorf("expected code %q, got %q", CodeSkillSourceError, cliErr.Code)
	}

	// Suppress unused warning for server — demonstrates httptest pattern
	_ = server
}

func TestSkillInstallHTTPSURLInvalidArchiveFormat(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	if err := os.MkdirAll(".agents/skills", 0755); err != nil {
		t.Fatal(err)
	}

	// URL without a recognized archive extension should fail
	_, err := executeSkillCmd("install", "https://example.invalid/skill.txt")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
	cliErr, ok := err.(*CLIError)
	if !ok {
		t.Fatalf("expected *CLIError, got %T: %v", err, err)
	}
	if cliErr.Code != CodeSkillSourceError {
		t.Errorf("expected code %q, got %q", CodeSkillSourceError, cliErr.Code)
	}
}
