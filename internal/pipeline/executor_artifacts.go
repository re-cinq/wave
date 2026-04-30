package pipeline

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/recinq/wave/internal/audit"
	"github.com/recinq/wave/internal/contract"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/security"
	"github.com/recinq/wave/internal/skill"
)

func (e *DefaultPipelineExecutor) buildStepPrompt(execution *PipelineExecution, step *Step) string {
	// Handle slash_command exec type
	if step.Exec.Type == "slash_command" && step.Exec.Command != "" {
		args := step.Exec.Args
		if execution.Context != nil {
			args = execution.Context.ResolvePlaceholders(args)
		}
		// Replace {{ input }} in args
		if execution.Input != "" {
			for _, pattern := range []string{"{{ input }}", "{{input}}", "{{ input}}", "{{input }}"} {
				args = strings.ReplaceAll(args, pattern, execution.Input)
			}
		}
		return skill.FormatSkillCommandPrompt(step.Exec.Command, args)
	}

	prompt := step.Exec.Source

	// Resolve source_path through template variables (e.g., {{ forge.prefix }})
	sourcePath := step.Exec.SourcePath
	if execution.Context != nil && sourcePath != "" {
		sourcePath = execution.Context.ResolvePlaceholders(sourcePath)
	}

	// Load prompt from external file if source_path is set
	if sourcePath != "" {
		e.trace(audit.TracePromptLoad, step.ID, 0, map[string]string{
			"source_path": sourcePath,
		})
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			e.trace(audit.TracePromptLoadError, step.ID, 0, map[string]string{
				"source_path": sourcePath,
				"error":       err.Error(),
			})
		} else {
			prompt = string(data)
			e.trace(audit.TracePromptLoad, step.ID, 0, map[string]string{
				"source_path": sourcePath,
				"size":        fmt.Sprintf("%d", len(prompt)),
			})
		}
	} else if e.debug && step.Exec.Source == "" {
		e.trace(audit.TracePromptLoadError, step.ID, 0, map[string]string{
			"error": "step has neither source nor source_path set",
		})
	}

	// Determine the input value to use (sanitized if provided, empty string if not)
	var sanitizedInput string
	if execution.Input != "" {
		// SECURITY FIX: Sanitize user input for prompt injection
		sanitizationRecord, tmpInput, sanitizeErr := e.sec.inputSanitizer.SanitizeInput(execution.Input, "task_description")
		if sanitizeErr != nil {
			// Security violation detected - log and reject
			e.sec.securityLogger.LogViolation(
				string(security.ViolationPromptInjection),
				string(security.SourceUserInput),
				fmt.Sprintf("User input sanitization failed for step %s", step.ID),
				security.SeverityCritical,
				true,
			)
			// In strict mode, this would cause the step to fail
			// For now, we'll use empty input to prevent the injection
			sanitizedInput = "[INPUT SANITIZED FOR SECURITY]"
		} else {
			// Log sanitization details
			if sanitizationRecord.ChangesDetected {
				e.sec.securityLogger.LogViolation(
					string(security.ViolationPromptInjection),
					string(security.SourceUserInput),
					fmt.Sprintf("User input sanitized for step %s (risk score: %d)", step.ID, sanitizationRecord.RiskScore),
					security.SeverityMedium,
					false,
				)
			}
			sanitizedInput = tmpInput
		}
	} else {
		// No input provided - use empty string
		sanitizedInput = ""
	}

	// Replace template variables with sanitized input (even if empty)
	for _, pattern := range []string{"{{ input }}", "{{input}}", "{{ input}}", "{{input }}"} {
		for idx := strings.Index(prompt, pattern); idx != -1; idx = strings.Index(prompt, pattern) {
			prompt = prompt[:idx] + sanitizedInput + prompt[idx+len(pattern):]
		}
	}

	// NOTE: Schema injection for json_schema contracts is handled exclusively by
	// buildContractPrompt → appended to user prompt (-p argument). Do NOT duplicate it here.
	// See: buildContractPrompt() which uses the correct output path from OutputArtifacts.

	// Resolve remaining template variables using pipeline context
	if execution.Context != nil {
		prompt = execution.Context.ResolvePlaceholders(prompt)
	}

	// Inject retry failure context when adapt_prompt is enabled
	execution.mu.Lock()
	attemptCtx := execution.AttemptContexts[step.ID]
	execution.mu.Unlock()

	if attemptCtx != nil {
		var sb strings.Builder
		if attemptCtx.FailedStepID != "" {
			// Rework context — this step is a rework target for a failed step
			sb.WriteString("## REWORK CONTEXT\n\n")
			fmt.Fprintf(&sb, "You are executing as a rework step for failed step %q.\n", attemptCtx.FailedStepID)
			fmt.Fprintf(&sb, "The original step failed after %d attempt(s) (ran for %s).\n\n", attemptCtx.Attempt, attemptCtx.StepDuration.Round(time.Second))
		} else {
			sb.WriteString("## RETRY CONTEXT\n\n")
			fmt.Fprintf(&sb, "This is attempt %d of %d. The previous attempt failed.\n\n", attemptCtx.Attempt, attemptCtx.MaxAttempts)
		}
		if attemptCtx.PriorError != "" {
			sb.WriteString("### Previous Error\n```\n")
			sb.WriteString(attemptCtx.PriorError)
			sb.WriteString("\n```\n\n")
		}
		if len(attemptCtx.ContractErrors) > 0 {
			sb.WriteString("### Contract Validation Errors\n")
			for _, ce := range attemptCtx.ContractErrors {
				sb.WriteString(fmt.Sprintf("- %s\n", ce))
			}
			sb.WriteString("\n")
		}
		if attemptCtx.PriorStdout != "" {
			sb.WriteString(fmt.Sprintf("### Previous Output (last %d chars)\n```\n", maxStdoutTailChars))
			sb.WriteString(attemptCtx.PriorStdout)
			sb.WriteString("\n```\n\n")
		}
		if len(attemptCtx.PartialArtifacts) > 0 {
			sb.WriteString("### Partial Artifacts from Failed Step\n")
			for name, path := range attemptCtx.PartialArtifacts {
				sb.WriteString(fmt.Sprintf("- %s: %s\n", name, path))
			}
			sb.WriteString("\n")
		}
		if attemptCtx.ReviewFeedbackPath != "" {
			sb.WriteString("### Agent Review Feedback\n\n")
			sb.WriteString(fmt.Sprintf("A review agent found issues with the previous implementation. Structured feedback is available at: `%s`\n", attemptCtx.ReviewFeedbackPath))
			sb.WriteString("Read this file to understand the specific issues and suggestions before making changes.\n\n")
		}
		if len(attemptCtx.ContractErrors) > 0 {
			sb.WriteString("Fix the specific failure above. Do not start from scratch.\n\n---\n\n")
		} else {
			sb.WriteString("Please address the issues from the previous attempt and try a different approach if needed.\n\n---\n\n")
		}
		sb.WriteString(prompt)
		prompt = sb.String()
	}

	// Inject thread conversation context when the step is part of a thread group
	if step.Thread != "" && execution.ThreadManager != nil {
		fidelity := step.EffectiveFidelity()
		transcript := execution.ThreadManager.GetTranscript(context.Background(), step.Thread, fidelity)
		if transcript != "" {
			var sb strings.Builder
			sb.WriteString("## THREAD CONTEXT\n\n")
			sb.WriteString("The following is conversation history from prior steps in this thread group.\n\n")
			sb.WriteString(transcript)
			sb.WriteString("\n---\n\n")
			sb.WriteString(prompt)
			prompt = sb.String()

			e.trace(audit.TraceThreadInject, step.ID, int64(len(transcript)), map[string]string{
				"thread":   step.Thread,
				"fidelity": fidelity,
				"size":     fmt.Sprintf("%d", len(transcript)),
			})
		}
	}

	// Inject input artifact paths so the persona knows where to read upstream files.
	// Paths mirror injectArtifacts() destination logic: filepath.Join(workspace, ".agents/artifacts", as|artifact).
	if len(step.Memory.InjectArtifacts) > 0 {
		var sb strings.Builder
		sb.WriteString("\n## Input Artifacts\n\n")
		sb.WriteString("Upstream artifacts have been placed in your workspace at these paths:\n\n")
		for _, ref := range step.Memory.InjectArtifacts {
			name := ref.As
			if name == "" {
				name = ref.Artifact
			}
			sb.WriteString(fmt.Sprintf("- `.agents/artifacts/%s` (from step `%s`, artifact `%s`)\n", name, ref.Step, ref.Artifact))
		}
		sb.WriteString("\nRead these files at the paths shown. They are guaranteed to exist before this step runs.\n\n")
		sb.WriteString(prompt)
		prompt = sb.String()
	}

	// Inject output artifact paths so the persona knows where to write artifacts
	if len(step.OutputArtifacts) > 0 {
		var sb strings.Builder
		sb.WriteString("\n## Output Artifacts\n\n")
		sb.WriteString("Write the requested artifacts to these paths (in workspace root):\n\n")
		for _, art := range step.OutputArtifacts {
			// Use Path if specified, otherwise just the Name
			artPath := art.Path
			if artPath == "" {
				artPath = art.Name
			}
			sb.WriteString(fmt.Sprintf("- `%s` (as: %s)\n", artPath, art.Name))
		}
		sb.WriteString("\nThe pipeline will validate these artifacts. Write to the exact paths above.\n\n")
		sb.WriteString(prompt)
		prompt = sb.String()
	}

	return prompt
}

func (e *DefaultPipelineExecutor) injectArtifacts(execution *PipelineExecution, step *Step, workspacePath string) error {
	if len(step.Memory.InjectArtifacts) == 0 {
		return nil
	}

	// Always inject into the workspace (agent's working directory) so the
	// agent can find artifacts at relative paths like ".agents/artifacts/<name>".
	// Do NOT redirect to the sidecar — the agent runs in workspacePath.
	artifactsDir := filepath.Join(workspacePath, ".agents", "artifacts")
	if err := os.MkdirAll(artifactsDir, 0755); err != nil {
		return fmt.Errorf("failed to create artifacts dir: %w", err)
	}

	pipelineID := execution.Status.ID

	// Build artifact type map for validation
	artifactTypes := e.buildArtifactTypeMap(execution)

	for _, ref := range step.Memory.InjectArtifacts {
		artName := ref.As
		if artName == "" {
			artName = ref.Artifact
		}
		destPath := filepath.Join(artifactsDir, artName)

		// Cross-pipeline artifact reference: look up from prior pipeline outputs
		if ref.Pipeline != "" && e.crossPipelineArtifacts != nil {
			pipelineArtifacts, hasPipeline := e.crossPipelineArtifacts[ref.Pipeline]
			if !hasPipeline || pipelineArtifacts == nil {
				if ref.Optional {
					e.emit(event.Event{
						Timestamp:  time.Now(),
						PipelineID: pipelineID,
						StepID:     step.ID,
						State:      "step_progress",
						Message:    fmt.Sprintf("optional cross-pipeline artifact '%s' from pipeline '%s' not found, skipping", ref.Artifact, ref.Pipeline),
					})
					continue
				}
				return fmt.Errorf("cross-pipeline artifact '%s' from pipeline '%s' not found", ref.Artifact, ref.Pipeline)
			}
			data, hasArtifact := pipelineArtifacts[ref.Artifact]
			if !hasArtifact {
				if ref.Optional {
					e.emit(event.Event{
						Timestamp:  time.Now(),
						PipelineID: pipelineID,
						StepID:     step.ID,
						State:      "step_progress",
						Message:    fmt.Sprintf("optional cross-pipeline artifact '%s' from pipeline '%s' not found, skipping", ref.Artifact, ref.Pipeline),
					})
					continue
				}
				return fmt.Errorf("cross-pipeline artifact '%s' not found in pipeline '%s' outputs", ref.Artifact, ref.Pipeline)
			}
			if err := os.WriteFile(destPath, data, 0644); err != nil {
				return fmt.Errorf("failed to write artifact '%s': %w", artName, err)
			}
			execution.Context.SetArtifactPath(artName, destPath)
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      "step_progress",
				Message:    fmt.Sprintf("injected cross-pipeline artifact %s from pipeline %s", artName, ref.Pipeline),
			})

			// Type validation (if specified)
			if ref.Type != "" {
				key := ref.Pipeline + ":" + ref.Artifact
				declaredType := artifactTypes[key]
				if declaredType != "" && declaredType != ref.Type {
					return fmt.Errorf("artifact '%s' type mismatch: expected %s, got %s", ref.Artifact, ref.Type, declaredType)
				}
			}

			// Schema validation for input artifacts (if schema_path is specified)
			if ref.SchemaPath != "" {
				schemaContent, err := e.sec.loadSchemaContent(step, ref.SchemaPath)
				if err != nil {
					return fmt.Errorf("input artifact '%s': %w", artName, err)
				}
				if schemaContent == "" {
					return fmt.Errorf("input artifact '%s': schema %s produced no content", artName, ref.SchemaPath)
				}
				if err := contract.ValidateInputArtifactContent(artName, schemaContent, destPath); err != nil {
					return fmt.Errorf("input artifact '%s' schema validation failed: %w", artName, err)
				}
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      "step_progress",
					Message:    fmt.Sprintf("validated artifact %s against schema %s", artName, ref.SchemaPath),
				})
			}
			continue
		}

		// Try registered artifact path first
		key := ref.Step + ":" + ref.Artifact
		execution.mu.Lock()
		artifactPath, ok := execution.ArtifactPaths[key]
		execution.mu.Unlock()

		// Existence validation
		if !ok {
			// Try fallback: check if we have stdout results from the step
			execution.mu.Lock()
			result, exists := execution.Results[ref.Step]
			execution.mu.Unlock()
			if exists {
				if stdout, ok := result["stdout"].(string); ok {
					// Type validation (if specified)
					if ref.Type != "" {
						declaredType := artifactTypes[key]
						if declaredType != "" && declaredType != ref.Type {
							return fmt.Errorf("artifact '%s' type mismatch: expected %s, got %s", ref.Artifact, ref.Type, declaredType)
						}
					}
					if err := os.WriteFile(destPath, []byte(stdout), 0644); err != nil {
						return fmt.Errorf("failed to write artifact '%s': %w", artName, err)
					}
					// Register artifact path in context for template resolution
					execution.Context.SetArtifactPath(artName, destPath)
					e.emit(event.Event{
						Timestamp:  time.Now(),
						PipelineID: pipelineID,
						StepID:     step.ID,
						State:      "step_progress",
						Message:    fmt.Sprintf("injected artifact %s from step %s stdout", artName, ref.Step),
					})
					continue
				}
			}

			// Artifact not found - check if optional
			if ref.Optional {
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      "step_progress",
					Message:    fmt.Sprintf("optional artifact '%s' from step '%s' not found, skipping", ref.Artifact, ref.Step),
				})
				continue
			}
			return fmt.Errorf("required artifact '%s' from step '%s' not found", ref.Artifact, ref.Step)
		}

		// Type validation (if specified)
		if ref.Type != "" {
			declaredType := artifactTypes[key]
			if declaredType != "" && declaredType != ref.Type {
				return fmt.Errorf("artifact '%s' type mismatch: expected %s, got %s", ref.Artifact, ref.Type, declaredType)
			}
		}

		srcData, err := os.ReadFile(artifactPath)
		if err != nil {
			if ref.Optional {
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      "step_progress",
					Message:    fmt.Sprintf("optional artifact '%s' could not be read, skipping: %v", ref.Artifact, err),
				})
				continue
			}
			return fmt.Errorf("failed to read required artifact '%s': %w", ref.Artifact, err)
		}

		if err := os.WriteFile(destPath, srcData, 0644); err != nil {
			return fmt.Errorf("failed to write artifact '%s': %w", artName, err)
		}
		// Register artifact path in context for template resolution
		execution.Context.SetArtifactPath(artName, destPath)
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "step_progress",
			Message:    fmt.Sprintf("injected artifact %s from %s (%s)", artName, ref.Step, artifactPath),
		})

		// Schema validation for input artifacts (if schema_path is specified)
		if ref.SchemaPath != "" {
			schemaContent, err := e.sec.loadSchemaContent(step, ref.SchemaPath)
			if err != nil {
				return fmt.Errorf("input artifact '%s': %w", artName, err)
			}
			if schemaContent == "" {
				return fmt.Errorf("input artifact '%s': schema %s produced no content", artName, ref.SchemaPath)
			}
			if err := contract.ValidateInputArtifactContent(artName, schemaContent, destPath); err != nil {
				return fmt.Errorf("input artifact '%s' schema validation failed: %w", artName, err)
			}
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      "step_progress",
				Message:    fmt.Sprintf("validated artifact %s against schema %s", artName, ref.SchemaPath),
			})
		}
	}

	return nil
}

// buildArtifactTypeMap builds a map of artifact keys to their declared types
func (e *DefaultPipelineExecutor) buildArtifactTypeMap(execution *PipelineExecution) map[string]string {
	types := make(map[string]string)
	for _, step := range execution.Pipeline.Steps {
		for _, art := range step.OutputArtifacts {
			key := step.ID + ":" + art.Name
			types[key] = art.Type
		}
	}
	return types
}

func (e *DefaultPipelineExecutor) writeOutputArtifacts(execution *PipelineExecution, step *Step, workspacePath string, stdout []byte) {
	// Get artifact directory for stdout artifacts
	artifactDir := execution.Manifest.Runtime.Artifacts.GetDefaultArtifactDir()

	for _, art := range step.OutputArtifacts {
		key := step.ID + ":" + art.Name
		var artPath string

		// Handle stdout artifacts differently
		if art.IsStdoutArtifact() {
			// Stdout artifacts go to .agents/artifacts/<step-id>/<name>
			artPath = filepath.Join(workspacePath, artifactDir, step.ID, art.Name)
			_ = os.MkdirAll(filepath.Dir(artPath), 0755)

			// Write stdout content to artifact
			if err := os.WriteFile(artPath, stdout, 0644); err != nil {
				e.trace(audit.TraceArtifactWrite, step.ID, 0, map[string]string{
					"artifact": art.Name,
					"path":     artPath,
					"error":    err.Error(),
				})
			}
			execution.mu.Lock()
			execution.ArtifactPaths[key] = artPath
			execution.mu.Unlock()

			e.trace(audit.TraceArtifactWrite, step.ID, 0, map[string]string{
				"artifact": art.Name,
				"path":     artPath,
				"size":     fmt.Sprintf("%d", len(stdout)),
			})
		} else {
			// File-based artifacts: resolve path using pipeline context
			resolvedPath := execution.Context.ResolveArtifactPath(art)
			artPath = filepath.Join(workspacePath, resolvedPath)

			// If the persona already wrote the file, trust it and don't overwrite
			if _, err := os.Stat(artPath); err == nil {
				execution.mu.Lock()
				execution.ArtifactPaths[key] = artPath
				execution.mu.Unlock()
				e.trace(audit.TraceArtifactPreserved, step.ID, 0, map[string]string{
					"artifact": art.Name,
					"path":     artPath,
				})
			} else if len(stdout) > 0 {
				// Fall back to writing ResultContent (skip when nil/empty
				// to avoid creating zero-byte files from empty adapter output)
				_ = os.MkdirAll(filepath.Dir(artPath), 0755)
				_ = os.WriteFile(artPath, stdout, 0644)
				execution.mu.Lock()
				execution.ArtifactPaths[key] = artPath
				execution.mu.Unlock()
			}
		}

		// Archive artifact to a step-specific path so shared-worktree steps
		// don't all point at the same file in the DB. The injection system
		// keeps using artPath (the workspace-relative location), but the DB
		// gets the archived copy which survives subsequent steps overwriting
		// the same relative path.
		registeredPath := artPath
		if !art.IsStdoutArtifact() {
			archiveDir := filepath.Join(workspacePath, ".agents", "artifacts", step.ID)
			archiveName := art.Name
			if art.Type == "json" && !strings.HasSuffix(archiveName, ".json") {
				archiveName += ".json"
			}
			archivePath := filepath.Join(archiveDir, archiveName)
			if data, readErr := os.ReadFile(artPath); readErr == nil {
				if mkErr := os.MkdirAll(archiveDir, 0755); mkErr == nil {
					if writeErr := os.WriteFile(archivePath, data, 0644); writeErr == nil {
						registeredPath = archivePath
					}
				}
			}
		}

		// Register artifact in DB for web dashboard visibility
		if e.store != nil {
			var size int64
			if info, err := os.Stat(registeredPath); err == nil {
				size = info.Size()
			}
			_ = e.store.RegisterArtifact(execution.Status.ID, step.ID, art.Name, registeredPath, art.Type, size)
		}
	}

	e.warnOnUnexpectedArtifacts(execution, step, workspacePath)
}

// warnOnUnexpectedArtifacts walks the workspace at end-of-step and emits a
// warning for any persona-created file outside the declared OutputArtifacts
// paths. This catches model drift like GLM hallucinating
// `specs/999-<branch>/<file>` subdirs because the project mount happened to
// contain a `specs/` tree — the file is harmless but the divergence is a
// signal that the prompt could be tightened or the artifact path moved.
//
// We deliberately skip:
//   - the .agents/ tree (Wave-managed: artifacts, traces, output, AGENTS.md)
//   - the project/ tree (read-only mount of the source repo)
//   - the .git/ tree (worktree metadata when git ops occurred)
//   - declared OutputArtifacts paths and their archive copies
//   - hidden dotfiles at the root (e.g. AGENTS.md is plain but workspaces
//     accumulate small bookkeeping that adapters write themselves)
//
// The check is best-effort — Walk errors are swallowed because a noisy
// warning path must not become a new failure mode.
func (e *DefaultPipelineExecutor) warnOnUnexpectedArtifacts(execution *PipelineExecution, step *Step, workspacePath string) {
	if workspacePath == "" {
		return
	}
	declared := make(map[string]bool, len(step.OutputArtifacts))
	for _, art := range step.OutputArtifacts {
		if art.IsStdoutArtifact() {
			continue
		}
		if execution.Context != nil {
			declared[filepath.Clean(execution.Context.ResolveArtifactPath(art))] = true
		}
		declared[filepath.Clean(art.Path)] = true
	}

	// Workspaces of type `worktree` contain a full checkout of the project,
	// so a naive WalkDir flags every existing file as "unexpected". Use
	// `git status --porcelain` to enumerate only files the step actually
	// created or modified relative to the worktree HEAD. Falls back to a
	// pruned WalkDir for non-git workspaces (mount/basic).
	unexpected := changedFilesViaGit(workspacePath)
	if unexpected == nil {
		_ = filepath.WalkDir(workspacePath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			rel, relErr := filepath.Rel(workspacePath, path)
			if relErr != nil || rel == "." {
				return nil
			}
			if d.IsDir() {
				switch rel {
				case ".agents", ".claude", "project", ".git", "node_modules", "vendor":
					return filepath.SkipDir
				}
				return nil
			}
			base := filepath.Base(rel)
			if strings.HasPrefix(base, ".") || base == "AGENTS.md" || base == "CLAUDE.md" {
				return nil
			}
			if declared[filepath.Clean(rel)] {
				return nil
			}
			unexpected = append(unexpected, rel)
			return nil
		})
	} else {
		// Filter declared paths from the git-reported list.
		filtered := unexpected[:0]
		for _, p := range unexpected {
			if !declared[filepath.Clean(p)] {
				filtered = append(filtered, p)
			}
		}
		unexpected = filtered
	}

	if len(unexpected) == 0 {
		return
	}
	const maxList = 5
	preview := unexpected
	if len(preview) > maxList {
		preview = append(append([]string{}, preview[:maxList]...), fmt.Sprintf("(+%d more)", len(unexpected)-maxList))
	}
	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: execution.Status.ID,
		StepID:     step.ID,
		State:      "warning",
		Message:    fmt.Sprintf("step wrote %d file(s) outside declared output_artifacts paths: %s", len(unexpected), strings.Join(preview, ", ")),
	})
}

// changedFilesViaGit returns paths reported by `git status --porcelain` from
// the given workspace dir, or nil if the workspace is not a git tree (caller
// then falls back to WalkDir). Pruning of tooling state (.agents/, .claude/,
// AGENTS.md/CLAUDE.md, hidden files) matches the WalkDir branch so warnings
// stay consistent across workspace types.
func changedFilesViaGit(workspacePath string) []string {
	gitDir := filepath.Join(workspacePath, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		return nil
	}
	cmd := exec.Command("git", "-C", workspacePath, "status", "--porcelain", "--untracked-files=all")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var files []string
	for _, line := range strings.Split(string(out), "\n") {
		if len(line) < 4 {
			continue
		}
		// porcelain format: XY <space> <path>; rename pairs use ` -> `.
		path := strings.TrimSpace(line[3:])
		if idx := strings.Index(path, " -> "); idx >= 0 {
			path = path[idx+4:]
		}
		path = strings.Trim(path, `"`)
		if path == "" {
			continue
		}
		// Same prune list as WalkDir branch.
		first := path
		if i := strings.Index(path, "/"); i >= 0 {
			first = path[:i]
		}
		if isPrunedTopLevel(first) {
			continue
		}
		base := filepath.Base(path)
		if strings.HasPrefix(base, ".") || base == "AGENTS.md" || base == "CLAUDE.md" {
			continue
		}
		files = append(files, path)
	}
	return files
}

// isPrunedTopLevel reports whether the top-level directory or file name is a
// dependency/cache/tooling-state artifact never normally tracked in git. The
// list is conservative — only entries that are essentially never intentionally
// committed to a repo. Build-output names (dist, build, out, bin, obj) are NOT
// pruned because some projects do commit them.
func isPrunedTopLevel(name string) bool {
	switch name {
	// Wave-internal state + project mount.
	case ".agents", ".claude", "project", ".git":
		return true
	// Dependency dirs (universally gitignored).
	case "node_modules", "vendor", "target":
		return true
	// Tooling/cache state (universally gitignored).
	case "__pycache__", ".venv", "venv", ".tox", ".pytest_cache", ".bundle",
		".cache", ".gradle", ".mvn", ".next", ".nuxt", ".turbo":
		return true
	}
	return false
}

// parseStallTimeout parses the stall timeout from the manifest runtime config.
// Returns 0 if not configured or invalid.

func (e *DefaultPipelineExecutor) trackStepDeliverables(execution *PipelineExecution, step *Step) {
	if e.outcomeTracker == nil {
		return
	}

	// Get workspace path for this step
	execution.mu.Lock()
	workspacePath, exists := execution.WorkspacePaths[step.ID]
	execution.mu.Unlock()
	if !exists {
		return
	}

	// Track explicit output artifacts (declared in pipeline YAML only)
	for _, artifact := range step.OutputArtifacts {
		resolvedPath := execution.Context.ResolveArtifactPath(artifact)
		artifactPath := filepath.Join(workspacePath, resolvedPath)

		// Get absolute path
		absPath, err := filepath.Abs(artifactPath)
		if err != nil {
			absPath = artifactPath
		}

		e.outcomeTracker.AddFile(step.ID, artifact.Name, absPath, artifact.Type)
		// NOTE: DB registration is handled by writeOutputArtifacts (with archiving).
		// Do NOT duplicate it here.
	}

}

// buildContractPrompt generates a contract compliance section that is appended
// to the user prompt (-p argument) at execution time. This tells the persona
// exactly what format the output must be in, so pipeline authors don't need to
// repeat format requirements in their prompts.
//
// This is the SINGLE source of truth for schema injection — it includes security
// validation (path traversal, content sanitization) and the full schema content.

func (e *DefaultPipelineExecutor) processStepOutcomes(execution *PipelineExecution, step *Step) {
	if e.outcomeTracker == nil || len(step.Outcomes) == 0 {
		return
	}

	pipelineID := execution.Status.ID
	execution.mu.Lock()
	workspacePath := execution.WorkspacePaths[step.ID]
	execution.mu.Unlock()
	if workspacePath == "" {
		return
	}

	for _, outcome := range step.Outcomes {
		artifactPath := filepath.Clean(filepath.Join(workspacePath, outcome.ExtractFrom))
		cleanWorkspace := filepath.Clean(workspacePath) + string(filepath.Separator)
		if !strings.HasPrefix(artifactPath, cleanWorkspace) {
			msg := fmt.Sprintf("[%s] outcome: path %q escapes workspace, skipping", step.ID, outcome.ExtractFrom)
			e.outcomeTracker.AddOutcomeWarning(msg)
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      "warning",
				Message:    msg,
			})
			continue
		}
		data, err := os.ReadFile(artifactPath)
		if err != nil {
			msg := fmt.Sprintf("[%s] outcome: cannot read %s: %v", step.ID, outcome.ExtractFrom, err)
			e.outcomeTracker.AddOutcomeWarning(msg)
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      "warning",
				Message:    msg,
			})
			continue
		}

		// file/artifact types: use the artifact path directly as the outcome value
		if outcome.Type == "file" || outcome.Type == "artifact" {
			label := outcome.Label
			if label == "" {
				label = outcome.Type
			}
			e.registerOutcome(step.ID, outcome.Type, label, artifactPath, fmt.Sprintf("Produced by step %s", step.ID))
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      stateRunning,
				Message:    fmt.Sprintf("outcome: %s = %s", label, artifactPath),
			})
			continue
		}

		// Wildcard path: extract all array elements as separate deliverables
		if ContainsWildcard(outcome.JSONPath) {
			e.processWildcardOutcome(execution, step, outcome, data)
			continue
		}

		value, err := ExtractJSONPath(data, outcome.JSONPath)
		if err != nil {
			var emptyErr *emptyArrayError
			if errors.As(err, &emptyErr) {
				// Empty array is a "no results" condition, not an error.
				// Show a friendly message in the summary only — skip the real-time warning event.
				msg := fmt.Sprintf("[%s] outcome: no items in %s — skipping %s extraction from %s", step.ID, emptyErr.Field, outcome.JSONPath, outcome.ExtractFrom)
				e.outcomeTracker.AddOutcomeWarning(msg)
			} else {
				msg := fmt.Sprintf("[%s] outcome: %s at %s: %v", step.ID, outcome.JSONPath, outcome.ExtractFrom, err)
				e.outcomeTracker.AddOutcomeWarning(msg)
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      "warning",
					Message:    msg,
				})
			}
			continue
		}

		label := outcome.Label
		if label == "" {
			label = outcome.Type
		}
		desc := fmt.Sprintf("Extracted from %s at %s", outcome.ExtractFrom, outcome.JSONPath)

		e.registerOutcome(step.ID, outcome.Type, label, value, desc)

		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      stateRunning,
			Message:    fmt.Sprintf("outcome: %s = %s", label, value),
		})
	}
}

// processWildcardOutcome handles outcome definitions with [*] wildcard paths,
// extracting all array elements and registering each as a separate deliverable.
func (e *DefaultPipelineExecutor) processWildcardOutcome(execution *PipelineExecution, step *Step, outcome OutcomeDef, data []byte) {
	pipelineID := execution.Status.ID

	values, err := ExtractJSONPathAll(data, outcome.JSONPath)
	if err != nil {
		msg := fmt.Sprintf("[%s] outcome: %s at %s: %v", step.ID, outcome.JSONPath, outcome.ExtractFrom, err)
		e.outcomeTracker.AddOutcomeWarning(msg)
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "warning",
			Message:    msg,
		})
		return
	}

	// Empty array — log friendly message and skip
	if len(values) == 0 {
		msg := fmt.Sprintf("[%s] outcome: empty array at %s — skipping extraction from %s", step.ID, outcome.JSONPath, outcome.ExtractFrom)
		e.outcomeTracker.AddOutcomeWarning(msg)
		return
	}

	// Extract per-item labels if json_path_label is set
	var labels []string
	if outcome.JSONPathLabel != "" && ContainsWildcard(outcome.JSONPathLabel) {
		labels, _ = ExtractJSONPathAll(data, outcome.JSONPathLabel)
	}

	baseLabel := outcome.Label
	if baseLabel == "" {
		baseLabel = outcome.Type
	}

	total := len(values)
	for i, value := range values {
		var label string
		if i < len(labels) && labels[i] != "" {
			label = fmt.Sprintf("%s: %s", baseLabel, labels[i])
		} else {
			label = fmt.Sprintf("%s (%d/%d)", baseLabel, i+1, total)
		}

		desc := fmt.Sprintf("Extracted from %s at %s [%d]", outcome.ExtractFrom, outcome.JSONPath, i)
		e.registerOutcome(step.ID, outcome.Type, label, value, desc)

		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      stateRunning,
			Message:    fmt.Sprintf("outcome: %s = %s", label, value),
		})
	}
}

// registerOutcome routes a declared step outcome through the appropriate
// OutcomeTracker convenience method based on its type.
func (e *DefaultPipelineExecutor) registerOutcome(stepID, outcomeType, label, value, desc string) {
	switch outcomeType {
	case "pr":
		e.outcomeTracker.AddPR(stepID, label, value, desc)
	case "issue":
		e.outcomeTracker.AddIssue(stepID, label, value, desc)
	case "deployment":
		e.outcomeTracker.AddDeployment(stepID, label, value, desc)
	case "file":
		e.outcomeTracker.AddFile(stepID, label, value, desc)
	case "artifact":
		e.outcomeTracker.AddArtifact(stepID, label, value, desc)
	default:
		// "url" or any unknown type → generic URL
		e.outcomeTracker.AddURL(stepID, label, value, desc)
	}
}

// GetOutcomesSummary returns the formatted outcome summary for the completed pipeline.
