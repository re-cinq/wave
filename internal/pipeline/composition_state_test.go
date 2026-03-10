package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIterationState_SaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	pipelineID := "test-pipeline"
	stepID := "iterate-step"

	state := &IterationState{
		StepID:         stepID,
		TotalItems:     5,
		CompletedItems: 3,
		Items: []IterationItemState{
			{Index: 0, Status: "completed", PipelineRunID: "run-1"},
			{Index: 1, Status: "completed", PipelineRunID: "run-2"},
			{Index: 2, Status: "completed", PipelineRunID: "run-3"},
			{Index: 3, Status: "pending"},
			{Index: 4, Status: "pending"},
		},
	}

	if err := SaveIterationState(tmpDir, pipelineID, stepID, state); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	loaded, err := LoadIterationState(tmpDir, pipelineID, stepID)
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	if loaded.TotalItems != 5 {
		t.Errorf("expected 5 total items, got %d", loaded.TotalItems)
	}
	if loaded.CompletedItems != 3 {
		t.Errorf("expected 3 completed, got %d", loaded.CompletedItems)
	}
	if len(loaded.Items) != 5 {
		t.Errorf("expected 5 items, got %d", len(loaded.Items))
	}
	if loaded.Items[0].Status != "completed" {
		t.Errorf("expected item 0 completed, got %q", loaded.Items[0].Status)
	}
	if loaded.Items[3].Status != "pending" {
		t.Errorf("expected item 3 pending, got %q", loaded.Items[3].Status)
	}
}

func TestIterationState_LoadNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := LoadIterationState(tmpDir, "nonexistent", "step")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestGateState_SaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	pipelineID := "test-pipeline"
	stepID := "gate-step"

	state := &GateState{
		StepID:     stepID,
		GateType:   "approval",
		Status:     "resolved",
		ResolvedBy: "user",
	}

	if err := SaveGateState(tmpDir, pipelineID, stepID, state); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	loaded, err := LoadGateState(tmpDir, pipelineID, stepID)
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	if loaded.GateType != "approval" {
		t.Errorf("expected approval, got %q", loaded.GateType)
	}
	if loaded.Status != "resolved" {
		t.Errorf("expected resolved, got %q", loaded.Status)
	}
	if loaded.ResolvedBy != "user" {
		t.Errorf("expected user, got %q", loaded.ResolvedBy)
	}
}

func TestGateState_LoadNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := LoadGateState(tmpDir, "nonexistent", "step")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestIterationState_StatePath(t *testing.T) {
	tmpDir := t.TempDir()
	state := &IterationState{
		StepID:     "test",
		TotalItems: 1,
		Items:      []IterationItemState{{Index: 0, Status: "pending"}},
	}

	if err := SaveIterationState(tmpDir, "my-pipeline", "test", state); err != nil {
		t.Fatal(err)
	}

	// Verify the file was created at the expected path
	expectedPath := filepath.Join(tmpDir, "my-pipeline", ".wave", "composition", "test-iteration.json")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Errorf("expected file at %s: %v", expectedPath, err)
	}
}
