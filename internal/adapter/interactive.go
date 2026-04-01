package adapter

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// InteractiveOptions configures an interactive Claude Code session.
type InteractiveOptions struct {
	Adapter      string   // Adapter binary name (e.g., "claude", "opencode", "gemini")
	Model        string   // Model to use (e.g., "sonnet", "opus")
	AllowedTools []string // Tools to allow (for --allowedTools)
	AddDirs      []string // Additional directories to include (--add-dir)
	SystemPrompt string   // Optional system prompt (--system-prompt)
	Resume       string   // Session ID to resume (--resume)
	Prompt       string   // Initial prompt to send (first positional arg)
}

// LaunchInteractive spawns Claude Code in interactive (non -p) mode.
// It passes through stdin/stdout/stderr so the user interacts directly with Claude.
// workspacePath is used as the working directory; it should contain CLAUDE.md and .claude/settings.json.
// Returns the session ID captured from stderr (empty string if not found) and any error.
func LaunchInteractive(workspacePath string, opts InteractiveOptions) (string, error) {
	adapterName := opts.Adapter
	if adapterName == "" {
		adapterName = "claude"
	}
	binaryPath, err := exec.LookPath(adapterName)
	if err != nil {
		return "", fmt.Errorf("adapter %q not found: %w", adapterName, err)
	}

	args := buildInteractiveArgs(opts)

	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = workspacePath
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	// Tee stderr to both os.Stderr (for user display) and a buffer (for session ID capture)
	var stderrBuf bytes.Buffer
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

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
				sessionID := extractSessionID(stderrBuf.String())
				return sessionID, nil
			}
		}
		return "", fmt.Errorf("adapter %q exited with error: %w", adapterName, err)
	}

	sessionID := extractSessionID(stderrBuf.String())
	return sessionID, nil
}

// sessionIDPattern matches session ID output from Claude Code.
// Looks for "session" followed by a colon or whitespace, then a hex/UUID string.
var sessionIDPattern = regexp.MustCompile(`(?i)session[:\s]+([a-f0-9-]{8,})`)

// extractSessionID parses Claude Code output for a session ID.
// Returns the session ID if found, or an empty string if not.
func extractSessionID(output string) string {
	matches := sessionIDPattern.FindStringSubmatch(output)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
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

	if opts.Resume != "" {
		args = append(args, "--resume", opts.Resume)
	}

	if opts.Prompt != "" {
		args = append(args, opts.Prompt)
	}

	return args
}
