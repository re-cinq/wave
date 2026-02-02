package contract

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

type jsonSchemaValidator struct{}

// cleanJSON removes comments and fixes common JSON formatting issues
func cleanJSON(data []byte) ([]byte, []string, error) {
	content := string(data)
	changes := []string{}

	// Remove single-line comments (// comment)
	singleLineCommentRegex := regexp.MustCompile(`//.*`)
	if singleLineCommentRegex.MatchString(content) {
		content = singleLineCommentRegex.ReplaceAllString(content, "")
		changes = append(changes, "removed_single_line_comments")
	}

	// Remove multi-line comments (/* comment */)
	multiLineCommentRegex := regexp.MustCompile(`/\*[^*]*\*+(?:[^/*][^*]*\*+)*/`)
	if multiLineCommentRegex.MatchString(content) {
		content = multiLineCommentRegex.ReplaceAllString(content, "")
		changes = append(changes, "removed_multi_line_comments")
	}

	// Remove hash comments (# comment) - sometimes used incorrectly in JSON
	hashCommentRegex := regexp.MustCompile(`#.*`)
	if hashCommentRegex.MatchString(content) {
		content = hashCommentRegex.ReplaceAllString(content, "")
		changes = append(changes, "removed_hash_comments")
	}

	// Remove trailing commas before } or ]
	trailingCommaRegex := regexp.MustCompile(`,(\s*[}\]])`)
	if trailingCommaRegex.MatchString(content) {
		content = trailingCommaRegex.ReplaceAllString(content, "$1")
		changes = append(changes, "removed_trailing_commas")
	}

	// Clean up multiple whitespace characters
	content = regexp.MustCompile(`\s+`).ReplaceAllString(content, " ")
	content = strings.TrimSpace(content)

	// Validate that the cleaned JSON is parseable
	var test interface{}
	if err := json.Unmarshal([]byte(content), &test); err != nil {
		return data, changes, fmt.Errorf("cleaned JSON is still invalid: %w", err)
	}

	return []byte(content), changes, nil
}

func (v *jsonSchemaValidator) Validate(cfg ContractConfig, workspacePath string) error {
	compiler := jsonschema.NewCompiler()
	schemaURL := "schema.json"

	if cfg.Schema != "" {
		var schemaDoc interface{}
		if err := json.Unmarshal([]byte(cfg.Schema), &schemaDoc); err != nil {
			return &ValidationError{
				ContractType: "json_schema",
				Message:      "failed to parse inline schema",
				Details:      []string{err.Error()},
				Retryable:    false,
			}
		}
		if err := compiler.AddResource(schemaURL, schemaDoc); err != nil {
			return &ValidationError{
				ContractType: "json_schema",
				Message:      "failed to add schema resource",
				Details:      []string{err.Error()},
				Retryable:    false,
			}
		}
	} else if cfg.SchemaPath != "" {
		data, err := os.ReadFile(cfg.SchemaPath)
		if err != nil {
			return &ValidationError{
				ContractType: "json_schema",
				Message:      fmt.Sprintf("failed to read schema file: %s", cfg.SchemaPath),
				Details:      []string{err.Error()},
				Retryable:    false,
			}
		}
		var schemaDoc interface{}
		if err := json.Unmarshal(data, &schemaDoc); err != nil {
			return &ValidationError{
				ContractType: "json_schema",
				Message:      fmt.Sprintf("failed to parse schema file: %s", cfg.SchemaPath),
				Details:      []string{err.Error()},
				Retryable:    false,
			}
		}
		if err := compiler.AddResource(cfg.SchemaPath, schemaDoc); err != nil {
			return &ValidationError{
				ContractType: "json_schema",
				Message:      "failed to add schema resource",
				Details:      []string{err.Error()},
				Retryable:    false,
			}
		}
		schemaURL = cfg.SchemaPath
	} else {
		return &ValidationError{
			ContractType: "json_schema",
			Message:      "no schema or schemaPath provided",
			Details:      []string{"specify either 'schema' (inline JSON) or 'schemaPath' (file path)"},
			Retryable:    false,
		}
	}

	schema, err := compiler.Compile(schemaURL)
	if err != nil {
		return &ValidationError{
			ContractType: "json_schema",
			Message:      "failed to compile schema",
			Details:      []string{err.Error()},
			Retryable:    false,
		}
	}

	// Use source path if provided, otherwise default to artifact.json
	artifactFile := "artifact.json"
	if cfg.Source != "" {
		artifactFile = cfg.Source
	}
	artifactPath := filepath.Join(workspacePath, artifactFile)
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		return &ValidationError{
			ContractType: "json_schema",
			Message:      fmt.Sprintf("failed to read artifact file: %s", artifactPath),
			Details:      []string{err.Error()},
			Retryable:    false,
		}
	}

	// SECURITY FIX: Clean JSON to handle comments and formatting issues
	cleanedData, cleaningChanges, cleanErr := cleanJSON(data)
	if cleanErr != nil {
		return &ValidationError{
			ContractType: "json_schema",
			Message:      "failed to clean malformed JSON",
			Details:      []string{cleanErr.Error(), fmt.Sprintf("file: %s", artifactPath)},
			Retryable:    true,
		}
	}

	// Log JSON cleaning if changes were made
	if len(cleaningChanges) > 0 {
		// In a production system, this would integrate with the security logger
		// For now, we include the information in validation details if validation fails
	}

	var artifact interface{}
	if err := json.Unmarshal(cleanedData, &artifact); err != nil {
		details := []string{err.Error(), fmt.Sprintf("file: %s", artifactPath)}
		if len(cleaningChanges) > 0 {
			details = append(details, fmt.Sprintf("JSON cleaning applied: %v", cleaningChanges))
		}
		return &ValidationError{
			ContractType: "json_schema",
			Message:      "failed to parse artifact JSON",
			Details:      details,
			Retryable:    true,
		}
	}

	if err := schema.Validate(artifact); err != nil {
		details := extractSchemaValidationDetails(err)

		// Add JSON cleaning information to validation details if cleaning occurred
		if len(cleaningChanges) > 0 {
			details = append(details, fmt.Sprintf("Note: JSON was automatically cleaned (%v)", cleaningChanges))
		}

		// SECURITY FIX: Respect must_pass setting with proper fallback logic
		// Prioritize MustPass when explicitly configured, only fallback to StrictMode when MustPass is not set
		mustPass := cfg.MustPass
		if !cfg.MustPass && cfg.StrictMode {
			// Only use StrictMode as fallback when MustPass is explicitly false/unset
			mustPass = cfg.StrictMode
		}

		validationErr := &ValidationError{
			ContractType: "json_schema",
			Message:      "artifact does not match schema",
			Details:      details,
			Retryable:    mustPass, // Only retry when must_pass is true
		}

		// Add context about must_pass mode to the message
		if !mustPass {
			validationErr.Message = "artifact does not match schema (must_pass: false)"
		}

		return validationErr
	}

	return nil
}

// extractSchemaValidationDetails extracts detailed validation errors from the schema validator.
func extractSchemaValidationDetails(err error) []string {
	errStr := err.Error()
	// Split multi-line error messages into separate details
	lines := strings.Split(errStr, "\n")
	details := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			details = append(details, line)
		}
	}
	if len(details) == 0 {
		details = append(details, errStr)
	}
	return details
}
