package webui

import (
	"net/http"

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
	if err := s.assets.templates["templates/compose.html"].Execute(w, data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// handleAPICompose handles GET /api/compose - returns composition pipeline list as JSON.
func (s *Server) handleAPICompose(w http.ResponseWriter, r *http.Request) {
	pipelines := getCompositionPipelines()
	writeJSON(w, http.StatusOK, CompositionListResponse{Pipelines: pipelines})
}
