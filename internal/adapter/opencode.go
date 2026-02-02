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
	cmd := exec.CommandContext(ctx, a.opencodePath, args...)
	cmd.Dir = workspacePath

	mergedEnv := append(os.Environ(), cfg.Env...)
	cmd.Env = mergedEnv

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start opencode: %w", err)
	}

	var stdoutBuf bytes.Buffer
	stdoutDone := make(chan error, 1)

	go func() {
		_, err := io.Copy(&stdoutBuf, stdoutPipe)
		stdoutDone <- err
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

func (a *OpenCodeAdapter) prepareWorkspace(workspacePath string, cfg AdapterRunConfig) error {
	// OpenCode uses .opencode/ directory for configuration
	settingsDir := filepath.Join(workspacePath, ".opencode")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return fmt.Errorf("failed to create .opencode directory: %w", err)
	}

	config := map[string]interface{}{
		"provider":    "anthropic",
		"model":       "claude-sonnet-4-20250514",
		"temperature": cfg.Temperature,
	}

	configData, _ := json.MarshalIndent(config, "", "  ")
	configPath := filepath.Join(settingsDir, "config.json")
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write config.json: %w", err)
	}

	// Project system prompt if available
	if cfg.SystemPrompt != "" {
		promptPath := filepath.Join(workspacePath, "AGENTS.md")
		if err := os.WriteFile(promptPath, []byte(cfg.SystemPrompt), 0644); err != nil {
			return fmt.Errorf("failed to write AGENTS.md: %w", err)
		}
	} else {
		personaPath := filepath.Join(".wave", "personas", cfg.Persona+".md")
		if data, err := os.ReadFile(personaPath); err == nil {
			promptPath := filepath.Join(workspacePath, "AGENTS.md")
			os.WriteFile(promptPath, data, 0644)
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
	default:
		return NewProcessGroupRunner()
	}
}
