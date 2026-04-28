package pipeline

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadPipelineLenient parses pipeline YAML bytes into a Pipeline value
// without running the strict validation passes that YAMLPipelineLoader
// applies (KnownFields, IO type checks, WLP enforcement, ...).
//
// It is the single canonical entry point for read-only scanners across
// internal/tui, internal/webui, internal/health, and internal/doctor that
// need to inspect pipeline definitions in bulk and silently skip ones that
// fail to parse. Strict validation belongs to the executor's load path
// (YAMLPipelineLoader), not to discovery scans.
func LoadPipelineLenient(data []byte) (*Pipeline, error) {
	var p Pipeline
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse pipeline: %w", err)
	}
	return &p, nil
}

// LoadPipelineFileLenient reads path and parses it as pipeline YAML using
// LoadPipelineLenient.
func LoadPipelineFileLenient(path string) (*Pipeline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read pipeline file %s: %w", path, err)
	}
	return LoadPipelineLenient(data)
}

// ScanPipelinesDir walks a pipelines directory and returns every YAML file
// successfully parsed into a Pipeline, in directory-listing order. Files
// that fail to read or parse are silently skipped — this matches the
// existing behaviour of every read-only scanner that previously inlined the
// walk + yaml.Unmarshal boilerplate.
//
// An empty dir argument returns (nil, nil); a missing/unreadable directory
// also returns (nil, nil) so callers do not need to special-case fresh
// installs.
func ScanPipelinesDir(dir string) []Pipeline {
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var out []Pipeline
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		p, err := LoadPipelineFileLenient(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		out = append(out, *p)
	}
	return out
}
