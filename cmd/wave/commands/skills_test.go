package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// skillTestEnv provides a testing environment for skills tests.
type skillTestEnv struct {
	t       *testing.T
	rootDir string
	origDir string
}

func newSkillTestEnv(t *testing.T) *skillTestEnv {
	t.Helper()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	return &skillTestEnv{t: t, rootDir: tmpDir, origDir: origDir}
}

func (e *skillTestEnv) cleanup() {
	if err := os.Chdir(e.origDir); err != nil {
		e.t.Errorf("failed to restore directory: %v", err)
	}
}

// createSkill creates a SKILL.md in the given skills root directory.
func (e *skillTestEnv) createSkill(root, name, description string) {
	e.t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		e.t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: " + description + "\n---\n\nSkill body.\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644); err != nil {
		e.t.Fatal(err)
	}
}

// executeSkillsCmd runs the skills command with given arguments and captures output.
func executeSkillsCmd(args ...string) (stdout string, err error) {
	cmd := NewSkillsCmd()

	// We need to capture stdout because JSON output goes to os.Stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(args)

	err = cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	var pipeBuf bytes.Buffer
	pipeBuf.ReadFrom(r)

	// Combine cmd.SetOut output and os.Stdout output
	combined := outBuf.String() + pipeBuf.String()
	return combined, err
}

// T019: TestSkillsListEmpty
func TestSkillsListEmpty(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	// Create empty skill directories
	os.MkdirAll(".wave/skills", 0755)

	out, err := executeSkillsCmd("list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "No skills installed") {
		t.Errorf("expected 'No skills installed' message, got: %s", out)
	}
	if !strings.Contains(out, "wave skills install") {
		t.Errorf("expected install hint in output, got: %s", out)
	}
}

// T020: TestSkillsListWithSkills
func TestSkillsListWithSkills(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	env.createSkill(".wave/skills", "golang", "Go development skill")
	env.createSkill(".wave/skills", "python", "Python development skill")

	out, err := executeSkillsCmd("list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "golang") {
		t.Errorf("expected 'golang' in output, got: %s", out)
	}
	if !strings.Contains(out, "Go development skill") {
		t.Errorf("expected description in output, got: %s", out)
	}
	if !strings.Contains(out, "python") {
		t.Errorf("expected 'python' in output, got: %s", out)
	}
}

// T021: TestSkillsListJSON
func TestSkillsListJSON(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	env.createSkill(".wave/skills", "golang", "Go development skill")

	out, err := executeSkillsCmd("list", "--format", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result SkillListOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nraw: %s", err, out)
	}
	if len(result.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(result.Skills))
	}
	if result.Skills[0].Name != "golang" {
		t.Errorf("expected skill name 'golang', got %q", result.Skills[0].Name)
	}
	if result.Skills[0].Description != "Go development skill" {
		t.Errorf("expected description 'Go development skill', got %q", result.Skills[0].Description)
	}
}

// T022: TestSkillsListDiscoveryWarnings
func TestSkillsListDiscoveryWarnings(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	// Create a valid skill
	env.createSkill(".wave/skills", "golang", "Go development skill")

	// Create a malformed SKILL.md (name mismatch triggers a DiscoveryError)
	dir := filepath.Join(".wave/skills", "badskill")
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: wrong-name\ndescription: bad\n---\n"), 0644)

	out, err := executeSkillsCmd("list", "--format", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result SkillListOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nraw: %s", err, out)
	}

	// Should have the valid skill
	if len(result.Skills) != 1 {
		t.Errorf("expected 1 valid skill, got %d", len(result.Skills))
	}
	// Should have warnings about the malformed skill
	if len(result.Warnings) == 0 {
		t.Error("expected warnings for malformed SKILL.md, got none")
	}
}

// T023: TestSkillsInstallUnknownPrefix
func TestSkillsInstallUnknownPrefix(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	os.MkdirAll(".wave/skills", 0755)

	_, err := executeSkillsCmd("install", "unknown:something")
	if err == nil {
		t.Fatal("expected error for unknown prefix")
	}
	cliErr, ok := err.(*CLIError)
	if !ok {
		t.Fatalf("expected *CLIError, got %T: %v", err, err)
	}
	if cliErr.Code != CodeSkillSourceError {
		t.Errorf("expected code %q, got %q", CodeSkillSourceError, cliErr.Code)
	}
	// Should mention recognized prefixes
	if !strings.Contains(cliErr.Suggestion, "tessl:") {
		t.Errorf("expected recognized prefix list in suggestion, got: %s", cliErr.Suggestion)
	}
}

// T024: TestSkillsInstallNoArgs
func TestSkillsInstallNoArgs(t *testing.T) {
	_, err := executeSkillsCmd("install")
	if err == nil {
		t.Fatal("expected error for missing args")
	}
}

// T025: TestSkillsInstallFileSource
func TestSkillsInstallFileSource(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	// Create the source skill directory
	srcDir := filepath.Join(env.rootDir, "my-skill")
	os.MkdirAll(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "SKILL.md"), []byte("---\nname: my-skill\ndescription: Test skill\n---\nBody.\n"), 0644)

	// Create target store directory
	os.MkdirAll(".wave/skills", 0755)

	out, err := executeSkillsCmd("install", "file:"+srcDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "my-skill") {
		t.Errorf("expected installed skill name in output, got: %s", out)
	}
}

// T026: TestSkillsInstallJSON
func TestSkillsInstallJSON(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	// Create a source skill
	srcDir := filepath.Join(env.rootDir, "test-skill")
	os.MkdirAll(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "SKILL.md"), []byte("---\nname: test-skill\ndescription: Test\n---\nBody.\n"), 0644)

	os.MkdirAll(".wave/skills", 0755)

	out, err := executeSkillsCmd("install", "file:"+srcDir, "--format", "json")
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
}

// T027: TestSkillsRemoveExisting
func TestSkillsRemoveExisting(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	env.createSkill(".wave/skills", "golang", "Go development skill")

	out, err := executeSkillsCmd("remove", "golang", "--yes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Removed") || !strings.Contains(out, "golang") {
		t.Errorf("expected removal confirmation, got: %s", out)
	}

	// Verify skill is gone
	if _, statErr := os.Stat(filepath.Join(".wave/skills", "golang")); !os.IsNotExist(statErr) {
		t.Error("expected skill directory to be deleted")
	}
}

// T028: TestSkillsRemoveNonexistent
func TestSkillsRemoveNonexistent(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	os.MkdirAll(".wave/skills", 0755)

	_, err := executeSkillsCmd("remove", "nonexistent", "--yes")
	if err == nil {
		t.Fatal("expected error for non-existent skill")
	}
	cliErr, ok := err.(*CLIError)
	if !ok {
		t.Fatalf("expected *CLIError, got %T: %v", err, err)
	}
	if cliErr.Code != CodeSkillNotFound {
		t.Errorf("expected code %q, got %q", CodeSkillNotFound, cliErr.Code)
	}
}

// T029: TestSkillsRemoveConfirmation
func TestSkillsRemoveConfirmation(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	env.createSkill(".wave/skills", "golang", "Go development skill")

	// Test confirmation with "y"
	cmd := NewSkillsCmd()
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"remove", "golang"})

	// Override the remove command's RunE to inject test stdin
	for _, sub := range cmd.Commands() {
		if sub.Name() == "remove" {
			origRunE := sub.RunE
			sub.RunE = func(c *cobra.Command, args []string) error {
				_ = origRunE
				var promptBuf bytes.Buffer
				return runSkillsRemove(c, args[0], "table", false, strings.NewReader("y\n"), &promptBuf)
			}
		}
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	err := cmd.Execute()
	w.Close()
	os.Stdout = oldStdout
	var pipeBuf bytes.Buffer
	pipeBuf.ReadFrom(r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, statErr := os.Stat(filepath.Join(".wave/skills", "golang")); !os.IsNotExist(statErr) {
		t.Error("expected skill to be deleted after 'y' confirmation")
	}

	// Test confirmation with "n" — recreate skill
	env.createSkill(".wave/skills", "golang", "Go development skill")

	cmd2 := NewSkillsCmd()
	cmd2.SetOut(&bytes.Buffer{})
	cmd2.SetErr(&bytes.Buffer{})
	cmd2.SetArgs([]string{"remove", "golang"})

	for _, sub := range cmd2.Commands() {
		if sub.Name() == "remove" {
			sub.RunE = func(c *cobra.Command, args []string) error {
				var promptBuf bytes.Buffer
				return runSkillsRemove(c, args[0], "table", false, strings.NewReader("n\n"), &promptBuf)
			}
		}
	}

	oldStdout = os.Stdout
	r, w, _ = os.Pipe()
	os.Stdout = w
	err = cmd2.Execute()
	w.Close()
	os.Stdout = oldStdout
	pipeBuf.Reset()
	pipeBuf.ReadFrom(r)

	if err != nil {
		t.Fatalf("unexpected error on cancel: %v", err)
	}

	// Skill should still exist
	if _, statErr := os.Stat(filepath.Join(".wave/skills", "golang")); os.IsNotExist(statErr) {
		t.Error("expected skill to still exist after 'n' confirmation")
	}
}

// T030: TestSkillsRemoveYesFlag
func TestSkillsRemoveYesFlag(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	env.createSkill(".wave/skills", "golang", "Go development skill")

	out, err := executeSkillsCmd("remove", "golang", "--yes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Removed") {
		t.Errorf("expected removal message, got: %s", out)
	}
}

// T031: TestSkillsRemoveJSON
func TestSkillsRemoveJSON(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	env.createSkill(".wave/skills", "golang", "Go development skill")

	out, err := executeSkillsCmd("remove", "golang", "--yes", "--format", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result SkillRemoveOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\nraw: %s", err, out)
	}
	if result.Removed != "golang" {
		t.Errorf("expected removed 'golang', got %q", result.Removed)
	}
}

// T032: TestSkillsSearchMissingTessl
func TestSkillsSearchMissingTessl(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	// Ensure tessl is not in PATH by using empty PATH
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", origPath)

	_, err := executeSkillsCmd("search", "golang")
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

// T033: TestSkillsSyncMissingTessl
func TestSkillsSyncMissingTessl(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", origPath)

	_, err := executeSkillsCmd("sync")
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

// T034: TestSkillsNoSubcommand
func TestSkillsNoSubcommand(t *testing.T) {
	cmd := NewSkillsCmd()
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := outBuf.String()
	// Should show help text listing subcommands
	if !strings.Contains(out, "list") {
		t.Errorf("expected 'list' in help output, got: %s", out)
	}
	if !strings.Contains(out, "install") {
		t.Errorf("expected 'install' in help output, got: %s", out)
	}
	if !strings.Contains(out, "remove") {
		t.Errorf("expected 'remove' in help output, got: %s", out)
	}
	if !strings.Contains(out, "search") {
		t.Errorf("expected 'search' in help output, got: %s", out)
	}
	if !strings.Contains(out, "sync") {
		t.Errorf("expected 'sync' in help output, got: %s", out)
	}
}
