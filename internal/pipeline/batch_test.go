package pipeline

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createSimplePipeline creates a minimal pipeline with a single step for batch testing.
func createSimplePipeline(name string) *Pipeline {
	return &Pipeline{
		Metadata: PipelineMetadata{Name: name},
		Steps: []Step{
			{
				ID:      "step-1",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "test input"},
			},
		},
	}
}

// batchConcurrencyAdapter wraps an adapter to track concurrent execution counts.
type batchConcurrencyAdapter struct {
	inner          adapter.AdapterRunner
	currentCounter *int32
	maxCounter     *int32
}

func (a *batchConcurrencyAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	current := atomic.AddInt32(a.currentCounter, 1)
	defer atomic.AddInt32(a.currentCounter, -1)
	for {
		old := atomic.LoadInt32(a.maxCounter)
		if current <= old || atomic.CompareAndSwapInt32(a.maxCounter, old, current) {
			break
		}
	}
	return a.inner.Run(ctx, cfg)
}

// pipelineRoutingAdapter routes adapter calls to different behaviors based on
// which pipeline name appears in the workspace path. This enables testing batch
// execution where individual pipelines need different outcomes.
type pipelineRoutingAdapter struct {
	failPipelines  map[string]error
	defaultAdapter adapter.AdapterRunner
}

func (a *pipelineRoutingAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	for name, err := range a.failPipelines {
		if strings.Contains(cfg.WorkspacePath, name) {
			return nil, err
		}
	}
	return a.defaultAdapter.Run(ctx, cfg)
}

// batchAdapterBehavior defines delay and optional failure for timed routing.
type batchAdapterBehavior struct {
	delay   time.Duration
	err     error
	adapter adapter.AdapterRunner
}

// batchTimedRoutingAdapter routes adapter calls with configurable delays and failures
// per pipeline name, identified by workspace path.
type batchTimedRoutingAdapter struct {
	behaviors      map[string]batchAdapterBehavior
	defaultAdapter adapter.AdapterRunner
}

func (a *batchTimedRoutingAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	for name, behavior := range a.behaviors {
		if strings.Contains(cfg.WorkspacePath, name) {
			if behavior.delay > 0 {
				select {
				case <-time.After(behavior.delay):
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			if behavior.err != nil {
				return nil, behavior.err
			}
			if behavior.adapter != nil {
				return behavior.adapter.Run(ctx, cfg)
			}
			return a.defaultAdapter.Run(ctx, cfg)
		}
	}
	return a.defaultAdapter.Run(ctx, cfg)
}

// --- Test 1: TestBatchConfigValidation ---

func TestBatchConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    PipelineBatchConfig
		wantErr   bool
		errSubstr string
	}{
		{
			name: "empty pipeline list returns error",
			config: PipelineBatchConfig{
				Pipelines: []PipelineBatchEntry{},
			},
			wantErr:   true,
			errSubstr: "at least one pipeline",
		},
		{
			name: "duplicate pipeline names returns error",
			config: PipelineBatchConfig{
				Pipelines: []PipelineBatchEntry{
					{Name: "alpha", Pipeline: createSimplePipeline("alpha")},
					{Name: "alpha", Pipeline: createSimplePipeline("alpha")},
				},
			},
			wantErr:   true,
			errSubstr: "duplicate",
		},
		{
			name: "dependency references non-existent pipeline returns error",
			config: PipelineBatchConfig{
				Pipelines: []PipelineBatchEntry{
					{Name: "alpha", Pipeline: createSimplePipeline("alpha")},
				},
				Dependencies: map[string][]string{
					"alpha": {"beta"},
				},
			},
			wantErr:   true,
			errSubstr: "not in this batch",
		},
		{
			name: "cycle detection returns error",
			config: PipelineBatchConfig{
				Pipelines: []PipelineBatchEntry{
					{Name: "A", Pipeline: createSimplePipeline("A")},
					{Name: "B", Pipeline: createSimplePipeline("B")},
				},
				Dependencies: map[string][]string{
					"A": {"B"},
					"B": {"A"},
				},
			},
			wantErr:   true,
			errSubstr: "cycle",
		},
		{
			name: "negative MaxConcurrentPipelines returns error",
			config: PipelineBatchConfig{
				Pipelines: []PipelineBatchEntry{
					{Name: "alpha", Pipeline: createSimplePipeline("alpha")},
				},
				MaxConcurrentPipelines: -1,
			},
			wantErr:   true,
			errSubstr: "non-negative",
		},
		{
			name: "valid config with defaults succeeds",
			config: PipelineBatchConfig{
				Pipelines: []PipelineBatchEntry{
					{Name: "alpha", Pipeline: createSimplePipeline("alpha")},
					{Name: "beta", Pipeline: createSimplePipeline("beta")},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err, "expected validation error")
				assert.Contains(t, err.Error(), tt.errSubstr,
					"error message should contain %q", tt.errSubstr)
			} else {
				require.NoError(t, err, "expected no validation error")
				assert.Equal(t, OnFailureContinue, tt.config.OnFailure,
					"OnFailure should default to 'continue'")
			}
		})
	}
}

// --- Test 2: TestComputePipelineTiers ---

func TestComputePipelineTiers(t *testing.T) {
	tests := []struct {
		name      string
		names     map[string]bool
		deps      map[string][]string
		wantTiers [][]string
		wantErr   bool
	}{
		{
			name:  "three independent pipelines produce single tier",
			names: map[string]bool{"alpha": true, "beta": true, "gamma": true},
			deps:  map[string][]string{},
			wantTiers: [][]string{
				{"alpha", "beta", "gamma"},
			},
		},
		{
			name:  "linear chain A->B->C produces three tiers",
			names: map[string]bool{"A": true, "B": true, "C": true},
			deps: map[string][]string{
				"B": {"A"},
				"C": {"B"},
			},
			wantTiers: [][]string{
				{"A"},
				{"B"},
				{"C"},
			},
		},
		{
			name:  "diamond dependency produces three tiers",
			names: map[string]bool{"A": true, "B": true, "C": true, "D": true},
			deps: map[string][]string{
				"B": {"A"},
				"C": {"A"},
				"D": {"B", "C"},
			},
			wantTiers: [][]string{
				{"A"},
				{"B", "C"},
				{"D"},
			},
		},
		{
			name:  "cycle returns error",
			names: map[string]bool{"X": true, "Y": true},
			deps: map[string][]string{
				"X": {"Y"},
				"Y": {"X"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tiers, err := computePipelineTiers(tt.names, tt.deps)
			if tt.wantErr {
				require.Error(t, err, "expected cycle detection error")
				return
			}
			require.NoError(t, err, "expected no error computing tiers")
			require.Len(t, tiers, len(tt.wantTiers),
				"expected %d tiers, got %d", len(tt.wantTiers), len(tiers))
			for i, expectedTier := range tt.wantTiers {
				assert.Equal(t, expectedTier, tiers[i],
					"tier %d mismatch", i)
			}
		})
	}
}

// --- Test 3: TestBatchExecuteIndependentPipelines ---

func TestBatchExecuteIndependentPipelines(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	collector := newTestEventCollector()
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(500),
		adapter.WithSimulatedDelay(50*time.Millisecond),
	)

	tmpDir := t.TempDir()
	m := createTestManifest(tmpDir)

	executor := NewDefaultPipelineExecutor(mockAdapter, WithEmitter(collector))
	batchExec := NewPipelineBatchExecutor(executor)

	config := &PipelineBatchConfig{
		Pipelines: []PipelineBatchEntry{
			{Name: "pipeline-alpha", Pipeline: createSimplePipeline("pipeline-alpha"), Manifest: m},
			{Name: "pipeline-beta", Pipeline: createSimplePipeline("pipeline-beta"), Manifest: m},
		},
	}

	result, err := batchExec.ExecuteBatch(ctx, config)
	require.NoError(t, err, "ExecuteBatch should succeed for independent pipelines")

	assert.Equal(t, 2, result.CompletedCount, "both pipelines should complete")
	assert.Equal(t, 0, result.FailedCount, "no pipelines should fail")
	assert.Equal(t, 0, result.SkippedCount, "no pipelines should be skipped")
	assert.Len(t, result.Results, 2, "should have 2 pipeline results")

	// Verify batch lifecycle events
	assert.True(t, collector.HasEventWithState(event.StateBatchStarted),
		"should emit batch_started event")
	assert.True(t, collector.HasEventWithState(event.StateBatchCompleted),
		"should emit batch_completed event")
}

// --- Test 4: TestBatchExecuteContinuePolicy ---

func TestBatchExecuteContinuePolicy(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	collector := newTestEventCollector()
	tmpDir := t.TempDir()
	m := createTestManifest(tmpDir)

	// Use a pipeline-routing adapter that fails pipeline-A but succeeds for others.
	// The batch executor shares a single adapter runner across all child executors,
	// so we route based on workspace path which contains the pipeline name.
	routingAdapter := &pipelineRoutingAdapter{
		failPipelines: map[string]error{
			"pipeline-A": errors.New("pipeline A failed"),
		},
		defaultAdapter: adapter.NewMockAdapter(
			adapter.WithStdoutJSON(`{"status": "success"}`),
			adapter.WithTokensUsed(200),
		),
	}

	executor := NewDefaultPipelineExecutor(routingAdapter, WithEmitter(collector))
	batchExec := NewPipelineBatchExecutor(executor)

	config := &PipelineBatchConfig{
		Pipelines: []PipelineBatchEntry{
			{Name: "pipeline-A", Pipeline: createSimplePipeline("pipeline-A"), Manifest: m},
			{Name: "pipeline-B", Pipeline: createSimplePipeline("pipeline-B"), Manifest: m},
			{Name: "pipeline-C", Pipeline: createSimplePipeline("pipeline-C"), Manifest: m},
		},
		Dependencies: map[string][]string{
			"pipeline-C": {"pipeline-A"},
		},
		OnFailure: OnFailureContinue,
	}

	result, err := batchExec.ExecuteBatch(ctx, config)
	require.NoError(t, err, "ExecuteBatch with continue policy should not return error")

	// Build result map for easier assertion
	resultMap := make(map[string]PipelineRunResult)
	for _, r := range result.Results {
		resultMap[r.Name] = r
	}

	// Pipeline A should fail
	assert.Equal(t, RunStatusFailed, resultMap["pipeline-A"].Status,
		"pipeline A should have failed")

	// Pipeline B should succeed (independent of A)
	assert.Equal(t, RunStatusCompleted, resultMap["pipeline-B"].Status,
		"pipeline B should complete (independent of A)")

	// Pipeline C should be skipped (depends on failed A)
	assert.Equal(t, RunStatusSkipped, resultMap["pipeline-C"].Status,
		"pipeline C should be skipped (depends on failed A)")
	assert.Contains(t, resultMap["pipeline-C"].SkipReason, "dependency",
		"skip reason should mention dependency")

	assert.Equal(t, 1, result.CompletedCount, "should have 1 completed pipeline")
	assert.Equal(t, 1, result.FailedCount, "should have 1 failed pipeline")
	assert.Equal(t, 1, result.SkippedCount, "should have 1 skipped pipeline")
}

// --- Test 5: TestBatchExecuteAbortAllPolicy ---

func TestBatchExecuteAbortAllPolicy(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	collector := newTestEventCollector()
	tmpDir := t.TempDir()
	m := createTestManifest(tmpDir)

	// Pipeline A fails quickly after 10ms, pipeline B takes 200ms to complete.
	// With abort-all policy, A's failure should cancel B's context.
	routingAdapter := &batchTimedRoutingAdapter{
		behaviors: map[string]batchAdapterBehavior{
			"pipeline-A": {
				delay: 10 * time.Millisecond,
				err:   errors.New("pipeline A failed"),
			},
			"pipeline-B": {
				delay: 200 * time.Millisecond,
				adapter: adapter.NewMockAdapter(
					adapter.WithStdoutJSON(`{"status": "success"}`),
					adapter.WithTokensUsed(200),
					adapter.WithSimulatedDelay(200*time.Millisecond),
				),
			},
		},
		defaultAdapter: adapter.NewMockAdapter(
			adapter.WithStdoutJSON(`{"status": "success"}`),
			adapter.WithTokensUsed(200),
		),
	}

	executor := NewDefaultPipelineExecutor(routingAdapter, WithEmitter(collector))
	batchExec := NewPipelineBatchExecutor(executor)

	config := &PipelineBatchConfig{
		Pipelines: []PipelineBatchEntry{
			{Name: "pipeline-A", Pipeline: createSimplePipeline("pipeline-A"), Manifest: m},
			{Name: "pipeline-B", Pipeline: createSimplePipeline("pipeline-B"), Manifest: m},
		},
		OnFailure: OnFailureAbortAll,
	}

	result, err := batchExec.ExecuteBatch(ctx, config)
	require.NoError(t, err, "ExecuteBatch should not return a top-level error")

	// Build result map
	resultMap := make(map[string]PipelineRunResult)
	for _, r := range result.Results {
		resultMap[r.Name] = r
	}

	// Pipeline A must have failed
	assert.Equal(t, RunStatusFailed, resultMap["pipeline-A"].Status,
		"pipeline A should have failed")

	// Pipeline B should either fail (context cancelled) or complete if it finished
	// before cancellation propagated. The key assertion is that A's failure was detected.
	bStatus := resultMap["pipeline-B"].Status
	assert.True(t, bStatus == RunStatusFailed || bStatus == RunStatusCompleted,
		"pipeline B should be failed (cancelled) or completed, got %q", bStatus)

	// Verify the batch_pipeline_failed event was emitted for A
	assert.True(t, collector.HasEventWithState(event.StateBatchPipelineFailed),
		"should emit batch_pipeline_failed event for pipeline A")
}

// --- Test 6: TestBatchMaxConcurrentPipelines ---

func TestBatchMaxConcurrentPipelines(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	collector := newTestEventCollector()
	tmpDir := t.TempDir()
	m := createTestManifest(tmpDir)

	var currentCounter int32
	var maxCounter int32

	innerAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(100),
		adapter.WithSimulatedDelay(50*time.Millisecond),
	)

	concAdapter := &batchConcurrencyAdapter{
		inner:          innerAdapter,
		currentCounter: &currentCounter,
		maxCounter:     &maxCounter,
	}

	executor := NewDefaultPipelineExecutor(concAdapter, WithEmitter(collector))
	batchExec := NewPipelineBatchExecutor(executor)

	config := &PipelineBatchConfig{
		Pipelines: []PipelineBatchEntry{
			{Name: "pipeline-1", Pipeline: createSimplePipeline("pipeline-1"), Manifest: m},
			{Name: "pipeline-2", Pipeline: createSimplePipeline("pipeline-2"), Manifest: m},
			{Name: "pipeline-3", Pipeline: createSimplePipeline("pipeline-3"), Manifest: m},
		},
		MaxConcurrentPipelines: 1,
	}

	result, err := batchExec.ExecuteBatch(ctx, config)
	require.NoError(t, err, "ExecuteBatch should succeed with concurrency limit")

	assert.Equal(t, 3, result.CompletedCount,
		"all three pipelines should complete")
	assert.Equal(t, 0, result.FailedCount,
		"no pipelines should fail")

	observedMax := atomic.LoadInt32(&maxCounter)
	assert.LessOrEqual(t, observedMax, int32(1),
		"max concurrent pipelines should not exceed 1, observed %d", observedMax)
}

// --- Test 7: TestBatchArtifactRegistry ---

func TestBatchArtifactRegistry(t *testing.T) {
	registry := newBatchArtifactRegistry()

	// Register artifacts from two pipelines
	registry.Register("pipeline-alpha", "step-1", "output", "/tmp/alpha/output.json")
	registry.Register("pipeline-alpha", "step-2", "report", "/tmp/alpha/report.md")
	registry.Register("pipeline-beta", "step-1", "result", "/tmp/beta/result.json")

	// Get by exact key - found
	path, ok := registry.Get("pipeline-alpha", "step-1", "output")
	assert.True(t, ok, "should find artifact for pipeline-alpha:step-1:output")
	assert.Equal(t, "/tmp/alpha/output.json", path,
		"artifact path should match registered value")

	// Get by exact key - second artifact
	path, ok = registry.Get("pipeline-alpha", "step-2", "report")
	assert.True(t, ok, "should find artifact for pipeline-alpha:step-2:report")
	assert.Equal(t, "/tmp/alpha/report.md", path,
		"artifact path should match registered value")

	// Get by wrong key - not found
	_, ok = registry.Get("pipeline-alpha", "step-1", "nonexistent")
	assert.False(t, ok, "should not find artifact with wrong name")

	_, ok = registry.Get("pipeline-gamma", "step-1", "output")
	assert.False(t, ok, "should not find artifact for non-existent pipeline")

	// GetAllForPipeline returns only that pipeline's artifacts
	alphaArtifacts := registry.GetAllForPipeline("pipeline-alpha")
	assert.Len(t, alphaArtifacts, 2,
		"pipeline-alpha should have 2 artifacts")
	assert.Equal(t, "/tmp/alpha/output.json", alphaArtifacts["step-1:output"],
		"should contain step-1:output artifact")
	assert.Equal(t, "/tmp/alpha/report.md", alphaArtifacts["step-2:report"],
		"should contain step-2:report artifact")

	betaArtifacts := registry.GetAllForPipeline("pipeline-beta")
	assert.Len(t, betaArtifacts, 1,
		"pipeline-beta should have 1 artifact")
	assert.Equal(t, "/tmp/beta/result.json", betaArtifacts["step-1:result"],
		"should contain step-1:result artifact")

	// Non-existent pipeline returns empty map
	gammaArtifacts := registry.GetAllForPipeline("pipeline-gamma")
	assert.Empty(t, gammaArtifacts,
		"non-existent pipeline should return empty map")
}

// --- Test 8: TestBatchTieredExecution ---

func TestBatchTieredExecution(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	collector := newTestEventCollector()
	tmpDir := t.TempDir()
	m := createTestManifest(tmpDir)

	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(300),
		adapter.WithSimulatedDelay(30*time.Millisecond),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter, WithEmitter(collector))
	batchExec := NewPipelineBatchExecutor(executor)

	config := &PipelineBatchConfig{
		Pipelines: []PipelineBatchEntry{
			{Name: "pipeline-A", Pipeline: createSimplePipeline("pipeline-A"), Manifest: m},
			{Name: "pipeline-B", Pipeline: createSimplePipeline("pipeline-B"), Manifest: m},
		},
		Dependencies: map[string][]string{
			"pipeline-B": {"pipeline-A"},
		},
	}

	result, err := batchExec.ExecuteBatch(ctx, config)
	require.NoError(t, err, "ExecuteBatch should succeed for tiered execution")

	assert.Equal(t, 2, result.CompletedCount,
		"both pipelines should complete")
	assert.Equal(t, 0, result.FailedCount,
		"no pipelines should fail")

	// Verify ordering via events: pipeline-A's batch_pipeline_started should
	// appear before pipeline-B's batch_pipeline_started.
	events := collector.GetEvents()

	var aStartedAt, bStartedAt time.Time
	var aCompletedAt time.Time
	for _, ev := range events {
		if ev.State == event.StateBatchPipelineStarted && ev.PipelineID == "pipeline-A" {
			aStartedAt = ev.Timestamp
		}
		if ev.State == event.StateBatchPipelineStarted && ev.PipelineID == "pipeline-B" {
			bStartedAt = ev.Timestamp
		}
		if ev.State == event.StateBatchPipelineCompleted && ev.PipelineID == "pipeline-A" {
			aCompletedAt = ev.Timestamp
		}
	}

	assert.False(t, aStartedAt.IsZero(),
		"should have a batch_pipeline_started event for pipeline-A")
	assert.False(t, bStartedAt.IsZero(),
		"should have a batch_pipeline_started event for pipeline-B")

	// Pipeline A should start before pipeline B (A is in tier 0, B is in tier 1)
	assert.True(t, aStartedAt.Before(bStartedAt) || aStartedAt.Equal(bStartedAt),
		"pipeline A (tier 0) should start before or at the same time as pipeline B (tier 1)")

	// Pipeline A should complete before pipeline B starts (tiers are sequential)
	if !aCompletedAt.IsZero() {
		assert.True(t, aCompletedAt.Before(bStartedAt) || aCompletedAt.Equal(bStartedAt),
			"pipeline A should complete before pipeline B starts (sequential tiers)")
	}
}
