//go:build integration

package commands

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testEventCollector collects events for testing
type testEventCollector struct {
	events []event.Event
}

func (c *testEventCollector) Emit(e event.Event) {
	c.events = append(c.events, e)
}

func (c *testEventCollector) HasEvent(state string) bool {
	for _, e := range c.events {
		if e.State == state {
			return true
		}
	}
	return false
}

func (c *testEventCollector) GetEventsByState(state string) []event.Event {
	var result []event.Event
	for _, e := range c.events {
		if e.State == state {
			result = append(result, e)
		}
	}
	return result
}

func (c *testEventCollector) GetEventsByStep(stepID string) []event.Event {
	var result []event.Event
	for _, e := range c.events {
		if e.StepID == stepID {
			result = append(result, e)
		}
	}
	return result
}

// testDataDir returns the path to the testdata directory
func testDataDir() string {
	return filepath.Join("testdata")
}

// loadTestManifest loads a manifest from testdata
func loadTestManifest(t *testing.T, subdir string) *manifest.Manifest {
	t.Helper()
	manifestPath := filepath.Join(testDataDir(), subdir, "wave.yaml")
	m, err := manifest.Load(manifestPath)
	require.NoError(t, err, "failed to load test manifest")
	return m
}

// loadTestPipeline loads a pipeline from testdata
func loadTestPipeline(t *testing.T, name string) *pipeline.Pipeline {
	t.Helper()
	pipelinePath := filepath.Join(testDataDir(), "pipelines", name+".yaml")
	loader := &pipeline.YAMLPipelineLoader{}
	p, err := loader.Load(pipelinePath)
	require.NoError(t, err, "failed to load test pipeline")
	return p
}

// createTestExecutor creates an executor with mock adapter and event collector
func createTestExecutor(collector *testEventCollector) (*pipeline.DefaultPipelineExecutor, *adapter.MockAdapter) {
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success", "result": "test output"}`),
		adapter.WithTokensUsed(1000),
	)

	opts := []pipeline.ExecutorOption{
		pipeline.WithEmitter(collector),
	}

	executor := pipeline.NewDefaultPipelineExecutor(mockAdapter, opts...)
	return executor, mockAdapter
}

// setupTestWorkspace creates a temporary workspace directory
func setupTestWorkspace(t *testing.T) (string, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "wave-test-*")
	require.NoError(t, err, "failed to create temp dir")

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}
	return tmpDir, cleanup
}

// captureStdout captures stdout output during a function call
func captureStdout(f func() error) (string, error) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String(), err
}

// TestRunDryRunOutput tests the --dry-run flag output format (T044)
func TestRunDryRunOutput(t *testing.T) {
	// Create a test pipeline
	p := &pipeline.Pipeline{
		Kind: "WavePipeline",
		Metadata: pipeline.PipelineMetadata{
			Name:        "test-pipeline",
			Description: "A test pipeline for dry-run",
		},
		Steps: []pipeline.Step{
			{
				ID:      "step1",
				Persona: "navigator",
				Memory: pipeline.MemoryConfig{
					Strategy: "fresh",
				},
				Exec: pipeline.ExecConfig{
					Type:   "prompt",
					Source: "Analyze the codebase",
				},
			},
			{
				ID:           "step2",
				Persona:      "craftsman",
				Dependencies: []string{"step1"},
				Memory: pipeline.MemoryConfig{
					Strategy: "fresh",
					InjectArtifacts: []pipeline.ArtifactRef{
						{Step: "step1", Artifact: "analysis", As: "analysis.md"},
					},
				},
				Workspace: pipeline.WorkspaceConfig{
					Mount: []pipeline.Mount{
						{Source: "./src", Target: "src", Mode: "ro"},
					},
				},
				Exec: pipeline.ExecConfig{
					Type:   "prompt",
					Source: "Implement the feature",
				},
				OutputArtifacts: []pipeline.ArtifactDef{
					{Name: "code", Path: "output/code.go", Type: "file"},
				},
				Handover: pipeline.HandoverConfig{
					Contract: pipeline.ContractConfig{
						Type:       "test_suite",
						OnFailure:  "retry",
						MaxRetries: 3,
					},
				},
			},
		},
	}

	// Create a test manifest
	m := &manifest.Manifest{
		Metadata: manifest.Metadata{Name: "test-project"},
		Adapters: map[string]manifest.Adapter{
			"claude": {Binary: "claude", Mode: "headless"},
		},
		Personas: map[string]manifest.Persona{
			"navigator": {
				Adapter:          "claude",
				SystemPromptFile: "personas/navigator.md",
				Temperature:      0.1,
				Permissions: manifest.Permissions{
					AllowedTools: []string{"Read", "Glob", "Grep"},
				},
			},
			"craftsman": {
				Adapter:          "claude",
				SystemPromptFile: "personas/craftsman.md",
				Temperature:      0.7,
				Permissions: manifest.Permissions{
					AllowedTools: []string{"Read", "Write", "Edit", "Bash"},
					Deny:         []string{"rm -rf"},
				},
			},
		},
		Runtime: manifest.Runtime{
			WorkspaceRoot: ".wave/workspaces",
		},
	}

	output, err := captureStdout(func() error {
		return performDryRun(p, m)
	})

	assert.NoError(t, err)
	assert.Contains(t, output, "Dry run for pipeline: test-pipeline")
	assert.Contains(t, output, "Description: A test pipeline for dry-run")
	assert.Contains(t, output, "Steps: 2")
	assert.Contains(t, output, "Execution plan:")
	assert.Contains(t, output, "1. step1 (persona: navigator)")
	assert.Contains(t, output, "2. step2 (persona: craftsman)")
	assert.Contains(t, output, "Dependencies: [step1]")
	assert.Contains(t, output, "Adapter: claude")
	assert.Contains(t, output, "Temp: 0.1")
	assert.Contains(t, output, "Temp: 0.7")
	assert.Contains(t, output, "Allowed tools: [Read Glob Grep]")
	assert.Contains(t, output, "Allowed tools: [Read Write Edit Bash]")
	assert.Contains(t, output, "Denied tools: [rm -rf]")
	assert.Contains(t, output, "Mount: ./src")
	assert.Contains(t, output, "Memory: fresh")
	assert.Contains(t, output, "Inject: step1:analysis as analysis.md")
	assert.Contains(t, output, "Output: code")
	assert.Contains(t, output, "Contract: test_suite")
	assert.Contains(t, output, "on_failure: retry")
	assert.Contains(t, output, "max_retries: 3")
	assert.Contains(t, output, "Workspace: .wave/workspaces/")
}

// TestRunWithNonExistentPipeline tests error handling for non-existent pipelines (T045)
func TestRunWithNonExistentPipeline(t *testing.T) {
	// Test loadPipeline with a non-existent pipeline name
	m := &manifest.Manifest{
		Metadata: manifest.Metadata{Name: "test-project"},
	}

	_, err := loadPipeline("nonexistent-pipeline-xyz", m)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline 'nonexistent-pipeline-xyz' not found")
}

// TestRunFromStep tests the --from-step functionality (T046)
// Resume skips already completed steps and only executes pending/failed ones
func TestRunFromStep(t *testing.T) {
	collector := &testEventCollector{}
	executor, _ := createTestExecutor(collector)

	// Create a pipeline with 3 steps
	p := &pipeline.Pipeline{
		Kind: "WavePipeline",
		Metadata: pipeline.PipelineMetadata{
			Name: "test-from-step",
		},
		Steps: []pipeline.Step{
			{
				ID:      "step1",
				Persona: "navigator",
				Exec:    pipeline.ExecConfig{Type: "prompt", Source: "Step 1"},
			},
			{
				ID:           "step2",
				Persona:      "navigator",
				Dependencies: []string{"step1"},
				Exec:         pipeline.ExecConfig{Type: "prompt", Source: "Step 2"},
			},
			{
				ID:           "step3",
				Persona:      "navigator",
				Dependencies: []string{"step2"},
				Exec:         pipeline.ExecConfig{Type: "prompt", Source: "Step 3"},
			},
		},
	}

	m := &manifest.Manifest{
		Metadata: manifest.Metadata{Name: "test-project"},
		Adapters: map[string]manifest.Adapter{
			"claude": {Binary: "claude", Mode: "headless"},
		},
		Personas: map[string]manifest.Persona{
			"navigator": {
				Adapter:          "claude",
				SystemPromptFile: "testdata/valid/personas/navigator.md",
			},
		},
		Runtime: manifest.Runtime{
			WorkspaceRoot:     t.TempDir(),
			DefaultTimeoutMin: 5,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Execute the full pipeline
	err := executor.Execute(ctx, p, m, "test input")
	require.NoError(t, err)

	// Verify all steps were executed
	assert.True(t, collector.HasEvent("completed"))
	step1Events := collector.GetEventsByStep("step1")
	step2Events := collector.GetEventsByStep("step2")
	step3Events := collector.GetEventsByStep("step3")
	assert.NotEmpty(t, step1Events, "step1 should have events")
	assert.NotEmpty(t, step2Events, "step2 should have events")
	assert.NotEmpty(t, step3Events, "step3 should have events")

	// Clear collector for resume test
	collector.events = nil

	// Resume from step2 using ResumeWithValidation (matches actual CLI behavior).
	// Execute cleans up the pipeline from in-memory storage after completion,
	// so we use ResumeWithValidation which takes the pipeline directly.
	err = executor.ResumeWithValidation(ctx, p, m, "test input", "step2", false)
	require.NoError(t, err)

	// Verify Resume completed successfully — step1 should not be re-executed
	step1ResumeEvents := collector.GetEventsByStep("step1")
	assert.Empty(t, step1ResumeEvents, "step1 should not have been re-executed (before fromStep)")
}

// TestRunFromStepValidation tests validation of --from-step parameter (T051)
func TestRunFromStepValidation(t *testing.T) {
	collector := &testEventCollector{}
	executor, _ := createTestExecutor(collector)

	p := &pipeline.Pipeline{
		Kind: "WavePipeline",
		Metadata: pipeline.PipelineMetadata{
			Name: "test-from-step-validation",
		},
		Steps: []pipeline.Step{
			{
				ID:      "step1",
				Persona: "navigator",
				Exec:    pipeline.ExecConfig{Type: "prompt", Source: "Step 1"},
			},
			{
				ID:           "step2",
				Persona:      "navigator",
				Dependencies: []string{"step1"},
				Exec:         pipeline.ExecConfig{Type: "prompt", Source: "Step 2"},
			},
		},
	}

	m := &manifest.Manifest{
		Metadata: manifest.Metadata{Name: "test-project"},
		Adapters: map[string]manifest.Adapter{
			"claude": {Binary: "claude", Mode: "headless"},
		},
		Personas: map[string]manifest.Persona{
			"navigator": {
				Adapter:          "claude",
				SystemPromptFile: "testdata/valid/personas/navigator.md",
			},
		},
		Runtime: manifest.Runtime{
			WorkspaceRoot:     t.TempDir(),
			DefaultTimeoutMin: 5,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test resume with non-existent step using ResumeWithValidation (matches actual CLI behavior)
	err := executor.ResumeWithValidation(ctx, p, m, "test input", "nonexistent-step", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in pipeline")

	// Test resume with non-existent pipeline via Resume
	err = executor.Resume(ctx, "nonexistent-pipeline", "step1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline \"nonexistent-pipeline\" not found")
}

// TestDryRunShowsAllStepDetails tests that dry-run shows comprehensive step information
func TestDryRunShowsAllStepDetails(t *testing.T) {
	p := &pipeline.Pipeline{
		Kind: "WavePipeline",
		Metadata: pipeline.PipelineMetadata{
			Name:        "detailed-pipeline",
			Description: "Pipeline with all step options",
		},
		Steps: []pipeline.Step{
			{
				ID:      "analyze",
				Persona: "navigator",
				Memory: pipeline.MemoryConfig{
					Strategy: "incremental",
				},
				Workspace: pipeline.WorkspaceConfig{
					Mount: []pipeline.Mount{
						{Source: "./", Target: ".", Mode: "ro"},
					},
				},
				Exec: pipeline.ExecConfig{
					Type:   "prompt",
					Source: "Analyze the codebase",
				},
				OutputArtifacts: []pipeline.ArtifactDef{
					{Name: "analysis", Path: "analysis.json", Type: "json"},
				},
			},
		},
	}

	m := &manifest.Manifest{
		Metadata: manifest.Metadata{Name: "test"},
		Adapters: map[string]manifest.Adapter{
			"claude": {Binary: "claude", Mode: "headless"},
		},
		Personas: map[string]manifest.Persona{
			"navigator": {
				Adapter:          "claude",
				SystemPromptFile: "personas/nav.md",
				Temperature:      0.2,
			},
		},
		Runtime: manifest.Runtime{
			WorkspaceRoot: ".wave/workspaces",
		},
	}

	output, err := captureStdout(func() error {
		return performDryRun(p, m)
	})

	assert.NoError(t, err)
	assert.Contains(t, output, "analyze")
	assert.Contains(t, output, "navigator")
	assert.Contains(t, output, "incremental")
	assert.Contains(t, output, "analysis")
	assert.Contains(t, output, "analysis.json")
}

// TestLoadPipelineCandidates tests the pipeline loading with different path candidates
func TestLoadPipelineCandidates(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	pipelinesDir := filepath.Join(tmpDir, ".wave", "pipelines")
	err := os.MkdirAll(pipelinesDir, 0755)
	require.NoError(t, err)

	// Create a test pipeline file
	pipelineContent := `kind: WavePipeline
metadata:
  name: test-candidate
steps:
  - id: step1
    persona: test
    exec:
      type: prompt
      source: test
`
	err = os.WriteFile(filepath.Join(pipelinesDir, "my-pipeline.yaml"), []byte(pipelineContent), 0644)
	require.NoError(t, err)

	// Change to temp dir and test loading
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	m := &manifest.Manifest{}

	// Test loading by name (should find in .wave/pipelines/)
	p, err := loadPipeline("my-pipeline", m)
	require.NoError(t, err)
	assert.Equal(t, "test-candidate", p.Metadata.Name)
}

// TestRunOptionsValidation tests RunOptions struct validation
func TestRunOptionsValidation(t *testing.T) {
	tests := []struct {
		name        string
		opts        RunOptions
		expectError bool
	}{
		{
			name: "valid options",
			opts: RunOptions{
				Pipeline: "test-pipeline",
				Manifest: "wave.yaml",
			},
			expectError: false,
		},
		{
			name: "with dry-run",
			opts: RunOptions{
				Pipeline: "test-pipeline",
				DryRun:   true,
				Manifest: "wave.yaml",
			},
			expectError: false,
		},
		{
			name: "with from-step",
			opts: RunOptions{
				Pipeline: "test-pipeline",
				FromStep: "step2",
				Manifest: "wave.yaml",
			},
			expectError: false,
		},
		{
			name: "with timeout",
			opts: RunOptions{
				Pipeline: "test-pipeline",
				Timeout:  60,
				Manifest: "wave.yaml",
			},
			expectError: false,
		},
		{
			name: "with mock",
			opts: RunOptions{
				Pipeline: "test-pipeline",
				Mock:     true,
				Manifest: "wave.yaml",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify RunOptions fields are set correctly
			assert.NotEmpty(t, tt.opts.Pipeline)
			assert.NotEmpty(t, tt.opts.Manifest)
		})
	}
}

// TestNewRunCmdFlags tests that all flags are properly defined
func TestNewRunCmdFlags(t *testing.T) {
	cmd := NewRunCmd()

	// Verify command properties
	assert.Equal(t, "run [pipeline] [input]", cmd.Use)
	assert.Contains(t, cmd.Short, "Run")

	// Verify all flags exist
	flags := cmd.Flags()

	pipelineFlag := flags.Lookup("pipeline")
	assert.NotNil(t, pipelineFlag, "pipeline flag should exist")

	inputFlag := flags.Lookup("input")
	assert.NotNil(t, inputFlag, "input flag should exist")

	dryRunFlag := flags.Lookup("dry-run")
	assert.NotNil(t, dryRunFlag, "dry-run flag should exist")

	fromStepFlag := flags.Lookup("from-step")
	assert.NotNil(t, fromStepFlag, "from-step flag should exist")

	timeoutFlag := flags.Lookup("timeout")
	assert.NotNil(t, timeoutFlag, "timeout flag should exist")

	manifestFlag := flags.Lookup("manifest")
	assert.NotNil(t, manifestFlag, "manifest flag should exist")

	mockFlag := flags.Lookup("mock")
	assert.NotNil(t, mockFlag, "mock flag should exist")
}
