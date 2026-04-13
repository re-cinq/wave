package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/state"
	"github.com/spf13/cobra"
)

// ChatOptions holds options for the chat command.
type ChatOptions struct {
	RunID    string
	Step     string
	Artifact string
	Manifest string
	Model    string
	Prompt   string
	List     bool
	Resume   string // --resume <session-id|"last">: resume a previous chat session
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
    wave chat --artifact plan.json       # focus on a specific artifact
    wave chat --model opus               # override model
    wave chat --prompt "explain the plan"  # initial question

  Resume:
    wave chat --resume last              # resume most recent session for a run
    wave chat --resume <session-id>      # resume a specific session

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
	cmd.Flags().StringVar(&opts.Artifact, "artifact", "", "Focus on a specific artifact by name")
	cmd.Flags().StringVar(&opts.Manifest, "manifest", "wave.yaml", "Path to manifest file")
	cmd.Flags().StringVar(&opts.Model, "model", "", "Model to use (default: sonnet)")
	cmd.Flags().StringVar(&opts.Prompt, "prompt", "", "Initial prompt/question to send")
	cmd.Flags().BoolVar(&opts.List, "list", false, "List recent runs")
	cmd.Flags().StringVar(&opts.Resume, "resume", "", "Resume a previous chat session (session ID or 'last')")

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
		return NewCLIError(CodeStateDBError, fmt.Sprintf("failed to open state database: %s", err), "Check .wave/state.db file permissions or run 'wave run' to create it").WithCause(err)
	}
	defer store.Close()

	// Handle --resume: load existing session and resume it
	if opts.Resume != "" {
		return resumeChatSession(store, opts)
	}

	// Resolve run ID
	runID := opts.RunID
	if runID == "" {
		runID, err = pipeline.MostRecentCompletedRunID(store)
		if err != nil {
			return NewCLIError(CodeRunNotFound, fmt.Sprintf("no runs found: %s", err), "Run a pipeline first with 'wave run'").WithCause(err)
		}
	}

	// Get run record to determine pipeline name
	run, err := store.GetRun(runID)
	if err != nil {
		return NewCLIError(CodeRunNotFound, fmt.Sprintf("run not found: %s", err), "Use 'wave status --all' to list available runs").WithCause(err)
	}

	// Load manifest
	mp, err := loadManifestStrict(opts.Manifest)
	if err != nil {
		return err
	}
	m := *mp

	// Load pipeline definition
	p, err := loadPipeline(run.PipelineName, &m)
	if err != nil {
		return NewCLIError(CodePipelineNotFound, fmt.Sprintf("failed to load pipeline %q: %s", run.PipelineName, err), "The pipeline definition may have been removed or renamed").WithCause(err)
	}

	// Get project root
	projectRoot, err := os.Getwd()
	if err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to get working directory: %s", err), "Check working directory permissions").WithCause(err)
	}

	// Build chat context
	chatCtx, err := pipeline.BuildChatContext(store, runID, p, projectRoot)
	if err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to build chat context: %s", err), "Run state or workspace may be incomplete").WithCause(err)
	}

	// Phase 2: Step manipulation
	if opts.Continue != "" || opts.Rewrite != "" || opts.Extend != "" {
		controller := pipeline.NewStepController(store, opts.Model)

		if opts.Continue != "" {
			fmt.Fprintf(os.Stderr, "  Mode:     continue step %q\n\n", opts.Continue)
			return controller.ContinueStep(context.Background(), chatCtx, opts.Continue)
		}
		if opts.Extend != "" {
			fmt.Fprintf(os.Stderr, "  Mode:     extend step %q\n\n", opts.Extend)
			return controller.ExtendStep(context.Background(), chatCtx, opts.Extend, "")
		}
		if opts.Rewrite != "" {
			fmt.Fprintf(os.Stderr, "  Mode:     rewrite step %q\n\n", opts.Rewrite)
			return controller.RewriteStep(context.Background(), chatCtx, opts.Rewrite, "")
		}
	}

	// Validate --step if provided
	if opts.Step != "" {
		found := false
		for _, step := range chatCtx.Steps {
			if step.StepID == opts.Step {
				found = true
				break
			}
		}
		if !found {
			availableSteps := make([]string, len(chatCtx.Steps))
			for i, s := range chatCtx.Steps {
				availableSteps[i] = s.StepID
			}
			return NewCLIError(CodeInvalidArgs, fmt.Sprintf("step %q not found in pipeline run (available: %s)", opts.Step, strings.Join(availableSteps, ", ")), "Use one of the available step names listed above")
		}
	}

	// Validate --artifact if provided
	if opts.Artifact != "" {
		found := false
		for _, art := range chatCtx.Artifacts {
			if art.Name == opts.Artifact {
				found = true
				break
			}
		}
		if !found {
			availableArts := make([]string, len(chatCtx.Artifacts))
			for i, a := range chatCtx.Artifacts {
				availableArts[i] = a.Name
			}
			return NewCLIError(CodeInvalidArgs, fmt.Sprintf("artifact %q not found in pipeline run (available: %s)", opts.Artifact, strings.Join(availableArts, ", ")), "Use one of the available artifact names listed above")
		}
	}

	// Prepare workspace
	wsOpts := pipeline.ChatWorkspaceOptions{
		Model:        opts.Model,
		Mode:         pipeline.ChatModeAnalysis,
		StepFilter:   opts.Step,
		ArtifactName: opts.Artifact,
	}
	wsPath, err := pipeline.PrepareChatWorkspace(chatCtx, wsOpts)
	if err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to prepare chat workspace: %s", err), "Check workspace directory permissions").WithCause(err)
	}

	// Print session header
	elapsed := ""
	if run.CompletedAt != nil {
		elapsed = formatElapsed(run.CompletedAt.Sub(run.StartedAt))
	}
	fmt.Fprintf(os.Stderr, "\n  Wave Chat — %s%s%s\n", statusColor(run.Status), run.Status, conditionalColor("\033[0m"))
	fmt.Fprintf(os.Stderr, "  Run:      %s\n", run.RunID)
	fmt.Fprintf(os.Stderr, "  Pipeline: %s\n", run.PipelineName)
	if elapsed != "" {
		fmt.Fprintf(os.Stderr, "  Duration: %s  Tokens: %s\n", elapsed, formatTokens(run.TotalTokens))
	}
	fmt.Fprintf(os.Stderr, "  Steps:    %d\n", len(chatCtx.Steps))
	if opts.Step != "" {
		fmt.Fprintf(os.Stderr, "  Focus:    step %q\n", opts.Step)
	}
	if opts.Artifact != "" {
		fmt.Fprintf(os.Stderr, "  Artifact: %s\n", opts.Artifact)
	}
	fmt.Fprintf(os.Stderr, "\n")

	// Build interactive options
	interactiveOpts := adapter.InteractiveOptions{
		Model:   opts.Model,
		Prompt:  opts.Prompt,
		AddDirs: []string{projectRoot},
	}

	// Add step workspace directories
	if opts.Step != "" {
		for _, step := range chatCtx.Steps {
			if step.StepID == opts.Step && step.WorkspacePath != "" {
				interactiveOpts.AddDirs = append(interactiveOpts.AddDirs, step.WorkspacePath)
			}
		}
	} else {
		for _, step := range chatCtx.Steps {
			if step.WorkspacePath != "" {
				interactiveOpts.AddDirs = append(interactiveOpts.AddDirs, step.WorkspacePath)
			}
		}
	}

	// Launch interactive Claude session
	sessionID, err := adapter.LaunchInteractive(wsPath, interactiveOpts)

	// Save chat session record for future resume
	if sessionID != "" {
		session := &state.ChatSession{
			SessionID:     sessionID,
			RunID:         runID,
			StepFilter:    opts.Step,
			WorkspacePath: wsPath,
			Model:         opts.Model,
		}
		if saveErr := store.SaveChatSession(session); saveErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to save chat session: %v\n", saveErr)
		}
	}

	return err
}

// resumeChatSession loads a previous session and resumes it.
func resumeChatSession(store state.StateStore, opts ChatOptions) error {
	var session *state.ChatSession

	if opts.Resume == "last" {
		// Resolve run ID first
		runID := opts.RunID
		if runID == "" {
			var err error
			runID, err = pipeline.MostRecentCompletedRunID(store)
			if err != nil {
				return NewCLIError(CodeRunNotFound, fmt.Sprintf("no runs found: %s", err), "Run a pipeline first with 'wave run'").WithCause(err)
			}
		}

		sessions, err := store.ListChatSessions(runID)
		if err != nil {
			return NewCLIError(CodeInternalError, fmt.Sprintf("failed to list chat sessions: %s", err), "State database may be corrupted").WithCause(err)
		}
		if len(sessions) == 0 {
			return NewCLIError(CodeRunNotFound, fmt.Sprintf("no chat sessions found for run %s", runID), "Start a new chat session with 'wave chat <run-id>'")
		}
		session = &sessions[0] // most recent (ordered by created_at DESC)
	} else {
		var err error
		session, err = store.GetChatSession(opts.Resume)
		if err != nil {
			return NewCLIError(CodeRunNotFound, fmt.Sprintf("chat session not found: %s", err), "Use 'wave chat --list' to see available sessions").WithCause(err)
		}
	}

	// Update last_resumed_at
	now := time.Now()
	session.LastResumedAt = &now
	if err := store.SaveChatSession(session); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to update session timestamp: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "\n  Wave Chat — Resuming session\n")
	fmt.Fprintf(os.Stderr, "  Session:  %s\n", session.SessionID)
	fmt.Fprintf(os.Stderr, "  Run:      %s\n", session.RunID)
	if session.StepFilter != "" {
		fmt.Fprintf(os.Stderr, "  Step:     %s\n", session.StepFilter)
	}
	fmt.Fprintf(os.Stderr, "\n")

	projectRoot, err := os.Getwd()
	if err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to get working directory: %s", err), "Check working directory permissions").WithCause(err)
	}

	interactiveOpts := adapter.InteractiveOptions{
		Model:   session.Model,
		Resume:  session.SessionID,
		Prompt:  opts.Prompt,
		AddDirs: []string{projectRoot},
	}

	_, err = adapter.LaunchInteractive(session.WorkspacePath, interactiveOpts)
	return err
}

// listRecentRunsForChat lists recent runs and their chat sessions.
func listRecentRunsForChat(dbPath string) error {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Println("No pipeline runs found")
		return nil
	}

	store, err := state.NewStateStore(dbPath)
	if err != nil {
		return NewCLIError(CodeStateDBError, fmt.Sprintf("failed to open state database: %s", err), "Check .wave/state.db file permissions").WithCause(err)
	}
	defer store.Close()

	runs, err := store.ListRuns(state.ListRunsOptions{Limit: 10})
	if err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to list runs: %s", err), "State database query failed").WithCause(err)
	}

	if len(runs) == 0 {
		fmt.Println("No pipeline runs found")
		return nil
	}

	fmt.Printf("\nRecent runs (use 'wave chat <run-id>' to open):\n\n")
	for _, run := range runs {
		elapsed := ""
		if run.CompletedAt != nil {
			elapsed = formatElapsed(run.CompletedAt.Sub(run.StartedAt))
		}
		fmt.Printf("  %s%-9s%s  %s  %-20s  %s\n",
			statusColor(run.Status), run.Status, conditionalColor("\033[0m"),
			run.RunID, run.PipelineName, elapsed)

		// Show chat sessions for this run
		sessions, sessErr := store.ListChatSessions(run.RunID)
		if sessErr == nil && len(sessions) > 0 {
			for _, s := range sessions {
				resumed := ""
				if s.LastResumedAt != nil {
					resumed = fmt.Sprintf("  (resumed %s)", s.LastResumedAt.Format("15:04:05"))
				}
				fmt.Printf("    └─ session %s  %s%s\n",
					s.SessionID, s.CreatedAt.Format("2006-01-02 15:04:05"), resumed)
			}
		}
	}
	fmt.Println()
	return nil
}
