package pipelinecatalog

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

// ===========================================================================
// T004: LoadPipelineByName Tests
// ===========================================================================

func TestLoadPipelineByName_ValidPipeline(t *testing.T) {
	dir := t.TempDir()
	content := `kind: WavePipeline
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
`
	err := os.WriteFile(filepath.Join(dir, "feature.yaml"), []byte(content), 0644)
	require.NoError(t, err)

	p, err := LoadPipelineByName(dir, "feature")
	require.NoError(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, "feature", p.Metadata.Name)
	assert.Equal(t, "Plan and implement a feature", p.Metadata.Description)
	assert.Equal(t, 2, len(p.Steps))
	assert.Equal(t, "explore", p.Steps[0].ID)
	assert.Equal(t, "implement", p.Steps[1].ID)
}

func TestLoadPipelineByName_NonexistentName(t *testing.T) {
	dir := t.TempDir()
	content := `kind: WavePipeline
metadata:
  name: feature
steps: []
`
	err := os.WriteFile(filepath.Join(dir, "feature.yaml"), []byte(content), 0644)
	require.NoError(t, err)

	_, err = LoadPipelineByName(dir, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline not found: nonexistent")
}

func TestLoadPipelineByName_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	_, err := LoadPipelineByName(dir, "anything")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline not found")
}

func TestLoadPipelineByName_MalformedYAML_Skipped(t *testing.T) {
	dir := t.TempDir()

	// Write a malformed YAML file
	err := os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte(`not: valid: yaml: [[[`), 0644)
	require.NoError(t, err)

	// Write a valid pipeline
	content := `kind: WavePipeline
metadata:
  name: good-pipeline
steps: []
`
	err = os.WriteFile(filepath.Join(dir, "good.yaml"), []byte(content), 0644)
	require.NoError(t, err)

	// Should find the good pipeline, skipping the bad one
	p, err := LoadPipelineByName(dir, "good-pipeline")
	require.NoError(t, err)
	assert.Equal(t, "good-pipeline", p.Metadata.Name)
}

func TestLoadPipelineByName_NonexistentDirectory(t *testing.T) {
	_, err := LoadPipelineByName("/nonexistent/path", "anything")
	assert.Error(t, err)
}

func TestLoadPipelineByName_YmlExtension(t *testing.T) {
	dir := t.TempDir()
	content := `kind: WavePipeline
metadata:
  name: yml-pipeline
steps: []
`
	err := os.WriteFile(filepath.Join(dir, "test.yml"), []byte(content), 0644)
	require.NoError(t, err)

	p, err := LoadPipelineByName(dir, "yml-pipeline")
	require.NoError(t, err)
	assert.Equal(t, "yml-pipeline", p.Metadata.Name)
}
