# Research: Run Options Parity (Phase 0)

**Feature**: #717 — Run Options Parity Across All Surfaces  
**Date**: 2026-04-11

## Unknowns Extracted from Spec

The spec has zero `[NEEDS CLARIFICATION]` markers after the clarify step. All five
ambiguities (C-001 through C-005) were resolved. No open unknowns remain.

## Technology Decisions

### TD-001: Canonical RunOptions Location

**Decision**: `cmd/wave/commands/run.go:36-61` (`commands.RunOptions`) remains the
authoritative definition. All other surfaces map to this struct.

**Rationale**: The CLI already defines every Tier 1–4 field. Other surfaces (API,
TUI, WebUI) currently carry a subset. Extending those subsets to match the CLI
avoids introducing a new shared type while keeping the dependency direction
clear (surfaces → CLI types via subprocess flags).

**Alternatives Rejected**:
- Shared `internal/options/run_options.go` type — adds coupling; the WebUI server
  and TUI don't import `cmd/wave/commands` and shouldn't. Each surface has its own
  request/config type that maps to CLI flags in the subprocess call.

### TD-002: WebUI Inline Form vs Modal

**Decision**: Replace the `showQuickStart()` modal (`<dialog>`) on
`pipeline_detail.html` with an inline tiered form rendered directly in the page.

**Rationale**: FR-001 requires inline form. The current modal is in
`templates/pipelines.html` (shared quickstart modal) and duplicated in
`pipeline_detail.html:161`. The pipeline detail page gets a dedicated inline
form; the pipelines list page retains its modal for quick launches.

**Alternatives Rejected**:
- Enhance existing modal with collapsible sections — still a modal, violates FR-001.
- Shared form component included via Go template partials — adds template
  complexity for a form that's only inline on one page.

### TD-003: API Request Type Extension Strategy

**Decision**: Extend `StartPipelineRequest` and `SubmitRunRequest` with missing
Tier 1–4 fields. Create new `StartIssueRequest` and `StartPRRequest` types
(currently inline anonymous structs) with Tier 1–3 fields.

**Rationale**: The current `handleAPIStartFromIssue` uses an anonymous struct
with only `IssueURL` and `PipelineName`. No adapter/model/timeout overrides are
possible. The spec requires Tier 1–3 options on issue/PR surfaces.

**Current state**:
- `StartPipelineRequest`: has Input, Model, Adapter, DryRun, Timeout, Steps, Exclude
- `SubmitRunRequest`: same fields plus Pipeline
- Issue handler: anonymous `{IssueURL, PipelineName}` — no overrides
- PR handler: no start-from-PR handler exists; only review (`POST /api/prs/{number}/review`)

### TD-004: WebUI RunOptions Wiring

**Decision**: Extend `webui.RunOptions` struct (currently: Model, Adapter, DryRun,
Timeout, Steps, Exclude) with the missing fields: FromStep, Force, Detach,
Continuous, Source, MaxIterations, Delay, OnFailure, Mock, PreserveWorkspace,
AutoApprove, NoRetro, ForceModel.

**Rationale**: `spawnDetachedRun()` already wires Model, Adapter, Timeout, Steps,
Exclude to subprocess flags. Adding new fields follows the same pattern — each
field maps to a `--flag` appended to the args slice.

### TD-005: TUI LaunchConfig Refactoring

**Decision**: Add typed fields to `LaunchConfig` (adapter, timeout, from-step,
steps, exclude, detach) instead of relying on `Flags []string`. The launcher
already maps `ModelOverride` as a typed field; extend this pattern.

**Rationale**: Typed fields enable form validation before launch. The current
`Flags []string` approach passes raw flag strings to the subprocess, which works
but prevents the TUI from validating combinations (e.g., continuous + from-step
mutual exclusion) before spawning.

**Alternatives Rejected**:
- Keep everything as `Flags []string` — no pre-launch validation possible,
  errors only surface after subprocess starts.

### TD-006: CLI Help Grouping

**Decision**: Use Cobra's `MarkFlagGroup` or custom `UsageFunc` to group flags
into four sections: Essential (Tier 1), Execution (Tier 2), Continuous (Tier 3),
Dev/Debug (Tier 4).

**Rationale**: Cobra supports custom usage templates. The `wave run` command
registers 20+ flags on a flat list. Grouping improves discoverability.

**Current state**: All flags are registered via `cmd.Flags().*Var()` calls
at `run.go:163-186`. Cobra supports `FlagGroup` annotations since v1.7.

### TD-007: Start-from-PR Handler

**Decision**: Create `POST /api/prs/start` endpoint mirroring the issue handler,
with a new `StartPRRequest` type carrying Tier 1–3 fields.

**Rationale**: No PR start handler exists. The PR detail page
(`templates/pr_detail.html`) has no "Run Pipeline" button. The issue detail page
has one (`templates/issue_detail.html:16-47`). Parity requires adding both the
handler and the UI.

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| WebUI inline form breaks existing pipeline detail layout | Medium | Feature-flag via CSS class; incremental rollout |
| `spawnDetachedRun` flag list grows unwieldy | Low | Already 12 flags; adding 8 more is linear growth |
| TUI form field count overwhelms small terminals | Medium | Use collapsible/scrollable form groups in huh |
| Mutual exclusion validation duplicated across surfaces | Low | CLI already validates; API/WebUI/TUI add pre-submit checks |
