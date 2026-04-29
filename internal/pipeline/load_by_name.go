package pipeline

import (
	"fmt"
	"os"
)

// LoadByName resolves a pipeline name to a *Pipeline using the canonical
// CLI lookup precedence:
//
//  1. .agents/pipelines/<name>.yaml
//  2. .agents/pipelines/<name>
//  3. <name> as a literal path
//
// Strict validation (KnownFields, IO type checks, WLP enforcement) is
// applied via YAMLPipelineLoader; this is the executor-side load path,
// distinct from LoadPipelineLenient used by read-only scanners.
//
// Returns a "pipeline '<name>' not found" error when none of the candidate
// paths exist on disk.
func LoadByName(name string) (*Pipeline, error) {
	candidates := []string{
		".agents/pipelines/" + name + ".yaml",
		".agents/pipelines/" + name,
		name,
	}

	var pipelinePath string
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			pipelinePath = candidate
			break
		}
	}

	if pipelinePath == "" {
		return nil, fmt.Errorf("pipeline '%s' not found (searched .agents/pipelines/)", name)
	}

	pipelineData, err := os.ReadFile(pipelinePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pipeline file: %w", err)
	}

	loader := &YAMLPipelineLoader{}
	return loader.Unmarshal(pipelineData)
}
