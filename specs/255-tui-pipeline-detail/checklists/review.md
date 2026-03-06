# Requirements Quality Review: TUI Pipeline Detail Right Pane

**Feature**: 255-tui-pipeline-detail | **Date**: 2026-03-06

## Completeness

- [ ] CHK001 - Are all fields listed for the available detail view (FR-003: name, description, category, step count, steps, inputs, outputs, dependencies) traceable to a concrete data source in the manifest YAML schema? [Completeness]
- [ ] CHK002 - Does FR-004 define which timestamp format is used for start time and end time (RFC3339, relative "2m ago", locale-specific)? [Completeness]
- [ ] CHK003 - Does the spec define the visual format of the step results table (column alignment, separator characters, header row) or is this left to implementation? [Completeness]
- [ ] CHK004 - Is the "Loading..." indicator in FR-016 defined with enough precision — does it specify a spinner, static text, or timeout behavior if loading takes too long? [Completeness]
- [ ] CHK005 - Does FR-006 define whether action hints are always visible or only visible when the user scrolls to the bottom of the detail content? [Completeness]
- [ ] CHK006 - Does the spec define the visual appearance of the focus indicator on the right pane (FR-010) — border color, title highlight, or other treatment? [Completeness]
- [ ] CHK007 - Are the available pipeline detail sections ordered? Does the spec define the rendering order of description, steps, inputs, outputs, and dependencies? [Completeness]
- [ ] CHK008 - Does the spec define what "category (if set)" means — is the category field optional in the manifest, and what does the detail pane show when it's absent? [Completeness]

## Clarity

- [ ] CHK009 - Is the distinction between "preview on cursor move" (C1) and "focus on Enter" precisely defined? Does preview imply the viewport is read-only (no keyboard scroll) until Enter is pressed? [Clarity]
- [ ] CHK010 - Does FR-008 clearly define what "NOT running or section header" means? Is the full set of focusable item kinds enumerated (available + finished only)? [Clarity]
- [ ] CHK011 - Does FR-019 clearly define backward compatibility? Will consumers of `PipelineSelectedMsg` that don't use `Name`/`Kind` continue to work with zero values? [Clarity]
- [ ] CHK012 - Is "dimmed selection highlight" (FR-010) defined concretely enough to test? Does "dimmed" mean a specific opacity, color change, or style attribute? [Clarity]
- [ ] CHK013 - Does the spec define whether the right pane border is always visible or only when focused? Is the unfocused right pane visually separated from the left pane? [Clarity]
- [ ] CHK014 - Is the term "step count" in FR-003 redundant with "list of steps"? Does the spec clarify whether both are displayed or step count is derived from the list? [Clarity]
- [ ] CHK015 - Does FR-020 define "elapsed time" for running pipelines precisely — is it wall-clock time since start, and does it update live or show a static snapshot? [Clarity]

## Consistency

- [ ] CHK016 - Are the action hints in FR-006 (`[Enter] Open chat`, `[b] Checkout branch`, `[d] View diff`, `[Esc] Back`) consistent with the key bindings defined elsewhere? Does Enter conflict with the focus transition from US-3? [Consistency]
- [ ] CHK017 - Is the `PipelineSelectedMsg` extension (FR-019) consistent with how the message is currently consumed by HeaderModel? Does adding `Kind` introduce any type-assertion or switch-case gaps in existing handlers? [Consistency]
- [ ] CHK018 - Does the `FocusChangedMsg` pattern (C5) follow the same conventions as `RunningCountMsg` and `PipelineSelectedMsg`? Is the message routed through the same parent-to-child forwarding path? [Consistency]
- [ ] CHK019 - Are the status bar hints in C5 ("↑↓: navigate  Enter: view  /: filter  q: quit") consistent with the current static hints in the existing StatusBarModel? Is `ctrl+c: exit` included or omitted? [Consistency]
- [ ] CHK020 - Does the running pipeline informational message (FR-020) reference issue #258 for future real-time progress? Is this consistent with the epic roadmap in #251? [Consistency]
- [ ] CHK021 - Are US-3 acceptance scenario 5 (Enter on section header collapses) and FR-011 fully aligned? Both must reference the same existing collapse behavior without introducing ambiguity about which component handles the key. [Consistency]

## Coverage

- [ ] CHK022 - Does the spec address what happens when a pipeline YAML file is malformed or missing required fields? How does the available detail view degrade? [Coverage]
- [ ] CHK023 - Is there a requirement for how the detail pane behaves when the user rapidly switches between pipeline items (cursor move debouncing for async fetches)? [Coverage]
- [ ] CHK024 - Does the spec address the transition when the user selects a different item while the right pane is focused? Does focus stay on the right pane or return to left? [Coverage]
- [ ] CHK025 - Is there a defined behavior when the finished pipeline's run record is missing from the state database (e.g., DB migration, deleted state)? [Coverage]
- [ ] CHK026 - Does the spec define accessibility considerations beyond NO_COLOR (FR-018)? Are there screen reader annotations or keyboard-only navigation guarantees? [Coverage]
- [ ] CHK027 - Is there a requirement addressing the initial render — when the TUI first loads, is the right pane placeholder shown immediately or after a brief layout calculation? [Coverage]
- [ ] CHK028 - Does the spec define what happens when the left pane is empty (no pipelines discovered, no finished runs)? Is the right pane permanently in placeholder state? [Coverage]
- [ ] CHK029 - Is there a requirement for the detail pane's behavior when the state database is locked by another Wave process (e.g., a running pipeline writing to it)? [Coverage]
