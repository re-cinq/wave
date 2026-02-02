package adapter

import (
	"fmt"
	"path/filepath"
	"strings"
)

// PermissionChecker validates tool operations against allow/deny patterns.
// It enforces deny-first precedence: if a deny pattern matches, the operation
// is blocked regardless of any allow patterns.
type PermissionChecker struct {
	PersonaName  string
	AllowedTools []string
	DenyTools    []string
}

// PermissionError represents a permission denied error with contextual information.
type PermissionError struct {
	PersonaName string
	Tool        string
	Argument    string
	Reason      string
}

func (e *PermissionError) Error() string {
	if e.Argument != "" {
		return fmt.Sprintf("permission denied: persona '%s' cannot use tool '%s' with argument '%s': %s",
			e.PersonaName, e.Tool, e.Argument, e.Reason)
	}
	return fmt.Sprintf("permission denied: persona '%s' cannot use tool '%s': %s",
		e.PersonaName, e.Tool, e.Reason)
}

// NewPermissionChecker creates a new PermissionChecker for a persona.
func NewPermissionChecker(personaName string, allowedTools, denyTools []string) *PermissionChecker {
	return &PermissionChecker{
		PersonaName:  personaName,
		AllowedTools: allowedTools,
		DenyTools:    denyTools,
	}
}

// CheckPermission validates whether a tool operation is permitted.
// It returns nil if the operation is allowed, or a PermissionError if denied.
//
// The check order is:
// 1. Check deny patterns first - if any match, operation is denied
// 2. Check allow patterns - if any match, operation is allowed
// 3. If no allow patterns defined, operation is allowed by default
// 4. If allow patterns exist but none match, operation is denied
func (pc *PermissionChecker) CheckPermission(tool string, argument string) error {
	// Step 1: Check deny patterns first (deny takes precedence)
	for _, denyPattern := range pc.DenyTools {
		if matchToolPattern(denyPattern, tool, argument) {
			return &PermissionError{
				PersonaName: pc.PersonaName,
				Tool:        tool,
				Argument:    argument,
				Reason:      fmt.Sprintf("blocked by deny pattern '%s'", denyPattern),
			}
		}
	}

	// Step 2: If no allow patterns, allow by default
	if len(pc.AllowedTools) == 0 {
		return nil
	}

	// Step 3: Check if any allow pattern matches
	for _, allowPattern := range pc.AllowedTools {
		if matchToolPattern(allowPattern, tool, argument) {
			return nil
		}
	}

	// Step 4: No allow pattern matched, deny
	return &PermissionError{
		PersonaName: pc.PersonaName,
		Tool:        tool,
		Argument:    argument,
		Reason:      "not in allowed tools list",
	}
}

// matchToolPattern checks if a tool invocation matches a permission pattern.
// Patterns can be:
// - Simple tool name: "Read", "Write"
// - Tool with glob argument: "Write(.wave/specs/*)", "Bash(git log*)"
// - Wildcard: "*" (matches any tool)
//
// The pattern format is: ToolName or ToolName(argumentPattern)
// Argument patterns support glob syntax: *, **, ?, [abc], [a-z]
func matchToolPattern(pattern, tool, argument string) bool {
	// Handle wildcard pattern
	if pattern == "*" {
		return true
	}

	// Parse pattern into tool and argument parts
	patternTool, patternArg := parseToolPattern(pattern)

	// Check if tool name matches
	if !matchGlob(patternTool, tool) {
		return false
	}

	// If pattern has no argument constraint, match any argument
	if patternArg == "" {
		return true
	}

	// Special case: (*) matches any argument
	if patternArg == "*" {
		return true
	}

	// Match argument against pattern
	return matchGlob(patternArg, argument)
}

// parseToolPattern splits a permission pattern into tool and argument parts.
// Examples:
//
//	"Read" -> ("Read", "")
//	"Write(*)" -> ("Write", "*")
//	"Bash(git log*)" -> ("Bash", "git log*")
//	"Write(.wave/specs/*)" -> ("Write", ".wave/specs/*")
func parseToolPattern(pattern string) (tool string, arg string) {
	openParen := strings.Index(pattern, "(")
	if openParen == -1 {
		return pattern, ""
	}

	closeParen := strings.LastIndex(pattern, ")")
	if closeParen == -1 || closeParen <= openParen {
		return pattern, ""
	}

	return pattern[:openParen], pattern[openParen+1 : closeParen]
}

// matchGlob performs glob-style pattern matching.
// Supports:
//   - * matches any sequence of characters
//   - ** matches any sequence including path separators
//   - ? matches any single character
//   - [abc] matches any character in the set
//   - [a-z] matches any character in the range
//
// For command-line arguments (containing spaces), we use a custom string-based
// matching that handles spaces correctly, since filepath.Match doesn't work
// well with spaces in patterns.
func matchGlob(pattern, text string) bool {
	// Handle empty pattern
	if pattern == "" {
		return text == ""
	}

	// Handle pure wildcard
	if pattern == "*" || pattern == "**" {
		return true
	}

	// Check for ** which matches path separators
	if strings.Contains(pattern, "**") {
		return matchDoubleStarGlob(pattern, text)
	}

	// For patterns containing spaces (command-line args), use string-based matching
	// since filepath.Match doesn't handle spaces well
	if strings.Contains(pattern, " ") || strings.Contains(text, " ") {
		return matchStringGlob(pattern, text)
	}

	// Use filepath.Match for file path patterns
	matched, err := filepath.Match(pattern, text)
	if err != nil {
		// If pattern is invalid, try string-based matching
		return matchStringGlob(pattern, text)
	}
	return matched
}

// matchStringGlob performs simple string-based glob matching.
// Handles * as wildcard that matches any sequence of characters.
// This is used for command-line argument patterns that may contain spaces.
func matchStringGlob(pattern, text string) bool {
	// No wildcards - exact match
	if !strings.Contains(pattern, "*") && !strings.Contains(pattern, "?") {
		return pattern == text
	}

	// Handle patterns ending with *
	if strings.HasSuffix(pattern, "*") && !strings.Contains(pattern[:len(pattern)-1], "*") {
		prefix := pattern[:len(pattern)-1]
		return strings.HasPrefix(text, prefix)
	}

	// Handle patterns starting with *
	if strings.HasPrefix(pattern, "*") && !strings.Contains(pattern[1:], "*") {
		suffix := pattern[1:]
		return strings.HasSuffix(text, suffix)
	}

	// Handle * in middle or multiple *s using recursive matching
	return matchWildcardRecursive(pattern, text)
}

// matchWildcardRecursive handles complex wildcard patterns recursively.
func matchWildcardRecursive(pattern, text string) bool {
	for len(pattern) > 0 {
		switch pattern[0] {
		case '*':
			// Skip consecutive *s
			for len(pattern) > 0 && pattern[0] == '*' {
				pattern = pattern[1:]
			}
			// If pattern ends with *, match rest of text
			if len(pattern) == 0 {
				return true
			}
			// Try matching remainder at each position
			for i := 0; i <= len(text); i++ {
				if matchWildcardRecursive(pattern, text[i:]) {
					return true
				}
			}
			return false
		case '?':
			if len(text) == 0 {
				return false
			}
			pattern = pattern[1:]
			text = text[1:]
		default:
			if len(text) == 0 || pattern[0] != text[0] {
				return false
			}
			pattern = pattern[1:]
			text = text[1:]
		}
	}
	return len(text) == 0
}

// matchDoubleStarGlob handles ** patterns that match across path separators.
func matchDoubleStarGlob(pattern, text string) bool {
	// Split on ** and match each segment
	parts := strings.Split(pattern, "**")
	if len(parts) == 1 {
		// No **, use standard matching
		matched, _ := filepath.Match(pattern, text)
		return matched
	}

	// For **-patterns, we need recursive matching
	// ** at start: suffix must match
	// ** at end: prefix must match
	// ** in middle: both prefix and suffix must match

	if len(parts) == 2 {
		prefix := parts[0]
		suffix := parts[1]

		// ** at start: check suffix
		if prefix == "" {
			if suffix == "" {
				return true
			}
			// Match suffix at any position
			suffix = strings.TrimPrefix(suffix, "/")
			if suffix == "" {
				return true
			}
			return matchSuffixGlob(suffix, text)
		}

		// ** at end: check prefix
		if suffix == "" {
			prefix = strings.TrimSuffix(prefix, "/")
			if prefix == "" {
				return true
			}
			return strings.HasPrefix(text, prefix)
		}

		// ** in middle: find any valid split point
		prefix = strings.TrimSuffix(prefix, "/")
		suffix = strings.TrimPrefix(suffix, "/")
		return matchMiddleStarGlob(prefix, suffix, text)
	}

	// Multiple ** segments - simplified recursive check
	return matchMultipleDoubleStars(parts, text)
}

// matchSuffixGlob matches when ** is at the start of the pattern.
func matchSuffixGlob(suffix, text string) bool {
	// Try to match suffix at end of text
	if len(text) < len(suffix) {
		// Text shorter than suffix - only match if suffix has wildcards
		matched, _ := filepath.Match(suffix, text)
		return matched
	}

	// Try exact suffix match
	textSuffix := text[len(text)-len(suffix):]
	matched, _ := filepath.Match(suffix, textSuffix)
	if matched {
		return true
	}

	// Try matching at any segment boundary
	for i := range text {
		if i == 0 || text[i-1] == '/' {
			remainder := text[i:]
			if m, _ := filepath.Match(suffix, remainder); m {
				return true
			}
		}
	}
	return false
}

// matchMiddleStarGlob handles ** in the middle of a pattern.
func matchMiddleStarGlob(prefix, suffix, text string) bool {
	if !strings.HasPrefix(text, prefix) {
		return false
	}

	remainder := text[len(prefix):]
	remainder = strings.TrimPrefix(remainder, "/")

	if suffix == "" {
		return true
	}

	return matchSuffixGlob(suffix, remainder)
}

// matchMultipleDoubleStars handles patterns with multiple ** segments.
func matchMultipleDoubleStars(parts []string, text string) bool {
	// Simplified: check first and last parts
	if len(parts) < 2 {
		return true
	}

	first := strings.TrimSuffix(parts[0], "/")
	if first != "" && !strings.HasPrefix(text, first) {
		return false
	}

	last := strings.TrimPrefix(parts[len(parts)-1], "/")
	if last != "" && !matchSuffixGlob(last, text) {
		return false
	}

	return true
}

// IsPermissionError checks if an error is a PermissionError.
func IsPermissionError(err error) bool {
	_, ok := err.(*PermissionError)
	return ok
}
