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
	Name       string // The artifact name (as mounted)
	SchemaPath string // Path to JSON schema file for validation
	Type       string // Expected artifact type
	Path       string // Path to the artifact file
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

	// If no schema path is specified, skip validation
	if cfg.SchemaPath == "" {
		return result
	}

	// Resolve artifact path
	artifactPath := cfg.Path
	if artifactPath == "" {
		artifactPath = filepath.Join(workspacePath, ".wave", "artifacts", cfg.Name)
	}

	// Read artifact content
	artifactData, err := os.ReadFile(artifactPath)
	if err != nil {
		result.Passed = false
		result.Error = fmt.Errorf("failed to read input artifact '%s': %w", cfg.Name, err)
		return result
	}

	// Read schema
	schemaPath := cfg.SchemaPath
	if !filepath.IsAbs(schemaPath) {
		schemaPath = filepath.Join(workspacePath, schemaPath)
	}

	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		result.Passed = false
		result.Error = fmt.Errorf("failed to read schema file '%s' for artifact '%s': %w", cfg.SchemaPath, cfg.Name, err)
		return result
	}

	// Parse schema
	var schemaDoc interface{}
	if err := json.Unmarshal(schemaData, &schemaDoc); err != nil {
		result.Passed = false
		result.Error = fmt.Errorf("failed to parse schema file '%s': %w", cfg.SchemaPath, err)
		return result
	}

	// Compile schema
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(cfg.SchemaPath, schemaDoc); err != nil {
		result.Passed = false
		result.Error = fmt.Errorf("failed to add schema resource '%s': %w", cfg.SchemaPath, err)
		return result
	}

	schema, err := compiler.Compile(cfg.SchemaPath)
	if err != nil {
		result.Passed = false
		result.Error = fmt.Errorf("failed to compile schema '%s': %w", cfg.SchemaPath, err)
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

// ValidateInputArtifact validates a single input artifact against its schema.
// This is a convenience function for validating one artifact at a time.
func ValidateInputArtifact(name string, schemaPath string, workspacePath string) error {
	cfg := InputArtifactConfig{
		Name:       name,
		SchemaPath: schemaPath,
	}

	result := validateSingleInputArtifact(cfg, workspacePath)
	if !result.Passed {
		return result.Error
	}
	return nil
}
