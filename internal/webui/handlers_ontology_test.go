//go:build ontology

package webui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/recinq/wave/internal/manifest"
)

// TestOntologyPageRendersEmptyHTML verifies that the ontology page renders HTML
// successfully when no ontology is configured.
func TestOntologyPageRendersEmptyHTML(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/ontology", nil)
	rec := httptest.NewRecorder()
	srv.handleOntologyPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Content-Type: expected %q, got %q", "text/html; charset=utf-8", contentType)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "<html>") {
		t.Error("expected HTML content in response")
	}
}

// TestOntologyPageRendersWithData verifies that the ontology page renders
// telos and context names from the manifest.
func TestOntologyPageRendersWithData(t *testing.T) {
	srv, _ := testServer(t)
	srv.manifest = &manifest.Manifest{
		Ontology: &manifest.Ontology{
			Telos: "Build the best orchestrator",
			Contexts: []manifest.OntologyContext{
				{Name: "pipeline", Description: "Pipeline execution", Invariants: []string{"steps are ordered"}},
				{Name: "adapter", Description: "Adapter management"},
			},
			Conventions: map[string]string{
				"naming": "kebab-case",
			},
		},
	}

	req := httptest.NewRequest("GET", "/ontology", nil)
	rec := httptest.NewRecorder()
	srv.handleOntologyPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Build the best orchestrator") {
		t.Error("expected body to contain telos")
	}
	if !strings.Contains(body, "adapter") {
		t.Error("expected body to contain context name 'adapter'")
	}
	if !strings.Contains(body, "pipeline") {
		t.Error("expected body to contain context name 'pipeline'")
	}
}

// TestOntologyAPIReturnsEmptyWhenNoOntology verifies the API returns empty
// data when no ontology is configured.
func TestOntologyAPIReturnsEmptyWhenNoOntology(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/ontology", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIOntology(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type: expected %q, got %q", "application/json", contentType)
	}

	var resp OntologyPageData
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.HasOntology {
		t.Error("expected HasOntology to be false")
	}
	if resp.Telos != "" {
		t.Errorf("expected empty telos, got %q", resp.Telos)
	}
}

// TestOntologyAPIReturnsFullData verifies the API returns complete ontology
// data including telos, contexts, and conventions.
func TestOntologyAPIReturnsFullData(t *testing.T) {
	srv, _ := testServer(t)
	srv.manifest = &manifest.Manifest{
		Ontology: &manifest.Ontology{
			Telos: "Ship quality software",
			Contexts: []manifest.OntologyContext{
				{
					Name:        "security",
					Description: "Security enforcement",
					Invariants:  []string{"no secrets in logs", "inputs sanitized"},
				},
			},
			Conventions: map[string]string{
				"commit_style": "conventional",
			},
		},
	}

	req := httptest.NewRequest("GET", "/api/ontology", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIOntology(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp OntologyPageData
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.HasOntology {
		t.Fatal("expected HasOntology to be true")
	}
	if resp.Telos != "Ship quality software" {
		t.Errorf("expected telos %q, got %q", "Ship quality software", resp.Telos)
	}
	if len(resp.Contexts) != 1 {
		t.Fatalf("expected 1 context, got %d", len(resp.Contexts))
	}
	ctx := resp.Contexts[0]
	if ctx.Name != "security" {
		t.Errorf("expected context name 'security', got %q", ctx.Name)
	}
	if ctx.InvariantCount != 2 {
		t.Errorf("expected 2 invariants, got %d", ctx.InvariantCount)
	}
	if resp.Conventions["commit_style"] != "conventional" {
		t.Errorf("expected convention 'commit_style'='conventional', got %v", resp.Conventions)
	}
}

// TestOntologyContextsSortedAlphabetically verifies that contexts are returned
// sorted by name.
func TestOntologyContextsSortedAlphabetically(t *testing.T) {
	srv, _ := testServer(t)
	srv.manifest = &manifest.Manifest{
		Ontology: &manifest.Ontology{
			Contexts: []manifest.OntologyContext{
				{Name: "zebra"},
				{Name: "alpha"},
				{Name: "middle"},
			},
		},
	}

	req := httptest.NewRequest("GET", "/api/ontology", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIOntology(rec, req)

	var resp OntologyPageData
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Contexts) != 3 {
		t.Fatalf("expected 3 contexts, got %d", len(resp.Contexts))
	}
	if resp.Contexts[0].Name != "alpha" {
		t.Errorf("expected first context 'alpha', got %q", resp.Contexts[0].Name)
	}
	if resp.Contexts[1].Name != "middle" {
		t.Errorf("expected second context 'middle', got %q", resp.Contexts[1].Name)
	}
	if resp.Contexts[2].Name != "zebra" {
		t.Errorf("expected third context 'zebra', got %q", resp.Contexts[2].Name)
	}
}

// TestFormatTimeAgo verifies the human-readable relative time formatting.
func TestFormatTimeAgo(t *testing.T) {
	tests := []struct {
		name string
		ago  time.Duration
		want string
	}{
		{"just now", 10 * time.Second, "just now"},
		{"minutes", 5 * time.Minute, "5m ago"},
		{"hours", 3 * time.Hour, "3h ago"},
		{"days", 48 * time.Hour, "2d ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTimeAgo(time.Now().Add(-tt.ago))
			if got != tt.want {
				t.Errorf("formatTimeAgo(-%v) = %q, want %q", tt.ago, got, tt.want)
			}
		})
	}
}
