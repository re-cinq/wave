# Implementation Plan: Resume Display Fix (#151)

## Objective

Fix two display bugs in Wave's pipeline resume feature: (1) prior completed steps showing as pending (○) instead of completed (✓) when using `--from-step`, and (2) shared-worktree steps incorrectly showing as concurrently running with duplicate activity lines.

## Approach

### Bug 1: Synthetic completion events for resumed steps

**Strategy**: After `loadResumeState()` populates `CompletedSteps`, emit synthetic `completed` events for each prior step before executing the subpipeline. This bridges the gap between the state layer (which knows steps are done) and the display layer (which only updates via events).

**Key insight**: The display (`CreateEmitter` in `output.go:51`) registers ALL pipeline steps upfront. When `ResumeFromStep` creates a subpipeline, it only contains steps from the resume point onward. But the display still tracks all steps. Without synthetic events, prior steps remain in `StateNotStarted`.

**Implementation location**: `internal/pipeline/resume.go`, in `ResumeFromStep()`, between lines 125-127 (after resume state summary events, before run ID generation). Emit `event.Event{State: "completed"}` for each step ID in `resumeState.CompletedSteps`.

### Bug 2: Step ID-based activity routing

**Strategy**: The executor's `OnStreamEvent` callback (`executor.go:651-661`) already tags `stream_activity` events with the correct `step.ID`. The display's `updateFromEvent` in `bubbletea_progress.go:209` correctly routes events by `evt.StepID`. The actual problem is more subtle — when steps share a worktree, there may be residual activity from a prior step's workspace that gets attributed to the next step because the adapter process hasn't fully terminated before the next step starts.

**Investigation needed**: Verify whether the issue is:
1. Events arriving after step completion with a stale step ID
2. Workspace-path-based lookup somewhere in the event chain
3. A timing issue in the throttled emitter coalescing events across step boundaries

**Implementation location**: Likely `internal/display/bubbletea_progress.go` — clean up `stepToolActivity` more aggressively on step completion events, and ensure activity events for completed steps are silently dropped.

## File Mapping

### Files to Modify

| File | Action | Purpose |
|------|--------|---------|
| `internal/pipeline/resume.go` | modify | Emit synthetic completion events for prior steps |
| `internal/display/bubbletea_progress.go` | modify | Drop activity events for completed/not-yet-started steps; clear stale activity |
| `internal/display/progress.go` | modify | Same fix for `BasicProgressDisplay` and `ProgressDisplay` |

### Files to Create

| File | Purpose |
|------|---------|
| `internal/pipeline/resume_display_test.go` | Tests for synthetic event emission on resume |
| `internal/display/bubbletea_progress_resume_test.go` | Tests for display state after receiving synthetic events |

### Files Unchanged (Read-Only Context)

| File | Relevance |
|------|-----------|
| `cmd/wave/commands/output.go` | Where `CreateEmitter` registers all steps — no change needed, the fix is upstream |
| `cmd/wave/commands/run.go` | Calls `ResumeWithValidation` — no change needed |
| `internal/event/emitter.go` | Event types — no new event types needed |
| `internal/display/types.go` | Display state types — no change needed |

## Architecture Decisions

1. **Synthetic events over display API**: Instead of adding a new method to `BubbleTeaProgressDisplay` to pre-mark steps as completed, we emit standard events. This ensures ALL display backends (BubbleTea, Basic, Quiet, JSON) handle resume correctly without separate code paths.

2. **No new event state**: Use existing `"completed"` state for synthetic events rather than introducing a new `"resumed_completed"` state. The display treats them identically. A `Message` field like `"Completed in prior run"` distinguishes them for logging/debugging.

3. **Guard on step state in display**: Activity events for steps that are already `StateCompleted` or still `StateNotStarted` (and not the target step) should be silently dropped. This prevents phantom activity leakage between shared-worktree steps.

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Synthetic events arrive before display is fully initialized | Low | Events go through the same emitter chain; display registers steps in `CreateEmitter` before `ResumeFromStep` runs |
| Race condition between synthetic events and real step-start events | Low | Synthetic events are emitted synchronously before `executeResumedPipeline` begins |
| Bug 2 root cause is different from activity leakage | Medium | Need to reproduce and trace events; fallback plan is to add workspace→stepID mapping |
| Breaking existing `BasicProgressDisplay` tests | Low | Only adding event handling, not changing existing behavior |

## Testing Strategy

### Unit Tests

1. **Resume synthetic events** (`internal/pipeline/resume_display_test.go`):
   - Verify `ResumeFromStep` emits N completion events for N prior steps
   - Verify synthetic events carry correct step IDs and personas
   - Verify no synthetic events emitted when resuming from the first step

2. **Display state after resume** (`internal/display/bubbletea_progress_resume_test.go`):
   - Feed synthetic completion events into `BubbleTeaProgressDisplay.updateFromEvent`
   - Verify `toPipelineContext()` reports correct `CompletedSteps` count
   - Verify step states show `StateCompleted` for prior steps

3. **Activity event guard** (add to existing display tests):
   - Send `stream_activity` events for already-completed steps
   - Verify they are silently dropped (no `stepToolActivity` entry)

### Integration Tests

4. **End-to-end resume display** (extend `cmd/wave/commands/run_test.go`):
   - Mock pipeline with 3+ steps, resume from step 2
   - Capture emitted events and verify:
     - Step 1 receives a synthetic completion event
     - Steps 2+ receive real started/completed events

### Manual Verification

5. Run an actual pipeline with `--from-step` and visually confirm:
   - Prior steps show ✓
   - Running step shows spinner
   - Pending steps show ○
