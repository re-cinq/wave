package runner

import (
	"context"
	"errors"
	"log"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/audit"
	"github.com/recinq/wave/internal/config"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/relay"
	"github.com/recinq/wave/internal/retro"
	"github.com/recinq/wave/internal/skill"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/workspace"
)

// ForegroundConfig collects everything needed to drive a synchronous,
// foreground pipeline run. Mirrors InProcessConfig but adds the CLI extras
// (debug tracer, retro generator, relay monitor, mock-override flag) and
// expects a pre-loaded *pipeline.Pipeline + *manifest.Manifest because the
// CLI has already validated and parsed those before calling.
type ForegroundConfig struct {
	RunID    string
	Pipeline *pipeline.Pipeline
	Manifest *manifest.Manifest
	Input    string

	Store            state.StateStore
	Emitter          event.EventEmitter
	WorkspaceManager workspace.WorkspaceManager
	AuditLogger      audit.AuditLogger
	DebugTracer      *audit.DebugTracer
	GateHandler      pipeline.GateHandler

	Runner       adapter.AdapterRunner
	MockOverride bool

	RetroGenerator *retro.Generator
	RelayMonitor   *relay.RelayMonitor
	SkillStore     skill.Store
	StepFilter     *pipeline.StepFilter

	// Runtime carries merged CLI flags (model/adapter/timeout/preserve/etc.).
	Runtime config.RuntimeConfig

	// FromStep, when non-empty, calls ResumeWithValidation instead of Execute.
	FromStep string
	// Force is paired with FromStep — skips resume validation.
	Force bool
	// PriorRunID, when non-empty, scopes resume artifact lookup to the
	// original run's workspace tree. The resume run itself still records
	// state under cfg.RunID; PriorRunID only affects artifact resolution
	// inside ResumeWithValidation. Ignored unless FromStep is set.
	PriorRunID string

	// Debug toggles WithDebug on the executor.
	Debug bool

	// SkipStatusUpdates leaves UpdateRunStatus calls to the caller. Both
	// the CLI and the webui leave this false (default) so LaunchForeground
	// owns the running → cancelled/rejected/failed/completed dispatch in
	// one place. Set true only when the caller already drives a status
	// state machine that would conflict with LaunchForeground writing the
	// same rows.
	SkipStatusUpdates bool

	// OnExecutorReady fires after the executor is built but before Execute
	// runs. The CLI uses this to attach the BubbleTea progress display to
	// the executor's outcome tracker. May be nil.
	OnExecutorReady func(*pipeline.DefaultPipelineExecutor)
}

// ForegroundResult exposes post-run state the CLI uses to format banners and
// summaries. Executor is returned because callers pull metrics (cost, outcome,
// progress) off it before discarding.
type ForegroundResult struct {
	Executor *pipeline.DefaultPipelineExecutor
	Tokens   int
	ExecErr  error
}

// LaunchForeground drives a pipeline synchronously in the calling goroutine.
// It builds the executor via BuildExecutorOptions (so behaviour matches
// LaunchInProcess), records pending → running → terminal transitions on the
// state store unless SkipStatusUpdates is set, and returns once Execute (or
// ResumeWithValidation, when FromStep is non-empty) finishes.
//
// The CLI calls this from its run command. Cancellation is the caller's ctx.
func LaunchForeground(ctx context.Context, cfg ForegroundConfig) *ForegroundResult {
	execOpts := BuildExecutorOptions(ExecutorBuildConfig{
		RunID:            cfg.RunID,
		Manifest:         cfg.Manifest,
		Store:            cfg.Store,
		Emitter:          cfg.Emitter,
		WorkspaceManager: cfg.WorkspaceManager,
		GateHandler:      cfg.GateHandler,
		AuditLogger:      cfg.AuditLogger,
		DebugTracer:      cfg.DebugTracer,
		Runtime:          cfg.Runtime,
		Runner:           cfg.Runner,
		MockOverride:     cfg.MockOverride,
		RetroGenerator:   cfg.RetroGenerator,
		RelayMonitor:     cfg.RelayMonitor,
		SkillStore:       cfg.SkillStore,
		StepFilter:       cfg.StepFilter,
		Debug:            cfg.Debug,
	})

	executor := pipeline.NewDefaultPipelineExecutor(cfg.Runner, execOpts...)
	if cfg.OnExecutorReady != nil {
		cfg.OnExecutorReady(executor)
	}

	if !cfg.SkipStatusUpdates && cfg.Store != nil {
		if err := cfg.Store.UpdateRunStatus(cfg.RunID, "running", "", 0); err != nil {
			log.Printf("Warning: failed to update run %s to running: %v", cfg.RunID, err)
		}
		_ = cfg.Store.UpdateRunHeartbeat(cfg.RunID)
		// Heartbeat goroutine: lets the reconciler distinguish a live run from
		// a parent process that died without updating the DB.
		hbCtx, hbCancel := context.WithCancel(context.Background())
		defer hbCancel()
		go state.RunHeartbeatLoop(hbCtx, cfg.Store, cfg.RunID)
	}

	p := cfg.Pipeline
	if p == nil {
		p = &pipeline.Pipeline{}
	}
	m := cfg.Manifest
	if m == nil {
		m = &manifest.Manifest{}
	}

	var execErr error
	if cfg.FromStep != "" {
		if cfg.PriorRunID != "" {
			execErr = executor.ResumeWithValidation(ctx, p, m, cfg.Input, cfg.FromStep, cfg.Force, cfg.PriorRunID)
		} else {
			execErr = executor.ResumeWithValidation(ctx, p, m, cfg.Input, cfg.FromStep, cfg.Force)
		}
	} else {
		execErr = executor.Execute(ctx, p, m, cfg.Input)
	}

	tokens := executor.GetTotalTokens()

	if !cfg.SkipStatusUpdates && cfg.Store != nil {
		var rejectionErr *pipeline.ContractRejectionError
		switch {
		case ctx.Err() != nil:
			_ = cfg.Store.UpdateRunStatus(cfg.RunID, "cancelled", "pipeline cancelled", tokens)
			_ = cfg.Store.ClearCancellation(cfg.RunID)
		case execErr != nil && errors.As(execErr, &rejectionErr):
			// Design rejection: persona deliberately reported the work is
			// non-actionable. Legitimate terminal verdict, not a runtime
			// failure.
			_ = cfg.Store.UpdateRunStatus(cfg.RunID, "rejected", execErr.Error(), tokens)
		case execErr != nil:
			_ = cfg.Store.UpdateRunStatus(cfg.RunID, "failed", execErr.Error(), tokens)
		default:
			_ = cfg.Store.UpdateRunStatus(cfg.RunID, "completed", "", tokens)
		}
	}

	return &ForegroundResult{
		Executor: executor,
		Tokens:   tokens,
		ExecErr:  execErr,
	}
}
