# Quality Review Checklist: TUI Live Output Streaming

**Feature**: #257 | **Date**: 2026-03-06

## Completeness

- [ ] CHK001 - Are all six user stories traceable to specific issue acceptance criteria from #257? [Completeness]
- [ ] CHK002 - Does the spec define behavior for ALL event states emitted by the executor (started, running, completed, failed, contract_validating, stream_activity, step_progress, eta_updated, compaction_progress)? [Completeness]
- [ ] CHK003 - Are requirements defined for what happens when the TUI is resized while each sub-component (header, viewport, footer) is rendering? [Completeness]
- [ ] CHK004 - Is the behavior specified for when a pipeline transitions to Finished while the user has the LEFT pane focused (not viewing live output)? [Completeness]
- [ ] CHK005 - Are cleanup/teardown requirements specified for all stateful resources (event buffers, tickers, transition timers) and their lifecycle triggers? [Completeness]
- [ ] CHK006 - Is there a requirement covering what happens when `program.Send()` is called after the TUI has exited or the program is nil? [Completeness]
- [ ] CHK007 - Are accessibility requirements defined beyond NO_COLOR (e.g., screen reader compatibility, keyboard-only navigation completeness)? [Completeness]
- [ ] CHK008 - Does the spec address behavior when the ring buffer drops lines that the user's viewport is currently displaying (scroll position invalidation)? [Completeness]

## Clarity

- [ ] CHK009 - Is "render cycle" in SC-001 ("within one render cycle of emission") precisely defined — does it mean the next Bubble Tea Update/View loop iteration? [Clarity]
- [ ] CHK010 - Is the distinction between `stateRunningInfo` and `stateRunningLive` clearly defined in terms of entry conditions, not just behavioral differences? [Clarity]
- [ ] CHK011 - Is the "2-second delay" in C5/FR-013 parameterized or hardcoded? Is there a requirement for it to be configurable? [Clarity]
- [ ] CHK012 - Are the exact event types for each display flag mode exhaustively enumerated, or could new event types added to the executor fall through unhandled? [Clarity]
- [ ] CHK013 - Is "scroll to bottom" in auto-scroll resume (C3) precisely defined — does `viewport.AtBottom()` account for partially visible last lines? [Clarity]
- [ ] CHK014 - Is the format of elapsed time in the live output HEADER (FR-011) specified as the same MM:SS/HH:MM:SS format used in the left pane (FR-015), or could they differ? [Clarity]
- [ ] CHK015 - Is the thread-safety model for `EventBuffer` clearly specified — who writes (executor goroutine via emitter) vs who reads (UI goroutine via viewport)? [Clarity]

## Consistency

- [ ] CHK016 - Does C13 (display flags act at formatting stage only) consistently align with ALL acceptance scenarios in US-2, particularly scenario 2's "no longer shown" phrasing? [Consistency]
- [ ] CHK017 - Is the footer content described in FR-005 (auto-scroll indicator) and FR-020 (display flag state) specified as a SINGLE footer area, or could they conflict for space? [Consistency]
- [ ] CHK018 - Does the `stateRunningLive` focus behavior (FR-022 Enter/Esc) follow the exact same pattern as `stateAvailableDetail` and `stateFinishedDetail` from #255/#256? [Consistency]
- [ ] CHK019 - Are the status bar hints (FR-023) consistent with key bindings actually handled in the LiveOutputModel — i.e., does the hint text exactly match the implemented shortcuts? [Consistency]
- [ ] CHK020 - Does C15 (deferred transition while scrolling) contradict US-3 scenario 2 ("when 2 seconds elapse") which doesn't mention the auto-scroll precondition? [Consistency]
- [ ] CHK021 - Is the running pipeline elapsed time format in the left pane (FR-015 MM:SS/HH:MM:SS) consistent with how completed pipelines show duration in the Finished section? [Consistency]
- [ ] CHK022 - Does C11 (NewProgressOnlyEmitter) align with the actual API of `event.NewProgressOnlyEmitter` in the codebase, or is the spec assuming an API that doesn't exist? [Consistency]

## Coverage

- [ ] CHK023 - Are there acceptance scenarios covering concurrent event delivery from multiple running pipelines to ensure events don't cross-contaminate buffers? [Coverage]
- [ ] CHK024 - Is there a test scenario for the race condition between the completion transition timer firing and the user pressing Esc simultaneously? [Coverage]
- [ ] CHK025 - Are error paths covered for emitter failures (e.g., `EmitProgress` returns error when program is shutting down)? [Coverage]
- [ ] CHK026 - Is there a scenario testing display flag toggles during the 2-second completion transition delay period? [Coverage]
- [ ] CHK027 - Are boundary dimensions (80x24 minimum, 300x100 maximum per SC-009) tested for the three-part layout — does 24 rows leave enough space for header (3) + footer (2) + viewport? [Coverage]
- [ ] CHK028 - Is there a scenario for what happens when a user launches a pipeline, navigates away, the pipeline completes, and the user navigates back to it — is it now in Finished or still showing stale live output? [Coverage]
- [ ] CHK029 - Are there requirements for graceful degradation when the event buffer capacity (1000 lines) is insufficient for extremely long-running pipelines with verbose+debug enabled? [Coverage]
- [ ] CHK030 - Is there a scenario testing the elapsed time ticker's interaction with system clock changes (e.g., NTP adjustments, sleep/wake)? [Coverage]
