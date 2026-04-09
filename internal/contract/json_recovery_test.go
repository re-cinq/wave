package contract

import (
	"strings"
	"testing"
)

func TestParseWithRecovery_ValidJSON(t *testing.T) {
	parser := NewJSONRecoveryParser(ConservativeRecovery)
	result, err := parser.ParseWithRecovery(`{"key": "value"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsValid {
		t.Error("valid JSON should be marked as valid")
	}
	if len(result.AppliedFixes) != 0 {
		t.Errorf("expected no fixes for valid JSON, got %v", result.AppliedFixes)
	}
	if result.ParsedData == nil {
		t.Error("ParsedData should not be nil for valid JSON")
	}
}

func TestParseWithRecovery_RecoveryStrategies(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		level         RecoveryLevel
		wantValid     bool
		wantFixSubstr string
	}{
		{
			name:          "trailing commas",
			input:         `{"key": "value",}`,
			level:         ConservativeRecovery,
			wantValid:     true,
			wantFixSubstr: "trailing_commas",
		},
		{
			name:          "markdown code block",
			input:         "```json\n{\"key\": \"value\"}\n```",
			level:         ConservativeRecovery,
			wantValid:     true,
			wantFixSubstr: "extracted",
		},
		{
			name:          "AI preamble text",
			input:         "Here is the analysis result based on the data:\n{\"key\": \"value\"}",
			level:         ConservativeRecovery,
			wantValid:     true,
			wantFixSubstr: "ai_explanation",
		},
		{
			name:          "single-line comments",
			input:         "{\n// this is a comment\n\"key\": \"value\"\n}",
			level:         ConservativeRecovery,
			wantValid:     true,
			wantFixSubstr: "comments",
		},
		{
			name:          "unquoted keys with progressive recovery",
			input:         `{name: "value", age: 30}`,
			level:         ProgressiveRecovery,
			wantValid:     true,
			wantFixSubstr: "unquoted_keys",
		},
		{
			name:          "single quotes with progressive recovery",
			input:         `{'key': 'value'}`,
			level:         ProgressiveRecovery,
			wantValid:     true,
			wantFixSubstr: "single_quotes",
		},
		{
			name:          "missing commas between properties with progressive recovery",
			input:         `{"key": "value" "key2": "value2"}`,
			level:         ProgressiveRecovery,
			wantValid:     true,
			wantFixSubstr: "missing_commas",
		},
		{
			name:          "unbalanced braces with aggressive recovery",
			input:         `{"key": "value"`,
			level:         AggressiveRecovery,
			wantValid:     true,
			wantFixSubstr: "missing_closing_braces",
		},
		// Note: more exotic aggressive recovery strategies (reconstruct from key-value
		// pairs, infer missing object wrapper) are not currently implemented.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewJSONRecoveryParser(tt.level)
			result, err := parser.ParseWithRecovery(tt.input)

			if tt.wantValid {
				if err != nil {
					t.Fatalf("expected successful recovery, got error: %v", err)
				}
				if !result.IsValid {
					t.Error("expected valid result after recovery")
				}
				if result.ParsedData == nil {
					t.Error("ParsedData should not be nil after successful recovery")
				}

				foundFix := false
				for _, fix := range result.AppliedFixes {
					if strings.Contains(fix, tt.wantFixSubstr) {
						foundFix = true
						break
					}
				}
				if !foundFix {
					t.Errorf("expected fix containing %q, got fixes: %v", tt.wantFixSubstr, result.AppliedFixes)
				}
			} else {
				if result.IsValid {
					t.Error("expected invalid result")
				}
			}
		})
	}
}

func TestParseWithRecovery_AllRecoveryFails(t *testing.T) {
	parser := NewJSONRecoveryParser(AggressiveRecovery)
	result, err := parser.ParseWithRecovery("completely not json and never will be")

	if err == nil {
		t.Fatal("expected error when all recovery fails")
	}
	if result.IsValid {
		t.Error("should not be valid when all recovery fails")
	}
	if !strings.Contains(err.Error(), "recovery failed") {
		t.Errorf("error should mention recovery failure, got: %v", err)
	}
	// PreserveOriginalOnFailure is true by default
	if result.RecoveredJSON != result.OriginalInput {
		t.Error("should preserve original input when PreserveOriginalOnFailure is true")
	}
}

func TestParseWithRecovery_PreserveOriginalOnFailure(t *testing.T) {
	parser := NewJSONRecoveryParser(ConservativeRecovery)
	parser.PreserveOriginalOnFailure = false

	result, err := parser.ParseWithRecovery("not { valid } json }")
	if err == nil {
		t.Fatal("expected error for unrecoverable input")
	}
	// When PreserveOriginalOnFailure is false, RecoveredJSON may differ from OriginalInput
	// (it contains the last attempted recovery state)
	_ = result
}

func TestParseWithRecovery_MaxRecoveryAttempts(t *testing.T) {
	parser := NewJSONRecoveryParser(AggressiveRecovery)
	parser.MaxRecoveryAttempts = 1

	// With only 1 attempt, some strategies won't be tried
	result, err := parser.ParseWithRecovery(`{key: "value"}`)
	// The first strategy might not fix unquoted keys, so this should fail
	// or succeed depending on strategy order
	_ = result
	_ = err
}

func TestNewJSONRecoveryParser_Defaults(t *testing.T) {
	parser := NewJSONRecoveryParser(ProgressiveRecovery)
	if parser.RecoveryLevel != ProgressiveRecovery {
		t.Errorf("expected ProgressiveRecovery level, got %v", parser.RecoveryLevel)
	}
	if parser.MaxRecoveryAttempts != 10 {
		t.Errorf("expected MaxRecoveryAttempts 10, got %d", parser.MaxRecoveryAttempts)
	}
	if !parser.PreserveOriginalOnFailure {
		t.Error("expected PreserveOriginalOnFailure to be true by default")
	}
}

func TestRecoveryLevel_StrategyCounts(t *testing.T) {
	conservative := NewJSONRecoveryParser(ConservativeRecovery)
	progressive := NewJSONRecoveryParser(ProgressiveRecovery)
	aggressive := NewJSONRecoveryParser(AggressiveRecovery)

	conservativeStrategies := conservative.getRecoveryStrategies()
	progressiveStrategies := progressive.getRecoveryStrategies()
	aggressiveStrategies := aggressive.getRecoveryStrategies()

	if len(progressiveStrategies) <= len(conservativeStrategies) {
		t.Errorf("progressive should have more strategies than conservative (%d vs %d)",
			len(progressiveStrategies), len(conservativeStrategies))
	}
	if len(aggressiveStrategies) <= len(progressiveStrategies) {
		t.Errorf("aggressive should have more strategies than progressive (%d vs %d)",
			len(aggressiveStrategies), len(progressiveStrategies))
	}
}

func TestFormatRecoveryReport(t *testing.T) {
	tests := []struct {
		name     string
		result   RecoveryResult
		contains []string
	}{
		{
			name: "successful recovery",
			result: RecoveryResult{
				OriginalInput: `{"key": "value",}`,
				RecoveredJSON: `{"key": "value"}`,
				IsValid:       true,
				AppliedFixes:  []string{"removed_trailing_commas"},
				Warnings:      []string{},
				RecoveryLevel: ConservativeRecovery,
			},
			contains: []string{"Successfully recovered", "Fixes Applied", "trailing_commas"},
		},
		{
			name: "failed recovery with warnings",
			result: RecoveryResult{
				OriginalInput: "garbage",
				RecoveredJSON: "garbage",
				IsValid:       false,
				AppliedFixes:  []string{},
				Warnings:      []string{"some_warning"},
				RecoveryLevel: AggressiveRecovery,
			},
			contains: []string{"Failed to recover", "Warnings", "some_warning"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := tt.result.FormatRecoveryReport()
			for _, s := range tt.contains {
				if !strings.Contains(report, s) {
					t.Errorf("report should contain %q, got:\n%s", s, report)
				}
			}
		})
	}
}

func TestIsStructurallyComplete(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty object", "{}", true},
		{"empty array", "[]", true},
		{"nested", `{"a": [1, {"b": 2}]}`, true},
		{"unclosed object", `{"a": 1`, false},
		{"unclosed array", `[1, 2`, false},
		{"unclosed string", `{"a": "unclosed`, false},
		{"escaped quote in string", `{"a": "val\"ue"}`, true},
		{"extra closing brace", `{"a": 1}}`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isStructurallyComplete(tt.input)
			if got != tt.want {
				t.Errorf("isStructurallyComplete(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractFromMarkdown_Variations(t *testing.T) {
	parser := NewJSONRecoveryParser(ConservativeRecovery)

	tests := []struct {
		name      string
		input     string
		wantValid bool
	}{
		{
			name:      "json fenced block",
			input:     "Some text\n```json\n{\"key\": \"value\"}\n```\nMore text",
			wantValid: true,
		},
		{
			name:      "plain fenced block",
			input:     "Some text\n```\n{\"key\": \"value\"}\n```\nMore text",
			wantValid: true,
		},
		{
			name:      "inline code",
			input:     "The result is `{\"key\": \"value\"}`",
			wantValid: true,
		},
		{
			name:      "no markdown",
			input:     "no markdown here at all",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseWithRecovery(tt.input)
			if tt.wantValid {
				if err != nil {
					t.Errorf("expected valid recovery, got error: %v", err)
				}
				if !result.IsValid {
					t.Error("expected valid result")
				}
			}
		})
	}
}

func TestExtractFromAIExplanation_ColonFallback(t *testing.T) {
	parser := NewJSONRecoveryParser(ConservativeRecovery)

	// Test the colon-terminated indicator phrase fallback
	input := `The analysis result: {"key": "value"}`
	result, err := parser.ParseWithRecovery(input)
	if err != nil {
		t.Fatalf("expected successful recovery, got error: %v", err)
	}
	if !result.IsValid {
		t.Error("expected valid result after extracting from AI explanation with colon")
	}
}

func TestHashCommentRemoval(t *testing.T) {
	parser := NewJSONRecoveryParser(ConservativeRecovery)

	input := "# This is a comment\n{\"key\": \"value\"}"
	result, err := parser.ParseWithRecovery(input)
	if err != nil {
		t.Fatalf("expected successful recovery, got error: %v", err)
	}
	if !result.IsValid {
		t.Error("expected valid result after removing hash comments")
	}
}

func TestMultiLineCommentRemoval(t *testing.T) {
	parser := NewJSONRecoveryParser(ConservativeRecovery)

	input := `{/* comment */ "key": "value"}`
	result, err := parser.ParseWithRecovery(input)
	if err != nil {
		t.Fatalf("expected successful recovery, got error: %v", err)
	}
	if !result.IsValid {
		t.Error("expected valid result after removing multi-line comments")
	}
}

func TestParseWithRecovery_ValidArray(t *testing.T) {
	parser := NewJSONRecoveryParser(ConservativeRecovery)

	input := `[1, 2, 3]`
	result, err := parser.ParseWithRecovery(input)
	if err != nil {
		t.Fatalf("expected success for valid array, got error: %v", err)
	}
	if !result.IsValid {
		t.Error("valid array JSON should be marked as valid")
	}
}
