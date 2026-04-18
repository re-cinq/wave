package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	// Build subprocess command: wave run --pipeline <name> --run <runID> --input <input> [flags...]
	args := []string{"run", "--pipeline", config.PipelineName, "--run", runID}
	if config.Input != "" {
		args = append(args, "--input", config.Input)
	}
	if config.ModelOverride != "" {
		args = append(args, "--model", config.ModelOverride)
	}
	if config.Adapter != "" {
		args = append(args, "--adapter", config.Adapter)
	}
	if config.Timeout > 0 {
		args = append(args, "--timeout", fmt.Sprintf("%d", config.Timeout))
	}
	if config.FromStep != "" {
		args = append(args, "--from-step", config.FromStep)
	}
	if config.Steps != "" {
		args = append(args, "--steps", config.Steps)
	}
	if config.Exclude != "" {
		args = append(args, "--exclude", config.Exclude)
	}
	if config.OnFailure != "" && config.OnFailure != "halt" {
		args = append(args, "--on-failure", config.OnFailure)
	}
	// Pass through user-selected flags to the subprocess.
	// Compound flags like "--output text" are split into separate args.
	for _, f := range config.Flags {
		parts := strings.SplitN(f, " ", 2)
		args = append(args, parts...)
	}

	cmd := exec.Command(os.Args[0], args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Env = buildPassthroughEnv(l.deps)

	// Redirect stdout/stderr to .agents/logs/<runID>.log so detached output is preserved
	logFile, logErr := openRunLog(runID)
	if logErr == nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	} else {
		cmd.Stdout = nil
		cmd.Stderr = nil
	}

	if startErr := cmd.Start(); startErr != nil {
		if logFile != nil {
			logFile.Close()
		}
		pipelineName := config.PipelineName
		return func() tea.Msg {
			return LaunchErrorMsg{PipelineName: pipelineName, Err: fmt.Errorf("starting subprocess: %w", startErr)}
		}
	}

	// Close the log file — the subprocess inherited the fd via fork
	if logFile != nil {
		logFile.Close()
	}

	// Record PID and release the process so it becomes fully detached
	if l.deps.Store != nil {
		_ = l.deps.Store.UpdateRunPID(runID, cmd.Process.Pid)
	}
	_ = cmd.Process.Release()

	// Return immediate PipelineLaunchedMsg — no blocking executor cmd
	pipelineName := config.PipelineName
	input := config.Input
	verbose := config.Verbose
	debug := config.Debug
	return func() tea.Msg {
		return PipelineLaunchedMsg{
			RunID:        runID,
			PipelineName: pipelineName,
			Input:        input,
			Verbose:      verbose,
			Debug:        debug,
		}
	}
}

// LaunchSequence starts an orchestrated pipeline sequence as a detached
// `wave compose` subprocess. When parallel is true the --parallel flag is
// passed and stages are separated by "--". Returns a single PipelineLaunchedMsg
// whose RunID is the compose group ID.
func (l *PipelineLauncher) LaunchSequence(names []string, input string, parallel bool, stages [][]int) tea.Cmd {
	if len(names) == 0 {
		return func() tea.Msg {
			return LaunchErrorMsg{PipelineName: "compose", Err: fmt.Errorf("no pipelines in sequence")}
		}
	}

	// Generate a run ID for the compose group
	groupRunID := pipeline.GenerateRunID("compose", 8)
	if l.deps.Store != nil {
		if rid, err := l.deps.Store.CreateRun("compose:"+strings.Join(names, "+"), input); err == nil {
			groupRunID = rid
		}
	}
	if l.deps.Store != nil {
		_ = l.deps.Store.UpdateRunStatus(groupRunID, "running", "", 0)
	}

	// Build args: wave compose [--parallel] [--input <input>] <names...>
	// With stages: wave compose --parallel A B -- C D
	args := []string{"compose"}
	if parallel {
		args = append(args, "--parallel")
	}
	if input != "" {
		args = append(args, "--input", input)
	}

	if parallel && len(stages) > 0 {
		// Insert stage groups separated by "--"
		for i, group := range stages {
			if i > 0 {
				args = append(args, "--")
			}
			for _, idx := range group {
				if idx < len(names) {
					args = append(args, names[idx])
				}
			}
		}
	} else {
		args = append(args, names...)
	}

	cmd := exec.Command(os.Args[0], args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Env = buildPassthroughEnv(l.deps)

	logFile, logErr := openRunLog(groupRunID)
	if logErr == nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	if startErr := cmd.Start(); startErr != nil {
		if logFile != nil {
			logFile.Close()
		}
		return func() tea.Msg {
			return LaunchErrorMsg{PipelineName: "compose", Err: fmt.Errorf("starting compose subprocess: %w", startErr)}
		}
	}

	if logFile != nil {
		logFile.Close()
	}

	if l.deps.Store != nil {
		_ = l.deps.Store.UpdateRunPID(groupRunID, cmd.Process.Pid)
	}
	_ = cmd.Process.Release()

	runID := groupRunID
	return func() tea.Msg {
		return PipelineLaunchedMsg{
			RunID:        runID,
			PipelineName: "compose:" + strings.Join(names, "+"),
			Input:        input,
		}
	}
}

// Cancel requests cancellation of a pipeline run via the state store.
// For live runs, the executor's pollCancellation goroutine picks this up.
// For stale runs (dead process), we directly mark them as cancelled since
// no executor is alive to act on the cancellation request.
func (l *PipelineLauncher) Cancel(runID string) {
	if l.deps.Store == nil {
		return
	}

	// Check if this run has a PID and whether the process is still alive.
	// If the process is dead, skip the cancellation request and directly
	// mark the run as cancelled — no executor will ever pick it up.
	runs, err := l.deps.Store.GetRunningRuns()
	if err == nil {
		for _, r := range runs {
			if r.RunID == runID && r.PID > 0 && !IsProcessAlive(r.PID) {
				_ = l.deps.Store.UpdateRunStatus(runID, "cancelled", "dismissed — process no longer running", 0)
				return
			}
		}
	}

	_ = l.deps.Store.RequestCancellation(runID, false)
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

// openRunLog creates the .agents/logs/ directory if needed and opens a log file for the run.
func openRunLog(runID string) (*os.File, error) {
	logsDir := filepath.Join(".agents", "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return nil, err
	}
	return os.OpenFile(filepath.Join(logsDir, runID+".log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
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
