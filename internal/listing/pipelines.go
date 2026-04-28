package listing

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/recinq/wave/internal/pipeline"
)

// DefaultPipelineDir is where Wave stores pipeline YAML files for a project.
const DefaultPipelineDir = ".agents/pipelines"

// ListPipelines reads all pipeline YAML files under DefaultPipelineDir and
// returns them in alphabetical order. A missing directory yields a nil slice
// and no error so callers can render an empty-state message.
func ListPipelines() ([]PipelineInfo, error) {
	entries, err := os.ReadDir(DefaultPipelineDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read pipelines directory: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var pipelines []PipelineInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".yaml")
		pipelinePath := filepath.Join(DefaultPipelineDir, entry.Name())

		p, err := pipeline.LoadPipelineFileLenient(pipelinePath)
		if err != nil {
			// Distinguish read errors from parse errors to preserve the
			// original UX (different placeholder text per failure mode).
			if _, statErr := os.Stat(pipelinePath); statErr != nil {
				pipelines = append(pipelines, PipelineInfo{Name: name, Description: "(error reading)"})
			} else {
				pipelines = append(pipelines, PipelineInfo{Name: name, Description: "(error parsing)"})
			}
			continue
		}

		stepIDs := make([]string, 0, len(p.Steps))
		for _, s := range p.Steps {
			stepIDs = append(stepIDs, s.ID)
		}

		pipelines = append(pipelines, PipelineInfo{
			Name:        name,
			Description: p.Metadata.Description,
			StepCount:   len(p.Steps),
			Steps:       stepIDs,
		})
	}

	return pipelines, nil
}

// ExtractPipelineName strips the run ID suffix from a workspace directory name
// by walking back through dash-separated segments and matching against pipeline
// files on disk. e.g. "adr-0718471d" -> "adr".
func ExtractPipelineName(wsName string) string {
	parts := strings.Split(wsName, "-")
	for i := len(parts) - 1; i >= 1; i-- {
		candidate := strings.Join(parts[:i], "-")
		if _, err := os.Stat(filepath.Join(DefaultPipelineDir, candidate+".yaml")); err == nil {
			return candidate
		}
	}
	return wsName
}
