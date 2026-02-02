package manifest

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type ValidationError struct {
	File   string
	Line   int
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s:%d: %s: %s", e.File, e.Line, e.Field, e.Reason)
	}
	return fmt.Sprintf("%s: %s: %s", e.File, e.Field, e.Reason)
}

type ManifestLoader interface {
	Load(path string) (*Manifest, error)
}

type yamlLoader struct{}

func NewLoader() ManifestLoader {
	return &yamlLoader{}
}

func (l *yamlLoader) Load(path string) (*Manifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open manifest file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}

	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	manifestPath := filepath.Dir(path)
	if errs := Validate(&manifest, manifestPath); len(errs) > 0 {
		return nil, errs[0]
	}

	return &manifest, nil
}

func Validate(m *Manifest, basePath string) []error {
	var errs []error

	if err := validateMetadata(&m.Metadata, basePath); err != nil {
		errs = append(errs, err)
	}

	if err := validateRuntime(&m.Runtime, basePath); err != nil {
		errs = append(errs, err)
	}

	if adapterErrs := validateAdapters(m.Adapters, basePath); len(adapterErrs) > 0 {
		errs = append(errs, adapterErrs...)
	}

	if personaErrs := validatePersonasList(m.Personas, m.Adapters, basePath); len(personaErrs) > 0 {
		errs = append(errs, personaErrs...)
	}

	if mountErrs := validateSkillMounts(m.SkillMounts, basePath); len(mountErrs) > 0 {
		errs = append(errs, mountErrs...)
	}

	return errs
}

func validateMetadata(m *Metadata, basePath string) *ValidationError {
	if strings.TrimSpace(m.Name) == "" {
		return &ValidationError{Field: "metadata.name", Reason: "is required"}
	}
	return nil
}

func validateRuntime(r *Runtime, basePath string) *ValidationError {
	if strings.TrimSpace(r.WorkspaceRoot) == "" {
		return &ValidationError{Field: "runtime.workspace_root", Reason: "is required"}
	}
	return nil
}

func validateAdapters(adapters map[string]Adapter, basePath string) []error {
	var errs []error
	for name, adapter := range adapters {
		if strings.TrimSpace(adapter.Binary) == "" {
			errs = append(errs, &ValidationError{
				Field:  fmt.Sprintf("adapters.%s.binary", name),
				Reason: "is required",
			})
		}
		if strings.TrimSpace(adapter.Mode) == "" {
			errs = append(errs, &ValidationError{
				Field:  fmt.Sprintf("adapters.%s.mode", name),
				Reason: "is required",
			})
		}
	}
	return errs
}

func validatePersonasList(personas map[string]Persona, adapters map[string]Adapter, basePath string) []error {
	var errs []error

	for name, persona := range personas {
		if strings.TrimSpace(persona.Adapter) == "" {
			errs = append(errs, &ValidationError{
				Field:  fmt.Sprintf("personas.%s.adapter", name),
				Reason: "is required",
			})
		} else if _, ok := adapters[persona.Adapter]; !ok {
			errs = append(errs, &ValidationError{
				Field:  fmt.Sprintf("personas.%s.adapter", name),
				Reason: fmt.Sprintf("adapter '%s' not found in adapters map", persona.Adapter),
			})
		}

		if strings.TrimSpace(persona.SystemPromptFile) == "" {
			errs = append(errs, &ValidationError{
				Field:  fmt.Sprintf("personas.%s.system_prompt_file", name),
				Reason: "is required",
			})
		} else {
			promptPath := persona.SystemPromptFile
			if !filepath.IsAbs(promptPath) {
				promptPath = filepath.Join(basePath, promptPath)
			}
			if _, err := os.Stat(promptPath); os.IsNotExist(err) {
				errs = append(errs, &ValidationError{
					Field:  fmt.Sprintf("personas.%s.system_prompt_file", name),
					Reason: fmt.Sprintf("file '%s' does not exist", persona.SystemPromptFile),
				})
			}
		}
	}
	return errs
}

func validateSkillMounts(mounts []SkillMount, basePath string) []error {
	var errs []error
	for i, mount := range mounts {
		if strings.TrimSpace(mount.Path) == "" {
			errs = append(errs, &ValidationError{
				Field:  fmt.Sprintf("skillMounts[%d].path", i),
				Reason: "is required",
			})
		}
	}
	return errs
}

func Load(path string) (*Manifest, error) {
	return NewLoader().Load(path)
}
