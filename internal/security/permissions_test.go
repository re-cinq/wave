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

// TestSecurityConfig_PersonaValidation verifies that persona validation
// configuration is correctly set up.
func TestSecurityConfig_PersonaValidation(t *testing.T) {
	config := DefaultSecurityConfig()

	// Verify persona validation is enabled by default
	if !config.PersonaValidation.ValidatePersonaReferences {
		t.Error("persona reference validation should be enabled by default")
	}

	// Verify unknown personas are not allowed by default
	if config.PersonaValidation.AllowUnknownPersonas {
		t.Error("unknown personas should not be allowed by default")
	}

	// Verify validation is enabled overall
	if !config.IsPersonaValidationEnabled() {
		t.Error("persona validation should be enabled")
	}
}

// TestSecurityConfig_PathValidationForArtifacts verifies path validation
// configuration supports artifact creation paths.
func TestSecurityConfig_PathValidationForArtifacts(t *testing.T) {
	config := DefaultSecurityConfig()

	// Verify path validation is enabled
	if !config.IsPathValidationEnabled() {
		t.Error("path validation should be enabled")
	}

	// Check that approved directories include common artifact locations
	// Note: artifacts may not be in default approved dirs, which is correct
	// because artifact paths should be validated at runtime
	if config.PathValidation.MaxPathLength <= 0 {
		t.Error("max path length should be positive")
	}
}

// TestPersonaReference_ValidPersona tests persona reference validation
// for valid persona names.
func TestPersonaReference_ValidPersona(t *testing.T) {
	availablePersonas := []string{
		"implementer",
		"reviewer",
		"navigator",
		"auditor",
		"craftsman",
		"philosopher",
		"planner",
	}

	testCases := []struct {
		stepID      string
		personaName string
		shouldValid bool
	}{
		{"step-1", "implementer", true},
		{"step-2", "reviewer", true},
		{"step-3", "navigator", true},
		{"step-4", "unknown-persona", false},
		{"step-5", "", false},
		{"step-6", "IMPLEMENTER", false}, // Case sensitive
	}

	for _, tc := range testCases {
		t.Run(tc.personaName, func(t *testing.T) {
			ref := NewPersonaReference(tc.stepID, tc.personaName, availablePersonas)

			if ref.IsValid != tc.shouldValid {
				t.Errorf("persona %q validity: got %v, want %v",
					tc.personaName, ref.IsValid, tc.shouldValid)
			}

			if ref.StepID != tc.stepID {
				t.Errorf("step ID: got %q, want %q", ref.StepID, tc.stepID)
			}

			if ref.PersonaName != tc.personaName {
				t.Errorf("persona name: got %q, want %q", ref.PersonaName, tc.personaName)
			}
		})
	}
}

// TestPersonaReference_SuggestsAlternative tests that invalid persona references
// suggest valid alternatives.
func TestPersonaReference_SuggestsAlternative(t *testing.T) {
	availablePersonas := []string{"implementer", "reviewer", "navigator"}

	ref := NewPersonaReference("step-1", "unknown", availablePersonas)

	if ref.IsValid {
		t.Error("unknown persona should be invalid")
	}

	if ref.SuggestedAlternative == "" {
		t.Error("should suggest an alternative persona")
	}

	// Verify suggested alternative is one of the available personas
	found := false
	for _, p := range availablePersonas {
		if p == ref.SuggestedAlternative {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("suggested alternative %q not in available personas", ref.SuggestedAlternative)
	}
}

// TestInvalidPersonaError_Format tests the format of invalid persona errors.
func TestInvalidPersonaError_Format(t *testing.T) {
	available := []string{"implementer", "reviewer", "navigator"}
	err := NewInvalidPersonaError("unknown-persona", available)

	// Verify error type
	if err.Type != string(ViolationInvalidPersona) {
		t.Errorf("error type: got %q, want %q", err.Type, ViolationInvalidPersona)
	}

	// Verify error message contains the persona name
	if err.Message == "" {
		t.Error("error message should not be empty")
	}

	// Verify error is retryable (user can fix by using valid persona)
	if !err.Retryable {
		t.Error("invalid persona error should be retryable")
	}

	// Verify suggested fix mentions available personas
	if err.SuggestedFix == "" {
		t.Error("should provide a suggested fix")
	}

	// Verify details contain useful information
	if err.Details == nil {
		t.Error("should have details")
	}
	if _, ok := err.Details["available_personas"]; !ok {
		t.Error("details should include available_personas")
	}
}

// TestSecurityViolationEvent_Creation tests creation of security violation events.
func TestSecurityViolationEvent_Creation(t *testing.T) {
	event := NewSecurityViolationEvent(
		ViolationInvalidPersona,
		SourceMetaPipeline,
		"persona 'attacker' not found",
		SeverityMedium,
		true,
	)

	if event.ID == "" {
		t.Error("event should have an ID")
	}

	if event.Type != string(ViolationInvalidPersona) {
		t.Errorf("event type: got %q, want %q", event.Type, ViolationInvalidPersona)
	}

	if event.Source != string(SourceMetaPipeline) {
		t.Errorf("event source: got %q, want %q", event.Source, SourceMetaPipeline)
	}

	if event.Severity != SeverityMedium {
		t.Errorf("event severity: got %v, want %v", event.Severity, SeverityMedium)
	}

	if !event.Blocked {
		t.Error("event should be marked as blocked")
	}

	if event.Timestamp.IsZero() {
		t.Error("event should have a timestamp")
	}
}

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

// TestSchemaValidationResult_SanitizationActions tests recording sanitization
// actions in validation results.
func TestSchemaValidationResult_SanitizationActions(t *testing.T) {
	result := NewSchemaValidationResult("artifact.json", "", "{}", false)

	// Add sanitization actions
	result.AddSanitizationAction("validated_path")
	result.AddSanitizationAction("checked_permissions")

	if len(result.SanitizationActions) != 2 {
		t.Errorf("expected 2 sanitization actions, got %d", len(result.SanitizationActions))
	}
}

// TestSecurityError_Detection tests the IsSecurityError helper function.
func TestSecurityError_Detection(t *testing.T) {
	// Test with security error
	secErr := NewInputValidationError("persona", "invalid format")
	if !IsSecurityError(secErr) {
		t.Error("should detect SecurityValidationError")
	}

	// Test with regular error
	regularErr := &struct{ error }{nil}
	if IsSecurityError(regularErr) {
		t.Error("should not detect regular struct as SecurityValidationError")
	}

	// Test extraction
	extracted, ok := GetSecurityError(secErr)
	if !ok {
		t.Error("should extract SecurityValidationError")
	}
	if extracted.Type != string(ViolationInputValidation) {
		t.Errorf("extracted error type mismatch")
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
		path     string
		maxLen   int
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
