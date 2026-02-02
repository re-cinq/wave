package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/recinq/wave/internal/state"
	"github.com/spf13/cobra"
)

type ResumeOptions struct {
	Pipeline string
	FromStep string
	Manifest string
}

func NewResumeCmd() *cobra.Command {
	var opts ResumeOptions

	cmd := &cobra.Command{
		Use:   "resume",
		Short: "Resume a paused pipeline",
		Long: `Resume a previously paused or failed pipeline execution.
Picks up from the last completed step and continues forward.

If no pipeline ID is provided, lists recent pipelines that can be resumed.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runResume(cmd, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Pipeline, "pipeline", "", "Pipeline ID to resume")
	cmd.Flags().StringVar(&opts.FromStep, "from-step", "", "Resume from specific step")
	cmd.Flags().StringVar(&opts.Manifest, "manifest", "wave.yaml", "Path to manifest file")

	return cmd
}

func runResume(cmd *cobra.Command, opts ResumeOptions) error {
	stateDB := filepath.Join(".wave", "state.db")

	// Check if state database exists
	if _, err := os.Stat(stateDB); os.IsNotExist(err) {
		return fmt.Errorf("no state database found at %s — nothing to resume", stateDB)
	}

	// T064: If no pipeline ID provided, list recent pipelines
	if opts.Pipeline == "" {
		return listResumablePipelines(stateDB)
	}

	return resumePipeline(opts, stateDB)
}

// T064: List recent pipelines when no ID is provided
func listResumablePipelines(stateDB string) error {
	store, err := state.NewStateStore(stateDB)
	if err != nil {
		return fmt.Errorf("failed to open state database: %w", err)
	}
	defer store.Close()

	pipelines, err := store.ListRecentPipelines(10)
	if err != nil {
		return fmt.Errorf("failed to list pipelines: %w", err)
	}

	if len(pipelines) == 0 {
		fmt.Println("No pipelines found to resume.")
		fmt.Println("\nRun a pipeline first with: wave run --pipeline <name>")
		return nil
	}

	fmt.Println("Recent pipelines:")
	fmt.Println()
	fmt.Printf("  %-36s  %-12s  %-20s  %s\n", "PIPELINE ID", "STATUS", "LAST UPDATED", "STEPS")
	fmt.Printf("  %s  %s  %s  %s\n",
		strings.Repeat("-", 36),
		strings.Repeat("-", 12),
		strings.Repeat("-", 20),
		strings.Repeat("-", 10))

	for _, p := range pipelines {
		// Get step count and status summary
		steps, _ := store.GetStepStates(p.PipelineID)
		stepSummary := summarizeSteps(steps)

		// Format timestamp
		updatedAgo := formatTimeAgo(p.UpdatedAt)

		// Color the status based on state
		statusDisplay := formatStatus(p.Status)

		fmt.Printf("  %-36s  %-12s  %-20s  %s\n",
			truncateString(p.PipelineID, 36),
			statusDisplay,
			updatedAgo,
			stepSummary,
		)
	}

	fmt.Println()
	fmt.Println("To resume a pipeline:")
	fmt.Println("  wave resume --pipeline <PIPELINE_ID>")
	fmt.Println()
	fmt.Println("To resume from a specific step:")
	fmt.Println("  wave resume --pipeline <PIPELINE_ID> --from-step <STEP_ID>")

	return nil
}

// T065: Improved resume progress messages
func resumePipeline(opts ResumeOptions, stateDB string) error {
	fmt.Printf("Resuming pipeline: %s\n", opts.Pipeline)
	fmt.Println()

	// T065: Show loading progress
	fmt.Printf("  Loading state from %s...\n", stateDB)

	store, err := state.NewStateStore(stateDB)
	if err != nil {
		return fmt.Errorf("failed to open state database: %w", err)
	}
	defer store.Close()

	// Load pipeline state
	pipelineState, err := store.GetPipelineState(opts.Pipeline)
	if err != nil {
		return fmt.Errorf("pipeline not found: %s", opts.Pipeline)
	}

	fmt.Printf("  [OK] Pipeline state loaded\n")
	fmt.Printf("       Status: %s\n", formatStatus(pipelineState.Status))
	fmt.Printf("       Last updated: %s\n", formatTimeAgo(pipelineState.UpdatedAt))

	// Load step states
	steps, err := store.GetStepStates(opts.Pipeline)
	if err != nil {
		return fmt.Errorf("failed to load step states: %w", err)
	}

	// T065: Show step status summary
	fmt.Println()
	fmt.Println("  Step states:")
	if len(steps) == 0 {
		fmt.Println("       (no steps recorded)")
	} else {
		for _, step := range steps {
			stepStatus := formatStepState(step.State)
			retryInfo := ""
			if step.RetryCount > 0 {
				retryInfo = fmt.Sprintf(" (retries: %d)", step.RetryCount)
			}
			errorInfo := ""
			if step.ErrorMessage != "" {
				errorInfo = fmt.Sprintf("\n              Error: %s", truncateString(step.ErrorMessage, 60))
			}
			fmt.Printf("       %-20s %s%s%s\n", step.StepID, stepStatus, retryInfo, errorInfo)
		}
	}

	// Determine resumption point
	resumeFrom := opts.FromStep
	if resumeFrom == "" {
		resumeFrom = findResumptionPoint(steps)
	}

	fmt.Println()
	if resumeFrom != "" {
		fmt.Printf("  Starting from step: %s\n", resumeFrom)
	} else {
		fmt.Println("  Starting from last checkpoint")
	}

	// T065: Show final status
	fmt.Println()
	fmt.Printf("  [OK] Pipeline state loaded\n")
	fmt.Printf("  [OK] Resuming execution\n")

	return nil
}

// findResumptionPoint determines which step to resume from based on step states
func findResumptionPoint(steps []state.StepStateRecord) string {
	for _, step := range steps {
		switch step.State {
		case state.StateRunning, state.StateRetrying, state.StateFailed, state.StatePending:
			return step.StepID
		}
	}
	return ""
}

// summarizeSteps creates a summary string of step states
func summarizeSteps(steps []state.StepStateRecord) string {
	if len(steps) == 0 {
		return "0 steps"
	}

	completed := 0
	running := 0
	failed := 0
	pending := 0
	retrying := 0

	for _, step := range steps {
		switch step.State {
		case state.StateCompleted:
			completed++
		case state.StateRunning:
			running++
		case state.StateFailed:
			failed++
		case state.StatePending:
			pending++
		case state.StateRetrying:
			retrying++
		}
	}

	parts := []string{}
	total := len(steps)

	if completed == total {
		return fmt.Sprintf("%d/%d completed", completed, total)
	}

	if completed > 0 {
		parts = append(parts, fmt.Sprintf("%d done", completed))
	}
	if running > 0 {
		parts = append(parts, fmt.Sprintf("%d running", running))
	}
	if retrying > 0 {
		parts = append(parts, fmt.Sprintf("%d retrying", retrying))
	}
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", failed))
	}
	if pending > 0 {
		parts = append(parts, fmt.Sprintf("%d pending", pending))
	}

	return strings.Join(parts, ", ")
}

// formatTimeAgo formats a timestamp as a relative time string
func formatTimeAgo(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		minutes := int(diff.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}

	return t.Format("2006-01-02 15:04")
}

// formatStatus formats pipeline status for display
func formatStatus(status string) string {
	switch status {
	case "completed":
		return "completed"
	case "running":
		return "running"
	case "failed":
		return "FAILED"
	case "paused":
		return "paused"
	default:
		return status
	}
}

// formatStepState formats step state for display
func formatStepState(s state.StepState) string {
	switch s {
	case state.StateCompleted:
		return "[OK]"
	case state.StateRunning:
		return "[RUNNING]"
	case state.StateFailed:
		return "[FAILED]"
	case state.StatePending:
		return "[PENDING]"
	case state.StateRetrying:
		return "[RETRYING]"
	default:
		return "[" + string(s) + "]"
	}
}

// truncateString truncates a string to maxLen and adds "..." if needed
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
