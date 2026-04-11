package webui

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/state"
)

// githubURLPattern matches GitHub issue and PR URLs.
var githubURLPattern = regexp.MustCompile(`https://github\.com/[\w.\-]+/[\w.\-]+/(?:issues|pull)/\d+`)

// formatSmartTime returns a compact human-readable timestamp.
// Same day: "15:04", same year: "Jan 2 15:04", older: "Jan 2, 2006".
func formatSmartTime(t time.Time) string {
	now := time.Now()
	y1, m1, d1 := now.Date()
	y2, m2, d2 := t.Date()
	if y1 == y2 && m1 == m2 && d1 == d2 {
		return t.Format("15:04")
	}
	if y1 == y2 {
		return t.Format("Jan 2 15:04")
	}
	return t.Format("Jan 2, 2006")
}

// parseLinkedURL extracts the first GitHub issue or PR URL from the input string.
func parseLinkedURL(input string) string {
	return githubURLPattern.FindString(input)
}

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
	stepDetails := s.buildStepDetails(runID, run.PipelineName, run.Status)

	// Get events
	events, err := s.store.GetEvents(runID, state.EventQueryOptions{Limit: 5000})
	if err != nil {
		log.Printf("[webui] failed to get events for run %s: %v", runID, err)
	}
	eventSummaries := make([]EventSummary, len(events))
	for i, e := range events {
		eventSummaries[i] = eventToSummary(e)
	}

	// Get all artifacts
	allArts, err := s.store.GetArtifacts(runID, "")
	if err != nil {
		log.Printf("[webui] failed to get artifacts for run %s: %v", runID, err)
	}
	artSummaries := deduplicateArtifacts(allArts)

	resp := RunDetailResponse{
		Run:       runToSummary(*run),
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

	runs, err := s.store.ListRuns(opts)
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

	// Query running runs for the running-pipelines section (FR-008: filter applied)
	runningRecs, err := s.store.ListRuns(state.ListRunsOptions{
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
	if s.store != nil {
		for i := range runningRuns {
			if children, err := s.store.GetChildRuns(runningRuns[i].RunID); err == nil {
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
	s.enrichRunSummaries(runningRuns, func() []state.RunRecord {
		var top []state.RunRecord
		for _, rec := range runningRecs {
			if rec.ParentRunID == "" {
				top = append(top, rec)
			}
		}
		return top
	}())

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
	if err := s.templates["templates/runs.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// handleRunDetailPage renders the Fat Gantt Shapes prototype at /runs2/{id}.
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

	stepDetails := s.buildStepDetails(runID, run.PipelineName, run.Status)

	// Enrich step I/O descriptions from pipeline definition
	if p, loadErr := loadPipelineYAML(run.PipelineName); loadErr == nil {
		type stepRef struct {
			deps    []string
			injects []string
		}
		stepRefs := make(map[string]stepRef)
		for _, ps := range p.Steps {
			var injects []string
			for _, ia := range ps.Memory.InjectArtifacts {
				injects = append(injects, ia.Step+"/"+ia.Artifact)
			}
			stepRefs[ps.ID] = stepRef{deps: ps.Dependencies, injects: injects}
		}
		for i, sd := range stepDetails {
			if ref, ok := stepRefs[sd.StepID]; ok {
				if len(ref.injects) > 0 {
					// Show artifact names: "spec/analysis, docs/feature-docs"
					stepDetails[i].Action = strings.Join(ref.injects, ", ")
				} else if len(ref.deps) > 0 {
					stepDetails[i].Action = strings.Join(ref.deps, " + ")
				}
			}
		}
	}

	stepStatusMap := make(map[string]string)
	stepDetailMap := make(map[string]StepDetail)
	for _, sd := range stepDetails {
		stepStatusMap[sd.StepID] = sd.State
		stepDetailMap[sd.StepID] = sd
	}

	events, err := s.store.GetEvents(runID, state.EventQueryOptions{Limit: 5000})
	if err != nil {
		log.Printf("[webui] failed to get events for run %s: %v", runID, err)
	}
	eventSummaries := make([]EventSummary, len(events))
	for i, e := range events {
		eventSummaries[i] = eventToSummary(e)
	}

	// Build run summary with step progress
	runSummary := runToSummary(*run)
	runSummary.StepsTotal = len(stepDetails)
	completed := 0
	for i, sd := range stepDetails {
		if sd.State == "completed" {
			completed++
		}
		// Mark pending steps as "skipped" if the pipeline is terminal
		if sd.State == "pending" && (run.Status == "completed" || run.Status == "failed" || run.Status == "cancelled") {
			stepDetails[i].State = "skipped"
		}
	}
	runSummary.StepsCompleted = completed
	if runSummary.StepsTotal > 0 {
		runSummary.Progress = (completed * 100) / runSummary.StepsTotal
	}
	if runSummary.TotalTokens == 0 {
		for _, sd := range stepDetails {
			runSummary.TotalTokens += sd.TokensUsed
		}
	}

	var pipelineDescription string
	if p, loadErr := loadPipelineYAML(run.PipelineName); loadErr == nil {
		pipelineDescription = p.Metadata.Description
	}

	// Build artifact groups
	var artifactGroups []StepArtifactGroup
	for _, sd := range stepDetails {
		if len(sd.Artifacts) > 0 {
			artifactGroups = append(artifactGroups, StepArtifactGroup{
				StepID:    sd.StepID,
				Artifacts: sd.Artifacts,
			})
		}
	}

	// Compute the last step's output artifacts for the OUTPUT card
	var outputSummary string
	var outputArtifacts []ArtifactSummary
	var outputStepID string
	if len(stepDetails) > 0 {
		last := stepDetails[len(stepDetails)-1]
		for i := len(stepDetails) - 1; i >= 0; i-- {
			if stepDetails[i].State == "completed" {
				last = stepDetails[i]
				break
			}
		}
		if len(last.Artifacts) > 0 {
			outputArtifacts = last.Artifacts
			outputStepID = last.StepID
			var names []string
			for _, a := range last.Artifacts {
				names = append(names, a.Name)
			}
			outputSummary = strings.Join(names, ", ")
		}
	}

	// Enrich linked URL with PR/Issue metadata from forge
	var linkedTitle, linkedState, linkedAuthor, linkedType string
	var linkedNumber int
	if runSummary.LinkedURL != "" && s.forgeClient != nil && s.repoSlug != "" {
		parts := strings.Split(s.repoSlug, "/")
		if len(parts) == 2 {
			owner, repo := parts[0], parts[1]
			ctx := r.Context()
			// Parse PR or issue number from URL
			urlParts := strings.Split(runSummary.LinkedURL, "/")
			for i, p := range urlParts {
				if (p == "pull" || p == "issues" || p == "merge_requests") && i+1 < len(urlParts) {
					if num, err := strconv.Atoi(strings.TrimRight(urlParts[i+1], "#/")); err == nil {
						linkedNumber = num
						switch p {
						case "pull", "merge_requests":
							linkedType = "pr"
							if pr, err := s.forgeClient.GetPullRequest(ctx, owner, repo, num); err == nil {
								linkedTitle = pr.Title
								linkedState = pr.State
								if pr.Merged {
									linkedState = "merged"
								}
								linkedAuthor = pr.Author
							}
						case "issues":
							linkedType = "issue"
							if iss, err := s.forgeClient.GetIssue(ctx, owner, repo, num); err == nil {
								linkedTitle = iss.Title
								linkedState = iss.State
								linkedAuthor = iss.Author
							}
						}
					}
				}
			}
		}
	}

	// Build template variable map for prompt resolution
	templateVars := map[string]string{
		"input": run.Input,
	}
	// Add forge variables
	forgeInfo, _ := forge.DetectFromGitRemotes()
	templateVars["forge.cli_tool"] = forgeInfo.CLITool
	templateVars["forge.type"] = string(forgeInfo.Type)
	templateVars["forge.pr_term"] = forgeInfo.PRTerm
	templateVars["forge.pr_command"] = forgeInfo.PRCommand
	// Add project variables from manifest
	if s.manifest != nil && s.manifest.Project != nil {
		for k, v := range s.manifest.Project.ProjectVars() {
			templateVars["project."+k] = v
		}
	}

	// Collect child runs for sub-pipeline steps
	childRuns := make(map[string][]RunSummary)
	if children, err := s.store.GetChildRuns(runID); err == nil {
		for _, cr := range children {
			summary := runToSummary(cr)
			childRuns[cr.ParentStepID] = append(childRuns[cr.ParentStepID], summary)
		}
	}

	data := struct {
		ActivePage          string
		Run                 RunSummary
		Steps               []StepDetail
		Events              []EventSummary
		PipelineDescription string
		ArtifactGroups      []StepArtifactGroup
		Adapters            []string
		Models              []string
		OutputSummary       string
		OutputArtifacts     []ArtifactSummary
		OutputStepID        string
		LinkedTitle         string
		LinkedState         string
		LinkedAuthor        string
		LinkedNumber        int
		LinkedType          string
		ChildRuns           map[string][]RunSummary
		TemplateVars        map[string]string
	}{
		ActivePage:          "runs",
		Run:                 runSummary,
		Steps:               stepDetails,
		Events:              eventSummaries,
		PipelineDescription: pipelineDescription,
		ArtifactGroups:      artifactGroups,
		Adapters:            uniqueStrings(collectStepField(stepDetails, func(sd StepDetail) string { return sd.Adapter })),
		Models:              uniqueStrings(collectStepField(stepDetails, func(sd StepDetail) string { return friendlyModelFunc(sd.Model) })),
		OutputSummary:       outputSummary,
		OutputArtifacts:     outputArtifacts,
		OutputStepID:        outputStepID,
		LinkedTitle:         linkedTitle,
		LinkedState:         linkedState,
		LinkedAuthor:        linkedAuthor,
		LinkedNumber:        linkedNumber,
		LinkedType:          linkedType,
		ChildRuns:           childRuns,
		TemplateVars:        templateVars,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates["templates/run_detail.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// enrichRunSummaries populates step progress for a batch of run summaries.
// Uses event_log (not step_state) because step_state has cross-run collisions.
func (s *Server) enrichRunSummaries(summaries []RunSummary, runs []state.RunRecord) {
	for i := range summaries {
		// Get total from pipeline definition
		if p, loadErr := loadPipelineYAML(runs[i].PipelineName); loadErr == nil {
			summaries[i].StepsTotal = len(p.Steps)
		}

		// Count completed steps from event_log
		events, err := s.store.GetEvents(runs[i].RunID, state.EventQueryOptions{Limit: 5000})
		if err != nil {
			continue
		}
		completedSteps := make(map[string]bool)
		adapterSet := make(map[string]bool)
		modelSet := make(map[string]bool)
		for _, ev := range events {
			if ev.StepID != "" && ev.State == "completed" {
				completedSteps[ev.StepID] = true
			}
			if ev.Adapter != "" {
				adapterSet[ev.Adapter] = true
			}
			if ev.Model != "" {
				modelSet[ev.Model] = true
			}
		}
		summaries[i].StepsCompleted = len(completedSteps)
		if summaries[i].StepsTotal > 0 {
			summaries[i].Progress = (len(completedSteps) * 100) / summaries[i].StepsTotal
		}
		for a := range adapterSet {
			summaries[i].Adapters = append(summaries[i].Adapters, a)
		}
		for m := range modelSet {
			summaries[i].Models = append(summaries[i].Models, friendlyModelFunc(m))
		}
		summaries[i].Models = uniqueStrings(summaries[i].Models)
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

	summary.BranchName = r.BranchName
	summary.ParentRunID = r.ParentRunID
	summary.ParentStepID = r.ParentStepID

	// Full input and truncated preview
	if r.Input != "" {
		summary.Input = r.Input
		preview := r.Input
		if len(preview) > 80 {
			preview = preview[:80] + "..."
		}
		summary.InputPreview = preview
		summary.LinkedURL = parseLinkedURL(r.Input)
	}

	// Human-readable timestamps — compact for list views.
	// Same day: "15:04", same year: "Jan 2 15:04", older: "Jan 2, 2006".
	summary.FormattedStartedAt = formatSmartTime(r.StartedAt)
	if r.CompletedAt != nil {
		summary.FormattedCompletedAt = formatSmartTime(*r.CompletedAt)
	}

	// Compute step progress from pipeline definition
	if p, err := loadPipelineYAML(r.PipelineName); err == nil {
		summary.StepsTotal = len(p.Steps)
	}

	return summary
}

// buildStepDetails derives step details from the event_log table combined with
// the pipeline definition. We use events rather than step_state because the
// step_state table has a unique constraint on step_id alone (not per-pipeline),
// causing cross-run collisions.
func (s *Server) buildStepDetails(runID, pipelineName string, runStatus ...string) []StepDetail {
	// Load pipeline definition to get ordered step list with personas
	p, err := loadPipelineYAML(pipelineName)
	if err != nil {
		log.Printf("[webui] buildStepDetails: failed to load pipeline %q: %v", pipelineName, err)
		return nil
	}

	// Get all events for this run
	events, err := s.store.GetEvents(runID, state.EventQueryOptions{Limit: 5000})
	if err != nil {
		log.Printf("[webui] buildStepDetails: failed to get events for run %s: %v", runID, err)
	}
	log.Printf("[webui] buildStepDetails: runID=%s pipeline=%s steps=%d events=%d", runID, pipelineName, len(p.Steps), len(events))

	// Build step state from events: track latest state, timestamps, tokens per step
	type stepInfo struct {
		state           string
		persona         string
		startedAt       *time.Time
		completedAt     *time.Time
		tokens          int
		durationMs      int64
		errMsg          string
		model           string
		configuredModel string
		adapter         string
		reviewVerdict   string
		reviewIssues    int
		reviewPersona   string
		reviewTokens    int
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
			si.persona = resolveForgeVars(ev.Persona)
		}

		// Track state transitions — terminal states (completed/failed) are final
		switch ev.State {
		case "running":
			if si.startedAt == nil {
				t := ev.Timestamp
				si.startedAt = &t
			}
			if si.state != "completed" && si.state != "failed" {
				si.state = "running"
			}
		case "completed":
			t := ev.Timestamp
			si.completedAt = &t
			si.state = "completed"
		case "failed":
			t := ev.Timestamp
			si.completedAt = &t
			si.state = "failed"
			si.errMsg = ev.Message
		case "review_completed":
			// Parse review verdict from message: "verdict=pass issues=0 reviewer=navigator"
			si.reviewVerdict, si.reviewIssues, si.reviewPersona = parseReviewCompletedMessage(ev.Message)
			si.reviewTokens += ev.TokensUsed
		case "review_failed":
			si.reviewVerdict = "fail"
			si.reviewTokens += ev.TokensUsed
		}

		if ev.TokensUsed > si.tokens {
			si.tokens = ev.TokensUsed
		}
		if ev.DurationMs > si.durationMs {
			si.durationMs = ev.DurationMs
		}
		if ev.Model != "" {
			si.model = ev.Model
		}
		if ev.ConfiguredModel != "" {
			si.configuredModel = ev.ConfiguredModel
		}
		if ev.Adapter != "" {
			si.adapter = ev.Adapter
		}
	}

	// Build details in pipeline step order
	details := make([]StepDetail, 0, len(p.Steps))
	for _, step := range p.Steps {
		// Determine effective step type
		stepType := step.Type
		if stepType == "" && step.Gate != nil {
			stepType = "gate"
		}
		if stepType == "" && step.SubPipeline != "" {
			stepType = "pipeline"
		}

		// Collect gate info
		var gatePrompt, gateChoices string
		if step.Gate != nil {
			gatePrompt = step.Gate.Prompt
			if gatePrompt == "" {
				gatePrompt = step.Gate.Message
			}
			var choiceLabels []string
			for _, c := range step.Gate.Choices {
				choiceLabels = append(choiceLabels, c.Label)
			}
			gateChoices = strings.Join(choiceLabels, ", ")
		}

		// Collect edge info
		var edgeInfo string
		if len(step.Edges) > 0 {
			var edgeParts []string
			for _, e := range step.Edges {
				if e.Condition != "" {
					edgeParts = append(edgeParts, e.Target+": "+e.Condition)
				} else {
					edgeParts = append(edgeParts, e.Target)
				}
			}
			edgeInfo = strings.Join(edgeParts, "; ")
		}

		// Extract contract info for display
		var contractType, contractSchemaName string
		if contracts := step.Handover.EffectiveContracts(); len(contracts) > 0 {
			contractType = contracts[0].Type
			if contracts[0].Schema != "" {
				contractSchemaName = contracts[0].Schema
			} else if contracts[0].SchemaPath != "" {
				contractSchemaName = contracts[0].SchemaPath
			}
		}

		sd := StepDetail{
			RunID:              runID,
			StepID:             step.ID,
			Persona:            resolveForgeVars(step.Persona),
			State:              "pending",
			StepType:           stepType,
			Script:             step.Script,
			SubPipeline:        stripUnresolvedVars(resolveForgeVars(step.SubPipeline)),
			GatePrompt:         gatePrompt,
			GateChoices:        gateChoices,
			EdgeInfo:           edgeInfo,
			Model:              step.Model,
			MaxVisits:          step.MaxVisits,
			Contract:           contractType,
			ContractSchemaName: contractSchemaName,
			Dependencies:       step.Dependencies,
		}

		// Populate output artifact names for OUT display
		if len(step.OutputArtifacts) > 0 {
			var names []string
			for _, a := range step.OutputArtifacts {
				names = append(names, a.Name)
			}
			sd.Output = strings.Join(names, ", ")
		}

		// Populate structured gate data for interactive UI
		if step.Gate != nil {
			sd.GateChoicesData = step.Gate.Choices
			sd.GateFreeform = step.Gate.Freeform
		}

		if si, ok := stepMap[step.ID]; ok {
			if si.state != "" {
				sd.State = si.state
			}
			if si.persona != "" {
				sd.Persona = si.persona
			}
			if si.model != "" {
				sd.Model = si.model
			}
			if si.configuredModel != "" {
				sd.ConfiguredModel = si.configuredModel
			}
			if si.adapter != "" {
				sd.Adapter = si.adapter
			}
			sd.StartedAt = si.startedAt
			if si.startedAt != nil {
				sd.FormattedStartedAt = si.startedAt.Format("15:04:05")
			}
			sd.CompletedAt = si.completedAt
			sd.TokensUsed = si.tokens
			sd.Error = si.errMsg

			// Populate agent review verdict fields if a review ran
			if si.reviewVerdict != "" {
				sd.ReviewVerdict = si.reviewVerdict
				sd.ReviewIssueCount = si.reviewIssues
				sd.ReviewerPersona = si.reviewPersona
				sd.ReviewTokens = si.reviewTokens
			}

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

			// Populate failure class from step attempts
			if sd.State == "failed" && s.store != nil {
				if attempts, err := s.store.GetStepAttempts(runID, step.ID); err == nil && len(attempts) > 0 {
					sd.FailureClass = attempts[len(attempts)-1].FailureClass
				}
			}

			// Populate visit count for graph loop steps
			if step.MaxVisits > 0 && s.store != nil {
				if vc, err := s.store.GetStepVisitCount(runID, step.ID); err == nil {
					sd.VisitCount = vc
				}
			}
		}

		// Look up failure class from step attempts for failed steps
		if sd.State == "failed" {
			attempts, attErr := s.store.GetStepAttempts(runID, step.ID)
			if attErr != nil {
				log.Printf("[webui] failed to get step attempts for run %s step %s: %v", runID, step.ID, attErr)
			}
			if len(attempts) > 0 {
				lastAttempt := attempts[len(attempts)-1]
				if lastAttempt.FailureClass != "" {
					sd.FailureClass = lastAttempt.FailureClass
				}
			}
		}

		arts, artErr := s.store.GetArtifacts(runID, step.ID)
		if artErr != nil {
			log.Printf("[webui] failed to get artifacts for run %s step %s: %v", runID, step.ID, artErr)
		}
		sd.Artifacts = deduplicateArtifacts(arts)

		// If the run is terminal but step still shows running, override to cancelled
		rs := ""
		if len(runStatus) > 0 {
			rs = runStatus[0]
		}
		if (rs == "cancelled" || rs == "failed") && (sd.State == "running" || sd.State == "started") {
			sd.State = "cancelled"
		}

		details = append(details, sd)
	}

	// Compute Gantt positions
	computeGanttPositions(details)

	return details
}

func computeGanttPositions(steps []StepDetail) {
	if len(steps) == 0 {
		return
	}
	var earliest, latest time.Time
	for _, s := range steps {
		if s.StartedAt != nil {
			if earliest.IsZero() || s.StartedAt.Before(earliest) {
				earliest = *s.StartedAt
			}
		}
		if s.CompletedAt != nil {
			if latest.IsZero() || s.CompletedAt.After(latest) {
				latest = *s.CompletedAt
			}
		}
	}
	totalDuration := latest.Sub(earliest)
	if totalDuration <= 0 {
		return
	}
	now := time.Now()
	for i := range steps {
		if steps[i].StartedAt != nil {
			left := float64(steps[i].StartedAt.Sub(earliest)) / float64(totalDuration) * 100
			var width float64
			if steps[i].CompletedAt != nil {
				width = float64(steps[i].CompletedAt.Sub(*steps[i].StartedAt)) / float64(totalDuration) * 100
			} else if steps[i].State == "running" {
				// Running step: extend to current time
				width = float64(now.Sub(*steps[i].StartedAt)) / float64(totalDuration) * 100
				if width > 100-left {
					width = 100 - left
				}
			}
			if width < 1 && (steps[i].State == "completed" || steps[i].State == "running") {
				width = 1
			}
			steps[i].GanttLeft = left
			steps[i].GanttWidth = width
		}
	}
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
		Model:      e.Model,
		Adapter:    e.Adapter,
	}
}

// deduplicateArtifacts keeps only the last artifact per name.
// Multiple writes to the same artifact name (e.g. retries) create
// duplicate rows — we show only the most recent one.
func deduplicateArtifacts(arts []state.ArtifactRecord) []ArtifactSummary {
	seen := make(map[string]int) // name → index in result
	var result []ArtifactSummary
	for _, a := range arts {
		s := artifactToSummary(a)
		if idx, ok := seen[s.Name]; ok {
			result[idx] = s // overwrite with later entry
		} else {
			seen[s.Name] = len(result)
			result = append(result, s)
		}
	}
	return result
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
	if d < time.Hour {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		if s == 0 {
			return fmt.Sprintf("%dm", m)
		}
		return fmt.Sprintf("%dm %ds", m, s)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if m == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}

func collectStepField(steps []StepDetail, fn func(StepDetail) string) []string {
	var result []string
	for _, s := range steps {
		if v := fn(s); v != "" {
			result = append(result, v)
		}
	}
	return result
}

func uniqueStrings(ss []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
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

// resolveForgeVars replaces {{ forge.* }} template variables in a string.
// Cached after first detection.
var (
	forgeOnce sync.Once
	forgeInfo forge.ForgeInfo
)

func resolveForgeVars(s string) string {
	if !strings.Contains(s, "{{ forge.") {
		return s
	}
	forgeOnce.Do(func() {
		forgeInfo, _ = forge.DetectFromGitRemotes()
	})
	r := strings.NewReplacer(
		"{{ forge.type }}", string(forgeInfo.Type),
		"{{ forge.cli_tool }}", forgeInfo.CLITool,
		"{{ forge.pr_term }}", forgeInfo.PRTerm,
		"{{ forge.pr_command }}", forgeInfo.PRCommand,
		"{{ forge.host }}", forgeInfo.Host,
		"{{ forge.owner }}", forgeInfo.Owner,
		"{{ forge.repo }}", forgeInfo.Repo,
		"{{ forge.prefix }}", forgeInfo.PipelinePrefix,
	)
	return r.Replace(s)
}

// stripUnresolvedVars removes remaining {{ var }} placeholders that weren't
// resolved by forge or runtime template expansion (e.g. {{ item }}, {{ input }}).
// Returns empty string if the entire value was a single placeholder.
func stripUnresolvedVars(s string) string {
	if !strings.Contains(s, "{{") {
		return s
	}
	// If the whole string is just a template var, return empty
	trimmed := strings.TrimSpace(s)
	if strings.HasPrefix(trimmed, "{{") && strings.HasSuffix(trimmed, "}}") && strings.Count(trimmed, "{{") == 1 {
		return ""
	}
	return s
}

// JSON response helpers

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// handleExportRuns handles GET /api/runs/export - exports runs as CSV or JSON.
func (s *Server) handleExportRuns(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format != "csv" && format != "json" {
		writeJSONError(w, http.StatusBadRequest, "format must be 'csv' or 'json'")
		return
	}

	// Respect any active filters
	status := r.URL.Query().Get("status")
	pipeline := r.URL.Query().Get("pipeline")
	opts := state.ListRunsOptions{
		Status:       status,
		PipelineName: pipeline,
		Limit:        10000, // reasonable upper bound for export
	}

	runs, err := s.store.ListRuns(opts)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list runs")
		return
	}

	switch format {
	case "csv":
		s.exportRunsCSV(w, runs)
	case "json":
		s.exportRunsJSON(w, runs)
	}
}

// exportRunsCSV writes runs as a CSV download.
func (s *Server) exportRunsCSV(w http.ResponseWriter, runs []state.RunRecord) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=wave-runs.csv")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	_ = writer.Write([]string{"run_id", "pipeline", "status", "started_at", "duration_seconds", "tokens", "branch"})

	for _, run := range runs {
		var durationSec string
		if run.CompletedAt != nil {
			dur := run.CompletedAt.Sub(run.StartedAt)
			durationSec = strconv.FormatFloat(dur.Seconds(), 'f', 1, 64)
		}
		_ = writer.Write([]string{
			run.RunID,
			run.PipelineName,
			run.Status,
			run.StartedAt.Format(time.RFC3339),
			durationSec,
			strconv.Itoa(run.TotalTokens),
			run.BranchName,
		})
	}
}

// runExportEntry is the JSON structure for a single exported run.
type runExportEntry struct {
	RunID           string   `json:"run_id"`
	Pipeline        string   `json:"pipeline"`
	Status          string   `json:"status"`
	StartedAt       string   `json:"started_at"`
	DurationSeconds *float64 `json:"duration_seconds,omitempty"`
	Tokens          int      `json:"tokens"`
	Branch          string   `json:"branch,omitempty"`
}

// exportRunsJSON writes runs as a JSON array download.
func (s *Server) exportRunsJSON(w http.ResponseWriter, runs []state.RunRecord) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=wave-runs.json")

	entries := make([]runExportEntry, len(runs))
	for i, run := range runs {
		entry := runExportEntry{
			RunID:     run.RunID,
			Pipeline:  run.PipelineName,
			Status:    run.Status,
			StartedAt: run.StartedAt.Format(time.RFC3339),
			Tokens:    run.TotalTokens,
			Branch:    run.BranchName,
		}
		if run.CompletedAt != nil {
			dur := run.CompletedAt.Sub(run.StartedAt).Seconds()
			entry.DurationSeconds = &dur
		}
		entries[i] = entry
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(entries)
}

// nestChildRuns filters child runs from the top-level list and nests them
// under their parent run. Child runs whose parent is not in the list remain
// at top level (they may belong to a parent on another page).
func nestChildRuns(all []RunSummary) []RunSummary {
	type indexedSummary struct {
		idx     int
		summary *RunSummary
	}
	byID := make(map[string]*indexedSummary, len(all))
	for i := range all {
		byID[all[i].RunID] = &indexedSummary{idx: i, summary: &all[i]}
	}

	var topLevel []RunSummary
	for i := range all {
		if all[i].ParentRunID != "" {
			if parent, ok := byID[all[i].ParentRunID]; ok {
				parent.summary.ChildRuns = append(parent.summary.ChildRuns, all[i])
				continue
			}
		}
		topLevel = append(topLevel, all[i])
	}

	// Re-sync ChildRuns for parents that were copied into topLevel
	for i := range topLevel {
		if is, ok := byID[topLevel[i].RunID]; ok {
			topLevel[i].ChildRuns = is.summary.ChildRuns
		}
	}

	return topLevel
}

// parseReviewCompletedMessage extracts verdict, issue count, and reviewer persona
// from a review_completed event message like:
// "agent review completed: verdict=pass issues=0 reviewer=navigator"
func parseReviewCompletedMessage(msg string) (verdict string, issueCount int, reviewer string) {
	for _, part := range strings.Fields(msg) {
		switch {
		case strings.HasPrefix(part, "verdict="):
			verdict = strings.TrimPrefix(part, "verdict=")
		case strings.HasPrefix(part, "issues="):
			_, _ = fmt.Sscanf(strings.TrimPrefix(part, "issues="), "%d", &issueCount)
		case strings.HasPrefix(part, "reviewer="):
			reviewer = strings.TrimPrefix(part, "reviewer=")
		}
	}
	return
}
