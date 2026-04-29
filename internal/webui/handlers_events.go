package webui

import (
	"log"
	"net/http"
	"strconv"

	"github.com/recinq/wave/internal/state"
)

// handleAPIStepEvents handles GET /api/runs/{id}/step-events - returns paginated
// events for a specific step within a run. Supports query parameters:
//   - step: filter events to a specific step ID
//   - offset: number of events to skip (default 0)
//   - limit: maximum events to return (default 200, max 5000)
func (s *Server) handleAPIStepEvents(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing run ID")
		return
	}

	// Verify run exists
	_, err := s.runtime.store.GetRun(runID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "run not found")
		return
	}

	stepID := r.URL.Query().Get("step")
	offset := parseIntParam(r, "offset", 0)
	limit := parseIntParam(r, "limit", 200)

	if limit <= 0 {
		limit = 200
	}
	if limit > 5000 {
		limit = 5000
	}
	if offset < 0 {
		offset = 0
	}

	opts := state.EventQueryOptions{
		StepID: stepID,
		Offset: offset,
		Limit:  limit + 1, // fetch one extra to determine hasMore
	}

	events, err := s.runtime.store.GetEvents(runID, opts)
	if err != nil {
		log.Printf("[webui] failed to get step events for run %s: %v", runID, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to get events")
		return
	}

	hasMore := len(events) > limit
	if hasMore {
		events = events[:limit]
	}

	summaries := make([]EventSummary, len(events))
	for i, e := range events {
		summaries[i] = eventToSummary(e)
	}

	resp := StepEventsResponse{
		Events:  summaries,
		HasMore: hasMore,
		Offset:  offset,
		Limit:   limit,
	}

	writeJSON(w, http.StatusOK, resp)
}

// parseIntParam parses an integer query parameter with a default value.
func parseIntParam(r *http.Request, name string, defaultVal int) int {
	s := r.URL.Query().Get(name)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}
