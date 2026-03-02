package proposal

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/recinq/wave/internal/pipeline"
	"gopkg.in/yaml.v3"
)

// CatalogEntry represents a discovered pipeline with its metadata.
type CatalogEntry struct {
	Name        string
	Description string
	Category    string
	Requires    *pipeline.Requires
	InputSource string // "cli", "meta", etc.
	FilePath    string // Absolute path to the pipeline YAML
}

// Catalog discovers and manages the available pipeline definitions.
type Catalog struct {
	entries []CatalogEntry
}

// NewCatalog creates a Catalog by scanning the given directories for
// pipeline YAML files. Pipelines are deduplicated by name — if the same
// pipeline name appears in multiple directories, the first occurrence wins.
// Disabled pipelines are excluded.
func NewCatalog(dirs ...string) (*Catalog, error) {
	seen := make(map[string]bool)
	var entries []CatalogEntry

	for _, dir := range dirs {
		dirEntries, err := scanDir(dir)
		if err != nil {
			// Skip directories that don't exist or can't be read.
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("scanning %s: %w", dir, err)
		}
		for _, e := range dirEntries {
			if seen[e.Name] {
				continue
			}
			seen[e.Name] = true
			entries = append(entries, e)
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	return &Catalog{entries: entries}, nil
}

// Entries returns a copy of all catalog entries.
func (c *Catalog) Entries() []CatalogEntry {
	out := make([]CatalogEntry, len(c.entries))
	copy(out, c.entries)
	return out
}

// Len returns the number of entries in the catalog.
func (c *Catalog) Len() int {
	return len(c.entries)
}

func scanDir(dir string) ([]CatalogEntry, error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var entries []CatalogEntry
	for _, de := range dirEntries {
		if de.IsDir() {
			continue
		}
		ext := filepath.Ext(de.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		entry, err := parseCatalogEntry(filepath.Join(dir, de.Name()))
		if err != nil {
			// Skip malformed files — don't block discovery.
			continue
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func parseCatalogEntry(path string) (CatalogEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return CatalogEntry{}, err
	}

	var p pipeline.Pipeline
	if err := yaml.Unmarshal(data, &p); err != nil {
		return CatalogEntry{}, err
	}

	if p.Metadata.Name == "" {
		return CatalogEntry{}, fmt.Errorf("pipeline at %s has no name", path)
	}

	if p.Metadata.Disabled {
		return CatalogEntry{}, fmt.Errorf("pipeline %s is disabled", p.Metadata.Name)
	}

	return CatalogEntry{
		Name:        p.Metadata.Name,
		Description: p.Metadata.Description,
		Category:    p.Metadata.Category,
		Requires:    p.Requires,
		InputSource: p.Input.Source,
		FilePath:    path,
	}, nil
}
