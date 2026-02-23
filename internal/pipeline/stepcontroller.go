package pipeline

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/state"
)

// StepController provides step-level manipulation for wave chat sessions.
type StepController interface {
	ContinueStep(ctx context.Context, chatCtx *ChatContext, stepID string) error
	ExtendStep(ctx context.Context, chatCtx *ChatContext, stepID string, instructions string) error
	RevertStep(ctx context.Context, chatCtx *ChatContext, stepID string) (*RevertPreview, error)
	ConfirmRevert(ctx context.Context, chatCtx *ChatContext, stepID string) error
	RewriteStep(ctx context.Context, chatCtx *ChatContext, stepID string, newPrompt string) error
}

// RevertPreview shows what would be reverted.
type RevertPreview struct {
	StepID        string
	WorkspacePath string
	FilesAffected int
	WorkspaceType string // "worktree" or "directory"
	Artifacts     []string
}

// DefaultStepController implements StepController.
type DefaultStepController struct {
	store state.StateStore
	model string // default model for interactive sessions
}

// NewStepController creates a new step controller with the given state store and default model.
func NewStepController(store state.StateStore, model string) *DefaultStepController {
	return &DefaultStepController{store: store, model: model}
}

// ContinueStep opens an interactive Claude session in the step's existing workspace,
// allowing the user to continue where the step left off.
func (c *DefaultStepController) ContinueStep(ctx context.Context, chatCtx *ChatContext, stepID string) error {
	step, err := c.findStep(chatCtx, stepID)
	if err != nil {
		return err
	}
	if step.WorkspacePath == "" {
		return fmt.Errorf("step %q has no preserved workspace — cannot continue", stepID)
	}

	// Write continue instructions to CLAUDE.md in workspace
	instructions := buildContinueInstructions(chatCtx, step)
	claudeMdPath := filepath.Join(step.WorkspacePath, "CLAUDE.md")
	if err := os.WriteFile(claudeMdPath, []byte(instructions), 0644); err != nil {
		return fmt.Errorf("failed to write CLAUDE.md: %w", err)
	}

	// Launch interactive session with write permissions
	return adapter.LaunchInteractive(step.WorkspacePath, adapter.InteractiveOptions{
		Model:   c.model,
		AddDirs: []string{chatCtx.ProjectRoot},
	})
}

// ExtendStep opens an interactive Claude session with the step's workspace,
// appending additional user-provided instructions to the context.
func (c *DefaultStepController) ExtendStep(ctx context.Context, chatCtx *ChatContext, stepID string, instructions string) error {
	step, err := c.findStep(chatCtx, stepID)
	if err != nil {
		return err
	}
	if step.WorkspacePath == "" {
		return fmt.Errorf("step %q has no preserved workspace — cannot extend", stepID)
	}
	if instructions == "" {
		return fmt.Errorf("extend instructions cannot be empty")
	}

	// Write extend instructions to CLAUDE.md in workspace
	extendMd := buildExtendInstructions(chatCtx, step, instructions)
	claudeMdPath := filepath.Join(step.WorkspacePath, "CLAUDE.md")
	if err := os.WriteFile(claudeMdPath, []byte(extendMd), 0644); err != nil {
		return fmt.Errorf("failed to write CLAUDE.md: %w", err)
	}

	// Launch interactive session with write permissions
	return adapter.LaunchInteractive(step.WorkspacePath, adapter.InteractiveOptions{
		Model:   c.model,
		AddDirs: []string{chatCtx.ProjectRoot},
	})
}

// RevertStep builds a preview of what would be reverted for the given step.
// It does NOT actually delete anything — call ConfirmRevert to execute.
func (c *DefaultStepController) RevertStep(ctx context.Context, chatCtx *ChatContext, stepID string) (*RevertPreview, error) {
	step, err := c.findStep(chatCtx, stepID)
	if err != nil {
		return nil, err
	}
	if step.WorkspacePath == "" {
		return nil, fmt.Errorf("step %q has no preserved workspace — nothing to revert", stepID)
	}

	// Determine workspace type from pipeline definition
	pipelineStep := findPipelineStep(chatCtx.Pipeline, stepID)
	wsType := "directory"
	if pipelineStep != nil && pipelineStep.Workspace.Type == "worktree" {
		wsType = "worktree"
	}

	// Count files in the workspace
	fileCount, err := countFiles(step.WorkspacePath)
	if err != nil {
		return nil, fmt.Errorf("failed to count workspace files: %w", err)
	}

	// Collect artifact names
	var artifactNames []string
	for _, art := range step.Artifacts {
		artifactNames = append(artifactNames, art.Name)
	}

	return &RevertPreview{
		StepID:        stepID,
		WorkspacePath: step.WorkspacePath,
		FilesAffected: fileCount,
		WorkspaceType: wsType,
		Artifacts:     artifactNames,
	}, nil
}

// ConfirmRevert actually deletes the workspace and updates state for the step.
// For worktree workspaces, it uses git worktree remove. For directories, it uses os.RemoveAll.
func (c *DefaultStepController) ConfirmRevert(ctx context.Context, chatCtx *ChatContext, stepID string) error {
	step, err := c.findStep(chatCtx, stepID)
	if err != nil {
		return err
	}
	if step.WorkspacePath == "" {
		return fmt.Errorf("step %q has no preserved workspace — nothing to revert", stepID)
	}

	// Determine workspace type from pipeline definition
	pipelineStep := findPipelineStep(chatCtx.Pipeline, stepID)
	isWorktree := pipelineStep != nil && pipelineStep.Workspace.Type == "worktree"

	if isWorktree {
		// Use git worktree remove for worktree workspaces
		if err := removeWorktreeWorkspace(chatCtx.ProjectRoot, step.WorkspacePath); err != nil {
			return fmt.Errorf("failed to remove worktree workspace: %w", err)
		}
	} else {
		// Use os.RemoveAll for directory workspaces
		if err := os.RemoveAll(step.WorkspacePath); err != nil {
			return fmt.Errorf("failed to remove workspace directory: %w", err)
		}
	}

	// Update run status in state store to reflect the revert
	if err := c.store.LogEvent(
		chatCtx.Run.RunID,
		stepID,
		"reverted",
		step.Persona,
		fmt.Sprintf("step %q reverted via wave chat", stepID),
		0,
		0,
	); err != nil {
		return fmt.Errorf("failed to log revert event: %w", err)
	}

	return nil
}

// RewriteStep creates a fresh workspace for the step with a new prompt,
// then launches an interactive session for the user to execute the rewrite.
func (c *DefaultStepController) RewriteStep(ctx context.Context, chatCtx *ChatContext, stepID string, newPrompt string) error {
	step, err := c.findStep(chatCtx, stepID)
	if err != nil {
		return err
	}
	if newPrompt == "" {
		return fmt.Errorf("rewrite prompt cannot be empty")
	}

	// Determine the workspace path for the rewrite
	// Use the existing workspace path or construct one from conventions
	wsPath := step.WorkspacePath
	if wsPath == "" {
		wsPath = filepath.Join(chatCtx.ProjectRoot, ".wave", "workspaces",
			chatCtx.Run.PipelineName, stepID)
	}

	// Clean out the old workspace if it exists
	if _, statErr := os.Stat(wsPath); statErr == nil {
		if err := os.RemoveAll(wsPath); err != nil {
			return fmt.Errorf("failed to clean workspace for rewrite: %w", err)
		}
	}

	// Create fresh workspace directory
	if err := os.MkdirAll(wsPath, 0755); err != nil {
		return fmt.Errorf("failed to create rewrite workspace: %w", err)
	}

	// Inject upstream artifacts into the fresh workspace
	if err := injectUpstreamArtifacts(chatCtx, stepID, wsPath); err != nil {
		return fmt.Errorf("failed to inject upstream artifacts: %w", err)
	}

	// Write rewrite instructions to CLAUDE.md
	rewriteMd := buildRewriteInstructions(chatCtx, step, newPrompt)
	claudeMdPath := filepath.Join(wsPath, "CLAUDE.md")
	if err := os.WriteFile(claudeMdPath, []byte(rewriteMd), 0644); err != nil {
		return fmt.Errorf("failed to write CLAUDE.md: %w", err)
	}

	// Log the rewrite event
	if err := c.store.LogEvent(
		chatCtx.Run.RunID,
		stepID,
		"rewriting",
		step.Persona,
		fmt.Sprintf("step %q rewrite initiated via wave chat", stepID),
		0,
		0,
	); err != nil {
		return fmt.Errorf("failed to log rewrite event: %w", err)
	}

	// Launch interactive session with write permissions
	return adapter.LaunchInteractive(wsPath, adapter.InteractiveOptions{
		Model:   c.model,
		AddDirs: []string{chatCtx.ProjectRoot},
	})
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

// findStep locates a ChatStepContext by step ID within the chat context.
func (c *DefaultStepController) findStep(chatCtx *ChatContext, stepID string) (*ChatStepContext, error) {
	if chatCtx == nil {
		return nil, fmt.Errorf("chat context is nil")
	}
	for i := range chatCtx.Steps {
		if chatCtx.Steps[i].StepID == stepID {
			return &chatCtx.Steps[i], nil
		}
	}
	return nil, fmt.Errorf("step %q not found in pipeline run %s", stepID, chatCtx.Run.RunID)
}

// findPipelineStep locates a Step definition by ID within the Pipeline.
func findPipelineStep(p *Pipeline, stepID string) *Step {
	if p == nil {
		return nil
	}
	for i := range p.Steps {
		if p.Steps[i].ID == stepID {
			return &p.Steps[i]
		}
	}
	return nil
}

// countFiles recursively counts files (not directories) under the given directory.
func countFiles(dir string) (int, error) {
	count := 0
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			count++
		}
		return nil
	})
	return count, err
}

// removeWorktreeWorkspace removes a worktree workspace using git worktree remove.
// Falls back to os.RemoveAll if git worktree operations fail.
// Uses os/exec directly to avoid importing the worktree package.
func removeWorktreeWorkspace(projectRoot, worktreePath string) error {
	// Attempt git worktree remove --force first
	cmd := exec.Command("git", "-C", projectRoot, "worktree", "remove", "--force", worktreePath)
	if err := cmd.Run(); err != nil {
		// Fallback: remove the directory manually and prune stale references
		if removeErr := os.RemoveAll(worktreePath); removeErr != nil {
			return fmt.Errorf("git worktree remove failed (%w) and manual cleanup also failed: %v", err, removeErr)
		}
		pruneCmd := exec.Command("git", "-C", projectRoot, "worktree", "prune")
		_ = pruneCmd.Run()
	}
	return nil
}

// injectUpstreamArtifacts copies artifacts from upstream (dependency) steps
// into the fresh workspace. This gives the rewritten step access to the same
// inputs the original step had.
func injectUpstreamArtifacts(chatCtx *ChatContext, stepID string, wsPath string) error {
	pipelineStep := findPipelineStep(chatCtx.Pipeline, stepID)
	if pipelineStep == nil {
		return nil // No pipeline definition means no dependencies to inject
	}

	// Collect artifacts from dependency steps
	depStepIDs := make(map[string]bool)
	for _, dep := range pipelineStep.Dependencies {
		depStepIDs[dep] = true
	}

	for _, step := range chatCtx.Steps {
		if !depStepIDs[step.StepID] {
			continue
		}
		for _, art := range step.Artifacts {
			if art.Path == "" {
				continue
			}
			// Copy artifact to workspace
			srcPath := art.Path
			// If the artifact path is relative, resolve it against the project root
			if !filepath.IsAbs(srcPath) {
				srcPath = filepath.Join(chatCtx.ProjectRoot, srcPath)
			}

			// Read source artifact
			data, err := os.ReadFile(srcPath)
			if err != nil {
				// Skip missing artifacts — they may have been cleaned up
				continue
			}

			// Write to workspace under .wave/artifacts/<step>/<name>
			destDir := filepath.Join(wsPath, ".wave", "artifacts", step.StepID)
			if err := os.MkdirAll(destDir, 0755); err != nil {
				return fmt.Errorf("failed to create artifact directory: %w", err)
			}
			destPath := filepath.Join(destDir, art.Name)
			if err := os.WriteFile(destPath, data, 0644); err != nil {
				return fmt.Errorf("failed to write artifact %s: %w", art.Name, err)
			}
		}
	}

	return nil
}

// buildContinueInstructions generates CLAUDE.md content for a continue session.
func buildContinueInstructions(chatCtx *ChatContext, step *ChatStepContext) string {
	var b strings.Builder

	b.WriteString("# Wave Step: Continue\n\n")
	fmt.Fprintf(&b, "You are continuing work on step **%s** of pipeline **%s**.\n\n", step.StepID, chatCtx.Run.PipelineName)

	writeStepContext(&b, chatCtx, step)

	b.WriteString("## Instructions\n\n")
	b.WriteString("Continue the work that was started in this step.\n")
	b.WriteString("The workspace contains the state from the previous execution.\n")
	b.WriteString("Pick up where the step left off and complete any remaining work.\n\n")

	if step.ErrorMessage != "" {
		b.WriteString("## Previous Error\n\n")
		fmt.Fprintf(&b, "The step previously failed with:\n```\n%s\n```\n\n", step.ErrorMessage)
		b.WriteString("Please address this error as part of your continuation.\n\n")
	}

	writeProjectReference(&b, chatCtx)

	return b.String()
}

// buildExtendInstructions generates CLAUDE.md content for an extend session.
func buildExtendInstructions(chatCtx *ChatContext, step *ChatStepContext, instructions string) string {
	var b strings.Builder

	b.WriteString("# Wave Step: Extend\n\n")
	fmt.Fprintf(&b, "You are extending step **%s** of pipeline **%s** with additional work.\n\n", step.StepID, chatCtx.Run.PipelineName)

	writeStepContext(&b, chatCtx, step)

	b.WriteString("## Extension Instructions\n\n")
	b.WriteString("The following additional instructions have been provided:\n\n")
	fmt.Fprintf(&b, "%s\n\n", instructions)
	b.WriteString("Apply these instructions while preserving the existing work in the workspace.\n\n")

	writeProjectReference(&b, chatCtx)

	return b.String()
}

// buildRewriteInstructions generates CLAUDE.md content for a rewrite session.
func buildRewriteInstructions(chatCtx *ChatContext, step *ChatStepContext, newPrompt string) string {
	var b strings.Builder

	b.WriteString("# Wave Step: Rewrite\n\n")
	fmt.Fprintf(&b, "You are rewriting step **%s** of pipeline **%s** from scratch.\n\n", step.StepID, chatCtx.Run.PipelineName)

	writeStepContext(&b, chatCtx, step)

	b.WriteString("## New Prompt\n\n")
	b.WriteString("Disregard the original step prompt. Execute the following instead:\n\n")
	fmt.Fprintf(&b, "%s\n\n", newPrompt)

	b.WriteString("## Upstream Artifacts\n\n")
	b.WriteString("Artifacts from upstream steps have been injected into `.wave/artifacts/` in this workspace.\n")
	b.WriteString("Use them as inputs for your work.\n\n")

	writeProjectReference(&b, chatCtx)

	return b.String()
}

// writeStepContext writes common step context to the string builder.
func writeStepContext(b *strings.Builder, chatCtx *ChatContext, step *ChatStepContext) {
	b.WriteString("## Step Context\n\n")
	b.WriteString("| Field | Value |\n")
	b.WriteString("|-------|-------|\n")
	fmt.Fprintf(b, "| Step ID | `%s` |\n", step.StepID)
	fmt.Fprintf(b, "| Persona | %s |\n", step.Persona)
	fmt.Fprintf(b, "| Status | %s |\n", stepStateDisplay(step.State))
	fmt.Fprintf(b, "| Run ID | `%s` |\n", chatCtx.Run.RunID)

	if step.Duration > 0 {
		fmt.Fprintf(b, "| Duration | %s |\n", chatFormatDuration(step.Duration))
	}
	if step.TokensUsed > 0 {
		fmt.Fprintf(b, "| Tokens | %s |\n", chatFormatTokens(step.TokensUsed))
	}
	if step.WorkspacePath != "" {
		fmt.Fprintf(b, "| Workspace | `%s` |\n", step.WorkspacePath)
	}

	b.WriteString("\n")

	// List artifacts produced by this step
	if len(step.Artifacts) > 0 {
		b.WriteString("### Step Artifacts\n\n")
		for _, art := range step.Artifacts {
			fmt.Fprintf(b, "- **%s** (`%s`): %s\n", art.Name, art.Path, artTypeDisplay(art.Type))
		}
		b.WriteString("\n")
	}
}

// writeProjectReference writes the project root reference section.
func writeProjectReference(b *strings.Builder, chatCtx *ChatContext) {
	b.WriteString("## Project Reference\n\n")
	fmt.Fprintf(b, "The project root is at: `%s`\n", chatCtx.ProjectRoot)
	b.WriteString("You have read access to the full project via the `--add-dir` flag.\n")
}

// stepStateDisplay returns a display string for the step state.
func stepStateDisplay(s string) string {
	if s == "" {
		return "pending"
	}
	return s
}

// artTypeDisplay returns a display string for an artifact type.
func artTypeDisplay(t string) string {
	if t == "" {
		return "file"
	}
	return t
}
