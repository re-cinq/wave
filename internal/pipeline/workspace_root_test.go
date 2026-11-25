package pipeline

import (
	"os"
	"testing"

	"github.com/recinq/wave/internal/workspace"
)

// TestMain keeps the default workspace root (under the user cache dir) of
// tests that run without runtime.workspace_root inside a throwaway dir.
func TestMain(m *testing.M) {
	cache, err := os.MkdirTemp("", "wave-pipeline-test-cache-")
	if err != nil {
		panic(err)
	}
	_ = os.Setenv("XDG_CACHE_HOME", cache)
	code := m.Run()
	_ = os.RemoveAll(cache)
	os.Exit(code)
}

// These helpers point resume/validation at the in-checkout legacy root, which
// is where the fixtures in resume_test.go and validation_test.go are built.

func newLegacyResumeManager(e *DefaultPipelineExecutor) *ResumeManager {
	m := NewResumeManager(e)
	m.workspaceRoot = workspace.LegacyRoot
	m.validator.WorkspaceRoot = workspace.LegacyRoot
	m.detector.WorkspaceRoot = workspace.LegacyRoot
	return m
}

func newLegacyPhaseSkipValidator() *PhaseSkipValidator {
	v := NewPhaseSkipValidator()
	v.WorkspaceRoot = workspace.LegacyRoot
	return v
}

func newLegacyStaleArtifactDetector() *StaleArtifactDetector {
	d := NewStaleArtifactDetector()
	d.WorkspaceRoot = workspace.LegacyRoot
	return d
}
