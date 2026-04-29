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
	"github.com/recinq/wave/internal/adapter/adaptertest"
	"github.com/recinq/wave/internal/ontology"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/skill"
	"github.com/recinq/wave/internal/workspace"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type MetaOptions struct {
	Save     string
	Manifest string
	Mock     bool
	DryRun   bool
	Output   OutputConfig
	Model    string
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
			opts.Output = GetOutputConfig(cmd)
			if err := ValidateOutputFormat(opts.Output.Format); err != nil {
				return err
			}
			input := strings.Join(args, " ")
			return runMeta(input, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Save, "save", "", "Save generated pipeline YAML to path")
	cmd.Flags().StringVar(&opts.Manifest, "manifest", "wave.yaml", "Path to manifest file")
	cmd.Flags().BoolVar(&opts.Mock, "mock", false, "Use mock adapter (for testing)")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Show generated pipeline without executing")
	cmd.Flags().StringVar(&opts.Model, "model", "", "Override adapter model for this run (e.g. haiku, opus)")

	return cmd
}

func runMeta(input string, opts MetaOptions) error {
	mp, err := loadManifestStrict(opts.Manifest)
	if err != nil {
		return err
	}
	m := *mp

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
		runner = adaptertest.NewMockAdapter()
	} else {
		var adapterName string
		for name := range m.Adapters {
			adapterName = name
			break
		}
		runner = adapter.ResolveAdapter(adapterName)
	}

	// Create emitter (no steps known yet for meta - generated dynamically)
	emitterResult := CreateEmitter(opts.Output, "meta", "meta", nil, &m)
	defer emitterResult.Cleanup()

	// Set up workspace manager
	wsRoot := m.Runtime.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = ".agents/workspaces"
	}
	wsManager, _ := workspace.NewWorkspaceManager(wsRoot)

	// Create child executor for running generated pipelines
	execOpts := []pipeline.ExecutorOption{
		pipeline.WithEmitter(emitterResult.Emitter),
		pipeline.WithOntologyService(ontology.NoOp{}),
	}
	if wsManager != nil {
		execOpts = append(execOpts, pipeline.WithWorkspaceManager(wsManager))
	}
	if opts.Model != "" {
		execOpts = append(execOpts, pipeline.WithModelOverride(opts.Model))
	}
	execOpts = append(execOpts, pipeline.WithSkillStore(skill.NewDirectoryStore(skill.DefaultSources()...)))
	childExecutor := pipeline.NewDefaultPipelineExecutor(runner, execOpts...)

	// Create meta-pipeline executor
	metaOpts := []pipeline.MetaExecutorOption{
		pipeline.WithMetaEmitter(emitterResult.Emitter),
		pipeline.WithChildExecutor(childExecutor),
	}
	if opts.Mock {
		metaOpts = append(metaOpts, pipeline.WithMockMode())
	}
	metaExecutor := pipeline.NewMetaPipelineExecutor(runner, metaOpts...)

	// Handle dry-run mode
	if opts.DryRun {
		fmt.Printf("Invoking philosopher to generate pipeline...\n")
		fmt.Printf("This may take a moment while the AI designs your pipeline.\n\n")
		p, err := metaExecutor.GenerateOnly(ctx, input, &m)
		if err != nil {
			return NewCLIError(CodeInternalError, fmt.Sprintf("meta pipeline generation failed: %s", err), "Check that the philosopher persona is configured in wave.yaml").WithCause(err)
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
		fmt.Printf("\nWorkspace: .agents/workspaces/meta/\n")
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
		return NewCLIError(CodeInternalError, fmt.Sprintf("meta pipeline execution failed: %s", err), "Run 'wave logs' to inspect execution details").WithCause(err)
	}

	// Save generated pipeline if requested
	if opts.Save != "" && result.GeneratedPipeline != nil {
		if err := saveMetaPipeline(result.GeneratedPipeline, opts.Save); err != nil {
			return err
		}
	}

	elapsed := time.Since(pipelineStart)
	if opts.Output.Format == OutputFormatAuto || opts.Output.Format == OutputFormatText {
		fmt.Fprintf(os.Stderr, "\nMeta pipeline completed (%.1fs)\n", elapsed.Seconds())
		fmt.Fprintf(os.Stderr, "  Total steps: %d, Total tokens: %d\n", result.TotalSteps, result.TotalTokens)
	}
	return nil
}

func saveMetaPipeline(p *pipeline.Pipeline, savePath string) error {
	if !strings.Contains(savePath, "/") {
		savePath = ".agents/pipelines/" + savePath
		if !strings.HasSuffix(savePath, ".yaml") {
			savePath += ".yaml"
		}
	}
	// Create parent directories for the save path
	if dir := filepath.Dir(savePath); dir != "" {
		_ = os.MkdirAll(dir, 0755)
	}
	data, err := yaml.Marshal(p)
	if err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to marshal pipeline: %s", err), "This is an internal serialization error -- please report a bug").WithCause(err)
	}
	if err := os.WriteFile(savePath, data, 0644); err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to save pipeline: %s", err), "Check write permissions for the target directory").WithCause(err)
	}
	fmt.Printf("Pipeline saved to %s\n", savePath)
	return nil
}
