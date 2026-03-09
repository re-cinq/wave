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
// It spawns detached subprocesses via `wave run` so pipelines survive TUI exit.
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

// Launch starts a pipeline as a detached subprocess and returns a tea.Cmd
// that immediately sends PipelineLaunchedMsg. Live output comes from polling
// SQLite events, not in-memory buffers.
func (l *PipelineLauncher) Launch(config LaunchConfig) tea.Cmd {
	// Load the full pipeline definition to validate it exists
	p, err := LoadPipelineByName(l.deps.PipelinesDir, config.PipelineName)
	if err != nil {
		pipelineName := config.PipelineName
		return func() tea.Msg {
			return LaunchErrorMsg{PipelineName: pipelineName, Err: fmt.Errorf("loading pipeline: %w", err)}
		}
	}

	// Generate run ID via StateStore so the run appears in the dashboard
	var runID string
	if l.deps.Store != nil {
		var storeErr error
		runID, storeErr = l.deps.Store.CreateRun(p.Metadata.Name, config.Input)
		if storeErr != nil {
			runID = pipeline.GenerateRunID(p.Metadata.Name, 8)
		}
	} else {
		runID = pipeline.GenerateRunID(p.Metadata.Name, 8)
	}

	// Transition run from pending -> running so wave status picks it up
	if l.deps.Store != nil {
		_ = l.deps.Store.UpdateRunStatus(runID, "running", "", 0)
	}

	// Check for --mock flag
	isMock := false
	for _, f := range config.Flags {
		if f == "--mock" {
			isMock = true
			break
		}
	}

	// Build subprocess command: wave run --pipeline <name> --run <runID> --input <input> [flags...]
	args := []string{"run", "--pipeline", config.PipelineName, "--run", runID}
	if config.Input != "" {
		args = append(args, "--input", config.Input)
	}
	if config.ModelOverride != "" {
		args = append(args, "--model", config.ModelOverride)
	}
	if isMock {
		args = append(args, "--mock")
	}

	cmd := exec.Command(os.Args[0], args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Env = buildPassthroughEnv(l.deps)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if startErr := cmd.Start(); startErr != nil {
		pipelineName := config.PipelineName
		return func() tea.Msg {
			return LaunchErrorMsg{PipelineName: pipelineName, Err: fmt.Errorf("starting subprocess: %w", startErr)}
		}
	}

	// Record PID and release the process so it becomes fully detached
	if l.deps.Store != nil {
		_ = l.deps.Store.UpdateRunPID(runID, cmd.Process.Pid)
	}
	_ = cmd.Process.Release()

	// Return immediate PipelineLaunchedMsg — no blocking executor cmd
	pipelineName := config.PipelineName
	return func() tea.Msg {
		return PipelineLaunchedMsg{
			RunID:        runID,
			PipelineName: pipelineName,
		}
	}
}

// Cancel requests cancellation of a pipeline run via the state store.
// The executor's pollCancellation goroutine will pick this up.
func (l *PipelineLauncher) Cancel(runID string) {
	if l.deps.Store != nil {
		_ = l.deps.Store.RequestCancellation(runID, false)
	}
}

// CancelAll is a no-op for detached pipelines — they survive TUI exit.
func (l *PipelineLauncher) CancelAll() {
	// Detached subprocesses manage their own lifecycle.
}

// Cleanup is a no-op for detached pipelines — subprocesses manage their own lifecycle.
func (l *PipelineLauncher) Cleanup(_ string) {
	// No in-process state to clean up.
}

// buildPassthroughEnv constructs a minimal environment for the subprocess.
// It includes HOME, PATH, and any vars specified in manifest runtime.sandbox.env_passthrough.
func buildPassthroughEnv(deps LaunchDependencies) []string {
	env := []string{
		"HOME=" + os.Getenv("HOME"),
		"PATH=" + os.Getenv("PATH"),
	}

	if deps.Manifest != nil && len(deps.Manifest.Runtime.Sandbox.EnvPassthrough) > 0 {
		for _, key := range deps.Manifest.Runtime.Sandbox.EnvPassthrough {
			if val, ok := os.LookupEnv(key); ok {
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
