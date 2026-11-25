package commands

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func gitOut(t *testing.T, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", args...).CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)
	return string(out)
}

// The interactive wizard asks for git, adapter, model and every project
// command it could not detect (regression: init asked nothing on a TTY
// although the docs promised a wizard).
func TestInitInteractiveWizard(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()
	t.Setenv("WAVE_FORCE_TTY", "1")

	cmd := NewInitCmd()
	// git: default yes; adapter: codex; model: default; test: make test; lint/build: skip
	cmd.SetIn(strings.NewReader("\ncodex\n\nmake test\n\n\n"))
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	require.NoError(t, cmd.Execute(), errBuf.String())

	assert.Contains(t, errBuf.String(), "Initialize a git repository")
	assert.Contains(t, errBuf.String(), "Adapter [claude/codex]")
	assert.Contains(t, errBuf.String(), "Model for codex")
	assert.Contains(t, errBuf.String(), "Test command (not detected")

	m, err := readYAML("wave.yaml")
	require.NoError(t, err)
	adapters := m["adapters"].(map[string]interface{})
	require.Len(t, adapters, 1)
	assert.Equal(t, "gpt-6.2-sol", adapters["codex"].(map[string]interface{})["default_model"])
	project := m["project"].(map[string]interface{})
	assert.Equal(t, "make test", project["test_command"])
	assert.NotContains(t, project, "lint_command")
	assert.Contains(t, out.String(), "Missing project commands: lint, build")
}

// --reconfigure keeps customised metadata, models and persona prompts, and
// writes a backup (regression: it silently rewrote everything).
func TestInitReconfigurePreservesCustomisations(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	_, _, err := executeInitCmd("--adapter", "claude", "--yes")
	require.NoError(t, err)

	data, err := os.ReadFile("wave.yaml")
	require.NoError(t, err)
	var doc map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &doc))
	doc["metadata"].(map[string]interface{})["name"] = "my-custom-name"
	doc["adapters"].(map[string]interface{})["claude"].(map[string]interface{})["default_model"] = "opus"
	customBytes, err := yaml.Marshal(doc)
	require.NoError(t, err)
	custom := string(customBytes)
	require.NoError(t, os.WriteFile("wave.yaml", []byte(custom), 0o644))
	persona := filepath.Join(".agents", "personas", "craftsman.md")
	orig, err := os.ReadFile(persona)
	require.NoError(t, err)
	edited := string(orig) + "\nMy custom rule.\n"
	require.NoError(t, os.WriteFile(persona, []byte(edited), 0o644))

	stdout, _, err := executeInitCmd("--reconfigure")
	require.NoError(t, err)
	assert.Contains(t, stdout, "Configuration updated successfully")

	m, err := readYAML("wave.yaml")
	require.NoError(t, err)
	assert.Equal(t, "my-custom-name", m["metadata"].(map[string]interface{})["name"])
	claude := m["adapters"].(map[string]interface{})["claude"].(map[string]interface{})
	assert.Equal(t, "opus", claude["default_model"])
	got, err := os.ReadFile(persona)
	require.NoError(t, err)
	assert.Equal(t, edited, string(got), "edited persona prompt must survive")
	bak, err := os.ReadFile("wave.yaml.bak")
	require.NoError(t, err)
	assert.Equal(t, custom, string(bak))
}

// --reconfigure --adapter switches every persona and pins the new model.
func TestInitReconfigureSwitchesAdapter(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	_, _, err := executeInitCmd("--adapter", "claude", "--yes")
	require.NoError(t, err)
	_, _, err = executeInitCmd("--reconfigure", "--adapter", "codex", "--yes")
	require.NoError(t, err)

	m, err := readYAML("wave.yaml")
	require.NoError(t, err)
	codex := m["adapters"].(map[string]interface{})["codex"].(map[string]interface{})
	assert.Equal(t, "gpt-6.2-sol", codex["default_model"])
	for name, p := range m["personas"].(map[string]interface{}) {
		assert.Equal(t, "codex", p.(map[string]interface{})["adapter"], "persona %s", name)
	}
}

// Re-running init must not add back adapters the user removed
// (regression: gemini/opencode were re-added on every run).
func TestInitRerunDoesNotReaddAdapters(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	_, _, err := executeInitCmd("--adapter", "codex", "--yes")
	require.NoError(t, err)
	require.NoError(t, os.Remove(filepath.Join(".agents", "pipelines", "plan-task.yaml")))

	_, _, err = executeInitCmd("--yes")
	require.NoError(t, err)
	m, err := readYAML("wave.yaml")
	require.NoError(t, err)
	adapters := m["adapters"].(map[string]interface{})
	assert.Len(t, adapters, 1)
	assert.Contains(t, adapters, "codex")
	assert.True(t, fileExists(filepath.Join(".agents", "pipelines", "plan-task.yaml")))
}

// Non-interactive re-run with missing files points at --yes, not --force.
func TestInitMergeNonInteractiveHint(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	_, _, err := executeInitCmd("--yes")
	require.NoError(t, err)
	require.NoError(t, os.Remove(filepath.Join(".agents", "pipelines", "plan-task.yaml")))

	_, _, err = executeInitCmd()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--yes")
	assert.NotContains(t, err.Error(), "--force")
}

// --force without input fails with an actionable message instead of
// "failed to read confirmation: EOF".
func TestInitForceWithoutInput(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()
	require.NoError(t, os.WriteFile("wave.yaml", []byte("apiVersion: v1\n"), 0o644))

	cmd := NewInitCmd()
	cmd.SetArgs([]string{"--force"})
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--force --yes")
	assert.NotContains(t, err.Error(), "EOF")
}

// init writes AGENTS.md with the project commands, CLAUDE.md importing it
// for claude, no GEMINI.md/CODEX.md, a .gitignore for runtime state, and the
// initial commit contains every file it created (regression: four stub
// files pointing at each other, runtime state untracked, partial commit).
func TestInitInstructionFilesGitignoreAndCommit(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()
	require.NoError(t, os.WriteFile("go.mod", []byte("module example.com/calc\n"), 0o644))
	require.NoError(t, os.WriteFile("README.md", []byte("# calc\n\nA tiny calculator module.\n"), 0o644))

	stdout, _, err := executeInitCmd("--adapter", "claude", "--yes")
	require.NoError(t, err)

	agents, err := os.ReadFile("AGENTS.md")
	require.NoError(t, err)
	assert.Contains(t, string(agents), "# calc")
	assert.Contains(t, string(agents), "A tiny calculator module.")
	assert.Contains(t, string(agents), "`go test ./...`")
	claudeMD, err := os.ReadFile("CLAUDE.md")
	require.NoError(t, err)
	assert.Equal(t, "@AGENTS.md\n", string(claudeMD))
	assert.False(t, fileExists("GEMINI.md"))
	assert.False(t, fileExists("CODEX.md"))

	gi, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	for _, e := range []string{".agents/workspaces/", ".agents/state.db*", ".agents/traces/"} {
		assert.Contains(t, string(gi), e+"\n")
	}
	assert.Contains(t, stdout, "AGENTS.md")
	assert.Contains(t, stdout, ".gitignore")

	m, err := readYAML("wave.yaml")
	require.NoError(t, err)
	assert.Equal(t, "A tiny calculator module.", m["metadata"].(map[string]interface{})["description"])

	// Everything init wrote is committed or ignored; only the user's own
	// files stay untracked.
	status := gitOut(t, "status", "--porcelain", "--untracked-files=all")
	assert.Equal(t, "?? README.md\n?? go.mod\n", status)
	tracked := gitOut(t, "ls-files")
	for _, f := range []string{"AGENTS.md", "CLAUDE.md", ".gitignore", "wave.yaml", ".agents/.onboarding-done"} {
		assert.Contains(t, tracked, f+"\n")
	}
	assert.Empty(t, strings.TrimSpace(gitOutAllowFail("config", "--local", "user.email")), "init must not write a repo-local identity")
}

func gitOutAllowFail(args ...string) string {
	out, _ := exec.Command("git", args...).Output()
	return string(out)
}

// --workspace creates the configured workspace root instead of
// .agents/workspaces.
func TestInitWorkspaceFlagCreatesDir(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	_, _, err := executeInitCmd("--workspace", "./ws", "--yes")
	require.NoError(t, err)
	assert.True(t, dirExists("ws"))
	assert.False(t, dirExists(filepath.Join(".agents", "workspaces")))
}

// The global -m/--manifest selects the output path when --manifest-path is
// not given, and next steps name the manifest for wave validate.
func TestInitHonoursGlobalManifestFlag(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	root := &cobra.Command{Use: "wave"}
	root.PersistentFlags().StringP("manifest", "m", "wave.yaml", "")
	root.AddCommand(NewInitCmd())
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"init", "-m", "cfg/wave.yaml", "-y"})
	require.NoError(t, root.Execute())

	assert.True(t, fileExists(filepath.Join("cfg", "wave.yaml")))
	assert.False(t, fileExists("wave.yaml"))
	// Manifest paths resolve relative to the manifest directory, so the
	// .agents tree must live next to it (regression: validate -m failed on
	// every persona prompt).
	assert.True(t, fileExists(filepath.Join("cfg", ".agents", "personas", "craftsman.md")))
	assert.False(t, dirExists(".agents"))
	assert.Contains(t, out.String(), "wave validate --manifest cfg/wave.yaml")
}

// Flags fill in project commands for projects without a detectable build.
func TestInitProjectCommandFlags(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	stdout, _, err := executeInitCmd("--yes", "--test-command", "make check", "--lint-command", "make lint", "--build-command", "make")
	require.NoError(t, err)
	m, err := readYAML("wave.yaml")
	require.NoError(t, err)
	project := m["project"].(map[string]interface{})
	assert.Equal(t, "make check", project["test_command"])
	assert.Equal(t, "make lint", project["lint_command"])
	assert.Equal(t, "make", project["build_command"])
	assert.NotContains(t, stdout, "Missing project commands")
}

// An empty directory gets a clear hint about the missing commands.
func TestInitEmptyDirWarnsAboutMissingCommands(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	stdout, _, err := executeInitCmd("--yes")
	require.NoError(t, err)
	assert.Contains(t, stdout, "Missing project commands: test, lint, build")
	assert.Contains(t, stdout, "wave doctor --optimize")
}

// Running init in a subdirectory of a repository warns about it.
func TestInitWarnsInSubdirectory(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()
	gitOut(t, "init", "-q")
	require.NoError(t, os.MkdirAll("inner", 0o755))
	require.NoError(t, os.Chdir("inner"))

	_, stderr, err := executeInitCmd("--yes")
	require.NoError(t, err)
	assert.Contains(t, stderr, "subdirectory of the git repository")
}

// The banner counts personas from the manifest, not prompt files.
func TestInitBannerPersonaCount(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	stdout, _, err := executeInitCmd("--yes")
	require.NoError(t, err)
	m, err := readYAML("wave.yaml")
	require.NoError(t, err)
	n := len(m["personas"].(map[string]interface{}))
	assert.Contains(t, stdout, ".agents/personas/          "+strconv.Itoa(n)+" personas")
}

// Commands that define their own --manifest flag shadow the global one, so
// they must keep the -m shorthand the root help advertises
// (regression: `wave validate -m x` -> unknown shorthand flag 'm').
func TestLocalManifestFlagsKeepShorthand(t *testing.T) {
	for _, c := range []*cobra.Command{
		NewValidateCmd(), NewRunCmd(), NewResumeCmd(), NewMetaCmd(), NewListCmd(), NewStatusCmd(),
		NewLogsCmd(), NewArtifactsCmd(), NewForkCmd(), NewRewindCmd(), NewDecisionsCmd(),
	} {
		f := c.Flags().Lookup("manifest")
		if f == nil {
			continue
		}
		assert.Equal(t, "m", f.Shorthand, "wave %s --manifest", c.Name())
	}
}
