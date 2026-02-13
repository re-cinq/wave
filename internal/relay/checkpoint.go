package relay

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Checkpoint validation errors
var (
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

