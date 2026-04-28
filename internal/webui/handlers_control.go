package webui

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/runner"
	"github.com/recinq/wave/internal/state"
)

// validPipelineName matches safe pipeline names: alphanumeric, hyphens, underscores, dots.
var validPipelineName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// RunOptions is the CLI-parity option set forwarded from the webui start
// form to internal/runner. Aliased so webui handlers and request DTOs keep
// their existing field names while sharing one canonical shape with the cmd
// path.
type RunOptions = runner.Options

// loggingEmitter wraps an event emitter and also logs events to the state
// store, so the webui dashboard sees both real-time SSE updates and the
// persistent event timeline.
type loggingEmitter struct {
	inner event.EventEmitter
	store state.EventStore
	runID string
}

func (l *loggingEmitter) Emit(ev event.Event) {
	// Always forward to SSE broker for real-time streaming.
	l.inner.Emit(ev)

	// Skip empty heartbeat ticks — they carry no useful info.
	if l.store != nil && !isHeartbeat(ev) {
		if err := l.store.LogEvent(l.runID, ev.StepID, ev.State, ev.Persona, ev.Message, ev.TokensUsed, ev.DurationMs, ev.Model, ev.ConfiguredModel, ev.Adapter); err != nil {
			log.Printf("Warning: failed to log event for run %s: %v", l.runID, err)
		}
	}
}

// isHeartbeat returns true for progress ticker events that carry no useful info.
func isHeartbeat(ev event.Event) bool {
	return ev.Message == "" && (ev.State == "step_progress" || ev.State == "stream_activity") && ev.TokensUsed == 0 && ev.DurationMs == 0
}

// launchPipelineExecution starts pipeline execution as a detached subprocess
// via internal/runner. The subprocess is fully independent of the server
// process — server shutdown does not cancel runs. Dry-run mode short-circuits
// to a synchronous status update because validation completes instantly.
//
// This helper is shared by handleStartPipeline, handleRetryRun, handleResumeRun,
// and handleForkRun. When fromStep is non-empty the subprocess resumes from
// that step.
func (s *Server) launchPipelineExecution(runID, pipelineName, input string, _ *pipeline.Pipeline, opts RunOptions, fromStep ...string) {
	// Dry-run: handle in-process (instant, no subprocess needed).
	if opts.DryRun {
		if err := s.rwStore.UpdateRunStatus(runID, "completed", "dry run (validation only)", 0); err != nil {
			log.Printf("Warning: failed to update run %s status for dry-run: %v", runID, err)
		}
		return
	}

	// Spawn a detached subprocess via the shared runner. Concurrency is
	// enforced atomically at CreateRunWithLimit by the calling handler.
	if err := s.spawnDetachedRun(runID, pipelineName, input, opts, fromStep...); err != nil {
		log.Printf("Error: failed to spawn detached run %s: %v — falling back to in-process", runID, err)
		s.launchInProcess(runID, pipelineName, input, opts, fromStep...)
	}
}

// spawnDetachedRun delegates to runner.Detach, reusing the run ID the handler
// already created in the state DB. The runner consumes the same flag-spec
// table the CLI uses, so flag changes only need to land in one place.
func (s *Server) spawnDetachedRun(runID, pipelineName, input string, opts RunOptions, fromStep ...string) error {
	opts.Pipeline = pipelineName
	opts.Input = input
	opts.RunID = runID
	if len(fromStep) > 0 && fromStep[0] != "" {
		opts.FromStep = fromStep[0]
	}
	// Never recurse into detached mode in the subprocess — runner.Detach
	// is already producing a Setsid'd child.
	opts.Detach = false
	// Force --debug for visibility into server-launched runs (matches the
	// pre-extraction behaviour where buildDetachedArgs always appended --debug).
	opts.Output.Verbose = true

	cfg := runner.DetachConfig{
		WorkDir:  s.repoDir,
		ExtraEnv: []string{"GH_TOKEN", "GITHUB_TOKEN"},
	}
	// runner.Detach reuses the pre-created run row when opts.RunID exists
	// in the store, so no extra coordination is needed.
	if _, err := runner.Detach(opts, s.rwStore, 0, cfg); err != nil {
		return err
	}
	log.Printf("Pipeline %s (%s) launched as detached process", pipelineName, runID)
	return nil
}

// launchInProcess runs the pipeline inside the server process via
// internal/runner. This is the fallback path when subprocess spawning fails;
// the server-shutdown path will cancel these via activeRuns.
func (s *Server) launchInProcess(runID, pipelineName, input string, opts RunOptions, fromStep ...string) {
	resolvedFromStep := ""
	if len(fromStep) > 0 {
		resolvedFromStep = fromStep[0]
	}

	emitter := &loggingEmitter{
		inner: s.broker,
		store: s.rwStore,
		runID: runID,
	}

	var gateHandler pipeline.GateHandler
	if s.gateRegistry != nil {
		gateHandler = NewWebUIGateHandler(runID, s.gateRegistry)
	}

	cancel := runner.LaunchInProcess(runner.InProcessConfig{
		RunID:            runID,
		PipelineName:     pipelineName,
		Input:            input,
		Manifest:         s.manifest,
		Store:            s.rwStore,
		Emitter:          emitter,
		WorkspaceManager: s.wsManager,
		GateHandler:      gateHandler,
		FromStep:         resolvedFromStep,
		Options:          opts,
		OnComplete: func(string, error) {
			// Invalidate issue/PR caches so fresh data shows after pipeline completion.
			s.cache.InvalidatePrefix("issues:")
			s.cache.InvalidatePrefix("prs:")

			s.mu.Lock()
			delete(s.activeRuns, runID)
			s.mu.Unlock()
		},
	})

	s.mu.Lock()
	s.activeRuns[runID] = cancel
	s.mu.Unlock()
}

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

	runID, err := s.rwStore.CreateRun(name, req.Input)
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

	run, err := s.store.GetRun(runID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "run not found")
		return
	}

	if !req.Force && run.Status != "running" && run.Status != "pending" {
		writeJSONError(w, http.StatusConflict, "run is not in a cancellable state (status: "+run.Status+")")
		return
	}

	s.mu.Lock()
	if cancelFn, ok := s.activeRuns[runID]; ok {
		cancelFn()
	}
	s.mu.Unlock()

	if err := s.rwStore.RequestCancellation(runID, req.Force); err != nil {
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

	originalRun, err := s.store.GetRun(runID)
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

	newRunID, err := s.rwStore.CreateRun(originalRun.PipelineName, originalRun.Input)
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

	originalRun, err := s.store.GetRun(runID)
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

	newRunID, err := s.rwStore.CreateRun(originalRun.PipelineName, originalRun.Input)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create resume run")
		return
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

	runID, err := s.rwStore.CreateRun(req.Pipeline, req.Input)
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

	if _, err := s.store.GetRun(runID); err != nil {
		writeJSONError(w, http.StatusNotFound, "run not found")
		return
	}

	events, err := s.store.GetEvents(runID, state.EventQueryOptions{})
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

// loadPipelineYAML loads a pipeline definition from .agents/pipelines/.
// The name must match [a-zA-Z0-9][a-zA-Z0-9._-]* to prevent path traversal.
func loadPipelineYAML(name string) (*pipeline.Pipeline, error) {
	if !validPipelineName.MatchString(name) {
		return nil, fmt.Errorf("invalid pipeline name")
	}

	candidates := []string{
		".agents/pipelines/" + name + ".yaml",
		".agents/pipelines/" + name,
	}

	var pipelinePath string
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			pipelinePath = candidate
			break
		}
	}

	if pipelinePath == "" {
		return nil, fmt.Errorf("pipeline not found")
	}

	data, err := os.ReadFile(pipelinePath)
	if err != nil {
		return nil, fmt.Errorf("pipeline not found")
	}

	loader := &pipeline.YAMLPipelineLoader{}
	return loader.Unmarshal(data)
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

	if s.gateRegistry == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "gate registry not initialized")
		return
	}

	gate := s.gateRegistry.GetPending(runID)
	if gate == nil {
		writeJSONError(w, http.StatusNotFound, "no pending gate for this run")
		return
	}

	pendingStepID := s.gateRegistry.GetPendingStepID(runID)
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

	if err := s.gateRegistry.Resolve(runID, decision); err != nil {
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

	originalRun, err := s.store.GetRun(runID)
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

	fm := pipeline.NewForkManager(s.rwStore)
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
		if err := s.rwStore.UpdateRunStatus(newRunID, "completed", "", 0); err != nil {
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

	run, err := s.store.GetRun(runID)
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

	if err := s.rwStore.DeleteCheckpointsAfterStep(runID, rewindIndex); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "rewind failed: "+err.Error())
		return
	}

	rewindMsg := "rewound to step: " + req.ToStep
	if err := s.rwStore.UpdateRunStatus(runID, "failed", rewindMsg, run.TotalTokens); err != nil {
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

// handleAPIAdapters handles GET /api/adapters — returns available adapter names.
func (s *Server) handleAPIAdapters(w http.ResponseWriter, r *http.Request) {
	var names []string
	if s.manifest != nil {
		for name := range s.manifest.Adapters {
			names = append(names, name)
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"adapters": names})
}

// handleAPIModels handles GET /api/models — returns suggested model names.
// Collects tier names (cheapest, balanced, strongest) plus all concrete model
// IDs from adapter default_model and tier_models values.
func (s *Server) handleAPIModels(w http.ResponseWriter, r *http.Request) {
	seen := map[string]bool{}
	var models []string
	add := func(m string) {
		if m == "" || m == "default" || seen[m] {
			return
		}
		seen[m] = true
		models = append(models, m)
	}
	add("cheapest")
	add("balanced")
	add("strongest")
	if s.manifest != nil {
		for _, a := range s.manifest.Adapters {
			add(a.DefaultModel)
			for _, m := range a.TierModels {
				add(m)
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"models": models})
}

// handleForkPoints handles GET /api/runs/{id}/fork-points
func (s *Server) handleForkPoints(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing run ID")
		return
	}

	if _, err := s.store.GetRun(runID); err != nil {
		writeJSONError(w, http.StatusNotFound, "run not found")
		return
	}

	fm := pipeline.NewForkManager(s.store)
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

// runOptionsFromStartRequest projects an HTTP StartPipelineRequest onto the
// shared runner.Options struct so the launch path is identical regardless of
// which handler triggered the run.
func runOptionsFromStartRequest(req StartPipelineRequest) runner.Options {
	return runner.Options{
		Model:             req.Model,
		Adapter:           req.Adapter,
		DryRun:            req.DryRun,
		FromStep:          req.FromStep,
		Force:             req.Force,
		Detach:            req.Detach,
		Timeout:           req.Timeout,
		Steps:             req.Steps,
		Exclude:           req.Exclude,
		OnFailure:         req.OnFailure,
		Continuous:        req.Continuous,
		Source:            req.Source,
		MaxIterations:     req.MaxIterations,
		Delay:             req.Delay,
		Mock:              req.Mock,
		PreserveWorkspace: req.PreserveWorkspace,
		AutoApprove:       req.AutoApprove,
		NoRetro:           req.NoRetro,
		ForceModel:        req.ForceModel,
	}
}

// runOptionsFromSubmitRequest mirrors runOptionsFromStartRequest for the
// /api/runs submit endpoint.
func runOptionsFromSubmitRequest(req SubmitRunRequest) runner.Options {
	return runner.Options{
		Model:             req.Model,
		Adapter:           req.Adapter,
		DryRun:            req.DryRun,
		FromStep:          req.FromStep,
		Force:             req.Force,
		Detach:            req.Detach,
		Timeout:           req.Timeout,
		Steps:             req.Steps,
		Exclude:           req.Exclude,
		OnFailure:         req.OnFailure,
		Continuous:        req.Continuous,
		Source:            req.Source,
		MaxIterations:     req.MaxIterations,
		Delay:             req.Delay,
		Mock:              req.Mock,
		PreserveWorkspace: req.PreserveWorkspace,
		AutoApprove:       req.AutoApprove,
		NoRetro:           req.NoRetro,
		ForceModel:        req.ForceModel,
	}
}

