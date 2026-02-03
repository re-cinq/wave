package contract

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// JSONCleaner provides utilities for fixing and validating JSON output
type JSONCleaner struct{}

// CleanJSONOutput attempts to fix common JSON formatting issues while preserving content
// This is useful for AI-generated JSON that may have minor syntax errors
func (jc *JSONCleaner) CleanJSONOutput(input string) (string, []string, error) {
	changes := []string{}

	// First, try to parse as-is
	var test interface{}
	if err := json.Unmarshal([]byte(input), &test); err == nil {
		// Already valid JSON
		return input, changes, nil
	}

	content := input

	// Fix unquoted property names (common in AI-generated JSON)
	// This is a limited fix for simple cases like: {name: "value"} -> {"name": "value"}
	// Only apply if it looks like it might be an object
	if strings.TrimSpace(content)[0] == '{' {
		// Pattern for unquoted keys: word: followed by quote or { or [
		unquotedKeyRegex := regexp.MustCompile(`(\{|,)\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*:`)
		if unquotedKeyRegex.MatchString(content) {
			content = unquotedKeyRegex.ReplaceAllString(content, `$1"$2":`)
			changes = append(changes, "quoted_unquoted_keys")
		}
	}

	// Remove single-line comments (// comment) - only at line level
	singleLineCommentRegex := regexp.MustCompile(`(?m)\s*//[^\n]*`)
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

	// Remove trailing commas before } or ]
	trailingCommaRegex := regexp.MustCompile(`,(\s*[}\]])`)
	if trailingCommaRegex.MatchString(content) {
		content = trailingCommaRegex.ReplaceAllString(content, "$1")
		changes = append(changes, "removed_trailing_commas")
	}

	// Fix single quotes around values (single-quoted strings instead of double-quoted)
	// Be careful: only fix if the JSON parser would fail otherwise
	// Pattern: 'string' that's not inside a double-quoted string
	singleQuoteRegex := regexp.MustCompile(`'([^']*)'`)
	if singleQuoteRegex.MatchString(content) {
		// Check if double quotes would parse better
		testWithDoubleQuotes := singleQuoteRegex.ReplaceAllString(content, `"$1"`)
		var testParse interface{}
		if err := json.Unmarshal([]byte(testWithDoubleQuotes), &testParse); err == nil {
			content = testWithDoubleQuotes
			changes = append(changes, "converted_single_quotes_to_double_quotes")
		}
	}

	// Normalize whitespace but preserve structure
	// Split by lines to preserve multiline strings (they should already be \n escaped)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		// Collapse multiple spaces/tabs to single space, but preserve structure
		line = regexp.MustCompile(`[ \t]+`).ReplaceAllString(line, " ")
		lines[i] = strings.TrimRight(line, " \t")
	}
	content = strings.Join(lines, "\n")
	content = strings.TrimSpace(content)

	// Final validation
	if err := json.Unmarshal([]byte(content), &test); err != nil {
		return input, changes, fmt.Errorf("JSON still invalid after cleaning: %w", err)
	}

	return content, changes, nil
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
	text = strings.TrimSpace(text)

	// Find the first { or [ and match it to the corresponding } or ]
	firstBrace := strings.IndexAny(text, "{[")
	if firstBrace == -1 {
		return "", fmt.Errorf("no JSON object or array found in text")
	}

	jsonStartChar := text[firstBrace]
	var jsonEndChar rune
	if jsonStartChar == '{' {
		jsonEndChar = '}'
	} else {
		jsonEndChar = ']'
	}

	// Simple bracket matching (handles nested structures)
	depth := 0
	inString := false
	escaped := false
	endPos := -1

	for i, ch := range text[firstBrace:] {
		if escaped {
			escaped = false
			continue
		}

		if ch == '\\' && inString {
			escaped = true
			continue
		}

		if ch == '"' {
			inString = !inString
			continue
		}

		if !inString {
			if rune(text[firstBrace+i]) == rune(jsonStartChar) {
				depth++
			} else if rune(text[firstBrace+i]) == jsonEndChar {
				depth--
				if depth == 0 {
					endPos = firstBrace + i + 1
					break
				}
			}
		}
	}

	if endPos == -1 {
		return "", fmt.Errorf("unmatched JSON braces")
	}

	jsonStr := text[firstBrace:endPos]

	// Validate the extracted JSON
	if !jc.IsValidJSON(jsonStr) {
		return "", fmt.Errorf("extracted text is not valid JSON")
	}

	return jsonStr, nil
}
