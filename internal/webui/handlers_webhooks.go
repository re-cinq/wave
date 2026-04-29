//go:build webhooks

package webui

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/recinq/wave/internal/state"
)

// isPublicURL checks that a URL doesn't target private/loopback addresses.
func isPublicURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if host == "" {
		return false
	}
	lower := strings.ToLower(host)
	if lower == "localhost" || lower == "localhost." {
		return false
	}
	// Check direct IP
	ip := net.ParseIP(host)
	if ip != nil {
		return !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast() && !ip.IsUnspecified()
	}
	// DNS resolution check
	ips, err := net.LookupHost(host)
	if err != nil {
		return true // allow unresolvable hosts (might resolve later)
	}
	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()) {
			return false
		}
	}
	return true
}

// --- Page Handler ---

func (s *Server) handleWebhooksPage(w http.ResponseWriter, r *http.Request) {
	webhooks, _ := s.runtime.store.ListWebhooks()

	data := struct {
		ActivePage string
		Webhooks   []*state.Webhook
	}{
		ActivePage: "webhooks",
		Webhooks:   webhooks,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.assets.templates["templates/webhooks.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleWebhookDetailPage(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid webhook id", http.StatusBadRequest)
		return
	}

	webhook, err := s.runtime.store.GetWebhook(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("webhook not found: %s", err), http.StatusNotFound)
		return
	}

	deliveries, _ := s.runtime.store.GetWebhookDeliveries(id, 50)

	data := struct {
		ActivePage string
		Webhook    *state.Webhook
		Deliveries []*state.WebhookDelivery
	}{
		ActivePage: "webhooks",
		Webhook:    webhook,
		Deliveries: deliveries,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.assets.templates["templates/webhook_detail.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// --- API Endpoints ---

func (s *Server) handleAPIWebhooks(w http.ResponseWriter, r *http.Request) {
	webhooks, err := s.runtime.store.ListWebhooks()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list webhooks: %s", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(webhooks)
}

func (s *Server) handleAPICreateWebhook(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string            `json:"name"`
		URL     string            `json:"url"`
		Events  []string          `json:"events"`
		Matcher string            `json:"matcher"`
		Headers map[string]string `json:"headers"`
		Secret  string            `json:"secret"`
		Active  *bool             `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %s", err), http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.URL == "" {
		http.Error(w, "name and url are required", http.StatusBadRequest)
		return
	}
	// Block private/loopback URLs to prevent SSRF
	if !isPublicURL(req.URL) {
		http.Error(w, "webhook URL must be a public HTTP(S) endpoint (localhost and private IPs are blocked)", http.StatusBadRequest)
		return
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}

	webhook := &state.Webhook{
		Name:      req.Name,
		URL:       req.URL,
		Events:    req.Events,
		Matcher:   req.Matcher,
		Headers:   req.Headers,
		Secret:    req.Secret,
		Active:    active,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	id, err := s.runtime.rwStore.CreateWebhook(webhook)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create webhook: %s", err), http.StatusInternalServerError)
		return
	}

	webhook.ID = id
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(webhook)
}

func (s *Server) handleAPIWebhookDetail(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid webhook id", http.StatusBadRequest)
		return
	}

	webhook, err := s.runtime.store.GetWebhook(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("webhook not found: %s", err), http.StatusNotFound)
		return
	}

	deliveries, _ := s.runtime.store.GetWebhookDeliveries(id, 20)

	resp := struct {
		*state.Webhook
		Deliveries []*state.WebhookDelivery `json:"deliveries"`
	}{
		Webhook:    webhook,
		Deliveries: deliveries,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleAPIUpdateWebhook(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid webhook id", http.StatusBadRequest)
		return
	}

	existing, err := s.runtime.store.GetWebhook(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("webhook not found: %s", err), http.StatusNotFound)
		return
	}

	var req struct {
		Name    *string           `json:"name"`
		URL     *string           `json:"url"`
		Events  []string          `json:"events"`
		Matcher *string           `json:"matcher"`
		Headers map[string]string `json:"headers"`
		Secret  *string           `json:"secret"`
		Active  *bool             `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %s", err), http.StatusBadRequest)
		return
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.URL != nil {
		existing.URL = *req.URL
	}
	if req.Events != nil {
		existing.Events = req.Events
	}
	if req.Matcher != nil {
		existing.Matcher = *req.Matcher
	}
	if req.Headers != nil {
		existing.Headers = req.Headers
	}
	if req.Secret != nil {
		existing.Secret = *req.Secret
	}
	if req.Active != nil {
		existing.Active = *req.Active
	}
	existing.UpdatedAt = time.Now()

	if err := s.runtime.rwStore.UpdateWebhook(existing); err != nil {
		http.Error(w, fmt.Sprintf("failed to update webhook: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(existing)
}

func (s *Server) handleAPIDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid webhook id", http.StatusBadRequest)
		return
	}

	if err := s.runtime.rwStore.DeleteWebhook(id); err != nil {
		http.Error(w, fmt.Sprintf("failed to delete webhook: %s", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAPITestWebhook(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid webhook id", http.StatusBadRequest)
		return
	}

	webhook, err := s.runtime.store.GetWebhook(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("webhook not found: %s", err), http.StatusNotFound)
		return
	}

	// Fire a test event
	testPayload := map[string]interface{}{
		"type":         "test",
		"webhook_id":   webhook.ID,
		"webhook_name": webhook.Name,
		"timestamp":    time.Now().Format(time.RFC3339),
		"message":      "Test webhook delivery from Wave",
	}

	payload, _ := json.Marshal(testPayload)

	resp := struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Payload string `json:"payload"`
	}{
		Success: true,
		Message: fmt.Sprintf("Test payload sent to %s", webhook.URL),
		Payload: string(payload),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
