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
