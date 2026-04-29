package retro

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/recinq/wave/internal/metrics"
)

// mockRetroIndexer implements RetroIndexer for testing.
type mockRetroIndexer struct {
	records      map[string]*metrics.RetrospectiveRecord
	lastSaved    *metrics.RetrospectiveRecord
	smoothUpdate string
	statusUpdate string
}

func newMockIndexer() *mockRetroIndexer {
	return &mockRetroIndexer{
		records: make(map[string]*metrics.RetrospectiveRecord),
	}
}

func (m *mockRetroIndexer) SaveRetrospective(record *metrics.RetrospectiveRecord) error {
	m.lastSaved = record
	m.records[record.RunID] = record
	return nil
}

func (m *mockRetroIndexer) GetRetrospective(runID string) (*metrics.RetrospectiveRecord, error) {
	r, ok := m.records[runID]
	if !ok {
		return nil, os.ErrNotExist
	}
	return r, nil
}

func (m *mockRetroIndexer) ListRetrospectives(opts metrics.ListRetrosOptions) ([]metrics.RetrospectiveRecord, error) {
	var result []metrics.RetrospectiveRecord
	for _, r := range m.records {
		if opts.PipelineName != "" && r.PipelineName != opts.PipelineName {
			continue
		}
		result = append(result, *r)
	}
	return result, nil
}

func (m *mockRetroIndexer) DeleteRetrospective(runID string) error {
	delete(m.records, runID)
	return nil
}

func (m *mockRetroIndexer) UpdateRetrospectiveSmoothness(runID string, smoothness string) error {
	m.smoothUpdate = smoothness
	if r, ok := m.records[runID]; ok {
		r.Smoothness = smoothness
	}
	return nil
}

func (m *mockRetroIndexer) UpdateRetrospectiveStatus(runID string, status string) error {
	m.statusUpdate = status
	if r, ok := m.records[runID]; ok {
		r.Status = status
	}
	return nil
}

func TestStorage_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	indexer := newMockIndexer()
	s := NewStorage(tmpDir, indexer)

	retro := &Retrospective{
		RunID:     "test-run-1",
		Pipeline:  "impl-issue",
		Timestamp: time.Now().Truncate(time.Second),
		Quantitative: &QuantitativeData{
			TotalDurationMs: 120000,
			TotalSteps:      2,
			SuccessCount:    2,
			Steps: []StepMetrics{
				{Name: "plan", DurationMs: 30000, Status: "success"},
				{Name: "implement", DurationMs: 90000, Status: "success"},
			},
		},
	}

	// Save
	if err := s.Save(retro); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	filePath := filepath.Join(tmpDir, "test-run-1.json")
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("file not created: %v", err)
	}

	// Verify index entry
	if indexer.lastSaved == nil {
		t.Fatal("index entry not saved")
	}
	if indexer.lastSaved.RunID != "test-run-1" {
		t.Errorf("indexed run ID: got %s, want test-run-1", indexer.lastSaved.RunID)
	}
	if indexer.lastSaved.Status != "quantitative" {
		t.Errorf("indexed status: got %s, want quantitative", indexer.lastSaved.Status)
	}

	// Load
	loaded, err := s.Load("test-run-1")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.RunID != retro.RunID {
		t.Errorf("loaded run ID: got %s, want %s", loaded.RunID, retro.RunID)
	}
	if loaded.Quantitative.TotalSteps != 2 {
		t.Errorf("loaded total steps: got %d, want 2", loaded.Quantitative.TotalSteps)
	}
}

func TestStorage_Update(t *testing.T) {
	tmpDir := t.TempDir()
	indexer := newMockIndexer()
	s := NewStorage(tmpDir, indexer)

	retro := &Retrospective{
		RunID:     "test-run-update",
		Pipeline:  "impl-issue",
		Timestamp: time.Now(),
		Quantitative: &QuantitativeData{
			TotalDurationMs: 60000,
			TotalSteps:      1,
			SuccessCount:    1,
		},
	}

	if err := s.Save(retro); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Add narrative and update
	retro.Narrative = &Narrative{
		Smoothness: SmoothnessSmooth,
		Intent:     "Test run",
		Outcome:    "Completed successfully",
	}

	if err := s.Update(retro); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if indexer.smoothUpdate != "smooth" {
		t.Errorf("smoothness not updated: got %s, want smooth", indexer.smoothUpdate)
	}
	if indexer.statusUpdate != "complete" {
		t.Errorf("status not updated: got %s, want complete", indexer.statusUpdate)
	}

	// Reload and verify
	loaded, err := s.Load("test-run-update")
	if err != nil {
		t.Fatalf("Load after update failed: %v", err)
	}
	if loaded.Narrative == nil {
		t.Fatal("narrative not persisted")
	}
	if loaded.Narrative.Smoothness != SmoothnessSmooth {
		t.Errorf("narrative smoothness: got %s, want smooth", loaded.Narrative.Smoothness)
	}
}

func TestStorage_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	indexer := newMockIndexer()
	s := NewStorage(tmpDir, indexer)

	retro := &Retrospective{
		RunID:        "test-run-delete",
		Pipeline:     "test",
		Timestamp:    time.Now(),
		Quantitative: &QuantitativeData{},
	}

	if err := s.Save(retro); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if err := s.Delete("test-run-delete"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, ok := indexer.records["test-run-delete"]; ok {
		t.Error("index entry not deleted")
	}
}

func TestStorage_LoadNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	indexer := newMockIndexer()
	s := NewStorage(tmpDir, indexer)

	_, err := s.Load("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent retro")
	}
}
