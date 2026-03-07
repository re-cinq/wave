# TUI Interaction Quality Checklist: Pipeline Composition UI (#261)

**Feature**: `261-tui-compose-ui` | **Date**: 2026-03-07
**Scope**: Quality validation of compose mode interaction patterns, state management, and keyboard navigation requirements

## State Machine Completeness

- [ ] CHK201 - Are all valid state transitions into and out of compose mode enumerated (entry via `s`, exit via `Esc`, exit via `Enter` start, exit via single-pipeline delegation)? [Completeness]
- [ ] CHK202 - Is the nested Esc behavior fully specified — first Esc returns from detail to list, second Esc exits compose mode, and this two-level exit is documented? [Clarity]
- [ ] CHK203 - Is the compose mode interaction with the picker sub-state fully defined — what messages are blocked while the picker is active? [Completeness]
- [ ] CHK204 - Is the behavior defined for when compose mode's `ComposeStartMsg` is emitted but pipeline loading fails during execution setup? [Coverage]
- [ ] CHK205 - Is the behavior specified for canceling compose mode when an async operation (e.g., pipeline loading for the picker) is in progress? [Coverage]

## Keyboard Navigation

- [ ] CHK206 - Are cursor boundary behaviors defined — what happens when pressing ↑ on the first item or ↓ on the last item in the sequence list? [Completeness]
- [ ] CHK207 - Are shift+↑/shift+↓ boundary behaviors defined — what happens when trying to reorder the first item up or the last item down? [Completeness]
- [ ] CHK208 - Is the `x` (remove) behavior defined for cursor adjustment — when the last item is removed, does the cursor move up? [Completeness]
- [ ] CHK209 - Is the key binding `a` (add pipeline) conflict-free with other Bubble Tea conventions and existing TUI keybindings? [Consistency]
- [ ] CHK210 - Are the compose-mode keybindings documented in the user-facing help or discoverable via the status bar (FR-011)? [Completeness]

## Pane Layout & Focus

- [ ] CHK211 - Is the left/right pane size allocation during compose mode specified — does it follow the existing proportional split or use a different ratio? [Completeness]
- [ ] CHK212 - Is the focus indicator styling (which pane is focused) consistent with the existing TUI focus conventions? [Consistency]
- [ ] CHK213 - Is the behavior specified for `Tab` key during compose mode — explicitly blocked or silently consumed? [Clarity]
- [ ] CHK214 - Is the right pane content defined for when only one pipeline is in the sequence (no boundaries to visualize)? [Coverage]

## CLI Parity

- [ ] CHK215 - Are the `wave compose` exit codes specified and consistent with other Wave CLI commands? [Consistency]
- [ ] CHK216 - Is the `wave compose` error output format (stderr vs stdout) specified for validation failures? [Completeness]
- [ ] CHK217 - Is the `--validate-only` flag's output format defined — human-readable, structured JSON, or both (per `--json` flag from #260)? [Clarity]
- [ ] CHK218 - Is the interaction between `wave compose` and `--no-tui` / `NO_COLOR` (from #260) documented? [Consistency]
