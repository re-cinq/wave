package onboarding

import (
	"fmt"
	"os"
	"path/filepath"
)

// waveCommandTemplate is the Claude Code command file content for the /wave slash command.
// It follows the same format as other .claude/commands/*.md files (YAML frontmatter + markdown body).
const waveCommandTemplate = `---
description: Run Wave multi-agent pipelines
---

## User Input

` + "```text" + `
$ARGUMENTS
` + "```" + `

## Instructions

You are invoking the Wave multi-agent pipeline orchestrator. Parse the user's arguments to determine which subcommand to run.

### Subcommand Routing

Based on the arguments provided:

**If arguments start with "run"** (e.g., ` + "`/wave run impl-issue -- \"fix bug\"`" + `):
- Execute: ` + "`wave run <remaining arguments>`" + `
- Example: ` + "`wave run -v impl-issue -- \"implement feature X\"`" + `

**If arguments start with "status"** (e.g., ` + "`/wave status`" + `):
- Execute: ` + "`wave list runs --limit 10`" + `
- Show the output to the user in a readable format

**If arguments start with "list"** (e.g., ` + "`/wave list`" + `):
- Execute: ` + "`wave list pipelines`" + `
- Show available pipelines to the user

**If arguments start with "logs"** (e.g., ` + "`/wave logs <run-id>`" + `):
- Execute: ` + "`wave logs <run-id>`" + `
- Show the pipeline run logs

**If no arguments or "help"**:
- Show available subcommands: run, status, list, logs
- Example usage for each subcommand
`

// WaveCommandStep generates a .claude/commands/wave.md file during onboarding,
// enabling /wave as a slash command in Claude Code sessions.
type WaveCommandStep struct{}

// Name returns the display name for this wizard step.
func (s *WaveCommandStep) Name() string { return "Wave Command Registration" }

// Run generates the .claude/commands/wave.md command file.
func (s *WaveCommandStep) Run(cfg *WizardConfig) (*StepResult, error) {
	// Determine base directory for .claude/commands/
	baseDir := "."
	if cfg.WaveDir != "" {
		baseDir = filepath.Dir(cfg.WaveDir)
	}

	commandDir := filepath.Join(baseDir, ".claude", "commands")
	if err := os.MkdirAll(commandDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .claude/commands directory: %w", err)
	}

	commandFile := filepath.Join(commandDir, "wave.md")
	if err := os.WriteFile(commandFile, []byte(waveCommandTemplate), 0644); err != nil {
		return nil, fmt.Errorf("failed to write wave command file: %w", err)
	}

	return &StepResult{
		Data: map[string]interface{}{
			"wave_command_generated": true,
		},
	}, nil
}
