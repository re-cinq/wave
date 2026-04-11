//go:build ontology

package webui

import "net/http"

func init() {
	EnabledFeatures.Ontology = true
	registerFeatureRoutes(func(s *Server, mux *http.ServeMux) {
		mux.HandleFunc("GET /ontology", s.handleOntologyPage)
		mux.HandleFunc("GET /api/ontology", s.handleAPIOntology)
	})
}
