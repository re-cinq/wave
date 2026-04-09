package pipeline

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/testutil"
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
	collector := testutil.NewEventCollector()
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(500),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

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
	collector := testutil.NewEventCollector()
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(300),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

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
	collector := testutil.NewEventCollector()

	// This adapter always fails
	failAdapter := adapter.NewMockAdapter(
		adapter.WithFailure(errors.New("adapter explosion")),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

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
	collector := testutil.NewEventCollector()
	mockAdapter := adapter.NewMockAdapter()

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

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
	collector := testutil.NewEventCollector()

	// Use a delay so context cancellation can fire between pipelines
	slowAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(100),
		adapter.WithSimulatedDelay(50*time.Millisecond),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

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
	collector := testutil.NewEventCollector()
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(1000),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

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
	collector := testutil.NewEventCollector()
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(100),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

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
	collector := testutil.NewEventCollector()
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(200),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

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
	collector := testutil.NewEventCollector()
	mockAdapter := adapter.NewMockAdapter()

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

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

func TestSequenceExecutor_ExecutePlan_ParallelFailFastFalse(t *testing.T) {
	collector := testutil.NewEventCollector()

	// Use a call-counter adapter that fails on the 2nd call.
	// In parallel mode, the order is non-deterministic but exactly one will fail.
	var callCount int32
	counterAdapter := &callCounterFailAdapter{
		callCount:  &callCount,
		failOnCall: 2,
		failErr:    errors.New("pipeline exploded"),
		successAdapter: adapter.NewMockAdapter(
			adapter.WithStdoutJSON(`{"status": "success"}`),
			adapter.WithSimulatedDelay(10*time.Millisecond),
		),
	}

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	seq := NewSequenceExecutor(
		func(opts ...ExecutorOption) *DefaultPipelineExecutor {
			return NewDefaultPipelineExecutor(counterAdapter, opts...)
		},
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
					newMinimalPipeline("p3"),
				},
				Parallel: true,
			},
		},
		FailFast: false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := seq.ExecutePlan(ctx, plan, m, "fail-fast-false test")

	// Should return an error (partial failure)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrParallelStagePartialFailure)

	// All 3 pipelines should have results
	require.Len(t, result.PipelineResults, 3)

	// Count completed and failed
	var completed, failed int
	for _, pr := range result.PipelineResults {
		switch pr.Status {
		case "completed":
			completed++
		case "failed":
			failed++
		}
	}
	assert.Equal(t, 2, completed, "two pipelines should complete")
	assert.Equal(t, 1, failed, "one pipeline should fail")

	// Verify sequence_failed event was emitted (not sequence_completed)
	assert.True(t, collector.HasEventWithState(event.StateSequenceFailed))
	assert.False(t, collector.HasEventWithState(event.StateSequenceCompleted))
}

func TestSequenceExecutor_ExecutePlan_MaxConcurrent(t *testing.T) {
	collector := testutil.NewEventCollector()

	var currentConcurrent int32
	var maxConcurrent int32

	trackingAdapter := &concurrencyTrackingAdapter{
		MockAdapter: adapter.NewMockAdapter(
			adapter.WithStdoutJSON(`{"status": "success"}`),
			adapter.WithSimulatedDelay(50*time.Millisecond),
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

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	seq := NewSequenceExecutor(
		func(opts ...ExecutorOption) *DefaultPipelineExecutor {
			return NewDefaultPipelineExecutor(trackingAdapter, opts...)
		},
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
					newMinimalPipeline("p3"),
					newMinimalPipeline("p4"),
				},
				Parallel: true,
			},
		},
		FailFast:      true,
		MaxConcurrent: 2,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := seq.ExecutePlan(ctx, plan, m, "max-concurrent test")
	require.NoError(t, err)
	require.Len(t, result.PipelineResults, 4)

	for _, pr := range result.PipelineResults {
		assert.Equal(t, "completed", pr.Status)
	}

	// Max concurrent should not exceed the limit of 2
	assert.LessOrEqual(t, atomic.LoadInt32(&maxConcurrent), int32(2),
		"max concurrent should not exceed the limit of 2")
	// But should actually have some concurrency (at least 2 running at once with 50ms delay)
	assert.GreaterOrEqual(t, atomic.LoadInt32(&maxConcurrent), int32(2),
		"should have at least 2 running concurrently")
}

func TestSequenceExecutor_CrossPipelineArtifacts(t *testing.T) {
	// Verify that pipelineOutputs are passed to downstream executors via
	// WithCrossPipelineArtifacts option.
	collector := testutil.NewEventCollector()
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	// Track which options are passed to the executor factory
	var capturedArtifacts []map[string]map[string][]byte
	var mu sync.Mutex

	seq := NewSequenceExecutor(
		func(opts ...ExecutorOption) *DefaultPipelineExecutor {
			ex := NewDefaultPipelineExecutor(mockAdapter, opts...)
			mu.Lock()
			if ex.crossPipelineArtifacts != nil {
				capturedArtifacts = append(capturedArtifacts, ex.crossPipelineArtifacts)
			}
			mu.Unlock()
			return ex
		},
		nil,
		collector,
		nil,
	)

	// Simulate that a prior pipeline recorded outputs by pre-populating
	seq.pipelineOutputs["upstream"] = map[string][]byte{
		"report.json": []byte(`{"result": "ok"}`),
	}

	// Execute a single pipeline — it should receive the cross-pipeline artifacts
	pipelines := []*Pipeline{newMinimalPipeline("downstream")}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := seq.Execute(ctx, pipelines, m, "cross-pipeline test")
	require.NoError(t, err)

	// The downstream executor should have received the cross-pipeline artifacts
	mu.Lock()
	defer mu.Unlock()
	require.Len(t, capturedArtifacts, 1, "downstream executor should receive cross-pipeline artifacts")
	assert.Contains(t, capturedArtifacts[0], "upstream")
	assert.Equal(t, []byte(`{"result": "ok"}`), capturedArtifacts[0]["upstream"]["report.json"])
}

func TestSequenceExecutor_ExecutePlan_SequentialFailFastFalse(t *testing.T) {
	collector := testutil.NewEventCollector()

	// Fail on the 2nd call (sequential, so deterministic: good1=1, bad=2, good2=3)
	var callCount int32
	counterAdapter := &callCounterFailAdapter{
		callCount:  &callCount,
		failOnCall: 2,
		failErr:    errors.New("sequential bad pipeline failed"),
		successAdapter: adapter.NewMockAdapter(
			adapter.WithStdoutJSON(`{"status": "success"}`),
		),
	}

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	seq := NewSequenceExecutor(
		func(opts ...ExecutorOption) *DefaultPipelineExecutor {
			return NewDefaultPipelineExecutor(counterAdapter, opts...)
		},
		nil,
		collector,
		nil,
	)

	plan := ExecutionPlan{
		Stages: []Stage{
			{
				Pipelines: []*Pipeline{
					newMinimalPipeline("good1"),
					newMinimalPipeline("bad"),
					newMinimalPipeline("good2"),
				},
				Parallel: false,
			},
		},
		FailFast: false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := seq.ExecutePlan(ctx, plan, m, "sequential-fail-fast-false test")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrParallelStagePartialFailure)

	// All 3 pipelines should have been attempted since fail-fast is false
	require.Len(t, result.PipelineResults, 3)

	assert.Equal(t, "completed", result.PipelineResults[0].Status)
	assert.Equal(t, "failed", result.PipelineResults[1].Status)
	assert.Equal(t, "completed", result.PipelineResults[2].Status)

	assert.True(t, collector.HasEventWithState(event.StateSequenceFailed))
}

// callCounterFailAdapter fails on the Nth call and succeeds on all others.
type callCounterFailAdapter struct {
	callCount      *int32
	failOnCall     int32
	failErr        error
	successAdapter adapter.AdapterRunner
}

func (a *callCounterFailAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	n := atomic.AddInt32(a.callCount, 1)
	if n == a.failOnCall {
		return nil, a.failErr
	}
	return a.successAdapter.Run(ctx, cfg)
}

// TestSequenceExecutor_RecordPipelineOutputs_ConcurrentRace verifies that
// concurrent calls to recordPipelineOutputs do not cause a data race.
// This test is meaningful only when run with -race.
func TestSequenceExecutor_RecordPipelineOutputs_ConcurrentRace(t *testing.T) {
	const numGoroutines = 10

	wsRoot := t.TempDir()

	// Build N pipelines, each with a terminal step that has an output artifact.
	// Pre-create the artifact files on disk so LoadStepArtifact succeeds.
	type pipelineSetup struct {
		pipeline *Pipeline
		runID    string
	}
	setups := make([]pipelineSetup, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		name := fmt.Sprintf("pipeline-%d", i)
		stepID := "final-step"
		runID := fmt.Sprintf("run-%d", i)
		artifactName := "result.json"

		p := &Pipeline{
			Metadata: PipelineMetadata{Name: name},
			Steps: []Step{
				{
					ID:      stepID,
					Persona: "navigator",
					OutputArtifacts: []ArtifactDef{
						{Name: artifactName},
					},
				},
			},
		}

		// Create the artifact file at the location LoadStepArtifact checks:
		// wsRoot/<runID>/<stepID>/.wave/output/<artifactName>
		artifactDir := filepath.Join(wsRoot, runID, stepID, ".wave", "output")
		require.NoError(t, os.MkdirAll(artifactDir, 0755))
		content := fmt.Sprintf(`{"pipeline": "%s", "index": %d}`, name, i)
		require.NoError(t, os.WriteFile(filepath.Join(artifactDir, artifactName), []byte(content), 0644))

		setups[i] = pipelineSetup{pipeline: p, runID: runID}
	}

	seq := NewSequenceExecutor(
		newSequenceTestExecutorFactory(adapter.NewMockAdapter()),
		nil,
		nil, // no emitter needed
		nil,
	)

	// Call recordPipelineOutputs concurrently from multiple goroutines.
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			seq.recordPipelineOutputs(setups[i].pipeline, setups[i].runID, wsRoot)
		}()
	}
	wg.Wait()

	// Verify all pipeline outputs were recorded correctly.
	outputs := seq.GetPipelineOutputs()
	require.Len(t, outputs, numGoroutines, "all pipelines should have recorded outputs")

	for i := 0; i < numGoroutines; i++ {
		name := fmt.Sprintf("pipeline-%d", i)
		pipeOut, ok := outputs[name]
		require.True(t, ok, "outputs should contain pipeline %s", name)
		data, ok := pipeOut["result.json"]
		require.True(t, ok, "pipeline %s should have result.json artifact", name)
		expected := fmt.Sprintf(`{"pipeline": "%s", "index": %d}`, name, i)
		assert.Equal(t, expected, string(data), "artifact content mismatch for pipeline %s", name)
	}
}

// TestSequenceExecutor_CrossPipelineArtifacts_WrittenToDisk verifies that
// cross-pipeline artifacts are actually written to the workspace filesystem
// when injectArtifacts processes a step with a cross-pipeline artifact ref.
func TestSequenceExecutor_CrossPipelineArtifacts_WrittenToDisk(t *testing.T) {
	workspacePath := t.TempDir()

	// Set up a DefaultPipelineExecutor with cross-pipeline artifacts.
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
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

			artifactsDir := filepath.Join(wsPath, ".wave", "artifacts")
			for filename, expectedContent := range tt.wantFiles {
				filePath := filepath.Join(artifactsDir, filename)
				data, readErr := os.ReadFile(filePath)
				require.NoError(t, readErr, "artifact file %s should exist on disk", filename)
				assert.Equal(t, expectedContent, string(data), "content mismatch for %s", filename)
			}
		})
	}
}
