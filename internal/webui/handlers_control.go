package webui

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/audit"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/state"
	"gopkg.in/yaml.v3"
)

// validPipelineName matches safe pipeline names: alphanumeric, hyphens, underscores, dots.
var validPipelineName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// RunOptions holds CLI-parity options passed from the webui start form.
type RunOptions struct {
	Model   string
	Adapter string
	DryRun  bool
	Timeout int
	Steps   string
	Exclude string
}

// loggingEmitter wraps an event emitter and also logs events to the state store.
type loggingEmitter struct {
	inner event.EventEmitter
	store state.StateStore
	runID string
}

func (l *loggingEmitter) Emit(ev event.Event) {
	// Always forward to SSE broker for real-time streaming
	l.inner.Emit(ev)

	// Only log meaningful events to the database — skip empty heartbeat ticks
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

// launchPipelineExecution starts pipeline execution as a detached subprocess.
// The subprocess runs `wave run --pipeline <name> --run <runID> --input <input>`,
// fully independent of the server process. Server shutdown does not cancel runs.
// Dry-run mode is handled in-process since it completes instantly.
// This is shared by handleStartPipeline, handleRetryRun, and handleResumeRun.
// When fromStep is non-empty, the subprocess resumes from that step.
func (s *Server) launchPipelineExecution(runID, pipelineName, input string, _ *pipeline.Pipeline, opts RunOptions, fromStep ...string) {
	// Dry-run: handle in-process (instant, no subprocess needed)
	if opts.DryRun {
		if err := s.rwStore.UpdateRunStatus(runID, "completed", "dry run (validation only)", 0); err != nil {
			log.Printf("Warning: failed to update run %s status for dry-run: %v", runID, err)
		}
		return
	}

	// Spawn a detached subprocess — same mechanism as `wave run --detach`
	// Concurrency is enforced atomically at CreateRunWithLimit in the callers.
	if err := s.spawnDetachedRun(runID, pipelineName, input, opts, fromStep...); err != nil {
		log.Printf("Error: failed to spawn detached run %s: %v — falling back to in-process", runID, err)
		s.launchInProcess(runID, pipelineName, input, opts, fromStep...)
		return
	}
}

// spawnDetachedRun launches a `wave run` subprocess that is fully detached from
// the server process. The subprocess inherits the run ID and writes to the shared
// state DB, so the web UI can track progress via SSE and the runs page.
func (s *Server) spawnDetachedRun(runID, pipelineName, input string, opts RunOptions, fromStep ...string) error {
	args := []string{"run", "--pipeline", pipelineName, "--run", runID}
	if input != "" {
		args = append(args, "--input", input)
	}
	if len(fromStep) > 0 && fromStep[0] != "" {
		args = append(args, "--from-step", fromStep[0])
	}
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	if opts.Adapter != "" {
		args = append(args, "--adapter", opts.Adapter)
	}
	if opts.Timeout > 0 {
		args = append(args, "--timeout", fmt.Sprintf("%d", opts.Timeout))
	}
	if opts.Steps != "" {
		args = append(args, "--steps", opts.Steps)
	}
	if opts.Exclude != "" {
		args = append(args, "--exclude", opts.Exclude)
	}
	args = append(args, "--debug")

	waveBin, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to find wave binary: %w", err)
	}

	cmd := exec.Command(waveBin, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Dir = s.repoDir
	cmd.Env = buildServerDetachEnv()

	// Redirect output to .wave/logs/<runID>.log
	logsDir := filepath.Join(s.repoDir, ".wave", "logs")
	if mkErr := os.MkdirAll(logsDir, 0o755); mkErr != nil {
		return fmt.Errorf("failed to create logs directory: %w", mkErr)
	}
	logPath := filepath.Join(logsDir, runID+".log")
	logFile, logErr := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if logErr != nil {
		return fmt.Errorf("failed to create log file: %w", logErr)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if startErr := cmd.Start(); startErr != nil {
		logFile.Close()
		return fmt.Errorf("failed to start detached pipeline: %w", startErr)
	}

	logFile.Close()

	_ = s.rwStore.UpdateRunPID(runID, cmd.Process.Pid)
	_ = cmd.Process.Release()

	log.Printf("Pipeline %s (%s) launched as detached process (PID %d)", pipelineName, runID, cmd.Process.Pid)
	return nil
}

// buildServerDetachEnv constructs environment for detached subprocesses spawned by the server.
func buildServerDetachEnv() []string {
	path := os.Getenv("PATH")
	home := os.Getenv("HOME")
	if home != "" {
		toolBin := filepath.Join(home, ".local", "bin")
		if !strings.Contains(path, toolBin) {
			path = toolBin + string(os.PathListSeparator) + path
		}
	}

	env := []string{
		"HOME=" + home,
		"PATH=" + path,
	}
	for _, key := range []string{
		"ANTHROPIC_API_KEY", "CLAUDE_CODE_USE_BEDROCK", "AWS_PROFILE", "AWS_REGION",
		"TERM", "USER", "SHELL", "GH_TOKEN", "GITHUB_TOKEN",
		"XDG_DATA_HOME", "XDG_CONFIG_HOME", "XDG_CACHE_HOME",
	} {
		if val, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+val)
		}
	}
	return env
}

// launchInProcess is the fallback when detached spawn fails. It runs the pipeline
// in a goroutine tied to the server process (original behavior).
func (s *Server) launchInProcess(runID, pipelineName, input string, opts RunOptions, fromStep ...string) {
	var runner adapter.AdapterRunner
	if opts.Adapter != "" {
		runner = adapter.ResolveAdapter(opts.Adapter)
	} else if s.manifest != nil {
		for adapterName := range s.manifest.Adapters {
			runner = adapter.ResolveAdapter(adapterName)
			break
		}
	}
	if runner == nil {
		runner = adapter.ResolveAdapter("claude-code")
	}

	emitter := &loggingEmitter{
		inner: s.broker,
		store: s.rwStore,
		runID: runID,
	}

	traceLogger, traceErr := audit.NewTraceLogger()
	if traceErr != nil {
		log.Printf("Warning: failed to create trace logger: %v", traceErr)
	}

	execOpts := []pipeline.ExecutorOption{
		pipeline.WithRunID(runID),
		pipeline.WithStateStore(s.rwStore),
		pipeline.WithEmitter(emitter),
		pipeline.WithDebug(true),
	}
	if s.wsManager != nil {
		execOpts = append(execOpts, pipeline.WithWorkspaceManager(s.wsManager))
	}
	if traceLogger != nil {
		execOpts = append(execOpts, pipeline.WithAuditLogger(traceLogger))
	}
	if s.gateRegistry != nil {
		execOpts = append(execOpts, pipeline.WithGateHandler(NewWebUIGateHandler(runID, s.gateRegistry)))
	}
	if opts.Model != "" {
		execOpts = append(execOpts, pipeline.WithModelOverride(opts.Model))
	}
	if opts.Adapter != "" {
		execOpts = append(execOpts, pipeline.WithAdapterOverride(opts.Adapter))
	}
	if opts.Timeout > 0 {
		execOpts = append(execOpts, pipeline.WithStepTimeout(time.Duration(opts.Timeout)*time.Minute))
	}
	if opts.Steps != "" || opts.Exclude != "" {
		execOpts = append(execOpts, pipeline.WithStepFilter(pipeline.ParseStepFilter(opts.Steps, opts.Exclude)))
	}

	executor := pipeline.NewDefaultPipelineExecutor(runner, execOpts...)

	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.activeRuns[runID] = cancel
	s.mu.Unlock()

	go func() {
		defer func() {
			if traceLogger != nil {
				traceLogger.Close()
			}
			s.mu.Lock()
			delete(s.activeRuns, runID)
			s.mu.Unlock()
			cancel()
		}()

		if err := s.rwStore.UpdateRunStatus(runID, "running", "", 0); err != nil {
			log.Printf("Warning: failed to update run %s to running: %v", runID, err)
		}

		m := s.manifest
		if m == nil {
			m = &manifest.Manifest{}
		}

		var execErr error
		if len(fromStep) > 0 && fromStep[0] != "" {
			execErr = executor.ResumeWithValidation(ctx, &pipeline.Pipeline{}, m, input, fromStep[0], false, runID)
		} else {
			execErr = executor.Execute(ctx, &pipeline.Pipeline{}, m, input)
		}

		tokens := executor.GetTotalTokens()
		if execErr != nil {
			log.Printf("Pipeline %s (%s) failed: %v", pipelineName, runID, execErr)
			if err := s.rwStore.UpdateRunStatus(runID, "failed", execErr.Error(), tokens); err != nil {
				log.Printf("Warning: failed to update run %s to failed: %v", runID, err)
			}
		} else {
			if err := s.rwStore.UpdateRunStatus(runID, "completed", "", tokens); err != nil {
				log.Printf("Warning: failed to update run %s to completed: %v", runID, err)
			}
		}

		// Invalidate issue/PR caches so fresh data shows after pipeline completion
		s.cache.InvalidatePrefix("issues:")
		s.cache.InvalidatePrefix("prs:")
	}()
}

// handleStartPipeline handles POST /api/pipelines/{name}/start
func (s *Server) handleStartPipeline(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "missing pipeline name")
		return
	}

	// Check if pipeline is disabled by admin
	if s.isPipelineDisabled(name) {
		writeJSONError(w, http.StatusForbidden, "pipeline is disabled")
		return
	}

	var req StartPipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Load pipeline definition from .wave/pipelines/
	p, err := loadPipelineYAML(name)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to load pipeline: "+err.Error())
		return
	}

	// Create the run record in the DB — this ID is used everywhere
	runID, err := s.rwStore.CreateRun(name, req.Input)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create run: "+err.Error())
		return
	}

	s.launchPipelineExecution(runID, name, req.Input, p, RunOptions{
		Model:   req.Model,
		Adapter: req.Adapter,
		DryRun:  req.DryRun,
		Timeout: req.Timeout,
		Steps:   req.Steps,
		Exclude: req.Exclude,
	})

	resp := StartPipelineResponse{
		RunID:        runID,
		PipelineName: name,
		Status:       "running",
		StartedAt:    time.Now(),
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
		_ = json.NewDecoder(r.Body).Decode(&req) // best-effort; defaults are fine
	}

	// Check run exists and is cancellable
	run, err := s.store.GetRun(runID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "run not found")
		return
	}

	if !req.Force && run.Status != "running" && run.Status != "pending" {
		writeJSONError(w, http.StatusConflict, "run is not in a cancellable state (status: "+run.Status+")")
		return
	}

	// Cancel the goroutine context if the run is active
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
	resp := CancelRunResponse{
		RunID:  runID,
		Status: status,
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

	// Load pipeline definition
	p, err := loadPipelineYAML(originalRun.PipelineName)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load pipeline: "+err.Error())
		return
	}

	// Create a new run with the same parameters
	newRunID, err := s.rwStore.CreateRun(originalRun.PipelineName, originalRun.Input)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create retry run")
		return
	}

	// Launch actual pipeline execution
	s.launchPipelineExecution(newRunID, originalRun.PipelineName, originalRun.Input, p, RunOptions{})

	resp := RetryRunResponse{
		RunID:         newRunID,
		OriginalRunID: runID,
		PipelineName:  originalRun.PipelineName,
		Status:        "running",
		StartedAt:     time.Now(),
	}

	writeJSON(w, http.StatusCreated, resp)
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

	// Get original run — must be in a resumable state
	originalRun, err := s.store.GetRun(runID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "run not found")
		return
	}

	if originalRun.Status != "failed" && originalRun.Status != "cancelled" {
		writeJSONError(w, http.StatusConflict, "run is not in a resumable state (status: "+originalRun.Status+")")
		return
	}

	// Load pipeline definition
	p, err := loadPipelineYAML(originalRun.PipelineName)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load pipeline: "+err.Error())
		return
	}

	// Validate that the step exists in the pipeline
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

	// Create a new run record for the resumed execution
	newRunID, err := s.rwStore.CreateRun(originalRun.PipelineName, originalRun.Input)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create resume run")
		return
	}

	// Launch execution with resume from the specified step
	s.launchPipelineExecution(newRunID, originalRun.PipelineName, originalRun.Input, p, RunOptions{}, req.FromStep)

	resp := ResumeRunResponse{
		RunID:         newRunID,
		OriginalRunID: runID,
		PipelineName:  originalRun.PipelineName,
		FromStep:      req.FromStep,
		Status:        "running",
		StartedAt:     time.Now(),
	}

	writeJSON(w, http.StatusCreated, resp)
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

	// Load pipeline definition
	p, err := loadPipelineYAML(req.Pipeline)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to load pipeline: "+err.Error())
		return
	}

	// Create run record
	runID, err := s.rwStore.CreateRun(req.Pipeline, req.Input)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create run: "+err.Error())
		return
	}

	s.launchPipelineExecution(runID, req.Pipeline, req.Input, p, RunOptions{
		Model:   req.Model,
		Adapter: req.Adapter,
		DryRun:  req.DryRun,
		Timeout: req.Timeout,
		Steps:   req.Steps,
		Exclude: req.Exclude,
	})

	resp := SubmitRunResponse{
		RunID:        runID,
		PipelineName: req.Pipeline,
		Status:       "running",
		StartedAt:    time.Now(),
	}

	writeJSON(w, http.StatusCreated, resp)
}

// handleRunLogs handles GET /api/runs/{id}/logs — get structured run logs.
func (s *Server) handleRunLogs(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing run ID")
		return
	}

	// Verify run exists
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

	writeJSON(w, http.StatusOK, RunLogsResponse{
		RunID: runID,
		Logs:  logs,
	})
}

// loadPipelineYAML loads a pipeline definition from .wave/pipelines/.
// The name must match [a-zA-Z0-9][a-zA-Z0-9._-]* to prevent path traversal.
func loadPipelineYAML(name string) (*pipeline.Pipeline, error) {
	if !validPipelineName.MatchString(name) {
		return nil, fmt.Errorf("invalid pipeline name")
	}

	candidates := []string{
		".wave/pipelines/" + name + ".yaml",
		".wave/pipelines/" + name,
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

	var p pipeline.Pipeline
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("invalid pipeline definition")
	}

	return &p, nil
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

	// Limit request body to 1MB to prevent abuse via oversized freeform text.
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

	// Check that a gate is actually pending for this run
	if s.gateRegistry == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "gate registry not initialized")
		return
	}

	gate := s.gateRegistry.GetPending(runID)
	if gate == nil {
		writeJSONError(w, http.StatusNotFound, "no pending gate for this run")
		return
	}

	// Verify that the step ID in the URL matches the actual pending gate step.
	// This prevents approving the wrong gate when steps change between request
	// construction and submission.
	pendingStepID := s.gateRegistry.GetPendingStepID(runID)
	if pendingStepID != "" && pendingStepID != stepID {
		writeJSONError(w, http.StatusConflict,
			fmt.Sprintf("step mismatch: pending gate is for step %q, not %q", pendingStepID, stepID))
		return
	}

	// Validate the choice key against the gate's choices
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

	// Limit request body size for consistency with other POST handlers.
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

	s.launchPipelineExecution(newRunID, originalRun.PipelineName, originalRun.Input, p, RunOptions{}, resumeStep)

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

	// Limit request body size for consistency with other POST handlers.
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
	names := []string{"claude-code"}
	if s.manifest != nil {
		for name := range s.manifest.Adapters {
			names = append(names, name)
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"adapters": names})
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
