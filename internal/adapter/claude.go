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

type ClaudeSettings struct {
	Model        string   `json:"model"`
	Temperature  float64  `json:"temperature"`
	OutputFormat string   `json:"output_format"`
	AllowedTools []string `json:"allowed_tools"`
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

	if err := a.prepareWorkspace(workspacePath, cfg); err != nil {
		return nil, fmt.Errorf("failed to prepare workspace: %w", err)
	}

	args := a.buildArgs(cfg)
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

func (a *ClaudeAdapter) prepareWorkspace(workspacePath string, cfg AdapterRunConfig) error {
	settingsDir := filepath.Join(workspacePath, ".claude")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
	}

	// Build allowed tools list from config
	allowedTools := cfg.AllowedTools
	if len(allowedTools) == 0 {
		allowedTools = []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep"}
	}

	// Generate settings.json for this step's persona
	settings := ClaudeSettings{
		Model:        "claude-sonnet-4-20250514",
		Temperature:  cfg.Temperature,
		OutputFormat: "json",
		AllowedTools: allowedTools,
	}

	settingsPath := filepath.Join(settingsDir, "settings.json")
	settingsData, _ := json.MarshalIndent(settings, "", "  ")
	if err := os.WriteFile(settingsPath, settingsData, 0644); err != nil {
		return fmt.Errorf("failed to write settings.json: %w", err)
	}

	// Project system prompt from the persona's .md file into CLAUDE.md
	claudeMdPath := filepath.Join(workspacePath, "CLAUDE.md")
	if cfg.SystemPrompt != "" {
		if err := os.WriteFile(claudeMdPath, []byte(cfg.SystemPrompt), 0644); err != nil {
			return fmt.Errorf("failed to write CLAUDE.md: %w", err)
		}
	} else {
		// Try loading from .wave/personas/<persona>.md
		personaPath := filepath.Join(".wave", "personas", cfg.Persona+".md")
		if data, err := os.ReadFile(personaPath); err == nil {
			if err := os.WriteFile(claudeMdPath, data, 0644); err != nil {
				return fmt.Errorf("failed to write CLAUDE.md: %w", err)
			}
		} else if _, err := os.Stat(claudeMdPath); os.IsNotExist(err) {
			defaultPrompt := fmt.Sprintf("# %s\n\nYou are operating as the %s persona.\n", cfg.Persona, cfg.Persona)
			if err := os.WriteFile(claudeMdPath, []byte(defaultPrompt), 0644); err != nil {
				return fmt.Errorf("failed to write CLAUDE.md: %w", err)
			}
		}
	}

	return nil
}

func (a *ClaudeAdapter) buildArgs(cfg AdapterRunConfig) []string {
	args := []string{"-p"}

	if len(cfg.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(cfg.AllowedTools, ","))
	}

	args = append(args, "--output-format", "json")
	args = append(args, "--temperature", fmt.Sprintf("%.1f", cfg.Temperature))
	args = append(args, "--no-continue")

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

