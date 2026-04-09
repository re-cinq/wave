package retro

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/recinq/wave/internal/state"
)

// RetroIndexer is the subset of state.StateStore needed by Storage.
type RetroIndexer interface {
	SaveRetrospective(record *state.RetrospectiveRecord) error
	GetRetrospective(runID string) (*state.RetrospectiveRecord, error)
	ListRetrospectives(opts state.ListRetrosOptions) ([]state.RetrospectiveRecord, error)
	DeleteRetrospective(runID string) error
	UpdateRetrospectiveSmoothness(runID string, smoothness string) error
	UpdateRetrospectiveStatus(runID string, status string) error
}

// Storage handles retrospective persistence (JSON files + SQLite index).
type Storage struct {
	retrosDir string
	indexer   RetroIndexer
}

// NewStorage creates a new Storage rooted at the given directory.
func NewStorage(retrosDir string, indexer RetroIndexer) *Storage {
	return &Storage{
		retrosDir: retrosDir,
		indexer:   indexer,
	}
}

// Save writes a retrospective to a JSON file and creates an index entry.
func (s *Storage) Save(retro *Retrospective) error {
	if err := os.MkdirAll(s.retrosDir, 0755); err != nil {
		return fmt.Errorf("failed to create retros directory: %w", err)
	}

	filePath := s.filePath(retro.RunID)
	data, err := json.MarshalIndent(retro, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal retrospective: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write retrospective file: %w", err)
	}

	// Index in SQLite
	smoothness := ""
	status := "quantitative"
	if retro.Narrative != nil {
		smoothness = string(retro.Narrative.Smoothness)
		status = "complete"
	}

	record := &state.RetrospectiveRecord{
		RunID:        retro.RunID,
		PipelineName: retro.Pipeline,
		Smoothness:   smoothness,
		Status:       status,
		FilePath:     filePath,
		CreatedAt:    retro.Timestamp,
	}

	if err := s.indexer.SaveRetrospective(record); err != nil {
		return fmt.Errorf("failed to save retrospective index: %w", err)
	}

	return nil
}

// Load reads a retrospective from its JSON file.
func (s *Storage) Load(runID string) (*Retrospective, error) {
	filePath := s.filePath(runID)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read retrospective file: %w", err)
	}

	var retro Retrospective
	if err := json.Unmarshal(data, &retro); err != nil {
		return nil, fmt.Errorf("failed to parse retrospective: %w", err)
	}
	return &retro, nil
}

// Update re-writes the retrospective JSON file and updates the index.
func (s *Storage) Update(retro *Retrospective) error {
	data, err := json.MarshalIndent(retro, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal retrospective: %w", err)
	}

	filePath := s.filePath(retro.RunID)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write retrospective file: %w", err)
	}

	if retro.Narrative != nil {
		if err := s.indexer.UpdateRetrospectiveSmoothness(retro.RunID, string(retro.Narrative.Smoothness)); err != nil {
			return fmt.Errorf("failed to update smoothness: %w", err)
		}
		if err := s.indexer.UpdateRetrospectiveStatus(retro.RunID, "complete"); err != nil {
			return fmt.Errorf("failed to update status: %w", err)
		}
	}

	return nil
}

// List returns retrospective records matching the given filters.
func (s *Storage) List(pipelineName string, since time.Time, limit int) ([]state.RetrospectiveRecord, error) {
	opts := state.ListRetrosOptions{
		PipelineName: pipelineName,
		Limit:        limit,
	}
	if !since.IsZero() {
		opts.SinceUnix = since.Unix()
	}
	return s.indexer.ListRetrospectives(opts)
}

// Delete removes a retrospective JSON file and its index entry.
func (s *Storage) Delete(runID string) error {
	filePath := s.filePath(runID)
	_ = os.Remove(filePath) // best effort file removal
	return s.indexer.DeleteRetrospective(runID)
}

// filePath returns the JSON file path for a given run ID.
func (s *Storage) filePath(runID string) string {
	return filepath.Join(s.retrosDir, runID+".json")
}
