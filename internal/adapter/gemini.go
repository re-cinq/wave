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

// GeminiAdapter wraps the Google Gemini CLI for subprocess execution.
type GeminiAdapter struct {
	geminiPath string
}

// NewGeminiAdapter creates a GeminiAdapter, locating the gemini binary in PATH.
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

	if err := a.prepareWorkspace(workspacePath, cfg); err != nil {
		return nil, fmt.Errorf("failed to prepare workspace: %w", err)
	}

	args := a.buildArgs(cfg)
	cmd := exec.CommandContext(ctx, a.geminiPath, args...)
	cmd.Dir = workspacePath
	cmd.Env = BuildCuratedEnvironment(cfg)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

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
		cmd.Wait()
		return nil, ctx.Err()
	case err := <-stdoutDone:
		if err != nil {
			return nil, fmt.Errorf("failed to read stdout: %w", err)
		}
	}

	cmdErr := cmd.Wait()
	result := &AdapterResult{
		ExitCode: 0,
		Stdout:   bytes.NewReader(stdoutBuf.Bytes()),
	}
	if cmdErr != nil {
		result.ExitCode = exitCodeFromError(cmdErr)
	}
	result.TokensUsed = estimateTokens(stdoutBuf.String())
	return result, nil
}

func (a *GeminiAdapter) prepareWorkspace(workspacePath string, cfg AdapterRunConfig) error {
	// Write system prompt as GEMINI.md for Gemini CLI
	if cfg.SystemPrompt != "" {
		promptPath := filepath.Join(workspacePath, "GEMINI.md")
		if err := os.WriteFile(promptPath, []byte(cfg.SystemPrompt), 0644); err != nil {
			return fmt.Errorf("failed to write GEMINI.md: %w", err)
		}
	}
	return nil
}

func (a *GeminiAdapter) buildArgs(cfg AdapterRunConfig) []string {
	var args []string

	if cfg.Prompt != "" {
		args = append(args, "-p", cfg.Prompt)
	}

	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
	}

	return args
}

// parseGeminiStreamLine parses a single NDJSON line from Gemini CLI output.
// Gemini CLI may emit structured events. This parser handles common event
// types and degrades gracefully for unrecognised formats.
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
	case "text":
		var content string
		if raw, ok := obj["content"]; ok {
			json.Unmarshal(raw, &content)
		}
		if content != "" {
			if len(content) > 200 {
				content = content[:200]
			}
			return StreamEvent{Type: "text", Content: content}, true
		}
		return StreamEvent{}, false

	case "tool_use":
		var name string
		if raw, ok := obj["name"]; ok {
			json.Unmarshal(raw, &name)
		}
		if name != "" {
			return StreamEvent{Type: "tool_use", ToolName: name}, true
		}
		return StreamEvent{}, false

	default:
		return StreamEvent{}, false
	}
}
