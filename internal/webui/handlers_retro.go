package webui

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/recinq/wave/internal/retro"
)

// handleAPIRunRetro returns the retrospective for a specific run.
func (s *Server) handleAPIRunRetro(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		http.Error(w, "missing run ID", http.StatusBadRequest)
		return
	}

	retroStore := retro.NewFileStore(filepath.Join(".wave", "retros"), s.store)
	result, err := retroStore.Get(runID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if result == nil {
		http.Error(w, "retrospective not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleAPIRetros returns a list of retrospectives with optional filters.
func (s *Server) handleAPIRetros(w http.ResponseWriter, r *http.Request) {
	opts := retro.ListOptions{}

	if pipeline := r.URL.Query().Get("pipeline"); pipeline != "" {
		opts.PipelineName = pipeline
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			opts.Limit = limit
		}
	}
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		if d, ok := parseDuration(sinceStr); ok {
			opts.Since = time.Now().Add(-d)
		}
	}
	if opts.Limit == 0 {
		opts.Limit = 50
	}

	retroStore := retro.NewFileStore(filepath.Join(".wave", "retros"), s.store)
	retros, err := retroStore.List(opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(retros)
}

// parseDuration parses human-friendly duration strings like "7d", "30d", "24h".
// Returns the parsed duration and true on success, or zero and false on failure.
func parseDuration(s string) (time.Duration, bool) {
	if strings.HasSuffix(s, "d") {
		numStr := strings.TrimSuffix(s, "d")
		days, err := strconv.Atoi(numStr)
		if err != nil || days <= 0 {
			return 0, false
		}
		return time.Duration(days) * 24 * time.Hour, true
	}
	if strings.HasSuffix(s, "h") {
		numStr := strings.TrimSuffix(s, "h")
		hours, err := strconv.Atoi(numStr)
		if err != nil || hours <= 0 {
			return 0, false
		}
		return time.Duration(hours) * time.Hour, true
	}
	return 0, false
}
