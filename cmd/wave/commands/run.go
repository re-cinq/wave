package commands

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/recinq/wave/internal/config"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/runner"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/suggest"
	"github.com/recinq/wave/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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
	if err := validateFlags(opts); err != nil {
		return err
	}

	ctx, cancel := setupSignalHandling()
	defer cancel()

	p, m, stepFilter, aborted, err := loadManifestAndPipeline(&opts, &debug)
	if err != nil {
		return err
	}
	if aborted {
		return nil
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
	executor := res.executor
	emitter := res.emitter
	wsRoot := res.wsRoot
	runner := res.runner
	execOpts := res.execOpts

	if opts.Continuous {
		return runContinuous(ctx, opts, &m, p, store, runner, emitter, execOpts)
	}

	pipelineStart := time.Now()
	execErr := runOnce(ctx, executor, opts, &m, p, store, runID)

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


