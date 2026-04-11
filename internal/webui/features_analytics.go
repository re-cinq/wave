//go:build analytics

package webui

import "net/http"

func init() {
	EnabledFeatures.Analytics = true
	registerFeatureRoutes(func(s *Server, mux *http.ServeMux) {
		mux.HandleFunc("GET /analytics", s.handleAnalyticsPage)
		mux.HandleFunc("GET /api/analytics", s.handleAPIAnalytics)
	})
}
