# State Transitions Checklist: TUI Finished Pipeline Interactions

**Purpose**: Validate requirements quality for TUI state machine, key binding gating, and message routing.
**Created**: 2026-03-06
**Feature**: [spec.md](../spec.md) | [data-model.md](../data-model.md)

## Key Binding State Machine

- [ ] CHK601 - Are all valid state+key combinations documented in a matrix or table — which keys are active in which pane/state? [Completeness]
- [ ] CHK602 - Is the gating logic for each key handler specified as a boolean expression — e.g., `paneState == stateFinishedDetail && focused && !branchDeleted && branchName != ""`? [Clarity]
- [ ] CHK603 - Is the behavior for invalid state+key combinations explicitly defined as no-ops, or is it left ambiguous? [Clarity]
- [ ] CHK604 - Does the Enter key have consistent semantics across pane states — left-pane Enter focuses right, right-pane stateFinishedDetail Enter launches chat? [Consistency]
- [ ] CHK605 - Is the Esc key behavior defined for all right-pane states including the new `stateFinishedDetail` with transient error? [Completeness]

## Message Flow Architecture

- [ ] CHK606 - Is the message routing path documented for each new message type through ContentModel → AppModel → StatusBarModel? [Completeness]
- [ ] CHK607 - Are message type names consistent with existing conventions — do they follow the `<Action><Result>Msg` naming pattern? [Consistency]
- [ ] CHK608 - Is the batching of messages specified where multiple effects must occur — e.g., ChatSessionEndedMsg triggers both data refresh AND git refresh? [Clarity]
- [ ] CHK609 - Are there message ordering guarantees specified, or can messages arrive in any order after a batch? [Coverage]

## Focus and Navigation

- [ ] CHK610 - Is the focus state after chat session exit specified — does focus remain on right pane showing the same finished detail? [Completeness]
- [ ] CHK611 - Is the focus state after branch checkout specified — does the user stay on the finished detail or navigate elsewhere? [Completeness]
- [ ] CHK612 - Is the `FinishedDetailActiveMsg` emission defined for all transitions into and out of the finished detail state — not just Enter/Esc but also cursor movement to a different pipeline? [Coverage]
- [ ] CHK613 - Does cursor movement to a non-finished pipeline while the right pane shows `stateFinishedDetail` correctly emit `FinishedDetailActiveMsg{Active: false}`? [Consistency]

## Error State Machine

- [ ] CHK614 - Is the transient error state orthogonal to the pane state — can `actionError` be set while `paneState` remains `stateFinishedDetail`? [Clarity]
- [ ] CHK615 - Is the error clearing trigger exhaustive — does it clear on ALL key presses, or only specific ones? [Clarity]
- [ ] CHK616 - Is the error display priority defined — if actionError and a BranchCheckoutMsg arrive simultaneously, which takes precedence? [Coverage]
