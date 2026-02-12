package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/audit"
	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/workspace"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type RunOptions struct {
	Pipeline string
	Input    string
	DryRun   bool
	FromStep string
	Force    bool
	Timeout  int
	Manifest string
	Mock     bool
	Output   OutputConfig
}

func NewRunCmd() *cobra.Command {
	var opts RunOptions

	cmd := &cobra.Command{
		Use:   "run [pipeline] [input]",
		Short: "Run a pipeline",
		Long: `Execute a pipeline from the wave manifest.
Supports dry-run mode, step resumption, and custom timeouts.

Arguments can be provided as positional args or flags:
  wave run code-review "Review auth module"
  wave run --pipeline code-review --input "Review auth module"
  wave run code-review --input "Review auth module"`,
		Example: `  wave run code-review "Review the authentication changes"
  wave run --pipeline speckit-flow --input "add user auth"
  wave run hotfix --dry-run
  wave run migrate --from-step validate`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Handle positional arguments
			if len(args) >= 1 && opts.Pipeline == "" {
				opts.Pipeline = args[0]
			}
			if len(args) >= 2 && opts.Input == "" {
				opts.Input = args[1]
			}

			// Validate pipeline is provided
			if opts.Pipeline == "" {
				return fmt.Errorf("pipeline name is required (use positional arg or --pipeline flag)")
			}

			opts.Output = GetOutputConfig(cmd)
			if err := ValidateOutputFormat(opts.Output.Format); err != nil {
				return err
			}

			debug, _ := cmd.Flags().GetBool("debug")
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

	return cmd
}

func runRun(opts RunOptions, debug bool) error {
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
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	var m manifest.Manifest
	if err := yaml.Unmarshal(manifestData, &m); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	p, err := loadPipeline(opts.Pipeline, &m)
	if err != nil {
		return fmt.Errorf("failed to load pipeline: %w", err)
	}

	if opts.DryRun {
		return performDryRun(p, &m)
	}

	// Resolve adapter — use mock if --mock or if no adapter binary found
	var runner adapter.AdapterRunner
	if opts.Mock {
		// Add simulated delay to see progress animations in action
		runner = adapter.NewMockAdapter(
			adapter.WithSimulatedDelay(5*time.Second),
		)
	} else {
		var adapterName string
		for name := range m.Adapters {
			adapterName = name
			break
		}
		runner = adapter.ResolveAdapter(adapterName)
	}

	// Generate run ID once — shared by display and executor
	runID := pipeline.GenerateRunID(p.Metadata.Name, m.Runtime.PipelineIDHashLength)

	// Initialize event emitter based on output format
	result := CreateEmitter(opts.Output, runID, p.Metadata.Name, p.Steps, &m)
	emitter := result.Emitter
	progressDisplay := result.Progress
	defer result.Cleanup()

	// Initialize workspace manager under .wave/workspaces
	wsRoot := m.Runtime.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = ".wave/workspaces"
	}
	wsManager, err := workspace.NewWorkspaceManager(wsRoot)
	if err != nil {
		return fmt.Errorf("failed to create workspace manager: %w", err)
	}

	// Initialize state store under .wave/
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

	executor := pipeline.NewDefaultPipelineExecutor(runner, execOpts...)

	// Connect deliverable tracker to progress display
	if btpd, ok := progressDisplay.(*display.BubbleTeaProgressDisplay); ok {
		btpd.SetDeliverableTracker(executor.GetDeliverableTracker())
	}

	timeout := time.Duration(opts.Timeout) * time.Minute
	if opts.Timeout == 0 {
		timeout = m.Runtime.GetDefaultTimeout()
	}

	execCtx, execCancel := context.WithTimeout(ctx, timeout)
	defer execCancel()

	pipelineStart := time.Now()

	var execErr error
	if opts.FromStep != "" {
		// Resume from specific step - uses ResumeWithValidation which handles artifacts
		execErr = executor.ResumeWithValidation(execCtx, p, &m, opts.Input, opts.FromStep, opts.Force)
	} else {
		execErr = executor.Execute(execCtx, p, &m, opts.Input)
	}
	if execErr != nil {
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

		if deliverables := executor.GetDeliverables(); deliverables != "" {
			fmt.Fprint(os.Stderr, "\n")
			lines := strings.Split(deliverables, "\n")
			for _, line := range lines {
				if line != "" {
					fmt.Fprintf(os.Stderr, "  %s\n", line)
				}
			}
			fmt.Fprint(os.Stderr, "\n")
		}
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

func performDryRun(p *pipeline.Pipeline, m *manifest.Manifest) error {
	fmt.Printf("Dry run for pipeline: %s\n", p.Metadata.Name)
	fmt.Printf("Description: %s\n", p.Metadata.Description)
	fmt.Printf("Steps: %d\n\n", len(p.Steps))
	fmt.Printf("Execution plan:\n")

	for i, step := range p.Steps {
		fmt.Printf("  %d. %s (persona: %s)\n", i+1, step.ID, step.Persona)

		if len(step.Dependencies) > 0 {
			fmt.Printf("     Dependencies: %v\n", step.Dependencies)
		}

		persona := m.GetPersona(step.Persona)
		if persona != nil {
			fmt.Printf("     Adapter: %s  Temp: %.1f\n", persona.Adapter, persona.Temperature)
			fmt.Printf("     System prompt: %s\n", persona.SystemPromptFile)
			if len(persona.Permissions.AllowedTools) > 0 {
				fmt.Printf("     Allowed tools: %v\n", persona.Permissions.AllowedTools)
			}
			if len(persona.Permissions.Deny) > 0 {
				fmt.Printf("     Denied tools: %v\n", persona.Permissions.Deny)
			}
		}

		if len(step.Workspace.Mount) > 0 {
			for _, mount := range step.Workspace.Mount {
				fmt.Printf("     Mount: %s → %s (%s)\n", mount.Source, mount.Target, mount.Mode)
			}
		}

		fmt.Printf("     Workspace: .wave/workspaces/%s/%s/\n", p.Metadata.Name, step.ID)

		if step.Memory.Strategy != "" {
			fmt.Printf("     Memory: %s\n", step.Memory.Strategy)
		}

		if len(step.Memory.InjectArtifacts) > 0 {
			for _, art := range step.Memory.InjectArtifacts {
				fmt.Printf("     Inject: %s:%s as %s\n", art.Step, art.Artifact, art.As)
			}
		}

		if len(step.OutputArtifacts) > 0 {
			for _, art := range step.OutputArtifacts {
				fmt.Printf("     Output: %s → %s (%s)\n", art.Name, art.Path, art.Type)
			}
		}

		if step.Handover.Contract.Type != "" {
			fmt.Printf("     Contract: %s", step.Handover.Contract.Type)
			if step.Handover.Contract.OnFailure != "" {
				fmt.Printf(" (on_failure: %s, max_retries: %d)", step.Handover.Contract.OnFailure, step.Handover.Contract.MaxRetries)
			}
			fmt.Println()
		}

		fmt.Println()
	}

	return nil
}
