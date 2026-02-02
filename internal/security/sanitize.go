package security

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
)

// InputSanitizer sanitizes user input for security
type InputSanitizer struct {
	config               SecurityConfig
	logger               *SecurityLogger
	promptInjectionRegex []*regexp.Regexp
}

// NewInputSanitizer creates a new input sanitizer
func NewInputSanitizer(config SecurityConfig, logger *SecurityLogger) *InputSanitizer {
	sanitizer := &InputSanitizer{
		config: config,
		logger: logger,
	}

	// Compile prompt injection patterns
	for _, pattern := range config.Sanitization.PromptInjectionPatterns {
		if regex, err := regexp.Compile(pattern); err == nil {
			sanitizer.promptInjectionRegex = append(sanitizer.promptInjectionRegex, regex)
		}
	}

	return sanitizer
}

// SanitizeInput sanitizes user input and returns sanitization record
func (is *InputSanitizer) SanitizeInput(input, inputType string) (*InputSanitizationRecord, string, error) {
	originalLength := len(input)
	sanitizedInput := input
	sanitizationRules := []string{}
	changesDetected := false

	// Check input length
	if originalLength > is.config.Sanitization.MaxInputLength {
		sanitizedInput = sanitizedInput[:is.config.Sanitization.MaxInputLength]
		sanitizationRules = append(sanitizationRules, "truncated_length")
		changesDetected = true
	}

	// Check for prompt injection if enabled
	if is.config.Sanitization.EnablePromptInjectionDetection {
		detectedPatterns := []string{}
		for _, regex := range is.promptInjectionRegex {
			if regex.MatchString(strings.ToLower(sanitizedInput)) {
				detectedPatterns = append(detectedPatterns, regex.String())
			}
		}

		if len(detectedPatterns) > 0 {
			if is.config.Sanitization.StrictMode {
				// In strict mode, reject the input
				is.logger.LogViolation(
					string(ViolationPromptInjection),
					string(SourceUserInput),
					fmt.Sprintf("Prompt injection detected in %s input", inputType),
					SeverityCritical,
					true,
				)
				return nil, "", NewPromptInjectionError(inputType, detectedPatterns)
			} else {
				// In non-strict mode, sanitize the input
				sanitizedInput = is.sanitizePromptInjection(sanitizedInput)
				sanitizationRules = append(sanitizationRules, "prompt_injection_sanitized")
				changesDetected = true

				is.logger.LogViolation(
					string(ViolationPromptInjection),
					string(SourceUserInput),
					fmt.Sprintf("Prompt injection sanitized in %s input", inputType),
					SeverityMedium,
					false,
				)
			}
		}
	}

	// Calculate risk score
	riskScore := is.calculateRiskScore(input, sanitizationRules)

	// Create sanitization record
	record := &InputSanitizationRecord{
		InputHash:         is.hashInput(input),
		InputType:         inputType,
		SanitizationRules: sanitizationRules,
		ChangesDetected:   changesDetected,
		SanitizedLength:   len(sanitizedInput),
		OriginalLength:    originalLength,
		RiskScore:         riskScore,
	}

	// Log sanitization
	is.logger.LogSanitization(inputType, changesDetected, riskScore)

	return record, sanitizedInput, nil
}

// SanitizeSchemaContent sanitizes schema content for AI processing
func (is *InputSanitizer) SanitizeSchemaContent(content string) (string, []string, error) {
	sanitizationActions := []string{}
	sanitizedContent := content

	// Remove potential prompt injection from schema descriptions
	if is.config.Sanitization.EnablePromptInjectionDetection {
		for _, regex := range is.promptInjectionRegex {
			if regex.MatchString(strings.ToLower(content)) {
				sanitizedContent = regex.ReplaceAllString(sanitizedContent, "[SANITIZED]")
				sanitizationActions = append(sanitizationActions, "removed_prompt_injection")
			}
		}
	}

	// Check content size
	if len(sanitizedContent) > is.config.Sanitization.ContentSizeLimit {
		is.logger.LogViolation(
			string(ViolationInputValidation),
			string(SourceSchemaPath),
			"Schema content exceeds size limit",
			SeverityHigh,
			true,
		)
		return "", sanitizationActions, NewInputValidationError("schema_content",
			fmt.Sprintf("exceeds size limit of %d bytes", is.config.Sanitization.ContentSizeLimit))
	}

	// Remove any embedded script tags or suspicious content
	sanitizedContent = is.removeSuspiciousContent(sanitizedContent)
	if sanitizedContent != content {
		sanitizationActions = append(sanitizationActions, "removed_suspicious_content")
	}

	return sanitizedContent, sanitizationActions, nil
}

// sanitizePromptInjection removes or neutralizes prompt injection attempts
func (is *InputSanitizer) sanitizePromptInjection(input string) string {
	sanitized := input

	// Replace instruction override patterns
	for _, regex := range is.promptInjectionRegex {
		sanitized = regex.ReplaceAllString(sanitized, " ")
	}

	// Remove multiple whitespaces
	sanitized = regexp.MustCompile(`\s+`).ReplaceAllString(sanitized, " ")
	sanitized = strings.TrimSpace(sanitized)

	return sanitized
}

// removeSuspiciousContent removes potentially dangerous content from schema
func (is *InputSanitizer) removeSuspiciousContent(content string) string {
	// Remove script tags
	scriptRegex := regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
	content = scriptRegex.ReplaceAllString(content, "")

	// Remove on* event handlers
	eventRegex := regexp.MustCompile(`(?i)on\w+\s*=\s*['"][^'"]*['"]`)
	content = eventRegex.ReplaceAllString(content, "")

	// Remove javascript: URLs
	jsRegex := regexp.MustCompile(`(?i)javascript:\s*[^'"]*`)
	content = jsRegex.ReplaceAllString(content, "")

	return content
}

// calculateRiskScore calculates a risk score for the input
func (is *InputSanitizer) calculateRiskScore(input string, sanitizationRules []string) int {
	score := 0

	// Base score for any sanitization
	if len(sanitizationRules) > 0 {
		score += 20
	}

	// Higher score for prompt injection
	for _, rule := range sanitizationRules {
		switch rule {
		case "prompt_injection_sanitized":
			score += 50
		case "truncated_length":
			score += 10
		case "removed_suspicious_content":
			score += 30
		}
	}

	// Factor in input characteristics
	lowerInput := strings.ToLower(input)
	suspiciousWords := []string{"password", "secret", "key", "token", "credential", "admin"}
	for _, word := range suspiciousWords {
		if strings.Contains(lowerInput, word) {
			score += 5
		}
	}

	// Cap at 100
	if score > 100 {
		score = 100
	}

	return score
}

// hashInput creates a SHA-256 hash of the input for tracking
func (is *InputSanitizer) hashInput(input string) string {
	hasher := sha256.New()
	hasher.Write([]byte(input))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// ValidateInputLength checks if input exceeds length limits
func (is *InputSanitizer) ValidateInputLength(input string, inputType string) error {
	if len(input) > is.config.Sanitization.MaxInputLength {
		return NewInputValidationError(inputType,
			fmt.Sprintf("exceeds maximum length of %d characters", is.config.Sanitization.MaxInputLength))
	}
	return nil
}

// IsHighRisk returns true if the input is considered high risk
func (is *InputSanitizer) IsHighRisk(record *InputSanitizationRecord) bool {
	return record.RiskScore >= 50
}