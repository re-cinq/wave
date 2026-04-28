// Package pipelinecatalog discovers and loads pipeline metadata from YAML files.
package pipelinecatalog

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/recinq/wave/internal/pipeline"
)

// PipelineInfo holds discoverable metadata about a pipeline.
type PipelineInfo struct {
	Name         string
	Description  string
	StepCount    int
	InputExample string
	Release      bool
	Category     string
}

// DiscoverPipelines scans the given directory for pipeline YAML files
// and returns metadata for each valid pipeline found.
func DiscoverPipelines(dir string) ([]PipelineInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var pipelines []PipelineInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		info, err := parsePipelineFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			// Skip malformed files — don’t block discovery.
			continue
		}
		pipelines = append(pipelines, info)
	}

	sort.Slice(pipelines, func(i, j int) bool {
		return pipelines[i].Name < pipelines[j].Name
	})

	return pipelines, nil
}

func parsePipelineFile(path string) (PipelineInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return PipelineInfo{}, err
	}

	loader := &pipeline.YAMLPipelineLoader{}
	p, err := loader.Unmarshal(data)
	if err != nil {
		return PipelineInfo{}, err
	}

	return PipelineInfo{
		Name:         p.Metadata.Name,
		Description:  p.Metadata.Description,
		StepCount:    len(p.Steps),
		InputExample: p.Input.Example,
		Release:      p.Metadata.Release,
		Category:     p.Metadata.Category,
	}, nil
}

// LoadPipelineByName scans the given directory for pipeline YAML files
// and returns the full Pipeline struct for the first match on metadata name.
func LoadPipelineByName(dir, name string) (*pipeline.Pipeline, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		loader := &pipeline.YAMLPipelineLoader{}
		pParsed, err := loader.Unmarshal(data)
		if err != nil {
			continue
		}

		if pParsed.Metadata.Name == name {
			return pParsed, nil
		}
	}

	return nil, fmt.Errorf("pipeline not found: %s", name)
}
