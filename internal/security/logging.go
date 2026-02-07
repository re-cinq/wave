package security

import (
	"fmt"
	"os"
	"time"
)

// SecurityLogger provides structured security event logging
type SecurityLogger struct {
	enabled bool
}

// NewSecurityLogger creates a new security logger instance
func NewSecurityLogger(enabled bool) *SecurityLogger {
	return &SecurityLogger{enabled: enabled}
}

// LogViolation logs a security violation event with sanitized details
func (sl *SecurityLogger) LogViolation(violationType, source, sanitizedDetails string, severity Severity, blocked bool) {
	if !sl.enabled {
		return
	}

	event := SecurityViolationEvent{
		ID:                GenerateEventID(),
		Timestamp:         time.Now(),
		Type:              violationType,
		Source:            source,
		SanitizedDetails:  sanitizedDetails,
		Severity:          severity,
		Blocked:           blocked,
	}

	// Log to structured output (would integrate with Wave's existing logging)
	fmt.Fprintf(os.Stderr, "[SECURITY] %s: %s from %s - %s (blocked: %v)\n",
		event.Severity,
		event.Type,
		event.Source,
		event.SanitizedDetails,
		event.Blocked,
	)
}

// LogSanitization logs input sanitization actions
func (sl *SecurityLogger) LogSanitization(inputType string, changesDetected bool, riskScore int) {
	if !sl.enabled {
		return
	}

	fmt.Fprintf(os.Stderr, "[SECURITY] Input sanitized: type=%s changes=%v risk_score=%d\n",
		inputType,
		changesDetected,
		riskScore,
	)
}

// LogPathValidation logs path validation attempts
func (sl *SecurityLogger) LogPathValidation(requestedPath, validatedPath string, securityFlags []string) {
	if !sl.enabled {
		return
	}

	fmt.Fprintf(os.Stderr, "[SECURITY] Path validated: requested=%s validated=%s flags=%v\n",
		SanitizePathForLogging(requestedPath),
		SanitizePathForLogging(validatedPath),
		securityFlags,
	)
}

// SanitizePathForLogging removes sensitive information from paths for safe logging
func SanitizePathForLogging(path string) string {
	// Remove actual path content, just show structure
	if len(path) > 50 {
		return fmt.Sprintf("<%d chars>", len(path))
	}
	return path
}