package pipeline

import (
	"fmt"
	"os"

	"github.com/recinq/wave/internal/defaults/embedfs"
)

// bootstrapPipelines may load directly from embedded defaults so a fresh
// repo with no .agents/pipelines/ scaffold can still run them. Other
// pipelines must be present on disk, preserving the test/manifest contract
// that pipelines listed in wave.yaml own their own runtime resolution.
var bootstrapPipelines = map[string]bool{
	"onboard-project": true,
	"ops-bootstrap":   true,
}

// LoadByName resolves a pipeline name to a *Pipeline using the canonical
// CLI lookup precedence:
//
//  1. .agents/pipelines/<name>.yaml
//  2. .agents/pipelines/<name>
//  3. <name> as a literal path
//  4. embedded defaults (only for the bootstrap allowlist)
//
// Strict validation (KnownFields, IO type checks, WLP enforcement) is
// applied via YAMLPipelineLoader; this is the executor-side load path,
// distinct from LoadPipelineLenient used by read-only scanners.
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

	loader := &YAMLPipelineLoader{}

	if pipelinePath != "" {
		pipelineData, err := os.ReadFile(pipelinePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read pipeline file: %w", err)
		}
		return loader.Unmarshal(pipelineData)
	}

	if bootstrapPipelines[name] {
		if embedded, ok := lookupEmbeddedPipeline(name); ok {
			return loader.Unmarshal([]byte(embedded))
		}
	}

	return nil, fmt.Errorf("pipeline '%s' not found (searched .agents/pipelines/)", name)
}

func lookupEmbeddedPipeline(name string) (string, bool) {
	pipelines, err := embedfs.GetPipelines()
	if err != nil {
		return "", false
	}
	if c, ok := pipelines[name+".yaml"]; ok {
		return c, true
	}
	if c, ok := pipelines[name]; ok {
		return c, true
	}
	return "", false
}
