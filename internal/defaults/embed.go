// Package defaults provides embedded default personas, pipelines, and contracts
// that are included in the Wave binary for use with `wave init`.
package defaults

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"gopkg.in/yaml.v3"
)

//go:embed personas/*.md
var personasFS embed.FS

//go:embed pipelines/*.yaml
var pipelinesFS embed.FS

//go:embed contracts/*.json contracts/*.md
var contractsFS embed.FS

//go:embed prompts/**/*.md
var promptsFS embed.FS

//go:embed personas/*.yaml
var personaConfigsFS embed.FS

//go:embed schemas/*.json
var schemasFS embed.FS

//go:embed skills/*/SKILL.md
var skillsFS embed.FS

// GetPersonas returns a map of filename to content for all default personas.
func GetPersonas() (map[string]string, error) {
	return readDir(personasFS, "personas")
}

// GetPersonaConfigs returns parsed persona configurations keyed by persona name
// (e.g. "navigator", not "navigator.yaml").
func GetPersonaConfigs() (map[string]manifest.Persona, error) {
	result := make(map[string]manifest.Persona)

	err := fs.WalkDir(personaConfigsFS, "personas", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		data, err := personaConfigsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		var p manifest.Persona
		if err := yaml.Unmarshal(data, &p); err != nil {
			return fmt.Errorf("parsing %s: %w", path, err)
		}

		name := strings.TrimSuffix(filepath.Base(path), ".yaml")
		result[name] = p
		return nil
	})

	return result, err
}

// GetPipelines returns a map of filename to content for all default pipelines.
func GetPipelines() (map[string]string, error) {
	return readDir(pipelinesFS, "pipelines")
}

// GetContracts returns a map of filename to content for all default contracts.
func GetContracts() (map[string]string, error) {
	return readDir(contractsFS, "contracts")
}

// GetPrompts returns a map of relative path to content for all default prompts.
// Keys are like "speckit-flow/specify.md" (preserving subdirectory structure).
func GetPrompts() (map[string]string, error) {
	return readDirNested(promptsFS, "prompts")
}

// GetSchemas returns a map of filename to content for all default JSON schemas.
func GetSchemas() (map[string]string, error) {
	return readDir(schemasFS, "schemas")
}

// GetSkillTemplates returns a map of skill name to SKILL.md content
// for all shipped skill templates.
func GetSkillTemplates() map[string][]byte {
	result := make(map[string][]byte)

	entries, err := fs.ReadDir(skillsFS, "skills")
	if err != nil {
		return result
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		path := filepath.Join("skills", name, "SKILL.md")
		data, err := skillsFS.ReadFile(path)
		if err != nil {
			continue
		}
		result[name] = data
	}

	return result
}

// SkillTemplateNames returns a sorted list of shipped skill template names.
func SkillTemplateNames() []string {
	templates := GetSkillTemplates()
	names := make([]string, 0, len(templates))
	for name := range templates {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func readDir(fsys embed.FS, dir string) (map[string]string, error) {
	result := make(map[string]string)

	err := fs.WalkDir(fsys, dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		content, err := fsys.ReadFile(path)
		if err != nil {
			return err
		}

		filename := filepath.Base(path)
		result[filename] = string(content)
		return nil
	})

	return result, err
}

func readDirNested(fsys embed.FS, dir string) (map[string]string, error) {
	result := make(map[string]string)

	err := fs.WalkDir(fsys, dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		content, err := fsys.ReadFile(path)
		if err != nil {
			return err
		}

		// Preserve relative path from the embed root directory
		// e.g. "prompts/speckit-flow/specify.md" → "speckit-flow/specify.md"
		relPath := strings.TrimPrefix(path, dir+"/")
		result[relPath] = string(content)
		return nil
	})

	return result, err
}

// PersonaNames returns a list of all persona filenames.
func PersonaNames() []string {
	personas, _ := GetPersonas()
	names := make([]string, 0, len(personas))
	for name := range personas {
		names = append(names, name)
	}
	return names
}

// PipelineNames returns a list of all pipeline filenames.
func PipelineNames() []string {
	pipelines, _ := GetPipelines()
	names := make([]string, 0, len(pipelines))
	for name := range pipelines {
		names = append(names, name)
	}
	return names
}

// ContractNames returns a list of all contract filenames.
func ContractNames() []string {
	contracts, _ := GetContracts()
	names := make([]string, 0, len(contracts))
	for name := range contracts {
		names = append(names, name)
	}
	return names
}

// PromptNames returns a list of all prompt relative paths.
func PromptNames() []string {
	prompts, _ := GetPrompts()
	names := make([]string, 0, len(prompts))
	for name := range prompts {
		names = append(names, name)
	}
	return names
}

// GetReleasePipelines returns only pipelines where metadata.release is true.
// Pipelines that fail to unmarshal are skipped with a warning.
// Returns an empty map (not nil) when no pipelines have release: true.
func GetReleasePipelines() (map[string]string, error) {
	all, err := GetPipelines()
	if err != nil {
		return make(map[string]string), err
	}

	result := make(map[string]string)
	for name, content := range all {
		var p pipeline.Pipeline
		if err := yaml.Unmarshal([]byte(content), &p); err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping pipeline %s: failed to unmarshal: %v\n", name, err)
			continue
		}
		if p.Metadata.Release {
			result[name] = content
		}
	}
	return result, nil
}

// ReleasePipelineNames returns a sorted list of filenames for pipelines
// where metadata.release is true.
func ReleasePipelineNames() []string {
	pipelines, _ := GetReleasePipelines()
	names := make([]string, 0, len(pipelines))
	for name := range pipelines {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
