package contract

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidationErrorFormatter provides enhanced error messages with actionable guidance
type ValidationErrorFormatter struct{}

// FormatJSONSchemaError creates detailed, actionable error messages for JSON schema validation failures
func (f *ValidationErrorFormatter) FormatJSONSchemaError(err error, recoveryResult *RecoveryResult, artifactPath string) *ValidationError {
	baseDetails := extractSchemaValidationDetails(err)

	// Analyze the error to provide specific guidance
	analysis := f.analyzeSchemaError(err.Error(), recoveryResult)

	// Build comprehensive error details
	details := []string{}

	// Add file context
	details = append(details, fmt.Sprintf("File: %s", artifactPath))

	// Add recovery information if applicable
	if recoveryResult != nil && len(recoveryResult.AppliedFixes) > 0 {
		details = append(details, fmt.Sprintf("JSON Recovery Applied: %s", strings.Join(recoveryResult.AppliedFixes, ", ")))
	}

	// Add the original error details
	details = append(details, "Schema Validation Errors:")
	for _, detail := range baseDetails {
		details = append(details, fmt.Sprintf("  • %s", detail))
	}

	// Add specific guidance based on error analysis
	if len(analysis.Suggestions) > 0 {
		details = append(details, "Suggested Fixes:")
		for i, suggestion := range analysis.Suggestions {
			details = append(details, fmt.Sprintf("  %d. %s", i+1, suggestion))
		}
	}

	// Add common pitfalls section
	if len(analysis.CommonPitfalls) > 0 {
		details = append(details, "Common Issues to Check:")
		for _, pitfall := range analysis.CommonPitfalls {
			details = append(details, fmt.Sprintf("  ⚠ %s", pitfall))
		}
	}

	// Add examples if available
	if analysis.Example != "" {
		details = append(details, "Example Fix:")
		details = append(details, analysis.Example)
	}

	// Add recovery warnings if present
	if recoveryResult != nil && len(recoveryResult.Warnings) > 0 {
		details = append(details, "Recovery Warnings:")
		for _, warning := range recoveryResult.Warnings {
			details = append(details, fmt.Sprintf("  ⚠ %s", warning))
		}
	}

	return &ValidationError{
		ContractType: "json_schema",
		Message:      analysis.MainMessage,
		Details:      details,
		Retryable:    analysis.Retryable,
	}
}

// SchemaErrorAnalysis contains analysis of a schema validation error
type SchemaErrorAnalysis struct {
	MainMessage     string
	ErrorType       string
	Suggestions     []string
	CommonPitfalls  []string
	Example         string
	Retryable       bool
}

// analyzeSchemaError analyzes the error message and provides specific guidance
func (f *ValidationErrorFormatter) analyzeSchemaError(errorMsg string, recoveryResult *RecoveryResult) SchemaErrorAnalysis {
	errorMsgLower := strings.ToLower(errorMsg)

	analysis := SchemaErrorAnalysis{
		MainMessage:    "JSON schema validation failed",
		Suggestions:    []string{},
		CommonPitfalls: []string{},
		Retryable:      true,
	}

	// Analyze for missing required fields
	if strings.Contains(errorMsgLower, "required") || strings.Contains(errorMsgLower, "missing property") {
		analysis.ErrorType = "missing_required_fields"
		analysis.MainMessage = "Required fields are missing from the JSON output"
		analysis.Suggestions = []string{
			"Check the schema to identify all required fields",
			"Ensure all mandatory properties are included in the output",
			"Verify that field names match the schema exactly (case-sensitive)",
		}
		analysis.CommonPitfalls = []string{
			"Field names with typos or incorrect casing",
			"Fields with null values when null is not allowed",
			"Missing nested required properties",
		}
		analysis.Example = `Example:
Required: {"name": "string", "type": "string", "required": true}
Invalid: {"name": "example"}  // missing 'type' and 'required'
Valid:   {"name": "example", "type": "feature", "required": true}`
	}

	// Analyze for type mismatches
	if (strings.Contains(errorMsgLower, "got") && strings.Contains(errorMsgLower, "want")) ||
	   (strings.Contains(errorMsgLower, "type") && (strings.Contains(errorMsgLower, "expected") || strings.Contains(errorMsgLower, "invalid"))) {
		analysis.ErrorType = "type_mismatch"
		analysis.MainMessage = "Field types don't match the schema requirements"
		analysis.Suggestions = []string{
			"Check that string values are quoted",
			"Ensure numbers are not quoted",
			"Verify boolean values are true/false (not quoted)",
			"Confirm array fields are in [...] brackets",
			"Verify object fields are in {...} braces",
		}
		analysis.CommonPitfalls = []string{
			"Numbers as strings: \"123\" instead of 123",
			"Booleans as strings: \"true\" instead of true",
			"Arrays as strings instead of proper JSON arrays",
			"Objects as strings instead of proper JSON objects",
		}
		analysis.Example = `Example:
Schema expects: {"count": number, "enabled": boolean}
Invalid: {"count": "5", "enabled": "true"}
Valid:   {"count": 5, "enabled": true}`
	}

	// Analyze for enum violations
	if strings.Contains(errorMsgLower, "enum") || strings.Contains(errorMsgLower, "not one of") {
		analysis.ErrorType = "enum_violation"
		analysis.MainMessage = "Field value is not in the allowed list of options"
		analysis.Suggestions = []string{
			"Check the schema for the exact allowed values (enum)",
			"Ensure the value matches exactly (case-sensitive)",
			"Remove any extra whitespace from the value",
		}
		analysis.CommonPitfalls = []string{
			"Case sensitivity: 'Bug' vs 'bug'",
			"Extra whitespace: ' bug ' vs 'bug'",
			"Using similar but not exact values",
		}
	}

	// Analyze for additional properties
	if strings.Contains(errorMsgLower, "additional") || strings.Contains(errorMsgLower, "not allowed") {
		analysis.ErrorType = "additional_properties"
		analysis.MainMessage = "Extra fields found that are not defined in the schema"
		analysis.Suggestions = []string{
			"Remove any fields not defined in the schema",
			"Check for typos in field names",
			"Ensure you're not adding extra properties",
		}
		analysis.CommonPitfalls = []string{
			"Adding explanation or metadata fields",
			"Misspelled required field names creating extra fields",
			"Including debug or development-only fields",
		}
	}

	// Analyze for array issues
	if strings.Contains(errorMsgLower, "array") {
		analysis.ErrorType = "array_issues"
		analysis.MainMessage = "Array field validation failed"
		analysis.Suggestions = []string{
			"Ensure array fields use proper JSON array syntax [...]",
			"Check that array items match the expected schema",
			"Verify minimum/maximum array length requirements",
		}
		analysis.CommonPitfalls = []string{
			"Using strings instead of arrays",
			"Incorrect item types within arrays",
			"Empty arrays when minimum length is required",
		}
		analysis.Example = `Example:
Schema expects: {"tags": ["string"]}
Invalid: {"tags": "tag1,tag2"}
Valid:   {"tags": ["tag1", "tag2"]}`
	}

	// Analyze for string format issues
	if strings.Contains(errorMsgLower, "format") {
		analysis.ErrorType = "format_violation"
		analysis.MainMessage = "String format validation failed"
		analysis.Suggestions = []string{
			"Check the expected string format in the schema",
			"Ensure dates are in ISO format (YYYY-MM-DD or RFC3339)",
			"Verify URLs are complete and valid",
			"Check email addresses for proper format",
		}
		analysis.CommonPitfalls = []string{
			"Partial URLs missing protocol (http/https)",
			"Invalid date formats",
			"Malformed email addresses",
		}
	}

	// Check for recovery-related insights
	if recoveryResult != nil {
		if len(recoveryResult.AppliedFixes) > 0 {
			analysis.Suggestions = append(analysis.Suggestions,
				"Note: The JSON was automatically corrected for formatting issues, but schema compliance still failed")
		}

		for _, warning := range recoveryResult.Warnings {
			if strings.Contains(warning, "aggressive") {
				analysis.CommonPitfalls = append(analysis.CommonPitfalls,
					"The input required aggressive reconstruction - consider outputting valid JSON directly")
			}
		}
	}

	// Add general guidance if no specific pattern was matched
	if analysis.ErrorType == "" {
		analysis.Suggestions = []string{
			"Review the schema carefully to understand the expected structure",
			"Ensure all required fields are present with correct types",
			"Verify that field names match the schema exactly",
			"Check that enum values are from the allowed list",
		}
		analysis.CommonPitfalls = []string{
			"Case-sensitive field names and enum values",
			"Forgetting to include all required fields",
			"Using incorrect data types for fields",
		}
	}

	// Always add universal JSON guidance
	analysis.Suggestions = append(analysis.Suggestions,
		"Output only valid JSON - no markdown code blocks or explanatory text")

	return analysis
}

// FormatProgressiveValidationWarning creates warning messages for progressive validation mode
func (f *ValidationErrorFormatter) FormatProgressiveValidationWarning(err error, recoveryResult *RecoveryResult) []string {
	warnings := []string{}

	// Add recovery information
	if recoveryResult != nil && len(recoveryResult.AppliedFixes) > 0 {
		warnings = append(warnings, fmt.Sprintf("JSON automatically corrected: %s", strings.Join(recoveryResult.AppliedFixes, ", ")))
	}

	// Add schema validation warning
	analysis := f.analyzeSchemaError(err.Error(), recoveryResult)
	warnings = append(warnings, fmt.Sprintf("Schema validation issue: %s", analysis.MainMessage))

	// Add key suggestions as warnings
	if len(analysis.Suggestions) > 0 {
		for _, suggestion := range analysis.Suggestions[:min(3, len(analysis.Suggestions))] { // Limit to top 3 suggestions
			warnings = append(warnings, fmt.Sprintf("Suggestion: %s", suggestion))
		}
	}

	return warnings
}

// ExtractFieldPath attempts to extract the field path from schema validation errors
func (f *ValidationErrorFormatter) ExtractFieldPath(errorMsg string) string {
	// Look for common JSON path patterns in error messages
	pathRegex := regexp.MustCompile(`at\s+'([^']+)'|path\s+'([^']+)'|field\s+'([^']+)'`)
	matches := pathRegex.FindStringSubmatch(errorMsg)

	for i := 1; i < len(matches); i++ {
		if matches[i] != "" {
			return matches[i]
		}
	}

	// Look for property names in quotes
	propRegex := regexp.MustCompile(`'([^']+)'`)
	propMatches := propRegex.FindAllStringSubmatch(errorMsg, -1)
	if len(propMatches) > 0 {
		return propMatches[0][1]
	}

	return ""
}

