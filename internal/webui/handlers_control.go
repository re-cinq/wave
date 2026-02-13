//go:build webui

package webui

import (
	"encoding/json"
	"net/http"
)

// handleStartPipeline handles POST /api/pipelines/{name}/start
func (s *Server) handleStartPipeline(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "missing pipeline name")
		return
	}

	var req StartPipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Create a new run via the read-write store
	runID, err := s.rwStore.CreateRun(name, req.Input)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create run: "+err.Error())
		return
	}

	run, err := s.rwStore.GetRun(runID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to get created run")
		return
	}

	resp := StartPipelineResponse{
		RunID:        run.RunID,
		PipelineName: run.PipelineName,
		Status:       run.Status,
		StartedAt:    run.StartedAt,
	}

	writeJSON(w, http.StatusCreated, resp)
}

// handleCancelRun handles POST /api/runs/{id}/cancel
func (s *Server) handleCancelRun(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing run ID")
		return
	}

	var req CancelRunRequest
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req)
	}

	// Check run exists and is cancellable
	run, err := s.store.GetRun(runID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "run not found")
		return
	}

	if run.Status != "running" && run.Status != "pending" {
		writeJSONError(w, http.StatusConflict, "run is not in a cancellable state (status: "+run.Status+")")
		return
	}

	if err := s.rwStore.RequestCancellation(runID, req.Force); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to request cancellation")
		return
	}

	resp := CancelRunResponse{
		RunID:  runID,
		Status: "cancelling",
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleRetryRun handles POST /api/runs/{id}/retry
func (s *Server) handleRetryRun(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing run ID")
		return
	}

	// Get original run to copy parameters
	originalRun, err := s.store.GetRun(runID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "run not found")
		return
	}

	if originalRun.Status != "failed" && originalRun.Status != "cancelled" {
		writeJSONError(w, http.StatusConflict, "run is not in a retryable state (status: "+originalRun.Status+")")
		return
	}

	// Create a new run with the same parameters
	newRunID, err := s.rwStore.CreateRun(originalRun.PipelineName, originalRun.Input)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create retry run")
		return
	}

	newRun, err := s.rwStore.GetRun(newRunID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to get retry run")
		return
	}

	resp := RetryRunResponse{
		RunID:         newRun.RunID,
		OriginalRunID: runID,
		PipelineName:  newRun.PipelineName,
		Status:        newRun.Status,
		StartedAt:     newRun.StartedAt,
	}

	writeJSON(w, http.StatusCreated, resp)
}
