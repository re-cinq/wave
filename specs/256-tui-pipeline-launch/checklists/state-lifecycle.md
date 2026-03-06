# State & Lifecycle Quality Review: TUI Pipeline Launch Flow

**Feature**: #256 — TUI Pipeline Launch Flow
**Date**: 2026-03-06
**Focus**: State machine transitions, goroutine lifecycle, and message routing requirements

## Completeness

- [ ] CHK101 - Are all transitions INTO `stateConfiguring` fully specified? The spec covers Enter-on-available but not whether re-selecting the same available pipeline after a successful launch should re-enter configuring. [Completeness]
- [ ] CHK102 - Are all transitions OUT OF `stateLaunching` defined? The spec mentions transition to Running section on success and `stateError` on failure, but does not specify a timeout for the launching state itself. [Completeness]
- [ ] CHK103 - Does the spec define the goroutine lifecycle for the executor `tea.Cmd` — specifically, is the goroutine expected to be non-cancellable once `PipelineLaunchedMsg` is emitted, or can it be interrupted between executor construction and execution? [Completeness]
- [ ] CHK104 - Are requirements defined for the cancel function map's thread safety guarantees (the plan uses `sync.Mutex`, but the spec doesn't mention concurrency requirements for the map)? [Completeness]
- [ ] CHK105 - Does the spec define whether `PipelineLaunchResultMsg` should trigger any UI update beyond cancel map cleanup (e.g., toast notification, status bar update, or detail pane refresh)? [Completeness]

## Clarity

- [ ] CHK106 - Is the ownership of the `context.CancelFunc` unambiguous? The spec entity definition puts it on `PipelineLaunchedMsg`, the plan stores it in `Launch()` before emitting the message, and the tasks reference it in both places. Which component is the authoritative holder? [Clarity]
- [ ] CHK107 - Is "background goroutine via `tea.Cmd`" (FR-005) clear about whether this is a single `tea.Cmd` or a `tea.Batch` of two commands (immediate + blocking)? The plan specifies batch, but the requirement doesn't. [Clarity]
- [ ] CHK108 - Is the meaning of "optional model name override" clear — does empty string mean "use default" or "don't override"? Are these semantically different in the executor? [Clarity]

## Consistency

- [ ] CHK109 - Is the `CancelAll()` invocation consistent between `q`-quit and `Ctrl+C` paths? FR-017 says both, but the existing `Ctrl+C` handler uses `shuttingDown` flag for double-press exit — does `CancelAll()` run before or after the flag check? [Consistency]
- [ ] CHK110 - Is the message routing for `LaunchErrorMsg` consistent between the plan (content routes to detail) and the launcher (which generates it)? The launcher returns it as a `tea.Cmd` result, but content must intercept it before it reaches detail — is the routing path specified? [Consistency]
- [ ] CHK111 - Is the `RunID` generation consistent between spec (state store `CreateRun()`) and tasks (fallback to `pipeline.GenerateRunID()`)? If the fallback is used, does the synthetic running entry still work without a state store record? [Consistency]

## Coverage

- [ ] CHK112 - Do requirements cover the scenario where two TUI-launched pipelines finish at the same moment, potentially causing concurrent `PipelineLaunchResultMsg` processing and cancel map mutations? [Coverage]
- [ ] CHK113 - Are requirements defined for what `PipelineListModel` does when a `PipelineLaunchedMsg` arrives while the list is in filter mode? Does the new running entry appear in filtered results? [Coverage]
- [ ] CHK114 - Do edge cases address the scenario where the manifest file changes on disk between TUI startup and pipeline launch (stale `LaunchDependencies.Manifest`)? [Coverage]
- [ ] CHK115 - Are test requirements defined for verifying that `CancelAll()` is idempotent (calling it twice doesn't panic or produce errors)? [Coverage]
