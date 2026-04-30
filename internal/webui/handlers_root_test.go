package webui

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/recinq/wave/internal/onboarding"
)

// TestHandleRoot drives Server.handleRoot directly via httptest, asserting
// that GET / branches on the presence of .agents/.onboarding-done under the
// configured repoDir.
func TestHandleRoot(t *testing.T) {
	tests := []struct {
		name           string
		writeSentinel  bool
		wantLocation   string
		repoDirOverlay func(t *testing.T, dir string) string
	}{
		{
			name:          "sentinel present redirects to /work",
			writeSentinel: true,
			wantLocation:  "/work",
		},
		{
			name:          "sentinel missing redirects to /onboard",
			writeSentinel: false,
			wantLocation:  "/onboard",
		},
		{
			name:          "empty repoDir treated as cwd and missing sentinel",
			writeSentinel: false,
			wantLocation:  "/onboard",
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
			}

			req := httptest.NewRequest("GET", "/", nil)
			rec := httptest.NewRecorder()
			srv.handleRoot(rec, req)

			if rec.Code != http.StatusFound {
				t.Fatalf("expected 302, got %d", rec.Code)
			}
			if got := rec.Header().Get("Location"); got != tc.wantLocation {
				t.Fatalf("Location: want %q, got %q", tc.wantLocation, got)
			}
			if rec.Header().Get("Location") == "/runs" {
				t.Fatalf("legacy /runs redirect leaked through handleRoot")
			}
		})
	}
}
