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

	// Warn about uninjected skills — non-claude adapters do not yet support
	// native skill provisioning. Skills declared in the pipeline manifest will
	// not reach the agent. See: https://github.com/re-cinq/wave/issues/1120
	if len(cfg.ResolvedSkills) > 0 {
		skillNames := make([]string, len(cfg.ResolvedSkills))
		for i, s := range cfg.ResolvedSkills {
			skillNames[i] = s.Name
		}
		fmt.Fprintf(os.Stderr, "[WARN] %s adapter: %d skill(s) declared but not injected (not yet supported): %v\n", cfg.Adapter, len(cfg.ResolvedSkills), skillNames)
	}

	if err := a.prepareWorkspace(workspacePath, cfg); err != nil {
		return nil, fmt.Errorf("failed to prepare codex workspace: %w", err)
	}

	args := a.buildArgs(cfg)
	cmd := exec.CommandContext(ctx, a.codexPath, args...)
	cmd.Dir = workspacePath
	cmd.Env = BuildCuratedEnvironment(cfg)
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

func (a *CodexAdapter) buildArgs(cfg AdapterRunConfig) []string {
	args := []string{"--full-auto"}

	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
	}

	if cfg.Prompt != "" {
		args = append(args, cfg.Prompt)
	}

	return args
}

func (a *CodexAdapter) parseOutput(output string) *AdapterResult {
	result := &AdapterResult{}

	lines := bytes.Split([]byte(output), []byte("\n"))
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(line, &obj); err != nil {
			continue
		}

		var eventType string
		if raw, ok := obj["type"]; ok {
			_ = json.Unmarshal(raw, &eventType)
		}

		if eventType == "result" || eventType == "message" {
			var resultEvt struct {
				Usage struct {
					InputTokens  int `json:"input_tokens"`
					OutputTokens int `json:"output_tokens"`
				} `json:"usage"`
				Content string `json:"content"`
			}
			if err := json.Unmarshal(line, &resultEvt); err == nil {
				if resultEvt.Usage.InputTokens > 0 {
					result.TokensIn = resultEvt.Usage.InputTokens
				}
				if resultEvt.Usage.OutputTokens > 0 {
					result.TokensOut = resultEvt.Usage.OutputTokens
				}
				result.TokensUsed = result.TokensIn + result.TokensOut
				if resultEvt.Content != "" {
					result.ResultContent = resultEvt.Content
				}
			}
		}
	}

	return result
}

// parseCodexStreamLine parses a single NDJSON line from Codex output.
func parseCodexStreamLine(line []byte) (StreamEvent, bool) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return StreamEvent{}, false
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(line, &obj); err != nil {
		return StreamEvent{}, false
	}

	var eventType string
	if raw, ok := obj["type"]; ok {
		if err := json.Unmarshal(raw, &eventType); err != nil {
			return StreamEvent{}, false
		}
	}

	switch eventType {
	case "function_call":
		var fc struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		}
		if err := json.Unmarshal(line, &fc); err == nil && fc.Name != "" {
			input := fc.Arguments
			if len(input) > 100 {
				input = input[:100]
			}
			return StreamEvent{Type: "tool_use", ToolName: fc.Name, ToolInput: input}, true
		}
	case "message":
		var msg struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal(line, &msg); err == nil && msg.Content != "" {
			text := msg.Content
			if len(text) > 200 {
				text = text[:200]
			}
			return StreamEvent{Type: "text", Content: text}, true
		}
	case "result":
		var res struct {
			Usage struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal(line, &res); err == nil {
			return StreamEvent{
				Type:      "result",
				TokensIn:  res.Usage.InputTokens,
				TokensOut: res.Usage.OutputTokens,
				Content:   res.Content,
			}, true
		}
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
