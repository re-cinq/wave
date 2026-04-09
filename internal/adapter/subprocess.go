package adapter

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/recinq/wave/internal/timeouts"
)

// streamLineParser parses a single NDJSON line and returns a StreamEvent.
type streamLineParser func(line []byte) (StreamEvent, bool)

// outputParser parses the complete buffered stdout into structured result data.
type outputParser func(data []byte) parseOutputResult

// runSubprocess is the shared subprocess execution loop used by Codex, Gemini,
// and OpenCode adapters. It manages context-based timeouts, NDJSON streaming,
// process group cleanup, failure classification, and result content extraction.
func runSubprocess(
	ctx context.Context,
	binaryPath string,
	args []string,
	workspacePath string,
	cfg AdapterRunConfig,
	parseLine streamLineParser,
	parseOut outputParser,
) (*AdapterResult, error) {
	var cancel context.CancelFunc
	if cfg.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, args...)
	cmd.Dir = workspacePath
	cmd.Env = BuildCuratedEnvironment(cfg)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	binaryName := binaryPath
	if idx := strings.LastIndex(binaryPath, "/"); idx >= 0 {
		binaryName = binaryPath[idx+1:]
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start %s: %w", binaryName, err)
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

			if cfg.OnStreamEvent != nil && parseLine != nil {
				if evt, ok := parseLine(line); ok {
					cfg.OnStreamEvent(evt)
				}
			}
		}
		stdoutDone <- scanner.Err()
	}()

	select {
	case <-ctx.Done():
		if cmd.Process != nil {
			killProcessGroup(cmd.Process, cfg.ProcessGrace)
		}
		drainTimeout := cfg.StdoutDrain
		if drainTimeout <= 0 {
			drainTimeout = timeouts.StdoutDrain
		}
		select {
		case <-stdoutDone:
		case <-time.After(drainTimeout):
		}
		_ = cmd.Wait()

		parsed := parseOut(stdoutBuf.Bytes())
		reason := ClassifyFailure(parsed.Subtype, parsed.ResultContent, ctx.Err())
		return nil, NewStepError(reason, ctx.Err(), parsed.Tokens, parsed.Subtype)

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

	parsed := parseOut(stdoutBuf.Bytes())
	result.TokensUsed = parsed.Tokens
	result.TokensIn = parsed.TokensIn
	result.TokensOut = parsed.TokensOut
	result.Artifacts = parsed.Artifacts
	result.Subtype = parsed.Subtype
	result.ResultContent = parsed.ResultContent

	if result.ExitCode != 0 || parsed.Subtype == "error_max_turns" || parsed.Subtype == "error_during_execution" {
		result.FailureReason = ClassifyFailure(parsed.Subtype, parsed.ResultContent, nil)
	}

	if cfg.Debug {
		fmt.Printf("[DEBUG] %s exit code: %d\n", binaryName, result.ExitCode)
		fmt.Printf("[DEBUG] %s tokens used: %d\n", binaryName, parsed.Tokens)
		fmt.Printf("[DEBUG] %s raw output (%d bytes)\n", binaryName, stdoutBuf.Len())
		fmt.Printf("[DEBUG] Extracted result content (%d chars)\n", len(parsed.ResultContent))
	}

	return result, nil
}
