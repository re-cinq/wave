//go:build webui

package webui

import (
	"net/http"
	"sort"
)

// PipelineSummary holds summary info about an available pipeline.
type PipelineSummary struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	StepCount   int      `json:"step_count"`
	Steps       []string `json:"steps,omitempty"`
}

// handlePipelinesPage handles GET /pipelines - serves the HTML pipelines page.
func (s *Server) handlePipelinesPage(w http.ResponseWriter, r *http.Request) {
	pipelines := s.getPipelineSummaries()

	data := struct {
		Pipelines []PipelineSummary
	}{
		Pipelines: pipelines,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates["templates/pipelines.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// handleAPIPipelines handles GET /api/pipelines - returns pipeline list as JSON.
func (s *Server) handleAPIPipelines(w http.ResponseWriter, r *http.Request) {
	pipelines := s.getPipelineSummaries()
	writeJSON(w, http.StatusOK, map[string]interface{}{"pipelines": pipelines})
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
		for _, step := range p.Steps {
			stepIDs = append(stepIDs, step.ID)
		}
		desc := p.Metadata.Description
		summaries = append(summaries, PipelineSummary{
			Name:        name,
			Description: desc,
			StepCount:   len(p.Steps),
			Steps:       stepIDs,
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Name < summaries[j].Name
	})

	return summaries
}
