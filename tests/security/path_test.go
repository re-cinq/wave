package security

import (
	"fmt"
	"testing"
	"time"

	"github.com/recinq/wave/internal/security"
)

func TestPathValidator_ValidatePath_PathTraversalPrevention(t *testing.T) {
	testUtils := NewSecurityTestUtils(t)
	config := testUtils.CreateTestConfig()
	logger := testUtils.CreateTestLogger()
	validator := security.NewPathValidator(config, logger)

	tests := []struct {
		name           string
		requestedPath  string
		expectValid    bool
		expectedFlags  []string
		expectError    bool
	}{
		{
			name:           "valid_schema_path",
			requestedPath:  "testdata/contracts/user-schema.json",
			expectValid:    true,
			expectedFlags:  []string{},
			expectError:    false,
		},
		{
			name:           "path_traversal_basic",
			requestedPath:  "../../../etc/passwd",
			expectValid:    false,
			expectedFlags:  []string{"traversal_attempt"},
			expectError:    true,
		},
		{
			name:           "path_traversal_encoded",
			requestedPath:  "..%2f..%2f..%2fetc%2fpasswd",
			expectValid:    false,
			expectedFlags:  []string{"traversal_attempt"},
			expectError:    true,
		},
		{
			name:           "path_traversal_double_encoded",
			requestedPath:  "..%252f..%252f..%252fetc%252fpasswd",
			expectValid:    false,
			expectedFlags:  []string{"traversal_attempt"},
			expectError:    true,
		},
		{
			name:           "path_traversal_mixed",
			requestedPath:  "./.././.././../etc/passwd",
			expectValid:    false,
			expectedFlags:  []string{"traversal_attempt"},
			expectError:    true,
		},
		{
			name:           "path_traversal_windows",
			requestedPath:  "..\\..\\..\\windows\\system32\\config\\sam",
			expectValid:    false,
			expectedFlags:  []string{"traversal_attempt"},
			expectError:    true,
		},
		{
			name:           "excessive_length_path",
			requestedPath:  string(make([]byte, 200)), // Exceeds max length
			expectValid:    false,
			expectedFlags:  []string{"excessive_length"},
			expectError:    true,
		},
		{
			name:           "outside_approved_directory",
			requestedPath:  "/etc/hosts",
			expectValid:    false,
			expectedFlags:  []string{"outside_approved_directories"},
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidatePath(tt.requestedPath)

			// Check error expectation
			if tt.expectError {
				testUtils.AssertSecurityError(err, "path_traversal")
			} else {
				testUtils.AssertNoSecurityError(err)
			}

			// Check validation result
			if result != nil {
				testUtils.AssertValidationResult(result, tt.expectValid, tt.expectedFlags)
			}
		})
	}
}

func TestPathValidator_ValidatePath_ApprovedDirectories(t *testing.T) {
	testUtils := NewSecurityTestUtils(t)
	config := testUtils.CreateTestConfig()
	logger := testUtils.CreateTestLogger()
	validator := security.NewPathValidator(config, logger)

	tests := []struct {
		name          string
		requestedPath string
		expectValid   bool
	}{
		{
			name:          "approved_contracts_directory",
			requestedPath: "testdata/contracts/api-spec.json",
			expectValid:   true,
		},
		{
			name:          "approved_schemas_directory",
			requestedPath: "testdata/schemas/user.json",
			expectValid:   true,
		},
		{
			name:          "unapproved_directory",
			requestedPath: "src/config/secrets.json",
			expectValid:   false,
		},
		{
			name:          "root_directory_attempt",
			requestedPath: "/etc/passwd",
			expectValid:   false,
		},
		{
			name:          "home_directory_attempt",
			requestedPath: "~/secrets.txt",
			expectValid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidatePath(tt.requestedPath)

			if tt.expectValid {
				testUtils.AssertNoSecurityError(err)
				if result != nil {
					testUtils.AssertValidationResult(result, true, []string{})
				}
			} else {
				testUtils.AssertSecurityError(err, "path_traversal")
				if result != nil {
					testUtils.AssertValidationResult(result, false, []string{"outside_approved_directories"})
				}
			}
		})
	}
}

func TestPathValidator_PathTraversalAttackVectors(t *testing.T) {
	testUtils := NewSecurityTestUtils(t)
	config := testUtils.CreateTestConfig()
	logger := testUtils.CreateTestLogger()
	validator := security.NewPathValidator(config, logger)

	// Test all attack vectors from test utils
	attackVectors := testUtils.PathTraversalAttacks()

	for i, attackPath := range attackVectors {
		t.Run(fmt.Sprintf("attack_vector_%d", i), func(t *testing.T) {
			result, err := validator.ValidatePath(attackPath)

			// All attack vectors should be blocked
			testUtils.AssertSecurityError(err, "path_traversal")

			if result != nil {
				// Should have security flags indicating the attack was detected
				if len(result.SecurityFlags) == 0 {
					t.Errorf("Expected security flags for attack vector %s", attackPath)
				}
				testUtils.AssertValidationResult(result, false, []string{"traversal_attempt"})
			}
		})
	}
}

func TestPathValidator_SymlinkHandling(t *testing.T) {
	testUtils := NewSecurityTestUtils(t)
	config := testUtils.CreateTestConfig()
	config.PathValidation.AllowSymlinks = false // Ensure symlinks are disabled
	logger := testUtils.CreateTestLogger()
	validator := security.NewPathValidator(config, logger)

	// We can't easily create actual symlinks in a test, but we can test the detection logic
	// This would be enhanced in integration tests with real filesystem setup

	tests := []struct {
		name          string
		path          string
		allowSymlinks bool
		expectValid   bool
	}{
		{
			name:          "symlinks_disabled_safe_path",
			path:          "testdata/contracts/safe.json",
			allowSymlinks: false,
			expectValid:   true, // Should pass if no actual symlinks present
		},
		{
			name:          "symlinks_enabled_safe_path",
			path:          "testdata/contracts/safe.json",
			allowSymlinks: true,
			expectValid:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.PathValidation.AllowSymlinks = tt.allowSymlinks
			validator := security.NewPathValidator(config, logger)

			result, err := validator.ValidatePath(tt.path)

			if tt.expectValid {
				testUtils.AssertNoSecurityError(err)
				if result != nil {
					testUtils.AssertValidationResult(result, true, []string{})
				}
			} else {
				testUtils.AssertSecurityError(err, "path_traversal")
			}
		})
	}
}

func TestPathValidator_PathSanitization(t *testing.T) {
	testUtils := NewSecurityTestUtils(t)
	config := testUtils.CreateTestConfig()
	logger := testUtils.CreateTestLogger()
	validator := security.NewPathValidator(config, logger)

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
			name:         "long_path_truncation",
			inputPath:    string(make([]byte, 100)),
			expectedSafe: "<path:100 chars>",
		},
		{
			name:         "path_with_traversal",
			inputPath:    "../../../etc/passwd",
			expectedSafe: "[..]/[..]/[..]/etc/passwd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitized := validator.SanitizePathForDisplay(tt.inputPath)

			if sanitized != tt.expectedSafe {
				t.Errorf("Expected sanitized path %q, got %q", tt.expectedSafe, sanitized)
			}
		})
	}
}

func TestPathValidator_ConfigurationValidation(t *testing.T) {
	testUtils := NewSecurityTestUtils(t)
	logger := testUtils.CreateTestLogger()

	tests := []struct {
		name        string
		config      security.SecurityConfig
		expectError bool
	}{
		{
			name:        "valid_config",
			config:      testUtils.CreateTestConfig(),
			expectError: false,
		},
		{
			name: "invalid_max_path_length",
			config: func() security.SecurityConfig {
				cfg := testUtils.CreateTestConfig()
				cfg.PathValidation.MaxPathLength = -1
				return cfg
			}(),
			expectError: true,
		},
		{
			name: "empty_approved_directories",
			config: func() security.SecurityConfig {
				cfg := testUtils.CreateTestConfig()
				cfg.PathValidation.ApprovedDirectories = []string{}
				return cfg
			}(),
			expectError: false, // Should allow empty (defaults to relative paths only)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError && err == nil {
				t.Error("Expected validation error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestPathValidator_PerformanceCharacteristics(t *testing.T) {
	testUtils := NewSecurityTestUtils(t)
	config := testUtils.CreateTestConfig()
	logger := testUtils.CreateTestLogger()
	validator := security.NewPathValidator(config, logger)

	// Test performance with valid paths (should be fast)
	validPath := "testdata/contracts/valid.json"

	// Warm up
	validator.ValidatePath(validPath)

	// Measure performance
	start := time.Now()
	for i := 0; i < 100; i++ {
		validator.ValidatePath(validPath)
	}
	elapsed := time.Since(start)

	// Should complete 100 validations in under 100ms (1ms per validation)
	if elapsed > 100*time.Millisecond {
		t.Errorf("Path validation too slow: %v for 100 validations", elapsed)
	}
}