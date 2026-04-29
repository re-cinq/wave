// Package defaults provides embedded default personas, pipelines, and contracts
// that are included in the Wave binary for use with `wave init`.
//
// Raw asset access lives in the sub-package internal/defaults/embedfs (no
// manifest dependency, used by internal/adapter + internal/pipeline cold-start
// fallback). This file adds the manifest-typed wrappers that internal/onboarding
// and the init command use.
package defaults

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/recinq/wave/internal/defaults/embedfs"
	"github.com/recinq/wave/internal/manifest"
	"gopkg.in/yaml.v3"
)

// GetPersonas returns a map of filename to content for all default personas.
func GetPersonas() (map[string]string, error) {
	return embedfs.GetPersonas()
}

// GetPersonaConfigs returns parsed persona configurations keyed by persona name
// (e.g. "navigator", not "navigator.yaml").
func GetPersonaConfigs() (map[string]manifest.Persona, error) {
	result := make(map[string]manifest.Persona)
	personaConfigsFS := embedfs.PersonaConfigsFS()

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
	return embedfs.GetPipelines()
}

// GetContracts returns a map of filename to content for all default contracts.
func GetContracts() (map[string]string, error) {
	return embedfs.GetContracts()
}

// GetPrompts returns a map of relative path to content for all default prompts.
// Keys are like "speckit-flow/specify.md" (preserving subdirectory structure).
func GetPrompts() (map[string]string, error) {
	return embedfs.GetPrompts()
}

// GetSchemas returns a map of filename to content for all default JSON schemas.
func GetSchemas() (map[string]string, error) {
	schemasFS := embedfs.SchemasFS()
	result := make(map[string]string)
	err := fs.WalkDir(schemasFS, "schemas", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, rerr := schemasFS.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		result[filepath.Base(path)] = string(data)
		return nil
	})
	return result, err
}

// GetSkillTemplates returns a map of skill name to SKILL.md content
// for all shipped skill templates.
func GetSkillTemplates() map[string][]byte {
	skillsFS := embedfs.SkillsFS()
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
		header, err := manifest.LoadPipelineHeader([]byte(content))
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping pipeline %s: failed to unmarshal: %v\n", name, err)
			continue
		}
		if header.Metadata.Release {
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
