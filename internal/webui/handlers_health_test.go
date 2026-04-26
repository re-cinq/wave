package webui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/suggest"
)

func TestHandleAPIHealth(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/health", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp HealthListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should always return checks (health provider runs without GitHub token)
	if len(resp.Checks) == 0 {
		t.Error("expected at least one health check result")
	}

	// Verify all checks have valid status values
	validStatuses := map[string]bool{"ok": true, "warn": true, "error": true}
	for _, check := range resp.Checks {
		if check.Name == "" {
			t.Error("expected non-empty check name")
		}
		if !validStatuses[check.Status] {
			t.Errorf("unexpected status %q for check %q", check.Status, check.Name)
		}
	}
}

func TestHandleHealthPage(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	srv.handleHealthPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected text/html content type, got %q", contentType)
	}
}

func TestHealthStatusString(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "ok"},
		{1, "warn"},
		{2, "error"},
		{3, "unknown"},
		{99, "unknown"},
	}

	for _, tt := range tests {
		// Cast to the suggest type through the int value
		got := healthStatusString(suggest.Status(tt.input))
		if got != tt.expected {
			t.Errorf("healthStatusString(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
