package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverPipelines(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string // filename → content
		want    []PipelineInfo
		wantErr bool
	}{
		{
			name: "discovers valid pipelines",
			files: map[string]string{
				"feature.yaml": `kind: WavePipeline
metadata:
  name: feature
  description: "Plan and implement a feature"
input:
  source: cli
  example: "add dark mode"
steps:
  - id: explore
    persona: navigator
  - id: implement
    persona: craftsman
`,
				"debug.yaml": `kind: WavePipeline
metadata:
  name: debug
  description: "Systematic debugging"
input:
  source: cli
  example: "fix nil pointer"
steps:
  - id: reproduce
    persona: debugger
`,
			},
			want: []PipelineInfo{
				{Name: "debug", Description: "Systematic debugging", StepCount: 1, InputExample: "fix nil pointer"},
				{Name: "feature", Description: "Plan and implement a feature", StepCount: 2, InputExample: "add dark mode"},
			},
		},
		{
			name:  "empty directory",
			files: map[string]string{},
			want:  nil,
		},
		{
			name: "skips non-yaml files",
			files: map[string]string{
				"readme.md": "# README",
				"feature.yaml": `kind: WavePipeline
metadata:
  name: feature
  description: "A feature pipeline"
steps:
  - id: step1
    persona: nav
`,
			},
			want: []PipelineInfo{
				{Name: "feature", Description: "A feature pipeline", StepCount: 1},
			},
		},
		{
			name: "skips malformed yaml",
			files: map[string]string{
				"bad.yaml": `not: valid: yaml: [[[`,
				"good.yaml": `kind: WavePipeline
metadata:
  name: good
  description: "Valid pipeline"
steps: []
`,
			},
			want: []PipelineInfo{
				{Name: "good", Description: "Valid pipeline", StepCount: 0},
			},
		},
		{
			name: "no description",
			files: map[string]string{
				"minimal.yaml": `kind: WavePipeline
metadata:
  name: minimal
steps:
  - id: one
    persona: p
`,
			},
			want: []PipelineInfo{
				{Name: "minimal", StepCount: 1},
			},
		},
		{
			name: "yml extension supported",
			files: map[string]string{
				"test.yml": `kind: WavePipeline
metadata:
  name: test
  description: "yml file"
steps: []
`,
			},
			want: []PipelineInfo{
				{Name: "test", Description: "yml file", StepCount: 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tt.files {
				err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
				require.NoError(t, err)
			}

			got, err := DiscoverPipelines(dir)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDiscoverPipelines_NonexistentDir(t *testing.T) {
	_, err := DiscoverPipelines("/nonexistent/path")
	assert.Error(t, err)
}

func TestDiscoverPipelines_SkipsDirectories(t *testing.T) {
	dir := t.TempDir()
	err := os.Mkdir(filepath.Join(dir, "subdir.yaml"), 0755)
	require.NoError(t, err)

	got, err := DiscoverPipelines(dir)
	require.NoError(t, err)
	assert.Nil(t, got)
}
