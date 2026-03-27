package retro

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/recinq/wave/internal/state"
)

// validRunID matches only alphanumeric characters, hyphens, and underscores.
var validRunID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// validateRunID checks that a run ID is safe for use in filesystem paths.
func validateRunID(runID string) error {
	if runID == "" {
		return errors.New("run ID must not be empty")
	}
	if !validRunID.MatchString(runID) {
		return fmt.Errorf("invalid run ID: %q", runID)
	}
	return nil
}

// Store defines the interface for persisting and retrieving retrospectives.
type Store interface {
	Save(retro *Retrospective) error
	Get(runID string) (*Retrospective, error)
	List(opts ListOptions) ([]Retrospective, error)
	UpdateNarrative(runID string, narrative *NarrativeData) error
}

// ListOptions specifies filters for listing retrospectives.
type ListOptions struct {
	PipelineName string
	Limit        int
	Since        time.Time
}

// FileStore persists retrospectives as JSON files on disk and mirrors
// records into SQLite via a state.StateStore for indexed querying.
type FileStore struct {
	baseDir    string           // e.g., ".wave/retros"
	stateStore state.StateStore // for SQLite persistence
}

// NewFileStore creates a new FileStore rooted at baseDir.
func NewFileStore(baseDir string, stateStore state.StateStore) *FileStore {
	return &FileStore{
		baseDir:    baseDir,
		stateStore: stateStore,
	}
}

// Save marshals a Retrospective to JSON, writes it to <baseDir>/<run-id>.json,
// and persists a corresponding record in SQLite.
func (fs *FileStore) Save(retro *Retrospective) error {
	if retro == nil {
		return errors.New("retrospective must not be nil")
	}
	if err := validateRunID(retro.RunID); err != nil {
		return fmt.Errorf("invalid retrospective run ID: %w", err)
	}

	// Ensure base directory exists.
	if err := os.MkdirAll(fs.baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create retro directory: %w", err)
	}

	// Marshal full retro to JSON for the file.
	data, err := json.MarshalIndent(retro, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal retrospective: %w", err)
	}

	filePath := filepath.Join(fs.baseDir, retro.RunID+".json")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write retrospective file: %w", err)
	}

	// Build the SQLite record.
	record, err := retroToRecord(retro)
	if err != nil {
		return fmt.Errorf("failed to build retrospective record: %w", err)
	}

	if err := fs.stateStore.SaveRetrospective(record); err != nil {
		return fmt.Errorf("failed to save retrospective to state store: %w", err)
	}

	return nil
}

// Get retrieves a Retrospective by run ID. It tries the local JSON file first;
// if the file does not exist it falls back to SQLite. Returns (nil, nil) when
// neither source has a record.
func (fs *FileStore) Get(runID string) (*Retrospective, error) {
	if err := validateRunID(runID); err != nil {
		return nil, fmt.Errorf("invalid run ID: %w", err)
	}

	// Try file first.
	filePath := filepath.Join(fs.baseDir, runID+".json")
	data, err := os.ReadFile(filePath)
	if err == nil {
		var retro Retrospective
		if jsonErr := json.Unmarshal(data, &retro); jsonErr != nil {
			return nil, fmt.Errorf("failed to parse retrospective file: %w", jsonErr)
		}
		return &retro, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("failed to read retrospective file: %w", err)
	}

	// File not found — try SQLite.
	record, err := fs.stateStore.GetRetrospective(runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get retrospective from state store: %w", err)
	}
	if record == nil {
		return nil, nil
	}

	retro, err := recordToRetro(record)
	if err != nil {
		return nil, fmt.Errorf("failed to convert retrospective record: %w", err)
	}
	return retro, nil
}

// List retrieves retrospectives matching the given options. It delegates to
// the SQLite state store for indexed, filtered querying and converts the
// returned records back to Retrospective values.
func (fs *FileStore) List(opts ListOptions) ([]Retrospective, error) {
	stateOpts := state.ListRetrospectivesOptions{
		PipelineName: opts.PipelineName,
		Limit:        opts.Limit,
	}
	if !opts.Since.IsZero() {
		stateOpts.SinceUnix = opts.Since.Unix()
	}

	records, err := fs.stateStore.ListRetrospectives(stateOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list retrospectives: %w", err)
	}

	retros := make([]Retrospective, 0, len(records))
	for i := range records {
		retro, err := recordToRetro(&records[i])
		if err != nil {
			return nil, fmt.Errorf("failed to convert retrospective record (run %s): %w", records[i].RunID, err)
		}
		retros = append(retros, *retro)
	}
	return retros, nil
}

// UpdateNarrative updates the narrative portion of an existing retrospective.
// It reads the current retro from file, merges the new narrative, rewrites
// the file, and updates the SQLite record.
func (fs *FileStore) UpdateNarrative(runID string, narrative *NarrativeData) error {
	if err := validateRunID(runID); err != nil {
		return fmt.Errorf("invalid run ID: %w", err)
	}
	if narrative == nil {
		return errors.New("narrative must not be nil")
	}

	retro, err := fs.Get(runID)
	if err != nil {
		return fmt.Errorf("failed to get retrospective for narrative update: %w", err)
	}
	if retro == nil {
		return fmt.Errorf("retrospective not found for run: %s", runID)
	}

	retro.Narrative = narrative

	// Rewrite the file.
	data, err := json.MarshalIndent(retro, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated retrospective: %w", err)
	}

	if err := os.MkdirAll(fs.baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create retro directory: %w", err)
	}

	filePath := filepath.Join(fs.baseDir, runID+".json")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write updated retrospective file: %w", err)
	}

	// Update SQLite.
	narrativeJSON, err := json.Marshal(narrative)
	if err != nil {
		return fmt.Errorf("failed to marshal narrative for state store: %w", err)
	}

	if err := fs.stateStore.UpdateRetrospectiveNarrative(runID, string(narrativeJSON), narrative.Smoothness); err != nil {
		return fmt.Errorf("failed to update narrative in state store: %w", err)
	}

	return nil
}

// retroToRecord converts a Retrospective to a RetrospectiveRecord suitable
// for SQLite storage.
func retroToRecord(retro *Retrospective) (*state.RetrospectiveRecord, error) {
	quantJSON, err := json.Marshal(retro.Quantitative)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal quantitative data: %w", err)
	}

	var narrativeJSON string
	var smoothness string
	if retro.Narrative != nil {
		nData, err := json.Marshal(retro.Narrative)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal narrative data: %w", err)
		}
		narrativeJSON = string(nData)
		smoothness = retro.Narrative.Smoothness
	}

	return &state.RetrospectiveRecord{
		RunID:            retro.RunID,
		PipelineName:     retro.Pipeline,
		QuantitativeJSON: string(quantJSON),
		NarrativeJSON:    narrativeJSON,
		Smoothness:       smoothness,
		GeneratedAt:      retro.Timestamp,
	}, nil
}

// recordToRetro converts a RetrospectiveRecord from SQLite back to a
// Retrospective.
func recordToRetro(record *state.RetrospectiveRecord) (*Retrospective, error) {
	retro := &Retrospective{
		RunID:     record.RunID,
		Pipeline:  record.PipelineName,
		Timestamp: record.GeneratedAt,
	}

	if record.QuantitativeJSON != "" {
		if err := json.Unmarshal([]byte(record.QuantitativeJSON), &retro.Quantitative); err != nil {
			return nil, fmt.Errorf("failed to parse quantitative JSON: %w", err)
		}
	}

	if record.NarrativeJSON != "" {
		var narrative NarrativeData
		if err := json.Unmarshal([]byte(record.NarrativeJSON), &narrative); err != nil {
			return nil, fmt.Errorf("failed to parse narrative JSON: %w", err)
		}
		retro.Narrative = &narrative
	}

	return retro, nil
}
