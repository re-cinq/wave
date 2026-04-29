package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/adapter/adaptertest"
	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/ontology"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/pipelinecatalog"
	"github.com/recinq/wave/internal/skill"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/tui"
	"github.com/recinq/wave/internal/workspace"
	"github.com/spf13/cobra"
)

// NewComposeCmd creates the compose command for validating and executing
// pipeline sequences.
func NewComposeCmd() *cobra.Command {
	var validateOnly bool
	var inputFlag string
	var mockFlag bool
	var manifestFlag string
	var parallelFlag bool
	var failFastFlag bool
	var maxConcurrentFlag int

	cmd := &cobra.Command{
		Use:   "compose [pipelines...]",
		Short: "Validate and execute a pipeline sequence",
		Long: `Validate artifact compatibility between adjacent pipelines in a sequence
and optionally execute them in order.

The compose command checks that each pipeline's output artifacts match the
next pipeline's expected input artifacts. This ensures data flows correctly
across pipeline boundaries before execution begins.

Use --validate-only to check compatibility without executing.`,
		Example: `  wave compose speckit-flow wave-evolve wave-review
  wave compose speckit-flow wave-evolve --validate-only
  wave compose pipeline-a pipeline-b pipeline-c --input "build feature X"
  wave compose --parallel A B -- C  (A+B parallel, then C sequential)`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkOnboarding(); err != nil {
				return NewCLIError(CodeOnboardingRequired,
					"onboarding not complete",
					"Run 'wave init' to complete setup before running pipelines")
			}

			outputCfg := GetOutputConfig(cmd)
			if err := ValidateOutputFormat(outputCfg.Format); err != nil {
				return err
			}

			cmd.SilenceUsage = true
			cmd.SilenceErrors = true

			pDir := pipelinesDir()

			// Load all pipelines from arguments
			var seq tui.Sequence
			for _, name := range args {
				p, err := pipelinecatalog.LoadPipelineByName(pDir, name)
				if err != nil {
					return NewCLIError(CodePipelineNotFound,
						fmt.Sprintf("pipeline not found: %s", name),
						"Run 'wave list pipelines' to see available pipelines")
				}
				seq.Add(name, p)
			}

			// Validate artifact compatibility across the sequence
			result := tui.ValidateSequence(seq)

			// Add composition template validation for all pipelines in the
			// sequence. Logic lives in internal/pipeline so webui/tui can reuse
			// it without depending on cobra.
			composeEntries := make([]pipeline.ComposeEntry, len(seq.Entries))
			for i, entry := range seq.Entries {
				composeEntries[i] = pipeline.ComposeEntry{
					Name:     entry.PipelineName,
					Pipeline: entry.Pipeline,
				}
			}
			templateErrors := pipeline.ValidateComposeSpec(composeEntries)

			if validateOnly {
				if err := renderValidationReport(args, result); err != nil {
					return err
				}
				if len(templateErrors) > 0 {
					fmt.Fprintln(os.Stdout, "Template validation errors:")
					for _, e := range templateErrors {
						fmt.Fprintf(os.Stdout, "  ✗ %s\n", e)
					}
					return NewCLIError(CodeContractViolation,
						"composition template validation failed",
						"Fix template references to point to valid step IDs")
				}
				return nil
			}

			// Not validate-only: check for errors before execution
			if result.Status == tui.CompatibilityError {
				renderValidationReport(args, result) //nolint:errcheck // best-effort stderr rendering before returning the real error
				return NewCLIError(CodeContractViolation,
					"sequence has incompatible artifact flows",
					"Run 'wave compose --validate-only' to see details, then fix pipeline artifacts")
			}

			// Sequence is valid or has warnings only — print informational message
			if result.Status == tui.CompatibilityWarning {
				fmt.Fprintf(os.Stderr, "Sequence validated with warnings:\n")
				for _, diag := range result.Diagnostics {
					fmt.Fprintf(os.Stderr, "  ! %s\n", diag)
				}
				fmt.Fprintln(os.Stderr)
			}

			debug, _ := cmd.Flags().GetBool("debug")

			if parallelFlag {
				plan := pipeline.BuildExecutionPlan(composeEntries, args)
				plan.FailFast = failFastFlag
				plan.MaxConcurrent = maxConcurrentFlag
				return runComposePlan(seq, plan, inputFlag, manifestFlag, mockFlag, outputCfg, debug)
			}

			return runCompose(seq, inputFlag, manifestFlag, mockFlag, outputCfg, debug)
		},
	}

	cmd.Flags().BoolVar(&validateOnly, "validate-only", false, "Check compatibility without executing")
	cmd.Flags().StringVar(&inputFlag, "input", "", "Input data passed to every pipeline in the sequence")
	cmd.Flags().BoolVar(&mockFlag, "mock", false, "Use mock adapter (for testing)")
	cmd.Flags().StringVar(&manifestFlag, "manifest", "wave.yaml", "Path to manifest file")
	cmd.Flags().BoolVar(&parallelFlag, "parallel", false, "Enable parallel execution (use -- to separate stages)")
	cmd.Flags().BoolVar(&failFastFlag, "fail-fast", true, "Stop on first failure (default true)")
	cmd.Flags().IntVar(&maxConcurrentFlag, "max-concurrent", 0, "Max concurrent pipelines per parallel stage (0 = unlimited)")

	return cmd
}

// renderValidationReport prints the compatibility report to stdout and
// returns an error if the result has CompatibilityError status.
func renderValidationReport(names []string, result tui.CompatibilityResult) error {
	fmt.Fprintf(os.Stdout, "Sequence validation: %s\n", formatSequenceArrow(names))
	fmt.Fprintln(os.Stdout)

	for i, flow := range result.Flows {
		fmt.Fprintf(os.Stdout, "Boundary %d: %s → %s\n", i+1, flow.SourcePipeline, flow.TargetPipeline)

		if len(flow.Matches) == 0 {
			fmt.Fprintf(os.Stdout, "  (no artifact flow)\n")
		}

		for _, match := range flow.Matches {
			switch match.Status {
			case tui.MatchCompatible:
				fmt.Fprintf(os.Stdout, "  ✓ %s → %s (compatible)\n", match.OutputName, match.InputName)
			case tui.MatchMissing:
				qualifier := "missing"
				if match.Optional {
					qualifier = "missing, optional"
				}
				fmt.Fprintf(os.Stdout, "  ✗ %s (%s — no matching output from %s)\n",
					match.InputName, qualifier, flow.SourcePipeline)
			case tui.MatchUnmatched:
				fmt.Fprintf(os.Stdout, "  ~ %s (output not consumed by %s)\n",
					match.OutputName, flow.TargetPipeline)
			}
		}

		fmt.Fprintln(os.Stdout)
	}

	// Count errors and warnings
	var errorCount, warningCount int
	for _, flow := range result.Flows {
		for _, match := range flow.Matches {
			if match.Status == tui.MatchMissing {
				if match.Optional {
					warningCount++
				} else {
					errorCount++
				}
			}
		}
	}

	fmt.Fprintf(os.Stdout, "Result: %d error(s), %d warning(s)\n", errorCount, warningCount)

	if result.Status == tui.CompatibilityError {
		return NewCLIError(CodeContractViolation,
			"sequence validation failed: incompatible artifact flows",
			"Fix pipeline artifacts to ensure outputs match expected inputs")
	}

	return nil
}

// formatSequenceArrow joins pipeline names with arrow separators.
func formatSequenceArrow(names []string) string {
	return strings.Join(names, " → ")
}

// composeRuntime bundles the shared dependencies wired up before sequence
// execution. Constructed once per compose run and reused by both Execute and
// ExecutePlan code paths.
type composeRuntime struct {
	ctx         context.Context
	cancel      context.CancelFunc
	manifest    manifest.Manifest
	seqExecutor *pipeline.SequenceExecutor
	store       interface{ Close() error }
}

// setupComposeRuntime constructs the manifest, adapter, state store, event
// emitter, workspace manager, and SequenceExecutor used by both compose
// execution paths. Caller is responsible for invoking close().
func setupComposeRuntime(manifestPath string, mock bool, outputCfg OutputConfig, debug bool) (*composeRuntime, error) {
	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		cancel()
	}()

	mp, err := loadManifestStrict(manifestPath)
	if err != nil {
		cancel()
		return nil, err
	}
	m := *mp

	var runner adapter.AdapterRunner
	if mock {
		runner = adaptertest.NewMockAdapter(adaptertest.WithSimulatedDelay(5 * time.Second))
	} else {
		var adapterName string
		for name := range m.Adapters {
			adapterName = name
			break
		}
		runner = adapter.ResolveAdapter(adapterName)
	}

	store, err := state.NewStateStore(".agents/state.db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: state persistence disabled: %v\n", err)
		store = nil
	}

	emitter := event.NewNDJSONEmitter()
	var eventEmitter event.EventEmitter = emitter
	if outputCfg.Format == OutputFormatAuto || outputCfg.Format == OutputFormatText {
		// Suppress JSON in text mode.
		eventEmitter = event.NewProgressOnlyEmitter(nil)
	}

	wsRoot := m.Runtime.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = ".agents/workspaces"
	}
	wsManager, err := workspace.NewWorkspaceManager(wsRoot)
	if err != nil {
		cancel()
		if store != nil {
			_ = store.Close()
		}
		return nil, fmt.Errorf("failed to create workspace manager: %w", err)
	}

	baseOpts := []pipeline.ExecutorOption{
		pipeline.WithDebug(debug),
		pipeline.WithOntologyService(ontology.NoOp{}),
	}
	if wsManager != nil {
		baseOpts = append(baseOpts, pipeline.WithWorkspaceManager(wsManager))
	}
	baseOpts = append(baseOpts, pipeline.WithSkillStore(skill.NewDirectoryStore(skill.DefaultSources()...)))

	newExecutor := func(opts ...pipeline.ExecutorOption) *pipeline.DefaultPipelineExecutor {
		return pipeline.NewDefaultPipelineExecutor(runner, opts...)
	}

	rt := &composeRuntime{
		ctx:         ctx,
		cancel:      cancel,
		manifest:    m,
		seqExecutor: pipeline.NewSequenceExecutor(newExecutor, baseOpts, eventEmitter, store),
	}
	if store != nil {
		rt.store = store
	}
	return rt, nil
}

func (rt *composeRuntime) close() {
	if rt == nil {
		return
	}
	if rt.store != nil {
		_ = rt.store.Close()
	}
	if rt.cancel != nil {
		rt.cancel()
	}
}

// printPipelineSummary prints per-pipeline status lines and an aggregate footer.
func printPipelineSummary(seqResult *pipeline.SequenceResult, footer string, elapsed time.Duration) {
	for _, pr := range seqResult.PipelineResults {
		status := "OK"
		if pr.Status == "failed" {
			status = "FAILED"
		}
		tokStr := ""
		if pr.TokensUsed > 0 {
			tokStr = fmt.Sprintf(", %s tokens", display.FormatTokenCount(pr.TokensUsed))
		}
		fmt.Fprintf(os.Stderr, "  [%s] %s (%.1fs%s)\n", status, pr.PipelineName, pr.Duration.Seconds(), tokStr)
	}
	fmt.Fprintln(os.Stderr)

	tokStr := ""
	if seqResult.TotalTokens > 0 {
		tokStr = fmt.Sprintf(", %s tokens", display.FormatTokenCount(seqResult.TotalTokens))
	}
	fmt.Fprintf(os.Stderr, "%s: %d pipelines in %.1fs%s\n",
		footer, len(seqResult.PipelineResults), elapsed.Seconds(), tokStr)
}

// runComposePlan executes a pipeline execution plan with parallel stage support.
func runComposePlan(_ tui.Sequence, plan pipeline.ExecutionPlan, input string, manifestPath string, mock bool, outputCfg OutputConfig, debug bool) error {
	rt, err := setupComposeRuntime(manifestPath, mock, outputCfg, debug)
	if err != nil {
		return err
	}
	defer rt.close()

	// Describe the plan
	for i, stage := range plan.Stages {
		names := make([]string, len(stage.Pipelines))
		for j, p := range stage.Pipelines {
			names[j] = p.Metadata.Name
		}
		mode := "sequential"
		if stage.Parallel {
			mode = "parallel"
		}
		fmt.Fprintf(os.Stderr, "Stage %d (%s): %s\n", i+1, mode, strings.Join(names, ", "))
	}
	fmt.Fprintln(os.Stderr)

	startTime := time.Now()
	seqResult, execErr := rt.seqExecutor.ExecutePlan(rt.ctx, plan, &rt.manifest, input)
	elapsed := time.Since(startTime)

	if execErr != nil {
		printPipelineSummary(seqResult, "Plan completed", elapsed)
		return fmt.Errorf("compose execution failed: %w", execErr)
	}
	printPipelineSummary(seqResult, "Plan completed", elapsed)
	return nil
}

// runCompose executes a validated pipeline sequence using SequenceExecutor.
func runCompose(seq tui.Sequence, input string, manifestPath string, mock bool, outputCfg OutputConfig, debug bool) error {
	rt, err := setupComposeRuntime(manifestPath, mock, outputCfg, debug)
	if err != nil {
		return err
	}
	defer rt.close()

	pipelines := make([]*pipeline.Pipeline, len(seq.Entries))
	pipelineNames := make([]string, len(seq.Entries))
	for i, entry := range seq.Entries {
		pipelines[i] = entry.Pipeline
		pipelineNames[i] = entry.PipelineName
	}

	fmt.Fprintf(os.Stderr, "Executing sequence: %s\n\n", formatSequenceArrow(pipelineNames))

	startTime := time.Now()
	seqResult, execErr := rt.seqExecutor.Execute(rt.ctx, pipelines, &rt.manifest, input)
	elapsed := time.Since(startTime)

	if execErr != nil {
		printPipelineSummary(seqResult, "Sequence completed", elapsed)
		return fmt.Errorf("compose execution failed: %w", execErr)
	}
	printPipelineSummary(seqResult, "Sequence completed", elapsed)
	return nil
}
