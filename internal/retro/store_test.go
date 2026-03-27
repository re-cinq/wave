package retro

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/testutil"
)

// createTestRun creates a pipeline_run record in SQLite so the FK constraint
// on the retrospective table is satisfied. Returns the auto-generated run ID.
func createTestRun(t *testing.T, ss state.StateStore, pipelineName string) string {
	t.Helper()
	runID, err := ss.CreateRun(pipelineName, "test input")
	if err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}
	return runID
}

// helper to build a sample Retrospective for tests.
func sampleRetro(runID, pipeline string) *Retrospective {
	return &Retrospective{
		RunID:    runID,
		Pipeline: pipeline,
		Timestamp: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC),
		Quantitative: QuantitativeData{
			TotalDurationMs: 5000,
			TotalSteps:      3,
			SuccessCount:    2,
			FailureCount:    1,
			TotalRetries:    1,
			Steps: []StepMetrics{
				{Name: "plan", DurationMs: 1000, Status: "completed", Adapter: "claude"},
				{Name: "implement", DurationMs: 3000, Status: "completed", Adapter: "claude"},
				{Name: "test", DurationMs: 1000, Status: "failed", Retries: 1, Adapter: "claude"},
			},
		},
		Narrative: &NarrativeData{
			Smoothness: SmoothnessSmooth,
			Intent:     "implement feature X",
			Outcome:    "feature implemented with minor test issues",
			FrictionPoints: []FrictionPoint{
				{Type: FrictionRetry, Step: "test", Detail: "test flaked once"},
			},
			Learnings: []Learning{
				{Category: LearningWorkflow, Detail: "retry policy handled flake"},
			},
			OpenItems: []OpenItem{
				{Type: OpenItemTestGap, Detail: "add integration test"},
			},
		},
	}
}

func TestFileStore_SaveAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "retros")
	mock := testutil.NewMockStateStore()
	store := NewFileStore(baseDir, mock)

	retro := sampleRetro("run-001", "impl-issue")

	// Save
	if err := store.Save(retro); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file was written
	filePath := filepath.Join(baseDir, "run-001.json")
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("expected file at %s: %v", filePath, err)
	}

	// Get via file
	got, err := store.Get("run-001")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil")
	}
	if got.RunID != "run-001" {
		t.Errorf("RunID = %q, want %q", got.RunID, "run-001")
	}
	if got.Pipeline != "impl-issue" {
		t.Errorf("Pipeline = %q, want %q", got.Pipeline, "impl-issue")
	}
	if got.Quantitative.TotalSteps != 3 {
		t.Errorf("TotalSteps = %d, want 3", got.Quantitative.TotalSteps)
	}
	if got.Narrative == nil {
		t.Fatal("Narrative is nil")
	}
	if got.Narrative.Smoothness != SmoothnessSmooth {
		t.Errorf("Smoothness = %q, want %q", got.Narrative.Smoothness, SmoothnessSmooth)
	}
	if len(got.Narrative.FrictionPoints) != 1 {
		t.Errorf("FrictionPoints count = %d, want 1", len(got.Narrative.FrictionPoints))
	}
}

func TestFileStore_SavePersistsToStateStore(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "retros")

	dbPath := filepath.Join(tmpDir, "state.db")
	ss, err := state.NewStateStore(dbPath)
	if err != nil {
		t.Fatalf("NewStateStore failed: %v", err)
	}
	defer ss.Close()

	// Create a pipeline_run record so the FK constraint is satisfied.
	runID := createTestRun(t, ss, "impl-issue")

	store := NewFileStore(baseDir, ss)
	retro := sampleRetro(runID, "impl-issue")

	if err := store.Save(retro); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify the record was persisted in SQLite.
	savedRecord, err := ss.GetRetrospective(runID)
	if err != nil {
		t.Fatalf("GetRetrospective from SQLite failed: %v", err)
	}
	if savedRecord == nil {
		t.Fatal("expected record in SQLite, got nil")
	}
	if savedRecord.PipelineName != "impl-issue" {
		t.Errorf("PipelineName = %q, want %q", savedRecord.PipelineName, "impl-issue")
	}
	if savedRecord.Smoothness != SmoothnessSmooth {
		t.Errorf("Smoothness = %q, want %q", savedRecord.Smoothness, SmoothnessSmooth)
	}

	// Verify quantitative JSON is valid.
	var quant QuantitativeData
	if err := json.Unmarshal([]byte(savedRecord.QuantitativeJSON), &quant); err != nil {
		t.Fatalf("failed to parse QuantitativeJSON: %v", err)
	}
	if quant.TotalSteps != 3 {
		t.Errorf("QuantitativeJSON TotalSteps = %d, want 3", quant.TotalSteps)
	}
}

func TestFileStore_GetNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "retros")
	mock := testutil.NewMockStateStore()
	store := NewFileStore(baseDir, mock)

	got, err := store.Get("nonexistent-run")
	if err != nil {
		t.Fatalf("Get returned unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for nonexistent run, got %+v", got)
	}
}

func TestFileStore_GetFallsBackToSQLite(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "retros")

	dbPath := filepath.Join(tmpDir, "state.db")
	ss, err := state.NewStateStore(dbPath)
	if err != nil {
		t.Fatalf("NewStateStore failed: %v", err)
	}
	defer ss.Close()

	runID := createTestRun(t, ss, "audit-dx")

	store := NewFileStore(baseDir, ss)
	retro := sampleRetro(runID, "audit-dx")

	// Save (creates both file and SQLite record).
	if err := store.Save(retro); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Remove the file so Get must fall back to SQLite.
	filePath := filepath.Join(baseDir, runID+".json")
	if err := os.Remove(filePath); err != nil {
		t.Fatalf("failed to remove file: %v", err)
	}

	got, err := store.Get(runID)
	if err != nil {
		t.Fatalf("Get (fallback) failed: %v", err)
	}
	if got == nil {
		t.Fatal("Get (fallback) returned nil")
	}
	if got.Pipeline != "audit-dx" {
		t.Errorf("Pipeline = %q, want %q", got.Pipeline, "audit-dx")
	}
	if got.Quantitative.TotalSteps != 3 {
		t.Errorf("TotalSteps = %d, want 3", got.Quantitative.TotalSteps)
	}
}

func TestFileStore_ListWithFilters(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "retros")

	dbPath := filepath.Join(tmpDir, "state.db")
	ss, err := state.NewStateStore(dbPath)
	if err != nil {
		t.Fatalf("NewStateStore failed: %v", err)
	}
	defer ss.Close()

	store := NewFileStore(baseDir, ss)

	// Create pipeline_run records for FK satisfaction.
	runA := createTestRun(t, ss, "impl-issue")
	runB := createTestRun(t, ss, "audit-dx")
	runC := createTestRun(t, ss, "impl-issue")

	// Save multiple retros with different pipelines and timestamps.
	r1 := sampleRetro(runA, "impl-issue")
	r1.Timestamp = time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC)
	r2 := sampleRetro(runB, "audit-dx")
	r2.Timestamp = time.Date(2026, 3, 26, 10, 0, 0, 0, time.UTC)
	r3 := sampleRetro(runC, "impl-issue")
	r3.Timestamp = time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)

	for _, r := range []*Retrospective{r1, r2, r3} {
		if err := store.Save(r); err != nil {
			t.Fatalf("Save %s failed: %v", r.RunID, err)
		}
	}

	// List all
	all, err := store.List(ListOptions{})
	if err != nil {
		t.Fatalf("List all failed: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("List all returned %d, want 3", len(all))
	}

	// List by pipeline
	implOnly, err := store.List(ListOptions{PipelineName: "impl-issue"})
	if err != nil {
		t.Fatalf("List by pipeline failed: %v", err)
	}
	if len(implOnly) != 2 {
		t.Errorf("List by pipeline returned %d, want 2", len(implOnly))
	}

	// List with since filter
	since := time.Date(2026, 3, 26, 0, 0, 0, 0, time.UTC)
	recent, err := store.List(ListOptions{Since: since})
	if err != nil {
		t.Fatalf("List with since failed: %v", err)
	}
	if len(recent) != 2 {
		t.Errorf("List with since returned %d, want 2", len(recent))
	}

	// List with limit
	limited, err := store.List(ListOptions{Limit: 1})
	if err != nil {
		t.Fatalf("List with limit failed: %v", err)
	}
	if len(limited) != 1 {
		t.Errorf("List with limit returned %d, want 1", len(limited))
	}
}

func TestFileStore_UpdateNarrative(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "retros")

	dbPath := filepath.Join(tmpDir, "state.db")
	ss, err := state.NewStateStore(dbPath)
	if err != nil {
		t.Fatalf("NewStateStore failed: %v", err)
	}
	defer ss.Close()

	runID := createTestRun(t, ss, "impl-issue")

	store := NewFileStore(baseDir, ss)

	// Save a retro without narrative.
	retro := sampleRetro(runID, "impl-issue")
	retro.Narrative = nil

	if err := store.Save(retro); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify no narrative initially.
	got, err := store.Get(runID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Narrative != nil {
		t.Fatalf("expected nil narrative initially, got %+v", got.Narrative)
	}

	// Update the narrative.
	newNarrative := &NarrativeData{
		Smoothness: SmoothnessBumpy,
		Intent:     "fix bug Y",
		Outcome:    "bug fixed after retry",
		FrictionPoints: []FrictionPoint{
			{Type: FrictionWrongApproach, Step: "implement", Detail: "wrong fix first"},
		},
	}
	if err := store.UpdateNarrative(runID, newNarrative); err != nil {
		t.Fatalf("UpdateNarrative failed: %v", err)
	}

	// Verify via Get (from file).
	got, err = store.Get(runID)
	if err != nil {
		t.Fatalf("Get after update failed: %v", err)
	}
	if got.Narrative == nil {
		t.Fatal("Narrative is nil after update")
	}
	if got.Narrative.Smoothness != SmoothnessBumpy {
		t.Errorf("Smoothness = %q, want %q", got.Narrative.Smoothness, SmoothnessBumpy)
	}
	if got.Narrative.Intent != "fix bug Y" {
		t.Errorf("Intent = %q, want %q", got.Narrative.Intent, "fix bug Y")
	}
	if len(got.Narrative.FrictionPoints) != 1 {
		t.Errorf("FrictionPoints count = %d, want 1", len(got.Narrative.FrictionPoints))
	}

	// Verify the SQLite record was also updated.
	record, err := ss.GetRetrospective(runID)
	if err != nil {
		t.Fatalf("GetRetrospective from SQLite failed: %v", err)
	}
	if record.Smoothness != SmoothnessBumpy {
		t.Errorf("SQLite Smoothness = %q, want %q", record.Smoothness, SmoothnessBumpy)
	}
}

func TestFileStore_UpdateNarrativeNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "retros")
	mock := testutil.NewMockStateStore()
	store := NewFileStore(baseDir, mock)

	err := store.UpdateNarrative("no-such-run", &NarrativeData{Smoothness: SmoothnessSmooth})
	if err == nil {
		t.Fatal("expected error for non-existent run, got nil")
	}
}

func TestFileStore_SaveNilRetro(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "retros")
	mock := testutil.NewMockStateStore()
	store := NewFileStore(baseDir, mock)

	if err := store.Save(nil); err == nil {
		t.Fatal("expected error for nil retro, got nil")
	}
}

func TestFileStore_SaveEmptyRunID(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "retros")
	mock := testutil.NewMockStateStore()
	store := NewFileStore(baseDir, mock)

	retro := sampleRetro("", "impl-issue")
	if err := store.Save(retro); err == nil {
		t.Fatal("expected error for empty run_id, got nil")
	}
}

func TestFileStore_UpdateNarrativeNilNarrative(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "retros")
	mock := testutil.NewMockStateStore()
	store := NewFileStore(baseDir, mock)

	if err := store.UpdateNarrative("run-001", nil); err == nil {
		t.Fatal("expected error for nil narrative, got nil")
	}
}

func TestFileStore_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "retros")
	mock := testutil.NewMockStateStore()
	store := NewFileStore(baseDir, mock)

	// Attempts to traverse the filesystem via crafted run IDs should be rejected.
	maliciousIDs := []string{
		"../../etc/passwd",
		"../secrets",
		"run/../../../config",
		"run/../../config",
		".hidden",
		"run id with spaces",
		"run;injection",
		"",
	}

	for _, id := range maliciousIDs {
		t.Run("Get_"+id, func(t *testing.T) {
			_, err := store.Get(id)
			if err == nil {
				t.Errorf("expected error for malicious run ID %q, got nil", id)
			}
		})

		t.Run("Save_"+id, func(t *testing.T) {
			retro := &Retrospective{
				RunID:    id,
				Pipeline: "test",
				Quantitative: QuantitativeData{
					TotalSteps: 1,
				},
				Timestamp: time.Now(),
			}
			err := store.Save(retro)
			if err == nil {
				t.Errorf("expected error for malicious run ID %q, got nil", id)
			}
		})

		t.Run("UpdateNarrative_"+id, func(t *testing.T) {
			err := store.UpdateNarrative(id, &NarrativeData{Smoothness: SmoothnessSmooth})
			if err == nil {
				t.Errorf("expected error for malicious run ID %q, got nil", id)
			}
		})
	}

	// Valid run IDs should be accepted by the validation function.
	validIDs := []string{
		"run-001",
		"abc123",
		"run_with_underscores",
		"Run-Mixed-Case-123",
	}
	for _, id := range validIDs {
		if err := validateRunID(id); err != nil {
			t.Errorf("valid run ID %q rejected: %v", id, err)
		}
	}
}
