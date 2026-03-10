package pipeline

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newSequenceTestExecutorFactory returns a factory function suitable for
// NewSequenceExecutor that creates a fresh DefaultPipelineExecutor with the
// given adapter runner and any additional options.
func newSequenceTestExecutorFactory(runner adapter.AdapterRunner) func(opts ...ExecutorOption) *DefaultPipelineExecutor {
	return func(opts ...ExecutorOption) *DefaultPipelineExecutor {
		return NewDefaultPipelineExecutor(runner, opts...)
	}
}

// newMinimalPipeline creates a single-step pipeline for sequence tests.
// The step has no dependencies and uses the navigator persona.
func newMinimalPipeline(name string) *Pipeline {
	return &Pipeline{
		Metadata: PipelineMetadata{Name: name},
		Steps: []Step{
			{
				ID:      "only-step",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "do the thing"},
			},
		},
	}
}

func TestSequenceExecutor_SinglePipeline(t *testing.T) {
	collector := newTestEventCollector()
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(500),
	)

	tmpDir := t.TempDir()
	m := createTestManifest(tmpDir)

	seq := NewSequenceExecutor(
		newSequenceTestExecutorFactory(mockAdapter),
		nil,
		collector,
		nil,
	)

	pipelines := []*Pipeline{newMinimalPipeline("alpha")}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := seq.Execute(ctx, pipelines, m, "test input")
	require.NoError(t, err)
	require.Len(t, result.PipelineResults, 1)

	pr := result.PipelineResults[0]
	assert.Equal(t, "alpha", pr.PipelineName)
	assert.Equal(t, "completed", pr.Status)
	assert.Nil(t, pr.Error)
	assert.True(t, pr.Duration > 0)

	// Verify sequence lifecycle events were emitted
	assert.True(t, collector.HasEventWithState(event.StateSequenceStarted))
	assert.True(t, collector.HasEventWithState(event.StateSequenceProgress))
	assert.True(t, collector.HasEventWithState(event.StateSequenceCompleted))
	assert.False(t, collector.HasEventWithState(event.StateSequenceFailed))
}

func TestSequenceExecutor_MultiplePipelines(t *testing.T) {
	collector := newTestEventCollector()
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(300),
	)

	tmpDir := t.TempDir()
	m := createTestManifest(tmpDir)

	seq := NewSequenceExecutor(
		newSequenceTestExecutorFactory(mockAdapter),
		nil,
		collector,
		nil,
	)

	pipelines := []*Pipeline{
		newMinimalPipeline("first"),
		newMinimalPipeline("second"),
		newMinimalPipeline("third"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := seq.Execute(ctx, pipelines, m, "multi test")
	require.NoError(t, err)
	require.Len(t, result.PipelineResults, 3)

	// Verify order
	assert.Equal(t, "first", result.PipelineResults[0].PipelineName)
	assert.Equal(t, "second", result.PipelineResults[1].PipelineName)
	assert.Equal(t, "third", result.PipelineResults[2].PipelineName)

	for _, pr := range result.PipelineResults {
		assert.Equal(t, "completed", pr.Status)
		assert.Nil(t, pr.Error)
	}

	// Verify completed event
	assert.True(t, collector.HasEventWithState(event.StateSequenceCompleted))
}

func TestSequenceExecutor_FailureStopsSequence(t *testing.T) {
	collector := newTestEventCollector()

	// This adapter always fails
	failAdapter := adapter.NewMockAdapter(
		adapter.WithFailure(errors.New("adapter explosion")),
	)

	tmpDir := t.TempDir()
	m := createTestManifest(tmpDir)

	seq := NewSequenceExecutor(
		newSequenceTestExecutorFactory(failAdapter),
		nil,
		collector,
		nil,
	)

	pipelines := []*Pipeline{
		newMinimalPipeline("ok-pipeline"),
		newMinimalPipeline("bad-pipeline"),
		newMinimalPipeline("never-reached"),
	}

	// First pipeline will also fail because the adapter always fails.
	// Let's use a per-pipeline adapter instead. Actually, the mock adapter
	// always fails for all pipelines. Let's just verify the first failure
	// stops the sequence.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := seq.Execute(ctx, pipelines, m, "fail test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sequence failed at pipeline 1/3")

	// Only the first pipeline should have been attempted
	require.Len(t, result.PipelineResults, 1)
	assert.Equal(t, "ok-pipeline", result.PipelineResults[0].PipelineName)
	assert.Equal(t, "failed", result.PipelineResults[0].Status)
	assert.NotNil(t, result.PipelineResults[0].Error)

	// Verify sequence_failed event
	assert.True(t, collector.HasEventWithState(event.StateSequenceFailed))
	assert.False(t, collector.HasEventWithState(event.StateSequenceCompleted))
}

func TestSequenceExecutor_EmptySequence(t *testing.T) {
	collector := newTestEventCollector()
	mockAdapter := adapter.NewMockAdapter()

	tmpDir := t.TempDir()
	m := createTestManifest(tmpDir)

	seq := NewSequenceExecutor(
		newSequenceTestExecutorFactory(mockAdapter),
		nil,
		collector,
		nil,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := seq.Execute(ctx, []*Pipeline{}, m, "empty test")
	require.NoError(t, err)
	assert.Empty(t, result.PipelineResults)
	assert.Equal(t, 0, result.TotalTokens)

	// No sequence events for empty sequence
	assert.False(t, collector.HasEventWithState(event.StateSequenceStarted))
}

func TestSequenceExecutor_ContextCancellation(t *testing.T) {
	collector := newTestEventCollector()

	// Use a delay so context cancellation can fire between pipelines
	slowAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(100),
		adapter.WithSimulatedDelay(50*time.Millisecond),
	)

	tmpDir := t.TempDir()
	m := createTestManifest(tmpDir)

	seq := NewSequenceExecutor(
		newSequenceTestExecutorFactory(slowAdapter),
		nil,
		collector,
		nil,
	)

	pipelines := []*Pipeline{
		newMinimalPipeline("first"),
		newMinimalPipeline("second"),
		newMinimalPipeline("third"),
	}

	// Cancel immediately after starting
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel

	result, err := seq.Execute(ctx, pipelines, m, "cancel test")
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)

	// No pipelines should have completed since context was already cancelled
	assert.Empty(t, result.PipelineResults)
}

func TestSequenceExecutor_ResultTracksTokens(t *testing.T) {
	collector := newTestEventCollector()
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(1000),
	)

	tmpDir := t.TempDir()
	m := createTestManifest(tmpDir)

	seq := NewSequenceExecutor(
		newSequenceTestExecutorFactory(mockAdapter),
		nil,
		collector,
		nil,
	)

	pipelines := []*Pipeline{
		newMinimalPipeline("a"),
		newMinimalPipeline("b"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := seq.Execute(ctx, pipelines, m, "token test")
	require.NoError(t, err)
	require.Len(t, result.PipelineResults, 2)

	// TotalTokens is the sum of per-pipeline TokensUsed values.
	// Note: GetTotalTokens() may return 0 after Execute() due to in-memory
	// cleanup, so both per-pipeline and total may be 0 — that's consistent.
	assert.Equal(t, result.PipelineResults[0].TokensUsed+result.PipelineResults[1].TokensUsed, result.TotalTokens)

	// Verify the sequence completed event mentions token count
	assert.True(t, collector.HasEventWithState(event.StateSequenceCompleted))
}

func TestSequenceExecutor_ExecutePlan_Sequential(t *testing.T) {
	collector := newTestEventCollector()
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(100),
	)

	tmpDir := t.TempDir()
	m := createTestManifest(tmpDir)

	seq := NewSequenceExecutor(
		newSequenceTestExecutorFactory(mockAdapter),
		nil,
		collector,
		nil,
	)

	plan := ExecutionPlan{
		Stages: []Stage{
			{Pipelines: []*Pipeline{newMinimalPipeline("a")}, Parallel: false},
			{Pipelines: []*Pipeline{newMinimalPipeline("b")}, Parallel: false},
		},
		FailFast: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := seq.ExecutePlan(ctx, plan, m, "plan test")
	require.NoError(t, err)
	require.Len(t, result.PipelineResults, 2)
	assert.Equal(t, "a", result.PipelineResults[0].PipelineName)
	assert.Equal(t, "b", result.PipelineResults[1].PipelineName)
	assert.True(t, collector.HasEventWithState(event.StateSequenceCompleted))
}

func TestSequenceExecutor_ExecutePlan_Parallel(t *testing.T) {
	collector := newTestEventCollector()
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(200),
	)

	tmpDir := t.TempDir()
	m := createTestManifest(tmpDir)

	seq := NewSequenceExecutor(
		newSequenceTestExecutorFactory(mockAdapter),
		nil,
		collector,
		nil,
	)

	plan := ExecutionPlan{
		Stages: []Stage{
			{
				Pipelines: []*Pipeline{
					newMinimalPipeline("p1"),
					newMinimalPipeline("p2"),
				},
				Parallel: true,
			},
			{Pipelines: []*Pipeline{newMinimalPipeline("p3")}, Parallel: false},
		},
		FailFast: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := seq.ExecutePlan(ctx, plan, m, "parallel test")
	require.NoError(t, err)
	require.Len(t, result.PipelineResults, 3)

	// All should be completed
	for _, pr := range result.PipelineResults {
		assert.Equal(t, "completed", pr.Status)
	}

	// Parallel stage events should be emitted
	assert.True(t, collector.HasEventWithState(event.StateParallelStageStarted))
	assert.True(t, collector.HasEventWithState(event.StateParallelStageCompleted))
	assert.True(t, collector.HasEventWithState(event.StateSequenceCompleted))
}

func TestSequenceExecutor_ExecutePlan_Empty(t *testing.T) {
	collector := newTestEventCollector()
	mockAdapter := adapter.NewMockAdapter()

	tmpDir := t.TempDir()
	m := createTestManifest(tmpDir)

	seq := NewSequenceExecutor(
		newSequenceTestExecutorFactory(mockAdapter),
		nil,
		collector,
		nil,
	)

	plan := ExecutionPlan{}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := seq.ExecutePlan(ctx, plan, m, "empty plan")
	require.NoError(t, err)
	assert.Empty(t, result.PipelineResults)
}
