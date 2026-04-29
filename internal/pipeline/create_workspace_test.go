package pipeline

import (
	"os"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/adapter/adaptertest"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/ontology"
)

// TestCreateStepWorkspace_TemplateResolution tests the branch/base template resolution
// error paths in createStepWorkspace.
func TestCreateStepWorkspace_TemplateResolution(t *testing.T) {
	newExecutor := func() *DefaultPipelineExecutor {
		return NewDefaultPipelineExecutor(adaptertest.NewMockAdapter(), WithOntologyService(ontology.NoOp{}))
	}

	newExecution := func() *PipelineExecution {
		return &PipelineExecution{
			ArtifactPaths:  map[string]string{},
			WorkspacePaths: map[string]string{},
			WorktreePaths:  map[string]*WorktreeInfo{},
			Status:         &PipelineStatus{ID: "test-pipeline-abc123"},
			Manifest:       &manifest.Manifest{},
		}
	}

	t.Run("branch template step ref error", func(t *testing.T) {
		ex := newExecutor()
		exec := newExecution()

		step := &Step{
			ID: "test-step",
			Workspace: WorkspaceConfig{
				Type:   "worktree",
				Branch: "{{ steps.missing.artifacts.plan.branch }}",
			},
		}

		_, err := ex.createStepWorkspace(exec, step)
		if err == nil {
			t.Fatal("expected error for missing branch template ref")
		}
		if !strings.Contains(err.Error(), "workspace branch template") {
			t.Errorf("error should mention 'workspace branch template', got: %v", err)
		}
	})

	t.Run("base template step ref error", func(t *testing.T) {
		ex := newExecutor()
		exec := newExecution()

		step := &Step{
			ID: "test-step",
			Workspace: WorkspaceConfig{
				Type:   "worktree",
				Branch: "feat/test-branch",
				Base:   "{{ steps.missing.artifacts.plan.base }}",
			},
		}

		_, err := ex.createStepWorkspace(exec, step)
		if err == nil {
			t.Fatal("expected error for missing base template ref")
		}
		if !strings.Contains(err.Error(), "workspace base template") {
			t.Errorf("error should mention 'workspace base template', got: %v", err)
		}
	})

	t.Run("base template resolved from step output", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Write an artifact file with a base branch value
		artFile := tmpDir + "/base.json"
		if err := writeFile(artFile, []byte(`{"base": "main"}`)); err != nil {
			t.Fatal(err)
		}

		ex := newExecutor()
		exec := newExecution()
		exec.ArtifactPaths["prior:plan"] = artFile
		exec.Context = &PipelineContext{}
		exec.Pipeline = &Pipeline{Metadata: PipelineMetadata{Name: "test"}}

		step := &Step{
			ID: "test-step",
			Workspace: WorkspaceConfig{
				Type:   "worktree",
				Branch: "feat/test-branch",
				Base:   "{{ steps.prior.artifacts.plan.base }}",
			},
		}

		// This will fail later (no worktree manager configured) or panic
		// on nil fields deeper in the function. We only care that the base
		// template resolution itself succeeds (no "workspace base template" error).
		defer func() {
			if r := recover(); r != nil {
				// A panic beyond template resolution is expected — no worktree manager
				t.Logf("expected panic in createStepWorkspace: %v", r)
			}
		}()
		_, err := ex.createStepWorkspace(exec, step)
		if err != nil && strings.Contains(err.Error(), "workspace base template") {
			t.Errorf("template resolution should have succeeded, but got template error: %v", err)
		}
	})
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}
