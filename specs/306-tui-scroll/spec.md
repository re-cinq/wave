# TUI configure view: pipeline list and form are not scrollable on small screens

**Issue**: [#306](https://github.com/re-cinq/wave/issues/306)
**Labels**: bug
**Author**: nextlevelshit
**State**: OPEN

## Problem

The TUI configure view (pipeline selection + launch form) does not scroll when the terminal window is too small to display all content. Both the pipeline list on the left and the pipeline detail/form panel on the right are clipped without any way to reach off-screen items.

## Current Behavior

Pipelines below the visible area and form fields below the visible area are not reachable. On terminals with fewer than ~40 rows, the list and form content is simply clipped.

The pipeline list (left pane) already has `scrollOffset` + `adjustScrollOffset()` working correctly for keyboard navigation. However:

1. The **compose list** (`ComposeListModel`) has **no scroll infrastructure at all** — it renders all lines directly and clips when they exceed the height.
2. The **launch form** in `stateConfiguring` renders a `huh.Form` whose height calculation (`m.height - 3`) doesn't account for small terminals — form fields below the visible area are unreachable.
3. The **right-side detail/form panel** in `stateConfiguring` has no viewport wrapping — the form is rendered directly without scroll support.

## Expected Behavior

1. **Keyboard scrolling**: Arrow keys (Up/Down) should scroll the pipeline list to reveal off-screen items, and the right panel should scroll when focused to reveal clipped form fields.
2. **Viewport auto-scroll**: When the selected item moves off-screen, the viewport should follow the cursor automatically.
3. **Mouse wheel support** (nice-to-have): Mouse wheel events should scroll the focused panel.

## Acceptance Criteria

- [ ] Pipeline list scrolls with arrow keys when content exceeds terminal height
- [ ] Selected pipeline remains visible (viewport follows cursor)
- [ ] Right-side detail/form panel scrolls when content exceeds available height
- [ ] All form fields (Input, Model override, Options) are reachable regardless of terminal size
- [ ] Mouse wheel scrolling works on both panels (stretch goal)

## Environment

- Wave TUI configure view (`wave run` interactive mode)
- Affects terminals with fewer than ~40 rows
