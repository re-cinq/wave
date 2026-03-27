package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// CodexAdapter wraps the OpenAI Codex CLI for subprocess execution.
type CodexAdapter struct {
	codexPath string
}

// NewCodexAdapter creates a CodexAdapter, locating the codex binary in PATH.
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

	workspacePath, err := resolveWorkspacePath(cfg.WorkspacePath)
	if err != nil {
		return nil, err
	}

	if err := a.prepareWorkspace(workspacePath, cfg); err != nil {
		return nil, fmt.Errorf("failed to prepare workspace: %w", err)
	}

	result, err := runSubprocess(ctx, subprocessConfig{
		BinaryPath:   a.codexPath,
		BinaryLabel:  "codex",
		Args:         a.buildArgs(cfg),
		WorkDir:      workspacePath,
		Env:          BuildCuratedEnvironment(cfg),
		ProcessGrace: cfg.ProcessGrace,
		ParseLine:    parseCodexStreamLine,
		OnEvent:      cfg.OnStreamEvent,
	})
	if err != nil {
		return nil, err
	}

	// Classify failure reason from exit code and output content
	stdoutContent := ""
	if result.Stdout != nil {
		data, _ := readReaderContent(result.Stdout)
		stdoutContent = string(data)
		result.Stdout = bytes.NewReader(data)
	}
	result.FailureReason = ClassifyFailure("", stdoutContent, ctx.Err())

	return result, nil
}

func (a *CodexAdapter) prepareWorkspace(workspacePath string, cfg AdapterRunConfig) error {
	// Build the instruction content: system prompt + restriction section
	content := cfg.SystemPrompt
	restrictions := buildRestrictionSection(cfg)
	if restrictions != "" {
		content += restrictions
	}

	// Write system prompt as AGENTS.md for Codex
	if content != "" {
		promptPath := filepath.Join(workspacePath, "AGENTS.md")
		if err := os.WriteFile(promptPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write AGENTS.md: %w", err)
		}
	}
	return nil
}

func (a *CodexAdapter) buildArgs(cfg AdapterRunConfig) []string {
	var args []string

	if cfg.Prompt != "" {
		args = append(args, cfg.Prompt)
	}

	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
	}

	// Codex uses --quiet for non-interactive mode
	args = append(args, "--quiet")

	return args
}

// parseCodexStreamLine parses a single NDJSON line from Codex CLI output.
// Codex emits events similar to Claude Code. This parser handles common event
// types and degrades gracefully for unrecognised formats.
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
	case "message":
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

	case "function_call":
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
