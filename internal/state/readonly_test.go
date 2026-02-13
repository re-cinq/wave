package state

import (
	"path/filepath"
	"testing"
)

func TestNewReadOnlyStateStore(t *testing.T) {
	// Create a temp dir for the test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// First create a normal store to initialize the database
	store, err := NewStateStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create state store: %v", err)
	}

	// Insert test data
	runID, err := store.CreateRun("test-pipeline", "test input")
	if err != nil {
		t.Fatalf("failed to create run: %v", err)
	}
	store.Close()

	// Now open as read-only
	roStore, err := NewReadOnlyStateStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create read-only store: %v", err)
	}
	defer roStore.Close()

	// Read should work
	run, err := roStore.GetRun(runID)
	if err != nil {
		t.Fatalf("failed to get run from read-only store: %v", err)
	}
	if run.PipelineName != "test-pipeline" {
		t.Errorf("expected pipeline name 'test-pipeline', got '%s'", run.PipelineName)
	}

	// ListRuns should work
	runs, err := roStore.ListRuns(ListRunsOptions{Limit: 10})
	if err != nil {
		t.Fatalf("failed to list runs from read-only store: %v", err)
	}
	if len(runs) != 1 {
		t.Errorf("expected 1 run, got %d", len(runs))
	}
}

func TestNewReadOnlyStateStore_NonExistentDB(t *testing.T) {
	// Use a path where the parent directory does not exist, which guarantees
	// the SQLite driver cannot create or open the file.
	dbPath := filepath.Join(t.TempDir(), "nodir", "subdir", "nonexistent.db")

	_, err := NewReadOnlyStateStore(dbPath)
	if err == nil {
		t.Error("expected error for non-existent database path")
	}
}

func TestNewReadOnlyStateStore_ConcurrentReads(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create and populate database
	store, err := NewStateStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create state store: %v", err)
	}
	for i := 0; i < 10; i++ {
		store.CreateRun("pipeline", "input")
	}
	store.Close()

	// Open as read-only
	roStore, err := NewReadOnlyStateStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create read-only store: %v", err)
	}
	defer roStore.Close()

	// Concurrent reads should not block
	done := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := roStore.ListRuns(ListRunsOptions{Limit: 10})
			done <- err
		}()
	}

	for i := 0; i < 10; i++ {
		if err := <-done; err != nil {
			t.Errorf("concurrent read failed: %v", err)
		}
	}
}
