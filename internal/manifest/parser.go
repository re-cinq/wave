package manifest

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ValidationError represents an error found during manifest validation.
// It includes context like file path, line number, field name, and suggestions.
type ValidationError struct {
	File       string
	Line       int
	Column     int
	Field      string
	Reason     string
	Suggestion string
}

func (e *ValidationError) Error() string {
	var sb strings.Builder

	// Build location prefix
	if e.File != "" {
		sb.WriteString(e.File)
		if e.Line > 0 {
			sb.WriteString(fmt.Sprintf(":%d", e.Line))
			if e.Column > 0 {
				sb.WriteString(fmt.Sprintf(":%d", e.Column))
			}
		}
		sb.WriteString(": ")
	}

	// Add field and reason
	if e.Field != "" {
		sb.WriteString(e.Field)
		sb.WriteString(": ")
	}
	sb.WriteString(e.Reason)

	// Add suggestion if present
	if e.Suggestion != "" {
		sb.WriteString("\n  Hint: ")
		sb.WriteString(e.Suggestion)
	}

	return sb.String()
}

// NewValidationError creates a ValidationError with the given field and reason.
func NewValidationError(field, reason string) *ValidationError {
	return &ValidationError{Field: field, Reason: reason}
}

// WithFile sets the file path on the error.
func (e *ValidationError) WithFile(file string) *ValidationError {
	e.File = file
	return e
}

// WithLine sets the line number on the error.
func (e *ValidationError) WithLine(line int) *ValidationError {
	e.Line = line
	return e
}

// WithSuggestion adds a helpful suggestion to the error message.
func (e *ValidationError) WithSuggestion(suggestion string) *ValidationError {
	e.Suggestion = suggestion
	return e
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
		if os.IsNotExist(err) {
			return nil, &ValidationError{
				File:       path,
				Reason:     "manifest file not found",
				Suggestion: "Run 'wave init' to create a new Wave project",
			}
		}
		return nil, fmt.Errorf("failed to open manifest file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}

	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		// Try to extract line number from YAML error
		return nil, parseYAMLError(path, err)
	}

	manifestPath := filepath.Dir(path)
	if errs := ValidateWithFile(&manifest, manifestPath, path); len(errs) > 0 {
		return nil, errs[0]
	}

	return &manifest, nil
}

// parseYAMLError extracts line/column information from a YAML parse error.
func parseYAMLError(file string, err error) error {
	// yaml.v3 errors include line numbers, try to preserve them
	errMsg := err.Error()

	// Look for "yaml: line X:" pattern
	if strings.Contains(errMsg, "line") {
		return &ValidationError{
			File:       file,
			Reason:     fmt.Sprintf("YAML syntax error: %s", errMsg),
			Suggestion: "Check for incorrect indentation, missing colons, or invalid characters",
		}
	}

	return &ValidationError{
		File:       file,
		Reason:     fmt.Sprintf("failed to parse YAML: %s", errMsg),
		Suggestion: "Ensure the file is valid YAML with correct indentation",
	}
}

// Validate validates a manifest without file context.
func Validate(m *Manifest, basePath string) []error {
	return ValidateWithFile(m, basePath, "")
}

// ValidateWithFile validates a manifest and includes file context in errors.
func ValidateWithFile(m *Manifest, basePath, filePath string) []error {
	var errs []error

	if err := validateMetadata(&m.Metadata, basePath); err != nil {
		if filePath != "" {
			err.File = filePath
		}
		errs = append(errs, err)
	}

	if err := validateRuntime(&m.Runtime, basePath); err != nil {
		if filePath != "" {
			err.File = filePath
		}
		errs = append(errs, err)
	}

	if adapterErrs := validateAdaptersWithFile(m.Adapters, basePath, filePath); len(adapterErrs) > 0 {
		errs = append(errs, adapterErrs...)
	}

	if personaErrs := validatePersonasListWithFile(m.Personas, m.Adapters, basePath, filePath); len(personaErrs) > 0 {
		errs = append(errs, personaErrs...)
	}

	if mountErrs := validateSkillMountsWithFile(m.SkillMounts, basePath, filePath); len(mountErrs) > 0 {
		errs = append(errs, mountErrs...)
	}

	if skillErrs := validateSkillsWithFile(m.Skills, filePath); len(skillErrs) > 0 {
		errs = append(errs, skillErrs...)
	}

	return errs
}

func validateMetadata(m *Metadata, basePath string) *ValidationError {
	if strings.TrimSpace(m.Name) == "" {
		return &ValidationError{
			Field:      "metadata.name",
			Reason:     "is required",
			Suggestion: "Add a 'name' field under 'metadata' to identify your project",
		}
	}
	return nil
}

func validateRuntime(r *Runtime, basePath string) *ValidationError {
	if strings.TrimSpace(r.WorkspaceRoot) == "" {
		return &ValidationError{
			Field:      "runtime.workspace_root",
			Reason:     "is required",
			Suggestion: "Set 'workspace_root' to a directory path like '.wave/workspaces'",
		}
	}
	return nil
}

func validateAdapters(adapters map[string]Adapter, basePath string) []error {
	return validateAdaptersWithFile(adapters, basePath, "")
}

func validateAdaptersWithFile(adapters map[string]Adapter, basePath, filePath string) []error {
	var errs []error
	for name, adapter := range adapters {
		if strings.TrimSpace(adapter.Binary) == "" {
			errs = append(errs, &ValidationError{
				File:       filePath,
				Field:      fmt.Sprintf("adapters.%s.binary", name),
				Reason:     "is required",
				Suggestion: fmt.Sprintf("Set 'binary' to the CLI executable name (e.g., 'claude', 'opencode')"),
			})
		}
		if strings.TrimSpace(adapter.Mode) == "" {
			errs = append(errs, &ValidationError{
				File:       filePath,
				Field:      fmt.Sprintf("adapters.%s.mode", name),
				Reason:     "is required",
				Suggestion: "Set 'mode' to 'headless' for non-interactive execution",
			})
		}
	}
	return errs
}

func validatePersonasList(personas map[string]Persona, adapters map[string]Adapter, basePath string) []error {
	return validatePersonasListWithFile(personas, adapters, basePath, "")
}

func validatePersonasListWithFile(personas map[string]Persona, adapters map[string]Adapter, basePath, filePath string) []error {
	var errs []error

	// Collect available adapter names for suggestions
	availableAdapters := make([]string, 0, len(adapters))
	for adapterName := range adapters {
		availableAdapters = append(availableAdapters, adapterName)
	}

	for name, persona := range personas {
		if strings.TrimSpace(persona.Adapter) == "" {
			errs = append(errs, &ValidationError{
				File:       filePath,
				Field:      fmt.Sprintf("personas.%s.adapter", name),
				Reason:     "is required",
				Suggestion: "Set 'adapter' to reference a defined adapter (e.g., 'claude')",
			})
		} else if _, ok := adapters[persona.Adapter]; !ok {
			suggestion := fmt.Sprintf("adapter '%s' not found", persona.Adapter)
			if len(availableAdapters) > 0 {
				suggestion = fmt.Sprintf("Available adapters: %v", availableAdapters)
			} else {
				suggestion = "Define an adapter in the 'adapters' section first"
			}
			errs = append(errs, &ValidationError{
				File:       filePath,
				Field:      fmt.Sprintf("personas.%s.adapter", name),
				Reason:     fmt.Sprintf("adapter '%s' not found in adapters map", persona.Adapter),
				Suggestion: suggestion,
			})
		}

		if strings.TrimSpace(persona.SystemPromptFile) == "" {
			errs = append(errs, &ValidationError{
				File:       filePath,
				Field:      fmt.Sprintf("personas.%s.system_prompt_file", name),
				Reason:     "is required",
				Suggestion: "Set 'system_prompt_file' to a markdown file path (e.g., '.wave/personas/navigator.md')",
			})
		} else {
			promptPath := persona.SystemPromptFile
			if !filepath.IsAbs(promptPath) {
				promptPath = filepath.Join(basePath, promptPath)
			}
			if _, err := os.Stat(promptPath); os.IsNotExist(err) {
				errs = append(errs, &ValidationError{
					File:       filePath,
					Field:      fmt.Sprintf("personas.%s.system_prompt_file", name),
					Reason:     fmt.Sprintf("file '%s' does not exist", persona.SystemPromptFile),
					Suggestion: fmt.Sprintf("Create the file at '%s' or update the path", promptPath),
				})
			}
		}
	}
	return errs
}

func validateSkillMounts(mounts []SkillMount, basePath string) []error {
	return validateSkillMountsWithFile(mounts, basePath, "")
}

func validateSkillMountsWithFile(mounts []SkillMount, basePath, filePath string) []error {
	var errs []error
	for i, mount := range mounts {
		if strings.TrimSpace(mount.Path) == "" {
			errs = append(errs, &ValidationError{
				File:       filePath,
				Field:      fmt.Sprintf("skill_mounts[%d].path", i),
				Reason:     "is required",
				Suggestion: "Set 'path' to a directory containing skill definitions",
			})
		}
	}
	return errs
}

// validateSkillsWithFile validates the skills configuration map.
func validateSkillsWithFile(skills map[string]SkillConfig, filePath string) []error {
	var errs []error
	for name, skill := range skills {
		// A skill must have at least a check command to verify installation
		if strings.TrimSpace(skill.Check) == "" {
			errs = append(errs, &ValidationError{
				File:       filePath,
				Field:      fmt.Sprintf("skills.%s.check", name),
				Reason:     "is required",
				Suggestion: "Set 'check' to a command that verifies the skill is installed (e.g., 'specify --version')",
			})
		}
	}
	return errs
}

func Load(path string) (*Manifest, error) {
	return NewLoader().Load(path)
}
