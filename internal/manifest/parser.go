package manifest

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/recinq/wave/internal/hooks"
	"github.com/recinq/wave/internal/scope"
	"github.com/recinq/wave/internal/skill"
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
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&manifest); err != nil {
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

	if timeoutErrs := validateTimeouts(&m.Runtime.Timeouts, filePath); len(timeoutErrs) > 0 {
		errs = append(errs, timeoutErrs...)
	}

	if adapterErrs := validateAdaptersWithFile(m.Adapters, basePath, filePath); len(adapterErrs) > 0 {
		errs = append(errs, adapterErrs...)
	}

	if personaErrs := validatePersonasListWithFile(m.Personas, m.Adapters, basePath, filePath); len(personaErrs) > 0 {
		errs = append(errs, personaErrs...)
	}

	if ontologyErrs := validateOntology(m.Ontology, filePath); len(ontologyErrs) > 0 {
		errs = append(errs, ontologyErrs...)
	}

	if hookErrs := validateHooks(m.Hooks, filePath); len(hookErrs) > 0 {
		errs = append(errs, hookErrs...)
	}

	if retroErrs := validateRetros(&m.Runtime.Retros, filePath); len(retroErrs) > 0 {
		errs = append(errs, retroErrs...)
	}

	if fallbackErrs := validateFallbackNames(m.Runtime.Fallbacks, filePath); len(fallbackErrs) > 0 {
		errs = append(errs, fallbackErrs...)
	}

	return errs
}

// validateRetros checks that retros configuration is valid.
func validateRetros(c *RetrosConfig, filePath string) []error {
	if c == nil {
		return nil
	}
	var errs []error
	if c.NarrateModel != "" {
		// Basic validation: model name should be non-empty and reasonable
		if len(c.NarrateModel) > 100 {
			err := &ValidationError{
				Field:      "runtime.retros.narrate_model",
				Reason:     "model name is too long",
				Suggestion: "Use a valid model identifier like 'claude-haiku-4-5'",
			}
			if filePath != "" {
				err.File = filePath
			}
			errs = append(errs, err)
		}
	}
	return errs
}

// validateFallbackNames checks that fallback chain provider names are non-empty.
func validateFallbackNames(fallbacks map[string][]string, filePath string) []error {
	var errs []error
	for provider, chain := range fallbacks {
		if strings.TrimSpace(provider) == "" {
			errs = append(errs, &ValidationError{
				File:       filePath,
				Field:      "runtime.fallbacks",
				Reason:     "provider name must not be empty",
				Suggestion: "Use a provider name like 'anthropic', 'openai', or 'gemini'",
			})
		}
		for i, fallback := range chain {
			if strings.TrimSpace(fallback) == "" {
				errs = append(errs, &ValidationError{
					File:       filePath,
					Field:      fmt.Sprintf("runtime.fallbacks.%s[%d]", provider, i),
					Reason:     "fallback provider name must not be empty",
					Suggestion: "Specify a valid provider name in the fallback chain",
				})
			}
		}
	}
	return errs
}

func validateMetadata(m *Metadata, _ string) *ValidationError {
	if strings.TrimSpace(m.Name) == "" {
		return &ValidationError{
			Field:      "metadata.name",
			Reason:     "is required",
			Suggestion: "Add a 'name' field under 'metadata' to identify your project",
		}
	}
	return nil
}

func validateRuntime(r *Runtime, _ string) *ValidationError {
	if strings.TrimSpace(r.WorkspaceRoot) == "" {
		return &ValidationError{
			Field:      "runtime.workspace_root",
			Reason:     "is required",
			Suggestion: "Set 'workspace_root' to a directory path like '.wave/workspaces'",
		}
	}
	return nil
}

func validateTimeouts(t *Timeouts, filePath string) []error {
	if t == nil {
		return nil
	}
	var errs []error
	fields := []struct {
		name string
		val  int
	}{
		{"step_default_minutes", t.StepDefaultMin},
		{"relay_compaction_minutes", t.RelayCompactionMin},
		{"meta_default_minutes", t.MetaDefaultMin},
		{"skill_install_seconds", t.SkillInstallSec},
		{"skill_cli_seconds", t.SkillCLISec},
		{"skill_http_seconds", t.SkillHTTPSec},
		{"skill_http_header_seconds", t.SkillHTTPHeaderSec},
		{"skill_publish_seconds", t.SkillPublishSec},
		{"process_grace_seconds", t.ProcessGraceSec},
		{"stdout_drain_seconds", t.StdoutDrainSec},
		{"gate_approval_hours", t.GateApprovalHours},
		{"gate_poll_interval_seconds", t.GatePollIntervalSec},
		{"gate_poll_timeout_minutes", t.GatePollTimeoutMin},
		{"git_command_seconds", t.GitCommandSec},
		{"forge_api_seconds", t.ForgeAPISec},
		{"retry_max_delay_seconds", t.RetryMaxDelaySec},
	}
	for _, f := range fields {
		if f.val < 0 {
			errs = append(errs, &ValidationError{
				File:       filePath,
				Field:      "runtime.timeouts." + f.name,
				Reason:     "must not be negative",
				Suggestion: "Use 0 to fall back to the default, or a positive value",
			})
		}
	}
	return errs
}

func validateAdaptersWithFile(adapters map[string]Adapter, _, filePath string) []error {
	var errs []error
	for name, adapter := range adapters {
		if strings.TrimSpace(adapter.Binary) == "" {
			errs = append(errs, &ValidationError{
				File:       filePath,
				Field:      fmt.Sprintf("adapters.%s.binary", name),
				Reason:     "is required",
				Suggestion: "Set 'binary' to the CLI executable name (e.g., 'claude', 'opencode')",
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
			var suggestion string
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

		// Validate token_scopes syntax
		if len(persona.TokenScopes) > 0 {
			if scopeErrs := scope.ValidateScopes(persona.TokenScopes); len(scopeErrs) > 0 {
				for _, scopeErr := range scopeErrs {
					errs = append(errs, &ValidationError{
						File:       filePath,
						Field:      fmt.Sprintf("personas.%s.token_scopes", name),
						Reason:     scopeErr.Error(),
						Suggestion: "Use format '<resource>:<permission>' where resource is one of: issues, pulls, repos, actions, packages and permission is: read, write, admin",
					})
				}
			}
		}
	}
	return errs
}

func validateOntology(o *Ontology, filePath string) []error {
	if o == nil {
		return nil
	}
	var errs []error
	seen := make(map[string]bool)
	for i, ctx := range o.Contexts {
		if strings.TrimSpace(ctx.Name) == "" {
			errs = append(errs, &ValidationError{
				File:       filePath,
				Field:      fmt.Sprintf("ontology.contexts[%d].name", i),
				Reason:     "is required",
				Suggestion: "Each bounded context must have a name",
			})
			continue
		}
		if seen[ctx.Name] {
			errs = append(errs, &ValidationError{
				File:       filePath,
				Field:      fmt.Sprintf("ontology.contexts[%d].name", i),
				Reason:     fmt.Sprintf("duplicate context name %q", ctx.Name),
				Suggestion: "Each bounded context name must be unique",
			})
		}
		seen[ctx.Name] = true
	}
	return errs
}

func Load(path string) (*Manifest, error) {
	return NewLoader().Load(path)
}

// SkillStore is a minimal interface for skill existence checks during manifest validation.
type SkillStore interface {
	Read(name string) (interface{}, error)
}

// LoadWithSkillStore loads a manifest and additionally validates all skill references
// (global and persona scopes) against the provided skill store.
func LoadWithSkillStore(path string, store SkillStore) (*Manifest, error) {
	m, err := Load(path)
	if err != nil {
		return nil, err
	}
	if store == nil {
		return m, nil
	}
	// Validate skill name format and existence against the store
	var validationErrs []string
	for _, name := range m.Skills {
		if err := skill.ValidateName(name); err != nil {
			validationErrs = append(validationErrs, fmt.Sprintf("global: invalid skill name %q: %v", name, err))
			continue
		}
		if _, readErr := store.Read(name); readErr != nil {
			validationErrs = append(validationErrs, fmt.Sprintf("global: skill %q not found in store", name))
		}
	}
	// Sort persona names for deterministic error ordering
	personaNames := make([]string, 0, len(m.Personas))
	for pName := range m.Personas {
		personaNames = append(personaNames, pName)
	}
	sort.Strings(personaNames)
	for _, pName := range personaNames {
		persona := m.Personas[pName]
		for _, name := range persona.Skills {
			if err := skill.ValidateName(name); err != nil {
				validationErrs = append(validationErrs, fmt.Sprintf("persona:%s: invalid skill name %q: %v", pName, name, err))
				continue
			}
			if _, readErr := store.Read(name); readErr != nil {
				validationErrs = append(validationErrs, fmt.Sprintf("persona:%s: skill %q not found in store", pName, name))
			}
		}
	}
	if len(validationErrs) > 0 {
		return nil, fmt.Errorf("skill validation failed: %s", strings.Join(validationErrs, "; "))
	}
	return m, nil
}

func validateHooks(hks []hooks.LifecycleHookDef, filePath string) []error {
	var errs []error
	seenNames := make(map[string]int, len(hks))
	for i, h := range hks {
		prefix := fmt.Sprintf("hooks[%d]", i)
		if strings.TrimSpace(h.Name) == "" {
			errs = append(errs, &ValidationError{
				File:       filePath,
				Field:      prefix + ".name",
				Reason:     "is required",
				Suggestion: "Each hook must have a unique name",
			})
		} else if prev, ok := seenNames[h.Name]; ok {
			errs = append(errs, &ValidationError{
				File:       filePath,
				Field:      prefix + ".name",
				Reason:     fmt.Sprintf("duplicate hook name %q (first defined at hooks[%d])", h.Name, prev),
				Suggestion: "Each hook must have a unique name",
			})
		} else {
			seenNames[h.Name] = i
		}
		if !hooks.ValidEventTypes[h.Event] {
			errs = append(errs, &ValidationError{
				File:       filePath,
				Field:      prefix + ".event",
				Reason:     fmt.Sprintf("invalid event type %q", h.Event),
				Suggestion: "Valid events: run_start, run_completed, run_failed, step_start, step_completed, step_failed, step_retrying, contract_validated, artifact_created, workspace_created",
			})
		}
		if !hooks.ValidHookTypes[h.Type] {
			errs = append(errs, &ValidationError{
				File:       filePath,
				Field:      prefix + ".type",
				Reason:     fmt.Sprintf("invalid hook type %q", h.Type),
				Suggestion: "Valid types: command, http, llm_judge, script",
			})
		}
		// Validate required fields per type
		switch h.Type {
		case hooks.HookTypeCommand:
			if strings.TrimSpace(h.Command) == "" {
				errs = append(errs, &ValidationError{
					File:       filePath,
					Field:      prefix + ".command",
					Reason:     "is required for command hooks",
					Suggestion: "Set 'command' to a shell command to execute",
				})
			}
		case hooks.HookTypeHTTP:
			if strings.TrimSpace(h.URL) == "" {
				errs = append(errs, &ValidationError{
					File:       filePath,
					Field:      prefix + ".url",
					Reason:     "is required for http hooks",
					Suggestion: "Set 'url' to the webhook endpoint URL",
				})
			}
		case hooks.HookTypeLLMJudge:
			if strings.TrimSpace(h.Prompt) == "" {
				errs = append(errs, &ValidationError{
					File:       filePath,
					Field:      prefix + ".prompt",
					Reason:     "is required for llm_judge hooks",
					Suggestion: "Set 'prompt' to the LLM evaluation prompt",
				})
			}
		case hooks.HookTypeScript:
			if strings.TrimSpace(h.Script) == "" {
				errs = append(errs, &ValidationError{
					File:       filePath,
					Field:      prefix + ".script",
					Reason:     "is required for script hooks",
					Suggestion: "Set 'script' to the inline script content",
				})
			}
		}
		// Validate matcher regex compiles
		if h.Matcher != "" {
			if _, err := regexp.Compile(h.Matcher); err != nil {
				errs = append(errs, &ValidationError{
					File:       filePath,
					Field:      prefix + ".matcher",
					Reason:     fmt.Sprintf("invalid regex: %v", err),
					Suggestion: "Use valid regex syntax (e.g., 'implement|fix', '.*')",
				})
			}
		}
	}
	return errs
}

// validateFallbacks checks runtime.fallbacks configuration for consistency.
func validateFallbacks(m *Manifest) []error {
	if len(m.Runtime.Fallbacks) == 0 {
		return nil
	}
	var errs []error
	for adapterName, fallbacks := range m.Runtime.Fallbacks {
		// Key must reference a known adapter
		if _, ok := m.Adapters[adapterName]; !ok {
			errs = append(errs, &ValidationError{
				Field:  "runtime.fallbacks." + adapterName,
				Reason: fmt.Sprintf("adapter %q is not defined in manifest adapters", adapterName),
			})
		}
		seen := make(map[string]bool)
		for _, fb := range fallbacks {
			// No self-reference
			if fb == adapterName {
				errs = append(errs, &ValidationError{
					Field:  "runtime.fallbacks." + adapterName,
					Reason: fmt.Sprintf("adapter %q cannot fall back to itself", adapterName),
				})
			}
			// Must reference known adapter
			if _, ok := m.Adapters[fb]; !ok {
				errs = append(errs, &ValidationError{
					Field:  "runtime.fallbacks." + adapterName,
					Reason: fmt.Sprintf("fallback adapter %q is not defined in manifest adapters", fb),
				})
			}
			// No duplicates
			if seen[fb] {
				errs = append(errs, &ValidationError{
					Field:  "runtime.fallbacks." + adapterName,
					Reason: fmt.Sprintf("duplicate fallback adapter %q", fb),
				})
			}
			seen[fb] = true
		}
	}
	return errs
}
