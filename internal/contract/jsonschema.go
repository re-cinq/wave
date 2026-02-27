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
// while preserving the integrity of string values
func cleanJSON(data []byte) ([]byte, []string, error) {
	content := string(data)
	changes := []string{}

	// First, try to parse as-is to see if it's already valid
	var test interface{}
	if err := json.Unmarshal([]byte(content), &test); err == nil {
		// Already valid JSON, no cleaning needed
		return data, changes, nil
	}

	// Remove single-line comments (// comment) - only outside of strings
	// This is a simplified approach; full comment removal would need proper parsing
	singleLineCommentRegex := regexp.MustCompile(`//[^\n]*`)
	if singleLineCommentRegex.MatchString(content) {
		content = singleLineCommentRegex.ReplaceAllString(content, "")
		changes = append(changes, "removed_single_line_comments")
	}

	// Remove multi-line comments (/* comment */)
	multiLineCommentRegex := regexp.MustCompile(`(?s)/\*.*?\*/`)
	if multiLineCommentRegex.MatchString(content) {
		content = multiLineCommentRegex.ReplaceAllString(content, "")
		changes = append(changes, "removed_multi_line_comments")
	}

	// Remove hash comments (# comment) - only at line start or after whitespace
	hashCommentRegex := regexp.MustCompile(`(?m)^\s*#.*$`)
	if hashCommentRegex.MatchString(content) {
		content = hashCommentRegex.ReplaceAllString(content, "")
		changes = append(changes, "removed_hash_comments")
	}

	// Remove trailing commas before } or ] - this is safe as JSON doesn't allow trailing commas
	trailingCommaRegex := regexp.MustCompile(`,(\s*[}\]])`)
	if trailingCommaRegex.MatchString(content) {
		content = trailingCommaRegex.ReplaceAllString(content, "$1")
		changes = append(changes, "removed_trailing_commas")
	}

	// Clean up excessive whitespace but preserve necessary structure
	// Only collapse multiple spaces/tabs on a line, preserve newlines in multiline content
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		// Collapse multiple spaces/tabs to single space
		line = regexp.MustCompile(`[ \t]+`).ReplaceAllString(line, " ")
		lines[i] = strings.TrimSpace(line)
	}
	content = strings.Join(lines, "\n")
	content = strings.TrimSpace(content)

	// Validate that the cleaned JSON is parseable
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
	artifactFile := ".wave/artifact.json"
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

	// NEW: Error wrapper detection and extraction
	// Detection is enabled by default, disabled only if explicitly configured
	if !cfg.DisableWrapperDetection {

		wrapperResult, wrapperErr := DetectErrorWrapper(data)
		if wrapperErr != nil {
			// Wrapper detection failed, but continue with original data
			// This ensures backward compatibility
			if cfg.DebugMode {
				fmt.Printf("[DEBUG] Wrapper detection failed: %v\n", wrapperErr)
			}
		} else if wrapperResult.IsWrapper {
			// Extract raw content from wrapper for validation
			originalDataLength := len(data)
			data = wrapperResult.RawContent

			// Log wrapper extraction for debugging if needed
			// Note: In production this should use proper logging infrastructure
			if cfg.DebugMode {
				debug := wrapperResult.GetDebugInfo(originalDataLength)
				fmt.Printf("[DEBUG] Error wrapper detected and extracted: %+v\n", debug)
			}
		} else if cfg.DebugMode {
			debug := wrapperResult.GetDebugInfo(len(data))
			fmt.Printf("[DEBUG] No error wrapper detected: %+v\n", debug)
		}
	}

	// Enhanced JSON recovery with progressive validation
	// Default to allowing recovery for better reliability
	allowRecovery := cfg.AllowRecovery
	if !cfg.AllowRecovery && cfg.RecoveryLevel == "" {
		// If recovery settings are not explicitly configured, enable progressive recovery by default
		allowRecovery = true
	}

	var artifact interface{}
	var recoveryResult *RecoveryResult

	if allowRecovery {
		recoveryLevel := determineRecoveryLevel(cfg)
		// Default to progressive recovery for better AI compatibility
		if recoveryLevel == ConservativeRecovery && cfg.RecoveryLevel == "" {
			recoveryLevel = ProgressiveRecovery
		}

		recoveryParser := NewJSONRecoveryParser(recoveryLevel)

		var recoveryErr error
		recoveryResult, recoveryErr = recoveryParser.ParseWithRecovery(string(data))
		if recoveryErr != nil || !recoveryResult.IsValid {
			details := []string{fmt.Sprintf("file: %s", artifactPath)}
			if recoveryErr != nil {
				details = append(details, recoveryErr.Error())
			}
			if recoveryResult != nil {
				if len(recoveryResult.AppliedFixes) > 0 {
					details = append(details, fmt.Sprintf("JSON Recovery Applied: %v", recoveryResult.AppliedFixes))
				}
				if len(recoveryResult.Warnings) > 0 {
					details = append(details, fmt.Sprintf("Warnings: %v", recoveryResult.Warnings))
				}
			}

			return &ValidationError{
				ContractType: "json_schema",
				Message:      "failed to parse artifact JSON after recovery attempts",
				Details:      details,
				Retryable:    true,
			}
		}

		// Use the recovered and parsed data
		artifact = recoveryResult.ParsedData
	} else {
		// No recovery - try to parse as-is
		if err := json.Unmarshal(data, &artifact); err != nil {
			return &ValidationError{
				ContractType: "json_schema",
				Message:      "failed to parse artifact JSON",
				Details:      []string{fmt.Sprintf("file: %s", artifactPath), err.Error()},
				Retryable:    true,
			}
		}

		// Create a simple recovery result for consistent error formatting
		recoveryResult = &RecoveryResult{
			OriginalInput:  string(data),
			RecoveredJSON:  string(data),
			IsValid:        true,
			AppliedFixes:   []string{},
			Warnings:       []string{},
			RecoveryLevel:  ConservativeRecovery,
			ParsedData:     artifact,
		}
	}

	if err := schema.Validate(artifact); err != nil {
		// Use enhanced error formatting for better user experience
		formatter := &ValidationErrorFormatter{}
		validationErr := formatter.FormatJSONSchemaError(err, recoveryResult, artifactPath)

		// Apply progressive validation logic
		mustPass := cfg.MustPass

		// Check if progressive validation is enabled
		if cfg.ProgressiveValidation && !mustPass {
			// In progressive mode, convert to warnings instead of blocking errors
			validationErr.Message = validationErr.Message + " (progressive validation: warning only)"
			validationErr.Retryable = false // Don't retry warnings

			// TODO: In a real implementation, these warnings would be logged
			// to the audit system rather than blocking the pipeline
			_ = formatter.FormatProgressiveValidationWarning(err, recoveryResult)
		} else {
			// Normal validation mode - respect must_pass setting
			validationErr.Retryable = mustPass
		}

		// Add context about validation mode
		if !mustPass && !cfg.ProgressiveValidation {
			validationErr.Message = validationErr.Message + " (must_pass: false)"
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

// determineRecoveryLevel determines the appropriate JSON recovery level based on contract configuration
func determineRecoveryLevel(cfg ContractConfig) RecoveryLevel {
	// If recovery is explicitly disabled, return conservative
	if !cfg.AllowRecovery {
		return ConservativeRecovery
	}

	// Check for explicit recovery level configuration
	switch cfg.RecoveryLevel {
	case "conservative":
		return ConservativeRecovery
	case "progressive":
		return ProgressiveRecovery
	case "aggressive":
		return AggressiveRecovery
	}

	// Progressive validation logic:
	// - If MustPass is true, use conservative recovery (maintain strict validation)
	// - If progressive validation is enabled, use progressive recovery
	// - If MustPass is false, use progressive recovery
	// - Default to conservative recovery for safety

	if cfg.MustPass {
		return ConservativeRecovery
	}

	// If progressive validation is enabled, use progressive recovery
	if cfg.ProgressiveValidation {
		return ProgressiveRecovery
	}

	// If must_pass is false, allow more progressive recovery
	if !cfg.MustPass {
		return ProgressiveRecovery
	}

	// Default to conservative recovery for safety
	return ConservativeRecovery
}
