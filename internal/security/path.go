package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	// Clean the path to normalize it
	cleanedPath := filepath.Clean(requestedPath)

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