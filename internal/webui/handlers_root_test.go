package webui

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/recinq/wave/internal/onboarding"
)

// minimalBridgeTemplates returns a template map with a stub bridge.html so
// handleBridge can be called without the full template set.
func minimalBridgeTemplates() map[string]*template.Template {
	m := map[string]*template.Template{}
	// The bridge handler calls ExecuteTemplate(w, "templates/layout.html", data),
	// so the stub must include a layout template that wraps the content block.
	m["templates/bridge.html"] = template.Must(
		template.New("base").Parse(`{{define "templates/layout.html"}}<!doctype html>
<html><head><title>{{block "title" .}}Bridge{{end}}</title></head>
<body>{{block "content" .}}{{end}}</body>
</html>{{end}}`),
	)
	return m
}

// TestHandleRoot drives Server.handleRoot directly via httptest, asserting
// that GET / branches on the presence of .agents/.onboarding-done under the
// configured repoDir. When sentinel is present, renders bridge (200).
// When sentinel is missing, redirects to /onboard.
func TestHandleRoot(t *testing.T) {
	tests := []struct {
		name          string
		writeSentinel bool
		wantCode      int
		wantLocation  string
		repoDir       string
		// repoDirOverlay, when non-nil, runs before each test to override the
		// repoDir. Returns the repoDir to use (empty = use tmp dir directly).
		repoDirOverlay func(t *testing.T, tmp string) string
	}{
		{
			name:          "sentinel present renders bridge",
			writeSentinel: true,
			wantCode:     http.StatusOK,
			wantLocation: "",
		},
		{
			name:          "sentinel missing redirects to /onboard",
			writeSentinel: false,
			wantCode:     http.StatusFound,
			wantLocation: "/onboard",
		},
		{
			name:          "empty repoDir treated as cwd and missing sentinel",
			writeSentinel: false,
			wantCode:     http.StatusFound,
			wantLocation: "/onboard",
			repoDirOverlay: func(t *testing.T, _ string) string {
				t.Helper()
				cwd := t.TempDir()
				oldwd, err := os.Getwd()
				if err != nil {
					t.Fatalf("getwd: %v", err)
				}
				if err := os.Chdir(cwd); err != nil {
					t.Fatalf("chdir: %v", err)
				}
				t.Cleanup(func() { _ = os.Chdir(oldwd) })
				return ""
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			if tc.writeSentinel {
				if err := os.MkdirAll(filepath.Join(tmp, ".agents"), 0o755); err != nil {
					t.Fatalf("mkdir .agents: %v", err)
				}
				if err := os.WriteFile(filepath.Join(tmp, onboarding.SentinelFile), []byte(""), 0o644); err != nil {
					t.Fatalf("write sentinel: %v", err)
				}
			}

			repoDir := tmp
			if tc.repoDirOverlay != nil {
				repoDir = tc.repoDirOverlay(t, tmp)
			}

			srv := &Server{
				runtime: serverRuntime{repoDir: repoDir},
				assets:  serverAssets{templates: minimalBridgeTemplates()},
			}

			req := httptest.NewRequest("GET", "/", nil)
			rec := httptest.NewRecorder()
			srv.handleRoot(rec, req)

			if rec.Code != tc.wantCode {
				t.Fatalf("expected %d, got %d", tc.wantCode, rec.Code)
			}
			if tc.wantLocation != "" {
				if got := rec.Header().Get("Location"); got != tc.wantLocation {
					t.Fatalf("Location: want %q, got %q", tc.wantLocation, got)
				}
			}
		})
	}
}
