package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/recinq/wave/internal/config"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/runner"
	"github.com/recinq/wave/internal/state"
	"github.com/spf13/cobra"
)

// RunOptions is aliased from internal/config — the canonical struct lives
// there (alongside the env snapshot) so the webui launch path consumes the
// exact same fields without a translation layer. The exhaustiveness test
// (TestDetachedArgsExhaustive) lives in internal/runner alongside the spec
// table it guards.
type RunOptions = config.RuntimeConfig

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

The --adapter flag selects the LLM backend (claude or codex; opencode and
gemini also work when configured in wave.yaml). Model formats vary by
adapter: claude uses short names or full IDs ("sonnet"), codex uses OpenAI
model IDs ("gpt-6.2-sol"), opencode uses "provider/model", gemini uses plain
names.`,
		Example: `  wave run ops-pr-review "Review the authentication changes"
  wave run --pipeline impl-speckit --input "add user auth"
  wave run impl-issue --dry-run
  wave run migrate --from-step validate
  wave run my-pipeline --model sonnet
  wave run my-pipeline --adapter codex --model gpt-6.2-sol
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

			if opts.Pipeline == "" {
				return fmt.Errorf("pipeline name is required (use positional arg or --pipeline flag)")
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
	cmd.Flags().StringVarP(&opts.Manifest, "manifest", "m", "wave.yaml", "Path to manifest file")
	cmd.Flags().BoolVar(&opts.Mock, "mock", false, "Use mock adapter (for testing)")
	cmd.Flags().StringVar(&opts.RunID, "run", "", "Resume from a specific run (uses that run's input)")
	cmd.Flags().StringVar(&opts.Model, "model", "", "Model for this run — tier name (cheapest/balanced/strongest) or literal (haiku/opus). Takes the cheaper of CLI and step tiers unless --force-model is set.")
	cmd.Flags().BoolVar(&opts.ForceModel, "force-model", false, "Force --model on all steps, ignoring per-step and per-persona model tiers")
	cmd.Flags().StringVar(&opts.Adapter, "adapter", "", "Override adapter for this run (e.g. claude, codex)")
	cmd.Flags().BoolVar(&opts.PreserveWorkspace, "preserve-workspace", false, "Preserve workspace from previous run (for debugging)")
	cmd.Flags().StringVar(&opts.Steps, "steps", "", "Run only the named steps (comma-separated)")
	cmd.Flags().StringVarP(&opts.Exclude, "exclude", "x", "", "Skip the named steps (comma-separated)")
	cmd.Flags().BoolVar(&opts.Detach, "detach", false, "Run pipeline as a detached background process")
	cmd.Flags().BoolVar(&opts.AutoApprove, "auto-approve", false, "Auto-approve all approval gates using default choices (required for --detach with gates)")
	cmd.Flags().BoolVar(&opts.NoRetro, "no-retro", false, "Skip retrospective generation for this run")

	// Group flags by tier for organized --help output
	essentialFlags := []string{"pipeline", "input", "model", "adapter"}
	executionFlags := []string{"from-step", "force", "dry-run", "timeout", "steps", "exclude", "detach"}
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
	if err := validateFlags(opts); err != nil {
		return err
	}

	ctx, cancel := setupSignalHandling()
	defer cancel()

	p, m, stepFilter, err := loadManifestAndPipeline(&opts)
	if err != nil {
		return err
	}

	if opts.DryRun {
		return performDryRun(p, &m, stepFilter, opts)
	}

	if !opts.Mock {
		if err := checkRunAdapters(p, &m, opts.Adapter); err != nil {
			return err
		}
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

	// Initialize state store under .agents/ — must happen before run ID generation
	// so we can use CreateRun() to produce IDs visible to the dashboard.
	store := buildStateStore()
	if store != nil {
		defer store.Close()
	}

	autoRecoverResumeInput(&opts, store, p)

	runID := resolveOrGenerateRunID(opts, store, p, &m)

	res, err := buildExecutor(opts, &m, p, store, stepFilter, runID, debug)
	if err != nil {
		return err
	}
	defer res.Close()
	emitter := res.emitter
	wsRoot := res.wsRoot

	pipelineStart := time.Now()
	executor, execErr := runOnce(ctx, res, opts)

	if execErr != nil {
		// Design rejection: contract with on_failure: rejected fired. The
		// persona reported the work is non-actionable (e.g. issue already
		// implemented, no real bug, superseded). Render with a distinct
		// non-red banner and exit 0 — this was a legitimate verdict, not
		// a runtime failure. Stop the TUI first so the banner reaches a
		// clean terminal.
		var rejectionErr *pipeline.ContractRejectionError
		if errors.As(execErr, &rejectionErr) {
			res.Close()
			printRejectionSummary(opts, p, rejectionErr, time.Since(pipelineStart), emitter, runID)
			return nil
		}
		return formatRecoveryError(execErr, opts, p, runID, wsRoot, emitter)
	}

	elapsed := time.Since(pipelineStart)

	// Stop the TUI before printing post-run output to avoid terminal corruption.
	// Cleanup is idempotent so the deferred call above becomes a no-op.
	res.Close()

	// Bootstrap pipelines mutate worktree-isolated state by design. After a
	// successful run, materialise the project-shaping outputs (sentinel,
	// wave.yaml, custom personas/pipelines/prompts, output JSONs) at the
	// project root so the user sees the result of the onboarding work.
	if bootstrapPipelines[opts.Pipeline] {
		if err := materialiseBootstrapOutputs(runID); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to materialise bootstrap outputs: %v\n", err)
		}
	}

	printSummary(opts, executor, p, runID, elapsed, emitter)
	return nil
}

// runDetached spawns a new `wave run` subprocess that is fully detached from
// the current process session via internal/runner. The subprocess inherits
// all flags except --detach and runs the pipeline in its own session group.
// runner.Detach is the single source of truth used by both the CLI path and
// the webui server, so changes to the spawn protocol live in exactly one
// place (and are exercised by TestDetachedArgsExhaustive).
func runDetached(opts RunOptions, p *pipeline.Pipeline, m *manifest.Manifest) error {
	// Cold-start repos have no .agents/ yet; create it so SQLite can open the
	// db file. mkdir-all is a no-op when the directory already exists.
	_ = os.MkdirAll(".agents", 0o755)
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

	return printDetachedLaunch(opts.Output.Format, p.Metadata.Name, runID)
}

// DetachedLaunchResult is the stable machine-readable acknowledgement emitted
// after Wave has reserved the run ID and released the detached subprocess.
type DetachedLaunchResult struct {
	RunID        string `json:"run_id"`
	PipelineName string `json:"pipeline_name"`
	Status       string `json:"status"`
	Detached     bool   `json:"detached"`
}

func printDetachedLaunch(format, pipelineName, runID string) error {
	if format == OutputFormatJSON {
		return json.NewEncoder(os.Stdout).Encode(DetachedLaunchResult{
			RunID: runID, PipelineName: pipelineName, Status: "running", Detached: true,
		})
	}
	if format == OutputFormatQuiet {
		return nil
	}
	fmt.Fprintf(os.Stderr, "  Pipeline '%s' launched (detached)\n", pipelineName)
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
func resolveRunID(runIDOpt string, store interface {
	CreateRun(pipelineName string, input string) (string, error)
}, pipelineName, input string) (string, error) {
	if runIDOpt != "" {
		return runIDOpt, nil
	}
	if store != nil {
		return store.CreateRun(pipelineName, input)
	}
	return "", nil
}

// pipelinesDir returns the default pipeline directory.
func pipelinesDir() string {
	return ".agents/pipelines"
}
