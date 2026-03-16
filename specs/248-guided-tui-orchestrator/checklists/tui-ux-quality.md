# TUI/UX Quality Checklist: Guided TUI Orchestrator

**Feature**: `248-guided-tui-orchestrator`
**Generated**: 2026-03-16
**Scope**: TUI-specific quality dimensions for interaction design and state management

---

## State Machine Integrity

- [ ] CHK101 - Are all valid state transitions enumerated, and are invalid transitions (e.g., Health→Attached, Proposals→HealthPhase) explicitly prohibited? [State Machine]
- [ ] CHK102 - Is the state machine deterministic — can the same input (key press + current state) ever produce different outcomes? [State Machine]
- [ ] CHK103 - Are re-entrant scenarios addressed (e.g., returning to proposals after fleet, does state reset or preserve)? [State Machine]
- [ ] CHK104 - Is the race condition between auto-transition timer and user Tab-skip defined (what wins if both fire simultaneously)? [State Machine]
- [ ] CHK105 - Does the spec define what happens to the transition timer if the user navigates away from health via a number key? [State Machine]

## Interaction Design

- [ ] CHK106 - Are all keyboard shortcuts documented without conflicts (e.g., `s` for skip vs any existing `s` binding in suggest view)? [Interaction]
- [ ] CHK107 - Is the input modification overlay (m key) defined with entry/exit semantics (how to cancel, how to confirm, what keys are available inside)? [Interaction]
- [ ] CHK108 - Are focus management rules defined for the overlay (does the list lose focus, can the user still Tab away)? [Interaction]
- [ ] CHK109 - Is the archive divider's interaction behavior specified (is it focusable, does cursor skip it, does it consume any keys)? [Interaction]
- [ ] CHK110 - Are multi-select visual indicators defined (how does the user know which proposals are selected vs the cursor position)? [Interaction]

## Error Recovery

- [ ] CHK111 - Is there a recovery path if the auto-transition to proposals fails (e.g., suggest provider initialization error)? [Error Recovery]
- [ ] CHK112 - If a launched pipeline fails immediately, does the fleet view handle the transition from running→failed gracefully in real-time? [Error Recovery]
- [ ] CHK113 - Is the behavior defined when the state store (SQLite) is unavailable during fleet view updates? [Error Recovery]
- [ ] CHK114 - Does the spec define behavior when the user resizes the terminal below the 80x24 minimum during guided flow? [Error Recovery]

## Performance Requirements

- [ ] CHK115 - Is the 500ms health check startup target (SC-001) measured from process start or TUI render? [Performance]
- [ ] CHK116 - Is the 2-second fleet update interval (SC-005) sufficient for real-time feedback, and is it justified against existing polling behavior? [Performance]
- [ ] CHK117 - Are memory constraints defined for 10+ concurrent tracked runs (SC-006) — e.g., per-run memory budget? [Performance]
- [ ] CHK118 - Is the DAG rendering performance specified for large sequences (10+ pipeline steps)? [Performance]

## Backward Compatibility

- [ ] CHK119 - Is the non-regression scope (Story 7) defined precisely — does it cover exit codes, stdout format, stderr behavior, and timing characteristics? [Compatibility]
- [ ] CHK120 - Does the spec address `wave` flag combinations that might conflict with guided mode (e.g., `wave --view pipelines`)? [Compatibility]
- [ ] CHK121 - Is the behavior defined when external tools parse `wave run` output — does guided mode affect stdout/stderr? [Compatibility]
- [ ] CHK122 - Are existing TUI keybindings (c for cancel, r for re-run, etc.) preserved in guided mode without conflicts? [Compatibility]
