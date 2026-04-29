package webui

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/humanize"
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

// runToSummary projects a state.RunRecord onto the RunSummary DTO consumed by
// templates and JSON responses.
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
		summary.Duration = humanize.Duration(dur)
	} else if r.Status == "running" {
		dur := time.Since(r.StartedAt)
		summary.Duration = humanize.Duration(dur)
	}

	summary.BranchName = r.BranchName
	summary.ParentRunID = r.ParentRunID
	summary.ParentStepID = r.ParentStepID
	summary.IterateIndex = r.IterateIndex
	summary.IterateTotal = r.IterateTotal
	summary.IterateMode = r.IterateMode
	summary.RunKind = r.RunKind
	summary.SubPipelineRef = r.SubPipelineRef

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
			// Clear any error message left over from a prior failed
			// attempt — a later "completed" event for the same step
			// means the step succeeded on retry / resume and the stale
			// error must not bleed into the final UI row (#1450 follow-up).
			si.errMsg = ""
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
		// Composition primitives (#1450): treat iterate/aggregate/branch/loop
		// as "pipeline" so the inline child-runs block under each step renders
		// for them too. Without this, only bare sub_pipeline steps got the
		// inline list; iterate steps that fan out children showed nothing.
		if stepType == "" {
			switch {
			case step.Iterate != nil, step.Aggregate != nil, step.Branch != nil, step.Loop != nil:
				stepType = "pipeline"
			}
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
			// Only use schema_path (filename) as display name, not inline schema JSON
			if contracts[0].SchemaPath != "" {
				contractSchemaName = contracts[0].SchemaPath
			}
		}

		sd := StepDetail{
			RunID:              runID,
			StepID:             step.ID,
			Persona:            resolveForgeVars(step.Persona),
			State:              "pending",
			StepType:           stepType,
			Script:             strings.TrimSpace(resolveForgeVars(step.Script)),
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

		// Populate injected input artifacts for IN display (clickable chips)
		if len(step.Memory.InjectArtifacts) > 0 {
			injectRefs := make([]InputArtifactRef, 0, len(step.Memory.InjectArtifacts))
			pairs := make([]string, 0, len(step.Memory.InjectArtifacts))
			for _, ia := range step.Memory.InjectArtifacts {
				injectRefs = append(injectRefs, InputArtifactRef{Step: ia.Step, Name: ia.Artifact})
				pairs = append(pairs, ia.Step+"/"+ia.Artifact)
			}
			sd.InputArtifacts = injectRefs
			sd.Action = strings.Join(pairs, ", ")
		} else if len(step.Dependencies) > 0 {
			sd.Action = strings.Join(step.Dependencies, " + ")
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
					sd.Duration = humanize.Duration(si.completedAt.Sub(*si.startedAt))
				} else if sd.State == "running" {
					sd.Duration = humanize.Duration(time.Since(*si.startedAt))
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

// computeGanttPositions populates GanttLeft/GanttWidth on each step relative
// to the run's wall-clock span so the run detail page can render the bars
// directly without further math.
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

// eventToSummary projects a state.LogRecord onto the EventSummary DTO used by
// the run detail page and JSON responses.
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

// extractStepID parses the step ID from a CurrentStep error string.
// Format: `step "analyze-coverage" failed: ...` → `analyze-coverage`
func extractStepID(currentStep string) string {
	if strings.HasPrefix(currentStep, "step \"") {
		if end := strings.Index(currentStep[6:], "\""); end > 0 {
			return currentStep[6 : 6+end]
		}
	}
	return currentStep
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

// artifactToSummary projects an ArtifactRecord onto the ArtifactSummary DTO.
func artifactToSummary(a state.ArtifactRecord) ArtifactSummary {
	return ArtifactSummary{
		ID:        a.ID,
		Name:      a.Name,
		Path:      a.Path,
		Type:      a.Type,
		SizeBytes: a.SizeBytes,
	}
}

// collectStepField extracts a string field from each step, dropping empties.
func collectStepField(steps []StepDetail, fn func(StepDetail) string) []string {
	var result []string
	for _, s := range steps {
		if v := fn(s); v != "" {
			result = append(result, v)
		}
	}
	return result
}

// uniqueStrings returns ss with duplicates removed, preserving first-seen order.
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
