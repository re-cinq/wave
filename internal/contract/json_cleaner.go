package contract

import (
	"encoding/json"
	"fmt"
	"strings"
)

// JSONCleaner provides utilities for fixing and validating JSON output
type JSONCleaner struct{}

// CleanJSONOutput attempts to fix common JSON formatting issues while preserving content
// This is useful for AI-generated JSON that may have minor syntax errors
// This is now a wrapper around the more sophisticated JSONRecoveryParser
func (jc *JSONCleaner) CleanJSONOutput(input string) (string, []string, error) {
	// Use the progressive recovery parser for comprehensive cleaning
	parser := NewJSONRecoveryParser(ProgressiveRecovery)
	result, err := parser.ParseWithRecovery(input)

	if err != nil || !result.IsValid {
		// If progressive recovery fails, try conservative
		conservativeParser := NewJSONRecoveryParser(ConservativeRecovery)
		conservativeResult, conservativeErr := conservativeParser.ParseWithRecovery(input)

		if conservativeErr == nil && conservativeResult.IsValid {
			return conservativeResult.RecoveredJSON, conservativeResult.AppliedFixes, nil
		}

		// If both fail, return the original progressive error
		if err != nil {
			return input, result.AppliedFixes, err
		}
		return input, result.AppliedFixes, fmt.Errorf("JSON recovery failed: %s", strings.Join(result.Warnings, ", "))
	}

	return result.RecoveredJSON, result.AppliedFixes, nil
}

// ValidateAndFormatJSON validates JSON and returns it in canonical format
func (jc *JSONCleaner) ValidateAndFormatJSON(input string) (string, error) {
	var data interface{}
	if err := json.Unmarshal([]byte(input), &data); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	// Re-marshal to canonical form with proper indentation
	formatted, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}

	return string(formatted), nil
}

// IsValidJSON checks if the input is valid JSON
func (jc *JSONCleaner) IsValidJSON(input string) bool {
	var test interface{}
	return json.Unmarshal([]byte(input), &test) == nil
}

// ExtractJSONFromText attempts to extract a JSON object or array from text
// Useful when AI output includes explanation text before/after JSON
func (jc *JSONCleaner) ExtractJSONFromText(text string) (string, error) {
	// Use the recovery parser's more sophisticated extraction
	parser := NewJSONRecoveryParser(ProgressiveRecovery)
	result, err := parser.ParseWithRecovery(text)

	if err == nil && result.IsValid {
		// Check if extraction was performed
		for _, fix := range result.AppliedFixes {
			if strings.Contains(fix, "extracted") {
				return result.RecoveredJSON, nil
			}
		}
		// If no extraction was needed, the input was already valid JSON
		return result.RecoveredJSON, nil
	}

	// Recovery parser failed — return the error directly
	return "", err
}

// NormalizeJSONFormat takes valid JSON and normalizes its formatting for consistency
func (jc *JSONCleaner) NormalizeJSONFormat(input string, indent string) (string, error) {
	var data interface{}
	if err := json.Unmarshal([]byte(input), &data); err != nil {
		return "", fmt.Errorf("input is not valid JSON: %w", err)
	}

	// Re-marshal with consistent formatting
	var formatted []byte
	var err error

	if indent == "" {
		formatted, err = json.Marshal(data)
	} else {
		formatted, err = json.MarshalIndent(data, "", indent)
	}

	if err != nil {
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}

	return string(formatted), nil
}

// CleanAndNormalizeJSON combines cleaning and normalization in one step
func (jc *JSONCleaner) CleanAndNormalizeJSON(input string, indent string) (string, []string, error) {
	// First clean the JSON
	cleaned, changes, err := jc.CleanJSONOutput(input)
	if err != nil {
		return input, changes, err
	}

	// Then normalize the format
	normalized, err := jc.NormalizeJSONFormat(cleaned, indent)
	if err != nil {
		// Return cleaned version even if normalization fails
		return cleaned, changes, nil
	}

	return normalized, changes, nil
}

// ValidateJSONStructure performs structural validation beyond basic JSON parsing
func (jc *JSONCleaner) ValidateJSONStructure(input string, requirements map[string]interface{}) ([]string, error) {
	var data interface{}
	if err := json.Unmarshal([]byte(input), &data); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	issues := []string{}

	// Convert to map for inspection if it's an object
	if obj, ok := data.(map[string]interface{}); ok {
		issues = append(issues, jc.validateObjectStructure(obj, requirements)...)
	}

	return issues, nil
}

// validateObjectStructure checks object-level requirements
func (jc *JSONCleaner) validateObjectStructure(obj map[string]interface{}, requirements map[string]interface{}) []string {
	issues := []string{}

	// Check for required fields
	if requiredFields, ok := requirements["required_fields"].([]string); ok {
		for _, field := range requiredFields {
			if _, exists := obj[field]; !exists {
				issues = append(issues, fmt.Sprintf("missing required field: %s", field))
			}
		}
	}

	// Check for forbidden fields
	if forbiddenFields, ok := requirements["forbidden_fields"].([]string); ok {
		for _, field := range forbiddenFields {
			if _, exists := obj[field]; exists {
				issues = append(issues, fmt.Sprintf("forbidden field found: %s", field))
			}
		}
	}

	// Check minimum number of fields
	if minFields, ok := requirements["min_fields"].(int); ok {
		if len(obj) < minFields {
			issues = append(issues, fmt.Sprintf("object has %d fields, minimum required: %d", len(obj), minFields))
		}
	}

	// Check for empty string values
	if checkEmpty, ok := requirements["no_empty_strings"].(bool); ok && checkEmpty {
		for key, value := range obj {
			if str, ok := value.(string); ok && strings.TrimSpace(str) == "" {
				issues = append(issues, fmt.Sprintf("field '%s' has empty string value", key))
			}
		}
	}

	return issues
}
