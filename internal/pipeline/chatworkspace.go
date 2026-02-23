package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ChatWorkspaceOptions configures the chat workspace preparation.
type ChatWorkspaceOptions struct {
	Model string // Model override (e.g., "sonnet", "opus")
}

// PrepareChatWorkspace creates a workspace directory with CLAUDE.md and settings.json
// for an interactive wave chat session. Returns the workspace path.
func PrepareChatWorkspace(ctx *ChatContext, opts ChatWorkspaceOptions) (string, error) {
	// 1. Create chat workspace directory
	wsDir := filepath.Join(ctx.ProjectRoot, ".wave", "chat", ctx.Run.RunID)
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create chat workspace: %w", err)
	}

	// 2. Build and write CLAUDE.md
	claudeMd := buildChatClaudeMd(ctx)
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

	settings := chatSettings{
		Model: model,
		Permissions: chatPermissions{
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
		},
	}

	settingsData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal settings: %w", err)
	}

	settingsPath := filepath.Join(settingsDir, "settings.json")
	if err := os.WriteFile(settingsPath, settingsData, 0644); err != nil {
		return "", fmt.Errorf("failed to write settings.json: %w", err)
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
func buildChatClaudeMd(ctx *ChatContext) string {
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

	// Instructions
	b.WriteString("\n## Instructions\n\n")
	b.WriteString("- Read artifacts to understand pipeline outputs\n")
	b.WriteString("- Inspect step workspaces to see what each step produced\n")
	b.WriteString("- Compare artifacts across steps to trace data flow\n")
	b.WriteString("- Diagnose failures by reading error messages and workspace state\n")
	fmt.Fprintf(&b, "- The project root is at: `%s`\n", ctx.ProjectRoot)

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
