package webui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/manifest"
)

// TestPersonasPageRendersWithNilManifest verifies that the personas page renders
// HTML successfully when the manifest is nil.
func TestPersonasPageRendersWithNilManifest(t *testing.T) {
	srv, _ := testServer(t)
	srv.runtime.manifest = nil

	req := httptest.NewRequest("GET", "/personas", nil)
	rec := httptest.NewRecorder()
	srv.handlePersonasPage(rec, req)

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

// TestPersonasPageRendersPersonaNames verifies that the personas page renders
// persona names from the manifest.
func TestPersonasPageRendersPersonaNames(t *testing.T) {
	srv, _ := testServer(t)
	srv.runtime.manifest = &manifest.Manifest{
		Personas: map[string]manifest.Persona{
			"navigator": {
				Adapter:     "claude",
				Description: "Plans and navigates",
			},
			"craftsman": {
				Adapter:     "claude",
				Description: "Implements code",
			},
		},
	}

	req := httptest.NewRequest("GET", "/personas", nil)
	rec := httptest.NewRecorder()
	srv.handlePersonasPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, "navigator") {
		t.Errorf("expected body to contain 'navigator', got: %s", body)
	}
	if !strings.Contains(body, "craftsman") {
		t.Errorf("expected body to contain 'craftsman', got: %s", body)
	}
}

// TestPersonasAPIReturnsEmptyWithNilManifest verifies the API returns an empty
// persona list when the manifest is nil.
func TestPersonasAPIReturnsEmptyWithNilManifest(t *testing.T) {
	srv, _ := testServer(t)
	srv.runtime.manifest = nil

	req := httptest.NewRequest("GET", "/api/personas", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIPersonas(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp PersonaListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Personas) != 0 {
		t.Errorf("expected 0 personas, got %d", len(resp.Personas))
	}
}

// TestPersonasAPIHandlesNilPersonasMap verifies the API handles a manifest
// with a nil personas map.
func TestPersonasAPIHandlesNilPersonasMap(t *testing.T) {
	srv, _ := testServer(t)
	srv.runtime.manifest = &manifest.Manifest{
		Personas: nil,
	}

	req := httptest.NewRequest("GET", "/api/personas", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIPersonas(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp PersonaListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Personas) != 0 {
		t.Errorf("expected 0 personas, got %d", len(resp.Personas))
	}
}

// TestPersonasAPIReturnsAllFields verifies the API returns persona summaries
// with all fields populated.
func TestPersonasAPIReturnsAllFields(t *testing.T) {
	srv, _ := testServer(t)
	srv.runtime.manifest = &manifest.Manifest{
		Personas: map[string]manifest.Persona{
			"navigator": {
				Adapter:     "claude",
				Description: "Plans and coordinates",
				Model:       "opus",
				Temperature: 0.7,
				Permissions: manifest.Permissions{
					AllowedTools: []string{"Read", "Grep"},
					Deny:         []string{"Bash(*)"},
				},
				Skills: []string{"git", "github"},
			},
		},
	}

	req := httptest.NewRequest("GET", "/api/personas", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIPersonas(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type: expected %q, got %q", "application/json", contentType)
	}

	var resp PersonaListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Personas) != 1 {
		t.Fatalf("expected 1 persona, got %d", len(resp.Personas))
	}

	p := resp.Personas[0]
	if p.Name != "navigator" {
		t.Errorf("expected name 'navigator', got %q", p.Name)
	}
	if p.Adapter != "claude" {
		t.Errorf("expected adapter 'claude', got %q", p.Adapter)
	}
	if p.Description != "Plans and coordinates" {
		t.Errorf("expected description 'Plans and coordinates', got %q", p.Description)
	}
	if p.Model != "opus" {
		t.Errorf("expected model 'opus', got %q", p.Model)
	}
	if p.Temperature != 0.7 {
		t.Errorf("expected temperature 0.7, got %f", p.Temperature)
	}
	if len(p.AllowedTools) != 2 {
		t.Errorf("expected 2 allowed tools, got %d", len(p.AllowedTools))
	}
	if len(p.DeniedTools) != 1 {
		t.Errorf("expected 1 denied tool, got %d", len(p.DeniedTools))
	}
	if len(p.Skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(p.Skills))
	}
}

// TestPersonasAPISortsByName verifies that personas are returned sorted
// by name for consistent display.
func TestPersonasAPISortsByName(t *testing.T) {
	srv, _ := testServer(t)
	srv.runtime.manifest = &manifest.Manifest{
		Personas: map[string]manifest.Persona{
			"zebra":  {Adapter: "claude"},
			"alpha":  {Adapter: "claude"},
			"middle": {Adapter: "claude"},
		},
	}

	req := httptest.NewRequest("GET", "/api/personas", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIPersonas(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp PersonaListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Personas) != 3 {
		t.Fatalf("expected 3 personas, got %d", len(resp.Personas))
	}

	if resp.Personas[0].Name != "alpha" {
		t.Errorf("expected first persona 'alpha', got %q", resp.Personas[0].Name)
	}
	if resp.Personas[1].Name != "middle" {
		t.Errorf("expected second persona 'middle', got %q", resp.Personas[1].Name)
	}
	if resp.Personas[2].Name != "zebra" {
		t.Errorf("expected third persona 'zebra', got %q", resp.Personas[2].Name)
	}
}
