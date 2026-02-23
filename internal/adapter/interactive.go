package adapter

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// InteractiveOptions configures an interactive Claude Code session.
type InteractiveOptions struct {
	Model        string   // Model to use (e.g., "sonnet", "opus")
	AllowedTools []string // Tools to allow (for --allowedTools)
	AddDirs      []string // Additional directories to include (--add-dir)
	SystemPrompt string   // Optional system prompt (--system-prompt)
}

// LaunchInteractive spawns Claude Code in interactive (non -p) mode.
// It passes through stdin/stdout/stderr so the user interacts directly with Claude.
// workspacePath is used as the working directory; it should contain CLAUDE.md and .claude/settings.json.
func LaunchInteractive(workspacePath string, opts InteractiveOptions) error {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude CLI not found: %w", err)
	}

	args := buildInteractiveArgs(opts)

	cmd := exec.Command(claudePath, args...)
	cmd.Dir = workspacePath
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set minimal environment
	cmd.Env = []string{
		"HOME=" + os.Getenv("HOME"),
		"PATH=" + os.Getenv("PATH"),
		"TERM=" + getenvDefault("TERM", "xterm-256color"),
		"TMPDIR=/tmp",
	}

	if err := cmd.Run(); err != nil {
		// Exit code 0 means normal exit, non-zero may be user Ctrl+C
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 130 {
				// SIGINT (Ctrl+C) — normal exit
				return nil
			}
		}
		return fmt.Errorf("claude exited with error: %w", err)
	}

	return nil
}

// buildInteractiveArgs constructs CLI arguments for interactive Claude Code.
func buildInteractiveArgs(opts InteractiveOptions) []string {
	var args []string

	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}

	if len(opts.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(opts.AllowedTools, ","))
	}

	// Skip permission prompts — Wave configures permissions via settings.json
	args = append(args, "--dangerously-skip-permissions")

	for _, dir := range opts.AddDirs {
		args = append(args, "--add-dir", dir)
	}

	if opts.SystemPrompt != "" {
		args = append(args, "--system-prompt", opts.SystemPrompt)
	}

	return args
}
