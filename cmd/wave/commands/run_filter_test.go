//go:build integration

package commands

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRunCmdFilterFlags(t *testing.T) {
	cmd := NewRunCmd()
	flags := cmd.Flags()

	stepsFlag := flags.Lookup("steps")
	assert.NotNil(t, stepsFlag, "--steps flag should exist")

	excludeFlag := flags.Lookup("exclude")
	assert.NotNil(t, excludeFlag, "--exclude flag should exist")
	assert.Equal(t, "x", excludeFlag.Shorthand, "-x should be shorthand for --exclude")
}

func TestRunStepsAndExcludeMutualExclusivity(t *testing.T) {
	cmd := NewRunCmd()
	cmd.SetArgs([]string{"test-pipeline", "--steps", "a", "--exclude", "b"})

	// Capture stderr to suppress output
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestRunFromStepAndStepsIncompatibility(t *testing.T) {
	cmd := NewRunCmd()
	cmd.SetArgs([]string{"test-pipeline", "--from-step", "b", "--steps", "c"})

	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--from-step and --steps are incompatible")
}

func TestRunFromStepAndExcludeIsValid(t *testing.T) {
	// This test verifies that --from-step + --exclude doesn't error at flag validation
	cmd := NewRunCmd()
	cmd.SetArgs([]string{"test-pipeline", "--from-step", "b", "--exclude", "d"})

	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	// Will fail because pipeline doesn't exist, but should NOT fail with mutual exclusivity error
	if err != nil {
		assert.NotContains(t, err.Error(), "mutually exclusive")
		assert.NotContains(t, err.Error(), "incompatible")
	}
}

func TestRunDryRunWithFilter(t *testing.T) {
	// Create a pipeline with 4 steps
	p := &pipeline.Pipeline{
		Kind: "WavePipeline",
		Metadata: pipeline.PipelineMetadata{
			Name:        "test-dry-run-filter",
			Description: "Test pipeline for dry-run filter",
		},
		Steps: []pipeline.Step{
			{ID: "a", Persona: "navigator"},
			{ID: "b", Persona: "navigator", Dependencies: []string{"a"}},
			{ID: "c", Persona: "craftsman", Dependencies: []string{"b"}},
			{ID: "d", Persona: "craftsman", Dependencies: []string{"c"}},
		},
	}
	m := &manifest.Manifest{}

	t.Run("exclude filter shows SKIP and RUN", func(t *testing.T) {
		// Capture stderr
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		filter := pipeline.StepFilter{Exclude: []string{"c", "d"}}
		err := performDryRun(p, m, filter)
		require.NoError(t, err)

		w.Close()
		var buf bytes.Buffer
		buf.ReadFrom(r)
		os.Stderr = oldStderr
		output := buf.String()

		assert.Contains(t, output, "[RUN]")
		assert.Contains(t, output, "[SKIP]")
		assert.Contains(t, output, "2 will run")
		assert.Contains(t, output, "2 skipped")
	})

	t.Run("include filter shows SKIP and RUN", func(t *testing.T) {
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		filter := pipeline.StepFilter{Include: []string{"a"}}
		err := performDryRun(p, m, filter)
		require.NoError(t, err)

		w.Close()
		var buf bytes.Buffer
		buf.ReadFrom(r)
		os.Stderr = oldStderr
		output := buf.String()

		assert.Contains(t, output, "[RUN]")
		assert.Contains(t, output, "[SKIP]")
		assert.Contains(t, output, "1 will run")
		assert.Contains(t, output, "3 skipped")
	})

	t.Run("no filter omits SKIP/RUN prefix", func(t *testing.T) {
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		filter := pipeline.StepFilter{}
		err := performDryRun(p, m, filter)
		require.NoError(t, err)

		w.Close()
		var buf bytes.Buffer
		buf.ReadFrom(r)
		os.Stderr = oldStderr
		output := buf.String()

		assert.NotContains(t, output, "[RUN]")
		assert.NotContains(t, output, "[SKIP]")
	})
}

func TestRunDryRunWithFilterArtifactWarnings(t *testing.T) {
	p := &pipeline.Pipeline{
		Kind: "WavePipeline",
		Metadata: pipeline.PipelineMetadata{
			Name:        "test-artifact-warning",
			Description: "Test pipeline for artifact warnings",
		},
		Steps: []pipeline.Step{
			{
				ID:      "a",
				Persona: "navigator",
				OutputArtifacts: []pipeline.ArtifactDef{
					{Name: "spec", Path: ".wave/output/spec.md"},
				},
			},
			{
				ID:       "b",
				Persona:  "craftsman",
				Dependencies: []string{"a"},
				Memory: pipeline.MemoryConfig{
					InjectArtifacts: []pipeline.ArtifactRef{
						{Step: "a", Artifact: "spec", As: "spec"},
					},
				},
			},
		},
	}
	m := &manifest.Manifest{}

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	filter := pipeline.StepFilter{Include: []string{"b"}}
	err := performDryRun(p, m, filter)
	require.NoError(t, err)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stderr = oldStderr
	output := buf.String()

	assert.Contains(t, output, "Artifact warnings")
	assert.Contains(t, output, "Step 'b' needs artifact 'spec' from skipped step 'a'")
}

func TestExecuteWithStepFilter(t *testing.T) {
	// Test that executor respects step filter
	collector := &testEventCollector{}
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(100),
	)

	p := &pipeline.Pipeline{
		Kind: "WavePipeline",
		Metadata: pipeline.PipelineMetadata{
			Name: "test-filter-exec",
		},
		Steps: []pipeline.Step{
			{ID: "a", Persona: "navigator"},
			{ID: "b", Persona: "navigator"},
			{ID: "c", Persona: "navigator"},
		},
	}
	m := &manifest.Manifest{}

	t.Run("include filter runs only selected steps", func(t *testing.T) {
		collector := &testEventCollector{}
		executor := pipeline.NewDefaultPipelineExecutor(mockAdapter,
			pipeline.WithEmitter(collector),
			pipeline.WithStepFilter(pipeline.StepFilter{Include: []string{"a", "c"}}),
		)

		tmpDir := t.TempDir()
		os.Chdir(tmpDir)
		os.MkdirAll(".wave/workspaces", 0755)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := executor.Execute(ctx, p, m, "test input")
		require.NoError(t, err)

		// Verify only steps a and c were started
		startedSteps := make(map[string]bool)
		for _, ev := range collector.events {
			if ev.State == "step_started" || ev.State == "started" {
				if ev.StepID != "" {
					startedSteps[ev.StepID] = true
				}
			}
		}
		assert.True(t, startedSteps["a"], "step a should have been executed")
		assert.False(t, startedSteps["b"], "step b should have been skipped")
		assert.True(t, startedSteps["c"], "step c should have been executed")
	})

	t.Run("exclude filter skips named steps", func(t *testing.T) {
		collector = &testEventCollector{}
		executor := pipeline.NewDefaultPipelineExecutor(mockAdapter,
			pipeline.WithEmitter(collector),
			pipeline.WithStepFilter(pipeline.StepFilter{Exclude: []string{"b"}}),
		)

		tmpDir := t.TempDir()
		os.Chdir(tmpDir)
		os.MkdirAll(".wave/workspaces", 0755)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := executor.Execute(ctx, p, m, "test input")
		require.NoError(t, err)

		startedSteps := make(map[string]bool)
		for _, ev := range collector.events {
			if ev.State == "step_started" || ev.State == "started" {
				if ev.StepID != "" {
					startedSteps[ev.StepID] = true
				}
			}
		}
		assert.True(t, startedSteps["a"], "step a should have been executed")
		assert.False(t, startedSteps["b"], "step b should have been skipped")
		assert.True(t, startedSteps["c"], "step c should have been executed")
	})

	t.Run("invalid step name in filter", func(t *testing.T) {
		executor := pipeline.NewDefaultPipelineExecutor(mockAdapter,
			pipeline.WithEmitter(&testEventCollector{}),
			pipeline.WithStepFilter(pipeline.StepFilter{Include: []string{"nonexistent"}}),
		)

		tmpDir := t.TempDir()
		os.Chdir(tmpDir)
		os.MkdirAll(".wave/workspaces", 0755)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := executor.Execute(ctx, p, m, "test input")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown step(s)")
	})

	t.Run("total steps reflects filtered count", func(t *testing.T) {
		collector = &testEventCollector{}
		executor := pipeline.NewDefaultPipelineExecutor(mockAdapter,
			pipeline.WithEmitter(collector),
			pipeline.WithStepFilter(pipeline.StepFilter{Include: []string{"a"}}),
		)

		tmpDir := t.TempDir()
		os.Chdir(tmpDir)
		os.MkdirAll(".wave/workspaces", 0755)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := executor.Execute(ctx, p, m, "test input")
		require.NoError(t, err)

		// Check that the started event shows filtered count
		for _, ev := range collector.events {
			if ev.State == "started" && ev.TotalSteps > 0 {
				assert.Equal(t, 1, ev.TotalSteps, "TotalSteps should reflect filtered count")
				break
			}
		}
	})
}
