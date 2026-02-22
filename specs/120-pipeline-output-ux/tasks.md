# Tasks: Pipeline Output UX — Surface Key Outcomes

**Branch**: `120-pipeline-output-ux` | **Date**: 2026-02-20
**Spec**: `specs/120-pipeline-output-ux/spec.md` | **Plan**: `specs/120-pipeline-output-ux/plan.md`

---

## Phase 1: Setup

- [X] T001 P1 Setup — Create `internal/display/outcome.go` with package declaration and imports for the new outcome rendering file
  - File: `internal/display/outcome.go`
  - Create the new file with `package display` declaration and standard imports (`fmt`, `strings`, `time`, `sort`, `internal/deliverable`)

---

## Phase 2: Foundational — Deliverable Type Extensions (Layer 1)

These tasks extend the deliverable system with `TypeBranch` and `TypeIssue` support. All Layer 2+ tasks depend on this phase.

- [X] T002 P1 US1 — Add `TypeBranch` and `TypeIssue` constants to `DeliverableType` const block
  - File: `internal/deliverable/types.go`
  - Add `TypeBranch DeliverableType = "branch"` and `TypeIssue DeliverableType = "issue"` to the existing const block (after `TypeArtifact`, before `TypeOther`)

- [X] T003 [P] P1 US1 — Add `NewBranchDeliverable()` constructor function
  - File: `internal/deliverable/types.go`
  - Create `NewBranchDeliverable(stepID, branchName, worktreePath, description string) *Deliverable` that returns a `TypeBranch` deliverable with `Metadata: map[string]any{"pushed": false}`
  - Follow the pattern of existing constructors (`NewPRDeliverable`, etc.)

- [X] T004 [P] P1 US1 — Add `NewIssueDeliverable()` constructor function
  - File: `internal/deliverable/types.go`
  - Create `NewIssueDeliverable(stepID, name, issueURL, description string) *Deliverable` that returns a `TypeIssue` deliverable with `Path` set to the issue URL
  - Follow the pattern of existing constructors

- [X] T005 P1 US1 — Add `TypeBranch` and `TypeIssue` icon cases to `Deliverable.String()`
  - File: `internal/deliverable/types.go`
  - Nerd font: `TypeBranch` → `"🌿"`, `TypeIssue` → `"📌"`
  - ASCII: `TypeBranch` → `"⎇"`, `TypeIssue` → `"!"`
  - Add both cases in each switch block (nerd font and ASCII), before the `default` case

- [X] T006 [P] P1 US1 — Add `Tracker.AddBranch()` and `Tracker.AddIssue()` convenience methods
  - File: `internal/deliverable/tracker.go`
  - `AddBranch(stepID, branchName, worktreePath, description string)` → calls `t.Add(NewBranchDeliverable(...))`
  - `AddIssue(stepID, name, issueURL, description string)` → calls `t.Add(NewIssueDeliverable(...))`
  - Follow the pattern of existing convenience methods (`AddPR`, `AddDeployment`, etc.)

- [X] T007 P1 US1 — Add `Tracker.UpdateMetadata()` method for post-creation metadata updates
  - File: `internal/deliverable/tracker.go`
  - `UpdateMetadata(deliverableType DeliverableType, name string, key string, value any)` — finds first matching deliverable by type+name and sets `Metadata[key] = value`
  - Must be thread-safe (acquire `t.mu.Lock()`)
  - Initialize `Metadata` map if nil

- [X] T008 P1 US1 — Add unit tests for new deliverable types, constructors, and tracker methods
  - File: `internal/deliverable/types_test.go` (new file)
  - Table-driven tests for: `NewBranchDeliverable` fields, `NewIssueDeliverable` fields, `String()` rendering for both types (nerd font and ASCII paths), `GetByType(TypeBranch)` and `GetByType(TypeIssue)`, `AddBranch()`/`AddIssue()` convenience methods, `UpdateMetadata()` with push status update, `UpdateMetadata()` on non-existent deliverable (no-op)

---

## Phase 3: US1 — Scannable Completion Summary (Outcome Aggregation & Rendering)

Depends on Phase 2. This is the core user-facing change.

- [X] T009 P1 US1 — Define `PipelineOutcome` and supporting types in `outcome.go`
  - File: `internal/display/outcome.go`
  - Define structs: `PipelineOutcome` (with fields: PipelineName, RunID, Success, Duration, Tokens, Branch, Pushed, RemoteRef, PushError, PullRequests, Issues, Deployments, Reports, ArtifactCount, ContractsPassed, ContractsFailed, ContractsTotal, FailedContracts, NextSteps, WorkspacePath, AllDeliverables)
  - Define supporting types: `OutcomeLink{Label, URL string}`, `OutcomeFile{Label, Path string}`, `ContractFailure{StepID, Type, Message string}`, `NextStep{Label, Command, URL string}`
  - See `data-model.md` for exact field definitions

- [X] T010 P1 US1 — Implement `BuildOutcome()` function to construct `PipelineOutcome` from tracker data
  - File: `internal/display/outcome.go`
  - Signature: `BuildOutcome(tracker *deliverable.Tracker, pipelineName, runID string, success bool, duration time.Duration, tokens int, workspacePath string) *PipelineOutcome`
  - Extract branch deliverables → populate `Branch`, `Pushed`, `RemoteRef`, `PushError` from metadata
  - Extract PR deliverables → populate `PullRequests` as `[]OutcomeLink`
  - Extract issue deliverables → populate `Issues`
  - Extract deployment deliverables → populate `Deployments`
  - Count detail-level deliverables (file, log, contract, artifact) → `ArtifactCount`
  - Count contract deliverables → `ContractsPassed`, `ContractsTotal` (all contracts assumed passed unless metadata says otherwise)
  - Extract top 5 outcome-worthy files by priority (PRs > issues > branches > deployments > reports)
  - Store all deliverables in `AllDeliverables` for verbose mode

- [X] T011 P1 US1 — Implement `GenerateNextSteps()` for contextual follow-up suggestions
  - File: `internal/display/outcome.go`
  - Rules: if PR exists → "Review the pull request" with URL; if branch pushed → "View changes on remote"; if worktree workspace path set → "Inspect workspace at <path>"
  - Return `[]NextStep`
  - Called by `BuildOutcome()` to populate `NextSteps` field

- [X] T012 P1 US1 — Implement `RenderOutcomeSummary()` for human-readable output
  - File: `internal/display/outcome.go`
  - Signature: `RenderOutcomeSummary(outcome *PipelineOutcome, verbose bool, formatter *Formatter) string`
  - Print "Outcomes" header using `formatter.Bold()`
  - For each non-empty outcome category (branch, PRs, issues, deployments): render with label and icon using `formatter.Success()` / `formatter.Primary()`
  - If `PushError` is non-empty: render push failure as warning using `formatter.Warning()`
  - Print artifact summary as single line: "N artifacts produced" using `formatter.Muted()`
  - Print contract summary as single line: "N/M contracts passed" — show failures prominently with `formatter.Error()`
  - If verbose: print full deliverable list after summary
  - Suppress empty categories per FR-008 (no "None" lines)
  - Return the formatted string (caller writes to stderr)

- [X] T013 P1 US1 — Add unit tests for `BuildOutcome()` with various deliverable combinations
  - File: `internal/display/outcome_test.go` (new file)
  - Test cases: pipeline with branch+PR+artifacts, pipeline with no outcomes (empty), pipeline with only artifacts (no branch/PR), pipeline with push failure, pipeline with 50+ deliverables (verify top-5 truncation), pipeline with contract failures
  - Verify field populations and counts are correct

- [X] T014 [P] P1 US1 — Add unit tests for `RenderOutcomeSummary()` output formatting
  - File: `internal/display/outcome_test.go`
  - Test cases: default mode renders summary lines, verbose mode includes full list, empty outcome suppression (FR-008), contract failure prominence in all modes, non-TTY produces no ANSI codes (use `NewFormatterWithConfig("off", true)`)
  - Use string matching to verify output structure

---

## Phase 4: US1 — Executor Instrumentation (Layer 2)

Depends on Phase 2. Records branch/issue deliverables during execution.

- [X] T015 P1 US1 — Record `TypeBranch` deliverable on worktree creation in `executor.go`
  - File: `internal/pipeline/executor.go`
  - After worktree creation succeeds (line ~801, after `execution.WorktreePaths[branch] = ...`), add: `e.deliverableTracker.AddBranch(step.ID, branch, absPath, "Feature branch")`
  - Only record when a new worktree is created (not when reusing existing via `execution.WorktreePaths[branch]`)

- [X] T016 [P] P1 US1 — Add issue URL detection to `trackCommonDeliverables()`
  - File: `internal/pipeline/executor.go`
  - In the results scanning section, add check for `results["issue_url"]` key → `e.deliverableTracker.AddIssue(...)`
  - Also scan string values for `github.com/.*/issues/\d+` URL patterns → add as `TypeIssue` deliverables
  - Import `regexp` if needed; compile pattern as package-level var

- [X] T017 P1 US1 — Update branch deliverable metadata after publish step push
  - File: `internal/pipeline/executor.go`
  - In `trackCommonDeliverables()`: when a step result contains `pushed: true` or when the step is detected as a publish step (by scanning for push-related keys in results), call `e.deliverableTracker.UpdateMetadata(deliverable.TypeBranch, branchName, "pushed", true)` and optionally set `"remote_ref"`
  - The branch name is available from `execution.WorktreePaths` (iterate to find matching branch)

---

## Phase 5: US2 — Artifact/Contract Details Relegated to Secondary Display

Depends on Phase 3 (rendering is in `RenderOutcomeSummary`). Most of this is already handled by the summary vs verbose logic in T012.

- [X] T018 P2 US2 — Integrate outcome summary into `run.go` replacing raw `FormatSummary()` call
  - File: `cmd/wave/commands/run.go`
  - In `runRun()` (lines 337-364), after `result.Cleanup()`:
    1. Get `tracker := executor.GetDeliverableTracker()`
    2. Call `outcome := display.BuildOutcome(tracker, p.Metadata.Name, runID, true, elapsed, executor.GetTotalTokens(), "")`
    3. For auto/text modes: replace the current `executor.GetDeliverables()` block with `display.RenderOutcomeSummary(outcome, opts.Output.Verbose, display.NewFormatter())`
    4. Print the rendered summary to stderr
  - Import `internal/display` (likely already imported) and remove the call to `executor.GetDeliverables()`

- [X] T019 [P] P2 US2 — Verify quiet mode suppresses outcomes (no changes needed, add test)
  - File: `cmd/wave/commands/run.go` (verify) + test
  - Verify the existing conditional `if opts.Output.Format == OutputFormatAuto || opts.Output.Format == OutputFormatText` already excludes quiet and json modes
  - Add a comment documenting that quiet mode suppresses outcomes by design (FR-012)

---

## Phase 6: US3 — Structured JSON Output for Automation (Layer 4)

Depends on Phase 3 (`PipelineOutcome` struct).

- [X] T020 [P] P3 US3 — Add `OutcomesJSON` and supporting types to `internal/event/emitter.go`
  - File: `internal/event/emitter.go`
  - Add types: `OutcomesJSON` struct with `Branch`, `Pushed`, `RemoteRef`, `PushError`, `PullRequests`, `Issues`, `Deployments`, `Deliverables` fields (all with json tags, omitempty where appropriate)
  - Add `OutcomeLinkJSON{Label, URL string}` and `DeliverableJSON{Type, Name, Path, Description, StepID string}` types
  - Add `Outcomes *OutcomesJSON \`json:"outcomes,omitempty"\`` field to `Event` struct

- [X] T021 P3 US3 — Implement `ToOutcomesJSON()` conversion from `PipelineOutcome` to `OutcomesJSON`
  - File: `internal/display/outcome.go`
  - `func (o *PipelineOutcome) ToOutcomesJSON() *event.OutcomesJSON`
  - Convert PullRequests, Issues, Deployments to `[]OutcomeLinkJSON`
  - Convert AllDeliverables to `[]DeliverableJSON`
  - Map Branch, Pushed, RemoteRef, PushError fields directly
  - Note: this creates a dependency from `display` → `event` package; if circular, define `OutcomesJSON` in a shared types package or in `display` itself and have `event` import it. Alternatively, define a `ToJSON() map[string]any` method to avoid the import.

- [X] T022 P3 US3 — Emit structured outcomes in final JSON completion event from `run.go`
  - File: `cmd/wave/commands/run.go`
  - For JSON output mode: after building `PipelineOutcome`, convert to `OutcomesJSON` and attach to the final completion event
  - The current JSON emitter emits events via `emitter.Emit(event.Event{...})` — set the `Outcomes` field on the final pipeline-completed event

- [X] T023 P3 US3 — Add unit tests for JSON serialization of `OutcomesJSON`
  - File: `internal/event/emitter_test.go` (extend existing or create)
  - Test cases: Event with populated Outcomes serializes correctly, Event with nil Outcomes omits field (backward compat), Outcomes with empty arrays serializes as `[]` not `null`

---

## Phase 7: US4 — Contextual Next Steps

Depends on Phase 3 (T011 already implements the core logic). This phase ensures next steps render correctly in all modes.

- [X] T024 [P] P3 US4 — Add "Next Steps" rendering section to `RenderOutcomeSummary()`
  - File: `internal/display/outcome.go`
  - After the outcomes and artifact/contract summary, render a "Next Steps" section if `outcome.NextSteps` is non-empty
  - Format each step as `"  → <Label>"` with optional command/URL
  - Skip rendering in quiet mode (caller controls this; renderer just needs to handle empty NextSteps gracefully)

- [X] T025 P3 US4 — Add unit tests for next steps generation and rendering
  - File: `internal/display/outcome_test.go`
  - Test cases: PR exists → suggests "Review the pull request", branch pushed → suggests remote view, worktree workspace → suggests inspection, no outcomes → no next steps, multiple next steps rendered correctly

---

## Phase 8: Polish & Cross-Cutting Concerns

- [X] T026 P1 — Handle edge case: failed step deliverables shown with warning marker
  - File: `internal/display/outcome.go`
  - In `BuildOutcome()` or `RenderOutcomeSummary()`: if a deliverable comes from a failed step (check `StepID` against failed step list or add a `Failed bool` to the outcome struct), prefix with `"[FAILED]"` marker
  - This requires passing failed step IDs to `BuildOutcome()` — extend the function signature or add a `FailedStepIDs []string` parameter

- [X] T027 [P] P2 — Handle edge case: large deliverable list truncation (50+ items)
  - File: `internal/display/outcome.go`
  - In `RenderOutcomeSummary()` default mode: if `ArtifactCount > 5`, show only top 5 by type priority + `"... and N more"` line
  - Verbose mode shows all
  - Priority ordering: PRs > issues > branches > deployments > reports > files

- [X] T028 [P] P2 — Handle edge case: non-TTY output has no ANSI codes in outcomes
  - File: `internal/display/outcome.go`
  - Verify all output uses `Formatter` methods (which auto-disable ANSI in non-TTY)
  - Add a test that creates a `Formatter` with ANSI disabled and verifies the output contains no escape codes (`\033[`)

- [X] T029 P1 — Run full test suite and fix any regressions
  - Command: `go test ./...`
  - Ensure all existing tests pass alongside new tests
  - Fix any compilation errors or test failures introduced by the changes
  - Run with `-race` flag to check for race conditions in new tracker methods

---

## Dependency Graph

```
Phase 1 (T001)
    ↓
Phase 2 (T002-T008) — foundational types
    ↓               ↘
Phase 3 (T009-T014)  Phase 4 (T015-T017)
    ↓      ↘             ↓
Phase 5     Phase 6    [merges into Phase 5]
(T018-T019) (T020-T023)
    ↓           ↓
Phase 7 (T024-T025)
    ↓
Phase 8 (T026-T029) — polish & validation
```

## Parallelization Notes

Tasks marked `[P]` can be worked in parallel with other tasks in the same phase:
- T003 ∥ T004 (independent constructors)
- T005 can start after T002 (needs type constants)
- T006 ∥ T007 (independent tracker methods, both need T003/T004)
- T013 ∥ T014 (independent test files)
- T019 ∥ T018 (verification vs integration)
- T020 ∥ T021 (types vs conversion, but T021 depends on T020's types)
- T024 ∥ T025 (rendering vs tests)
- T027 ∥ T028 (independent edge cases)
