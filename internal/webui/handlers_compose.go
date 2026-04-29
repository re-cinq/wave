package webui

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/state"
)

// handleComposePage handles GET /compose - serves the HTML composition pipelines page.
func (s *Server) handleComposePage(w http.ResponseWriter, r *http.Request) {
	pipelines := getCompositionPipelines()

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

	data := struct {
		ActivePage string
		Pipelines  []CompositionPipeline
	}{
		ActivePage: "compose",
		Pipelines:  pipelines,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.assets.templates["templates/compose.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// handleAPICompose handles GET /api/compose - returns composition pipeline list as JSON.
func (s *Server) handleAPICompose(w http.ResponseWriter, r *http.Request) {
	pipelines := getCompositionPipelines()
	writeJSON(w, http.StatusOK, CompositionListResponse{Pipelines: pipelines})
}

// getCompositionPipelines reads pipeline YAML files and returns only those with
// composition primitives, along with their structure details.
func getCompositionPipelines() []CompositionPipeline {
	names := listPipelineNames()
	var result []CompositionPipeline

	for _, name := range names {
		p, err := loadPipelineYAML(name)
		if err != nil {
			continue
		}

		hasComposition := false
		for _, step := range p.Steps {
			if step.IsCompositionStep() {
				hasComposition = true
				break
			}
		}
		if !hasComposition {
			continue
		}

		var steps []CompositionStep
		for _, step := range p.Steps {
			cs := classifyStep(&step)
			steps = append(steps, cs)
		}

		result = append(result, CompositionPipeline{
			Name:        name,
			Description: p.Metadata.Description,
			Category:    p.Metadata.Category,
			StepCount:   len(p.Steps),
			Steps:       steps,
			Skills:      filterTemplateVars(p.Skills),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// classifyStep inspects a pipeline step and returns a CompositionStep
// with the appropriate type label and details.
func classifyStep(step *pipeline.Step) CompositionStep {
	cs := CompositionStep{
		ID:      step.ID,
		Details: make(map[string]string),
	}

	switch {
	case step.Iterate != nil:
		cs.Type = "iterate"
		cs.SubPipeline = step.SubPipeline
		cs.Details["mode"] = step.Iterate.Mode
		if step.Iterate.Mode == "" {
			cs.Details["mode"] = "sequential"
		}
		if step.Iterate.MaxConcurrent > 0 {
			cs.Details["max_concurrent"] = fmt.Sprintf("%d", step.Iterate.MaxConcurrent)
		}
	case step.Branch != nil:
		cs.Type = "branch"
		var cases []string
		for k, v := range step.Branch.Cases {
			cases = append(cases, k+"->"+v)
		}
		sort.Strings(cases)
		cs.Details["cases"] = strings.Join(cases, ", ")
	case step.Gate != nil:
		cs.Type = "gate"
		cs.Details["gate_type"] = step.Gate.Type
		if step.Gate.Timeout != "" {
			cs.Details["timeout"] = step.Gate.Timeout
		}
		if step.Gate.Message != "" {
			cs.Details["message"] = step.Gate.Message
		}
	case step.Loop != nil:
		cs.Type = "loop"
		cs.Details["max_iterations"] = fmt.Sprintf("%d", step.Loop.MaxIterations)
		if step.Loop.Until != "" {
			cs.Details["until"] = step.Loop.Until
		}
		if len(step.Loop.Steps) > 0 {
			var subIDs []string
			for _, s := range step.Loop.Steps {
				subIDs = append(subIDs, s.ID)
			}
			cs.Details["sub_steps"] = strings.Join(subIDs, ", ")
		}
		cs.SubPipeline = step.SubPipeline
	case step.Aggregate != nil:
		cs.Type = "aggregate"
		cs.Details["strategy"] = step.Aggregate.Strategy
		cs.Details["into"] = step.Aggregate.Into
	case step.SubPipeline != "":
		cs.Type = "sub_pipeline"
		cs.SubPipeline = step.SubPipeline
	default:
		cs.Type = "persona"
		cs.Persona = resolveForgeVars(step.Persona)
	}

	return cs
}
