package doctor

import (
	"sort"

	"github.com/recinq/wave/internal/pipeline"
)

// collectRequiredTools scans all pipeline YAML files and returns a sorted,
// deduplicated list of required tools.
func collectRequiredTools(pipelinesDir string) []string {
	toolSet := make(map[string]bool)
	for _, pl := range pipeline.ScanPipelinesDir(pipelinesDir) {
		if pl.Requires != nil {
			for _, tool := range pl.Requires.Tools {
				toolSet[tool] = true
			}
		}
	}

	tools := make([]string, 0, len(toolSet))
	for tool := range toolSet {
		tools = append(tools, tool)
	}
	sort.Strings(tools)
	return tools
}

// collectRequiredSkills scans all pipeline YAML files and returns a map of
// skill name → check command.
func collectRequiredSkills(pipelinesDir string) map[string]string {
	skills := make(map[string]string)
	for _, pl := range pipeline.ScanPipelinesDir(pipelinesDir) {
		if pl.Requires != nil {
			for name, cfg := range pl.Requires.Skills {
				skills[name] = cfg.Check
			}
		}
	}
	return skills
}
