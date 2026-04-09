package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/state"
)

// ChatMode determines the permission level for a chat workspace.
type ChatMode string

const (
	ChatModeAnalysis   ChatMode = "analysis"   // Phase 1: read-only
	ChatModeManipulate ChatMode = "manipulate" // Phase 2: read-write
)

// ChatWorkspaceOptions configures the chat workspace preparation.
type ChatWorkspaceOptions struct {
	Model        string   // Model override (e.g., "sonnet", "opus")
	Mode         ChatMode // defaults to ChatModeAnalysis if empty
	StepFilter   string   // If set, scope the chat context to this step ID
	ArtifactName string   // If set, focus on a specific artifact by name
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
	claudeMd := buildChatClaudeMd(ctx, mode, opts.StepFilter, opts.ArtifactName)
	instructionFile := adapter.InstructionFilename("claude")
	mdPath := filepath.Join(wsDir, instructionFile)
	if err := os.WriteFile(mdPath, []byte(claudeMd), 0644); err != nil {
		return "", fmt.Errorf("failed to write %s: %w", instructionFile, err)
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
// stepFilter scopes the context to a single step when non-empty.
// artifactName focuses the context on a specific artifact when non-empty.
func buildChatClaudeMd(ctx *ChatContext, mode ChatMode, stepFilter, artifactName string) string {
	var b strings.Builder

	switch {
	case stepFilter != "":
		fmt.Fprintf(&b, "# Wave Step Analysis: %s\n\n", stepFilter)
		fmt.Fprintf(&b, "You are analyzing step **%s** from a Wave pipeline run.\n\n", stepFilter)
	case artifactName != "":
		fmt.Fprintf(&b, "# Wave Artifact Analysis: %s\n\n", artifactName)
		fmt.Fprintf(&b, "You are analyzing artifact **%s** from a Wave pipeline run.\n\n", artifactName)
	default:
		b.WriteString("# Wave Pipeline Analysis\n\n")
		b.WriteString("You are analyzing a completed Wave pipeline run.\n\n")
	}

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

	// Filter steps based on stepFilter
	steps := ctx.Steps
	if stepFilter != "" {
		var filtered []ChatStepContext
		for _, step := range ctx.Steps {
			if step.StepID == stepFilter {
				filtered = append(filtered, step)
			}
		}
		steps = filtered
	}

	// Step results table
	b.WriteString("\n## Step Results\n\n")
	b.WriteString("| # | Step | Persona | Status | Duration | Tokens |\n")
	b.WriteString("|---|------|---------|--------|----------|--------|\n")
	for i, step := range steps {
		state := step.State
		if state == "" {
			state = StatePending
		}
		fmt.Fprintf(&b, "| %d | %s | %s | %s | %s | %s |\n",
			i+1, step.StepID, step.Persona, state,
			chatFormatDuration(step.Duration), chatFormatTokens(step.TokensUsed))
	}

	// Filter artifacts based on stepFilter or artifactName
	artifacts := ctx.Artifacts
	if stepFilter != "" {
		var filtered []state.ArtifactRecord
		for _, art := range ctx.Artifacts {
			if art.StepID == stepFilter {
				filtered = append(filtered, art)
			}
		}
		artifacts = filtered
	}
	if artifactName != "" {
		var filtered []state.ArtifactRecord
		for _, art := range artifacts {
			if art.Name == artifactName {
				filtered = append(filtered, art)
			}
		}
		artifacts = filtered
	}

	// Artifacts inventory
	if len(artifacts) > 0 {
		b.WriteString("\n## Artifacts\n\n")
		b.WriteString("| Step | Name | Type | Path | Size |\n")
		b.WriteString("|------|------|------|------|------|\n")
		for _, art := range artifacts {
			artType := art.Type
			if artType == "" {
				artType = "-"
			}
			fmt.Fprintf(&b, "| %s | %s | %s | `%s` | %s |\n",
				art.StepID, art.Name, artType, art.Path, chatFormatSize(art.SizeBytes))
		}
	}

	// Inject focused artifact content when --artifact is specified
	if artifactName != "" {
		b.WriteString("\n## Focused Artifact Content\n\n")
		b.WriteString("The following artifact has been pre-loaded for immediate analysis:\n\n")
		for _, art := range artifacts {
			if art.Name == artifactName {
				fullPath := art.Path
				if !filepath.IsAbs(fullPath) {
					fullPath = filepath.Join(ctx.ProjectRoot, fullPath)
				}
				content, err := SummarizeArtifact(fullPath, 32000) // generous budget for focused artifact
				if err != nil {
					fmt.Fprintf(&b, "### %s\n\n*Could not read artifact: %v*\n\n", art.Name, err)
				} else {
					fence := "```"
					if strings.Contains(content, "```") {
						fence = "``````"
					}
					fmt.Fprintf(&b, "### %s\n\n%s\n%s\n%s\n\n", art.Name, fence, content, fence)
				}
			}
		}
	}

	// Step workspaces
	hasWorkspaces := false
	for _, step := range steps {
		if step.WorkspacePath != "" {
			hasWorkspaces = true
			break
		}
	}
	if hasWorkspaces {
		b.WriteString("\n## Step Workspaces\n\n")
		b.WriteString("Each step's workspace is preserved. You can read files from these locations:\n\n")
		for _, step := range steps {
			if step.WorkspacePath != "" {
				fmt.Fprintf(&b, "- **%s** (%s): `%s`\n", step.StepID, step.Persona, step.WorkspacePath)
			}
		}
	}

	// Failure details
	hasFailures := false
	for _, step := range steps {
		if step.ErrorMessage != "" {
			hasFailures = true
			break
		}
	}
	if hasFailures {
		b.WriteString("\n## Failures\n\n")
		for _, step := range steps {
			if step.ErrorMessage != "" {
				fmt.Fprintf(&b, "### Step: %s\n\n```\n%s\n```\n\n", step.StepID, step.ErrorMessage)
			}
		}
	}

	// Injected artifact content (from chat_context config)
	if len(ctx.ArtifactContents) > 0 {
		b.WriteString("\n## Key Artifact Content\n\n")
		b.WriteString("The following artifact content has been pre-loaded for immediate reference:\n\n")
		artifactNames := make([]string, 0, len(ctx.ArtifactContents))
		for name := range ctx.ArtifactContents {
			artifactNames = append(artifactNames, name)
		}
		sort.Strings(artifactNames)
		for _, name := range artifactNames {
			content := ctx.ArtifactContents[name]
			fence := "```"
			if strings.Contains(content, "```") {
				fence = "``````"
			}
			fmt.Fprintf(&b, "### %s\n\n%s\n%s\n%s\n\n", name, fence, content, fence)
		}
	}

	// Suggested questions (from pipeline chat_context config)
	if ctx.ChatConfig != nil && len(ctx.ChatConfig.SuggestedQuestions) > 0 {
		b.WriteString("\n## Suggested Questions\n\n")
		b.WriteString("The pipeline author suggests starting with these questions:\n\n")
		for _, q := range ctx.ChatConfig.SuggestedQuestions {
			fmt.Fprintf(&b, "- %s\n", q)
		}
		b.WriteString("\n")
	}

	// Focus areas (from pipeline chat_context config)
	if ctx.ChatConfig != nil && len(ctx.ChatConfig.FocusAreas) > 0 {
		b.WriteString("\n## Focus Areas\n\n")
		b.WriteString("Pay special attention to:\n\n")
		for _, area := range ctx.ChatConfig.FocusAreas {
			fmt.Fprintf(&b, "- %s\n", area)
		}
		b.WriteString("\n")
	}

	// Post-mortem questions (auto-generated based on run context)
	postMortems := generatePostMortemQuestions(ctx)
	if len(postMortems) > 0 {
		b.WriteString("\n## Post-Mortem Questions\n\n")
		b.WriteString("Based on this pipeline run, consider asking the user these questions:\n\n")
		for _, q := range postMortems {
			fmt.Fprintf(&b, "- %s\n", q)
		}
		b.WriteString("\n")
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
		minutes %= 60
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

	cmdNames := make([]string, 0, len(commands))
	for name := range commands {
		cmdNames = append(cmdNames, name)
	}
	sort.Strings(cmdNames)
	for _, name := range cmdNames {
		path := filepath.Join(commandsDir, name)
		if err := os.WriteFile(path, []byte(commands[name]), 0644); err != nil {
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

// generatePostMortemQuestions creates 3 context-aware questions based on the pipeline run.
func generatePostMortemQuestions(ctx *ChatContext) []string {
	var questions []string

	// Find failed steps
	var failedSteps []string
	for _, step := range ctx.Steps {
		if step.State == StateFailed {
			failedSteps = append(failedSteps, step.StepID)
		}
	}

	// Check for specific artifact types
	hasPR := false
	hasReview := false
	hasAnalysis := false
	for _, art := range ctx.Artifacts {
		switch {
		case strings.Contains(art.Name, "pr-result") || strings.Contains(art.Name, "pr_result"):
			hasPR = true
		case strings.Contains(art.Name, "review") || strings.Contains(art.Name, "verdict"):
			hasReview = true
		case strings.Contains(art.Name, "analysis") || strings.Contains(art.Name, "assessment"):
			hasAnalysis = true
		}
	}

	// Determine pipeline category from name
	pipelineName := ""
	if ctx.Pipeline != nil {
		pipelineName = ctx.Pipeline.PipelineName()
	}

	switch {
	case ctx.Run.Status == StateFailed:
		// Failed run questions
		if len(failedSteps) > 0 {
			questions = append(questions, fmt.Sprintf("What caused the failure in step '%s' and how can it be resolved?", failedSteps[0]))
		} else {
			questions = append(questions, "What caused the pipeline failure and how can it be resolved?")
		}
		questions = append(questions, "Should we retry the pipeline with modified parameters or a different approach?")
		questions = append(questions, "Are there any pre-conditions or dependencies that were not met?")
	case hasPR:
		// Implementation/PR pipeline
		questions = append(questions, "Would you like to review the changes in the pull request?")
		questions = append(questions, "Are there any edge cases or error scenarios not covered by the implementation?")
		questions = append(questions, "Should we add additional tests or documentation?")
	case hasReview:
		// Review pipeline
		questions = append(questions, "What are the most critical findings from the review?")
		questions = append(questions, "Are there any blocking concerns that must be addressed before merging?")
		questions = append(questions, "What improvements should be prioritized?")
	case hasAnalysis:
		// Analysis pipeline
		questions = append(questions, "What are the key findings from the analysis?")
		questions = append(questions, "Which items need immediate attention?")
		questions = append(questions, "What is the recommended next step based on the analysis?")
	case strings.Contains(pipelineName, "implement"):
		questions = append(questions, "Would you like to review the implementation changes?")
		questions = append(questions, "Are there any areas that need additional testing?")
		questions = append(questions, "Should we refine any part of the implementation?")
	case strings.Contains(pipelineName, "review"):
		questions = append(questions, "What are the most important review findings?")
		questions = append(questions, "Are there any security or quality concerns?")
		questions = append(questions, "What should be addressed before the next review?")
	default:
		// Generic completed pipeline
		questions = append(questions, "What are the key outputs from this pipeline run?")
		questions = append(questions, "Are there any issues or areas of concern in the results?")
		questions = append(questions, "What should be the next step based on these results?")
	}

	return questions
}
