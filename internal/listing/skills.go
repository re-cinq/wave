package listing

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// CollectSkillsFromPipelines scans all pipeline YAML files and returns a merged
// map of skill name to config. When multiple pipelines declare the same skill,
// the first definition wins (pipelines are scanned in alphabetical order).
func CollectSkillsFromPipelines() map[string]PipelineSkillConfig {
	merged := make(map[string]PipelineSkillConfig)

	entries, err := os.ReadDir(DefaultPipelineDir)
	if err != nil {
		return merged
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		pipelinePath := filepath.Join(DefaultPipelineDir, entry.Name())
		data, err := os.ReadFile(pipelinePath)
		if err != nil {
			continue
		}

		var p struct {
			Requires *struct {
				Skills map[string]PipelineSkillConfig `yaml:"skills"`
			} `yaml:"requires"`
		}
		if err := yaml.Unmarshal(data, &p); err != nil {
			continue
		}

		if p.Requires != nil {
			for name, cfg := range p.Requires.Skills {
				if _, exists := merged[name]; !exists {
					merged[name] = cfg
				}
			}
		}
	}

	return merged
}

// CollectSkillPipelineUsage scans pipeline YAML files and returns a map of
// skill name to the pipelines that require it. Pipeline names per skill are
// returned in alphabetical order.
func CollectSkillPipelineUsage() map[string][]string {
	usage := make(map[string][]string)

	entries, err := os.ReadDir(DefaultPipelineDir)
	if err != nil {
		return usage
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		pipelineName := strings.TrimSuffix(entry.Name(), ".yaml")
		pipelinePath := filepath.Join(DefaultPipelineDir, entry.Name())

		data, err := os.ReadFile(pipelinePath)
		if err != nil {
			continue
		}

		var p struct {
			Requires *struct {
				Skills map[string]PipelineSkillConfig `yaml:"skills"`
			} `yaml:"requires"`
		}
		if err := yaml.Unmarshal(data, &p); err != nil {
			continue
		}

		if p.Requires != nil {
			for skillName := range p.Requires.Skills {
				usage[skillName] = append(usage[skillName], pipelineName)
			}
		}
	}

	for skill := range usage {
		sort.Strings(usage[skill])
	}

	return usage
}

// ListSkills resolves the supplied skill map into SkillInfo records, running
// each skill's `check` command to determine its installed status and attaching
// the list of pipelines that require it.
func ListSkills(skills map[string]PipelineSkillConfig) []SkillInfo {
	if len(skills) == 0 {
		return nil
	}

	names := make([]string, 0, len(skills))
	for name := range skills {
		names = append(names, name)
	}
	sort.Strings(names)

	pipelineUsage := CollectSkillPipelineUsage()

	result := make([]SkillInfo, 0, len(names))
	for _, name := range names {
		skill := skills[name]

		installed := false
		if skill.Check != "" {
			cmd := exec.Command("sh", "-c", skill.Check)
			if err := cmd.Run(); err == nil {
				installed = true
			}
		}

		result = append(result, SkillInfo{
			Name:      name,
			Check:     skill.Check,
			Install:   skill.Install,
			Installed: installed,
			UsedBy:    pipelineUsage[name],
		})
	}

	return result
}
