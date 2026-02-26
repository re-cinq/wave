# fix(resume): --from-step shows prior steps as pending instead of completed

**Issue**: [#151](https://github.com/re-cinq/wave/issues/151)
**Feature Branch**: `151-resume-display-status`
**Labels**: bug, display, resume
**Severity**: Low (cosmetic)
**Author**: nextlevelshit

## Problem

When using `--from-step` to resume a pipeline, prior completed steps show as pending (○) instead of completed (✓).

## Expected Behavior

Prior steps that were already completed before the resume point should display with a ✓ checkmark, reflecting their actual completed status:

```
wave run code-review --input <url> --from-step security-review

✓ diff-analysis (navigator)          ← completed in prior run
✓ security-review (auditor) (475.6s)
⠌ quality-review (auditor) (33s)
○ summary (summarizer)
○ publish (github-commenter)
```

## Observed Behavior

```
wave run code-review --input <url> --from-step security-review

○ diff-analysis (navigator)          ← should be ✓
✓ security-review (auditor) (475.6s)
⠌ quality-review (auditor) (33s)
○ summary (summarizer)
○ publish (github-commenter)
```

## Root Cause Analysis

Two issues identified:

### Bug 1: Prior steps shown as pending during resume

1. **Display initialized before resume** (`cmd/wave/commands/output.go:51`): `CreateEmitter` receives the full pipeline (`len(p.Steps)` = 5) and registers all steps. The `ResumeManager` later creates a subpipeline with fewer steps, but the display already shows all steps.

2. **No completion events for prior steps**: `loadResumeState()` in `internal/pipeline/resume.go` populates `execution.Status.CompletedSteps` but never emits completion events. The display only updates via events.

### Bug 2: Shared-worktree steps shown as concurrent (from issue comment)

When multiple steps share the same worktree (`branch: "{{ pipeline_id }}"`), the display shows them as both running simultaneously with duplicate activity lines, even though the executor runs them sequentially.

**Root cause**: Activity events from the adapter (TodoWrite, Bash, etc.) are keyed by workspace path rather than step ID. When two steps share a worktree, both get the same activity updates in the display.

Example:
```
2/4 steps (2 ok, 2 running)    ← should be "1 running"

✓ scan-issues (github-analyst) (135.1s)
✓ plan-enhancements (github-analyst) (86.0s)
⠹ apply-enhancements (github-enhancer) (43s)
   TodoWrite → Verify each enhanced issue via gh CLI
⠹ verify-enhancements (github-analyst) (43s)       ← should be ○ (pending)
   TodoWrite → Verify each enhanced issue via gh CLI  ← duplicate activity
```

## Proposed Fix

### Bug 1 Fix

In `ResumeFromStep()`, after `loadResumeState()`, emit synthetic completion events for each step in `resumeState.CompletedSteps` so the display marks them as done. The events should include the step's persona and a zero-duration marker indicating these are synthetic/recovered completions.

### Bug 2 Fix

Ensure stream_activity events are keyed by step ID, not workspace path. The `OnStreamEvent` callback in `executor.go:650-662` already uses `step.ID` correctly, so the issue may be in how the BubbleTea display matches events to steps when multiple steps share the same workspace. Investigate the `stepToolActivity` map and `updateFromEvent` in `bubbletea_progress.go`.

## Acceptance Criteria

1. Prior completed steps display ✓ (not ○) when resuming with `--from-step`
2. Shared-worktree steps do not show as concurrently running when they are sequential
3. Activity lines only appear under the currently executing step
4. All existing resume tests continue to pass
5. All existing display tests continue to pass

## Notes

- The resume logic itself works correctly — artifacts are recovered and injected properly.
- These are purely display issues that do not affect pipeline execution or output correctness.
- The `stream_activity` events already carry `step.ID` in the executor callback (line 655), so Bug 2 may be in the display layer's event routing.
