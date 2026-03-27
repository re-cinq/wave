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
	return runSubprocess(ctx, a.codexPath, args, workspacePath, cfg, parseCodexStreamLine, a.parseOutput)
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

	case "result":
		var usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		}
		if raw, ok := obj["usage"]; ok {
			json.Unmarshal(raw, &usage)
		}
		var subtype string
		if raw, ok := obj["subtype"]; ok {
			json.Unmarshal(raw, &subtype)
		}
		return StreamEvent{
			Type:      "result",
			TokensIn:  usage.InputTokens,
			TokensOut: usage.OutputTokens,
			Subtype:   subtype,
		}, true

	default:
		return StreamEvent{}, false
	}
}

// parseOutput extracts result content, token usage, and failure metadata from
// the complete NDJSON output stream.
func (a *CodexAdapter) parseOutput(data []byte) parseOutputResult {
	var tokens int
	var resultContent string
	var subtype string

	lines := bytes.Split(data, []byte("\n"))
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		var obj struct {
			Type    string `json:"type"`
			Subtype string `json:"subtype"`
			Content string `json:"content"`
			Result  string `json:"result"`
			Usage   struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(line, &obj); err != nil {
			continue
		}

		switch obj.Type {
		case "result":
			tokens = obj.Usage.InputTokens + obj.Usage.OutputTokens
			subtype = obj.Subtype
			resultContent = obj.Result
			if resultContent == "" {
				resultContent = obj.Content
			}
		case "message":
			if resultContent == "" && obj.Content != "" {
				resultContent = obj.Content
			}
		}
	}

	if tokens == 0 {
		tokens = len(data) / 4
	}

	return parseOutputResult{
		Tokens:        tokens,
		ResultContent: resultContent,
		Subtype:       subtype,
	}
}
