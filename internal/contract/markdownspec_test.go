package contract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMarkdownSpecValidator_Validate(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "markdownspec_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test specification schema
	schemaContent := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["title", "user_stories"],
		"properties": {
			"title": {
				"type": "string",
				"minLength": 5
			},
			"user_stories": {
				"type": "array",
				"items": {
					"type": "object",
					"required": ["as_a", "i_want", "so_that", "acceptance_criteria"],
					"properties": {
						"as_a": {"type": "string"},
						"i_want": {"type": "string"},
						"so_that": {"type": "string"},
						"acceptance_criteria": {
							"type": "array",
							"items": {"type": "string"}
						}
					}
				},
				"minItems": 1
			}
		}
	}`

	schemaPath := filepath.Join(tempDir, "test-schema.json")
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a valid test markdown specification
	validMarkdown := `# User Authentication Feature

## Overview
This feature adds user authentication to the application.

## User Stories

### Story 1
As a: user
I want: to be able to log in with my email and password
So that: I can access my personal dashboard

Acceptance criteria:
- User can enter email and password
- System validates credentials against database
- Successful login redirects to dashboard
- Failed login shows error message

### Story 2
As a: admin
I want: to manage user accounts
So that: I can control access to the system

Acceptance criteria:
- Admin can create new user accounts
- Admin can disable user accounts
- Admin can reset user passwords

## Data Model

User object with fields:
- id: unique identifier
- email: user's email address
- password_hash: encrypted password
- role: user role (admin/user)
- created_at: account creation timestamp

## API Design

Authentication endpoints:
- POST /api/auth/login
- POST /api/auth/logout
- GET /api/auth/me

## Edge Cases

- Password reset flow
- Account lockout after failed attempts
- Session timeout handling

## Testing Strategy

Unit tests for authentication logic
Integration tests for API endpoints
E2E tests for login flow
`

	validSpecPath := filepath.Join(tempDir, "valid-spec.md")
	if err := os.WriteFile(validSpecPath, []byte(validMarkdown), 0644); err != nil {
		t.Fatal(err)
	}

	// Create an invalid markdown (missing required user stories)
	invalidMarkdown := `# Invalid Feature

## Overview
This is an incomplete specification.

## Data Model
Some data model here.
`

	invalidSpecPath := filepath.Join(tempDir, "invalid-spec.md")
	if err := os.WriteFile(invalidSpecPath, []byte(invalidMarkdown), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		config      ContractConfig
		expectError bool
		description string
	}{
		{
			name: "valid_markdown_spec_with_schema",
			config: ContractConfig{
				Type:       "markdown_spec",
				Source:     "valid-spec.md",
				SchemaPath: schemaPath,
			},
			expectError: false,
			description: "Valid markdown specification should pass validation",
		},
		{
			name: "invalid_markdown_spec_with_schema",
			config: ContractConfig{
				Type:       "markdown_spec",
				Source:     "invalid-spec.md",
				SchemaPath: schemaPath,
			},
			expectError: true,
			description: "Invalid markdown specification should fail validation",
		},
		{
			name: "markdown_spec_without_schema",
			config: ContractConfig{
				Type:   "markdown_spec",
				Source: "valid-spec.md",
			},
			expectError: false,
			description: "Markdown spec without schema should parse successfully",
		},
		{
			name: "missing_source_file",
			config: ContractConfig{
				Type:       "markdown_spec",
				Source:     "nonexistent.md",
				SchemaPath: schemaPath,
			},
			expectError: true,
			description: "Missing source file should return error",
		},
		{
			name: "missing_source_config",
			config: ContractConfig{
				Type: "markdown_spec",
			},
			expectError: true,
			description: "Missing source config should return error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := &markdownSpecValidator{}
			err := validator.Validate(tt.config, tempDir)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none: %s", tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v (%s)", err, tt.description)
			}

			// For valid cases, verify JSON output was created
			if !tt.expectError && err == nil && tt.config.Source != "" {
				jsonPath := filepath.Join(tempDir, tt.config.Source[:len(tt.config.Source)-3]+".json")
				if _, statErr := os.Stat(jsonPath); os.IsNotExist(statErr) {
					t.Errorf("Expected JSON output file to be created at %s", jsonPath)
				} else {
					// Verify JSON is valid
					jsonData, readErr := os.ReadFile(jsonPath)
					if readErr != nil {
						t.Errorf("Failed to read JSON output: %v", readErr)
					}
					var parsed interface{}
					if parseErr := json.Unmarshal(jsonData, &parsed); parseErr != nil {
						t.Errorf("Generated JSON is not valid: %v", parseErr)
					}
				}
			}
		})
	}
}

func TestMarkdownSpecValidator_SpeckitPathResolution(t *testing.T) {
	// Create temporary directory structure mimicking Speckit
	tempDir, err := os.MkdirTemp("", "speckit_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create specs directory with feature subdirectory
	specsDir := filepath.Join(tempDir, "specs")
	featureDir := filepath.Join(specsDir, "018-enhanced-progress")
	if err := os.MkdirAll(featureDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create spec file
	specContent := `# Enhanced Progress Feature

## User Stories

As a: developer
I want: enhanced progress tracking
So that: I can monitor pipeline execution better

Acceptance criteria:
- Progress events are emitted
- Events include step status
- Events are structured
`

	specPath := filepath.Join(featureDir, "spec.md")
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		workspacePath string
		relativePath  string
		expectedPath  string
		description   string
	}{
		{
			name:          "template_path_resolution",
			workspacePath: tempDir,
			relativePath:  "specs/{{pipeline_context.branch_name}}/spec.md",
			expectedPath:  specPath,
			description:   "Should resolve template path to actual Speckit directory",
		},
		{
			name:          "direct_path",
			workspacePath: tempDir,
			relativePath:  "specs/018-enhanced-progress/spec.md",
			expectedPath:  filepath.Join(tempDir, "specs/018-enhanced-progress/spec.md"),
			description:   "Should handle direct paths correctly",
		},
		{
			name:          "absolute_path",
			workspacePath: "/some/workspace",
			relativePath:  specPath,
			expectedPath:  specPath,
			description:   "Should handle absolute paths as-is",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved := resolveSpeckitPath(tt.workspacePath, tt.relativePath)

			if resolved != tt.expectedPath {
				t.Errorf("Expected path %s but got %s (%s)", tt.expectedPath, resolved, tt.description)
			}

			// For template paths, verify the file actually exists
			if tt.relativePath == "specs/{{pipeline_context.branch_name}}/spec.md" {
				if _, err := os.Stat(resolved); os.IsNotExist(err) {
					t.Errorf("Resolved path %s does not exist", resolved)
				}
			}
		})
	}
}

func TestParseUserStories(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected int
		description string
	}{
		{
			name: "standard_format",
			content: `As a: user
I want: to log in
So that: I can access my account

Acceptance criteria:
- Enter username and password
- System validates credentials
- Redirect on success

As a: admin
I want: to manage users
So that: I can control access

Acceptance criteria:
- Create user accounts
- Delete user accounts`,
			expected: 2,
			description: "Should parse multiple user stories",
		},
		{
			name: "alternative_format",
			content: `As a user, I want to log in so that I can access my account.

Acceptance criteria:
* Enter username and password
* System validates credentials
* Redirect on success`,
			expected: 1,
			description: "Should parse comma-separated format",
		},
		{
			name:     "empty_content",
			content:  "",
			expected: 0,
			description: "Should handle empty content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stories, err := parseUserStories(tt.content)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if len(stories) != tt.expected {
				t.Errorf("Expected %d stories but got %d (%s)", tt.expected, len(stories), tt.description)
			}

			// Verify stories have required fields when expected
			for i, story := range stories {
				if tt.expected > 0 && len(story.AcceptanceCriteria) == 0 {
					t.Errorf("Story %d should have acceptance criteria", i)
				}
			}
		})
	}
}