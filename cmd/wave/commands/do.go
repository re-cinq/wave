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

type DoOptions struct {
	Persona  string
	Save     string
	Manifest string
	Mock     bool
	DryRun   bool
	UseMeta  bool
}

func NewDoCmd() *cobra.Command {
	var opts DoOptions

	cmd := &cobra.Command{
		Use:   "do [task description]",
		Short: "Execute an ad-hoc task",
		Long: `Generate and run a minimal navigate→execute pipeline for a one-off task.
The task description is passed as arguments or via stdin.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := strings.Join(args, " ")
			return runDo(input, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Persona, "persona", "", "Override execute persona")
	cmd.Flags().StringVar(&opts.Save, "save", "", "Save generated pipeline YAML to path")
	cmd.Flags().StringVar(&opts.Manifest, "manifest", "wave.yaml", "Path to manifest file")
	cmd.Flags().BoolVar(&opts.Mock, "mock", false, "Use mock adapter (for testing)")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Show what would be executed without running")
	cmd.Flags().BoolVar(&opts.UseMeta, "meta", false, "Generate pipeline dynamically using philosopher persona")

	return cmd
}

func runDo(input string, opts DoOptions) error {
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

	if opts.UseMeta {
		return runMetaDo(input, opts, &m)
	}

	return runAdHocDo(input, opts, &m)
}

func runMetaDo(input string, opts DoOptions, m *manifest.Manifest) error {
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
		p, err := metaExecutor.GenerateOnly(ctx, input, m)
		if err != nil {
			return fmt.Errorf("meta-pipeline generation failed: %w", err)
		}

		// Save if requested
		if opts.Save != "" {
			if err := savePipeline(p, opts.Save); err != nil {
				return err
			}
		}

		fmt.Printf("Meta-pipeline: philosopher → generated pipeline\n")
		fmt.Printf("  Input: %s\n", input)
		fmt.Printf("  Generated pipeline: %s\n", p.Metadata.Name)
		fmt.Printf("  Steps:\n")
		for i, step := range p.Steps {
			fmt.Printf("    %d. %s (persona: %s)\n", i+1, step.ID, step.Persona)
		}
		fmt.Printf("  Workspace: .wave/workspaces/meta/\n")
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
	result, err := metaExecutor.Execute(execCtx, input, m)
	if err != nil {
		return fmt.Errorf("meta-pipeline execution failed: %w", err)
	}

	// Save generated pipeline if requested
	if opts.Save != "" && result.GeneratedPipeline != nil {
		if err := savePipeline(result.GeneratedPipeline, opts.Save); err != nil {
			return err
		}
	}

	elapsed := time.Since(pipelineStart)
	fmt.Printf("\n✓ Meta-pipeline task completed (%.1fs)\n", elapsed.Seconds())
	fmt.Printf("  Total steps: %d, Total tokens: %d\n", result.TotalSteps, result.TotalTokens)
	return nil
}

func runAdHocDo(input string, opts DoOptions, m *manifest.Manifest) error {
	executePersona := opts.Persona
	if executePersona == "" {
		executePersona = "craftsman"
	}

	adHocOpts := pipeline.AdHocOptions{
		Input:          input,
		ExecutePersona: executePersona,
		Manifest:       m,
	}

	p, err := pipeline.GenerateAdHocPipeline(adHocOpts)
	if err != nil {
		return fmt.Errorf("failed to generate pipeline: %w", err)
	}

	if opts.Save != "" {
		if err := savePipeline(p, opts.Save); err != nil {
			return err
		}
	}

	if opts.DryRun {
		fmt.Printf("Ad-hoc pipeline: navigate → execute\n")
		fmt.Printf("  Input: %s\n", input)
		fmt.Printf("  Steps:\n")
		for i, step := range p.Steps {
			fmt.Printf("    %d. %s (persona: %s)\n", i+1, step.ID, step.Persona)
		}
		fmt.Printf("  Workspace: .wave/workspaces/adhoc/\n")
		return nil
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
		var adapterName string
		for name := range m.Adapters {
			adapterName = name
			break
		}
		runner = adapter.ResolveAdapter(adapterName)
	}

	emitter := event.NewNDJSONEmitterWithHumanReadable()

	wsRoot := m.Runtime.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = ".wave/workspaces"
	}
	wsManager, _ := workspace.NewWorkspaceManager(wsRoot)

	execOpts := []pipeline.ExecutorOption{
		pipeline.WithEmitter(emitter),
	}
	if wsManager != nil {
		execOpts = append(execOpts, pipeline.WithWorkspaceManager(wsManager))
	}

	executor := pipeline.NewDefaultPipelineExecutor(runner, execOpts...)

	timeout := m.Runtime.GetDefaultTimeout()
	execCtx, execCancel := context.WithTimeout(ctx, timeout)
	defer execCancel()

	pipelineStart := time.Now()

	if err := executor.Execute(execCtx, p, m, input); err != nil {
		return fmt.Errorf("ad-hoc execution failed: %w", err)
	}

	elapsed := time.Since(pipelineStart)
	fmt.Printf("\n✓ Ad-hoc task completed (%.1fs)\n", elapsed.Seconds())
	return nil
}

func savePipeline(p *pipeline.Pipeline, savePath string) error {
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
	fmt.Printf("✓ Pipeline saved to %s\n", savePath)
	return nil
}
