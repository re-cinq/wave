package contract

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// InputValidationResult represents the result of validating input artifacts.
type InputValidationResult struct {
	Passed      bool   // Overall validation result
	ArtifactRef string // The artifact reference being validated
	ArtifactAs  string // The 'as' name of the artifact
	Error       error  // Validation error if failed
	TypeMatch   bool   // Whether declared type matched
	SchemaValid bool   // Whether schema validation passed (if applicable)
}

// InputArtifactConfig holds configuration for validating an input artifact.
type InputArtifactConfig struct {
	Name          string // The artifact name (as mounted)
	SchemaContent string // Pre-loaded JSON schema content (empty = skip validation)
	Type          string // Expected artifact type
	Path          string // Path to the artifact file
}

// ValidateInputArtifacts validates all input artifacts against their schemas.
// Returns a list of validation results and an error if any required validations failed.
func ValidateInputArtifacts(configs []InputArtifactConfig, workspacePath string) ([]InputValidationResult, error) {
	results := make([]InputValidationResult, 0, len(configs))

	for _, cfg := range configs {
		result := validateSingleInputArtifact(cfg, workspacePath)
		results = append(results, result)

		// Fail fast on validation errors
		if !result.Passed && result.Error != nil {
			return results, result.Error
		}
	}

	return results, nil
}

// validateSingleInputArtifact validates a single input artifact.
func validateSingleInputArtifact(cfg InputArtifactConfig, workspacePath string) InputValidationResult {
	result := InputValidationResult{
		ArtifactRef: cfg.Name,
		ArtifactAs:  cfg.Name,
		Passed:      true,
		TypeMatch:   true,
		SchemaValid: true,
	}

	// If no schema content is specified, skip validation
	if cfg.SchemaContent == "" {
		return result
	}

	// Resolve artifact path
	artifactPath := cfg.Path
	if artifactPath == "" {
		artifactPath = filepath.Join(workspacePath, ".agents", "artifacts", cfg.Name)
	}

	// Read artifact content
	artifactData, err := os.ReadFile(artifactPath)
	if err != nil {
		result.Passed = false
		result.Error = fmt.Errorf("failed to read input artifact '%s': %w", cfg.Name, err)
		return result
	}

	// Parse schema
	var schemaDoc interface{}
	if err := json.Unmarshal([]byte(cfg.SchemaContent), &schemaDoc); err != nil {
		result.Passed = false
		result.Error = fmt.Errorf("failed to parse schema content: %w", err)
		return result
	}

	// Compile schema
	compiler := jsonschema.NewCompiler()
	schemaURI := "input-schema://" + cfg.Name
	if err := compiler.AddResource(schemaURI, schemaDoc); err != nil {
		result.Passed = false
		result.Error = fmt.Errorf("failed to compile schema for artifact '%s': %w", cfg.Name, err)
		return result
	}

	schema, err := compiler.Compile(schemaURI)
	if err != nil {
		result.Passed = false
		result.Error = fmt.Errorf("failed to compile schema for artifact '%s': %w", cfg.Name, err)
		return result
	}

	// Parse artifact content
	var artifact interface{}
	if err := json.Unmarshal(artifactData, &artifact); err != nil {
		result.Passed = false
		result.SchemaValid = false
		result.Error = &ValidationError{
			ContractType: "input_schema",
			Message:      fmt.Sprintf("input artifact '%s' contains invalid JSON", cfg.Name),
			Details:      []string{err.Error()},
			Retryable:    false,
		}
		return result
	}

	// Validate against schema
	if err := schema.Validate(artifact); err != nil {
		result.Passed = false
		result.SchemaValid = false
		result.Error = &ValidationError{
			ContractType: "input_schema",
			Message:      fmt.Sprintf("input artifact '%s' failed schema validation", cfg.Name),
			Details:      extractSchemaValidationDetails(err),
			Retryable:    false,
		}
		return result
	}

	return result
}

// ValidateInputArtifactContent validates a single input artifact against
// pre-loaded schema content. The caller is responsible for loading the schema
// (with path traversal protection and sanitization) before calling this.
func ValidateInputArtifactContent(name string, schemaContent string, artifactPath string) error {
	cfg := InputArtifactConfig{
		Name:          name,
		SchemaContent: schemaContent,
		Path:          artifactPath,
	}

	result := validateSingleInputArtifact(cfg, filepath.Dir(artifactPath))
	if !result.Passed {
		return result.Error
	}
	return nil
}
