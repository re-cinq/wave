package recovery

import "strings"

// ShellEscape escapes a string for safe use in a POSIX shell command.
// It uses single-quote wrapping with interior single-quote escaping.
// Empty strings return ''. Strings without special characters are returned as-is.
func ShellEscape(s string) string {
	if s == "" {
		return "''"
	}

	// If the string contains no special characters, return it as-is
	safe := true
	for _, c := range s {
		if !isShellSafe(c) {
			safe = false
			break
		}
	}
	if safe {
		return s
	}

	// Wrap in single quotes, escaping interior single quotes
	// 'it'\''s' → closes quote, adds escaped single quote, reopens quote
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// isShellSafe returns true if the character is safe to use unquoted in a shell.
func isShellSafe(c rune) bool {
	if c >= 'a' && c <= 'z' {
		return true
	}
	if c >= 'A' && c <= 'Z' {
		return true
	}
	if c >= '0' && c <= '9' {
		return true
	}
	switch c {
	case '-', '_', '.', '/', ':', ',', '+', '=':
		return true
	}
	return false
}
