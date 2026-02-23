package commands

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/state"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	_ "modernc.org/sqlite"
)

// ChatOptions holds options for the chat command.
type ChatOptions struct {
	RunID    string
	Step     string
	Manifest string
	Model    string
	List     bool
	// Phase 2: step manipulation
	Continue string // --continue <step-id>: continue work in step's workspace
	Rewrite  string // --rewrite <step-id>: re-execute step with new prompt
	Extend   string // --extend <step-id>: add instructions to step
}

// NewChatCmd creates the chat command.
func NewChatCmd() *cobra.Command {
	var opts ChatOptions

	cmd := &cobra.Command{
		Use:   "chat [run-id]",
		Short: "Open interactive analysis of a pipeline run",
		Long: `Open an interactive Claude Code session with context from a completed
pipeline run. The session includes run summary, step results, artifact
inventory, and access to preserved step workspaces.

Without arguments, opens the most recent completed run.

  Analyze (read-only):
    wave chat                            # latest completed run
    wave chat <run-id>                   # specific run
    wave chat --list                     # pick from recent runs
    wave chat --step implement           # focus on one step
    wave chat --model opus               # override model

  Manipulate (read-write):
    wave chat --continue <step>          # resume work in step workspace
    wave chat --extend <step>            # add instructions to a step
    wave chat --rewrite <step>           # re-execute with new prompt`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.RunID = args[0]
			}
			return runChat(opts)
		},
	}

	cmd.Flags().StringVar(&opts.Step, "step", "", "Focus context on a specific step")
	cmd.Flags().StringVar(&opts.Manifest, "manifest", "wave.yaml", "Path to manifest file")
	cmd.Flags().StringVar(&opts.Model, "model", "", "Model to use (default: sonnet)")
	cmd.Flags().BoolVar(&opts.List, "list", false, "List recent runs")

	// Phase 2: step manipulation flags
	cmd.Flags().StringVar(&opts.Continue, "continue", "", "Continue work in a step's workspace (read-write)")
	cmd.Flags().StringVar(&opts.Rewrite, "rewrite", "", "Re-execute a step with modified prompt")
	cmd.Flags().StringVar(&opts.Extend, "extend", "", "Add supplementary instructions to a step")

	return cmd
}

func runChat(opts ChatOptions) error {
	dbPath := ".wave/state.db"

	// --list: show recent runs and exit
	if opts.List {
		return listRecentRunsForChat(dbPath)
	}

	// Open state store
	store, err := state.NewStateStore(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open state database: %w", err)
	}
	defer store.Close()

	// Resolve run ID
	runID := opts.RunID
	if runID == "" {
		runID, err = pipeline.MostRecentCompletedRunID(store)
		if err != nil {
			return fmt.Errorf("no runs found: %w", err)
		}
	}

	// Get run record to determine pipeline name
	run, err := store.GetRun(runID)
	if err != nil {
		return fmt.Errorf("run not found: %w", err)
	}

	// Load manifest
	manifestData, err := os.ReadFile(opts.Manifest)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}
	var m manifest.Manifest
	if err := yaml.Unmarshal(manifestData, &m); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Load pipeline definition
	p, err := loadPipeline(run.PipelineName, &m)
	if err != nil {
		return fmt.Errorf("failed to load pipeline %q: %w", run.PipelineName, err)
	}

	// Get project root
	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Build chat context
	chatCtx, err := pipeline.BuildChatContext(store, runID, p, projectRoot)
	if err != nil {
		return fmt.Errorf("failed to build chat context: %w", err)
	}

	// Phase 2: Step manipulation
	if opts.Continue != "" || opts.Rewrite != "" || opts.Extend != "" {
		controller := pipeline.NewStepController(store, opts.Model)

		if opts.Continue != "" {
			fmt.Fprintf(os.Stderr, "  Mode:     continue step %q\n\n", opts.Continue)
			return controller.ContinueStep(context.Background(), chatCtx, opts.Continue)
		}
		if opts.Extend != "" {
			// For extend, we need additional instructions from stdin or a prompt
			fmt.Fprintf(os.Stderr, "  Mode:     extend step %q\n\n", opts.Extend)
			return controller.ExtendStep(context.Background(), chatCtx, opts.Extend, "")
		}
		if opts.Rewrite != "" {
			fmt.Fprintf(os.Stderr, "  Mode:     rewrite step %q\n\n", opts.Rewrite)
			return controller.RewriteStep(context.Background(), chatCtx, opts.Rewrite, "")
		}
	}

	// Prepare workspace
	wsOpts := pipeline.ChatWorkspaceOptions{
		Model: opts.Model,
		Mode:  pipeline.ChatModeAnalysis,
	}
	wsPath, err := pipeline.PrepareChatWorkspace(chatCtx, wsOpts)
	if err != nil {
		return fmt.Errorf("failed to prepare chat workspace: %w", err)
	}

	// Print session header
	elapsed := ""
	if run.CompletedAt != nil {
		elapsed = formatElapsed(run.CompletedAt.Sub(run.StartedAt))
	}
	fmt.Fprintf(os.Stderr, "\n  Wave Chat — %s%s%s\n", statusColor(run.Status), run.Status, colorReset)
	fmt.Fprintf(os.Stderr, "  Run:      %s\n", run.RunID)
	fmt.Fprintf(os.Stderr, "  Pipeline: %s\n", run.PipelineName)
	if elapsed != "" {
		fmt.Fprintf(os.Stderr, "  Duration: %s  Tokens: %s\n", elapsed, formatTokens(run.TotalTokens))
	}
	fmt.Fprintf(os.Stderr, "  Steps:    %d\n\n", len(chatCtx.Steps))

	// Build interactive options
	interactiveOpts := adapter.InteractiveOptions{
		Model:   opts.Model,
		AddDirs: []string{projectRoot},
	}

	// Add step workspace directories
	for _, step := range chatCtx.Steps {
		if step.WorkspacePath != "" {
			interactiveOpts.AddDirs = append(interactiveOpts.AddDirs, step.WorkspacePath)
		}
	}

	// Launch interactive Claude session
	return adapter.LaunchInteractive(wsPath, interactiveOpts)
}

// listRecentRunsForChat lists recent runs using the same DB query as status --all.
func listRecentRunsForChat(dbPath string) error {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Println("No pipeline runs found")
		return nil
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open state database: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	runs, err := queryRecentRuns(db, 10)
	if err != nil {
		return err
	}

	if len(runs) == 0 {
		fmt.Println("No pipeline runs found")
		return nil
	}

	fmt.Printf("\nRecent runs (use 'wave chat <run-id>' to open):\n\n")
	return outputRuns(runs, StatusOptions{Format: "table"})
}
