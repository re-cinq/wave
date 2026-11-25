package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/recinq/wave/internal/adapter/adaptertest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSequenceExecutor_CrossPipelineArtifacts_WrittenToDisk verifies that
// cross-pipeline artifacts are actually written to the workspace filesystem
// when injectArtifacts processes a step with a cross-pipeline artifact ref.
func TestSequenceExecutor_CrossPipelineArtifacts_WrittenToDisk(t *testing.T) {
	workspacePath := t.TempDir()

	// Set up a DefaultPipelineExecutor with cross-pipeline artifacts.
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
	)
	executor := NewDefaultPipelineExecutor(
		mockAdapter,
		WithCrossPipelineArtifacts(map[string]map[string][]byte{
			"upstream-pipeline": {
				"analysis.json": []byte(`{"score": 95, "passed": true}`),
				"summary.md":    []byte("# Summary\n\nAll checks passed."),
			},
		}),
	)

	// Create a pipeline execution context (required by injectArtifacts).
	execution := &PipelineExecution{
		Pipeline: &Pipeline{
			Metadata: PipelineMetadata{Name: "downstream-pipeline"},
		},
		States:        make(map[string]string),
		Results:       make(map[string]map[string]interface{}),
		ArtifactPaths: make(map[string]string),
		Context:       NewPipelineContext("downstream-run-abc123", "downstream-pipeline", "consume-step"),
		Status:        &PipelineStatus{ID: "downstream-run-abc123"},
	}

	tests := []struct {
		name        string
		step        Step
		wantFiles   map[string]string // filename -> expected content
		wantErr     bool
		errContains string
	}{
		{
			name: "required cross-pipeline artifact is written to disk",
			step: Step{
				ID:      "consume-step",
				Persona: "navigator",
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{
						{
							Pipeline: "upstream-pipeline",
							Artifact: "analysis.json",
						},
					},
				},
			},
			wantFiles: map[string]string{
				"analysis.json": `{"score": 95, "passed": true}`,
			},
		},
		{
			name: "cross-pipeline artifact with alias writes under alias name",
			step: Step{
				ID:      "consume-step",
				Persona: "navigator",
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{
						{
							Pipeline: "upstream-pipeline",
							Artifact: "summary.md",
							As:       "upstream-summary.md",
						},
					},
				},
			},
			wantFiles: map[string]string{
				"upstream-summary.md": "# Summary\n\nAll checks passed.",
			},
		},
		{
			name: "multiple cross-pipeline artifacts written in one step",
			step: Step{
				ID:      "consume-step",
				Persona: "navigator",
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{
						{
							Pipeline: "upstream-pipeline",
							Artifact: "analysis.json",
						},
						{
							Pipeline: "upstream-pipeline",
							Artifact: "summary.md",
							As:       "report.md",
						},
					},
				},
			},
			wantFiles: map[string]string{
				"analysis.json": `{"score": 95, "passed": true}`,
				"report.md":     "# Summary\n\nAll checks passed.",
			},
		},
		{
			name: "missing required cross-pipeline artifact errors",
			step: Step{
				ID:      "consume-step",
				Persona: "navigator",
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{
						{
							Pipeline: "nonexistent-pipeline",
							Artifact: "data.json",
						},
					},
				},
			},
			wantErr:     true,
			errContains: "cross-pipeline artifact 'data.json' from pipeline 'nonexistent-pipeline' not found",
		},
		{
			name: "missing optional cross-pipeline artifact is skipped",
			step: Step{
				ID:      "consume-step",
				Persona: "navigator",
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{
						{
							Pipeline: "nonexistent-pipeline",
							Artifact: "data.json",
							Optional: true,
						},
					},
				},
			},
			wantFiles: map[string]string{}, // no files written
		},
		{
			name: "missing artifact name from existing pipeline errors",
			step: Step{
				ID:      "consume-step",
				Persona: "navigator",
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{
						{
							Pipeline: "upstream-pipeline",
							Artifact: "nonexistent.json",
						},
					},
				},
			},
			wantErr:     true,
			errContains: "cross-pipeline artifact 'nonexistent.json' not found in pipeline 'upstream-pipeline' outputs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use a fresh workspace subdirectory for each subtest.
			wsPath := filepath.Join(workspacePath, tt.name)
			require.NoError(t, os.MkdirAll(wsPath, 0755))

			err := executor.injectArtifacts(execution, &tt.step, wsPath)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}
			require.NoError(t, err)

			artifactsDir := filepath.Join(wsPath, ".agents", "artifacts")
			for filename, expectedContent := range tt.wantFiles {
				filePath := filepath.Join(artifactsDir, filename)
				data, readErr := os.ReadFile(filePath)
				require.NoError(t, readErr, "artifact file %s should exist on disk", filename)
				assert.Equal(t, expectedContent, string(data), "content mismatch for %s", filename)
			}
		})
	}
}
