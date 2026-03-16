# UX & Interaction Requirements Quality Checklist

**Feature**: Guided Workflow Orchestrator TUI (#248)
**Focus**: User interaction patterns, visual feedback, and responsive design

---

## Input Handling

- [ ] CHK201 - Are all keybindings documented in a single reference table showing key, action, and which views it applies to? [Completeness]
- [ ] CHK202 - Is the input overlay (`m` key) interaction fully specified — does it capture all keys (blocking underlying navigation), and how does it handle special keys like Tab, Ctrl+C? [Completeness]
- [ ] CHK203 - Is the multi-select (Space) interaction specified for edge cases — what if all proposals are selected, what if only skipped proposals remain? [Completeness]
- [ ] CHK204 - Are mouse interaction requirements stated — is mouse support explicitly included or excluded for the guided flow? [Coverage]

## Visual Feedback

- [ ] CHK205 - Are loading/progress indicator requirements specified for proposal data fetching (after health phase completes, before proposals render)? [Completeness]
- [ ] CHK206 - Are visual feedback requirements defined for launch confirmation — is there a success indicator, toast message, or immediate view switch? [Clarity]
- [ ] CHK207 - Are the health check spinner animation requirements specified — frame rate, glyph set, color? [Clarity]
- [ ] CHK208 - Is the "active section" visual indicator specified for when the user is in the list vs detail pane of proposals? [Clarity]

## Responsive Layout

- [ ] CHK209 - Are layout requirements defined for terminal widths between 80 and 200 columns — does the DAG preview scale or truncate? [Completeness]
- [ ] CHK210 - Are requirements defined for the proposals view layout split (list vs detail pane) — fixed width, percentage, or adaptive? [Clarity]
- [ ] CHK211 - Is the behavior defined when the terminal is too narrow for the DAG preview — does it collapse, scroll horizontally, or show a simplified view? [Completeness]
- [ ] CHK212 - Are requirements defined for very long proposal lists (20+ proposals) — scrolling behavior, visible count, scroll indicators? [Completeness]
