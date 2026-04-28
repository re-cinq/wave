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

// WizardWaveDirs extends DefaultWaveDirs with the .claude/commands dir used by the
// interactive wizard for slash-command scaffolding.
var WizardWaveDirs = append(append([]string{}, DefaultWaveDirs...), ".claude/commands")

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

// EnsureGitRepo checks if the current directory is inside a git repository and
// initializes one if not. Uses git rev-parse to correctly detect parent repos.
func EnsureGitRepo(w io.Writer) error {
	check := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	check.Stdout = io.Discard
	check.Stderr = io.Discard
	if check.Run() == nil {
		return nil // already inside a git repo
	}

	fmt.Fprintf(w, "  Initializing git repository...\n")
	cmd := exec.Command("git", "init")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}
	return nil
}

// CreateInitialCommit creates an initial commit with wave files if no commits
// exist yet. This ensures worktree operations have at least one commit to work with.
func CreateInitialCommit(w io.Writer, outputPath string) error {
	check := exec.Command("git", "rev-parse", "HEAD")
	check.Stdout = io.Discard
	check.Stderr = io.Discard
	if check.Run() == nil {
		return nil // commits already exist
	}

	fmt.Fprintf(w, "  Creating initial commit...\n")

	for _, kv := range [][2]string{
		{"user.name", "wave"},
		{"user.email", "wave@localhost"},
	} {
		check := exec.Command("git", "config", kv[0])
		check.Stdout = io.Discard
		check.Stderr = io.Discard
		if check.Run() != nil {
			cfg := exec.Command("git", "config", kv[0], kv[1])
			cfg.Stdout = io.Discard
			cfg.Stderr = io.Discard
			_ = cfg.Run()
		}
	}

	add := exec.Command("git", "add", outputPath, ".agents/")
	add.Stdout = io.Discard
	add.Stderr = io.Discard
	if err := add.Run(); err != nil {
		return fmt.Errorf("failed to stage wave files: %w", err)
	}

	commit := exec.Command("git", "-c", "commit.gpgsign=false", "commit", "-m", "chore: initialize wave project")
	commit.Stdout = io.Discard
	commit.Stderr = io.Discard
	if err := commit.Run(); err != nil {
		return fmt.Errorf("failed to create initial commit: %w", err)
	}
	return nil
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

// CreateProjectInstructionFiles seeds AGENTS.md and the per-adapter alias files
// at the project root if they don't yet exist.
func CreateProjectInstructionFiles() error {
	files := map[string]string{
		"AGENTS.md": "See CLAUDE.md for project guidelines.",
		"CLAUDE.md": "See AGENTS.md for project guidelines.",
		"GEMINI.md": "See AGENTS.md for project guidelines.",
		"CODEX.md":  "See AGENTS.md for project guidelines.",
	}
	for filename, content := range files {
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", filename, err)
			}
		}
	}
	return nil
}

// RemoveDeselectedPipelines deletes pipeline YAML files in pipelinesDir whose
// stem (filename without .yaml extension) is not in the selected list.
func RemoveDeselectedPipelines(pipelinesDir string, selected []string) error {
	keep := make(map[string]bool)
	for _, name := range selected {
		keep[name] = true
	}
	entries, err := os.ReadDir(pipelinesDir)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".yaml")
		if name == e.Name() {
			continue
		}
		if !keep[name] {
			_ = os.Remove(filepath.Join(pipelinesDir, e.Name()))
		}
	}
	return nil
}
