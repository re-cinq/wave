package webui

import (
	"net/http"
	"testing"
)

// TestNewFeatureRegistryReturnsNonNil ensures the constructor never returns nil
// regardless of which feature build tags are active.
func TestNewFeatureRegistryReturnsNonNil(t *testing.T) {
	r := NewFeatureRegistry()
	if r == nil {
		t.Fatal("NewFeatureRegistry returned nil")
	}
}

// TestAddRoutesAccumulates verifies addRoutes appends to routeFns in order.
func TestAddRoutesAccumulates(t *testing.T) {
	r := &FeatureRegistry{}
	r.addRoutes(func(_ *Server, _ *http.ServeMux) {})
	r.addRoutes(func(_ *Server, _ *http.ServeMux) {})
	if got := len(r.routeFns); got != 2 {
		t.Fatalf("expected 2 route fns, got %d", got)
	}
}

// TestRouteFnRegistersPath verifies a route fn invoked against a mux installs
// the expected path. This is the seam tests need to enable features without
// mutating globals.
func TestRouteFnRegistersPath(t *testing.T) {
	r := &FeatureRegistry{}
	r.Features.Analytics = true
	r.addRoutes(func(_ *Server, mux *http.ServeMux) {
		mux.HandleFunc("GET /test/analytics", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	mux := http.NewServeMux()
	for _, fn := range r.routeFns {
		fn(nil, mux)
	}

	req, _ := http.NewRequest("GET", "/test/analytics", nil)
	_, pattern := mux.Handler(req)
	if pattern != "GET /test/analytics" {
		t.Fatalf("expected pattern 'GET /test/analytics', got %q", pattern)
	}
}

// TestMultiFeatureComposition verifies multiple feature contributions accumulate
// independently and all routes register on the same mux.
func TestMultiFeatureComposition(t *testing.T) {
	r := &FeatureRegistry{}
	r.Features.Analytics = true
	r.Features.Webhooks = true
	r.addRoutes(func(_ *Server, mux *http.ServeMux) {
		mux.HandleFunc("GET /a", func(w http.ResponseWriter, _ *http.Request) {})
	})
	r.addRoutes(func(_ *Server, mux *http.ServeMux) {
		mux.HandleFunc("GET /b", func(w http.ResponseWriter, _ *http.Request) {})
	})

	mux := http.NewServeMux()
	for _, fn := range r.routeFns {
		fn(nil, mux)
	}

	for _, path := range []string{"/a", "/b"} {
		req, _ := http.NewRequest("GET", path, nil)
		_, pattern := mux.Handler(req)
		if pattern == "" {
			t.Errorf("path %s not registered", path)
		}
	}
}
