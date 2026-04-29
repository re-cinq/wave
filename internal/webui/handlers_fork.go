package webui

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/recinq/wave/internal/runner"
)

// handleForkRun handles POST /api/runs/{id}/fork
func (s *Server) handleForkRun(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing run ID")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req ForkRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.FromStep == "" {
		writeJSONError(w, http.StatusBadRequest, "from_step is required")
		return
	}

	originalRun, err := s.runtime.store.GetRun(runID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "run not found")
		return
	}

	if originalRun.Status == "running" {
		writeJSONError(w, http.StatusConflict, "cannot fork a running run")
		return
	}

	allowFailed := originalRun.Status != "completed"
	fc := runner.NewForkController(s.runtime.rwStore)
	newRunID, err := fc.Fork(runID, req.FromStep, originalRun.PipelineName, allowFailed)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "fork failed: "+err.Error())
		return
	}

	resumeStep, err := fc.ResumeStepAfter(originalRun.PipelineName, req.FromStep)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load pipeline: "+err.Error())
		return
	}

	if resumeStep == "" {
		if err := s.runtime.rwStore.UpdateRunStatus(newRunID, "completed", "", 0); err != nil {
			log.Printf("Warning: failed to update forked run %s status: %v", newRunID, err)
		}
		writeJSON(w, http.StatusCreated, ForkRunResponse{
			RunID:        newRunID,
			SourceRunID:  runID,
			FromStep:     req.FromStep,
			PipelineName: originalRun.PipelineName,
			Status:       "completed",
			StartedAt:    time.Now(),
		})
		return
	}

	s.launchPipelineExecution(newRunID, originalRun.PipelineName, originalRun.Input, runner.Options{}, resumeStep)

	writeJSON(w, http.StatusCreated, ForkRunResponse{
		RunID:        newRunID,
		SourceRunID:  runID,
		FromStep:     req.FromStep,
		PipelineName: originalRun.PipelineName,
		Status:       "running",
		StartedAt:    time.Now(),
	})
}

// handleRewindRun handles POST /api/runs/{id}/rewind
func (s *Server) handleRewindRun(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing run ID")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req RewindRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ToStep == "" {
		writeJSONError(w, http.StatusBadRequest, "to_step is required")
		return
	}

	run, err := s.runtime.store.GetRun(runID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "run not found")
		return
	}

	if run.Status == "running" {
		writeJSONError(w, http.StatusConflict, "cannot rewind a running run")
		return
	}

	plan, err := runner.NewForkController(s.runtime.rwStore).PlanRewind(run.PipelineName, req.ToStep)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load pipeline: "+err.Error())
		return
	}
	if plan.StepIndex == -1 {
		writeJSONError(w, http.StatusBadRequest, "step not found in pipeline: "+req.ToStep)
		return
	}
	if len(plan.StepsDeleted) == 0 {
		writeJSONError(w, http.StatusBadRequest, "nothing to rewind")
		return
	}

	if err := s.runtime.rwStore.DeleteCheckpointsAfterStep(runID, plan.StepIndex); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "rewind failed: "+err.Error())
		return
	}

	rewindMsg := "rewound to step: " + req.ToStep
	if err := s.runtime.rwStore.UpdateRunStatus(runID, "failed", rewindMsg, run.TotalTokens); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to update run status: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, RewindRunResponse{
		RunID:        runID,
		ToStep:       req.ToStep,
		StepsDeleted: plan.StepsDeleted,
		Status:       "failed",
	})
}

// handleForkPoints handles GET /api/runs/{id}/fork-points
func (s *Server) handleForkPoints(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing run ID")
		return
	}

	if _, err := s.runtime.store.GetRun(runID); err != nil {
		writeJSONError(w, http.StatusNotFound, "run not found")
		return
	}

	points, err := runner.NewForkController(s.runtime.store).ListForkPoints(runID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list fork points: "+err.Error())
		return
	}

	resp := ForkPointsResponse{
		RunID:      runID,
		ForkPoints: make([]ForkPointResponse, len(points)),
	}
	for i, pt := range points {
		resp.ForkPoints[i] = ForkPointResponse{
			StepID:    pt.StepID,
			StepIndex: pt.StepIndex,
			HasSHA:    pt.HasSHA,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}
