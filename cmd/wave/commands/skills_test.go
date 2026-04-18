package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// skillTestEnv provides a testing environment for skills tests.
type skillTestEnv struct {
	t       *testing.T
	rootDir string
	origDir string
	homeDir string
	origHome string
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
	homeDir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", homeDir)
	return &skillTestEnv{t: t, rootDir: tmpDir, origDir: origDir, homeDir: homeDir, origHome: origHome}
}

func (e *skillTestEnv) cleanup() {
	if err := os.Chdir(e.origDir); err != nil {
		e.t.Errorf("failed to restore directory: %v", err)
	}
}

// createSkill creates a SKILL.md in the given root.
func (e *skillTestEnv) createSkillIn(root, name, description string) {
	e.t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		e.t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: " + description + "\n---\n\nSkill body.\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		e.t.Fatal(err)
	}
}

func (e *skillTestEnv) createProjectSkill(name, description string) {
	e.createSkillIn(".agents/skills", name, description)
}

// executeSkillsCmd runs the skills command and captures stdout.
func executeSkillsCmd(args ...string) (string, error) {
	cmd := NewSkillsCmd()

	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(args)

	err := cmd.Execute()
	return outBuf.String(), err
}

// --- list ---

func TestSkillsListEmpty(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	if err := os.MkdirAll(".agents/skills", 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := executeSkillsCmd("list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "No skills installed") {
		t.Errorf("expected 'No skills installed' message, got: %s", out)
	}
	if !strings.Contains(out, "wave skills add") {
		t.Errorf("expected add hint, got: %s", out)
	}
}

func TestSkillsListWithSkills(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	env.createProjectSkill("my-skill", "Test skill description")

	out, err := executeSkillsCmd("list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "my-skill") {
		t.Errorf("expected skill name in output, got: %s", out)
	}
	if !strings.Contains(out, "Test skill description") {
		t.Errorf("expected description in output, got: %s", out)
	}
}

func TestSkillsListJSON(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	env.createProjectSkill("alpha", "Alpha skill")

	out, err := executeSkillsCmd("list", "--format", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var output SkillListOutput
	if err := json.Unmarshal([]byte(out), &output); err != nil {
		t.Fatalf("failed to parse JSON: %v\noutput: %s", err, out)
	}
	if len(output.Skills) != 1 || output.Skills[0].Name != "alpha" {
		t.Errorf("expected 1 skill named alpha, got: %+v", output.Skills)
	}
}

// --- check ---

func TestSkillsCheckExisting(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	env.createProjectSkill("foo", "Foo description")

	out, err := executeSkillsCmd("check", "foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "OK") || !strings.Contains(out, "foo") {
		t.Errorf("expected OK + name, got: %s", out)
	}
}

func TestSkillsCheckMissing(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	if err := os.MkdirAll(".agents/skills", 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := executeSkillsCmd("check", "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing skill")
	}
}

func TestSkillsCheckJSON(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	env.createProjectSkill("bar", "Bar description")

	out, err := executeSkillsCmd("check", "bar", "--format", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var output SkillCheckOutput
	if err := json.Unmarshal([]byte(out), &output); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if output.Name != "bar" || !output.OK {
		t.Errorf("expected ok=true name=bar, got: %+v", output)
	}
}

// --- add ---

func TestSkillsAddNoArgs(t *testing.T) {
	_, err := executeSkillsCmd("add")
	if err == nil {
		t.Fatal("expected error for no args")
	}
}

func TestSkillsAddProjectFromLocalPath(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	src := filepath.Join(env.rootDir, "src-skill")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: imported\ndescription: Imported skill\n---\n\nBody.\n"
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := executeSkillsCmd("add", "file:"+src, "--project", "--format", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v\nout: %s", err, out)
	}
	var output SkillAddOutput
	if err := json.Unmarshal([]byte(out), &output); err != nil {
		t.Fatalf("failed to parse JSON: %v\nout: %s", err, out)
	}
	if len(output.Installed) != 1 || output.Installed[0] != "imported" {
		t.Errorf("expected installed=[imported], got: %+v", output)
	}
	if _, statErr := os.Stat(filepath.Join(".agents/skills", "imported", "SKILL.md")); statErr != nil {
		t.Errorf("expected SKILL.md installed: %v", statErr)
	}
}

// --- doctor ---

func TestSkillsDoctorClean(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	if err := os.MkdirAll(".agents/skills", 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := executeSkillsCmd("doctor", "--format", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var output SkillDoctorOutput
	if err := json.Unmarshal([]byte(out), &output); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if !output.OK {
		t.Errorf("expected ok=true, got: %+v", output)
	}
}

func TestSkillsDoctorDeprecated(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	if err := os.MkdirAll(".wave/skills", 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := executeSkillsCmd("doctor", "--format", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var output SkillDoctorOutput
	if err := json.Unmarshal([]byte(out), &output); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if output.OK {
		t.Errorf("expected ok=false with deprecated dir, got: %+v", output)
	}
	if len(output.Deprecated) == 0 {
		t.Errorf("expected deprecated entry, got: %+v", output.Deprecated)
	}
}

// --- top-level ---

func TestSkillsNoSubcommand(t *testing.T) {
	out, err := executeSkillsCmd()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, sub := range []string{"list", "check", "add", "doctor"} {
		if !strings.Contains(out, sub) {
			t.Errorf("expected %q in help output", sub)
		}
	}
}

func TestClassifySkillError(t *testing.T) {
	cliErr := classifySkillError(errNotFoundForTest{})
	if cliErr == nil {
		t.Fatal("expected CLIError")
	}
}

type errNotFoundForTest struct{}

func (errNotFoundForTest) Error() string { return "not found-ish" }
