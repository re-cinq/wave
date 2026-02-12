package dashboard

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/recinq/wave/internal/state"
)

func (s *Server) handleListRuns(w http.ResponseWriter, r *http.Request) {
	opts := state.ListRunsOptions{}

	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			opts.Limit = n
		}
	}
	if opts.Limit == 0 {
		opts.Limit = 50
	}

	if v := r.URL.Query().Get("status"); v != "" {
		opts.Status = v
	}
	if v := r.URL.Query().Get("pipeline"); v != "" {
		opts.PipelineName = v
	}

	runs, err := s.store.ListRuns(opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list runs", err.Error())
		return
	}

	resp := RunListResponse{
		Runs:  make([]RunResponse, 0, len(runs)),
		Total: len(runs),
	}

	for _, run := range runs {
		resp.Runs = append(resp.Runs, runToResponse(run))
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGetRun(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		writeError(w, http.StatusBadRequest, "missing run ID", "")
		return
	}

	run, err := s.store.GetRun(runID)
	if err != nil {
		writeError(w, http.StatusNotFound, "run not found", err.Error())
		return
	}

	resp := RunDetailResponse{
		Run: runToResponse(*run),
	}

	// Get step progress
	steps, err := s.store.GetAllStepProgress(runID)
	if err == nil && len(steps) > 0 {
		resp.Steps = make([]StepProgressResponse, 0, len(steps))
		for _, step := range steps {
			resp.Steps = append(resp.Steps, stepToResponse(step))
		}
	}

	// Get pipeline progress
	progress, err := s.store.GetPipelineProgress(runID)
	if err == nil && progress != nil {
		resp.PipelineProgress = &PipelineProgressResponse{
			RunID:                 progress.RunID,
			TotalSteps:            progress.TotalSteps,
			CompletedSteps:        progress.CompletedSteps,
			CurrentStepIndex:      progress.CurrentStepIndex,
			OverallProgress:       progress.OverallProgress,
			EstimatedCompletionMs: progress.EstimatedCompletionMs,
			UpdatedAt:             progress.UpdatedAt,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGetRunEvents(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		writeError(w, http.StatusBadRequest, "missing run ID", "")
		return
	}

	opts := state.EventQueryOptions{}

	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			opts.Limit = n
		}
	}
	if v := r.URL.Query().Get("step"); v != "" {
		opts.StepID = v
	}
	if r.URL.Query().Get("errors_only") == "true" {
		opts.ErrorsOnly = true
	}

	events, err := s.store.GetEvents(runID, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get events", err.Error())
		return
	}

	resp := EventListResponse{
		Events: make([]EventResponse, 0, len(events)),
	}

	for _, ev := range events {
		resp.Events = append(resp.Events, EventResponse{
			ID:         ev.ID,
			RunID:      ev.RunID,
			Timestamp:  ev.Timestamp,
			StepID:     ev.StepID,
			State:      ev.State,
			Persona:    ev.Persona,
			Message:    ev.Message,
			TokensUsed: ev.TokensUsed,
			DurationMs: ev.DurationMs,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGetRunSteps(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		writeError(w, http.StatusBadRequest, "missing run ID", "")
		return
	}

	steps, err := s.store.GetAllStepProgress(runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get steps", err.Error())
		return
	}

	resp := make([]StepProgressResponse, 0, len(steps))
	for _, step := range steps {
		resp = append(resp, stepToResponse(step))
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGetRunArtifacts(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		writeError(w, http.StatusBadRequest, "missing run ID", "")
		return
	}

	stepID := r.URL.Query().Get("step")
	artifacts, err := s.store.GetArtifacts(runID, stepID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get artifacts", err.Error())
		return
	}

	resp := ArtifactListResponse{
		Artifacts: make([]ArtifactResponse, 0, len(artifacts)),
	}

	for _, a := range artifacts {
		resp.Artifacts = append(resp.Artifacts, ArtifactResponse{
			ID:        a.ID,
			RunID:     a.RunID,
			StepID:    a.StepID,
			Name:      a.Name,
			Path:      a.Path,
			Type:      a.Type,
			SizeBytes: a.SizeBytes,
			CreatedAt: a.CreatedAt,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGetRunProgress(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		writeError(w, http.StatusBadRequest, "missing run ID", "")
		return
	}

	progress, err := s.store.GetPipelineProgress(runID)
	if err != nil {
		writeError(w, http.StatusNotFound, "progress not found", err.Error())
		return
	}

	resp := PipelineProgressResponse{
		RunID:                 progress.RunID,
		TotalSteps:            progress.TotalSteps,
		CompletedSteps:        progress.CompletedSteps,
		CurrentStepIndex:      progress.CurrentStepIndex,
		OverallProgress:       progress.OverallProgress,
		EstimatedCompletionMs: progress.EstimatedCompletionMs,
		UpdatedAt:             progress.UpdatedAt,
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	s.broker.ServeHTTP(w, r)
}

// Conversion helpers

func runToResponse(run state.RunRecord) RunResponse {
	resp := RunResponse{
		RunID:        run.RunID,
		PipelineName: run.PipelineName,
		Status:       run.Status,
		Input:        run.Input,
		CurrentStep:  run.CurrentStep,
		TotalTokens:  run.TotalTokens,
		StartedAt:    run.StartedAt,
		CompletedAt:  run.CompletedAt,
		CancelledAt:  run.CancelledAt,
		ErrorMessage: run.ErrorMessage,
		Tags:         run.Tags,
	}

	if run.CompletedAt != nil {
		resp.DurationMs = run.CompletedAt.Sub(run.StartedAt).Milliseconds()
	} else {
		resp.DurationMs = time.Since(run.StartedAt).Milliseconds()
	}

	return resp
}

func stepToResponse(step state.StepProgressRecord) StepProgressResponse {
	return StepProgressResponse{
		StepID:                step.StepID,
		RunID:                 step.RunID,
		Persona:               step.Persona,
		State:                 step.State,
		Progress:              step.Progress,
		CurrentAction:         step.CurrentAction,
		Message:               step.Message,
		StartedAt:             step.StartedAt,
		UpdatedAt:             step.UpdatedAt,
		EstimatedCompletionMs: step.EstimatedCompletionMs,
		TokensUsed:            step.TokensUsed,
	}
}

// Response helpers

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string, details string) {
	writeJSON(w, status, ErrorResponse{
		Error:   message,
		Code:    status,
		Details: details,
	})
}
