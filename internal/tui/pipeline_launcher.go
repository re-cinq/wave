package tui

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/pipeline"
)

// PipelineLauncher manages pipeline execution from the TUI.
// It spawns detached subprocesses that survive TUI exit and tracks them via SQLite.
type PipelineLauncher struct {
	deps    LaunchDependencies
	program *tea.Program
	mu      sync.Mutex
}

// NewPipelineLauncher creates a new launcher with the given dependencies.
func NewPipelineLauncher(deps LaunchDependencies) *PipelineLauncher {
	return &PipelineLauncher{
		deps: deps,
	}
}

// SetProgram sets the Bubble Tea program reference for sending messages.
func (l *PipelineLauncher) SetProgram(p *tea.Program) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.program = p
}

// Launch spawns a pipeline as a detached subprocess via exec.Command.
// The subprocess re-executes the wave binary with "run" arguments and survives TUI exit.
// Returns tea.Cmd for immediate UI feedback (PipelineLaunchedMsg).
func (l *PipelineLauncher) Launch(config LaunchConfig) tea.Cmd {
	// Load the full pipeline definition to get the canonical name
	p, err := LoadPipelineByName(l.deps.PipelinesDir, config.PipelineName)
	if err != nil {
		pipelineName := config.PipelineName
		return func() tea.Msg {
			return LaunchErrorMsg{PipelineName: pipelineName, Err: fmt.Errorf("loading pipeline: %w", err)}
		}
	}

	store := l.deps.Store

	// Generate run ID — must be created before subprocess spawn so we can pass it via --run
	var runID string
	if store != nil {
		var storeErr error
		runID, storeErr = store.CreateRun(p.Metadata.Name, config.Input)
		if storeErr != nil {
			runID = pipeline.GenerateRunID(p.Metadata.Name, 8)
		}
	} else {
		runID = pipeline.GenerateRunID(p.Metadata.Name, 8)
	}

	// Build subprocess command: wave run <pipeline> --run <runID> --input <input>
	args := []string{"run", "--pipeline", config.PipelineName, "--run", runID}
	if config.Input != "" {
		args = append(args, "--input", config.Input)
	}
	if config.ModelOverride != "" {
		args = append(args, "--model", config.ModelOverride)
	}
	for _, f := range config.Flags {
		args = append(args, f)
	}

	cmd := exec.Command(os.Args[0], args...)

	// Detach: create a new session so the subprocess survives TUI exit and terminal SIGHUP
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	// Build environment with passthrough vars (FR-012: no credentials in CLI args)
	cmd.Env = buildPassthroughEnv(l.deps)

	// Suppress subprocess stdout/stderr — all output goes through SQLite events
	cmd.Stdout = nil
	cmd.Stderr = nil

	pipelineName := config.PipelineName

	// Spawn the subprocess
	if err := cmd.Start(); err != nil {
		return func() tea.Msg {
			return LaunchErrorMsg{PipelineName: pipelineName, Err: fmt.Errorf("spawning subprocess: %w", err)}
		}
	}

	// Store PID for liveness detection
	if store != nil && cmd.Process != nil {
		_ = store.UpdateRunPID(runID, cmd.Process.Pid)
	}

	// Release the process — we don't wait for it, it's fully detached
	if cmd.Process != nil {
		_ = cmd.Process.Release()
	}

	// Return immediate feedback — no blocking executor cmd since the subprocess is detached
	return func() tea.Msg {
		return PipelineLaunchedMsg{
			RunID:        runID,
			PipelineName: pipelineName,
		}
	}
}

// Cancel requests cancellation of a detached pipeline via the persistent store (FR-005).
func (l *PipelineLauncher) Cancel(runID string) {
	if l.deps.Store != nil {
		_ = l.deps.Store.RequestCancellation(runID, false)
	}
}

// CancelAll is a no-op for detached subprocesses (FR-004).
// Detached pipelines survive TUI exit — CancelAll only cleans up TUI-side state.
func (l *PipelineLauncher) CancelAll() {
	// No-op: detached subprocesses manage their own lifecycle
}

// Cleanup is a no-op for detached subprocesses — they manage their own lifecycle.
func (l *PipelineLauncher) Cleanup(runID string) {
	// No-op: subprocess handles its own state transitions and cleanup
}

// buildPassthroughEnv constructs the subprocess environment from the manifest's
// runtime.sandbox.env_passthrough configuration. Only explicitly allowed
// environment variables are passed through (FR-012).
func buildPassthroughEnv(deps LaunchDependencies) []string {
	// Start with minimal required env vars
	env := []string{}

	// Always pass HOME and PATH for basic operation
	if home := os.Getenv("HOME"); home != "" {
		env = append(env, "HOME="+home)
	}
	if path := os.Getenv("PATH"); path != "" {
		env = append(env, "PATH="+path)
	}

	// Pass through vars from manifest configuration
	if deps.Manifest != nil {
		for _, key := range deps.Manifest.Runtime.Sandbox.EnvPassthrough {
			if val := os.Getenv(key); val != "" {
				env = append(env, key+"="+val)
			}
		}
	}

	return env
}

// TUIProgressEmitter implements event.ProgressEmitter to bridge executor events
// into the Bubble Tea event loop via program.Send().
type TUIProgressEmitter struct {
	program *tea.Program
	runID   string
}

// EmitProgress sends the event as a PipelineEventMsg to the TUI program.
func (e *TUIProgressEmitter) EmitProgress(evt event.Event) error {
	if e.program != nil {
		e.program.Send(PipelineEventMsg{RunID: e.runID, Event: evt})
	}
	return nil
}
