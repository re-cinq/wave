package commands

// Stage helpers extracted from runRun. Each function corresponds to one
// phase of pipeline execution (flag validation, signal wiring, manifest +
// pipeline loading, state-store init, executor build, single-run vs
// continuous dispatch, recovery formatting, post-run summary). runRun in
// run.go is now a thin orchestrator that calls these in order; behaviour
// is unchanged versus the pre-split version.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/adapter/adaptertest"
	"github.com/recinq/wave/internal/audit"
	"github.com/recinq/wave/internal/continuous"
	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/metrics"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/recovery"
	"github.com/recinq/wave/internal/relay"
	"github.com/recinq/wave/internal/retro"
	"github.com/recinq/wave/internal/runner"
	"github.com/recinq/wave/internal/skill"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/suggest"
	"github.com/recinq/wave/internal/tui"
	"github.com/recinq/wave/internal/workspace"
)

// validateFlags enforces the onboarding gate and CLI-level mutual exclusions
// that must hold before any I/O happens. Each guard returns a CLIError so the
// caller surfaces a structured exit code.
func validateFlags(opts RunOptions) error {
	// Gate on onboarding completion — skip when --force is set
	if !opts.Force {
		if err := checkOnboarding(); err != nil {
			return NewCLIError(CodeOnboardingRequired,
				"onboarding not complete",
				"Run 'wave init' to complete setup before running pipelines")
		}
	}

	// Validate mutual exclusion: --continuous and --from-step cannot be combined
	if opts.Continuous && opts.FromStep != "" {
		return NewCLIError(CodeInvalidArgs,
			"--continuous and --from-step are mutually exclusive",
			"Use --continuous for batch processing or --from-step for resuming a single run")
	}

	// Validate --continuous requires --source
	if opts.Continuous && opts.Source == "" {
		return NewCLIError(CodeInvalidArgs,
			"--continuous requires --source",
			"Specify a source URI, e.g., --source \"github:label=bug\" or --source \"file:queue.txt\"")
	}

	return nil
}

// setupSignalHandling returns a context that is cancelled when the process
// receives an interrupt signal. The returned cancel function should be
// deferred by the caller so the goroutine exits when runRun returns.
func setupSignalHandling() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		cancel()
	}()
	return ctx, cancel
}

// loadManifestAndPipeline loads the manifest, resolves the pipeline (falling
// back to the interactive selector when the named pipeline is missing in TTY
// mode), warns on input/pipeline mismatches, and validates step filter flags.
//
// Mutates opts (and debug) when the user picks a different pipeline through
// the TUI, so both are taken by pointer. Returns aborted=true when the user
// cancelled the selector — the caller should exit cleanly without an error.
func loadManifestAndPipeline(opts *RunOptions, debug *bool) (*pipeline.Pipeline, manifest.Manifest, *pipeline.StepFilter, bool, error) {
	mp, err := loadManifestStrict(opts.Manifest)
	if err != nil {
		return nil, manifest.Manifest{}, nil, false, err
	}
	m := *mp

	p, err := pipeline.LoadByName(opts.Pipeline)
	if err != nil {
		// Pipeline not found — if interactive, try TUI with partial name as filter
		if isInteractive() {
			sel, tuiErr := tui.RunPipelineSelector(pipelinesDir(), opts.Pipeline)
			if tuiErr != nil {
				if errors.Is(tuiErr, huh.ErrUserAborted) {
					return nil, m, nil, true, nil
				}
				return nil, m, nil, false, tuiErr
			}
			applySelection(opts, sel, debug)
			p, err = pipeline.LoadByName(opts.Pipeline)
			if err != nil {
				return nil, m, nil, false, NewCLIError(CodePipelineNotFound,
					fmt.Sprintf("pipeline '%s' not found", opts.Pipeline),
					"Run 'wave list pipelines' to see available pipelines")
			}
		} else {
			return nil, m, nil, false, NewCLIError(CodePipelineNotFound,
				fmt.Sprintf("pipeline '%s' not found", opts.Pipeline),
				"Run 'wave list pipelines' to see available pipelines")
		}
	}

	// Warn on input/pipeline mismatch (non-blocking)
	if opts.Input != "" {
		if mismatch := suggest.CheckInputPipelineMismatch(opts.Input, opts.Pipeline); mismatch != nil {
			fmt.Fprintf(os.Stderr, "  warning: %s\n", mismatch.SuggestedReason)
		}
	}

	// Parse and validate step filter flags
	stepFilter := pipeline.ParseStepFilter(opts.Steps, opts.Exclude)
	if stepFilter != nil {
		if err := stepFilter.Validate(p); err != nil {
			return nil, m, nil, false, err
		}
		if err := stepFilter.ValidateCombinations(opts.FromStep); err != nil {
			return nil, m, nil, false, err
		}
	}

	return p, m, stepFilter, false, nil
}

// buildStateStore opens the SQLite-backed state store under `.agents/state.db`.
// State persistence is best-effort: a failure to open the DB downgrades the
// run to in-memory operation with a warning, returning nil so callers can
// nil-check without separate error plumbing.
func buildStateStore() state.StateStore {
	store, err := state.NewStateStore(".agents/state.db")
	if err != nil {
		// Non-fatal: continue without state persistence
		fmt.Fprintf(os.Stderr, "warning: state persistence disabled: %v\n", err)
		return nil
	}
	return store
}

// autoRecoverResumeInput rehydrates opts.Input when --from-step is used
// without an explicit --input by reading from the state store. Resume needs
// the original input so the executor can replay deterministic prompts; this
// is best-effort and silently leaves opts.Input empty if no record is found.
func autoRecoverResumeInput(opts *RunOptions, store state.StateStore, p *pipeline.Pipeline) {
	if opts.FromStep == "" || opts.Input != "" || store == nil {
		return
	}
	if opts.RunID != "" {
		if run, err := store.GetRun(opts.RunID); err == nil && run.Input != "" {
			opts.Input = run.Input
			fmt.Fprintf(os.Stderr, "  Resuming with input from run %s: %s\n", opts.RunID, truncateString(opts.Input, 80))
		}
		return
	}
	runs, err := store.ListRuns(state.ListRunsOptions{
		PipelineName: p.Metadata.Name,
		Limit:        1,
	})
	if err == nil && len(runs) > 0 && runs[0].Input != "" {
		opts.Input = runs[0].Input
		fmt.Fprintf(os.Stderr, "  Resuming with input from previous run: %s\n", truncateString(opts.Input, 80))
	}
}

// resolveOrGenerateRunID picks the run ID for this execution. When --run is
// set (detach subprocess or TUI launch) it is reused verbatim; otherwise the
// state store mints a new ID via CreateRun so the run shows up in the
// dashboard. Falls back to GenerateRunID when the store is unavailable.
func resolveOrGenerateRunID(opts RunOptions, store state.StateStore, p *pipeline.Pipeline, m *manifest.Manifest) string {
	runID, resolveIDErr := resolveRunID(opts.RunID, store, p.Metadata.Name, opts.Input)
	if resolveIDErr != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to create run record: %v\n", resolveIDErr)
	}
	if runID == "" {
		runID = pipeline.GenerateRunID(p.Metadata.Name, m.Runtime.PipelineIDHashLength)
	}
	return runID
}

// runResources bundles every component the runRun execution path needs after
// the executor is wired (executor, emitter, adapter runner, workspace root,
// raw execOpts so continuous mode can clone them per iteration). Close runs
// the cleanups for emitter, audit logger, and debug tracer in reverse order
// so the caller can defer a single Close() instead of tracking each.
type runResources struct {
	emitter     event.EventEmitter
	emitterAux  *EmitterResult
	wsRoot      string
	runner      adapter.AdapterRunner
	execOpts    []pipeline.ExecutorOption
	// Foreground carries the resolved ForegroundConfig for runOnce. The
	// executor is built lazily inside runner.LaunchForeground rather than
	// here so the CLI no longer reaches into pipeline.NewDefaultPipelineExecutor
	// directly — both webui and CLI funnel through the same launcher.
	foreground  runner.ForegroundConfig
	closeFns    []func()
	cleanupOnce bool
}

// Close runs all registered cleanup functions in LIFO order. Safe to call
// multiple times — the second call is a no-op so post-run logic in runRun
// can stop the TUI eagerly without breaking the deferred Close.
func (r *runResources) Close() {
	if r == nil || r.cleanupOnce {
		return
	}
	r.cleanupOnce = true
	for i := len(r.closeFns) - 1; i >= 0; i-- {
		r.closeFns[i]()
	}
}

// resolveAdapterRunner picks between the mock adapter and the real claude
// binary. Kept as a tiny helper so the buildExecutor body stays focused on
// option assembly rather than adapter selection details.
func resolveAdapterRunner(mock bool) adapter.AdapterRunner {
	if mock {
		return adaptertest.NewMockAdapter(
			adaptertest.WithSimulatedDelay(5 * time.Second),
		)
	}
	return adapter.ResolveAdapter("claude")
}

// buildExecutor wires the workspace manager, audit logger, debug tracer,
// adapter registry, skill store, retro generator, and relay monitor into a
// fresh DefaultPipelineExecutor. The returned runResources owns every
// resource that needs cleanup; callers must defer Close().
//
// Behaviour mirrors the original inline block exactly — option ordering,
// nil-checks, and warning text are preserved so this refactor stays a
// no-op at runtime.
func buildExecutor(opts RunOptions, m *manifest.Manifest, p *pipeline.Pipeline, store state.StateStore, stepFilter *pipeline.StepFilter, runID string, debug bool) (*runResources, error) {
	res := &runResources{}

	res.runner = resolveAdapterRunner(opts.Mock)

	// Initialize event emitter based on output format
	emitterResult := CreateEmitter(opts.Output, runID, p.Metadata.Name, p.Steps, m)
	res.emitterAux = &emitterResult
	res.closeFns = append(res.closeFns, emitterResult.Cleanup)

	// Wrap with DB logging so "wave logs <run-id>" returns full history for CLI runs.
	var emitter event.EventEmitter = emitterResult.Emitter
	if store != nil {
		emitter = &event.DBLoggingEmitter{Inner: emitterResult.Emitter, Store: store, RunID: runID}
	}
	res.emitter = emitter

	// Initialize workspace manager under .agents/workspaces
	wsRoot := m.Runtime.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = ".agents/workspaces"
	}
	res.wsRoot = wsRoot
	wsManager, err := workspace.NewWorkspaceManager(wsRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace manager: %w", err)
	}

	// Initialize audit logger under .agents/traces/
	var logger audit.AuditLogger
	if m.Runtime.Audit.LogAllToolCalls {
		traceDir := m.Runtime.Audit.LogDir
		if traceDir == "" {
			traceDir = ".agents/traces"
		}
		if l, err := audit.NewTraceLoggerWithDir(traceDir); err == nil {
			logger = l
			res.closeFns = append(res.closeFns, func() { _ = l.Close() })
		}
	}

	// Initialize debug tracer when --debug is set
	var debugTracer *audit.DebugTracer
	if debug {
		traceDir := m.Runtime.Audit.LogDir
		if traceDir == "" {
			traceDir = ".agents/traces"
		}
		if dt, dtErr := audit.NewDebugTracer(traceDir, runID, audit.WithStderrMirror(true)); dtErr == nil {
			debugTracer = dt
			res.closeFns = append(res.closeFns, func() { _ = dt.Close() })
			fmt.Fprintf(os.Stderr, "  Debug trace: %s\n", dt.TracePath())
		} else {
			fmt.Fprintf(os.Stderr, "warning: failed to create debug tracer: %v\n", dtErr)
		}
		// Enable debug verbosity on the emitter for richer NDJSON output
		emitterResult.Emitter.SetDebugVerbosity(true)
	}

	execOpts := assembleExecutorOptions(opts, m, store, stepFilter, runID, debug, emitter, wsManager, logger, debugTracer, res.runner)
	res.execOpts = execOpts

	skillStore := skill.NewDirectoryStore(skill.DefaultSources()...)

	var retroGen *retro.Generator
	if store != nil && !opts.NoRetro {
		mstore := metrics.NewStore(state.UnderlyingDB(store))
		retroGen = retro.NewGenerator(store, mstore, res.runner, ".agents/retros", &m.Runtime.Retros)
	}

	var relayMon *relay.RelayMonitor
	if m.Runtime.Relay.TokenThresholdPercent > 0 {
		relayCfg := relay.RelayMonitorConfig{
			DefaultThreshold:   m.Runtime.Relay.TokenThresholdPercent,
			MinTokensToCompact: 1000,
			ContextWindow:      m.Runtime.Relay.ContextWindow,
			CompactionTimeout:  m.Runtime.Timeouts.GetRelayCompaction(),
		}
		registry := adapter.NewAdapterRegistry(nil)
		for name, a := range m.Adapters {
			if a.Binary != "" {
				registry.SetBinary(name, a.Binary)
			}
		}
		compactionAdapter := relay.NewAdapterCompactionRunner(registry, m)
		relayMon = relay.NewRelayMonitor(relayCfg, compactionAdapter)
	}

	res.foreground = runner.ForegroundConfig{
		RunID:            runID,
		Pipeline:         p,
		Manifest:         m,
		Store:            store,
		Emitter:          emitter,
		WorkspaceManager: wsManager,
		AuditLogger:      logger,
		DebugTracer:      debugTracer,
		Runner:           res.runner,
		MockOverride:     opts.Mock,
		RetroGenerator:   retroGen,
		RelayMonitor:     relayMon,
		SkillStore:       skillStore,
		StepFilter:       stepFilter,
		Runtime:          opts,
		Force:            opts.Force,
		Debug:            debug,
		// runOnce sets Input + FromStep before calling LaunchForeground.
		OnExecutorReady: func(executor *pipeline.DefaultPipelineExecutor) {
			if btpd, ok := emitterResult.Progress.(*display.BubbleTeaProgressDisplay); ok {
				btpd.SetOutcomeTracker(executor.GetOutcomeTracker())
			}
		},
	}

	return res, nil
}

// assembleExecutorOptions builds the ExecutorOption slice. Delegates to
// runner.BuildExecutorOptions so both the CLI's foreground path and the
// webui's in-process launcher share one source of truth — the previous
// duplication is what allowed LaunchInProcess to silently miss
// pipeline.WithRegistry (manifest adapter binaries).
func assembleExecutorOptions(opts RunOptions, m *manifest.Manifest, store state.StateStore, stepFilter *pipeline.StepFilter, runID string, debug bool, emitter event.EventEmitter, wsManager workspace.WorkspaceManager, logger audit.AuditLogger, debugTracer *audit.DebugTracer, adapterRunner adapter.AdapterRunner) []pipeline.ExecutorOption {
	skillStore := skill.NewDirectoryStore(skill.DefaultSources()...)

	var retroGen *retro.Generator
	if store != nil && !opts.NoRetro {
		mstore := metrics.NewStore(state.UnderlyingDB(store))
		retroGen = retro.NewGenerator(store, mstore, adapterRunner, ".agents/retros", &m.Runtime.Retros)
	}

	var relayMon *relay.RelayMonitor
	if m.Runtime.Relay.TokenThresholdPercent > 0 {
		relayCfg := relay.RelayMonitorConfig{
			DefaultThreshold:   m.Runtime.Relay.TokenThresholdPercent,
			MinTokensToCompact: 1000,
			ContextWindow:      m.Runtime.Relay.ContextWindow,
			CompactionTimeout:  m.Runtime.Timeouts.GetRelayCompaction(),
		}
		// Compaction routes through a registry seeded from the same manifest;
		// runner.BuildExecutorOptions wires the same shape internally.
		registry := adapter.NewAdapterRegistry(nil)
		for name, a := range m.Adapters {
			if a.Binary != "" {
				registry.SetBinary(name, a.Binary)
			}
		}
		compactionAdapter := relay.NewAdapterCompactionRunner(registry, m)
		relayMon = relay.NewRelayMonitor(relayCfg, compactionAdapter)
	}

	return runner.BuildExecutorOptions(runner.ExecutorBuildConfig{
		RunID:            runID,
		Manifest:         m,
		Store:            store,
		Emitter:          emitter,
		WorkspaceManager: wsManager,
		AuditLogger:      logger,
		DebugTracer:      debugTracer,
		Runtime:          opts,
		Runner:           adapterRunner,
		MockOverride:     opts.Mock,
		RetroGenerator:   retroGen,
		RelayMonitor:     relayMon,
		SkillStore:       skillStore,
		StepFilter:       stepFilter,
		Debug:            debug,
	})
}

// runOnce executes the pipeline a single time. It transitions the run from
// pending → running, spawns the heartbeat goroutine, dispatches to either
// Execute or ResumeWithValidation depending on --from-step, and records the
// terminal status (cancelled/failed/completed) so the dashboard converges
// even when the caller bails on the returned error.
// runOnce executes the pipeline a single time via runner.LaunchForeground.
// LaunchForeground owns the running/heartbeat/terminal status transitions
// (cancelled/rejected/failed/completed) so the CLI no longer duplicates them.
// Returns the executor (for printSummary) alongside the exec error.
func runOnce(ctx context.Context, res *runResources, opts RunOptions) (*pipeline.DefaultPipelineExecutor, error) {
	cfg := res.foreground
	cfg.Input = opts.Input
	cfg.FromStep = opts.FromStep
	// --run with --from-step: resume but resolve artifacts from the original
	// run's workspace tree. The new resume run row stays under cfg.RunID.
	if opts.FromStep != "" && opts.RunID != "" {
		cfg.PriorRunID = opts.RunID
	}
	result := runner.LaunchForeground(ctx, cfg)
	return result.Executor, result.ExecErr
}

// runContinuous drives the --continuous batch loop. Each work item from the
// configured source spawns a fresh executor that clones execOpts and pins a
// new run ID, so iterations stay independent in the dashboard. Returns
// non-nil when the loop itself fails or any iteration fails (with a count).
func runContinuous(ctx context.Context, opts RunOptions, m *manifest.Manifest, p *pipeline.Pipeline, store state.StateStore, runner adapter.AdapterRunner, emitter event.EventEmitter, execOpts []pipeline.ExecutorOption) error {
	// Parse source URI
	srcCfg, err := continuous.ParseSourceURI(opts.Source)
	if err != nil {
		return fmt.Errorf("invalid --source: %w", err)
	}
	src, err := continuous.NewSourceFromConfig(srcCfg)
	if err != nil {
		return fmt.Errorf("failed to create source: %w", err)
	}

	// Parse delay
	delay, err := time.ParseDuration(opts.Delay)
	if err != nil {
		return fmt.Errorf("invalid --delay %q: %w", opts.Delay, err)
	}

	contRunner := &continuous.Runner{
		Source:        src,
		PipelineName:  p.Metadata.Name,
		OnFailure:     continuous.ParseFailurePolicy(opts.OnFailure),
		MaxIterations: opts.MaxIterations,
		Delay:         delay,
		Emitter:       emitter,
		ExecutorFactory: func(input string) continuous.ExecutorFunc {
			return func(execCtx context.Context, execInput string) (string, error) {
				// Create a new run ID for each iteration
				var iterRunID string
				if store != nil {
					iterRunID, _ = store.CreateRun(p.Metadata.Name, execInput)
				}
				if iterRunID == "" {
					iterRunID = pipeline.GenerateRunID(p.Metadata.Name, m.Runtime.PipelineIDHashLength)
				}

				// Create a fresh executor for this iteration
				iterOpts := make([]pipeline.ExecutorOption, len(execOpts))
				copy(iterOpts, execOpts)
				iterOpts = append(iterOpts, pipeline.WithRunID(iterRunID))

				iterExecutor := pipeline.NewDefaultPipelineExecutor(runner, iterOpts...)
				execErr := iterExecutor.Execute(execCtx, p, m, execInput)

				// Update run status
				if store != nil {
					tokens := iterExecutor.GetTotalTokens()
					if execErr != nil {
						if updateErr := store.UpdateRunStatus(iterRunID, "failed", execErr.Error(), tokens); updateErr != nil {
							fmt.Fprintf(os.Stderr, "warning: failed to update run status: %v\n", updateErr)
						}
					} else {
						if updateErr := store.UpdateRunStatus(iterRunID, "completed", "", tokens); updateErr != nil {
							fmt.Fprintf(os.Stderr, "warning: failed to update run status: %v\n", updateErr)
						}
					}
				}

				return iterRunID, execErr
			}
		},
	}

	summary, contErr := contRunner.Run(ctx)
	if contErr != nil {
		return fmt.Errorf("continuous run failed: %w", contErr)
	}

	// Print summary
	if opts.Output.Format == OutputFormatAuto || opts.Output.Format == OutputFormatText {
		fmt.Fprintf(os.Stderr, "\n  %s\n", summary.String())
	}

	// Exit code 1 if any failures
	if summary.HasFailures() {
		return fmt.Errorf("continuous run completed with %d failures", summary.Failed)
	}
	return nil
}

// formatRecoveryError classifies execErr, builds a recovery hint block, and
// either emits structured hints (JSON mode) or appends them to the wrapped
// error string (text/auto/quiet mode). Always returns a non-nil error so
// callers can `return formatRecoveryError(...)` directly.
func formatRecoveryError(execErr error, opts RunOptions, p *pipeline.Pipeline, runID, wsRoot string, emitter event.EventEmitter) error {
	// Extract step ID from StepExecutionError when available; fall back gracefully
	// so recovery hints are shown for all failure paths (including resume).
	var (
		stepErr *pipeline.StepExecutionError
		stepID  string
		cause   = execErr
	)
	if errors.As(execErr, &stepErr) {
		stepID = stepErr.StepID
		cause = stepErr.Err
	}

	errClass := recovery.ClassifyError(cause)

	// Extract preflight metadata when the error is a preflight failure
	var preflightMeta *recovery.PreflightMetadata
	if errClass == recovery.ClassPreflight {
		preflightMeta = recovery.ExtractPreflightMetadata(cause)
	}

	block := recovery.BuildRecoveryBlock(recovery.RecoveryBlockOpts{
		PipelineName:  p.Metadata.Name,
		Input:         opts.Input,
		StepID:        stepID,
		RunID:         runID,
		WorkspaceRoot: wsRoot,
		ErrClass:      errClass,
		PreflightMeta: preflightMeta,
	})

	if opts.Output.Format == OutputFormatJSON {
		// In JSON mode, emit recovery hints as structured data.
		// The executor already emits a bare "failed" event; this enriched
		// event carries the hints so consumers only need one event.
		hints := make([]event.RecoveryHintJSON, len(block.Hints))
		for i, h := range block.Hints {
			hints[i] = event.RecoveryHintJSON{
				Label:   h.Label,
				Command: h.Command,
				Type:    string(h.Type),
			}
		}
		emitter.Emit(event.Event{
			Timestamp:     time.Now(),
			PipelineID:    runID,
			StepID:        stepID,
			State:         "recovery",
			Message:       execErr.Error(),
			RecoveryHints: hints,
		})
	} else {
		// In text/auto/quiet modes, append recovery hints after the error
		// line by embedding them in the returned error message.
		hintBlock := recovery.FormatRecoveryBlock(block)
		if hintBlock != "" {
			return fmt.Errorf("pipeline execution failed: %w\n%s", execErr, hintBlock)
		}
	}
	return fmt.Errorf("pipeline execution failed: %w", execErr)
}

// printSummary writes the post-run human or JSON outcome surface. Text/auto
// modes print the green "completed" banner plus an indented outcome
// breakdown to stderr; JSON mode emits a final "completed" event with the
// structured outcomes payload. Quiet mode is intentionally silent.
// printRejectionSummary renders the post-run banner for a design-rejection
// terminal state. It is the rejection sibling of printSummary — the run was
// halted because a contract with `on_failure: rejected` fired (e.g. an issue
// assessment returned `implementable: false`). The banner uses a yellow
// "rejected" colour rather than red, the CLI exits 0, and JSON consumers
// receive a `state: "rejected"` event so dashboards can route it to the
// dedicated rejected status.
//
// Banner sample (text mode):
//
//	! Pipeline 'inception-bugfix' rejected (1.5s) — no implementable issue
//	  step "fetch-assess": value must be true at /implementable
func printRejectionSummary(opts RunOptions, p *pipeline.Pipeline, rejectionErr *pipeline.ContractRejectionError, elapsed time.Duration, emitter event.EventEmitter, runID string) {
	if opts.Output.Format == OutputFormatJSON {
		emitter.Emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: runID,
			StepID:     rejectionErr.StepID,
			State:      "rejected",
			DurationMs: elapsed.Milliseconds(),
			Message:    rejectionErr.Error(),
		})
		return
	}
	if opts.Output.Format != OutputFormatAuto && opts.Output.Format != OutputFormatText {
		return
	}
	// Yellow ANSI bang glyph keeps the run visually distinct from a green
	// completion (✓) or red failure (✕). The colour code is the same one
	// `display.FormatStateBadge` uses for warning-class signals.
	const yellow = "\033[33m"
	const reset = "\033[0m"
	fmt.Fprintf(os.Stderr, "\n  %s!%s Pipeline '%s' rejected (%.1fs) — no implementable result\n",
		yellow, reset, p.Metadata.Name, elapsed.Seconds())
	if rejectionErr.StepID != "" {
		fmt.Fprintf(os.Stderr, "    step %q: %s\n", rejectionErr.StepID, rejectionErr.Reason)
	} else if rejectionErr.Reason != "" {
		fmt.Fprintf(os.Stderr, "    %s\n", rejectionErr.Reason)
	}
	fmt.Fprintf(os.Stderr, "    This is not a runtime failure — the pipeline declared the work non-actionable by design.\n\n")
}

func printSummary(opts RunOptions, executor *pipeline.DefaultPipelineExecutor, p *pipeline.Pipeline, runID string, elapsed time.Duration, emitter event.EventEmitter) {
	// Show human summary only in auto/text modes — json and quiet stay clean
	if opts.Output.Format == OutputFormatAuto || opts.Output.Format == OutputFormatText {
		totalTokens := executor.GetTotalTokens()
		if totalTokens > 0 {
			fmt.Fprintf(os.Stderr, "\n  ✓ Pipeline '%s' completed successfully (%.1fs, %s tokens)\n",
				p.Metadata.Name, elapsed.Seconds(), display.FormatTokenCount(totalTokens))
		} else {
			fmt.Fprintf(os.Stderr, "\n  ✓ Pipeline '%s' completed successfully (%.1fs)\n",
				p.Metadata.Name, elapsed.Seconds())
		}
		// Build structured outcome summary from outcome tracker
		tracker := executor.GetOutcomeTracker()
		outcome := display.BuildOutcome(tracker, p.Metadata.Name, runID, true, elapsed, totalTokens, "", nil)
		summary := display.RenderOutcomeSummary(outcome, opts.Output.Verbose, display.NewFormatter())
		if summary != "" {
			fmt.Fprint(os.Stderr, "\n")
			lines := strings.Split(summary, "\n")
			for _, line := range lines {
				if line != "" {
					fmt.Fprintf(os.Stderr, "  %s\n", line)
				} else {
					fmt.Fprint(os.Stderr, "\n")
				}
			}
			fmt.Fprint(os.Stderr, "\n")
		}
	}

	// For JSON output mode, emit structured outcomes in the final completion event
	if opts.Output.Format == OutputFormatJSON {
		tracker := executor.GetOutcomeTracker()
		outcome := display.BuildOutcome(tracker, p.Metadata.Name, runID, true, elapsed, executor.GetTotalTokens(), "", nil)
		outJSON := outcome.ToOutcomesJSON()
		emitter.Emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: runID,
			State:      "completed",
			DurationMs: elapsed.Milliseconds(),
			Message:    fmt.Sprintf("Pipeline '%s' completed", p.Metadata.Name),
			Outcomes:   outJSON,
		})
	}
}
