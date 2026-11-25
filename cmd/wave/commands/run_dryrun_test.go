package commands

import (
	"bytes"
	"os"
	"testing"

	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/stretchr/testify/assert"
)

// captureDryRunStderr runs f with os.Stderr redirected and returns the output.
func captureDryRunStderr(f func() error) (string, error) {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	err := f()
	_ = w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String(), err
}

// TestDryRunShowsResolvedModelWorkspaceAndRetry is a regression test: the
// dry-run plan used to omit the model, print .agents/workspaces/<pipeline>/<step>
// regardless of workspace_root/type, and ignore step retry attempts.
func TestDryRunShowsResolvedModelWorkspaceAndRetry(t *testing.T) {
	p := &pipeline.Pipeline{
		Metadata: pipeline.PipelineMetadata{Name: "plan-check"},
		Steps: []pipeline.Step{
			{
				ID:      "analyze",
				Persona: "navigator",
				Model:   "strongest",
				Retry:   pipeline.RetryConfig{Policy: "standard"},
				Exec:    pipeline.ExecConfig{Type: "prompt", Source: "x"},
			},
			{
				ID:        "implement",
				Persona:   "craftsman",
				Workspace: pipeline.WorkspaceConfig{Type: "worktree", Branch: "feat/{{ pipeline_id }}"},
				Exec:      pipeline.ExecConfig{Type: "prompt", Source: "y"},
			},
			{
				ID:           "review",
				Persona:      "craftsman",
				Dependencies: []string{"implement"},
				Workspace:    pipeline.WorkspaceConfig{Ref: "implement"},
				Exec:         pipeline.ExecConfig{Type: "prompt", Source: "z"},
			},
		},
	}
	m := &manifest.Manifest{
		Metadata: manifest.Metadata{Name: "test"},
		Adapters: map[string]manifest.Adapter{
			"claude": {Binary: "claude", Mode: "headless", TierModels: map[string]string{"strongest": "opus"}},
			"codex":  {Binary: "codex", Mode: "headless", DefaultModel: "gpt-6.2-sol"},
		},
		Personas: map[string]manifest.Persona{
			"navigator": {Adapter: "claude", SystemPromptFile: "personas/nav.md"},
			"craftsman": {Adapter: "claude", Model: "sonnet", SystemPromptFile: "personas/craft.md"},
		},
		Runtime: manifest.Runtime{WorkspaceRoot: "/tmp/ws-root"},
	}

	output, err := captureDryRunStderr(func() error {
		return performDryRun(p, m, nil, RunOptions{})
	})
	assert.NoError(t, err)
	assert.Contains(t, output, "Model: opus")
	assert.Contains(t, output, "Model: sonnet")
	assert.Contains(t, output, "Workspace: /tmp/ws-root/<run-id>/analyze/")
	assert.Contains(t, output, "Workspace: git worktree under /tmp/ws-root/<run-id>/ (branch feat/<run-id>)")
	assert.Contains(t, output, `Workspace: shared with step "implement"`)
	assert.Contains(t, output, "Retry: up to 3 attempts (policy: standard)")
	assert.NotContains(t, output, ".agents/workspaces/plan-check/")

	// --adapter codex without --model: persona models are Claude IDs, so the
	// plan must show codex's default_model, as the executor does.
	output, err = captureDryRunStderr(func() error {
		return performDryRun(p, m, nil, RunOptions{Adapter: "codex"})
	})
	assert.NoError(t, err)
	assert.Contains(t, output, "Model: gpt-6.2-sol")
	assert.NotContains(t, output, "Model: sonnet")

	// --model with --force-model wins everywhere.
	output, err = captureDryRunStderr(func() error {
		return performDryRun(p, m, nil, RunOptions{Model: "haiku", ForceModel: true})
	})
	assert.NoError(t, err)
	assert.Contains(t, output, "Model: haiku")
	assert.NotContains(t, output, "Model: opus")
}
