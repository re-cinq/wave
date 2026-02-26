# fix(resume): --from-step shows prior steps as pending instead of completed

**Issue**: [#151](https://github.com/re-cinq/wave/issues/151)
**Author**: nextlevelshit
**Labels**: bug, display, resume
**Severity**: Low (cosmetic)

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

### Bug 1: Display initialized before resume state applied

`CreateEmitter` (`cmd/wave/commands/output.go:51`) receives the full pipeline steps (`len(p.Steps)` = 5) and registers all steps as `StateNotStarted`. The `ResumeManager` later creates a subpipeline with only the remaining steps, but the display already shows all 5 steps. No completion events are emitted for prior steps, so they remain as pending (○).

In `ResumeFromStep()` (`internal/pipeline/resume.go:98-125`), after `loadResumeState()` populates `resumeState.CompletedSteps`, the code emits informational "resuming" events but never emits `StateCompleted` events. The display only updates step status via events.

### Bug 2: Shared-worktree activity misattribution (from comment)

When multiple steps share the same worktree via `workspace.ref`, the display shows them as both running simultaneously with duplicate activity lines, even though the executor runs them sequentially. Activity events from the adapter are keyed by `step.ID` in `executor.go:651-660`, so this appears to be a display-side issue where the BubbleTea model shows stale tool activity for steps that share a workspace.

## Acceptance Criteria

1. **Prior steps show as completed**: When `--from-step` is used, all steps before the resume point that have valid workspace/artifact state should display as ✓ completed
2. **Duration shown for prior steps**: Prior completed steps should show a duration indicator (e.g., "prior run") since exact timing isn't available
3. **All display backends supported**: Fix must work for BubbleTea TUI (auto), BasicProgressDisplay (text), and QuietProgressDisplay (quiet) — all three receive events from the same emitter
4. **No behavioral changes**: Pipeline execution logic, artifact recovery, and contract validation remain unchanged
5. **Shared-worktree activity isolation** (secondary): Steps sharing a worktree should not show duplicate activity lines

## Notes

- The resume logic itself works correctly — artifacts are recovered and injected properly
- This is purely a display issue that does not affect pipeline execution or output correctness
- The shared-worktree activity bug (from the issue comment) has a different root cause and may warrant a separate issue
