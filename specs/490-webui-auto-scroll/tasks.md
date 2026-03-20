# Tasks

## Phase 1: Foundation — Auto-Scroll State

- [X] Task 1.1: Add `autoScroll: true` property to section objects in `LogViewer.prototype.createSection`
- [X] Task 1.2: Add `_isNearBottom(element)` helper method that returns true when `scrollTop + clientHeight >= scrollHeight - 50`

## Phase 2: Core Implementation

- [X] Task 2.1: Add Jump to Bottom button element to `step_card.html` inside `.step-log` div [P]
- [X] Task 2.2: Add CSS rules for `.jump-to-bottom` button (fixed position within step log, hidden by default, visible via `.visible` class) under the existing placeholder comment in `style.css` [P]
- [X] Task 2.3: Attach scroll listener to `.step-log-content` elements in `init()` — on scroll, update `section.autoScroll` based on `_isNearBottom`, toggle button visibility
- [X] Task 2.4: Add auto-scroll call at end of `flushBatch()` — after appending fragment, if `section.autoScroll` is true, set `logBody.scrollTop = logBody.scrollHeight`
- [X] Task 2.5: Wire Jump to Bottom button click handler — scroll to bottom, set `section.autoScroll = true`, hide button
- [X] Task 2.6: Disable auto-scroll in `_scrollToCurrentMatch()` — set the matched section's `autoScroll = false` to prevent fighting with search navigation

## Phase 3: Edge Cases & Polish

- [X] Task 3.1: Ensure `reattach()` re-binds scroll listeners and preserves auto-scroll state after polling rebuilds step cards
- [X] Task 3.2: Ensure `_getLogBody()` (dynamic log container creation) also attaches scroll listener when creating new containers
- [X] Task 3.3: Handle section expand/collapse — when expanding a section, auto-scroll to bottom if `autoScroll` is true

## Phase 4: Validation

- [X] Task 4.1: Run `go test ./...` to confirm no regressions
- [X] Task 4.2: Verify template parsing with `go build ./...`
