package state

import (
	"syscall"
	"time"
)

// ZombieAgeThreshold is the default age beyond which a "running" run with no
// tracked PID and no heartbeat is treated as orphaned. Five minutes is short
// enough that real dead processes do not linger, long enough that a freshly
// launched run is not mistaken for a zombie before its first event is recorded.
const ZombieAgeThreshold = 5 * time.Minute

// HeartbeatStaleThreshold is the maximum gap between heartbeats before a
// running run is considered orphaned. wave run writes a heartbeat every 30s,
// so a 90s threshold tolerates two missed beats before reaping. This is the
// primary liveness signal: when present, it overrides PID and age checks.
const HeartbeatStaleThreshold = 90 * time.Second

// ReconcileZombies marks "running" pipeline runs as failed when their owning
// process is gone. Returns the number of runs reclaimed. Liveness signals are
// checked in priority order:
//
//  1. Heartbeat: if the run wrote a heartbeat within HeartbeatStaleThreshold,
//     it is alive — regardless of PID or age. If the last heartbeat is older,
//     it is a zombie.
//  2. PID: if the run has a tracked PID and the OS reports ESRCH, it is a
//     zombie. A live PID without a recent heartbeat keeps the run.
//  3. Age fallback: legacy runs with no PID and no heartbeat are reaped once
//     their started_at is older than ageThreshold.
//
// ZombieReconciler is the minimal store surface ReconcileZombies needs:
// list "running" runs and mark stale ones as failed. Defined here so any
// caller-side narrow interface can pass through without inheriting the
// full RunStore.
type ZombieReconciler interface {
	ListRuns(opts ListRunsOptions) ([]RunRecord, error)
	UpdateRunStatus(runID string, status string, currentStep string, tokens int) error
}

// Pass the zero value for ageThreshold to use ZombieAgeThreshold.
func ReconcileZombies(store ZombieReconciler, ageThreshold time.Duration) int {
	if ageThreshold <= 0 {
		ageThreshold = ZombieAgeThreshold
	}
	runs, err := store.ListRuns(ListRunsOptions{Status: "running", Limit: 1000})
	if err != nil {
		return 0
	}
	reclaimed := 0
	for _, r := range runs {
		if !isZombie(r, ageThreshold) {
			continue
		}
		if err := store.UpdateRunStatus(r.RunID, "failed", "process gone (orphaned)", r.TotalTokens); err == nil {
			reclaimed++
		}
	}
	return reclaimed
}

// isZombie reports whether a "running" record has lost its owning process.
// Heartbeat freshness is the primary signal; PID and age are fallbacks for
// runs that have not yet started writing heartbeats (legacy data, or a run
// reaped before the first heartbeat goroutine tick).
func isZombie(r RunRecord, ageThreshold time.Duration) bool {
	// Sub-pipeline child runs (composition: iterate, sub_pipeline, branch,
	// loop) execute inside the parent process's goroutines and never get
	// their own PID, never write their own heartbeats. Their liveness IS
	// the parent's liveness. The reaper must not apply the age fallback to
	// them — a 6-minute apply-fix run is not a zombie just because the
	// child run row has no PID. The parent run row carries the heartbeat;
	// reap on the parent if anything.
	if r.ParentRunID != "" {
		return false
	}
	if !r.LastHeartbeat.IsZero() {
		// Heartbeat exists: trust it absolutely. A late heartbeat is a zombie.
		return time.Since(r.LastHeartbeat) > HeartbeatStaleThreshold
	}
	if r.PID > 0 {
		if err := syscall.Kill(r.PID, 0); err == syscall.ESRCH {
			return true
		}
		return false
	}
	return time.Since(r.StartedAt) > ageThreshold
}
