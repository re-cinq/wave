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
		return nil, fmt.Errorf("failed to prepare workspace: %w", err)
	}

	args := a.buildArgs(cfg)
	return runSubprocess(ctx, a.opencodePath, args, workspacePath, cfg, parseOpenCodeStreamLine, a.parseOutput)
}

func (a *OpenCodeAdapter) parseOutput(data []byte) parseOutputResult {
	var tokens int
	var resultContent string
	var subtype string

	lines := bytes.Split(data, []byte("\n"))
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

		if eventType == "step_finish" {
			var evt struct {
				Part struct {
					Tokens struct {
						Total  int `json:"total"`
						Input  int `json:"input"`
						Output int `json:"output"`
					} `json:"tokens"`
				} `json:"part"`
			}
			if err := json.Unmarshal(line, &evt); err == nil {
				tokens = evt.Part.Tokens.Total
			}
		}

		if eventType == "result" {
			var evt struct {
				Subtype string `json:"subtype"`
				Result  string `json:"result"`
				Content string `json:"content"`
				Usage   struct {
					InputTokens  int `json:"input_tokens"`
					OutputTokens int `json:"output_tokens"`
				} `json:"usage"`
			}
			if err := json.Unmarshal(line, &evt); err == nil {
				usageTokens := evt.Usage.InputTokens + evt.Usage.OutputTokens
				if usageTokens > tokens {
					tokens = usageTokens
				}
				subtype = evt.Subtype
				resultContent = evt.Result
				if resultContent == "" {
					resultContent = evt.Content
				}
			}
		}

		if eventType == "text" && resultContent == "" {
			var evt struct {
				Part struct {
					Text string `json:"text"`
				} `json:"part"`
				Message struct {
					Content []struct {
						Type string `json:"type"`
						Text string `json:"text,omitempty"`
					} `json:"content"`
				} `json:"message"`
			}
			if err := json.Unmarshal(line, &evt); err == nil {
				if evt.Part.Text != "" {
					resultContent = evt.Part.Text
				} else if len(evt.Message.Content) > 0 && evt.Message.Content[0].Text != "" {
					resultContent = evt.Message.Content[0].Text
				}
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

func parseOpenCodeStreamLine(line []byte) (StreamEvent, bool) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return StreamEvent{}, false
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(line, &obj); err == nil {
		var eventType string
		if raw, ok := obj["type"]; ok {
			if err := json.Unmarshal(raw, &eventType); err == nil {
				switch eventType {
				case "text":
					var evt struct {
						Part struct {
							Text string `json:"text"`
						} `json:"part"`
					}
					if err := json.Unmarshal(line, &evt); err == nil && evt.Part.Text != "" {
						text := evt.Part.Text
						if len(text) > 200 {
							text = text[:200]
						}
						return StreamEvent{Type: "text", Content: text}, true
					}
					return StreamEvent{}, false

				case "tool", "tool_call":
					var evt struct {
						Part struct {
							Type  string `json:"type"`
							Tool  string `json:"tool"`
							Input string `json:"input"`
							Name  string `json:"name"`
						} `json:"part"`
					}
					if err := json.Unmarshal(line, &evt); err == nil {
						toolName := evt.Part.Tool
						if toolName == "" {
							toolName = evt.Part.Name
						}
						if toolName != "" {
							input := evt.Part.Input
							if len(input) > 100 {
								input = input[:100]
							}
							return StreamEvent{Type: "tool_use", ToolName: toolName, ToolInput: input}, true
						}
					}
					return StreamEvent{}, false

				case "step_finish", "result":
					var evt struct {
						Part struct {
							Tokens struct {
								Total  int `json:"total"`
								Input  int `json:"input"`
								Output int `json:"output"`
							} `json:"tokens"`
						} `json:"part"`
						Usage struct {
							InputTokens  int `json:"input_tokens"`
							OutputTokens int `json:"output_tokens"`
						} `json:"usage"`
						Content string `json:"content"`
						Result  string `json:"result"`
						Subtype string `json:"subtype"`
					}
					if err := json.Unmarshal(line, &evt); err == nil {
						tokensIn := evt.Usage.InputTokens
						tokensOut := evt.Usage.OutputTokens
						if tokensIn == 0 {
							tokensIn = evt.Part.Tokens.Input
						}
						if tokensOut == 0 {
							tokensOut = evt.Part.Tokens.Output
						}
						content := evt.Result
						if content == "" {
							content = evt.Content
						}
						return StreamEvent{
							Type:      "result",
							TokensIn:  tokensIn,
							TokensOut: tokensOut,
							Content:   content,
							Subtype:   evt.Subtype,
						}, true
					}
					return StreamEvent{}, false

				case "system":
					return StreamEvent{Type: "system"}, true
				}
			}
		}

		_ = eventType

		var toolEvt struct {
			Tool  string          `json:"tool"`
			Input json.RawMessage `json:"input"`
		}
		if err := json.Unmarshal(line, &toolEvt); err == nil && toolEvt.Tool != "" {
			target := extractToolTarget(toolEvt.Tool, toolEvt.Input)
			return StreamEvent{
				Type:      "tool_use",
				ToolName:  toolEvt.Tool,
				ToolInput: target,
			}, true
		}
	}

	return StreamEvent{}, false
}

func (a *OpenCodeAdapter) prepareWorkspace(workspacePath string, cfg AdapterRunConfig) error {
	if cfg.SystemPrompt != "" {
		promptPath := filepath.Join(workspacePath, "AGENTS.md")
		if err := os.WriteFile(promptPath, []byte(cfg.SystemPrompt), 0644); err != nil {
			return fmt.Errorf("failed to write AGENTS.md: %w", err)
		}
	} else {
		personaPath := filepath.Join(".agents", "personas", cfg.Persona+".md")
		if data, err := os.ReadFile(personaPath); err == nil {
			promptPath := filepath.Join(workspacePath, "AGENTS.md")
			_ = os.WriteFile(promptPath, data, 0644)
		}
	}

	return nil
}

func (a *OpenCodeAdapter) buildArgs(cfg AdapterRunConfig) []string {
	args := []string{"run"}

	if cfg.Model != "" && cfg.Model != "default" {
		args = append(args, "--model", cfg.Model)
	}

	args = append(args, "--format", "json")

	if cfg.Prompt != "" {
		args = append(args, "--", cfg.Prompt)
	}

	return args
}

func ResolveAdapter(adapterName string) AdapterRunner {
	name := strings.ToLower(adapterName)
	switch {
	case name == "claude":
		return NewClaudeAdapter()
	case name == "opencode" || strings.HasPrefix(name, "opencode-"):
		return NewOpenCodeAdapter()
	case name == "codex":
		return NewCodexAdapter()
	case name == "gemini":
		return NewGeminiAdapter()
	case name == "browser":
		return NewBrowserAdapter()
	default:
		return NewProcessGroupRunner()
	}
}
