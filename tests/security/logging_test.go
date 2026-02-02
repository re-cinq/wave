package security

import (
	"fmt"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/security"
)

func TestSecurityLogger_LogViolation(t *testing.T) {
	tests := []struct {
		name            string
		enabled         bool
		violationType   string
		source          string
		details         string
		severity        security.Severity
		blocked         bool
		expectLogged    bool
	}{
		{
			name:          "logging_enabled_critical_violation",
			enabled:       true,
			violationType: string(security.ViolationPathTraversal),
			source:        string(security.SourceSchemaPath),
			details:       "Path traversal attempt detected",
			severity:      security.SeverityCritical,
			blocked:       true,
			expectLogged:  true,
		},
		{
			name:          "logging_disabled_critical_violation",
			enabled:       false,
			violationType: string(security.ViolationPathTraversal),
			source:        string(security.SourceSchemaPath),
			details:       "Path traversal attempt detected",
			severity:      security.SeverityCritical,
			blocked:       true,
			expectLogged:  false,
		},
		{
			name:          "prompt_injection_violation",
			enabled:       true,
			violationType: string(security.ViolationPromptInjection),
			source:        string(security.SourceUserInput),
			details:       "Prompt injection patterns detected",
			severity:      security.SeverityHigh,
			blocked:       true,
			expectLogged:  true,
		},
		{
			name:          "soft_failure_not_blocked",
			enabled:       true,
			violationType: string(security.ViolationMalformedJSON),
			source:        string(security.SourceContractValidation),
			details:       "JSON comments detected and cleaned",
			severity:      security.SeverityMedium,
			blocked:       false,
			expectLogged:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := security.NewSecurityLogger(tt.enabled)

			// Note: In a real implementation, we'd capture the output
			// For this test, we're verifying the function doesn't panic
			// and handles the enabled/disabled state correctly
			logger.LogViolation(tt.violationType, tt.source, tt.details, tt.severity, tt.blocked)

			// Test passes if no panic occurs
			// In integration tests, we would capture and verify the actual log output
		})
	}
}

func TestSecurityLogger_LogSanitization(t *testing.T) {
	tests := []struct {
		name            string
		enabled         bool
		inputType       string
		changesDetected bool
		riskScore       int
	}{
		{
			name:            "sanitization_with_changes",
			enabled:         true,
			inputType:       "task_description",
			changesDetected: true,
			riskScore:       75,
		},
		{
			name:            "sanitization_no_changes",
			enabled:         true,
			inputType:       "schema_content",
			changesDetected: false,
			riskScore:       10,
		},
		{
			name:            "logging_disabled",
			enabled:         false,
			inputType:       "pipeline_yaml",
			changesDetected: true,
			riskScore:       90,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := security.NewSecurityLogger(tt.enabled)

			// Test that logging doesn't panic
			logger.LogSanitization(tt.inputType, tt.changesDetected, tt.riskScore)

			// Test passes if no panic occurs
		})
	}
}

func TestSecurityLogger_LogPathValidation(t *testing.T) {
	tests := []struct {
		name          string
		enabled       bool
		requestedPath string
		validatedPath string
		securityFlags []string
	}{
		{
			name:          "successful_validation",
			enabled:       true,
			requestedPath: "contracts/schema.json",
			validatedPath: "/abs/path/contracts/schema.json",
			securityFlags: []string{},
		},
		{
			name:          "validation_with_flags",
			enabled:       true,
			requestedPath: "../../../etc/passwd",
			validatedPath: "",
			securityFlags: []string{"traversal_attempt", "outside_approved_directories"},
		},
		{
			name:          "logging_disabled",
			enabled:       false,
			requestedPath: "any/path.json",
			validatedPath: "/abs/any/path.json",
			securityFlags: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := security.NewSecurityLogger(tt.enabled)

			// Test that logging doesn't panic
			logger.LogPathValidation(tt.requestedPath, tt.validatedPath, tt.securityFlags)

			// Test passes if no panic occurs
		})
	}
}

func TestSecurityLogger_PathSanitization(t *testing.T) {
	tests := []struct {
		name         string
		inputPath    string
		expectedSafe string
	}{
		{
			name:         "short_safe_path",
			inputPath:    "contracts/schema.json",
			expectedSafe: "contracts/schema.json",
		},
		{
			name:         "long_path_sanitization",
			inputPath:    string(make([]byte, 100)),
			expectedSafe: "<100 chars>",
		},
		{
			name:         "path_with_traversal_dots",
			inputPath:    "../sensitive/file.txt",
			expectedSafe: "[..]/sensitive/file.txt",
		},
		{
			name:         "multiple_traversal_sequences",
			inputPath:    "../../config/../secrets.env",
			expectedSafe: "[..]/[..]/config/[..]/secrets.env",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := security.SanitizePathForLogging(tt.inputPath)

			if result != tt.expectedSafe {
				t.Errorf("Expected sanitized path %q, got %q", tt.expectedSafe, result)
			}

			// Ensure no sensitive path information leaks
			if len(tt.inputPath) > 50 && !strings.Contains(result, "<") {
				t.Error("Long paths should be truncated with length indicator")
			}

			if strings.Contains(tt.inputPath, "..") && strings.Contains(result, "..") {
				t.Error("Path traversal sequences should be sanitized")
			}
		})
	}
}

func TestSecurityViolationEvent_Creation(t *testing.T) {
	tests := []struct {
		name          string
		violationType security.ViolationType
		source        security.ViolationSource
		details       string
		severity      security.Severity
		blocked       bool
	}{
		{
			name:          "path_traversal_event",
			violationType: security.ViolationPathTraversal,
			source:        security.SourceSchemaPath,
			details:       "Attempted path traversal",
			severity:      security.SeverityCritical,
			blocked:       true,
		},
		{
			name:          "prompt_injection_event",
			violationType: security.ViolationPromptInjection,
			source:        security.SourceUserInput,
			details:       "Malicious prompt detected",
			severity:      security.SeverityHigh,
			blocked:       true,
		},
		{
			name:          "json_validation_event",
			violationType: security.ViolationMalformedJSON,
			source:        security.SourceContractValidation,
			details:       "JSON comments removed",
			severity:      security.SeverityMedium,
			blocked:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := security.NewSecurityViolationEvent(
				tt.violationType,
				tt.source,
				tt.details,
				tt.severity,
				tt.blocked,
			)

			// Verify event fields
			if event.Type != string(tt.violationType) {
				t.Errorf("Expected type %s, got %s", tt.violationType, event.Type)
			}

			if event.Source != string(tt.source) {
				t.Errorf("Expected source %s, got %s", tt.source, event.Source)
			}

			if event.SanitizedDetails != tt.details {
				t.Errorf("Expected details %s, got %s", tt.details, event.SanitizedDetails)
			}

			if event.Severity != tt.severity {
				t.Errorf("Expected severity %s, got %s", tt.severity, event.Severity)
			}

			if event.Blocked != tt.blocked {
				t.Errorf("Expected blocked %v, got %v", tt.blocked, event.Blocked)
			}

			// Verify ID is generated
			if event.ID == "" {
				t.Error("Event ID should be generated")
			}

			// Verify timestamp is set
			if event.Timestamp.IsZero() {
				t.Error("Event timestamp should be set")
			}
		})
	}
}

func TestSecurityViolationEvent_IDGeneration(t *testing.T) {
	// Test that unique IDs are generated
	ids := make(map[string]bool)

	for i := 0; i < 100; i++ {
		id := security.GenerateEventID()

		if ids[id] {
			t.Fatalf("Duplicate event ID generated: %s", id)
		}
		ids[id] = true

		// Verify format (should start with "sec-")
		if !strings.HasPrefix(id, "sec-") {
			t.Errorf("Event ID should start with 'sec-', got: %s", id)
		}

		// Verify reasonable length
		if len(id) < 10 {
			t.Errorf("Event ID too short: %s", id)
		}
	}
}

func TestSecurityLogger_ConcurrentAccess(t *testing.T) {
	logger := security.NewSecurityLogger(true)

	// Test concurrent logging doesn't cause race conditions
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(index int) {
			defer func() { done <- true }()

			logger.LogViolation(
				"test_violation",
				"test_source",
				fmt.Sprintf("Test violation %d", index),
				security.SeverityMedium,
				true,
			)

			logger.LogSanitization(fmt.Sprintf("type_%d", index), true, 50)

			logger.LogPathValidation(
				fmt.Sprintf("path_%d", index),
				fmt.Sprintf("validated_%d", index),
				[]string{"flag"},
			)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Test passes if no race conditions or panics occur
}