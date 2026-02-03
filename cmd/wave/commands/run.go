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
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/workspace"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type RunOptions struct {
	Pipeline     string
	Input        string
	DryRun       bool
	FromStep     string
	Timeout      int
	Manifest     string
	Mock         bool
	NoProgress   bool
	PlainProgress bool
	NoLogs       bool
}

func NewRunCmd() *cobra.Command {
	var opts RunOptions

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a pipeline",
		Long: `Execute a pipeline from the wave manifest.
Supports dry-run mode, step resumption, and custom timeouts.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			debug, _ := cmd.Flags().GetBool("debug")
			return runRun(opts, debug)
		},
	}

	cmd.Flags().StringVar(&opts.Pipeline, "pipeline", "", "Pipeline name to run (required)")
	cmd.Flags().StringVar(&opts.Input, "input", "", "Input data for the pipeline")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Show what would be executed without running")
	cmd.Flags().StringVar(&opts.FromStep, "from-step", "", "Start execution from specific step")
	cmd.Flags().IntVar(&opts.Timeout, "timeout", 0, "Timeout in minutes (overrides manifest)")
	cmd.Flags().StringVar(&opts.Manifest, "manifest", "wave.yaml", "Path to manifest file")
	cmd.Flags().BoolVar(&opts.Mock, "mock", false, "Use mock adapter (for testing)")
	cmd.Flags().BoolVar(&opts.NoProgress, "no-progress", false, "Disable enhanced progress display")
	cmd.Flags().BoolVar(&opts.PlainProgress, "plain", false, "Use plain text progress (no colors/animations)")
	cmd.Flags().BoolVar(&opts.NoLogs, "no-logs", false, "Suppress JSON log output (show only progress display)")

	cmd.MarkFlagRequired("pipeline")

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

	// Initialize event emitter with optional enhanced progress display
	var emitter *event.NDJSONEmitter
	var progressDisplay event.ProgressEmitter

	// Detect terminal capabilities and user preferences
	termInfo := display.NewTerminalInfo()
	useEnhancedProgress := !opts.NoProgress && !opts.PlainProgress && termInfo.IsTTY() && termInfo.SupportsANSI()

	if useEnhancedProgress {
		// Create bubbletea enhanced progress display with deliverable tracking
		progressDisplay = display.NewBubbleTeaProgressDisplay(p.Metadata.Name, p.Metadata.Name, len(p.Steps), nil) // Will be set later

		// Register steps for tracking
		btpd := progressDisplay.(*display.BubbleTeaProgressDisplay)
		for _, step := range p.Steps {
			// Get persona name for display
			personaName := step.Persona
			if persona := m.GetPersona(step.Persona); persona != nil {
				personaName = step.Persona
			}
			btpd.AddStep(step.ID, step.ID, personaName)
		}

		// Create emitter with progress display
		if opts.NoLogs {
			emitter = event.NewProgressOnlyEmitter(progressDisplay)
		} else {
			emitter = event.NewNDJSONEmitterWithProgress(progressDisplay)
		}
	} else if opts.PlainProgress {
		// Use basic text progress
		progressDisplay = display.NewBasicProgressDisplay()
		if opts.NoLogs {
			emitter = event.NewProgressOnlyEmitter(progressDisplay)
		} else {
			emitter = event.NewNDJSONEmitterWithProgress(progressDisplay)
		}
	} else {
		// Use standard human-readable output
		if opts.NoLogs {
			// Avoid NDJSON logs; use a progress-only emitter with a basic display.
			progressDisplay = display.NewBasicProgressDisplay()
			emitter = event.NewProgressOnlyEmitter(progressDisplay)
		} else {
			emitter = event.NewNDJSONEmitterWithHumanReadable()
		}
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

	// Ensure progress display cleanup on exit
	if btpd, ok := progressDisplay.(*display.BubbleTeaProgressDisplay); ok {
		defer btpd.Finish()
	}

	var execErr error
	if opts.FromStep != "" {
		// Resume from specific step - uses ResumeWithValidation which handles artifacts
		execErr = executor.ResumeWithValidation(execCtx, p, &m, opts.Input, opts.FromStep)
	} else {
		execErr = executor.Execute(execCtx, p, &m, opts.Input)
	}
	if execErr != nil {
		// Clear progress display before showing error
		if btpd, ok := progressDisplay.(*display.BubbleTeaProgressDisplay); ok {
			btpd.Clear()
		}
		return fmt.Errorf("pipeline execution failed: %w", execErr)
	}

	elapsed := time.Since(pipelineStart)

	// Clear enhanced progress display before final message
	if btpd, ok := progressDisplay.(*display.BubbleTeaProgressDisplay); ok {
		btpd.Clear()
	}

	// Add spacing after Press section and ensure clean cursor position
	fmt.Print("\r")  // Move to start of line
	fmt.Print("\n")  // Space after Press section
	fmt.Printf("  ✓ Pipeline '%s' completed successfully (%.1fs)\n", p.Metadata.Name, elapsed.Seconds())

	// Show deliverables summary with proper spacing and indentation
	if deliverables := executor.GetDeliverables(); deliverables != "" {
		fmt.Print("\n")
		// Add left padding to each line of deliverables
		lines := strings.Split(deliverables, "\n")
		for _, line := range lines {
			if line != "" {
				fmt.Printf("  %s\n", line)
			}
		}
		fmt.Print("\n") // Bottom spacing
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
