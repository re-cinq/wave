package adapter

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	assert.Equal(t, 3, config.MaxAttempts)
	assert.Equal(t, 1*time.Second, config.BaseDelay)
	assert.Equal(t, 30*time.Second, config.MaxDelay)
	assert.Equal(t, 2.0, config.BackoffMultiplier)
	assert.True(t, config.EnableJSONRecovery)
	assert.True(t, config.ProgressiveEnhancement)
	assert.True(t, config.AllowPartialResults)
}

func TestOutputFormatCorrector_JSONCorrection(t *testing.T) {
	corrector := NewOutputFormatCorrector(nil)

	tests := []struct {
		name           string
		input          string
		expectedSuccess bool
		expectedStrategy string
	}{
		{
			name:             "valid JSON passes through",
			input:            `{"name": "test", "value": 42}`,
			expectedSuccess:  true,
			expectedStrategy: "direct_json_validation",
		},
		{
			name:             "JSON in markdown block",
			input:            "Here's the result:\n\n```json\n{\"name\": \"extracted\", \"status\": \"success\"}\n```\n\nThat should work!",
			expectedSuccess:  true,
			expectedStrategy: "markdown_extraction",
		},
		{
			name:             "JSON with trailing comma",
			input:            `{"name": "test", "value": 42,}`,
			expectedSuccess:  true,
			expectedStrategy: "heuristic_json_recovery",
		},
		{
			name:             "JSON with explanatory text",
			input:            `The result is: {"name": "test", "value": 42} - this should work`,
			expectedSuccess:  true,
			expectedStrategy: "regex_json_extraction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := corrector.CorrectOutput(tt.input, "json", 1)

			if tt.expectedSuccess {
				require.NoError(t, err)
				assert.True(t, result.Success)
				assert.Equal(t, tt.expectedStrategy, result.AppliedStrategy)

				// Validate that the corrected content is valid JSON
				var js json.RawMessage
				assert.NoError(t, json.Unmarshal([]byte(result.CorrectedContent), &js))
			} else {
				assert.Error(t, err)
				assert.False(t, result.Success)
			}
		})
	}
}

func TestDirectJSONValidationStrategy(t *testing.T) {
	strategy := &DirectJSONValidationStrategy{}

	tests := []struct {
		name        string
		input       string
		format      string
		shouldPass  bool
	}{
		{
			name:       "valid JSON object",
			input:      `{"test": true}`,
			format:     "json",
			shouldPass: true,
		},
		{
			name:       "valid JSON array",
			input:      `[1, 2, 3]`,
			format:     "json",
			shouldPass: true,
		},
		{
			name:       "invalid JSON",
			input:      `{test: true}`, // missing quotes
			format:     "json",
			shouldPass: false,
		},
		{
			name:       "wrong format",
			input:      `{"test": true}`,
			format:     "yaml",
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, metadata, err := strategy.Apply(tt.input, tt.format)

			if tt.shouldPass {
				assert.NoError(t, err)
				assert.NotEmpty(t, result)
				assert.NotNil(t, metadata)
				assert.Equal(t, "passed", metadata["validation"])
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestMarkdownCodeBlockExtractionStrategy(t *testing.T) {
	strategy := &MarkdownCodeBlockExtractionStrategy{}

	tests := []struct {
		name     string
		input    string
		expected string
		shouldWork bool
	}{
		{
			name:        "single JSON block",
			input:       "Here's the data:\n\n```json\n{\"name\": \"test\", \"value\": 42}\n```\n\nHope that helps!",
			expected:    `{"name": "test", "value": 42}`,
			shouldWork:  true,
		},
		{
			name: "no JSON block",
			input: `Just some text without any JSON blocks.`,
			shouldWork: false,
		},
		{
			name:        "invalid JSON in block",
			input:       "Here's broken JSON:\n\n```json\n{name: test}\n```",
			shouldWork:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, metadata, err := strategy.Apply(tt.input, "json")

			if tt.shouldWork {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
				assert.Equal(t, "markdown_code_block", metadata["extracted_from"])
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestRegexJSONExtractionStrategy(t *testing.T) {
	strategy := &RegexJSONExtractionStrategy{}

	tests := []struct {
		name       string
		input      string
		shouldWork bool
	}{
		{
			name:       "JSON object in text",
			input:      `The result is {"name": "test", "value": 42} for you.`,
			shouldWork: true,
		},
		{
			name:       "JSON array in text",
			input:      `Here are the values: [1, 2, 3, 4] that we found.`,
			shouldWork: true,
		},
		{
			name:       "nested JSON",
			input:      `Result: {"user": {"name": "John", "age": 30}}`,
			shouldWork: true,
		},
		{
			name:       "no valid JSON patterns",
			input:      `Just some plain text without any JSON.`,
			shouldWork: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, metadata, err := strategy.Apply(tt.input, "json")

			if tt.shouldWork {
				assert.NoError(t, err)
				assert.NotEmpty(t, result)

				// Verify the extracted content is valid JSON
				var js json.RawMessage
				assert.NoError(t, json.Unmarshal([]byte(result), &js))

				assert.Contains(t, metadata["extracted_from"].(string), "regex")
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestHeuristicJSONRecoveryStrategy(t *testing.T) {
	strategy := &HeuristicJSONRecoveryStrategy{}

	tests := []struct {
		name       string
		input      string
		shouldWork bool
	}{
		{
			name:       "fix trailing comma",
			input:      `{"name": "test", "value": 42,}`,
			shouldWork: true,
		},
		{
			name:       "fix single quotes",
			input:      `{'name': 'test', 'value': 42}`,
			shouldWork: true,
		},
		{
			name:       "remove prefix text",
			input:      `Here's the JSON: {"name": "test"}`,
			shouldWork: true,
		},
		{
			name:       "fix unquoted keys",
			input:      `{name: "test", value: 42}`,
			shouldWork: true,
		},
		{
			name:       "completely malformed",
			input:      `this is not JSON at all`,
			shouldWork: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, metadata, err := strategy.Apply(tt.input, "json")

			if tt.shouldWork {
				assert.NoError(t, err)
				assert.NotEmpty(t, result)

				// Verify the repaired content is valid JSON
				var js json.RawMessage
				assert.NoError(t, json.Unmarshal([]byte(result), &js))

				fixes, ok := metadata["fixes_applied"].([]string)
				assert.True(t, ok)
				assert.NotEmpty(t, fixes)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestPartialJSONRecoveryStrategy(t *testing.T) {
	strategy := &PartialJSONRecoveryStrategy{}

	tests := []struct {
		name       string
		input      string
		shouldWork bool
	}{
		{
			name: "partial properties",
			input: `Some text here
			"name": "test"
			"value": 42
			"status": "active"
			More text`,
			shouldWork: true,
		},
		{
			name: "mixed valid and invalid properties",
			input: `"name": "test"
			invalid line here
			"value": 42`,
			shouldWork: true,
		},
		{
			name: "no valid properties",
			input: `No JSON properties here at all`,
			shouldWork: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, metadata, err := strategy.Apply(tt.input, "json")

			if tt.shouldWork {
				assert.NoError(t, err)
				assert.NotEmpty(t, result)

				// Verify the result is valid JSON
				var js json.RawMessage
				assert.NoError(t, json.Unmarshal([]byte(result), &js))

				recoveredCount, ok := metadata["recovered_properties"].(int)
				assert.True(t, ok)
				assert.Greater(t, recoveredCount, 0)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestTemplateBasedRecoveryStrategy(t *testing.T) {
	strategy := &TemplateBasedRecoveryStrategy{}

	tests := []struct {
		name   string
		format string
	}{
		{"JSON template", "json"},
		{"YAML template", "yaml"},
		{"Markdown template", "markdown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, metadata, err := strategy.Apply("broken input", tt.format)

			assert.NoError(t, err)
			assert.NotEmpty(t, result)
			assert.Equal(t, "default_"+tt.format, metadata["template_used"])

			// For JSON, verify it's valid
			if tt.format == "json" {
				var js json.RawMessage
				assert.NoError(t, json.Unmarshal([]byte(result), &js))
			}
		})
	}
}

func TestStructuredErrorReportStrategy(t *testing.T) {
	strategy := &StructuredErrorReportStrategy{}

	result, metadata, err := strategy.Apply("malformed content", "json")

	assert.NoError(t, err)
	assert.NotEmpty(t, result)

	// Verify the result is valid JSON
	var js map[string]interface{}
	assert.NoError(t, json.Unmarshal([]byte(result), &js))

	// Check required fields
	assert.Equal(t, "output_format_recovery_failed", js["error_type"])
	assert.Equal(t, "json", js["target_format"])
	assert.Contains(t, js, "timestamp")
	assert.Contains(t, js, "analysis")
	assert.Contains(t, js, "recommendations")

	assert.Equal(t, "structured_error", metadata["report_type"])
	assert.Equal(t, true, metadata["is_fallback"])
}

func TestContentExtractionStrategy(t *testing.T) {
	strategy := &ContentExtractionStrategy{format: "text"}

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "meaningful content",
			input:    "This is a long sentence with meaningful content that should be extracted.",
			expected: true,
		},
		{
			name:     "content with noise",
			input:    "```\nshort\n```\nThis is the actual meaningful content that we want to extract.",
			expected: true,
		},
		{
			name:     "only short lines",
			input:    "a\nb\nc\n",
			expected: false, // Falls back to original
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, metadata, err := strategy.Apply(tt.input, "text")

			assert.NoError(t, err)
			assert.NotEmpty(t, result)
			assert.Equal(t, "meaningful_content", metadata["extraction_type"])

			if tt.expected {
				assert.NotEqual(t, tt.input, result, "Should extract different content")
			}
		})
	}
}

func TestFallbackStrategy(t *testing.T) {
	strategy := &FallbackStrategy{}

	result, metadata, err := strategy.Apply("any content", "any format")

	assert.NoError(t, err)
	assert.Contains(t, result, "RECOVERY_FAILED:")
	assert.Contains(t, result, "any content")
	assert.Equal(t, "fallback", metadata["strategy"])
	assert.Equal(t, "all correction attempts failed", metadata["warning"])
}

func TestCleanJSONProperty(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid property",
			input:    `"name": "test"`,
			expected: `"name": "test"`,
		},
		{
			name:     "property with trailing comma",
			input:    `"name": "test",`,
			expected: `"name": "test"`,
		},
		{
			name:     "invalid property",
			input:    "just text",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanJSONProperty(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDefaultTemplate(t *testing.T) {
	tests := []struct {
		format   string
		validate func(string) bool
	}{
		{
			format: "json",
			validate: func(s string) bool {
				var js json.RawMessage
				return json.Unmarshal([]byte(s), &js) == nil
			},
		},
		{
			format: "yaml",
			validate: func(s string) bool {
				return strings.Contains(s, ":") && strings.Contains(s, "error")
			},
		},
		{
			format: "markdown",
			validate: func(s string) bool {
				return strings.Contains(s, "#") && strings.Contains(s, "Recovery")
			},
		},
	}

	for _, tt := range tests {
		t.Run("template for "+tt.format, func(t *testing.T) {
			template := getDefaultTemplate(tt.format)
			assert.NotEmpty(t, template)
			assert.True(t, tt.validate(template), "Template should be valid for format %s", tt.format)
		})
	}
}

func TestAnalyzeContent(t *testing.T) {
	analysis := analyzeContent(`{
		"name": "test",
		"values": [1, 2, 3]
	}`)

	assert.True(t, analysis["has_json_brackets"].(bool))
	assert.True(t, analysis["has_array_brackets"].(bool))
	assert.True(t, analysis["has_colons"].(bool))
	assert.True(t, analysis["has_quotes"].(bool))
	assert.Equal(t, 4, analysis["line_count"].(int))
	assert.False(t, analysis["has_markdown_blocks"].(bool))
}

func TestGetRecoveryRecommendations(t *testing.T) {
	recommendations := getRecoveryRecommendations("Some text without JSON", "json")

	assert.NotEmpty(t, recommendations)
	assert.Contains(t, recommendations, "Ensure output starts with { or [")
	assert.Contains(t, recommendations, "Include proper key:value pairs")

	// Test with markdown content
	recommendations = getRecoveryRecommendations("```json\n{}\n```", "json")
	assert.Contains(t, recommendations, "Remove markdown code block formatting")
}

func TestExtractMeaningfulContent(t *testing.T) {
	input := "\n\n\t```json\n\tThis is a meaningful line with substantial content\n\tshort\n\tAnother meaningful line that should be included\n\t```\n\n\t"

	result := extractMeaningfulContent(input)

	assert.NotContains(t, result, "```")
	assert.Contains(t, result, "meaningful line with substantial content")
	assert.Contains(t, result, "meaningful line that should be included")
	assert.NotContains(t, result, "short")
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a very long string", 10, "this is a ..."},
		{"", 5, ""},
	}

	for _, tt := range tests {
		result := truncateString(tt.input, tt.maxLen)
		assert.Equal(t, tt.expected, result)
		assert.True(t, len(result) <= tt.maxLen+3) // +3 for "..."
	}
}

func TestCorrectionResult(t *testing.T) {
	t.Run("successful correction summary", func(t *testing.T) {
		result := &CorrectionResult{
			Success:         true,
			AppliedStrategy: "direct_json_validation",
			Duration:        150 * time.Millisecond,
		}

		summary := result.FormatSummary()
		assert.Contains(t, summary, "✓")
		assert.Contains(t, summary, "direct_json_validation")
		assert.Contains(t, summary, "150ms")
	})

	t.Run("failed correction summary", func(t *testing.T) {
		result := &CorrectionResult{
			Success: false,
			StrategiesAttempted: []string{"strategy1", "strategy2", "strategy3"},
			Duration: 500 * time.Millisecond,
		}

		summary := result.FormatSummary()
		assert.Contains(t, summary, "✗")
		assert.Contains(t, summary, "3 strategies")
		assert.Contains(t, summary, "500ms")
	})
}

func TestOutputFormatCorrector_ProgressiveStrategies(t *testing.T) {
	corrector := NewOutputFormatCorrector(nil)

	// Test that different attempts try different strategy combinations
	strategies1 := corrector.getCorrectionStrategies("json", 1)
	strategies2 := corrector.getCorrectionStrategies("json", 2)
	strategies3 := corrector.getCorrectionStrategies("json", 3)

	assert.True(t, len(strategies2) > len(strategies1), "Second attempt should have more strategies")
	assert.True(t, len(strategies3) > len(strategies2), "Third attempt should have most strategies")

	// Verify that certain strategies only appear in later attempts
	strategyNames3 := make([]string, len(strategies3))
	for i, s := range strategies3 {
		strategyNames3[i] = s.Name()
	}

	assert.Contains(t, strategyNames3, "partial_json_recovery")
	assert.Contains(t, strategyNames3, "ai_assisted_recovery")
	assert.Contains(t, strategyNames3, "structured_error_report")
}

func BenchmarkOutputCorrection(b *testing.B) {
	corrector := NewOutputFormatCorrector(nil)
	malformedJSON := `{
		"name": "test",
		"data": {
			"values": [1, 2, 3,]
		},
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := corrector.CorrectOutput(malformedJSON, "json", 1)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONExtraction(b *testing.B) {
	content := "Here's some explanatory text followed by:\n\n```json\n{\n  \"name\": \"benchmark\",\n  \"values\": [1, 2, 3, 4, 5],\n  \"nested\": {\n    \"key\": \"value\"\n  }\n}\n```\n\nAnd some more text after."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExtractJSONFromMarkdown(content)
	}
}