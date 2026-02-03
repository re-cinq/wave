package contract

import (
	"strings"
	"testing"
)

// Test JSONCleaner basic functionality
func TestJSONCleaner_CleanJSONOutput(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldPass  bool
		shouldClean bool
	}{
		{
			name:        "already valid JSON",
			input:       `{"key":"value","number":123}`,
			shouldPass:  true,
			shouldClean: false,
		},
		{
			name:        "trailing comma",
			input:       `{"key":"value",}`,
			shouldPass:  true,
			shouldClean: true,
		},
		{
			name:        "multiple trailing commas",
			input:       `{"key":"value","arr":[1,2,3,]}`,
			shouldPass:  true,
			shouldClean: true,
		},
		{
			name:        "single line comments",
			input:       `{"key":"value"} // comment`,
			shouldPass:  true,
			shouldClean: true,
		},
		{
			name:        "multiline comments",
			input:       `{"key":"value" /* comment */}`,
			shouldPass:  true,
			shouldClean: true,
		},
		{
			name:        "mixed valid JSON with comments and trailing commas",
			input:       `{"key":"value", // inline comment` + "\n" + `"arr":[1,2,3,], /* block */ }`,
			shouldPass:  true,
			shouldClean: true,
		},
		{
			name:        "newlines preserved in strings",
			input:       `{"description":"Line 1\nLine 2\nLine 3"}`,
			shouldPass:  true,
			shouldClean: false,
		},
		{
			name:        "invalid JSON cannot be fixed",
			input:       `{invalid json}`,
			shouldPass:  false,
			shouldClean: true,
		},
	}

	cleaner := &JSONCleaner{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleaned, changes, err := cleaner.CleanJSONOutput(tt.input)

			if tt.shouldPass {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				if !cleaner.IsValidJSON(cleaned) {
					t.Fatal("cleaned output is not valid JSON")
				}
			} else {
				if err == nil {
					t.Fatal("expected error for invalid JSON")
				}
			}

			if tt.shouldClean && len(changes) == 0 && tt.shouldPass {
				t.Logf("expected changes but got none. Input: %q", tt.input)
			}
		})
	}
}

// Test that multiline strings are preserved
func TestJSONCleaner_PreservesMultilineStrings(t *testing.T) {
	cleaner := &JSONCleaner{}

	input := `{
  "title": "Test Issue",
  "description": "Line 1\nLine 2\nLine 3"
}`

	cleaned, _, err := cleaner.CleanJSONOutput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that newlines are still escaped in output
	if !strings.Contains(cleaned, "\\n") {
		t.Error("multiline string escaping was not preserved")
	}

	// Verify it still parses correctly
	if !cleaner.IsValidJSON(cleaned) {
		t.Error("cleaned JSON is not valid")
	}
}

// Test ValidateAndFormatJSON
func TestJSONCleaner_ValidateAndFormatJSON(t *testing.T) {
	cleaner := &JSONCleaner{}

	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{
			name:      "valid JSON",
			input:     `{"key":"value"}`,
			shouldErr: false,
		},
		{
			name:      "invalid JSON",
			input:     `{invalid}`,
			shouldErr: true,
		},
		{
			name:      "minified JSON",
			input:     `{"a":1,"b":2,"c":{"d":3}}`,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted, err := cleaner.ValidateAndFormatJSON(tt.input)

			if tt.shouldErr {
				if err == nil {
					t.Fatal("expected error")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !cleaner.IsValidJSON(formatted) {
					t.Error("formatted output is not valid JSON")
				}
				// Check that output has indentation
				if !strings.Contains(formatted, "\n") {
					t.Error("formatted output should be indented")
				}
			}
		})
	}
}

// Test IsValidJSON
func TestJSONCleaner_IsValidJSON(t *testing.T) {
	cleaner := &JSONCleaner{}

	tests := []struct {
		name      string
		input     string
		isValid   bool
	}{
		{
			name:    "valid object",
			input:   `{"key":"value"}`,
			isValid: true,
		},
		{
			name:    "valid array",
			input:   `[1,2,3]`,
			isValid: true,
		},
		{
			name:    "valid primitive",
			input:   `"string"`,
			isValid: true,
		},
		{
			name:    "invalid object",
			input:   `{invalid}`,
			isValid: false,
		},
		{
			name:    "unclosed brace",
			input:   `{"key":"value"`,
			isValid: false,
		},
		{
			name:    "trailing comma",
			input:   `{"key":"value",}`,
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := cleaner.IsValidJSON(tt.input)
			if valid != tt.isValid {
				t.Errorf("expected %v, got %v for: %q", tt.isValid, valid, tt.input)
			}
		})
	}
}

// Test ExtractJSONFromText
func TestJSONCleaner_ExtractJSONFromText(t *testing.T) {
	cleaner := &JSONCleaner{}

	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{
			name:      "JSON with preceding text",
			input:     `Here is some analysis: {"key":"value"}`,
			shouldErr: false,
		},
		{
			name:      "JSON with trailing text",
			input:     `{"key":"value"} and some explanation`,
			shouldErr: false,
		},
		{
			name:      "nested JSON with text",
			input:     `Analysis: {"outer":{"inner":"value"}}. Done.`,
			shouldErr: false,
		},
		{
			name:      "JSON array with text",
			input:     `Results: [1,2,3] end`,
			shouldErr: false,
		},
		{
			name:      "no JSON in text",
			input:     `Just some plain text without any JSON`,
			shouldErr: true,
		},
		{
			name:      "malformed JSON in text",
			input:     `Here: {incomplete json} and more text`,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extracted, err := cleaner.ExtractJSONFromText(tt.input)

			if tt.shouldErr {
				if err == nil {
					t.Error("expected error")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !cleaner.IsValidJSON(extracted) {
					t.Error("extracted text is not valid JSON")
				}
			}
		})
	}
}

// Test CleanJSONOutput with real-world scenarios
func TestJSONCleaner_RealWorldScenarios(t *testing.T) {
	cleaner := &JSONCleaner{}

	// Scenario 1: GitHub issue analysis output with comments
	githubAnalysis := `{
  "repository": {
    "owner": "re-cinq",
    "name": "wave"
  },
  // Total issues analyzed
  "total_issues": 10,
  "poor_quality_issues": [
    {
      "number": 20,
      "title": "Poorly written issue",
      /* inline comment */ "quality_score": 45,
    }, // trailing comma issue
  ],
  // end of data
}`

	cleaned, changes, err := cleaner.CleanJSONOutput(githubAnalysis)
	if err != nil {
		t.Fatalf("failed to clean GitHub analysis: %v", err)
	}

	if len(changes) == 0 {
		t.Error("expected cleaning changes")
	}

	if !cleaner.IsValidJSON(cleaned) {
		t.Error("cleaned GitHub analysis is not valid JSON")
	}

	// Scenario 2: Multiline descriptions preserved
	if !strings.Contains(cleaned, "Poorly written issue") {
		t.Error("content was lost during cleaning")
	}
}
