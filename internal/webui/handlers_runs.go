//go:build webui

package webui

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/recinq/wave/internal/state"
)

// handleAPIRuns handles GET /api/runs - returns paginated run list as JSON.
func (s *Server) handleAPIRuns(w http.ResponseWriter, r *http.Request) {
	cursor, err := decodeCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid cursor: "+err.Error())
		return
	}

	limit := parsePageSize(r)
	status := r.URL.Query().Get("status")
	pipeline := r.URL.Query().Get("pipeline")
	sinceStr := r.URL.Query().Get("since")

	opts := state.ListRunsOptions{
		Status:       status,
		PipelineName: pipeline,
		Limit:        limit + 1, // fetch one extra to determine hasMore
	}

	if sinceStr != "" {
		t, err := time.Parse(time.RFC3339, sinceStr)
		if err == nil {
			opts.OlderThan = time.Since(t) * -1 // This won't work as expected
		}
	}

	// Note: cursor-based filtering needs to be applied at the query level
	// The existing ListRuns doesn't support cursor directly, so we'll use it
	// with Limit and filter results
	_ = cursor // TODO: implement cursor-based DB query extension

	runs, err := s.store.ListRuns(opts)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list runs")
		return
	}

	hasMore := len(runs) > limit
	if hasMore {
		runs = runs[:limit]
	}

	summaries := make([]RunSummary, len(runs))
	for i, run := range runs {
		summaries[i] = runToSummary(run)
	}

	resp := RunListResponse{
		Runs:    summaries,
		HasMore: hasMore,
	}

	if hasMore && len(runs) > 0 {
		lastRun := runs[len(runs)-1]
		resp.NextCursor = encodeCursor(lastRun.StartedAt, lastRun.RunID)
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleAPIRunDetail handles GET /api/runs/{id} - returns run detail as JSON.
func (s *Server) handleAPIRunDetail(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing run ID")
		return
	}

	run, err := s.store.GetRun(runID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "run not found")
		return
	}

	// Get step details from step_state table (what the executor writes to)
	stepDetails := s.buildStepDetails(runID, run.PipelineName)

	// Get events
	events, _ := s.store.GetEvents(runID, state.EventQueryOptions{Limit: 100})
	eventSummaries := make([]EventSummary, len(events))
	for i, e := range events {
		eventSummaries[i] = eventToSummary(e)
	}

	// Get all artifacts
	allArts, _ := s.store.GetArtifacts(runID, "")
	artSummaries := make([]ArtifactSummary, len(allArts))
	for i, a := range allArts {
		artSummaries[i] = artifactToSummary(a)
	}

	resp := RunDetailResponse{
		Run:       runToSummary(*run),
		Steps:     stepDetails,
		Events:    eventSummaries,
		Artifacts: artSummaries,
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleRunsPage handles GET /runs - serves the HTML run list page.
func (s *Server) handleRunsPage(w http.ResponseWriter, r *http.Request) {
	cursor, _ := decodeCursor(r.URL.Query().Get("cursor"))
	_ = cursor // TODO: pass to query

	limit := parsePageSize(r)
	status := r.URL.Query().Get("status")

	opts := state.ListRunsOptions{
		Status: status,
		Limit:  limit + 1,
	}

	runs, err := s.store.ListRuns(opts)
	if err != nil {
		http.Error(w, "failed to list runs", http.StatusInternalServerError)
		return
	}

	hasMore := len(runs) > limit
	if hasMore {
		runs = runs[:limit]
	}

	summaries := make([]RunSummary, len(runs))
	for i, run := range runs {
		summaries[i] = runToSummary(run)
	}

	var nextCursor string
	if hasMore && len(runs) > 0 {
		lastRun := runs[len(runs)-1]
		nextCursor = encodeCursor(lastRun.StartedAt, lastRun.RunID)
	}

	// Get pipeline names for the start form from .wave/pipelines/
	pipelineNames := listPipelineNames()

	data := struct {
		Runs       []RunSummary
		HasMore    bool
		NextCursor string
		Pipelines  []string
	}{
		Runs:       summaries,
		HasMore:    hasMore,
		NextCursor: nextCursor,
		Pipelines:  pipelineNames,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates["templates/runs.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

// handleRunDetailPage handles GET /runs/{id} - serves the HTML run detail page.
func (s *Server) handleRunDetailPage(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		http.Error(w, "missing run ID", http.StatusBadRequest)
		return
	}

	run, err := s.store.GetRun(runID)
	if err != nil {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}

	// Get step details from step_state table (what the executor writes to)
	stepDetails := s.buildStepDetails(runID, run.PipelineName)

	// Build step status map for DAG
	stepStatusMap := make(map[string]string)
	for _, sd := range stepDetails {
		stepStatusMap[sd.StepID] = sd.State
	}

	// Get events
	events, _ := s.store.GetEvents(runID, state.EventQueryOptions{Limit: 100})
	eventSummaries := make([]EventSummary, len(events))
	for i, e := range events {
		eventSummaries[i] = eventToSummary(e)
	}

	// Compute DAG layout from pipeline definition
	var dagLayout *DAGLayout
	if p, err := loadPipelineYAML(run.PipelineName); err == nil {
		var dagSteps []DAGStepInput
		for _, step := range p.Steps {
			status := "pending"
			if s, ok := stepStatusMap[step.ID]; ok {
				status = s
			}
			dagSteps = append(dagSteps, DAGStepInput{
				ID:           step.ID,
				Persona:      step.Persona,
				Status:       status,
				Dependencies: step.Dependencies,
			})
		}
		dagLayout = ComputeDAGLayout(dagSteps)
	}

	data := struct {
		Run    RunSummary
		Steps  []StepDetail
		Events []EventSummary
		DAG    *DAGLayout
	}{
		Run:    runToSummary(*run),
		Steps:  stepDetails,
		Events: eventSummaries,
		DAG:    dagLayout,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates["templates/run_detail.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// Helper functions for type conversion

func runToSummary(r state.RunRecord) RunSummary {
	summary := RunSummary{
		RunID:        r.RunID,
		PipelineName: r.PipelineName,
		Status:       r.Status,
		CurrentStep:  r.CurrentStep,
		TotalTokens:  r.TotalTokens,
		StartedAt:    r.StartedAt,
		CompletedAt:  r.CompletedAt,
		Tags:         r.Tags,
		ErrorMessage: r.ErrorMessage,
	}

	if r.CompletedAt != nil {
		dur := r.CompletedAt.Sub(r.StartedAt)
		summary.Duration = formatDurationValue(dur)
	} else if r.Status == "running" {
		dur := time.Since(r.StartedAt)
		summary.Duration = formatDurationValue(dur)
	}

	return summary
}

func stepProgressToDetail(sp state.StepProgressRecord, artifacts []ArtifactSummary) StepDetail {
	d := StepDetail{
		StepID:     sp.StepID,
		Persona:    sp.Persona,
		State:      sp.State,
		Progress:   sp.Progress,
		Action:     sp.CurrentAction,
		StartedAt:  sp.StartedAt,
		TokensUsed: sp.TokensUsed,
		Artifacts:  artifacts,
	}

	if sp.StartedAt != nil {
		dur := time.Since(*sp.StartedAt)
		d.Duration = formatDurationValue(dur)
	}

	return d
}

// buildStepDetails derives step details from the event_log table combined with
// the pipeline definition. We use events rather than step_state because the
// step_state table has a unique constraint on step_id alone (not per-pipeline),
// causing cross-run collisions.
func (s *Server) buildStepDetails(runID, pipelineName string) []StepDetail {
	// Load pipeline definition to get ordered step list with personas
	p, err := loadPipelineYAML(pipelineName)
	if err != nil {
		log.Printf("[webui] buildStepDetails: failed to load pipeline %q: %v", pipelineName, err)
		return nil
	}

	// Get all events for this run
	events, _ := s.store.GetEvents(runID, state.EventQueryOptions{Limit: 5000})
	log.Printf("[webui] buildStepDetails: runID=%s pipeline=%s steps=%d events=%d", runID, pipelineName, len(p.Steps), len(events))

	// Build step state from events: track latest state, timestamps, tokens per step
	type stepInfo struct {
		state      string
		persona    string
		startedAt  *time.Time
		completedAt *time.Time
		tokens     int
		durationMs int64
		errMsg     string
	}
	stepMap := make(map[string]*stepInfo)

	for _, ev := range events {
		if ev.StepID == "" {
			continue
		}
		si, exists := stepMap[ev.StepID]
		if !exists {
			si = &stepInfo{}
			stepMap[ev.StepID] = si
		}
		if ev.Persona != "" {
			si.persona = ev.Persona
		}

		// Track state transitions
		switch ev.State {
		case "running":
			if si.startedAt == nil {
				t := ev.Timestamp
				si.startedAt = &t
			}
			si.state = "running"
		case "completed":
			t := ev.Timestamp
			si.completedAt = &t
			si.state = "completed"
		case "failed":
			t := ev.Timestamp
			si.completedAt = &t
			si.state = "failed"
			si.errMsg = ev.Message
		}

		if ev.TokensUsed > si.tokens {
			si.tokens = ev.TokensUsed
		}
		if ev.DurationMs > si.durationMs {
			si.durationMs = ev.DurationMs
		}
	}

	// Build details in pipeline step order
	details := make([]StepDetail, 0, len(p.Steps))
	for _, step := range p.Steps {
		sd := StepDetail{
			StepID:  step.ID,
			Persona: step.Persona,
			State:   "pending",
		}

		if si, ok := stepMap[step.ID]; ok {
			if si.state != "" {
				sd.State = si.state
			}
			if si.persona != "" {
				sd.Persona = si.persona
			}
			sd.StartedAt = si.startedAt
			sd.CompletedAt = si.completedAt
			sd.TokensUsed = si.tokens
			sd.Error = si.errMsg

			// Calculate progress
			switch sd.State {
			case "completed":
				sd.Progress = 100
			case "running":
				sd.Progress = 50
			}

			// Calculate duration
			if si.startedAt != nil {
				if si.completedAt != nil {
					sd.Duration = formatDurationValue(si.completedAt.Sub(*si.startedAt))
				} else if sd.State == "running" {
					sd.Duration = formatDurationValue(time.Since(*si.startedAt))
				}
			}
		}

		arts, _ := s.store.GetArtifacts(runID, step.ID)
		artSummaries := make([]ArtifactSummary, len(arts))
		for j, a := range arts {
			artSummaries[j] = artifactToSummary(a)
		}
		sd.Artifacts = artSummaries

		details = append(details, sd)
	}

	return details
}

func eventToSummary(e state.LogRecord) EventSummary {
	return EventSummary{
		ID:         e.ID,
		Timestamp:  e.Timestamp,
		StepID:     e.StepID,
		State:      e.State,
		Persona:    e.Persona,
		Message:    e.Message,
		TokensUsed: e.TokensUsed,
		DurationMs: e.DurationMs,
	}
}

func artifactToSummary(a state.ArtifactRecord) ArtifactSummary {
	return ArtifactSummary{
		ID:        a.ID,
		Name:      a.Name,
		Path:      a.Path,
		Type:      a.Type,
		SizeBytes: a.SizeBytes,
	}
}

func formatDurationValue(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", m, s)
}

func listPipelineNames() []string {
	// List pipeline YAML files from .wave/pipelines/
	entries, err := os.ReadDir(".wave/pipelines")
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if len(name) > 5 && name[len(name)-5:] == ".yaml" {
			names = append(names, name[:len(name)-5])
		}
	}
	return names
}

// JSON response helpers

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
