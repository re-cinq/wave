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

	if errs := validateAdapters(m.Adapters, basePath); errs != nil {
		errs = append(errs, errs...)
	}

	if errs := validatePersonasList(m.Personas, m.Adapters, basePath); errs != nil {
		errs = append(errs, errs...)
	}

	if errs := validateSkillMounts(m.SkillMounts, basePath); errs != nil {
		errs = append(errs, errs...)
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
		return &ValidationError{Field: "runtime.workspaceRoot", Reason: "is required"}
	}
	return nil
}

func validateAdapters(adapters []Adapter, basePath string) []error {
	var errs []error
	for i, adapter := range adapters {
		if strings.TrimSpace(adapter.Binary) == "" {
			errs = append(errs, &ValidationError{
				Field:  fmt.Sprintf("adapters[%d].binary", i),
				Reason: "is required",
			})
		}
		if strings.TrimSpace(adapter.Mode) == "" {
			errs = append(errs, &ValidationError{
				Field:  fmt.Sprintf("adapters[%d].mode", i),
				Reason: "is required",
			})
		}
	}
	return errs
}

func validatePersonasList(personas []Persona, adapters []Adapter, basePath string) []error {
	var errs []error
	adapterNames := make(map[string]bool)
	for _, adapter := range adapters {
		adapterNames[adapter.Binary] = true
	}

	for i, persona := range personas {
		if strings.TrimSpace(persona.Adapter) == "" {
			errs = append(errs, &ValidationError{
				Field:  fmt.Sprintf("personas[%d].adapter", i),
				Reason: "is required",
			})
		} else if !adapterNames[persona.Adapter] {
			errs = append(errs, &ValidationError{
				Field:  fmt.Sprintf("personas[%d].adapter", i),
				Reason: fmt.Sprintf("adapter '%s' not found in adapters list", persona.Adapter),
			})
		}

		if strings.TrimSpace(persona.SystemPromptFile) == "" {
			errs = append(errs, &ValidationError{
				Field:  fmt.Sprintf("personas[%d].systemPromptFile", i),
				Reason: "is required",
			})
		} else {
			promptPath := persona.SystemPromptFile
			if !filepath.IsAbs(promptPath) {
				promptPath = filepath.Join(basePath, promptPath)
			}
			if _, err := os.Stat(promptPath); os.IsNotExist(err) {
				errs = append(errs, &ValidationError{
					Field:  fmt.Sprintf("personas[%d].systemPromptFile", i),
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
