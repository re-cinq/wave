package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type OpenCodeAdapter struct {
	opencodePath string
}

func NewOpenCodeAdapter() *OpenCodeAdapter {
	path := "opencode"
	if p, err := exec.LookPath("opencode"); err == nil {
		path = p
	}
	return &OpenCodeAdapter{opencodePath: path}
}

func (a *OpenCodeAdapter) Run(ctx context.Context, cfg AdapterRunConfig) (*AdapterResult, error) {
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

	return runSubprocess(ctx, subprocessConfig{
		BinaryPath:   a.opencodePath,
		BinaryLabel:  "opencode",
		Args:         a.buildArgs(cfg),
		WorkDir:      workspacePath,
		Env:          BuildCuratedEnvironment(cfg),
		ProcessGrace: cfg.ProcessGrace,
		ParseLine:    parseOpenCodeStreamLine,
		OnEvent:      cfg.OnStreamEvent,
	})
}

// parseOpenCodeStreamLine parses a single NDJSON line from opencode's JSON output
// and converts it to a StreamEvent. Returns (event, true) if the line produced a
// meaningful event, or (zero, false) if it should be skipped (malformed or unrecognised).
//
// OpenCode event format mapping (--output-format json):
//
//	{"type":"assistant","message":{"content":[{"type":"text","text":"..."}]}}
//	  → StreamEvent{Type:"text", Content:"..."}
//
//	{"type":"tool","tool":"Read","input":{"file_path":"..."}}
//	  → StreamEvent{Type:"tool_use", ToolName:"Read", ToolInput:"..."}
//
//	{"type":"result","usage":{"input_tokens":N,"output_tokens":M},"content":"..."}
//	  → StreamEvent{Type:"result", TokensIn:N, TokensOut:M, Content:"..."}
//
//	{"type":"system","message":"..."}
//	  → StreamEvent{Type:"system"}
//
// Unrecognised or malformed lines are silently skipped.
func parseOpenCodeStreamLine(line []byte) (StreamEvent, bool) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return StreamEvent{}, false
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(line, &obj); err != nil {
		// Malformed NDJSON — skip gracefully.
		return StreamEvent{}, false
	}

	var eventType string
	if raw, ok := obj["type"]; ok {
		if err := json.Unmarshal(raw, &eventType); err != nil {
			return StreamEvent{}, false
		}
	}

	switch eventType {
	case "system":
		return StreamEvent{Type: "system"}, true

	case "assistant":
		// OpenCode assistant events carry message content blocks.
		var msg struct {
			Message struct {
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text,omitempty"`
				} `json:"content"`
			} `json:"message"`
		}
		data, _ := json.Marshal(obj)
		if err := json.Unmarshal(data, &msg); err != nil {
			return StreamEvent{}, false
		}
		for _, block := range msg.Message.Content {
			if block.Type == "text" && block.Text != "" {
				text := block.Text
				if len(text) > 200 {
					text = text[:200]
				}
				return StreamEvent{Type: "text", Content: text}, true
			}
		}
		return StreamEvent{}, false

	case "tool":
		// OpenCode tool events carry the tool name and input fields.
		var toolEvt struct {
			Tool  string          `json:"tool"`
			Input json.RawMessage `json:"input"`
		}
		data, _ := json.Marshal(obj)
		if err := json.Unmarshal(data, &toolEvt); err != nil {
			return StreamEvent{}, false
		}
		if toolEvt.Tool == "" {
			return StreamEvent{}, false
		}
		target := extractToolTarget(toolEvt.Tool, toolEvt.Input)
		return StreamEvent{
			Type:      "tool_use",
			ToolName:  toolEvt.Tool,
			ToolInput: target,
		}, true

	case "result":
		// OpenCode result events carry cumulative usage and final content.
		var resultEvt struct {
			Usage struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
			Content string `json:"content"`
			Subtype string `json:"subtype"`
		}
		data, _ := json.Marshal(obj)
		if err := json.Unmarshal(data, &resultEvt); err != nil {
			return StreamEvent{}, false
		}
		return StreamEvent{
			Type:      "result",
			TokensIn:  resultEvt.Usage.InputTokens,
			TokensOut: resultEvt.Usage.OutputTokens,
			Content:   resultEvt.Content,
			Subtype:   resultEvt.Subtype,
		}, true

	default:
		return StreamEvent{}, false
	}
}

func (a *OpenCodeAdapter) prepareWorkspace(workspacePath string, cfg AdapterRunConfig) error {
	// OpenCode uses .opencode/ directory for configuration
	settingsDir := filepath.Join(workspacePath, ".opencode")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return fmt.Errorf("failed to create .opencode directory: %w", err)
	}

	pm := ParseProviderModel(cfg.Model)
	config := map[string]interface{}{
		"provider":    pm.Provider,
		"model":       pm.Model,
		"temperature": cfg.Temperature,
	}

	configData, _ := json.MarshalIndent(config, "", "  ")
	configPath := filepath.Join(settingsDir, "config.json")
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write config.json: %w", err)
	}

	// Build instruction content: system prompt + restriction section
	content := cfg.SystemPrompt
	if content == "" {
		personaPath := filepath.Join(".wave", "personas", cfg.Persona+".md")
		if data, err := os.ReadFile(personaPath); err == nil {
			content = string(data)
		}
	}
	restrictions := buildRestrictionSection(cfg)
	if restrictions != "" {
		content += restrictions
	}
	if content != "" {
		promptPath := filepath.Join(workspacePath, "AGENTS.md")
		if err := os.WriteFile(promptPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write AGENTS.md: %w", err)
		}
	}

	return nil
}

func (a *OpenCodeAdapter) buildArgs(cfg AdapterRunConfig) []string {
	args := []string{}

	if cfg.Prompt != "" {
		args = append(args, "--prompt", cfg.Prompt)
	}

	args = append(args, "--output-format", "json")
	args = append(args, "--non-interactive")

	return args
}

func ResolveAdapter(adapterName string) AdapterRunner {
	switch strings.ToLower(adapterName) {
	case "claude":
		return NewClaudeAdapter()
	case "opencode":
		return NewOpenCodeAdapter()
	case "codex":
		return NewCodexAdapter()
	case "gemini":
		return NewGeminiAdapter()
	case "browser":
		return NewBrowserAdapter()
	default:
		return NewProcessGroupRunner()
	}
}
