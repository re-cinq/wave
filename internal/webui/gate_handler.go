package webui

import (
	"fmt"
	"sync"

	"github.com/recinq/wave/internal/runner"
)

// pendingGate holds the state for a gate that is waiting for a human
// decision via the WebUI HTTP endpoint.
type pendingGate struct {
	StepID   string // step this gate belongs to (for URL verification)
	Gate     *runner.WebUIGate
	Response chan *runner.WebUIGateDecision
}

// GateRegistry tracks pending gate decisions across all active pipeline runs.
// It is safe for concurrent use. Each run can have at most one pending gate
// at a time (gates are sequential within a pipeline execution).
//
// The registry trades exclusively in the runner package's transport types
// (runner.WebUIGate / runner.WebUIGateDecision) so this file no longer
// depends on internal/pipeline. The runner-side WebUIGateHandler bridges
// pipeline.GateConfig and pipeline.GateDecision into and out of the
// registrar.
type GateRegistry struct {
	mu      sync.Mutex
	pending map[string]*pendingGate // runID -> pending gate
}

// NewGateRegistry creates an empty gate registry.
func NewGateRegistry() *GateRegistry {
	return &GateRegistry{
		pending: make(map[string]*pendingGate),
	}
}

// Register stores a pending gate for the given run and returns a channel
// that will receive the decision when it arrives from the HTTP endpoint.
func (r *GateRegistry) Register(runID, stepID string, gate *runner.WebUIGate) chan *runner.WebUIGateDecision {
	r.mu.Lock()
	defer r.mu.Unlock()

	ch := make(chan *runner.WebUIGateDecision, 1)
	r.pending[runID] = &pendingGate{
		StepID:   stepID,
		Gate:     gate,
		Response: ch,
	}
	return ch
}

// Resolve sends a decision to the pending gate for the given run.
// Returns an error if no gate is pending for that run. The send is
// performed under the lock to prevent a concurrent Remove() from
// racing between the map deletion and the channel send.
func (r *GateRegistry) Resolve(runID string, decision *runner.WebUIGateDecision) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	pg, ok := r.pending[runID]
	if !ok {
		return fmt.Errorf("no pending gate for run %q", runID)
	}
	delete(r.pending, runID)
	pg.Response <- decision
	return nil
}

// GetPending returns the pending gate config for a run, or nil if none.
func (r *GateRegistry) GetPending(runID string) *runner.WebUIGate {
	r.mu.Lock()
	defer r.mu.Unlock()

	if pg, ok := r.pending[runID]; ok {
		return pg.Gate
	}
	return nil
}

// GetPendingStepID returns the step ID of the pending gate for a run, or empty string.
func (r *GateRegistry) GetPendingStepID(runID string) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	if pg, ok := r.pending[runID]; ok {
		return pg.StepID
	}
	return ""
}

// Remove removes a pending gate without resolving it (e.g. on context cancellation).
func (r *GateRegistry) Remove(runID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.pending, runID)
}
