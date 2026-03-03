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
	// AggressiveRecovery is deprecated and treated as ProgressiveRecovery
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
	// Treat AggressiveRecovery as ProgressiveRecovery
	if level > ProgressiveRecovery {
		level = ProgressiveRecovery
	}
	return &JSONRecoveryParser{
		RecoveryLevel:             level,
		MaxRecoveryAttempts:       10,
		PreserveOriginalOnFailure: true,
	}
}

// ParseWithRecovery attempts to parse JSON with progressive error recovery
func (p *JSONRecoveryParser) ParseWithRecovery(input string) (*RecoveryResult, error) {
	result := &RecoveryResult{
		OriginalInput: input,
		RecoveredJSON: input,
		RecoveryLevel: p.RecoveryLevel,
		AppliedFixes:  []string{},
		Warnings:      []string{},
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
		p.extractJSONFromText,
		p.extractFromMarkdown,
		p.removeComments,
		p.fixTrailingCommas,
		p.normalizeWhitespace,
	}

	if p.RecoveryLevel >= ProgressiveRecovery {
		strategies = append(strategies,
			p.quoteUnquotedKeys,
			p.fixSingleQuotes,
			p.fixMissingCommas,
		)
	}

	return strategies
}

// Recovery strategy implementations

// extractJSONFromText handles AI explanatory text, indicator phrases, and raw text
// surrounding JSON content. This is the primary extraction strategy that detects
// AI-generated explanatory prefixes and extracts JSON from surrounding text.
func (p *JSONRecoveryParser) extractJSONFromText(input string) (string, []string, []string) {
	fixes := []string{}
	warnings := []string{}

	trimmed := strings.TrimSpace(input)

	// Phase 1: Check for known AI indicator phrases followed by JSON
	indicatorPhrases := []string{
		"The enhanced data is:",
		"The results are:",
		"The data is:",
		"Here's the data:",
		"Here's the JSON:",
		"Output:",
		"Result:",
		"Data:",
		"Analysis:",
		"Let me extract just the JSON portion from what I read:",
		"I can provide the clean JSON output",
		"provide the corrected JSON output directly",
	}

	for _, phrase := range indicatorPhrases {
		if strings.Contains(trimmed, phrase) {
			phraseIndex := strings.Index(trimmed, phrase)
			remaining := strings.TrimSpace(trimmed[phraseIndex+len(phrase):])

			firstBrace := strings.IndexAny(remaining, "{[")
			if firstBrace != -1 {
				candidate := remaining[firstBrace:]
				if extracted, ok := findMatchingJSON(candidate); ok {
					fixes = append(fixes, "extracted_json_from_text")
					return extracted, fixes, warnings
				}
			}
		}
	}

	// Phase 2: Check for colon-based patterns where AI explains before outputting JSON
	colonPhrases := []string{
		"Based on the analysis",
		"clean JSON output",
		"provide clean JSON",
		"matches the schema",
		"required schema",
		"JSON format",
		"JSON structure",
	}

	for _, phrase := range colonPhrases {
		if strings.Contains(trimmed, phrase) {
			phraseIndex := strings.Index(trimmed, phrase)
			afterPhrase := trimmed[phraseIndex+len(phrase):]
			colonIndex := strings.Index(afterPhrase, ":")
			if colonIndex != -1 {
				afterColon := strings.TrimSpace(afterPhrase[colonIndex+1:])
				firstBrace := strings.IndexAny(afterColon, "{[")
				if firstBrace != -1 {
					candidate := afterColon[firstBrace:]
					if extracted, ok := findMatchingJSON(candidate); ok {
						fixes = append(fixes, "extracted_json_from_text")
						return extracted, fixes, warnings
					}
				}
			}
		}
	}

	// Phase 3: Look for explanatory paragraphs followed by JSON
	lines := strings.Split(trimmed, "\n")
	jsonStart := -1

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "{") || strings.HasPrefix(line, "[") {
			jsonStart = i
			break
		}
	}

	if jsonStart > 0 {
		jsonPart := strings.TrimSpace(strings.Join(lines[jsonStart:], "\n"))
		if extracted, ok := findMatchingJSON(jsonPart); ok {
			fixes = append(fixes, "extracted_json_from_text")
			return extracted, fixes, warnings
		}
	}

	// Phase 4: Find JSON object/array anywhere in the text
	firstBrace := strings.IndexAny(trimmed, "{[")
	if firstBrace != -1 {
		candidate := trimmed[firstBrace:]
		if extracted, ok := findMatchingJSON(candidate); ok {
			fixes = append(fixes, "extracted_json_from_text")
			return extracted, fixes, warnings
		}
		warnings = append(warnings, "found_json_structure_but_still_invalid")
	}

	return input, fixes, warnings
}

// findMatchingJSON finds a complete JSON object or array starting from the beginning
// of the input by tracking brace/bracket depth. Returns the extracted JSON and true
// if valid JSON was found.
func findMatchingJSON(input string) (string, bool) {
	if len(input) == 0 {
		return "", false
	}

	startChar := rune(input[0])
	var endChar rune
	if startChar == '{' {
		endChar = '}'
	} else if startChar == '[' {
		endChar = ']'
	} else {
		return "", false
	}

	depth := 0
	inString := false
	escaped := false

	for i, ch := range input {
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
			if ch == startChar {
				depth++
			} else if ch == endChar {
				depth--
				if depth == 0 {
					extracted := input[:i+1]
					var testData interface{}
					if err := json.Unmarshal([]byte(extracted), &testData); err == nil {
						return extracted, true
					}
					return "", false
				}
			}
		}
	}

	return "", false
}

func (p *JSONRecoveryParser) extractFromMarkdown(input string) (string, []string, []string) {
	fixes := []string{}
	warnings := []string{}

	// Handle markdown code blocks
	jsonBlockStart := strings.Index(input, "```json")
	if jsonBlockStart == -1 {
		jsonBlockStart = strings.Index(input, "```")
		if jsonBlockStart == -1 {
			return p.extractInlineCode(input, fixes, warnings)
		}
	}

	// Find the content after the opening ```
	startPos := jsonBlockStart
	for startPos < len(input) && input[startPos] != '\n' {
		startPos++
	}
	if startPos < len(input) {
		startPos++
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
		lineStart := candidate
		for lineStart > 0 && input[lineStart-1] != '\n' {
			lineStart--
		}

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

	singleLineRegex := regexp.MustCompile(`\s*//[^\n]*`)
	if singleLineRegex.MatchString(output) {
		output = singleLineRegex.ReplaceAllString(output, "")
		fixes = append(fixes, "removed_single_line_comments")
	}

	multiLineRegex := regexp.MustCompile(`(?s)/\*.*?\*/`)
	if multiLineRegex.MatchString(output) {
		output = multiLineRegex.ReplaceAllString(output, "")
		fixes = append(fixes, "removed_multi_line_comments")
	}

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

	normalized := strings.ReplaceAll(input, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	lines := strings.Split(normalized, "\n")
	for i, line := range lines {
		line = regexp.MustCompile(`[ \t]+`).ReplaceAllString(line, " ")
		lines[i] = strings.TrimSpace(line)
	}

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

	singleQuoteRegex := regexp.MustCompile(`'([^']*)'`)
	if singleQuoteRegex.MatchString(input) {
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

func (p *JSONRecoveryParser) fixMissingCommas(input string) (string, []string, []string) {
	fixes := []string{}
	warnings := []string{}

	missingCommaRegex := regexp.MustCompile(`("(?:[^"\\]|\\.)*")\s+("(?:[^"\\]|\\.)*"\s*:)`)
	if missingCommaRegex.MatchString(input) {
		output := missingCommaRegex.ReplaceAllString(input, "$1,$2")
		fixes = append(fixes, "added_missing_commas_between_properties")
		return output, fixes, warnings
	}

	missingArrayCommaRegex := regexp.MustCompile(`("(?:[^"\\]|\\.)*")\s+("(?:[^"\\]|\\.)*")`)
	if missingArrayCommaRegex.MatchString(input) {
		output := missingArrayCommaRegex.ReplaceAllString(input, "$1,$2")
		fixes = append(fixes, "added_missing_commas_between_array_elements")
		return output, fixes, warnings
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

		for i := 0; i < len(r.OriginalInput) && i < len(r.RecoveredJSON); i++ {
			if r.OriginalInput[i] != r.RecoveredJSON[i] {
				start := max(0, i-20)
				originalEnd := min(len(r.OriginalInput), i+20)
				recoveredEnd := min(len(r.RecoveredJSON), i+20)

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
