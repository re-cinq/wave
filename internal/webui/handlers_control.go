//go:build webui

package webui

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/audit"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/state"
	"gopkg.in/yaml.v3"
)

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
		l.store.LogEvent(l.runID, ev.StepID, ev.State, ev.Persona, ev.Message, ev.TokensUsed, ev.DurationMs)
	}
}

// isHeartbeat returns true for progress ticker events that carry no useful info.
func isHeartbeat(ev event.Event) bool {
	return ev.Message == "" && (ev.State == "step_progress" || ev.State == "stream_activity") && ev.TokensUsed == 0 && ev.DurationMs == 0
}

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

	// Resolve adapter from manifest
	var runner adapter.AdapterRunner
	if s.manifest != nil {
		for adapterName := range s.manifest.Adapters {
			runner = adapter.ResolveAdapter(adapterName)
			break
		}
	}
	if runner == nil {
		runner = adapter.ResolveAdapter("claude-code")
	}

	// Create a logging emitter that writes to both SSE broker and state store
	emitter := &loggingEmitter{
		inner: s.broker,
		store: s.rwStore,
		runID: runID,
	}

	// Create audit trace logger for this run
	traceLogger, traceErr := audit.NewTraceLogger()
	if traceErr != nil {
		log.Printf("Warning: failed to create trace logger: %v", traceErr)
	}

	// Create executor — use the DB runID as the executor's pipeline ID
	// so that SaveStepState/SavePipelineState writes match what the dashboard queries.
	// Always enable debug mode for detailed event messages in the dashboard.
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

	executor := pipeline.NewDefaultPipelineExecutor(runner, execOpts...)

	// Execute in background goroutine
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

		// Update to running
		s.rwStore.UpdateRunStatus(runID, "running", "", 0)

		m := s.manifest
		if m == nil {
			m = &manifest.Manifest{}
		}

		execErr := executor.Execute(ctx, p, m, req.Input)

		tokens := executor.GetTotalTokens()
		if execErr != nil {
			log.Printf("Pipeline %s (%s) failed: %v", name, runID, execErr)
			s.rwStore.UpdateRunStatus(runID, "failed", execErr.Error(), tokens)
		} else {
			s.rwStore.UpdateRunStatus(runID, "completed", "", tokens)
		}
	}()

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

// loadPipelineYAML loads a pipeline definition from .wave/pipelines/.
func loadPipelineYAML(name string) (*pipeline.Pipeline, error) {
	candidates := []string{
		".wave/pipelines/" + name + ".yaml",
		".wave/pipelines/" + name,
		name,
	}

	var pipelinePath string
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			pipelinePath = candidate
			break
		}
	}

	if pipelinePath == "" {
		return nil, os.ErrNotExist
	}

	data, err := os.ReadFile(pipelinePath)
	if err != nil {
		return nil, err
	}

	var p pipeline.Pipeline
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, err
	}

	return &p, nil
}
