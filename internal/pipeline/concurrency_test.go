package pipeline

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcurrencyExecutor_BasicExecution(t *testing.T) {
	var callCount atomic.Int32
	collector := testutil.NewEventCollector()
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(1000),
	)

	// Wrap to count calls
	countingAdapter := &concurrencyCountingAdapter{
		MockAdapter: mockAdapter,
		callCount:   &callCount,
	}

	executor := NewDefaultPipelineExecutor(countingAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "concurrency-test"},
		Steps: []Step{
			{
				ID:          "concurrent-step",
				Persona:     "navigator",
				Concurrency: 3,
				Exec:        ExecConfig{Source: "do work"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	// Verify 3 agents were spawned
	assert.Equal(t, int32(3), callCount.Load(), "should spawn 3 agents")

	// Verify events
	assert.True(t, collector.HasEventWithState("concurrency_start"), "should emit concurrency_start")
	assert.True(t, collector.HasEventWithState("concurrency_complete"), "should emit concurrency_complete")

	// Verify aggregated results
	events := collector.GetEvents()
	var pipelineID string
	for _, e := range events {
		if e.PipelineID != "" {
			pipelineID = e.PipelineID
			break
		}
	}
	require.NotEmpty(t, pipelineID)
}

func TestConcurrencyExecutor_FailFast(t *testing.T) {
	collector := testutil.NewEventCollector()
	callCount := &atomic.Int32{}

	failAdapter := &concurrencyFailAdapter{
		callCount:  callCount,
		failOnCall: 2, // Second agent fails
	}

	executor := NewDefaultPipelineExecutor(failAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "failfast-test"},
		Steps: []Step{
			{
				ID:          "failing-step",
				Persona:     "navigator",
				Concurrency: 3,
				Exec:        ExecConfig{Source: "do work"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.Error(t, err, "step should fail when an agent fails")
	assert.Contains(t, err.Error(), "concurrent execution failed")

	// Verify failure events
	assert.True(t, collector.HasEventWithState("concurrency_agent_failed"), "should emit agent failure event")
}

func TestConcurrencyExecutor_MaxConcurrencyCap(t *testing.T) {
	var maxConcurrent atomic.Int32
	var currentConcurrent atomic.Int32
	collector := testutil.NewEventCollector()

	// This adapter tracks max concurrent execution
	trackingAdapter := &concurrencyConcurrentTracker{
		currentConcurrent: &currentConcurrent,
		maxConcurrent:     &maxConcurrent,
	}

	executor := NewDefaultPipelineExecutor(trackingAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)
	// Set max_concurrency to 5
	m.Runtime.MaxConcurrency = 5

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "cap-test"},
		Steps: []Step{
			{
				ID:          "capped-step",
				Persona:     "navigator",
				Concurrency: 20, // Requests 20 but should be capped at 5
				Exec:        ExecConfig{Source: "do work"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	// Verify concurrency was capped at 5
	assert.True(t, collector.HasEventWithState("concurrency_start"), "should emit concurrency_start")
	events := collector.GetEvents()
	for _, e := range events {
		if e.State == "concurrency_start" {
			assert.Contains(t, e.Message, "5 concurrent agents")
		}
	}
}

func TestConcurrencyExecutor_SingleAgent(t *testing.T) {
	collector := testutil.NewEventCollector()
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(1000),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "single-test"},
		Steps: []Step{
			{
				ID:          "single-step",
				Persona:     "navigator",
				Concurrency: 1, // Should NOT trigger concurrency executor
				Exec:        ExecConfig{Source: "do work"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	// Should NOT have concurrency events — falls through to normal execution
	assert.False(t, collector.HasEventWithState("concurrency_start"), "concurrency=1 should not trigger concurrency executor")
}

func TestConcurrencyExecutor_ZeroConcurrency(t *testing.T) {
	collector := testutil.NewEventCollector()
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(1000),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "zero-test"},
		Steps: []Step{
			{
				ID:          "zero-step",
				Persona:     "navigator",
				Concurrency: 0, // Should NOT trigger concurrency executor
				Exec:        ExecConfig{Source: "do work"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	// Should NOT have concurrency events — falls through to normal execution
	assert.False(t, collector.HasEventWithState("concurrency_start"), "concurrency=0 should not trigger concurrency executor")
}

func TestConcurrencyExecutor_ResultAggregation(t *testing.T) {
	collector := testutil.NewEventCollector()
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(1000),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	step := &Step{
		ID:          "agg-step",
		Persona:     "navigator",
		Concurrency: 3,
		Exec:        ExecConfig{Source: "do work"},
	}

	execution := &PipelineExecution{
		Pipeline: &Pipeline{Metadata: PipelineMetadata{Name: "agg-test"}},
		Manifest: m,
		States:   make(map[string]string),
		Results:  make(map[string]map[string]interface{}),
		ArtifactPaths:  make(map[string]string),
		WorkspacePaths: make(map[string]string),
		WorktreePaths:  make(map[string]*WorktreeInfo),
		Input:          "test",
		Status:         &PipelineStatus{ID: "agg-test-12345678", PipelineName: "agg-test"},
		Context:        &PipelineContext{},
		AttemptContexts: make(map[string]*AttemptContext),
	}

	concurrencyExecutor := NewConcurrencyExecutor(executor)
	ctx := context.Background()
	err := concurrencyExecutor.Execute(ctx, execution, step)
	require.NoError(t, err)

	// Verify aggregated result format
	result, ok := execution.Results[step.ID]
	require.True(t, ok, "should have aggregated results")

	assert.Equal(t, 3, result["total_agents"], "total_agents should be 3")
	assert.Equal(t, 3, result["success_count"], "success_count should be 3")
	assert.Equal(t, 0, result["fail_count"], "fail_count should be 0")

	agentResults, ok := result["agent_results"].([]map[string]interface{})
	require.True(t, ok, "agent_results should be a slice")
	assert.Len(t, agentResults, 3, "should have 3 agent results")

	agentWorkspaces, ok := result["agent_workspaces"].([]string)
	require.True(t, ok, "agent_workspaces should be a string slice")
	assert.Len(t, agentWorkspaces, 3, "should have 3 agent workspaces")
}

func TestConcurrencyExecutor_WorkspaceIsolation(t *testing.T) {
	collector := testutil.NewEventCollector()
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(1000),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	step := &Step{
		ID:          "iso-step",
		Persona:     "navigator",
		Concurrency: 3,
		Exec:        ExecConfig{Source: "do work"},
	}

	execution := &PipelineExecution{
		Pipeline: &Pipeline{Metadata: PipelineMetadata{Name: "iso-test"}},
		Manifest: m,
		States:   make(map[string]string),
		Results:  make(map[string]map[string]interface{}),
		ArtifactPaths:  make(map[string]string),
		WorkspacePaths: make(map[string]string),
		WorktreePaths:  make(map[string]*WorktreeInfo),
		Input:          "test",
		Status:         &PipelineStatus{ID: "iso-test-12345678", PipelineName: "iso-test"},
		Context:        &PipelineContext{},
		AttemptContexts: make(map[string]*AttemptContext),
	}

	concurrencyExecutor := NewConcurrencyExecutor(executor)
	ctx := context.Background()
	err := concurrencyExecutor.Execute(ctx, execution, step)
	require.NoError(t, err)

	// Verify each agent got a unique workspace
	result := execution.Results[step.ID]
	agentWorkspaces := result["agent_workspaces"].([]string)

	// All workspace paths should be unique
	seen := make(map[string]bool)
	for _, ws := range agentWorkspaces {
		assert.False(t, seen[ws], "workspace path %q should be unique", ws)
		seen[ws] = true
		assert.Contains(t, ws, "agent_", "workspace should contain agent_ prefix")
	}
}

// --- Test helpers ---

// concurrencyCountingAdapter wraps MockAdapter and counts calls.
type concurrencyCountingAdapter struct {
	*adapter.MockAdapter
	callCount *atomic.Int32
}

func (a *concurrencyCountingAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	a.callCount.Add(1)
	return a.MockAdapter.Run(ctx, cfg)
}

// concurrencyFailAdapter fails on a specific call number.
type concurrencyFailAdapter struct {
	callCount  *atomic.Int32
	failOnCall int32
}

func (a *concurrencyFailAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	n := a.callCount.Add(1)
	if n == a.failOnCall {
		return nil, errors.New("simulated agent failure")
	}
	return adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(1000),
	).Run(ctx, cfg)
}

// concurrencyConcurrentTracker tracks the max concurrent executions.
type concurrencyConcurrentTracker struct {
	currentConcurrent *atomic.Int32
	maxConcurrent     *atomic.Int32
}

func (a *concurrencyConcurrentTracker) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	cur := a.currentConcurrent.Add(1)
	// Track maximum
	for {
		old := a.maxConcurrent.Load()
		if cur <= old || a.maxConcurrent.CompareAndSwap(old, cur) {
			break
		}
	}
	// Small delay to allow concurrent calls to overlap
	time.Sleep(10 * time.Millisecond)
	a.currentConcurrent.Add(-1)

	return adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(1000),
	).Run(ctx, cfg)
}
