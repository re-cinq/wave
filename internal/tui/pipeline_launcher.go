package tui

import (
	"fmt"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/recinq/wave/internal/config"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/pipelinecatalog"
	"github.com/recinq/wave/internal/runner"
)

// PipelineLauncher manages pipeline execution from the TUI.
// It spawns detached subprocesses via internal/runner so pipelines survive
// TUI exit. The runner package owns the actual fork/exec dance, env filter,
// log-file routing, and PID record — this type is purely the TUI-facing
// adapter that translates LaunchConfig into config.RuntimeConfig and emits
// Bubble Tea messages on success/failure.
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
//
// Subprocess spawning is delegated to runner.Detach so the TUI launch path
// shares one canonical detach contract (argv, env, log file, PID record)
// with the foreground CLI (`wave run --detach`) and the webui server. The
// flag-spec exhaustiveness test in internal/runner guards the argv shape.
func (l *PipelineLauncher) Launch(config LaunchConfig) tea.Cmd {
	// Load the full pipeline definition to validate it exists.
	p, err := pipelinecatalog.LoadPipelineByName(l.deps.PipelinesDir, config.PipelineName)
	if err != nil {
		pipelineName := config.PipelineName
		return func() tea.Msg {
			return LaunchErrorMsg{PipelineName: pipelineName, Err: fmt.Errorf("loading pipeline: %w", err)}
		}
	}

	cfg := runner.DetachConfig{
		ExtraEnv: manifestEnvPassthrough(l.deps.Manifest),
	}

	// Two paths preserve the original TUI behaviour exactly:
	//
	//  - With a state store, delegate to runner.Detach. It owns run-id
	//    reservation (reusing a pre-created row when opts.RunID is set,
	//    falling through to CreateRunWithLimit otherwise), the running
	//    transition, and the spawn dance. We capture the canonical id it
	//    returns and forward that to the TUI.
	//  - Without a state store the TUI used to launch with a locally
	//    generated id and skip persistence entirely. runner.Detach refuses
	//    a nil store, so we drop down to runner.SpawnDetached for that
	//    edge case so subprocesses still survive TUI exit.
	var canonicalRunID string
	if l.deps.Store != nil {
		preID, storeErr := l.deps.Store.CreateRun(p.Metadata.Name, config.Input)
		if storeErr != nil {
			// Generate a local id; runner.Detach will see no matching row
			// and reserve a fresh canonical id via CreateRunWithLimit.
			preID = pipeline.GenerateRunID(p.Metadata.Name, 8)
		}
		opts := launchConfigToOptions(config, preID)
		rid, err := runner.Detach(opts, l.deps.Store, 0, cfg)
		if err != nil {
			pipelineName := config.PipelineName
			return func() tea.Msg {
				return LaunchErrorMsg{PipelineName: pipelineName, Err: fmt.Errorf("starting subprocess: %w", err)}
			}
		}
		canonicalRunID = rid
	} else {
		runID := pipeline.GenerateRunID(p.Metadata.Name, 8)
		opts := launchConfigToOptions(config, runID)
		args := runner.BuildDetachedArgs(opts, runID)
		if err := runner.SpawnDetached(args, runID, nil, cfg); err != nil {
			pipelineName := config.PipelineName
			return func() tea.Msg {
				return LaunchErrorMsg{PipelineName: pipelineName, Err: fmt.Errorf("starting subprocess: %w", err)}
			}
		}
		canonicalRunID = runID
	}

	// Return immediate PipelineLaunchedMsg — no blocking executor cmd
	pipelineName := config.PipelineName
	input := config.Input
	verbose := config.Verbose
	debug := config.Debug
	return func() tea.Msg {
		return PipelineLaunchedMsg{
			RunID:        canonicalRunID,
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
//
// The fork/exec dance, env filter, and log-file routing are delegated to
// runner.SpawnDetached so the TUI does not maintain its own copy of that
// logic. Only the `wave compose` argv assembly is TUI-specific.
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

	cfg := runner.DetachConfig{
		ExtraEnv: manifestEnvPassthrough(l.deps.Manifest),
	}

	var recorder runner.PIDRecorder
	if l.deps.Store != nil {
		recorder = l.deps.Store
	}
	if err := runner.SpawnDetached(args, groupRunID, recorder, cfg); err != nil {
		return func() tea.Msg {
			return LaunchErrorMsg{PipelineName: "compose", Err: fmt.Errorf("starting compose subprocess: %w", err)}
		}
	}

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

// launchConfigToOptions translates the form-driven LaunchConfig into the
// config.RuntimeConfig surface. Known UI-level "extra flags" (--verbose,
// --debug, --output text|json, --dry-run, --mock, --detach) are mapped onto
// typed RuntimeConfig fields so the runner.BuildDetachedArgs spec table owns
// argv shaping. Unknown flags are silently dropped — the form only exposes
// the known set via DefaultFlags().
func launchConfigToOptions(lc LaunchConfig, runID string) config.RuntimeConfig {
	opts := config.RuntimeConfig{
		Pipeline:  lc.PipelineName,
		Input:     lc.Input,
		RunID:     runID,
		Model:     lc.ModelOverride,
		Adapter:   lc.Adapter,
		Timeout:   lc.Timeout,
		FromStep:  lc.FromStep,
		Steps:     lc.Steps,
		Exclude:   lc.Exclude,
		OnFailure: lc.OnFailure,
	}

	// Translate known UI-flag tokens into typed RuntimeConfig fields.
	// "--output X" appears as a single space-joined token in lc.Flags.
	for _, f := range lc.Flags {
		switch {
		case f == "--verbose":
			opts.Output.Verbose = true
		case f == "--debug":
			opts.Output.Debug = true
		case f == "--dry-run":
			opts.DryRun = true
		case f == "--mock":
			opts.Mock = true
		case f == "--detach":
			// Already detaching via runner.Detach — no-op.
		case strings.HasPrefix(f, "--output "):
			opts.Output.Format = strings.TrimPrefix(f, "--output ")
		}
	}
	return opts
}

// manifestEnvPassthrough returns the additional env-variable names declared
// in the manifest's runtime.sandbox.env_passthrough list. The result is
// passed to runner.DetachConfig.ExtraEnv so the runner's BuildDetachEnv
// forwards them alongside its standard set.
func manifestEnvPassthrough(m *manifest.Manifest) []string {
	if m == nil {
		return nil
	}
	if len(m.Runtime.Sandbox.EnvPassthrough) == 0 {
		return nil
	}
	out := make([]string, 0, len(m.Runtime.Sandbox.EnvPassthrough))
	out = append(out, m.Runtime.Sandbox.EnvPassthrough...)
	return out
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
