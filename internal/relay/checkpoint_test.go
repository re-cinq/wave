package relay

import (
	"os"
	"path/filepath"
	"testing"
)

// =============================================================================
// T067: Checkpoint Parsing Tests
// =============================================================================

func TestParseCheckpoint_ValidFormats(t *testing.T) {
	testCases := []struct {
		name              string
		content           string
		expectedSummary   string
		expectedDecisions []string
		expectedGenerated string
	}{
		{
			name: "standard checkpoint format",
			content: `# Checkpoint

## Summary
This is the summary of the conversation.
It spans multiple lines.

## Decisions
Decision 1: Use Go for implementation
Decision 2: Use SQLite for persistence

---
*Generated at checkpoint - resume from here*
`,
			expectedSummary:   "This is the summary of the conversation.\nIt spans multiple lines.",
			expectedDecisions: []string{"Decision 1: Use Go for implementation", "Decision 2: Use SQLite for persistence"},
			expectedGenerated: "Generated at checkpoint - resume from here",
		},
		{
			name: "checkpoint with Decision singular header",
			content: `# Checkpoint

## Summary
Short summary.

## Decision
Single decision here

---
*Generated at test*
`,
			expectedSummary:   "Short summary.",
			expectedDecisions: []string{"Single decision here"},
			expectedGenerated: "Generated at test",
		},
		{
			name: "checkpoint with empty decisions",
			content: `# Checkpoint

## Summary
Just a summary, no decisions.

---
*Generated at checkpoint*
`,
			expectedSummary:   "Just a summary, no decisions.",
			expectedDecisions: nil,
			expectedGenerated: "Generated at checkpoint",
		},
		{
			name: "minimal checkpoint",
			content: `# Checkpoint

## Summary
Minimal content.
`,
			expectedSummary:   "Minimal content.",
			expectedDecisions: nil,
			expectedGenerated: "",
		},
		{
			name: "checkpoint with extra sections",
			content: `# Checkpoint

## Summary
Main summary here.

## Context
This section should be ignored.

## Decisions
Important decision

## Notes
This should also be ignored.

---
*Generated at test*
`,
			expectedSummary:   "Main summary here.",
			expectedDecisions: []string{"Important decision"},
			expectedGenerated: "Generated at test",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			workspacePath := t.TempDir()
			checkpointPath := filepath.Join(workspacePath, CheckpointFilename)
			err := os.WriteFile(checkpointPath, []byte(tc.content), 0644)
			if err != nil {
				t.Fatalf("failed to write test checkpoint: %v", err)
			}

			checkpoint, err := ParseCheckpoint(workspacePath)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if checkpoint.Summary != tc.expectedSummary {
				t.Errorf("summary mismatch:\ngot:  %q\nwant: %q", checkpoint.Summary, tc.expectedSummary)
			}

			if len(checkpoint.Decisions) != len(tc.expectedDecisions) {
				t.Errorf("decisions count mismatch: got %d, want %d", len(checkpoint.Decisions), len(tc.expectedDecisions))
			} else {
				for i, decision := range checkpoint.Decisions {
					if decision != tc.expectedDecisions[i] {
						t.Errorf("decision[%d] mismatch:\ngot:  %q\nwant: %q", i, decision, tc.expectedDecisions[i])
					}
				}
			}

			if checkpoint.Generated != tc.expectedGenerated {
				t.Errorf("generated mismatch:\ngot:  %q\nwant: %q", checkpoint.Generated, tc.expectedGenerated)
			}
		})
	}
}

func TestParseCheckpoint_ErrorCases(t *testing.T) {
	testCases := []struct {
		name          string
		setup         func(workspacePath string) error
		expectedError string
	}{
		{
			name: "checkpoint file not found",
			setup: func(workspacePath string) error {
				// Don't create any file
				return nil
			},
			expectedError: "checkpoint file not found",
		},
		{
			name: "checkpoint file is directory",
			setup: func(workspacePath string) error {
				// Create a directory instead of file
				return os.MkdirAll(filepath.Join(workspacePath, CheckpointFilename), 0755)
			},
			expectedError: "checkpoint file not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			workspacePath := t.TempDir()
			if err := tc.setup(workspacePath); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			checkpoint, err := ParseCheckpoint(workspacePath)
			if err == nil {
				t.Fatalf("expected error, got checkpoint: %+v", checkpoint)
			}

			if tc.expectedError != "" && !containsString(err.Error(), tc.expectedError) {
				t.Errorf("error mismatch:\ngot:  %q\nwant to contain: %q", err.Error(), tc.expectedError)
			}
		})
	}
}

func TestParseCheckpoint_EdgeCases(t *testing.T) {
	testCases := []struct {
		name    string
		content string
	}{
		{
			name:    "empty file",
			content: "",
		},
		{
			name:    "only header",
			content: "# Checkpoint\n",
		},
		{
			name:    "whitespace only",
			content: "   \n\t\n   \n",
		},
		{
			name: "no summary section",
			content: `# Checkpoint

## Decisions
Some decision

---
*Generated*
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			workspacePath := t.TempDir()
			checkpointPath := filepath.Join(workspacePath, CheckpointFilename)
			err := os.WriteFile(checkpointPath, []byte(tc.content), 0644)
			if err != nil {
				t.Fatalf("failed to write test checkpoint: %v", err)
			}

			// Should not panic or error - just return empty/partial checkpoint
			checkpoint, err := ParseCheckpoint(workspacePath)
			if err != nil {
				t.Fatalf("unexpected error for edge case: %v", err)
			}
			if checkpoint == nil {
				t.Fatal("checkpoint should not be nil")
			}
		})
	}
}

// =============================================================================
// T068: Checkpoint Injection Tests
// =============================================================================

func TestInjectCheckpointPrompt_Success(t *testing.T) {
	testCases := []struct {
		name             string
		content          string
		expectedContains []string
	}{
		{
			name: "full checkpoint injection",
			content: `# Checkpoint

## Summary
This is the conversation summary.

## Decisions
Decision about architecture

---
*Generated at checkpoint*
`,
			expectedContains: []string{
				"READ CHECKPOINT.MD FIRST",
				"This is the conversation summary",
				"Decision about architecture",
				"END CHECKPOINT",
			},
		},
		{
			name: "injection with only summary",
			content: `# Checkpoint

## Summary
Only a summary, no decisions.
`,
			expectedContains: []string{
				"READ CHECKPOINT.MD FIRST",
				"Only a summary, no decisions",
				"END CHECKPOINT",
			},
		},
		{
			name: "injection with multiple decisions",
			content: `# Checkpoint

## Summary
Brief summary.

## Decisions
First decision
Second decision
Third decision

---
*Generated at test*
`,
			expectedContains: []string{
				"Key Decisions",
				"First decision",
				"Second decision",
				"Third decision",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			workspacePath := t.TempDir()
			checkpointPath := filepath.Join(workspacePath, CheckpointFilename)
			err := os.WriteFile(checkpointPath, []byte(tc.content), 0644)
			if err != nil {
				t.Fatalf("failed to write test checkpoint: %v", err)
			}

			prompt, err := InjectCheckpointPrompt(workspacePath)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, expected := range tc.expectedContains {
				if !containsString(prompt, expected) {
					t.Errorf("prompt should contain %q\ngot: %s", expected, prompt)
				}
			}
		})
	}
}

func TestInjectCheckpointPrompt_NoCheckpoint(t *testing.T) {
	workspacePath := t.TempDir()

	_, err := InjectCheckpointPrompt(workspacePath)
	if err == nil {
		t.Fatal("expected error when no checkpoint exists")
	}

	if !containsString(err.Error(), "checkpoint file not found") {
		t.Errorf("error should mention checkpoint file not found, got: %v", err)
	}
}

// =============================================================================
// T070: Checkpoint Format Validation Tests
// =============================================================================

func TestValidateCheckpointFormat(t *testing.T) {
	testCases := []struct {
		name        string
		content     string
		expectValid bool
		expectError string
	}{
		{
			name: "valid checkpoint",
			content: `# Checkpoint

## Summary
Valid summary content.

## Decisions
Valid decision

---
*Generated at checkpoint*
`,
			expectValid: true,
		},
		{
			name:        "empty content",
			content:     "",
			expectValid: false,
			expectError: "empty checkpoint content",
		},
		{
			name:        "missing checkpoint header",
			content:     "## Summary\nSome content\n",
			expectValid: false,
			expectError: "missing checkpoint header",
		},
		{
			name: "missing summary section",
			content: `# Checkpoint

## Decisions
Just decisions, no summary
`,
			expectValid: false,
			expectError: "missing summary section",
		},
		{
			name: "empty summary",
			content: `# Checkpoint

## Summary

## Decisions
Has decisions but no summary
`,
			expectValid: false,
			expectError: "empty summary",
		},
		{
			name: "valid with only summary",
			content: `# Checkpoint

## Summary
Has content in summary.
`,
			expectValid: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCheckpointFormat(tc.content)

			if tc.expectValid {
				if err != nil {
					t.Errorf("expected valid checkpoint, got error: %v", err)
				}
			} else {
				if err == nil {
					t.Error("expected validation error, got nil")
				} else if tc.expectError != "" && !containsString(err.Error(), tc.expectError) {
					t.Errorf("error mismatch:\ngot:  %q\nwant to contain: %q", err.Error(), tc.expectError)
				}
			}
		})
	}
}

func TestValidateCheckpointFile(t *testing.T) {
	t.Run("valid checkpoint file", func(t *testing.T) {
		workspacePath := t.TempDir()
		content := `# Checkpoint

## Summary
Valid summary.

---
*Generated*
`
		checkpointPath := filepath.Join(workspacePath, CheckpointFilename)
		err := os.WriteFile(checkpointPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to write checkpoint: %v", err)
		}

		err = ValidateCheckpointFile(workspacePath)
		if err != nil {
			t.Errorf("expected valid file, got error: %v", err)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		workspacePath := t.TempDir()
		err := ValidateCheckpointFile(workspacePath)
		if err == nil {
			t.Error("expected error for missing file")
		}
	})

	t.Run("invalid content", func(t *testing.T) {
		workspacePath := t.TempDir()
		content := "invalid content without proper format"
		checkpointPath := filepath.Join(workspacePath, CheckpointFilename)
		err := os.WriteFile(checkpointPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to write checkpoint: %v", err)
		}

		err = ValidateCheckpointFile(workspacePath)
		if err == nil {
			t.Error("expected validation error for invalid content")
		}
	})
}

// =============================================================================
// Generate Checkpoint Tests
// =============================================================================

func TestGenerateCheckpoint(t *testing.T) {
	testCases := []struct {
		name              string
		summarizedContext string
		expectedContains  []string
	}{
		{
			name: "generates from summarized context",
			summarizedContext: `This is a summary of the conversation about building a CLI tool.
We discussed various approaches and decided to use Go.
The architecture decision was made to use a plugin system.`,
			expectedContains: []string{
				"# Checkpoint",
				"## Summary",
				"## Decisions",
			},
		},
		{
			name: "extracts decisions from text",
			summarizedContext: `Summary text here.
We decided to use PostgreSQL for the database.
The team chose React for the frontend.
Selected TypeScript as the language.`,
			expectedContains: []string{
				"# Checkpoint",
				"## Summary",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			workspacePath := t.TempDir()

			err := GenerateCheckpoint(tc.summarizedContext, workspacePath)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Read the generated file
			checkpointPath := filepath.Join(workspacePath, CheckpointFilename)
			content, err := os.ReadFile(checkpointPath)
			if err != nil {
				t.Fatalf("failed to read generated checkpoint: %v", err)
			}

			for _, expected := range tc.expectedContains {
				if !containsString(string(content), expected) {
					t.Errorf("checkpoint should contain %q\ngot: %s", expected, string(content))
				}
			}

			// Verify the generated checkpoint is valid
			err = ValidateCheckpointFormat(string(content))
			if err != nil {
				t.Errorf("generated checkpoint is not valid: %v", err)
			}
		})
	}
}

func TestExtractSummary(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "extracts first substantive lines",
			input: `This is the first line with enough content.
This is the second line with enough content.
This is the third line with enough content.
This is the fourth line which should not be included.`,
			expected: "This is the first line with enough content.\nThis is the second line with enough content.\nThis is the third line with enough content.",
		},
		{
			name: "skips short lines",
			input: `Short
Another short
This line has enough content to be included.
This is another good line with content.`,
			expected: "This line has enough content to be included.\nThis is another good line with content.",
		},
		{
			name: "skips bullet points",
			input: `- This is a bullet point
* This is another bullet point
This is a normal line with content.`,
			expected: "This is a normal line with content.",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractSummary(tc.input)
			if result != tc.expected {
				t.Errorf("summary mismatch:\ngot:  %q\nwant: %q", result, tc.expected)
			}
		})
	}
}

func TestExtractDecisions(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "extracts decided pattern",
			input:    "We decided to use Go for the implementation.",
			expected: []string{"to use Go for the implementation."},
		},
		{
			name:     "extracts chose pattern",
			input:    "The team chose React for the frontend.",
			expected: []string{"React for the frontend."},
		},
		{
			name:     "extracts selected pattern",
			input:    "Selected TypeScript as the language.",
			expected: []string{"TypeScript as the language."},
		},
		{
			name: "extracts multiple decisions",
			input: `We decided to use PostgreSQL.
The team chose microservices architecture.
Selected Kubernetes for deployment.`,
			expected: []string{
				"to use PostgreSQL.",
				"microservices architecture.",
				"Kubernetes for deployment.",
			},
		},
		{
			name:     "no decisions found",
			input:    "This text has no relevant patterns in it.",
			expected: nil,
		},
		{
			name:     "empty input",
			input:    "",
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractDecisions(tc.input)
			if len(result) != len(tc.expected) {
				t.Errorf("decisions count mismatch: got %d, want %d\ngot: %v\nwant: %v", len(result), len(tc.expected), result, tc.expected)
				return
			}
			for i, decision := range result {
				if decision != tc.expected[i] {
					t.Errorf("decision[%d] mismatch:\ngot:  %q\nwant: %q", i, decision, tc.expected[i])
				}
			}
		})
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
