package state

import (
	"syscall"
	"time"
)

// ZombieAgeThreshold is the default age beyond which a "running" run with no
// tracked PID is treated as orphaned. Five minutes is short enough that real
// dead processes do not linger, long enough that a freshly launched run is not
// mistaken for a zombie before its first event is recorded.
const ZombieAgeThreshold = 5 * time.Minute

// ReconcileZombies marks "running" pipeline runs as failed when their tracked
// process is gone, or when no PID is tracked and the run is older than
// ageThreshold. Returns the number of runs reclaimed.
//
// Reconciliation is the single defense against parent processes that die
// without updating the DB (terminal close, OOM, signal, kernel panic). Without
// it, the webui run list and CLI status accumulate "running" rows forever.
//
// Pass the zero value to use ZombieAgeThreshold.
func ReconcileZombies(store StateStore, ageThreshold time.Duration) int {
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

// isZombie reports whether a "running" record has lost its underlying process.
// Returns true if the tracked PID no longer exists, or if the run has no PID
// and is older than ageThreshold (proxy for "this should have finished by now").
func isZombie(r RunRecord, ageThreshold time.Duration) bool {
	if r.PID > 0 {
		if err := syscall.Kill(r.PID, 0); err == syscall.ESRCH {
			return true
		}
		return false
	}
	return time.Since(r.StartedAt) > ageThreshold
}
