//go:build webui

package webui

import (
	"net/http"
	"sort"
)

// handleAPIPersonas handles GET /api/personas - returns persona list as JSON.
func (s *Server) handleAPIPersonas(w http.ResponseWriter, r *http.Request) {
	personas := s.getPersonaSummaries()
	writeJSON(w, http.StatusOK, PersonaListResponse{Personas: personas})
}

// handlePersonasPage handles GET /personas - serves the HTML personas page.
func (s *Server) handlePersonasPage(w http.ResponseWriter, r *http.Request) {
	personas := s.getPersonaSummaries()

	data := struct {
		Personas []PersonaSummary
	}{
		Personas: personas,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates["templates/personas.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// getPersonaSummaries returns persona summaries from the manifest.
func (s *Server) getPersonaSummaries() []PersonaSummary {
	if s.manifest == nil || s.manifest.Personas == nil {
		return nil
	}

	var personas []PersonaSummary
	for name, p := range s.manifest.Personas {
		personas = append(personas, PersonaSummary{
			Name:         name,
			Description:  p.Description,
			Adapter:      p.Adapter,
			Model:        p.Model,
			Temperature:  p.Temperature,
			AllowedTools: p.Permissions.AllowedTools,
			DeniedTools:  p.Permissions.Deny,
		})
	}

	// Sort by name for consistent display
	sort.Slice(personas, func(i, j int) bool {
		return personas[i].Name < personas[j].Name
	})

	return personas
}
