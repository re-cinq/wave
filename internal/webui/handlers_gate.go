package webui

import (
	"net/http"
	"sync"
)

// GateRegistry manages gate resolution channels for HTTP-based gate triggering.
type GateRegistry struct {
	mu       sync.Mutex
	channels map[string]chan struct{} // key: "runID:stepID"
}

// NewGateRegistry creates a new gate registry.
func NewGateRegistry() *GateRegistry {
	return &GateRegistry{
		channels: make(map[string]chan struct{}),
	}
}

// Register creates a gate channel for the given run and step.
// Returns the channel that will be closed when the gate is resolved.
func (g *GateRegistry) Register(runID, stepID string) <-chan struct{} {
	key := runID + ":" + stepID
	ch := make(chan struct{})
	g.mu.Lock()
	g.channels[key] = ch
	g.mu.Unlock()
	return ch
}

// Resolve closes the gate channel for the given run and step.
// Returns true if the gate was found and resolved, false if not found.
func (g *GateRegistry) Resolve(runID, stepID string) bool {
	key := runID + ":" + stepID
	g.mu.Lock()
	ch, ok := g.channels[key]
	if ok {
		delete(g.channels, key)
	}
	g.mu.Unlock()

	if !ok {
		return false
	}

	// Use sync.Once pattern — safe even if called twice since we delete first
	close(ch)
	return true
}

// Cleanup removes the gate channel without resolving it.
func (g *GateRegistry) Cleanup(runID, stepID string) {
	key := runID + ":" + stepID
	g.mu.Lock()
	delete(g.channels, key)
	g.mu.Unlock()
}

// handleResolveGate handles POST /api/runs/{id}/gate/{step}
func (s *Server) handleResolveGate(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	stepID := r.PathValue("step")

	if runID == "" || stepID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing run ID or step ID")
		return
	}

	if s.gates == nil {
		writeJSONError(w, http.StatusNotFound, "no gate waiting for resolution")
		return
	}

	if !s.gates.Resolve(runID, stepID) {
		writeJSONError(w, http.StatusNotFound, "no gate waiting for resolution")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"run_id":  runID,
		"step_id": stepID,
		"status":  "resolved",
	})
}
