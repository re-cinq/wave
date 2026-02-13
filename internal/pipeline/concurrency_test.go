package pipeline

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
)

// concurrencyTestEventCollector collects events for concurrency tests
type concurrencyTestEventCollector struct {
	mu     sync.Mutex
	events []event.Event
}

func newConcurrencyTestEventCollector() *concurrencyTestEventCollector {
	return &concurrencyTestEventCollector{
		events: make([]event.Event, 0),
	}
}

func (c *concurrencyTestEventCollector) Emit(e event.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, e)
}

func (c *concurrencyTestEventCollector) GetEvents() []event.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]event.Event, len(c.events))
	copy(result, c.events)
	return result
}

func TestConcurrencyExecutor_ConcurrencyOneIsNonConcurrent(t *testing.T) {
	executor := &DefaultPipelineExecutor{}
	concExecutor := NewConcurrencyExecutor(executor)

	execution := &PipelineExecution{
		Pipeline: &Pipeline{Metadata: PipelineMetadata{Name: "test"}},
		Status:   &PipelineStatus{ID: "test", PipelineName: "test"},
	}

	step := &Step{ID: "test_step", Concurrency: 1}

	err := concExecutor.Execute(context.Background(), execution, step)
	if err == nil {
		t.Error("Expected error for concurrency=1, which should not use ConcurrencyExecutor")
	}
}

func TestConcurrencyExecutor_ConcurrencyZeroIsNonConcurrent(t *testing.T) {
	executor := &DefaultPipelineExecutor{}
	concExecutor := NewConcurrencyExecutor(executor)

	execution := &PipelineExecution{
		Pipeline: &Pipeline{Metadata: PipelineMetadata{Name: "test"}},
		Status:   &PipelineStatus{ID: "test", PipelineName: "test"},
	}

	step := &Step{ID: "test_step", Concurrency: 0}

	err := concExecutor.Execute(context.Background(), execution, step)
	if err == nil {
		t.Error("Expected error for concurrency=0, which should not use ConcurrencyExecutor")
	}
}

func TestConcurrencyExecutor_SpawnsCorrectWorkerCount(t *testing.T) {
	tests := []struct {
		name              string
		concurrency       int
		maxWorkers        int
		expectedCount     int
	}{
		{
			name:          "3 workers spawned",
			concurrency:   3,
			maxWorkers:    10,
			expectedCount: 3,
		},
		{
			name:          "5 workers spawned",
			concurrency:   5,
			maxWorkers:    10,
			expectedCount: 5,
		},
		{
			name:          "capped by max_concurrent_workers",
			concurrency:   20,
			maxWorkers:    5,
			expectedCount: 5,
		},
		{
			name:          "default cap of 10",
			concurrency:   15,
			maxWorkers:    0, // 0 means use default cap of 10
			expectedCount: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			workerTracker := &workerSpawnTracker{
				spawned: make(map[int]bool),
			}

			trackingAdapter := &workerTrackingAdapter{
				tracker: workerTracker,
				baseAdapter: adapter.NewMockAdapter(
					adapter.WithStdoutJSON(`{"status": "success"}`),
					adapter.WithTokensUsed(100),
				),
			}

			eventCollector := newConcurrencyTestEventCollector()
			executor := NewDefaultPipelineExecutor(trackingAdapter, WithEmitter(eventCollector))
			concExecutor := NewConcurrencyExecutor(executor)

			m := &manifest.Manifest{
				Personas: map[string]manifest.Persona{
					"worker": {Adapter: "claude"},
				},
				Adapters: map[string]manifest.Adapter{
					"claude": {Binary: "claude"},
				},
				Runtime: manifest.Runtime{
					WorkspaceRoot:        tmpDir,
					MaxConcurrentWorkers: tt.maxWorkers,
				},
			}

			execution := &PipelineExecution{
				Pipeline:       &Pipeline{Metadata: PipelineMetadata{Name: "conc-test"}},
				Manifest:       m,
				States:         make(map[string]string),
				Results:        make(map[string]map[string]interface{}),
				ArtifactPaths:  make(map[string]string),
				WorkspacePaths: make(map[string]string),
				Context:        NewPipelineContext("conc-test", "conc-test", "conc_step"),
				Status:         &PipelineStatus{ID: "conc-test", PipelineName: "conc-test"},
			}

			step := &Step{
				ID:          "conc_step",
				Persona:     "worker",
				Concurrency: tt.concurrency,
				Exec: ExecConfig{
					Type:   "prompt",
					Source: "Process work",
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			err := concExecutor.Execute(ctx, execution, step)
			if err != nil {
				t.Fatalf("Concurrent execution failed: %v", err)
			}

			// Verify correct number of workers spawned
			workerTracker.mu.Lock()
			actualCount := len(workerTracker.spawned)
			workerTracker.mu.Unlock()

			if actualCount != tt.expectedCount {
				t.Errorf("Expected %d workers spawned, got %d", tt.expectedCount, actualCount)
			}

			// Verify results aggregation
			results := execution.Results[step.ID]
			if results == nil {
				t.Fatal("Expected aggregated results")
			}

			if totalWorkers, ok := results["total_workers"].(int); ok {
				if totalWorkers != tt.expectedCount {
					t.Errorf("Expected total_workers=%d, got %d", tt.expectedCount, totalWorkers)
				}
			} else {
				t.Error("Missing total_workers in results")
			}

			if successCount, ok := results["success_count"].(int); ok {
				if successCount != tt.expectedCount {
					t.Errorf("Expected success_count=%d, got %d", tt.expectedCount, successCount)
				}
			}
		})
	}
}

func TestConcurrencyExecutor_ResultsAggregated(t *testing.T) {
	tmpDir := t.TempDir()

	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(100),
	)

	executor := NewDefaultPipelineExecutor(mockAdapter)
	concExecutor := NewConcurrencyExecutor(executor)

	m := &manifest.Manifest{
		Personas: map[string]manifest.Persona{
			"worker": {Adapter: "claude"},
		},
		Adapters: map[string]manifest.Adapter{
			"claude": {Binary: "claude"},
		},
		Runtime: manifest.Runtime{
			WorkspaceRoot:        tmpDir,
			MaxConcurrentWorkers: 10,
		},
	}

	execution := &PipelineExecution{
		Pipeline:       &Pipeline{Metadata: PipelineMetadata{Name: "agg-test"}},
		Manifest:       m,
		States:         make(map[string]string),
		Results:        make(map[string]map[string]interface{}),
		ArtifactPaths:  make(map[string]string),
		WorkspacePaths: make(map[string]string),
		Context:        NewPipelineContext("agg-test", "agg-test", "agg_step"),
		Status:         &PipelineStatus{ID: "agg-test", PipelineName: "agg-test"},
	}

	step := &Step{
		ID:          "agg_step",
		Persona:     "worker",
		Concurrency: 3,
		Exec: ExecConfig{
			Type:   "prompt",
			Source: "Process work",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := concExecutor.Execute(ctx, execution, step)
	if err != nil {
		t.Fatalf("Concurrent execution failed: %v", err)
	}

	results := execution.Results[step.ID]
	if results == nil {
		t.Fatal("Expected aggregated results")
	}

	// Verify all expected keys exist
	if _, ok := results["worker_results"]; !ok {
		t.Error("Missing worker_results key")
	}
	if _, ok := results["worker_workspaces"]; !ok {
		t.Error("Missing worker_workspaces key")
	}
	if results["total_workers"] != 3 {
		t.Errorf("Expected total_workers=3, got %v", results["total_workers"])
	}
	if results["success_count"] != 3 {
		t.Errorf("Expected success_count=3, got %v", results["success_count"])
	}
	if results["fail_count"] != 0 {
		t.Errorf("Expected fail_count=0, got %v", results["fail_count"])
	}

	// Verify worker workspaces were created
	workspaces, ok := results["worker_workspaces"].([]string)
	if !ok {
		t.Fatal("worker_workspaces should be a []string")
	}
	if len(workspaces) != 3 {
		t.Errorf("Expected 3 workspace paths, got %d", len(workspaces))
	}
}

func TestConcurrencyExecutor_PartialFailureCancelsRemaining(t *testing.T) {
	tmpDir := t.TempDir()

	failingSet := map[int]bool{1: true} // Second worker fails

	failAdapter := &partialFailureAdapter{
		failingIndices: failingSet,
		callCount:      0,
		baseAdapter: adapter.NewMockAdapter(
			adapter.WithStdoutJSON(`{"status": "success"}`),
			adapter.WithTokensUsed(100),
		),
	}

	eventCollector := newConcurrencyTestEventCollector()
	executor := NewDefaultPipelineExecutor(failAdapter, WithEmitter(eventCollector))
	concExecutor := NewConcurrencyExecutor(executor)

	m := &manifest.Manifest{
		Personas: map[string]manifest.Persona{
			"worker": {Adapter: "claude"},
		},
		Adapters: map[string]manifest.Adapter{
			"claude": {Binary: "claude"},
		},
		Runtime: manifest.Runtime{
			WorkspaceRoot:        tmpDir,
			MaxConcurrentWorkers: 10,
		},
	}

	execution := &PipelineExecution{
		Pipeline:       &Pipeline{Metadata: PipelineMetadata{Name: "fail-test"}},
		Manifest:       m,
		States:         make(map[string]string),
		Results:        make(map[string]map[string]interface{}),
		ArtifactPaths:  make(map[string]string),
		WorkspacePaths: make(map[string]string),
		Context:        NewPipelineContext("fail-test", "fail-test", "fail_step"),
		Status:         &PipelineStatus{ID: "fail-test", PipelineName: "fail-test"},
	}

	step := &Step{
		ID:          "fail_step",
		Persona:     "worker",
		Concurrency: 3,
		Exec: ExecConfig{
			Type:   "prompt",
			Source: "Process work",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := concExecutor.Execute(ctx, execution, step)
	if err == nil {
		t.Error("Expected error due to worker failure")
	}

	// Verify results still aggregated despite failure
	results := execution.Results[step.ID]
	if results == nil {
		t.Fatal("Expected aggregated results even on partial failure")
	}

	failCount, _ := results["fail_count"].(int)
	if failCount == 0 {
		t.Error("Expected at least one failed worker")
	}

	// Verify failure events were emitted
	events := eventCollector.GetEvents()
	hasFailedEvent := false
	for _, e := range events {
		if e.State == "concurrent_worker_failed" {
			hasFailedEvent = true
		}
	}
	if !hasFailedEvent {
		t.Error("Expected concurrent_worker_failed event")
	}
}

func TestConcurrencyExecutor_EventLifecycle(t *testing.T) {
	tmpDir := t.TempDir()

	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(100),
	)

	eventCollector := newConcurrencyTestEventCollector()
	executor := NewDefaultPipelineExecutor(mockAdapter, WithEmitter(eventCollector))
	concExecutor := NewConcurrencyExecutor(executor)

	m := &manifest.Manifest{
		Personas: map[string]manifest.Persona{
			"worker": {Adapter: "claude"},
		},
		Adapters: map[string]manifest.Adapter{
			"claude": {Binary: "claude"},
		},
		Runtime: manifest.Runtime{
			WorkspaceRoot:        tmpDir,
			MaxConcurrentWorkers: 10,
		},
	}

	execution := &PipelineExecution{
		Pipeline:       &Pipeline{Metadata: PipelineMetadata{Name: "event-test"}},
		Manifest:       m,
		States:         make(map[string]string),
		Results:        make(map[string]map[string]interface{}),
		ArtifactPaths:  make(map[string]string),
		WorkspacePaths: make(map[string]string),
		Context:        NewPipelineContext("event-test", "event-test", "event_step"),
		Status:         &PipelineStatus{ID: "event-test", PipelineName: "event-test"},
	}

	step := &Step{
		ID:          "event_step",
		Persona:     "worker",
		Concurrency: 2,
		Exec: ExecConfig{
			Type:   "prompt",
			Source: "Process work",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := concExecutor.Execute(ctx, execution, step)
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	events := eventCollector.GetEvents()

	// Verify lifecycle events
	stateMap := make(map[string]int)
	for _, e := range events {
		stateMap[e.State]++
	}

	if stateMap["concurrent_start"] != 1 {
		t.Errorf("Expected 1 concurrent_start event, got %d", stateMap["concurrent_start"])
	}
	if stateMap["concurrent_worker_start"] != 2 {
		t.Errorf("Expected 2 concurrent_worker_start events, got %d", stateMap["concurrent_worker_start"])
	}
	if stateMap["concurrent_worker_complete"] != 2 {
		t.Errorf("Expected 2 concurrent_worker_complete events, got %d", stateMap["concurrent_worker_complete"])
	}
	if stateMap["concurrent_complete"] != 1 {
		t.Errorf("Expected 1 concurrent_complete event, got %d", stateMap["concurrent_complete"])
	}
}

func TestConcurrencyExecutor_CreateWorkerWorkspace(t *testing.T) {
	tmpDir := t.TempDir()

	executor := &DefaultPipelineExecutor{}
	concExecutor := NewConcurrencyExecutor(executor)

	execution := &PipelineExecution{
		Pipeline: &Pipeline{Metadata: PipelineMetadata{Name: "ws-test"}},
		Manifest: &manifest.Manifest{
			Runtime: manifest.Runtime{
				WorkspaceRoot: tmpDir,
			},
		},
		Status: &PipelineStatus{ID: "ws-test", PipelineName: "ws-test"},
	}

	step := &Step{ID: "ws_step"}

	for _, idx := range []int{0, 1, 2} {
		wsPath, err := concExecutor.createWorkerWorkspace(execution, step, idx)
		if err != nil {
			t.Fatalf("Failed to create workspace for worker %d: %v", idx, err)
		}

		expectedSuffix := fmt.Sprintf("ws-test/ws_step/worker_%d", idx)
		if !contains(wsPath, expectedSuffix) {
			t.Errorf("Workspace path %q does not contain expected suffix %q", wsPath, expectedSuffix)
		}

		// Verify directory was created
		if _, err := os.Stat(wsPath); os.IsNotExist(err) {
			t.Errorf("Workspace directory for worker %d was not created", idx)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestConcurrencyExecutor_ConcurrencyLimitRespected verifies that concurrent worker
// count does not exceed the configured limit.
func TestConcurrencyExecutor_ConcurrencyLimitRespected(t *testing.T) {
	tmpDir := t.TempDir()

	concurrencyTracker := &concurrencyLimitTracker{
		maxObserved: 0,
		current:     0,
	}

	concurrencyAdapter := &concurrencyTrackingMatrixAdapter{
		tracker: concurrencyTracker,
		baseAdapter: adapter.NewMockAdapter(
			adapter.WithStdoutJSON(`{"status": "success"}`),
			adapter.WithTokensUsed(100),
			adapter.WithSimulatedDelay(50*time.Millisecond),
		),
	}

	executor := NewDefaultPipelineExecutor(concurrencyAdapter)
	concExecutor := NewConcurrencyExecutor(executor)

	m := &manifest.Manifest{
		Personas: map[string]manifest.Persona{
			"worker": {Adapter: "claude"},
		},
		Adapters: map[string]manifest.Adapter{
			"claude": {Binary: "claude"},
		},
		Runtime: manifest.Runtime{
			WorkspaceRoot:        tmpDir,
			MaxConcurrentWorkers: 2,
		},
	}

	execution := &PipelineExecution{
		Pipeline:       &Pipeline{Metadata: PipelineMetadata{Name: "limit-test"}},
		Manifest:       m,
		States:         make(map[string]string),
		Results:        make(map[string]map[string]interface{}),
		ArtifactPaths:  make(map[string]string),
		WorkspacePaths: make(map[string]string),
		Context:        NewPipelineContext("limit-test", "limit-test", "limit_step"),
		Status:         &PipelineStatus{ID: "limit-test", PipelineName: "limit-test"},
	}

	step := &Step{
		ID:          "limit_step",
		Persona:     "worker",
		Concurrency: 5, // Request 5 but max is 2
		Exec: ExecConfig{
			Type:   "prompt",
			Source: "Process work",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err := concExecutor.Execute(ctx, execution, step)
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	concurrencyTracker.mu.Lock()
	maxObserved := concurrencyTracker.maxObserved
	concurrencyTracker.mu.Unlock()

	// Should not exceed the cap of 2
	if maxObserved > 2 {
		t.Errorf("Concurrency exceeded limit: observed %d, expected max 2", maxObserved)
	}

	// Verify only 2 workers were spawned (capped)
	results := execution.Results[step.ID]
	if totalWorkers, ok := results["total_workers"].(int); ok {
		if totalWorkers != 2 {
			t.Errorf("Expected 2 workers (capped), got %d", totalWorkers)
		}
	}
}
