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
	return runSubprocess(ctx, a.geminiPath, args, workspacePath, cfg, parseGeminiStreamLine, a.parseOutput)
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

// parseOutput extracts result content, token usage, and failure metadata.
func (a *GeminiAdapter) parseOutput(data []byte) parseOutputResult {
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
		case "text":
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
