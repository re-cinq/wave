//go:build webui

package webui

import (
	"html"
	"net/http"
	"os"
	"sort"
	"time"
)

// handleAPIStatistics handles GET /api/statistics?range={24h|7d|30d|all}
func (s *Server) handleAPIStatistics(w http.ResponseWriter, r *http.Request) {
	rangeParam := r.URL.Query().Get("range")
	timeRange, since := parseTimeRange(rangeParam)

	// Get aggregate statistics
	stats, err := s.store.GetRunStatistics(since)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to get statistics")
		return
	}

	aggregate := RunStatistics{
		Total:     stats.Total,
		Succeeded: stats.Succeeded,
		Failed:    stats.Failed,
		Cancelled: stats.Cancelled,
		Pending:   stats.Pending,
		Running:   stats.Running,
	}
	if aggregate.Total > 0 {
		aggregate.SuccessRate = float64(aggregate.Succeeded) / float64(aggregate.Total) * 100
	}

	// Get trends
	trendRecords, _ := s.store.GetRunTrends(since)
	trends := make([]RunTrendPoint, len(trendRecords))
	for i, tr := range trendRecords {
		trends[i] = RunTrendPoint{
			Date:      tr.Date,
			Total:     tr.Total,
			Succeeded: tr.Succeeded,
			Failed:    tr.Failed,
		}
		if tr.Total > 0 {
			trends[i].SuccessRate = float64(tr.Succeeded) / float64(tr.Total) * 100
		}
	}

	// Get per-pipeline statistics
	pipelineRecords, _ := s.store.GetPipelineStatistics(since)
	pipelines := make([]PipelineStatistics, len(pipelineRecords))
	for i, pr := range pipelineRecords {
		pipelines[i] = PipelineStatistics{
			PipelineName:  pr.PipelineName,
			RunCount:      pr.RunCount,
			AvgDurationMs: pr.AvgDurationMs,
			AvgTokens:     pr.AvgTokens,
		}
		if pr.RunCount > 0 {
			pipelines[i].SuccessRate = float64(pr.Succeeded) / float64(pr.RunCount) * 100
		}
	}

	writeJSON(w, http.StatusOK, StatisticsResponse{
		Aggregate: aggregate,
		Trends:    trends,
		Pipelines: pipelines,
		TimeRange: timeRange,
	})
}

// handleStatisticsPage handles GET /statistics - serves the HTML statistics page.
func (s *Server) handleStatisticsPage(w http.ResponseWriter, r *http.Request) {
	rangeParam := r.URL.Query().Get("range")
	timeRange, _ := parseTimeRange(rangeParam)

	data := struct {
		TimeRange string
	}{
		TimeRange: timeRange,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates["templates/statistics.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// handleAPIPipelineDetail handles GET /api/pipelines/{name} - returns full pipeline config.
func (s *Server) handleAPIPipelineDetail(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "missing pipeline name")
		return
	}

	p, err := loadPipelineYAML(name)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "pipeline not found")
		return
	}

	// Build step details
	steps := make([]PipelineStepDetail, len(p.Steps))
	for i, step := range p.Steps {
		sd := PipelineStepDetail{
			ID:           step.ID,
			Persona:      step.Persona,
			Dependencies: step.Dependencies,
			Workspace: WorkspaceDetail{
				Type: step.Workspace.Type,
				Root: step.Workspace.Root,
			},
			Memory: MemoryDetail{
				Strategy: step.Memory.Strategy,
			},
		}

		// Add mounts
		for _, m := range step.Workspace.Mount {
			sd.Workspace.Mounts = append(sd.Workspace.Mounts, MountDetail{
				Source: m.Source,
				Target: m.Target,
				Mode:   m.Mode,
			})
		}

		// Add contract from handover config
		if step.Handover.Contract.Type != "" {
			sd.Contract = &ContractDetail{
				Type:       step.Handover.Contract.Type,
				Schema:     step.Handover.Contract.Schema,
				MustPass:   true,
				MaxRetries: step.Handover.MaxRetries,
			}
		}

		// Add output artifacts
		for _, a := range step.OutputArtifacts {
			sd.Artifacts = append(sd.Artifacts, ArtifactDefDetail{
				Name:     a.Name,
				Path:     a.Path,
				Type:     a.Type,
				Required: a.Required,
			})
		}

		// Add injected artifacts
		for _, ref := range step.Memory.InjectArtifacts {
			sd.Memory.Injected = append(sd.Memory.Injected, InjectedArtifact{
				FromStep: ref.Step,
				Artifact: ref.Artifact,
				As:       ref.As,
			})
		}

		steps[i] = sd
	}

	// Get last run for this pipeline
	var lastRun *RunSummary
	if lr, err := s.store.GetLastRunForPipeline(name); err == nil && lr != nil {
		summary := runToSummary(*lr)
		lastRun = &summary
	}

	resp := PipelineDetailResponse{
		Name:        p.Metadata.Name,
		Description: p.Metadata.Description,
		StepCount:   len(p.Steps),
		Input: PipelineInputDetail{
			Source:  p.Input.Source,
			Example: p.Input.Example,
		},
		Steps:   steps,
		LastRun: lastRun,
	}

	if p.Input.Schema != nil {
		resp.Input.Schema = &InputSchemaDetail{
			Type:        p.Input.Schema.Type,
			Description: p.Input.Schema.Description,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// handlePipelineDetailPage handles GET /pipelines/{name} - serves the HTML pipeline detail page.
func (s *Server) handlePipelineDetailPage(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "missing pipeline name", http.StatusBadRequest)
		return
	}

	p, err := loadPipelineYAML(name)
	if err != nil {
		http.Error(w, "pipeline not found", http.StatusNotFound)
		return
	}

	data := struct {
		Name        string
		Description string
		StepCount   int
	}{
		Name:        p.Metadata.Name,
		Description: p.Metadata.Description,
		StepCount:   len(p.Steps),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates["templates/pipeline_detail.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// handleAPIPersonaDetail handles GET /api/personas/{name} - returns full persona config.
func (s *Server) handleAPIPersonaDetail(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "missing persona name")
		return
	}

	if s.manifest == nil || s.manifest.Personas == nil {
		writeJSONError(w, http.StatusNotFound, "persona not found")
		return
	}

	persona, ok := s.manifest.Personas[name]
	if !ok {
		writeJSONError(w, http.StatusNotFound, "persona not found")
		return
	}

	resp := PersonaDetailResponse{
		Name:             name,
		Description:      persona.Description,
		Adapter:          persona.Adapter,
		Model:            persona.Model,
		Temperature:      persona.Temperature,
		SystemPromptFile: persona.SystemPromptFile,
		AllowedTools:     persona.Permissions.AllowedTools,
		DeniedTools:      persona.Permissions.Deny,
	}

	// Load system prompt content if file exists
	if persona.SystemPromptFile != "" {
		if data, err := os.ReadFile(persona.SystemPromptFile); err == nil {
			resp.SystemPrompt = html.EscapeString(string(data))
		}
	}

	// Add hooks
	if len(persona.Hooks.PreToolUse) > 0 || len(persona.Hooks.PostToolUse) > 0 {
		resp.Hooks = &HooksDetail{}
		for _, h := range persona.Hooks.PreToolUse {
			resp.Hooks.PreToolUse = append(resp.Hooks.PreToolUse, HookRuleDetail{
				Matcher: h.Matcher,
				Command: h.Command,
			})
		}
		for _, h := range persona.Hooks.PostToolUse {
			resp.Hooks.PostToolUse = append(resp.Hooks.PostToolUse, HookRuleDetail{
				Matcher: h.Matcher,
				Command: h.Command,
			})
		}
	}

	// Add sandbox
	if persona.Sandbox != nil {
		resp.Sandbox = &SandboxDetail{
			AllowedDomains: persona.Sandbox.AllowedDomains,
		}
	}

	// Find which pipelines use this persona
	pipelineNames := listPipelineNames()
	var usedIn []string
	for _, pName := range pipelineNames {
		if p, err := loadPipelineYAML(pName); err == nil {
			for _, step := range p.Steps {
				if step.Persona == name {
					usedIn = append(usedIn, pName)
					break
				}
			}
		}
	}
	sort.Strings(usedIn)
	resp.UsedInPipelines = usedIn

	writeJSON(w, http.StatusOK, resp)
}

// handlePersonaDetailPage handles GET /personas/{name} - serves the HTML persona detail page.
func (s *Server) handlePersonaDetailPage(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "missing persona name", http.StatusBadRequest)
		return
	}

	if s.manifest == nil || s.manifest.Personas == nil {
		http.Error(w, "persona not found", http.StatusNotFound)
		return
	}

	persona, ok := s.manifest.Personas[name]
	if !ok {
		http.Error(w, "persona not found", http.StatusNotFound)
		return
	}

	data := struct {
		Name        string
		Description string
	}{
		Name:        name,
		Description: persona.Description,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates["templates/persona_detail.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// parseTimeRange converts the range query parameter to a time range label and since time.
func parseTimeRange(rangeParam string) (string, time.Time) {
	switch rangeParam {
	case "24h":
		return "24h", time.Now().Add(-24 * time.Hour)
	case "7d", "":
		return "7d", time.Now().Add(-7 * 24 * time.Hour)
	case "30d":
		return "30d", time.Now().Add(-30 * 24 * time.Hour)
	case "all":
		return "all", time.Time{} // zero time = since epoch
	default:
		return "7d", time.Now().Add(-7 * 24 * time.Hour)
	}
}
