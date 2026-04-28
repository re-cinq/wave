package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/workspace"
	"github.com/recinq/wave/internal/worktree"
)

func isWorktreeClean(workspacePath string) bool {
	// Find the worktree directory: it's typically a __wt_* subdirectory
	entries, err := os.ReadDir(workspacePath)
	if err != nil {
		return false
	}
	wtDir := ""
	for _, e := range entries {
		if e.IsDir() && len(e.Name()) > 5 && e.Name()[:5] == "__wt_" {
			wtDir = filepath.Join(workspacePath, e.Name())
			break
		}
	}
	if wtDir == "" {
		return false // not a worktree workspace or can't find it
	}

	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = wtDir
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(bytes.TrimSpace(out)) == 0
}

// resolveModel applies model precedence:
//
// When --model is a tier name (cheapest/balanced/strongest):
//
//	The effective tier is the CHEAPER of the CLI tier and the step/persona tier.
//	This means --model balanced + step model: cheapest → cheapest (step wins).
//	The CLI flag sets a ceiling, not a floor.
//
// When --model is a literal model name (e.g., "claude-sonnet-4"):
//
//	The literal model is used for all steps regardless of YAML tiers.
//
// When --force-model is set:
//
//	The CLI model overrides everything unconditionally.
//
// Otherwise: step model > persona model > auto-route > adapter tier_models > global routing > adapter default.

func (e *DefaultPipelineExecutor) createStepWorkspace(execution *PipelineExecution, step *Step) (string, error) {
	pipelineID := execution.Status.ID
	wsRoot := execution.Manifest.Runtime.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = ".agents/workspaces"
	}

	// Handle workspace ref — share another step's workspace
	if step.Workspace.Ref != "" {
		// Special "parent" ref: use the parent sub-pipeline step's workspace
		if step.Workspace.Ref == "parent" && e.parentWorkspacePath != "" {
			return e.parentWorkspacePath, nil
		}
		execution.mu.Lock()
		refPath, ok := execution.WorkspacePaths[step.Workspace.Ref]
		execution.mu.Unlock()
		if !ok {
			return "", fmt.Errorf("referenced workspace step %q has not been executed yet", step.Workspace.Ref)
		}
		return refPath, nil
	}

	// Handle worktree workspace type
	if step.Workspace.Type == "worktree" {
		// Resolve branch name from template variables.
		// Step output references ({{ steps.X.artifacts.Y.field }}) are resolved first
		// so that branch names can be derived from prior step outputs (e.g. PR head branch).
		branch := step.Workspace.Branch
		if branch != "" {
			resolved, err := e.resolveWorkspaceStepRefs(branch, execution)
			if err != nil {
				return "", fmt.Errorf("workspace branch template %q: %w", branch, err)
			}
			branch = resolved
		}
		if execution.Context != nil && branch != "" {
			branch = execution.Context.ResolvePlaceholders(branch)
		}

		// Resolve base ref from template variables (same two-pass resolution).
		base := step.Workspace.Base
		if base != "" {
			resolved, err := e.resolveWorkspaceStepRefs(base, execution)
			if err != nil {
				return "", fmt.Errorf("workspace base template %q: %w", base, err)
			}
			base = resolved
		}
		if execution.Context != nil && base != "" {
			base = execution.Context.ResolvePlaceholders(base)
		}
		// Stacked matrix execution: override base branch from parent tier
		if e.stackedBaseBranch != "" && base == "" {
			base = e.stackedBaseBranch
		}

		if branch == "" && base == "" {
			// Fall back to pipeline context branch or generate one
			branch = execution.Context.BranchName
			if branch == "" {
				branch = fmt.Sprintf("wave/%s/%s", pipelineID, step.ID)
			}
		}

		// Reuse existing worktree for the same branch
		execution.mu.Lock()
		info, ok := execution.WorktreePaths[branch]
		if ok {
			execution.WorkspacePaths[step.ID+"__worktree_repo_root"] = info.RepoRoot
		}
		execution.mu.Unlock()
		if ok {
			return info.AbsPath, nil
		}

		// Branch-keyed path for sharing across steps. Use the executor's
		// workspace run ID override so resume reuses the original run's
		// worktree dir instead of creating an empty one at the resume
		// timestamp; falls back to pipelineID for fresh runs.
		sanitized := SanitizeBranchName(branch)
		wtKey := "__wt_" + sanitized
		wsPath := filepath.Join(wsRoot, e.workspaceRunIDFor(pipelineID), wtKey)

		absPath, err := filepath.Abs(wsPath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve workspace path: %w", err)
		}

		mgr, err := worktree.NewManager("")
		if err != nil {
			return "", fmt.Errorf("failed to create worktree manager: %w", err)
		}

		if err := mgr.Create(absPath, branch, base); err != nil {
			return "", fmt.Errorf("failed to create worktree workspace: %w", err)
		}

		// Register for reuse and cleanup
		execution.mu.Lock()
		execution.WorktreePaths[branch] = &WorktreeInfo{AbsPath: absPath, RepoRoot: mgr.RepoRoot()}
		execution.WorkspacePaths[step.ID+"__worktree_repo_root"] = mgr.RepoRoot()
		execution.mu.Unlock()

		// Persist worktree branch name for TUI header display
		if e.store != nil {
			if branchErr := e.store.UpdateRunBranch(e.runID, branch); branchErr != nil {
				// Log warning but don't fail the step — branch display is non-critical
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      "warn",
					Message:    fmt.Sprintf("failed to persist branch name: %v", branchErr),
				})
			}
		}

		// Record branch creation as a deliverable for outcome tracking
		e.outcomeTracker.AddBranch(step.ID, branch, absPath, "Feature branch")

		// Mark CLAUDE.md as skip-worktree so prepareWorkspace() changes
		// don't get staged by git add -A in implement steps
		_ = exec.Command("git", "-C", absPath, "update-index", "--skip-worktree", "AGENTS.md").Run()

		// Run skill init commands inside the worktree (only on first creation)
		if execution.Pipeline.Requires != nil {
			for _, skillName := range execution.Pipeline.Requires.SkillNames() {
				cfg := execution.Pipeline.Requires.Skills[skillName]
				if cfg.Init == "" {
					continue
				}
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      "skill_init",
					Message:    fmt.Sprintf("running init for skill %q in worktree", skillName),
				})
				initCmd := exec.Command("sh", "-c", cfg.Init)
				initCmd.Dir = absPath
				if out, err := initCmd.CombinedOutput(); err != nil {
					return "", fmt.Errorf("skill %q init failed in worktree: %w\noutput: %s", skillName, err, string(out))
				}
			}
		}

		return absPath, nil
	}

	if e.wsManager != nil && len(step.Workspace.Mount) > 0 {
		// Update pipeline context with current step
		execution.Context.StepID = step.ID

		// Use pipeline context for template variables
		templateVars := execution.Context.ToTemplateVars()

		// Issue #1453 — resolve subset_from on each mount before passing
		// to the workspace manager. When set, the materialised subset
		// tree replaces the mount Source so the resulting workspace only
		// contains files listed in the named artifact's JSON path.
		mounts := step.Workspace.Mount
		resolvedMounts := make([]Mount, len(mounts))
		copy(resolvedMounts, mounts)
		for i := range resolvedMounts {
			if resolvedMounts[i].SubsetFrom == "" {
				continue
			}
			subsetSrc, err := e.materialiseMountSubset(execution, step.ID, i, resolvedMounts[i])
			if err != nil {
				return "", fmt.Errorf("step %q mount subset: %w", step.ID, err)
			}
			resolvedMounts[i].Source = subsetSrc
		}

		wsPath, err := e.wsManager.Create(workspace.WorkspaceConfig{
			Root:  wsRoot,
			Mount: toWorkspaceMounts(resolvedMounts),
		}, templateVars)
		if err != nil {
			return "", err
		}

		// Anchor Claude Code path resolution to the workspace root.
		// Without .git, Claude Code walks up the directory tree and resolves
		// relative paths against the project root instead of the workspace.
		_ = exec.Command("git", "init", "-q", wsPath).Run()
		return wsPath, nil
	}

	// Create directory under .agents/workspaces/<pipeline>/<step>/. Use the
	// executor's workspace run ID override so resume reads from the original
	// run's tree; falls back to pipelineID for fresh runs.
	wsPath := filepath.Join(wsRoot, e.workspaceRunIDFor(pipelineID), step.ID)
	if err := os.MkdirAll(wsPath, 0755); err != nil {
		return "", err
	}
	// Anchor Claude Code path resolution (see mount-based workspace above)
	_ = exec.Command("git", "init", "-q", wsPath).Run()
	return wsPath, nil
}

// materialiseMountSubset reads the artifact named in mount.SubsetFrom,
// extracts the path list at the dotted JSON path, and copies only
// those files (preserving directory structure) into a fresh temp dir
// rooted under .agents/workspaces/_subsets/. Returns the temp-dir path
// to be used as the mount source. Issue #1453.
//
// SubsetFrom format: "<step>.<artifact>.<json-path>".
// The first two segments name the artifact; the remainder is a JSON
// path navigated via ExtractJSONPath. The extracted value must be a
// JSON array of strings, each interpreted as a path relative to the
// original mount.Source.
func (e *DefaultPipelineExecutor) materialiseMountSubset(execution *PipelineExecution, ownerStepID string, mountIdx int, mount Mount) (string, error) {
	parts := strings.SplitN(mount.SubsetFrom, ".", 3)
	if len(parts) < 3 {
		return "", fmt.Errorf("subset_from %q: must be '<step>.<artifact>.<json-path>'", mount.SubsetFrom)
	}
	stepID, artifactName, jsonPath := parts[0], parts[1], parts[2]

	// Resolve artifact path via the same lookup tiers as the auto-injector.
	path, ok := e.locateDepArtifact(execution, stepID, artifactName)
	if !ok {
		// Soft fallback: when the artifact is unavailable (e.g. the
		// audit-* pipeline runs standalone without ops-pr-respond
		// providing pr-context), keep the original mount.Source. The
		// LLM-side scope guard from #1411 still narrows behaviour but
		// the workspace itself remains the full project tree. Issue
		// #1453 — this preserves backwards compatibility for ad-hoc
		// audit-* runs while letting ops-pr-respond enforce scope.
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: execution.Status.ID,
			StepID:     ownerStepID,
			State:      "step_progress",
			Message:    fmt.Sprintf("subset_from %q artifact not found — falling back to full mount.Source", mount.SubsetFrom),
		})
		return mount.Source, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read subset artifact: %w", err)
	}

	listJSON, err := ExtractJSONPath(data, "."+jsonPath)
	if err != nil {
		return "", fmt.Errorf("extract %q: %w", jsonPath, err)
	}

	var files []string
	if err := json.Unmarshal([]byte(listJSON), &files); err != nil {
		return "", fmt.Errorf("subset_from %q: expected array of strings, got %s", mount.SubsetFrom, string(listJSON))
	}

	// Resolve original source path for relative copy. EvalSymlinks
	// gives us the canonical path so the security check below catches
	// cases where the source itself is a symlink.
	source := mount.Source
	if !filepath.IsAbs(source) {
		if abs, err := filepath.Abs(source); err == nil {
			source = abs
		}
	}
	canonicalSource, err := filepath.EvalSymlinks(source)
	if err != nil {
		return "", fmt.Errorf("resolve source: %w", err)
	}

	// Materialise into a path unique to (run, ownerStep, mountIdx) so two
	// concurrent steps with the same SubsetFrom can't collide on the
	// RemoveAll/MkdirAll race.
	pipelineID := execution.Status.ID
	subsetRoot := filepath.Join(".agents", "workspaces", "_subsets", pipelineID, ownerStepID, fmt.Sprintf("mount%d", mountIdx))
	if err := os.RemoveAll(subsetRoot); err != nil {
		return "", fmt.Errorf("clean subset dir: %w", err)
	}
	if err := os.MkdirAll(subsetRoot, 0755); err != nil {
		return "", fmt.Errorf("create subset dir: %w", err)
	}

	for _, rel := range files {
		// Reject path traversal — entries must be repo-relative.
		clean := filepath.Clean(rel)
		if filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
			continue
		}
		srcFile := filepath.Join(source, clean)
		info, err := os.Lstat(srcFile)
		if err != nil {
			// Listed but missing — skip silently. PR file lists may
			// include deleted paths.
			continue
		}
		if info.IsDir() {
			continue
		}
		// Reject symlinks — even if the JSON path itself looks safe,
		// a symlink in the source tree could point outside it. Belt
		// and suspenders below.
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		// Belt and suspenders: resolve through any parent symlinks
		// and confirm the canonical path is still under source.
		canonicalSrc, err := filepath.EvalSymlinks(srcFile)
		if err != nil {
			continue
		}
		if !strings.HasPrefix(canonicalSrc, canonicalSource+string(filepath.Separator)) && canonicalSrc != canonicalSource {
			// Canonical path escaped source — drop.
			continue
		}
		dstFile := filepath.Join(subsetRoot, clean)
		if err := os.MkdirAll(filepath.Dir(dstFile), 0755); err != nil {
			return "", fmt.Errorf("mkdir subset parent: %w", err)
		}
		// Copy rather than symlink — readonly mode chmods the tree
		// later, which symlinks don't carry.
		if err := copySubsetFile(canonicalSrc, dstFile); err != nil {
			return "", fmt.Errorf("copy %q: %w", clean, err)
		}
	}

	return subsetRoot, nil
}

// copySubsetFile copies a single file (regular or symlink target)
// from src to dst, preserving mode bits.
func copySubsetFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func toWorkspaceMounts(mounts []Mount) []workspace.Mount {
	result := make([]workspace.Mount, len(mounts))
	for i, m := range mounts {
		result[i] = workspace.Mount{
			Source: m.Source,
			Target: m.Target,
			Mode:   m.Mode,
		}
	}
	return result
}

// resolveCommandWorkDir determines the working directory for a command step.
// When the step uses mount-based workspaces, the project files live under the
// mount target directory (e.g. workspacePath/project/) rather than the bare
// workspace root. This function finds the first mount whose source is "./"
// (the project root) and returns the corresponding target path inside the
// workspace. If no project-root mount is found, or the step has no mounts,
// the original workspace path is returned unchanged.
func resolveCommandWorkDir(workspacePath string, step *Step) string {
	// For mount-based workspaces, find the project root mount
	for _, m := range step.Workspace.Mount {
		if m.Source == "./" || m.Source == "." {
			target := strings.TrimPrefix(m.Target, "/")
			if target == "" {
				continue
			}
			candidate := filepath.Join(workspacePath, target)
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				return candidate
			}
		}
	}

	// For worktree workspaces, look for a __wt_ directory inside the workspace
	if entries, err := os.ReadDir(workspacePath); err == nil {
		for _, e := range entries {
			if e.IsDir() && strings.HasPrefix(e.Name(), "__wt_") {
				return filepath.Join(workspacePath, e.Name())
			}
		}
	}

	// If the workspace is bare (no source files) and looks empty,
	// fall back to the project root so commands like "go test ./..." find packages.
	// Check for common project markers to distinguish a real workspace from bare.
	projectMarkers := []string{"go.mod", "package.json", "Cargo.toml", "pyproject.toml", "Makefile"}
	hasMarker := false
	for _, marker := range projectMarkers {
		if _, err := os.Stat(filepath.Join(workspacePath, marker)); err == nil {
			hasMarker = true
			break
		}
	}

	// Auto-injected dep artifacts (#1452) populate .agents/artifacts and
	// .agents/output before the command runs. Treat their presence as a
	// "this workspace is real" signal so commands keep CWD here and find
	// the auto-injected files at relative paths.
	if !hasMarker {
		for _, d := range []string{".agents/artifacts", ".agents/output"} {
			if info, err := os.Stat(filepath.Join(workspacePath, d)); err == nil && info.IsDir() {
				hasMarker = true
				break
			}
		}
	}

	if !hasMarker {
		if cwd, err := os.Getwd(); err == nil {
			// Only fall back if CWD has a project marker
			for _, marker := range projectMarkers {
				if _, err := os.Stat(filepath.Join(cwd, marker)); err == nil {
					return cwd
				}
			}
		}
	}

	return workspacePath
}

func (e *DefaultPipelineExecutor) cleanupWorktrees(execution *PipelineExecution, pipelineID string) {
	cleaned := map[string]bool{}
	for key, repoRoot := range execution.WorkspacePaths {
		if !strings.HasSuffix(key, "__worktree_repo_root") {
			continue
		}
		stepID := strings.TrimSuffix(key, "__worktree_repo_root")
		wsPath := execution.WorkspacePaths[stepID]
		if wsPath == "" {
			continue
		}
		// Skip already-cleaned paths (shared worktrees used by multiple steps)
		if cleaned[wsPath] {
			continue
		}
		cleaned[wsPath] = true
		mgr, err := worktree.NewManager(repoRoot)
		if err != nil {
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     stepID,
				State:      "warning",
				Message:    fmt.Sprintf("worktree cleanup skipped: %v", err),
			})
			continue
		}
		if err := mgr.Remove(wsPath); err != nil {
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     stepID,
				State:      "warning",
				Message:    fmt.Sprintf("worktree cleanup failed: %v", err),
			})
		}
	}
}

// executeCompositionStep handles steps that reference sub-pipelines (via the
// `pipeline:` field) rather than executing a persona directly. It loads the
// referenced pipeline YAML, resolves the step's input template, and delegates
// execution to a fresh DefaultPipelineExecutor instance.
// can be retrieved from persistent storage via GetStatus.
