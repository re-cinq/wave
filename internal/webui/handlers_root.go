package webui

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/recinq/wave/internal/onboarding"
)

// handleRoot serves GET / by branching on the onboarding sentinel.
// If .agents/.onboarding-done exists under the project root, redirect to
// /work; otherwise redirect to /onboard so the operator can finish setup.
// Stat errors fall through to /onboard — the safer default for a project
// whose onboarding state cannot be confirmed.
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	projectDir := s.runtime.repoDir
	if projectDir == "" {
		projectDir = "."
	}
	if _, err := os.Stat(filepath.Join(projectDir, onboarding.SentinelFile)); err == nil {
		http.Redirect(w, r, "/work", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/onboard", http.StatusFound)
}
