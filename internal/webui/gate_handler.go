package webui

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/recinq/wave/internal/pipeline"
)

// Sentinel errors returned by GateRegistry.ResolveChoice so HTTP handlers can
// map registry-layer outcomes onto status codes without leaking pipeline types
// into the transport layer.
var (
	// ErrGateNotPending indicates no gate is currently pending for the run.
	ErrGateNotPending = errors.New("no pending gate for run")
	// ErrGateStepMismatch indicates the request's step does not match the
	// step that registered the pending gate.
	ErrGateStepMismatch = errors.New("step mismatch for pending gate")
	// ErrGateInvalidChoice indicates the supplied choice key is not present
	// in the pending gate's choice list.
	ErrGateInvalidChoice = errors.New("invalid choice for pending gate")
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

// ResolveChoice validates a human's choice against the pending gate for the
// given run and resolves it. It is the transport-friendly wrapper around
// Resolve: callers supply only string fields (choiceKey, text) and a stepID
// for verification. The pipeline.GateDecision construction is hidden inside
// the registry so HTTP handlers can stay free of internal/pipeline imports.
//
// Returns the resolved choice key and label on success. On failure returns
// one of the sentinel errors (ErrGateNotPending, ErrGateStepMismatch,
// ErrGateInvalidChoice) wrapped with extra context, or any error returned
// by Resolve (e.g. double-resolve).
func (r *GateRegistry) ResolveChoice(runID, stepID, choiceKey, text string) (string, string, error) {
	pendingStepID := r.GetPendingStepID(runID)
	if pendingStepID == "" {
		// No pending gate at all.
		return "", "", ErrGateNotPending
	}
	if stepID != "" && pendingStepID != stepID {
		return "", "", fmt.Errorf("%w: pending step %q, request step %q", ErrGateStepMismatch, pendingStepID, stepID)
	}

	gate := r.GetPending(runID)
	if gate == nil {
		// Gate disappeared between GetPendingStepID and GetPending (race).
		return "", "", ErrGateNotPending
	}

	choice := gate.FindChoiceByKey(choiceKey)
	if choice == nil {
		return "", "", fmt.Errorf("%w: %q", ErrGateInvalidChoice, choiceKey)
	}

	decision := &pipeline.GateDecision{
		Choice: choice.Key,
		Label:  choice.Label,
		Text:   text,
		Target: choice.Target,
	}
	if err := r.Resolve(runID, decision); err != nil {
		return "", "", err
	}
	return choice.Key, choice.Label, nil
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
