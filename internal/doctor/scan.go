package doctor

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/recinq/wave/internal/pipeline"
	"gopkg.in/yaml.v3"
)

// collectRequiredTools scans all pipeline YAML files and returns a sorted,
// deduplicated list of required tools.
func collectRequiredTools(pipelinesDir string) []string {
	toolSet := make(map[string]bool)
	for _, pl := range loadAllPipelines(pipelinesDir) {
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
	for _, pl := range loadAllPipelines(pipelinesDir) {
		if pl.Requires != nil {
			for name, cfg := range pl.Requires.Skills {
				skills[name] = cfg.Check
			}
		}
	}
	return skills
}

// loadAllPipelines reads and parses all YAML files in the pipelines directory.
func loadAllPipelines(pipelinesDir string) []pipeline.Pipeline {
	if pipelinesDir == "" {
		return nil
	}
	entries, err := os.ReadDir(pipelinesDir)
	if err != nil {
		return nil
	}

	var pipelines []pipeline.Pipeline
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(pipelinesDir, entry.Name()))
		if err != nil {
			continue
		}
		var pl pipeline.Pipeline
		if err := yaml.Unmarshal(data, &pl); err != nil {
			continue
		}
		pipelines = append(pipelines, pl)
	}
	return pipelines
}
