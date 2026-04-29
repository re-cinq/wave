// loader.go centralises on-disk pipeline YAML loading for the runner package.
// Both webui (via ForkController, etc.) and the cmd path can resolve a
// pipeline by name without importing internal/pipeline directly.
package runner

import (
	"fmt"
	"os"
	"regexp"

	"github.com/recinq/wave/internal/pipeline"
)

// validPipelineName matches safe pipeline names: alphanumeric, hyphens,
// underscores, dots. Used to prevent path traversal when resolving pipeline
// files from .agents/pipelines/.
var validPipelineName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// LoadPipelineByName loads a pipeline definition from .agents/pipelines/.
// The name must match [a-zA-Z0-9][a-zA-Z0-9._-]* to prevent path traversal.
//
// This mirrors the webui-internal loader but is exported so adapter helpers
// (ForkController, etc.) can resolve pipelines without forcing webui to
// import internal/pipeline.
func LoadPipelineByName(name string) (*pipeline.Pipeline, error) {
	if !validPipelineName.MatchString(name) {
		return nil, fmt.Errorf("invalid pipeline name")
	}

	candidates := []string{
		".agents/pipelines/" + name + ".yaml",
		".agents/pipelines/" + name,
	}

	var pipelinePath string
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			pipelinePath = candidate
			break
		}
	}

	if pipelinePath == "" {
		return nil, fmt.Errorf("pipeline not found")
	}

	data, err := os.ReadFile(pipelinePath)
	if err != nil {
		return nil, fmt.Errorf("pipeline not found")
	}

	loader := &pipeline.YAMLPipelineLoader{}
	return loader.Unmarshal(data)
}

// PipelineHasStep reports whether the named pipeline contains a step with
// the given ID. It is a convenience wrapper around LoadPipelineByName that
// keeps callers (notably internal/webui handlers) from importing the
// pipeline package solely to range over Steps.
func PipelineHasStep(pipelineName, stepID string) (bool, error) {
	p, err := LoadPipelineByName(pipelineName)
	if err != nil {
		return false, err
	}
	for _, step := range p.Steps {
		if step.ID == stepID {
			return true, nil
		}
	}
	return false, nil
}
