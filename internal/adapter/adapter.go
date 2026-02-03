package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type AdapterRunner interface {
	Run(ctx context.Context, cfg AdapterRunConfig) (*AdapterResult, error)
}

type AdapterRunConfig struct {
	Adapter       string
	Persona       string
	WorkspacePath string
	Prompt        string
	SystemPrompt  string
	Timeout       time.Duration
	Env           []string
	Temperature   float64
	AllowedTools  []string
	DenyTools     []string
	OutputFormat  string
	Debug         bool
}

type AdapterResult struct {
	ExitCode      int
	Stdout        io.Reader
	TokensUsed    int
	Artifacts     []string
	ResultContent string // Extracted content from the adapter response
}

type ProcessGroupRunner struct{}

func NewProcessGroupRunner() *ProcessGroupRunner {
	return &ProcessGroupRunner{}
}

func (r *ProcessGroupRunner) Run(ctx context.Context, cfg AdapterRunConfig) (*AdapterResult, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Minute
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, cfg.Adapter)
	cmd.Dir = cfg.WorkspacePath

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
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	var stdoutBuf bytes.Buffer
	copyDone := make(chan error, 1)
	go func() {
		_, err := io.Copy(&stdoutBuf, stdoutPipe)
		copyDone <- err
	}()

	select {
	case <-ctx.Done():
		killProcessGroup(cmd.Process)
		cmd.Wait()
		return nil, ctx.Err()
	case err := <-copyDone:
		if err != nil {
			return nil, fmt.Errorf("failed to read stdout: %w", err)
		}
		if err := cmd.Wait(); err != nil {
			return &AdapterResult{
				ExitCode:   exitCodeFromError(err),
				Stdout:     bytes.NewReader(stdoutBuf.Bytes()),
				TokensUsed: 0,
				Artifacts:  nil,
			}, nil
		}
	}

	var result AdapterResult
	result.ExitCode = 0
	result.Stdout = bytes.NewReader(stdoutBuf.Bytes())
	result.TokensUsed = estimateTokens(stdoutBuf.String())

	parseArtifacts(stdoutBuf.Bytes(), &result.Artifacts)

	return &result, nil
}

func killProcessGroup(process *os.Process) {
	_ = syscall.Kill(-process.Pid, syscall.SIGKILL)
	_ = process.Release()
}

func exitCodeFromError(err error) int {
	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}
	return -1
}

func estimateTokens(text string) int {
	return len(text) / 4
}

func parseArtifacts(data []byte, artifacts *[]string) {
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return
	}
	if artifactList, ok := parsed["artifacts"].([]interface{}); ok {
		for _, a := range artifactList {
			if s, ok := a.(string); ok {
				*artifacts = append(*artifacts, s)
			}
		}
	}
}
