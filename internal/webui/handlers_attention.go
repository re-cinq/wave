package webui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// handleAttentionSSE handles GET /api/attention/events — a global SSE stream
// that broadcasts attention summary changes across all active runs.
func (s *Server) handleAttentionSSE(w http.ResponseWriter, r *http.Request) {
	if s.attention == nil {
		http.Error(w, "attention broker not initialized", http.StatusServiceUnavailable)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	fmt.Fprintf(w, "retry: 5000\n\n")
	flusher.Flush()

	// Send current state immediately.
	summary := s.attention.Summary()
	data, _ := json.Marshal(summary)
	fmt.Fprintf(w, "event: attention\ndata: %s\n\n", data)
	flusher.Flush()

	ch := s.attention.Subscribe()
	defer s.attention.Unsubscribe(ch)

	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	ctx := r.Context()
	for {
		select {
		case summary, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(summary)
			fmt.Fprintf(w, "event: attention\ndata: %s\n\n", data)
			flusher.Flush()
		case <-keepalive.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}

// handleAttentionSummary handles GET /api/attention — returns current attention summary as JSON.
func (s *Server) handleAttentionSummary(w http.ResponseWriter, r *http.Request) {
	if s.attention == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"worst_state":"autonomous","total_runs":0}`)
		return
	}

	summary := s.attention.Summary()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(summary); err != nil {
		http.Error(w, "failed to encode attention summary", http.StatusInternalServerError)
	}
}
