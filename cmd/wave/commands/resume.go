package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/audit"
	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/recovery"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/workspace"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// ResumeOptions holds options for the resume command.
type ResumeOptions struct {
	RunID    string
	FromStep string
	Force    bool
	Model    string
	Manifest string
	Mock     bool
	Output   OutputConfig
}

// NewResumeCmd creates the resume command.
func NewResumeCmd() *cobra.Command {
	var opts ResumeOptions

	cmd := &cobra.Command{
		Use:   "resume <run-id>",
		Short: "Resume a failed pipeline run",
		Long: `Resume a previously failed or interrupted pipeline run.

Without --from-step, wave resume auto-detects the failed step from
the run's event log and resumes from there.

With --from-step, execution resumes from the specified step regardless
of which step failed.

The --force flag skips phase-sequence and stale-artifact validation,
matching the behaviour of 'wave run --force'.`,
		Example: `  wave resume impl-speckit-20240315-abc123
  wave resume impl-speckit-20240315-abc123 --from-step implement
  wave resume impl-speckit-20240315-abc123 --from-step plan --force
  wave resume impl-speckit-20240315-abc123 --model opus`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RunID = args[0]
			opts.Output = GetOutputConfig(cmd)
			debug, _ := cmd.Flags().GetBool("debug")

			if err := ValidateOutputFormat(opts.Output.Format); err != nil {
				return err
			}

			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return runResume(opts, debug)
		},
	}

	cmd.Flags().StringVar(&opts.FromStep, "from-step", "", "Resume from this specific step (default: auto-detect failed step)")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Skip validation checks (phase sequence, stale artifacts)")
	cmd.Flags().StringVar(&opts.Model, "model", "", "Override adapter model for this run (e.g. haiku, opus)")
	cmd.Flags().StringVar(&opts.Manifest, "manifest", "wave.yaml", "Path to manifest file")
	cmd.Flags().BoolVar(&opts.Mock, "mock", false, "Use mock adapter (for testing)")

	return cmd
}

func runResume(opts ResumeOptions, debug bool) error {
	// Gate on onboarding completion — skip when --force is set.
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

	// Open state store — required so we can look up the prior run.
	stateDB := ".wave/state.db"
	store, err := state.NewStateStore(stateDB)
	if err != nil {
		return NewCLIError(CodeInvalidArgs,
			fmt.Sprintf("failed to open state database: %v", err),
			"Run 'wave init' to set up the project, or check that .wave/state.db exists")
	}
	defer store.Close()

	// Look up the run record.
	run, err := store.GetRun(opts.RunID)
	if err != nil {
		return NewCLIError(CodeInvalidArgs,
			fmt.Sprintf("run %q not found", opts.RunID),
			"Use 'wave list runs' to see available run IDs")
	}

	// Refuse to resume a run that completed successfully.
	if run.Status == "completed" {
		return NewCLIError(CodeInvalidArgs,
			fmt.Sprintf("run %q already completed successfully", opts.RunID),
			"Nothing to resume — start a fresh run with 'wave run "+run.PipelineName+"'")
	}

	// If no --from-step was specified, auto-detect the failed step.
	fromStep := opts.FromStep
	if fromStep == "" {
		fromStep, err = detectFailedStep(store, run)
		if err != nil {
			return NewCLIError(CodeInvalidArgs,
				fmt.Sprintf("could not determine which step to resume from: %v", err),
				"Specify the step explicitly with --from-step <step>")
		}
		if opts.Output.Format != OutputFormatJSON {
			fmt.Fprintf(os.Stderr, "  Auto-detected failed step: %s\n", fromStep)
		}
	}

	// Load manifest.
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

	// Load the pipeline that was used in the original run.
	p, err := loadPipeline(run.PipelineName, &m)
	if err != nil {
		return NewCLIError(CodePipelineNotFound,
			fmt.Sprintf("pipeline %q not found (needed by run %s)", run.PipelineName, opts.RunID),
			"Run 'wave list pipelines' to see available pipelines")
	}

	// Resolve adapter.
	var runner adapter.AdapterRunner
	if opts.Mock {
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

	// Create a new run record for this resume execution so it appears in the
	// dashboard alongside the original run.
	resumeRunID, err := store.CreateRun(run.PipelineName, run.Input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to create resume run record: %v\n", err)
		resumeRunID = pipeline.GenerateRunID(run.PipelineName, m.Runtime.PipelineIDHashLength)
	}

	// Show what we are resuming.
	if opts.Output.Format == OutputFormatAuto || opts.Output.Format == OutputFormatText {
		fmt.Fprintf(os.Stderr, "\n  Resuming run %s\n", opts.RunID)
		fmt.Fprintf(os.Stderr, "  Pipeline: %s\n", run.PipelineName)
		fmt.Fprintf(os.Stderr, "  From step: %s\n", fromStep)
		if run.Input != "" {
			fmt.Fprintf(os.Stderr, "  Input: %s\n\n", truncateString(run.Input, 80))
		} else {
			fmt.Fprintln(os.Stderr)
		}
	}

	// Initialize event emitter.
	result := CreateEmitter(opts.Output, resumeRunID, p.Metadata.Name, p.Steps, &m)
	progressDisplay := result.Progress
	defer result.Cleanup()

	var emitter event.EventEmitter = result.Emitter
	emitter = &dbLoggingEmitter{inner: emitter, store: store, runID: resumeRunID}

	// Initialize workspace manager.
	wsRoot := m.Runtime.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = ".wave/workspaces"
	}
	wsManager, err := workspace.NewWorkspaceManager(wsRoot)
	if err != nil {
		return fmt.Errorf("failed to create workspace manager: %w", err)
	}

	// Initialize audit logger.
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

	// Build executor.
	execOpts := []pipeline.ExecutorOption{
		pipeline.WithEmitter(emitter),
		pipeline.WithDebug(debug),
		pipeline.WithRunID(resumeRunID),
	}
	if wsManager != nil {
		execOpts = append(execOpts, pipeline.WithWorkspaceManager(wsManager))
	}
	execOpts = append(execOpts, pipeline.WithStateStore(store))
	if logger != nil {
		execOpts = append(execOpts, pipeline.WithAuditLogger(logger))
	}
	if opts.Model != "" {
		execOpts = append(execOpts, pipeline.WithModelOverride(opts.Model))
	}

	executor := pipeline.NewDefaultPipelineExecutor(runner, execOpts...)

	// Connect deliverable tracker to progress display.
	if btpd, ok := progressDisplay.(*display.BubbleTeaProgressDisplay); ok {
		btpd.SetDeliverableTracker(executor.GetDeliverableTracker())
	}

	// Transition new run record to running.
	_ = store.UpdateRunStatus(resumeRunID, "running", "", 0)

	pipelineStart := time.Now()

	execErr := executor.ResumeWithValidation(ctx, p, &m, run.Input, fromStep, opts.Force, opts.RunID)

	// Update run status.
	tokens := executor.GetTotalTokens()
	switch {
	case ctx.Err() != nil:
		_ = store.UpdateRunStatus(resumeRunID, "cancelled", "pipeline cancelled", tokens)
		_ = store.ClearCancellation(resumeRunID)
	case execErr != nil:
		_ = store.UpdateRunStatus(resumeRunID, "failed", execErr.Error(), tokens)
	default:
		_ = store.UpdateRunStatus(resumeRunID, "completed", "", tokens)
	}

	if execErr != nil {
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

		var preflightMeta *recovery.PreflightMetadata
		if errClass == recovery.ClassPreflight {
			preflightMeta = extractPreflightMetadata(cause)
		}

		block := recovery.BuildRecoveryBlock(recovery.RecoveryBlockOpts{
			PipelineName:  p.Metadata.Name,
			Input:         run.Input,
			StepID:        stepID,
			RunID:         resumeRunID,
			WorkspaceRoot: wsRoot,
			ErrClass:      errClass,
			PreflightMeta: preflightMeta,
		})

		if opts.Output.Format == OutputFormatJSON {
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
				PipelineID:    resumeRunID,
				StepID:        stepID,
				State:         "recovery",
				Message:       execErr.Error(),
				RecoveryHints: hints,
			})
		} else {
			hintBlock := recovery.FormatRecoveryBlock(block)
			if hintBlock != "" {
				return fmt.Errorf("pipeline execution failed: %w\n%s", execErr, hintBlock)
			}
		}
		return fmt.Errorf("pipeline execution failed: %w", execErr)
	}

	elapsed := time.Since(pipelineStart)

	result.Cleanup()

	if opts.Output.Format == OutputFormatAuto || opts.Output.Format == OutputFormatText {
		totalTokens := executor.GetTotalTokens()
		if totalTokens > 0 {
			fmt.Fprintf(os.Stderr, "\n  ✓ Pipeline '%s' completed successfully (%.1fs, %s tokens)\n",
				p.Metadata.Name, elapsed.Seconds(), display.FormatTokenCount(totalTokens))
		} else {
			fmt.Fprintf(os.Stderr, "\n  ✓ Pipeline '%s' completed successfully (%.1fs)\n",
				p.Metadata.Name, elapsed.Seconds())
		}
	}

	if opts.Output.Format == OutputFormatJSON {
		tracker := executor.GetDeliverableTracker()
		outcome := display.BuildOutcome(tracker, p.Metadata.Name, resumeRunID, true, elapsed, executor.GetTotalTokens(), "", nil)
		outJSON := outcome.ToOutcomesJSON()
		emitter.Emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: resumeRunID,
			State:      "completed",
			DurationMs: elapsed.Milliseconds(),
			Message:    fmt.Sprintf("Pipeline '%s' completed", p.Metadata.Name),
			Outcomes:   outJSON,
		})
	}

	return nil
}

// detectFailedStep inspects the run's event log to find which step failed.
// It prefers an explicit "failed" event for a step. As a fallback it uses
// the run's CurrentStep field (the step that was active when the run was
// recorded as failed).
func detectFailedStep(store state.StateStore, run *state.RunRecord) (string, error) {
	// Query the event log for a "failed" state entry tied to a specific step.
	events, err := store.GetEvents(run.RunID, state.EventQueryOptions{
		Limit: 200,
	})
	if err == nil {
		// Walk backwards: the last step with state=="failed" is the one to resume.
		for i := len(events) - 1; i >= 0; i-- {
			ev := events[i]
			if ev.State == "failed" && ev.StepID != "" {
				return ev.StepID, nil
			}
		}
	}

	// Fallback: use the CurrentStep field on the run record.
	if run.CurrentStep != "" {
		return run.CurrentStep, nil
	}

	return "", fmt.Errorf("no failed step found in run %q; specify --from-step explicitly", run.RunID)
}
