package pipeline

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/recinq/wave/internal/manifest"
)

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
		Pipeline: &Pipeline{Metadata: PipelineMetadata{Name: "test"}},
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
		ArtifactPaths:  make(map[string]string),
		Results:        make(map[string]map[string]interface{}),
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
