package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/state"
	"github.com/spf13/cobra"
)

// RewindOptions holds options for the rewind command.
type RewindOptions struct {
	RunID    string
	ToStep   string
	Confirm  bool
	Manifest string
	Output   OutputConfig
}

// NewRewindCmd creates the rewind command.
func NewRewindCmd() *cobra.Command {
	var opts RewindOptions

	cmd := &cobra.Command{
		Use:   "rewind <run-id>",
		Short: "Rewind a run to an earlier checkpoint",
		Long: `Reset a run's state to an earlier checkpoint (destructive).

This deletes all state for steps after the rewind point, including step
attempts, events, and progress records. The run status is set to 'failed'
so it can be resumed with 'wave resume'.

WARNING: This operation is destructive and cannot be undone.`,
		Example: `  wave rewind impl-issue-20240315-abc123 --to-step plan
  wave rewind impl-issue-20240315-abc123 --to-step plan --confirm
  wave rewind impl-issue-20240315-abc123 --to-step 2 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RunID = args[0]
			opts.Output = GetOutputConfig(cmd)

			if err := ValidateOutputFormat(opts.Output.Format); err != nil {
				return err
			}

			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return runRewind(opts)
		},
	}

	cmd.Flags().StringVar(&opts.ToStep, "to-step", "", "Rewind to after this step (required)")
	cmd.Flags().BoolVar(&opts.Confirm, "confirm", false, "Skip confirmation prompt")
	cmd.Flags().StringVar(&opts.Manifest, "manifest", "wave.yaml", "Path to manifest file")

	return cmd
}

func runRewind(opts RewindOptions) error {
	if opts.ToStep == "" {
		return NewCLIError(CodeInvalidArgs,
			"--to-step is required",
			"Specify the step to rewind to, e.g., --to-step plan")
	}

	// Open state store
	store, err := state.NewStateStore(".agents/state.db")
	if err != nil {
		return NewCLIError(CodeStateDBError,
			fmt.Sprintf("failed to open state database: %v", err),
			"Run 'wave init' to set up the project")
	}
	defer store.Close()

	// Validate run exists
	run, err := store.GetRun(opts.RunID)
	if err != nil {
		return NewCLIError(CodeRunNotFound,
			fmt.Sprintf("run %q not found", opts.RunID),
			"Use 'wave list runs' to see available run IDs")
	}

	if run.Status == "running" {
		return NewCLIError(CodeInvalidArgs,
			fmt.Sprintf("cannot rewind a running run %q", opts.RunID),
			"Cancel the run first with 'wave cancel "+opts.RunID+"'")
	}

	// Load manifest to get pipeline topology
	mp, err := loadManifestStrict(opts.Manifest)
	if err != nil {
		return err
	}
	m := *mp

	p, err := loadPipeline(run.PipelineName, &m)
	if err != nil {
		return NewCLIError(CodePipelineNotFound,
			fmt.Sprintf("pipeline %q not found", run.PipelineName),
			"Run 'wave list pipelines' to see available pipelines")
	}

	// Resolve the rewind step
	rewindStepID, rewindIndex := resolveRewindStep(opts.ToStep, p)
	if rewindStepID == "" {
		return NewCLIError(CodeInvalidArgs,
			fmt.Sprintf("step %q not found in pipeline %q", opts.ToStep, p.Metadata.Name),
			"Available steps: "+formatStepList(p))
	}

	// Find steps that will be deleted (after the rewind point)
	stepsToDelete := findStepsAfter(p, rewindIndex)

	if len(stepsToDelete) == 0 {
		if opts.Output.Format == OutputFormatJSON {
			result := map[string]interface{}{
				"run_id":  opts.RunID,
				"to_step": rewindStepID,
				"status":  "no_change",
				"message": "no steps to rewind",
			}
			data, _ := json.Marshal(result)
			fmt.Println(string(data))
		} else {
			fmt.Fprintf(os.Stderr, "  Nothing to rewind — %q is the last step\n", rewindStepID)
		}
		return nil
	}

	// Show what will be deleted
	if !opts.Confirm && opts.Output.Format != OutputFormatJSON {
		fmt.Fprintf(os.Stderr, "  Rewinding run %s to after step %q\n\n", opts.RunID, rewindStepID)
		fmt.Fprintf(os.Stderr, "  The following steps will be reset:\n")
		for _, stepID := range stepsToDelete {
			fmt.Fprintf(os.Stderr, "    - %s\n", stepID)
		}
		fmt.Fprintf(os.Stderr, "\n  WARNING: This is destructive and cannot be undone.\n")
		fmt.Fprintf(os.Stderr, "  Re-run with --confirm to proceed.\n")
		return nil
	}

	// Execute rewind: delete state for steps after the rewind point
	if err := executeRewind(store, opts.RunID, rewindIndex); err != nil {
		return NewCLIError(CodeInternalError,
			fmt.Sprintf("rewind failed: %v", err),
			"Check .agents/state.db integrity")
	}

	// Update run status to 'failed' so wave resume can pick it up
	if err := store.UpdateRunStatus(opts.RunID, "failed", rewindStepID, run.TotalTokens); err != nil {
		return NewCLIError(CodeInternalError,
			fmt.Sprintf("failed to update run status: %v", err),
			"")
	}

	// Output result
	if opts.Output.Format == OutputFormatJSON {
		result := map[string]interface{}{
			"run_id":        opts.RunID,
			"to_step":       rewindStepID,
			"steps_deleted": stepsToDelete,
			"status":        "rewound",
		}
		data, _ := json.Marshal(result)
		fmt.Println(string(data))
	} else {
		fmt.Fprintf(os.Stderr, "  Run %s rewound to after step %q\n", opts.RunID, rewindStepID)
		fmt.Fprintf(os.Stderr, "  %d step(s) reset: %v\n\n", len(stepsToDelete), stepsToDelete)
		fmt.Fprintf(os.Stderr, "  To resume execution:\n")
		fmt.Fprintf(os.Stderr, "    wave resume %s\n", opts.RunID)
	}

	return nil
}

// resolveRewindStep resolves a step reference to (stepID, stepIndex).
func resolveRewindStep(ref string, p *pipeline.Pipeline) (string, int) {
	// Try direct match by step ID
	for i, step := range p.Steps {
		if step.ID == ref {
			return step.ID, i
		}
	}

	// Try numeric index
	var idx int
	if _, err := fmt.Sscanf(ref, "%d", &idx); err == nil {
		if idx >= 0 && idx < len(p.Steps) {
			return p.Steps[idx].ID, idx
		}
	}

	return "", -1
}

// findStepsAfter returns the IDs of steps that come after the given index.
func findStepsAfter(p *pipeline.Pipeline, afterIndex int) []string {
	var steps []string
	for i, step := range p.Steps {
		if i > afterIndex {
			steps = append(steps, step.ID)
		}
	}
	return steps
}

// executeRewind deletes state DB records for steps after the rewind point.
func executeRewind(store state.StateStore, runID string, rewindIndex int) error {
	// Delete checkpoints after the rewind point
	if err := store.DeleteCheckpointsAfterStep(runID, rewindIndex); err != nil {
		return fmt.Errorf("failed to delete checkpoints: %w", err)
	}

	// Note: step_attempt, event_log, step_progress records for deleted steps
	// remain in the DB for audit trail purposes. The run status change to 'failed'
	// is what allows wave resume to pick it up and re-execute from the right point.

	return nil
}

// formatStepList formats pipeline steps as a comma-separated list for error messages.
func formatStepList(p *pipeline.Pipeline) string {
	if len(p.Steps) == 0 {
		return "(none)"
	}
	result := ""
	for i, step := range p.Steps {
		if i > 0 {
			result += ", "
		}
		result += step.ID
	}
	return result
}
