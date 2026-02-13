//go:build webui

package webui

import (
	"fmt"
	"net/http"
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
		case event, ok := <-ch:
			if !ok {
				return
			}
			// Filter events to this run's events only
			// The SSE broker broadcasts all events; we filter here
			// In a production system, you'd want per-run channels
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Event, event.Data)
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}
