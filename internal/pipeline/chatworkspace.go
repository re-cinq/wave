package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ChatMode determines the permission level for a chat workspace.
type ChatMode string

const (
	ChatModeAnalysis   ChatMode = "analysis"   // Phase 1: read-only
	ChatModeManipulate ChatMode = "manipulate"  // Phase 2: read-write
)

// ChatWorkspaceOptions configures the chat workspace preparation.
type ChatWorkspaceOptions struct {
	Model string   // Model override (e.g., "sonnet", "opus")
	Mode  ChatMode // defaults to ChatModeAnalysis if empty
}

func (opts ChatWorkspaceOptions) effectiveMode() ChatMode {
	if opts.Mode == "" {
		return ChatModeAnalysis
	}
	return opts.Mode
}

// PrepareChatWorkspace creates a workspace directory with CLAUDE.md and settings.json
// for an interactive wave chat session. Returns the workspace path.
func PrepareChatWorkspace(ctx *ChatContext, opts ChatWorkspaceOptions) (string, error) {
	// 1. Create chat workspace directory
	wsDir := filepath.Join(ctx.ProjectRoot, ".wave", "chat", ctx.Run.RunID)
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create chat workspace: %w", err)
	}

	mode := opts.effectiveMode()

	// 2. Build and write CLAUDE.md
	claudeMd := buildChatClaudeMd(ctx, mode)
	claudeMdPath := filepath.Join(wsDir, "CLAUDE.md")
	if err := os.WriteFile(claudeMdPath, []byte(claudeMd), 0644); err != nil {
		return "", fmt.Errorf("failed to write CLAUDE.md: %w", err)
	}

	// 3. Build and write .claude/settings.json
	settingsDir := filepath.Join(wsDir, ".claude")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create .claude directory: %w", err)
	}

	model := opts.Model
	if model == "" {
		model = "sonnet"
	}

	var perms chatPermissions
	switch mode {
	case ChatModeManipulate:
		perms = chatPermissions{
			Allow: []string{
				"Read", "Glob", "Grep", "Write", "Edit",
				"Bash(git:*)", "Bash(go:*)", "Bash(make:*)",
				"Bash(ls:*)", "Bash(cat:*)", "Bash(find:*)", "Bash(wc:*)",
				"Bash(mkdir:*)", "Bash(cp:*)", "Bash(mv:*)",
			},
			Deny: []string{
				"NotebookEdit",
				"Bash(rm -rf /)*",
			},
		}
	default: // ChatModeAnalysis
		perms = chatPermissions{
			Allow: []string{
				"Read",
				"Glob",
				"Grep",
				"Bash(git log:*)",
				"Bash(git diff:*)",
				"Bash(git show:*)",
				"Bash(ls:*)",
				"Bash(cat:*)",
				"Bash(find:*)",
				"Bash(wc:*)",
			},
			Deny: []string{
				"Write",
				"Edit",
				"NotebookEdit",
				"Bash(rm:*)",
				"Bash(mv:*)",
			},
		}
	}

	settings := chatSettings{
		Model:       model,
		Permissions: perms,
	}

	settingsData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal settings: %w", err)
	}

	settingsPath := filepath.Join(settingsDir, "settings.json")
	if err := os.WriteFile(settingsPath, settingsData, 0644); err != nil {
		return "", fmt.Errorf("failed to write settings.json: %w", err)
	}

	// 4. Provision slash commands for manipulate mode
	if mode == ChatModeManipulate {
		if err := provisionSlashCommands(wsDir); err != nil {
			return "", fmt.Errorf("failed to provision slash commands: %w", err)
		}
	}

	return wsDir, nil
}

type chatSettings struct {
	Model       string          `json:"model"`
	Permissions chatPermissions `json:"permissions"`
}

type chatPermissions struct {
	Allow []string `json:"allow"`
	Deny  []string `json:"deny"`
}

// buildChatClaudeMd generates the CLAUDE.md content for a chat session.
func buildChatClaudeMd(ctx *ChatContext, mode ChatMode) string {
	var b strings.Builder

	b.WriteString("# Wave Pipeline Analysis\n\n")
	b.WriteString("You are analyzing a completed Wave pipeline run.\n\n")

	// Run summary table
	b.WriteString("## Run Summary\n\n")
	b.WriteString("| Field | Value |\n")
	b.WriteString("|-------|-------|\n")
	fmt.Fprintf(&b, "| Run ID | `%s` |\n", ctx.Run.RunID)
	fmt.Fprintf(&b, "| Pipeline | %s |\n", ctx.Run.PipelineName)
	fmt.Fprintf(&b, "| Status | %s |\n", ctx.Run.Status)

	if ctx.Run.CompletedAt != nil {
		elapsed := ctx.Run.CompletedAt.Sub(ctx.Run.StartedAt)
		fmt.Fprintf(&b, "| Duration | %s |\n", chatFormatDuration(elapsed))
	}
	fmt.Fprintf(&b, "| Tokens | %s |\n", chatFormatTokens(ctx.Run.TotalTokens))

	if ctx.Run.Input != "" {
		input := ctx.Run.Input
		if len(input) > 100 {
			input = input[:97] + "..."
		}
		fmt.Fprintf(&b, "| Input | %s |\n", input)
	}

	if ctx.Run.ErrorMessage != "" {
		fmt.Fprintf(&b, "| Error | %s |\n", ctx.Run.ErrorMessage)
	}

	// Step results table
	b.WriteString("\n## Step Results\n\n")
	b.WriteString("| # | Step | Persona | Status | Duration | Tokens |\n")
	b.WriteString("|---|------|---------|--------|----------|--------|\n")
	for i, step := range ctx.Steps {
		state := step.State
		if state == "" {
			state = "pending"
		}
		fmt.Fprintf(&b, "| %d | %s | %s | %s | %s | %s |\n",
			i+1, step.StepID, step.Persona, state,
			chatFormatDuration(step.Duration), chatFormatTokens(step.TokensUsed))
	}

	// Artifacts inventory
	if len(ctx.Artifacts) > 0 {
		b.WriteString("\n## Artifacts\n\n")
		b.WriteString("| Step | Name | Type | Path | Size |\n")
		b.WriteString("|------|------|------|------|------|\n")
		for _, art := range ctx.Artifacts {
			artType := art.Type
			if artType == "" {
				artType = "-"
			}
			fmt.Fprintf(&b, "| %s | %s | %s | `%s` | %s |\n",
				art.StepID, art.Name, artType, art.Path, chatFormatSize(art.SizeBytes))
		}
	}

	// Step workspaces
	hasWorkspaces := false
	for _, step := range ctx.Steps {
		if step.WorkspacePath != "" {
			hasWorkspaces = true
			break
		}
	}
	if hasWorkspaces {
		b.WriteString("\n## Step Workspaces\n\n")
		b.WriteString("Each step's workspace is preserved. You can read files from these locations:\n\n")
		for _, step := range ctx.Steps {
			if step.WorkspacePath != "" {
				fmt.Fprintf(&b, "- **%s** (%s): `%s`\n", step.StepID, step.Persona, step.WorkspacePath)
			}
		}
	}

	// Failure details
	hasFailures := false
	for _, step := range ctx.Steps {
		if step.ErrorMessage != "" {
			hasFailures = true
			break
		}
	}
	if hasFailures {
		b.WriteString("\n## Failures\n\n")
		for _, step := range ctx.Steps {
			if step.ErrorMessage != "" {
				fmt.Fprintf(&b, "### Step: %s\n\n```\n%s\n```\n\n", step.StepID, step.ErrorMessage)
			}
		}
	}

	// Wave infrastructure reference
	b.WriteString("\n## Wave Infrastructure\n\n")
	b.WriteString("Use these CLI commands — do NOT query the database directly.\n\n")
	b.WriteString("```bash\n")
	fmt.Fprintf(&b, "wave status %s          # run details + step states\n", ctx.Run.RunID)
	fmt.Fprintf(&b, "wave logs %s            # event log for this run\n", ctx.Run.RunID)
	fmt.Fprintf(&b, "wave artifacts %s       # list artifacts for this run\n", ctx.Run.RunID)
	b.WriteString("wave chat --list                    # recent runs\n")
	b.WriteString("```\n\n")
	b.WriteString("If you must query state directly:\n")
	b.WriteString("- Database: `.wave/state.db` (SQLite)\n")
	b.WriteString("- Tables: `pipeline_run`, `event_log`, `artifact`, `step_state`, `pipeline_state`\n")
	b.WriteString("- Run ID column: `run_id` (not `id`)\n")
	b.WriteString("- Step column: `step_id` (not `step`)\n")
	b.WriteString("- There is NO table called `steps` or `runs`\n\n")
	b.WriteString("Key paths:\n")
	b.WriteString("- State DB: `.wave/state.db`\n")
	b.WriteString("- Workspaces: `.wave/workspaces/<pipeline>/<step>/`\n")
	b.WriteString("- Artifacts: `.wave/artifacts/` and `.wave/output/`\n")
	b.WriteString("- Traces: `.wave/traces/`\n")

	// Instructions
	b.WriteString("\n## Instructions\n\n")
	b.WriteString("- Use `wave status` and `wave logs` instead of raw SQL\n")
	b.WriteString("- Read artifacts to understand pipeline outputs\n")
	b.WriteString("- Inspect step workspaces to see what each step produced\n")
	b.WriteString("- Compare artifacts across steps to trace data flow\n")
	b.WriteString("- Diagnose failures by reading error messages and workspace state\n")
	fmt.Fprintf(&b, "- The project root is at: `%s`\n", ctx.ProjectRoot)

	// Manipulate mode: write access instructions
	if mode == ChatModeManipulate {
		b.WriteString("\n## Write Access\n\n")
		b.WriteString("You have write access to this workspace. You can:\n")
		b.WriteString("- Edit files in step workspaces\n")
		b.WriteString("- Run commands (go test, make, git)\n")
		b.WriteString("- Create new files\n")
		b.WriteString("- Modify artifacts\n\n")
		b.WriteString("Use `/wave-status` to check pipeline state.\n")
		b.WriteString("Use `/wave-diff` to see workspace changes.\n")
	}

	return b.String()
}

// chatFormatDuration formats a duration for display.
func chatFormatDuration(d time.Duration) string {
	if d == 0 {
		return "-"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	if minutes >= 60 {
		hours := minutes / 60
		minutes = minutes % 60
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}

// chatFormatTokens formats a token count.
func chatFormatTokens(tokens int) string {
	if tokens == 0 {
		return "-"
	}
	if tokens < 1000 {
		return fmt.Sprintf("%d", tokens)
	}
	if tokens < 1000000 {
		return fmt.Sprintf("%dk", tokens/1000)
	}
	return fmt.Sprintf("%.1fM", float64(tokens)/1000000.0)
}

// chatFormatSize formats byte count.
func chatFormatSize(bytes int64) string {
	if bytes == 0 {
		return "-"
	}
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024.0)
	}
	return fmt.Sprintf("%.1f MB", float64(bytes)/(1024.0*1024.0))
}

// provisionSlashCommands creates .claude/commands/ with Wave-specific slash commands.
func provisionSlashCommands(wsDir string) error {
	commandsDir := filepath.Join(wsDir, ".claude", "commands")
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		return err
	}

	commands := map[string]string{
		"wave-status.md": waveStatusCommand,
		"wave-diff.md":   waveDiffCommand,
	}

	for name, content := range commands {
		path := filepath.Join(commandsDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write slash command %s: %w", name, err)
		}
	}
	return nil
}

const waveStatusCommand = `Show the current state of the pipeline run.

List each step with its status, duration, and key artifacts.
Highlight any failures or warnings.
`

const waveDiffCommand = `Show what has changed in the current step's workspace.

Run git diff or compare files against the original state.
Summarize the key changes made.
`
