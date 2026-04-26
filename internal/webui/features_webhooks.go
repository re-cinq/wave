//go:build webhooks

package webui

import "net/http"

func registerWebhooks(r *FeatureRegistry) {
	r.Features.Webhooks = true
	r.addRoutes(func(s *Server, mux *http.ServeMux) {
		// Pages
		mux.HandleFunc("GET /webhooks", s.handleWebhooksPage)
		mux.HandleFunc("GET /webhooks/{id}", s.handleWebhookDetailPage)
		// API
		mux.HandleFunc("GET /api/webhooks", s.handleAPIWebhooks)
		mux.HandleFunc("POST /api/webhooks", s.handleAPICreateWebhook)
		mux.HandleFunc("GET /api/webhooks/{id}", s.handleAPIWebhookDetail)
		mux.HandleFunc("PUT /api/webhooks/{id}", s.handleAPIUpdateWebhook)
		mux.HandleFunc("DELETE /api/webhooks/{id}", s.handleAPIDeleteWebhook)
		mux.HandleFunc("POST /api/webhooks/{id}/test", s.handleAPITestWebhook)
	})
}
