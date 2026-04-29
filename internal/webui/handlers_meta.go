package webui

import "net/http"

// handleAPIAdapters handles GET /api/adapters — returns available adapter names.
func (s *Server) handleAPIAdapters(w http.ResponseWriter, r *http.Request) {
	var names []string
	if s.runtime.manifest != nil {
		for name := range s.runtime.manifest.Adapters {
			names = append(names, name)
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"adapters": names})
}

// handleAPIModels handles GET /api/models — returns suggested model names.
// Collects tier names (cheapest, balanced, strongest) plus all concrete model
// IDs from adapter default_model and tier_models values.
func (s *Server) handleAPIModels(w http.ResponseWriter, r *http.Request) {
	seen := map[string]bool{}
	var models []string
	add := func(m string) {
		if m == "" || m == "default" || seen[m] {
			return
		}
		seen[m] = true
		models = append(models, m)
	}
	add("cheapest")
	add("balanced")
	add("strongest")
	if s.runtime.manifest != nil {
		for _, a := range s.runtime.manifest.Adapters {
			add(a.DefaultModel)
			for _, m := range a.TierModels {
				add(m)
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"models": models})
}
