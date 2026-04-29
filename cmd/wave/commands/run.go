package commands

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
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/recovery"
	"github.com/recinq/wave/internal/relay"
	"github.com/recinq/wave/internal/retro"
	"github.com/recinq/wave/internal/runner"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/skill"
	"github.com/recinq/wave/internal/suggest"
	"github.com/recinq/wave/internal/tui"
	"github.com/recinq/wave/internal/workspace"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// RunOptions is aliased from internal/runner — the canonical struct lives
// there so the webui launch path consumes the exact same fields without a
// translation layer. The exhaustiveness test (TestDetachedArgsExhaustive)
// also lives in internal/runner alongside the spec table it guards.
type RunOptions = runner.Options

func NewRunCmd() *cobra.Command {
	var opts RunOptions

	cmd := &cobra.Command{
		Use:   "run [pipeline] [input]",
		Short: "Run a pipeline",
		Long: `Execute a pipeline from the wave manifest.
Supports dry-run mode, step resumption, custom timeouts, model override,
and detached execution (--detach) for background runs that survive shell exit.

The --model flag overrides the adapter model for all steps in the run,
including any per-persona model pinning in wave.yaml.

The --adapter flag selects the LLM backend (claude, opencode, gemini, codex).
Model formats vary by adapter: claude uses "haiku"/"opus", opencode uses
"provider/model", gemini uses "gemini-2.0-pro", codex uses "gpt-4o".`,
		Example: `  wave run ops-pr-review "Review the authentication changes"
  wave run --pipeline impl-speckit --input "add user auth"
  wave run impl-issue --dry-run
  wave run migrate --from-step validate
  wave run my-pipeline --model haiku
  wave run my-pipeline --adapter opencode --model openai/gpt-4o
  wave run my-pipeline --preserve-workspace
  wave run --steps clarify,plan impl-speckit
  wave run -x implement,create-pr impl-speckit
  wave run --from-step clarify -x create-pr impl-speckit
  wave run --detach impl-issue "fix login bug"         # detach: run in background`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Handle positional arguments
			if len(args) >= 1 && opts.Pipeline == "" {
				opts.Pipeline = args[0]
			}
			if len(args) >= 2 && opts.Input == "" {
				opts.Input = args[1]
			}

			opts.Output = GetOutputConfig(cmd)
			debug, _ := cmd.Flags().GetBool("debug")

			// Smart input routing: when only one positional arg is given and
			// it doesn't look like a pipeline name, treat it as input and
			// auto-suggest a pipeline.
			if opts.Pipeline != "" && opts.Input == "" && len(args) == 1 {
				inputType := suggest.ClassifyInput(opts.Pipeline)
				if inputType != suggest.InputTypeFreeText {
					// The "pipeline" arg is actually input — reclassify
					opts.Input = opts.Pipeline
					opts.Pipeline = ""
				}
			}

			// If no pipeline specified, try smart routing from input type
			if opts.Pipeline == "" && opts.Input != "" {
				suggested := suggestPipelineFromInput(opts.Input)
				if suggested != "" {
					if isInteractive() {
						sel, err := tui.RunPipelineSelector(pipelinesDir(), suggested)
						if err != nil {
							if errors.Is(err, huh.ErrUserAborted) {
								return nil
							}
							return err
						}
						applySelection(&opts, sel, &debug)
					} else {
						// Non-interactive: auto-select the first suggestion
						opts.Pipeline = suggested
						inputType := suggest.ClassifyInput(opts.Input)
						fmt.Fprintf(os.Stderr, "  Auto-selected pipeline %q for %s input\n", suggested, inputType)
					}
				}
			}

			// If still no pipeline, fall back to interactive selector or error
			if opts.Pipeline == "" {
				if isInteractive() {
					sel, err := tui.RunPipelineSelector(pipelinesDir(), "")
					if err != nil {
						if errors.Is(err, huh.ErrUserAborted) {
							return nil
						}
						return err
					}
					applySelection(&opts, sel, &debug)
				} else {
					return fmt.Errorf("pipeline name is required (use positional arg or --pipeline flag)")
				}
			}

			if err := ValidateOutputFormat(opts.Output.Format); err != nil {
				return err
			}

			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return runRun(opts, debug)
		},
	}

	cmd.Flags().StringVar(&opts.Pipeline, "pipeline", "", "Pipeline name to run")
	cmd.Flags().StringVar(&opts.Input, "input", "", "Input data for the pipeline")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Show what would be executed without running")
	cmd.Flags().StringVar(&opts.FromStep, "from-step", "", "Start execution from specific step")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Skip validation checks when using --from-step")
	cmd.Flags().IntVar(&opts.Timeout, "timeout", 0, "Timeout in minutes (overrides manifest)")
	cmd.Flags().StringVar(&opts.Manifest, "manifest", "wave.yaml", "Path to manifest file")
	cmd.Flags().BoolVar(&opts.Mock, "mock", false, "Use mock adapter (for testing)")
	cmd.Flags().StringVar(&opts.RunID, "run", "", "Resume from a specific run (uses that run's input)")
	cmd.Flags().StringVar(&opts.Model, "model", "", "Model for this run — tier name (cheapest/balanced/strongest) or literal (haiku/opus). Takes the cheaper of CLI and step tiers unless --force-model is set.")
	cmd.Flags().BoolVar(&opts.ForceModel, "force-model", false, "Force --model on all steps, ignoring per-step and per-persona model tiers")
	cmd.Flags().StringVar(&opts.Adapter, "adapter", "", "Override adapter for this run (e.g. claude, gemini, opencode, codex)")
	cmd.Flags().BoolVar(&opts.PreserveWorkspace, "preserve-workspace", false, "Preserve workspace from previous run (for debugging)")
	cmd.Flags().StringVar(&opts.Steps, "steps", "", "Run only the named steps (comma-separated)")
	cmd.Flags().StringVarP(&opts.Exclude, "exclude", "x", "", "Skip the named steps (comma-separated)")
	cmd.Flags().BoolVar(&opts.Continuous, "continuous", false, "Run pipeline in continuous mode, iterating over work items from --source")
	cmd.Flags().StringVar(&opts.Source, "source", "", "Work item source URI (e.g., github:label=bug, file:queue.txt)")
	cmd.Flags().IntVar(&opts.MaxIterations, "max-iterations", 0, "Maximum number of iterations (0 = unlimited)")
	cmd.Flags().StringVar(&opts.Delay, "delay", "0s", "Delay between iterations (e.g., 5s, 1m)")
	cmd.Flags().StringVar(&opts.OnFailure, "on-failure", "halt", "Failure policy: halt (default) or skip")
	cmd.Flags().BoolVar(&opts.Detach, "detach", false, "Run pipeline as a detached background process")
	cmd.Flags().BoolVar(&opts.AutoApprove, "auto-approve", false, "Auto-approve all approval gates using default choices (required for --detach with gates)")
	cmd.Flags().BoolVar(&opts.NoRetro, "no-retro", false, "Skip retrospective generation for this run")

	// Group flags by tier for organized --help output
	essentialFlags := []string{"pipeline", "input", "model", "adapter"}
	executionFlags := []string{"from-step", "force", "dry-run", "timeout", "steps", "exclude", "on-failure", "detach"}
	continuousFlags := []string{"continuous", "source", "max-iterations", "delay"}
	devDebugFlags := []string{"mock", "preserve-workspace", "auto-approve", "no-retro", "force-model", "run", "manifest"}

	cmd.SetUsageFunc(func(c *cobra.Command) error {
		fmt.Fprintf(c.OutOrStderr(), "Usage:\n  %s\n\n", c.UseLine())

		printFlagGroup := func(title string, names []string) {
			fmt.Fprintf(c.OutOrStderr(), "%s:\n", title)
			for _, name := range names {
				f := c.Flags().Lookup(name)
				if f == nil {
					continue
				}
				shorthand := ""
				if f.Shorthand != "" {
					shorthand = fmt.Sprintf("-%s, ", f.Shorthand)
				}
				defVal := ""
				if f.DefValue != "" && f.DefValue != "false" && f.DefValue != "0" {
					defVal = fmt.Sprintf(" (default %s)", f.DefValue)
				}
				fmt.Fprintf(c.OutOrStderr(), "      %s--%s %s%s\n", shorthand, f.Name, f.Usage, defVal)
			}
			fmt.Fprintln(c.OutOrStderr())
		}

		printFlagGroup("Essential", essentialFlags)
		printFlagGroup("Execution", executionFlags)
		printFlagGroup("Continuous", continuousFlags)
		printFlagGroup("Dev/Debug", devDebugFlags)

		// Print inherited persistent flags so parent flags (--verbose, --debug, etc.) appear
		parentFlags := c.InheritedFlags()
		if parentFlags.HasFlags() {
			fmt.Fprintf(c.OutOrStderr(), "Global Flags:\n")
			fmt.Fprintln(c.OutOrStderr(), parentFlags.FlagUsages())
		}

		return nil
	})

	return cmd
}

func runRun(opts RunOptions, debug bool) error {
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		cancel()
	}()

	mp, err := loadManifestStrict(opts.Manifest)
	if err != nil {
		return err
	}
	m := *mp

	p, err := pipeline.LoadByName(opts.Pipeline)
	if err != nil {
		// Pipeline not found — if interactive, try TUI with partial name as filter
		if isInteractive() {
			sel, tuiErr := tui.RunPipelineSelector(pipelinesDir(), opts.Pipeline)
			if tuiErr != nil {
				if errors.Is(tuiErr, huh.ErrUserAborted) {
					return nil
				}
				return tuiErr
			}
			applySelection(&opts, sel, &debug)
			p, err = pipeline.LoadByName(opts.Pipeline)
			if err != nil {
				return NewCLIError(CodePipelineNotFound,
					fmt.Sprintf("pipeline '%s' not found", opts.Pipeline),
					"Run 'wave list pipelines' to see available pipelines")
			}
		} else {
			return NewCLIError(CodePipelineNotFound,
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
			return err
		}
		if err := stepFilter.ValidateCombinations(opts.FromStep); err != nil {
			return err
		}
	}

	if opts.DryRun {
		return performDryRun(p, &m, stepFilter)
	}

	// Detached mode: re-exec ourselves as a detached subprocess and return immediately.
	// This reuses the same pattern as the TUI's pipeline_launcher.go.
	if opts.Detach {
		// Validate: if pipeline has approval gates with choices, --auto-approve is required
		if !opts.AutoApprove && p.HasApprovalGates() {
			return NewCLIError(CodeInvalidArgs,
				"--detach with approval gates requires --auto-approve",
				"Add --auto-approve to auto-approve gates in detached mode, or remove --detach for interactive execution")
		}
		return runDetached(opts, p, &m)
	}

	// Resolve adapter — use mock if --mock or if no adapter binary found
	var runner adapter.AdapterRunner
	if opts.Mock {
		runner = adaptertest.NewMockAdapter(
			adaptertest.WithSimulatedDelay(5 * time.Second),
		)
	} else {
		runner = adapter.ResolveAdapter("claude")
	}

	// Initialize state store under .agents/ — must happen before run ID generation
	// so we can use CreateRun() to produce IDs visible to the dashboard.
	stateDB := ".agents/state.db"
	store, err := state.NewStateStore(stateDB)
	if err != nil {
		// Non-fatal: continue without state persistence
		fmt.Fprintf(os.Stderr, "warning: state persistence disabled: %v\n", err)
		store = nil
	}
	if store != nil {
		defer store.Close()
	}

	// Auto-recover input when resuming without explicit --input
	if opts.FromStep != "" && opts.Input == "" && store != nil {
		if opts.RunID != "" {
			if run, err := store.GetRun(opts.RunID); err == nil && run.Input != "" {
				opts.Input = run.Input
				fmt.Fprintf(os.Stderr, "  Resuming with input from run %s: %s\n", opts.RunID, truncateString(opts.Input, 80))
			}
		} else {
			runs, err := store.ListRuns(state.ListRunsOptions{
				PipelineName: p.Metadata.Name,
				Limit:        1,
			})
			if err == nil && len(runs) > 0 && runs[0].Input != "" {
				opts.Input = runs[0].Input
				fmt.Fprintf(os.Stderr, "  Resuming with input from previous run: %s\n", truncateString(opts.Input, 80))
			}
		}
	}

	// Generate run ID — use pre-created ID when --run is set (covers both the detach
	// subprocess path and TUI subprocesses, regardless of whether --from-step is also set),
	// or prefer CreateRun() so CLI runs appear in the dashboard.
	// Falls back to GenerateRunID() if the state store is unavailable.
	runID, resolveIDErr := resolveRunID(opts.RunID, store, p.Metadata.Name, opts.Input)
	if resolveIDErr != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to create run record: %v\n", resolveIDErr)
	}
	if runID == "" {
		runID = pipeline.GenerateRunID(p.Metadata.Name, m.Runtime.PipelineIDHashLength)
	}

	// Initialize event emitter based on output format
	result := CreateEmitter(opts.Output, runID, p.Metadata.Name, p.Steps, &m)
	progressDisplay := result.Progress
	defer result.Cleanup()
	// Wrap with DB logging so "wave logs <run-id>" returns full history for CLI runs.
	var emitter event.EventEmitter = result.Emitter
	if store != nil {
		emitter = &event.DBLoggingEmitter{Inner: result.Emitter, Store: store, RunID: runID}
	}

	// Initialize workspace manager under .agents/workspaces
	wsRoot := m.Runtime.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = ".agents/workspaces"
	}
	wsManager, err := workspace.NewWorkspaceManager(wsRoot)
	if err != nil {
		return fmt.Errorf("failed to create workspace manager: %w", err)
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
			defer l.Close()
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
			defer dt.Close()
			fmt.Fprintf(os.Stderr, "  Debug trace: %s\n", dt.TracePath())
		} else {
			fmt.Fprintf(os.Stderr, "warning: failed to create debug tracer: %v\n", dtErr)
		}

		// Enable debug verbosity on the emitter for richer NDJSON output
		result.Emitter.SetDebugVerbosity(true)
	}

	// Build executor with all components
	execOpts := []pipeline.ExecutorOption{
		pipeline.WithEmitter(emitter),
		pipeline.WithDebug(debug),
		pipeline.WithRunID(runID),
	}
	if debugTracer != nil {
		execOpts = append(execOpts, pipeline.WithDebugTracer(debugTracer))
	}
	if wsManager != nil {
		execOpts = append(execOpts, pipeline.WithWorkspaceManager(wsManager))
	}
	if store != nil {
		execOpts = append(execOpts, pipeline.WithStateStore(store))
	}
	if logger != nil {
		execOpts = append(execOpts, pipeline.WithAuditLogger(logger))
	}
	if opts.Timeout > 0 {
		execOpts = append(execOpts, pipeline.WithStepTimeout(time.Duration(opts.Timeout)*time.Minute))
	}
	if opts.Model != "" {
		execOpts = append(execOpts, pipeline.WithModelOverride(opts.Model))
	}
	if opts.ForceModel {
		execOpts = append(execOpts, pipeline.WithForceModel(true))
	}
	registry := adapter.NewAdapterRegistry(nil)
	for name, a := range m.Adapters {
		if a.Binary != "" {
			registry.SetBinary(name, a.Binary)
		}
	}
	if opts.Mock {
		// Route every adapter declared in the manifest through the mock runner.
		// "mock" itself is always registered so pipelines that pin adapter: mock
		// resolve correctly even when the manifest does not enumerate it.
		registry.RegisterOverride("mock", runner)
		for name := range m.Adapters {
			registry.RegisterOverride(name, runner)
		}
	}
	execOpts = append(execOpts, pipeline.WithRegistry(registry))

	// Wire skill store so declared skills are provisioned into adapter workspaces.
	// Source ordering (project skills/ wins over installed .agents/skills/) is
	// owned by the skill package — see skill.DefaultSources.
	skillStore := skill.NewDirectoryStore(skill.DefaultSources()...)
	execOpts = append(execOpts, pipeline.WithSkillStore(skillStore))
	if opts.Adapter != "" {
		execOpts = append(execOpts, pipeline.WithAdapterOverride(opts.Adapter))
	}
	if opts.PreserveWorkspace {
		execOpts = append(execOpts, pipeline.WithPreserveWorkspace(true))
	}
	if stepFilter != nil {
		execOpts = append(execOpts, pipeline.WithStepFilter(stepFilter))
	}
	if opts.AutoApprove {
		execOpts = append(execOpts, pipeline.WithAutoApprove(true))
	}
	if store != nil && !opts.NoRetro {
		retroGen := retro.NewGenerator(store, runner, ".agents/retros", &m.Runtime.Retros)
		execOpts = append(execOpts, pipeline.WithRetroGenerator(retroGen))
	}

	// Wire relay context compaction — prevents long-running steps from exhausting
	// the Claude context window by summarizing conversation at a token threshold.
	if m.Runtime.Relay.TokenThresholdPercent > 0 {
		relayCfg := relay.RelayMonitorConfig{
			DefaultThreshold:   m.Runtime.Relay.TokenThresholdPercent,
			MinTokensToCompact: 1000,
			ContextWindow:      m.Runtime.Relay.ContextWindow,
			CompactionTimeout:  m.Runtime.Timeouts.GetRelayCompaction(),
		}
		compactionAdapter := relay.NewAdapterCompactionRunner(registry, &m)
		relayMon := relay.NewRelayMonitor(relayCfg, compactionAdapter)
		execOpts = append(execOpts, pipeline.WithRelayMonitor(relayMon))
	}

	executor := pipeline.NewDefaultPipelineExecutor(runner, execOpts...)

	// Connect outcome tracker to progress display
	if btpd, ok := progressDisplay.(*display.BubbleTeaProgressDisplay); ok {
		btpd.SetOutcomeTracker(executor.GetOutcomeTracker())
	}

	if opts.Continuous {
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
					execErr := iterExecutor.Execute(execCtx, p, &m, execInput)

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

	pipelineStart := time.Now()

	// Transition run from pending → running so dashboards and wave status reflect active execution.
	if store != nil {
		_ = store.UpdateRunStatus(runID, "running", "", 0)
		_ = store.UpdateRunHeartbeat(runID)
		// Periodic heartbeat: lets the reconciler distinguish a live run from
		// a parent process that died without updating the DB. Goroutine exits
		// when heartbeatCancel fires (deferred just below this block).
		heartbeatCtx, heartbeatCancel := context.WithCancel(context.Background())
		defer heartbeatCancel()
		go state.RunHeartbeatLoop(heartbeatCtx, store, runID)
	}

	var execErr error
	if opts.FromStep != "" {
		// Resume from specific step - uses ResumeWithValidation which handles artifacts.
		// When --run is specified, pass the run ID so artifact paths resolve from
		// that specific run's workspace instead of scanning for the most recent match.
		if opts.RunID != "" {
			execErr = executor.ResumeWithValidation(ctx, p, &m, opts.Input, opts.FromStep, opts.Force, opts.RunID)
		} else {
			execErr = executor.ResumeWithValidation(ctx, p, &m, opts.Input, opts.FromStep, opts.Force)
		}
	} else {
		execErr = executor.Execute(ctx, p, &m, opts.Input)
	}

	// Update the pipeline_run record so the dashboard reflects final status
	if store != nil {
		tokens := executor.GetTotalTokens()
		switch {
		case ctx.Err() != nil:
			_ = store.UpdateRunStatus(runID, "cancelled", "pipeline cancelled", tokens)
			_ = store.ClearCancellation(runID)
		case execErr != nil:
			_ = store.UpdateRunStatus(runID, "failed", execErr.Error(), tokens)
		default:
			_ = store.UpdateRunStatus(runID, "completed", "", tokens)
		}
	}

	if execErr != nil {
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

	elapsed := time.Since(pipelineStart)

	// Stop the TUI before printing post-run output to avoid terminal corruption.
	// Cleanup is idempotent so the deferred call above becomes a no-op.
	result.Cleanup()

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

	return nil
}

// runDetached spawns a new `wave run` subprocess that is fully detached from
// the current process session via internal/runner. The subprocess inherits
// all flags except --detach and runs the pipeline in its own session group.
// runner.Detach is the single source of truth used by both the CLI path and
// the webui server, so changes to the spawn protocol live in exactly one
// place (and are exercised by TestDetachedArgsExhaustive).
func runDetached(opts RunOptions, p *pipeline.Pipeline, m *manifest.Manifest) error {
	stateDB := ".agents/state.db"
	store, err := state.NewStateStore(stateDB)
	if err != nil {
		return fmt.Errorf("detach requires state store: %w", err)
	}
	defer store.Close()

	maxWorkers := 5
	if m != nil && m.Runtime.MaxConcurrentWorkers > 0 {
		maxWorkers = m.Runtime.MaxConcurrentWorkers
	}

	// runner.Detach defaults Pipeline name onto opts; ensure it is set so
	// CreateRunWithLimit records the right pipeline_name.
	if opts.Pipeline == "" {
		opts.Pipeline = p.Metadata.Name
	}

	runID, err := runner.Detach(opts, store, maxWorkers, runner.DetachConfig{})
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "  Pipeline '%s' launched (detached)\n", p.Metadata.Name)
	fmt.Fprintf(os.Stderr, "  Run ID:  %s\n", runID)
	fmt.Fprintf(os.Stderr, "  Logs:    wave logs %s\n", runID)
	fmt.Fprintf(os.Stderr, "  Cancel:  wave cancel %s\n", runID)
	return nil
}

// resolveRunID selects or creates the run ID for a pipeline execution.
// When runIDOpt is non-empty (set via --run by the --detach subprocess or TUI),
// it is always reused regardless of whether --from-step is also set — preventing
// a second CreateRun call and the phantom run records reported in issue #700.
// When a state store is available and no run ID was pre-created, CreateRun is
// called so the run is visible in the dashboard.
// Returns ("", nil) when neither source yields an ID; the caller should then
// fall back to GenerateRunID.
func resolveRunID(runIDOpt string, store state.StateStore, pipelineName, input string) (string, error) {
	if runIDOpt != "" {
		return runIDOpt, nil
	}
	if store != nil {
		return store.CreateRun(pipelineName, input)
	}
	return "", nil
}

// isInteractive returns true when stdin is a TTY and interactive selection is possible.
func isInteractive() bool {
	if v := os.Getenv("WAVE_FORCE_TTY"); v != "" {
		return v == "1" || v == "true"
	}
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// suggestPipelineFromInput classifies the input and returns the best pipeline
// suggestion. Returns empty string if no suggestion is available.
func suggestPipelineFromInput(input string) string {
	inputType := suggest.ClassifyInput(input)
	suggestions := suggest.SuggestPipelineForInput(inputType)
	if len(suggestions) == 0 {
		return ""
	}
	return suggestions[0]
}

// pipelinesDir returns the default pipeline directory.
func pipelinesDir() string {
	return ".agents/pipelines"
}

// applySelection maps a TUI selection back to RunOptions.
func applySelection(opts *RunOptions, sel *tui.Selection, debug *bool) {
	opts.Pipeline = sel.Pipeline
	if sel.Input != "" {
		opts.Input = sel.Input
	}
	for _, flag := range sel.Flags {
		switch flag {
		case "--verbose":
			opts.Output.Verbose = true
		case "--output json":
			opts.Output.Format = OutputFormatJSON
		case "--output text":
			opts.Output.Format = OutputFormatText
		case "--dry-run":
			opts.DryRun = true
		case "--mock":
			opts.Mock = true
		case "--debug":
			*debug = true
		}
	}
}

func performDryRun(p *pipeline.Pipeline, m *manifest.Manifest, filter *pipeline.StepFilter) error {
	fmt.Fprintf(os.Stderr, "Dry run for pipeline: %s\n", p.Metadata.Name)
	fmt.Fprintf(os.Stderr, "Description: %s\n", p.Metadata.Description)
	fmt.Fprintf(os.Stderr, "Steps: %d\n\n", len(p.Steps))
	fmt.Fprintf(os.Stderr, "Execution plan:\n")

	for i, step := range p.Steps {
		// Show [SKIP] or [RUN] status when a filter is active
		status := ""
		if filter != nil && filter.IsActive() {
			if filter.ShouldRun(step.ID) {
				status = " [RUN]"
			} else {
				status = " [SKIP]"
			}
		}
		if step.SubPipeline != "" {
			fmt.Fprintf(os.Stderr, "  %d. %s (pipeline: %s)%s\n", i+1, step.ID, step.SubPipeline, status)
		} else {
			fmt.Fprintf(os.Stderr, "  %d. %s (persona: %s)%s\n", i+1, step.ID, step.Persona, status)
		}

		if len(step.Dependencies) > 0 {
			fmt.Fprintf(os.Stderr, "     Dependencies: %v\n", step.Dependencies)
		}

		persona := m.GetPersona(step.Persona)
		if persona != nil {
			fmt.Fprintf(os.Stderr, "     Adapter: %s  Temp: %.1f\n", persona.Adapter, persona.Temperature)
			fmt.Fprintf(os.Stderr, "     System prompt: %s\n", persona.SystemPromptFile)
			if len(persona.Permissions.AllowedTools) > 0 {
				fmt.Fprintf(os.Stderr, "     Allowed tools: %v\n", persona.Permissions.AllowedTools)
			}
			if len(persona.Permissions.Deny) > 0 {
				fmt.Fprintf(os.Stderr, "     Denied tools: %v\n", persona.Permissions.Deny)
			}
		}

		if len(step.Workspace.Mount) > 0 {
			for _, mount := range step.Workspace.Mount {
				fmt.Fprintf(os.Stderr, "     Mount: %s → %s (%s)\n", mount.Source, mount.Target, mount.Mode)
			}
		}

		fmt.Fprintf(os.Stderr, "     Workspace: .agents/workspaces/%s/%s/\n", p.Metadata.Name, step.ID)

		if step.Memory.Strategy != "" {
			fmt.Fprintf(os.Stderr, "     Memory: %s\n", step.Memory.Strategy)
		}

		if len(step.Memory.InjectArtifacts) > 0 {
			for _, art := range step.Memory.InjectArtifacts {
				fmt.Fprintf(os.Stderr, "     Inject: %s:%s as %s\n", art.Step, art.Artifact, art.As)
			}
		}

		if len(step.OutputArtifacts) > 0 {
			for _, art := range step.OutputArtifacts {
				fmt.Fprintf(os.Stderr, "     Output: %s → %s (%s)\n", art.Name, art.Path, art.Type)
			}
		}

		if step.Handover.Contract.Type != "" {
			fmt.Fprintf(os.Stderr, "     Contract: %s", step.Handover.Contract.Type)
			if step.Handover.Contract.OnFailure != "" {
				fmt.Fprintf(os.Stderr, " (on_failure: %s, max_retries: %d)", step.Handover.Contract.OnFailure, step.Handover.Contract.MaxRetries)
			}
			fmt.Fprintln(os.Stderr)
		}

		fmt.Fprintln(os.Stderr)
	}

	// Show artifact warnings when a filter is active
	if filter != nil && filter.IsActive() {
		skippedSteps := make(map[string]bool)
		for _, step := range p.Steps {
			if !filter.ShouldRun(step.ID) {
				skippedSteps[step.ID] = true
			}
		}
		var warnings []string
		for _, step := range p.Steps {
			if !filter.ShouldRun(step.ID) {
				continue
			}
			for _, dep := range step.Dependencies {
				if skippedSteps[dep] {
					warnings = append(warnings, fmt.Sprintf("  ⚠ Step %q depends on skipped step %q — ensure prior artifacts exist", step.ID, dep))
				}
			}
		}
		if len(warnings) > 0 {
			fmt.Fprintln(os.Stderr, "Artifact warnings:")
			for _, w := range warnings {
				fmt.Fprintln(os.Stderr, w)
			}
			fmt.Fprintln(os.Stderr)
		}
	}

	// Run composition validation and report findings.
	validator := pipeline.NewDryRunValidator(pipelinesDir())
	report := validator.Validate(p, m)
	fmt.Fprint(os.Stderr, "\n")
	fmt.Fprint(os.Stderr, report.Format())

	if report.HasErrors() {
		return fmt.Errorf("dry-run validation found %d error(s) — pipeline is not safe to execute", report.ErrorCount())
	}
	return nil
}

