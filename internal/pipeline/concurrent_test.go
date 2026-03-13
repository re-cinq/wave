package pipeline

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcurrentExecutor_SingleAgent(t *testing.T) {
	// concurrency=1 should behave identically to non-concurrent path
	tmpDir := t.TempDir()
	collector := newTestEventCollector()

	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "ok"}`),
		adapter.WithTokensUsed(1000),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter, WithEmitter(collector))

	m := createTestManifest(tmpDir)
	p := &Pipeline{
		Kind:     "WavePipeline",
		Metadata: PipelineMetadata{Name: "test-concurrent"},
		Steps: []Step{
			{
				ID:          "step-a",
				Persona:     "craftsman",
				Concurrency: 1,
				Exec:        ExecConfig{Type: "prompt", Source: "do work"},
			},
		},
	}

	err := executor.Execute(context.Background(), p, m, "test input")
	require.NoError(t, err)

	// concurrency=1 should NOT trigger ConcurrentExecutor
	assert.False(t, collector.HasEventWithState("concurrent_start"))
}

func TestConcurrentExecutor_MultipleAgents(t *testing.T) {
	tmpDir := t.TempDir()
	collector := newTestEventCollector()

	var callCount atomic.Int32
	countingAdapter := &concurrentCountingAdapter{
		MockAdapter: adapter.NewMockAdapter(
			adapter.WithStdoutJSON(`{"result": "done"}`),
			adapter.WithTokensUsed(500),
		),
		callCount: &callCount,
	}

	executor := NewDefaultPipelineExecutor(countingAdapter, WithEmitter(collector))

	m := createTestManifest(tmpDir)
	p := &Pipeline{
		Kind:     "WavePipeline",
		Metadata: PipelineMetadata{Name: "test-concurrent-multi"},
		Steps: []Step{
			{
				ID:          "step-a",
				Persona:     "craftsman",
				Concurrency: 3,
				Exec:        ExecConfig{Type: "prompt", Source: "do parallel work"},
			},
		},
	}

	err := executor.Execute(context.Background(), p, m, "test input")
	require.NoError(t, err)

	// Should have spawned exactly 3 agents
	assert.Equal(t, int32(3), callCount.Load())

	// Should have concurrent start/complete events
	assert.True(t, collector.HasEventWithState("concurrent_start"))
	assert.True(t, collector.HasEventWithState("concurrent_complete"))
}

func TestConcurrentExecutor_CappedAt10(t *testing.T) {
	tmpDir := t.TempDir()
	collector := newTestEventCollector()

	var callCount atomic.Int32
	countingAdapter := &concurrentCountingAdapter{
		MockAdapter: adapter.NewMockAdapter(
			adapter.WithStdoutJSON(`{"result": "done"}`),
			adapter.WithTokensUsed(500),
		),
		callCount: &callCount,
	}

	executor := NewDefaultPipelineExecutor(countingAdapter, WithEmitter(collector))

	m := createTestManifest(tmpDir)
	p := &Pipeline{
		Kind:     "WavePipeline",
		Metadata: PipelineMetadata{Name: "test-concurrent-cap"},
		Steps: []Step{
			{
				ID:          "step-a",
				Persona:     "craftsman",
				Concurrency: 15, // Should be capped at 10
				Exec:        ExecConfig{Type: "prompt", Source: "do parallel work"},
			},
		},
	}

	err := executor.Execute(context.Background(), p, m, "test input")
	require.NoError(t, err)

	// Should cap at 10 agents (default max)
	assert.Equal(t, int32(10), callCount.Load())
}

func TestConcurrentExecutor_CappedByManifest(t *testing.T) {
	tmpDir := t.TempDir()
	collector := newTestEventCollector()

	var callCount atomic.Int32
	countingAdapter := &concurrentCountingAdapter{
		MockAdapter: adapter.NewMockAdapter(
			adapter.WithStdoutJSON(`{"result": "done"}`),
			adapter.WithTokensUsed(500),
		),
		callCount: &callCount,
	}

	executor := NewDefaultPipelineExecutor(countingAdapter, WithEmitter(collector))

	m := createTestManifest(tmpDir)
	m.Runtime.MaxStepConcurrency = 5 // Lower cap

	p := &Pipeline{
		Kind:     "WavePipeline",
		Metadata: PipelineMetadata{Name: "test-concurrent-manifest-cap"},
		Steps: []Step{
			{
				ID:          "step-a",
				Persona:     "craftsman",
				Concurrency: 8, // Should be capped at 5 by manifest
				Exec:        ExecConfig{Type: "prompt", Source: "do parallel work"},
			},
		},
	}

	err := executor.Execute(context.Background(), p, m, "test input")
	require.NoError(t, err)

	// Should cap at 5 agents (manifest max)
	assert.Equal(t, int32(5), callCount.Load())
}

func TestConcurrentExecutor_FailFast(t *testing.T) {
	tmpDir := t.TempDir()
	collector := newTestEventCollector()

	var callCount atomic.Int32
	failingAdapter := &concurrentFailingAdapter{
		MockAdapter: adapter.NewMockAdapter(
			adapter.WithStdoutJSON(`{"result": "done"}`),
			adapter.WithTokensUsed(500),
		),
		callCount:  &callCount,
		failOnIndex: 1, // Second agent fails
	}

	executor := NewDefaultPipelineExecutor(failingAdapter, WithEmitter(collector))

	m := createTestManifest(tmpDir)
	p := &Pipeline{
		Kind:     "WavePipeline",
		Metadata: PipelineMetadata{Name: "test-concurrent-fail"},
		Steps: []Step{
			{
				ID:          "step-a",
				Persona:     "craftsman",
				Concurrency: 3,
				Exec:        ExecConfig{Type: "prompt", Source: "do parallel work"},
			},
		},
	}

	err := executor.Execute(context.Background(), p, m, "test input")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "concurrent execution failed")
	assert.True(t, collector.HasEventWithState("concurrent_failed"))
}

func TestConcurrentExecutor_AllSucceed_MergedArtifacts(t *testing.T) {
	tmpDir := t.TempDir()
	collector := newTestEventCollector()

	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"result": "done"}`),
		adapter.WithTokensUsed(500),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter, WithEmitter(collector))

	m := createTestManifest(tmpDir)
	p := &Pipeline{
		Kind:     "WavePipeline",
		Metadata: PipelineMetadata{Name: "test-concurrent-artifacts"},
		Steps: []Step{
			{
				ID:          "step-a",
				Persona:     "craftsman",
				Concurrency: 2,
				Exec:        ExecConfig{Type: "prompt", Source: "do parallel work"},
			},
		},
	}

	err := executor.Execute(context.Background(), p, m, "test input")
	require.NoError(t, err)

	// Verify the execution has aggregated results
	assert.True(t, collector.HasEventWithState("concurrent_complete"))
}

func TestConcurrentExecutor_PerAgentStateTracking(t *testing.T) {
	tmpDir := t.TempDir()
	collector := newTestEventCollector()
	stateStore := NewMockStateStore()

	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"result": "done"}`),
		adapter.WithTokensUsed(500),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
		WithStateStore(stateStore),
	)

	m := createTestManifest(tmpDir)
	p := &Pipeline{
		Kind:     "WavePipeline",
		Metadata: PipelineMetadata{Name: "test-concurrent-state"},
		Steps: []Step{
			{
				ID:          "step-a",
				Persona:     "craftsman",
				Concurrency: 2,
				Exec:        ExecConfig{Type: "prompt", Source: "do parallel work"},
			},
		},
	}

	err := executor.Execute(context.Background(), p, m, "test input")
	require.NoError(t, err)

	// Check that per-agent states were tracked
	pipelineID := collector.GetPipelineID()
	states, _ := stateStore.GetStepStates(pipelineID)

	// Should have state records for agent_0 and agent_1
	hasAgent0 := false
	hasAgent1 := false
	for _, s := range states {
		if s.StepID == "step-a_agent_0" {
			hasAgent0 = true
		}
		if s.StepID == "step-a_agent_1" {
			hasAgent1 = true
		}
	}
	assert.True(t, hasAgent0, "should have state for agent_0")
	assert.True(t, hasAgent1, "should have state for agent_1")
}

// TestConcurrentExecutor_RaceCondition verifies no race conditions
// during concurrent adapter runs writing to shared state.
// This test is designed to be run with `go test -race`.
func TestConcurrentExecutor_RaceCondition(t *testing.T) {
	tmpDir := t.TempDir()
	collector := newTestEventCollector()

	// Use a small delay to increase chance of concurrent access
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"result": "done"}`),
		adapter.WithTokensUsed(500),
		adapter.WithSimulatedDelay(5*time.Millisecond),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter, WithEmitter(collector))

	m := createTestManifest(tmpDir)
	p := &Pipeline{
		Kind:     "WavePipeline",
		Metadata: PipelineMetadata{Name: "test-concurrent-race"},
		Steps: []Step{
			{
				ID:          "step-a",
				Persona:     "craftsman",
				Concurrency: 5,
				Exec:        ExecConfig{Type: "prompt", Source: "race test"},
			},
		},
	}

	// Run multiple times to increase race detection probability
	for i := 0; i < 3; i++ {
		err := executor.Execute(context.Background(), p, m, "test input")
		require.NoError(t, err)
	}
}

// TestConcurrentExecutor_ZeroConcurrency verifies that concurrency=0
// behaves like single-agent execution.
func TestConcurrentExecutor_ZeroConcurrency(t *testing.T) {
	tmpDir := t.TempDir()
	collector := newTestEventCollector()

	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "ok"}`),
		adapter.WithTokensUsed(1000),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter, WithEmitter(collector))

	m := createTestManifest(tmpDir)
	p := &Pipeline{
		Kind:     "WavePipeline",
		Metadata: PipelineMetadata{Name: "test-concurrent-zero"},
		Steps: []Step{
			{
				ID:          "step-a",
				Persona:     "craftsman",
				Concurrency: 0,
				Exec:        ExecConfig{Type: "prompt", Source: "do work"},
			},
		},
	}

	err := executor.Execute(context.Background(), p, m, "test input")
	require.NoError(t, err)

	// concurrency=0 should NOT trigger ConcurrentExecutor
	assert.False(t, collector.HasEventWithState("concurrent_start"))
}

// concurrentCountingAdapter wraps MockAdapter and tracks how many times Run is called.
type concurrentCountingAdapter struct {
	*adapter.MockAdapter
	callCount *atomic.Int32
}

func (a *concurrentCountingAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	a.callCount.Add(1)
	return a.MockAdapter.Run(ctx, cfg)
}

// concurrentFailingAdapter wraps MockAdapter and fails on a specific agent index.
type concurrentFailingAdapter struct {
	*adapter.MockAdapter
	callCount   *atomic.Int32
	failOnIndex int32
}

func (a *concurrentFailingAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	idx := a.callCount.Add(1) - 1
	if idx == a.failOnIndex {
		return nil, errors.New("simulated agent failure")
	}
	return a.MockAdapter.Run(ctx, cfg)
}

// Test helper functions

func TestMergeJSONArtifacts(t *testing.T) {
	contents := [][]byte{
		[]byte(`{"a": 1}`),
		[]byte(`{"b": 2}`),
		[]byte(`{"c": 3}`),
	}
	result := mergeJSONArtifacts(contents)

	// Should be a JSON array
	assert.Contains(t, string(result), "[")
	assert.Contains(t, string(result), `"a"`)
	assert.Contains(t, string(result), `"b"`)
	assert.Contains(t, string(result), `"c"`)
}

func TestMergeTextArtifacts(t *testing.T) {
	contents := [][]byte{
		[]byte("output from agent 0"),
		[]byte("output from agent 1"),
	}
	result := mergeTextArtifacts(contents)

	assert.Contains(t, string(result), "--- Agent 0 ---")
	assert.Contains(t, string(result), "--- Agent 1 ---")
	assert.Contains(t, string(result), "output from agent 0")
	assert.Contains(t, string(result), "output from agent 1")
}

func TestEffectiveConcurrency(t *testing.T) {
	tests := []struct {
		name               string
		concurrency        int
		maxStepConcurrency int
		expected           int
	}{
		{"zero returns 1", 0, 10, 1},
		{"one returns 1", 1, 10, 1},
		{"five returns 5", 5, 10, 5},
		{"fifteen capped at 10", 15, 10, 10},
		{"capped by manifest max", 8, 5, 5},
		{"negative returns 1", -1, 10, 1},
		{"zero max defaults to 10", 5, 0, 5},
		{"manifest max over 10 capped", 15, 20, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &Step{Concurrency: tt.concurrency}
			assert.Equal(t, tt.expected, step.EffectiveConcurrency(tt.maxStepConcurrency))
		})
	}
}

func TestGetMaxStepConcurrency(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		expected int
	}{
		{"zero returns default 10", 0, 10},
		{"negative returns default 10", -1, 10},
		{"five returns 5", 5, 5},
		{"ten returns 10", 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runtime := &manifest.Runtime{MaxStepConcurrency: tt.value}
			assert.Equal(t, tt.expected, runtime.GetMaxStepConcurrency())
		})
	}
}
