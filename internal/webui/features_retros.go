//go:build retros

package webui

import "net/http"

func registerRetros(r *FeatureRegistry) {
	r.Features.Retros = true
	r.addRoutes(func(s *Server, mux *http.ServeMux) {
		mux.HandleFunc("GET /retros", s.handleRetrosPage)
		mux.HandleFunc("GET /api/retros", s.handleAPIRetros)
		mux.HandleFunc("GET /api/retros/{id}", s.handleAPIRetroDetail)
		mux.HandleFunc("POST /api/retros/{id}/narrate", s.handleNarrateRetro)
	})
}
