package adapter

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// RetryConfig defines configuration for retry mechanisms
type RetryConfig struct {
	MaxAttempts       int           `json:"max_attempts"`
	BaseDelay         time.Duration `json:"base_delay"`
	MaxDelay          time.Duration `json:"max_delay"`
	BackoffMultiplier float64       `json:"backoff_multiplier"`
	EnableJitterDelay bool          `json:"enable_jitter_delay"`

	// Output format correction settings
	EnableJSONRecovery        bool `json:"enable_json_recovery"`
	EnableStructureRecovery   bool `json:"enable_structure_recovery"`
	EnableContentExtraction   bool `json:"enable_content_extraction"`

	// Progressive enhancement settings
	ProgressiveEnhancement    bool `json:"progressive_enhancement"`
	IncrementalGuidance      bool `json:"incremental_guidance"`

	// Graceful degradation settings
	AllowPartialResults      bool `json:"allow_partial_results"`
	GenerateErrorReports     bool `json:"generate_error_reports"`
}

// DefaultRetryConfig returns sensible defaults for retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:               3,
		BaseDelay:                 1 * time.Second,
		MaxDelay:                  30 * time.Second,
		BackoffMultiplier:         2.0,
		EnableJitterDelay:         true,
		EnableJSONRecovery:        true,
		EnableStructureRecovery:   true,
		EnableContentExtraction:   true,
		ProgressiveEnhancement:    true,
		IncrementalGuidance:      true,
		AllowPartialResults:      true,
		GenerateErrorReports:     true,
	}
}

// OutputFormatCorrector provides advanced output format correction capabilities
type OutputFormatCorrector struct {
	config *RetryConfig
}

// NewOutputFormatCorrector creates a new output format corrector
func NewOutputFormatCorrector(config *RetryConfig) *OutputFormatCorrector {
	if config == nil {
		config = DefaultRetryConfig()
	}
	return &OutputFormatCorrector{config: config}
}

// CorrectOutput attempts to fix malformed output using various strategies
func (c *OutputFormatCorrector) CorrectOutput(rawOutput string, expectedFormat string, attempt int) (*CorrectionResult, error) {
	result := &CorrectionResult{
		OriginalContent: rawOutput,
		TargetFormat:   expectedFormat,
		Attempt:        attempt,
		StartTime:      time.Now(),
	}

	// Try different correction strategies in order of sophistication
	strategies := c.getCorrectionStrategies(expectedFormat, attempt)

	for _, strategy := range strategies {
		corrected, metadata, err := strategy.Apply(rawOutput, expectedFormat)

		result.StrategiesAttempted = append(result.StrategiesAttempted, strategy.Name())

		if err == nil && corrected != "" {
			// Validate the corrected output
			if c.validateCorrectedOutput(corrected, expectedFormat) {
				result.Success = true
				result.CorrectedContent = corrected
				result.AppliedStrategy = strategy.Name()
				result.Metadata = metadata
				result.Duration = time.Since(result.StartTime)
				return result, nil
			}
		}

		// Record failed strategy for debugging
		result.FailedStrategies = append(result.FailedStrategies, FailedStrategy{
			Name:  strategy.Name(),
			Error: err,
		})
	}

	result.Duration = time.Since(result.StartTime)
	return result, fmt.Errorf("all correction strategies failed after %d attempts", len(strategies))
}

// CorrectionStrategy defines an interface for different correction approaches
type CorrectionStrategy interface {
	Name() string
	Apply(content, format string) (corrected string, metadata map[string]interface{}, err error)
}

// getCorrectionStrategies returns appropriate strategies based on format and attempt
func (c *OutputFormatCorrector) getCorrectionStrategies(format string, attempt int) []CorrectionStrategy {
	var strategies []CorrectionStrategy

	if format == "json" {
		// For JSON, try increasingly sophisticated strategies
		strategies = append(strategies,
			&DirectJSONValidationStrategy{},
			&MarkdownCodeBlockExtractionStrategy{},
			&RegexJSONExtractionStrategy{},
			&HeuristicJSONRecoveryStrategy{},
		)

		// On later attempts, try more aggressive strategies
		if attempt > 1 {
			strategies = append(strategies,
				&PartialJSONRecoveryStrategy{},
				&TemplateBasedRecoveryStrategy{},
			)
		}

		// Last resort strategies
		if attempt >= 3 {
			strategies = append(strategies,
				&AIAssistedRecoveryStrategy{},
				&StructuredErrorReportStrategy{},
			)
		}
	}

	// Add format-agnostic strategies
	strategies = append(strategies,
		&ContentExtractionStrategy{format: format},
		&FallbackStrategy{},
	)

	return strategies
}

// validateCorrectedOutput performs validation on corrected content
func (c *OutputFormatCorrector) validateCorrectedOutput(content, format string) bool {
	switch format {
	case "json":
		var js json.RawMessage
		return json.Unmarshal([]byte(content), &js) == nil
	case "yaml":
		// Basic YAML validation - could be enhanced with actual YAML parser
		return strings.Contains(content, ":") && !strings.HasPrefix(content, "{")
	case "markdown":
		// Basic markdown validation
		return len(content) > 10 && (strings.Contains(content, "#") || strings.Contains(content, "-"))
	default:
		// For unknown formats, assume non-empty content is valid
		return len(strings.TrimSpace(content)) > 0
	}
}

// DirectJSONValidationStrategy tries to parse content as-is
type DirectJSONValidationStrategy struct{}

func (s *DirectJSONValidationStrategy) Name() string { return "direct_json_validation" }

func (s *DirectJSONValidationStrategy) Apply(content, format string) (string, map[string]interface{}, error) {
	if format != "json" {
		return "", nil, fmt.Errorf("strategy only applies to JSON format")
	}

	var js json.RawMessage
	if err := json.Unmarshal([]byte(content), &js); err != nil {
		return "", nil, fmt.Errorf("content is not valid JSON: %w", err)
	}

	// Re-marshal to ensure consistent formatting
	formatted, err := json.MarshalIndent(js, "", "  ")
	if err != nil {
		return content, map[string]interface{}{"validation": "passed"}, nil
	}

	return string(formatted), map[string]interface{}{"validation": "passed", "reformatted": true}, nil
}

// MarkdownCodeBlockExtractionStrategy extracts JSON from markdown code blocks
type MarkdownCodeBlockExtractionStrategy struct{}

func (s *MarkdownCodeBlockExtractionStrategy) Name() string { return "markdown_extraction" }

func (s *MarkdownCodeBlockExtractionStrategy) Apply(content, format string) (string, map[string]interface{}, error) {
	if format != "json" {
		return "", nil, fmt.Errorf("strategy only applies to JSON format")
	}

	extracted := ExtractJSONFromMarkdown(content)
	if extracted == "" {
		return "", nil, fmt.Errorf("no JSON found in markdown code blocks")
	}

	return extracted, map[string]interface{}{"extracted_from": "markdown_code_block"}, nil
}

// RegexJSONExtractionStrategy uses regex to find JSON-like structures
type RegexJSONExtractionStrategy struct{}

func (s *RegexJSONExtractionStrategy) Name() string { return "regex_json_extraction" }

func (s *RegexJSONExtractionStrategy) Apply(content, format string) (string, map[string]interface{}, error) {
	if format != "json" {
		return "", nil, fmt.Errorf("strategy only applies to JSON format")
	}

	// Try to find JSON object patterns
	objectRegex := regexp.MustCompile(`\{[^{}]*(?:\{[^{}]*\}[^{}]*)*\}`)
	arrayRegex := regexp.MustCompile(`\[[^\[\]]*(?:\[[^\[\]]*\][^\[\]]*)*\]`)

	// First try to find complete objects
	if matches := objectRegex.FindAllString(content, -1); len(matches) > 0 {
		for _, match := range matches {
			var js json.RawMessage
			if json.Unmarshal([]byte(match), &js) == nil {
				return match, map[string]interface{}{"extracted_from": "regex_object_pattern"}, nil
			}
		}
	}

	// Then try arrays
	if matches := arrayRegex.FindAllString(content, -1); len(matches) > 0 {
		for _, match := range matches {
			var js json.RawMessage
			if json.Unmarshal([]byte(match), &js) == nil {
				return match, map[string]interface{}{"extracted_from": "regex_array_pattern"}, nil
			}
		}
	}

	return "", nil, fmt.Errorf("no valid JSON patterns found")
}

// HeuristicJSONRecoveryStrategy uses heuristics to repair broken JSON
type HeuristicJSONRecoveryStrategy struct{}

func (s *HeuristicJSONRecoveryStrategy) Name() string { return "heuristic_json_recovery" }

func (s *HeuristicJSONRecoveryStrategy) Apply(content, format string) (string, map[string]interface{}, error) {
	if format != "json" {
		return "", nil, fmt.Errorf("strategy only applies to JSON format")
	}

	fixes := make([]string, 0)
	repaired := content

	// Remove common prefixes/suffixes that aren't JSON
	patterns := []struct {
		description string
		regex       *regexp.Regexp
		replacement string
	}{
		{"remove explanatory prefix", regexp.MustCompile(`^[^{[]*`), ""},
		{"remove explanatory suffix", regexp.MustCompile(`[\}\]][^}\]]*$`), "}"},
		{"fix trailing comma in object", regexp.MustCompile(`,(\s*})`), "$1"},
		{"fix trailing comma in array", regexp.MustCompile(`,(\s*\])`), "$1"},
		{"add missing quotes to keys", regexp.MustCompile(`(\w+)(\s*:\s*)`), `"$1"$2`},
		{"fix single quotes to double", regexp.MustCompile(`'([^']*)'`), `"$1"`},
	}

	for _, pattern := range patterns {
		if pattern.regex.MatchString(repaired) {
			repaired = pattern.regex.ReplaceAllString(repaired, pattern.replacement)
			fixes = append(fixes, pattern.description)
		}
	}

	// Validate the repaired JSON
	var js json.RawMessage
	if err := json.Unmarshal([]byte(repaired), &js); err != nil {
		return "", nil, fmt.Errorf("heuristic repair failed: %w", err)
	}

	metadata := map[string]interface{}{
		"fixes_applied": fixes,
		"repair_count": len(fixes),
	}

	return repaired, metadata, nil
}

// PartialJSONRecoveryStrategy extracts valid partial JSON structures
type PartialJSONRecoveryStrategy struct{}

func (s *PartialJSONRecoveryStrategy) Name() string { return "partial_json_recovery" }

func (s *PartialJSONRecoveryStrategy) Apply(content, format string) (string, map[string]interface{}, error) {
	if format != "json" {
		return "", nil, fmt.Errorf("strategy only applies to JSON format")
	}

	// Try to construct a minimal valid JSON from identifiable key-value pairs
	lines := strings.Split(content, "\n")
	validProperties := make([]string, 0)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Look for property-like patterns
		if strings.Contains(trimmed, ":") &&
		   (strings.Contains(trimmed, "\"") || strings.Contains(trimmed, "'")) {

			// Try to clean and validate this property
			cleaned := cleanJSONProperty(trimmed)
			if cleaned != "" {
				validProperties = append(validProperties, cleaned)
			}
		}
	}

	if len(validProperties) == 0 {
		return "", nil, fmt.Errorf("no valid JSON properties found")
	}

	// Construct minimal JSON object
	jsonStr := "{\n  " + strings.Join(validProperties, ",\n  ") + "\n}"

	var js json.RawMessage
	if err := json.Unmarshal([]byte(jsonStr), &js); err != nil {
		return "", nil, fmt.Errorf("partial recovery failed: %w", err)
	}

	metadata := map[string]interface{}{
		"recovered_properties": len(validProperties),
		"recovery_type": "partial_properties",
	}

	return jsonStr, metadata, nil
}

// cleanJSONProperty attempts to clean a single JSON property line
func cleanJSONProperty(line string) string {
	// Remove trailing commas
	line = strings.TrimSuffix(strings.TrimSpace(line), ",")

	// Basic validation - should contain a colon and quotes
	if !strings.Contains(line, ":") {
		return ""
	}

	// Try to parse as a single property in an object
	testJSON := "{" + line + "}"
	var js json.RawMessage
	if json.Unmarshal([]byte(testJSON), &js) == nil {
		return line
	}

	return ""
}

// TemplateBasedRecoveryStrategy uses common patterns to reconstruct output
type TemplateBasedRecoveryStrategy struct{}

func (s *TemplateBasedRecoveryStrategy) Name() string { return "template_recovery" }

func (s *TemplateBasedRecoveryStrategy) Apply(content, format string) (string, map[string]interface{}, error) {
	// This strategy would use common templates for different output types
	// For now, implement a simple fallback template

	template := getDefaultTemplate(format)
	if template == "" {
		return "", nil, fmt.Errorf("no template available for format %s", format)
	}

	metadata := map[string]interface{}{
		"template_used": "default_" + format,
		"original_content_length": len(content),
	}

	return template, metadata, nil
}

func getDefaultTemplate(format string) string {
	switch format {
	case "json":
		return `{
  "error": "malformed_output_recovery",
  "message": "Original output could not be parsed, using fallback template",
  "status": "partial_failure",
  "recovered": true
}`
	case "yaml":
		return `error: malformed_output_recovery
message: Original output could not be parsed, using fallback template
status: partial_failure
recovered: true`
	case "markdown":
		return `# Output Recovery

The original output could not be processed due to formatting issues.
This is a fallback response indicating partial failure.

## Status
- Recovery: Attempted
- Success: Partial
- Original Content: Preserved in metadata`
	default:
		return "Output recovery failed - no template available"
	}
}

// AIAssistedRecoveryStrategy would use AI to fix the output (placeholder for future)
type AIAssistedRecoveryStrategy struct{}

func (s *AIAssistedRecoveryStrategy) Name() string { return "ai_assisted_recovery" }

func (s *AIAssistedRecoveryStrategy) Apply(content, format string) (string, map[string]interface{}, error) {
	// This would integrate with an AI service to fix malformed output
	// For now, return an error indicating this feature is not yet implemented
	return "", nil, fmt.Errorf("AI-assisted recovery not yet implemented")
}

// StructuredErrorReportStrategy generates a detailed error report
type StructuredErrorReportStrategy struct{}

func (s *StructuredErrorReportStrategy) Name() string { return "structured_error_report" }

func (s *StructuredErrorReportStrategy) Apply(content, format string) (string, map[string]interface{}, error) {
	report := map[string]interface{}{
		"error_type": "output_format_recovery_failed",
		"timestamp": time.Now().Format(time.RFC3339),
		"target_format": format,
		"content_length": len(content),
		"content_preview": truncateString(content, 200),
		"analysis": analyzeContent(content),
		"recommendations": getRecoveryRecommendations(content, format),
	}

	jsonBytes, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate error report: %w", err)
	}

	metadata := map[string]interface{}{
		"report_type": "structured_error",
		"is_fallback": true,
	}

	return string(jsonBytes), metadata, nil
}

// ContentExtractionStrategy extracts meaningful content regardless of format
type ContentExtractionStrategy struct {
	format string
}

func (s *ContentExtractionStrategy) Name() string { return "content_extraction" }

func (s *ContentExtractionStrategy) Apply(content, format string) (string, map[string]interface{}, error) {
	// Extract the most meaningful content regardless of format issues
	extracted := extractMeaningfulContent(content)
	if extracted == "" {
		return "", nil, fmt.Errorf("no meaningful content could be extracted")
	}

	metadata := map[string]interface{}{
		"extraction_type": "meaningful_content",
		"original_length": len(content),
		"extracted_length": len(extracted),
	}

	return extracted, metadata, nil
}

// FallbackStrategy provides last resort handling
type FallbackStrategy struct{}

func (s *FallbackStrategy) Name() string { return "fallback" }

func (s *FallbackStrategy) Apply(content, format string) (string, map[string]interface{}, error) {
	// Return the original content with a warning marker
	fallbackContent := fmt.Sprintf("RECOVERY_FAILED: %s", content)

	metadata := map[string]interface{}{
		"strategy": "fallback",
		"warning": "all correction attempts failed",
		"preserve_original": true,
	}

	return fallbackContent, metadata, nil
}

// Helper functions

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func analyzeContent(content string) map[string]interface{} {
	analysis := make(map[string]interface{})

	analysis["has_json_brackets"] = strings.Contains(content, "{") && strings.Contains(content, "}")
	analysis["has_array_brackets"] = strings.Contains(content, "[") && strings.Contains(content, "]")
	analysis["has_colons"] = strings.Contains(content, ":")
	analysis["has_quotes"] = strings.Contains(content, "\"")
	analysis["line_count"] = len(strings.Split(content, "\n"))
	analysis["has_markdown_blocks"] = strings.Contains(content, "```")

	return analysis
}

func getRecoveryRecommendations(content, format string) []string {
	recommendations := make([]string, 0)

	if format == "json" {
		if !strings.Contains(content, "{") && !strings.Contains(content, "[") {
			recommendations = append(recommendations, "Ensure output starts with { or [")
		}
		if strings.Contains(content, "```") {
			recommendations = append(recommendations, "Remove markdown code block formatting")
		}
		if !strings.Contains(content, ":") {
			recommendations = append(recommendations, "Include proper key:value pairs")
		}
	}

	recommendations = append(recommendations, "Review AI persona instructions for output format clarity")
	recommendations = append(recommendations, "Consider adjusting temperature or model parameters")
	recommendations = append(recommendations, "Verify that the required schema is properly communicated")

	return recommendations
}

func extractMeaningfulContent(content string) string {
	lines := strings.Split(content, "\n")
	meaningfulLines := make([]string, 0)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines
		if len(trimmed) == 0 {
			continue
		}

		// Skip lines that look like markdown artifacts
		if strings.HasPrefix(trimmed, "```") {
			continue
		}

		// Include lines with substantial content
		if len(trimmed) > 10 {
			meaningfulLines = append(meaningfulLines, trimmed)
		}
	}

	if len(meaningfulLines) == 0 {
		return content // Return original if no extraction possible
	}

	return strings.Join(meaningfulLines, "\n")
}

// CorrectionResult captures the outcome of an output correction attempt
type CorrectionResult struct {
	Success              bool                     `json:"success"`
	OriginalContent      string                   `json:"original_content"`
	CorrectedContent     string                   `json:"corrected_content"`
	TargetFormat        string                   `json:"target_format"`
	AppliedStrategy     string                   `json:"applied_strategy"`
	StrategiesAttempted []string                 `json:"strategies_attempted"`
	FailedStrategies    []FailedStrategy         `json:"failed_strategies"`
	Metadata            map[string]interface{}   `json:"metadata"`
	Attempt             int                      `json:"attempt"`
	StartTime           time.Time                `json:"start_time"`
	Duration            time.Duration            `json:"duration"`
}

// FailedStrategy records information about a failed correction strategy
type FailedStrategy struct {
	Name  string `json:"name"`
	Error error  `json:"error"`
}

// FormatSummary returns a human-readable summary of the correction attempt
func (r *CorrectionResult) FormatSummary() string {
	if r.Success {
		return fmt.Sprintf("✓ Output corrected using %s strategy (%s)", r.AppliedStrategy, r.Duration.Truncate(time.Millisecond))
	}

	return fmt.Sprintf("✗ Correction failed after trying %d strategies (%s)",
		len(r.StrategiesAttempted), r.Duration.Truncate(time.Millisecond))
}