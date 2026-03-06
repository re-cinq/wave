# Focus Management & Navigation Quality: TUI Pipeline Detail Right Pane

**Feature**: 255-tui-pipeline-detail | **Date**: 2026-03-06

## Focus State Requirements

- [ ] CHK030 - Does the spec define the complete set of focus states (left-focused, right-focused)? Are there any intermediate states (e.g., transitioning, neither-focused) that need to be addressed? [Completeness]
- [ ] CHK031 - Is the initial focus state explicitly defined (left pane focused on startup), or is it implicitly assumed from the existing behavior? [Completeness]
- [ ] CHK032 - Does the spec define what happens to focus when the selected pipeline disappears (e.g., list refresh removes the item while right pane is focused)? [Coverage]
- [ ] CHK033 - Is the focus ownership hierarchy clear — does `ContentModel` exclusively own focus state, or can child models independently change it? [Clarity]

## Key Routing Requirements

- [ ] CHK034 - Does the spec fully enumerate which keys are consumed by each pane in each focus state? Is there an explicit key routing table or matrix? [Completeness]
- [ ] CHK035 - Is the Enter key overloading between "focus right pane" (US-3), "collapse section" (FR-011), and "open chat" (FR-006 action hint) clearly disambiguated? Are all three contexts mutually exclusive? [Consistency]
- [ ] CHK036 - Does the spec define what happens when the user presses a left-pane key (e.g., `/` for filter) while the right pane is focused? Is it ignored, or does it trigger a focus switch? [Coverage]
- [ ] CHK037 - Is the `q` quit behavior defined for both focus states? Can the user quit from either pane, or must they return to the left pane first? [Coverage]
- [ ] CHK038 - Does the spec define what happens when Page Up/Page Down or Home/End keys are pressed in the right pane? Are only ↑/↓ supported for scrolling? [Completeness]

## Visual Focus Indicators

- [ ] CHK039 - Are the visual focus indicators testable without screenshots? Does the spec define them in terms of lipgloss styles or Bubble Tea rendering properties that can be asserted in unit tests? [Clarity]
- [ ] CHK040 - Is the left pane "dimmed" state defined to be reversible — does the left pane fully restore its appearance when focus returns? [Completeness]
- [ ] CHK041 - Does the spec address whether the right pane shows a visual border when unfocused, or only when focused? Is the pane separation always visible? [Clarity]

## Interaction Edge Cases

- [ ] CHK042 - Does the spec define behavior when Enter is pressed on a pipeline item and the detail data fetch fails? Does focus transfer regardless of data availability? [Coverage]
- [ ] CHK043 - Is there a requirement for what happens when Esc is pressed and the left pane's previously-selected item no longer exists? [Coverage]
- [ ] CHK044 - Does the spec address double-Enter (pressing Enter twice rapidly) — does the second Enter trigger "Open chat" from the action hints, or is it a no-op? [Coverage]
