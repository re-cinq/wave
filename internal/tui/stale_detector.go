package tui

import (
	"os"
	"syscall"

	"github.com/recinq/wave/internal/state"
)

// StaleRunDetector checks for dead subprocess PIDs and transitions stale runs to "failed".
type StaleRunDetector struct {
	store state.StateStore
}

// NewStaleRunDetector creates a new detector using the given state store.
func NewStaleRunDetector(store state.StateStore) *StaleRunDetector {
	return &StaleRunDetector{store: store}
}

// DetectStaleRuns queries all "running" runs, checks each PID for liveness,
// and transitions dead runs to "failed". Returns the run IDs that were marked stale.
func (d *StaleRunDetector) DetectStaleRuns() ([]string, error) {
	runs, err := d.store.GetRunningRuns()
	if err != nil {
		return nil, err
	}

	var staleIDs []string
	for _, run := range runs {
		// Skip runs with no PID (in-process or legacy runs)
		if run.PID == 0 {
			continue
		}

		if !IsProcessAlive(run.PID) {
			_ = d.store.UpdateRunStatus(run.RunID, "failed", "stale: subprocess exited unexpectedly", run.TotalTokens)
			staleIDs = append(staleIDs, run.RunID)
		}
	}

	return staleIDs, nil
}

// IsProcessAlive checks if a process with the given PID is still running.
// Uses os.FindProcess + signal 0 (the standard Unix approach for liveness checks).
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Signal 0 doesn't actually send a signal — it just checks if the process
	// exists and the caller has permission to signal it.
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}
