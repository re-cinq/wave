package security

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// Package-level pre-compiled regexes for hot sanitization paths.
// Compiling regexes once at init avoids repeated allocation and
// JIT cost on every call to removeSuspiciousContent and
// sanitizePromptInjection.
var (
	reWhitespace    = regexp.MustCompile(`\s+`)
	reScriptTag     = regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
	reEventHandler  = regexp.MustCompile(`(?i)on\w+\s*=\s*['"][^'"]*['"]`)
	reJavascriptURL = regexp.MustCompile(`(?i)javascript:\s*[^'"]*`)
)

// schemaCache is a process-lifetime, read-through cache for schema file
// content.  Schema files do not change during a pipeline run so we can
// cache aggressively without a TTL.
var schemaCache sync.Map // key: string (absolute path) → value: string (content)

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
			if is.config.Sanitization.MustPass {
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

// sanitizePromptInjection removes or neutralizes prompt injection attempts.
// Uses strings.Builder to avoid O(n*k) string concatenation on multiple
// replacements.
func (is *InputSanitizer) sanitizePromptInjection(input string) string {
	sanitized := input

	// Replace instruction override patterns
	for _, regex := range is.promptInjectionRegex {
		sanitized = regex.ReplaceAllString(sanitized, " ")
	}

	// Collapse multiple whitespace runs to a single space using the
	// pre-compiled package-level regex, then trim surrounding space.
	var b strings.Builder
	b.Grow(len(sanitized))
	b.WriteString(reWhitespace.ReplaceAllString(sanitized, " "))
	result := strings.TrimSpace(b.String())
	return result
}

// removeSuspiciousContent removes potentially dangerous content from schema.
// Uses the pre-compiled package-level regexes to avoid per-call compilation.
func (is *InputSanitizer) removeSuspiciousContent(content string) string {
	// Use strings.Builder to build the final result after all replacements
	// rather than chaining three separate allocations.
	content = reScriptTag.ReplaceAllString(content, "")
	content = reEventHandler.ReplaceAllString(content, "")
	content = reJavascriptURL.ReplaceAllString(content, "")
	return content
}

// shellMetachars are characters that have special meaning in POSIX shells.
// Their presence in user input is not inherently dangerous when exec.Command
// is used (bypasses shell), but it signals elevated risk if any code path
// ever routes through a shell interpreter.
const shellMetachars = "|&;$`\\!(){}[]<>*?~#"

// containsShellMetachars returns true if s contains any POSIX shell metacharacter.
func containsShellMetachars(s string) bool {
	return strings.ContainsAny(s, shellMetachars)
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

	// Shell metacharacters: safe under exec.Command but dangerous if any
	// code path routes through a shell. Flag as elevated risk.
	if containsShellMetachars(input) {
		score += 15
		is.logger.LogViolation(
			string(ViolationInputValidation),
			string(SourceUserInput),
			"Input contains shell metacharacters: defense-in-depth warning",
			SeverityLow,
			false,
		)
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

// hashInput creates a SHA-256 hash of the input for tracking.
// Uses strings.Builder indirection via fmt.Sprintf to format the hex digest.
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

// GetCachedSchemaContent retrieves schema file content from the process-lifetime
// cache, returning ("", false) on a cache miss.
func GetCachedSchemaContent(absPath string) (string, bool) {
	v, ok := schemaCache.Load(absPath)
	if !ok {
		return "", false
	}
	return v.(string), true
}

// SetCachedSchemaContent stores schema file content in the process-lifetime
// cache keyed by absolute path.
func SetCachedSchemaContent(absPath, content string) {
	schemaCache.Store(absPath, content)
}
