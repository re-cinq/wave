# Feature Specification: Pipeline Output UX — Surface Key Outcomes

**Feature Branch**: `120-pipeline-output-ux`
**Created**: 2026-02-20
**Status**: Draft
**Input**: https://github.com/re-cinq/wave/issues/120

## User Scenarios & Testing _(mandatory)_

### User Story 1 — Scannable Completion Summary (Priority: P1)

As a developer running a Wave pipeline, I want the completion output to prominently display the key outcomes (branch created, push status, issues/PRs created, reports generated) so that I can immediately understand what happened without scrolling through verbose artifact and contract details.

**Why this priority**: This is the core problem described in the issue. The current "wall of text" buries the most actionable information. Surfacing key outcomes is the single most impactful change.

**Independent Test**: Can be tested by running any pipeline to completion and verifying that the final output section shows a structured summary with key outcomes listed prominently, above any artifact/contract details.

**Acceptance Scenarios**:

1. **Given** a pipeline completes successfully and created a git branch, **When** the output is rendered, **Then** the branch name is displayed in a dedicated "Outcomes" section near the top of the completion output.
2. **Given** a pipeline completes and pushed changes to a remote, **When** the output is rendered, **Then** the push status (pushed/not pushed) and remote ref are displayed in the outcomes section.
3. **Given** a pipeline creates a GitHub issue or pull request, **When** the output is rendered, **Then** the issue/PR URL is displayed prominently with a descriptive label (e.g., "Pull Request" or "Issue").
4. **Given** a pipeline generates report files or key deliverables, **When** the output is rendered, **Then** report paths are listed in the outcomes section with human-readable labels.
5. **Given** a pipeline completes with no special outcomes (no branch, no PR, no reports), **When** the output is rendered, **Then** only the success status, duration, and token usage are shown without empty outcome sections.

---

### User Story 2 — Artifact and Contract Details Relegated to Secondary Display (Priority: P2)

As a developer reviewing pipeline output, I want artifact and contract validation details to be secondary (collapsed, summarized, or available on demand) so that the key outcomes are not obscured by verbose technical details.

**Why this priority**: Reducing noise is essential for making the outcome summary useful. Without demoting artifact/contract verbosity, the new outcomes section would just add more text rather than improve readability.

**Independent Test**: Can be tested by running a pipeline that produces multiple artifacts and contracts, and verifying that default output shows only a count/summary line for artifacts and contracts, with full details accessible via `--verbose`.

**Acceptance Scenarios**:

1. **Given** a pipeline completes with 5 artifact files, **When** the default output is rendered, **Then** artifacts are summarized as a single line (e.g., "5 artifacts produced") rather than listing each path individually.
2. **Given** a pipeline completes with all contracts passing, **When** the default output is rendered, **Then** contract validation is summarized as a single status line (e.g., "3/3 contracts passed") rather than showing each validation detail.
3. **Given** the user runs the pipeline with `--verbose`, **When** the output is rendered, **Then** full artifact paths and contract validation details are displayed alongside the summary.
4. **Given** a contract fails validation, **When** the default output is rendered, **Then** the failing contract is shown prominently (not hidden) with the failure reason, regardless of verbosity level.

---

### User Story 3 — Structured JSON Output for Automation (Priority: P3)

As a CI system or script consuming Wave output, I want the JSON output format (`--output json`) to include a structured completion summary with key outcomes so that I can programmatically extract branch names, PR URLs, and artifact paths without parsing human-readable text.

**Why this priority**: Machine-readable output is important for pipeline automation and integration, but human UX is a higher priority since that is the primary complaint in the issue.

**Independent Test**: Can be tested by running a pipeline with `--output json` and verifying the final NDJSON event contains a structured `outcomes` object with branch, push, PR/issue, and deliverable fields.

**Acceptance Scenarios**:

1. **Given** a pipeline completes with `--output json`, **When** the final event is emitted, **Then** it contains an `outcomes` field with `branch`, `pushed`, `pull_request`, `issues`, and `deliverables` sub-fields.
2. **Given** a pipeline creates a PR and pushes a branch, **When** the JSON output is parsed, **Then** the `outcomes.branch` field contains the branch name and `outcomes.pushed` is `true`.
3. **Given** no GitHub issue was created, **When** the JSON output is parsed, **Then** the `outcomes.issues` field is an empty array (not null or missing).

---

### User Story 4 — Contextual Next Steps (Priority: P3)

As a developer who just completed a pipeline, I want the output to suggest relevant next steps (e.g., "Review the pull request", "Run tests", "Clean up workspace") so that I know what actions to take after the pipeline finishes.

**Why this priority**: This is a usability enhancement that builds on the outcomes summary. It adds value but is not required to solve the core "wall of text" problem.

**Independent Test**: Can be tested by running a pipeline that creates a PR and verifying the output includes a "Next Steps" section with actionable suggestions.

**Acceptance Scenarios**:

1. **Given** a pipeline completes and created a pull request, **When** the output is rendered, **Then** a "Next Steps" section suggests reviewing the PR with its URL.
2. **Given** a pipeline completes in a worktree workspace, **When** the output is rendered, **Then** a "Next Steps" section includes the workspace path for inspection.
3. **Given** the user runs with `--output quiet` mode, **When** the output is rendered, **Then** the next steps section is not shown.

---

### Edge Cases

- What happens when a pipeline step produces deliverables but the step ultimately fails? Failed step deliverables should still be shown but clearly marked as from a failed step.
- How does the output behave when the terminal width is very narrow (< 60 columns)? The output should degrade gracefully, wrapping or truncating long paths rather than producing garbled formatting.
- What happens when a pipeline produces a very large number of deliverables (50+)? Default mode should show a count with the top 5 most relevant items (prioritized by type: PRs > issues > branches > reports > files); verbose mode should show all.
- How does the output handle deliverables with very long file paths or URLs? Paths should be truncated or abbreviated (e.g., relative to the workspace root) in default mode, shown in full in verbose mode.
- What happens when the terminal is not a TTY (piped to a file or another program)? Output should use plain text without colors or Unicode decorations, matching existing TTY detection behavior.
- What happens when a pipeline completes but the git push operation failed? The push failure should be surfaced as a warning in the outcomes section, not silently hidden.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST display a structured "Outcomes" section after pipeline completion that lists key results (branch name, push status, issue/PR URLs, report paths) prominently before any artifact or contract details.
- **FR-002**: System MUST summarize artifact details as a count and brief description in default output mode, showing full paths only in verbose mode.
- **FR-003**: System MUST summarize contract validation results as a pass/fail count in default output mode, showing full validation details only in verbose mode.
- **FR-004**: System MUST prominently display any contract validation failures in all output modes, including the failure reason and the affected step.
- **FR-005**: System MUST track and surface git branch creation and push status as first-class pipeline outcomes, not just as artifacts.
- **FR-006**: System MUST track and surface GitHub issue and pull request creation as first-class pipeline outcomes with clickable URLs.
- **FR-007**: System MUST include structured outcome data in the JSON output format (`--output json`) as part of the final completion event.
- **FR-008**: System MUST suppress empty outcome categories (e.g., if no PR was created, the "Pull Request" line should not appear rather than showing "None").
- **FR-009**: System MUST degrade gracefully in non-TTY environments, producing plain text without ANSI escape codes or Unicode decorations for the outcomes section.
- **FR-010**: System MUST support the existing `--verbose` flag to toggle between summary and detailed artifact/contract display without requiring new flags.
- **FR-011**: System MUST display a contextual "Next Steps" section after pipeline completion when actionable follow-ups are available (e.g., PR review URL, workspace path).
- **FR-012**: System MUST respect the `--output quiet` format by suppressing the outcomes summary and next steps sections, showing only the final pass/fail status. (Note: quiet is an output format value, not a separate flag — i.e., `--output quiet` or `-o quiet`.)

### Key Entities

- **PipelineOutcome**: Represents the structured summary of a pipeline execution's key results. Contains branch information, push status, issue/PR references, report paths, and overall status. Constructed in `cmd/wave/commands/run.go` after execution completes, by querying the executor's deliverable tracker and worktree metadata. This is a **read-only summary struct** — not persisted in the database.
- **OutcomeSummary**: The formatted, human-readable rendering of a PipelineOutcome. Lives in the `internal/display/` package. Adapts its presentation based on output format (text/json), verbosity level, and terminal capabilities.
- **Deliverable** (existing entity, extended): Currently tracks artifacts by type via `DeliverableType`. Extended with two new type constants: `TypeBranch` (for git branch creation/push tracking) and `TypeIssue` (for GitHub issue tracking, complementing the existing `TypePR`). The distinction between "outcome-worthy" deliverables (PRs, issues, branches, reports) and "detail-level" deliverables (intermediate artifacts, contract outputs) is determined by type: `TypePR`, `TypeIssue`, `TypeBranch`, `TypeDeployment` are outcome-worthy; `TypeFile`, `TypeLog`, `TypeContract`, `TypeArtifact` are detail-level.

### Architectural Boundaries

- **Outcome collection** happens during pipeline execution via the existing `deliverable.Tracker`. Branch/worktree info is recorded as `TypeBranch` deliverables when worktree workspaces are created in `executor.go`. Push status is recorded via metadata on the branch deliverable (`metadata["pushed"]`, `metadata["remote_ref"]`).
- **Outcome aggregation** happens in the run command (`cmd/wave/commands/run.go`) after execution completes, by constructing a `PipelineOutcome` from the deliverable tracker's contents.
- **Outcome rendering** happens in `internal/display/` via `OutcomeSummary`, which formats the `PipelineOutcome` for the appropriate output mode.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: After pipeline completion, a user can identify the branch name, push status, and any PR/issue URLs within the first 5 lines of the completion output (excluding the progress display).
- **SC-002**: Default output for a pipeline producing 10+ artifacts is reduced to no more than 3 lines of artifact summary (count + key items), compared to the current behavior of listing all artifact paths.
- **SC-003**: Contract validation results in default mode occupy no more than 1 line per step when all contracts pass.
- **SC-004**: The JSON output format includes a well-defined `outcomes` object that can be parsed by downstream tools without regex or string manipulation.
- **SC-005**: All existing pipeline output tests continue to pass, with new tests added for the outcomes summary formatting.
- **SC-006**: Output in non-TTY environments contains no ANSI escape codes or Unicode decorations in the outcomes section.

## Clarifications

The following ambiguities were identified during specification review and resolved based on codebase analysis:

### C-001: New DeliverableType constants for branches and issues

**Question**: The existing `DeliverableType` enum in `internal/deliverable/types.go` has `TypePR`, `TypeURL`, `TypeFile`, etc., but no constants for git branches or GitHub issues. How should branch creation, push status, and issue tracking be represented?

**Resolution**: Add two new `DeliverableType` constants: `TypeBranch DeliverableType = "branch"` and `TypeIssue DeliverableType = "issue"`. Branch deliverables use the `Metadata` map for push-related fields (`metadata["pushed"] = true/false`, `metadata["remote_ref"] = "origin/branch-name"`, `metadata["push_error"] = "error message"`). This follows the existing pattern where `TypePR` already exists as a first-class type and the `Metadata` field is available for type-specific data.

**Rationale**: Adding explicit types (rather than overloading `TypeURL` or using metadata-only conventions) keeps the type system expressive and allows `GetByType()` queries to work naturally. The existing `NewPRDeliverable` constructor pattern is extended with `NewBranchDeliverable` and `NewIssueDeliverable`.

### C-002: Default cap (N) for large deliverable lists

**Question**: Edge case 3 says "Default mode should show a count with the top N most relevant items" but N was undefined, as was the relevance ordering.

**Resolution**: N = 5. Relevance ordering by deliverable type priority: PRs > issues > branches > deployments > reports (files with report-like names) > other files. Within the same type, items are ordered by creation time (newest first). This ensures the most actionable items appear in the default view.

**Rationale**: 5 items fits comfortably in a terminal summary without scroll. The priority ordering matches what developers typically care about most after a pipeline run. This is consistent with common CLI tools (e.g., `git log --oneline -5`).

### C-003: Architectural boundary for PipelineOutcome construction

**Question**: The spec introduced `PipelineOutcome` and `OutcomeSummary` as key entities but didn't specify where in the architecture each is constructed — executor, display, or run command.

**Resolution**: `PipelineOutcome` is constructed in `cmd/wave/commands/run.go` after `executor.Execute()` returns, by reading from the executor's `deliverable.Tracker`. `OutcomeSummary` lives in `internal/display/` and handles formatting. The executor itself only records deliverables — it does not construct or render summaries. This aligns with the existing pattern where `run.go` already calls `executor.GetDeliverables()` and `executor.GetTotalTokens()` post-execution.

**Rationale**: Keeping outcome aggregation in the command layer (not the executor) maintains separation of concerns. The executor owns data collection; the command orchestrates the user-facing output. This also avoids coupling the executor to display concerns.

### C-004: How branch/push outcomes are detected during execution

**Question**: Branch creation and push happen in the worktree workspace setup code (`executor.go` lines 753-802), but are not currently recorded as deliverables. How should the executor surface these?

**Resolution**: When the executor creates a worktree workspace (workspace type "worktree"), it records a `TypeBranch` deliverable via the existing `deliverableTracker.Add()` mechanism immediately after successful worktree creation. The branch name is the deliverable's `Name`, the worktree path is the `Path`, and push status is tracked in `Metadata`. Push events (if any occur in publish steps) update the branch deliverable's metadata. This requires instrumenting the worktree creation path in `executeStep` and any publish step logic.

**Rationale**: Using the existing `deliverable.Tracker` infrastructure avoids introducing a parallel tracking mechanism. The tracker already handles concurrent access, deduplication, and per-step attribution.

### C-005: Quiet mode is an output format, not a separate flag

**Question**: The spec referenced `--quiet` flag in FR-012 and US4-AC3, but the codebase implements quiet as an output format value (`--output quiet` / `-o quiet`), not as a standalone `--quiet` flag.

**Resolution**: Updated FR-012 and US4 acceptance criteria to use `--output quiet` terminology, consistent with the existing `OutputConfig.Format` values (`auto`, `json`, `text`, `quiet`) defined in `cmd/wave/commands/output.go`. No new flags are introduced.

**Rationale**: Following the existing flag convention avoids breaking the CLI interface or adding redundant flags. The `OutputFormatQuiet` constant already exists and is handled in `CreateEmitter()`.
