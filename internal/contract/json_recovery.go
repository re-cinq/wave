package contract

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// JSONRecoveryParser provides advanced JSON parsing with intelligent error recovery
type JSONRecoveryParser struct {
	// RecoveryLevel determines how aggressive the recovery attempts should be
	RecoveryLevel RecoveryLevel
	// MaxRecoveryAttempts limits the number of recovery strategies to try
	MaxRecoveryAttempts int
	// PreserveOriginalOnFailure keeps the original input if all recovery fails
	PreserveOriginalOnFailure bool
}

// RecoveryLevel defines the aggressiveness of JSON recovery attempts
type RecoveryLevel int

const (
	// ConservativeRecovery only fixes safe, obvious issues
	ConservativeRecovery RecoveryLevel = iota
	// ProgressiveRecovery attempts more complex fixes
	ProgressiveRecovery
	// AggressiveRecovery tries all available recovery strategies
	AggressiveRecovery
)

// RecoveryResult contains the results of JSON recovery attempts
type RecoveryResult struct {
	// OriginalInput is the unmodified input
	OriginalInput string
	// RecoveredJSON is the cleaned/recovered JSON
	RecoveredJSON string
	// IsValid indicates if the recovered JSON is parseable
	IsValid bool
	// AppliedFixes lists all fixes that were applied
	AppliedFixes []string
	// Warnings lists potential issues that couldn't be fixed
	Warnings []string
	// RecoveryLevel used for this parsing
	RecoveryLevel RecoveryLevel
	// ParsedData contains the unmarshaled JSON data if successful
	ParsedData interface{}
}

// NewJSONRecoveryParser creates a new parser with specified recovery level
func NewJSONRecoveryParser(level RecoveryLevel) *JSONRecoveryParser {
	return &JSONRecoveryParser{
		RecoveryLevel:             level,
		MaxRecoveryAttempts:       10,
		PreserveOriginalOnFailure: true,
	}
}

// ParseWithRecovery attempts to parse JSON with progressive error recovery
func (p *JSONRecoveryParser) ParseWithRecovery(input string) (*RecoveryResult, error) {
	result := &RecoveryResult{
		OriginalInput:  input,
		RecoveredJSON:  input,
		RecoveryLevel:  p.RecoveryLevel,
		AppliedFixes:   []string{},
		Warnings:       []string{},
	}

	// First, try parsing as-is
	var testData interface{}
	if err := json.Unmarshal([]byte(input), &testData); err == nil {
		result.IsValid = true
		result.ParsedData = testData
		return result, nil
	}

	// Apply recovery strategies progressively
	recoveryStrategies := p.getRecoveryStrategies()

	currentJSON := input
	for i, strategy := range recoveryStrategies {
		if i >= p.MaxRecoveryAttempts {
			break
		}

		recovered, fixes, warnings := strategy(currentJSON)

		// Test if this recovery worked
		var testData interface{}
		if err := json.Unmarshal([]byte(recovered), &testData); err == nil {
			result.RecoveredJSON = recovered
			result.IsValid = true
			result.ParsedData = testData
			result.AppliedFixes = append(result.AppliedFixes, fixes...)
			result.Warnings = append(result.Warnings, warnings...)
			return result, nil
		}

		// If this strategy helped but didn't fully fix, continue with the improved version
		if recovered != currentJSON {
			currentJSON = recovered
			result.AppliedFixes = append(result.AppliedFixes, fixes...)
			result.Warnings = append(result.Warnings, warnings...)
		}
	}

	// All recovery attempts failed
	if p.PreserveOriginalOnFailure {
		result.RecoveredJSON = input
	} else {
		result.RecoveredJSON = currentJSON
	}

	return result, fmt.Errorf("JSON recovery failed after %d attempts", len(recoveryStrategies))
}

// getRecoveryStrategies returns the list of recovery strategies based on recovery level
func (p *JSONRecoveryParser) getRecoveryStrategies() []func(string) (string, []string, []string) {
	strategies := []func(string) (string, []string, []string){
		p.extractFromAIExplanation,  // New: Handle AI explanatory text first
		p.extractFromMarkdown,
		p.removeComments,
		p.fixTrailingCommas,
		p.normalizeWhitespace,
	}

	if p.RecoveryLevel >= ProgressiveRecovery {
		strategies = append(strategies,
			p.quoteUnquotedKeys,
			p.fixSingleQuotes,
			p.extractJSONFromText,
			p.fixMissingCommas,
		)
	}

	if p.RecoveryLevel >= AggressiveRecovery {
		strategies = append(strategies,
			p.fixUnbalancedBraces,
			p.reconstructFromParts,
			p.inferMissingFields,
		)
	}

	return strategies
}

// Recovery strategy implementations

func (p *JSONRecoveryParser) extractFromAIExplanation(input string) (string, []string, []string) {
	fixes := []string{}
	warnings := []string{}

	// Handle AI-generated explanatory text before JSON
	// Common patterns where AI explains what it's doing before outputting JSON

	aiExplanationPatterns := [][]string{
		// Pattern: explanation text followed by colon and newlines, then JSON
		{"Based on the analysis", "I can provide", ":", "\n"},
		{"Let me extract", "JSON", ":", "\n"},
		{"Here", "JSON", ":", "\n"},
		{"The current file has invalid JSON", "provide", ":", "\n"},
		{"Since I need Write permission", "directly:", "\n"},
	}

	for _, patterns := range aiExplanationPatterns {
		// Check if all pattern elements exist in the input
		allFound := true
		for _, pattern := range patterns {
			if !strings.Contains(input, pattern) {
				allFound = false
				break
			}
		}

		if allFound {
			// Find the last colon followed by whitespace, then JSON
			colonIndex := strings.LastIndex(input, ":")
			if colonIndex != -1 && colonIndex < len(input)-1 {
				remaining := input[colonIndex+1:]
				remaining = strings.TrimSpace(remaining)

				// Check if what follows looks like JSON
				if strings.HasPrefix(remaining, "{") || strings.HasPrefix(remaining, "[") {
					fixes = append(fixes, "removed_ai_explanation_text")
					return remaining, fixes, warnings
				}
			}
		}
	}

	// Pattern: Look for phrases that indicate AI is providing data/output
	aiIndicatorPhrases := []string{
		"The enhanced data is:",
		"The results are:",
		"The data is:",
		"Here's the data:",
		"Output:",
		"Result:",
		"Data:",
		"Analysis:",
	}

	for _, phrase := range aiIndicatorPhrases {
		if strings.Contains(input, phrase) {
			// Find this phrase and extract JSON after it
			phraseIndex := strings.Index(input, phrase)
			if phraseIndex != -1 {
				remaining := input[phraseIndex+len(phrase):]
				remaining = strings.TrimSpace(remaining)

				// Look for JSON start
				firstBrace := strings.IndexAny(remaining, "{[")
				if firstBrace != -1 {
					jsonStart := remaining[firstBrace:]
					fixes = append(fixes, "removed_ai_explanation_text")
					return jsonStart, fixes, warnings
				}
			}
		}
	}

	// Additional pattern: Look for phrases that contain indicators followed by colons
	colonBasedPhrases := []string{
		"clean JSON output",
		"provide clean JSON",
		"matches the schema",
		"required schema",
		"JSON format",
		"JSON structure",
	}

	for _, phrase := range colonBasedPhrases {
		if strings.Contains(input, phrase) {
			// Find the phrase and look for a colon after it, then JSON
			phraseIndex := strings.Index(input, phrase)
			if phraseIndex != -1 {
				afterPhrase := input[phraseIndex+len(phrase):]
				// Look for a colon in the text after this phrase
				colonIndex := strings.Index(afterPhrase, ":")
				if colonIndex != -1 {
					afterColon := afterPhrase[colonIndex+1:]
					afterColon = strings.TrimSpace(afterColon)

					// Look for JSON start
					firstBrace := strings.IndexAny(afterColon, "{[")
					if firstBrace != -1 {
						jsonStart := afterColon[firstBrace:]
						fixes = append(fixes, "removed_ai_explanation_text")
						return jsonStart, fixes, warnings
					}
				}
			}
		}
	}

	// Pattern: Look for any explanatory paragraph followed by JSON
	lines := strings.Split(input, "\n")
	var jsonStart int = -1

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "{") || strings.HasPrefix(line, "[") {
			// This might be the start of JSON
			jsonStart = i
			break
		}
	}

	if jsonStart > 0 && jsonStart < len(lines) {
		// Check if there's explanatory text before the JSON
		hasExplanation := false
		for i := 0; i < jsonStart; i++ {
			line := strings.TrimSpace(lines[i])
			if len(line) > 15 && (strings.Contains(line, "analysis") ||
			   strings.Contains(line, "provide") ||
			   strings.Contains(line, "output") ||
			   strings.Contains(line, "JSON") ||
			   strings.Contains(line, "extract") ||
			   strings.Contains(line, "data") ||
			   strings.Contains(line, "result") ||
			   strings.Contains(line, "schema")) {
				hasExplanation = true
				break
			}
		}

		if hasExplanation {
			// Extract everything from the JSON start
			jsonPart := strings.Join(lines[jsonStart:], "\n")
			jsonPart = strings.TrimSpace(jsonPart)

			// Verify this looks like JSON
			if strings.HasPrefix(jsonPart, "{") || strings.HasPrefix(jsonPart, "[") {
				fixes = append(fixes, "extracted_json_after_explanation")
				return jsonPart, fixes, warnings
			}
		}
	}

	return input, fixes, warnings
}

func (p *JSONRecoveryParser) extractFromMarkdown(input string) (string, []string, []string) {
	fixes := []string{}
	warnings := []string{}

	// Handle markdown code blocks more carefully to avoid nested block issues
	// Look for ```json or ``` at the start of the JSON content
	jsonBlockStart := strings.Index(input, "```json")
	if jsonBlockStart == -1 {
		jsonBlockStart = strings.Index(input, "```")
		if jsonBlockStart == -1 {
			// Try inline code
			return p.extractInlineCode(input, fixes, warnings)
		}
	}

	// Find the content after the opening ```
	startPos := jsonBlockStart
	for startPos < len(input) && input[startPos] != '\n' {
		startPos++
	}
	if startPos < len(input) {
		startPos++ // Skip the newline
	}

	// Find the closing ``` that's on its own line
	endPos := -1
	searchPos := startPos
	for {
		nextTriple := strings.Index(input[searchPos:], "```")
		if nextTriple == -1 {
			break
		}

		candidate := searchPos + nextTriple
		// Check if this ``` is at the start of a line (or has only whitespace before it)
		lineStart := candidate
		for lineStart > 0 && input[lineStart-1] != '\n' {
			lineStart--
		}

		// Check if there's only whitespace between line start and ```
		onlyWhitespace := true
		for i := lineStart; i < candidate; i++ {
			if input[i] != ' ' && input[i] != '\t' {
				onlyWhitespace = false
				break
			}
		}

		if onlyWhitespace {
			endPos = candidate
			break
		}

		searchPos = candidate + 3
	}

	if endPos != -1 && endPos > startPos {
		extracted := input[startPos:endPos]
		extracted = strings.TrimSpace(extracted)

		// Verify this looks like JSON
		if strings.HasPrefix(extracted, "{") || strings.HasPrefix(extracted, "[") {
			fixes = append(fixes, "extracted_from_markdown_code_block")
			return extracted, fixes, warnings
		}
	}

	// Fallback to simple regex for simple cases
	markdownRegex := regexp.MustCompile("(?s)^.*?```(?:json)?\\s*\\n?(.*?)\\n?```.*$")
	if markdownRegex.MatchString(input) {
		extracted := markdownRegex.ReplaceAllString(input, "$1")
		fixes = append(fixes, "extracted_from_markdown_code_block")
		return strings.TrimSpace(extracted), fixes, warnings
	}

	return p.extractInlineCode(input, fixes, warnings)
}

func (p *JSONRecoveryParser) extractInlineCode(input string, fixes []string, warnings []string) (string, []string, []string) {
	// Remove inline code markers (`json content`)
	inlineCodeRegex := regexp.MustCompile("^.*?`([^`]+)`.*$")
	if inlineCodeRegex.MatchString(input) && strings.Contains(input, "{") {
		extracted := inlineCodeRegex.ReplaceAllString(input, "$1")
		if strings.Contains(extracted, "{") || strings.Contains(extracted, "[") {
			fixes = append(fixes, "extracted_from_inline_code")
			return strings.TrimSpace(extracted), fixes, warnings
		}
	}

	return input, fixes, warnings
}

func (p *JSONRecoveryParser) removeComments(input string) (string, []string, []string) {
	fixes := []string{}
	warnings := []string{}
	output := input

	// Remove single-line comments (// comment)
	singleLineRegex := regexp.MustCompile(`\s*//[^\n]*`)
	if singleLineRegex.MatchString(output) {
		output = singleLineRegex.ReplaceAllString(output, "")
		fixes = append(fixes, "removed_single_line_comments")
	}

	// Remove multi-line comments (/* comment */)
	multiLineRegex := regexp.MustCompile(`(?s)/\*.*?\*/`)
	if multiLineRegex.MatchString(output) {
		output = multiLineRegex.ReplaceAllString(output, "")
		fixes = append(fixes, "removed_multi_line_comments")
	}

	// Remove hash comments at line start
	hashRegex := regexp.MustCompile(`(?m)^\s*#.*$`)
	if hashRegex.MatchString(output) {
		output = hashRegex.ReplaceAllString(output, "")
		fixes = append(fixes, "removed_hash_comments")
	}

	return output, fixes, warnings
}

func (p *JSONRecoveryParser) fixTrailingCommas(input string) (string, []string, []string) {
	fixes := []string{}
	warnings := []string{}

	// Remove trailing commas before } or ]
	trailingCommaRegex := regexp.MustCompile(`,(\s*[}\]])`)
	if trailingCommaRegex.MatchString(input) {
		output := trailingCommaRegex.ReplaceAllString(input, "$1")
		fixes = append(fixes, "removed_trailing_commas")
		return output, fixes, warnings
	}

	return input, fixes, warnings
}

func (p *JSONRecoveryParser) normalizeWhitespace(input string) (string, []string, []string) {
	fixes := []string{}
	warnings := []string{}

	// Normalize line endings
	normalized := strings.ReplaceAll(input, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	// Remove excessive whitespace but preserve structure
	lines := strings.Split(normalized, "\n")
	for i, line := range lines {
		// Collapse multiple spaces/tabs to single space
		line = regexp.MustCompile(`[ \t]+`).ReplaceAllString(line, " ")
		lines[i] = strings.TrimSpace(line)
	}

	// Remove empty lines within JSON structure
	var cleanedLines []string
	for _, line := range lines {
		if line != "" {
			cleanedLines = append(cleanedLines, line)
		}
	}

	output := strings.Join(cleanedLines, "\n")
	if output != input {
		fixes = append(fixes, "normalized_whitespace")
	}

	return output, fixes, warnings
}

func (p *JSONRecoveryParser) quoteUnquotedKeys(input string) (string, []string, []string) {
	fixes := []string{}
	warnings := []string{}

	// Quote unquoted object keys: {key: "value"} -> {"key": "value"}
	unquotedKeyRegex := regexp.MustCompile(`(\{|,)\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*:`)
	if unquotedKeyRegex.MatchString(input) {
		output := unquotedKeyRegex.ReplaceAllString(input, `$1"$2":`)
		fixes = append(fixes, "quoted_unquoted_keys")
		return output, fixes, warnings
	}

	return input, fixes, warnings
}

func (p *JSONRecoveryParser) fixSingleQuotes(input string) (string, []string, []string) {
	fixes := []string{}
	warnings := []string{}

	// Convert single quotes to double quotes for string values
	// Be careful not to convert single quotes within double-quoted strings
	singleQuoteRegex := regexp.MustCompile(`'([^']*)'`)
	if singleQuoteRegex.MatchString(input) {
		// Test if converting would make valid JSON
		testOutput := singleQuoteRegex.ReplaceAllString(input, `"$1"`)
		var testData interface{}
		if err := json.Unmarshal([]byte(testOutput), &testData); err == nil {
			fixes = append(fixes, "converted_single_quotes_to_double_quotes")
			return testOutput, fixes, warnings
		} else {
			warnings = append(warnings, "single_quotes_found_but_conversion_failed")
		}
	}

	return input, fixes, warnings
}

func (p *JSONRecoveryParser) extractJSONFromText(input string) (string, []string, []string) {
	fixes := []string{}
	warnings := []string{}

	// Find JSON within surrounding text
	input = strings.TrimSpace(input)

	// Look for common AI patterns where they explain before outputting JSON
	// Pattern 1: "Let me extract just the JSON portion from what I read:"
	// Pattern 2: "Here's the JSON:"
	// Pattern 3: "Based on the analysis... I can provide the clean JSON output..."

	aiPrefixPatterns := []string{
		"Let me extract just the JSON portion from what I read:",
		"Here's the JSON:",
		"I can provide the clean JSON output",
		"The current file has invalid JSON due to explanatory text. Since I need Write permission to fix the artifact.json file, let me provide the corrected JSON output directly:",
		"clean JSON output that matches the required schema",
		"provide the corrected JSON output directly",
	}

	for _, pattern := range aiPrefixPatterns {
		if strings.Contains(input, pattern) {
			// Find the position after this pattern
			pos := strings.Index(input, pattern)
			if pos != -1 {
				// Start looking for JSON after this pattern
				searchStart := pos + len(pattern)
				if searchStart < len(input) {
					input = strings.TrimSpace(input[searchStart:])
					fixes = append(fixes, "removed_ai_explanation_prefix")
					break
				}
			}
		}
	}

	// Look for JSON object or array
	firstBrace := strings.IndexAny(input, "{[")
	if firstBrace == -1 {
		return input, fixes, warnings
	}

	jsonStartChar := rune(input[firstBrace])
	var jsonEndChar rune
	if jsonStartChar == '{' {
		jsonEndChar = '}'
	} else {
		jsonEndChar = ']'
	}

	// Find matching closing brace/bracket
	depth := 0
	inString := false
	escaped := false
	endPos := -1

	for i, ch := range input[firstBrace:] {
		if escaped {
			escaped = false
			continue
		}

		if ch == '\\' && inString {
			escaped = true
			continue
		}

		if ch == '"' && !escaped {
			inString = !inString
			continue
		}

		if !inString {
			if ch == jsonStartChar {
				depth++
			} else if ch == jsonEndChar {
				depth--
				if depth == 0 {
					endPos = firstBrace + i + 1
					break
				}
			}
		}
	}

	if endPos > firstBrace {
		extracted := input[firstBrace:endPos]
		// Validate that the extracted part is better JSON
		var testData interface{}
		if err := json.Unmarshal([]byte(extracted), &testData); err == nil {
			fixes = append(fixes, "extracted_json_from_text")
			return extracted, fixes, warnings
		} else {
			warnings = append(warnings, "found_json_structure_but_still_invalid")
		}
	}

	return input, fixes, warnings
}

func (p *JSONRecoveryParser) fixMissingCommas(input string) (string, []string, []string) {
	fixes := []string{}
	warnings := []string{}

	// Look for missing commas between object properties: "key":"value" "key2":"value2"
	missingCommaRegex := regexp.MustCompile(`("(?:[^"\\]|\\.)*")\s+("(?:[^"\\]|\\.)*"\s*:)`)
	if missingCommaRegex.MatchString(input) {
		output := missingCommaRegex.ReplaceAllString(input, "$1,$2")
		fixes = append(fixes, "added_missing_commas_between_properties")
		return output, fixes, warnings
	}

	// Look for missing commas between array elements: "value" "value2"
	missingArrayCommaRegex := regexp.MustCompile(`("(?:[^"\\]|\\.)*")\s+("(?:[^"\\]|\\.)*")`)
	if missingArrayCommaRegex.MatchString(input) {
		output := missingArrayCommaRegex.ReplaceAllString(input, "$1,$2")
		fixes = append(fixes, "added_missing_commas_between_array_elements")
		return output, fixes, warnings
	}

	return input, fixes, warnings
}

func (p *JSONRecoveryParser) fixUnbalancedBraces(input string) (string, []string, []string) {
	fixes := []string{}
	warnings := []string{}

	// Count braces and brackets
	openBraces := strings.Count(input, "{")
	closeBraces := strings.Count(input, "}")
	openBrackets := strings.Count(input, "[")
	closeBrackets := strings.Count(input, "]")

	output := input

	// Add missing closing braces
	if openBraces > closeBraces {
		missing := openBraces - closeBraces
		output += strings.Repeat("}", missing)
		fixes = append(fixes, fmt.Sprintf("added_%d_missing_closing_braces", missing))
	}

	// Add missing closing brackets
	if openBrackets > closeBrackets {
		missing := openBrackets - closeBrackets
		output += strings.Repeat("]", missing)
		fixes = append(fixes, fmt.Sprintf("added_%d_missing_closing_brackets", missing))
	}

	// Remove extra closing braces/brackets (conservative approach)
	if closeBraces > openBraces || closeBrackets > openBrackets {
		warnings = append(warnings, "detected_extra_closing_braces_or_brackets")
	}

	if output != input {
		return output, fixes, warnings
	}

	return input, fixes, warnings
}

func (p *JSONRecoveryParser) reconstructFromParts(input string) (string, []string, []string) {
	fixes := []string{}
	warnings := []string{}

	// This is an aggressive strategy that tries to reconstruct JSON from fragments
	// Look for key-value pairs and try to build an object

	// Find quoted strings that could be keys or values
	keyValueRegex := regexp.MustCompile(`"([^"]+)"\s*:\s*("(?:[^"\\]|\\.)*"|[0-9]+(?:\.[0-9]+)?|true|false|null)`)
	matches := keyValueRegex.FindAllStringSubmatch(input, -1)

	if len(matches) > 0 {
		// Try to build a JSON object from found key-value pairs
		var pairs []string
		for _, match := range matches {
			if len(match) >= 3 {
				pairs = append(pairs, fmt.Sprintf(`"%s":%s`, match[1], match[2]))
			}
		}

		if len(pairs) > 0 {
			reconstructed := "{" + strings.Join(pairs, ",") + "}"

			// Validate the reconstructed JSON
			var testData interface{}
			if err := json.Unmarshal([]byte(reconstructed), &testData); err == nil {
				fixes = append(fixes, "reconstructed_from_key_value_pairs")
				warnings = append(warnings, "aggressive_reconstruction_applied")
				return reconstructed, fixes, warnings
			}
		}
	}

	return input, fixes, warnings
}

func (p *JSONRecoveryParser) inferMissingFields(input string) (string, []string, []string) {
	fixes := []string{}
	warnings := []string{}

	// This is the most aggressive strategy - try to infer missing structure
	// Only apply for very specific common patterns

	// If input starts with a quote but no opening brace, assume it should be an object
	trimmed := strings.TrimSpace(input)
	if strings.HasPrefix(trimmed, `"`) && strings.Contains(trimmed, ":") && !strings.HasPrefix(trimmed, "{") {
		reconstructed := "{" + trimmed + "}"

		var testData interface{}
		if err := json.Unmarshal([]byte(reconstructed), &testData); err == nil {
			fixes = append(fixes, "inferred_missing_object_wrapper")
			warnings = append(warnings, "aggressive_structure_inference_applied")
			return reconstructed, fixes, warnings
		}
	}

	// If input looks like an array of values but missing brackets
	if strings.Contains(trimmed, ",") && !strings.HasPrefix(trimmed, "[") && !strings.HasPrefix(trimmed, "{") {
		reconstructed := "[" + trimmed + "]"

		var testData interface{}
		if err := json.Unmarshal([]byte(reconstructed), &testData); err == nil {
			fixes = append(fixes, "inferred_missing_array_wrapper")
			warnings = append(warnings, "aggressive_structure_inference_applied")
			return reconstructed, fixes, warnings
		}
	}

	return input, fixes, warnings
}

// FormatRecoveryReport generates a human-readable report of the recovery process
func (r *RecoveryResult) FormatRecoveryReport() string {
	var sb strings.Builder

	sb.WriteString("=== JSON Recovery Report ===\n\n")

	if r.IsValid {
		sb.WriteString("✓ Successfully recovered valid JSON\n")
	} else {
		sb.WriteString("✗ Failed to recover valid JSON\n")
	}

	sb.WriteString(fmt.Sprintf("Recovery Level: %v\n", r.RecoveryLevel))
	sb.WriteString(fmt.Sprintf("Applied Fixes: %d\n", len(r.AppliedFixes)))
	sb.WriteString(fmt.Sprintf("Warnings: %d\n", len(r.Warnings)))

	if len(r.AppliedFixes) > 0 {
		sb.WriteString("\nFixes Applied:\n")
		for i, fix := range r.AppliedFixes {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, fix))
		}
	}

	if len(r.Warnings) > 0 {
		sb.WriteString("\nWarnings:\n")
		for i, warning := range r.Warnings {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, warning))
		}
	}

	if r.RecoveredJSON != r.OriginalInput {
		sb.WriteString("\nOriginal vs Recovered:\n")
		sb.WriteString(fmt.Sprintf("Original Length: %d chars\n", len(r.OriginalInput)))
		sb.WriteString(fmt.Sprintf("Recovered Length: %d chars\n", len(r.RecoveredJSON)))

		// Show first difference
		for i := 0; i < len(r.OriginalInput) && i < len(r.RecoveredJSON); i++ {
			if r.OriginalInput[i] != r.RecoveredJSON[i] {
				start := max(0, i-20)
				originalEnd := min(len(r.OriginalInput), i+20)
				recoveredEnd := min(len(r.RecoveredJSON), i+20)

				// Ensure start positions are valid for both strings
				originalStart := min(start, len(r.OriginalInput))
				recoveredStart := min(start, len(r.RecoveredJSON))

				sb.WriteString(fmt.Sprintf("First difference at position %d:\n", i))
				if originalStart < originalEnd {
					sb.WriteString(fmt.Sprintf("Original: ...%s...\n", r.OriginalInput[originalStart:originalEnd]))
				}
				if recoveredStart < recoveredEnd {
					sb.WriteString(fmt.Sprintf("Recovered: ...%s...\n", r.RecoveredJSON[recoveredStart:recoveredEnd]))
				}
				break
			}
		}
	}

	return sb.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}