package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type ClaudeAdapter struct {
	claudePath string
}

func NewClaudeAdapter() *ClaudeAdapter {
	path := "/usr/local/bin/claude"
	if p, err := exec.LookPath("claude"); err == nil {
		path = p
	}
	return &ClaudeAdapter{claudePath: path}
}

type PersonaConfig struct {
	Name             string   `json:"name"`
	Permissions      []string `json:"permissions"`
	SystemPrompt     string   `json:"system_prompt"`
	SystemPromptFile string   `json:"system_prompt_file"`
	Temperature      float64  `json:"temperature"`
	Model            string   `json:"model"`
	PreExecuteHooks  []string `json:"pre_execute_hooks"`
	PostExecuteHooks []string `json:"post_execute_hooks"`
}

type HookConfig struct {
	PreExecute  []string `json:"pre_execute"`
	PostExecute []string `json:"post_execute"`
}

type ClaudeSettings struct {
	Model        string     `json:"model"`
	Temperature  float64    `json:"temperature"`
	OutputFormat string     `json:"output_format"`
	AllowedTools []string   `json:"allowed_tools"`
	Hooks        HookConfig `json:"hooks"`
}

func (a *ClaudeAdapter) Run(ctx context.Context, cfg AdapterRunConfig) (*AdapterResult, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Minute
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	workspacePath := cfg.WorkspacePath
	if workspacePath == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		workspacePath = wd
	}

	if err := a.prepareWorkspace(workspacePath, cfg.Persona); err != nil {
		return nil, fmt.Errorf("failed to prepare workspace: %w", err)
	}

	persona, err := a.loadPersona(cfg.Persona)
	if err != nil {
		return nil, fmt.Errorf("failed to load persona: %w", err)
	}

	args := a.buildArgs(persona, cfg)
	cmd := exec.CommandContext(ctx, a.claudePath, args...)
	cmd.Dir = workspacePath

	mergedEnv := append(os.Environ(), cfg.Env...)
	cmd.Env = mergedEnv

	// Set up process group for clean timeout kill
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start claude: %w", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	stdoutDone := make(chan error, 1)
	stderrDone := make(chan error, 1)

	go func() {
		_, err := io.Copy(&stdoutBuf, stdoutPipe)
		stdoutDone <- err
	}()

	go func() {
		_, err := io.Copy(&stderrBuf, stderrPipe)
		stderrDone <- err
	}()

	select {
	case <-ctx.Done():
		if cmd.Process != nil {
			killProcessGroup(cmd.Process)
		}
		cmd.Wait()
		return nil, ctx.Err()
	case err := <-stdoutDone:
		if err != nil {
			return nil, fmt.Errorf("failed to read stdout: %w", err)
		}
	case err := <-stderrDone:
		if err != nil {
			return nil, fmt.Errorf("failed to read stderr: %w", err)
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

	tokens, artifacts := a.parseOutput(stdoutBuf.Bytes())
	result.TokensUsed = tokens
	result.Artifacts = artifacts

	return result, nil
}

func (a *ClaudeAdapter) prepareWorkspace(workspacePath, personaName string) error {
	settingsDir := filepath.Join(workspacePath, ".claude")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
	}

	// Load persona to get configuration
	persona, err := a.loadPersona(personaName)
	if err != nil {
		return fmt.Errorf("failed to load persona for workspace setup: %w", err)
	}

	// Generate settings based on persona configuration
	settings := ClaudeSettings{
		Model:        persona.Model,
		Temperature:  persona.Temperature,
		OutputFormat: "json",
		AllowedTools: persona.Permissions,
		Hooks: HookConfig{
			PreExecute:  persona.PreExecuteHooks,
			PostExecute: persona.PostExecuteHooks,
		},
	}

	// Set defaults if not specified in persona
	if settings.Model == "" {
		settings.Model = "claude-sonnet-4-20250514"
	}
	if settings.Temperature == 0 {
		settings.Temperature = 0.0
	}

	settingsPath := filepath.Join(settingsDir, "settings.json")
	settingsData, _ := json.MarshalIndent(settings, "", "  ")
	if err := os.WriteFile(settingsPath, settingsData, 0644); err != nil {
		return fmt.Errorf("failed to write settings.json: %w", err)
	}

	// Project system prompt from persona's system_prompt_file if specified
	claudeMdPath := filepath.Join(workspacePath, "CLAUDE.md")
	if persona.SystemPromptFile != "" {
		if err := a.projectSystemPrompt(persona.SystemPromptFile, claudeMdPath); err != nil {
			return fmt.Errorf("failed to project system prompt: %w", err)
		}
	} else if _, err := os.Stat(claudeMdPath); os.IsNotExist(err) {
		// Use persona's system prompt or default
		systemPrompt := persona.SystemPrompt
		if systemPrompt == "" {
			systemPrompt = "# CLAUDE.md\n\nThis is a Claude Code project.\n"
		}
		if err := os.WriteFile(claudeMdPath, []byte(systemPrompt), 0644); err != nil {
			return fmt.Errorf("failed to write CLAUDE.md: %w", err)
		}
	}

	return nil
}

func (a *ClaudeAdapter) loadPersona(name string) (*PersonaConfig, error) {
	if name == "" {
		name = "default"
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	personaPaths := []string{
		filepath.Join(homeDir, ".config", "wave", "personas", name+".json"),
		filepath.Join(".", "personas", name+".json"),
		filepath.Join("/etc/wave", "personas", name+".json"),
	}

	for _, path := range personaPaths {
		data, err := os.ReadFile(path)
		if err == nil {
			var persona PersonaConfig
			if err := json.Unmarshal(data, &persona); err == nil {
				return &persona, nil
			}
		}
	}

	return &PersonaConfig{
		Name:             name,
		Permissions:      []string{"Read", "Write", "Execute", "Edit", "Glob", "Grep", "LS", "WebFetch"},
		SystemPrompt:     "You are a helpful AI assistant.",
		SystemPromptFile: "",
		Temperature:      0.0,
		Model:            "",
		PreExecuteHooks:  []string{},
		PostExecuteHooks: []string{},
	}, nil
}

func (a *ClaudeAdapter) projectSystemPrompt(sourceFile, targetFile string) error {
	// Try to resolve source file relative to persona directories
	homeDir, _ := os.UserHomeDir()
	sourcePaths := []string{
		sourceFile,
		filepath.Join(homeDir, ".config", "wave", "personas", sourceFile),
		filepath.Join(".", "personas", sourceFile),
		filepath.Join("/etc/wave", "personas", sourceFile),
	}

	var sourceData []byte
	var sourcePath string
	for _, path := range sourcePaths {
		if data, err := os.ReadFile(path); err == nil {
			sourceData = data
			sourcePath = path
			break
		}
	}

	if sourceData == nil {
		return fmt.Errorf("system prompt file not found: %s", sourceFile)
	}

	// Copy the system prompt file to CLAUDE.md
	if err := os.WriteFile(targetFile, sourceData, 0644); err != nil {
		return fmt.Errorf("failed to copy system prompt from %s to %s: %w", sourcePath, targetFile, err)
	}

	return nil
}

func (a *ClaudeAdapter) buildArgs(persona *PersonaConfig, cfg AdapterRunConfig) []string {
	args := []string{"-p"}

	if len(persona.Permissions) > 0 {
		args = append(args, "--allowedTools", strings.Join(persona.Permissions, ","))
	}

	args = append(args, "--output-format", "json")

	// Use persona-specific temperature
	temp := persona.Temperature
	if temp == 0 {
		temp = 0.0
	}
	args = append(args, "--temperature", fmt.Sprintf("%.1f", temp))

	args = append(args, "--no-continue")

	// Use persona-specific model if specified
	if persona.Model != "" {
		args = append(args, "--model", persona.Model)
	}

	if cfg.Prompt != "" {
		args = append(args, cfg.Prompt)
	}

	return args
}

func (a *ClaudeAdapter) parseOutput(data []byte) (int, []string) {
	var tokens int
	var artifacts []string

	lines := bytes.Split(data, []byte("\n"))
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		var chunk struct {
			Type    string `json:"type"`
			Content struct {
				Text      string   `json:"text"`
				Tokens    int      `json:"tokens"`
				Artifacts []string `json:"artifacts,omitempty"`
			} `json:"content,omitempty"`
		}

		if err := json.Unmarshal(line, &chunk); err != nil {
			continue
		}

		if chunk.Type == "result" || chunk.Type == "output" {
			tokens += chunk.Content.Tokens
			artifacts = append(artifacts, chunk.Content.Artifacts...)
		}
	}

	if tokens == 0 {
		tokens = len(data) / 4
	}

	return tokens, artifacts
}

func (a *ClaudeAdapter) GenerateHookConfig(preExecute, postExecute []string) HookConfig {
	return HookConfig{
		PreExecute:  preExecute,
		PostExecute: postExecute,
	}
}

type ClaudeAdapterOption func(*ClaudeSettings)

func WithModel(model string) ClaudeAdapterOption {
	return func(s *ClaudeSettings) {
		s.Model = model
	}
}

func WithTemperature(temp float64) ClaudeAdapterOption {
	return func(s *ClaudeSettings) {
		s.Temperature = temp
	}
}

func WithAllowedTools(tools []string) ClaudeAdapterOption {
	return func(s *ClaudeSettings) {
		s.AllowedTools = tools
	}
}

func WithHooks(hooks HookConfig) ClaudeAdapterOption {
	return func(s *ClaudeSettings) {
		s.Hooks = hooks
	}
}

func (a *ClaudeAdapter) Configure(opts ...ClaudeAdapterOption) ClaudeSettings {
	settings := ClaudeSettings{
		Model:        "claude-sonnet-4-20250514",
		Temperature:  0.0,
		OutputFormat: "json",
		AllowedTools: []string{},
		Hooks: HookConfig{
			PreExecute:  []string{},
			PostExecute: []string{},
		},
	}
	for _, opt := range opts {
		opt(&settings)
	}
	return settings
}
