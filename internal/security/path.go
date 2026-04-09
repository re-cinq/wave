package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// PathValidator validates file paths for security
type PathValidator struct {
	config SecurityConfig
	logger *SecurityLogger
}

// NewPathValidator creates a new path validator
func NewPathValidator(config SecurityConfig, logger *SecurityLogger) *PathValidator {
	return &PathValidator{
		config: config,
		logger: logger,
	}
}

// ValidatePath validates a file path against security policies
func (pv *PathValidator) ValidatePath(requestedPath string) (*SchemaValidationResult, error) {
	result := NewSchemaValidationResult(requestedPath, "", "", false)

	// Unicode normalization: reject paths that contain Unicode homographs or
	// mixed-script sequences that could be used to bypass string comparisons.
	if err := pv.validateUnicode(requestedPath); err != nil {
		result.AddSecurityFlag("unicode_homograph")
		pv.logger.LogViolation(
			string(ViolationPathTraversal),
			string(SourceSchemaPath),
			fmt.Sprintf("Unicode homograph or encoding attack detected in path (length: %d)", len(requestedPath)),
			SeverityCritical,
			true,
		)
		return result, err
	}

	// NFC-normalise the path before all further checks so that visually
	// identical but differently-encoded Unicode paths are handled uniformly.
	normalizedPath := norm.NFC.String(requestedPath)

	// Clean the path to normalize it
	cleanedPath := filepath.Clean(normalizedPath)

	// Check for path traversal attempts
	if pv.containsTraversal(cleanedPath) {
		result.AddSecurityFlag("traversal_attempt")
		pv.logger.LogViolation(
			string(ViolationPathTraversal),
			string(SourceSchemaPath),
			fmt.Sprintf("Path traversal attempt detected in path (length: %d)", len(requestedPath)),
			SeverityCritical,
			true,
		)
		return result, NewPathTraversalError(requestedPath, pv.config.PathValidation.ApprovedDirectories)
	}

	// Check path length
	if len(cleanedPath) > pv.config.PathValidation.MaxPathLength {
		result.AddSecurityFlag("excessive_length")
		pv.logger.LogViolation(
			string(ViolationInputValidation),
			string(SourceSchemaPath),
			fmt.Sprintf("Path exceeds maximum length: %d > %d", len(cleanedPath), pv.config.PathValidation.MaxPathLength),
			SeverityHigh,
			true,
		)
		return result, NewInputValidationError("path", fmt.Sprintf("exceeds maximum length of %d", pv.config.PathValidation.MaxPathLength))
	}

	// Check if path is within approved directories
	validatedPath, err := pv.validateApprovedDirectory(cleanedPath)
	if err != nil {
		result.AddSecurityFlag("outside_approved_directories")
		pv.logger.LogViolation(
			string(ViolationPathTraversal),
			string(SourceSchemaPath),
			"Path outside approved directories",
			SeverityHigh,
			true,
		)
		return result, err
	}

	// Check for symbolic links if not allowed
	if !pv.config.PathValidation.AllowSymlinks {
		if pv.containsSymlinks(validatedPath) {
			result.AddSecurityFlag("symbolic_link")
			pv.logger.LogViolation(
				string(ViolationPathTraversal),
				string(SourceSchemaPath),
				"Symbolic link detected in path",
				SeverityMedium,
				true,
			)
			return result, NewPathTraversalError(requestedPath, pv.config.PathValidation.ApprovedDirectories)
		}
	}

	// Path validation successful
	result.ValidatedPath = validatedPath
	result.IsValid = true
	pv.logger.LogPathValidation(requestedPath, validatedPath, result.SecurityFlags)

	return result, nil
}

// validateUnicode checks for Unicode-based path attacks:
//   - UTF-7 encoded sequences (+AD4- style) that may bypass ASCII checks
//   - Mixed-script homograph attacks (e.g. Cyrillic 'а' alongside Latin 'a')
//   - Non-NFC input that might be crafted to fool comparison logic
func (pv *PathValidator) validateUnicode(path string) error {
	// UTF-7 detection: UTF-7 encodes characters as +<base64>- sequences.
	// This encoding is not used in filesystem paths and its presence strongly
	// indicates an encoding attack attempt.
	if strings.Contains(path, "+A") || strings.Contains(path, "+/") {
		// Only flag if the sequence looks like a real UTF-7 escape (+XX-)
		if containsUTF7Sequence(path) {
			return NewInputValidationError("path", "UTF-7 encoding detected")
		}
	}

	// Mixed-script homograph detection: iterate runes and collect Unicode
	// scripts.  A path containing characters from two or more incompatible
	// scripts (e.g. Latin + Cyrillic) is a homograph attack candidate.
	if err := detectMixedScript(path); err != nil {
		return err
	}

	return nil
}

// containsUTF7Sequence returns true if s contains a UTF-7 escape sequence of
// the form +<one-or-more-base64-chars>-.
// This is a heuristic: it matches the +<ALPHA/DIGIT/+//>+- pattern which
// covers all real UTF-7 encoded code points.
func containsUTF7Sequence(s string) bool {
	inSeq := false
	seqLen := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !inSeq {
			if c == '+' && i+1 < len(s) && s[i+1] != '-' {
				inSeq = true
				seqLen = 0
			}
		} else {
			if isBase64Char(c) {
				seqLen++
			} else if c == '-' {
				if seqLen > 0 {
					return true
				}
				inSeq = false
			} else {
				inSeq = false
			}
		}
	}
	return false
}

// isBase64Char returns true for characters that appear in base64 encoded data.
func isBase64Char(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') || c == '+' || c == '/'
}

// dominantScript represents the Unicode script family used for mixed-script
// detection.  We classify scripts into broad families to reduce false positives
// from legitimate multi-script content (e.g. CJK paths on Asian systems).
type dominantScript int

const (
	scriptNone  dominantScript = iota
	scriptLatin                // Latin, ASCII
	scriptCyrillic
	scriptGreek
	scriptArabic
	scriptHebrew
	scriptOther // All other scripts — CJK, Devanagari, etc.
)

// runeScript returns a coarse script classification for a rune.
func runeScript(r rune) dominantScript {
	switch {
	case r <= 0x007F: // Basic ASCII — treat as Latin
		return scriptLatin
	case unicode.Is(unicode.Latin, r):
		return scriptLatin
	case unicode.Is(unicode.Cyrillic, r):
		return scriptCyrillic
	case unicode.Is(unicode.Greek, r):
		return scriptGreek
	case unicode.Is(unicode.Arabic, r):
		return scriptArabic
	case unicode.Is(unicode.Hebrew, r):
		return scriptHebrew
	default:
		return scriptOther
	}
}

// detectMixedScript returns an error if path contains characters from more
// than one of the confusable-script families (Latin, Cyrillic, Greek, Arabic,
// Hebrew).  CJK and other scripts are not considered confusable with Latin
// in practice, so they do not trigger the check.
func detectMixedScript(path string) error {
	// confusableScripts are the scripts that can produce look-alike
	// characters for Latin.
	confusable := map[dominantScript]bool{
		scriptLatin:    false,
		scriptCyrillic: false,
		scriptGreek:    false,
		scriptArabic:   false,
		scriptHebrew:   false,
	}

	for _, r := range path {
		s := runeScript(r)
		if _, tracked := confusable[s]; tracked {
			confusable[s] = true
		}
	}

	// Count how many confusable script families are present.
	present := 0
	for _, seen := range confusable {
		if seen {
			present++
		}
	}

	if present >= 2 {
		return NewInputValidationError("path", "mixed-script Unicode homograph attack detected")
	}

	return nil
}

// containsTraversal checks for path traversal sequences
func (pv *PathValidator) containsTraversal(path string) bool {
	// Check for common path traversal patterns
	traversalPatterns := []string{
		"..",
		"./",
		"../",
		"..\\",
		".\\",
		"..\\\\",
	}

	for _, pattern := range traversalPatterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}

	// Check for encoded traversal attempts
	if strings.Contains(path, "%2e%2e") ||
		strings.Contains(path, "%252e%252e") ||
		strings.Contains(path, "..%2f") ||
		strings.Contains(path, "..%5c") {
		return true
	}

	return false
}

// validateApprovedDirectory checks if path is within approved directories
func (pv *PathValidator) validateApprovedDirectory(path string) (string, error) {
	if len(pv.config.PathValidation.ApprovedDirectories) == 0 {
		// If no approved directories configured, allow relative paths in current directory
		if filepath.IsAbs(path) {
			return "", NewPathTraversalError(path, []string{"relative paths only"})
		}
		return path, nil
	}

	// Convert to absolute path for comparison
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", NewInputValidationError("path", "cannot resolve absolute path")
	}

	// Check if path is within any approved directory
	for _, approvedDir := range pv.config.PathValidation.ApprovedDirectories {
		approvedAbs, err := filepath.Abs(approvedDir)
		if err != nil {
			continue
		}

		// Check if the path is within the approved directory
		if pv.isWithinDirectory(absPath, approvedAbs) {
			return absPath, nil
		}

		// Also check relative path matching
		if strings.HasPrefix(path, approvedDir) {
			return absPath, nil
		}
	}

	return "", NewPathTraversalError(path, pv.config.PathValidation.ApprovedDirectories)
}

// isWithinDirectory checks if a path is within a directory
func (pv *PathValidator) isWithinDirectory(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}

	// If the relative path starts with "..", it's outside the directory
	return !strings.HasPrefix(rel, "..")
}

// containsSymlinks checks if the path contains symbolic links
func (pv *PathValidator) containsSymlinks(path string) bool {
	// Check each component of the path
	parts := strings.Split(path, string(filepath.Separator))
	currentPath := ""

	for i, part := range parts {
		if i == 0 {
			currentPath = part
		} else {
			currentPath = filepath.Join(currentPath, part)
		}

		// Check if current path component is a symbolic link
		if info, err := os.Lstat(currentPath); err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return true
			}
		}
	}

	return false
}

// SanitizePathForDisplay removes sensitive information for display/logging
func (pv *PathValidator) SanitizePathForDisplay(path string) string {
	// For security, don't show the actual path content in errors
	if len(path) > 50 {
		return fmt.Sprintf("<path:%d chars>", len(path))
	}
	// Remove any parent directory references for safety
	return strings.ReplaceAll(path, "..", "[..]")
}
