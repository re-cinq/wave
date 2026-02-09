package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/workspace"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type DoOptions struct {
	Persona  string
	Manifest string
	Mock     bool
	DryRun   bool
	Output   OutputConfig
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

	executePersona := opts.Persona
	if executePersona == "" {
		executePersona = "craftsman"
	}

	adHocOpts := pipeline.AdHocOptions{
		Input:          input,
		ExecutePersona: executePersona,
		Manifest:       &m,
	}

	p, err := pipeline.GenerateAdHocPipeline(adHocOpts)
	if err != nil {
		return fmt.Errorf("failed to generate pipeline: %w", err)
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

	result := CreateEmitter(opts.Output, "adhoc", p.Steps, &m)
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

	executor := pipeline.NewDefaultPipelineExecutor(runner, execOpts...)

	timeout := m.Runtime.GetDefaultTimeout()
	execCtx, execCancel := context.WithTimeout(ctx, timeout)
	defer execCancel()

	pipelineStart := time.Now()

	if err := executor.Execute(execCtx, p, &m, input); err != nil {
		return fmt.Errorf("ad-hoc execution failed: %w", err)
	}

	elapsed := time.Since(pipelineStart)
	if opts.Output.Format == OutputFormatAuto || opts.Output.Format == OutputFormatText {
		fmt.Fprintf(os.Stderr, "\nAd-hoc task completed (%.1fs)\n", elapsed.Seconds())
	}
	return nil
}
