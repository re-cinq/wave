package contract

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRollbackManager_CreateCheckpoint(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rollback-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewRollbackManager(tmpDir)

	artifacts := map[string]string{
		"analysis": "/path/to/analysis.json",
		"result":   "/path/to/result.json",
	}

	checkpoint, err := manager.CreateCheckpoint("test-pipeline", "step-1", "/workspace", artifacts)
	if err != nil {
		t.Fatalf("CreateCheckpoint failed: %v", err)
	}

	if checkpoint.PipelineID != "test-pipeline" {
		t.Errorf("PipelineID = %q, want %q", checkpoint.PipelineID, "test-pipeline")
	}
	if checkpoint.StepID != "step-1" {
		t.Errorf("StepID = %q, want %q", checkpoint.StepID, "step-1")
	}
	if !checkpoint.CanRollback {
		t.Error("CanRollback should be true")
	}

	// Verify checkpoint was saved
	loaded, err := manager.LoadCheckpoint("test-pipeline", "step-1")
	if err != nil {
		t.Fatalf("LoadCheckpoint failed: %v", err)
	}

	if loaded.PipelineID != checkpoint.PipelineID {
		t.Errorf("loaded PipelineID = %q, want %q", loaded.PipelineID, checkpoint.PipelineID)
	}
}

func TestRollbackManager_RollbackLog(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rollback-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewRollbackManager(tmpDir)

	// Initialize log
	log, err := manager.InitRollbackLog("test-pipeline")
	if err != nil {
		t.Fatalf("InitRollbackLog failed: %v", err)
	}

	if len(log.Operations) != 0 {
		t.Errorf("new log should have 0 operations, got %d", len(log.Operations))
	}

	// Log operations
	op1 := RollbackOperation{
		Type:      "file_created",
		Target:    "/path/to/file1.go",
		CanRevert: true,
	}
	if err := manager.LogOperation("test-pipeline", op1); err != nil {
		t.Fatalf("LogOperation failed: %v", err)
	}

	op2 := RollbackOperation{
		Type:      "file_modified",
		Target:    "/path/to/file2.go",
		Backup:    "/path/to/backup/file2.go",
		CanRevert: true,
	}
	if err := manager.LogOperation("test-pipeline", op2); err != nil {
		t.Fatalf("LogOperation failed: %v", err)
	}

	// Load log and verify
	loadedLog, err := manager.loadRollbackLog("test-pipeline")
	if err != nil {
		t.Fatalf("loadRollbackLog failed: %v", err)
	}

	if len(loadedLog.Operations) != 2 {
		t.Errorf("expected 2 operations, got %d", len(loadedLog.Operations))
	}

	if loadedLog.Operations[0].Type != "file_created" {
		t.Errorf("first operation type = %q, want %q", loadedLog.Operations[0].Type, "file_created")
	}
	if loadedLog.Operations[1].Type != "file_modified" {
		t.Errorf("second operation type = %q, want %q", loadedLog.Operations[1].Type, "file_modified")
	}
}

func TestRollbackManager_GetRollbackPlan(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rollback-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewRollbackManager(tmpDir)

	// Create log with operations
	_, _ = manager.InitRollbackLog("test-pipeline")

	manager.LogOperation("test-pipeline", RollbackOperation{
		Type:      "file_created",
		Target:    "/path/to/new_file.go",
		CanRevert: true,
	})

	manager.LogOperation("test-pipeline", RollbackOperation{
		Type:        "git_commit",
		Target:      "abc123",
		CanRevert:   false,
		RevertSteps: []string{"git revert abc123", "git push"},
	})

	// Get rollback plan
	plan, err := manager.GetRollbackPlan("test-pipeline")
	if err != nil {
		t.Fatalf("GetRollbackPlan failed: %v", err)
	}

	// Verify plan contains expected information
	if !contains(plan, "test-pipeline") {
		t.Error("plan should contain pipeline ID")
	}
	if !contains(plan, "file_created") {
		t.Error("plan should contain operation types")
	}
	if !contains(plan, "git_commit") {
		t.Error("plan should contain git operation")
	}
	if !contains(plan, "manual intervention") {
		t.Error("plan should indicate manual steps required")
	}
}

func TestRollbackManager_Rollback(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rollback-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewRollbackManager(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test_file.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Initialize log and record file creation
	manager.InitRollbackLog("test-pipeline")
	manager.LogOperation("test-pipeline", RollbackOperation{
		Type:      "file_created",
		Target:    testFile,
		CanRevert: true,
	})

	// Perform rollback
	if err := manager.Rollback("test-pipeline", nil); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Verify file was deleted
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("file should have been deleted by rollback")
	}
}

func TestRollbackManager_CreateBackup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rollback-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewRollbackManager(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test_file.txt")
	originalContent := []byte("original content")
	if err := os.WriteFile(testFile, originalContent, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create backup
	backupPath, err := manager.CreateBackup("test-pipeline", testFile)
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	// Verify backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Errorf("backup file does not exist at %s", backupPath)
	}

	// Verify backup content matches original
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("failed to read backup: %v", err)
	}

	if string(backupContent) != string(originalContent) {
		t.Errorf("backup content = %q, want %q", backupContent, originalContent)
	}
}

func TestRollbackManager_CheckpointWithRollback(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rollback-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewRollbackManager(tmpDir)

	// Create checkpoint at step 1
	checkpoint1, err := manager.CreateCheckpoint("test-pipeline", "step-1", "/workspace", nil)
	if err != nil {
		t.Fatalf("CreateCheckpoint failed: %v", err)
	}

	// Initialize log
	manager.InitRollbackLog("test-pipeline")

	// Create test file after checkpoint
	time.Sleep(10 * time.Millisecond) // Ensure timestamp is after checkpoint
	testFile := filepath.Join(tmpDir, "test_file.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Log the operation
	manager.LogOperation("test-pipeline", RollbackOperation{
		Type:      "file_created",
		Target:    testFile,
		Timestamp: time.Now(),
		CanRevert: true,
	})

	// Rollback to checkpoint 1
	if err := manager.Rollback("test-pipeline", checkpoint1); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Verify file was deleted
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("file should have been deleted by rollback to checkpoint")
	}
}

func TestRollbackManager_CleanupCheckpoints(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rollback-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewRollbackManager(tmpDir)

	// Create multiple checkpoints
	manager.CreateCheckpoint("test-pipeline", "step-1", "/workspace", nil)
	manager.CreateCheckpoint("test-pipeline", "step-2", "/workspace", nil)
	manager.InitRollbackLog("test-pipeline")

	// Verify checkpoints exist
	pipelineDir := filepath.Join(tmpDir, "test-pipeline")
	if _, err := os.Stat(pipelineDir); os.IsNotExist(err) {
		t.Fatal("pipeline directory should exist")
	}

	// Cleanup
	if err := manager.CleanupCheckpoints("test-pipeline"); err != nil {
		t.Fatalf("CleanupCheckpoints failed: %v", err)
	}

	// Verify cleanup
	if _, err := os.Stat(pipelineDir); !os.IsNotExist(err) {
		t.Error("pipeline directory should have been removed")
	}
}
