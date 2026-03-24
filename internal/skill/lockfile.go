package skill

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PublishRecord represents a published skill entry in the lockfile.
type PublishRecord struct {
	Name        string    `json:"name"`
	Digest      string    `json:"digest"`
	Registry    string    `json:"registry"`
	URL         string    `json:"url"`
	PublishedAt time.Time `json:"published_at"`
}

// Lockfile represents the skill publish lockfile.
type Lockfile struct {
	Version   int             `json:"version"`
	Published []PublishRecord `json:"published"`
}

// LoadLockfile reads and parses a lockfile from path.
// Returns an empty Lockfile with Version 1 if the file does not exist.
func LoadLockfile(path string) (*Lockfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Lockfile{Version: 1}, nil
		}
		return nil, fmt.Errorf("failed to read lockfile: %w", err)
	}

	var lf Lockfile
	if err := json.Unmarshal(data, &lf); err != nil {
		return nil, fmt.Errorf("failed to parse lockfile: %w", err)
	}
	return &lf, nil
}

// Save writes the lockfile to path atomically via a temp file and rename.
func (lf *Lockfile) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create lockfile directory: %w", err)
	}

	data, err := json.MarshalIndent(lf, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal lockfile: %w", err)
	}
	data = append(data, '\n')

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp lockfile: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to rename lockfile: %w", err)
	}

	return nil
}

// FindByName returns the publish record matching name, or nil if not found.
func (lf *Lockfile) FindByName(name string) *PublishRecord {
	for i := range lf.Published {
		if lf.Published[i].Name == name {
			return &lf.Published[i]
		}
	}
	return nil
}

// Upsert inserts or replaces a publish record by name.
func (lf *Lockfile) Upsert(record PublishRecord) {
	for i := range lf.Published {
		if lf.Published[i].Name == record.Name {
			lf.Published[i] = record
			return
		}
	}
	lf.Published = append(lf.Published, record)
}
