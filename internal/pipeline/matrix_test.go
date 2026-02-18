package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
)

// matrixTestEventCollector for matrix tests
type matrixTestEventCollector struct {
	mu     sync.Mutex
	events []event.Event
}

func newMatrixTestEventCollector() *matrixTestEventCollector {
	return &matrixTestEventCollector{
		events: make([]event.Event, 0),
	}
}

func (c *matrixTestEventCollector) Emit(e event.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, e)
}

func (c *matrixTestEventCollector) GetEvents() []event.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]event.Event, len(c.events))
	copy(result, c.events)
	return result
}

func TestMatrixExecutor_ReadItemsSource(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "matrix_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test JSON file with items
	items := []map[string]interface{}{
		{"name": "task1", "priority": 1},
		{"name": "task2", "priority": 2},
		{"name": "task3", "priority": 3},
	}
	itemsJSON, _ := json.Marshal(items)
	itemsFile := filepath.Join(tmpDir, "items.json")
	os.WriteFile(itemsFile, itemsJSON, 0644)

	executor := &DefaultPipelineExecutor{}
	matrixExecutor := NewMatrixExecutor(executor)

	execution := &PipelineExecution{
		WorkspacePaths: map[string]string{},
		WorktreePaths:  make(map[string]*WorktreeInfo),
		ArtifactPaths:  map[string]string{},
	}

	strategy := &MatrixStrategy{
		Type:        "matrix",
		ItemsSource: itemsFile,
	}

	result, err := matrixExecutor.readItemsSource(execution, strategy)
	if err != nil {
		t.Fatalf("Failed to read items source: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result))
	}
}

func TestMatrixExecutor_ReadItemsSource_WithItemKey(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "matrix_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a JSON file with nested structure
	data := map[string]interface{}{
		"metadata": map[string]interface{}{
			"version": "1.0",
		},
		"tasks": []map[string]interface{}{
			{"id": "t1"},
			{"id": "t2"},
		},
	}
	dataJSON, _ := json.Marshal(data)
	dataFile := filepath.Join(tmpDir, "data.json")
	os.WriteFile(dataFile, dataJSON, 0644)

	executor := &DefaultPipelineExecutor{}
	matrixExecutor := NewMatrixExecutor(executor)

	execution := &PipelineExecution{
		WorkspacePaths: map[string]string{},
		WorktreePaths:  make(map[string]*WorktreeInfo),
		ArtifactPaths:  map[string]string{},
	}

	strategy := &MatrixStrategy{
		Type:        "matrix",
		ItemsSource: dataFile,
		ItemKey:     "tasks",
	}

	result, err := matrixExecutor.readItemsSource(execution, strategy)
	if err != nil {
		t.Fatalf("Failed to read items source: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result))
	}
}

func TestMatrixExecutor_ReadItemsSource_FromPreviousStep(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "matrix_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a workspace structure mimicking a previous step's output
	prevStepWS := filepath.Join(tmpDir, "prev_step")
	os.MkdirAll(prevStepWS, 0755)

	items := []string{"item1", "item2", "item3"}
	itemsJSON, _ := json.Marshal(items)
	os.WriteFile(filepath.Join(prevStepWS, "output.json"), itemsJSON, 0644)

	executor := &DefaultPipelineExecutor{}
	matrixExecutor := NewMatrixExecutor(executor)

	execution := &PipelineExecution{
		WorkspacePaths: map[string]string{
			"analyze": prevStepWS,
		},
		WorktreePaths: make(map[string]*WorktreeInfo),
		ArtifactPaths: map[string]string{},
	}

	strategy := &MatrixStrategy{
		Type:        "matrix",
		ItemsSource: "analyze/output.json",
	}

	result, err := matrixExecutor.readItemsSource(execution, strategy)
	if err != nil {
		t.Fatalf("Failed to read items source: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result))
	}
}

func TestMatrixExecutor_ExtractByKey(t *testing.T) {
	tests := []struct {
		name        string
		data        interface{}
		key         string
		expectLen   int
		expectError bool
	}{
		{
			name: "simple key",
			data: map[string]interface{}{
				"items": []interface{}{"a", "b", "c"},
			},
			key:       "items",
			expectLen: 3,
		},
		{
			name: "nested key",
			data: map[string]interface{}{
				"result": map[string]interface{}{
					"data": []interface{}{1, 2},
				},
			},
			key:       "result.data",
			expectLen: 2,
		},
		{
			name: "missing key",
			data: map[string]interface{}{
				"other": []interface{}{},
			},
			key:         "items",
			expectError: true,
		},
		{
			name:      "empty key returns original",
			data:      []interface{}{"x", "y"},
			key:       "",
			expectLen: 2,
		},
	}

	executor := &DefaultPipelineExecutor{}
	matrixExecutor := NewMatrixExecutor(executor)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := matrixExecutor.extractByKey(tt.data, tt.key)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			arr, ok := result.([]interface{})
			if !ok {
				t.Fatal("Expected array result")
			}

			if len(arr) != tt.expectLen {
				t.Errorf("Expected length %d, got %d", tt.expectLen, len(arr))
			}
		})
	}
}

func TestMatrixExecutor_CreateWorkerStep(t *testing.T) {
	executor := &DefaultPipelineExecutor{}
	matrixExecutor := NewMatrixExecutor(executor)

	step := &Step{
		ID:      "test_step",
		Persona: "worker",
		Exec: ExecConfig{
			Type:   "prompt",
			Source: "Process this task: {{ task }}",
		},
	}

	item := map[string]interface{}{
		"name":   "test_task",
		"params": []string{"a", "b"},
	}

	workerStep := matrixExecutor.createWorkerStep(step, item)

	// Verify the step is a copy
	if workerStep == step {
		t.Error("Worker step should be a copy, not the same pointer")
	}

	// Verify the template was replaced
	expectedJSON, _ := json.Marshal(item)
	expectedPrompt := "Process this task: " + string(expectedJSON)

	if workerStep.Exec.Source != expectedPrompt {
		t.Errorf("Expected prompt %q, got %q", expectedPrompt, workerStep.Exec.Source)
	}
}

func TestMatrixExecutor_DetectFileConflicts(t *testing.T) {
	executor := &DefaultPipelineExecutor{}
	matrixExecutor := NewMatrixExecutor(executor)

	tests := []struct {
		name        string
		results     []MatrixResult
		expectError bool
	}{
		{
			name: "no conflicts",
			results: []MatrixResult{
				{ItemIndex: 0, ModifiedFiles: []string{"file1.txt", "file2.txt"}},
				{ItemIndex: 1, ModifiedFiles: []string{"file3.txt", "file4.txt"}},
			},
			expectError: false,
		},
		{
			name: "conflict detected",
			results: []MatrixResult{
				{ItemIndex: 0, ModifiedFiles: []string{"shared.txt", "file1.txt"}},
				{ItemIndex: 1, ModifiedFiles: []string{"shared.txt", "file2.txt"}},
			},
			expectError: true,
		},
		{
			name: "skips failed workers",
			results: []MatrixResult{
				{ItemIndex: 0, ModifiedFiles: []string{"shared.txt"}, Error: nil},
				{ItemIndex: 1, ModifiedFiles: []string{"shared.txt"}, Error: context.Canceled},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := matrixExecutor.detectFileConflicts(tt.results)
			if tt.expectError && err == nil {
				t.Error("Expected conflict error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestMatrixExecutor_AggregateResults(t *testing.T) {
	executor := &DefaultPipelineExecutor{}
	matrixExecutor := NewMatrixExecutor(executor)

	execution := &PipelineExecution{
		Results:        make(map[string]map[string]interface{}),
		WorkspacePaths: make(map[string]string),
		WorktreePaths:  make(map[string]*WorktreeInfo),
	}

	step := &Step{ID: "matrix_step"}

	results := []MatrixResult{
		{
			ItemIndex:     0,
			WorkspacePath: "/tmp/worker_0",
			Output:        map[string]interface{}{"stdout": "output1"},
		},
		{
			ItemIndex:     1,
			WorkspacePath: "/tmp/worker_1",
			Output:        map[string]interface{}{"stdout": "output2"},
		},
		{
			ItemIndex:     2,
			WorkspacePath: "/tmp/worker_2",
			Error:         context.Canceled,
		},
	}

	matrixExecutor.aggregateResults(execution, step, results)

	aggregated := execution.Results[step.ID]
	if aggregated == nil {
		t.Fatal("Expected aggregated results")
	}

	if aggregated["total_workers"] != 3 {
		t.Errorf("Expected total_workers=3, got %v", aggregated["total_workers"])
	}

	if aggregated["success_count"] != 2 {
		t.Errorf("Expected success_count=2, got %v", aggregated["success_count"])
	}

	if aggregated["fail_count"] != 1 {
		t.Errorf("Expected fail_count=1, got %v", aggregated["fail_count"])
	}

	workerResults := aggregated["worker_results"].([]map[string]interface{})
	if len(workerResults) != 2 {
		t.Errorf("Expected 2 worker results, got %d", len(workerResults))
	}
}

func TestMatrixExecutor_Execute_NoStrategy(t *testing.T) {
	executor := &DefaultPipelineExecutor{}
	matrixExecutor := NewMatrixExecutor(executor)

	execution := &PipelineExecution{
		Pipeline:      &Pipeline{Metadata: PipelineMetadata{Name: "test"}},
		Status:        &PipelineStatus{ID: "test", PipelineName: "test"},
		WorktreePaths: make(map[string]*WorktreeInfo),
	}

	step := &Step{ID: "test_step"}

	err := matrixExecutor.Execute(context.Background(), execution, step)
	if err == nil {
		t.Error("Expected error for step without matrix strategy")
	}
}

func TestMatrixExecutor_Execute_EmptyItems(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "matrix_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create empty items file
	os.WriteFile(filepath.Join(tmpDir, "empty.json"), []byte("[]"), 0644)

	executor := &DefaultPipelineExecutor{}
	matrixExecutor := NewMatrixExecutor(executor)

	execution := &PipelineExecution{
		Pipeline: &Pipeline{Metadata: PipelineMetadata{Name: "test"}},
		Manifest: &manifest.Manifest{
			Runtime: manifest.Runtime{
				WorkspaceRoot: tmpDir,
			},
		},
		WorkspacePaths: make(map[string]string),
		WorktreePaths:  make(map[string]*WorktreeInfo),
		ArtifactPaths:  make(map[string]string),
		Results:        make(map[string]map[string]interface{}),
		Status:         &PipelineStatus{ID: "test", PipelineName: "test"},
	}

	step := &Step{
		ID: "matrix_step",
		Strategy: &MatrixStrategy{
			Type:        "matrix",
			ItemsSource: filepath.Join(tmpDir, "empty.json"),
		},
	}

	err = matrixExecutor.Execute(context.Background(), execution, step)
	if err != nil {
		t.Errorf("Unexpected error for empty items: %v", err)
	}
}

func TestMatrixExecutor_CreateWorkerWorkspace(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "matrix_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	executor := &DefaultPipelineExecutor{}
	matrixExecutor := NewMatrixExecutor(executor)

	execution := &PipelineExecution{
		Pipeline: &Pipeline{Metadata: PipelineMetadata{Name: "test-pipeline"}},
		Manifest: &manifest.Manifest{
			Runtime: manifest.Runtime{
				WorkspaceRoot: tmpDir,
			},
		},
		Status:        &PipelineStatus{ID: "test-pipeline", PipelineName: "test-pipeline"},
		WorktreePaths: make(map[string]*WorktreeInfo),
	}

	step := &Step{ID: "matrix_step"}

	wsPath, err := matrixExecutor.createWorkerWorkspace(execution, step, 5)
	if err != nil {
		t.Fatalf("Failed to create worker workspace: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "test-pipeline", "matrix_step", "worker_5")
	if wsPath != expectedPath {
		t.Errorf("Expected workspace path %q, got %q", expectedPath, wsPath)
	}

	// Verify the directory was created
	if _, err := os.Stat(wsPath); os.IsNotExist(err) {
		t.Error("Worker workspace directory was not created")
	}
}

// ============================================================================
// T078: Test for matrix spawns correct worker count
// ============================================================================

func TestMatrixExecutor_SpawnsCorrectWorkerCount(t *testing.T) {
	tests := []struct {
		name          string
		itemCount     int
		expectedCount int
	}{
		{
			name:          "single item spawns 1 worker",
			itemCount:     1,
			expectedCount: 1,
		},
		{
			name:          "5 items spawns 5 workers",
			itemCount:     5,
			expectedCount: 5,
		},
		{
			name:          "10 items spawns 10 workers",
			itemCount:     10,
			expectedCount: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "matrix_worker_count_test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Track worker spawns
			workerSpawns := &workerSpawnTracker{
				spawned: make(map[int]bool),
			}

			// Create items file
			items := make([]map[string]interface{}, tt.itemCount)
			for i := 0; i < tt.itemCount; i++ {
				items[i] = map[string]interface{}{"id": i, "name": fmt.Sprintf("task_%d", i)}
			}
			itemsJSON, _ := json.Marshal(items)
			itemsFile := filepath.Join(tmpDir, "items.json")
			os.WriteFile(itemsFile, itemsJSON, 0644)

			// Create a tracking adapter
			trackingAdapter := &workerTrackingAdapter{
				tracker: workerSpawns,
				baseAdapter: adapter.NewMockAdapter(
					adapter.WithStdoutJSON(`{"status": "success"}`),
					adapter.WithTokensUsed(100),
				),
			}

			executor := NewDefaultPipelineExecutor(trackingAdapter)
			matrixExecutor := NewMatrixExecutor(executor)

			m := &manifest.Manifest{
				Personas: map[string]manifest.Persona{
					"worker": {Adapter: "claude"},
				},
				Adapters: map[string]manifest.Adapter{
					"claude": {Binary: "claude"},
				},
				Runtime: manifest.Runtime{
					WorkspaceRoot: tmpDir,
				},
			}

			execution := &PipelineExecution{
				Pipeline: &Pipeline{Metadata: PipelineMetadata{Name: "worker-count-test"}},
				Manifest: m,
				States:   make(map[string]string),
				Results:  make(map[string]map[string]interface{}),
				ArtifactPaths:  make(map[string]string),
				WorkspacePaths: make(map[string]string),
				WorktreePaths:  make(map[string]*WorktreeInfo),
				Context:        NewPipelineContext("worker-count-test", "worker-count-test", "matrix_step"), // Fix: Add missing context
				Status:         &PipelineStatus{ID: "worker-count-test", PipelineName: "worker-count-test"},
			}

			step := &Step{
				ID:      "matrix_step",
				Persona: "worker",
				Strategy: &MatrixStrategy{
					Type:        "matrix",
					ItemsSource: itemsFile,
				},
				Exec: ExecConfig{
					Type:   "prompt",
					Source: "Process: {{ task }}",
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			err = matrixExecutor.Execute(ctx, execution, step)
			if err != nil {
				t.Fatalf("Matrix execution failed: %v", err)
			}

			// Verify correct number of workers spawned
			workerSpawns.mu.Lock()
			actualCount := len(workerSpawns.spawned)
			workerSpawns.mu.Unlock()

			if actualCount != tt.expectedCount {
				t.Errorf("Expected %d workers, got %d", tt.expectedCount, actualCount)
			}

			// Verify results aggregation
			if results, ok := execution.Results[step.ID]; ok {
				if totalWorkers, ok := results["total_workers"].(int); ok {
					if totalWorkers != tt.expectedCount {
						t.Errorf("Expected total_workers=%d, got %d", tt.expectedCount, totalWorkers)
					}
				}
			}
		})
	}
}

// ============================================================================
// T079: Test for max_concurrency limit
// ============================================================================

func TestMatrixExecutor_MaxConcurrencyLimit(t *testing.T) {
	tests := []struct {
		name           string
		itemCount      int
		maxConcurrency int
		expectedMax    int
	}{
		{
			name:           "concurrency of 2 with 5 items",
			itemCount:      5,
			maxConcurrency: 2,
			expectedMax:    2,
		},
		{
			name:           "concurrency of 3 with 10 items",
			itemCount:      10,
			maxConcurrency: 3,
			expectedMax:    3,
		},
		{
			name:           "concurrency equals item count",
			itemCount:      4,
			maxConcurrency: 4,
			expectedMax:    4,
		},
		{
			name:           "concurrency exceeds item count",
			itemCount:      3,
			maxConcurrency: 10,
			expectedMax:    3, // Limited by actual items
		},
		{
			name:           "zero concurrency means unlimited",
			itemCount:      5,
			maxConcurrency: 0,
			expectedMax:    5, // All items can run in parallel
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "matrix_concurrency_test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Track concurrent execution
			concurrencyTracker := &concurrencyLimitTracker{
				maxObserved: 0,
				current:     0,
			}

			// Create items file
			items := make([]map[string]interface{}, tt.itemCount)
			for i := 0; i < tt.itemCount; i++ {
				items[i] = map[string]interface{}{"id": i}
			}
			itemsJSON, _ := json.Marshal(items)
			itemsFile := filepath.Join(tmpDir, "items.json")
			os.WriteFile(itemsFile, itemsJSON, 0644)

			// Create adapter that tracks concurrency
			concurrencyAdapter := &concurrencyTrackingMatrixAdapter{
				tracker: concurrencyTracker,
				baseAdapter: adapter.NewMockAdapter(
					adapter.WithStdoutJSON(`{"status": "success"}`),
					adapter.WithTokensUsed(100),
					adapter.WithSimulatedDelay(50*time.Millisecond), // Add delay to test concurrency
				),
			}

			executor := NewDefaultPipelineExecutor(concurrencyAdapter)
			matrixExecutor := NewMatrixExecutor(executor)

			m := &manifest.Manifest{
				Personas: map[string]manifest.Persona{
					"worker": {Adapter: "claude"},
				},
				Adapters: map[string]manifest.Adapter{
					"claude": {Binary: "claude"},
				},
				Runtime: manifest.Runtime{
					WorkspaceRoot: tmpDir,
				},
			}

			execution := &PipelineExecution{
				Pipeline: &Pipeline{Metadata: PipelineMetadata{Name: "concurrency-test"}},
				Manifest: m,
				States:   make(map[string]string),
				Results:  make(map[string]map[string]interface{}),
				ArtifactPaths:  make(map[string]string),
				WorkspacePaths: make(map[string]string),
				WorktreePaths:  make(map[string]*WorktreeInfo),
				Context:        NewPipelineContext("concurrency-test", "concurrency-test", "matrix_step"), // Fix: Add missing context
				Status:         &PipelineStatus{ID: "concurrency-test", PipelineName: "concurrency-test"},
			}

			step := &Step{
				ID:      "matrix_step",
				Persona: "worker",
				Strategy: &MatrixStrategy{
					Type:           "matrix",
					ItemsSource:    itemsFile,
					MaxConcurrency: tt.maxConcurrency,
				},
				Exec: ExecConfig{
					Type:   "prompt",
					Source: "Process: {{ task }}",
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			err = matrixExecutor.Execute(ctx, execution, step)
			if err != nil {
				t.Fatalf("Matrix execution failed: %v", err)
			}

			// Verify concurrency limit was respected
			concurrencyTracker.mu.Lock()
			maxObserved := concurrencyTracker.maxObserved
			concurrencyTracker.mu.Unlock()

			if maxObserved > tt.expectedMax {
				t.Errorf("Concurrency exceeded limit: observed %d, expected max %d", maxObserved, tt.expectedMax)
			}

			// With delay, we should hit near the expected max
			// Allow some tolerance for timing issues
			if tt.maxConcurrency > 0 && maxObserved < tt.expectedMax-1 && tt.itemCount >= tt.maxConcurrency {
				t.Logf("Note: max observed concurrency (%d) was lower than expected (%d), which may be due to timing", maxObserved, tt.expectedMax)
			}
		})
	}
}

// ============================================================================
// T080: Test for partial failure handling
// ============================================================================

func TestMatrixExecutor_PartialFailureHandling(t *testing.T) {
	tests := []struct {
		name               string
		itemCount          int
		failingIndices     []int
		expectOverallError bool
		expectSuccessCount int
		expectFailCount    int
	}{
		{
			name:               "first worker fails, others continue",
			itemCount:          5,
			failingIndices:     []int{0},
			expectOverallError: true,
			expectSuccessCount: 4,
			expectFailCount:    1,
		},
		{
			name:               "middle worker fails, others continue",
			itemCount:          5,
			failingIndices:     []int{2},
			expectOverallError: true,
			expectSuccessCount: 4,
			expectFailCount:    1,
		},
		{
			name:               "multiple workers fail",
			itemCount:          5,
			failingIndices:     []int{1, 3},
			expectOverallError: true,
			expectSuccessCount: 3,
			expectFailCount:    2,
		},
		{
			name:               "all workers fail",
			itemCount:          3,
			failingIndices:     []int{0, 1, 2},
			expectOverallError: true,
			expectSuccessCount: 0,
			expectFailCount:    3,
		},
		{
			name:               "no workers fail",
			itemCount:          3,
			failingIndices:     []int{},
			expectOverallError: false,
			expectSuccessCount: 3,
			expectFailCount:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "matrix_partial_failure_test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Create items file
			items := make([]map[string]interface{}, tt.itemCount)
			for i := 0; i < tt.itemCount; i++ {
				items[i] = map[string]interface{}{"id": i}
			}
			itemsJSON, _ := json.Marshal(items)
			itemsFile := filepath.Join(tmpDir, "items.json")
			os.WriteFile(itemsFile, itemsJSON, 0644)

			// Create adapter that fails for specific indices
			failingSet := make(map[int]bool)
			for _, idx := range tt.failingIndices {
				failingSet[idx] = true
			}

			partialFailAdapter := &partialFailureAdapter{
				failingIndices: failingSet,
				callCount:      0,
				baseAdapter: adapter.NewMockAdapter(
					adapter.WithStdoutJSON(`{"status": "success"}`),
					adapter.WithTokensUsed(100),
				),
			}

			// Collect events to verify failure reporting
			eventCollector := newMatrixTestEventCollector()

			executor := NewDefaultPipelineExecutor(partialFailAdapter, WithEmitter(eventCollector))
			matrixExecutor := NewMatrixExecutor(executor)

			m := &manifest.Manifest{
				Personas: map[string]manifest.Persona{
					"worker": {Adapter: "claude"},
				},
				Adapters: map[string]manifest.Adapter{
					"claude": {Binary: "claude"},
				},
				Runtime: manifest.Runtime{
					WorkspaceRoot: tmpDir,
				},
			}

			execution := &PipelineExecution{
				Pipeline: &Pipeline{Metadata: PipelineMetadata{Name: "partial-failure-test"}},
				Manifest: m,
				States:   make(map[string]string),
				Results:  make(map[string]map[string]interface{}),
				ArtifactPaths:  make(map[string]string),
				WorkspacePaths: make(map[string]string),
				WorktreePaths:  make(map[string]*WorktreeInfo),
				Status:         &PipelineStatus{ID: "partial-failure-test", PipelineName: "partial-failure-test"},
			}

			step := &Step{
				ID:      "matrix_step",
				Persona: "worker",
				Strategy: &MatrixStrategy{
					Type:        "matrix",
					ItemsSource: itemsFile,
				},
				Exec: ExecConfig{
					Type:   "prompt",
					Source: "Process: {{ task }}",
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			err = matrixExecutor.Execute(ctx, execution, step)

			// Check if error was returned as expected
			if tt.expectOverallError {
				if err == nil {
					t.Error("Expected error due to partial failures, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Did not expect error, got: %v", err)
				}
			}

			// Verify aggregated results
			if results, ok := execution.Results[step.ID]; ok {
				successCount, _ := results["success_count"].(int)
				failCount, _ := results["fail_count"].(int)

				if successCount != tt.expectSuccessCount {
					t.Errorf("Expected success_count=%d, got %d", tt.expectSuccessCount, successCount)
				}

				if failCount != tt.expectFailCount {
					t.Errorf("Expected fail_count=%d, got %d", tt.expectFailCount, failCount)
				}
			}

			// Verify failure events were emitted
			events := eventCollector.GetEvents()
			workerFailedCount := 0
			for _, e := range events {
				if e.State == "matrix_worker_failed" {
					workerFailedCount++
				}
			}

			if workerFailedCount != tt.expectFailCount {
				t.Errorf("Expected %d matrix_worker_failed events, got %d", tt.expectFailCount, workerFailedCount)
			}
		})
	}
}

// ============================================================================
// T081: Test for zero tasks in matrix
// ============================================================================

func TestMatrixExecutor_ZeroTasks(t *testing.T) {
	tests := []struct {
		name        string
		items       interface{}
		expectError bool
		description string
	}{
		{
			name:        "empty array returns early without error",
			items:       []interface{}{},
			expectError: false,
			description: "Empty task list should complete successfully with no workers",
		},
		{
			name:        "null array in nested structure",
			items:       map[string]interface{}{"tasks": []interface{}{}},
			expectError: false,
			description: "Empty nested array should complete successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "matrix_zero_tasks_test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Create items file
			var itemsJSON []byte
			switch v := tt.items.(type) {
			case []interface{}:
				itemsJSON, _ = json.Marshal(v)
			case map[string]interface{}:
				itemsJSON, _ = json.Marshal(v)
			}
			itemsFile := filepath.Join(tmpDir, "items.json")
			os.WriteFile(itemsFile, itemsJSON, 0644)

			eventCollector := newMatrixTestEventCollector()

			executor := NewDefaultPipelineExecutor(
				adapter.NewMockAdapter(adapter.WithStdoutJSON(`{"status": "success"}`)),
				WithEmitter(eventCollector),
			)
			matrixExecutor := NewMatrixExecutor(executor)

			m := &manifest.Manifest{
				Personas: map[string]manifest.Persona{
					"worker": {Adapter: "claude"},
				},
				Adapters: map[string]manifest.Adapter{
					"claude": {Binary: "claude"},
				},
				Runtime: manifest.Runtime{
					WorkspaceRoot: tmpDir,
				},
			}

			execution := &PipelineExecution{
				Pipeline: &Pipeline{Metadata: PipelineMetadata{Name: "zero-tasks-test"}},
				Manifest: m,
				States:   make(map[string]string),
				Results:  make(map[string]map[string]interface{}),
				ArtifactPaths:  make(map[string]string),
				WorkspacePaths: make(map[string]string),
				WorktreePaths:  make(map[string]*WorktreeInfo),
				Status:         &PipelineStatus{ID: "zero-tasks-test", PipelineName: "zero-tasks-test"},
			}

			strategy := &MatrixStrategy{
				Type:        "matrix",
				ItemsSource: itemsFile,
			}

			// Add item_key for nested structure
			if _, ok := tt.items.(map[string]interface{}); ok {
				strategy.ItemKey = "tasks"
			}

			step := &Step{
				ID:       "matrix_step",
				Persona:  "worker",
				Strategy: strategy,
				Exec: ExecConfig{
					Type:   "prompt",
					Source: "Process: {{ task }}",
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err = matrixExecutor.Execute(ctx, execution, step)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Did not expect error, got: %v", err)
				}
			}

			// Verify matrix_complete event was emitted
			events := eventCollector.GetEvents()
			hasComplete := false
			for _, e := range events {
				if e.State == "matrix_complete" {
					hasComplete = true
					if e.Message != "No items to process" {
						t.Logf("matrix_complete message: %s", e.Message)
					}
				}
			}

			if !hasComplete {
				t.Error("Expected matrix_complete event to be emitted")
			}

			// No worker events should be emitted
			for _, e := range events {
				if e.State == "matrix_worker_start" || e.State == "matrix_worker_complete" {
					t.Error("No worker events should be emitted for zero tasks")
				}
			}
		})
	}
}

// TestMatrixExecutor_ZeroTasksEdgeCases tests additional edge cases for zero/empty task handling
func TestMatrixExecutor_ZeroTasksEdgeCases(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "matrix_zero_edge_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("missing items_source file", func(t *testing.T) {
		executor := NewDefaultPipelineExecutor(adapter.NewMockAdapter())
		matrixExecutor := NewMatrixExecutor(executor)

		execution := &PipelineExecution{
			Pipeline: &Pipeline{Metadata: PipelineMetadata{Name: "missing-file-test"}},
			Manifest: &manifest.Manifest{
				Runtime: manifest.Runtime{WorkspaceRoot: tmpDir},
			},
			WorkspacePaths: make(map[string]string),
			WorktreePaths:  make(map[string]*WorktreeInfo),
			ArtifactPaths:  make(map[string]string),
			Status:         &PipelineStatus{ID: "missing-file-test", PipelineName: "missing-file-test"},
		}

		step := &Step{
			ID: "matrix_step",
			Strategy: &MatrixStrategy{
				Type:        "matrix",
				ItemsSource: "/nonexistent/path/items.json",
			},
		}

		err := matrixExecutor.Execute(context.Background(), execution, step)
		if err == nil {
			t.Error("Expected error for missing items_source file")
		}
	})

	t.Run("invalid JSON in items_source", func(t *testing.T) {
		invalidJSONFile := filepath.Join(tmpDir, "invalid.json")
		os.WriteFile(invalidJSONFile, []byte("not valid json{"), 0644)

		executor := NewDefaultPipelineExecutor(adapter.NewMockAdapter())
		matrixExecutor := NewMatrixExecutor(executor)

		execution := &PipelineExecution{
			Pipeline: &Pipeline{Metadata: PipelineMetadata{Name: "invalid-json-test"}},
			Manifest: &manifest.Manifest{
				Runtime: manifest.Runtime{WorkspaceRoot: tmpDir},
			},
			WorkspacePaths: make(map[string]string),
			WorktreePaths:  make(map[string]*WorktreeInfo),
			ArtifactPaths:  make(map[string]string),
			Status:         &PipelineStatus{ID: "invalid-json-test", PipelineName: "invalid-json-test"},
		}

		step := &Step{
			ID: "matrix_step",
			Strategy: &MatrixStrategy{
				Type:        "matrix",
				ItemsSource: invalidJSONFile,
			},
		}

		err := matrixExecutor.Execute(context.Background(), execution, step)
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
	})

	t.Run("items_source is object not array", func(t *testing.T) {
		objectJSONFile := filepath.Join(tmpDir, "object.json")
		os.WriteFile(objectJSONFile, []byte(`{"key": "value"}`), 0644)

		executor := NewDefaultPipelineExecutor(adapter.NewMockAdapter())
		matrixExecutor := NewMatrixExecutor(executor)

		execution := &PipelineExecution{
			Pipeline: &Pipeline{Metadata: PipelineMetadata{Name: "object-json-test"}},
			Manifest: &manifest.Manifest{
				Runtime: manifest.Runtime{WorkspaceRoot: tmpDir},
			},
			WorkspacePaths: make(map[string]string),
			WorktreePaths:  make(map[string]*WorktreeInfo),
			ArtifactPaths:  make(map[string]string),
			Status:         &PipelineStatus{ID: "object-json-test", PipelineName: "object-json-test"},
		}

		step := &Step{
			ID: "matrix_step",
			Strategy: &MatrixStrategy{
				Type:        "matrix",
				ItemsSource: objectJSONFile,
				// No item_key means it will try to use root as array
			},
		}

		err := matrixExecutor.Execute(context.Background(), execution, step)
		if err == nil {
			t.Error("Expected error when items_source is not an array")
		}
	})
}

// ============================================================================
// Helper types for matrix tests
// ============================================================================

// workerSpawnTracker tracks which workers were spawned
type workerSpawnTracker struct {
	mu      sync.Mutex
	spawned map[int]bool
}

// workerTrackingAdapter tracks worker execution
type workerTrackingAdapter struct {
	tracker     *workerSpawnTracker
	baseAdapter adapter.AdapterRunner
	mu          sync.Mutex
	callIndex   int
}

func (a *workerTrackingAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	a.mu.Lock()
	idx := a.callIndex
	a.callIndex++
	a.mu.Unlock()

	a.tracker.mu.Lock()
	a.tracker.spawned[idx] = true
	a.tracker.mu.Unlock()

	return a.baseAdapter.Run(ctx, cfg)
}

// concurrencyLimitTracker tracks concurrent executions
type concurrencyLimitTracker struct {
	mu          sync.Mutex
	current     int
	maxObserved int
}

// concurrencyTrackingMatrixAdapter tracks and enforces concurrency for matrix
type concurrencyTrackingMatrixAdapter struct {
	tracker     *concurrencyLimitTracker
	baseAdapter adapter.AdapterRunner
}

func (a *concurrencyTrackingMatrixAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	a.tracker.mu.Lock()
	a.tracker.current++
	if a.tracker.current > a.tracker.maxObserved {
		a.tracker.maxObserved = a.tracker.current
	}
	a.tracker.mu.Unlock()

	defer func() {
		a.tracker.mu.Lock()
		a.tracker.current--
		a.tracker.mu.Unlock()
	}()

	return a.baseAdapter.Run(ctx, cfg)
}

// partialFailureAdapter fails for specific worker indices
type partialFailureAdapter struct {
	failingIndices map[int]bool
	mu             sync.Mutex
	callCount      int
	baseAdapter    adapter.AdapterRunner
}

func (a *partialFailureAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	a.mu.Lock()
	idx := a.callCount
	a.callCount++
	shouldFail := a.failingIndices[idx]
	a.mu.Unlock()

	if shouldFail {
		return nil, fmt.Errorf("simulated failure for worker %d", idx)
	}

	return a.baseAdapter.Run(ctx, cfg)
}
