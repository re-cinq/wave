package tui

import (
	"testing"

	"github.com/recinq/wave/internal/pipelinecatalog"
	"github.com/stretchr/testify/assert"
)

func TestFilterPipelines(t *testing.T) {
	pipelines := []pipelinecatalog.PipelineInfo{
		{Name: "feature", Description: "Plan and implement"},
		{Name: "hotfix", Description: "Quick fix"},
		{Name: "pr-review", Description: "Review PRs"},
		{Name: "debug", Description: "Debug issues"},
		{Name: "refactor", Description: "Safe refactoring"},
	}

	tests := []struct {
		name   string
		filter string
		want   []string
	}{
		{
			name:   "exact match",
			filter: "feature",
			want:   []string{"feature"},
		},
		{
			name:   "partial match",
			filter: "feat",
			want:   []string{"feature"},
		},
		{
			name:   "multiple matches",
			filter: "fix",
			want:   []string{"hotfix"},
		},
		{
			name:   "substring in multiple names",
			filter: "re",
			want:   []string{"feature", "pr-review", "refactor"},
		},
		{
			name:   "case insensitive",
			filter: "DEBUG",
			want:   []string{"debug"},
		},
		{
			name:   "no match",
			filter: "nonexistent",
			want:   nil,
		},
		{
			name:   "empty filter returns all",
			filter: "",
			want:   []string{"feature", "hotfix", "pr-review", "debug", "refactor"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterPipelines(pipelines, tt.filter)
			var names []string
			for _, p := range got {
				names = append(names, p.Name)
			}
			assert.Equal(t, tt.want, names)
		})
	}
}

func TestComposeCommand(t *testing.T) {
	tests := []struct {
		name     string
		pipeline string
		input    string
		flags    []string
		want     string
	}{
		{
			name:     "pipeline only",
			pipeline: "feature",
			want:     "wave run feature",
		},
		{
			name:     "with input",
			pipeline: "feature",
			input:    "add user auth",
			want:     `wave run feature "add user auth"`,
		},
		{
			name:     "with flags",
			pipeline: "debug",
			flags:    []string{"--mock"},
			want:     "wave run debug --mock",
		},
		{
			name:     "with input and flags",
			pipeline: "feature",
			input:    "add dark mode",
			flags:    []string{"--mock"},
			want:     `wave run feature "add dark mode" --mock`,
		},
		{
			name:     "empty input excluded",
			pipeline: "hotfix",
			input:    "",
			flags:    []string{"--mock"},
			want:     "wave run hotfix --mock",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComposeCommand(tt.pipeline, tt.input, tt.flags)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildPipelineOptions(t *testing.T) {
	pipelines := []pipelinecatalog.PipelineInfo{
		{Name: "feature", Description: "Plan and implement"},
		{Name: "minimal"},
	}

	options := buildPipelineOptions(pipelines)
	assert.Len(t, options, 2)

	// Option values should be pipeline names.
	assert.Equal(t, "feature", options[0].Value)
	assert.Equal(t, "minimal", options[1].Value)

	// First option key should contain both name and description.
	assert.Contains(t, options[0].Key, "feature")
	assert.Contains(t, options[0].Key, "Plan and implement")

	// Second option key should just be the name (no description).
	assert.Equal(t, "minimal", options[1].Key)
}

func TestBuildFlagOptions(t *testing.T) {
	flags := DefaultFlags()
	options := buildFlagOptions(flags)
	assert.Len(t, options, 7)

	assert.Equal(t, "--verbose", options[0].Value)
	assert.Equal(t, "--mock", options[5].Value)
}

func TestDefaultFlags(t *testing.T) {
	flags := DefaultFlags()
	assert.Len(t, flags, 7)

	names := make([]string, len(flags))
	for i, f := range flags {
		names[i] = f.Name
		assert.NotEmpty(t, f.Description)
	}
	assert.Contains(t, names, "--verbose")
	assert.Contains(t, names, "--debug")
	assert.Contains(t, names, "--output text")
	assert.Contains(t, names, "--output json")
	assert.Contains(t, names, "--dry-run")
	assert.Contains(t, names, "--mock")
	assert.Contains(t, names, "--detach")
}

func TestSelectionStruct(t *testing.T) {
	s := Selection{
		Pipeline: "feature",
		Input:    "add auth",
		Flags:    []string{"--verbose", "--dry-run"},
	}

	assert.Equal(t, "feature", s.Pipeline)
	assert.Equal(t, "add auth", s.Input)
	assert.Equal(t, []string{"--verbose", "--dry-run"}, s.Flags)
}
