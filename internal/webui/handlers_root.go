package webui

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/recinq/wave/internal/onboarding"
)

// handleRoot serves GET /.
// If .agents/.onboarding-done exists, render the bridge dashboard.
// Otherwise redirect to /onboard so the operator can finish setup.
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	projectDir := s.runtime.repoDir
	if projectDir == "" {
		projectDir = "."
	}
	if _, err := os.Stat(filepath.Join(projectDir, onboarding.SentinelFile)); err == nil {
		s.handleBridge(w, r)
		return
	}
	http.Redirect(w, r, "/onboard", http.StatusFound)
}
