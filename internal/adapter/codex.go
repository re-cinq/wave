package adapter

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// CodexAdapter runs the OpenAI Codex CLI (codex) as a subprocess.
type CodexAdapter struct {
	codexPath string
}

// NewCodexAdapter creates a CodexAdapter, locating the codex binary on PATH.
func NewCodexAdapter() *CodexAdapter {
	path := "codex"
	if p, err := exec.LookPath("codex"); err == nil {
		path = p
	}
	return &CodexAdapter{codexPath: path}
}

func (a *CodexAdapter) Run(ctx context.Context, cfg AdapterRunConfig) (*AdapterResult, error) {
	var cancel context.CancelFunc
	if cfg.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	workspacePath := cfg.WorkspacePath
	if workspacePath == "" {
		return nil, fmt.Errorf("WorkspacePath is required — refusing to use project root as workspace")
	}

	// Provision skills into .agents/skills/ — codex natively scans this path.
	if err := ProvisionSkills(workspacePath, ".agents/skills", cfg.ResolvedSkills); err != nil {
		return nil, fmt.Errorf("provision codex skills: %w", err)
	}

	if err := a.prepareWorkspace(workspacePath, cfg); err != nil {
		return nil, fmt.Errorf("failed to prepare codex workspace: %w", err)
	}

	args := a.buildArgs(cfg)
	cmd := exec.CommandContext(ctx, a.codexPath, args...)
	cmd.Dir = workspacePath
	cmd.Env = BuildCuratedEnvironment(cfg)
	// Prompt goes via stdin (`codex exec -`), same reason as the claude
	// adapter: large prompts overflow ARG_MAX as argv.
	cmd.Stdin = strings.NewReader(cfg.Prompt)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: 0}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start codex: %w", err)
	}

	var stdoutBuf bytes.Buffer
	stdoutDone := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
		for scanner.Scan() {
			line := scanner.Bytes()
			stdoutBuf.Write(line)
			stdoutBuf.WriteByte('\n')

			if cfg.OnStreamEvent != nil {
				if evt, ok := parseCodexStreamLine(line); ok {
					cfg.OnStreamEvent(evt)
				}
			}
		}
		stdoutDone <- scanner.Err()
	}()

	select {
	case <-ctx.Done():
		if cmd.Process != nil {
			killProcessGroup(cmd.Process, cfg.ProcessGrace)
		}
		_ = cmd.Wait()
		return nil, ctx.Err()
	case err := <-stdoutDone:
		if err != nil {
			return nil, fmt.Errorf("failed to read codex stdout: %w", err)
		}
	}

	cmdErr := cmd.Wait()
	result := a.parseOutput(stdoutBuf.String())
	if cmdErr != nil {
		result.ExitCode = exitCodeFromError(cmdErr)
		if result.FailureReason == "" {
			result.FailureReason = classifyCodexFailure(result.ExitCode)
		}
	}
	result.Stdout = bytes.NewReader(stdoutBuf.Bytes())

	return result, nil
}

func (a *CodexAdapter) prepareWorkspace(workspacePath string, cfg AdapterRunConfig) error {
	// Write system prompt as AGENTS.md for Codex
	if cfg.SystemPrompt != "" {
		promptPath := filepath.Join(workspacePath, "AGENTS.md")
		if err := os.WriteFile(promptPath, []byte(cfg.SystemPrompt), 0644); err != nil {
			return fmt.Errorf("failed to write AGENTS.md: %w", err)
		}
	}
	return nil
}

// buildArgs builds a non-interactive `codex exec` invocation that streams
// JSONL events on stdout. The bare `codex [PROMPT]` form starts the TUI and
// cannot run headless.
//
// Approvals and the codex sandbox are bypassed the same way the claude
// adapter passes --dangerously-skip-permissions: Wave owns isolation
// (ephemeral worktree, curated env, outer sandbox), and the codex sandbox
// would block git writes to the worktree's common dir and network access
// needed by forge personas.
func (a *CodexAdapter) buildArgs(cfg AdapterRunConfig) []string {
	args := []string{
		"exec",
		"--json",
		"--skip-git-repo-check",
		"--ephemeral",
		"--color", "never",
		"--dangerously-bypass-approvals-and-sandbox",
	}

	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
	}

	// Read the prompt from stdin.
	args = append(args, "-")

	return args
}

// codexEvent is one JSONL line emitted by `codex exec --json`.
type codexEvent struct {
	Type string `json:"type"`
	Item struct {
		Type             string `json:"type"`
		Text             string `json:"text"`
		Command          string `json:"command"`
		AggregatedOutput string `json:"aggregated_output"`
		Status           string `json:"status"`
	} `json:"item"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
	Message string `json:"message"`
}

func decodeCodexEvent(line []byte) (codexEvent, bool) {
	var evt codexEvent
	line = bytes.TrimSpace(line)
	if len(line) == 0 || line[0] != '{' {
		return evt, false
	}
	if err := json.Unmarshal(line, &evt); err != nil || evt.Type == "" {
		return evt, false
	}
	return evt, true
}

// parseOutput folds the JSONL event stream into an AdapterResult: token
// usage is summed over turn.completed events, the last agent_message is the
// result content, and turn.failed / error events become the failure reason.
func (a *CodexAdapter) parseOutput(output string) *AdapterResult {
	result := &AdapterResult{}
	var failure string

	for _, line := range bytes.Split([]byte(output), []byte("\n")) {
		evt, ok := decodeCodexEvent(line)
		if !ok {
			continue
		}
		switch evt.Type {
		case "item.completed":
			if evt.Item.Type == "agent_message" && evt.Item.Text != "" {
				result.ResultContent = evt.Item.Text
			}
		case "turn.completed":
			result.TokensIn += evt.Usage.InputTokens
			result.TokensOut += evt.Usage.OutputTokens
		case "turn.failed":
			failure = evt.Error.Message
		case "error":
			if evt.Message != "" {
				failure = evt.Message
			}
		}
	}
	result.TokensUsed = result.TokensIn + result.TokensOut

	if failure != "" {
		result.FailureReason = ClassifyFailure("", failure, nil)
		if result.ResultContent == "" {
			result.ResultContent = failure
		}
	}

	return result
}

// parseCodexStreamLine maps a single `codex exec --json` line to a StreamEvent.
func parseCodexStreamLine(line []byte) (StreamEvent, bool) {
	evt, ok := decodeCodexEvent(line)
	if !ok {
		return StreamEvent{}, false
	}

	switch evt.Type {
	case "item.started":
		if evt.Item.Type == "command_execution" {
			return StreamEvent{Type: "tool_use", ToolName: "Bash", ToolInput: truncate(evt.Item.Command, 100)}, true
		}
	case "item.completed":
		switch evt.Item.Type {
		case "agent_message":
			if evt.Item.Text != "" {
				return StreamEvent{Type: "text", Content: truncate(evt.Item.Text, 200)}, true
			}
		case "command_execution":
			return StreamEvent{Type: "tool_result", ToolName: "Bash", Content: truncate(evt.Item.AggregatedOutput, 200)}, true
		}
	case "turn.completed":
		return StreamEvent{
			Type:      "result",
			Subtype:   "success",
			TokensIn:  evt.Usage.InputTokens,
			TokensOut: evt.Usage.OutputTokens,
		}, true
	case "turn.failed":
		return StreamEvent{Type: "result", Subtype: "error_during_execution", Content: evt.Error.Message}, true
	}

	return StreamEvent{}, false
}

// classifyCodexFailure maps Codex exit codes to failure reasons.
func classifyCodexFailure(exitCode int) string {
	switch exitCode {
	case 124, 137:
		return "timeout"
	case 1:
		return "general_error"
	default:
		return "general_error"
	}
}

// truncate shortens s to at most n bytes for stream event summaries.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
