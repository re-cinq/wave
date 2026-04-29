package webui

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/recinq/wave/internal/pipeline"
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

	p, err := loadPipelineYAML(originalRun.PipelineName)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load pipeline: "+err.Error())
		return
	}

	fm := pipeline.NewForkManager(s.runtime.rwStore)
	allowFailed := originalRun.Status != "completed"
	newRunID, err := fm.Fork(runID, req.FromStep, p, allowFailed)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "fork failed: "+err.Error())
		return
	}

	resumeStep := ""
	for i, step := range p.Steps {
		if step.ID == req.FromStep && i+1 < len(p.Steps) {
			resumeStep = p.Steps[i+1].ID
			break
		}
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

	s.launchPipelineExecution(newRunID, originalRun.PipelineName, originalRun.Input, p, runner.Options{}, resumeStep)

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

	p, err := loadPipelineYAML(run.PipelineName)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load pipeline: "+err.Error())
		return
	}

	rewindIndex := -1
	for i, step := range p.Steps {
		if step.ID == req.ToStep {
			rewindIndex = i
			break
		}
	}
	if rewindIndex == -1 {
		writeJSONError(w, http.StatusBadRequest, "step not found in pipeline: "+req.ToStep)
		return
	}

	var stepsDeleted []string
	for i, step := range p.Steps {
		if i > rewindIndex {
			stepsDeleted = append(stepsDeleted, step.ID)
		}
	}

	if len(stepsDeleted) == 0 {
		writeJSONError(w, http.StatusBadRequest, "nothing to rewind")
		return
	}

	if err := s.runtime.rwStore.DeleteCheckpointsAfterStep(runID, rewindIndex); err != nil {
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
		StepsDeleted: stepsDeleted,
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

	fm := pipeline.NewForkManager(s.runtime.store)
	points, err := fm.ListForkPoints(runID)
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
