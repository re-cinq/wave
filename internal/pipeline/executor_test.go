package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/adapter/adaptertest"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/skill"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStepOrdering verifies steps execute in topological order (T047)
func TestStepOrdering(t *testing.T) {
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
		adaptertest.WithTokensUsed(1000),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

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
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
		adaptertest.WithTokensUsed(500),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

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

// TestParallelStepExecution tests that independent steps actually run in parallel (T048)
func TestParallelStepExecution(t *testing.T) {
	collector := testutil.NewEventCollector()

	// Track concurrent execution
	var maxConcurrent int32
	var currentConcurrent int32

	// Create a mock adapter that tracks concurrency. The simulated delay
	// has to be long enough that both B and C are still running when the
	// other starts; 50ms was on the edge under CI load and produced
	// sporadic max-concurrent=1 false negatives. 500ms gives a 10x
	// margin against scheduler jitter while keeping wall time under 2s.
	concurrentAdapter := &concurrencyTrackingAdapter{
		MockAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(500),
			adaptertest.WithSimulatedDelay(500*time.Millisecond),
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
	m := testutil.CreateTestManifest(tmpDir)

	// Pipeline with independent steps B and C that run in parallel
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

	// Verify B and C actually ran concurrently
	assert.GreaterOrEqual(t, atomic.LoadInt32(&maxConcurrent), int32(2),
		"B and C should run concurrently (max concurrent >= 2)")

	// Verify ordering constraints: A before B,C and B,C before D
	order := collector.GetStepExecutionOrder()
	posA := indexOfInSlice(order, "step-a")
	posD := indexOfInSlice(order, "step-d")

	assert.True(t, posA >= 0, "step-a should be in execution order")
	assert.True(t, posD >= 0, "step-d should be in execution order")
	assert.True(t, posA < posD, "A must come before D")
}

// TestConcurrentStepFailure tests that when one concurrent step fails,
// the batch returns an error and other steps get cancelled via context.
func TestConcurrentStepFailure(t *testing.T) {
	collector := testutil.NewEventCollector()

	// Track which steps were started
	var startedSteps sync.Map

	// Create an adapter that fails for step-b but succeeds (slowly) for step-c
	failingConcurrentAdapter := &stepAwareAdapter{
		defaultAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(500),
			adaptertest.WithSimulatedDelay(200*time.Millisecond),
		),
		stepAdapters: map[string]adapter.AdapterRunner{
			"step-b": adaptertest.NewMockAdapter(
				adaptertest.WithFailure(errors.New("step-b intentional failure")),
			),
		},
		onStart: func(stepID string) {
			startedSteps.Store(stepID, true)
		},
	}

	executor := NewDefaultPipelineExecutor(failingConcurrentAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	// A -> (B, C) where B fails
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "concurrent-fail-test"},
		Steps: []Step{
			{ID: "step-a", Persona: "navigator", Exec: ExecConfig{Source: "A"}},
			{ID: "step-b", Persona: "navigator", Dependencies: []string{"step-a"}, Exec: ExecConfig{Source: "B"}},
			{ID: "step-c", Persona: "navigator", Dependencies: []string{"step-a"}, Exec: ExecConfig{Source: "C"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	assert.Error(t, err, "pipeline should fail when a concurrent step fails")

	var stepErr *StepExecutionError
	if errors.As(err, &stepErr) {
		assert.Equal(t, "step-b", stepErr.StepID, "failed step should be step-b")
	}

	// Verify failure event was emitted
	hasFailed := collector.HasEventWithState("failed")
	assert.True(t, hasFailed, "should have failed event")
}

// TestSingleStepBatchNoOverhead tests that pipelines with only sequential
// dependencies run through the single-step fast path (no goroutine overhead).
func TestSingleStepBatchNoOverhead(t *testing.T) {
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
		adaptertest.WithTokensUsed(500),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	// Linear pipeline: A -> B -> C (no parallelism possible)
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "sequential-test"},
		Steps: []Step{
			{ID: "step-a", Persona: "navigator", Exec: ExecConfig{Source: "A"}},
			{ID: "step-b", Persona: "navigator", Dependencies: []string{"step-a"}, Exec: ExecConfig{Source: "B"}},
			{ID: "step-c", Persona: "navigator", Dependencies: []string{"step-b"}, Exec: ExecConfig{Source: "C"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	// Verify all steps completed in order
	order := collector.GetStepExecutionOrder()
	require.Len(t, order, 3, "all 3 steps should have executed")

	posA := indexOfInSlice(order, "step-a")
	posB := indexOfInSlice(order, "step-b")
	posC := indexOfInSlice(order, "step-c")

	assert.True(t, posA < posB, "step-a should execute before step-b")
	assert.True(t, posB < posC, "step-b should execute before step-c")
}

// TestFailedStepAlwaysHasID ensures that StepExecutionError always carries the step ID,
// even when the step fails on a single-step batch (no concurrency).
func TestFailedStepAlwaysHasID(t *testing.T) {
	collector := testutil.NewEventCollector()
	failingAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithFailure(errors.New("simulated timeout")),
	)

	_ = NewDefaultPipelineExecutor(failingAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	// Linear pipeline: A -> B where B fails (single-step batch)
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "step-id-test"},
		Steps: []Step{
			{ID: "step-a", Persona: "navigator", Exec: ExecConfig{Source: "A"}},
			{ID: "step-b", Persona: "navigator", Dependencies: []string{"step-a"}, Exec: ExecConfig{Source: "B"}},
		},
	}

	// Make step-a succeed, step-b fail
	stepAdapter := &stepAwareAdapter{
		defaultAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		),
		stepAdapters: map[string]adapter.AdapterRunner{
			"step-b": failingAdapter,
		},
	}
	executor := NewDefaultPipelineExecutor(stepAdapter, WithEmitter(collector))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.Error(t, err)

	var stepErr *StepExecutionError
	require.True(t, errors.As(err, &stepErr), "error should be a StepExecutionError")
	assert.Equal(t, "step-b", stepErr.StepID, "StepExecutionError must carry the failed step ID")
	assert.NotEmpty(t, stepErr.StepID, "StepExecutionError.StepID must never be empty")
}

// TestConcurrentStepWideFanOut tests a wide fan-out pattern with many parallel steps.
func TestConcurrentStepWideFanOut(t *testing.T) {
	collector := testutil.NewEventCollector()

	var maxConcurrent int32
	var currentConcurrent int32

	concurrentAdapter := &concurrencyTrackingAdapter{
		MockAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
			adaptertest.WithSimulatedDelay(200*time.Millisecond),
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
	m := testutil.CreateTestManifest(tmpDir)

	// Root -> (B, C, D, E) -> Final
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "wide-fanout-test"},
		Steps: []Step{
			{ID: "root", Persona: "navigator", Exec: ExecConfig{Source: "root"}},
			{ID: "branch-b", Persona: "navigator", Dependencies: []string{"root"}, Exec: ExecConfig{Source: "B"}},
			{ID: "branch-c", Persona: "navigator", Dependencies: []string{"root"}, Exec: ExecConfig{Source: "C"}},
			{ID: "branch-d", Persona: "navigator", Dependencies: []string{"root"}, Exec: ExecConfig{Source: "D"}},
			{ID: "branch-e", Persona: "navigator", Dependencies: []string{"root"}, Exec: ExecConfig{Source: "E"}},
			{ID: "final", Persona: "navigator", Dependencies: []string{"branch-b", "branch-c", "branch-d", "branch-e"}, Exec: ExecConfig{Source: "final"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	// All 6 steps should complete
	events := collector.GetEvents()
	completedSteps := 0
	for _, e := range events {
		if e.State == "completed" && e.StepID != "" {
			completedSteps++
		}
	}
	assert.Equal(t, 6, completedSteps, "all 6 steps should complete")

	// B, C, D, E should have run concurrently (max concurrent >= 2 at minimum;
	// full overlap of 4 is ideal but depends on goroutine scheduling)
	assert.GreaterOrEqual(t, atomic.LoadInt32(&maxConcurrent), int32(2),
		"B, C, D, E should run concurrently (max concurrent >= 2)")
}

// stepAwareAdapter routes execution to different adapters based on step ID.
type stepAwareAdapter struct {
	defaultAdapter adapter.AdapterRunner
	stepAdapters   map[string]adapter.AdapterRunner
	onStart        func(stepID string)
}

func (a *stepAwareAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	// Extract step ID from the prompt or persona — we use the workspace path
	// which contains the step ID as the last path component
	stepID := filepath.Base(cfg.WorkspacePath)
	if a.onStart != nil {
		a.onStart(stepID)
	}
	if adapter, ok := a.stepAdapters[stepID]; ok {
		return adapter.Run(ctx, cfg)
	}
	return a.defaultAdapter.Run(ctx, cfg)
}

// TestContractFailureRetry tests retry behavior on contract validation failure (T049)
func TestContractFailureRetry(t *testing.T) {
	collector := testutil.NewEventCollector()

	// Track retry attempts
	var attemptCount int32

	// Create an adapter that fails the first 2 attempts
	retryAdapter := &retryTrackingAdapter{
		attempts:  &attemptCount,
		failUntil: 2,
		successAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(1000),
		),
		failAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithFailure(errors.New("contract validation failed")),
		),
	}

	executor := NewDefaultPipelineExecutor(retryAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "retry-test"},
		Steps: []Step{
			{
				ID:      "step-with-retry",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "test"},
				Retry: RetryConfig{
					MaxAttempts: 3,
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
	collector := testutil.NewEventCollector()

	// Create an adapter that always fails
	failingAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithFailure(errors.New("persistent failure")),
	)

	executor := NewDefaultPipelineExecutor(failingAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "exhausted-retry-test"},
		Steps: []Step{
			{
				ID:      "failing-step",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "test"},
				Retry: RetryConfig{
					MaxAttempts: 2,
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

// TestBuildContractPrompt_JSONSchema tests that contract compliance prompt is generated
// for json_schema contracts with output artifacts and required fields.
func TestBuildContractPrompt_JSONSchema(t *testing.T) {
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "test.schema.json")
	_ = os.WriteFile(schemaPath, []byte(`{"required": ["name", "status", "results"], "properties": {"name": {"type": "string"}, "status": {"type": "string"}, "results": {"type": "array"}}}`), 0644)

	executor := createSchemaTestExecutor(tmpDir)

	step := &Step{
		ID: "test-step",
		OutputArtifacts: []ArtifactDef{
			{Name: "output", Path: "artifact.json"},
		},
		Handover: HandoverConfig{
			Contract: ContractConfig{
				Type:       "json_schema",
				SchemaPath: schemaPath,
			},
		},
	}

	prompt := executor.buildContractPrompt(step, nil)

	assert.Contains(t, prompt, "Output Requirements")
	assert.Contains(t, prompt, "artifact.json")
	assert.Contains(t, prompt, "valid JSON")
	assert.Contains(t, prompt, "Contract Schema")
	assert.Contains(t, prompt, "`name`, `status`, `results`")
	assert.Contains(t, prompt, "Example structure")
}

// TestBuildContractPrompt_TestSuite tests contract prompt for test_suite contracts.
func TestBuildContractPrompt_TestSuite(t *testing.T) {
	tmpDir := t.TempDir()
	executor := createSchemaTestExecutor(tmpDir)

	step := &Step{
		ID: "test-step",
		Handover: HandoverConfig{
			Contract: ContractConfig{
				Type:    "test_suite",
				Command: "go test ./...",
			},
		},
	}

	prompt := executor.buildContractPrompt(step, nil)

	assert.Contains(t, prompt, "Test Validation")
	assert.Contains(t, prompt, "go test ./...")
	assert.Contains(t, prompt, "tests fail")
}

// TestBuildContractPrompt_NoContract tests that no prompt is generated when no contract exists.
func TestBuildContractPrompt_NoContract(t *testing.T) {
	tmpDir := t.TempDir()
	executor := createSchemaTestExecutor(tmpDir)

	step := &Step{ID: "test-step"}

	prompt := executor.buildContractPrompt(step, nil)
	assert.Empty(t, prompt)
}

// TestProgressEventEmission tests that progress events are emitted during execution (T052)
func TestProgressEventEmission(t *testing.T) {
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
		adaptertest.WithTokensUsed(2500),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

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
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
		adaptertest.WithTokensUsed(3000),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

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
	assert.True(t, strings.HasPrefix(completedEvent.PipelineID, "event-fields-test-"), "PipelineID should have name prefix with hash suffix")
	assert.Equal(t, "my-step", completedEvent.StepID)
	assert.Equal(t, "craftsman", completedEvent.Persona)
	assert.Equal(t, 3000, completedEvent.TokensUsed)
	assert.GreaterOrEqual(t, completedEvent.DurationMs, int64(0), "duration should be non-negative")
}

// TestExecutorWithoutEmitter tests executor works without an emitter configured
func TestExecutorWithoutEmitter(t *testing.T) {
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
	)

	// Create executor without emitter
	executor := NewDefaultPipelineExecutor(mockAdapter)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

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
	mockStore := testutil.NewMockStateStore()
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
		WithStateStore(mockStore),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

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
	runtimeID := collector.GetPipelineID()
	require.NotEmpty(t, runtimeID, "should have a pipeline ID from events")
	status, err := executor.GetStatus(runtimeID)
	require.NoError(t, err)
	assert.Equal(t, runtimeID, status.ID)
	assert.Equal(t, stateCompleted, status.State)
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
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter()

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

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
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter()

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

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
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

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
	runtimeID := collector.GetPipelineID()
	require.NotEmpty(t, runtimeID, "should have a pipeline ID from events")
	workspacePath := tmpDir + "/" + runtimeID + "/step1"
	_, err = os.Stat(workspacePath)
	assert.NoError(t, err, "workspace directory should exist")
}

// TestEmptyResultContentDoesNotOverwriteArtifacts is a regression test to ensure
// that when ResultContent is empty (due to relay compaction, parsing failures, etc),
// artifacts are not written with empty content, preserving any existing artifacts.
// This prevents the bug where artifacts get overwritten with empty content during
// token limit scenarios or adapter failures.
func TestEmptyResultContentDoesNotOverwriteArtifacts(t *testing.T) {
	tmpDir := t.TempDir()

	// Create existing artifact file with content
	artifactPath := tmpDir + "/workspace-test/step1/output.json"
	_ = os.MkdirAll(tmpDir+"/workspace-test/step1", 0755)
	existingContent := `{"previous": "step-result"}`
	err := os.WriteFile(artifactPath, []byte(existingContent), 0644)
	require.NoError(t, err)

	// Mock adapter that returns empty ResultContent (simulating parsing failure or compaction effect)
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"type": "result", "result": ""}`), // Empty result in JSON
		adaptertest.WithTokensUsed(1000),
	)

	collector := testutil.NewEventCollector()
	executor := NewDefaultPipelineExecutor(mockAdapter, WithEmitter(collector))

	m := testutil.CreateTestManifest(tmpDir)

	// Create pipeline with output artifact
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "artifact-test"},
		Steps: []Step{
			{
				ID:      "step1",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "generate output"},
				OutputArtifacts: []ArtifactDef{
					{Name: "result", Path: "output.json"},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = executor.Execute(ctx, p, m, "workspace-test")
	require.NoError(t, err)

	// Verify that existing artifact content is preserved (not overwritten with empty content)
	finalContent, err := os.ReadFile(artifactPath)
	require.NoError(t, err)
	assert.Equal(t, existingContent, string(finalContent),
		"Existing artifact content should be preserved when ResultContent is empty")
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
	*adaptertest.MockAdapter
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

// TestMemoryCleanupAfterCompletion tests that completed pipelines are cleaned up from memory
// to prevent memory leaks, but can still be retrieved via GetStatus from persistent storage.
func TestMemoryCleanupAfterCompletion(t *testing.T) {
	// Use a mock state store to test persistent storage fallback
	mockStore := testutil.NewMockStateStore()
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
		WithStateStore(mockStore),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "memory-cleanup-test"},
		Steps: []Step{
			{ID: "step1", Persona: "navigator", Exec: ExecConfig{Source: "test"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Execute the pipeline
	err := executor.Execute(ctx, p, m, "test input")
	require.NoError(t, err)

	// Verify pipeline is cleaned up from in-memory storage
	// (accessing the internal map to verify cleanup)
	runtimeID := collector.GetPipelineID()
	require.NotEmpty(t, runtimeID, "should have a pipeline ID from events")
	exec, ok := getExecutorPipeline(executor, runtimeID)
	assert.False(t, ok, "Pipeline should be cleaned up from in-memory storage after completion")
	assert.Nil(t, exec, "Pipeline execution should be nil after cleanup")

	// Verify GetStatus still works by querying persistent storage
	status, err := executor.GetStatus(runtimeID)
	require.NoError(t, err)
	assert.Equal(t, runtimeID, status.ID)
	assert.Equal(t, stateCompleted, status.State)
	assert.NotEmpty(t, status.CompletedSteps)
	assert.NotNil(t, status.CompletedAt)
}

// TestMemoryCleanupAfterFailure tests that failed pipelines are also cleaned up from memory.
func TestMemoryCleanupAfterFailure(t *testing.T) {
	mockStore := testutil.NewMockStateStore()
	collector := testutil.NewEventCollector()
	// Use a failing adapter
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithFailure(errors.New("step failure")),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
		WithStateStore(mockStore),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "memory-cleanup-fail-test"},
		Steps: []Step{
			{ID: "step1", Persona: "navigator", Exec: ExecConfig{Source: "test"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Execute the pipeline (should fail)
	err := executor.Execute(ctx, p, m, "test input")
	require.Error(t, err)

	// Verify pipeline is cleaned up from in-memory storage even after failure
	runtimeID := collector.GetPipelineID()
	require.NotEmpty(t, runtimeID, "should have a pipeline ID from events")
	exec, ok := getExecutorPipeline(executor, runtimeID)
	assert.False(t, ok, "Failed pipeline should be cleaned up from in-memory storage")
	assert.Nil(t, exec, "Failed pipeline execution should be nil after cleanup")

	// Verify GetStatus still works for failed pipeline
	status, err := executor.GetStatus(runtimeID)
	require.NoError(t, err)
	assert.Equal(t, runtimeID, status.ID)
	assert.Equal(t, stateFailed, status.State)
	assert.NotEmpty(t, status.FailedSteps)
}

// TestRegressionProductionIssues tests the specific production issues that were fixed:
// 1. Memory leaks from pipelines not being cleaned up
// 2. Empty input handling that caused template replacement issues
// 3. Nil pointer dereference in buildStepPrompt when Context is nil
func TestRegressionProductionIssues(t *testing.T) {
	t.Run("EmptyInputDoesNotCauseIssues", func(t *testing.T) {
		mockStore := testutil.NewMockStateStore()
		collector := testutil.NewEventCollector()
		mockAdapter := adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
		)

		executor := NewDefaultPipelineExecutor(mockAdapter,
			WithEmitter(collector),
			WithStateStore(mockStore),
		)

		tmpDir := t.TempDir()
		m := testutil.CreateTestManifest(tmpDir)

		p := &Pipeline{
			Metadata: PipelineMetadata{Name: "empty-input-test"},
			Steps: []Step{
				{
					ID:      "step1",
					Persona: "navigator",
					Exec:    ExecConfig{Source: "Process input: {{ input }} - should handle empty gracefully"},
				},
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Execute with empty input - this used to cause issues with template replacement
		err := executor.Execute(ctx, p, m, "")
		assert.NoError(t, err, "Empty input should be handled gracefully")

		// Verify pipeline was cleaned up from memory
		runtimeID := collector.GetPipelineID()
		require.NotEmpty(t, runtimeID, "should have a pipeline ID from events")
		exec, exists := getExecutorPipeline(executor, runtimeID)
		assert.False(t, exists, "Pipeline should be cleaned up from memory")
		assert.Nil(t, exec)

		// Verify status can still be retrieved from persistent storage
		status, err := executor.GetStatus(runtimeID)
		require.NoError(t, err)
		assert.Equal(t, stateCompleted, status.State)
	})

	t.Run("NilContextIsHandledDefensively", func(t *testing.T) {
		// Create a pipeline execution with nil context to test defensive handling
		mockAdapter := adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
		)

		executor := NewDefaultPipelineExecutor(mockAdapter)

		// Create execution without Context field (simulating the original bug)
		tmpDir := t.TempDir()
		m := testutil.CreateTestManifest(tmpDir)

		execution := &PipelineExecution{
			Pipeline:       &Pipeline{Metadata: PipelineMetadata{Name: "nil-context-test"}},
			Manifest:       m,
			States:         make(map[string]string),
			Results:        make(map[string]map[string]interface{}),
			WorkspacePaths: make(map[string]string),
			WorktreePaths:  make(map[string]*WorktreeInfo),
			Input:          "test input",
			Status:         &PipelineStatus{ID: "nil-context-test", PipelineName: "nil-context-test"},
			// Context: nil  // Deliberately omitted to test nil handling
		}

		step := &Step{
			ID:      "test-step",
			Persona: "navigator",
			Exec:    ExecConfig{Source: "Test prompt with {{ input }}"},
		}

		// This used to panic with nil pointer dereference
		// The buildStepPrompt function should handle nil context gracefully
		assert.NotPanics(t, func() {
			// Call buildStepPrompt directly to test the defensive fix
			prompt := executor.buildStepPrompt(execution, step)
			assert.Contains(t, prompt, "test input", "Input should still be replaced even with nil context")
		}, "Should not panic with nil context")
	})

	t.Run("MatrixExecutorContextPropagation", func(t *testing.T) {
		// Test that matrix executor properly propagates context to worker executions
		mockAdapter := adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
		)

		executor := NewDefaultPipelineExecutor(mockAdapter)
		matrixExecutor := NewMatrixExecutor(executor)

		tmpDir := t.TempDir()

		// Create items file for matrix execution
		items := []map[string]interface{}{
			{"id": 1, "name": "item1"},
		}
		itemsJSON, _ := json.Marshal(items)
		itemsFile := filepath.Join(tmpDir, "items.json")
		_ = os.WriteFile(itemsFile, itemsJSON, 0644)

		m := testutil.CreateTestManifest(tmpDir)

		execution := &PipelineExecution{
			Pipeline:       &Pipeline{Metadata: PipelineMetadata{Name: "matrix-context-test"}},
			Manifest:       m,
			States:         make(map[string]string),
			Results:        make(map[string]map[string]interface{}),
			ArtifactPaths:  make(map[string]string),
			WorkspacePaths: make(map[string]string),
			WorktreePaths:  make(map[string]*WorktreeInfo),
			Input:          "test input",
			Context:        NewPipelineContext("matrix-context-test", "matrix-context-test", "matrix-step"), // Proper context
			Status:         &PipelineStatus{ID: "matrix-context-test", PipelineName: "matrix-context-test"},
		}

		step := &Step{
			ID:      "matrix-step",
			Persona: "navigator",
			Strategy: &MatrixStrategy{
				Type:        "matrix",
				ItemsSource: itemsFile,
			},
			Exec: ExecConfig{Source: "Process {{ input }} for item {{ item.name }}"},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// This used to panic due to missing Context in worker executions
		assert.NotPanics(t, func() {
			err := matrixExecutor.Execute(ctx, execution, step)
			assert.NoError(t, err, "Matrix execution should succeed with proper context propagation")
		}, "Matrix execution should not panic with proper context propagation")
	})
}

// TestNilStatusHandlingInTests tests that test code handles nil status properly
// This is a regression test for a bug where test code didn't check for nil status
// after GetStatus returned an error, causing a panic when accessing status.CompletedSteps.
func TestNilStatusHandlingInTests(t *testing.T) {
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithFailure(errors.New("simulated failure")),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "status-handling-test"},
		Steps: []Step{
			{ID: "step1", Persona: "navigator", Exec: ExecConfig{Source: "test"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Execute pipeline that will fail
	err := executor.Execute(ctx, p, m, "test")
	require.Error(t, err, "Pipeline should fail due to mock adapter failure")

	// Try to get status - should return error because pipeline was cleaned up after failure
	status, err := executor.GetStatus("status-handling-test")
	if err != nil {
		// This is expected behavior - when an error occurs, we should handle it gracefully
		// and NOT try to access status fields when status is nil
		assert.Nil(t, status, "Status should be nil when GetStatus returns an error")
		return // Test passes - this is the expected path
	}

	// If we somehow get a status back, it should be valid
	if status != nil {
		assert.Equal(t, stateFailed, status.State)
		assert.NotEmpty(t, status.FailedSteps)
	}
}

// TestWriteOutputArtifactsPreservesExistingFiles verifies that when a persona writes an artifact
// file during execution, writeOutputArtifacts does not overwrite it with ResultContent.
func TestWriteOutputArtifactsPreservesExistingFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create existing artifact file with persona-written content
	artifactDir := filepath.Join(tmpDir, "workspace-test", "step1", "output")
	_ = os.MkdirAll(artifactDir, 0755)
	artifactPath := filepath.Join(artifactDir, "issue-content.json")
	personaContent := `{"issue": "structured data from persona"}`
	err := os.WriteFile(artifactPath, []byte(personaContent), 0644)
	require.NoError(t, err)

	// Mock adapter returns non-empty ResultContent (conversational prose)
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"type": "result", "result": "I analyzed the issue and wrote the file."}`),
		adaptertest.WithTokensUsed(1000),
	)

	collector := testutil.NewEventCollector()
	executor := NewDefaultPipelineExecutor(mockAdapter, WithEmitter(collector))

	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "preserve-artifact-test"},
		Steps: []Step{
			{
				ID:      "step1",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "generate output"},
				OutputArtifacts: []ArtifactDef{
					{Name: "issue-content", Path: ".agents/output/issue-content.json"},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = executor.Execute(ctx, p, m, "workspace-test")
	require.NoError(t, err)

	// Verify that persona-written content is preserved, not overwritten with ResultContent
	finalContent, err := os.ReadFile(artifactPath)
	require.NoError(t, err)
	assert.Equal(t, personaContent, string(finalContent),
		"Persona-written artifact should be preserved when file already exists")
}

// TestCommandStepOutputArtifactsRegisteredForInjection is a regression test for
// #1490. A `type: command` step that writes a file declared in
// `output_artifacts` must register the file path in
// `execution.ArtifactPaths[step.ID+":"+art.Name]` so a downstream step's
// `memory.inject_artifacts` lookup resolves to actual content rather than
// silently falling through to the (usually empty) stdout fallback.
func TestCommandStepOutputArtifactsRegisteredForInjection(t *testing.T) {
	tmpDir := t.TempDir()

	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"type": "result", "result": "ok"}`),
		adaptertest.WithTokensUsed(10),
	)
	collector := testutil.NewEventCollector()
	executor := NewDefaultPipelineExecutor(mockAdapter, WithEmitter(collector))

	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "command-output-artifact-test"},
		Steps: []Step{
			{
				ID:     "produce",
				Type:   StepTypeCommand,
				Script: `mkdir -p .agents/output && printf '{"items":[{"x":1}]}' > .agents/output/data.json`,
				OutputArtifacts: []ArtifactDef{
					{Name: "data", Path: ".agents/output/data.json", Type: "json"},
				},
			},
			{
				ID:           "consume",
				Persona:      "navigator",
				Dependencies: []string{"produce"},
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{
						{Step: "produce", Artifact: "data", As: "data"},
					},
				},
				Exec: ExecConfig{Source: "consume artifact"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "workspace-cmd-art")
	require.NoError(t, err)

	// The injected artifact should be the JSON the command wrote, not a
	// 0-byte file from the stdout fallback path. Walk the tmpDir for the
	// `data` injection target — its exact location depends on the
	// workspace-creation strategy in effect.
	var injectedPath string
	walkErr := filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filepath.Base(path) == "data" && strings.Contains(path, filepath.Join("consume", ".agents", "artifacts")) {
			injectedPath = path
		}
		return nil
	})
	require.NoError(t, walkErr)
	require.NotEmpty(t, injectedPath, "injected artifact must exist somewhere under %s", tmpDir)

	stat, err := os.Stat(injectedPath)
	require.NoError(t, err)
	assert.Greater(t, stat.Size(), int64(0),
		"injected artifact must be non-empty — see #1490")

	content, err := os.ReadFile(injectedPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), `"items"`,
		"injected content must match what the command wrote, not stdout fallback")
}

// configCapturingAdapter wraps MockAdapter and captures the AdapterRunConfig passed to Run
type configCapturingAdapter struct {
	*adaptertest.MockAdapter
	mu         sync.Mutex
	lastConfig adapter.AdapterRunConfig
}

func (a *configCapturingAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	a.mu.Lock()
	a.lastConfig = cfg
	a.mu.Unlock()
	return a.MockAdapter.Run(ctx, cfg)
}

func (a *configCapturingAdapter) getLastConfig() adapter.AdapterRunConfig {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.lastConfig
}

// TestExecuteStep_NonZeroExitCode_EmitsWarning verifies that a non-zero adapter exit code
// emits a warning event but still allows the step to complete (work may have been done).
func TestExecuteStep_NonZeroExitCode_EmitsWarning(t *testing.T) {
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithExitCode(1),
		adaptertest.WithTokensUsed(100),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "exit-code-test"},
		Steps: []Step{
			{ID: "crash-step", Persona: "navigator", Exec: ExecConfig{Source: "do something"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err, "non-zero exit code should warn, not fail the step")

	// Should have a warning event about the exit code
	events := collector.GetEvents()
	var hasWarning, hasCompleted bool
	for _, e := range events {
		if e.StepID == "crash-step" && e.State == "warning" {
			assert.Contains(t, e.Message, "adapter exited with code 1")
			hasWarning = true
		}
		if e.State == "completed" && e.StepID == "" {
			hasCompleted = true
		}
	}
	assert.True(t, hasWarning, "should emit a warning event for non-zero exit code")
	assert.True(t, hasCompleted, "step should still complete despite non-zero exit code")
}

// TestExecuteStep_NonZeroExitCode_ContinuesSubsequentSteps verifies that when a step
// exits with a non-zero code, subsequent steps still execute (work may have been done).
func TestExecuteStep_NonZeroExitCode_ContinuesSubsequentSteps(t *testing.T) {
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithExitCode(1),
		adaptertest.WithTokensUsed(100),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "exit-code-chain-test"},
		Steps: []Step{
			{ID: "step-a", Persona: "navigator", Exec: ExecConfig{Source: "first"}},
			{ID: "step-b", Persona: "navigator", Dependencies: []string{"step-a"}, Exec: ExecConfig{Source: "second"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err, "pipeline should complete despite non-zero exit codes")

	// Both steps should have executed
	order := collector.GetStepExecutionOrder()
	assert.Equal(t, []string{"step-a", "step-b"}, order, "both steps should execute")

	// Both steps should have warning events
	stepAEvents := collector.GetEventsByStep("step-a")
	stepBEvents := collector.GetEventsByStep("step-b")

	var aWarned, bWarned bool
	for _, e := range stepAEvents {
		if e.State == "warning" {
			aWarned = true
		}
	}
	for _, e := range stepBEvents {
		if e.State == "warning" {
			bWarned = true
		}
	}
	assert.True(t, aWarned, "step-a should have a warning event")
	assert.True(t, bWarned, "step-b should have a warning event")
}

// streamEventAdapter wraps MockAdapter and fires OnStreamEvent callbacks before delegating Run.
// This lets us test the stream-activity event bridge in the executor.
type streamEventAdapter struct {
	*adaptertest.MockAdapter
	streamEvents []adapter.StreamEvent
}

func (a *streamEventAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	// Fire each pre-configured stream event through the callback, if set
	if cfg.OnStreamEvent != nil {
		for _, evt := range a.streamEvents {
			cfg.OnStreamEvent(evt)
		}
	}
	return a.MockAdapter.Run(ctx, cfg)
}

// TestStreamActivityEventBridge verifies that the OnStreamEvent callback in the executor
// correctly emits pipeline-enriched stream_activity events for valid tool_use events,
// and silently ignores non-tool_use events and tool_use events with empty ToolName.
func TestStreamActivityEventBridge(t *testing.T) {
	collector := testutil.NewEventCollector()

	// Configure three stream events:
	// 1. Valid tool_use with ToolName and ToolInput -> SHOULD emit stream_activity
	// 2. Non-tool_use event (type "text") -> should NOT emit stream_activity
	// 3. tool_use with empty ToolName -> should NOT emit stream_activity
	streamAdapter := &streamEventAdapter{
		MockAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(500),
		),
		streamEvents: []adapter.StreamEvent{
			{
				Type:      "tool_use",
				ToolName:  "Write",
				ToolInput: "/workspace/output.json",
			},
			{
				Type:    "text",
				Content: "Here is some reasoning text",
			},
			{
				Type:      "tool_use",
				ToolName:  "",
				ToolInput: "should-be-ignored",
			},
		},
	}

	executor := NewDefaultPipelineExecutor(streamAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "stream-bridge-test"},
		Steps: []Step{
			{ID: "stream-step", Persona: "craftsman", Exec: ExecConfig{Source: "do work"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	// Collect all stream_activity events
	allEvents := collector.GetEvents()
	var streamActivityEvents []event.Event
	for _, e := range allEvents {
		if e.State == event.StateStreamActivity {
			streamActivityEvents = append(streamActivityEvents, e)
		}
	}

	// Exactly one stream_activity event should have been emitted (the valid tool_use)
	require.Len(t, streamActivityEvents, 1,
		"exactly one stream_activity event expected (valid tool_use); got %d", len(streamActivityEvents))

	sa := streamActivityEvents[0]

	// Verify pipeline-enriched fields
	assert.True(t, strings.HasPrefix(sa.PipelineID, "stream-bridge-test-"), "PipelineID should have name prefix with hash suffix")
	assert.Equal(t, "stream-step", sa.StepID, "StepID should match")
	assert.Equal(t, "craftsman", sa.Persona, "Persona should match the step persona")
	assert.Equal(t, "Write", sa.ToolName, "ToolName should be the tool from the stream event")
	assert.Equal(t, "/workspace/output.json", sa.ToolTarget, "ToolTarget should be the tool input")
	assert.False(t, sa.Timestamp.IsZero(), "Timestamp should be set")

	// Double-check: no stream_activity events for the text or empty-ToolName cases
	for _, e := range allEvents {
		if e.State == event.StateStreamActivity {
			assert.NotEmpty(t, e.ToolName, "stream_activity events must have non-empty ToolName")
		}
	}
}

func TestCreateStepWorkspace_Ref(t *testing.T) {
	executor := NewDefaultPipelineExecutor(&adaptertest.MockAdapter{})
	m := &manifest.Manifest{}
	tmpDir := t.TempDir()

	// Simulate a prior step that created a workspace
	execution := &PipelineExecution{
		Pipeline:       &Pipeline{Metadata: PipelineMetadata{Name: "test-ref"}},
		Manifest:       m,
		WorkspacePaths: map[string]string{"specify": tmpDir},
		WorktreePaths:  make(map[string]*WorktreeInfo),
		Status:         &PipelineStatus{ID: "test-ref-abc"},
	}

	// Step that references specify's workspace
	step := &Step{
		ID:        "implement",
		Workspace: WorkspaceConfig{Ref: "specify"},
	}

	wsPath, err := executor.createStepWorkspace(execution, step)
	require.NoError(t, err)
	assert.Equal(t, tmpDir, wsPath, "ref workspace should return referenced step's path")
}

func TestCreateStepWorkspace_RefMissing(t *testing.T) {
	executor := NewDefaultPipelineExecutor(&adaptertest.MockAdapter{})
	m := &manifest.Manifest{}

	execution := &PipelineExecution{
		Pipeline:       &Pipeline{Metadata: PipelineMetadata{Name: "test-ref"}},
		Manifest:       m,
		WorkspacePaths: map[string]string{}, // no prior workspaces
		WorktreePaths:  make(map[string]*WorktreeInfo),
		Status:         &PipelineStatus{ID: "test-ref-abc"},
	}

	step := &Step{
		ID:        "implement",
		Workspace: WorkspaceConfig{Ref: "specify"},
	}

	_, err := executor.createStepWorkspace(execution, step)
	assert.Error(t, err, "should error when referenced workspace doesn't exist")
	assert.Contains(t, err.Error(), "specify")
}

func TestCreateStepWorkspace_SharedWorktree(t *testing.T) {
	// Test that two steps with the same branch reuse the same worktree path
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
	)
	executor := NewDefaultPipelineExecutor(mockAdapter)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	execution := &PipelineExecution{
		Pipeline:       &Pipeline{Metadata: PipelineMetadata{Name: "shared-wt-test"}},
		Manifest:       m,
		States:         make(map[string]string),
		Results:        make(map[string]map[string]interface{}),
		ArtifactPaths:  make(map[string]string),
		WorkspacePaths: make(map[string]string),
		WorktreePaths:  make(map[string]*WorktreeInfo),
		Input:          "test",
		Context:        NewPipelineContext("shared-wt-test", "shared-wt-test", "step1"),
		Status:         &PipelineStatus{ID: "shared-wt-test", PipelineName: "shared-wt-test"},
	}

	// Pre-populate WorktreePaths to simulate a previously created worktree
	branch := "feature/test-branch"
	expectedPath := "/tmp/test-worktree-path"
	expectedRepoRoot := "/tmp/test-repo-root"
	execution.WorktreePaths[branch] = &WorktreeInfo{
		AbsPath:  expectedPath,
		RepoRoot: expectedRepoRoot,
	}

	step1 := &Step{
		ID:      "step1",
		Persona: "navigator",
		Workspace: WorkspaceConfig{
			Type:   "worktree",
			Branch: branch,
		},
	}

	step2 := &Step{
		ID:      "step2",
		Persona: "craftsman",
		Workspace: WorkspaceConfig{
			Type:   "worktree",
			Branch: branch,
		},
	}

	// Both steps should return the cached path without creating new worktrees
	path1, err := executor.createStepWorkspace(execution, step1)
	assert.NoError(t, err)
	assert.Equal(t, expectedPath, path1)
	assert.Equal(t, expectedRepoRoot, execution.WorkspacePaths["step1__worktree_repo_root"])

	path2, err := executor.createStepWorkspace(execution, step2)
	assert.NoError(t, err)
	assert.Equal(t, expectedPath, path2)
	assert.Equal(t, expectedRepoRoot, execution.WorkspacePaths["step2__worktree_repo_root"])

	// Both should point to the same worktree
	assert.Equal(t, path1, path2, "Steps with the same branch should share the same worktree")
}

func TestCreateStepWorkspace_DifferentBranches(t *testing.T) {
	// Test that two steps with different branches get separate worktree entries
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
	)
	executor := NewDefaultPipelineExecutor(mockAdapter)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	execution := &PipelineExecution{
		Pipeline:       &Pipeline{Metadata: PipelineMetadata{Name: "diff-branch-test"}},
		Manifest:       m,
		States:         make(map[string]string),
		Results:        make(map[string]map[string]interface{}),
		ArtifactPaths:  make(map[string]string),
		WorkspacePaths: make(map[string]string),
		WorktreePaths:  make(map[string]*WorktreeInfo),
		Input:          "test",
		Context:        NewPipelineContext("diff-branch-test", "diff-branch-test", "step1"),
		Status:         &PipelineStatus{ID: "diff-branch-test", PipelineName: "diff-branch-test"},
	}

	// Pre-populate two different branches
	execution.WorktreePaths["branch-a"] = &WorktreeInfo{
		AbsPath:  "/tmp/worktree-a",
		RepoRoot: "/tmp/repo",
	}
	execution.WorktreePaths["branch-b"] = &WorktreeInfo{
		AbsPath:  "/tmp/worktree-b",
		RepoRoot: "/tmp/repo",
	}

	stepA := &Step{
		ID:      "step-a",
		Persona: "navigator",
		Workspace: WorkspaceConfig{
			Type:   "worktree",
			Branch: "branch-a",
		},
	}

	stepB := &Step{
		ID:      "step-b",
		Persona: "craftsman",
		Workspace: WorkspaceConfig{
			Type:   "worktree",
			Branch: "branch-b",
		},
	}

	pathA, err := executor.createStepWorkspace(execution, stepA)
	assert.NoError(t, err)
	assert.Equal(t, "/tmp/worktree-a", pathA)

	pathB, err := executor.createStepWorkspace(execution, stepB)
	assert.NoError(t, err)
	assert.Equal(t, "/tmp/worktree-b", pathB)

	assert.NotEqual(t, pathA, pathB, "Different branches should get different worktree paths")
}

func TestCleanupWorktrees_Dedup(t *testing.T) {
	// Test that shared worktree paths are only cleaned up once
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
	)
	executor := NewDefaultPipelineExecutor(mockAdapter)

	sharedPath := "/tmp/shared-worktree"
	repoRoot := "/tmp/test-repo"

	execution := &PipelineExecution{
		Pipeline:      &Pipeline{Metadata: PipelineMetadata{Name: "dedup-test"}},
		States:        make(map[string]string),
		Results:       make(map[string]map[string]interface{}),
		ArtifactPaths: make(map[string]string),
		WorkspacePaths: map[string]string{
			"step1":                     sharedPath,
			"step1__worktree_repo_root": repoRoot,
			"step2":                     sharedPath,
			"step2__worktree_repo_root": repoRoot,
			"step3":                     sharedPath,
			"step3__worktree_repo_root": repoRoot,
		},
		WorktreePaths: make(map[string]*WorktreeInfo),
		Input:         "test",
		Status:        &PipelineStatus{ID: "dedup-test", PipelineName: "dedup-test"},
	}

	// cleanupWorktrees should not panic even though all steps share the same path
	// It will try to create a worktree manager for a non-existent repo root,
	// but the important thing is it only attempts cleanup once per unique path
	assert.NotPanics(t, func() {
		executor.cleanupWorktrees(execution, "dedup-test")
	}, "Cleanup should not panic with shared worktree paths")
}

// getExecutorPipeline is a helper function to access the internal pipelines map for testing
func getExecutorPipeline(executor PipelineExecutor, pipelineID string) (*PipelineExecution, bool) {
	if defaultExec, ok := executor.(*DefaultPipelineExecutor); ok {
		defaultExec.mu.RLock()
		defer defaultExec.mu.RUnlock()
		exec, exists := defaultExec.pipelines[pipelineID]
		return exec, exists
	}
	return nil, false
}

// TestStdoutArtifactCapture tests that stdout artifacts are correctly captured and available to downstream steps
func TestStdoutArtifactCapture(t *testing.T) {
	collector := testutil.NewEventCollector()
	stdoutContent := `{"analysis": "test analysis data", "score": 42}`
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(stdoutContent),
		adaptertest.WithTokensUsed(100),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	// Pipeline with stdout artifact
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "stdout-artifact-test"},
		Steps: []Step{
			{
				ID:      "analyze",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "analyze data"},
				OutputArtifacts: []ArtifactDef{
					{Name: "analysis-report", Source: "stdout", Type: "json"},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	// Verify artifact was written to correct location
	// Workspace path is: wsRoot/pipelineID/stepID
	// Artifact path is: workspace/.agents/artifacts/stepID/artifactName
	pipelineID := collector.GetPipelineID()
	artifactPath := filepath.Join(tmpDir, pipelineID, "analyze", ".agents", "artifacts", "analyze", "analysis-report")

	// Check artifact exists
	_, err = os.Stat(artifactPath)
	assert.NoError(t, err, "stdout artifact should be written to filesystem")

	// Read content and verify
	content, err := os.ReadFile(artifactPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "analysis")
}

// TestStdoutArtifactSizeLimitEnforced tests that size limit is enforced for stdout artifacts
func TestStdoutArtifactSizeLimitEnforced(t *testing.T) {
	collector := testutil.NewEventCollector()
	// Create a large stdout (over 10MB would be too slow, so we'll configure a smaller limit)
	largeContent := strings.Repeat("x", 1000)
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(largeContent),
		adaptertest.WithTokensUsed(100),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)
	// Set a very small limit to test the enforcement
	m.Runtime.Artifacts.MaxStdoutSize = 100 // 100 bytes

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "size-limit-test"},
		Steps: []Step{
			{
				ID:      "produce",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "produce output"},
				OutputArtifacts: []ArtifactDef{
					{Name: "large-output", Source: "stdout", Type: "text"},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	// Should fail due to size limit
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds limit")
}

// TestStdoutArtifactWrittenToCorrectLocation tests that stdout artifact paths follow the expected convention
func TestStdoutArtifactWrittenToCorrectLocation(t *testing.T) {
	collector := testutil.NewEventCollector()
	expectedContent := "test content for stdout artifact"
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(expectedContent),
		adaptertest.WithTokensUsed(100),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "stdout-path-test"},
		Steps: []Step{
			{
				ID:      "produce",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "produce content"},
				OutputArtifacts: []ArtifactDef{
					{Name: "my-artifact", Source: "stdout", Type: "text"},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	// Verify artifact exists at the correct path
	// Workspace path is: wsRoot/pipelineID/stepID
	// Artifact path is: workspace/.agents/artifacts/stepID/artifactName
	pipelineID := collector.GetPipelineID()
	artifactPath := filepath.Join(tmpDir, pipelineID, "produce", ".agents", "artifacts", "produce", "my-artifact")

	info, err := os.Stat(artifactPath)
	require.NoError(t, err, "stdout artifact should exist at expected path")
	assert.True(t, info.Size() > 0, "stdout artifact should have content")
}

// TestMissingRequiredArtifactFailsBeforeStep tests that missing required artifacts fail before step execution
func TestMissingRequiredArtifactFailsBeforeStep(t *testing.T) {
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
		adaptertest.WithTokensUsed(100),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	// Pipeline where step2 references a non-existent artifact from a step that doesn't exist
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "missing-artifact-test"},
		Steps: []Step{
			{
				ID:      "step1",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "do work"},
			},
			{
				ID:           "step2",
				Persona:      "navigator",
				Dependencies: []string{"step1"},
				Exec:         ExecConfig{Source: "consume artifact"},
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{
						// Reference a step that doesn't exist - this should fail clearly
						{Step: "nonexistent-step", Artifact: "missing-artifact", As: "data"},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required artifact")
	assert.Contains(t, err.Error(), "not found")
}

// TestOptionalMissingArtifactProceeds tests that optional missing artifacts don't fail the step
func TestOptionalMissingArtifactProceeds(t *testing.T) {
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
		adaptertest.WithTokensUsed(100),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	// Pipeline where step2 references an optional non-existent artifact
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "optional-artifact-test"},
		Steps: []Step{
			{
				ID:      "step1",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "do work"},
			},
			{
				ID:           "step2",
				Persona:      "navigator",
				Dependencies: []string{"step1"},
				Exec:         ExecConfig{Source: "consume artifact"},
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{
						{Step: "step1", Artifact: "optional-artifact", As: "data", Optional: true},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	// Should succeed despite missing optional artifact
	require.NoError(t, err)

	// Verify step2 completed
	events := collector.GetEventsByStep("step2")
	hasCompleted := false
	for _, e := range events {
		if e.State == "completed" {
			hasCompleted = true
			break
		}
	}
	assert.True(t, hasCompleted, "step2 should have completed despite optional artifact missing")
}

// TestTypeMismatchFailsWithClearError tests that type mismatch produces a clear error
func TestTypeMismatchFailsWithClearError(t *testing.T) {
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
		adaptertest.WithTokensUsed(100),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	// Pipeline where step2 expects json but step1 produces markdown
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "type-mismatch-test"},
		Steps: []Step{
			{
				ID:      "step1",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "produce markdown"},
				OutputArtifacts: []ArtifactDef{
					{Name: "output", Path: ".agents/output.md", Type: "markdown"},
				},
			},
			{
				ID:           "step2",
				Persona:      "navigator",
				Dependencies: []string{"step1"},
				Exec:         ExecConfig{Source: "consume as json"},
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{
						{Step: "step1", Artifact: "output", As: "data", Type: "json"},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "type mismatch")
	assert.Contains(t, err.Error(), "expected json")
	assert.Contains(t, err.Error(), "got markdown")
}

// TestTypeNotDeclaredSkipsValidation tests that missing type declaration skips validation
func TestTypeNotDeclaredSkipsValidation(t *testing.T) {
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
		adaptertest.WithTokensUsed(100),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	// Pipeline where neither side declares a type - should pass
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "no-type-test"},
		Steps: []Step{
			{
				ID:      "step1",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "produce output"},
				OutputArtifacts: []ArtifactDef{
					{Name: "output", Path: ".agents/output.txt"}, // No type
				},
			},
			{
				ID:           "step2",
				Persona:      "navigator",
				Dependencies: []string{"step1"},
				Exec:         ExecConfig{Source: "consume output"},
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{
						{Step: "step1", Artifact: "output", As: "data"}, // No type
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	// Should succeed since no type validation is performed
	require.NoError(t, err)
}

// TestOutcomeExtractionRegistersDeliverables verifies that step outcomes declared in
// pipeline YAML are extracted from JSON artifacts and registered with the deliverable tracker.
func TestOutcomeExtractionRegistersDeliverables(t *testing.T) {
	collector := testutil.NewEventCollector()

	artifactJSON := `{"comment_url": "https://github.com/re-cinq/wave/pull/42#issuecomment-999", "pr": "42"}`
	outcomeAdapter := &outcomeTestAdapter{
		MockAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		),
		artifactJSON: artifactJSON,
	}

	executor := NewDefaultPipelineExecutor(outcomeAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "outcome-test"},
		Steps: []Step{
			{
				ID:      "publish",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "post review"},
				OutputArtifacts: []ArtifactDef{
					{Name: "publish-result", Path: "output/publish-result.json", Type: "json"},
				},
				Outcomes: []OutcomeDef{
					{
						Type:        "url",
						ExtractFrom: "output/publish-result.json",
						JSONPath:    ".comment_url",
						Label:       "Review Comment",
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	// The outcome extraction should have registered a URL deliverable
	tracker := executor.GetOutcomeTracker()
	require.NotNil(t, tracker)

	urls := tracker.GetByType(state.OutcomeTypeURL)
	require.Len(t, urls, 1, "should have 1 URL outcome registered")
	assert.Equal(t, "https://github.com/re-cinq/wave/pull/42#issuecomment-999", urls[0].Value)
	assert.Equal(t, "Review Comment", urls[0].Label)

	// Verify outcome event was emitted
	pipelineID := collector.GetPipelineID()
	events := collector.GetEvents()
	var hasOutcomeEvent bool
	for _, e := range events {
		if e.PipelineID == pipelineID && strings.Contains(e.Message, "outcome:") {
			hasOutcomeEvent = true
			break
		}
	}
	assert.True(t, hasOutcomeEvent, "should emit outcome extraction event")
}

// TestOutcomeExtractionMissingFileWarns verifies that missing artifact files produce
// warnings but don't fail the step.
func TestOutcomeExtractionMissingFileWarns(t *testing.T) {
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
		adaptertest.WithTokensUsed(100),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "outcome-missing-test"},
		Steps: []Step{
			{
				ID:      "publish",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "post review"},
				Outcomes: []OutcomeDef{
					{
						Type:        "pr",
						ExtractFrom: "output/nonexistent.json",
						JSONPath:    ".pr_url",
						Label:       "Pull Request",
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Should complete successfully despite missing artifact
	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	// Should have a warning event about the missing file
	events := collector.GetEvents()
	var hasWarning bool
	for _, e := range events {
		if e.State == "warning" && strings.Contains(e.Message, "outcome:") {
			hasWarning = true
			break
		}
	}
	assert.True(t, hasWarning, "should emit warning for missing outcome artifact")
}

// TestOutcomeExtractionPRType verifies PR outcomes are registered as PR deliverables
func TestOutcomeExtractionPRType(t *testing.T) {
	collector := testutil.NewEventCollector()

	prJSON := `{"pr_url": "https://github.com/re-cinq/wave/pull/99", "title": "feat: add feature"}`
	outcomeAdapter := &outcomeTestAdapter{
		MockAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		),
		artifactJSON: prJSON,
	}

	executor := NewDefaultPipelineExecutor(outcomeAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "outcome-pr-test"},
		Steps: []Step{
			{
				ID:      "create-pr",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "create pr"},
				OutputArtifacts: []ArtifactDef{
					{Name: "pr-result", Path: ".agents/output/pr-result.json", Type: "json"},
				},
				Outcomes: []OutcomeDef{
					{
						Type:        "pr",
						ExtractFrom: ".agents/output/pr-result.json",
						JSONPath:    ".pr_url",
						Label:       "Pull Request",
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	tracker := executor.GetOutcomeTracker()
	prs := tracker.GetByType(state.OutcomeTypePR)
	require.Len(t, prs, 1, "should have 1 PR outcome")
	assert.Equal(t, "https://github.com/re-cinq/wave/pull/99", prs[0].Value)
	assert.Equal(t, "Pull Request", prs[0].Label)
}

// TestOutcomeExtractionIssueType verifies issue outcomes are registered as issue deliverables.
func TestOutcomeExtractionIssueType(t *testing.T) {
	collector := testutil.NewEventCollector()

	issueJSON := `{"issue_url": "https://github.com/re-cinq/wave/issues/55"}`
	outcomeAdapter := &outcomeTestAdapter{
		MockAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		),
		artifactJSON: issueJSON,
	}

	executor := NewDefaultPipelineExecutor(outcomeAdapter, WithEmitter(collector))
	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "outcome-issue-test"},
		Steps: []Step{{
			ID: "report", Persona: "navigator",
			Exec:            ExecConfig{Source: "report issue"},
			OutputArtifacts: []ArtifactDef{{Name: "issue-result", Path: "output/issue-result.json", Type: "json"}},
			Outcomes: []OutcomeDef{{
				Type: "issue", ExtractFrom: "output/issue-result.json",
				JSONPath: ".issue_url", Label: "Bug Report",
			}},
		}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	require.NoError(t, executor.Execute(ctx, p, m, "test"))

	tracker := executor.GetOutcomeTracker()
	issues := tracker.GetByType(state.OutcomeTypeIssue)
	require.Len(t, issues, 1, "should have 1 issue outcome")
	assert.Equal(t, "https://github.com/re-cinq/wave/issues/55", issues[0].Value)
	assert.Equal(t, "Bug Report", issues[0].Label)
}

// TestOutcomeExtractionDeploymentType verifies deployment outcomes are registered as deployment deliverables.
func TestOutcomeExtractionDeploymentType(t *testing.T) {
	collector := testutil.NewEventCollector()

	deployJSON := `{"deploy_url": "https://staging.example.com"}`
	outcomeAdapter := &outcomeTestAdapter{
		MockAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		),
		artifactJSON: deployJSON,
	}

	executor := NewDefaultPipelineExecutor(outcomeAdapter, WithEmitter(collector))
	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "outcome-deploy-test"},
		Steps: []Step{{
			ID: "deploy", Persona: "navigator",
			Exec:            ExecConfig{Source: "deploy"},
			OutputArtifacts: []ArtifactDef{{Name: "deploy-result", Path: "output/deploy-result.json", Type: "json"}},
			Outcomes: []OutcomeDef{{
				Type: "deployment", ExtractFrom: "output/deploy-result.json",
				JSONPath: ".deploy_url", Label: "Staging Deploy",
			}},
		}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	require.NoError(t, executor.Execute(ctx, p, m, "test"))

	tracker := executor.GetOutcomeTracker()
	deploys := tracker.GetByType(state.OutcomeTypeDeployment)
	require.Len(t, deploys, 1, "should have 1 deployment outcome")
	assert.Equal(t, "https://staging.example.com", deploys[0].Value)
	assert.Equal(t, "Staging Deploy", deploys[0].Label)
}

// TestOutcomeExtractionUnknownTypeFallsBackToURL verifies that unrecognized outcome types
// fall back to URL deliverables.
func TestOutcomeExtractionUnknownTypeFallsBackToURL(t *testing.T) {
	collector := testutil.NewEventCollector()

	artifactJSON := `{"link": "https://example.com/report"}`
	outcomeAdapter := &outcomeTestAdapter{
		MockAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		),
		artifactJSON: artifactJSON,
	}

	executor := NewDefaultPipelineExecutor(outcomeAdapter, WithEmitter(collector))
	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "outcome-unknown-test"},
		Steps: []Step{{
			ID: "publish", Persona: "navigator",
			Exec:            ExecConfig{Source: "publish"},
			OutputArtifacts: []ArtifactDef{{Name: "publish-result", Path: "output/publish-result.json", Type: "json"}},
			Outcomes: []OutcomeDef{{
				Type: "unknown-type", ExtractFrom: "output/publish-result.json",
				JSONPath: ".link", Label: "Report Link",
			}},
		}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	require.NoError(t, executor.Execute(ctx, p, m, "test"))

	tracker := executor.GetOutcomeTracker()
	urls := tracker.GetByType(state.OutcomeTypeURL)
	require.Len(t, urls, 1, "unknown type should fall back to URL")
	assert.Equal(t, "https://example.com/report", urls[0].Value)
}

// TestOutcomeExtractionPathTraversal verifies that extract_from paths that escape the
// workspace are rejected with a warning.
func TestOutcomeExtractionPathTraversal(t *testing.T) {
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
		adaptertest.WithTokensUsed(100),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter, WithEmitter(collector))
	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "outcome-traversal-test"},
		Steps: []Step{{
			ID: "evil", Persona: "navigator",
			Exec: ExecConfig{Source: "do work"},
			Outcomes: []OutcomeDef{{
				Type: "url", ExtractFrom: "../../../etc/passwd",
				JSONPath: ".url",
			}},
		}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	require.NoError(t, executor.Execute(ctx, p, m, "test"))

	// Should have a warning about path escaping workspace
	events := collector.GetEvents()
	var hasTraversalWarning bool
	for _, e := range events {
		if e.State == "warning" && strings.Contains(e.Message, "escapes workspace") {
			hasTraversalWarning = true
			break
		}
	}
	assert.True(t, hasTraversalWarning, "should emit warning for path traversal attempt")
}

// TestOutcomeExtractionInvalidJSONPath verifies that an invalid JSON path produces a warning.
func TestOutcomeExtractionInvalidJSONPath(t *testing.T) {
	collector := testutil.NewEventCollector()

	artifactJSON := `{"url": "https://example.com"}`
	outcomeAdapter := &outcomeTestAdapter{
		MockAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		),
		artifactJSON: artifactJSON,
	}

	executor := NewDefaultPipelineExecutor(outcomeAdapter, WithEmitter(collector))
	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "outcome-badpath-test"},
		Steps: []Step{{
			ID: "publish", Persona: "navigator",
			Exec:            ExecConfig{Source: "publish"},
			OutputArtifacts: []ArtifactDef{{Name: "publish-result", Path: "output/publish-result.json", Type: "json"}},
			Outcomes: []OutcomeDef{{
				Type: "url", ExtractFrom: "output/publish-result.json",
				JSONPath: ".nonexistent.deep.path", Label: "Missing",
			}},
		}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	require.NoError(t, executor.Execute(ctx, p, m, "test"))

	// Should have a warning about JSON path extraction failure
	events := collector.GetEvents()
	var hasPathWarning bool
	for _, e := range events {
		if e.State == "warning" && strings.Contains(e.Message, "outcome:") {
			hasPathWarning = true
			break
		}
	}
	assert.True(t, hasPathWarning, "should emit warning for invalid JSON path")
}

// TestOutcomeDefValidation tests the OutcomeDef.Validate method.
func TestOutcomeDefValidation(t *testing.T) {
	tests := []struct {
		name    string
		outcome OutcomeDef
		wantErr string
	}{
		{name: "valid pr", outcome: OutcomeDef{Type: "pr", ExtractFrom: "out.json", JSONPath: ".url"}},
		{name: "valid issue", outcome: OutcomeDef{Type: "issue", ExtractFrom: "out.json", JSONPath: ".url"}},
		{name: "valid url", outcome: OutcomeDef{Type: "url", ExtractFrom: "out.json", JSONPath: ".url"}},
		{name: "valid deployment", outcome: OutcomeDef{Type: "deployment", ExtractFrom: "out.json", JSONPath: ".url"}},
		{name: "missing type", outcome: OutcomeDef{ExtractFrom: "out.json", JSONPath: ".url"}, wantErr: "type is required"},
		{name: "unknown type", outcome: OutcomeDef{Type: "comment", ExtractFrom: "out.json", JSONPath: ".url"}, wantErr: "unknown type"},
		{name: "missing extract_from", outcome: OutcomeDef{Type: "pr", JSONPath: ".url"}, wantErr: "extract_from is required"},
		{name: "missing json_path", outcome: OutcomeDef{Type: "pr", ExtractFrom: "out.json"}, wantErr: "json_path is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.outcome.Validate("test-step", 0)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

// outcomeTestAdapter wraps MockAdapter and writes an artifact JSON file during execution
// so that outcome extraction can find it afterward.
type outcomeTestAdapter struct {
	*adaptertest.MockAdapter
	artifactJSON string
}

func (a *outcomeTestAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	// Write the artifact file to the workspace so outcome extraction can read it
	// We need to find and write all output artifact paths
	if a.artifactJSON != "" && cfg.WorkspacePath != "" {
		// Write to common output locations
		for _, dir := range []string{"output", ".agents/output"} {
			outDir := filepath.Join(cfg.WorkspacePath, dir)
			_ = os.MkdirAll(outDir, 0755)
			// Write all JSON files in this directory
			entries, _ := filepath.Glob(filepath.Join(outDir, "*.json"))
			if len(entries) == 0 {
				// Pre-create common artifact files
				_ = os.WriteFile(filepath.Join(outDir, "publish-result.json"), []byte(a.artifactJSON), 0644)
				_ = os.WriteFile(filepath.Join(outDir, "pr-result.json"), []byte(a.artifactJSON), 0644)
				_ = os.WriteFile(filepath.Join(outDir, "issue-result.json"), []byte(a.artifactJSON), 0644)
				_ = os.WriteFile(filepath.Join(outDir, "deploy-result.json"), []byte(a.artifactJSON), 0644)
			}
		}
	}
	return a.MockAdapter.Run(ctx, cfg)
}

// TestOutcomeExtractionEmptyArrayFriendlyMessage verifies that when an outcome
// json_path indexes into an empty array, the system produces a friendly warning
// in the summary (via the tracker) but does NOT emit a real-time warning event.
func TestOutcomeExtractionEmptyArrayFriendlyMessage(t *testing.T) {
	collector := testutil.NewEventCollector()

	// Artifact contains an empty array — a valid "no results" condition
	artifactJSON := `{"enhanced_issues": []}`
	outcomeAdapter := &outcomeTestAdapter{
		MockAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		),
		artifactJSON: artifactJSON,
	}

	executor := NewDefaultPipelineExecutor(outcomeAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "outcome-empty-array-test"},
		Steps: []Step{
			{
				ID:      "apply-enhancements",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "apply enhancements"},
				OutputArtifacts: []ArtifactDef{
					{Name: "publish-result", Path: "output/publish-result.json", Type: "json"},
				},
				Outcomes: []OutcomeDef{
					{
						Type:        "url",
						ExtractFrom: "output/publish-result.json",
						JSONPath:    ".enhanced_issues[0].url",
						Label:       "Enhanced Issue",
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	// The tracker should have a friendly warning message about empty array
	tracker := executor.GetOutcomeTracker()
	require.NotNil(t, tracker)
	warnings := tracker.OutcomeWarnings()
	require.Len(t, warnings, 1, "should have 1 outcome warning for empty array")
	assert.Contains(t, warnings[0], "no items in enhanced_issues")
	assert.Contains(t, warnings[0], "skipping")

	// Crucially: no real-time warning event should have been emitted for this case.
	// The warning appears only in the summary via the tracker.
	pipelineID := collector.GetPipelineID()
	events := collector.GetEvents()
	for _, e := range events {
		if e.PipelineID == pipelineID && e.StepID == "apply-enhancements" && e.State == "warning" {
			if strings.Contains(e.Message, "outcome:") && strings.Contains(e.Message, "enhanced_issues") {
				t.Errorf("should NOT emit real-time warning event for empty array, but got: %s", e.Message)
			}
		}
	}
}

// TestOutcomeExtractionNonEmptyArrayOOBStillEmitsWarning verifies that a genuine
// out-of-bounds error (non-empty array) still emits both a tracker warning AND
// a real-time warning event.
func TestOutcomeExtractionNonEmptyArrayOOBStillEmitsWarning(t *testing.T) {
	collector := testutil.NewEventCollector()

	// Array has 1 element but the outcome path asks for index 5
	artifactJSON := `{"items": [{"url": "https://example.com"}]}`
	outcomeAdapter := &outcomeTestAdapter{
		MockAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		),
		artifactJSON: artifactJSON,
	}

	executor := NewDefaultPipelineExecutor(outcomeAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "outcome-oob-test"},
		Steps: []Step{
			{
				ID:      "apply",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "apply"},
				OutputArtifacts: []ArtifactDef{
					{Name: "publish-result", Path: "output/publish-result.json", Type: "json"},
				},
				Outcomes: []OutcomeDef{
					{
						Type:        "url",
						ExtractFrom: "output/publish-result.json",
						JSONPath:    ".items[5].url",
						Label:       "Item URL",
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	// Tracker should have a warning with the technical error message
	tracker := executor.GetOutcomeTracker()
	require.NotNil(t, tracker)
	warnings := tracker.OutcomeWarnings()
	require.Len(t, warnings, 1, "should have 1 outcome warning for OOB")
	assert.Contains(t, warnings[0], "array index 5 out of bounds")

	// A real-time warning event SHOULD have been emitted for non-empty-array OOB
	pipelineID := collector.GetPipelineID()
	events := collector.GetEvents()
	var hasRealtimeWarning bool
	for _, e := range events {
		if e.PipelineID == pipelineID && e.StepID == "apply" && e.State == "warning" {
			if strings.Contains(e.Message, "outcome:") && strings.Contains(e.Message, "array index 5 out of bounds") {
				hasRealtimeWarning = true
				break
			}
		}
	}
	assert.True(t, hasRealtimeWarning, "should emit real-time warning for non-empty-array OOB error")
}

// modelCapturingAdapter captures the AdapterRunConfig.Model for each step execution.
type modelCapturingAdapter struct {
	mu     sync.Mutex
	models map[string]string // stepID -> model
	inner  adapter.AdapterRunner
}

func newModelCapturingAdapter() *modelCapturingAdapter {
	return &modelCapturingAdapter{
		models: make(map[string]string),
		inner: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		),
	}
}

func (a *modelCapturingAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	stepID := filepath.Base(cfg.WorkspacePath)
	a.mu.Lock()
	a.models[stepID] = cfg.Model
	a.mu.Unlock()
	return a.inner.Run(ctx, cfg)
}

func (a *modelCapturingAdapter) getModel(stepID string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.models[stepID]
}

// TestWithModelOverrideOption verifies that the WithModelOverride option sets the field
func TestWithModelOverrideOption(t *testing.T) {
	mockAdapter := adaptertest.NewMockAdapter()
	executor := NewDefaultPipelineExecutor(mockAdapter, WithModelOverride("haiku"))
	assert.Equal(t, "haiku", executor.modelOverride)
}

// TestWithModelOverrideEmpty verifies that empty string override is not set
func TestWithModelOverrideEmpty(t *testing.T) {
	mockAdapter := adaptertest.NewMockAdapter()
	executor := NewDefaultPipelineExecutor(mockAdapter, WithModelOverride(""))
	assert.Equal(t, "", executor.modelOverride)
}

// TestModelOverridePrecedence tests the three-tier model precedence logic
func TestModelOverridePrecedence(t *testing.T) {
	tests := []struct {
		name          string
		personaModel  string
		modelOverride string
		expectedModel string
	}{
		{
			name:          "override applied when persona has no model",
			personaModel:  "",
			modelOverride: "haiku",
			expectedModel: "haiku",
		},
		{
			name:          "CLI override takes precedence over persona pinning",
			personaModel:  "opus",
			modelOverride: "haiku",
			expectedModel: "haiku",
		},
		{
			name:          "no override and no persona model yields empty",
			personaModel:  "",
			modelOverride: "",
			expectedModel: "",
		},
		{
			name:          "persona model used when no override",
			personaModel:  "sonnet",
			modelOverride: "",
			expectedModel: "sonnet",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			capturer := newModelCapturingAdapter()

			var opts []ExecutorOption
			opts = append(opts, WithEmitter(testutil.NewEventCollector()))
			if tc.modelOverride != "" {
				opts = append(opts, WithModelOverride(tc.modelOverride))
			}

			executor := NewDefaultPipelineExecutor(capturer, opts...)

			tmpDir := t.TempDir()
			m := &manifest.Manifest{
				Metadata: manifest.Metadata{Name: "model-test"},
				Adapters: map[string]manifest.Adapter{
					"claude": {Binary: "claude", Mode: "headless"},
				},
				Personas: map[string]manifest.Persona{
					"test-persona": {
						Adapter:     "claude",
						Temperature: 0.7,
						Model:       tc.personaModel,
					},
				},
				Runtime: manifest.Runtime{
					WorkspaceRoot:     tmpDir,
					DefaultTimeoutMin: 5,
				},
			}

			p := &Pipeline{
				Metadata: PipelineMetadata{Name: "model-precedence"},
				Steps: []Step{
					{ID: "step-1", Persona: "test-persona", Exec: ExecConfig{Source: "test"}},
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			err := executor.Execute(ctx, p, m, "test")
			require.NoError(t, err)

			// The model capturing adapter records the model from AdapterRunConfig
			capturedModel := capturer.getModel("step-1")
			assert.Equal(t, tc.expectedModel, capturedModel,
				"expected model %q but got %q", tc.expectedModel, capturedModel)
		})
	}
}

// TestModelOverrideInChildExecutor verifies that NewChildExecutor inherits modelOverride
func TestModelOverrideInChildExecutor(t *testing.T) {
	mockAdapter := adaptertest.NewMockAdapter()
	parent := NewDefaultPipelineExecutor(mockAdapter, WithModelOverride("haiku"))

	child := parent.NewChildExecutor()
	assert.Equal(t, "haiku", child.modelOverride, "child executor should inherit modelOverride")
}

// TestModelOverrideIntegration is an integration test verifying the model string
// reaches AdapterRunConfig.Model through the full execution path
func TestModelOverrideIntegration(t *testing.T) {
	capturer := newModelCapturingAdapter()
	collector := testutil.NewEventCollector()

	executor := NewDefaultPipelineExecutor(capturer,
		WithEmitter(collector),
		WithModelOverride("haiku"),
	)

	tmpDir := t.TempDir()
	m := &manifest.Manifest{
		Metadata: manifest.Metadata{Name: "integration-model-test"},
		Adapters: map[string]manifest.Adapter{
			"claude": {Binary: "claude", Mode: "headless"},
		},
		Personas: map[string]manifest.Persona{
			"navigator": {
				Adapter:     "claude",
				Temperature: 0.1,
				// No model set — should use override
			},
			"craftsman": {
				Adapter:     "claude",
				Temperature: 0.7,
				Model:       "opus", // Pinned model — should NOT be overridden
			},
		},
		Runtime: manifest.Runtime{
			WorkspaceRoot:     tmpDir,
			DefaultTimeoutMin: 5,
		},
	}

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "integration-model"},
		Steps: []Step{
			{ID: "navigate", Persona: "navigator", Exec: ExecConfig{Source: "navigate"}},
			{ID: "implement", Persona: "craftsman", Dependencies: []string{"navigate"}, Exec: ExecConfig{Source: "implement"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test integration")
	require.NoError(t, err)

	// Navigator has no model pinned — should use CLI override "haiku"
	assert.Equal(t, "haiku", capturer.getModel("navigate"),
		"unpinned persona should use CLI model override")

	// Craftsman has model pinned to "opus" — CLI override takes precedence
	assert.Equal(t, "haiku", capturer.getModel("implement"),
		"CLI model override should take precedence over persona pinning")
}

// TestResolveModelMethod tests the resolveModel method directly
func TestResolveModelMethod(t *testing.T) {
	executor := &DefaultPipelineExecutor{modelOverride: "haiku"}

	// Persona with no model — use override
	p1 := &manifest.Persona{Model: ""}
	assert.Equal(t, "haiku", executor.resolveModel(nil, p1, nil, "navigator", nil))

	// Persona with pinned model — CLI override still wins
	p2 := &manifest.Persona{Model: "opus"}
	assert.Equal(t, "haiku", executor.resolveModel(nil, p2, nil, "navigator", nil))

	// No override, no persona model — empty
	executor2 := &DefaultPipelineExecutor{modelOverride: ""}
	p3 := &manifest.Persona{Model: ""}
	assert.Equal(t, "", executor2.resolveModel(nil, p3, nil, "navigator", nil))
}

func TestResolveModel_ForceModelEmpty(t *testing.T) {
	// forceModel=true but modelOverride="" — should fall through to step/persona/auto-route
	executor := &DefaultPipelineExecutor{forceModel: true, modelOverride: ""}
	persona := &manifest.Persona{Model: "opus"}

	got := executor.resolveModel(nil, persona, nil, "craftsman", nil)
	// forceModel with empty override does NOT return early — falls through
	// persona model "opus" is not a tier, so returned as-is
	assert.Equal(t, "opus", got)
}

func TestResolveModel_ForceModelWithValue(t *testing.T) {
	executor := &DefaultPipelineExecutor{forceModel: true, modelOverride: "claude-haiku-4-5"}
	persona := &manifest.Persona{Model: "opus"}

	got := executor.resolveModel(nil, persona, nil, "craftsman", nil)
	assert.Equal(t, "claude-haiku-4-5", got)
}

func TestResolveModel_StepModelBalancedTier(t *testing.T) {
	// step.Model="balanced" with nil routing — ResolveComplexityModel returns "" for balanced
	executor := &DefaultPipelineExecutor{}
	persona := &manifest.Persona{}
	step := &Step{Model: "balanced"}

	got := executor.resolveModel(step, persona, nil, "test", nil)
	// resolveTierModel("balanced", nil, nil) → routing.ResolveComplexityModel("balanced") → ""
	// isTier=true, so returns ""
	assert.Equal(t, "", got)
}

func TestResolveModel_StepModelCheapestTier(t *testing.T) {
	executor := &DefaultPipelineExecutor{}
	persona := &manifest.Persona{}
	step := &Step{Model: "cheapest"}

	got := executor.resolveModel(step, persona, nil, "test", nil)
	assert.Equal(t, "claude-haiku-4-5", got)
}

func TestResolveModel_StepModelStrongestTier(t *testing.T) {
	executor := &DefaultPipelineExecutor{}
	persona := &manifest.Persona{}
	step := &Step{Model: "strongest"}

	got := executor.resolveModel(step, persona, nil, "test", nil)
	assert.Equal(t, "claude-opus-4", got)
}

func TestResolveModel_StepModelLiteral(t *testing.T) {
	executor := &DefaultPipelineExecutor{}
	persona := &manifest.Persona{}
	step := &Step{Model: "claude-sonnet-4"}

	got := executor.resolveModel(step, persona, nil, "test", nil)
	assert.Equal(t, "claude-sonnet-4", got)
}

func TestResolveModel_PersonaModelTier(t *testing.T) {
	executor := &DefaultPipelineExecutor{}
	persona := &manifest.Persona{Model: "cheapest"}

	got := executor.resolveModel(nil, persona, nil, "navigator", nil)
	assert.Equal(t, "claude-haiku-4-5", got)
}

func TestResolveModel_AutoRoute_BalancedReturnsEmpty(t *testing.T) {
	// auto_route=true, no step/persona model, ClassifyStepComplexity returns balanced for generic persona
	executor := &DefaultPipelineExecutor{}
	persona := &manifest.Persona{}
	step := &Step{ID: "generic-step"}
	routing := &manifest.RoutingConfig{AutoRoute: true}

	got := executor.resolveModel(step, persona, routing, "generic", nil)
	// ClassifyStepComplexity returns "balanced" for generic persona
	// routing.ResolveComplexityModel("balanced") returns "" (balanced maps to empty)
	// model == "" → falls through, returns ""
	assert.Equal(t, "", got)
}

func TestResolveModel_AutoRoute_CheapestReturnsModel(t *testing.T) {
	executor := &DefaultPipelineExecutor{}
	persona := &manifest.Persona{}
	step := &Step{ID: "navigate", Type: StepTypeCommand}
	routing := &manifest.RoutingConfig{AutoRoute: true}

	got := executor.resolveModel(step, persona, routing, "navigator", nil)
	// ClassifyStepComplexity: step type "command" → cheapest
	// routing.ResolveComplexityModel("cheapest") → "claude-haiku-4-5"
	assert.Equal(t, "claude-haiku-4-5", got)
}

func TestResolveModel_CLITierVsStepTier_CheaperWins(t *testing.T) {
	// CLI override is "cheapest", step model is "strongest" → cheapest wins
	executor := &DefaultPipelineExecutor{modelOverride: "cheapest"}
	persona := &manifest.Persona{}
	step := &Step{Model: "strongest"}

	got := executor.resolveModel(step, persona, nil, "test", nil)
	// Both are tiers, CheaperTier("cheapest","strongest") → "cheapest"
	// resolveTierModel("cheapest", nil, nil) → "claude-haiku-4-5"
	assert.Equal(t, "claude-haiku-4-5", got)
}

func TestResolveModel_CLITierVsStepTier_BalancedResolvesEmpty(t *testing.T) {
	// CLI override is "balanced", step model is "strongest" → balanced wins (cheaper)
	// But balanced resolves to empty string
	executor := &DefaultPipelineExecutor{modelOverride: "balanced"}
	persona := &manifest.Persona{}
	step := &Step{Model: "strongest"}

	got := executor.resolveModel(step, persona, nil, "test", nil)
	// CheaperTier("balanced","strongest") → "balanced"
	// resolveTierModel("balanced", nil, nil) → "", isTier=true
	assert.Equal(t, "", got)
}

func TestResolveModel_CLILiteralOverridesStepTier(t *testing.T) {
	// CLI is a literal model name, step has a tier — CLI literal wins
	executor := &DefaultPipelineExecutor{modelOverride: "claude-sonnet-4"}
	persona := &manifest.Persona{}
	step := &Step{Model: "strongest"}

	got := executor.resolveModel(step, persona, nil, "test", nil)
	// TierRank("claude-sonnet-4") = -1 (not a tier), so CLI literal wins
	assert.Equal(t, "claude-sonnet-4", got)
}

func TestResolveTierModel(t *testing.T) {
	tests := []struct {
		name              string
		model             string
		routing           *manifest.RoutingConfig
		adapterTierModels map[string]string
		want              string
		isTier            bool
	}{
		{
			name:              "cheapest tier nil routing",
			model:             "cheapest",
			routing:           nil,
			adapterTierModels: nil,
			want:              "claude-haiku-4-5",
			isTier:            true,
		},
		{
			name:              "balanced tier nil routing returns empty",
			model:             "balanced",
			routing:           nil,
			adapterTierModels: nil,
			want:              "",
			isTier:            true,
		},
		{
			name:              "strongest tier nil routing",
			model:             "strongest",
			routing:           nil,
			adapterTierModels: nil,
			want:              "claude-opus-4",
			isTier:            true,
		},
		{
			name:              "literal model not a tier",
			model:             "claude-sonnet-4",
			routing:           nil,
			adapterTierModels: nil,
			want:              "",
			isTier:            false,
		},
		{
			name:  "cheapest tier with custom routing",
			model: "cheapest",
			routing: &manifest.RoutingConfig{
				ComplexityMap: map[string]string{
					"cheapest": "my-custom-model",
				},
			},
			adapterTierModels: nil,
			want:              "my-custom-model",
			isTier:            true,
		},
		{
			name:  "adapter tier_models takes priority over routing",
			model: "cheapest",
			routing: &manifest.RoutingConfig{
				ComplexityMap: map[string]string{
					"cheapest": "routing-model",
				},
			},
			adapterTierModels: map[string]string{
				"cheapest": "adapter-model",
			},
			want:   "adapter-model",
			isTier: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, isTier := resolveTierModel(tt.model, tt.routing, tt.adapterTierModels)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.isTier, isTier)
		})
	}
}

// cancellableMockStore embeds testutil.MockStateStore and adds configurable CheckCancellation.
type cancellableMockStore struct {
	testutil.MockStateStore
	mu        sync.Mutex
	cancelled bool
}

func (c *cancellableMockStore) CheckCancellation(runID string) (*state.CancellationRecord, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cancelled {
		return &state.CancellationRecord{RunID: runID, RequestedAt: time.Now()}, nil
	}
	return nil, nil
}

func (c *cancellableMockStore) setCancelled() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cancelled = true
}

func TestPollCancellation_CancelsContext(t *testing.T) {
	store := &cancellableMockStore{}
	executor := &DefaultPipelineExecutor{store: store}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start polling
	go executor.pollCancellation(ctx, "test-run", cancel)

	// Context should still be active
	assert.NoError(t, ctx.Err())

	// Trigger cancellation in the DB
	store.setCancelled()

	// Wait for the poller to pick it up (polls every 2s, give it 5s)
	deadline := time.After(5 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("pollCancellation did not cancel context within 5s")
		default:
			if ctx.Err() != nil {
				// Success — context was cancelled by the poller
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func TestPollCancellation_StopsWhenContextCancelled(t *testing.T) {
	store := &cancellableMockStore{}
	executor := &DefaultPipelineExecutor{store: store}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		executor.pollCancellation(ctx, "test-run", cancel)
		close(done)
	}()

	// Cancel the context externally
	cancel()

	// Goroutine should exit promptly
	select {
	case <-done:
		// Good — goroutine exited
	case <-time.After(3 * time.Second):
		t.Fatal("pollCancellation did not exit after context cancellation")
	}
}

// countingFailAdapter fails the first N calls then succeeds.
type countingFailAdapter struct {
	mu          sync.Mutex
	failCount   int // how many calls should fail
	callCount   int
	failError   error
	successMock *adaptertest.MockAdapter
	lastConfigs []adapter.AdapterRunConfig
}

func newCountingFailAdapter(failCount int, failErr error) *countingFailAdapter {
	return &countingFailAdapter{
		failCount:   failCount,
		failError:   failErr,
		successMock: adaptertest.NewMockAdapter(adaptertest.WithStdoutJSON(`{"status":"ok"}`)),
	}
}

func (a *countingFailAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	a.mu.Lock()
	a.callCount++
	call := a.callCount
	a.lastConfigs = append(a.lastConfigs, cfg)
	a.mu.Unlock()
	if call <= a.failCount {
		return nil, a.failError
	}
	return a.successMock.Run(ctx, cfg)
}

func (a *countingFailAdapter) getCallCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.callCount
}

func (a *countingFailAdapter) getLastConfigs() []adapter.AdapterRunConfig {
	a.mu.Lock()
	defer a.mu.Unlock()
	dst := make([]adapter.AdapterRunConfig, len(a.lastConfigs))
	copy(dst, a.lastConfigs)
	return dst
}

// attemptTrackingStore extends testutil.MockStateStore to track RecordStepAttempt calls.
type attemptTrackingStore struct {
	*testutil.MockStateStore
	mu       sync.Mutex
	attempts []state.StepAttemptRecord
}

func newAttemptTrackingStore() *attemptTrackingStore {
	return &attemptTrackingStore{
		MockStateStore: testutil.NewMockStateStore(),
	}
}

func (s *attemptTrackingStore) RecordStepAttempt(record *state.StepAttemptRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attempts = append(s.attempts, *record)
	return nil
}

func (s *attemptTrackingStore) getAttempts() []state.StepAttemptRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	dst := make([]state.StepAttemptRecord, len(s.attempts))
	copy(dst, s.attempts)
	return dst
}

// TestExecuteStep_RetryConfig_MaxAttempts verifies that the retry count is respected.
func TestExecuteStep_RetryConfig_MaxAttempts(t *testing.T) {
	failAdapter := newCountingFailAdapter(2, errors.New("step failure"))
	collector := testutil.NewEventCollector()
	store := newAttemptTrackingStore()

	executor := NewDefaultPipelineExecutor(failAdapter,
		WithEmitter(collector),
		WithStateStore(store),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "retry-test"},
		Steps: []Step{
			{
				ID:      "step-1",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "do something"},
				Retry: RetryConfig{
					MaxAttempts: 3,
					BaseDelay:   "1ms", // fast for tests
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test input")
	assert.NoError(t, err, "should succeed on third attempt")
	assert.Equal(t, 3, failAdapter.getCallCount(), "adapter should have been called 3 times")

	// Verify attempt records
	attempts := store.getAttempts()
	// We expect: running(1), failed(1), running(2), failed(2), running(3), succeeded(3)
	assert.GreaterOrEqual(t, len(attempts), 3, "should have at least 3 attempt records")
}

// TestExecuteStep_RetryConfig_OnFailureSkip verifies that on_failure=skip skips the step.
func TestExecuteStep_RetryConfig_OnFailureSkip(t *testing.T) {
	failAdapter := newCountingFailAdapter(5, errors.New("always fails"))
	collector := testutil.NewEventCollector()

	executor := NewDefaultPipelineExecutor(failAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "skip-test"},
		Steps: []Step{
			{
				ID:      "step-1",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "do something"},
				Retry: RetryConfig{
					MaxAttempts: 2,
					BaseDelay:   "1ms",
					OnFailure:   "skip",
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test input")
	assert.NoError(t, err, "pipeline should succeed because step is skipped")

	// Verify the skip event was emitted
	events := collector.GetEvents()
	foundSkip := false
	for _, evt := range events {
		if evt.State == "skipped" && evt.StepID == "step-1" {
			foundSkip = true
			break
		}
	}
	assert.True(t, foundSkip, "should have emitted a skipped event")
}

// TestExecuteStep_RetryConfig_OnFailureContinue verifies that on_failure=continue continues.
func TestExecuteStep_RetryConfig_OnFailureContinue(t *testing.T) {
	failAdapter := newCountingFailAdapter(5, errors.New("always fails"))
	collector := testutil.NewEventCollector()

	executor := NewDefaultPipelineExecutor(failAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "continue-test"},
		Steps: []Step{
			{
				ID:      "step-1",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "do something"},
				Retry: RetryConfig{
					MaxAttempts: 1,
					BaseDelay:   "1ms",
					OnFailure:   "continue",
				},
			},
			{
				ID:           "step-2",
				Persona:      "navigator",
				Dependencies: []string{"step-1"},
				Exec:         ExecConfig{Source: "do something else"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Pipeline should not return an error because on_failure=continue
	err := executor.Execute(ctx, p, m, "test input")
	// step-2 may fail because step-1 failed but pipeline should try to continue
	// The main thing: executor.Execute should NOT fail on step-1
	// step-2 depends on step-1, so whether step-2 runs depends on DAG validation
	_ = err

	// Verify a failed event was emitted for step-1 with "continues" message
	events := collector.GetEvents()
	foundContinue := false
	for _, evt := range events {
		if evt.State == "failed" && evt.StepID == "step-1" && strings.Contains(evt.Message, "continues") {
			foundContinue = true
			break
		}
	}
	assert.True(t, foundContinue, "should have emitted a failed-but-continues event")
}

// TestExecuteStep_AdaptPrompt_InjectsFailureContext verifies prompt adaptation on retry.
func TestExecuteStep_AdaptPrompt_InjectsFailureContext(t *testing.T) {
	failAdapter := newCountingFailAdapter(1, errors.New("contract validation failed: missing field"))
	collector := testutil.NewEventCollector()

	executor := NewDefaultPipelineExecutor(failAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "adapt-test"},
		Steps: []Step{
			{
				ID:      "step-1",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "implement the feature"},
				Retry: RetryConfig{
					MaxAttempts: 2,
					BaseDelay:   "1ms",
					AdaptPrompt: true,
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test input")
	assert.NoError(t, err, "should succeed on second attempt")

	// Verify that the second call got retry context injected
	configs := failAdapter.getLastConfigs()
	require.Len(t, configs, 2, "should have captured 2 adapter configs")

	// First call should have the original prompt
	assert.NotContains(t, configs[0].Prompt, "RETRY CONTEXT")

	// Second call should have retry context prepended
	assert.Contains(t, configs[1].Prompt, "RETRY CONTEXT")
	assert.Contains(t, configs[1].Prompt, "attempt 2 of 2")
	assert.Contains(t, configs[1].Prompt, "contract validation failed: missing field")
	assert.Contains(t, configs[1].Prompt, "implement the feature")
}

// TestStepTimeoutMinutes_OverridesManifestDefault verifies that step-level timeout_minutes
// takes precedence over runtime.default_timeout_minutes from the manifest.
func TestStepTimeoutMinutes_OverridesManifestDefault(t *testing.T) {
	capturingAdapter := &configCapturingAdapter{
		MockAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		),
	}

	executor := NewDefaultPipelineExecutor(capturingAdapter)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)
	m.Runtime.DefaultTimeoutMin = 10 // manifest says 10 minutes

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "timeout-test"},
		Steps: []Step{
			{
				ID:             "long-step",
				Persona:        "navigator",
				TimeoutMinutes: 90, // step says 90 minutes
				Exec:           ExecConfig{Source: "implement feature"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	cfg := capturingAdapter.getLastConfig()
	assert.Equal(t, 90*time.Minute, cfg.Timeout,
		"step-level timeout_minutes should override manifest default_timeout_minutes")
}

// TestStepTimeoutMinutes_OverridesCLITimeout verifies that step-level timeout_minutes
// takes precedence over the CLI --timeout flag.
func TestStepTimeoutMinutes_OverridesCLITimeout(t *testing.T) {
	capturingAdapter := &configCapturingAdapter{
		MockAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		),
	}

	executor := NewDefaultPipelineExecutor(capturingAdapter,
		WithStepTimeout(15*time.Minute), // CLI says 15 minutes
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)
	m.Runtime.DefaultTimeoutMin = 5 // manifest says 5 minutes

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "timeout-cli-test"},
		Steps: []Step{
			{
				ID:             "override-step",
				Persona:        "navigator",
				TimeoutMinutes: 60, // step says 60 minutes — wins
				Exec:           ExecConfig{Source: "do work"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	cfg := capturingAdapter.getLastConfig()
	assert.Equal(t, 60*time.Minute, cfg.Timeout,
		"step-level timeout_minutes should override CLI --timeout flag")
}

// TestStepTimeoutMinutes_FallsBackToCLIWhenUnset verifies that when no step-level
// timeout is configured, the CLI --timeout flag is used.
func TestStepTimeoutMinutes_FallsBackToCLIWhenUnset(t *testing.T) {
	capturingAdapter := &configCapturingAdapter{
		MockAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		),
	}

	executor := NewDefaultPipelineExecutor(capturingAdapter,
		WithStepTimeout(20*time.Minute), // CLI says 20 minutes
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)
	m.Runtime.DefaultTimeoutMin = 5 // manifest says 5 minutes

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "timeout-fallback-test"},
		Steps: []Step{
			{
				ID:      "default-step",
				Persona: "navigator",
				// No timeout_minutes set
				Exec: ExecConfig{Source: "quick task"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	cfg := capturingAdapter.getLastConfig()
	assert.Equal(t, 20*time.Minute, cfg.Timeout,
		"when no step-level timeout, CLI --timeout should be used")
}

// TestStepTimeoutMinutes_FallsBackToManifestWhenNoCLI verifies that when neither
// step-level timeout nor CLI --timeout is set, the manifest default is used.
func TestStepTimeoutMinutes_FallsBackToManifestWhenNoCLI(t *testing.T) {
	capturingAdapter := &configCapturingAdapter{
		MockAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		),
	}

	executor := NewDefaultPipelineExecutor(capturingAdapter)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)
	m.Runtime.DefaultTimeoutMin = 8 // manifest says 8 minutes

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "timeout-manifest-test"},
		Steps: []Step{
			{
				ID:      "manifest-step",
				Persona: "navigator",
				// No timeout_minutes, no CLI override
				Exec: ExecConfig{Source: "default task"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	cfg := capturingAdapter.getLastConfig()
	assert.Equal(t, 8*time.Minute, cfg.Timeout,
		"when no step-level or CLI timeout, manifest default should be used")
}

// TestMaxConcurrentAgents_FlowsToAdapterConfig verifies that MaxConcurrentAgents
// set on a pipeline step is correctly passed through to the AdapterRunConfig.
func TestMaxConcurrentAgents_FlowsToAdapterConfig(t *testing.T) {
	capturingAdapter := &configCapturingAdapter{
		MockAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		),
	}

	executor := NewDefaultPipelineExecutor(capturingAdapter)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "concurrency-test"},
		Steps: []Step{
			{
				ID:                  "parallel-step",
				Persona:             "navigator",
				MaxConcurrentAgents: 5,
				Exec:                ExecConfig{Source: "do parallel work"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	cfg := capturingAdapter.getLastConfig()
	assert.Equal(t, 5, cfg.MaxConcurrentAgents,
		"MaxConcurrentAgents should flow from step to adapter config")
}

// TestMaxConcurrentAgents_ZeroWhenUnset verifies that MaxConcurrentAgents defaults
// to 0 in AdapterRunConfig when not set on the step.
func TestMaxConcurrentAgents_ZeroWhenUnset(t *testing.T) {
	capturingAdapter := &configCapturingAdapter{
		MockAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		),
	}

	executor := NewDefaultPipelineExecutor(capturingAdapter)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "concurrency-default-test"},
		Steps: []Step{
			{
				ID:      "single-step",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "do work"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	cfg := capturingAdapter.getLastConfig()
	assert.Equal(t, 0, cfg.MaxConcurrentAgents,
		"MaxConcurrentAgents should default to 0 when unset")
}

// TestStepTimeoutMinutes_PerStepDifferentTimeouts verifies that different steps
// in the same pipeline can have different timeouts.
func TestStepTimeoutMinutes_PerStepDifferentTimeouts(t *testing.T) {
	var mu sync.Mutex
	configs := make(map[string]adapter.AdapterRunConfig)

	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
		adaptertest.WithTokensUsed(100),
	)

	// Custom adapter that captures configs per step
	wrappedAdapter := &perStepCapturingAdapter{
		MockAdapter: mockAdapter,
		configs:     configs,
		mu:          &mu,
	}

	executor := NewDefaultPipelineExecutor(wrappedAdapter)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)
	m.Runtime.DefaultTimeoutMin = 10

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "multi-timeout-test"},
		Steps: []Step{
			{
				ID:             "quick-step",
				Persona:        "navigator",
				TimeoutMinutes: 5,
				Exec:           ExecConfig{Source: "quick"},
			},
			{
				ID:             "long-step",
				Persona:        "navigator",
				TimeoutMinutes: 90,
				Dependencies:   []string{"quick-step"},
				Exec:           ExecConfig{Source: "long"},
			},
			{
				ID:           "default-step",
				Persona:      "navigator",
				Dependencies: []string{"long-step"},
				Exec:         ExecConfig{Source: "default"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()

	assert.Equal(t, 5*time.Minute, configs["quick-step"].Timeout,
		"quick-step should use its step-level timeout")
	assert.Equal(t, 90*time.Minute, configs["long-step"].Timeout,
		"long-step should use its step-level timeout")
	assert.Equal(t, 10*time.Minute, configs["default-step"].Timeout,
		"default-step should fall back to manifest default")
}

// perStepCapturingAdapter captures AdapterRunConfig per step based on prompt content.
type perStepCapturingAdapter struct {
	*adaptertest.MockAdapter
	mu      *sync.Mutex
	configs map[string]adapter.AdapterRunConfig
}

func (a *perStepCapturingAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	a.mu.Lock()
	// Use the Persona field to identify step — but that's the same for all.
	// Instead, use Prompt which matches exec.source.
	switch {
	case strings.Contains(cfg.Prompt, "quick"):
		a.configs["quick-step"] = cfg
	case strings.Contains(cfg.Prompt, "long"):
		a.configs["long-step"] = cfg
	case strings.Contains(cfg.Prompt, "default"):
		a.configs["default-step"] = cfg
	}
	a.mu.Unlock()
	return a.MockAdapter.Run(ctx, cfg)
}

// --- Optional Step Tests ---

// TestOptionalStep_FailsPipelineContinues verifies that when an optional step fails,
// the pipeline continues to the next independent step and completes successfully.
func TestOptionalStep_FailsPipelineContinues(t *testing.T) {
	collector := testutil.NewEventCollector()
	successAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "ok"}`),
	)
	failAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithFailure(errors.New("optional step failed")),
	)

	sa := &stepAwareAdapter{
		defaultAdapter: successAdapter,
		stepAdapters: map[string]adapter.AdapterRunner{
			"optional-step": failAdapter,
		},
	}

	executor := NewDefaultPipelineExecutor(sa, WithEmitter(collector))

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "optional-fail-test"},
		Steps: []Step{
			{ID: "step-1", Persona: "navigator", Exec: ExecConfig{Source: "first"}},
			{ID: "optional-step", Persona: "navigator", Optional: true, Dependencies: []string{"step-1"}, Exec: ExecConfig{Source: "optional work"}},
			{ID: "step-3", Persona: "navigator", Dependencies: []string{"step-1"}, Exec: ExecConfig{Source: "independent"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	assert.NoError(t, err, "pipeline should succeed when only optional step fails")

	// Verify step-3 ran (it depends only on step-1 which succeeded)
	order := collector.GetStepExecutionOrder()
	assert.Contains(t, order, "step-3", "step-3 should have executed")

	// Verify optional-step has a failed event with "continues" message
	events := collector.GetEvents()
	foundContinue := false
	for _, evt := range events {
		if evt.State == "failed" && evt.StepID == "optional-step" && strings.Contains(evt.Message, "continues") {
			foundContinue = true
			break
		}
	}
	assert.True(t, foundContinue, "should have emitted a failed-but-continues event for optional step")
}

// TestOptionalStep_SucceedsNormally verifies that an optional step that succeeds
// behaves identically to a required step.
func TestOptionalStep_SucceedsNormally(t *testing.T) {
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "ok"}`),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter, WithEmitter(collector))

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "optional-success-test"},
		Steps: []Step{
			{ID: "step-1", Persona: "navigator", Exec: ExecConfig{Source: "first"}},
			{ID: "optional-step", Persona: "navigator", Optional: true, Dependencies: []string{"step-1"}, Exec: ExecConfig{Source: "optional work"}},
			{ID: "step-3", Persona: "navigator", Dependencies: []string{"optional-step"}, Exec: ExecConfig{Source: "after optional"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	assert.NoError(t, err, "pipeline should succeed")

	// All three steps should have executed in order
	order := collector.GetStepExecutionOrder()
	require.Len(t, order, 3, "all steps should have executed")
	assert.Equal(t, "step-1", order[0])
	assert.Equal(t, "optional-step", order[1])
	assert.Equal(t, "step-3", order[2])
}

// TestOptionalStep_DefaultBehaviorPreserved verifies that a step without the optional
// field still halts the pipeline on failure (regression test).
func TestOptionalStep_DefaultBehaviorPreserved(t *testing.T) {
	collector := testutil.NewEventCollector()
	failAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithFailure(errors.New("required step failed")),
	)

	executor := NewDefaultPipelineExecutor(failAdapter, WithEmitter(collector))

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "default-behavior-test"},
		Steps: []Step{
			{ID: "step-1", Persona: "navigator", Exec: ExecConfig{Source: "do work"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	assert.Error(t, err, "pipeline should fail when required step fails")

	var stepErr *StepExecutionError
	assert.True(t, errors.As(err, &stepErr), "error should be a StepExecutionError")
	assert.Equal(t, "step-1", stepErr.StepID)
}

// TestOptionalStep_WithRetries verifies that an optional step with max_attempts > 1
// retries all attempts before continuing.
func TestOptionalStep_WithRetries(t *testing.T) {
	collector := testutil.NewEventCollector()
	// Fails 5 times — more than max_attempts
	failAdapter := newCountingFailAdapter(5, errors.New("transient failure"))

	executor := NewDefaultPipelineExecutor(failAdapter, WithEmitter(collector))

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "optional-retry-test"},
		Steps: []Step{
			{
				ID:       "optional-step",
				Persona:  "navigator",
				Optional: true,
				Exec:     ExecConfig{Source: "do work"},
				Retry: RetryConfig{
					MaxAttempts: 3,
					BaseDelay:   "1ms",
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	assert.NoError(t, err, "pipeline should succeed after optional step exhausts retries")

	// Verify all 3 attempts were made
	assert.Equal(t, 3, failAdapter.getCallCount(), "should have attempted 3 times")

	// Verify retrying events were emitted
	events := collector.GetEvents()
	retryCount := 0
	for _, evt := range events {
		if evt.State == "retrying" && evt.StepID == "optional-step" {
			retryCount++
		}
	}
	assert.Equal(t, 2, retryCount, "should have 2 retry events (attempts 2 and 3)")
}

// TestOptionalStep_DependentSkipped verifies that a step depending on a failed
// optional step is skipped.
func TestOptionalStep_DependentSkipped(t *testing.T) {
	collector := testutil.NewEventCollector()
	successAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "ok"}`),
	)
	failAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithFailure(errors.New("optional failure")),
	)

	sa := &stepAwareAdapter{
		defaultAdapter: successAdapter,
		stepAdapters: map[string]adapter.AdapterRunner{
			"optional-step": failAdapter,
		},
	}

	executor := NewDefaultPipelineExecutor(sa, WithEmitter(collector))

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "dep-skip-test"},
		Steps: []Step{
			{ID: "optional-step", Persona: "navigator", Optional: true, Exec: ExecConfig{Source: "optional work"}},
			{ID: "dependent-step", Persona: "navigator", Dependencies: []string{"optional-step"}, Exec: ExecConfig{Source: "depends on optional"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	assert.NoError(t, err, "pipeline should succeed")

	// dependent-step should have been skipped, not executed
	order := collector.GetStepExecutionOrder()
	assert.NotContains(t, order, "dependent-step", "dependent-step should not have executed")

	// Verify skip event for dependent-step
	events := collector.GetEvents()
	foundSkip := false
	for _, evt := range events {
		if evt.State == "skipped" && evt.StepID == "dependent-step" {
			foundSkip = true
			break
		}
	}
	assert.True(t, foundSkip, "dependent-step should have been skipped")
}

// TestOptionalStep_TransitiveDependencySkip verifies that C depends on B depends on
// optional A — when A fails, both B and C are skipped.
func TestOptionalStep_TransitiveDependencySkip(t *testing.T) {
	collector := testutil.NewEventCollector()
	successAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "ok"}`),
	)
	failAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithFailure(errors.New("optional failure")),
	)

	sa := &stepAwareAdapter{
		defaultAdapter: successAdapter,
		stepAdapters: map[string]adapter.AdapterRunner{
			"step-a": failAdapter,
		},
	}

	executor := NewDefaultPipelineExecutor(sa, WithEmitter(collector))

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "transitive-skip-test"},
		Steps: []Step{
			{ID: "step-a", Persona: "navigator", Optional: true, Exec: ExecConfig{Source: "optional work"}},
			{ID: "step-b", Persona: "navigator", Dependencies: []string{"step-a"}, Exec: ExecConfig{Source: "depends on A"}},
			{ID: "step-c", Persona: "navigator", Dependencies: []string{"step-b"}, Exec: ExecConfig{Source: "depends on B"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	assert.NoError(t, err, "pipeline should succeed")

	// Neither B nor C should have executed
	order := collector.GetStepExecutionOrder()
	assert.NotContains(t, order, "step-b", "step-b should not have executed")
	assert.NotContains(t, order, "step-c", "step-c should not have executed")

	// Both B and C should have been skipped
	events := collector.GetEvents()
	skippedSteps := make(map[string]bool)
	for _, evt := range events {
		if evt.State == "skipped" {
			skippedSteps[evt.StepID] = true
		}
	}
	assert.True(t, skippedSteps["step-b"], "step-b should have been skipped")
	assert.True(t, skippedSteps["step-c"], "step-c should have been skipped")
}

// TestOptionalStep_ExplicitOnFailurePrecedence verifies that optional: true with
// retry.on_failure: "fail" results in pipeline halt (explicit wins).
func TestOptionalStep_ExplicitOnFailurePrecedence(t *testing.T) {
	collector := testutil.NewEventCollector()
	failAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithFailure(errors.New("step failed")),
	)

	executor := NewDefaultPipelineExecutor(failAdapter, WithEmitter(collector))

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "precedence-test"},
		Steps: []Step{
			{
				ID:       "step-1",
				Persona:  "navigator",
				Optional: true,
				Exec:     ExecConfig{Source: "do work"},
				Retry: RetryConfig{
					OnFailure: "fail", // Explicit on_failure takes precedence over optional
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	assert.Error(t, err, "pipeline should fail because explicit on_failure: fail takes precedence")

	var stepErr *StepExecutionError
	assert.True(t, errors.As(err, &stepErr), "error should be a StepExecutionError")
}

// TestOptionalStep_PipelineStatusCompleted verifies that pipeline status is completed
// when only optional steps fail.
func TestOptionalStep_PipelineStatusCompleted(t *testing.T) {
	collector := testutil.NewEventCollector()
	stateStore := testutil.NewMockStateStore()
	successAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "ok"}`),
	)
	failAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithFailure(errors.New("optional failure")),
	)

	sa := &stepAwareAdapter{
		defaultAdapter: successAdapter,
		stepAdapters: map[string]adapter.AdapterRunner{
			"optional-step": failAdapter,
		},
	}

	executor := NewDefaultPipelineExecutor(sa,
		WithEmitter(collector),
		WithStateStore(stateStore),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "status-test"},
		Steps: []Step{
			{ID: "required-step", Persona: "navigator", Exec: ExecConfig{Source: "required work"}},
			{ID: "optional-step", Persona: "navigator", Optional: true, Dependencies: []string{"required-step"}, Exec: ExecConfig{Source: "optional work"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	assert.NoError(t, err, "pipeline should succeed")

	// Verify pipeline state is completed via state store
	pipelineID := collector.GetPipelineID()
	require.NotEmpty(t, pipelineID)

	record, storeErr := stateStore.GetPipelineState(pipelineID)
	require.NoError(t, storeErr)
	assert.Equal(t, stateCompleted, record.Status, "pipeline status should be completed")

	// Verify the completed event was emitted
	events := collector.GetEvents()
	foundCompleted := false
	for _, evt := range events {
		if evt.State == "completed" && evt.StepID == "" {
			foundCompleted = true
			break
		}
	}
	assert.True(t, foundCompleted, "should have emitted a completed event for the pipeline")
}

func TestPreserveWorkspaceKeepsExistingContent(t *testing.T) {
	tmpDir := t.TempDir()

	// Pre-create workspace directory with test content that should survive
	runID := "preserve-test-run"
	pipelineWsPath := filepath.Join(tmpDir, runID)
	require.NoError(t, os.MkdirAll(pipelineWsPath, 0755))
	markerFile := filepath.Join(pipelineWsPath, "debug-marker.txt")
	require.NoError(t, os.WriteFile(markerFile, []byte("preserved"), 0644))

	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
		WithRunID(runID),
		WithPreserveWorkspace(true),
	)

	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "preserve-ws-test"},
		Steps: []Step{
			{ID: "step1", Persona: "navigator", Exec: ExecConfig{Source: "test"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	// Verify pre-existing content was preserved
	content, err := os.ReadFile(markerFile)
	require.NoError(t, err)
	assert.Equal(t, "preserved", string(content), "marker file should be preserved")

	// Verify warning event was emitted
	events := collector.GetEvents()
	foundWarning := false
	for _, evt := range events {
		if evt.State == "warning" && strings.Contains(evt.Message, "--preserve-workspace active") {
			foundWarning = true
			break
		}
	}
	assert.True(t, foundWarning, "should have emitted a preserve-workspace warning event")
}

func TestDefaultBehaviorCleansWorkspace(t *testing.T) {
	tmpDir := t.TempDir()

	// Pre-create workspace directory with test content that should be cleaned
	runID := "cleanup-test-run"
	pipelineWsPath := filepath.Join(tmpDir, runID)
	require.NoError(t, os.MkdirAll(pipelineWsPath, 0755))
	markerFile := filepath.Join(pipelineWsPath, "stale-marker.txt")
	require.NoError(t, os.WriteFile(markerFile, []byte("should-be-removed"), 0644))

	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
		WithRunID(runID),
	)

	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "cleanup-ws-test"},
		Steps: []Step{
			{ID: "step1", Persona: "navigator", Exec: ExecConfig{Source: "test"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	// Verify pre-existing content was cleaned
	_, err = os.Stat(markerFile)
	assert.True(t, os.IsNotExist(err), "marker file should have been removed by workspace cleanup")
}

// TestExecuteWithIncludeFilter verifies that --steps filter runs only the named steps
func TestExecuteWithIncludeFilter(t *testing.T) {
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
		adaptertest.WithTokensUsed(100),
	)

	filter := &StepFilter{Include: []string{"step-a", "step-c"}}
	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
		WithStepFilter(filter),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "include-filter-test"},
		Steps: []Step{
			{ID: "step-a", Persona: "navigator", Exec: ExecConfig{Source: "A"}},
			{ID: "step-b", Persona: "navigator", Dependencies: []string{"step-a"}, Exec: ExecConfig{Source: "B"}},
			{ID: "step-c", Persona: "navigator", Dependencies: []string{"step-a"}, Exec: ExecConfig{Source: "C"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	order := collector.GetStepExecutionOrder()
	assert.Equal(t, []string{"step-a", "step-c"}, order, "only included steps should execute")
}

// TestExecuteWithExcludeFilter verifies that --exclude filter skips the named steps
func TestExecuteWithExcludeFilter(t *testing.T) {
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
		adaptertest.WithTokensUsed(100),
	)

	filter := &StepFilter{Exclude: []string{"step-b"}}
	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
		WithStepFilter(filter),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "exclude-filter-test"},
		Steps: []Step{
			{ID: "step-a", Persona: "navigator", Exec: ExecConfig{Source: "A"}},
			{ID: "step-b", Persona: "navigator", Dependencies: []string{"step-a"}, Exec: ExecConfig{Source: "B"}},
			{ID: "step-c", Persona: "navigator", Dependencies: []string{"step-a"}, Exec: ExecConfig{Source: "C"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	order := collector.GetStepExecutionOrder()
	assert.Equal(t, []string{"step-a", "step-c"}, order, "excluded step should not execute")
}

// TestExecuteWithInvalidStepFilter verifies that invalid step names in filter produce errors
func TestExecuteWithInvalidStepFilter(t *testing.T) {
	mockAdapter := adaptertest.NewMockAdapter()
	filter := &StepFilter{Include: []string{"nonexistent"}}
	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithStepFilter(filter),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "invalid-filter-test"},
		Steps: []Step{
			{ID: "step-a", Persona: "navigator", Exec: ExecConfig{Source: "A"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown step")
}

// TestExecuteWithNilFilter verifies that nil filter runs all steps (no-op)
func TestExecuteWithNilFilter(t *testing.T) {
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
		adaptertest.WithTokensUsed(100),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
		// No WithStepFilter — defaults to nil
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "nil-filter-test"},
		Steps: []Step{
			{ID: "step-a", Persona: "navigator", Exec: ExecConfig{Source: "A"}},
			{ID: "step-b", Persona: "navigator", Dependencies: []string{"step-a"}, Exec: ExecConfig{Source: "B"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	order := collector.GetStepExecutionOrder()
	assert.Equal(t, []string{"step-a", "step-b"}, order, "all steps should execute with nil filter")
}

// ============================================================================
// Rework Branching Tests
// ============================================================================

// promptFailAdapter fails when the prompt contains a specific substring, succeeds otherwise.
type promptFailAdapter struct {
	mu          sync.Mutex
	failPrompt  string // fail if prompt contains this substring
	failError   error
	successMock *adaptertest.MockAdapter
	lastConfigs []adapter.AdapterRunConfig
}

func newPromptFailAdapter(failPrompt string, failErr error) *promptFailAdapter {
	return &promptFailAdapter{
		failPrompt:  failPrompt,
		failError:   failErr,
		successMock: adaptertest.NewMockAdapter(adaptertest.WithStdoutJSON(`{"status":"ok"}`)),
	}
}

func (a *promptFailAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	a.mu.Lock()
	a.lastConfigs = append(a.lastConfigs, cfg)
	a.mu.Unlock()
	if strings.Contains(cfg.Prompt, a.failPrompt) {
		return nil, a.failError
	}
	return a.successMock.Run(ctx, cfg)
}

// TestExecuteStep_OnFailureRework_TriggersReworkStep verifies that on_failure=rework
// executes the rework target step after the original step exhausts its retries.
func TestExecuteStep_OnFailureRework_TriggersReworkStep(t *testing.T) {
	// Adapter fails when prompt contains "do something" (step-1), succeeds for rework.
	failAdapter := newPromptFailAdapter("do something", errors.New("step-1 failed"))
	collector := testutil.NewEventCollector()

	executor := NewDefaultPipelineExecutor(failAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	// Use a 3-step pipeline: step-0 → step-1 (fails, reworks to rework-step)
	// This ensures step-1 and rework-step don't run in the same concurrent batch.
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "rework-test"},
		Steps: []Step{
			{
				ID:      "step-0",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "init"},
			},
			{
				ID:           "step-1",
				Persona:      "navigator",
				Dependencies: []string{"step-0"},
				Exec:         ExecConfig{Source: "do something"},
				Retry: RetryConfig{
					MaxAttempts: 1,
					BaseDelay:   "1ms",
					OnFailure:   "rework",
					ReworkStep:  "rework-step",
				},
			},
			{
				ID:         "rework-step",
				Persona:    "navigator",
				ReworkOnly: true,
				Exec:       ExecConfig{Source: "rework fallback"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test input")
	assert.NoError(t, err, "pipeline should succeed because rework step succeeds")

	// Verify the reworking event was emitted
	events := collector.GetEvents()
	foundRework := false
	for _, evt := range events {
		if evt.State == "reworking" && evt.StepID == "step-1" {
			foundRework = true
			break
		}
	}
	assert.True(t, foundRework, "should have emitted a reworking event for step-1")

	// Verify rework-step was completed
	foundReworkComplete := false
	for _, evt := range events {
		if evt.State == "completed" && evt.StepID == "rework-step" {
			foundReworkComplete = true
			break
		}
	}
	assert.True(t, foundReworkComplete, "should have emitted a completed event for rework-step")
}

// TestExecuteStep_OnFailureRework_ReworkStepFailsPropagates verifies that when the
// rework step also fails, the error propagates and the pipeline fails.
func TestExecuteStep_OnFailureRework_ReworkStepFailsPropagates(t *testing.T) {
	// Adapter fails when prompt contains "fail-me" — matches both step-1 and rework-step
	// but not step-0 which has prompt "init".
	failAdapter := newPromptFailAdapter("fail-me", errors.New("always fails"))
	collector := testutil.NewEventCollector()

	executor := NewDefaultPipelineExecutor(failAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "rework-fail-test"},
		Steps: []Step{
			{
				ID:      "step-0",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "init"},
			},
			{
				ID:           "step-1",
				Persona:      "navigator",
				Dependencies: []string{"step-0"},
				Exec:         ExecConfig{Source: "fail-me primary"},
				Retry: RetryConfig{
					MaxAttempts: 1,
					BaseDelay:   "1ms",
					OnFailure:   "rework",
					ReworkStep:  "rework-step",
				},
			},
			{
				ID:         "rework-step",
				Persona:    "navigator",
				ReworkOnly: true,
				Exec:       ExecConfig{Source: "fail-me fallback"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test input")
	require.Error(t, err, "pipeline should fail when rework step also fails")

	// Verify the rework step failure event
	events := collector.GetEvents()
	foundReworkFail := false
	for _, evt := range events {
		if evt.State == "failed" && evt.StepID == "rework-step" {
			foundReworkFail = true
			break
		}
	}
	assert.True(t, foundReworkFail, "should have emitted a failed event for rework-step")
}

// TestExecuteStep_OnFailureRework_ExistingOnFailureBehaviorsUnchanged verifies that
// existing on_failure behaviors (fail, skip, continue) work as before.
func TestExecuteStep_OnFailureRework_ExistingOnFailureBehaviorsUnchanged(t *testing.T) {
	// Regression test: verify "fail" still works
	failAdapter := newCountingFailAdapter(5, errors.New("always fails"))
	collector := testutil.NewEventCollector()

	executor := NewDefaultPipelineExecutor(failAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "fail-regression-test"},
		Steps: []Step{
			{
				ID:      "step-1",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "do something"},
				Retry: RetryConfig{
					MaxAttempts: 1,
					BaseDelay:   "1ms",
					OnFailure:   "fail",
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test input")
	require.Error(t, err, "pipeline should fail with on_failure=fail")
}

// TestExecuteStep_OnFailureRework_FailureContextInjected verifies that rework step
// receives the failure context from the failed step.
func TestExecuteStep_OnFailureRework_FailureContextInjected(t *testing.T) {
	// Adapter fails when prompt contains "do something" (step-1), succeeds for rework.
	failAdapter := newPromptFailAdapter("do something", errors.New("contract validation failed: missing field"))
	collector := testutil.NewEventCollector()

	executor := NewDefaultPipelineExecutor(failAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	// Use step-0 → step-1 to ensure step-1 and rework-step don't run in the
	// same concurrent batch (rework-step executes inline during step-1's rework).
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "rework-context-test"},
		Steps: []Step{
			{
				ID:      "step-0",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "init"},
			},
			{
				ID:           "step-1",
				Persona:      "navigator",
				Dependencies: []string{"step-0"},
				Exec:         ExecConfig{Source: "do something"},
				Retry: RetryConfig{
					MaxAttempts: 1,
					BaseDelay:   "1ms",
					OnFailure:   "rework",
					ReworkStep:  "rework-step",
				},
			},
			{
				ID:         "rework-step",
				Persona:    "navigator",
				ReworkOnly: true,
				Exec:       ExecConfig{Source: "rework fallback"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test input")
	assert.NoError(t, err, "pipeline should succeed via rework")

	// Verify that the rework step's prompt contained failure context
	failAdapter.mu.Lock()
	configs := make([]adapter.AdapterRunConfig, len(failAdapter.lastConfigs))
	copy(configs, failAdapter.lastConfigs)
	failAdapter.mu.Unlock()

	// Find the rework step config — look for the one with REWORK CONTEXT in prompt
	var reworkPrompt string
	for _, cfg := range configs {
		if strings.Contains(cfg.Prompt, "REWORK CONTEXT") {
			reworkPrompt = cfg.Prompt
			break
		}
	}
	require.NotEmpty(t, reworkPrompt, "should have found a config with REWORK CONTEXT in prompt")
	assert.Contains(t, reworkPrompt, "step-1", "rework step prompt should reference the failed step ID")
	assert.Contains(t, reworkPrompt, "contract validation failed", "rework step prompt should contain prior error")
}

// TestExecuteStep_OnFailureRework_DownstreamStepsRun verifies that after successful rework,
// downstream steps that depend on the failed step still execute (not skipped).
func TestExecuteStep_OnFailureRework_DownstreamStepsRun(t *testing.T) {
	// Adapter fails when prompt contains "do something" (step-1), succeeds for everything else.
	failAdapter := newPromptFailAdapter("do something", errors.New("step-1 failed"))
	collector := testutil.NewEventCollector()

	executor := NewDefaultPipelineExecutor(failAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	// Pipeline: step-0 → step-1 (fails, reworks) → step-downstream
	// The downstream step depends on step-1 and should run after rework succeeds.
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "rework-downstream-test"},
		Steps: []Step{
			{
				ID:      "step-0",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "init"},
			},
			{
				ID:           "step-1",
				Persona:      "navigator",
				Dependencies: []string{"step-0"},
				Exec:         ExecConfig{Source: "do something"},
				Retry: RetryConfig{
					MaxAttempts: 1,
					OnFailure:   "rework",
					ReworkStep:  "rework-step",
				},
			},
			{
				ID:         "rework-step",
				Persona:    "navigator",
				ReworkOnly: true,
				Exec:       ExecConfig{Source: "rework fallback"},
			},
			{
				ID:           "step-downstream",
				Persona:      "navigator",
				Dependencies: []string{"step-1"},
				Exec:         ExecConfig{Source: "downstream work"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test input")
	assert.NoError(t, err, "pipeline should succeed — downstream step should run after rework")

	// Verify downstream step completed
	events := collector.GetEvents()
	var downstreamCompleted bool
	for _, evt := range events {
		if evt.State == "completed" && evt.StepID == "step-downstream" {
			downstreamCompleted = true
			break
		}
	}
	assert.True(t, downstreamCompleted, "step-downstream should have completed after successful rework")

	// Verify the pipeline state is completed, not failed
	var pipelineCompleted bool
	for _, evt := range events {
		if evt.State == "completed" && evt.StepID == "" {
			pipelineCompleted = true
			break
		}
	}
	assert.True(t, pipelineCompleted, "pipeline should be in completed state")
}

// TestExecuteStep_OnFailureRework_ReworkOnlyNotScheduled verifies that rework_only steps
// are not scheduled in the normal DAG pass.
func TestExecuteStep_OnFailureRework_ReworkOnlyNotScheduled(t *testing.T) {
	// Adapter succeeds for everything — step-1 should NOT fail, so rework step should never run.
	mockAdapter := adaptertest.NewMockAdapter(adaptertest.WithStdoutJSON(`{"status":"ok"}`))
	collector := testutil.NewEventCollector()

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "rework-only-test"},
		Steps: []Step{
			{
				ID:      "step-1",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "do something"},
			},
			{
				ID:         "rework-step",
				Persona:    "navigator",
				ReworkOnly: true,
				Exec:       ExecConfig{Source: "should not run"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test input")
	assert.NoError(t, err, "pipeline should succeed")

	// Verify rework-step was never executed
	events := collector.GetEvents()
	for _, evt := range events {
		if evt.StepID == "rework-step" && (evt.State == "running" || evt.State == "completed") {
			t.Fatal("rework-step should not have been scheduled in normal DAG pass")
		}
	}
}

// TestExecuteWithoutSkillsField (T019) verifies backward compatibility:
// a pipeline with NO skills: fields at any scope (manifest, persona, pipeline)
// produces the same behavior as before the skill hierarchy feature was added.
func TestExecuteWithoutSkillsField(t *testing.T) {
	t.Run("pipeline_with_only_requires_skills_unchanged", func(t *testing.T) {
		// A pipeline using only the legacy requires.skills field (SkillConfig map)
		// should execute without errors and without needing a skill store.
		// The check command uses "true" which always succeeds on Linux.
		collector := testutil.NewEventCollector()
		mockAdapter := adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		)

		// No withSkillStore option — executor has nil skillStore
		executor := NewDefaultPipelineExecutor(mockAdapter,
			WithEmitter(collector),
		)

		tmpDir := t.TempDir()
		m := testutil.CreateTestManifest(tmpDir)
		// Manifest has no Skills field (zero value: nil slice)
		assert.Nil(t, m.Skills, "CreateTestManifest should not set manifest-level Skills")
		// Personas have no Skills field (zero value: nil slice)
		for name, p := range m.Personas {
			assert.Nil(t, p.Skills, "persona %q should not have Skills set", name)
		}

		p := &Pipeline{
			Metadata: PipelineMetadata{Name: "legacy-requires-skills"},
			Requires: &Requires{
				Skills: map[string]skill.SkillConfig{
					"test-skill": {Check: "true"},
				},
			},
			// No Skills field set (zero value: nil slice)
			Steps: []Step{
				{ID: "step-1", Persona: "navigator", Exec: ExecConfig{Source: "do work"}},
			},
		}
		// Pipeline.Skills is nil (no new-style skill references)
		assert.Nil(t, p.Skills, "pipeline Skills should be nil for legacy-only usage")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := executor.Execute(ctx, p, m, "test input")
		assert.NoError(t, err, "pipeline with only requires.skills should execute without error")

		// Verify the step ran
		order := collector.GetStepExecutionOrder()
		assert.Equal(t, []string{"step-1"}, order, "step-1 should have executed")
	})

	t.Run("validateSkillRefs_nil_store_returns_nil", func(t *testing.T) {
		// When the executor has no skill store, validateSkillRefs should
		// return nil regardless of what skill references exist.
		mockAdapter := adaptertest.NewMockAdapter()
		executor := NewDefaultPipelineExecutor(mockAdapter)

		// Pipeline with skills references at all scopes
		p := &Pipeline{
			Metadata: PipelineMetadata{Name: "no-store-test"},
			Skills:   []string{"skill-a", "skill-b"},
			Steps:    []Step{{ID: "s1", Persona: "navigator", Exec: ExecConfig{Source: "x"}}},
		}
		m := &manifest.Manifest{
			Skills: []string{"global-skill"},
			Personas: map[string]manifest.Persona{
				"navigator": {
					Adapter: "claude",
					Skills:  []string{"persona-skill"},
				},
			},
		}

		errs := executor.sec.validateSkillRefs(p.Skills, p.Metadata.Name, m)
		assert.Nil(t, errs, "validateSkillRefs should return nil when skill store is nil")
	})

	t.Run("no_skills_at_any_scope_executes_normally", func(t *testing.T) {
		// A pipeline with zero skill references anywhere should behave
		// identically to pre-skill-hierarchy pipelines.
		collector := testutil.NewEventCollector()
		mockAdapter := adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"result": "ok"}`),
			adaptertest.WithTokensUsed(200),
		)
		executor := NewDefaultPipelineExecutor(mockAdapter,
			WithEmitter(collector),
		)

		tmpDir := t.TempDir()
		m := testutil.CreateTestManifest(tmpDir)

		p := &Pipeline{
			Metadata: PipelineMetadata{Name: "no-skills-anywhere"},
			// No Requires, no Skills
			Steps: []Step{
				{ID: "a", Persona: "navigator", Exec: ExecConfig{Source: "first"}},
				{ID: "b", Persona: "craftsman", Dependencies: []string{"a"}, Exec: ExecConfig{Source: "second"}},
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := executor.Execute(ctx, p, m, "test")
		require.NoError(t, err, "pipeline with no skills at any scope should succeed")

		order := collector.GetStepExecutionOrder()
		require.Len(t, order, 2, "both steps should execute")
		assert.Equal(t, "a", order[0])
		assert.Equal(t, "b", order[1])
	})

	t.Run("ResolveSkills_all_empty_returns_nil", func(t *testing.T) {
		// ResolveSkills with nil/empty slices at all scopes should return
		// nil, meaning no change to existing behavior.
		result := skill.ResolveSkills(nil, nil, nil)
		assert.Nil(t, result, "ResolveSkills(nil, nil, nil) should return nil")

		result = skill.ResolveSkills([]string{}, []string{}, []string{})
		assert.Nil(t, result, "ResolveSkills(empty, empty, empty) should return nil")

		result = skill.ResolveSkills(nil, []string{}, nil)
		assert.Nil(t, result, "ResolveSkills(nil, empty, nil) should return nil")
	})
}

// TestSkillProvisioningIntegration verifies the executor correctly provisions
// DirectoryStore skills and passes skill metadata to the adapter config.
func TestSkillProvisioningIntegration(t *testing.T) {
	t.Run("executor_passes_resolved_skills_to_adapter_config", func(t *testing.T) {
		// Create a skill store with a test skill
		storeDir := t.TempDir()
		skillSrc := filepath.Join(storeDir, "test-skill")
		require.NoError(t, os.MkdirAll(skillSrc, 0o755))

		skillMD := "---\nname: test-skill\ndescription: A test skill\n---\n# Test Skill\n\nBody.\n"
		require.NoError(t, os.WriteFile(filepath.Join(skillSrc, "SKILL.md"), []byte(skillMD), 0o644))

		store := skill.NewDirectoryStore(skill.SkillSource{Root: storeDir, Precedence: 0})

		capturingAdapter := &configCapturingAdapter{
			MockAdapter: adaptertest.NewMockAdapter(
				adaptertest.WithStdoutJSON(`{"status": "success"}`),
				adaptertest.WithTokensUsed(100),
			),
		}

		executor := NewDefaultPipelineExecutor(capturingAdapter,
			withSkillStore(store),
		)

		tmpDir := t.TempDir()
		m := testutil.CreateTestManifest(tmpDir)
		m.Skills = []string{"test-skill"} // Global skill reference

		p := &Pipeline{
			Metadata: PipelineMetadata{Name: "skill-provision-test"},
			Steps: []Step{
				{ID: "step-1", Persona: "navigator", Exec: ExecConfig{Source: "do work"}},
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := executor.Execute(ctx, p, m, "test input")
		require.NoError(t, err)

		cfg := capturingAdapter.getLastConfig()
		require.Len(t, cfg.ResolvedSkills, 1, "should have 1 resolved skill")
		assert.Equal(t, "test-skill", cfg.ResolvedSkills[0].Name)
		assert.Equal(t, "A test skill", cfg.ResolvedSkills[0].Description)
	})

	t.Run("step_level_skills_propagate_to_adapter_config", func(t *testing.T) {
		storeDir := t.TempDir()
		skillSrc := filepath.Join(storeDir, "step-only-skill")
		require.NoError(t, os.MkdirAll(skillSrc, 0o755))
		skillMD := "---\nname: step-only-skill\ndescription: Step-scoped skill\n---\n# Body\n"
		require.NoError(t, os.WriteFile(filepath.Join(skillSrc, "SKILL.md"), []byte(skillMD), 0o644))

		store := skill.NewDirectoryStore(skill.SkillSource{Root: storeDir, Precedence: 0})

		capturingAdapter := &configCapturingAdapter{
			MockAdapter: adaptertest.NewMockAdapter(
				adaptertest.WithStdoutJSON(`{"status": "success"}`),
				adaptertest.WithTokensUsed(100),
			),
		}

		executor := NewDefaultPipelineExecutor(capturingAdapter, withSkillStore(store))

		tmpDir := t.TempDir()
		m := testutil.CreateTestManifest(tmpDir)
		// no global, no persona skills — only the step declares it
		p := &Pipeline{
			Metadata: PipelineMetadata{Name: "step-skill-test"},
			Steps: []Step{
				{
					ID:      "implement",
					Persona: "navigator",
					Skills:  []string{"step-only-skill"},
					Exec:    ExecConfig{Source: "do work"},
				},
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := executor.Execute(ctx, p, m, "test input")
		require.NoError(t, err)

		cfg := capturingAdapter.getLastConfig()
		require.Len(t, cfg.ResolvedSkills, 1, "step-level skill should propagate to adapter config")
		assert.Equal(t, "step-only-skill", cfg.ResolvedSkills[0].Name)
	})

	t.Run("step_level_skills_fail_preflight_when_missing", func(t *testing.T) {
		storeDir := t.TempDir()
		store := skill.NewDirectoryStore(skill.SkillSource{Root: storeDir, Precedence: 0})

		mockAdapter := adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		)
		executor := NewDefaultPipelineExecutor(mockAdapter, withSkillStore(store))

		tmpDir := t.TempDir()
		m := testutil.CreateTestManifest(tmpDir)
		p := &Pipeline{
			Metadata: PipelineMetadata{Name: "missing-step-skill"},
			Steps: []Step{
				{
					ID:      "implement",
					Persona: "navigator",
					Skills:  []string{"nonexistent-skill"},
					Exec:    ExecConfig{Source: "do work"},
				},
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := executor.Execute(ctx, p, m, "test input")
		require.Error(t, err, "expected preflight failure for missing step skill")
		assert.Contains(t, err.Error(), "wave skills add", "error should suggest install command")
	})

	t.Run("executor_returns_error_for_missing_store_skill", func(t *testing.T) {
		// Create an empty store — no skills available
		storeDir := t.TempDir()
		store := skill.NewDirectoryStore(skill.SkillSource{Root: storeDir, Precedence: 0})

		mockAdapter := adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		)

		executor := NewDefaultPipelineExecutor(mockAdapter,
			withSkillStore(store),
		)

		tmpDir := t.TempDir()
		m := testutil.CreateTestManifest(tmpDir)
		m.Skills = []string{"nonexistent-skill"}

		p := &Pipeline{
			Metadata: PipelineMetadata{Name: "missing-skill-test"},
			Steps: []Step{
				{ID: "step-1", Persona: "navigator", Exec: ExecConfig{Source: "do work"}},
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := executor.Execute(ctx, p, m, "test input")
		require.Error(t, err, "should return error for missing skill")
		assert.Contains(t, err.Error(), "nonexistent-skill")
	})
}

// TestTransitiveSkip_DiamondDependency verifies transitive skip propagation
// through a diamond-shaped dependency graph:
//
//	  A (optional, fails)
//	 / \
//	B   C
//	 \ /
//	  D
//
// All of B, C, D should be skipped. Pipeline should succeed because A is optional.
func TestTransitiveSkip_DiamondDependency(t *testing.T) {
	collector := testutil.NewEventCollector()
	successAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "ok"}`),
	)
	failAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithFailure(errors.New("optional failure")),
	)

	sa := &stepAwareAdapter{
		defaultAdapter: successAdapter,
		stepAdapters: map[string]adapter.AdapterRunner{
			"step-a": failAdapter,
		},
	}

	executor := NewDefaultPipelineExecutor(sa, WithEmitter(collector))

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "diamond-skip-test"},
		Steps: []Step{
			{ID: "step-a", Persona: "navigator", Optional: true, Exec: ExecConfig{Source: "optional root"}},
			{ID: "step-b", Persona: "navigator", Dependencies: []string{"step-a"}, Exec: ExecConfig{Source: "left branch"}},
			{ID: "step-c", Persona: "navigator", Dependencies: []string{"step-a"}, Exec: ExecConfig{Source: "right branch"}},
			{ID: "step-d", Persona: "navigator", Dependencies: []string{"step-b", "step-c"}, Exec: ExecConfig{Source: "diamond bottom"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	assert.NoError(t, err, "pipeline should succeed — A is optional")

	// None of B, C, D should have executed
	order := collector.GetStepExecutionOrder()
	assert.NotContains(t, order, "step-b", "step-b should not have executed")
	assert.NotContains(t, order, "step-c", "step-c should not have executed")
	assert.NotContains(t, order, "step-d", "step-d should not have executed")

	// All of B, C, D should have been skipped
	events := collector.GetEvents()
	skippedSteps := make(map[string]bool)
	for _, evt := range events {
		if evt.State == "skipped" {
			skippedSteps[evt.StepID] = true
		}
	}
	assert.True(t, skippedSteps["step-b"], "step-b should have been skipped")
	assert.True(t, skippedSteps["step-c"], "step-c should have been skipped")
	assert.True(t, skippedSteps["step-d"], "step-d should have been skipped")
}

// TestTransitiveSkip_IndependentPathsExecute verifies that steps on independent
// paths (not through a failed optional dependency) still execute normally.
//
//	A (optional, fails)    E (succeeds)
//	|                       |
//	B (skipped)            F (should execute)
func TestTransitiveSkip_IndependentPathsExecute(t *testing.T) {
	collector := testutil.NewEventCollector()
	successAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "ok"}`),
	)
	failAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithFailure(errors.New("optional failure")),
	)

	sa := &stepAwareAdapter{
		defaultAdapter: successAdapter,
		stepAdapters: map[string]adapter.AdapterRunner{
			"step-a": failAdapter,
		},
	}

	executor := NewDefaultPipelineExecutor(sa, WithEmitter(collector))

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "independent-paths-test"},
		Steps: []Step{
			{ID: "step-a", Persona: "navigator", Optional: true, Exec: ExecConfig{Source: "optional work"}},
			{ID: "step-b", Persona: "navigator", Dependencies: []string{"step-a"}, Exec: ExecConfig{Source: "depends on A"}},
			{ID: "step-e", Persona: "navigator", Exec: ExecConfig{Source: "independent root"}},
			{ID: "step-f", Persona: "navigator", Dependencies: []string{"step-e"}, Exec: ExecConfig{Source: "depends on E"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	assert.NoError(t, err, "pipeline should succeed — failed step is optional")

	// B should be skipped, but E and F should execute
	order := collector.GetStepExecutionOrder()
	assert.NotContains(t, order, "step-b", "step-b should not have executed")
	assert.Contains(t, order, "step-e", "step-e should have executed")
	assert.Contains(t, order, "step-f", "step-f should have executed")

	// Verify skip event for B
	events := collector.GetEvents()
	foundSkip := false
	for _, evt := range events {
		if evt.State == "skipped" && evt.StepID == "step-b" {
			foundSkip = true
			break
		}
	}
	assert.True(t, foundSkip, "step-b should have been skipped")

	// Verify E and F completed
	completedSteps := make(map[string]bool)
	for _, evt := range events {
		if evt.State == "completed" {
			completedSteps[evt.StepID] = true
		}
	}
	assert.True(t, completedSteps["step-e"], "step-e should have completed")
	assert.True(t, completedSteps["step-f"], "step-f should have completed")
}

// slowAdapter is a test adapter that waits for a delay before returning,
// respecting context cancellation.
type slowAdapter struct {
	delay  time.Duration
	result *adapter.AdapterResult
}

func (a *slowAdapter) Run(ctx context.Context, _ adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	select {
	case <-time.After(a.delay):
		return a.result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// TestConcurrentBatchCancellation verifies that when one step in a concurrent
// batch fails (non-optional, default on_failure: fail), the errgroup cancels
// remaining sibling steps and the pipeline returns a StepExecutionError.
//
// Steps A, B, C have no dependencies so they all land in the same ready batch.
// B fails immediately; A and C are slow and should be cancelled.
func TestConcurrentBatchCancellation(t *testing.T) {
	collector := testutil.NewEventCollector()

	slowResult := &adapter.AdapterResult{
		ExitCode:   0,
		Stdout:     strings.NewReader(`{"status": "ok"}`),
		TokensUsed: 100,
	}

	failAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithFailure(errors.New("step-b exploded")),
	)

	sa := &stepAwareAdapter{
		defaultAdapter: &slowAdapter{delay: 5 * time.Second, result: slowResult},
		stepAdapters: map[string]adapter.AdapterRunner{
			"step-b": failAdapter,
		},
	}

	executor := NewDefaultPipelineExecutor(sa, WithEmitter(collector))

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "concurrent-cancel-test"},
		Steps: []Step{
			{ID: "step-a", Persona: "navigator", Exec: ExecConfig{Source: "slow work A"}},
			{ID: "step-b", Persona: "navigator", Exec: ExecConfig{Source: "fails immediately"}},
			{ID: "step-c", Persona: "navigator", Exec: ExecConfig{Source: "slow work C"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	err := executor.Execute(ctx, p, m, "test")
	elapsed := time.Since(start)

	// Pipeline should fail
	require.Error(t, err, "pipeline should fail when a non-optional step fails")

	// The error should be a StepExecutionError (could be step-b's failure or a
	// sibling's context-cancelled error — errgroup returns whichever
	// goroutine finishes first).
	var stepErr *StepExecutionError
	require.True(t, errors.As(err, &stepErr), "error should be a StepExecutionError, got: %T", err)

	// Pipeline should complete quickly (not wait for slow steps to finish)
	assert.Less(t, elapsed, 4*time.Second, "pipeline should not wait for slow steps after cancellation")
}

// promptCapturingAdapter captures prompt from each step's AdapterRunConfig.
type promptCapturingAdapter struct {
	*adaptertest.MockAdapter
	mu      sync.Mutex
	prompts map[string]string // stepID (from workspace path) -> prompt
}

func newPromptCapturingAdapter(opts ...adaptertest.MockOption) *promptCapturingAdapter {
	return &promptCapturingAdapter{
		MockAdapter: adaptertest.NewMockAdapter(opts...),
		prompts:     make(map[string]string),
	}
}

func (a *promptCapturingAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	a.mu.Lock()
	// Use the last path component (step ID) as key
	parts := strings.Split(cfg.WorkspacePath, "/")
	stepID := parts[len(parts)-1]
	a.prompts[stepID] = cfg.Prompt
	a.mu.Unlock()
	return a.MockAdapter.Run(ctx, cfg)
}

func (a *promptCapturingAdapter) getPrompt(stepID string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.prompts[stepID]
}

// TestThreadSharing_TwoStepsSameThread verifies that step B in the same thread
// as step A receives step A's output in its prompt via THREAD CONTEXT header.
func TestThreadSharing_TwoStepsSameThread(t *testing.T) {
	capturing := newPromptCapturingAdapter(
		adaptertest.WithStdoutJSON(`{"status":"ok"}`),
		adaptertest.WithTokensUsed(100),
	)

	executor := NewDefaultPipelineExecutor(capturing)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "thread-test"},
		Steps: []Step{
			{
				ID:      "implement",
				Persona: "navigator",
				Thread:  "impl",
				Exec:    ExecConfig{Source: "Implement the feature"},
			},
			{
				ID:           "fix",
				Persona:      "navigator",
				Thread:       "impl",
				Dependencies: []string{"implement"},
				Exec:         ExecConfig{Source: "Fix the failing tests"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	// The fix step should contain THREAD CONTEXT with the implement step's output
	fixPrompt := capturing.getPrompt("fix")
	assert.Contains(t, fixPrompt, "## THREAD CONTEXT", "fix step should have thread context header")
	assert.Contains(t, fixPrompt, "Fix the failing tests", "fix step should contain its own prompt")
}

// TestThreadIsolation_DifferentThreads verifies that steps in different threads
// do NOT share transcripts.
func TestThreadIsolation_DifferentThreads(t *testing.T) {
	capturing := newPromptCapturingAdapter(
		adaptertest.WithStdoutJSON(`{"status":"ok"}`),
		adaptertest.WithTokensUsed(100),
	)

	executor := NewDefaultPipelineExecutor(capturing)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "isolation-test"},
		Steps: []Step{
			{
				ID:      "step-a",
				Persona: "navigator",
				Thread:  "thread-1",
				Exec:    ExecConfig{Source: "Step A work"},
			},
			{
				ID:           "step-b",
				Persona:      "navigator",
				Thread:       "thread-2",
				Dependencies: []string{"step-a"},
				Exec:         ExecConfig{Source: "Step B work"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	// step-b is in thread-2, so it should NOT have step-a's content (thread-1)
	stepBPrompt := capturing.getPrompt("step-b")
	// Thread-2 has no prior entries, so no THREAD CONTEXT should be injected
	assert.NotContains(t, stepBPrompt, "## THREAD CONTEXT",
		"step-b in different thread should not receive thread context from thread-1")
}

// TestNoThread_FreshMemory verifies that steps without a thread attribute
// do NOT receive any thread context (fresh memory behavior).
func TestNoThread_FreshMemory(t *testing.T) {
	capturing := newPromptCapturingAdapter(
		adaptertest.WithStdoutJSON(`{"status":"ok"}`),
		adaptertest.WithTokensUsed(100),
	)

	executor := NewDefaultPipelineExecutor(capturing)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "fresh-test"},
		Steps: []Step{
			{
				ID:      "step-a",
				Persona: "navigator",
				Thread:  "impl",
				Exec:    ExecConfig{Source: "Step A with thread"},
			},
			{
				ID:           "review",
				Persona:      "navigator",
				Dependencies: []string{"step-a"},
				Exec:         ExecConfig{Source: "Review the work"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	// review step has no thread, so it should not have any thread context
	reviewPrompt := capturing.getPrompt("review")
	assert.NotContains(t, reviewPrompt, "## THREAD CONTEXT",
		"unthreaded step should not receive thread context")
}

// TestThreadValidation_InvalidFidelity verifies the executor rejects invalid fidelity values.
func TestThreadValidation_InvalidFidelity(t *testing.T) {
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status":"ok"}`),
		adaptertest.WithTokensUsed(100),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "invalid-fidelity"},
		Steps: []Step{
			{
				ID:       "step-a",
				Persona:  "navigator",
				Thread:   "impl",
				Fidelity: "bogus",
				Exec:     ExecConfig{Source: "do work"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid fidelity value")
}

// TestExecutor_GateStep_AutoApprove verifies that a pipeline with a gate step
// using auto-approve mode completes all steps successfully. The gate has choices
// with a default, and auto-approve selects the default (approve -> implement).
func TestExecutor_GateStep_AutoApprove(t *testing.T) {
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
		adaptertest.WithTokensUsed(500),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
		WithAutoApprove(true),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	// Approve choice has empty target — just proceed naturally via DAG ordering.
	// Revise targets "plan" (backward reference). Abort targets "_fail".
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "gate-auto-approve-test"},
		Steps: []Step{
			{ID: "plan", Persona: "navigator", Exec: ExecConfig{Source: "create a plan"}},
			{
				ID:           "approve-plan",
				Dependencies: []string{"plan"},
				Gate: &GateConfig{
					Type: "approval",
					Choices: []GateChoice{
						{Label: "Approve", Key: "a"},
						{Label: "Revise", Key: "r", Target: "plan"},
						{Label: "Abort", Key: "q", Target: "_fail"},
					},
					Default: "a",
				},
			},
			{ID: "implement", Persona: "navigator", Dependencies: []string{"approve-plan"}, Exec: ExecConfig{Source: "implement the plan"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test auto-approve gate")
	require.NoError(t, err)

	// Verify persona steps executed in correct order
	order := collector.GetStepExecutionOrder()
	posP := indexOfInSlice(order, "plan")
	posI := indexOfInSlice(order, "implement")
	assert.True(t, posP >= 0, "plan should be in execution order")
	assert.True(t, posI >= 0, "implement should be in execution order")
	assert.True(t, posP < posI, "plan should execute before implement")

	// Verify gate step completed (check events for gate step completion)
	events := collector.GetEvents()
	gateCompleted := false
	for _, ev := range events {
		if ev.StepID == "approve-plan" && ev.State == "completed" {
			gateCompleted = true
			break
		}
	}
	assert.True(t, gateCompleted, "gate step approve-plan should have completed")

	// Verify gate resolved event was emitted
	assert.True(t, collector.HasEventWithState("gate_resolved"), "should have gate_resolved event")
}

// TestExecutor_GateStep_Abort verifies that a pipeline aborts when the gate handler
// returns a choice targeting _fail. The pipeline should fail with a gateAbortError.
func TestExecutor_GateStep_Abort(t *testing.T) {
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
		adaptertest.WithTokensUsed(500),
	)

	// Custom handler that always selects "abort" (the _fail target)
	abortHandler := &staticGateHandler{
		choice: "q",
	}

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
		WithGateHandler(abortHandler),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "gate-abort-test"},
		Steps: []Step{
			{ID: "plan", Persona: "navigator", Exec: ExecConfig{Source: "create a plan"}},
			{
				ID:           "approve-plan",
				Dependencies: []string{"plan"},
				Gate: &GateConfig{
					Type: "approval",
					Choices: []GateChoice{
						{Label: "Approve", Key: "a"},
						{Label: "Revise", Key: "r", Target: "plan"},
						{Label: "Abort", Key: "q", Target: "_fail"},
					},
					Default: "a",
				},
			},
			{ID: "implement", Persona: "navigator", Dependencies: []string{"approve-plan"}, Exec: ExecConfig{Source: "implement the plan"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test abort gate")
	require.Error(t, err, "pipeline should fail when gate aborts")

	// The executor wraps gateAbortError inside a StepExecutionError
	var stepErr *StepExecutionError
	require.True(t, errors.As(err, &stepErr), "error should be a StepExecutionError")
	assert.Equal(t, "approve-plan", stepErr.StepID, "failed step should be approve-plan")

	// Unwrap to find the gateAbortError
	var abortErr *gateAbortError
	require.True(t, errors.As(err, &abortErr), "error chain should contain gateAbortError")
	assert.Equal(t, "approve-plan", abortErr.StepID, "abort error should reference approve-plan")
}

// staticGateHandler is a test helper that returns a fixed choice, optionally
// cycling through multiple choices on successive calls.
type staticGateHandler struct {
	choice  string   // single choice key (used when choices is empty)
	choices []string // multiple choice keys, cycled in order
	calls   int
}

func (h *staticGateHandler) Prompt(_ context.Context, gate *GateConfig) (*GateDecision, error) {
	key := h.choice
	if len(h.choices) > 0 {
		key = h.choices[h.calls%len(h.choices)]
	}
	h.calls++

	choice := gate.FindChoiceByKey(key)
	if choice == nil {
		return nil, fmt.Errorf("staticGateHandler: key %q not found in gate choices", key)
	}
	return &GateDecision{
		Choice:    choice.Key,
		Label:     choice.Label,
		Timestamp: time.Now(),
		Target:    choice.Target,
	}, nil
}

// TestExecutor_GateStep_ChoiceRouting_Revise verifies the revision loop pattern:
// the gate handler returns "revise" on the first call (routing back to "plan"),
// then "approve" on the second call (routing forward to "implement").
// The "plan" step should execute twice.
func TestExecutor_GateStep_ChoiceRouting_Revise(t *testing.T) {
	collector := testutil.NewEventCollector()

	// Track how many times each step executes via the adapter
	var planCount int32
	var implCount int32

	countingAdapter := &stepAwareAdapter{
		defaultAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(500),
		),
		onStart: func(stepID string) {
			switch stepID {
			case "plan":
				atomic.AddInt32(&planCount, 1)
			case "implement":
				atomic.AddInt32(&implCount, 1)
			}
		},
	}

	// First call: revise (routes back to plan); second call: approve (routes to implement)
	reviseHandler := &staticGateHandler{
		choices: []string{"r", "a"},
	}

	executor := NewDefaultPipelineExecutor(countingAdapter,
		WithEmitter(collector),
		WithGateHandler(reviseHandler),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "gate-revise-test"},
		Steps: []Step{
			{ID: "plan", Persona: "navigator", Exec: ExecConfig{Source: "create a plan"}},
			{
				ID:           "approve",
				Dependencies: []string{"plan"},
				Gate: &GateConfig{
					Type: "approval",
					Choices: []GateChoice{
						{Label: "Approve", Key: "a"},
						{Label: "Revise", Key: "r", Target: "plan"},
						{Label: "Abort", Key: "q", Target: "_fail"},
					},
					Default: "a",
				},
			},
			{ID: "implement", Persona: "navigator", Dependencies: []string{"approve"}, Exec: ExecConfig{Source: "implement the plan"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test revision loop")
	require.NoError(t, err)

	// Plan should have executed twice (initial + after revise routing)
	assert.Equal(t, int32(2), atomic.LoadInt32(&planCount), "plan should execute twice due to revision loop")

	// Implement should have executed once (after approve on second gate pass)
	assert.Equal(t, int32(1), atomic.LoadInt32(&implCount), "implement should execute once after final approval")

	// The gate handler should have been called exactly twice
	assert.Equal(t, 2, reviseHandler.calls, "gate handler should be called twice (revise then approve)")
}

// TestExecutor_GateStep_TemplateVars verifies that the gate decision is stored
// in the PipelineContext after a gate step completes with auto-approve.
func TestExecutor_GateStep_TemplateVars(t *testing.T) {
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
		adaptertest.WithTokensUsed(500),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
		WithAutoApprove(true),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "gate-template-vars-test"},
		Steps: []Step{
			{
				ID: "approve",
				Gate: &GateConfig{
					Type: "approval",
					Choices: []GateChoice{
						{Label: "Approve", Key: "a"},
						{Label: "Reject", Key: "r", Target: "_fail"},
					},
					Default: "a",
				},
			},
			{ID: "next", Persona: "navigator", Dependencies: []string{"approve"}, Exec: ExecConfig{Source: "do next thing"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test gate template vars")
	require.NoError(t, err)

	// Access the pipeline execution to check the context
	// The executor stores the last execution internally
	executor.mu.RLock()
	lastExec := executor.lastExecution
	executor.mu.RUnlock()

	require.NotNil(t, lastExec, "last execution should be available")
	require.NotNil(t, lastExec.Context, "pipeline context should be set")
	require.NotNil(t, lastExec.Context.GateDecisions, "gate decisions should be populated")

	decision, ok := lastExec.Context.GateDecisions["approve"]
	require.True(t, ok, "gate decision for 'approve' step should exist")
	assert.Equal(t, "a", decision.Choice, "decision choice should be the default 'a'")
	assert.Equal(t, "Approve", decision.Label, "decision label should be 'Approve'")
	assert.Empty(t, decision.Target, "decision target should be empty (natural DAG flow)")
}

// TestExecuteStep_FailureClassification_Transient verifies that a transient failure
// (rate limit StepError) is classified correctly and the step retries.
func TestExecuteStep_FailureClassification_Transient(t *testing.T) {
	collector := testutil.NewEventCollector()
	store := newAttemptTrackingStore()

	// Fails once with a rate limit StepError, then succeeds.
	failAdapter := newCountingFailAdapter(1, adapter.NewStepError(adapter.FailureReasonRateLimit, errors.New("rate limited"), 0, ""))

	executor := NewDefaultPipelineExecutor(failAdapter,
		WithEmitter(collector),
		WithStateStore(store),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "transient-classification-test"},
		Steps: []Step{
			{
				ID:      "step-1",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "do work"},
				Retry: RetryConfig{
					MaxAttempts: 2,
					BaseDelay:   "1ms",
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err, "should succeed on second attempt after transient failure")

	// Should have been called twice: 1 failure + 1 success
	assert.Equal(t, 2, failAdapter.getCallCount(), "adapter should be called twice")

	// Verify the failed attempt was classified as transient
	attempts := store.getAttempts()
	var failedAttempts []state.StepAttemptRecord
	for _, a := range attempts {
		if a.State == stateFailed {
			failedAttempts = append(failedAttempts, a)
		}
	}
	require.Len(t, failedAttempts, 1, "should have exactly 1 failed attempt record")
	assert.Equal(t, "transient", failedAttempts[0].FailureClass, "failure class should be transient for rate limit error")
}

// TestExecuteStep_FailureClassification_Deterministic_SkipsRetry verifies that a
// deterministic failure (e.g. invalid API key) skips remaining retries immediately.
func TestExecuteStep_FailureClassification_Deterministic_SkipsRetry(t *testing.T) {
	collector := testutil.NewEventCollector()
	store := newAttemptTrackingStore()

	// Always fails with a deterministic error message.
	failAdapter := newCountingFailAdapter(10, errors.New("invalid api key"))

	executor := NewDefaultPipelineExecutor(failAdapter,
		WithEmitter(collector),
		WithStateStore(store),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "deterministic-skip-test"},
		Steps: []Step{
			{
				ID:      "step-1",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "do work"},
				Retry: RetryConfig{
					MaxAttempts: 3,
					BaseDelay:   "1ms",
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.Error(t, err, "should fail with deterministic error")

	// Deterministic failures are not retryable — only 1 attempt should be made
	assert.Equal(t, 1, failAdapter.getCallCount(), "adapter should only be called once for deterministic failure")

	// Verify the failed attempt was classified as deterministic
	attempts := store.getAttempts()
	var failedAttempts []state.StepAttemptRecord
	for _, a := range attempts {
		if a.State == stateFailed {
			failedAttempts = append(failedAttempts, a)
		}
	}
	require.Len(t, failedAttempts, 1, "should have exactly 1 failed attempt record")
	assert.Equal(t, "deterministic", failedAttempts[0].FailureClass, "failure class should be deterministic")

	// Verify event was emitted about skipping retries
	events := collector.GetEvents()
	foundSkipMsg := false
	for _, e := range events {
		if strings.Contains(e.Message, "non-retryable failure class") {
			foundSkipMsg = true
			break
		}
	}
	assert.True(t, foundSkipMsg, "should emit event about non-retryable failure class")
}

// TestExecuteStep_CircuitBreaker_TripsOnRepeatedFailures verifies that the circuit
// breaker trips when the same failure fingerprint repeats beyond the configured limit.
func TestExecuteStep_CircuitBreaker_TripsOnRepeatedFailures(t *testing.T) {
	collector := testutil.NewEventCollector()
	store := newAttemptTrackingStore()

	// Always fails with a test_failure class error
	failAdapter := newCountingFailAdapter(10, errors.New("test failed: compile error"))

	executor := NewDefaultPipelineExecutor(failAdapter,
		WithEmitter(collector),
		WithStateStore(store),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)
	m.Runtime.CircuitBreaker = manifest.CircuitBreakerConfig{
		Limit:          2,
		TrackedClasses: []string{"test_failure", "deterministic", "contract_failure"},
	}

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "circuit-breaker-test"},
		Steps: []Step{
			{
				ID:      "step-1",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "do work"},
				Retry: RetryConfig{
					MaxAttempts: 5,
					BaseDelay:   "1ms",
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.Error(t, err, "should fail due to circuit breaker tripping")

	// The circuit breaker has limit=2, so the step should trip after 2 failures,
	// not exhaust all 5 attempts.
	callCount := failAdapter.getCallCount()
	assert.Less(t, callCount, 5, "circuit breaker should trip before all 5 attempts")

	// Verify circuit breaker tripped event
	events := collector.GetEvents()
	foundCircuitBreaker := false
	for _, e := range events {
		if strings.Contains(e.Message, "circuit breaker tripped") {
			foundCircuitBreaker = true
			break
		}
	}
	assert.True(t, foundCircuitBreaker, "should emit circuit breaker tripped event")
}

// TestExecuteStep_FailureClassification_Canceled verifies that a pre-canceled context
// is classified as "canceled" and the step does not retry.
func TestExecuteStep_FailureClassification_Canceled(t *testing.T) {
	collector := testutil.NewEventCollector()
	store := newAttemptTrackingStore()

	// Use a mock adapter with a simulated delay so that it respects ctx.Done()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithSimulatedDelay(10 * time.Second),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
		WithStateStore(store),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "canceled-classification-test"},
		Steps: []Step{
			{
				ID:      "step-1",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "do work"},
				Retry: RetryConfig{
					MaxAttempts: 3,
					BaseDelay:   "1ms",
				},
			},
		},
	}

	// Cancel the context before executing
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.Error(t, err, "should fail with canceled context")

	// Verify the failed attempt was classified as canceled
	attempts := store.getAttempts()
	var failedAttempts []state.StepAttemptRecord
	for _, a := range attempts {
		if a.State == stateFailed {
			failedAttempts = append(failedAttempts, a)
		}
	}
	require.NotEmpty(t, failedAttempts, "should have at least 1 failed attempt record")
	assert.Equal(t, "canceled", failedAttempts[0].FailureClass, "failure class should be canceled")
}

// TestThreadedSteps_FreshFidelity verifies that fidelity: fresh suppresses transcript injection.
func TestThreadedSteps_FreshFidelity(t *testing.T) {
	capAdapter := &allConfigCapturingAdapter{
		MockAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		),
	}

	executor := NewDefaultPipelineExecutor(capAdapter)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "thread-fresh-test"},
		Steps: []Step{
			{
				ID:      "implement",
				Persona: "navigator",
				Thread:  "impl",
				Exec:    ExecConfig{Source: "implement"},
			},
			{
				ID:           "fix",
				Persona:      "navigator",
				Thread:       "impl",
				Fidelity:     FidelityFresh,
				Dependencies: []string{"implement"},
				Exec:         ExecConfig{Source: "fix"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	configs := capAdapter.getConfigs()
	require.Len(t, configs, 2)

	// Fix step has fidelity: fresh, so it should NOT receive prior context
	assert.NotContains(t, configs[1].Prompt, "Prior Conversation Context",
		"fidelity:fresh should suppress thread transcript injection")
}

// allConfigCapturingAdapter captures AdapterRunConfig for every call, keyed by step workspace path.
type allConfigCapturingAdapter struct {
	*adaptertest.MockAdapter
	mu      sync.Mutex
	configs []adapter.AdapterRunConfig
}

func (a *allConfigCapturingAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	a.mu.Lock()
	a.configs = append(a.configs, cfg)
	a.mu.Unlock()
	return a.MockAdapter.Run(ctx, cfg)
}

func (a *allConfigCapturingAdapter) getConfigs() []adapter.AdapterRunConfig {
	a.mu.Lock()
	defer a.mu.Unlock()
	result := make([]adapter.AdapterRunConfig, len(a.configs))
	copy(result, a.configs)
	return result
}

// TestIterateInDAG_Sequential verifies that the iterate composition primitive
// fans out over items and runs a sub-pipeline for each.
func TestIterateInDAG_Sequential(t *testing.T) {
	collector := testutil.NewEventCollector()
	capAdapter := &allConfigCapturingAdapter{
		MockAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"findings": []}`),
			adaptertest.WithTokensUsed(100),
		),
	}

	executor := NewDefaultPipelineExecutor(capAdapter, WithEmitter(collector))

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	// Create two child pipeline files on disk
	pipelinesDir := filepath.Join(tmpDir, ".agents", "pipelines")
	require.NoError(t, os.MkdirAll(pipelinesDir, 0755))

	childYAML := `kind: WavePipeline
metadata:
  name: %s
steps:
  - id: scan
    persona: navigator
    exec:
      type: prompt
      source: "scan for %s"
`
	for _, name := range []string{"audit-alpha", "audit-beta"} {
		path := filepath.Join(pipelinesDir, name+".yaml")
		require.NoError(t, os.WriteFile(path, []byte(fmt.Sprintf(childYAML, name, name)), 0644))
	}

	// Override CWD so the executor finds .agents/pipelines/ relative to tmpDir
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "iterate-test"},
		Steps: []Step{
			{
				ID:          "run-audits",
				SubPipeline: "{{ item }}",
				SubInput:    "{{ input }}",
				Iterate: &IterateConfig{
					Over: `["audit-alpha", "audit-beta"]`,
					Mode: "sequential",
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test-scope")
	require.NoError(t, err)

	// Both child pipelines should have been executed (each has 1 step)
	configs := capAdapter.getConfigs()
	assert.GreaterOrEqual(t, len(configs), 2,
		"adapter should be called at least twice (once per iterate item)")
}

// TestIterateInDAG_Parallel verifies parallel iterate runs items concurrently.
func TestIterateInDAG_Parallel(t *testing.T) {
	collector := testutil.NewEventCollector()
	capAdapter := &allConfigCapturingAdapter{
		MockAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"findings": []}`),
			adaptertest.WithTokensUsed(100),
		),
	}

	executor := NewDefaultPipelineExecutor(capAdapter, WithEmitter(collector))

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	pipelinesDir := filepath.Join(tmpDir, ".agents", "pipelines")
	require.NoError(t, os.MkdirAll(pipelinesDir, 0755))

	childYAML := `kind: WavePipeline
metadata:
  name: %s
steps:
  - id: scan
    persona: navigator
    exec:
      type: prompt
      source: "scan"
`
	for _, name := range []string{"scan-a", "scan-b", "scan-c"} {
		path := filepath.Join(pipelinesDir, name+".yaml")
		require.NoError(t, os.WriteFile(path, []byte(fmt.Sprintf(childYAML, name)), 0644))
	}

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "parallel-iterate-test"},
		Steps: []Step{
			{
				ID:          "fan-out",
				SubPipeline: "{{ item }}",
				Iterate: &IterateConfig{
					Over:          `["scan-a", "scan-b", "scan-c"]`,
					Mode:          "parallel",
					MaxConcurrent: 2,
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	configs := capAdapter.getConfigs()
	assert.GreaterOrEqual(t, len(configs), 3,
		"adapter should be called at least 3 times (once per parallel item)")
}

// TestIterateInDAG_CollectsOutputs verifies that after an iterate step completes,
// the collected output is registered under the step's ID in ArtifactPaths so
// downstream steps can reference {{ stepID.output }}.
func TestIterateInDAG_CollectsOutputs(t *testing.T) {
	collector := testutil.NewEventCollector()
	capAdapter := &allConfigCapturingAdapter{
		MockAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"findings": ["issue-1"]}`),
			adaptertest.WithTokensUsed(100),
		),
	}

	executor := NewDefaultPipelineExecutor(capAdapter, WithEmitter(collector))

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	pipelinesDir := filepath.Join(tmpDir, ".agents", "pipelines")
	require.NoError(t, os.MkdirAll(pipelinesDir, 0755))

	// Child pipelines that produce a stdout artifact so it gets stored
	childYAML := `kind: WavePipeline
metadata:
  name: %s
steps:
  - id: scan
    persona: navigator
    exec:
      type: prompt
      source: "scan"
    output_artifacts:
      - name: result
        source: stdout
`
	for _, name := range []string{"audit-alpha", "audit-beta", "audit-gamma"} {
		path := filepath.Join(pipelinesDir, name+".yaml")
		require.NoError(t, os.WriteFile(path, []byte(fmt.Sprintf(childYAML, name)), 0644))
	}

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "iterate-collect-test"},
		Steps: []Step{
			{
				ID:          "run-audits",
				SubPipeline: "{{ item }}",
				Iterate: &IterateConfig{
					Over: `["audit-alpha", "audit-beta", "audit-gamma"]`,
					Mode: "sequential",
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	// Verify that ArtifactPaths has the collected output under the iterate step's ID
	exec := executor.LastExecution()
	require.NotNil(t, exec)

	collectedPath, ok := exec.ArtifactPaths["run-audits:collected-output"]
	assert.True(t, ok, "ArtifactPaths should contain run-audits:collected-output")
	assert.NotEmpty(t, collectedPath, "collected output path should not be empty")

	// Read the collected file and verify it's a JSON array
	data, err := os.ReadFile(collectedPath)
	require.NoError(t, err)

	var collected []json.RawMessage
	require.NoError(t, json.Unmarshal(data, &collected))
	assert.Len(t, collected, 3, "collected output should have 3 entries")

	// Each entry should be the stdout JSON from the child pipeline
	for i, entry := range collected {
		var parsed map[string]interface{}
		err := json.Unmarshal(entry, &parsed)
		require.NoError(t, err, "entry %d should be valid JSON", i)
		assert.Contains(t, parsed, "findings", "entry %d should contain findings key", i)
	}
}

// TestIterateInDAG_Parallel_CollectsOutputs verifies parallel iterate also collects outputs.
func TestIterateInDAG_Parallel_CollectsOutputs(t *testing.T) {
	collector := testutil.NewEventCollector()
	capAdapter := &allConfigCapturingAdapter{
		MockAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "done"}`),
			adaptertest.WithTokensUsed(100),
		),
	}

	executor := NewDefaultPipelineExecutor(capAdapter, WithEmitter(collector))

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	pipelinesDir := filepath.Join(tmpDir, ".agents", "pipelines")
	require.NoError(t, os.MkdirAll(pipelinesDir, 0755))

	childYAML := `kind: WavePipeline
metadata:
  name: %s
steps:
  - id: process
    persona: navigator
    exec:
      type: prompt
      source: "process"
    output_artifacts:
      - name: output
        source: stdout
`
	for _, name := range []string{"job-a", "job-b"} {
		path := filepath.Join(pipelinesDir, name+".yaml")
		require.NoError(t, os.WriteFile(path, []byte(fmt.Sprintf(childYAML, name)), 0644))
	}

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "parallel-collect-test"},
		Steps: []Step{
			{
				ID:          "fan-out",
				SubPipeline: "{{ item }}",
				Iterate: &IterateConfig{
					Over: `["job-a", "job-b"]`,
					Mode: "parallel",
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	exec := executor.LastExecution()
	require.NotNil(t, exec)

	collectedPath, ok := exec.ArtifactPaths["fan-out:collected-output"]
	assert.True(t, ok, "ArtifactPaths should contain fan-out:collected-output")

	data, err := os.ReadFile(collectedPath)
	require.NoError(t, err)

	var collected []json.RawMessage
	require.NoError(t, json.Unmarshal(data, &collected))
	assert.Len(t, collected, 2, "collected output should have 2 entries")
}

// TestIterateInDAG_OutputResolvesInAggregate verifies the end-to-end flow:
// iterate step produces collected output, then aggregate step references it
// via {{ stepID.output }}.
func TestIterateInDAG_OutputResolvesInAggregate(t *testing.T) {
	collector := testutil.NewEventCollector()
	capAdapter := &allConfigCapturingAdapter{
		MockAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"findings": ["f1"]}`),
			adaptertest.WithTokensUsed(100),
		),
	}

	executor := NewDefaultPipelineExecutor(capAdapter, WithEmitter(collector))

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	pipelinesDir := filepath.Join(tmpDir, ".agents", "pipelines")
	require.NoError(t, os.MkdirAll(pipelinesDir, 0755))

	childYAML := `kind: WavePipeline
metadata:
  name: %s
steps:
  - id: scan
    persona: navigator
    exec:
      type: prompt
      source: "scan"
    output_artifacts:
      - name: result
        source: stdout
`
	for _, name := range []string{"audit-a", "audit-b"} {
		path := filepath.Join(pipelinesDir, name+".yaml")
		require.NoError(t, os.WriteFile(path, []byte(fmt.Sprintf(childYAML, name)), 0644))
	}

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	outputPath := filepath.Join(tmpDir, ".agents", "output", "merged.json")

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "iterate-aggregate-test"},
		Steps: []Step{
			{
				ID:          "run-audits",
				SubPipeline: "{{ item }}",
				Iterate: &IterateConfig{
					Over: `["audit-a", "audit-b"]`,
					Mode: "sequential",
				},
			},
			{
				ID:           "merge-findings",
				Dependencies: []string{"run-audits"},
				Aggregate: &AggregateConfig{
					From:     "{{ run-audits.output }}",
					Into:     outputPath,
					Strategy: "merge_arrays",
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	// Verify the aggregate output was written
	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	// The merged output should be a flattened array from both child pipelines
	var merged []json.RawMessage
	require.NoError(t, json.Unmarshal(data, &merged))
	assert.GreaterOrEqual(t, len(merged), 2,
		"merged output should contain entries from both child pipelines")
}

// TestAggregateInDAG verifies the aggregate primitive merges output to a file.
func TestAggregateInDAG(t *testing.T) {
	collector := testutil.NewEventCollector()
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "ok"}`),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter, WithEmitter(collector))

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	outputPath := filepath.Join(tmpDir, ".agents", "output", "merged.json")

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "aggregate-test"},
		Steps: []Step{
			{
				ID: "merge",
				Aggregate: &AggregateConfig{
					From:     `[[1,2],[3,4]]`,
					Into:     outputPath,
					Strategy: "merge_arrays",
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Equal(t, "[1,2,3,4]", string(data))
}

// TestBranchInDAG verifies the branch primitive routes to the matching pipeline.
func TestBranchInDAG(t *testing.T) {
	collector := testutil.NewEventCollector()
	capAdapter := &allConfigCapturingAdapter{
		MockAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "ok"}`),
			adaptertest.WithTokensUsed(100),
		),
	}

	executor := NewDefaultPipelineExecutor(capAdapter, WithEmitter(collector))

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	pipelinesDir := filepath.Join(tmpDir, ".agents", "pipelines")
	require.NoError(t, os.MkdirAll(pipelinesDir, 0755))

	childYAML := `kind: WavePipeline
metadata:
  name: hotfix
steps:
  - id: fix
    persona: navigator
    exec:
      type: prompt
      source: "fix it"
`
	require.NoError(t, os.WriteFile(filepath.Join(pipelinesDir, "hotfix.yaml"), []byte(childYAML), 0644))

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "branch-test"},
		Steps: []Step{
			{
				ID: "route",
				Branch: &BranchConfig{
					On:    "high",
					Cases: map[string]string{"high": "hotfix", "low": "skip"},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	configs := capAdapter.getConfigs()
	assert.GreaterOrEqual(t, len(configs), 1,
		"hotfix pipeline should have been executed")
}

// TestBranchInDAG_Skip verifies skip branches complete without running a pipeline.
func TestBranchInDAG_Skip(t *testing.T) {
	collector := testutil.NewEventCollector()
	capAdapter := &allConfigCapturingAdapter{
		MockAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "ok"}`),
		),
	}

	executor := NewDefaultPipelineExecutor(capAdapter, WithEmitter(collector))

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "branch-skip-test"},
		Steps: []Step{
			{
				ID: "route",
				Branch: &BranchConfig{
					On:    "low",
					Cases: map[string]string{"high": "hotfix", "low": "skip"},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err)

	configs := capAdapter.getConfigs()
	assert.Equal(t, 0, len(configs),
		"skip branch should not invoke any adapter")
}

// TestRetryInjectsContractFailureContext verifies that when a step's contract
// validation fails and the step retries, the next attempt's prompt includes
// the contract failure details so the agent can fix the specific error.
func TestRetryInjectsContractFailureContext(t *testing.T) {
	// Use a counting adapter that captures every prompt. Both attempts
	// succeed at the adapter level — the failure comes from the contract.
	capAdapter := &countingFailAdapter{
		failCount:   0, // adapter always succeeds
		successMock: adaptertest.NewMockAdapter(adaptertest.WithStdoutJSON(`{"status":"ok"}`)),
	}

	collector := testutil.NewEventCollector()
	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	executor := NewDefaultPipelineExecutor(capAdapter,
		WithEmitter(collector),
	)

	// Create a shell script that fails on the first call and succeeds on
	// the second. Uses a counter file to track invocations.
	counterFile := filepath.Join(tmpDir, "contract_counter")
	scriptFile := filepath.Join(tmpDir, "contract_check.sh")
	scriptContent := fmt.Sprintf(`#!/bin/sh
count=$(cat "%s" 2>/dev/null || echo 0)
count=$((count+1))
echo $count > "%s"
if [ $count -le 1 ]; then
  echo "FAIL: TestWidget expected 42 got 0"
  exit 1
fi
`, counterFile, counterFile)
	require.NoError(t, os.WriteFile(scriptFile, []byte(scriptContent), 0755))

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "retry-contract-ctx"},
		Steps: []Step{
			{
				ID:      "step-1",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "implement the feature"},
				Retry: RetryConfig{
					MaxAttempts: 2,
					BaseDelay:   "1ms",
				},
				Handover: HandoverConfig{
					Contract: ContractConfig{
						Type:      "test_suite",
						Command:   scriptFile,
						OnFailure: OnFailureFail,
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test input")
	assert.NoError(t, err, "pipeline should succeed on second attempt after contract passes")

	// The adapter should have been called exactly twice
	configs := capAdapter.getLastConfigs()
	require.Equal(t, 2, len(configs), "adapter should be called twice (first attempt + retry)")

	// First attempt: no retry context in prompt
	assert.NotContains(t, configs[0].Prompt, "RETRY CONTEXT",
		"first attempt should NOT have retry context")

	// Second attempt: must contain retry context with contract failure details
	secondPrompt := configs[1].Prompt
	assert.Contains(t, secondPrompt, "RETRY CONTEXT",
		"second attempt should have RETRY CONTEXT header")
	assert.Contains(t, secondPrompt, "attempt 2 of 2",
		"second attempt should show attempt number")
	assert.Contains(t, secondPrompt, "Contract Validation Errors",
		"second attempt should contain contract validation errors section")
	assert.Contains(t, secondPrompt, "TestWidget",
		"second attempt should contain the specific test failure output")
	assert.Contains(t, secondPrompt, "Fix the specific failure above",
		"second attempt should instruct agent not to start from scratch")
}

// TestResolveWorkspaceStepRefs_ArtifactsNamedField verifies that
// {{ steps.STEP_ID.artifacts.ARTIFACT_NAME.JSON_PATH }} is resolved from
// execution.ArtifactPaths at workspace creation time.
func TestResolveWorkspaceStepRefs_ArtifactsNamedField(t *testing.T) {
	executor := NewDefaultPipelineExecutor(&adaptertest.MockAdapter{})

	tmpDir := t.TempDir()

	// Write a JSON artifact file that the prior step would have produced.
	artFile := filepath.Join(tmpDir, "review-findings.json")
	err := os.WriteFile(artFile, []byte(`{"head_branch":"feat/my-feature","number":42}`), 0644)
	require.NoError(t, err)

	execution := &PipelineExecution{
		Pipeline:       &Pipeline{Metadata: PipelineMetadata{Name: "ws-stepref-test"}},
		Manifest:       &manifest.Manifest{},
		WorkspacePaths: make(map[string]string),
		WorktreePaths:  make(map[string]*WorktreeInfo),
		ArtifactPaths:  map[string]string{"fetch-review:review-findings": artFile},
		Status:         &PipelineStatus{ID: "ws-stepref-test"},
		Context:        NewPipelineContext("ws-stepref-test", "ws-stepref-test", "apply-fixes"),
	}

	// Resolve branch template referencing prior step artifact field.
	branch, err := executor.resolveWorkspaceStepRefs(
		"{{ steps.fetch-review.artifacts.review-findings.head_branch }}", execution)
	require.NoError(t, err)
	assert.Equal(t, "feat/my-feature", branch)
}

// TestResolveWorkspaceStepRefs_Output verifies that
// {{ steps.STEP_ID.output.JSON_FIELD }} is resolved from the first artifact.
func TestResolveWorkspaceStepRefs_Output(t *testing.T) {
	executor := NewDefaultPipelineExecutor(&adaptertest.MockAdapter{})

	tmpDir := t.TempDir()
	artFile := filepath.Join(tmpDir, "step-output.json")
	err := os.WriteFile(artFile, []byte(`{"branch":"fix/issue-123"}`), 0644)
	require.NoError(t, err)

	execution := &PipelineExecution{
		Pipeline:       &Pipeline{Metadata: PipelineMetadata{Name: "ws-output-test"}},
		Manifest:       &manifest.Manifest{},
		WorkspacePaths: make(map[string]string),
		WorktreePaths:  make(map[string]*WorktreeInfo),
		ArtifactPaths:  map[string]string{"fetch-pr:pr-meta": artFile},
		Status:         &PipelineStatus{ID: "ws-output-test"},
		Context:        NewPipelineContext("ws-output-test", "ws-output-test", "apply-fixes"),
	}

	branch, err := executor.resolveWorkspaceStepRefs(
		"{{ steps.fetch-pr.output.branch }}", execution)
	require.NoError(t, err)
	assert.Equal(t, "fix/issue-123", branch)
}

// TestResolveWorkspaceStepRefs_MissingStep verifies a clear error when the
// referenced step artifact does not exist.
func TestResolveWorkspaceStepRefs_MissingStep(t *testing.T) {
	executor := NewDefaultPipelineExecutor(&adaptertest.MockAdapter{})

	execution := &PipelineExecution{
		Pipeline:       &Pipeline{Metadata: PipelineMetadata{Name: "ws-missing-test"}},
		Manifest:       &manifest.Manifest{},
		WorkspacePaths: make(map[string]string),
		WorktreePaths:  make(map[string]*WorktreeInfo),
		ArtifactPaths:  make(map[string]string), // empty — step never ran
		Status:         &PipelineStatus{ID: "ws-missing-test"},
		Context:        NewPipelineContext("ws-missing-test", "ws-missing-test", "apply-fixes"),
	}

	_, err := executor.resolveWorkspaceStepRefs(
		"{{ steps.fetch-review.artifacts.review-findings.head_branch }}", execution)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetch-review")
}

// TestResolveWorkspaceStepRefs_NoStepsRef verifies that non-steps templates
// are passed through unchanged (they are resolved by ResolvePlaceholders later).
func TestResolveWorkspaceStepRefs_NoStepsRef(t *testing.T) {
	executor := NewDefaultPipelineExecutor(&adaptertest.MockAdapter{})

	execution := &PipelineExecution{
		Pipeline:       &Pipeline{Metadata: PipelineMetadata{Name: "ws-passthrough-test"}},
		Manifest:       &manifest.Manifest{},
		WorkspacePaths: make(map[string]string),
		WorktreePaths:  make(map[string]*WorktreeInfo),
		ArtifactPaths:  make(map[string]string),
		Status:         &PipelineStatus{ID: "ws-passthrough-test"},
		Context:        NewPipelineContext("ws-passthrough-test", "ws-passthrough-test", "step1"),
	}

	// A plain pipeline_id reference should be returned unchanged.
	result, err := executor.resolveWorkspaceStepRefs("wave/{{ pipeline_id }}/my-branch", execution)
	require.NoError(t, err)
	assert.Equal(t, "wave/{{ pipeline_id }}/my-branch", result, "non-steps templates should pass through unchanged")
}

// TestCreateStepWorkspace_DeferredBranch verifies end-to-end that a worktree
// workspace whose branch is derived from a prior step's artifact is correctly
// resolved via resolveWorkspaceStepRefs before worktree creation.
//
// The test does not actually create a real git worktree; it short-circuits via
// the WorktreePaths cache (pre-populated to simulate a worktree that was
// already created on the resolved branch).
func TestCreateStepWorkspace_DeferredBranch(t *testing.T) {
	executor := NewDefaultPipelineExecutor(&adaptertest.MockAdapter{})

	tmpDir := t.TempDir()

	// Write the artifact that encodes the dynamic branch name.
	artFile := filepath.Join(tmpDir, "pr-info.json")
	err := os.WriteFile(artFile, []byte(`{"headRefName":"feat/deferred-123"}`), 0644)
	require.NoError(t, err)

	resolvedBranch := "feat/deferred-123"
	cachedPath := "/tmp/deferred-worktree-path"
	cachedRepoRoot := "/tmp/deferred-repo-root"

	execution := &PipelineExecution{
		Pipeline: &Pipeline{Metadata: PipelineMetadata{Name: "deferred-branch-test"}},
		Manifest: &manifest.Manifest{},
		ArtifactPaths: map[string]string{
			"fetch-pr:pr-info": artFile,
		},
		WorkspacePaths: make(map[string]string),
		WorktreePaths: map[string]*WorktreeInfo{
			resolvedBranch: {AbsPath: cachedPath, RepoRoot: cachedRepoRoot},
		},
		Status:  &PipelineStatus{ID: "deferred-branch-test"},
		Context: NewPipelineContext("deferred-branch-test", "deferred-branch-test", "apply-fixes"),
	}

	step := &Step{
		ID:      "apply-fixes",
		Persona: "craftsman",
		Workspace: WorkspaceConfig{
			Type:   "worktree",
			Branch: "{{ steps.fetch-pr.artifacts.pr-info.headRefName }}",
		},
	}

	wsPath, err := executor.createStepWorkspace(execution, step)
	require.NoError(t, err)
	assert.Equal(t, cachedPath, wsPath, "workspace should be the pre-cached worktree for the resolved branch")
	assert.Equal(t, cachedRepoRoot, execution.WorkspacePaths["apply-fixes__worktree_repo_root"])
}

// artifactCapture records the args of a single RegisterArtifact call.
type artifactCapture struct {
	runID, stepID, name, path, artifactType string
	size                                    int64
	called                                  bool
}

// newCapturingArtifactStore returns a MockStateStore wired with a
// RegisterArtifact handler that records the most recent call into the given
// capture struct. Use to assert that aggregate / iterate registration fires.
func newCapturingArtifactStore(cap *artifactCapture) *testutil.MockStateStore {
	return testutil.NewMockStateStore(testutil.WithRegisterArtifact(
		func(runID, stepID, name, path, artifactType string, size int64) error {
			cap.called = true
			cap.runID = runID
			cap.stepID = stepID
			cap.name = name
			cap.path = path
			cap.artifactType = artifactType
			cap.size = size
			return nil
		},
	))
}

// TestWorkspaceRunIDFor covers the override accessor used by resume to keep
// step-workspace paths pointed at the original run's tree.
func TestWorkspaceRunIDFor(t *testing.T) {
	t.Run("falls back to pipelineID when no override", func(t *testing.T) {
		executor := NewDefaultPipelineExecutor(&adaptertest.MockAdapter{})
		assert.Equal(t, "runtime-id", executor.workspaceRunIDFor("runtime-id"))
	})
	t.Run("override wins when set", func(t *testing.T) {
		executor := NewDefaultPipelineExecutor(&adaptertest.MockAdapter{},
			WithWorkspaceRunID("original-run"),
		)
		assert.Equal(t, "original-run", executor.workspaceRunIDFor("resume-run"))
	})
	t.Run("empty override defers to pipelineID", func(t *testing.T) {
		executor := NewDefaultPipelineExecutor(&adaptertest.MockAdapter{},
			WithWorkspaceRunID(""),
		)
		assert.Equal(t, "fresh-run", executor.workspaceRunIDFor("fresh-run"))
	})
}

// TestExecuteAggregateInDAG_RegistersArtifact verifies the aggregate output is
// recorded in the artifact table — without this, downstream steps depending on
// the aggregate output cannot resume.
func TestExecuteAggregateInDAG_RegistersArtifact(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "merged-findings.json")

	cap := &artifactCapture{}
	store := newCapturingArtifactStore(cap)

	executor := NewDefaultPipelineExecutor(&adaptertest.MockAdapter{},
		WithStateStore(store),
		WithRunID("test-run-1"),
	)

	pipelineCtx := NewPipelineContext("test-run-1", "test-pipeline", "merge-findings")

	execution := &PipelineExecution{
		Pipeline:       &Pipeline{Metadata: PipelineMetadata{Name: "test-pipeline"}},
		Manifest:       &manifest.Manifest{},
		States:         make(map[string]string),
		Results:        make(map[string]map[string]interface{}),
		ArtifactPaths:  make(map[string]string),
		WorkspacePaths: make(map[string]string),
		WorktreePaths:  make(map[string]*WorktreeInfo),
		Status:         &PipelineStatus{ID: "test-run-1"},
		Context:        pipelineCtx,
	}

	step := &Step{
		ID: "merge-findings",
		Aggregate: &AggregateConfig{
			From:     `[{"id":1},{"id":2}]`,
			Into:     outputPath,
			Strategy: "concat",
		},
	}

	err := executor.executeAggregateInDAG(context.Background(), execution, step)
	require.NoError(t, err)

	require.True(t, cap.called, "store.RegisterArtifact must be called for aggregate steps")
	assert.Equal(t, "test-run-1", cap.runID)
	assert.Equal(t, "merge-findings", cap.stepID)
	assert.Equal(t, "merged-findings", cap.name, "artifact name derived from filepath.Base sans ext")
	assert.Equal(t, outputPath, cap.path)
	assert.Equal(t, "json", cap.artifactType)
	assert.Greater(t, cap.size, int64(0), "size must be populated from on-disk file")
}

// TestExecuteAggregateInDAG_NoStore preserves the test ergonomic where
// executors created without a store don't panic on aggregate steps.
func TestExecuteAggregateInDAG_NoStore(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "out.json")

	executor := NewDefaultPipelineExecutor(&adaptertest.MockAdapter{})

	execution := &PipelineExecution{
		Pipeline:       &Pipeline{Metadata: PipelineMetadata{Name: "p"}},
		Manifest:       &manifest.Manifest{},
		States:         make(map[string]string),
		Results:        make(map[string]map[string]interface{}),
		ArtifactPaths:  make(map[string]string),
		WorkspacePaths: make(map[string]string),
		WorktreePaths:  make(map[string]*WorktreeInfo),
		Status:         &PipelineStatus{ID: "p"},
	}

	step := &Step{
		ID: "agg",
		Aggregate: &AggregateConfig{
			From:     `[1,2,3]`,
			Into:     outputPath,
			Strategy: "concat",
		},
	}

	err := executor.executeAggregateInDAG(context.Background(), execution, step)
	require.NoError(t, err)
}

// TestCollectIterateOutputs_RegistersArtifact verifies that the iterate-step's
// collected output is registered as a "collected-output" artifact so resume
// preflight can locate it.
func TestCollectIterateOutputs_RegistersArtifact(t *testing.T) {
	// Run from a temp cwd: collectIterateOutputs writes the collected file to
	// .agents/output/<stepID>-collected.json relative to cwd.
	origDir, err := os.Getwd()
	require.NoError(t, err)
	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	cap := &artifactCapture{}
	store := newCapturingArtifactStore(cap)

	executor := NewDefaultPipelineExecutor(&adaptertest.MockAdapter{},
		WithStateStore(store),
		WithRunID("iterate-run"),
	)

	pipelineCtx := NewPipelineContext("iterate-run", "iterate-pipeline", "run-audits")
	// Each child sub-pipeline's output is keyed by "<childPipeline>.<artifactName>".
	pipelineCtx.SetArtifactPath("audit-alpha.scan", filepath.Join(tmpDir, "alpha.json"))
	pipelineCtx.SetArtifactPath("audit-beta.scan", filepath.Join(tmpDir, "beta.json"))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "alpha.json"), []byte(`{"id":"a"}`), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "beta.json"), []byte(`{"id":"b"}`), 0644))

	execution := &PipelineExecution{
		Pipeline:       &Pipeline{Metadata: PipelineMetadata{Name: "iterate-pipeline"}},
		Manifest:       &manifest.Manifest{},
		States:         make(map[string]string),
		Results:        make(map[string]map[string]interface{}),
		ArtifactPaths:  make(map[string]string),
		WorkspacePaths: make(map[string]string),
		WorktreePaths:  make(map[string]*WorktreeInfo),
		Status:         &PipelineStatus{ID: "iterate-run"},
		Context:        pipelineCtx,
	}

	step := &Step{ID: "run-audits"}
	err = executor.collectIterateOutputs(execution, step, []string{"audit-alpha", "audit-beta"})
	require.NoError(t, err)

	require.True(t, cap.called, "store.RegisterArtifact must be called for iterate steps")
	assert.Equal(t, "iterate-run", cap.runID)
	assert.Equal(t, "run-audits", cap.stepID)
	assert.Equal(t, "collected-output", cap.name)
	assert.Equal(t, "json", cap.artifactType)
	assert.True(t, strings.HasSuffix(cap.path, "run-audits-collected.json"), "path: %s", cap.path)
	assert.Greater(t, cap.size, int64(0))
}

// TestCreateStepWorkspace_UsesEffectiveWorkspaceRunID verifies the resume
// override threads through to step-workspace path computation. Without the
// override, resume would create an empty dir under the new run's timestamp;
// with it, the resumed step reads from the original run's tree.
func TestCreateStepWorkspace_UsesEffectiveWorkspaceRunID(t *testing.T) {
	tmpDir := t.TempDir()
	wsRoot := filepath.Join(tmpDir, "workspaces")

	executor := NewDefaultPipelineExecutor(&adaptertest.MockAdapter{},
		WithRunID("resume-run-2"),
		WithWorkspaceRunID("original-run-1"),
	)

	execution := &PipelineExecution{
		Pipeline:       &Pipeline{Metadata: PipelineMetadata{Name: "test-pipeline"}},
		Manifest:       &manifest.Manifest{Runtime: manifest.Runtime{WorkspaceRoot: wsRoot}},
		WorkspacePaths: make(map[string]string),
		WorktreePaths:  make(map[string]*WorktreeInfo),
		ArtifactPaths:  make(map[string]string),
		Status:         &PipelineStatus{ID: "resume-run-2"},
		Context:        NewPipelineContext("resume-run-2", "test-pipeline", "triage"),
	}

	step := &Step{ID: "triage"}

	wsPath, err := executor.createStepWorkspace(execution, step)
	require.NoError(t, err)

	expected := filepath.Join(wsRoot, "original-run-1", "triage")
	assert.Equal(t, expected, wsPath, "step workspace must use effective workspace run ID, not e.runID")
}
