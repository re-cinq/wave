package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/state"
	"github.com/spf13/cobra"
)

// ForkOptions holds options for the fork command.
type ForkOptions struct {
	RunID       string
	FromStep    string
	List        bool
	AllowFailed bool
	Input       string
	Model       string
	Manifest    string
	Mock        bool
	Output      OutputConfig
}

// NewForkCmd creates the fork command.
func NewForkCmd() *cobra.Command {
	var opts ForkOptions

	cmd := &cobra.Command{
		Use:   "fork <run-id>",
		Short: "Fork a run from a checkpoint",
		Long: `Create a new independent run branching from a specific step of an existing run.

The forked run copies artifacts and workspace state from all steps up to the
fork point, then starts a fresh execution from the step after the fork point.
The original run is not modified.

Use --list to see available fork points (completed steps with checkpoints).`,
		Example: `  wave fork impl-issue-20240315-abc123 --from-step plan
  wave fork impl-issue-20240315-abc123 --from-step 3
  wave fork impl-issue-20240315-abc123 --list
  wave fork impl-issue-20240315-abc123 --list --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RunID = args[0]
			opts.Output = GetOutputConfig(cmd)

			if err := ValidateOutputFormat(opts.Output.Format); err != nil {
				return err
			}

			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return runFork(opts)
		},
	}

	cmd.Flags().StringVar(&opts.FromStep, "from-step", "", "Fork from after this step (required unless --list)")
	cmd.Flags().BoolVar(&opts.List, "list", false, "List available fork points")
	cmd.Flags().BoolVar(&opts.AllowFailed, "allow-failed", false, "Allow forking non-completed (failed/cancelled) runs")
	cmd.Flags().StringVar(&opts.Input, "input", "", "Override input for the forked run")
	cmd.Flags().StringVar(&opts.Model, "model", "", "Override adapter model for the forked run")
	cmd.Flags().StringVar(&opts.Manifest, "manifest", "wave.yaml", "Path to manifest file")
	cmd.Flags().BoolVar(&opts.Mock, "mock", false, "Use mock adapter (for testing)")

	return cmd
}

func runFork(opts ForkOptions) error {
	// Open state store
	store, err := state.NewStateStore(".wave/state.db")
	if err != nil {
		return NewCLIError(CodeStateDBError,
			fmt.Sprintf("failed to open state database: %v", err),
			"Run 'wave init' to set up the project, or check that .wave/state.db exists")
	}
	defer store.Close()

	// Validate run exists
	run, err := store.GetRun(opts.RunID)
	if err != nil {
		return NewCLIError(CodeRunNotFound,
			fmt.Sprintf("run %q not found", opts.RunID),
			"Use 'wave list runs' to see available run IDs")
	}

	// Load manifest for pipeline info
	mp, err := loadManifestStrict(opts.Manifest)
	if err != nil {
		return err
	}
	m := *mp

	// Load pipeline
	p, err := loadPipeline(run.PipelineName, &m)
	if err != nil {
		return NewCLIError(CodePipelineNotFound,
			fmt.Sprintf("pipeline %q not found", run.PipelineName),
			"Run 'wave list pipelines' to see available pipelines")
	}

	forkMgr := pipeline.NewForkManager(store)

	// Handle --list mode
	if opts.List {
		return listForkPoints(forkMgr, opts)
	}

	// --from-step is required for actual fork
	if opts.FromStep == "" {
		return NewCLIError(CodeInvalidArgs,
			"--from-step is required for fork",
			"Use --list to see available fork points, then specify --from-step <step>")
	}

	// Execute fork
	newRunID, err := forkMgr.Fork(opts.RunID, opts.FromStep, p, opts.AllowFailed)
	if err != nil {
		return NewCLIError(CodeInvalidArgs,
			fmt.Sprintf("fork failed: %v", err),
			"Use 'wave fork <run-id> --list' to see available fork points")
	}

	// Output result
	if opts.Output.Format == OutputFormatJSON {
		result := map[string]string{
			"run_id":        newRunID,
			"forked_from":   opts.RunID,
			"from_step":     opts.FromStep,
			"pipeline_name": run.PipelineName,
			"status":        "created",
		}
		data, _ := json.Marshal(result)
		fmt.Println(string(data))
	} else {
		fmt.Fprintf(os.Stderr, "  Forked run created: %s\n", newRunID)
		fmt.Fprintf(os.Stderr, "  Source: %s (from after step %q)\n", opts.RunID, opts.FromStep)
		fmt.Fprintf(os.Stderr, "  Pipeline: %s\n\n", run.PipelineName)
		fmt.Fprintf(os.Stderr, "  To execute the forked run:\n")
		fmt.Fprintf(os.Stderr, "    wave resume %s --from-step <next-step>\n", newRunID)
	}

	return nil
}

func listForkPoints(forkMgr *pipeline.ForkManager, opts ForkOptions) error {
	points, err := forkMgr.ListForkPoints(opts.RunID)
	if err != nil {
		return NewCLIError(CodeInvalidArgs,
			fmt.Sprintf("failed to list fork points: %v", err),
			"Check that the run ID is correct")
	}

	if opts.Output.Format == OutputFormatJSON {
		result := map[string]interface{}{
			"run_id":      opts.RunID,
			"fork_points": points,
		}
		data, _ := json.Marshal(result)
		fmt.Println(string(data))
	} else {
		if len(points) == 0 {
			fmt.Fprintf(os.Stderr, "  No fork points available for run %s\n", opts.RunID)
			fmt.Fprintf(os.Stderr, "  (Run must have completed steps with checkpoint data)\n")
		} else {
			fmt.Fprintf(os.Stderr, "  Fork points for run %s:\n\n", opts.RunID)
			for _, p := range points {
				sha := ""
				if p.HasSHA {
					sha = " (has workspace SHA)"
				}
				fmt.Fprintf(os.Stderr, "    [%d] %s%s\n", p.StepIndex, p.StepID, sha)
			}
			fmt.Fprintf(os.Stderr, "\n  Usage: wave fork %s --from-step <step>\n", opts.RunID)
		}
	}
	return nil
}
