//go:build retros

package webui

import (
	"net/http"
	"strconv"
	"time"

	"github.com/recinq/wave/internal/metrics"
	"github.com/recinq/wave/internal/retro"
	"github.com/recinq/wave/internal/state"
)

// handleAPIRetros returns a list of retrospectives.
func (s *Server) handleAPIRetros(w http.ResponseWriter, r *http.Request) {
	pipeline := r.URL.Query().Get("pipeline")
	sinceStr := r.URL.Query().Get("since")
	limitStr := r.URL.Query().Get("limit")

	var sinceUnix int64
	if sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			sinceUnix = t.Unix()
		} else if secs, err := strconv.ParseInt(sinceStr, 10, 64); err == nil {
			sinceUnix = secs
		}
	}

	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	mstore := metrics.NewStore(state.UnderlyingDB(s.runtime.store))
	records, err := mstore.ListRetrospectives(metrics.ListRetrosOptions{
		PipelineName: pipeline,
		SinceUnix:    sinceUnix,
		Limit:        limit,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, records)
}

// handleAPIRetroDetail returns a single retrospective.
func (s *Server) handleAPIRetroDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing run ID"})
		return
	}

	storage := retro.NewStorage(".agents/retros", metrics.NewStore(state.UnderlyingDB(s.runtime.store)))
	retro, err := storage.Load(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, retro)
}

// handleNarrateRetro triggers narrative generation for a retrospective.
func (s *Server) handleNarrateRetro(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing run ID"})
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{
		"status":  "accepted",
		"message": "Narrative generation is not available from the web UI — use 'wave retro " + id + " --narrate' from the CLI",
	})
}
