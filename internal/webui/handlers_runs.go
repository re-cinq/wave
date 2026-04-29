package webui

import (
	"log"
	"net/http"
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
			opts.SinceUnix = t.Unix()
		}
	}

	if cursor != nil {
		opts.BeforeUnix = cursor.Timestamp
		opts.BeforeRunID = cursor.RunID
	}

	runs, err := s.runtime.store.ListRuns(opts)
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
	s.enrichRunSummaries(summaries, runs)

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

// handleAPIRunChildren handles GET /api/runs/{id}/children — returns
// the immediate child runs of a parent run, plus the rolled-up subtree
// token total. Used by the WebUI to render iterate / sub-pipeline
// children inline on the parent run page (issue #1450).
func (s *Server) handleAPIRunChildren(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing run ID")
		return
	}

	if _, err := s.runtime.store.GetRun(runID); err != nil {
		writeJSONError(w, http.StatusNotFound, "run not found")
		return
	}

	children, err := s.runtime.store.GetChildRuns(runID)
	if err != nil {
		log.Printf("[webui] failed to get children for run %s: %v", runID, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to query children")
		return
	}
	summaries := make([]RunSummary, len(children))
	for i, c := range children {
		summaries[i] = runToSummary(c)
	}

	subtreeTokens, err := s.runtime.store.GetSubtreeTokens(runID)
	if err != nil {
		log.Printf("[webui] failed to compute subtree tokens for run %s: %v", runID, err)
		// Soft-fail: still return children, just without the rollup.
	}

	resp := struct {
		ParentRunID   string       `json:"parent_run_id"`
		Children      []RunSummary `json:"children"`
		SubtreeTokens int64        `json:"subtree_tokens"`
	}{
		ParentRunID:   runID,
		Children:      summaries,
		SubtreeTokens: subtreeTokens,
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

	run, err := s.runtime.store.GetRun(runID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "run not found")
		return
	}

	// Get step details from step_state table (what the executor writes to)
	stepDetails := s.buildStepDetails(runID, run.PipelineName, run.Status)

	// Get events
	events, err := s.runtime.store.GetEvents(runID, state.EventQueryOptions{Limit: 5000})
	if err != nil {
		log.Printf("[webui] failed to get events for run %s: %v", runID, err)
	}
	eventSummaries := make([]EventSummary, len(events))
	for i, e := range events {
		eventSummaries[i] = eventToSummary(e)
	}

	// Get all artifacts
	allArts, err := s.runtime.store.GetArtifacts(runID, "")
	if err != nil {
		log.Printf("[webui] failed to get artifacts for run %s: %v", runID, err)
	}
	artSummaries := deduplicateArtifacts(allArts)

	runSummary := runToSummary(*run)
	if subtree, err := s.runtime.store.GetSubtreeTokens(runID); err == nil {
		runSummary.SubtreeTokens = subtree
	}

	resp := RunDetailResponse{
		Run:       runSummary,
		Steps:     stepDetails,
		Events:    eventSummaries,
		Artifacts: artSummaries,
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleRunsPage serves GET /runs — runs list with Fat Gantt design.
func (s *Server) handleRunsPage(w http.ResponseWriter, r *http.Request) {
	cursor, err := decodeCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		log.Printf("[webui] invalid cursor parameter: %v", err)
	}
	limit := parsePageSize(r)
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "all"
	}
	pipelineFilter := r.URL.Query().Get("pipeline")

	// "all" means no status filter
	queryStatus := status
	if queryStatus == "all" {
		queryStatus = ""
	}

	opts := state.ListRunsOptions{
		Status:       queryStatus,
		PipelineName: pipelineFilter,
		Limit:        limit + 1,
	}
	if cursor != nil {
		opts.BeforeUnix = cursor.Timestamp
		opts.BeforeRunID = cursor.RunID
	}

	runs, err := s.runtime.store.ListRuns(opts)
	if err != nil {
		http.Error(w, "failed to list runs", http.StatusInternalServerError)
		return
	}

	hasMore := len(runs) > limit
	if hasMore {
		runs = runs[:limit]
	}

	allSummaries := make([]RunSummary, 0, len(runs))
	filteredRuns := make([]state.RunRecord, 0, len(runs))
	for _, run := range runs {
		// Running runs are always shown in the dedicated running-pipelines section;
		// exclude them from the main list to avoid duplication.
		if run.Status == "running" {
			continue
		}
		allSummaries = append(allSummaries, runToSummary(run))
		filteredRuns = append(filteredRuns, run)
	}
	s.enrichRunSummaries(allSummaries, filteredRuns)
	summaries := nestChildRuns(allSummaries)

	var nextCursor string
	if hasMore && len(runs) > 0 {
		lastRun := runs[len(runs)-1]
		nextCursor = encodeCursor(lastRun.StartedAt, lastRun.RunID)
	}

	// Collect unique pipeline names for filter
	pipelineNames := make(map[string]bool)
	for _, r := range allSummaries {
		pipelineNames[r.PipelineName] = true
	}
	var pipelines []string
	for name := range pipelineNames {
		pipelines = append(pipelines, name)
	}

	runningRuns := s.collectRunningRuns(pipelineFilter)

	data := struct {
		ActivePage     string
		Runs           []RunSummary
		HasMore        bool
		NextCursor     string
		Pipelines      []string
		FilterStatus   string
		FilterPipeline string
		RunningRuns    []RunSummary
		RunningCount   int
	}{
		ActivePage:     "runs",
		Runs:           summaries,
		HasMore:        hasMore,
		NextCursor:     nextCursor,
		Pipelines:      pipelines,
		FilterStatus:   status,
		FilterPipeline: pipelineFilter,
		RunningRuns:    runningRuns,
		RunningCount:   len(runningRuns),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.assets.templates["templates/runs.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// collectRunningRuns queries running runs for the running-pipelines section
// of the runs page, then nests their child runs (running plus already-finished
// children of running parents) underneath each parent.
func (s *Server) collectRunningRuns(pipelineFilter string) []RunSummary {
	runningRecs, err := s.runtime.store.ListRuns(state.ListRunsOptions{
		Status:       "running",
		PipelineName: pipelineFilter,
		Limit:        0,
	})
	if err != nil {
		log.Printf("[webui] failed to list running runs: %v", err)
		runningRecs = nil
	}
	runningRuns := make([]RunSummary, 0)
	// Build parent→children map for nesting child runs under their parent
	childMap := make(map[string][]RunSummary)
	for _, rec := range runningRecs {
		if rec.ParentRunID != "" {
			childMap[rec.ParentRunID] = append(childMap[rec.ParentRunID], runToSummary(rec))
			continue
		}
		runningRuns = append(runningRuns, runToSummary(rec))
	}
	// Also fetch completed/failed children of running parents from the full run list
	if s.runtime.store != nil {
		for i := range runningRuns {
			if children, err := s.runtime.store.GetChildRuns(runningRuns[i].RunID); err == nil {
				for _, ch := range children {
					found := false
					for _, existing := range childMap[runningRuns[i].RunID] {
						if existing.RunID == ch.RunID {
							found = true
							break
						}
					}
					if !found {
						childMap[runningRuns[i].RunID] = append(childMap[runningRuns[i].RunID], runToSummary(ch))
					}
				}
			}
		}
	}
	for i := range runningRuns {
		runningRuns[i].ChildRuns = childMap[runningRuns[i].RunID]
	}
	topRecs := make([]state.RunRecord, 0, len(runningRecs))
	for _, rec := range runningRecs {
		if rec.ParentRunID == "" {
			topRecs = append(topRecs, rec)
		}
	}
	s.enrichRunSummaries(runningRuns, topRecs)
	return runningRuns
}
