package commands

import (
	"sync"
	"testing"

	"github.com/recinq/wave/internal/testutil"
)

// TestRunID_ReusedWhenRunIDAndFromStepBothSet verifies that resolveRunID never
// calls store.CreateRun when a pre-created run ID is supplied (--run flag).
//
// This is the unit-level guard for the fix in issue #700: before the fix the
// runImpl condition was `opts.RunID != "" && opts.FromStep == ""`, which silently
// ignored the pre-created ID whenever --from-step was also set, causing a second
// CreateRun call and a phantom run record in the dashboard.
func TestRunID_ReusedWhenRunIDAndFromStepBothSet(t *testing.T) {
	var mu sync.Mutex
	createCount := 0

	store := testutil.NewMockStateStore(
		testutil.WithCreateRun(func(pipelineName, input string) (string, error) {
			mu.Lock()
			createCount++
			mu.Unlock()
			return "should-not-be-called", nil
		}),
	)

	preCreatedID := "detach-pre-created-id"

	// Simulate the --detach subprocess path: RunID is set AND FromStep is set.
	// resolveRunID must return the pre-created ID without touching the store.
	id, err := resolveRunID(preCreatedID, store, "my-pipeline", "test input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != preCreatedID {
		t.Errorf("expected pre-created run ID %q to be reused, got %q", preCreatedID, id)
	}

	mu.Lock()
	count := createCount
	mu.Unlock()

	if count != 0 {
		t.Errorf("expected 0 CreateRun calls when RunID is pre-set, got %d", count)
	}
}

// TestRunID_CreatesRunWhenNoPreCreatedID verifies that resolveRunID calls
// store.CreateRun exactly once when no pre-created ID is supplied.
func TestRunID_CreatesRunWhenNoPreCreatedID(t *testing.T) {
	var mu sync.Mutex
	createCount := 0

	store := testutil.NewMockStateStore(
		testutil.WithCreateRun(func(pipelineName, input string) (string, error) {
			mu.Lock()
			createCount++
			mu.Unlock()
			return "store-generated-id", nil
		}),
	)

	id, err := resolveRunID("", store, "my-pipeline", "test input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "store-generated-id" {
		t.Errorf("expected store-generated ID, got %q", id)
	}

	mu.Lock()
	count := createCount
	mu.Unlock()

	if count != 1 {
		t.Errorf("expected exactly 1 CreateRun call when no pre-created ID, got %d", count)
	}
}

// TestRunID_ReturnsEmptyWhenNoStoreAndNoPreCreatedID verifies that resolveRunID
// returns ("", nil) when there is no store and no pre-created ID.
// The caller is responsible for falling back to GenerateRunID in this case.
func TestRunID_ReturnsEmptyWhenNoStoreAndNoPreCreatedID(t *testing.T) {
	id, err := resolveRunID("", nil, "my-pipeline", "test input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "" {
		t.Errorf("expected empty ID when no store and no pre-created ID, got %q", id)
	}
}
