package tui

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/recinq/wave/internal/pipeline"
	"gopkg.in/yaml.v3"
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
			// Skip malformed files — don't block discovery.
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

	var p pipeline.Pipeline
	if err := yaml.Unmarshal(data, &p); err != nil {
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
