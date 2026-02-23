package commands

import (
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
}

// NewChatCmd creates the chat command.
func NewChatCmd() *cobra.Command {
	var opts ChatOptions

	cmd := &cobra.Command{
		Use:   "chat [run-id]",
		Short: "Open interactive analysis of a pipeline run",
		Long: `Open an interactive Claude Code session with context from a completed pipeline run.

Claude is launched with read-only access to run artifacts, step workspaces,
and the project source code. The CLAUDE.md in the session contains the full
run summary, step results, and artifact inventory.

Without arguments, opens the most recent completed run.
With --list, shows recent runs to choose from.

Examples:
  wave chat                              # Most recent completed run
  wave chat speckit-flow-20260223-140000  # Specific run
  wave chat --list                        # List recent runs
  wave chat --step implement             # Focus on a specific step
  wave chat --model opus                 # Use a specific model`,
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

	// Prepare workspace
	wsOpts := pipeline.ChatWorkspaceOptions{
		Model: opts.Model,
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
