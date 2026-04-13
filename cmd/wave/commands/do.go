package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/classify"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/workspace"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type DoOptions struct {
	Persona    string
	Manifest   string
	Mock       bool
	DryRun     bool
	NoClassify bool
	Output     OutputConfig
	Model      string
	Adapter    string
	Detach     bool
}

func NewDoCmd() *cobra.Command {
	var opts DoOptions

	cmd := &cobra.Command{
		Use:   "do [task description]",
		Short: "Execute an ad-hoc task",
		Long: `Generate and run a minimal navigate→execute pipeline for a one-off task.
The task description is passed as arguments.

For dynamically generated multi-step pipelines, use 'wave meta' instead.

Examples:
  wave do "fix the login bug"
  wave do "add input validation to the form"
  wave do "refactor the database queries" --persona craftsman`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Output = GetOutputConfig(cmd)
			if err := ValidateOutputFormat(opts.Output.Format); err != nil {
				return err
			}
			input := strings.Join(args, " ")
			return runDo(input, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Persona, "persona", "", "Override execute persona (default: craftsman)")
	cmd.Flags().StringVar(&opts.Manifest, "manifest", "wave.yaml", "Path to manifest file")
	cmd.Flags().BoolVar(&opts.Mock, "mock", false, "Use mock adapter (for testing)")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Show what would be executed without running")
	cmd.Flags().StringVar(&opts.Model, "model", "", "Override adapter model for this run (e.g. haiku, opus)")
	cmd.Flags().BoolVar(&opts.NoClassify, "no-classify", false, "Bypass task classification and use ad-hoc pipeline")
	cmd.Flags().StringVar(&opts.Adapter, "adapter", "", "Override adapter (claude, opencode, gemini, codex)")
	cmd.Flags().BoolVar(&opts.Detach, "detach", false, "Run in background as detached process")

	return cmd
}

func runDo(input string, opts DoOptions) error {
	// Gate on onboarding completion
	if err := checkOnboarding(); err != nil {
		return err
	}

	manifestData, err := os.ReadFile(opts.Manifest)
	if err != nil {
		if os.IsNotExist(err) {
			return NewCLIError(CodeManifestMissing, fmt.Sprintf("manifest file not found: %s", opts.Manifest), "Run 'wave init' to create a new Wave project or specify --manifest path")
		}
		return NewCLIError(CodeManifestMissing, fmt.Sprintf("failed to read manifest: %s", err), "Check file permissions and path").WithCause(err)
	}

	var m manifest.Manifest
	if err := yaml.Unmarshal(manifestData, &m); err != nil {
		return NewCLIError(CodeManifestInvalid, fmt.Sprintf("failed to parse manifest %s: %s", opts.Manifest, err), "Ensure the file is valid YAML with correct indentation").WithCause(err)
	}

	// Classification: when not bypassed and no explicit persona, classify input
	// to select the best pipeline from the manifest.
	useClassification := !opts.NoClassify && opts.Persona == ""

	var profile classify.TaskProfile
	var pipelineCfg classify.PipelineConfig
	var classifiedPipeline *pipeline.Pipeline

	if useClassification {
		profile = classify.Classify(input, "")
		pipelineCfg = classify.SelectPipeline(profile)

		// Try to load the classified pipeline from disk
		if p, err := loadPipeline(pipelineCfg.Name, &m); err == nil {
			classifiedPipeline = p
		} else {
			fmt.Fprintf(os.Stderr, "classified pipeline %q not found, falling back to ad-hoc\n", pipelineCfg.Name)
		}
	}

	// Determine which pipeline to execute
	var p *pipeline.Pipeline
	var pipelineLabel string

	if classifiedPipeline != nil {
		p = classifiedPipeline
		pipelineLabel = pipelineCfg.Name
	} else {
		// Fallback: generate the ad-hoc navigate→execute pipeline
		executePersona := opts.Persona
		if executePersona == "" {
			executePersona = "craftsman"
		}

		adHocOpts := pipeline.AdHocOptions{
			Input:          input,
			ExecutePersona: executePersona,
			Manifest:       &m,
		}

		generated, err := pipeline.GenerateAdHocPipeline(adHocOpts)
		if err != nil {
			return NewCLIError(CodeInternalError, fmt.Sprintf("failed to generate pipeline: %s", err), "Check manifest personas and adapter configuration").WithCause(err)
		}
		p = generated
		pipelineLabel = "adhoc"
	}

	if opts.DryRun {
		if useClassification {
			fmt.Printf("Classification:\n")
			fmt.Printf("  Domain:       %s\n", profile.Domain)
			fmt.Printf("  Complexity:   %s\n", profile.Complexity)
			fmt.Printf("  Blast radius: %.1f\n", profile.BlastRadius)
			fmt.Printf("  Pipeline:     %s\n", pipelineCfg.Name)
			fmt.Printf("  Reason:       %s\n", pipelineCfg.Reason)
			if classifiedPipeline == nil {
				fmt.Printf("  Fallback:     ad-hoc (classified pipeline not found)\n")
			}
			fmt.Printf("\n")
		}
		fmt.Printf("Ad-hoc pipeline: navigate → execute\n")
		fmt.Printf("  Input: %s\n", input)
		fmt.Printf("  Steps:\n")
		for i, step := range p.Steps {
			fmt.Printf("    %d. %s (persona: %s)\n", i+1, step.ID, step.Persona)
		}
		fmt.Printf("  Workspace: .wave/workspaces/%s/\n", pipelineLabel)
		return nil
	}

	// Open state store for decision tracking and executor wiring
	var store state.StateStore
	if s, err := state.NewStateStore(".wave/state.db"); err == nil {
		store = s
		defer store.Close()
	}


	// Execute the pipeline
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		cancel()
	}()

	var runner adapter.AdapterRunner
	if opts.Mock {
		runner = adapter.NewMockAdapter()
	} else {
		adapterName := opts.Adapter
		if adapterName == "" {
			for name := range m.Adapters {
				adapterName = name
				break
			}
		}
		runner = adapter.ResolveAdapter(adapterName)
	}

	result := CreateEmitter(opts.Output, pipelineLabel, pipelineLabel, p.Steps, &m)
	defer result.Cleanup()

	wsRoot := m.Runtime.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = ".wave/workspaces"
	}
	wsManager, _ := workspace.NewWorkspaceManager(wsRoot)

	execOpts := []pipeline.ExecutorOption{
		pipeline.WithEmitter(result.Emitter),
	}
	if wsManager != nil {
		execOpts = append(execOpts, pipeline.WithWorkspaceManager(wsManager))
	}
	if opts.Model != "" {
		execOpts = append(execOpts, pipeline.WithModelOverride(opts.Model))
	}
	if store != nil {
		execOpts = append(execOpts, pipeline.WithStateStore(store))
	}

	executor := pipeline.NewDefaultPipelineExecutor(runner, execOpts...)

	timeout := m.Runtime.GetDefaultTimeout()
	execCtx, execCancel := context.WithTimeout(ctx, timeout)
	defer execCancel()

	pipelineStart := time.Now()

	if err := executor.Execute(execCtx, p, &m, input); err != nil {
		if store != nil && useClassification {
			elapsed := time.Since(pipelineStart)
			recordOrchestrationDecision(store, executor.GetRunID(), input, profile, pipelineCfg, "failed", executor.GetTotalTokens(), elapsed.Milliseconds())
		}
		return NewCLIError(CodeInternalError, fmt.Sprintf("execution failed: %s", err), "Run 'wave logs' to inspect execution details").WithCause(err)
	}

	elapsed := time.Since(pipelineStart)
	if store != nil && useClassification {
		recordOrchestrationDecision(store, executor.GetRunID(), input, profile, pipelineCfg, "completed", executor.GetTotalTokens(), elapsed.Milliseconds())
	}
	if opts.Output.Format == OutputFormatAuto || opts.Output.Format == OutputFormatText {
		fmt.Fprintf(os.Stderr, "\nTask completed (%.1fs)\n", elapsed.Seconds())
	}
	return nil
}

// recordOrchestrationDecision persists the classification → pipeline routing decision
// so it appears in `wave analyze --decisions`.
func recordOrchestrationDecision(store state.StateStore, runID, input string, profile classify.TaskProfile, cfg classify.PipelineConfig, outcome string, tokensUsed int, durationMs int64) {
	if runID == "" {
		return
	}
	_ = store.RecordOrchestrationDecision(&state.OrchestrationDecision{
		RunID:        runID,
		InputText:    input,
		Domain:       string(profile.Domain),
		Complexity:   string(profile.Complexity),
		PipelineName: cfg.Name,
		ModelTier:    string(cfg.ModelTier),
		Reason:       cfg.Reason,
		Outcome:      outcome,
		TokensUsed:   tokensUsed,
		DurationMs:   durationMs,
		CreatedAt:    time.Now(),
	})
}
