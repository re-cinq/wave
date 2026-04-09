package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/skill"
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

// createSkill creates a SKILL.md in the .wave/skills directory.
func (e *skillTestEnv) createSkill(name, description string) {
	e.t.Helper()
	dir := filepath.Join(".wave/skills", name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		e.t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: " + description + "\n---\n\nSkill body.\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644); err != nil {
		e.t.Fatal(err)
	}
}

// executeSkillsCmd runs the skills command with given arguments and captures output.
// Since all output (including JSON) goes through cmd.OutOrStdout(), we only need
// to capture via cmd.SetOut.
func executeSkillsCmd(args ...string) (stdout string, err error) {
	cmd := NewSkillsCmd()

	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(args)

	err = cmd.Execute()
	return outBuf.String(), err
}

// T019: TestSkillsListEmpty
func TestSkillsListEmpty(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	// Create empty skill directories
	if err := os.MkdirAll(".wave/skills", 0755); err != nil {
		t.Fatal(err)
	}

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

	env.createSkill("golang", "Go development skill")
	env.createSkill("python", "Python development skill")

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

	env.createSkill("golang", "Go development skill")

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
	env.createSkill("golang", "Go development skill")

	// Create a malformed SKILL.md (name mismatch triggers a DiscoveryError)
	dir := filepath.Join(".wave/skills", "badskill")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: wrong-name\ndescription: bad\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}

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

	if err := os.MkdirAll(".wave/skills", 0755); err != nil {
		t.Fatal(err)
	}

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
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "SKILL.md"), []byte("---\nname: my-skill\ndescription: Test skill\n---\nBody.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create target store directory
	if err := os.MkdirAll(".wave/skills", 0755); err != nil {
		t.Fatal(err)
	}

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
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "SKILL.md"), []byte("---\nname: test-skill\ndescription: Test\n---\nBody.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(".wave/skills", 0755); err != nil {
		t.Fatal(err)
	}

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

	env.createSkill("golang", "Go development skill")

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

	if err := os.MkdirAll(".wave/skills", 0755); err != nil {
		t.Fatal(err)
	}

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

	env.createSkill("golang", "Go development skill")

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

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, statErr := os.Stat(filepath.Join(".wave/skills", "golang")); !os.IsNotExist(statErr) {
		t.Error("expected skill to be deleted after 'y' confirmation")
	}

	// Test confirmation with "n" — recreate skill
	env.createSkill("golang", "Go development skill")

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

	err = cmd2.Execute()
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

	env.createSkill("golang", "Go development skill")

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

	env.createSkill("golang", "Go development skill")

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
	t.Setenv("PATH", "")

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

	t.Setenv("PATH", "")

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

// --- T013: TestSkillsHelpOutput — SC-007: --help includes all subcommand descriptions ---

func TestSkillsHelpOutput(t *testing.T) {
	out, err := executeSkillsCmd("--help")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, subcmd := range []string{"list", "install", "remove", "search", "sync"} {
		if !strings.Contains(out, subcmd) {
			t.Errorf("expected %q in --help output, got: %s", subcmd, out)
		}
	}

	// Verify it includes the long description
	if !strings.Contains(out, "Manage skills installed") {
		t.Errorf("expected command description in --help output, got: %s", out)
	}
}

// --- T014: TestParseTesslSearchOutput — unit test for search output parsing ---

func TestParseTesslSearchOutput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []SkillSearchResult
	}{
		{
			name:  "empty input",
			input: "",
			want:  nil,
		},
		{
			name:  "single result with 3+ fields",
			input: "golang ★★★ Go development skill\n",
			want: []SkillSearchResult{
				{Name: "golang", Rating: "★★★", Description: "Go development skill"},
			},
		},
		{
			name:  "multiple results",
			input: "golang ★★★ Go development\npython ★★ Python dev\nrust ★★★★ Rust programming\n",
			want: []SkillSearchResult{
				{Name: "golang", Rating: "★★★", Description: "Go development"},
				{Name: "python", Rating: "★★", Description: "Python dev"},
				{Name: "rust", Rating: "★★★★", Description: "Rust programming"},
			},
		},
		{
			name:  "single field line skipped",
			input: "onefield\ngolang ★★★ Go skill\n",
			want: []SkillSearchResult{
				{Name: "golang", Rating: "★★★", Description: "Go skill"},
			},
		},
		{
			name:  "empty lines ignored",
			input: "\n\ngolang ★★★ Go skill\n\n",
			want: []SkillSearchResult{
				{Name: "golang", Rating: "★★★", Description: "Go skill"},
			},
		},
		{
			name:  "two field line becomes name + description",
			input: "golang description-only\n",
			want: []SkillSearchResult{
				{Name: "golang", Description: "description-only"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTesslSearchOutput(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d results, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i].Name != tt.want[i].Name {
					t.Errorf("[%d] Name = %q, want %q", i, got[i].Name, tt.want[i].Name)
				}
				if got[i].Rating != tt.want[i].Rating {
					t.Errorf("[%d] Rating = %q, want %q", i, got[i].Rating, tt.want[i].Rating)
				}
				if got[i].Description != tt.want[i].Description {
					t.Errorf("[%d] Description = %q, want %q", i, got[i].Description, tt.want[i].Description)
				}
			}
		})
	}
}

// --- T015: TestParseTesslSyncOutput — unit test for sync output parsing ---

func TestParseTesslSyncOutput(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantSynced   []string
		wantWarnings []string
	}{
		{
			name:  "empty input",
			input: "",
		},
		{
			name:       "installed skills",
			input:      "installed golang\ninstalled spec-kit\n",
			wantSynced: []string{"golang", "spec-kit"},
		},
		{
			name:       "updated skills",
			input:      "updated golang\nupdated spec-kit\n",
			wantSynced: []string{"golang", "spec-kit"},
		},
		{
			name:         "warnings parsed",
			input:        "warning: foo\nWarning: bar\n",
			wantWarnings: []string{"foo", "bar"},
		},
		{
			name:         "mixed output with empty lines",
			input:        "installed golang\n\nupdated spec-kit\nwarning: foo\n\n",
			wantSynced:   []string{"golang", "spec-kit"},
			wantWarnings: []string{"foo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synced, warnings := parseTesslSyncOutput(tt.input)
			if len(synced) != len(tt.wantSynced) {
				t.Fatalf("synced: got %d, want %d: %v", len(synced), len(tt.wantSynced), synced)
			}
			for i := range synced {
				if synced[i] != tt.wantSynced[i] {
					t.Errorf("synced[%d] = %q, want %q", i, synced[i], tt.wantSynced[i])
				}
			}
			if len(warnings) != len(tt.wantWarnings) {
				t.Fatalf("warnings: got %d, want %d: %v", len(warnings), len(tt.wantWarnings), warnings)
			}
			for i := range warnings {
				if warnings[i] != tt.wantWarnings[i] {
					t.Errorf("warnings[%d] = %q, want %q", i, warnings[i], tt.wantWarnings[i])
				}
			}
		})
	}
}

// --- Audit tests ---

func TestSkillsAuditTable(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	env.createSkill("golang", "Go development skill")
	env.createSkill("python", "Python development skill")

	out, err := executeSkillsCmd("audit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "golang") {
		t.Errorf("expected golang in audit output, got: %s", out)
	}
	if !strings.Contains(out, "standalone") {
		t.Errorf("expected 'standalone' classification in output, got: %s", out)
	}
	if !strings.Contains(out, "total") {
		t.Errorf("expected summary in output, got: %s", out)
	}
}

func TestSkillsAuditJSON(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	env.createSkill("golang", "Go development skill")

	out, err := executeSkillsCmd("audit", "--format", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result SkillAuditOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\nraw: %s", err, out)
	}
	if len(result.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(result.Skills))
	}
	if result.Skills[0].Name != "golang" {
		t.Errorf("expected skill name 'golang', got %q", result.Skills[0].Name)
	}
	if result.Skills[0].Classification != "standalone" {
		t.Errorf("expected classification 'standalone', got %q", result.Skills[0].Classification)
	}
	if result.Summary.Total != 1 {
		t.Errorf("expected total 1, got %d", result.Summary.Total)
	}
}

func TestSkillsAuditEmpty(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	if err := os.MkdirAll(".wave/skills", 0755); err != nil {
		t.Fatal(err)
	}

	out, err := executeSkillsCmd("audit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "No skills found") {
		t.Errorf("expected 'No skills found' message, got: %s", out)
	}
}

// --- Publish tests ---

func TestSkillsPublishNoArgs(t *testing.T) {
	_, err := executeSkillsCmd("publish")
	if err == nil {
		t.Fatal("expected error for missing args")
	}
}

func TestSkillsPublishNonexistent(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	if err := os.MkdirAll(".wave/skills", 0755); err != nil {
		t.Fatal(err)
	}

	out, err := executeSkillsCmd("publish", "nonexistent", "--dry-run")
	// The command should either return an error or output indicating failure
	if err != nil {
		if !strings.Contains(err.Error(), "nonexistent") && !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected error to mention missing skill, got: %v", err)
		}
	} else {
		if !strings.Contains(out, "not found") && !strings.Contains(out, "error") {
			t.Errorf("expected output to indicate failure for nonexistent skill, got: %s", out)
		}
	}
}

func TestSkillsPublishDryRun(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	env.createSkill("golang", "Go development skill")

	out, err := executeSkillsCmd("publish", "golang", "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "golang") {
		t.Errorf("expected skill name in output, got: %s", out)
	}
}

func TestSkillsPublishDryRunJSON(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	env.createSkill("golang", "Go development skill")

	out, err := executeSkillsCmd("publish", "golang", "--dry-run", "--format", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result SkillPublishOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\nraw: %s", err, out)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].Name != "golang" {
		t.Errorf("expected name 'golang', got %q", result.Results[0].Name)
	}
	if result.Results[0].Status != "published" {
		t.Errorf("expected status 'published' for dry-run, got %q", result.Results[0].Status)
	}
}

func TestSkillsPublishAllConflict(t *testing.T) {
	_, err := executeSkillsCmd("publish", "golang", "--all")
	if err == nil {
		t.Fatal("expected error for --all with name arg")
	}
	cliErr, ok := err.(*CLIError)
	if !ok {
		t.Fatalf("expected *CLIError, got %T: %v", err, err)
	}
	if cliErr.Code != CodeFlagConflict {
		t.Errorf("expected code %q, got %q", CodeFlagConflict, cliErr.Code)
	}
}

// --- Verify tests ---

func TestSkillsVerifyNoLockfile(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	if err := os.MkdirAll(".wave", 0755); err != nil {
		t.Fatal(err)
	}

	out, err := executeSkillsCmd("verify")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "No published skills") {
		t.Errorf("expected 'No published skills' message, got: %s", out)
	}
}

func TestSkillsVerifyWithMatchingDigests(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	env.createSkill("golang", "Go development skill")

	// Compute actual digest
	store := skill.NewDirectoryStore(skill.SkillSource{Root: ".wave/skills", Precedence: 1})
	s, err := store.Read("golang")
	if err != nil {
		t.Fatal(err)
	}
	digest, err := skill.ComputeDigest(s)
	if err != nil {
		t.Fatal(err)
	}

	// Write lockfile with matching digest
	lockfile := fmt.Sprintf(`{"version":1,"published":[{"name":"golang","digest":%q,"registry":"tessl","url":"https://tessl.io","published_at":"2026-03-24T12:00:00Z"}]}`, digest)
	if err := os.WriteFile(".wave/skills.lock", []byte(lockfile), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := executeSkillsCmd("verify", "--format", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result SkillVerifyOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\nraw: %s", err, out)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].Status != "ok" {
		t.Errorf("expected status 'ok', got %q", result.Results[0].Status)
	}
	if result.Summary.OK != 1 {
		t.Errorf("expected OK count 1, got %d", result.Summary.OK)
	}
}

func TestSkillsVerifyModified(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	env.createSkill("golang", "Go development skill")

	// Write lockfile with wrong digest
	lockfile := `{"version":1,"published":[{"name":"golang","digest":"sha256:wrong","registry":"tessl","url":"https://tessl.io","published_at":"2026-03-24T12:00:00Z"}]}`
	if err := os.WriteFile(".wave/skills.lock", []byte(lockfile), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := executeSkillsCmd("verify", "--format", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result SkillVerifyOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\nraw: %s", err, out)
	}
	if result.Results[0].Status != "modified" {
		t.Errorf("expected status 'modified', got %q", result.Results[0].Status)
	}
}

func TestSkillsVerifyMissing(t *testing.T) {
	env := newSkillTestEnv(t)
	defer env.cleanup()

	if err := os.MkdirAll(".wave/skills", 0755); err != nil {
		t.Fatal(err)
	}

	// Write lockfile for a skill that doesn't exist locally
	lockfile := `{"version":1,"published":[{"name":"deleted-skill","digest":"sha256:abc","registry":"tessl","url":"https://tessl.io","published_at":"2026-03-24T12:00:00Z"}]}`
	if err := os.WriteFile(".wave/skills.lock", []byte(lockfile), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := executeSkillsCmd("verify", "--format", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result SkillVerifyOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\nraw: %s", err, out)
	}
	if result.Results[0].Status != "missing" {
		t.Errorf("expected status 'missing', got %q", result.Results[0].Status)
	}
	if result.Summary.Missing != 1 {
		t.Errorf("expected Missing count 1, got %d", result.Summary.Missing)
	}
}

// --- T016: TestClassifySkillError — verify all error code mappings ---

func TestClassifySkillError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode string
	}{
		{
			name: "DependencyError maps to CodeSkillDependencyMissing",
			err: &skill.DependencyError{
				Binary:       "tessl",
				Instructions: "npm i -g @tessl/cli",
			},
			wantCode: CodeSkillDependencyMissing,
		},
		{
			name:     "ErrNotFound maps to CodeSkillNotFound",
			err:      fmt.Errorf("wrap: %w", skill.ErrNotFound),
			wantCode: CodeSkillNotFound,
		},
		{
			name:     "unknown prefix string maps to CodeSkillSourceError",
			err:      fmt.Errorf("unknown source prefix \"bad\""),
			wantCode: CodeSkillSourceError,
		},
		{
			name:     "generic error maps to CodeSkillSourceError",
			err:      fmt.Errorf("something else went wrong"),
			wantCode: CodeSkillSourceError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cliErr := classifySkillError(tt.err)
			if cliErr.Code != tt.wantCode {
				t.Errorf("classifySkillError() code = %q, want %q", cliErr.Code, tt.wantCode)
			}
		})
	}
}
