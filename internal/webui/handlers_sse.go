//go:build webui

package webui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// handleSSE handles GET /api/runs/{id}/events - SSE stream for real-time updates.
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		http.Error(w, "missing run ID", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Send retry directive (3 seconds, under the 5-second NFR-004 requirement)
	fmt.Fprintf(w, "retry: 3000\n\n")
	flusher.Flush()

	// Subscribe to events
	ch := s.broker.Subscribe()
	defer s.broker.Unsubscribe(ch)

	ctx := r.Context()
	for {
		select {
		case sseEvent, ok := <-ch:
			if !ok {
				return
			}
			// Filter events to this run's events only
			if !matchesRunID(sseEvent.Data, runID) {
				continue
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", sseEvent.Event, sseEvent.Data)
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}

// matchesRunID checks if the SSE event data belongs to the given run ID.
func matchesRunID(data string, runID string) bool {
	// Quick check before full JSON parse
	if !strings.Contains(data, runID) {
		return false
	}
	var partial struct {
		PipelineID string `json:"pipeline_id"`
	}
	if err := json.Unmarshal([]byte(data), &partial); err != nil {
		return false
	}
	return partial.PipelineID == runID
}
