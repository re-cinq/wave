package state

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func newReconcileStore(t *testing.T) *stateStore {
	t.Helper()
	dir := t.TempDir()
	store, err := NewStateStore(filepath.Join(dir, "wave.db"))
	if err != nil {
		t.Fatalf("NewStateStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store.(*stateStore)
}

func TestReconcileZombies_AgeBased(t *testing.T) {
	store := newReconcileStore(t)

	freshID, err := store.CreateRun("p", "in")
	if err != nil {
		t.Fatalf("CreateRun fresh: %v", err)
	}
	if err := store.UpdateRunStatus(freshID, "running", "", 0); err != nil {
		t.Fatalf("UpdateRunStatus fresh: %v", err)
	}
	staleID, err := store.CreateRun("p", "in")
	if err != nil {
		t.Fatalf("CreateRun stale: %v", err)
	}
	if err := store.UpdateRunStatus(staleID, "running", "", 0); err != nil {
		t.Fatalf("UpdateRunStatus stale: %v", err)
	}
	if _, err := store.db.Exec(
		`UPDATE pipeline_run SET started_at = ? WHERE run_id = ?`,
		time.Now().Add(-30*time.Minute).Unix(), staleID,
	); err != nil {
		t.Fatalf("seed stale started_at: %v", err)
	}

	if got := ReconcileZombies(store, ZombieAgeThreshold); got != 1 {
		t.Fatalf("ReconcileZombies = %d, want 1", got)
	}

	freshStatus, err := store.GetRunStatus(freshID)
	if err != nil {
		t.Fatalf("GetRunStatus fresh: %v", err)
	}
	if freshStatus != "running" {
		t.Errorf("fresh run status = %q, want running", freshStatus)
	}
	staleStatus, err := store.GetRunStatus(staleID)
	if err != nil {
		t.Fatalf("GetRunStatus stale: %v", err)
	}
	if staleStatus != "failed" {
		t.Errorf("stale run status = %q, want failed", staleStatus)
	}
}

func TestReconcileZombies_PIDLive(t *testing.T) {
	store := newReconcileStore(t)
	runID, err := store.CreateRun("p", "in")
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	if err := store.UpdateRunStatus(runID, "running", "", 0); err != nil {
		t.Fatalf("UpdateRunStatus running: %v", err)
	}
	if err := store.UpdateRunPID(runID, os.Getpid()); err != nil {
		t.Fatalf("UpdateRunPID: %v", err)
	}
	if _, err := store.db.Exec(
		`UPDATE pipeline_run SET started_at = ? WHERE run_id = ?`,
		time.Now().Add(-30*time.Minute).Unix(), runID,
	); err != nil {
		t.Fatalf("seed started_at: %v", err)
	}

	if got := ReconcileZombies(store, ZombieAgeThreshold); got != 0 {
		t.Fatalf("ReconcileZombies (live PID) = %d, want 0", got)
	}
	status, _ := store.GetRunStatus(runID)
	if status != "running" {
		t.Errorf("status = %q, want running (live PID must not be reaped)", status)
	}
}

func TestReconcileZombies_PIDGone(t *testing.T) {
	store := newReconcileStore(t)
	runID, err := store.CreateRun("p", "in")
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	if err := store.UpdateRunStatus(runID, "running", "", 0); err != nil {
		t.Fatalf("UpdateRunStatus running: %v", err)
	}
	deadPID := findDeadPID(t)
	if err := store.UpdateRunPID(runID, deadPID); err != nil {
		t.Fatalf("UpdateRunPID: %v", err)
	}

	if got := ReconcileZombies(store, ZombieAgeThreshold); got != 1 {
		t.Fatalf("ReconcileZombies (dead PID) = %d, want 1", got)
	}
	status, _ := store.GetRunStatus(runID)
	if status != "failed" {
		t.Errorf("status = %q, want failed", status)
	}
}

// findDeadPID picks a PID that is not currently in use. Skips the test if a
// dead PID cannot be located (extremely unlikely on a typical system).
func findDeadPID(t *testing.T) int {
	t.Helper()
	for pid := 99999; pid > 90000; pid-- {
		if err := syscall.Kill(pid, 0); err == syscall.ESRCH {
			return pid
		}
	}
	t.Skip("could not locate dead PID for test")
	return 0
}
