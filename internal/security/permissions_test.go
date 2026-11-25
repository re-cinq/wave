package security

import (
	"testing"
)

// =============================================================================
// Persona Permission Security Tests
//
// These tests verify security aspects of the permission model including:
// - Deny pattern enforcement
// - Permission validation for artifact creation
// - Security boundary enforcement
// =============================================================================

// TestSchemaValidationResult_SecurityFlags tests adding security flags
// to validation results.
func TestSchemaValidationResult_SecurityFlags(t *testing.T) {
	result := NewSchemaValidationResult("artifact.json", "/workspace/artifact.json", "{}", true)

	// Initially no flags
	if len(result.SecurityFlags) != 0 {
		t.Errorf("initially should have no flags, got %d", len(result.SecurityFlags))
	}

	// Add a flag
	result.AddSecurityFlag("write_permission_required")
	if len(result.SecurityFlags) != 1 {
		t.Error("should have 1 flag after adding")
	}

	// Add another flag
	result.AddSecurityFlag("artifact_creation")
	if len(result.SecurityFlags) != 2 {
		t.Error("should have 2 flags after adding another")
	}

	// Verify flags are preserved
	foundWriteFlag := false
	foundArtifactFlag := false
	for _, flag := range result.SecurityFlags {
		if flag == "write_permission_required" {
			foundWriteFlag = true
		}
		if flag == "artifact_creation" {
			foundArtifactFlag = true
		}
	}

	if !foundWriteFlag || !foundArtifactFlag {
		t.Error("security flags not preserved correctly")
	}
}

// TestSecurityConfig_ValidationRequired tests that security configuration
// validation catches invalid settings.
func TestSecurityConfig_ValidationRequired(t *testing.T) {
	testCases := []struct {
		name        string
		modifier    func(*SecurityConfig)
		expectError bool
	}{
		{
			name:        "valid default config",
			modifier:    func(c *SecurityConfig) {},
			expectError: false,
		},
		{
			name: "invalid max path length",
			modifier: func(c *SecurityConfig) {
				c.PathValidation.MaxPathLength = -1
			},
			expectError: true,
		},
		{
			name: "invalid max input length",
			modifier: func(c *SecurityConfig) {
				c.Sanitization.MaxInputLength = 0
			},
			expectError: true,
		},
		{
			name: "invalid content size limit",
			modifier: func(c *SecurityConfig) {
				c.Sanitization.ContentSizeLimit = -100
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := DefaultSecurityConfig()
			tc.modifier(config)

			err := config.Validate()

			if tc.expectError && err == nil {
				t.Error("expected validation error")
			}
			if !tc.expectError && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}
		})
	}
}

// TestInputSanitizer_PersonaNameValidation tests that persona names are
// properly sanitized.
func TestInputSanitizer_PersonaNameValidation(t *testing.T) {
	config := DefaultSecurityConfig()
	logger := NewSecurityLogger(false) // Disable logging for tests
	sanitizer := NewInputSanitizer(*config, logger)

	testCases := []struct {
		name          string
		personaName   string
		expectChanges bool
		expectError   bool
	}{
		{
			name:          "valid persona name",
			personaName:   "implementer",
			expectChanges: false,
			expectError:   false,
		},
		{
			name:          "valid persona with hyphen",
			personaName:   "github-analyst",
			expectChanges: false,
			expectError:   false,
		},
		{
			name:          "persona name too long",
			personaName:   string(make([]byte, 20000)),
			expectChanges: true,
			expectError:   false, // Will be truncated, not error
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			record, _, err := sanitizer.SanitizeInput(tc.personaName, "persona_name")

			if tc.expectError && err == nil {
				t.Error("expected error")
			}
			if !tc.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if record != nil && tc.expectChanges != record.ChangesDetected {
				t.Errorf("changes detected: got %v, want %v",
					record.ChangesDetected, tc.expectChanges)
			}
		})
	}
}

// TestSecurityLogger_LogsViolations tests that the security logger properly
// records violations (without actually logging when disabled).
func TestSecurityLogger_LogsViolations(t *testing.T) {
	// Create disabled logger to avoid output during tests
	logger := NewSecurityLogger(false)

	// Should not panic when logging with disabled logger
	logger.LogViolation(
		string(ViolationInvalidPersona),
		string(SourceMetaPipeline),
		"test violation",
		SeverityHigh,
		true,
	)

	logger.LogSanitization("persona", true, 50)
	logger.LogPathValidation("artifact.json", "/workspace/artifact.json", []string{"validated"})

	// If we get here without panic, the test passes
}

// TestGenerateEventID_Uniqueness tests that event IDs are unique.
func TestGenerateEventID_Uniqueness(t *testing.T) {
	ids := make(map[string]bool)
	const numIDs = 1000

	for i := 0; i < numIDs; i++ {
		id := GenerateEventID()

		if id == "" {
			t.Error("generated empty ID")
		}

		if ids[id] {
			t.Errorf("duplicate ID generated: %s", id)
		}
		ids[id] = true

		// Verify ID format (should start with "sec-")
		if len(id) < 4 || id[:4] != "sec-" {
			t.Errorf("ID should start with 'sec-', got: %s", id)
		}
	}
}

// TestPathSanitization_ForDisplay tests path sanitization for safe display.
func TestPathSanitization_ForDisplay(t *testing.T) {
	testCases := []struct {
		path   string
		maxLen int
	}{
		{"artifact.json", 50},
		{"a/very/long/path/that/exceeds/fifty/characters/and/should/be/truncated/artifact.json", 50},
	}

	for _, tc := range testCases {
		result := SanitizePathForLogging(tc.path)

		// Short paths should be returned as-is
		if len(tc.path) <= tc.maxLen {
			if result != tc.path {
				t.Errorf("short path should be unchanged: got %q, want %q", result, tc.path)
			}
		} else {
			// Long paths should be sanitized
			if result == tc.path {
				t.Error("long path should be sanitized")
			}
		}
	}
}
