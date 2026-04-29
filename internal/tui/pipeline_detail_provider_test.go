package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/recinq/wave/internal/metrics"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// detailMockStore — overrides GetRun, GetPerformanceMetrics, GetArtifacts.
// ---------------------------------------------------------------------------

type detailMockStore struct {
	baseStateStore
	run          *state.RunRecord
	runErr       error
	perfMetrics  []metrics.PerformanceMetricRecord
	metricsErr   error
	artifacts    []state.ArtifactRecord
	artifactsErr error
}

func (m *detailMockStore) GetRun(string) (*state.RunRecord, error) {
	return m.run, m.runErr
}

func (m *detailMockStore) GetPerformanceMetrics(string, string) ([]metrics.PerformanceMetricRecord, error) {
	return m.perfMetrics, m.metricsErr
}

func (m *detailMockStore) GetArtifacts(string, string) ([]state.ArtifactRecord, error) {
	return m.artifacts, m.artifactsErr
}

// ---------------------------------------------------------------------------
// Tests for DefaultDetailDataProvider
// ---------------------------------------------------------------------------

func TestDefaultDetailDataProvider_FetchAvailableDetail(t *testing.T) {
	dir := t.TempDir()
	pipelineYAML := `kind: WavePipeline
metadata:
  name: test-pipeline
  description: A test pipeline
  category: testing
requires:
  skills:
    speckit:
      version: 1.0
  tools:
    - gh
    - jq
input:
  source: github_issue_url
  example: https://github.com/org/repo/issues/1
steps:
  - id: step1
    persona: navigator
    output_artifacts:
      - name: spec_info
  - id: step2
    persona: craftsman
    output_artifacts:
      - name: code_output
      - name: test_results
`
	err := os.WriteFile(filepath.Join(dir, "test-pipeline.yaml"), []byte(pipelineYAML), 0644)
	require.NoError(t, err)

	store := &detailMockStore{}
	provider := NewDefaultDetailDataProvider(store, dir)

	got, err := provider.FetchAvailableDetail("test-pipeline")
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, "test-pipeline", got.Name)
	assert.Equal(t, "A test pipeline", got.Description)
	assert.Equal(t, "testing", got.Category)
	assert.Equal(t, 2, got.StepCount)

	require.Len(t, got.Steps, 2)
	assert.Equal(t, "step1", got.Steps[0].ID)
	assert.Equal(t, "navigator", got.Steps[0].Persona)
	assert.Equal(t, "step2", got.Steps[1].ID)
	assert.Equal(t, "craftsman", got.Steps[1].Persona)

	assert.Equal(t, "github_issue_url", got.InputSource)
	assert.Equal(t, "https://github.com/org/repo/issues/1", got.InputExample)

	assert.Contains(t, got.Artifacts, "spec_info")
	assert.Contains(t, got.Artifacts, "code_output")
	assert.Contains(t, got.Artifacts, "test_results")

	assert.Contains(t, got.Skills, "speckit")

	assert.Contains(t, got.Tools, "gh")
	assert.Contains(t, got.Tools, "jq")
}

func TestDefaultDetailDataProvider_FetchAvailableDetail_NotFound(t *testing.T) {
	dir := t.TempDir()
	// Write a different pipeline to the dir so it is not empty.
	pipelineYAML := `kind: WavePipeline
metadata:
  name: other-pipeline
input:
  source: cli
steps:
  - id: step1
    persona: navigator
`
	err := os.WriteFile(filepath.Join(dir, "other-pipeline.yaml"), []byte(pipelineYAML), 0644)
	require.NoError(t, err)

	store := &detailMockStore{}
	provider := NewDefaultDetailDataProvider(store, dir)

	got, err := provider.FetchAvailableDetail("nonexistent-pipeline")
	assert.Nil(t, got)
	assert.EqualError(t, err, "pipeline not found: nonexistent-pipeline")
}

func TestDefaultDetailDataProvider_FetchFinishedDetail_Completed(t *testing.T) {
	completedAt := timePtr(timeAt(10, 45))
	store := &detailMockStore{
		run: &state.RunRecord{
			RunID:        "run-abc",
			PipelineName: "speckit-flow",
			Status:       "completed",
			BranchName:   "feat/my-branch",
			StartedAt:    timeAt(10, 0),
			CompletedAt:  completedAt,
		},
		perfMetrics: []metrics.PerformanceMetricRecord{
			{
				StepID:     "step1",
				Persona:    "navigator",
				DurationMs: 5000,
				Success:    true,
			},
			{
				StepID:     "step2",
				Persona:    "craftsman",
				DurationMs: 10000,
				Success:    true,
			},
		},
		artifacts: []state.ArtifactRecord{
			{
				Name: "spec_info",
				Path: ".agents/artifacts/spec_info",
				Type: "json",
			},
		},
	}

	provider := NewDefaultDetailDataProvider(store, "")
	got, err := provider.FetchFinishedDetail("run-abc")
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, "run-abc", got.RunID)
	assert.Equal(t, "speckit-flow", got.Name)
	assert.Equal(t, "completed", got.Status)
	assert.Equal(t, 45*time.Minute, got.Duration)
	assert.Equal(t, timeAt(10, 45), got.CompletedAt)

	require.Len(t, got.Steps, 2)
	assert.Equal(t, "step1", got.Steps[0].ID)
	assert.Equal(t, "completed", got.Steps[0].Status)
	assert.Equal(t, 5*time.Second, got.Steps[0].Duration)
	assert.Equal(t, "navigator", got.Steps[0].Persona)

	assert.Equal(t, "step2", got.Steps[1].ID)
	assert.Equal(t, "completed", got.Steps[1].Status)
	assert.Equal(t, 10*time.Second, got.Steps[1].Duration)
	assert.Equal(t, "craftsman", got.Steps[1].Persona)

	require.Len(t, got.Artifacts, 1)
	assert.Equal(t, "spec_info", got.Artifacts[0].Name)
	assert.Equal(t, ".agents/artifacts/spec_info", got.Artifacts[0].Path)
	assert.Equal(t, "json", got.Artifacts[0].Type)

	// No failed step for a completed run.
	assert.Empty(t, got.FailedStep)
}

func TestDefaultDetailDataProvider_FetchFinishedDetail_Failed(t *testing.T) {
	completedAt := timePtr(timeAt(9, 10))
	store := &detailMockStore{
		run: &state.RunRecord{
			RunID:        "run-fail",
			PipelineName: "wave-evolve",
			Status:       "failed",
			StartedAt:    timeAt(9, 0),
			CompletedAt:  completedAt,
			ErrorMessage: "step2 contract validation failed",
		},
		perfMetrics: []metrics.PerformanceMetricRecord{
			{
				StepID:     "step1",
				Persona:    "navigator",
				DurationMs: 3000,
				Success:    true,
			},
			{
				StepID:       "step2",
				Persona:      "craftsman",
				DurationMs:   7000,
				Success:      false,
				ErrorMessage: "contract validation failed",
			},
		},
	}

	provider := NewDefaultDetailDataProvider(store, "")
	got, err := provider.FetchFinishedDetail("run-fail")
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, "failed", got.Status)
	assert.Equal(t, "step2 contract validation failed", got.ErrorMessage)
	assert.Equal(t, "step2", got.FailedStep)

	require.Len(t, got.Steps, 2)
	assert.Equal(t, "completed", got.Steps[0].Status)
	assert.Equal(t, "failed", got.Steps[1].Status)
}

func TestDefaultDetailDataProvider_FetchFinishedDetail_ZeroArtifacts(t *testing.T) {
	completedAt := timePtr(timeAt(11, 30))
	store := &detailMockStore{
		run: &state.RunRecord{
			RunID:       "run-noart",
			Status:      "completed",
			StartedAt:   timeAt(11, 0),
			CompletedAt: completedAt,
		},
		perfMetrics: []metrics.PerformanceMetricRecord{},
		artifacts: nil,
	}

	provider := NewDefaultDetailDataProvider(store, "")
	got, err := provider.FetchFinishedDetail("run-noart")
	require.NoError(t, err)
	require.NotNil(t, got)

	// Artifacts should be nil/empty — not an error.
	assert.Empty(t, got.Artifacts)
}

func TestDefaultDetailDataProvider_FetchFinishedDetail_NotFound(t *testing.T) {
	store := &detailMockStore{
		run:    nil,
		runErr: nil,
	}

	provider := NewDefaultDetailDataProvider(store, "")
	got, err := provider.FetchFinishedDetail("run-missing")

	assert.Nil(t, got)
	assert.EqualError(t, err, "run not found: run-missing")
}

func TestDefaultDetailDataProvider_FetchFinishedDetail_WorkspacePath(t *testing.T) {
	t.Chdir(t.TempDir())

	completedAt := timePtr(timeAt(10, 30))
	store := &detailMockStore{
		run: &state.RunRecord{
			RunID:        "run-ws-test",
			PipelineName: "test-pipeline",
			Status:       "completed",
			BranchName:   "feat/my-feature",
			StartedAt:    timeAt(10, 0),
			CompletedAt:  completedAt,
		},
		perfMetrics: []metrics.PerformanceMetricRecord{},
	}

	// Create the expected workspace directory.
	wsDir := filepath.Join(".agents", "workspaces", "run-ws-test", "__wt_feat-my-feature")
	require.NoError(t, os.MkdirAll(wsDir, 0755))

	provider := NewDefaultDetailDataProvider(store, "")
	got, err := provider.FetchFinishedDetail("run-ws-test")
	require.NoError(t, err)
	assert.Equal(t, wsDir, got.WorkspacePath)
}

func TestDefaultDetailDataProvider_FetchFinishedDetail_WorkspacePathMissing(t *testing.T) {
	t.Chdir(t.TempDir())

	completedAt := timePtr(timeAt(10, 30))
	store := &detailMockStore{
		run: &state.RunRecord{
			RunID:        "run-no-ws",
			PipelineName: "test-pipeline",
			Status:       "completed",
			BranchName:   "feat/gone",
			StartedAt:    timeAt(10, 0),
			CompletedAt:  completedAt,
		},
		perfMetrics: []metrics.PerformanceMetricRecord{},
	}

	provider := NewDefaultDetailDataProvider(store, "")
	got, err := provider.FetchFinishedDetail("run-no-ws")
	require.NoError(t, err)
	assert.Empty(t, got.WorkspacePath)
}

func TestDefaultDetailDataProvider_FetchFinishedDetail_EmptyBranchGlob(t *testing.T) {
	t.Chdir(t.TempDir())

	completedAt := timePtr(timeAt(10, 30))
	store := &detailMockStore{
		run: &state.RunRecord{
			RunID:        "run-glob-test",
			PipelineName: "test-pipeline",
			Status:       "completed",
			BranchName:   "",
			StartedAt:    timeAt(10, 0),
			CompletedAt:  completedAt,
		},
		perfMetrics: []metrics.PerformanceMetricRecord{},
	}

	// Create a worktree directory with any name.
	wsDir := filepath.Join(".agents", "workspaces", "run-glob-test", "__wt_some-branch")
	require.NoError(t, os.MkdirAll(wsDir, 0755))

	provider := NewDefaultDetailDataProvider(store, "")
	got, err := provider.FetchFinishedDetail("run-glob-test")
	require.NoError(t, err)
	assert.Equal(t, wsDir, got.WorkspacePath)
}

// TestSanitizeBranchName_TUIParity locks in the branch-sanitization output the
// TUI workspace-path derivation depends on. Before the dedup, this lived as a
// local sanitizeBranch() mirror in pipeline_detail_provider.go; the cases here
// match what that mirror would have produced so any drift is caught.
func TestSanitizeBranchName_TUIParity(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		expected string
	}{
		{name: "feature_slash", branch: "feat/my-feature", expected: "feat-my-feature"},
		{name: "colon_in_branch", branch: "feature/branch:name", expected: "feature-branch-name"},
		{name: "consecutive_dashes_collapsed", branch: "feat//my--branch", expected: "feat-my-branch"},
		{name: "leading_trailing_trimmed", branch: "/feat/x/", expected: "feat-x"},
		{name: "fifty_char_cap", branch: "this-is-a-very-long-branch-name-that-should-be-truncated-to-fifty-characters-maximum", expected: "this-is-a-very-long-branch-name-that-should-be-tru"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pipeline.SanitizeBranchName(tt.branch)
			assert.Equal(t, tt.expected, got)
			assert.LessOrEqual(t, len(got), 50)
		})
	}
}

func TestStepTypeLabel(t *testing.T) {
	tests := []struct {
		name     string
		step     pipeline.Step
		expected string
	}{
		{
			name:     "sub-pipeline step",
			step:     pipeline.Step{SubPipeline: "child"},
			expected: "pipeline:child",
		},
		{
			name:     "branch step",
			step:     pipeline.Step{Branch: &pipeline.BranchConfig{On: "{{ outcome }}"}},
			expected: "branch",
		},
		{
			name:     "gate step",
			step:     pipeline.Step{Gate: &pipeline.GateConfig{Type: "approval"}},
			expected: "gate",
		},
		{
			name:     "loop step",
			step:     pipeline.Step{Loop: &pipeline.LoopConfig{MaxIterations: 3}},
			expected: "loop",
		},
		{
			name:     "aggregate step",
			step:     pipeline.Step{Aggregate: &pipeline.AggregateConfig{From: "steps.*"}},
			expected: "aggregate",
		},
		{
			name:     "persona step returns step fallback",
			step:     pipeline.Step{Persona: "navigator"},
			expected: "step",
		},
		{
			name:     "empty step returns step fallback",
			step:     pipeline.Step{},
			expected: "step",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, stepTypeLabel(tt.step))
		})
	}
}
