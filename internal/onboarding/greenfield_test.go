package onboarding

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/defaults"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// chdirTemp switches the working directory to a fresh temp dir for the test.
func chdirTemp(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(orig) })
	return dir
}

func git(t *testing.T, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", args...).CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)
	return strings.TrimSpace(string(out))
}

// goProjectRepo creates an uncommitted Go project in a fresh git repository
// with a GitHub remote and makes it the working directory.
func goProjectRepo(t *testing.T) string {
	t.Helper()
	dir := chdirTemp(t)
	git(t, "init", "-q")
	git(t, "remote", "add", "origin", "https://github.com/acme/calc.git")
	writeFile(t, "go.mod", "module example.com/calc\n\ngo 1.25\n")
	writeFile(t, "README.md", "# calc\n\nA tiny calculator module.\n")
	return dir
}

func readManifest(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var m map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &m))
	return m
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func TestGreenfield_GoProjectOnGitHub(t *testing.T) {
	goProjectRepo(t)
	var stderr bytes.Buffer

	res, err := Greenfield(GreenfieldOpts{
		OutputPath: "wave.yaml",
		Commands:   ProjectCommands{Lint: "golangci-lint run ./..."},
		Stderr:     &stderr,
	})
	require.NoError(t, err)

	assert.Equal(t, "claude", res.Adapter, "adapter defaults to claude")
	assert.Equal(t, "sonnet", res.Model, "model defaults to the adapter's pinned model")
	assert.Equal(t, "go", res.Flavour.Flavour)
	assert.Equal(t, []string{"AGENTS.md", "CLAUDE.md"}, res.InstructionFiles)
	assert.True(t, res.GitignoreUpdated)
	assert.True(t, res.Committed)

	m := readManifest(t, "wave.yaml")
	meta := m["metadata"].(map[string]interface{})
	assert.Equal(t, "calc", meta["name"])
	assert.Equal(t, "A tiny calculator module.", meta["description"])
	assert.Equal(t, "acme/calc", meta["repo"])
	assert.Equal(t, "github", meta["forge"])

	project := m["project"].(map[string]interface{})
	assert.Equal(t, "go test ./...", project["test_command"])
	assert.Equal(t, "golangci-lint run ./...", project["lint_command"], "--lint-command overrides the detected value")
	assert.Equal(t, "go build ./...", project["build_command"])

	adapters := m["adapters"].(map[string]interface{})
	assert.Equal(t, []string{"claude"}, sortedKeys(adapters))
	assert.NotContains(t, m["runtime"].(map[string]interface{}), "workspace_root", "default workspaces live outside the checkout")

	// Personas: system personas and the GitHub ones are present, each with
	// its prompt file on disk; no other forge's personas leak in.
	personas := m["personas"].(map[string]interface{})
	for _, name := range []string{"navigator", "summarizer", "philosopher", "github-analyst"} {
		require.Contains(t, personas, name)
		p := personas[name].(map[string]interface{})
		assert.Equal(t, "claude", p["adapter"])
		assert.FileExists(t, p["system_prompt_file"].(string))
	}
	for name := range personas {
		assert.False(t, strings.HasPrefix(name, "gitlab-") || strings.HasPrefix(name, "gitea-"), name)
	}
	assert.FileExists(t, ".agents/personas/base-protocol.md")

	// Only release pipelines are installed by default.
	release, err := defaults.GetReleasePipelines()
	require.NoError(t, err)
	installed, err := os.ReadDir(".agents/pipelines")
	require.NoError(t, err)
	assert.Len(t, installed, len(release))
	for name := range release {
		assert.FileExists(t, filepath.Join(".agents", "pipelines", name))
	}
	assert.NoDirExists(t, ".agents/workspaces")
	assert.DirExists(t, ".agents/traces")

	agents, err := os.ReadFile("AGENTS.md")
	require.NoError(t, err)
	assert.Contains(t, string(agents), "# calc\n\nA tiny calculator module.\n")
	assert.Contains(t, string(agents), "- Test: `go test ./...`")
	assert.Contains(t, string(agents), "- Lint: `golangci-lint run ./...`")
	claude, err := os.ReadFile("CLAUDE.md")
	require.NoError(t, err)
	assert.Equal(t, "@AGENTS.md\n", string(claude))

	gitignore, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	assert.NotContains(t, string(gitignore), ".agents/workspaces/", "default workspaces are outside the checkout")
	for _, entry := range RuntimeGitignoreEntries {
		assert.Contains(t, string(gitignore), entry+"\n")
	}
	assert.FileExists(t, SentinelFile)

	assert.Equal(t, "chore: initialize wave project", git(t, "log", "-1", "--format=%s"))
	committed := git(t, "show", "--name-only", "--format=", "HEAD")
	for _, f := range []string{"wave.yaml", "AGENTS.md", "CLAUDE.md", ".gitignore", ".agents/pipelines/ops-hello-world.yaml"} {
		assert.Contains(t, committed, f)
	}
	assert.NotContains(t, committed, "go.mod", "only Wave's own files are committed")
	assert.Contains(t, stderr.String(), "Creating initial commit")
}

func TestGreenfield_CodexWorkspaceAndAll(t *testing.T) {
	goProjectRepo(t)
	git(t, "remote", "remove", "origin")
	git(t, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "existing")

	res, err := Greenfield(GreenfieldOpts{
		Adapter:    "codex",
		Workspace:  "tmp/ws",
		OutputPath: "config/wave.yaml",
		All:        true,
	})
	require.NoError(t, err)

	assert.Equal(t, "gpt-6.2-sol", res.Model)
	assert.Equal(t, []string{"AGENTS.md"}, res.InstructionFiles, "CLAUDE.md is only written for claude")
	assert.False(t, res.Committed, "an existing history is never committed to")
	assert.DirExists(t, "tmp/ws", "the configured workspace root is created")

	m := readManifest(t, "config/wave.yaml")
	meta := m["metadata"].(map[string]interface{})
	assert.NotContains(t, meta, "repo", "no remote, no repo slug")
	assert.NotContains(t, meta, "forge")
	codex := m["adapters"].(map[string]interface{})["codex"].(map[string]interface{})
	assert.Equal(t, "gpt-6.2-sol", codex["default_model"])
	assert.Equal(t, "tmp/ws", m["runtime"].(map[string]interface{})["workspace_root"])

	// --all installs every embedded pipeline and keeps forge personas.
	all, err := defaults.GetPipelines()
	require.NoError(t, err)
	installed, err := os.ReadDir(".agents/pipelines")
	require.NoError(t, err)
	assert.Len(t, installed, len(all))
	assert.Contains(t, m["personas"], "github-analyst")

	gitignore, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	assert.Contains(t, string(gitignore), "tmp/ws/\n")
}

func TestBaselineService_StartSession(t *testing.T) {
	dir := goProjectRepo(t)
	svc := NewBaselineService(nil)
	ui := &captureUI{}

	sess, err := svc.StartSession(context.Background(), "", StartOptions{
		Adapter:    "codex",
		OutputPath: "wave.yaml",
		UI:         ui,
	})
	require.NoError(t, err)

	assert.Equal(t, SessionDone, sess.Status)
	assert.Equal(t, ".", sess.ProjectDir, "empty project dir defaults to the working directory")
	assert.Len(t, sess.ID, 32)
	assert.Equal(t, "codex", sess.Result.Adapter)
	assert.Same(t, sess.Result.Assets, sess.Assets)
	assert.Equal(t, "go", sess.Flavour.Flavour)
	require.Len(t, ui.events, 2)
	assert.Equal(t, "starting non-interactive baseline onboarding", ui.events[0].Message)
	assert.Equal(t, "onboarding complete", ui.events[1].Message)
	assert.FileExists(t, filepath.Join(dir, SentinelFile))
}

func TestBaselineService_StartSession_CancelledContext(t *testing.T) {
	chdirTemp(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	sess, err := NewBaselineService(io.Discard).StartSession(ctx, "proj", StartOptions{})
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, SessionAborted, sess.Status)
	_, statErr := os.Stat("wave.yaml")
	assert.True(t, os.IsNotExist(statErr), "nothing is scaffolded for a cancelled session")
}

func TestBaselineService_StartSession_ScaffoldFailure(t *testing.T) {
	goProjectRepo(t)
	// A file where the output directory should go makes writing wave.yaml fail.
	writeFile(t, "blocked", "")

	sess, err := NewBaselineService(io.Discard).StartSession(context.Background(), "", StartOptions{
		OutputPath: "blocked/wave.yaml",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "greenfield scaffold")
	assert.Equal(t, SessionFailed, sess.Status)
}

func TestGitHelpers(t *testing.T) {
	dir := chdirTemp(t)
	var out bytes.Buffer

	require.NoError(t, EnsureGitRepo(&out))
	assert.Contains(t, out.String(), "Initializing git repository")
	assert.True(t, InsideGitRepo())
	top, err := filepath.EvalSymlinks(GitTopLevel())
	require.NoError(t, err)
	want, err := filepath.EvalSymlinks(dir)
	require.NoError(t, err)
	assert.Equal(t, want, top)

	out.Reset()
	require.NoError(t, EnsureGitRepo(&out))
	assert.Empty(t, out.String(), "an existing repository is left alone")

	writeFile(t, "a.txt", "a")
	writeFile(t, "b.txt", "b")
	committed, err := CreateInitialCommit(io.Discard, []string{"a.txt", "missing.txt"})
	require.NoError(t, err)
	assert.True(t, committed)
	assert.Equal(t, "a.txt", git(t, "show", "--name-only", "--format=", "HEAD"), "only existing listed paths are staged")

	committed, err = CreateInitialCommit(io.Discard, []string{"b.txt"})
	require.NoError(t, err)
	assert.False(t, committed, "no second commit once history exists")
}

func TestIsInteractive_ForcedByEnv(t *testing.T) {
	t.Setenv("WAVE_FORCE_TTY", "1")
	assert.True(t, IsInteractive())
	t.Setenv("WAVE_FORCE_TTY", "true")
	assert.True(t, IsInteractive())
	t.Setenv("WAVE_FORCE_TTY", "0")
	assert.False(t, IsInteractive())
}

func TestMarkDoneAt(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, MarkDoneAt(dir))
	sentinel := filepath.Join(dir, SentinelFile)
	require.FileExists(t, sentinel)

	require.NoError(t, os.WriteFile(sentinel, []byte("keep"), 0o644))
	require.NoError(t, MarkDoneAt(dir), "marking twice is a no-op")
	data, err := os.ReadFile(sentinel)
	require.NoError(t, err)
	assert.Equal(t, "keep", string(data))
}

func TestNoopUI_NotifyWritesToOut(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, NoopUI{Out: &out}.Notify(Event{Kind: "warning", Message: "careful"}))
	assert.Equal(t, "[warning] careful\n", out.String())

	got, err := NoopUI{}.PromptChoice(Question{Default: "x"})
	require.NoError(t, err)
	assert.Equal(t, "x", got, "no choices: the default is returned as-is")
}
