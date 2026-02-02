package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/recinq/wave/internal/state"
	"github.com/spf13/cobra"
)

// CancelOptions holds options for the cancel command
type CancelOptions struct {
	RunID  string // Specific run to cancel (default: most recent running)
	Force  bool   // Interrupt immediately vs wait for step
	Format string // Output format (text, json)
}

// CancelResult represents the result of a cancel operation for JSON output
type CancelResult struct {
	Success      bool   `json:"success"`
	RunID        string `json:"run_id,omitempty"`
	PipelineName string `json:"pipeline_name,omitempty"`
	Message      string `json:"message"`
	Force        bool   `json:"force"`
}

// NewCancelCmd creates the cancel command
func NewCancelCmd() *cobra.Command {
	var opts CancelOptions

	cmd := &cobra.Command{
		Use:   "cancel [run-id]",
		Short: "Cancel a running pipeline",
		Long: `Cancel a running pipeline execution.

Without a run-id, cancels the most recently started running pipeline.
With a run-id, cancels that specific run.

Graceful cancellation (default):
  - Sets a cancellation flag in the database
  - The executor will stop after the current step completes
  - The pipeline status is marked as "cancelled"

Force cancellation (--force):
  - Immediately sends SIGTERM to the adapter process group
  - Waits 5 seconds, then sends SIGKILL if still running
  - The current step may be incomplete

Examples:
  wave cancel                    # Cancel most recent running pipeline
  wave cancel abc123             # Cancel specific run
  wave cancel --force            # Forcibly terminate immediately
  wave cancel --format json      # Output result as JSON`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.RunID = args[0]
			}
			return runCancel(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Interrupt immediately (send SIGTERM/SIGKILL)")
	cmd.Flags().StringVar(&opts.Format, "format", "text", "Output format (text, json)")

	return cmd
}

func runCancel(opts CancelOptions) error {
	// Initialize state store
	stateDB := ".wave/state.db"
	store, err := state.NewStateStore(stateDB)
	if err != nil {
		return outputCancelResult(opts.Format, CancelResult{
			Success: false,
			Message: fmt.Sprintf("Failed to open state database: %v", err),
		})
	}
	defer store.Close()

	var targetRun *state.RunRecord

	if opts.RunID != "" {
		// Cancel specific run
		run, err := store.GetRun(opts.RunID)
		if err != nil {
			return outputCancelResult(opts.Format, CancelResult{
				Success: false,
				RunID:   opts.RunID,
				Message: fmt.Sprintf("Run not found: %s", opts.RunID),
			})
		}
		if run.Status != "running" {
			return outputCancelResult(opts.Format, CancelResult{
				Success:      false,
				RunID:        opts.RunID,
				PipelineName: run.PipelineName,
				Message:      fmt.Sprintf("Run %s is not running (status: %s)", opts.RunID, run.Status),
			})
		}
		targetRun = run
	} else {
		// Find most recent running pipeline
		runs, err := store.GetRunningRuns()
		if err != nil {
			return outputCancelResult(opts.Format, CancelResult{
				Success: false,
				Message: fmt.Sprintf("Failed to query running pipelines: %v", err),
			})
		}
		if len(runs) == 0 {
			return outputCancelResult(opts.Format, CancelResult{
				Success: false,
				Message: "No running pipelines to cancel",
			})
		}
		// Get most recently started running pipeline
		targetRun = &runs[0]
		for i := range runs {
			if runs[i].StartedAt.After(targetRun.StartedAt) {
				targetRun = &runs[i]
			}
		}
	}

	// Request cancellation
	if err := store.RequestCancellation(targetRun.RunID, opts.Force); err != nil {
		return outputCancelResult(opts.Format, CancelResult{
			Success:      false,
			RunID:        targetRun.RunID,
			PipelineName: targetRun.PipelineName,
			Message:      fmt.Sprintf("Failed to request cancellation: %v", err),
		})
	}

	// For force cancellation, try to kill the process group
	if opts.Force {
		if err := forceKillRun(targetRun.RunID); err != nil {
			// Log warning but don't fail - the cancellation flag is still set
			fmt.Fprintf(os.Stderr, "Warning: could not forcibly terminate process: %v\n", err)
		}
	}

	// Update run status to cancelled
	if err := store.UpdateRunStatus(targetRun.RunID, "cancelled", targetRun.CurrentStep, targetRun.TotalTokens); err != nil {
		// Log warning but don't fail - cancellation was requested
		fmt.Fprintf(os.Stderr, "Warning: could not update run status: %v\n", err)
	}

	message := fmt.Sprintf("Cancellation requested for pipeline '%s' (run %s)", targetRun.PipelineName, targetRun.RunID)
	if opts.Force {
		message = fmt.Sprintf("Force cancellation sent to pipeline '%s' (run %s)", targetRun.PipelineName, targetRun.RunID)
	}

	return outputCancelResult(opts.Format, CancelResult{
		Success:      true,
		RunID:        targetRun.RunID,
		PipelineName: targetRun.PipelineName,
		Message:      message,
		Force:        opts.Force,
	})
}

func outputCancelResult(format string, result CancelResult) error {
	if format == "json" {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal result: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Text format
	if result.Success {
		fmt.Println(result.Message)
	} else {
		return fmt.Errorf("%s", result.Message)
	}
	return nil
}

// forceKillRun attempts to terminate the process group associated with a run.
// It reads the PID from a pidfile if available, sends SIGTERM, waits 5 seconds,
// then sends SIGKILL if the process is still running.
func forceKillRun(runID string) error {
	// Validate runID to prevent path traversal
	if strings.Contains(runID, "..") || strings.ContainsAny(runID, `/\`) || filepath.IsAbs(runID) {
		return fmt.Errorf("invalid run ID: %s", runID)
	}

	// Try to read PID from pidfile
	pidFile := filepath.Join(".wave", "pids", runID+".pid")
	pidData, err := os.ReadFile(pidFile)
	if err != nil {
		// No pidfile - the process may have already exited or we don't track it
		return nil
	}

	var pid int
	if _, err := fmt.Sscanf(string(pidData), "%d", &pid); err != nil {
		return fmt.Errorf("invalid pid in %s: %w", pidFile, err)
	}

	// Send SIGTERM to the process group
	if err := syscall.Kill(-pid, syscall.SIGTERM); err != nil {
		// Process may have already exited
		if err == syscall.ESRCH {
			os.Remove(pidFile)
			return nil
		}
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	// Wait up to 5 seconds for graceful termination
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		// Check if process is still running
		if err := syscall.Kill(-pid, 0); err == syscall.ESRCH {
			os.Remove(pidFile)
			return nil
		}
	}

	// Process still running after 5 seconds - send SIGKILL
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
		if err != syscall.ESRCH {
			return fmt.Errorf("failed to send SIGKILL: %w", err)
		}
	}

	os.Remove(pidFile)
	return nil
}
