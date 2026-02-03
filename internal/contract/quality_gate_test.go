package contract

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinkValidationGate(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		expectedIssues int
		shouldPass     bool
	}{
		{
			name: "valid internal links",
			content: `# Test Document
[Link to file](./README.md)
[Link to section](#section)
`,
			expectedIssues: 0,
			shouldPass:     true,
		},
		{
			name: "valid external links",
			content: `# Test Document
[External](https://example.com)
[HTTP](http://example.com)
`,
			expectedIssues: 0,
			shouldPass:     true,
		},
		{
			name: "broken internal link",
			content: `# Test Document
[Broken](./nonexistent.md)
`,
			expectedIssues: 1,
			shouldPass:     false,
		},
		{
			name: "undefined reference link",
			content: `# Test Document
[Reference Link][ref]

This has no definition for [ref].
`,
			expectedIssues: 1,
			shouldPass:     false,
		},
		{
			name: "defined reference link",
			content: `# Test Document
[Reference Link][ref]

[ref]: https://example.com
`,
			expectedIssues: 0,
			shouldPass:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory and file
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.md")
			require.NoError(t, os.WriteFile(testFile, []byte(tt.content), 0644))

			// Create README.md for valid link test
			readmeFile := filepath.Join(tmpDir, "README.md")
			require.NoError(t, os.WriteFile(readmeFile, []byte("# README"), 0644))

			gate := &LinkValidationGate{}
			config := QualityGateConfig{
				Type:   "link_validation",
				Target: "test.md",
			}

			violations, err := gate.Check(tmpDir, config)
			require.NoError(t, err)

			if tt.shouldPass {
				assert.Len(t, violations, 0, "Expected no violations for valid links")
			} else {
				assert.Greater(t, len(violations), 0, "Expected violations for broken links")
				if len(violations) > 0 {
					assert.Contains(t, violations[0].Message, "broken")
				}
			}
		})
	}
}

func TestMarkdownStructureGate(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		requiredSections []string
		expectedIssues int
	}{
		{
			name: "proper heading hierarchy",
			content: `# Title
## Section 1
### Subsection 1.1
## Section 2
`,
			expectedIssues: 0,
		},
		{
			name: "skipped heading level",
			content: `# Title
### Subsection (skipped h2)
`,
			expectedIssues: 1,
		},
		{
			name: "required sections present",
			content: `# Title
## Overview
## Implementation
## Testing
`,
			requiredSections: []string{"Overview", "Implementation", "Testing"},
			expectedIssues:   0,
		},
		{
			name: "missing required sections",
			content: `# Title
## Overview
`,
			requiredSections: []string{"Overview", "Implementation", "Testing"},
			expectedIssues:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.md")
			require.NoError(t, os.WriteFile(testFile, []byte(tt.content), 0644))

			gate := &MarkdownStructureGate{}
			config := QualityGateConfig{
				Type:   "markdown_structure",
				Target: "test.md",
			}

			if len(tt.requiredSections) > 0 {
				config.Parameters = map[string]interface{}{
					"required_sections": interfaceSlice(tt.requiredSections),
				}
			}

			violations, err := gate.Check(tmpDir, config)
			require.NoError(t, err)

			if tt.expectedIssues == 0 {
				assert.Len(t, violations, 0, "Expected no violations")
			} else {
				assert.Greater(t, len(violations), 0, "Expected violations")
			}
		})
	}
}

func TestJSONStructureGate(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
	}{
		{
			name: "valid JSON",
			content: `{
  "name": "test",
  "value": 123
}`,
			expectError: false,
		},
		{
			name:        "invalid JSON",
			content:     `{name: "test", value: 123}`,
			expectError: true,
		},
		{
			name:        "malformed JSON",
			content:     `{"name": "test", "value": 123,}`,
			expectError: true,
		},
		{
			name: "unformatted JSON",
			content: `{"name":"test","value":123}`,
			expectError: false, // Valid but info level violation for formatting
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.json")
			require.NoError(t, os.WriteFile(testFile, []byte(tt.content), 0644))

			gate := &JSONStructureGate{}
			config := QualityGateConfig{
				Type:   "json_structure",
				Target: "test.json",
			}

			violations, err := gate.Check(tmpDir, config)
			require.NoError(t, err)

			if tt.expectError {
				assert.Greater(t, len(violations), 0, "Expected violations for invalid JSON")
				if len(violations) > 0 {
					assert.Equal(t, "error", violations[0].Severity)
				}
			}
		})
	}
}

func TestRequiredFieldsGate(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		requiredFields []string
		expectMissing  bool
	}{
		{
			name: "all fields present",
			content: `{
  "name": "test",
  "version": "1.0.0",
  "description": "A test"
}`,
			requiredFields: []string{"name", "version", "description"},
			expectMissing:  false,
		},
		{
			name: "missing fields",
			content: `{
  "name": "test"
}`,
			requiredFields: []string{"name", "version", "description"},
			expectMissing:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.json")
			require.NoError(t, os.WriteFile(testFile, []byte(tt.content), 0644))

			gate := &RequiredFieldsGate{}
			config := QualityGateConfig{
				Type:     "required_fields",
				Target:   "test.json",
				Required: true,
				Parameters: map[string]interface{}{
					"fields": interfaceSlice(tt.requiredFields),
				},
			}

			violations, err := gate.Check(tmpDir, config)
			require.NoError(t, err)

			if tt.expectMissing {
				assert.Greater(t, len(violations), 0, "Expected violations for missing fields")
				if len(violations) > 0 {
					assert.Contains(t, violations[0].Message, "missing")
				}
			} else {
				assert.Len(t, violations, 0, "Expected no violations")
			}
		})
	}
}

func TestContentCompletenessGate(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		minWords    int
		expectIssue bool
	}{
		{
			name:        "sufficient content",
			content:     "This is a test document with enough words to meet the minimum requirement for completeness validation. We need to ensure quality.",
			minWords:    10,
			expectIssue: false,
		},
		{
			name:        "insufficient content",
			content:     "Too short",
			minWords:    100,
			expectIssue: true,
		},
		{
			name:        "content with placeholders",
			content:     "This is a document with TODO items and FIXME notes that need addressing.",
			minWords:    10,
			expectIssue: true, // Should flag placeholders
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.md")
			require.NoError(t, os.WriteFile(testFile, []byte(tt.content), 0644))

			gate := &ContentCompletenessGate{}
			config := QualityGateConfig{
				Type:      "content_completeness",
				Target:    "test.md",
				Threshold: 70,
				Parameters: map[string]interface{}{
					"min_words": float64(tt.minWords),
				},
			}

			violations, err := gate.Check(tmpDir, config)
			require.NoError(t, err)

			if tt.expectIssue {
				assert.Greater(t, len(violations), 0, "Expected violations")
			}
		})
	}
}

func TestQualityGateRunner(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	goodJSON := filepath.Join(tmpDir, "good.json")
	require.NoError(t, os.WriteFile(goodJSON, []byte(`{
  "name": "test",
  "version": "1.0.0",
  "description": "Valid JSON with all required fields"
}`), 0644))

	badJSON := filepath.Join(tmpDir, "bad.json")
	require.NoError(t, os.WriteFile(badJSON, []byte(`{name: bad}`), 0644))

	goodMD := filepath.Join(tmpDir, "good.md")
	require.NoError(t, os.WriteFile(goodMD, []byte(`# Title
## Section 1
This is a well-formed markdown document with proper structure.
## Section 2
More content here.
`), 0644))

	runner := NewQualityGateRunner()

	configs := []QualityGateConfig{
		{
			Type:     "json_structure",
			Target:   "good.json",
			Required: true,
		},
		{
			Type:   "markdown_structure",
			Target: "good.md",
		},
	}

	result, err := runner.RunGates(tmpDir, configs)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Passed, "All gates should pass for valid files")

	// Test with bad JSON
	badConfigs := []QualityGateConfig{
		{
			Type:     "json_structure",
			Target:   "bad.json",
			Required: true,
		},
	}

	badResult, err := runner.RunGates(tmpDir, badConfigs)
	require.NoError(t, err)
	assert.NotNil(t, badResult)
	assert.False(t, badResult.Passed, "Should fail for invalid JSON")
	assert.Greater(t, len(badResult.Violations), 0, "Should have violations")
}

func TestQualityGateResultFormatting(t *testing.T) {
	result := &QualityGateResult{
		Passed: false,
		Score:  65,
		Violations: []QualityViolation{
			{
				Gate:     "json_structure",
				Severity: "error",
				Message:  "Invalid JSON format",
				Details:  []string{"unexpected token at position 5"},
				Suggestions: []string{
					"Check for missing quotes",
					"Verify bracket matching",
				},
			},
			{
				Gate:     "content_completeness",
				Severity: "warning",
				Message:  "Content below threshold",
				Score:    60,
				Threshold: 70,
			},
		},
	}

	guidance := result.FormatGuidance()
	assert.Contains(t, guidance, "Quality gates failed")
	assert.Contains(t, guidance, "score: 65/100")
	assert.Contains(t, guidance, "1 errors, 1 warnings")
	assert.Contains(t, guidance, "Invalid JSON format")
	assert.Contains(t, guidance, "Check for missing quotes")
}

// Helper function to convert string slice to interface slice
func interfaceSlice(slice []string) []interface{} {
	result := make([]interface{}, len(slice))
	for i, v := range slice {
		result[i] = v
	}
	return result
}
