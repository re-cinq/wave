package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/workspace"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type MetaOptions struct {
	Save     string
	Manifest string
	Mock     bool
	DryRun   bool
}

func NewMetaCmd() *cobra.Command {
	var opts MetaOptions

	cmd := &cobra.Command{
		Use:   "meta [task description]",
		Short: "Generate and run a custom pipeline dynamically",
		Long: `Generate a custom multi-step pipeline using the philosopher persona,
then execute it. The philosopher analyzes your task and designs an
appropriate pipeline with steps, personas, and contracts.

Examples:
  wave meta "implement user authentication"
  wave meta "add database migrations"
  wave meta "create REST API endpoints"

  # Preview without executing
  wave meta "add caching layer" --dry-run

  # Save the generated pipeline for later use
  wave meta "add monitoring" --save monitoring-pipeline.yaml`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := strings.Join(args, " ")
			return runMeta(input, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Save, "save", "", "Save generated pipeline YAML to path")
	cmd.Flags().StringVar(&opts.Manifest, "manifest", "wave.yaml", "Path to manifest file")
	cmd.Flags().BoolVar(&opts.Mock, "mock", false, "Use mock adapter (for testing)")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Show generated pipeline without executing")

	return cmd
}

func runMeta(input string, opts MetaOptions) error {
	manifestData, err := os.ReadFile(opts.Manifest)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("manifest file not found: %s\nRun 'wave init' to create a new Wave project or specify --manifest path", opts.Manifest)
		}
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	var m manifest.Manifest
	if err := yaml.Unmarshal(manifestData, &m); err != nil {
		return fmt.Errorf("failed to parse manifest %s: %w\nEnsure the file is valid YAML with correct indentation", opts.Manifest, err)
	}

	// Set up context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		cancel()
	}()

	// Resolve adapter
	var runner adapter.AdapterRunner
	if opts.Mock {
		runner = adapter.NewMockAdapter()
	} else {
		var adapterName string
		for name := range m.Adapters {
			adapterName = name
			break
		}
		runner = adapter.ResolveAdapter(adapterName)
	}

	emitter := event.NewNDJSONEmitterWithHumanReadable()

	// Set up workspace manager
	wsRoot := m.Runtime.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = ".wave/workspaces"
	}
	wsManager, _ := workspace.NewWorkspaceManager(wsRoot)

	// Create child executor for running generated pipelines
	execOpts := []pipeline.ExecutorOption{
		pipeline.WithEmitter(emitter),
	}
	if wsManager != nil {
		execOpts = append(execOpts, pipeline.WithWorkspaceManager(wsManager))
	}
	childExecutor := pipeline.NewDefaultPipelineExecutor(runner, execOpts...)

	// Create meta-pipeline executor
	metaOpts := []pipeline.MetaExecutorOption{
		pipeline.WithMetaEmitter(emitter),
		pipeline.WithChildExecutor(childExecutor),
	}
	metaExecutor := pipeline.NewMetaPipelineExecutor(runner, metaOpts...)

	// Handle dry-run mode
	if opts.DryRun {
		fmt.Printf("Invoking philosopher to generate pipeline...\n")
		fmt.Printf("This may take a moment while the AI designs your pipeline.\n\n")
		p, err := metaExecutor.GenerateOnly(ctx, input, &m)
		if err != nil {
			return fmt.Errorf("meta pipeline generation failed: %w", err)
		}

		// Save if requested
		if opts.Save != "" {
			if err := saveMetaPipeline(p, opts.Save); err != nil {
				return err
			}
		}

		fmt.Printf("Generated pipeline: %s\n", p.Metadata.Name)
		fmt.Printf("Description: %s\n", p.Metadata.Description)
		fmt.Printf("\nSteps:\n")
		for i, step := range p.Steps {
			deps := ""
			if len(step.Dependencies) > 0 {
				deps = fmt.Sprintf(" (depends on: %s)", strings.Join(step.Dependencies, ", "))
			}
			fmt.Printf("  %d. %s [%s]%s\n", i+1, step.ID, step.Persona, deps)
		}
		fmt.Printf("\nWorkspace: .wave/workspaces/meta/\n")
		return nil
	}

	// Apply timeout from meta-pipeline config
	timeout := m.Runtime.GetDefaultTimeout()
	if m.Runtime.MetaPipeline.TimeoutMin > 0 {
		timeout = time.Duration(m.Runtime.MetaPipeline.TimeoutMin) * time.Minute
	}
	execCtx, execCancel := context.WithTimeout(ctx, timeout)
	defer execCancel()

	pipelineStart := time.Now()

	// Execute meta-pipeline
	result, err := metaExecutor.Execute(execCtx, input, &m)
	if err != nil {
		return fmt.Errorf("meta pipeline execution failed: %w", err)
	}

	// Save generated pipeline if requested
	if opts.Save != "" && result.GeneratedPipeline != nil {
		if err := saveMetaPipeline(result.GeneratedPipeline, opts.Save); err != nil {
			return err
		}
	}

	elapsed := time.Since(pipelineStart)
	fmt.Printf("\nMeta pipeline completed (%.1fs)\n", elapsed.Seconds())
	fmt.Printf("  Total steps: %d, Total tokens: %d\n", result.TotalSteps, result.TotalTokens)
	return nil
}

func saveMetaPipeline(p *pipeline.Pipeline, savePath string) error {
	if !strings.Contains(savePath, "/") {
		savePath = ".wave/pipelines/" + savePath
		if !strings.HasSuffix(savePath, ".yaml") {
			savePath += ".yaml"
		}
	}
	// Create parent directories for the save path
	if dir := filepath.Dir(savePath); dir != "" {
		os.MkdirAll(dir, 0755)
	}
	data, err := yaml.Marshal(p)
	if err != nil {
		return fmt.Errorf("failed to marshal pipeline: %w", err)
	}
	if err := os.WriteFile(savePath, data, 0644); err != nil {
		return fmt.Errorf("failed to save pipeline: %w", err)
	}
	fmt.Printf("Pipeline saved to %s\n", savePath)
	return nil
}
