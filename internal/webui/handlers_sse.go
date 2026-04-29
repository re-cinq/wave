package webui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/recinq/wave/internal/state"
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

	// Check for Last-Event-ID header for reconnection backfill
	var lastEventID int64
	if idStr := r.Header.Get("Last-Event-ID"); idStr != "" {
		if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
			lastEventID = id
		}
	}

	// Backfill missed events from DB on reconnection
	if lastEventID > 0 {
		events, err := s.runtime.store.GetEvents(runID, state.EventQueryOptions{
			AfterID: lastEventID,
		})
		if err == nil {
			for _, ev := range events {
				data, _ := json.Marshal(ev)
				fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", ev.ID, ev.State, string(data))
			}
			flusher.Flush()
		}
	}

	// Subscribe to live events
	ch := s.realtime.broker.Subscribe()
	defer s.realtime.broker.Unsubscribe(ch)

	// Keepalive ticker prevents idle connection timeouts.
	// SSE comments (lines starting with ':') are ignored by EventSource
	// but keep the TCP connection alive through proxies and browsers.
	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

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
			if sseEvent.ID != "" {
				fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", sseEvent.ID, sseEvent.Event, sseEvent.Data)
			} else {
				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", sseEvent.Event, sseEvent.Data)
			}
			flusher.Flush()
		case <-keepalive.C:
			// SSE comment keeps connection alive
			fmt.Fprintf(w, ": keepalive\n\n")
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
