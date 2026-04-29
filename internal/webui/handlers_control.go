package webui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/runner"
	"github.com/recinq/wave/internal/state"
)

// handleStartPipeline handles POST /api/pipelines/{name}/start
func (s *Server) handleStartPipeline(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "missing pipeline name")
		return
	}

	if s.isPipelineDisabled(name) {
		writeJSONError(w, http.StatusForbidden, "pipeline is disabled")
		return
	}

	var req StartPipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Continuous && req.FromStep != "" {
		writeJSONError(w, http.StatusBadRequest, "--continuous and --from-step are mutually exclusive")
		return
	}
	if req.OnFailure != "" && req.OnFailure != "halt" && req.OnFailure != "skip" {
		writeJSONError(w, http.StatusBadRequest, "on_failure must be 'halt' or 'skip'")
		return
	}

	p, err := loadPipelineYAML(name)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to load pipeline: "+err.Error())
		return
	}

	runID, err := s.runtime.rwStore.CreateRun(name, req.Input)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create run: "+err.Error())
		return
	}

	opts := runOptionsFromStartRequest(req)

	if req.FromStep != "" {
		s.launchPipelineExecution(runID, name, req.Input, p, opts, req.FromStep)
	} else {
		s.launchPipelineExecution(runID, name, req.Input, p, opts)
	}

	writeJSON(w, http.StatusCreated, StartPipelineResponse{
		RunID:        runID,
		PipelineName: name,
		Status:       "running",
		StartedAt:    time.Now(),
	})
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
		_ = json.NewDecoder(r.Body).Decode(&req) // best-effort; defaults are fine
	}

	run, err := s.runtime.store.GetRun(runID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "run not found")
		return
	}

	if !req.Force && run.Status != "running" && run.Status != "pending" {
		writeJSONError(w, http.StatusConflict, "run is not in a cancellable state (status: "+run.Status+")")
		return
	}

	s.mu.Lock()
	if cancelFn, ok := s.realtime.activeRuns[runID]; ok {
		cancelFn()
	}
	s.mu.Unlock()

	if err := s.runtime.rwStore.RequestCancellation(runID, req.Force); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to request cancellation")
		return
	}

	status := "cancelling"
	if req.Force {
		status = "cancelled"
	}
	writeJSON(w, http.StatusOK, CancelRunResponse{RunID: runID, Status: status})
}

// handleRetryRun handles POST /api/runs/{id}/retry
func (s *Server) handleRetryRun(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing run ID")
		return
	}

	originalRun, err := s.runtime.store.GetRun(runID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "run not found")
		return
	}

	if originalRun.Status != "failed" && originalRun.Status != "cancelled" {
		writeJSONError(w, http.StatusConflict, "run is not in a retryable state (status: "+originalRun.Status+")")
		return
	}

	p, err := loadPipelineYAML(originalRun.PipelineName)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load pipeline: "+err.Error())
		return
	}

	newRunID, err := s.runtime.rwStore.CreateRun(originalRun.PipelineName, originalRun.Input)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create retry run")
		return
	}

	s.launchPipelineExecution(newRunID, originalRun.PipelineName, originalRun.Input, p, runner.Options{})

	writeJSON(w, http.StatusCreated, RetryRunResponse{
		RunID:         newRunID,
		OriginalRunID: runID,
		PipelineName:  originalRun.PipelineName,
		Status:        "running",
		StartedAt:     time.Now(),
	})
}

// handleResumeRun handles POST /api/runs/{id}/resume
func (s *Server) handleResumeRun(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing run ID")
		return
	}

	var req ResumeRunRequest
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

	if originalRun.Status != "failed" && originalRun.Status != "cancelled" {
		writeJSONError(w, http.StatusConflict, "run is not in a resumable state (status: "+originalRun.Status+")")
		return
	}

	p, err := loadPipelineYAML(originalRun.PipelineName)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load pipeline: "+err.Error())
		return
	}

	stepFound := false
	for _, step := range p.Steps {
		if step.ID == req.FromStep {
			stepFound = true
			break
		}
	}
	if !stepFound {
		writeJSONError(w, http.StatusBadRequest, "step not found in pipeline: "+req.FromStep)
		return
	}

	newRunID, err := s.runtime.rwStore.CreateRun(originalRun.PipelineName, originalRun.Input)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create resume run")
		return
	}

	// Link the resume run back to the failed parent so the WebUI can render
	// the breadcrumb on the resumed run and a "Resumed by" pill on the
	// parent failed run (issue #1510). Best-effort.
	if err := s.runtime.rwStore.SetParentRun(newRunID, runID, req.FromStep); err != nil {
		fmt.Printf("warning: failed to link resume run %s to parent %s: %v\n", newRunID, runID, err)
	}
	if err := s.runtime.rwStore.SetRunComposition(newRunID, state.RunKindResume, "", "", nil, nil); err != nil {
		fmt.Printf("warning: failed to set resume run kind on %s: %v\n", newRunID, err)
	}

	s.launchPipelineExecution(newRunID, originalRun.PipelineName, originalRun.Input, p, runner.Options{}, req.FromStep)

	writeJSON(w, http.StatusCreated, ResumeRunResponse{
		RunID:         newRunID,
		OriginalRunID: runID,
		PipelineName:  originalRun.PipelineName,
		FromStep:      req.FromStep,
		Status:        "running",
		StartedAt:     time.Now(),
	})
}

// handleSubmitRun handles POST /api/runs — submit a new pipeline run.
func (s *Server) handleSubmitRun(w http.ResponseWriter, r *http.Request) {
	var req SubmitRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Pipeline == "" {
		writeJSONError(w, http.StatusBadRequest, "pipeline name is required")
		return
	}

	if req.Continuous && req.FromStep != "" {
		writeJSONError(w, http.StatusBadRequest, "--continuous and --from-step are mutually exclusive")
		return
	}
	if req.OnFailure != "" && req.OnFailure != "halt" && req.OnFailure != "skip" {
		writeJSONError(w, http.StatusBadRequest, "on_failure must be 'halt' or 'skip'")
		return
	}

	p, err := loadPipelineYAML(req.Pipeline)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to load pipeline: "+err.Error())
		return
	}

	runID, err := s.runtime.rwStore.CreateRun(req.Pipeline, req.Input)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create run: "+err.Error())
		return
	}

	opts := runOptionsFromSubmitRequest(req)

	if req.FromStep != "" {
		s.launchPipelineExecution(runID, req.Pipeline, req.Input, p, opts, req.FromStep)
	} else {
		s.launchPipelineExecution(runID, req.Pipeline, req.Input, p, opts)
	}

	writeJSON(w, http.StatusCreated, SubmitRunResponse{
		RunID:        runID,
		PipelineName: req.Pipeline,
		Status:       "running",
		StartedAt:    time.Now(),
	})
}

// handleRunLogs handles GET /api/runs/{id}/logs — get structured run logs.
func (s *Server) handleRunLogs(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing run ID")
		return
	}

	if _, err := s.runtime.store.GetRun(runID); err != nil {
		writeJSONError(w, http.StatusNotFound, "run not found")
		return
	}

	events, err := s.runtime.store.GetEvents(runID, state.EventQueryOptions{})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to get events")
		return
	}

	logs := make([]RunLogEntry, 0, len(events))
	for _, ev := range events {
		logs = append(logs, RunLogEntry{
			Timestamp:  ev.Timestamp,
			StepID:     ev.StepID,
			State:      ev.State,
			Persona:    ev.Persona,
			Message:    ev.Message,
			TokensUsed: ev.TokensUsed,
			DurationMs: ev.DurationMs,
		})
	}

	writeJSON(w, http.StatusOK, RunLogsResponse{RunID: runID, Logs: logs})
}

// handleGateApprove handles POST /api/runs/{id}/gates/{step}/approve
func (s *Server) handleGateApprove(w http.ResponseWriter, r *http.Request) {
	// CSRF protection: require a custom header that triggers CORS preflight
	// for cross-origin requests, preventing drive-by gate approvals.
	if r.Header.Get("X-Wave-Request") != "1" {
		writeJSONError(w, http.StatusForbidden, "missing required X-Wave-Request header")
		return
	}

	runID := r.PathValue("id")
	if runID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing run ID")
		return
	}

	stepID := r.PathValue("step")
	if stepID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing step ID")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req GateApproveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Choice == "" {
		writeJSONError(w, http.StatusBadRequest, "choice is required")
		return
	}

	if s.realtime.gateRegistry == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "gate registry not initialized")
		return
	}

	gate := s.realtime.gateRegistry.GetPending(runID)
	if gate == nil {
		writeJSONError(w, http.StatusNotFound, "no pending gate for this run")
		return
	}

	pendingStepID := s.realtime.gateRegistry.GetPendingStepID(runID)
	if pendingStepID != "" && pendingStepID != stepID {
		writeJSONError(w, http.StatusConflict,
			fmt.Sprintf("step mismatch: pending gate is for step %q, not %q", pendingStepID, stepID))
		return
	}

	choice := gate.FindChoiceByKey(req.Choice)
	if choice == nil {
		writeJSONError(w, http.StatusBadRequest, "invalid choice key: "+req.Choice)
		return
	}

	decision := &pipeline.GateDecision{
		Choice: choice.Key,
		Label:  choice.Label,
		Text:   req.Text,
		Target: choice.Target,
	}

	if err := s.realtime.gateRegistry.Resolve(runID, decision); err != nil {
		writeJSONError(w, http.StatusConflict, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, GateApproveResponse{
		RunID:  runID,
		StepID: stepID,
		Choice: choice.Key,
		Label:  choice.Label,
	})
}
