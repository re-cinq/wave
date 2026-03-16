# State Machine Requirements Quality Checklist

**Feature**: Guided Workflow Orchestrator TUI (#248)
**Focus**: View state machine transitions, navigation model, and mode coexistence

---

## Transition Completeness

- [ ] CHK101 - Is every state-to-state transition explicitly defined with trigger event, guard condition, and resulting action? [Completeness]
- [ ] CHK102 - Are invalid transitions specified — what happens if the user tries to attach (Enter) from Proposals, or detach (Esc) from Proposals? [Completeness]
- [ ] CHK103 - Is the initial state entry condition complete — what if `wave` starts and the health provider is not yet initialized? [Completeness]
- [ ] CHK104 - Are re-entry transitions defined — what state is restored when returning to Proposals from Fleet (cursor position, scroll offset, selections)? [Completeness]
- [ ] CHK105 - Is the terminal state defined — is there a clean shutdown transition from every state when the user quits? [Completeness]

## Mode Coexistence

- [ ] CHK106 - Is the interaction between guided mode and non-guided mode clearly bounded — can the user switch from guided to non-guided mid-session? [Clarity]
- [ ] CHK107 - Are the secondary view access keybindings (`[`/`]` or `v`) specified in the requirements or only in the clarification — should they be promoted to FRs? [Coverage]
- [ ] CHK108 - Is the guided state machine independent of the pipeline execution state machine — are there cross-dependencies that could cause deadlocks or race conditions? [Consistency]

## Navigation Edge Cases

- [ ] CHK109 - Is the behavior defined when Tab is pressed rapidly multiple times — could it cause view flickering or state desynchronization? [Completeness]
- [ ] CHK110 - Is the behavior defined when the user launches a pipeline and immediately presses Tab back to Proposals before the fleet view renders? [Completeness]
- [ ] CHK111 - Is the behavior defined for the Attached state when the attached pipeline completes — does the view auto-return to Fleet or stay in Attached? [Completeness]
- [ ] CHK112 - Are focus and cursor state preservation requirements specified across view transitions — does switching views reset or preserve cursor position? [Completeness]
