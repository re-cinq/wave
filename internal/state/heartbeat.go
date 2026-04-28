package state

import (
	"context"
	"time"
)

// HeartbeatInterval is the cadence at which RunHeartbeatLoop refreshes
// pipeline_run.last_heartbeat. Chosen well below HeartbeatStaleThreshold
// (90s) so two missed beats are tolerated before the reconciler reaps the
// run.
const HeartbeatInterval = 30 * time.Second

// HeartbeatUpdater is the minimal persistence surface RunHeartbeatLoop
// needs. *stateStore (and any StateStore implementation) satisfies it.
type HeartbeatUpdater interface {
	UpdateRunHeartbeat(runID string) error
}

// RunHeartbeatLoop periodically refreshes pipeline_run.last_heartbeat for
// the running pipeline. The reconciler reads this column to distinguish
// live runs from runs whose owning process died without updating the DB.
//
// Returns when ctx is cancelled. Errors from UpdateRunHeartbeat are
// swallowed — a transient DB hiccup is benign; the reconciler tolerates
// missed beats.
func RunHeartbeatLoop(ctx context.Context, store HeartbeatUpdater, runID string) {
	if store == nil {
		return
	}
	ticker := time.NewTicker(HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			_ = store.UpdateRunHeartbeat(runID)
		case <-ctx.Done():
			return
		}
	}
}
