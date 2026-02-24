package recovery

import (
	"strings"
	"testing"
)

func TestFormatRecoveryBlock_AllHintTypes(t *testing.T) {
	block := BuildRecoveryBlock("feature", "add auth", "implement", "feature-abc123", "", ClassContractValidation, nil)
	output := FormatRecoveryBlock(block)

	if !strings.Contains(output, "Recovery options:") {
		t.Error("expected 'Recovery options:' header in output")
	}
	if !strings.Contains(output, "Resume from failed step") {
		t.Error("expected resume hint in output")
	}
	if !strings.Contains(output, "Resume and skip validation checks") {
		t.Error("expected force hint in output")
	}
	if !strings.Contains(output, "Inspect workspace artifacts") {
		t.Error("expected workspace hint in output")
	}
	if !strings.Contains(output, "wave run feature") {
		t.Error("expected wave run command in output")
	}

	// Verify ≤ 8 content lines (excluding blank separator lines)
	contentLines := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) != "" {
			contentLines++
		}
	}
	if contentLines > 8 {
		t.Errorf("content lines = %d, want ≤ 8", contentLines)
	}
}

func TestFormatRecoveryBlock_ResumeOnly(t *testing.T) {
	block := BuildRecoveryBlock("feature", "add auth", "implement", "feature-abc123", "", ClassSecurityViolation, nil)
	output := FormatRecoveryBlock(block)

	if !strings.Contains(output, "Recovery options:") {
		t.Error("expected 'Recovery options:' header in output")
	}
	if !strings.Contains(output, "Resume from failed step") {
		t.Error("expected resume hint in output")
	}
	// Security errors should not have force or debug hints
	if strings.Contains(output, "Resume and skip validation checks") {
		t.Error("unexpected force hint in security error output")
	}
	if strings.Contains(output, "Re-run with debug output") {
		t.Error("unexpected debug hint in security error output")
	}
}

func TestFormatRecoveryBlock_NilBlock(t *testing.T) {
	output := FormatRecoveryBlock(nil)
	if output != "" {
		t.Errorf("expected empty output for nil block, got %q", output)
	}
}

func TestFormatRecoveryBlock_EmptyHints(t *testing.T) {
	block := &RecoveryBlock{Hints: []RecoveryHint{}}
	output := FormatRecoveryBlock(block)
	if output != "" {
		t.Errorf("expected empty output for empty hints, got %q", output)
	}
}

func TestFormatRecoveryBlock_Indentation(t *testing.T) {
	block := BuildRecoveryBlock("feature", "add auth", "implement", "feature-abc123", "", ClassRuntimeError, nil)
	output := FormatRecoveryBlock(block)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if line == "Recovery options:" {
			continue
		}
		// Labels should be indented with 2 spaces
		if strings.HasSuffix(line, ":") && !strings.HasPrefix(line, "  ") {
			t.Errorf("label line not properly indented: %q", line)
		}
		// Commands should be indented with 4 spaces
		if strings.HasPrefix(line, "    wave ") || strings.HasPrefix(line, "    ls ") {
			continue // properly indented command
		}
	}
}

func TestFormatRecoveryBlock_LineCount(t *testing.T) {
	// Runtime error block has resume, workspace, debug = 3 hints = 7 content lines (1 header + 3*2 label+command)
	block := BuildRecoveryBlock("feature", "add auth", "implement", "feature-abc123", "", ClassRuntimeError, nil)
	output := FormatRecoveryBlock(block)

	contentLines := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) != "" {
			contentLines++
		}
	}
	if contentLines > 8 {
		t.Errorf("content lines = %d, want ≤ 8", contentLines)
	}
}
