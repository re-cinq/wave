package webui

import (
	"net/http"
	"os"
	"sort"
	"strings"
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
		ActivePage string
		Personas   []PersonaSummary
	}{
		ActivePage: "personas",
		Personas:   personas,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.assets.templates["templates/personas.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// handlePersonaDetailPage handles GET /personas/{name} - serves the HTML persona detail page.
func (s *Server) handlePersonaDetailPage(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "missing persona name", http.StatusBadRequest)
		return
	}

	if s.runtime.manifest == nil || s.runtime.manifest.Personas == nil {
		http.Error(w, "persona not found", http.StatusNotFound)
		return
	}

	p, ok := s.runtime.manifest.Personas[name]
	if !ok {
		http.Error(w, "persona not found", http.StatusNotFound)
		return
	}

	var prompt string
	if p.SystemPromptFile != "" {
		promptPath := p.GetSystemPromptPath(s.runtime.repoDir)
		if data, err := os.ReadFile(promptPath); err == nil {
			prompt = string(data)
		}
	}

	persona := PersonaSummary{
		Name:         name,
		Description:  p.Description,
		Adapter:      p.Adapter,
		Model:        p.Model,
		Temperature:  p.Temperature,
		AllowedTools: p.Permissions.AllowedTools,
		DeniedTools:  p.Permissions.Deny,
		Skills:       p.Skills,
		Prompt:       prompt,
	}

	// Find pipeline usage
	var usedBy []PersonaUsageRef
	pipelineNames := listPipelineNames()
	for _, pName := range pipelineNames {
		pl, err := loadPipelineYAML(pName)
		if err != nil {
			continue
		}
		for _, step := range pl.Steps {
			resolved := resolveForgeVars(step.Persona)
			if resolved == name || strings.EqualFold(resolved, name) {
				usedBy = append(usedBy, PersonaUsageRef{
					Pipeline: pName,
					StepID:   step.ID,
				})
			}
		}
	}

	var allowedDomains []string
	if p.Sandbox != nil {
		allowedDomains = p.Sandbox.AllowedDomains
	}

	data := PersonaDetailData{
		ActivePage:     "personas",
		Persona:        persona,
		TokenScopes:    p.TokenScopes,
		AllowedDomains: allowedDomains,
		UsedBy:         usedBy,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.assets.templates["templates/persona_detail.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// getPersonaSummaries returns persona summaries from the manifest.
func (s *Server) getPersonaSummaries() []PersonaSummary {
	if s.runtime.manifest == nil || s.runtime.manifest.Personas == nil {
		return nil
	}

	var personas []PersonaSummary
	for name, p := range s.runtime.manifest.Personas {
		var prompt string
		if p.SystemPromptFile != "" {
			promptPath := p.GetSystemPromptPath(s.runtime.repoDir)
			if data, err := os.ReadFile(promptPath); err == nil {
				prompt = string(data)
			}
		}
		personas = append(personas, PersonaSummary{
			Name:         name,
			Description:  p.Description,
			Adapter:      p.Adapter,
			Model:        p.Model,
			Temperature:  p.Temperature,
			AllowedTools: p.Permissions.AllowedTools,
			DeniedTools:  p.Permissions.Deny,
			Skills:       p.Skills,
			Prompt:       prompt,
		})
	}

	// Sort by name for consistent display
	sort.Slice(personas, func(i, j int) bool {
		return personas[i].Name < personas[j].Name
	})

	return personas
}
