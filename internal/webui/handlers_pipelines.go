package webui

import (
	"encoding/json"
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
	if s.runtime.store != nil {
		allRuns, err := s.runtime.store.ListRuns(state.ListRunsOptions{Limit: 10000})
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

	// Sort pipelines by run count (most used first), then alphabetically
	sort.SliceStable(pipelines, func(i, j int) bool {
		if pipelines[i].RunCount != pipelines[j].RunCount {
			return pipelines[i].RunCount > pipelines[j].RunCount
		}
		return pipelines[i].Name < pipelines[j].Name
	})

	// Top frequent pipelines (run count > 0, capped at 8)
	var frequent []PipelineSummary
	for _, p := range pipelines {
		if p.RunCount > 0 {
			frequent = append(frequent, p)
			if len(frequent) >= 8 {
				break
			}
		}
	}

	data := struct {
		ActivePage        string
		Pipelines         []PipelineSummary
		Categories        []string
		FrequentPipelines []PipelineSummary
	}{
		ActivePage:        "pipelines",
		Pipelines:         pipelines,
		Categories:        catList,
		FrequentPipelines: frequent,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.assets.templates["templates/pipelines.html"].Execute(w, data); err != nil {
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
	Model              string   `json:"model,omitempty"`
	Adapter            string   `json:"adapter,omitempty"`
	Dependencies       []string `json:"dependencies,omitempty"`
	Timeout            int      `json:"timeout,omitempty"`
	Optional           bool     `json:"optional,omitempty"`
	ReworkOnly         bool     `json:"rework_only,omitempty"`
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
	Depth              int      `json:"depth,omitempty"` // DAG depth for indentation
	Script             string   `json:"script,omitempty"`
	GatePrompt         string   `json:"gate_prompt,omitempty"`
	GateType           string   `json:"gate_type,omitempty"`
	EdgeConditions     string   `json:"edge_conditions,omitempty"`
	IterateOver        []string `json:"iterate_over,omitempty"`
	IterateMode        string   `json:"iterate_mode,omitempty"`
	AggregateStrategy  string   `json:"aggregate_strategy,omitempty"`
	WorkspaceType      string   `json:"workspace_type,omitempty"`
	WorkspaceBranch    string   `json:"workspace_branch,omitempty"`
	WorkspaceBase      string   `json:"workspace_base,omitempty"`
	WorkspaceRef       string   `json:"workspace_ref,omitempty"`
	MountMode          string   `json:"mount_mode,omitempty"`
	MaxConcurrentAgents int    `json:"max_concurrent_agents,omitempty"`
}

// PipelineDetail holds full pipeline info for the detail dialog.
type PipelineDetail struct {
	Name          string               `json:"name"`
	Description   string               `json:"description,omitempty"`
	Category      string               `json:"category,omitempty"`
	IsComposition bool                 `json:"is_composition,omitempty"`
	Skills        []string             `json:"skills,omitempty"`
	Steps         []PipelineDetailStep `json:"steps"`
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
			prompt = resolveForgeVars(step.Exec.Source)
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

		// Gate/command details
		var gatePrompt, gateType, edgeConditions, script string
		if step.Gate != nil {
			gatePrompt = step.Gate.Message
			gateType = step.Gate.Type
		}
		if step.Branch != nil {
			var edges []string
			for k, v := range step.Branch.Cases {
				edges = append(edges, k+" -> "+v)
			}
			edgeConditions = strings.Join(edges, "; ")
		}
		if len(step.Edges) > 0 {
			var edges []string
			for _, e := range step.Edges {
				label := e.Target
				if e.Condition != "" {
					label += " (" + e.Condition + ")"
				}
				edges = append(edges, label)
			}
			edgeConditions = strings.Join(edges, ", ")
		}
		script = strings.TrimSpace(resolveForgeVars(step.Script))

		// Iterate/aggregate config
		var iterateOver []string
		var iterateMode, aggregateStrategy string
		if step.Iterate != nil {
			iterateMode = step.Iterate.Mode
			if iterateMode == "" {
				iterateMode = "sequential"
			}
			// Parse the over field — it's a JSON array string
			if step.Iterate.Over != "" {
				var items []string
				if err := json.Unmarshal([]byte(step.Iterate.Over), &items); err == nil {
					iterateOver = items
				}
			}
		}
		if step.Aggregate != nil {
			aggregateStrategy = step.Aggregate.Strategy
		}

		// Workspace config
		var wsType, wsBranch, wsBase, wsRef, mountMode string
		if step.Workspace.Type != "" {
			wsType = step.Workspace.Type
		}
		if step.Workspace.Branch != "" {
			wsBranch = step.Workspace.Branch
		}
		if step.Workspace.Base != "" {
			wsBase = step.Workspace.Base
		}
		if step.Workspace.Ref != "" {
			wsRef = step.Workspace.Ref
		}
		if len(step.Workspace.Mount) > 0 {
			modes := make(map[string]bool)
			for _, m := range step.Workspace.Mount {
				if m.Mode != "" {
					modes[m.Mode] = true
				} else {
					modes["readwrite"] = true
				}
			}
			var modeList []string
			for m := range modes {
				modeList = append(modeList, m)
			}
			mountMode = strings.Join(modeList, "+")
		}

		steps = append(steps, PipelineDetailStep{
			ID:                 step.ID,
			Type:               step.Type,
			Persona:            resolveForgeVars(step.Persona),
			Model:              step.Model,
			Adapter:            step.Adapter,
			Dependencies:       step.Dependencies,
			Timeout:            step.TimeoutMinutes,
			Optional:           step.Optional,
			ReworkOnly:         step.ReworkOnly,
			Artifacts:          artifactNames,
			InputArtifacts:     inputArtifacts,
			Contract:           contract,
			ContractSchemaName: contractSchemaName,
			OnFailure:          onFailure,
			RetryPolicy:        retryPolicy,
			MaxAttempts:        maxAttempts,
			Prompt:             prompt,
			SubPipeline:        stripUnresolvedVars(resolveForgeVars(step.SubPipeline)),
			Thread:             step.Thread,
			Script:             script,
			GatePrompt:         gatePrompt,
			GateType:           gateType,
			EdgeConditions:     edgeConditions,
			IterateOver:        iterateOver,
			IterateMode:        iterateMode,
			AggregateStrategy:  aggregateStrategy,
			WorkspaceType:      wsType,
			WorkspaceBranch:    wsBranch,
			WorkspaceBase:      wsBase,
			WorkspaceRef:       wsRef,
			MountMode:           mountMode,
		MaxConcurrentAgents: step.MaxConcurrentAgents,
		})
	}
	// Compute DAG depth for indentation
	depthMap := make(map[string]int)
	var computeDepth func(id string) int
	stepIndex := make(map[string][]string)
	for _, s := range steps {
		stepIndex[s.ID] = s.Dependencies
	}
	computeDepth = func(id string) int {
		if d, ok := depthMap[id]; ok {
			return d
		}
		maxDep := 0
		for _, dep := range stepIndex[id] {
			if dd := computeDepth(dep) + 1; dd > maxDep {
				maxDep = dd
			}
		}
		depthMap[id] = maxDep
		return maxDep
	}
	for i := range steps {
		steps[i].Depth = computeDepth(steps[i].ID)
	}
	// Rework-only steps: infer depth from the step that references them
	reworkRefs := make(map[string]string) // rework_step -> referencing step
	for _, step := range p.Steps {
		if step.Retry.ReworkStep != "" {
			reworkRefs[step.Retry.ReworkStep] = step.ID
		}
	}
	for i := range steps {
		if steps[i].Depth == 0 && reworkRefs[steps[i].ID] != "" {
			steps[i].Depth = depthMap[reworkRefs[steps[i].ID]]
		}
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
	if s.runtime.store != nil {
		runs, err := s.runtime.store.ListRuns(state.ListRunsOptions{
			PipelineName: name,
			Limit:        1000,
			// Filter out composition children (issue #1450) — sub-pipeline
			// runs spawned by ops-pr-respond / audit-issue / impl-issue
			// otherwise flood the recent-runs table for audit-* etc.
			TopLevelOnly: true,
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
	if err := s.assets.templates["templates/pipeline_detail.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
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
		// Infer category from name prefix if not explicitly set
		cat := p.Metadata.Category
		if cat == "" {
			if idx := strings.Index(name, "-"); idx > 0 {
				cat = name[:idx]
			}
		}

		summaries = append(summaries, PipelineSummary{
			Name:          name,
			Description:   p.Metadata.Description,
			Category:      cat,
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
