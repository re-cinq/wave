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

// GeminiAdapter runs the Google Gemini CLI as a subprocess.
type GeminiAdapter struct {
	geminiPath string
}

// NewGeminiAdapter creates a GeminiAdapter, locating the gemini binary on PATH.
func NewGeminiAdapter() *GeminiAdapter {
	path := "gemini"
	if p, err := exec.LookPath("gemini"); err == nil {
		path = p
	}
	return &GeminiAdapter{geminiPath: path}
}

func (a *GeminiAdapter) Run(ctx context.Context, cfg AdapterRunConfig) (*AdapterResult, error) {
	var cancel context.CancelFunc
	if cfg.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	workspacePath := cfg.WorkspacePath
	if workspacePath == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		workspacePath = wd
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
		return nil, fmt.Errorf("failed to prepare gemini workspace: %w", err)
	}

	args := a.buildArgs(cfg)
	cmd := exec.CommandContext(ctx, a.geminiPath, args...)
	cmd.Dir = workspacePath

	if cfg.Debug {
		fmt.Printf("[DEBUG] Gemini command: %s %s\n", a.geminiPath, shelljoinArgs(args))
		fmt.Printf("[DEBUG] Working directory: %s\n", workspacePath)
	}

	cmd.Env = BuildCuratedEnvironment(cfg)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: 0}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start gemini: %w", err)
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
				if evt, ok := parseGeminiStreamLine(line); ok {
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
			return nil, fmt.Errorf("failed to read gemini stdout: %w", err)
		}
	}

	cmdErr := cmd.Wait()
	result := a.parseOutput(stdoutBuf.String())
	if cmdErr != nil {
		result.ExitCode = exitCodeFromError(cmdErr)
		if result.FailureReason == "" {
			result.FailureReason = classifyGeminiFailure(result.ExitCode)
		}
	}
	result.Stdout = bytes.NewReader(stdoutBuf.Bytes())

	return result, nil
}

func (a *GeminiAdapter) prepareWorkspace(workspacePath string, cfg AdapterRunConfig) error {
	// Write system prompt as GEMINI.md for context
	if cfg.SystemPrompt != "" {
		promptPath := filepath.Join(workspacePath, "GEMINI.md")
		if err := os.WriteFile(promptPath, []byte(cfg.SystemPrompt), 0644); err != nil {
			return fmt.Errorf("failed to write GEMINI.md: %w", err)
		}
	}
	return nil
}

func (a *GeminiAdapter) buildArgs(cfg AdapterRunConfig) []string {
	args := []string{}

	if cfg.Model != "" && cfg.Model != "default" {
		args = append(args, "--model", cfg.Model)
	}

	args = append(args, "--yolo")
	args = append(args, "--output-format", "stream-json")

	if cfg.Prompt != "" {
		args = append(args, "-p", cfg.Prompt)
	}

	return args
}

func (a *GeminiAdapter) parseOutput(output string) *AdapterResult {
	result := &AdapterResult{}

	lines := bytes.Split([]byte(output), []byte("\n"))
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(line, &obj); err != nil {
			// Gemini may output plain text — capture it as result content
			if result.ResultContent == "" {
				result.ResultContent = string(line)
			}
			continue
		}

		var eventType string
		if raw, ok := obj["type"]; ok {
			_ = json.Unmarshal(raw, &eventType)
		}

		if eventType == "result" {
			var resultEvt struct {
				Usage struct {
					InputTokens  int `json:"input_tokens"`
					OutputTokens int `json:"output_tokens"`
				} `json:"usage"`
				Content string `json:"content"`
			}
			if err := json.Unmarshal(line, &resultEvt); err == nil {
				result.TokensIn = resultEvt.Usage.InputTokens
				result.TokensOut = resultEvt.Usage.OutputTokens
				result.TokensUsed = result.TokensIn + result.TokensOut
				if resultEvt.Content != "" {
					result.ResultContent = resultEvt.Content
				}

				var errEvt struct {
					Status string `json:"status"`
					Error  struct {
						Type    string `json:"type"`
						Message string `json:"message"`
					} `json:"error"`
				}
				if err := json.Unmarshal(line, &errEvt); err == nil {
					if errEvt.Status == "error" && errEvt.Error.Message != "" {
						result.FailureReason = "adapter error: " + errEvt.Error.Message
					}
				}
			}
		}
	}

	return result
}

// parseGeminiStreamLine parses a single line from Gemini CLI output.
func parseGeminiStreamLine(line []byte) (StreamEvent, bool) {
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
	case "tool_use":
		var tool struct {
			Name  string `json:"name"`
			Input string `json:"input"`
		}
		if err := json.Unmarshal(line, &tool); err == nil && tool.Name != "" {
			input := tool.Input
			if len(input) > 100 {
				input = input[:100]
			}
			return StreamEvent{Type: "tool_use", ToolName: tool.Name, ToolInput: input}, true
		}
	case "text":
		var text struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal(line, &text); err == nil && text.Content != "" {
			content := text.Content
			if len(content) > 200 {
				content = content[:200]
			}
			return StreamEvent{Type: "text", Content: content}, true
		}
	case "result":
		var errEvt struct {
			Status string `json:"status"`
			Error  struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(line, &errEvt); err == nil && errEvt.Status == "error" && errEvt.Error.Message != "" {
			return StreamEvent{Type: "system", Content: errEvt.Error.Message}, true
		}
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

// classifyGeminiFailure maps Gemini exit codes to failure reasons.
func classifyGeminiFailure(exitCode int) string {
	switch exitCode {
	case 124, 137:
		return "timeout"
	case 1:
		return "general_error"
	default:
		return "general_error"
	}
}
