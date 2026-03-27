package webui

import (
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/recinq/wave/internal/retro"
)

// validRunIDPattern matches only alphanumeric characters, hyphens, and underscores.
var validRunIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// maxRetroListLimit caps the limit query parameter to prevent abuse.
const maxRetroListLimit = 500

// handleAPIRunRetro returns the retrospective for a specific run.
func (s *Server) handleAPIRunRetro(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		http.Error(w, "missing run ID", http.StatusBadRequest)
		return
	}
	if !validRunIDPattern.MatchString(runID) {
		http.Error(w, "invalid run ID", http.StatusBadRequest)
		return
	}

	retroStore := retro.NewFileStore(filepath.Join(".wave", "retros"), s.store)
	result, err := retroStore.Get(runID)
	if err != nil {
		log.Printf("[webui] failed to get retrospective for run %s: %v", runID, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
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
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
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
	if opts.Limit > maxRetroListLimit {
		opts.Limit = maxRetroListLimit
	}

	retroStore := retro.NewFileStore(filepath.Join(".wave", "retros"), s.store)
	retros, err := retroStore.List(opts)
	if err != nil {
		log.Printf("[webui] failed to list retrospectives: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
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
