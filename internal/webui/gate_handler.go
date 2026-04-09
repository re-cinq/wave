package webui

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/recinq/wave/internal/pipeline"
)

// pendingGate holds the state for a gate that is waiting for a human decision
// via the WebUI HTTP endpoint.
type pendingGate struct {
	StepID   string // step this gate belongs to (for URL verification)
	Gate     *pipeline.GateConfig
	Response chan *pipeline.GateDecision
}

// GateRegistry tracks pending gate decisions across all active pipeline runs.
// It is safe for concurrent use. Each run can have at most one pending gate
// at a time (gates are sequential within a pipeline execution).
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
func (r *GateRegistry) Register(runID, stepID string, gate *pipeline.GateConfig) chan *pipeline.GateDecision {
	r.mu.Lock()
	defer r.mu.Unlock()

	ch := make(chan *pipeline.GateDecision, 1)
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
func (r *GateRegistry) Resolve(runID string, decision *pipeline.GateDecision) error {
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
func (r *GateRegistry) GetPending(runID string) *pipeline.GateConfig {
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

// WebUIGateHandler implements pipeline.GateHandler by registering the gate in
// a shared GateRegistry and blocking until the HTTP endpoint resolves it.
// One instance is created per pipeline run via NewWebUIGateHandler.
type WebUIGateHandler struct {
	runID    string
	registry *GateRegistry
}

// NewWebUIGateHandler creates a gate handler for a specific pipeline run.
func NewWebUIGateHandler(runID string, registry *GateRegistry) *WebUIGateHandler {
	return &WebUIGateHandler{
		runID:    runID,
		registry: registry,
	}
}

// Prompt registers the gate in the registry and blocks until a decision
// arrives from the HTTP endpoint or the context is cancelled.
func (h *WebUIGateHandler) Prompt(ctx context.Context, gate *pipeline.GateConfig) (*pipeline.GateDecision, error) {
	ch := h.registry.Register(h.runID, gate.RuntimeStepID, gate)

	select {
	case <-ctx.Done():
		h.registry.Remove(h.runID)
		return nil, ctx.Err()
	case decision := <-ch:
		if decision == nil {
			return nil, fmt.Errorf("gate decision channel closed without a decision")
		}
		decision.Timestamp = time.Now()
		return decision, nil
	}
}
