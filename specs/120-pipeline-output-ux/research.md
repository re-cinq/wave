# Research: Pipeline Output UX — Surface Key Outcomes

**Date**: 2026-02-20
**Spec**: `specs/120-pipeline-output-ux/spec.md`
**Branch**: `120-pipeline-output-ux`

## Current Architecture Analysis

### 1. Deliverable Tracking System

**File**: `internal/deliverable/types.go`, `internal/deliverable/tracker.go`

The existing deliverable system has:
- **8 types**: `file`, `url`, `pr`, `deployment`, `log`, `contract`, `artifact`, `other`
- **Missing types**: No `branch` or `issue` type constants
- **Tracker**: Thread-safe `Tracker` struct with `Add()`, `GetAll()`, `GetByType()`, `GetByStep()`
- **Constructors**: `NewFileDeliverable`, `NewURLDeliverable`, `NewPRDeliverable`, etc.
- **FormatSummary()**: Renders all deliverables as a flat list with icons — this is the "wall of text" problem

**Finding**: The current `FormatSummary()` renders *all* deliverables identically with no distinction between high-value outcomes (PRs, branches) and low-value details (log files, contract artifacts). This is the core UX issue.

### 2. Post-Execution Output (run.go)

**File**: `cmd/wave/commands/run.go:337-364`

Current completion flow:
1. `result.Cleanup()` stops the TUI
2. For auto/text modes: prints success line with elapsed time and tokens
3. Calls `executor.GetDeliverables()` which returns `tracker.FormatSummary()` — a flat list of all artifacts
4. No structured outcome data — branch name, push status, PR URLs are buried in the flat deliverable list

**Finding**: The run command directly calls `FormatSummary()` and prints it. There's no filtering, grouping, or prioritization. The fix requires intercepting at this boundary and introducing a structured summary.

### 3. Executor & Worktree Integration

**File**: `internal/pipeline/executor.go:753-802`

Worktree creation flow:
1. Branch name is resolved from `step.Workspace.Branch` template variable
2. `worktree.NewManager("")` creates the git worktree
3. Path is registered in `execution.WorktreePaths[branch]`
4. **No deliverable is recorded** for branch creation

**Finding**: Branch creation happens but is invisible to the deliverable system. The branch name is in `execution.WorktreePaths` but never surfaces as a deliverable. Adding a `TypeBranch` deliverable here is the natural instrumentation point.

### 4. PR/Issue Detection

**File**: `internal/pipeline/executor.go:1304-1347`

`trackCommonDeliverables()` detects PRs:
- Checks `results["pr_url"]` — but this depends on the adapter extracting PR URLs from step output
- Scans all result values for HTTP URLs
- No dedicated issue detection

**Finding**: PR detection is opportunistic, relying on step results containing `pr_url`. Publish steps (e.g., `feature.yaml:publish`) use `gh pr create` and `git push` but their results aren't reliably parsed. Need to ensure the PR URL from `gh pr create` output reaches `results["pr_url"]`.

### 5. Event & Output System

**File**: `cmd/wave/commands/output.go`, `internal/event/emitter.go`

Output modes:
- `auto` → BubbleTea TUI (TTY) or BasicProgressDisplay (pipe)
- `json` → NDJSON to stdout
- `text` → BasicProgressDisplay to stderr
- `quiet` → QuietProgressDisplay (only pipeline-level events)

**Finding**: The JSON emitter (`NDJSONEmitter`) emits `Event` structs directly. The final completion event currently has no `outcomes` field. Adding structured outcomes requires extending `Event` with an optional `Outcomes` field emitted only in the final completion event.

### 6. Display Infrastructure

**File**: `internal/display/formatter.go`, `internal/display/types.go`

Rich formatting infrastructure exists:
- `Formatter` with `Bold()`, `Success()`, `Muted()`, `Primary()`, `Box()`, `BulletList()`
- `TerminalCapabilities` detection (TTY, ANSI, Unicode support)
- `ANSICodec` for safe escape sequence management
- Non-TTY detection already works via `TerminalInfo.IsTTY()`

**Finding**: The display package has all the formatting primitives needed. `OutcomeSummary` rendering can leverage `Formatter` directly for styled output, with plain text fallback via the existing ANSI detection.

## Decisions

### D-001: New DeliverableType Constants

**Decision**: Add `TypeBranch DeliverableType = "branch"` and `TypeIssue DeliverableType = "issue"` to `internal/deliverable/types.go`
**Rationale**: Explicit type constants enable `GetByType()` queries and are consistent with the existing `TypePR` pattern. Using metadata-only conventions would require callers to know magic metadata keys.
**Alternatives Rejected**:
- Overloading `TypeURL` for branches/issues → breaks type-based querying, loses semantic meaning
- Using `TypeOther` with metadata → same problem, no queryability

### D-002: PipelineOutcome Struct Location

**Decision**: Define `PipelineOutcome` in a new file `internal/display/outcome.go` within the display package
**Rationale**: `PipelineOutcome` is a read-only summary struct used exclusively for rendering. The display package already owns `PipelineContext`, `StepProgress`, and other rendering-focused types. Construction logic lives in `run.go`, which already imports `display`.
**Alternatives Rejected**:
- Put in `internal/deliverable/` → conflates tracking with presentation
- Put in `cmd/wave/commands/` → not reusable by other commands or tests

### D-003: Outcome Collection Strategy

**Decision**: Branch deliverables are recorded in `executor.go` when worktrees are created (line ~800). Push status is recorded by updating the branch deliverable's `Metadata["pushed"]` after publish steps complete. PR/issue deliverables use existing `trackCommonDeliverables()` enriched with issue detection.
**Rationale**: Uses the existing `deliverable.Tracker` infrastructure. No parallel tracking system needed. The tracker already handles concurrency, deduplication, and per-step attribution.
**Alternatives Rejected**:
- New `OutcomeTracker` type → unnecessary duplication of tracker semantics
- Post-hoc scanning of git state → fragile, depends on git being available post-execution

### D-004: Verbose vs Default Mode

**Decision**: Reuse the existing `--verbose` flag (`OutputConfig.Verbose`). Default mode shows structured outcome summary + count-based artifact/contract summary. Verbose mode shows full deliverable list after the outcome summary.
**Rationale**: FR-010 explicitly requires using the existing `--verbose` flag. No new flags needed. The `OutputConfig` struct already carries `Verbose bool`.
**Alternatives Rejected**:
- New `--summary` flag → violates FR-010
- Separate `--outcomes` flag → unnecessary complexity

### D-005: JSON Output Extension

**Decision**: Add an `Outcomes` field to `event.Event` (type `*OutcomesJSON`), populated only in the final `completed` event for the pipeline. This keeps the NDJSON stream backward-compatible (field is `omitempty`).
**Rationale**: Extends the existing event model without breaking consumers. The `omitempty` tag means old consumers ignore it. SC-004 requires a well-defined `outcomes` object in JSON output.
**Alternatives Rejected**:
- Separate outcomes event → consumers must correlate two events
- Post-stream summary object → breaks NDJSON format (each line must be valid JSON)

### D-006: Deliverable Priority Ordering

**Decision**: Outcome-worthy types have priority: `TypePR` > `TypeIssue` > `TypeBranch` > `TypeDeployment`. Detail-level types: `TypeFile`, `TypeLog`, `TypeContract`, `TypeArtifact`. Default mode shows the top 5 outcome-worthy items. Within same type, newest first.
**Rationale**: Spec clarification C-002 defines N=5 and the priority ordering. This is implemented as a sort function in the display layer, not as a tracker concern.
**Alternatives Rejected**:
- Metadata-based priority → requires callers to set priority values manually
- Step-order based priority → doesn't reflect deliverable importance

### D-007: Next Steps Generation

**Decision**: `OutcomeSummary` generates contextual next steps from `PipelineOutcome` data. Rules: if PR exists → "Review the PR at <url>"; if branch pushed → "View changes at <remote_ref>"; if worktree workspace → "Inspect workspace at <path>". Suppressed in quiet mode.
**Rationale**: FR-011 requires contextual next steps. The rules are simple pattern matching on outcome data, not requiring external state.
**Alternatives Rejected**:
- Configurable next-step rules → over-engineering for prototype phase
- Step-defined next steps in pipeline YAML → couples UX to pipeline definitions

## Technology Choices

| Component | Choice | Justification |
|-----------|--------|---------------|
| Language | Go 1.25+ | Existing project |
| New dependencies | None | All display primitives exist in `internal/display/` |
| Storage | None (in-memory struct) | `PipelineOutcome` is not persisted |
| Testing | `go test` with table-driven tests | Existing project convention |
| Platform | Cross-platform (TTY + non-TTY) | Existing `TerminalInfo` handles detection |

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Publish step output not reliably parsed for PR URLs | Medium | Medium | Enhance `trackCommonDeliverables` to also scan adapter output for `github.com` PR/issue URL patterns |
| Branch deliverable duplicated across shared worktree steps | Low | Low | Tracker dedup already checks path+stepID; branch deliverable keyed by branch name |
| JSON consumers break on new `outcomes` field | Low | Low | Field is `omitempty` — absent when not set |
| Non-TTY output still contains ANSI codes in outcomes section | Low | Medium | All new rendering uses `Formatter` which checks `DetectANSISupport()` |
