package relay

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Checkpoint validation errors
var (
	ErrEmptyCheckpoint      = errors.New("empty checkpoint content")
	ErrMissingHeader        = errors.New("missing checkpoint header")
	ErrMissingSummary       = errors.New("missing summary section")
	ErrEmptySummary         = errors.New("empty summary")
	ErrCheckpointNotFound   = errors.New("checkpoint file not found")
	ErrInvalidCheckpoint    = errors.New("invalid checkpoint format")
)

const CheckpointFilename = "checkpoint.md"

type Checkpoint struct {
	Summary   string
	Decisions []string
	Context   map[string]string
	Generated string
}

func ParseCheckpoint(workspacePath string) (*Checkpoint, error) {
	checkpointPath := filepath.Join(workspacePath, CheckpointFilename)
	content, err := os.ReadFile(checkpointPath)
	if err != nil {
		return nil, fmt.Errorf("checkpoint file not found: %w", err)
	}

	checkpoint := &Checkpoint{
		Context: make(map[string]string),
	}

	lines := strings.Split(string(content), "\n")

	currentSection := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(line, "# Checkpoint"):
			continue
		case strings.HasPrefix(line, "## Summary"):
			currentSection = "summary"
			continue
		case strings.HasPrefix(line, "## Decisions") || strings.HasPrefix(line, "## Decision"):
			currentSection = "decisions"
			continue
		case strings.HasPrefix(line, "##"):
			currentSection = ""
			continue
		case strings.HasPrefix(line, "---"):
			continue
		case strings.HasPrefix(line, "*Generated"):
			checkpoint.Generated = strings.Trim(line, "* ")
			continue
		}

		if line == "" {
			continue
		}

		switch currentSection {
		case "summary":
			if checkpoint.Summary != "" {
				checkpoint.Summary += "\n"
			}
			checkpoint.Summary += line
		case "decisions":
			checkpoint.Decisions = append(checkpoint.Decisions, line)
		}
	}

	return checkpoint, nil
}

func InjectCheckpointPrompt(workspacePath string) (string, error) {
	checkpoint, err := ParseCheckpoint(workspacePath)
	if err != nil {
		return "", err
	}

	var parts []string
	parts = append(parts, "=== READ CHECKPOINT.MD FIRST ===\n")

	if checkpoint.Summary != "" {
		parts = append(parts, fmt.Sprintf("## Summary\n%s\n", checkpoint.Summary))
	}

	if len(checkpoint.Decisions) > 0 {
		parts = append(parts, "## Key Decisions\n")
		for _, decision := range checkpoint.Decisions {
			parts = append(parts, fmt.Sprintf("- %s", decision))
		}
		parts = append(parts, "")
	}

	if checkpoint.Generated != "" {
		parts = append(parts, fmt.Sprintf("*Checkpoint generated: %s*\n", checkpoint.Generated))
	}

	parts = append(parts, "=== END CHECKPOINT ===\n")

	return strings.Join(parts, "\n"), nil
}

func GenerateCheckpoint(summarizedContext string, workspacePath string) error {
	checkpointPath := filepath.Join(workspacePath, CheckpointFilename)

	summary := extractSummary(summarizedContext)
	decisions := extractDecisions(summarizedContext)

	content := fmt.Sprintf(`# Checkpoint

## Summary
%s

## Decisions
%s

---
*Generated at checkpoint - resume from here*
`, summary, strings.Join(decisions, "\n"))

	return os.WriteFile(checkpointPath, []byte(content), 0644)
}

func extractSummary(text string) string {
	lines := strings.Split(text, "\n")
	var summaryLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 20 && !strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "*") {
			summaryLines = append(summaryLines, line)
		}
		if len(summaryLines) >= 3 {
			break
		}
	}

	return strings.Join(summaryLines, "\n")
}

func extractDecisions(text string) []string {
	var decisions []string
	decisionPattern := regexp.MustCompile(`(?i)(?:decided|decision|chose|chosen|selected)[:\s]+(.+)`)

	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if matches := decisionPattern.FindStringSubmatch(line); len(matches) > 1 {
			decisions = append(decisions, matches[1])
		}
	}

	return decisions
}

// ValidateCheckpointFormat validates that checkpoint content follows the expected format.
// It checks for:
// - Non-empty content
// - Checkpoint header (# Checkpoint)
// - Summary section with content
func ValidateCheckpointFormat(content string) error {
	if strings.TrimSpace(content) == "" {
		return ErrEmptyCheckpoint
	}

	lines := strings.Split(content, "\n")
	hasHeader := false
	hasSummarySection := false
	hasSummaryContent := false
	inSummary := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for checkpoint header
		if strings.HasPrefix(trimmed, "# Checkpoint") {
			hasHeader = true
			continue
		}

		// Check for summary section
		if strings.HasPrefix(trimmed, "## Summary") {
			hasSummarySection = true
			inSummary = true
			continue
		}

		// Exit summary section on new header
		if strings.HasPrefix(trimmed, "##") {
			inSummary = false
			continue
		}

		// Check for summary content
		if inSummary && trimmed != "" && !strings.HasPrefix(trimmed, "---") && !strings.HasPrefix(trimmed, "*") {
			hasSummaryContent = true
		}
	}

	if !hasHeader {
		return ErrMissingHeader
	}

	if !hasSummarySection {
		return ErrMissingSummary
	}

	if !hasSummaryContent {
		return ErrEmptySummary
	}

	return nil
}

// ValidateCheckpointFile reads and validates a checkpoint file from the workspace.
func ValidateCheckpointFile(workspacePath string) error {
	checkpointPath := filepath.Join(workspacePath, CheckpointFilename)
	content, err := os.ReadFile(checkpointPath)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCheckpointNotFound, err)
	}

	if err := ValidateCheckpointFormat(string(content)); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidCheckpoint, err)
	}

	return nil
}
