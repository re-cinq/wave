# State Transition Quality Checklist: TUI Live Output Streaming

**Feature**: #257 | **Date**: 2026-03-06
**Focus**: Focus management, state machine transitions, and completion lifecycle requirements quality.

## Focus and Navigation Requirements

- [ ] CHK201 - Is the focus transition path fully specified for ALL states a running pipeline can be in: stateRunningLive → Esc → left pane → re-Enter → stateRunningLive (round-trip)? [Completeness]
- [ ] CHK202 - Does the spec define key handling priority when multiple key bindings overlap — e.g., does `↑` scroll the viewport or navigate the left pane list depending on focus? [Clarity]
- [ ] CHK203 - Is the behavior specified for pressing Enter on a running pipeline that transitions to Finished BETWEEN the Enter keypress and the focus change processing? [Coverage]
- [ ] CHK204 - Are tab/shift-tab or other navigation keys addressed, or only Enter/Esc? Could unhandled keys cause unexpected state changes? [Completeness]
- [ ] CHK205 - Does FR-022 define whether `cursorOnFocusableItem()` change affects keyboard navigation (↑/↓) of the list, or only Enter activation? [Clarity]

## Completion Lifecycle Requirements

- [ ] CHK206 - Is the complete state machine for completion transition documented: running → terminal event → (summary appended) → (timer check) → (timer fires) → finished detail? [Completeness]
- [ ] CHK207 - Does C15 define what happens if the user presses Esc during the 2-second transition delay — is the timer cancelled, or does it fire after the user returns? [Coverage]
- [ ] CHK208 - Is the relationship between the PipelineLaunchResultMsg (from #256) and the TransitionTimerMsg specified — which fires first, and does the order matter? [Consistency]
- [ ] CHK209 - Does the spec address the scenario where a pipeline fails immediately (step 1 fails) — is there enough content in the live output for the 2-second delay to be meaningful? [Coverage]
- [ ] CHK210 - Is there a requirement preventing duplicate transitions — e.g., if both the transition timer and a PipelineLaunchResultMsg trigger a move to Finished? [Coverage]

## Multi-Pipeline State Requirements

- [ ] CHK211 - Is the maximum number of concurrent running pipelines with live output buffers specified or bounded? [Completeness]
- [ ] CHK212 - Does FR-016 (preserve buffer/scroll/auto-scroll per pipeline) specify when this state is created — on launch, on first selection, or on first event? [Clarity]
- [ ] CHK213 - Is the memory footprint of maintaining buffers for N concurrent pipelines addressed — 1000 lines × N pipelines × average line length? [Completeness]
- [ ] CHK214 - Does the spec define behavior when switching from a completed-but-not-yet-transitioned pipeline to another running pipeline and back? [Coverage]
- [ ] CHK215 - Is there a requirement for the elapsed time ticker to handle the case where new running pipelines appear (from polling) while the ticker is already active? [Coverage]
