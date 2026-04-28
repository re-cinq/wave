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

func TestReconcileZombies_HeartbeatFresh(t *testing.T) {
	store := newReconcileStore(t)
	runID, err := store.CreateRun("p", "in")
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	if err := store.UpdateRunStatus(runID, "running", "", 0); err != nil {
		t.Fatalf("UpdateRunStatus: %v", err)
	}
	if err := store.UpdateRunHeartbeat(runID); err != nil {
		t.Fatalf("UpdateRunHeartbeat: %v", err)
	}
	// Fresh heartbeat must keep the run alive even with a dead tracked PID
	// and an old started_at — heartbeat is the priority signal.
	if err := store.UpdateRunPID(runID, findDeadPID(t)); err != nil {
		t.Fatalf("UpdateRunPID: %v", err)
	}
	if _, err := store.db.Exec(
		`UPDATE pipeline_run SET started_at = ? WHERE run_id = ?`,
		time.Now().Add(-1*time.Hour).Unix(), runID,
	); err != nil {
		t.Fatalf("seed started_at: %v", err)
	}

	if got := ReconcileZombies(store, ZombieAgeThreshold); got != 0 {
		t.Fatalf("ReconcileZombies (fresh heartbeat) = %d, want 0", got)
	}
	status, _ := store.GetRunStatus(runID)
	if status != "running" {
		t.Errorf("status = %q, want running (fresh heartbeat must outrank PID/age)", status)
	}
}

func TestReconcileZombies_HeartbeatStale(t *testing.T) {
	store := newReconcileStore(t)
	runID, err := store.CreateRun("p", "in")
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	if err := store.UpdateRunStatus(runID, "running", "", 0); err != nil {
		t.Fatalf("UpdateRunStatus: %v", err)
	}
	// Stale heartbeat (older than HeartbeatStaleThreshold) must reap even
	// when the tracked PID is alive — process may be wedged.
	if err := store.UpdateRunPID(runID, os.Getpid()); err != nil {
		t.Fatalf("UpdateRunPID: %v", err)
	}
	if _, err := store.db.Exec(
		`UPDATE pipeline_run SET last_heartbeat = ? WHERE run_id = ?`,
		time.Now().Add(-5*time.Minute).Unix(), runID,
	); err != nil {
		t.Fatalf("seed last_heartbeat: %v", err)
	}

	if got := ReconcileZombies(store, ZombieAgeThreshold); got != 1 {
		t.Fatalf("ReconcileZombies (stale heartbeat) = %d, want 1", got)
	}
	status, _ := store.GetRunStatus(runID)
	if status != "failed" {
		t.Errorf("status = %q, want failed (stale heartbeat must reap even with live PID)", status)
	}
}

// TestReconcileZombies_SubPipelineChildSpared is a regression test: a child
// run row with a non-empty parent_run_id must NEVER be reaped, even when its
// PID/heartbeat columns are empty and started_at is older than the age
// threshold. Sub-pipelines execute in the parent process's goroutines and
// inherit the parent's liveness signal. Without this carve-out, the reaper
// silently kills active impl-finding / audit-* fan-out children at 5 minutes.
func TestReconcileZombies_SubPipelineChildSpared(t *testing.T) {
	store := newReconcileStore(t)
	parentID, err := store.CreateRun("ops-pr-respond", "1472")
	if err != nil {
		t.Fatalf("CreateRun parent: %v", err)
	}
	if err := store.UpdateRunStatus(parentID, "running", "", 0); err != nil {
		t.Fatalf("UpdateRunStatus parent: %v", err)
	}

	childID, err := store.CreateRun("impl-finding", "f1")
	if err != nil {
		t.Fatalf("CreateRun child: %v", err)
	}
	if err := store.UpdateRunStatus(childID, "running", "", 0); err != nil {
		t.Fatalf("UpdateRunStatus child: %v", err)
	}
	// Wire up the parent linkage AND age the started_at past the age
	// threshold AND clear PID/heartbeat — the exact shape of a live
	// sub-pipeline child.
	if _, err := store.db.Exec(
		`UPDATE pipeline_run
		    SET parent_run_id = ?,
		        started_at    = ?,
		        pid           = 0,
		        last_heartbeat = 0
		  WHERE run_id = ?`,
		parentID, time.Now().Add(-30*time.Minute).Unix(), childID,
	); err != nil {
		t.Fatalf("seed child run: %v", err)
	}

	if got := ReconcileZombies(store, ZombieAgeThreshold); got != 0 {
		t.Fatalf("ReconcileZombies reaped sub-pipeline child = %d, want 0", got)
	}

	status, _ := store.GetRunStatus(childID)
	if status != "running" {
		t.Errorf("child run status = %q, want running (must NOT be reaped — see #1467)", status)
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
