# Requirements Quality Review: TUI Pipeline Launch Flow

**Feature**: #256 — TUI Pipeline Launch Flow
**Date**: 2026-03-06
**Spec**: `specs/256-tui-pipeline-launch/spec.md`

## Completeness

- [ ] CHK001 - Are all user-facing states of the right pane fully enumerated with entry/exit conditions? The spec lists 8 states (`stateEmpty` through `stateError`) but only defines transitions for a subset (e.g., `stateRunningInfo` entry conditions are not specified for TUI-launched pipelines). [Completeness]
- [ ] CHK002 - Does the spec define what happens when a pipeline finishes while the user is viewing its argument form for a second launch of the same pipeline? [Completeness]
- [ ] CHK003 - Are accessibility requirements specified for the argument form (e.g., screen reader announcements, focus indicators, high-contrast theme support)? [Completeness]
- [ ] CHK004 - Does the spec define the visual representation of the "launching" state (duration, animation, fallback for non-Unicode terminals)? [Completeness]
- [ ] CHK005 - Are requirements defined for the `PipelineLaunchedMsg` synthetic entry format — specifically what fields (status indicator, elapsed time format, name truncation) the Running section displays before the first state store refresh? [Completeness]
- [ ] CHK006 - Does the spec address what keyboard shortcuts are available in each right-pane state? Only `stateConfiguring` (Tab, Enter, Esc) and default (Esc) are described; `stateError`, `stateLaunching`, and `stateRunningInfo` key handling is not specified. [Completeness]
- [ ] CHK007 - Are error message content requirements specified beyond "actionable error message"? (e.g., error categorization, suggested remediation, truncation for long errors) [Completeness]
- [ ] CHK008 - Does the spec define the maximum number of concurrent TUI-launched pipelines, or is it explicitly unbounded? [Completeness]
- [ ] CHK009 - Are requirements specified for what happens to the cancel function map if the TUI process crashes (SIGKILL) without graceful shutdown? [Completeness]

## Clarity

- [ ] CHK010 - Is the relationship between `LaunchConfig.DryRun` and `LaunchConfig.Flags` containing `--dry-run` unambiguous? FR-015 uses "when the `--dry-run` flag is selected" while `LaunchConfig` has a separate `DryRun bool` field. [Clarity]
- [ ] CHK011 - Is the scope of "fresh form" in FR-014 clearly defined — does it mean zero-valued fields, or fields pre-populated with pipeline-specific defaults (e.g., `InputExample` as placeholder vs. pre-filled value)? [Clarity]
- [ ] CHK012 - Is the distinction between `PipelineLaunchedMsg` (executor started) and `PipelineLaunchResultMsg` (executor finished) clear about which carries the cancel function? The spec says `PipelineLaunchedMsg` carries `CancelFunc` but FR-011 says the map is "keyed by run ID" — it's unclear when the map entry is created (at `Launch()` call or upon `PipelineLaunchedMsg`). [Clarity]
- [ ] CHK013 - Is "immediately" in FR-007 ("MUST immediately insert the pipeline at the top") defined in terms of render cycles, or is it qualitative? [Clarity]
- [ ] CHK014 - Does "the CLI path" in FR-006 unambiguously identify which code path is being referenced (which function in `runRun()`, which options are "applicable")? [Clarity]

## Consistency

- [ ] CHK015 - Is the `PipelineLaunchedMsg` struct definition consistent between the spec (carrying `CancelFunc`) and the plan/tasks (not including `CancelFunc` in the struct)? The plan Phase 3 stores cancel in the map inside `Launch()` before emitting the message, but the spec's entity definition includes `CancelFunc context.CancelFunc` in the message. [Consistency]
- [ ] CHK016 - Are the flag names consistent between the argument form (`DefaultFlags()` names) and the `LaunchConfig.Flags []string` field? Does `DefaultFlags()` return display labels or CLI flag strings? [Consistency]
- [ ] CHK017 - Is focus management consistent between US-1 (focus returns to left after launch) and the `stateError` path (FR-013 says focus returns to left, but the error is displayed in the right pane — who renders it if focus is left)? [Consistency]
- [ ] CHK018 - Does the plan's `ConfigureFormMsg` (not in spec requirements) align with FR-001's trigger ("when the user presses Enter")? The spec says Enter triggers the transition, but the plan introduces an intermediate message not in the requirements. [Consistency]
- [ ] CHK019 - Is the `q`-quit guard condition consistent between spec (FR-019: `m.content.focus == FocusPaneLeft`) and the existing filter guard (`!m.content.list.filtering`)? Are both conditions required simultaneously? [Consistency]

## Coverage

- [ ] CHK020 - Do acceptance criteria cover the scenario where the state store becomes unavailable mid-launch (after `CreateRun` succeeds but before executor completes)? [Coverage]
- [ ] CHK021 - Do edge cases cover terminal resize during the `stateLaunching` brief indicator? [Coverage]
- [ ] CHK022 - Are requirements defined for the interaction between the existing 5-second `PipelineRefreshTickMsg` and the synthetic running entry — specifically, does the refresh replace, duplicate, or merge with the synthetic entry? [Coverage]
- [ ] CHK023 - Do user stories cover the scenario of launching a pipeline when `LaunchDependencies` has nil/zero-value fields (e.g., no manifest loaded)? [Coverage]
- [ ] CHK024 - Are test requirements specified for goroutine leak prevention (ensuring executor goroutines terminate after cancel)? [Coverage]
- [ ] CHK025 - Do success criteria address the performance requirement that form creation and display should not introduce perceptible latency (e.g., < 100ms)? [Coverage]
- [ ] CHK026 - Does the spec cover what happens when the user rapidly presses Enter multiple times on an available pipeline (potential race condition creating multiple forms or launches)? [Coverage]
