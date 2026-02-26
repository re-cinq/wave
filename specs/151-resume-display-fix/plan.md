# Implementation Plan: Resume Display Fix

## Objective

Fix the display bug where prior completed steps show as pending (○) when using `--from-step` to resume a pipeline, and address the secondary issue of shared-worktree activity misattribution.

## Approach

The fix follows the proposed approach from the issue: emit synthetic completion events for prior steps after `loadResumeState()` in `ResumeFromStep()`. This leverages the existing event-driven display architecture — all three display backends (BubbleTea, BasicProgressDisplay, QuietProgressDisplay) already handle "completed" events correctly.

### Strategy: Synthetic completion events in ResumeManager

After `loadResumeState()` populates `resumeState.CompletedSteps`, emit a `StateCompleted` event for each prior step. This way:

1. **BubbleTeaProgressDisplay** receives the events via `updateFromEvent()` → sets `step.State = StateCompleted` → renders ✓
2. **BasicProgressDisplay** receives the events → prints `[HH:MM:SS] ✓ stepID completed`
3. **ProgressDisplay** (fallback) receives the events → calls `step.UpdateState(StateCompleted)` → renders ✓

For the shared-worktree activity bug, clear stale per-step tool activity entries when a step completes.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/pipeline/resume.go` | modify | Emit synthetic `StateCompleted` events for prior steps after `loadResumeState()` |
| `internal/display/bubbletea_progress.go` | modify | Clear `stepToolActivity` for non-running steps to prevent stale activity display |
| `internal/pipeline/resume_test.go` | modify | Add test for synthetic completion event emission |
| `internal/display/bubbletea_progress_test.go` | create | Add test for stale activity cleanup on shared worktrees |
| `internal/display/progress_test.go` | modify | Add test that BasicProgressDisplay handles synthetic completion events |

## Architecture Decisions

### AD-1: Emit events in ResumeManager, not in CreateEmitter

**Decision**: Emit synthetic events from `ResumeFromStep()` rather than modifying `CreateEmitter` to accept resume state.

**Rationale**: `CreateEmitter` is called in `run.go` before the executor is built. Adding resume-awareness there would require passing resume state through the command layer. Emitting from `ResumeManager` keeps the fix localized to the resume code path and uses the existing event infrastructure.

### AD-2: Use existing "completed" state, not a new "prior_completed" state

**Decision**: Emit standard `StateCompleted` events rather than introducing a new event state.

**Rationale**: The display already handles "completed" state correctly. Introducing a new state would require changes across all display backends. The synthetic events should include a message like "completed in prior run" to distinguish them from live completions, and set `DurationMs: 0` since exact timing isn't available.

### AD-3: Address shared-worktree bug minimally

**Decision**: Clear stale `stepToolActivity` entries when a step transitions to completed/failed/skipped in `BubbleTeaProgressDisplay.updateFromEvent()`.

**Rationale**: The activity events are correctly keyed by `step.ID` in `executor.go:651-660`. The issue is that stale entries in `stepToolActivity` persist after a step completes, and if another step shares the same worktree, the display may show confusing activity. Cleaning up on state transition is sufficient.

## Risks

| Risk | Mitigation |
|------|------------|
| Synthetic events could confuse JSON consumers expecting only live events | Use the existing "resuming" state or add a `Message` field indicating "prior run" — JSON consumers already handle unknown messages gracefully |
| BasicProgressDisplay prints "[HH:MM:SS] ✓ stepID completed (0.0s, 0 tokens)" which looks odd | Set `DurationMs` to 0 and use a message like "prior run" so the output reads naturally |
| Race condition: events emitted before display is ready | Events are emitted after executor construction in `ResumeFromStep()` and the emitter is already set on the executor — the display is guaranteed to be receiving by this point |
| Shared-worktree fix might suppress legitimate activity | Only clear activity for steps that are no longer running — running steps keep their activity |

## Testing Strategy

### Unit Tests

1. **`resume_test.go`**: Test that `ResumeFromStep()` emits synthetic completion events for each prior step, verify event fields (StepID, State, Message)
2. **`bubbletea_progress_test.go`**: Test that `EmitProgress` with a "completed" event clears `stepToolActivity` for that step; test that shared-worktree steps don't show duplicate activity after one completes
3. **`progress_test.go`**: Test that `BasicProgressDisplay.EmitProgress` handles synthetic completion events with zero duration gracefully

### Integration Tests

4. **`resume_test.go`**: End-to-end test with mock adapter: run a 3-step pipeline, resume from step 2, verify display shows step 1 as ✓ completed

### Manual Verification

- Run `wave run code-review --input <url> --from-step <step>` and visually confirm prior steps show ✓
- Test with `--output text` and `--output json` to verify all backends
