package pipeline

import (
	"fmt"
	"os"

	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/security"
	"github.com/recinq/wave/internal/skill"
)

// securityLayer owns path validation, input sanitization, schema sanitization,
// and skill reference validation. It is constructed by NewDefaultPipelineExecutor
// and accessed by the coordinator and other layers via DefaultPipelineExecutor.sec.
//
// The back-pointer to DefaultPipelineExecutor exists so the layer can read
// coordinator-owned collaborators (e.g. skillStore) without relying on a wider
// dependency-injection contract. Narrow interfaces are a possible follow-up.
type securityLayer struct {
	e *DefaultPipelineExecutor

	securityConfig *security.SecurityConfig
	pathValidator  *security.PathValidator
	inputSanitizer *security.InputSanitizer
	securityLogger *security.SecurityLogger
}

// newSecurityLayer wires a fresh securityLayer with default config and a logger
// whose output respects the executor's --debug flag.
func newSecurityLayer(e *DefaultPipelineExecutor) *securityLayer {
	cfg := security.DefaultSecurityConfig()
	logger := security.NewSecurityLogger(cfg.LoggingEnabled && e.debug)
	return &securityLayer{
		e:              e,
		securityConfig: cfg,
		pathValidator:  security.NewPathValidator(*cfg, logger),
		inputSanitizer: security.NewInputSanitizer(*cfg, logger),
		securityLogger: logger,
	}
}

// loadSchemaContent securely loads schema content from a path, applying
// path validation and sanitization. Returns the content or an error.
func (s *securityLayer) loadSchemaContent(step *Step, schemaPath string) (string, error) {
	if schemaPath == "" {
		return "", nil
	}
	if s.pathValidator != nil {
		validationResult, pathErr := s.pathValidator.ValidatePath(schemaPath)
		if pathErr != nil {
			if s.securityLogger != nil {
				s.securityLogger.LogViolation(
					string(security.ViolationPathTraversal),
					string(security.SourceSchemaPath),
					fmt.Sprintf("Schema path validation failed: %s", schemaPath),
					security.SeverityCritical,
					true,
				)
			}
			return "", fmt.Errorf("schema path validation failed: %w", pathErr)
		}
		if !validationResult.IsValid {
			return "", fmt.Errorf("schema path rejected by validator")
		}
		data, readErr := os.ReadFile(validationResult.ValidatedPath)
		if readErr != nil {
			return "", fmt.Errorf("read schema: %w", readErr)
		}
		sanitized, sanitizeErr := s.sanitizeSchemaContent(step, string(data))
		if sanitizeErr != nil {
			return "", sanitizeErr
		}
		return sanitized, nil
	}
	// No path validator (e.g. in tests) — read directly
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return "", fmt.Errorf("read schema: %w", err)
	}
	return string(data), nil
}

// loadSecureSchemaContent loads schema content for a step's handover contract,
// honoring either SchemaPath (preferred) or inline Schema. Returns an empty
// string with nil error when neither is specified, a wrapped error when
// loading/sanitization fails, and the sanitized content otherwise.
func (s *securityLayer) loadSecureSchemaContent(step *Step) (string, error) {
	if step.Handover.Contract.SchemaPath != "" {
		return s.loadSchemaContent(step, step.Handover.Contract.SchemaPath)
	}

	if step.Handover.Contract.Schema != "" {
		return s.sanitizeSchemaContent(step, step.Handover.Contract.Schema)
	}

	return "", nil
}

// sanitizeSchemaContent applies prompt injection sanitization to schema content.
// Returns the sanitized content, or a wrapped error if sanitization fails.
func (s *securityLayer) sanitizeSchemaContent(step *Step, content string) (string, error) {
	if s.inputSanitizer == nil {
		return content, nil
	}
	sanitized, sanitizationActions, err := s.inputSanitizer.SanitizeSchemaContent(content)
	if err != nil {
		stepLabel := "unknown"
		if step != nil {
			stepLabel = step.ID
		}
		if s.securityLogger != nil {
			s.securityLogger.LogViolation(
				string(security.ViolationInputValidation),
				string(security.SourceSchemaPath),
				fmt.Sprintf("Schema content sanitization failed for step %s", stepLabel),
				security.SeverityHigh,
				true,
			)
		}
		return "", fmt.Errorf("schema content sanitization failed")
	}
	if len(sanitizationActions) > 0 {
		stepLabel := "unknown"
		if step != nil {
			stepLabel = step.ID
		}
		if s.securityLogger != nil {
			s.securityLogger.LogViolation(
				string(security.ViolationPromptInjection),
				string(security.SourceSchemaPath),
				fmt.Sprintf("Schema content sanitized for step %s: %v", stepLabel, sanitizationActions),
				security.SeverityMedium,
				false,
			)
		}
	}
	return sanitized, nil
}

// validateSkillRefs validates skill references at manifest (global + persona)
// and pipeline scope against the configured skill store.
func (s *securityLayer) validateSkillRefs(pipelineSkills []string, pipelineName string, m *manifest.Manifest) []error {
	if s.e.skillStore == nil {
		return nil
	}

	// Validate manifest-level skills (global + persona scopes)
	var personas []skill.PersonaSkills
	for name, persona := range m.Personas {
		if len(persona.Skills) > 0 {
			personas = append(personas, skill.PersonaSkills{Name: name, Skills: persona.Skills})
		}
	}
	errs := skill.ValidateManifestSkills(m.Skills, personas, s.e.skillStore)

	// Validate pipeline-level skills (already template-resolved by caller)
	if len(pipelineSkills) > 0 {
		errs = append(errs, skill.ValidateSkillRefs(pipelineSkills, "pipeline:"+pipelineName, s.e.skillStore)...)
	}

	return errs
}

// schemaFieldPlaceholder returns a JSON placeholder value for a schema property,
// used in the contract compliance example skeleton.
func schemaFieldPlaceholder(_ string, prop map[string]any) string {
	if prop == nil {
		return "\"...\""
	}
	t, _ := prop["type"].(string)
	switch t {
	case "string":
		return "\"...\""
	case "integer", "number":
		return "0"
	case "boolean":
		return "false"
	case "array":
		return "[...]"
	case "object":
		return "{...}"
	default:
		return "\"...\""
	}
}
