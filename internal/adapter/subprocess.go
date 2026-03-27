package adapter

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// streamLineParser converts a raw NDJSON line into a StreamEvent.
// Returns (event, true) if the line produced a meaningful event,
// or (zero, false) if it should be skipped.
type streamLineParser func(line []byte) (StreamEvent, bool)

// subprocessConfig holds the adapter-specific parameters for runSubprocess.
type subprocessConfig struct {
	BinaryPath   string           // Absolute path or basename of the adapter binary
	BinaryLabel  string           // Human-readable label for error messages (e.g. "codex")
	Args         []string         // CLI arguments
	WorkDir      string           // Working directory for the subprocess
	Env          []string         // Environment variables
	ProcessGrace time.Duration    // SIGTERM→SIGKILL grace period (0 uses default)
	ParseLine    streamLineParser // Adapter-specific NDJSON line parser
	OnEvent      func(StreamEvent) // Stream event callback (from cfg.OnStreamEvent)
}

// runSubprocess executes an adapter binary as a subprocess, streaming its
// stdout line-by-line through the provided parser. This extracts the ~80 lines
// of identical boilerplate shared by CodexAdapter, GeminiAdapter, and
// OpenCodeAdapter into a single reusable function.
//
// The returned AdapterResult has ExitCode, Stdout, and TokensUsed populated.
// Callers are responsible for setting FailureReason via ClassifyFailure.
func runSubprocess(ctx context.Context, sc subprocessConfig) (*AdapterResult, error) {
	cmd := exec.CommandContext(ctx, sc.BinaryPath, sc.Args...)
	cmd.Dir = sc.WorkDir
	if len(sc.Env) > 0 {
		cmd.Env = sc.Env
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start %s: %w", sc.BinaryLabel, err)
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

			if sc.OnEvent != nil && sc.ParseLine != nil {
				if evt, ok := sc.ParseLine(line); ok {
					sc.OnEvent(evt)
				}
			}
		}
		stdoutDone <- scanner.Err()
	}()

	select {
	case <-ctx.Done():
		if cmd.Process != nil {
			killProcessGroup(cmd.Process, sc.ProcessGrace)
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

// resolveWorkspacePath returns the workspace path from config, falling back
// to the current working directory.
func resolveWorkspacePath(cfgPath string) (string, error) {
	if cfgPath != "" {
		return cfgPath, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	return wd, nil
}

// readReaderContent reads all content from an io.Reader and returns the bytes.
// This is used to capture stdout content for failure classification while
// preserving the ability to re-read the data.
func readReaderContent(r io.Reader) ([]byte, error) {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r)
	return buf.Bytes(), err
}
