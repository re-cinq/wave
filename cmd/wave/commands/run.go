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
	"github.com/recinq/wave/internal/audit"
	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/preflight"
	"github.com/recinq/wave/internal/recovery"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/tui"
	"github.com/recinq/wave/internal/workspace"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

type RunOptions struct {
	Pipeline          string
	Input             string
	DryRun            bool
	FromStep          string
	Force             bool
	Timeout           int
	Manifest          string
	Mock              bool
	RunID             string
	Output            OutputConfig
	Model             string
	PreserveWorkspace bool
	Steps             string // Comma-separated step names to include (--steps)
	Exclude           string // Comma-separated step names to exclude (-x/--exclude)
}

func NewRunCmd() *cobra.Command {
	var opts RunOptions

	cmd := &cobra.Command{
		Use:   "run [pipeline] [input]",
		Short: "Run a pipeline",
		Long: `Execute a pipeline from the wave manifest.
Supports dry-run mode, step resumption, custom timeouts, and model override.

The --model flag overrides the adapter model for all steps in the run,
including any per-persona model pinning in wave.yaml.

Arguments can be provided as positional args or flags:
  wave run pr-review "Review auth module"
  wave run --pipeline pr-review --input "Review auth module"
  wave run pr-review --input "Review auth module"`,
		Example: `  wave run pr-review "Review the authentication changes"
  wave run --pipeline plan-speckit --input "add user auth"
  wave run impl-hotfix --dry-run
  wave run migrate --from-step validate
  wave run my-pipeline --model haiku
  wave run my-pipeline --preserve-workspace
  wave run --steps clarify,plan plan-speckit
  wave run -x implement,create-pr plan-speckit
  wave run --from-step clarify -x create-pr plan-speckit`,
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

			// If no pipeline specified and stdin is a TTY, launch interactive selector
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
	cmd.Flags().StringVar(&opts.Model, "model", "", "Override adapter model for this run (e.g. haiku, opus)")
	cmd.Flags().BoolVar(&opts.PreserveWorkspace, "preserve-workspace", false, "Preserve workspace from previous run (for debugging)")
	cmd.Flags().StringVar(&opts.Steps, "steps", "", "Run only the named steps (comma-separated)")
	cmd.Flags().StringVarP(&opts.Exclude, "exclude", "x", "", "Skip the named steps (comma-separated)")

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		cancel()
	}()

	manifestData, err := os.ReadFile(opts.Manifest)
	if err != nil {
		return NewCLIError(CodeManifestMissing,
			fmt.Sprintf("manifest file not found: %s", opts.Manifest),
			"Run 'wave init' to create a manifest")
	}

	var m manifest.Manifest
	if err := yaml.Unmarshal(manifestData, &m); err != nil {
		return NewCLIError(CodeManifestInvalid,
			fmt.Sprintf("failed to parse manifest: %s", err),
			"Check wave.yaml syntax — run 'wave validate' to diagnose")
	}

	p, err := loadPipeline(opts.Pipeline, &m)
	if err != nil {
		// Check if this is a deprecated forge-prefixed name
		if resolved, deprecated := pipeline.ResolveDeprecatedName(opts.Pipeline); deprecated {
			if pp, resolveErr := loadPipeline(resolved, &m); resolveErr == nil {
				fmt.Fprintf(os.Stderr, "⚠ Pipeline %q is deprecated. Use %q instead.\n", opts.Pipeline, resolved)
				p = pp
				err = nil
				opts.Pipeline = resolved
			}
		}
	}
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
			p, err = loadPipeline(opts.Pipeline, &m)
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
		performDryRun(p, &m, stepFilter)
		return nil
	}

	// Resolve adapter — use mock if --mock or if no adapter binary found
	var runner adapter.AdapterRunner
	if opts.Mock {
		// Add simulated delay to see progress animations in action
		runner = adapter.NewMockAdapter(
			adapter.WithSimulatedDelay(5 * time.Second),
		)
	} else {
		var adapterName string
		for name := range m.Adapters {
			adapterName = name
			break
		}
		runner = adapter.ResolveAdapter(adapterName)
	}

	// Initialize state store under .wave/ — must happen before run ID generation
	// so we can use CreateRun() to produce IDs visible to the dashboard.
	stateDB := ".wave/state.db"
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

	// Generate run ID — use pre-created ID from TUI subprocess (--run flag without --from-step),
	// or prefer CreateRun() so CLI runs appear in the dashboard.
	// Falls back to GenerateRunID() if the state store is unavailable.
	var runID string
	if opts.RunID != "" && opts.FromStep == "" {
		// TUI subprocess passes a pre-created run ID — reuse it instead of creating a new one
		runID = opts.RunID
	} else if store != nil {
		runID, err = store.CreateRun(p.Metadata.Name, opts.Input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to create run record: %v\n", err)
		}
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
		emitter = &dbLoggingEmitter{inner: result.Emitter, store: store, runID: runID}
	}

	// Initialize workspace manager under .wave/workspaces
	wsRoot := m.Runtime.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = ".wave/workspaces"
	}
	wsManager, err := workspace.NewWorkspaceManager(wsRoot)
	if err != nil {
		return fmt.Errorf("failed to create workspace manager: %w", err)
	}

	// Initialize audit logger under .wave/traces/
	var logger audit.AuditLogger
	if m.Runtime.Audit.LogAllToolCalls {
		traceDir := m.Runtime.Audit.LogDir
		if traceDir == "" {
			traceDir = ".wave/traces"
		}
		if l, err := audit.NewTraceLoggerWithDir(traceDir); err == nil {
			logger = l
			defer l.Close()
		}
	}

	// Build executor with all components
	execOpts := []pipeline.ExecutorOption{
		pipeline.WithEmitter(emitter),
		pipeline.WithDebug(debug),
		pipeline.WithRunID(runID),
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
	if opts.PreserveWorkspace {
		execOpts = append(execOpts, pipeline.WithPreserveWorkspace(true))
	}
	if stepFilter != nil {
		execOpts = append(execOpts, pipeline.WithStepFilter(stepFilter))
	}

	executor := pipeline.NewDefaultPipelineExecutor(runner, execOpts...)

	// Connect deliverable tracker to progress display
	if btpd, ok := progressDisplay.(*display.BubbleTeaProgressDisplay); ok {
		btpd.SetDeliverableTracker(executor.GetDeliverableTracker())
	}

	pipelineStart := time.Now()

	// Transition run from pending → running so dashboards and wave status reflect active execution.
	if store != nil {
		store.UpdateRunStatus(runID, "running", "", 0)
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
		if ctx.Err() != nil {
			store.UpdateRunStatus(runID, "cancelled", "pipeline cancelled", tokens)
			store.ClearCancellation(runID)
		} else if execErr != nil {
			store.UpdateRunStatus(runID, "failed", execErr.Error(), tokens)
		} else {
			store.UpdateRunStatus(runID, "completed", "", tokens)
		}
	}

	if execErr != nil {
		// Extract step ID from StepError when available; fall back gracefully
		// so recovery hints are shown for all failure paths (including resume).
		var (
			stepErr *pipeline.StepError
			stepID  string
			cause   error = execErr
		)
		if errors.As(execErr, &stepErr) {
			stepID = stepErr.StepID
			cause = stepErr.Err
		}

		errClass := recovery.ClassifyError(cause)

		// Extract preflight metadata when the error is a preflight failure
		var preflightMeta *recovery.PreflightMetadata
		if errClass == recovery.ClassPreflight {
			preflightMeta = extractPreflightMetadata(cause)
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
		// Build structured outcome summary from deliverable tracker
		tracker := executor.GetDeliverableTracker()
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
		tracker := executor.GetDeliverableTracker()
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

func loadPipeline(name string, m *manifest.Manifest) (*pipeline.Pipeline, error) {
	candidates := []string{
		".wave/pipelines/" + name + ".yaml",
		".wave/pipelines/" + name,
		name,
	}

	var pipelinePath string
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			pipelinePath = candidate
			break
		}
	}

	if pipelinePath == "" {
		return nil, fmt.Errorf("pipeline '%s' not found (searched .wave/pipelines/)", name)
	}

	pipelineData, err := os.ReadFile(pipelinePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pipeline file: %w", err)
	}

	var p pipeline.Pipeline
	if err := yaml.Unmarshal(pipelineData, &p); err != nil {
		return nil, fmt.Errorf("failed to parse pipeline: %w", err)
	}

	return &p, nil
}

// isInteractive returns true when stdin is a TTY and interactive selection is possible.
func isInteractive() bool {
	if v := os.Getenv("WAVE_FORCE_TTY"); v != "" {
		return v == "1" || v == "true"
	}
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// pipelinesDir returns the default pipeline directory.
func pipelinesDir() string {
	return ".wave/pipelines"
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

func performDryRun(p *pipeline.Pipeline, m *manifest.Manifest, filter *pipeline.StepFilter) {
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
		fmt.Fprintf(os.Stderr, "  %d. %s (persona: %s)%s\n", i+1, step.ID, step.Persona, status)

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

		fmt.Fprintf(os.Stderr, "     Workspace: .wave/workspaces/%s/%s/\n", p.Metadata.Name, step.ID)

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

}

// extractPreflightMetadata extracts missing skills and tools from preflight errors.
// It walks the error chain using errors.As to find SkillError or ToolError types.
func extractPreflightMetadata(err error) *recovery.PreflightMetadata {
	if err == nil {
		return nil
	}

	meta := &recovery.PreflightMetadata{}

	var skillErr *preflight.SkillError
	if errors.As(err, &skillErr) {
		meta.MissingSkills = skillErr.MissingSkills
	}

	var toolErr *preflight.ToolError
	if errors.As(err, &toolErr) {
		meta.MissingTools = toolErr.MissingTools
	}

	// Return nil if no metadata was found
	if len(meta.MissingSkills) == 0 && len(meta.MissingTools) == 0 {
		return nil
	}

	return meta
}

// dbLoggingEmitter wraps an EventEmitter and also persists each event to the
// state database so that "wave logs <run-id>" returns a complete history for
// CLI-launched runs (mirrors the loggingEmitter used by the WebUI server).
type dbLoggingEmitter struct {
	inner event.EventEmitter
	store state.StateStore
	runID string
}

func (d *dbLoggingEmitter) Emit(ev event.Event) {
	d.inner.Emit(ev)
	// Skip empty heartbeat ticks — they carry no useful information.
	// Keep stream_activity events that carry ToolName (Claude Code tool calls).
	if ev.Message == "" && ev.ToolName == "" && (ev.State == "step_progress" || ev.State == "stream_activity") && ev.TokensUsed == 0 && ev.DurationMs == 0 {
		return
	}
	// Compose message from ToolName+ToolTarget for stream_activity persistence.
	msg := ev.Message
	if msg == "" && ev.ToolName != "" {
		msg = ev.ToolName + " " + ev.ToolTarget
	}
	d.store.LogEvent(d.runID, ev.StepID, ev.State, ev.Persona, msg, ev.TokensUsed, ev.DurationMs)
}
