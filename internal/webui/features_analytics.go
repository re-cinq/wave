//go:build analytics

package webui

import "net/http"

func registerAnalytics(r *FeatureRegistry) {
	r.Features.Analytics = true
	r.addRoutes(func(s *Server, mux *http.ServeMux) {
		mux.HandleFunc("GET /analytics", s.handleAnalyticsPage)
		mux.HandleFunc("GET /api/analytics", s.handleAPIAnalytics)
	})
}
