package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/recinq/wave/internal/sandbox"
	"github.com/recinq/wave/internal/timeouts"
)

type AdapterRunner interface {
	Run(ctx context.Context, cfg AdapterRunConfig) (*AdapterResult, error)
}

// StreamEvent represents a real-time event from Claude Code's stream-json output.
type StreamEvent struct {
	Type      string // "tool_use", "tool_result", "text", "result", "system"
	ToolName  string // e.g. "Read", "Write", "Bash", "Glob", "Grep"
	ToolInput string // summary of input (file path, command, pattern)
	Content   string // text content or result summary
	TokensIn  int    // cumulative input tokens
	TokensOut int    // cumulative output tokens
	Subtype   string // result event subtype: "success", "error_max_turns", "error_during_execution"
}

// SkillRef holds metadata about a resolved skill for CLAUDE.md injection.
type SkillRef struct {
	Name        string
	Description string
	SourcePath  string // absolute path to skill source directory; used by claude adapter to provision .claude/skills/
}

type AdapterRunConfig struct {
	Adapter       string
	Persona       string
	WorkspacePath string
	Prompt        string
	SystemPrompt  string
	Timeout       time.Duration // 0 means no timeout; callers set per use case
	ProcessGrace  time.Duration // SIGTERM→SIGKILL grace period; 0 uses timeouts.ProcessGrace
	StdoutDrain   time.Duration // post-cancel stdout drain wait; 0 uses timeouts.StdoutDrain
	Env           []string
	Temperature   float64
	AllowedTools  []string
	DenyTools     []string
	OutputFormat  string
	Debug         bool
	Model         string // Model to use; tier names (cheapest, balanced, strongest) or literal IDs (e.g., "claude-opus-4-5-20251101")

	// Sandbox configuration derived from manifest
	SandboxEnabled bool     // Master switch from runtime.sandbox.enabled
	AllowedDomains []string // Network domain allowlist
	EnvPassthrough []string // Env var names to pass through from host
	SandboxBackend string   // Sandbox backend type: "docker", "bubblewrap", "none"
	DockerImage    string   // Docker image when SandboxBackend == "docker"

	// Skill provisioning
	SkillCommandsDir string     // Source directory containing skill command files to copy into workspace
	ResolvedSkills   []SkillRef // Skills resolved from hierarchical config, provisioned into workspace

	// Maximum concurrent sub-agents the persona may spawn (0 or 1 = no hint).
	MaxConcurrentAgents int

	// OnStreamEvent is called for each real-time event during Claude Code execution.
	// If nil, streaming events are silently ignored.
	OnStreamEvent func(StreamEvent)
}

type AdapterResult struct {
	ExitCode      int
	Stdout        io.Reader
	TokensUsed    int
	TokensIn      int // Input tokens (prompt + cache creation)
	TokensOut     int // Output tokens (completion)
	Artifacts     []string
	ResultContent string // Extracted content from the adapter response
	FailureReason string // Classification: "timeout", "context_exhaustion", "general_error"
	Subtype       string // Result event subtype from Claude Code NDJSON
}

type ProcessGroupRunner struct{}

func NewProcessGroupRunner() *ProcessGroupRunner {
	return &ProcessGroupRunner{}
}

func (r *ProcessGroupRunner) Run(ctx context.Context, cfg AdapterRunConfig) (*AdapterResult, error) {
	var cancel context.CancelFunc
	if cfg.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	args := strings.Fields(cfg.Prompt)
	cmd := exec.CommandContext(ctx, cfg.Adapter, args...)
	cmd.Dir = cfg.WorkspacePath

	// Apply sandbox wrapping if configured
	if cfg.SandboxBackend == "docker" {
		sb, err := sandbox.NewSandbox(sandbox.SandboxBackendDocker)
		if err != nil {
			return nil, fmt.Errorf("failed to create docker sandbox: %w", err)
		}
		defer func() { _ = sb.Cleanup(ctx) }()

		sandboxCfg := sandbox.Config{
			Backend:        sandbox.SandboxBackendDocker,
			DockerImage:    cfg.DockerImage,
			AllowedDomains: cfg.AllowedDomains,
			EnvPassthrough: cfg.EnvPassthrough,
			WorkspacePath:  cfg.WorkspacePath,
			Debug:          cfg.Debug,
		}
		cmd, err = sb.Wrap(ctx, cmd, sandboxCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to wrap command in docker sandbox: %w", err)
		}
	}

	mergedEnv := append(os.Environ(), cfg.Env...)
	cmd.Env = mergedEnv

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	var stdoutBuf bytes.Buffer
	copyDone := make(chan error, 1)
	go func() {
		_, err := io.Copy(&stdoutBuf, stdoutPipe)
		copyDone <- err
	}()

	select {
	case <-ctx.Done():
		killProcessGroup(cmd.Process, cfg.ProcessGrace)
		_ = cmd.Wait()
		return nil, ctx.Err()
	case err := <-copyDone:
		if err != nil {
			return nil, fmt.Errorf("failed to read stdout: %w", err)
		}
		if err := cmd.Wait(); err != nil {
			return &AdapterResult{
				ExitCode:   exitCodeFromError(err),
				Stdout:     bytes.NewReader(stdoutBuf.Bytes()),
				TokensUsed: 0,
				Artifacts:  nil,
			}, nil
		}
	}

	var result AdapterResult
	result.ExitCode = 0
	result.Stdout = bytes.NewReader(stdoutBuf.Bytes())
	result.TokensUsed = estimateTokens(stdoutBuf.String())

	parseArtifacts(stdoutBuf.Bytes(), &result.Artifacts)

	return &result, nil
}

// killProcessGroup sends SIGTERM to the process group, then SIGKILL after the
// grace period if the process hasn't exited. It does NOT call process.Wait() —
// callers must call cmd.Wait() themselves to avoid "wait: no child processes"
// errors from double-waiting.
func killProcessGroup(process *os.Process, grace time.Duration) {
	if grace <= 0 {
		grace = timeouts.ProcessGrace
	}
	// Send SIGTERM first for graceful shutdown
	_ = syscall.Kill(-process.Pid, syscall.SIGTERM)

	// Schedule a forced kill after the grace period. The caller's cmd.Wait()
	// will reap the process; if it hasn't exited by then, SIGKILL finishes it.
	go func() {
		time.Sleep(grace)
		// SIGKILL is harmless if the process already exited
		_ = syscall.Kill(-process.Pid, syscall.SIGKILL)
	}()
}

func exitCodeFromError(err error) int {
	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}
	return -1
}

func estimateTokens(text string) int {
	return len(text) / 4
}

func InstructionFilename(adapterName string) string {
	switch strings.ToLower(adapterName) {
	case "claude":
		return "CLAUDE.md"
	case "gemini":
		return "GEMINI.md"
	case "opencode", "codex":
		return "AGENTS.md"
	default:
		return "AGENTS.md"
	}
}

func parseArtifacts(data []byte, artifacts *[]string) {
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return
	}
	if artifactList, ok := parsed["artifacts"].([]interface{}); ok {
		for _, a := range artifactList {
			if s, ok := a.(string); ok {
				*artifacts = append(*artifacts, s)
			}
		}
	}
}
