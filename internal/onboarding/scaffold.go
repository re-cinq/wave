package onboarding

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/term"
)

// DefaultWaveDirs lists the directory tree that init/merge always ensures exists.
var DefaultWaveDirs = []string{
	".agents/personas",
	".agents/pipelines",
	".agents/contracts",
	".agents/prompts",
	".agents/traces",
	".agents/workspaces",
}

// EnsureWaveDirs creates the standard .agents directory tree.
func EnsureWaveDirs(dirs []string) error {
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			absDir, _ := filepath.Abs(dir)
			return fmt.Errorf("failed to create directory %s: %w", absDir, err)
		}
	}
	return nil
}

// IsInteractive reports whether stdin is a TTY (or WAVE_FORCE_TTY is truthy).
func IsInteractive() bool {
	if v := os.Getenv("WAVE_FORCE_TTY"); v != "" {
		return v == "1" || v == "true"
	}
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// InsideGitRepo reports whether the working directory is inside a git work
// tree.
func InsideGitRepo() bool {
	check := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	check.Stdout = io.Discard
	check.Stderr = io.Discard
	return check.Run() == nil
}

// GitTopLevel returns the root of the enclosing git work tree, or "".
func GitTopLevel() string {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// EnsureGitRepo initializes a git repository in the working directory when it
// is not already inside one. Wave runs every step in a git worktree, so a
// repository with at least one commit is required. Uses git rev-parse to
// correctly detect parent repos.
func EnsureGitRepo(w io.Writer) error {
	if InsideGitRepo() {
		return nil
	}

	fmt.Fprintf(w, "  Initializing git repository (Wave runs each step in a git worktree)...\n")
	cmd := exec.Command("git", "init")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}
	return nil
}

// CreateInitialCommit creates an initial commit of the files init wrote if the
// repository has no commits yet, so worktree operations have a base commit.
// It never changes the user's git config: when no identity is configured the
// commit is authored as "wave <wave@localhost>" via one-off -c flags.
// Returns true when a commit was created.
func CreateInitialCommit(w io.Writer, paths []string) (bool, error) {
	check := exec.Command("git", "rev-parse", "HEAD")
	check.Stdout = io.Discard
	check.Stderr = io.Discard
	if check.Run() == nil {
		return false, nil // commits already exist
	}

	fmt.Fprintf(w, "  Creating initial commit (the repository had no commits)...\n")

	var existing []string
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			existing = append(existing, p)
		}
	}

	add := exec.Command("git", append([]string{"add", "--"}, existing...)...)
	add.Stdout = io.Discard
	add.Stderr = io.Discard
	if err := add.Run(); err != nil {
		return false, fmt.Errorf("failed to stage wave files: %w", err)
	}

	args := []string{"-c", "commit.gpgsign=false"}
	for _, kv := range [][2]string{{"user.name", "wave"}, {"user.email", "wave@localhost"}} {
		probe := exec.Command("git", "config", kv[0])
		probe.Stdout = io.Discard
		probe.Stderr = io.Discard
		if probe.Run() != nil {
			args = append(args, "-c", kv[0]+"="+kv[1])
		}
	}
	args = append(args, "commit", "-m", "chore: initialize wave project")
	commit := exec.Command("git", args...)
	commit.Stdout = io.Discard
	commit.Stderr = io.Discard
	if err := commit.Run(); err != nil {
		return false, fmt.Errorf("failed to create initial commit: %w", err)
	}
	return true, nil
}

// WriteAssetMap writes each (filename, content) pair under baseDir, creating
// parent directories as needed (used for prompts which may include subdirs).
func WriteAssetMap(baseDir string, assets map[string]string) error {
	for relPath, content := range assets {
		path := filepath.Join(baseDir, relPath)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", path, err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			absPath, _ := filepath.Abs(path)
			return fmt.Errorf("failed to write %s: %w", absPath, err)
		}
	}
	return nil
}

// CreateExamplePersonas writes embedded persona prompts under .agents/personas/.
func CreateExamplePersonas(personas map[string]string) error {
	return WriteAssetMap(filepath.Join(".agents", "personas"), personas)
}

// CreateExamplePipelines writes embedded pipeline YAML under .agents/pipelines/.
func CreateExamplePipelines(pipelines map[string]string) error {
	return WriteAssetMap(filepath.Join(".agents", "pipelines"), pipelines)
}

// CreateExampleContracts writes embedded JSON-schema contracts under .agents/contracts/.
func CreateExampleContracts(contracts map[string]string) error {
	return WriteAssetMap(filepath.Join(".agents", "contracts"), contracts)
}

// CreateExamplePrompts writes embedded prompt templates under .agents/prompts/.
func CreateExamplePrompts(prompts map[string]string) error {
	return WriteAssetMap(filepath.Join(".agents", "prompts"), prompts)
}

// RuntimeGitignoreEntries are the Wave runtime paths that must never be
// committed. The workspace entry is added separately from the configured
// workspace root.
var RuntimeGitignoreEntries = []string{
	".agents/state.db*",
	".agents/traces/",
	".agents/artifacts/",
	".agents/output/",
	".agents/retros/",
	".agents/logs/",
	".agents/chat/",
}

// EnsureGitignore appends any missing Wave runtime paths to dir/.gitignore.
// Returns true when the file was created or changed.
func EnsureGitignore(dir, workspace string) (bool, error) {
	path := filepath.Join(dir, ".gitignore")
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("failed to read %s: %w", path, err)
	}
	present := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		present[strings.TrimSpace(line)] = true
	}

	entries := []string{}
	if ws := strings.TrimPrefix(filepath.ToSlash(filepath.Clean(workspace)), "./"); ws != "" && ws != "." && !filepath.IsAbs(workspace) {
		entries = append(entries, ws+"/")
	}
	entries = append(entries, RuntimeGitignoreEntries...)

	var missing []string
	for _, e := range entries {
		if !present[e] && !present["/"+e] {
			missing = append(missing, e)
		}
	}
	if len(missing) == 0 {
		return false, nil
	}

	var b strings.Builder
	b.Write(data)
	if len(data) > 0 && !strings.HasSuffix(string(data), "\n") {
		b.WriteString("\n")
	}
	if len(data) > 0 {
		b.WriteString("\n")
	}
	b.WriteString("# Wave runtime state (added by wave init)\n")
	for _, e := range missing {
		b.WriteString(e + "\n")
	}
	if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
		return false, fmt.Errorf("failed to write %s: %w", path, err)
	}
	return true, nil
}

// CreateProjectInstructionFiles seeds dir/AGENTS.md with the project name,
// description and commands when it does not exist yet. Wave copies AGENTS.md
// into every agent workspace (adapters.*.project_files) and Codex reads it
// natively. For the claude adapter a CLAUDE.md that imports AGENTS.md is
// added when missing. Existing files are never touched. Returns the files
// that were created.
func CreateProjectInstructionFiles(dir, adapter string, meta ProjectMetadata, project map[string]interface{}) ([]string, error) {
	var created []string
	write := func(name, content string) error {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return nil
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", name, err)
		}
		created = append(created, name)
		return nil
	}

	if err := write("AGENTS.md", agentsMarkdown(meta, project)); err != nil {
		return created, err
	}
	if adapter == "claude" {
		if err := write("CLAUDE.md", "@AGENTS.md\n"); err != nil {
			return created, err
		}
	}
	return created, nil
}

func agentsMarkdown(meta ProjectMetadata, project map[string]interface{}) string {
	var b strings.Builder
	name := meta.Name
	if name == "" {
		name = "Project"
	}
	fmt.Fprintf(&b, "# %s\n\n", name)
	if meta.Description != "" {
		fmt.Fprintf(&b, "%s\n\n", meta.Description)
	}
	b.WriteString("## Commands\n\n")
	hasCmd := false
	for _, kv := range [][2]string{{"Test", "test_command"}, {"Lint", "lint_command"}, {"Build", "build_command"}, {"Format", "format_command"}} {
		if v, _ := project[kv[1]].(string); v != "" {
			fmt.Fprintf(&b, "- %s: `%s`\n", kv[0], v)
			hasCmd = true
		}
	}
	if !hasCmd {
		b.WriteString("- Not detected yet: add the test, lint and build commands here and in the `project:` block of wave.yaml.\n")
	}
	b.WriteString("\n## Guidelines\n\n")
	b.WriteString("- Add coding conventions, architecture notes and review rules here. Wave copies this file into every agent workspace.\n")
	return b.String()
}
