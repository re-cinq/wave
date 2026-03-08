package tui

import (
	"context"
	"time"
	"fmt"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/workspace"
)

// PipelineLauncher manages pipeline execution from the TUI.
// It constructs executors on demand and tracks cancel functions for running pipelines.
type PipelineLauncher struct {
	deps      LaunchDependencies
	cancelFns map[string]context.CancelFunc
	buffers   map[string]*EventBuffer
	program   *tea.Program
	mu        sync.Mutex
}

// NewPipelineLauncher creates a new launcher with the given dependencies.
func NewPipelineLauncher(deps LaunchDependencies) *PipelineLauncher {
	return &PipelineLauncher{
		deps:      deps,
		cancelFns: make(map[string]context.CancelFunc),
		buffers:   make(map[string]*EventBuffer),
	}
}

// SetProgram sets the Bubble Tea program reference for sending messages.
func (l *PipelineLauncher) SetProgram(p *tea.Program) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.program = p
}

// GetBuffer returns the event buffer for a pipeline run (nil for external pipelines).
func (l *PipelineLauncher) GetBuffer(runID string) *EventBuffer {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buffers[runID]
}

// HasBuffer returns true if the pipeline was TUI-launched and has an event buffer.
func (l *PipelineLauncher) HasBuffer(runID string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, ok := l.buffers[runID]
	return ok
}

// Launch starts a pipeline in a background goroutine and returns tea.Cmds
// for immediate UI feedback (PipelineLaunchedMsg) and eventual completion (PipelineLaunchResultMsg).
func (l *PipelineLauncher) Launch(config LaunchConfig) tea.Cmd {
	// Load the full pipeline definition
	p, err := LoadPipelineByName(l.deps.PipelinesDir, config.PipelineName)
	if err != nil {
		pipelineName := config.PipelineName
		return func() tea.Msg {
			return LaunchErrorMsg{PipelineName: pipelineName, Err: fmt.Errorf("loading pipeline: %w", err)}
		}
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Resolve adapter: check for --mock flag, then use manifest adapters map
	var runner adapter.AdapterRunner
	isMock := false
	for _, f := range config.Flags {
		if f == "--mock" {
			isMock = true
			break
		}
	}
	if isMock {
		runner = adapter.NewMockAdapter()
	} else if l.deps.Manifest != nil && len(l.deps.Manifest.Adapters) > 0 {
		// Pick the first adapter name from the manifest map (mirrors CLI behavior)
		var adapterName string
		for name := range l.deps.Manifest.Adapters {
			adapterName = name
			break
		}
		runner = adapter.ResolveAdapter(adapterName)
	} else {
		runner = adapter.ResolveAdapter("claude")
	}

	// Generate run ID -- prefer StateStore.CreateRun so the run appears in the dashboard
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

	// Store cancel function for later cancellation
	l.mu.Lock()
	l.cancelFns[runID] = cancel
	l.mu.Unlock()

	// Create event buffer for this pipeline
	buffer := NewEventBuffer(1000)
	l.mu.Lock()
	l.buffers[runID] = buffer
	prog := l.program
	l.mu.Unlock()

	// Build emitter — use progress-only emitter for TUI to avoid corrupting stdout
	var emitter event.EventEmitter
	if prog != nil {
		tuiEmitter := &TUIProgressEmitter{program: prog, runID: runID}
		emitter = event.NewProgressOnlyEmitter(tuiEmitter)
	} else {
		emitter = event.NewNDJSONEmitter()
	}

	// Wrap emitter with DB logging so events persist across TUI sessions
	if l.deps.Store != nil {
		emitter = &dbLoggingEmitter{inner: emitter, store: l.deps.Store, runID: runID}
	}

	var execOpts []pipeline.ExecutorOption
	execOpts = append(execOpts, pipeline.WithEmitter(emitter))
	execOpts = append(execOpts, pipeline.WithRunID(runID))

	if l.deps.Store != nil {
		execOpts = append(execOpts, pipeline.WithStateStore(l.deps.Store))
	}

	// Create workspace manager
	wsManager, wsErr := workspace.NewWorkspaceManager(".wave/workspaces")
	if wsErr == nil {
		execOpts = append(execOpts, pipeline.WithWorkspaceManager(wsManager))
	}

	// Apply flags
	isDebug := false
	for _, f := range config.Flags {
		if f == "--debug" {
			isDebug = true
		}
	}
	if isDebug {
		execOpts = append(execOpts, pipeline.WithDebug(true))
	}

	if config.ModelOverride != "" {
		execOpts = append(execOpts, pipeline.WithModelOverride(config.ModelOverride))
	}

	executor := pipeline.NewDefaultPipelineExecutor(runner, execOpts...)

	// Capture values for closures
	pipelineName := config.PipelineName
	input := config.Input
	manifest := l.deps.Manifest
	store := l.deps.Store

	// Return batched commands: immediate launched msg + blocking executor cmd
	immediateCmd := func() tea.Msg {
		return PipelineLaunchedMsg{
			RunID:        runID,
			PipelineName: pipelineName,
			CancelFunc:   cancel,
		}
	}

	executorCmd := func() tea.Msg {
		var execErr error
		if manifest != nil {
			execErr = executor.Execute(ctx, p, manifest, input)
		} else {
			execErr = fmt.Errorf("manifest not available")
		}

		// Update run status in store
		if store != nil {
			status := "completed"
			errMsg := ""
			if execErr != nil {
				status = "failed"
				errMsg = execErr.Error()
			}
			if ctx.Err() != nil {
				status = "cancelled"
				errMsg = ctx.Err().Error()
			}
			_ = store.UpdateRunStatus(runID, status, errMsg, executor.GetTotalTokens())
			_ = store.ClearCancellation(runID)
		}

		// Emit pipeline-level failure event so live output shows the error
		if execErr != nil && prog != nil {
			prog.Send(PipelineEventMsg{
				RunID: runID,
				Event: event.Event{
					Timestamp:     time.Now(),
					PipelineID:    pipelineName,
					State:         event.StateFailed,
					Message:       execErr.Error(),
					FailureReason: "execution",
				},
			})
		}

		return PipelineLaunchResultMsg{RunID: runID, Err: execErr}
	}

	return tea.Batch(immediateCmd, executorCmd)
}

// Cancel cancels a specific running pipeline by run ID.
func (l *PipelineLauncher) Cancel(runID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if cancel, ok := l.cancelFns[runID]; ok {
		cancel()
	}
}


// DismissRun cancels or dismisses a pipeline run.
// For active TUI-launched runs, it calls the in-memory cancel function.
// For stale previous-session runs, it updates the DB status directly.
func (l *PipelineLauncher) DismissRun(runID string) {
	l.mu.Lock()
	cancel, hasCancel := l.cancelFns[runID]
	l.mu.Unlock()

	if hasCancel {
		cancel()
		return
	}

	// Cross-process run (CLI or previous session) — request cancellation via DB.
	// The executor's pollCancellation goroutine will pick this up and cancel the context.
	// If the process is already dead, the stale run status will be cleaned up on next list refresh.
	if l.deps.Store != nil {
		_ = l.deps.Store.RequestCancellation(runID, false)
	}
}
// CancelAll cancels all running pipelines (called on TUI exit).
func (l *PipelineLauncher) CancelAll() {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, cancel := range l.cancelFns {
		cancel()
	}
	l.cancelFns = make(map[string]context.CancelFunc)
}

// Cleanup removes a cancel function entry and buffer after a pipeline finishes.
func (l *PipelineLauncher) Cleanup(runID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.cancelFns, runID)
	delete(l.buffers, runID)
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

// dbLoggingEmitter wraps an EventEmitter and persists each event to the state
// database so event logs survive TUI exit. Mirrors the pattern in cmd/wave/commands/run.go.
type dbLoggingEmitter struct {
	inner event.EventEmitter
	store state.StateStore
	runID string
}

func (d *dbLoggingEmitter) Emit(ev event.Event) {
	d.inner.Emit(ev)
	// Skip empty heartbeat ticks — they carry no useful information.
	if ev.Message == "" && (ev.State == "step_progress" || ev.State == "stream_activity") && ev.TokensUsed == 0 && ev.DurationMs == 0 {
		return
	}
	d.store.LogEvent(d.runID, ev.StepID, ev.State, ev.Persona, ev.Message, ev.TokensUsed, ev.DurationMs)
}
