package webui

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/state"
)

// handleRunDetailPage renders the Fat Gantt Shapes prototype at /runs2/{id}.
func (s *Server) handleRunDetailPage(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		http.Error(w, "missing run ID", http.StatusBadRequest)
		return
	}

	run, err := s.runtime.store.GetRun(runID)
	if err != nil {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}

	stepDetails := s.buildStepDetails(runID, run.PipelineName, run.Status)

	stepStatusMap := make(map[string]string)
	stepDetailMap := make(map[string]StepDetail)
	for _, sd := range stepDetails {
		stepStatusMap[sd.StepID] = sd.State
		stepDetailMap[sd.StepID] = sd
	}

	events, err := s.runtime.store.GetEvents(runID, state.EventQueryOptions{Limit: 5000})
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

	outputSummary, outputArtifacts, outputStepID := buildOutputCard(stepDetails)
	linkedTitle, linkedState, linkedAuthor, linkedType, linkedNumber := s.enrichLinkedURL(r, runSummary.LinkedURL)
	templateVars := s.buildTemplateVars(run.Input)

	// Collect child runs for sub-pipeline steps and resume / composition
	// children (#1510, #1450 follow-up).
	//
	// Resume children render as a "Resumed by" pill at the header.
	// Composition children (iterate / sub-pipeline / branch / loop /
	// aggregate) render both inline under their parent step (childRuns map)
	// AND in a header-level "Children" section grouped by run_kind so the
	// parent <-> child link is discoverable from either direction.
	childRuns := make(map[string][]RunSummary)
	var resumeChildren []RunSummary
	var compositionChildren []RunSummary
	if children, err := s.runtime.store.GetChildRuns(runID); err == nil {
		for _, cr := range children {
			summary := runToSummary(cr)
			if cr.RunKind == state.RunKindResume {
				resumeChildren = append(resumeChildren, summary)
				continue
			}
			// Inline list under the originating parent step keeps the
			// existing per-step widget working.
			childRuns[cr.ParentStepID] = append(childRuns[cr.ParentStepID], summary)
			// Header-level grouped section also surfaces every composition
			// child regardless of which step launched it.
			if isCompositionRunKind(cr.RunKind) {
				compositionChildren = append(compositionChildren, summary)
			}
		}
	}
	compositionChildGroups := groupChildrenByKind(compositionChildren)

	// Resolve the parent run summary so the breadcrumb can render kind-aware
	// labels ("← iterate parent", "← branch parent", etc.) without re-querying
	// in the template. Soft-fail: a missing parent simply means no breadcrumb.
	var parentRun *RunSummary
	if run.ParentRunID != "" {
		if pr, err := s.runtime.store.GetRun(run.ParentRunID); err == nil && pr != nil {
			ps := runToSummary(*pr)
			parentRun = &ps
		}
	}

	runConfigItems := s.buildRunConfigItems()

	// Extract adapter and model override from the pipeline "started" event.
	// The started event carries the launch-level --adapter and --model flags,
	// not per-step resolved values. For resumed runs, use the last started event.
	var runAdapter, runModelTier string
	for i := len(events) - 1; i >= 0; i-- {
		ev := events[i]
		if ev.State == "started" && ev.StepID == "" {
			runAdapter = ev.Adapter
			runModelTier = ev.ConfiguredModel
			break
		}
	}

	// Reconstruct the wave run command
	rerunCmd := "wave run " + run.PipelineName
	if runAdapter != "" {
		rerunCmd += " --adapter " + runAdapter
	}
	if runModelTier != "" {
		rerunCmd += " --model " + runModelTier
	}
	if run.Input != "" {
		rerunCmd += " -- " + strconv.Quote(run.Input)
	}

	data := struct {
		ActivePage             string
		Run                    RunSummary
		Steps                  []StepDetail
		Events                 []EventSummary
		PipelineDescription    string
		ArtifactGroups         []StepArtifactGroup
		Adapters               []string
		Models                 []string
		OutputSummary          string
		OutputArtifacts        []ArtifactSummary
		OutputStepID           string
		LinkedTitle            string
		LinkedState            string
		LinkedAuthor           string
		LinkedNumber           int
		LinkedType             string
		ChildRuns              map[string][]RunSummary
		ResumeChildren         []RunSummary
		CompositionChildGroups []childRunGroup
		ParentRun              *RunSummary
		TemplateVars           map[string]string
		RunConfigItems         []struct{ Label, Value, Tooltip string }
		RerunCommand           string
		RunAdapter             string
		RunModelTier           string
		FailedStepID           string
	}{
		ActivePage:             "runs",
		Run:                    runSummary,
		Steps:                  stepDetails,
		Events:                 eventSummaries,
		PipelineDescription:    pipelineDescription,
		ArtifactGroups:         artifactGroups,
		Adapters:               uniqueStrings(collectStepField(stepDetails, func(sd StepDetail) string { return sd.Adapter })),
		Models:                 uniqueStrings(collectStepField(stepDetails, func(sd StepDetail) string { return friendlyModelFunc(sd.Model) })),
		OutputSummary:          outputSummary,
		OutputArtifacts:        outputArtifacts,
		OutputStepID:           outputStepID,
		LinkedTitle:            linkedTitle,
		LinkedState:            linkedState,
		LinkedAuthor:           linkedAuthor,
		LinkedNumber:           linkedNumber,
		LinkedType:             linkedType,
		ChildRuns:              childRuns,
		ResumeChildren:         resumeChildren,
		CompositionChildGroups: compositionChildGroups,
		ParentRun:              parentRun,
		TemplateVars:           templateVars,
		RunConfigItems:         runConfigItems,
		RerunCommand:           rerunCmd,
		RunAdapter:             runAdapter,
		RunModelTier:           runModelTier,
		FailedStepID:           extractStepID(run.CurrentStep),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.assets.templates["templates/run_detail.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// buildOutputCard finds the most recent completed step and returns its
// artifacts, the comma-joined name list, and its step ID for use by the
// run-detail OUTPUT card.
func buildOutputCard(stepDetails []StepDetail) (summary string, artifacts []ArtifactSummary, stepID string) {
	if len(stepDetails) == 0 {
		return "", nil, ""
	}
	last := stepDetails[len(stepDetails)-1]
	for i := len(stepDetails) - 1; i >= 0; i-- {
		if stepDetails[i].State == "completed" {
			last = stepDetails[i]
			break
		}
	}
	if len(last.Artifacts) == 0 {
		return "", nil, ""
	}
	names := make([]string, 0, len(last.Artifacts))
	for _, a := range last.Artifacts {
		names = append(names, a.Name)
	}
	return strings.Join(names, ", "), last.Artifacts, last.StepID
}

// enrichLinkedURL fetches PR/issue metadata from the forge for the run's
// linked URL. Returns empty values when no URL is set, the forge client is
// unavailable, or the URL doesn't match a recognized PR/issue path.
func (s *Server) enrichLinkedURL(r *http.Request, linkedURL string) (title, state, author, kind string, number int) {
	if linkedURL == "" || s.runtime.forgeClient == nil || s.runtime.repoSlug == "" {
		return
	}
	parts := strings.Split(s.runtime.repoSlug, "/")
	if len(parts) != 2 {
		return
	}
	owner, repo := parts[0], parts[1]
	ctx := r.Context()
	urlParts := strings.Split(linkedURL, "/")
	for i, p := range urlParts {
		if (p != "pull" && p != "issues" && p != "merge_requests") || i+1 >= len(urlParts) {
			continue
		}
		num, err := strconv.Atoi(strings.TrimRight(urlParts[i+1], "#/"))
		if err != nil {
			continue
		}
		number = num
		switch p {
		case "pull", "merge_requests":
			kind = "pr"
			if pr, err := s.runtime.forgeClient.GetPullRequest(ctx, owner, repo, num); err == nil {
				title = pr.Title
				state = pr.State
				if pr.Merged {
					state = "merged"
				}
				author = pr.Author
			}
		case "issues":
			kind = "issue"
			if iss, err := s.runtime.forgeClient.GetIssue(ctx, owner, repo, num); err == nil {
				title = iss.Title
				state = iss.State
				author = iss.Author
			}
		}
	}
	return
}

// buildTemplateVars assembles the template variable map (input, forge.*,
// project.*) for the run-detail page so the template can resolve prompt
// placeholders the same way the executor does.
func (s *Server) buildTemplateVars(input string) map[string]string {
	templateVars := map[string]string{
		"input": input,
	}
	forgeInfo, _ := forge.DetectFromGitRemotes()
	templateVars["forge.cli_tool"] = forgeInfo.CLITool
	templateVars["forge.type"] = string(forgeInfo.Type)
	templateVars["forge.pr_term"] = forgeInfo.PRTerm
	templateVars["forge.pr_command"] = forgeInfo.PRCommand
	if s.runtime.manifest != nil && s.runtime.manifest.Project != nil {
		for k, v := range s.runtime.manifest.Project.ProjectVars() {
			templateVars["project."+k] = v
		}
	}
	return templateVars
}

// buildRunConfigItems returns the human-readable runtime config rows
// (timeout, stall timeout) for the run-detail page.
func (s *Server) buildRunConfigItems() []struct{ Label, Value, Tooltip string } {
	var items []struct{ Label, Value, Tooltip string }
	if s.runtime.manifest == nil {
		return items
	}
	if timeout := s.runtime.manifest.Runtime.GetDefaultTimeout(); timeout > 0 {
		items = append(items, struct{ Label, Value, Tooltip string }{"Timeout", timeout.String(), "Maximum duration per step before it is cancelled"})
	}
	if s.runtime.manifest.Runtime.StallTimeout != "" {
		items = append(items, struct{ Label, Value, Tooltip string }{"Stall timeout", s.runtime.manifest.Runtime.StallTimeout, "Step is cancelled if no tool activity for this duration"})
	}
	return items
}
