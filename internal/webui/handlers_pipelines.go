package webui

import (
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/state"
)

// PipelineSummary holds summary info about an available pipeline.
type PipelineSummary struct {
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	Category      string   `json:"category,omitempty"`
	StepCount     int      `json:"step_count"`
	Steps         []string `json:"steps,omitempty"`
	IsComposition bool     `json:"is_composition,omitempty"`
	Skills        []string `json:"skills,omitempty"`
	Disabled      bool     `json:"disabled"`
	RunCount      int      `json:"run_count,omitempty"`
}

// handlePipelinesPage handles GET /pipelines - serves the HTML pipelines page.
func (s *Server) handlePipelinesPage(w http.ResponseWriter, r *http.Request) {
	pipelines := s.getPipelineSummaries()

	// Enrich with run counts
	if s.store != nil {
		allRuns, err := s.store.ListRuns(state.ListRunsOptions{Limit: 10000})
		if err == nil {
			counts := make(map[string]int)
			for _, run := range allRuns {
				counts[run.PipelineName]++
			}
			for i := range pipelines {
				pipelines[i].RunCount = counts[pipelines[i].Name]
			}
		}
	}

	// Collect unique categories
	categories := make(map[string]bool)
	for _, p := range pipelines {
		if p.Category != "" {
			categories[p.Category] = true
		}
	}
	var catList []string
	for c := range categories {
		catList = append(catList, c)
	}
	sort.Strings(catList)

	data := struct {
		ActivePage string
		Pipelines  []PipelineSummary
		Categories []string
	}{
		ActivePage: "pipelines",
		Pipelines:  pipelines,
		Categories: catList,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates["templates/pipelines.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// handleAPIPipelines handles GET /api/pipelines - returns pipeline list as JSON.
func (s *Server) handleAPIPipelines(w http.ResponseWriter, r *http.Request) {
	pipelines := s.getPipelineSummaries()
	disabled := s.getDisabledPipelineSet()
	for i := range pipelines {
		if disabled[pipelines[i].Name] {
			pipelines[i].Disabled = true
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"pipelines": pipelines})
}

// handleAPIPipelineInfo handles GET /api/pipelines/info - returns pipeline metadata
// for the enhanced start form (description, step count, category).
func (s *Server) handleAPIPipelineInfo(w http.ResponseWriter, r *http.Request) {
	infos := getPipelineStartInfos()
	writeJSON(w, http.StatusOK, map[string]interface{}{"pipelines": infos})
}

// PipelineDetailStep holds step info for the pipeline detail view.
type PipelineDetailStep struct {
	ID                 string   `json:"id"`
	Type               string   `json:"type,omitempty"`
	Persona            string   `json:"persona"`
	Dependencies       []string `json:"dependencies,omitempty"`
	Timeout            int      `json:"timeout,omitempty"`
	Optional           bool     `json:"optional,omitempty"`
	Artifacts          []string `json:"artifacts,omitempty"`
	InputArtifacts     []string `json:"input_artifacts,omitempty"`
	Contract           string   `json:"contract,omitempty"`
	ContractSchemaName string   `json:"contract_schema_name,omitempty"`
	OnFailure          string   `json:"on_failure,omitempty"`
	RetryPolicy        string   `json:"retry_policy,omitempty"`
	MaxAttempts        int      `json:"max_attempts,omitempty"`
	Prompt             string   `json:"prompt,omitempty"`
	SubPipeline        string   `json:"sub_pipeline,omitempty"`
	Thread             string   `json:"thread,omitempty"`
}

// PipelineDetail holds full pipeline info for the detail dialog.
type PipelineDetail struct {
	Name          string                `json:"name"`
	Description   string                `json:"description,omitempty"`
	Category      string                `json:"category,omitempty"`
	IsComposition bool                  `json:"is_composition,omitempty"`
	Skills        []string              `json:"skills,omitempty"`
	Steps         []PipelineDetailStep  `json:"steps"`
}

// buildPipelineDetail constructs a PipelineDetail from a loaded pipeline.
func buildPipelineDetail(name string, p *pipeline.Pipeline) PipelineDetail {
	var steps []PipelineDetailStep
	hasComposition := false
	for _, step := range p.Steps {
		if step.IsCompositionStep() {
			hasComposition = true
		}
		var artifactNames []string
		for _, a := range step.OutputArtifacts {
			artifactNames = append(artifactNames, a.Name)
		}
		var contract string
		var contractSchemaName string
		if step.Handover.Contract.Type != "" {
			contract = step.Handover.Contract.Type
			if step.Handover.Contract.SchemaPath != "" {
				contract += " (" + step.Handover.Contract.SchemaPath + ")"
				base := filepath.Base(step.Handover.Contract.SchemaPath)
				contractSchemaName = strings.TrimSuffix(base, ".schema.json")
				if !strings.HasSuffix(base, ".schema.json") {
					contractSchemaName = strings.TrimSuffix(base, ".json")
				}
			}
		}
		// Collect input artifact references
		var inputArtifacts []string
		for _, ia := range step.Memory.InjectArtifacts {
			ref := ia.Step + "/" + ia.Artifact
			if ia.As != "" {
				ref += " as " + ia.As
			}
			inputArtifacts = append(inputArtifacts, ref)
		}

		// Extract prompt
		var prompt string
		if step.Exec.Source != "" {
			prompt = step.Exec.Source
		}

		// On-failure strategy
		var onFailure string
		if step.Handover.Contract.OnFailure != "" {
			onFailure = step.Handover.Contract.OnFailure
		}

		// Retry
		var retryPolicy string
		var maxAttempts int
		if step.Retry.Policy != "" {
			retryPolicy = step.Retry.Policy
		}
		if step.Retry.MaxAttempts > 0 {
			maxAttempts = step.Retry.MaxAttempts
		}

		steps = append(steps, PipelineDetailStep{
			ID:                 step.ID,
			Type:               step.Type,
			Persona:            resolveForgeVars(step.Persona),
			Dependencies:       step.Dependencies,
			Timeout:            step.TimeoutMinutes,
			Optional:           step.Optional,
			Artifacts:          artifactNames,
			InputArtifacts:     inputArtifacts,
			Contract:           contract,
			ContractSchemaName: contractSchemaName,
			OnFailure:          onFailure,
			RetryPolicy:        retryPolicy,
			MaxAttempts:        maxAttempts,
			Prompt:             prompt,
			SubPipeline:        step.SubPipeline,
			Thread:             step.Thread,
		})
	}
	return PipelineDetail{
		Name:          name,
		Description:   p.Metadata.Description,
		Category:      p.Metadata.Category,
		IsComposition: hasComposition,
		Skills:        filterTemplateVars(p.Skills),
		Steps:         steps,
	}
}

// filterTemplateVars removes unresolved {{ ... }} template placeholders from a string slice.
func filterTemplateVars(items []string) []string {
	var out []string
	for _, s := range items {
		if !strings.Contains(s, "{{") {
			out = append(out, s)
		}
	}
	return out
}

// handleAPIPipelineDetail handles GET /api/pipelines/{name} - returns full pipeline detail.
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

	writeJSON(w, http.StatusOK, buildPipelineDetail(name, p))
}

// handlePipelineDetailPage handles GET /pipelines/{name} - serves an HTML detail page.
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

	// Build DAG layout — skip rework-only steps (internal retry mechanics)
	var dagSteps []DAGStepInput
	excludedSteps := make(map[string]bool)
	for _, step := range p.Steps {
		if step.ReworkOnly {
			excludedSteps[step.ID] = true
			continue
		}
		var contract string
		if step.Handover.Contract.Type != "" {
			contract = step.Handover.Contract.Type
		}
		var artifactNames []string
		for _, a := range step.OutputArtifacts {
			artifactNames = append(artifactNames, a.Name)
		}
		dagSteps = append(dagSteps, DAGStepInput{
			ID:           step.ID,
			Persona:      resolveForgeVars(step.Persona),
			Status:       "pending",
			Contract:     contract,
			Artifacts:    strings.Join(artifactNames, ", "),
			Dependencies: step.Dependencies,
		})
	}
	stripExcludedDeps(dagSteps, excludedSteps)

	// Fetch recent runs for this pipeline
	var recentRuns []RunSummary
	var runCount int
	if s.store != nil {
		runs, err := s.store.ListRuns(state.ListRunsOptions{
			PipelineName: name,
			Limit:        1000,
		})
		if err == nil {
			runCount = len(runs)
			// Build summaries for the most recent 10 runs
			limit := 10
			if len(runs) < limit {
				limit = len(runs)
			}
			for _, r := range runs[:limit] {
				recentRuns = append(recentRuns, runToSummary(r))
			}
			if len(recentRuns) > 0 {
				s.enrichRunSummaries(recentRuns, runs[:limit])
			}
		}
	}

	data := struct {
		ActivePage string
		Pipeline   PipelineDetail
		DAG        *DAGLayout
		RunCount   int
		Runs       []RunSummary
	}{
		ActivePage: "pipelines",
		Pipeline:   buildPipelineDetail(name, p),
		DAG:        ComputeDAGLayout(dagSteps),
		RunCount:   runCount,
		Runs:       recentRuns,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates["templates/pipeline_detail.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// getPipelineStartInfos returns lightweight pipeline metadata for the start form.
func getPipelineStartInfos() []PipelineStartInfo {
	names := listPipelineNames()
	var infos []PipelineStartInfo

	for _, name := range names {
		p, err := loadPipelineYAML(name)
		if err != nil {
			infos = append(infos, PipelineStartInfo{Name: name})
			continue
		}
		infos = append(infos, PipelineStartInfo{
			Name:        name,
			Description: p.Metadata.Description,
			Category:    p.Metadata.Category,
			StepCount:   len(p.Steps),
		})
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name < infos[j].Name
	})

	return infos
}

// getPipelineSummaries reads pipeline YAML files and returns summaries.
func (s *Server) getPipelineSummaries() []PipelineSummary {
	names := listPipelineNames()
	var summaries []PipelineSummary

	for _, name := range names {
		p, err := loadPipelineYAML(name)
		if err != nil {
			continue
		}
		var stepIDs []string
		hasComposition := false
		for _, step := range p.Steps {
			stepIDs = append(stepIDs, step.ID)
			if step.IsCompositionStep() {
				hasComposition = true
			}
		}
		summaries = append(summaries, PipelineSummary{
			Name:          name,
			Description:   p.Metadata.Description,
			Category:      p.Metadata.Category,
			StepCount:     len(p.Steps),
			Steps:         stepIDs,
			IsComposition: hasComposition,
			Skills:        filterTemplateVars(p.Skills),
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Name < summaries[j].Name
	})

	return summaries
}
