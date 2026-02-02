package pipeline

import (
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testEventCollector collects events emitted during execution
type testEventCollector struct {
	mu     sync.Mutex
	events []event.Event
}

func newTestEventCollector() *testEventCollector {
	return &testEventCollector{
		events: make([]event.Event, 0),
	}
}

func (c *testEventCollector) Emit(e event.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, e)
}

func (c *testEventCollector) GetEvents() []event.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]event.Event, len(c.events))
	copy(result, c.events)
	return result
}

func (c *testEventCollector) HasEventWithState(state string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, e := range c.events {
		if e.State == state {
			return true
		}
	}
	return false
}

func (c *testEventCollector) GetEventsByStep(stepID string) []event.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	var result []event.Event
	for _, e := range c.events {
		if e.StepID == stepID {
			result = append(result, e)
		}
	}
	return result
}

func (c *testEventCollector) GetStepExecutionOrder() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	var order []string
	seen := make(map[string]bool)
	for _, e := range c.events {
		if e.StepID != "" && e.State == "running" && !seen[e.StepID] {
			order = append(order, e.StepID)
			seen[e.StepID] = true
		}
	}
	return order
}

// createTestManifest creates a manifest for testing
func createTestManifest(workspaceRoot string) *manifest.Manifest {
	return &manifest.Manifest{
		Metadata: manifest.Metadata{Name: "test-project"},
		Adapters: map[string]manifest.Adapter{
			"claude": {Binary: "claude", Mode: "headless"},
		},
		Personas: map[string]manifest.Persona{
			"navigator": {
				Adapter:          "claude",
				SystemPromptFile: "",
				Temperature:      0.1,
			},
			"craftsman": {
				Adapter:          "claude",
				SystemPromptFile: "",
				Temperature:      0.7,
			},
		},
		Runtime: manifest.Runtime{
			WorkspaceRoot:     workspaceRoot,
			DefaultTimeoutMin: 5,
		},
	}
}

// TestStepOrdering verifies steps execute in topological order (T047)
func TestStepOrdering(t *testing.T) {
	collector := newTestEventCollector()
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(1000),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := createTestManifest(tmpDir)

	// Create a pipeline with dependencies: a -> b -> c
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "ordering-test"},
		Steps: []Step{
			{ID: "step-c", Persona: "navigator", Dependencies: []string{"step-b"}, Exec: ExecConfig{Source: "C"}},
			{ID: "step-a", Persona: "navigator", Dependencies: []string{}, Exec: ExecConfig{Source: "A"}},
			{ID: "step-b", Persona: "navigator", Dependencies: []string{"step-a"}, Exec: ExecConfig{Source: "B"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	// Verify execution order
	order := collector.GetStepExecutionOrder()
	require.Len(t, order, 3, "all steps should have executed")

	// Find positions
	posA := indexOfInSlice(order, "step-a")
	posB := indexOfInSlice(order, "step-b")
	posC := indexOfInSlice(order, "step-c")

	assert.True(t, posA < posB, "step-a should execute before step-b")
	assert.True(t, posB < posC, "step-b should execute before step-c")
}

// TestComplexDAGOrdering tests a more complex DAG structure
func TestComplexDAGOrdering(t *testing.T) {
	collector := newTestEventCollector()
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(500),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := createTestManifest(tmpDir)

	// Diamond dependency pattern:
	//     A
	//    / \
	//   B   C
	//    \ /
	//     D
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "complex-dag-test"},
		Steps: []Step{
			{ID: "step-d", Persona: "navigator", Dependencies: []string{"step-b", "step-c"}, Exec: ExecConfig{Source: "D"}},
			{ID: "step-b", Persona: "navigator", Dependencies: []string{"step-a"}, Exec: ExecConfig{Source: "B"}},
			{ID: "step-a", Persona: "navigator", Dependencies: []string{}, Exec: ExecConfig{Source: "A"}},
			{ID: "step-c", Persona: "navigator", Dependencies: []string{"step-a"}, Exec: ExecConfig{Source: "C"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	order := collector.GetStepExecutionOrder()
	require.Len(t, order, 4)

	posA := indexOfInSlice(order, "step-a")
	posB := indexOfInSlice(order, "step-b")
	posC := indexOfInSlice(order, "step-c")
	posD := indexOfInSlice(order, "step-d")

	// A must come before B and C
	assert.True(t, posA < posB, "step-a should execute before step-b")
	assert.True(t, posA < posC, "step-a should execute before step-c")

	// B and C must come before D
	assert.True(t, posB < posD, "step-b should execute before step-d")
	assert.True(t, posC < posD, "step-c should execute before step-d")
}

// TestParallelStepExecution tests that independent steps could run in parallel (T048)
// Note: The current executor runs steps sequentially, but this test verifies
// the DAG correctly identifies independent steps that COULD run in parallel.
func TestParallelStepExecution(t *testing.T) {
	collector := newTestEventCollector()

	// Track concurrent execution
	var maxConcurrent int32
	var currentConcurrent int32

	// Create a mock adapter that tracks concurrency
	concurrentAdapter := &concurrencyTrackingAdapter{
		MockAdapter: adapter.NewMockAdapter(
			adapter.WithStdoutJSON(`{"status": "success"}`),
			adapter.WithTokensUsed(500),
			adapter.WithSimulatedDelay(10*time.Millisecond),
		),
		onStart: func() {
			current := atomic.AddInt32(&currentConcurrent, 1)
			for {
				old := atomic.LoadInt32(&maxConcurrent)
				if current <= old || atomic.CompareAndSwapInt32(&maxConcurrent, old, current) {
					break
				}
			}
		},
		onEnd: func() {
			atomic.AddInt32(&currentConcurrent, -1)
		},
	}

	executor := NewDefaultPipelineExecutor(concurrentAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := createTestManifest(tmpDir)

	// Pipeline with independent steps B and C that could run in parallel
	//     A
	//    / \
	//   B   C
	//    \ /
	//     D
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "parallel-test"},
		Steps: []Step{
			{ID: "step-a", Persona: "navigator", Exec: ExecConfig{Source: "A"}},
			{ID: "step-b", Persona: "navigator", Dependencies: []string{"step-a"}, Exec: ExecConfig{Source: "B"}},
			{ID: "step-c", Persona: "navigator", Dependencies: []string{"step-a"}, Exec: ExecConfig{Source: "C"}},
			{ID: "step-d", Persona: "navigator", Dependencies: []string{"step-b", "step-c"}, Exec: ExecConfig{Source: "D"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	// Verify all steps completed
	events := collector.GetEvents()
	completedSteps := 0
	for _, e := range events {
		if e.State == "completed" && e.StepID != "" {
			completedSteps++
		}
	}
	assert.Equal(t, 4, completedSteps, "all 4 steps should complete")

	// Verify ordering constraints are met even in sequential execution
	order := collector.GetStepExecutionOrder()
	posA := indexOfInSlice(order, "step-a")
	posB := indexOfInSlice(order, "step-b")
	posC := indexOfInSlice(order, "step-c")
	posD := indexOfInSlice(order, "step-d")

	assert.True(t, posA < posB && posA < posC, "A must come before B and C")
	assert.True(t, posB < posD && posC < posD, "B and C must come before D")
}

// TestContractFailureRetry tests retry behavior on contract validation failure (T049)
func TestContractFailureRetry(t *testing.T) {
	collector := newTestEventCollector()

	// Track retry attempts
	var attemptCount int32

	// Create an adapter that fails the first 2 attempts
	retryAdapter := &retryTrackingAdapter{
		attempts: &attemptCount,
		failUntil: 2,
		successAdapter: adapter.NewMockAdapter(
			adapter.WithStdoutJSON(`{"status": "success"}`),
			adapter.WithTokensUsed(1000),
		),
		failAdapter: adapter.NewMockAdapter(
			adapter.WithFailure(errors.New("contract validation failed")),
		),
	}

	executor := NewDefaultPipelineExecutor(retryAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := createTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "retry-test"},
		Steps: []Step{
			{
				ID:      "step-with-retry",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "test"},
				Handover: HandoverConfig{
					MaxRetries: 3,
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	// Verify retries occurred
	assert.Equal(t, int32(3), atomic.LoadInt32(&attemptCount), "should have 3 attempts (2 failures + 1 success)")

	// Check for retry events
	hasRetrying := collector.HasEventWithState("retrying")
	assert.True(t, hasRetrying, "should have retrying events")
}

// TestContractFailureExhaustsRetries tests that execution fails when retries are exhausted
func TestContractFailureExhaustsRetries(t *testing.T) {
	collector := newTestEventCollector()

	// Create an adapter that always fails
	failingAdapter := adapter.NewMockAdapter(
		adapter.WithFailure(errors.New("persistent failure")),
	)

	executor := NewDefaultPipelineExecutor(failingAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := createTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "exhausted-retry-test"},
		Steps: []Step{
			{
				ID:      "failing-step",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "test"},
				Handover: HandoverConfig{
					MaxRetries: 2,
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed")

	// Verify failure event was emitted
	hasFailed := collector.HasEventWithState("failed")
	assert.True(t, hasFailed, "should have failed event")
}

// TestProgressEventEmission tests that progress events are emitted during execution (T052)
func TestProgressEventEmission(t *testing.T) {
	collector := newTestEventCollector()
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(2500),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := createTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "progress-test"},
		Steps: []Step{
			{ID: "step1", Persona: "navigator", Exec: ExecConfig{Source: "test1"}},
			{ID: "step2", Persona: "navigator", Dependencies: []string{"step1"}, Exec: ExecConfig{Source: "test2"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	events := collector.GetEvents()

	// Verify pipeline-level events
	var pipelineStarted, pipelineCompleted bool
	for _, e := range events {
		if e.StepID == "" {
			if e.State == "started" {
				pipelineStarted = true
			}
			if e.State == "completed" {
				pipelineCompleted = true
			}
		}
	}
	assert.True(t, pipelineStarted, "pipeline started event should be emitted")
	assert.True(t, pipelineCompleted, "pipeline completed event should be emitted")

	// Verify step-level events
	step1Events := collector.GetEventsByStep("step1")
	step2Events := collector.GetEventsByStep("step2")

	assert.NotEmpty(t, step1Events, "step1 should have events")
	assert.NotEmpty(t, step2Events, "step2 should have events")

	// Check that running and completed events exist for each step
	hasStep1Running := false
	hasStep1Completed := false
	for _, e := range step1Events {
		if e.State == "running" {
			hasStep1Running = true
		}
		if e.State == "completed" {
			hasStep1Completed = true
		}
	}
	assert.True(t, hasStep1Running, "step1 should have running event")
	assert.True(t, hasStep1Completed, "step1 should have completed event")

	// Verify completed events include token usage
	for _, e := range events {
		if e.State == "completed" && e.StepID != "" {
			assert.Greater(t, e.TokensUsed, 0, "completed step event should include token count")
		}
	}
}

// TestProgressEventFields tests that progress events have correct field values
func TestProgressEventFields(t *testing.T) {
	collector := newTestEventCollector()
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(3000),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := createTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "event-fields-test"},
		Steps: []Step{
			{ID: "my-step", Persona: "craftsman", Exec: ExecConfig{Source: "test"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	events := collector.GetEvents()

	// Find the step completed event
	var completedEvent *event.Event
	for i := range events {
		if events[i].StepID == "my-step" && events[i].State == "completed" {
			completedEvent = &events[i]
			break
		}
	}

	require.NotNil(t, completedEvent, "should find step completed event")
	assert.Equal(t, "event-fields-test", completedEvent.PipelineID)
	assert.Equal(t, "my-step", completedEvent.StepID)
	assert.Equal(t, "craftsman", completedEvent.Persona)
	assert.Equal(t, 3000, completedEvent.TokensUsed)
	assert.GreaterOrEqual(t, completedEvent.DurationMs, int64(0), "duration should be non-negative")
}

// TestExecutorWithoutEmitter tests executor works without an emitter configured
func TestExecutorWithoutEmitter(t *testing.T) {
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
	)

	// Create executor without emitter
	executor := NewDefaultPipelineExecutor(mockAdapter)

	tmpDir := t.TempDir()
	m := createTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "no-emitter-test"},
		Steps: []Step{
			{ID: "step1", Persona: "navigator", Exec: ExecConfig{Source: "test"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Should not panic even without emitter
	err := executor.Execute(ctx, p, m, "test")
	assert.NoError(t, err)
}

// TestGetStatus tests the GetStatus method
func TestGetStatus(t *testing.T) {
	collector := newTestEventCollector()
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := createTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "status-test"},
		Steps: []Step{
			{ID: "step1", Persona: "navigator", Exec: ExecConfig{Source: "test"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	// Get status after execution
	status, err := executor.GetStatus("status-test")
	require.NoError(t, err)
	assert.Equal(t, "status-test", status.ID)
	assert.Equal(t, StateCompleted, status.State)
	assert.Contains(t, status.CompletedSteps, "step1")
	assert.Empty(t, status.FailedSteps)
	assert.NotNil(t, status.CompletedAt)

	// Test non-existent pipeline
	_, err = executor.GetStatus("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestDAGCycleDetection tests that cycles are detected and rejected
func TestDAGCycleDetection(t *testing.T) {
	collector := newTestEventCollector()
	mockAdapter := adapter.NewMockAdapter()

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := createTestManifest(tmpDir)

	// Create a pipeline with a cycle: A -> B -> C -> A
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "cycle-test"},
		Steps: []Step{
			{ID: "step-a", Persona: "navigator", Dependencies: []string{"step-c"}, Exec: ExecConfig{Source: "A"}},
			{ID: "step-b", Persona: "navigator", Dependencies: []string{"step-a"}, Exec: ExecConfig{Source: "B"}},
			{ID: "step-c", Persona: "navigator", Dependencies: []string{"step-b"}, Exec: ExecConfig{Source: "C"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cycle")
}

// TestMissingDependency tests that missing dependencies are caught
func TestMissingDependency(t *testing.T) {
	collector := newTestEventCollector()
	mockAdapter := adapter.NewMockAdapter()

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := createTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "missing-dep-test"},
		Steps: []Step{
			{ID: "step-a", Persona: "navigator", Dependencies: []string{"nonexistent"}, Exec: ExecConfig{Source: "A"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

// TestWorkspaceCreation tests that workspaces are created for each step
func TestWorkspaceCreation(t *testing.T) {
	collector := newTestEventCollector()
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := createTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "workspace-test"},
		Steps: []Step{
			{ID: "step1", Persona: "navigator", Exec: ExecConfig{Source: "test"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	// Verify workspace directory was created
	workspacePath := tmpDir + "/workspace-test/step1"
	_, err = os.Stat(workspacePath)
	assert.NoError(t, err, "workspace directory should exist")
}

// indexOfInSlice is a helper function to find index in slice
func indexOfInSlice(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}

// concurrencyTrackingAdapter wraps MockAdapter to track concurrent executions
type concurrencyTrackingAdapter struct {
	*adapter.MockAdapter
	onStart func()
	onEnd   func()
}

func (a *concurrencyTrackingAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	if a.onStart != nil {
		a.onStart()
	}
	defer func() {
		if a.onEnd != nil {
			a.onEnd()
		}
	}()
	return a.MockAdapter.Run(ctx, cfg)
}

// retryTrackingAdapter tracks retry attempts and can be configured to fail N times
type retryTrackingAdapter struct {
	attempts       *int32
	failUntil      int32
	successAdapter adapter.AdapterRunner
	failAdapter    adapter.AdapterRunner
}

func (a *retryTrackingAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	attempt := atomic.AddInt32(a.attempts, 1)
	if attempt <= a.failUntil {
		return a.failAdapter.Run(ctx, cfg)
	}
	return a.successAdapter.Run(ctx, cfg)
}
