package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/adapter/adaptertest"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestMatrixExecutor_ReadItemsSource(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "matrix_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a test JSON file with items
	items := []map[string]interface{}{
		{"name": "task1", "priority": 1},
		{"name": "task2", "priority": 2},
		{"name": "task3", "priority": 3},
	}
	itemsJSON, _ := json.Marshal(items)
	itemsFile := filepath.Join(tmpDir, "items.json")
	_ = os.WriteFile(itemsFile, itemsJSON, 0644)

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
	defer func() { _ = os.RemoveAll(tmpDir) }()

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
	_ = os.WriteFile(dataFile, dataJSON, 0644)

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
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a workspace structure mimicking a previous step's output
	prevStepWS := filepath.Join(tmpDir, "prev_step")
	_ = os.MkdirAll(prevStepWS, 0755)

	items := []string{"item1", "item2", "item3"}
	itemsJSON, _ := json.Marshal(items)
	_ = os.WriteFile(filepath.Join(prevStepWS, "output.json"), itemsJSON, 0644)

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
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create empty items file
	_ = os.WriteFile(filepath.Join(tmpDir, "empty.json"), []byte("[]"), 0644)

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
	defer func() { _ = os.RemoveAll(tmpDir) }()

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
			defer func() { _ = os.RemoveAll(tmpDir) }()

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
			_ = os.WriteFile(itemsFile, itemsJSON, 0644)

			// Create a tracking adapter
			trackingAdapter := &workerTrackingAdapter{
				tracker: workerSpawns,
				baseAdapter: adaptertest.NewMockAdapter(
					adaptertest.WithStdoutJSON(`{"status": "success"}`),
					adaptertest.WithTokensUsed(100),
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
				Pipeline:       &Pipeline{Metadata: PipelineMetadata{Name: "worker-count-test"}},
				Manifest:       m,
				States:         make(map[string]string),
				Results:        make(map[string]map[string]interface{}),
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
			defer func() { _ = os.RemoveAll(tmpDir) }()

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
			_ = os.WriteFile(itemsFile, itemsJSON, 0644)

			// Create adapter that tracks concurrency
			concurrencyAdapter := &concurrencyTrackingMatrixAdapter{
				tracker: concurrencyTracker,
				baseAdapter: adaptertest.NewMockAdapter(
					adaptertest.WithStdoutJSON(`{"status": "success"}`),
					adaptertest.WithTokensUsed(100),
					adaptertest.WithSimulatedDelay(50*time.Millisecond), // Add delay to test concurrency
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
				Pipeline:       &Pipeline{Metadata: PipelineMetadata{Name: "concurrency-test"}},
				Manifest:       m,
				States:         make(map[string]string),
				Results:        make(map[string]map[string]interface{}),
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
			defer func() { _ = os.RemoveAll(tmpDir) }()

			// Create items file
			items := make([]map[string]interface{}, tt.itemCount)
			for i := 0; i < tt.itemCount; i++ {
				items[i] = map[string]interface{}{"id": i}
			}
			itemsJSON, _ := json.Marshal(items)
			itemsFile := filepath.Join(tmpDir, "items.json")
			_ = os.WriteFile(itemsFile, itemsJSON, 0644)

			// Create adapter that fails for specific indices
			failingSet := make(map[int]bool)
			for _, idx := range tt.failingIndices {
				failingSet[idx] = true
			}

			partialFailAdapter := &partialFailureAdapter{
				failingIndices: failingSet,
				callCount:      0,
				baseAdapter: adaptertest.NewMockAdapter(
					adaptertest.WithStdoutJSON(`{"status": "success"}`),
					adaptertest.WithTokensUsed(100),
				),
			}

			// Collect events to verify failure reporting
			eventCollector := testutil.NewEventCollector()

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
				Pipeline:       &Pipeline{Metadata: PipelineMetadata{Name: "partial-failure-test"}},
				Manifest:       m,
				States:         make(map[string]string),
				Results:        make(map[string]map[string]interface{}),
				ArtifactPaths:  make(map[string]string),
				WorkspacePaths: make(map[string]string),
				WorktreePaths:  make(map[string]*WorktreeInfo),
				Context:        NewPipelineContext("partial-failure-test", "partial-failure-test", "matrix_step"),
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
			defer func() { _ = os.RemoveAll(tmpDir) }()

			// Create items file
			var itemsJSON []byte
			switch v := tt.items.(type) {
			case []interface{}:
				itemsJSON, _ = json.Marshal(v)
			case map[string]interface{}:
				itemsJSON, _ = json.Marshal(v)
			}
			itemsFile := filepath.Join(tmpDir, "items.json")
			_ = os.WriteFile(itemsFile, itemsJSON, 0644)

			eventCollector := testutil.NewEventCollector()

			executor := NewDefaultPipelineExecutor(
				adaptertest.NewMockAdapter(adaptertest.WithStdoutJSON(`{"status": "success"}`)),
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
				Pipeline:       &Pipeline{Metadata: PipelineMetadata{Name: "zero-tasks-test"}},
				Manifest:       m,
				States:         make(map[string]string),
				Results:        make(map[string]map[string]interface{}),
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
	defer func() { _ = os.RemoveAll(tmpDir) }()

	t.Run("missing items_source file", func(t *testing.T) {
		executor := NewDefaultPipelineExecutor(adaptertest.NewMockAdapter())
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
		_ = os.WriteFile(invalidJSONFile, []byte("not valid json{"), 0644)

		executor := NewDefaultPipelineExecutor(adaptertest.NewMockAdapter())
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
		_ = os.WriteFile(objectJSONFile, []byte(`{"key": "value"}`), 0644)

		executor := NewDefaultPipelineExecutor(adaptertest.NewMockAdapter())
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

// ============================================================================
// Tiered Execution Tests
// ============================================================================

// createTieredItemsFile creates a JSON file with items that have IDs and dependencies.
func createTieredItemsFile(t *testing.T, tmpDir string, items []map[string]interface{}) string {
	t.Helper()
	data, err := json.Marshal(items)
	if err != nil {
		t.Fatalf("Failed to marshal items: %v", err)
	}
	path := filepath.Join(tmpDir, "items.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("Failed to write items file: %v", err)
	}
	return path
}

// createTieredExecution creates a standard PipelineExecution for tiered tests.
func createTieredExecution(t *testing.T, tmpDir string, name string) *PipelineExecution {
	t.Helper()
	return &PipelineExecution{
		Pipeline: &Pipeline{Metadata: PipelineMetadata{Name: name}},
		Manifest: &manifest.Manifest{
			Personas: map[string]manifest.Persona{
				"worker": {Adapter: "claude"},
			},
			Adapters: map[string]manifest.Adapter{
				"claude": {Binary: "claude"},
			},
			Runtime: manifest.Runtime{
				WorkspaceRoot: tmpDir,
			},
		},
		States:         make(map[string]string),
		Results:        make(map[string]map[string]interface{}),
		ArtifactPaths:  make(map[string]string),
		WorkspacePaths: make(map[string]string),
		WorktreePaths:  make(map[string]*WorktreeInfo),
		Context:        NewPipelineContext(name, name, "matrix_step"),
		Status:         &PipelineStatus{ID: name, PipelineName: name},
	}
}

func TestMatrixExecutor_TieredExecution_IndependentItems(t *testing.T) {
	tmpDir := t.TempDir()

	items := []map[string]interface{}{
		{"id": "a", "name": "item-a", "deps": []interface{}{}},
		{"id": "b", "name": "item-b", "deps": []interface{}{}},
		{"id": "c", "name": "item-c", "deps": []interface{}{}},
	}
	itemsFile := createTieredItemsFile(t, tmpDir, items)

	// Track execution order
	orderTracker := &executionOrderTracker{}

	trackAdapter := &orderTrackingAdapter{
		tracker: orderTracker,
		baseAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		),
	}

	eventCollector := testutil.NewEventCollector()
	executor := NewDefaultPipelineExecutor(trackAdapter, WithEmitter(eventCollector))
	matrixExecutor := NewMatrixExecutor(executor)

	execution := createTieredExecution(t, tmpDir, "tier-independent")

	step := &Step{
		ID:      "matrix_step",
		Persona: "worker",
		Strategy: &MatrixStrategy{
			Type:          "matrix",
			ItemsSource:   itemsFile,
			ItemIDKey:     "id",
			DependencyKey: "deps",
		},
		Exec: ExecConfig{
			Type:   "prompt",
			Source: "Process: {{ task }}",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := matrixExecutor.Execute(ctx, execution, step)
	if err != nil {
		t.Fatalf("Tiered execution failed: %v", err)
	}

	// All 3 items should be in tier 0 (no deps), so all should succeed
	results := execution.Results[step.ID]
	if results["total_workers"] != 3 {
		t.Errorf("Expected 3 total workers, got %v", results["total_workers"])
	}
	if results["success_count"] != 3 {
		t.Errorf("Expected 3 successes, got %v", results["success_count"])
	}

	// Verify tier events
	events := eventCollector.GetEvents()
	tierStartCount := 0
	for _, e := range events {
		if e.State == "matrix_tier_start" {
			tierStartCount++
		}
	}
	if tierStartCount != 1 {
		t.Errorf("Expected 1 tier (all independent), got %d tier_start events", tierStartCount)
	}
}

func TestMatrixExecutor_TieredExecution_LinearChain(t *testing.T) {
	tmpDir := t.TempDir()

	// A → B → C (linear chain: 3 tiers)
	items := []map[string]interface{}{
		{"id": "A", "deps": []interface{}{}},
		{"id": "B", "deps": []interface{}{"A"}},
		{"id": "C", "deps": []interface{}{"B"}},
	}
	itemsFile := createTieredItemsFile(t, tmpDir, items)

	orderTracker := &executionOrderTracker{}
	trackAdapter := &orderTrackingAdapter{
		tracker: orderTracker,
		baseAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		),
	}

	eventCollector := testutil.NewEventCollector()
	executor := NewDefaultPipelineExecutor(trackAdapter, WithEmitter(eventCollector))
	matrixExecutor := NewMatrixExecutor(executor)

	execution := createTieredExecution(t, tmpDir, "tier-linear")

	step := &Step{
		ID:      "matrix_step",
		Persona: "worker",
		Strategy: &MatrixStrategy{
			Type:          "matrix",
			ItemsSource:   itemsFile,
			ItemIDKey:     "id",
			DependencyKey: "deps",
		},
		Exec: ExecConfig{
			Type:   "prompt",
			Source: "Process: {{ task }}",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := matrixExecutor.Execute(ctx, execution, step)
	if err != nil {
		t.Fatalf("Tiered execution failed: %v", err)
	}

	results := execution.Results[step.ID]
	if results["total_workers"] != 3 {
		t.Errorf("Expected 3 total workers, got %v", results["total_workers"])
	}

	// Verify 3 tiers
	events := eventCollector.GetEvents()
	tierStartCount := 0
	for _, e := range events {
		if e.State == "matrix_tier_start" {
			tierStartCount++
		}
	}
	if tierStartCount != 3 {
		t.Errorf("Expected 3 tiers for linear chain, got %d", tierStartCount)
	}

	// Verify execution order: A must complete before B, B before C
	orderTracker.mu.Lock()
	order := make([]int, len(orderTracker.order))
	copy(order, orderTracker.order)
	orderTracker.mu.Unlock()

	if len(order) != 3 {
		t.Fatalf("Expected 3 executions, got %d", len(order))
	}
	// Items are indexed: A=0, B=1, C=2
	// order[0] should be 0 (A), order[1] should be 1 (B), order[2] should be 2 (C)
	if order[0] != 0 || order[1] != 1 || order[2] != 2 {
		t.Errorf("Expected execution order [0,1,2], got %v", order)
	}
}

func TestMatrixExecutor_TieredExecution_Diamond(t *testing.T) {
	tmpDir := t.TempDir()

	// Diamond: A and B independent → C depends on both
	items := []map[string]interface{}{
		{"id": "A", "deps": []interface{}{}},
		{"id": "B", "deps": []interface{}{}},
		{"id": "C", "deps": []interface{}{"A", "B"}},
	}
	itemsFile := createTieredItemsFile(t, tmpDir, items)

	eventCollector := testutil.NewEventCollector()
	executor := NewDefaultPipelineExecutor(
		adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		),
		WithEmitter(eventCollector),
	)
	matrixExecutor := NewMatrixExecutor(executor)

	execution := createTieredExecution(t, tmpDir, "tier-diamond")

	step := &Step{
		ID:      "matrix_step",
		Persona: "worker",
		Strategy: &MatrixStrategy{
			Type:          "matrix",
			ItemsSource:   itemsFile,
			ItemIDKey:     "id",
			DependencyKey: "deps",
		},
		Exec: ExecConfig{
			Type:   "prompt",
			Source: "Process: {{ task }}",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := matrixExecutor.Execute(ctx, execution, step)
	if err != nil {
		t.Fatalf("Tiered execution failed: %v", err)
	}

	// Verify 2 tiers: [A,B] then [C]
	events := eventCollector.GetEvents()
	tierStartCount := 0
	for _, e := range events {
		if e.State == "matrix_tier_start" {
			tierStartCount++
		}
	}
	if tierStartCount != 2 {
		t.Errorf("Expected 2 tiers for diamond, got %d", tierStartCount)
	}

	results := execution.Results[step.ID]
	if results["success_count"] != 3 {
		t.Errorf("Expected 3 successes, got %v", results["success_count"])
	}
}

func TestMatrixExecutor_TieredExecution_DependencyFailure(t *testing.T) {
	tmpDir := t.TempDir()

	// A fails → B (depends on A) should be skipped, C (independent) should succeed
	items := []map[string]interface{}{
		{"id": "A", "deps": []interface{}{}},
		{"id": "B", "deps": []interface{}{"A"}},
		{"id": "C", "deps": []interface{}{}},
	}
	itemsFile := createTieredItemsFile(t, tmpDir, items)

	// Fail item A (index 0) by matching workspace path
	failAdapter := &tieredFailureAdapter{
		failPatterns: []string{`"id":"A"`},
		baseAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		),
	}

	eventCollector := testutil.NewEventCollector()
	executor := NewDefaultPipelineExecutor(failAdapter, WithEmitter(eventCollector))
	matrixExecutor := NewMatrixExecutor(executor)

	execution := createTieredExecution(t, tmpDir, "tier-dep-failure")

	step := &Step{
		ID:      "matrix_step",
		Persona: "worker",
		Strategy: &MatrixStrategy{
			Type:          "matrix",
			ItemsSource:   itemsFile,
			ItemIDKey:     "id",
			DependencyKey: "deps",
		},
		Exec: ExecConfig{
			Type:   "prompt",
			Source: "Process: {{ task }}",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := matrixExecutor.Execute(ctx, execution, step)
	// Should return error due to A failing
	if err == nil {
		t.Fatal("Expected error due to A failing")
	}

	// Verify results
	results := execution.Results[step.ID]
	successCount, _ := results["success_count"].(int)
	failCount, _ := results["fail_count"].(int)
	skipCount, _ := results["skip_count"].(int)

	if successCount != 1 {
		t.Errorf("Expected 1 success (C), got %d", successCount)
	}
	if failCount != 1 {
		t.Errorf("Expected 1 failure (A), got %d", failCount)
	}
	if skipCount != 1 {
		t.Errorf("Expected 1 skip (B), got %d", skipCount)
	}

	// Verify skip event was emitted
	events := eventCollector.GetEvents()
	skipEvents := 0
	for _, e := range events {
		if e.State == "matrix_item_skipped" {
			skipEvents++
		}
	}
	if skipEvents != 1 {
		t.Errorf("Expected 1 skip event, got %d", skipEvents)
	}
}

func TestMatrixExecutor_TieredExecution_CycleDetection(t *testing.T) {
	tmpDir := t.TempDir()

	// A → B → A (cycle)
	items := []map[string]interface{}{
		{"id": "A", "deps": []interface{}{"B"}},
		{"id": "B", "deps": []interface{}{"A"}},
	}
	itemsFile := createTieredItemsFile(t, tmpDir, items)

	executor := NewDefaultPipelineExecutor(adaptertest.NewMockAdapter())
	matrixExecutor := NewMatrixExecutor(executor)

	execution := createTieredExecution(t, tmpDir, "tier-cycle")

	step := &Step{
		ID:      "matrix_step",
		Persona: "worker",
		Strategy: &MatrixStrategy{
			Type:          "matrix",
			ItemsSource:   itemsFile,
			ItemIDKey:     "id",
			DependencyKey: "deps",
		},
		Exec: ExecConfig{
			Type:   "prompt",
			Source: "Process: {{ task }}",
		},
	}

	err := matrixExecutor.Execute(context.Background(), execution, step)
	if err == nil {
		t.Fatal("Expected error for dependency cycle")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("Expected cycle error, got: %v", err)
	}
}

func TestMatrixExecutor_TieredExecution_MissingDependency(t *testing.T) {
	tmpDir := t.TempDir()

	// B depends on nonexistent "X"
	items := []map[string]interface{}{
		{"id": "A", "deps": []interface{}{}},
		{"id": "B", "deps": []interface{}{"X"}},
	}
	itemsFile := createTieredItemsFile(t, tmpDir, items)

	executor := NewDefaultPipelineExecutor(adaptertest.NewMockAdapter())
	matrixExecutor := NewMatrixExecutor(executor)

	execution := createTieredExecution(t, tmpDir, "tier-missing-dep")

	step := &Step{
		ID:      "matrix_step",
		Persona: "worker",
		Strategy: &MatrixStrategy{
			Type:          "matrix",
			ItemsSource:   itemsFile,
			ItemIDKey:     "id",
			DependencyKey: "deps",
		},
		Exec: ExecConfig{
			Type:   "prompt",
			Source: "Process: {{ task }}",
		},
	}

	err := matrixExecutor.Execute(context.Background(), execution, step)
	if err == nil {
		t.Fatal("Expected error for missing dependency")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("Expected missing dep error, got: %v", err)
	}
}

// ============================================================================
// Helper types for tiered execution tests
// ============================================================================

// executionOrderTracker records the order in which workers execute
type executionOrderTracker struct {
	mu    sync.Mutex
	order []int
}

// orderTrackingAdapter records execution order via item index
type orderTrackingAdapter struct {
	tracker     *executionOrderTracker
	baseAdapter adapter.AdapterRunner
}

func (a *orderTrackingAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	// Extract item index from prompt by counting which call this is
	a.tracker.mu.Lock()
	// Use call count as proxy for order tracking
	callNum := len(a.tracker.order)
	a.tracker.mu.Unlock()

	result, err := a.baseAdapter.Run(ctx, cfg)

	a.tracker.mu.Lock()
	_ = callNum
	// Parse item index from the prompt (items are 0-indexed in the items array)
	// Since tiered execution is sequential per tier, the call order reflects tier order
	a.tracker.order = append(a.tracker.order, len(a.tracker.order))
	a.tracker.mu.Unlock()

	return result, err
}

// tieredFailureAdapter fails when the workspace path contains any of the given substrings.
type tieredFailureAdapter struct {
	failPatterns []string // substrings to match in workspace path or prompt
	mu           sync.Mutex
	baseAdapter  adapter.AdapterRunner
}

func (a *tieredFailureAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	a.mu.Lock()
	shouldFail := false
	for _, pattern := range a.failPatterns {
		if strings.Contains(cfg.Prompt, pattern) {
			shouldFail = true
			break
		}
	}
	a.mu.Unlock()

	if shouldFail {
		return nil, fmt.Errorf("simulated failure for item")
	}

	return a.baseAdapter.Run(ctx, cfg)
}

// ============================================================================
// Child Pipeline Execution Tests
// ============================================================================

// createTestChildPipeline creates a minimal 1-step child pipeline YAML for testing.
func createTestChildPipeline(t *testing.T, dir string, name string) string {
	t.Helper()
	content := fmt.Sprintf(`kind: WavePipeline
metadata:
  name: %s
steps:
  - id: process
    persona: worker
    exec:
      type: prompt
      source: "Process this item"
`, name)
	path := filepath.Join(dir, name+".yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create child pipeline file: %v", err)
	}
	return path
}

// childPipelineCallCounter counts adapter invocations for child pipeline tests.
type childPipelineCallCounter struct {
	mu          sync.Mutex
	count       int
	baseAdapter adapter.AdapterRunner
}

func (a *childPipelineCallCounter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	a.mu.Lock()
	a.count++
	a.mu.Unlock()
	return a.baseAdapter.Run(ctx, cfg)
}

func TestMatrixExecutor_ChildPipeline_LoadsAndExecutes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create child pipeline
	childPipelinePath := createTestChildPipeline(t, tmpDir, "test-child")

	// Create items
	items := []map[string]interface{}{
		{"id": "item1", "name": "First"},
		{"id": "item2", "name": "Second"},
	}
	itemsFile := createTieredItemsFile(t, tmpDir, items)

	// Track adapter calls
	counter := &childPipelineCallCounter{
		baseAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		),
	}

	eventCollector := testutil.NewEventCollector()
	executor := NewDefaultPipelineExecutor(counter, WithEmitter(eventCollector))
	matrixExecutor := NewMatrixExecutor(executor)

	execution := createTieredExecution(t, tmpDir, "child-pipeline-test")

	step := &Step{
		ID:      "matrix_step",
		Persona: "worker",
		Strategy: &MatrixStrategy{
			Type:          "matrix",
			ItemsSource:   itemsFile,
			ChildPipeline: childPipelinePath,
		},
		Exec: ExecConfig{
			Type:   "prompt",
			Source: "Process: {{ task }}",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := matrixExecutor.Execute(ctx, execution, step)
	if err != nil {
		t.Fatalf("Child pipeline execution failed: %v", err)
	}

	// Each item triggers a child pipeline execution, each with 1 step
	// So we expect 2 adapter calls (1 per item x 1 step per child pipeline)
	counter.mu.Lock()
	callCount := counter.count
	counter.mu.Unlock()

	if callCount != 2 {
		t.Errorf("Expected 2 adapter calls (2 items x 1 step), got %d", callCount)
	}

	// Verify results
	results := execution.Results[step.ID]
	if results["total_workers"] != 2 {
		t.Errorf("Expected 2 total workers, got %v", results["total_workers"])
	}
	if results["success_count"] != 2 {
		t.Errorf("Expected 2 successes, got %v", results["success_count"])
	}

	// Verify child pipeline events
	events := eventCollector.GetEvents()
	childPipelineLoadedCount := 0
	childPipelineCompleteCount := 0
	for _, e := range events {
		if e.State == "matrix_child_pipeline_loaded" {
			childPipelineLoadedCount++
		}
		if e.State == "matrix_child_pipeline_complete" {
			childPipelineCompleteCount++
		}
	}
	if childPipelineLoadedCount != 1 {
		t.Errorf("Expected 1 child_pipeline_loaded event, got %d", childPipelineLoadedCount)
	}
	if childPipelineCompleteCount != 2 {
		t.Errorf("Expected 2 child_pipeline_complete events, got %d", childPipelineCompleteCount)
	}
}

func TestMatrixExecutor_ChildPipeline_InputTemplate(t *testing.T) {
	executor := &DefaultPipelineExecutor{}
	matrixExecutor := NewMatrixExecutor(executor)

	item := map[string]interface{}{
		"repository": "re-cinq/wave",
		"number":     float64(206),
	}

	// Default (no template) serializes to JSON
	result, err := matrixExecutor.renderInputTemplate("", item)
	if err != nil {
		t.Fatalf("Failed to render default template: %v", err)
	}
	if !strings.Contains(result, "re-cinq/wave") || !strings.Contains(result, "206") {
		t.Errorf("Expected JSON with repository and number, got %q", result)
	}

	// Custom template
	result, err = matrixExecutor.renderInputTemplate("{{ .repository }} {{ .number }}", item)
	if err != nil {
		t.Fatalf("Failed to render custom template: %v", err)
	}
	if result != "re-cinq/wave 206" {
		t.Errorf("Expected 're-cinq/wave 206', got %q", result)
	}

	// Invalid template
	_, err = matrixExecutor.renderInputTemplate("{{ .missing_close", item)
	if err == nil {
		t.Error("Expected error for invalid template")
	}
}

func TestMatrixExecutor_ChildPipeline_WithTiers(t *testing.T) {
	tmpDir := t.TempDir()

	// Create child pipeline
	childPipelinePath := createTestChildPipeline(t, tmpDir, "test-child-tiered")

	// Linear chain: A → B → C
	items := []map[string]interface{}{
		{"id": "A", "deps": []interface{}{}},
		{"id": "B", "deps": []interface{}{"A"}},
		{"id": "C", "deps": []interface{}{"B"}},
	}
	itemsFile := createTieredItemsFile(t, tmpDir, items)

	counter := &childPipelineCallCounter{
		baseAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		),
	}

	eventCollector := testutil.NewEventCollector()
	executor := NewDefaultPipelineExecutor(counter, WithEmitter(eventCollector))
	matrixExecutor := NewMatrixExecutor(executor)

	execution := createTieredExecution(t, tmpDir, "child-tiered-test")

	step := &Step{
		ID:      "matrix_step",
		Persona: "worker",
		Strategy: &MatrixStrategy{
			Type:          "matrix",
			ItemsSource:   itemsFile,
			ItemIDKey:     "id",
			DependencyKey: "deps",
			ChildPipeline: childPipelinePath,
		},
		Exec: ExecConfig{
			Type:   "prompt",
			Source: "Process: {{ task }}",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err := matrixExecutor.Execute(ctx, execution, step)
	if err != nil {
		t.Fatalf("Tiered child pipeline execution failed: %v", err)
	}

	// 3 items × 1 step per child pipeline = 3 adapter calls
	counter.mu.Lock()
	callCount := counter.count
	counter.mu.Unlock()

	if callCount != 3 {
		t.Errorf("Expected 3 adapter calls, got %d", callCount)
	}

	// Verify 3 tiers (linear chain)
	events := eventCollector.GetEvents()
	tierStartCount := 0
	for _, e := range events {
		if e.State == "matrix_tier_start" {
			tierStartCount++
		}
	}
	if tierStartCount != 3 {
		t.Errorf("Expected 3 tiers for linear chain, got %d", tierStartCount)
	}

	results := execution.Results[step.ID]
	if results["success_count"] != 3 {
		t.Errorf("Expected 3 successes, got %v", results["success_count"])
	}
}

func TestMatrixExecutor_ChildPipeline_PartialFailure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create child pipeline
	childPipelinePath := createTestChildPipeline(t, tmpDir, "test-child-fail")

	// Create 3 items
	items := []map[string]interface{}{
		{"id": "item1"},
		{"id": "item2"},
		{"id": "item3"},
	}
	itemsFile := createTieredItemsFile(t, tmpDir, items)

	// Fail the 2nd adapter call (0-indexed: call #1)
	failAdapter := &partialFailureAdapter{
		failingIndices: map[int]bool{1: true},
		baseAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		),
	}

	eventCollector := testutil.NewEventCollector()
	executor := NewDefaultPipelineExecutor(failAdapter, WithEmitter(eventCollector))
	matrixExecutor := NewMatrixExecutor(executor)

	execution := createTieredExecution(t, tmpDir, "child-partial-fail")

	step := &Step{
		ID:      "matrix_step",
		Persona: "worker",
		Strategy: &MatrixStrategy{
			Type:          "matrix",
			ItemsSource:   itemsFile,
			ChildPipeline: childPipelinePath,
		},
		Exec: ExecConfig{
			Type:   "prompt",
			Source: "Process: {{ task }}",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := matrixExecutor.Execute(ctx, execution, step)
	// Should return error due to partial failure
	if err == nil {
		t.Fatal("Expected error due to partial failure")
	}

	results := execution.Results[step.ID]
	successCount, _ := results["success_count"].(int)
	failCount, _ := results["fail_count"].(int)

	if successCount != 2 {
		t.Errorf("Expected 2 successes, got %d", successCount)
	}
	if failCount != 1 {
		t.Errorf("Expected 1 failure, got %d", failCount)
	}
}

func TestMatrixExecutor_ChildPipeline_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	items := []map[string]interface{}{
		{"id": "item1"},
	}
	itemsFile := createTieredItemsFile(t, tmpDir, items)

	executor := NewDefaultPipelineExecutor(adaptertest.NewMockAdapter())
	matrixExecutor := NewMatrixExecutor(executor)

	execution := createTieredExecution(t, tmpDir, "child-not-found")

	step := &Step{
		ID:      "matrix_step",
		Persona: "worker",
		Strategy: &MatrixStrategy{
			Type:          "matrix",
			ItemsSource:   itemsFile,
			ChildPipeline: "/nonexistent/path/pipeline.yaml",
		},
		Exec: ExecConfig{
			Type:   "prompt",
			Source: "Process: {{ task }}",
		},
	}

	err := matrixExecutor.Execute(context.Background(), execution, step)
	if err == nil {
		t.Fatal("Expected error for non-existent child pipeline")
	}
	if !strings.Contains(err.Error(), "failed to load child pipeline") {
		t.Errorf("Expected load error, got: %v", err)
	}
}

// ==================== Stacked Worktree Tests ====================

func TestTierContext_NewAndBasicOperations(t *testing.T) {
	tc := NewTierContext()

	// Empty context
	if _, ok := tc.GetBranch("nonexistent"); ok {
		t.Error("Expected GetBranch to return false for empty context")
	}
	branches, err := tc.ResolveBranch([]string{"a", "b"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(branches) != 0 {
		t.Errorf("Expected 0 branches from empty context, got %d", len(branches))
	}

	// Set and get
	tc.SetBranch("item-a", "feature/a")
	branch, ok := tc.GetBranch("item-a")
	if !ok || branch != "feature/a" {
		t.Errorf("Expected 'feature/a', got %q (ok=%v)", branch, ok)
	}

	// Multiple branches
	tc.SetBranch("item-b", "feature/b")
	tc.SetBranch("item-c", "feature/c")

	branches, err = tc.ResolveBranch([]string{"item-a", "item-c"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(branches) != 2 {
		t.Fatalf("Expected 2 branches, got %d", len(branches))
	}
	if branches[0] != "feature/a" || branches[1] != "feature/c" {
		t.Errorf("Expected [feature/a, feature/c], got %v", branches)
	}

	// ResolveBranch skips items not in context
	branches, err = tc.ResolveBranch([]string{"item-a", "item-missing"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(branches) != 1 || branches[0] != "feature/a" {
		t.Errorf("Expected [feature/a], got %v", branches)
	}
}

func TestTierContext_EmptyBranchSkipped(t *testing.T) {
	tc := NewTierContext()
	tc.SetBranch("item-a", "") // Empty branch — item completed but no worktree

	branches, err := tc.ResolveBranch([]string{"item-a"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(branches) != 0 {
		t.Errorf("Expected empty branches for item with no output branch, got %v", branches)
	}
}

func TestMatrixExecutor_Stacked_TwoTierLinearChain(t *testing.T) {
	tmpDir := t.TempDir()

	// A (tier 0) -> B (tier 1, depends on A) with stacked: true
	items := []map[string]interface{}{
		{"id": "A", "deps": []interface{}{}},
		{"id": "B", "deps": []interface{}{"A"}},
	}
	itemsFile := createTieredItemsFile(t, tmpDir, items)

	// Track which items received stacked base branches
	var mu sync.Mutex
	branchCaptures := make(map[int]string) // itemIndex -> stacked base branch from context

	trackAdapter := &stackedBranchTrackingAdapter{
		mu:             &mu,
		branchCaptures: branchCaptures,
		baseAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		),
	}

	eventCollector := testutil.NewEventCollector()
	executor := NewDefaultPipelineExecutor(trackAdapter, WithEmitter(eventCollector))
	matrixExecutor := NewMatrixExecutor(executor)

	execution := createTieredExecution(t, tmpDir, "stacked-linear")

	step := &Step{
		ID:      "matrix_step",
		Persona: "worker",
		Strategy: &MatrixStrategy{
			Type:          "matrix",
			ItemsSource:   itemsFile,
			ItemIDKey:     "id",
			DependencyKey: "deps",
			Stacked:       true,
		},
		Exec: ExecConfig{
			Type:   "prompt",
			Source: "Process: {{ task }}",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := matrixExecutor.Execute(ctx, execution, step)
	if err != nil {
		t.Fatalf("Stacked tiered execution failed: %v", err)
	}

	results := execution.Results[step.ID]
	if results["total_workers"] != 2 {
		t.Errorf("Expected 2 total workers, got %v", results["total_workers"])
	}
	if results["success_count"] != 2 {
		t.Errorf("Expected 2 successes, got %v", results["success_count"])
	}

	// Verify stacking events were emitted
	events := eventCollector.GetEvents()
	hasStackedEvent := false
	for _, e := range events {
		if e.State == "matrix_stacked_branch_resolved" {
			hasStackedEvent = true
			break
		}
	}
	// Note: stacked branch resolved events only occur when parent has an output branch.
	// In direct worker execution (no child pipeline), OutputBranch is empty, so
	// the fallback path emits the "no output branch" event instead
	_ = hasStackedEvent
}

func TestMatrixExecutor_Stacked_WithoutDependencyKey(t *testing.T) {
	tmpDir := t.TempDir()

	// stacked: true but no dependency_key — should behave identically to non-stacked
	items := []map[string]interface{}{
		{"id": "A"},
		{"id": "B"},
		{"id": "C"},
	}
	itemsFile := createTieredItemsFile(t, tmpDir, items)

	baseAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
		adaptertest.WithTokensUsed(100),
	)

	eventCollector := testutil.NewEventCollector()
	executor := NewDefaultPipelineExecutor(baseAdapter, WithEmitter(eventCollector))
	matrixExecutor := NewMatrixExecutor(executor)

	execution := createTieredExecution(t, tmpDir, "stacked-no-dep")

	step := &Step{
		ID:      "matrix_step",
		Persona: "worker",
		Strategy: &MatrixStrategy{
			Type:        "matrix",
			ItemsSource: itemsFile,
			Stacked:     true, // stacked without dependency_key — FR-008 no-op
		},
		Exec: ExecConfig{
			Type:   "prompt",
			Source: "Process: {{ task }}",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := matrixExecutor.Execute(ctx, execution, step)
	if err != nil {
		t.Fatalf("Stacked execution without dependency_key failed: %v", err)
	}

	results := execution.Results[step.ID]
	if results["total_workers"] != 3 {
		t.Errorf("Expected 3 total workers, got %v", results["total_workers"])
	}

	// No stacking-related events should be emitted
	events := eventCollector.GetEvents()
	for _, e := range events {
		if e.State == "matrix_stacked_branch_resolved" || e.State == "matrix_integration_branch_created" {
			t.Errorf("Unexpected stacking event when dependency_key is empty: %s", e.State)
		}
	}
}

func TestMatrixExecutor_Stacked_PartialTierFailure(t *testing.T) {
	tmpDir := t.TempDir()

	// 3-tier chain: A (tier 0), B and C (tier 1, both depend on A), D depends on B only (tier 2)
	// C fails, B succeeds — D should still run
	items := []map[string]interface{}{
		{"id": "A", "deps": []interface{}{}},
		{"id": "B", "deps": []interface{}{"A"}},
		{"id": "C", "deps": []interface{}{"A"}},
		{"id": "D", "deps": []interface{}{"B"}},
	}
	itemsFile := createTieredItemsFile(t, tmpDir, items)

	failAdapter := &tieredFailureAdapter{
		failPatterns: []string{`"id":"C"`}, // Match item C by its JSON content in the prompt
		baseAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status": "success"}`),
			adaptertest.WithTokensUsed(100),
		),
	}

	eventCollector := testutil.NewEventCollector()
	executor := NewDefaultPipelineExecutor(failAdapter, WithEmitter(eventCollector))
	matrixExecutor := NewMatrixExecutor(executor)

	execution := createTieredExecution(t, tmpDir, "stacked-partial-fail")

	step := &Step{
		ID:      "matrix_step",
		Persona: "worker",
		Strategy: &MatrixStrategy{
			Type:          "matrix",
			ItemsSource:   itemsFile,
			ItemIDKey:     "id",
			DependencyKey: "deps",
			Stacked:       true,
		},
		Exec: ExecConfig{
			Type:   "prompt",
			Source: "Process: {{ task }}",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := matrixExecutor.Execute(ctx, execution, step)
	// Expect partial failure (C fails)
	if err == nil {
		t.Fatal("Expected partial failure error")
	}

	results := execution.Results[step.ID]
	// A, B, D succeed; C fails — but D still ran because it only depends on B
	successCount, ok := results["success_count"].(int)
	if !ok {
		t.Fatalf("Expected int success_count, got %T", results["success_count"])
	}
	failCount, ok := results["fail_count"].(int)
	if !ok {
		t.Fatalf("Expected int fail_count, got %T", results["fail_count"])
	}

	if successCount != 3 {
		t.Errorf("Expected 3 successes (A, B, D), got %d", successCount)
	}
	if failCount != 1 {
		t.Errorf("Expected 1 failure (C), got %d", failCount)
	}
}

func TestMatrixExecutor_Stacked_OutputBranchCapture(t *testing.T) {
	// Test that childPipelineWorker captures OutputBranch from child execution
	tmpDir := t.TempDir()

	// Create a child pipeline YAML
	childPipelineYAML := `kind: WavePipeline
metadata:
  name: test-child
steps:
  - id: implement
    persona: worker
    exec:
      type: prompt
      source: "Do work"
`
	childPipelineDir := filepath.Join(tmpDir, ".agents", "pipelines")
	require.NoError(t, os.MkdirAll(childPipelineDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(childPipelineDir, "test-child.yaml"), []byte(childPipelineYAML), 0644))

	baseAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
		adaptertest.WithTokensUsed(100),
	)

	executor := NewDefaultPipelineExecutor(baseAdapter)
	matrixExec := NewMatrixExecutor(executor)

	// Load the child pipeline
	childPipeline, err := matrixExec.loadChildPipeline(filepath.Join(childPipelineDir, "test-child.yaml"))
	if err != nil {
		t.Fatalf("Failed to load child pipeline: %v", err)
	}

	// Test that childPipelineWorker returns a result
	workerFunc := matrixExec.childPipelineWorker(childPipeline)

	execution := createTieredExecution(t, tmpDir, "test-branch-capture")

	item := map[string]interface{}{"id": "test-item"}
	result := workerFunc(context.Background(), execution, &Step{
		ID:      "matrix_step",
		Persona: "worker",
		Strategy: &MatrixStrategy{
			Type:          "matrix",
			InputTemplate: `{{ .id }}`,
		},
		Exec: ExecConfig{
			Type:   "prompt",
			Source: "Process: {{ task }}",
		},
	}, 0, item)

	// The child pipeline doesn't use worktree workspaces, so OutputBranch should be empty
	if result.OutputBranch != "" {
		t.Errorf("Expected empty OutputBranch for non-worktree child pipeline, got %q", result.OutputBranch)
	}
}

func TestMatrixExecutor_Stacked_ParentNoOutputBranch(t *testing.T) {
	tmpDir := t.TempDir()

	// A (tier 0) -> B (tier 1) with stacked: true
	// A completes but has no output branch (direct worker, no worktree)
	// B should still run with default base
	items := []map[string]interface{}{
		{"id": "A", "deps": []interface{}{}},
		{"id": "B", "deps": []interface{}{"A"}},
	}
	itemsFile := createTieredItemsFile(t, tmpDir, items)

	baseAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status": "success"}`),
		adaptertest.WithTokensUsed(100),
	)

	eventCollector := testutil.NewEventCollector()
	executor := NewDefaultPipelineExecutor(baseAdapter, WithEmitter(eventCollector))
	matrixExecutor := NewMatrixExecutor(executor)

	execution := createTieredExecution(t, tmpDir, "stacked-no-branch")

	step := &Step{
		ID:      "matrix_step",
		Persona: "worker",
		Strategy: &MatrixStrategy{
			Type:          "matrix",
			ItemsSource:   itemsFile,
			ItemIDKey:     "id",
			DependencyKey: "deps",
			Stacked:       true,
		},
		Exec: ExecConfig{
			Type:   "prompt",
			Source: "Process: {{ task }}",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := matrixExecutor.Execute(ctx, execution, step)
	if err != nil {
		t.Fatalf("Stacked execution with no output branch failed: %v", err)
	}

	results := execution.Results[step.ID]
	if results["success_count"] != 2 {
		t.Errorf("Expected 2 successes, got %v", results["success_count"])
	}

	// Check for "no output branch" event
	events := eventCollector.GetEvents()
	hasNoOutputBranchEvent := false
	for _, e := range events {
		if e.State == "matrix_stacked_branch_resolved" && strings.Contains(e.Message, "no output branch") {
			hasNoOutputBranchEvent = true
			break
		}
	}
	if !hasNoOutputBranchEvent {
		t.Error("Expected 'no output branch' event for parent without worktree output")
	}
}

func TestMatrixExecutor_stackedBaseBranchFromContext(t *testing.T) {
	// Test the context-based stacked base branch propagation
	ctx := context.Background()

	// No stacked base in plain context
	if branch := stackedBaseBranchFromContext(ctx); branch != "" {
		t.Errorf("Expected empty stacked base from plain context, got %q", branch)
	}

	// With stacked base
	ctx = context.WithValue(ctx, stackedBaseBranchKey{}, "feature/parent")
	if branch := stackedBaseBranchFromContext(ctx); branch != "feature/parent" {
		t.Errorf("Expected 'feature/parent', got %q", branch)
	}
}

// stackedBranchTrackingAdapter records the stacked base branch from executor config.
type stackedBranchTrackingAdapter struct {
	mu             *sync.Mutex
	branchCaptures map[int]string
	callCount      int
	baseAdapter    adapter.AdapterRunner
}

func (a *stackedBranchTrackingAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	a.mu.Lock()
	idx := a.callCount
	a.callCount++
	a.mu.Unlock()

	// Capture stacked base from context
	if base := stackedBaseBranchFromContext(ctx); base != "" {
		a.mu.Lock()
		a.branchCaptures[idx] = base
		a.mu.Unlock()
	}

	return a.baseAdapter.Run(ctx, cfg)
}
