//go:build ontology

package webui

import "net/http"

func registerOntology(r *FeatureRegistry) {
	r.Features.Ontology = true
	r.addRoutes(func(s *Server, mux *http.ServeMux) {
		mux.HandleFunc("GET /ontology", s.handleOntologyPage)
		mux.HandleFunc("GET /api/ontology", s.handleAPIOntology)
	})
}
