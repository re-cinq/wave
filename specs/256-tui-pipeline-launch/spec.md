# Feature Specification: TUI Pipeline Launch Flow

**Feature Branch**: `256-tui-pipeline-launch`  
**Created**: 2026-03-06  
**Status**: Draft  
**Issue**: [#256](https://github.com/re-cinq/wave/issues/256) (part 5 of 10, parent: [#251](https://github.com/re-cinq/wave/issues/251))  
**Input**: Implement the pipeline launch flow from the TUI. When a user selects an available pipeline and presses Enter, the right pane shows a configuration menu with pipeline arguments (input field, flag toggles, model override). Pressing Enter from the argument menu starts the pipeline via the executor, moves it from Available to Running in the left pane, and re-focuses the left pane with the new running pipeline auto-selected. Esc cancels at any point. `c` cancels a running pipeline.

## Clarifications

The following ambiguities were identified and resolved during specification refinement:

### C1: Argument menu rendering approach — embedded form vs custom widgets

**Ambiguity**: The issue says "argument menu" in the right pane, but doesn't specify whether this is a full `huh` form (the library already used in `run_selector.go`), a custom Bubble Tea model with manual key handling, or something else.

**Resolution**: Use `huh.Form` embedded within the `PipelineDetailModel` as a child Bubble Tea model. The `huh` library supports embedding — its `Form` type implements `tea.Model` (Init/Update/View). When an available pipeline is selected and Enter is pressed, the right pane transitions from the detail preview to an embedded `huh.Form` containing the input field, flag multi-select, and model override text field. This reuses the existing `WaveTheme()` and `DefaultFlags()` from `run_selector.go`, maintaining visual consistency. The form's completion event triggers pipeline launch.

### C2: Right pane state machine — detail view vs argument menu vs confirmation

**Ambiguity**: The right pane currently has three states (placeholder, loading, detail). Adding an argument menu introduces new states. It's unclear when the detail preview disappears and the form appears, and whether there's a confirmation step (like the CLI's "Run this command?" confirm).

**Resolution**: The right pane gains two additional states: `configuring` (the argument form is active) and `launching` (brief "Starting..." indicator while executor spins up). The state transitions are:
- **Available selected + Enter** (from left pane): right pane switches from detail preview to `configuring` state (the form). Focus moves to the right pane.
- **Form completed** (Enter on last field or explicit submit): right pane briefly shows `launching` state, then the pipeline starts. Focus returns to left pane.
- **Esc during form**: right pane reverts to detail preview, focus returns to left pane. No pipeline started.

There is no separate "Run this command?" confirmation dialog. The CLI selector has one because it's a standalone flow; in the TUI, the user has already deliberately navigated to the pipeline and filled out the form. Adding a confirmation dialog would be an unnecessary friction point. The Esc key provides a clear escape hatch at any point.

### C3: Executor lifecycle and TUI event loop integration

**Ambiguity**: The executor's `Execute()` method is synchronous and blocking (runs all steps sequentially). The TUI runs a Bubble Tea event loop that must not block. It's unclear how these two systems integrate.

**Resolution**: The executor runs in a background goroutine launched via a `tea.Cmd`. The `tea.Cmd` closure creates a cancellable context, constructs the executor with the appropriate options, calls `executor.Execute(ctx, pipeline, manifest, input)`, and returns a `PipelineLaunchResultMsg` when the executor finishes (success or error). The cancel function is stored on the model so the `c` key can invoke it. The state store's existing polling (5-second `PipelineRefreshTickMsg` in the pipeline list) picks up the new running pipeline and reflects it in the Running section without any special wiring.

### C4: Which arguments/flags to present in the menu

**Ambiguity**: The issue mentions "model selector, verbose toggle, debug toggle, required input fields". The existing `DefaultFlags()` includes 6 flags (verbose, output json, output text, dry-run, mock, debug). It's unclear whether the TUI menu should show all 6, a subset, or add a model selector that doesn't exist in the current flags.

**Resolution**: The argument menu presents:
1. **Input text field** — free-text pipeline input (with placeholder from `PipelineInfo.InputExample`). Always shown.
2. **Model override text field** — optional model name override (maps to `--model` CLI flag). Empty means use default. Shown as a text input, not a dropdown, because available models depend on the adapter.
3. **Flag toggles** — multi-select from the existing `DefaultFlags()` set: verbose, output json, output text, dry-run, mock, debug. These map directly to the existing `Selection.Flags` and `applySelection()` logic.

This provides parity with the CLI's `RunOptions` while keeping the form simple. The `--from-step`, `--force`, `--timeout`, and `--run` flags remain CLI-only since they apply to resume scenarios not available through the TUI launch flow.

### C5: How the launched pipeline appears in the Running section

**Ambiguity**: After launching, the issue says the pipeline "moves from Available to Running" and the left pane "re-focuses with the new running pipeline auto-selected at top". But the pipeline list uses polling (5-second tick). If the state store hasn't been updated yet, the new pipeline won't appear in the Running section until the next poll.

**Resolution**: On successful executor start, emit a `PipelineLaunchedMsg` carrying the run ID and pipeline name. The `PipelineListModel` handles this message by immediately inserting a synthetic `RunningPipeline` entry at the top of the Running section (before the next poll fetches it from the state store). The cursor moves to this entry. On the next data refresh tick, the synthetic entry is replaced by the real state store entry. This provides instant visual feedback without waiting for the 5-second polling interval.

### C6: Cancel mechanism for running pipelines

**Ambiguity**: The issue says "`c` key cancels a selected running pipeline" but the current cancellation mechanism is context-driven (SIGINT → cancel()). There's no per-pipeline cancel API on the executor. The TUI needs a way to cancel a specific running pipeline.

**Resolution**: When a pipeline is launched from the TUI, the cancel function from `context.WithCancel()` is stored in a map keyed by run ID on the `ContentModel` (or a new `PipelineLauncher` component). When the user presses `c` with a running pipeline selected, the stored cancel function is invoked, which propagates context cancellation to the executor's step loop and adapter subprocess. The map entry is cleaned up when the `PipelineLaunchResultMsg` arrives (indicating the executor goroutine has exited). Pipelines started from outside the TUI (via CLI) cannot be cancelled from the TUI — this is a known limitation documented in edge cases.

### C7: PipelineLauncher dependency injection — how does the launcher get executor dependencies?

**Ambiguity**: The executor (`pipeline.NewDefaultPipelineExecutor`) requires an `adapter.AdapterRunner`, `state.StateStore`, `workspace.WorkspaceManager`, `audit.AuditLogger`, `event.EventEmitter`, and a loaded `manifest.Manifest`. The spec says `PipelineLauncher` "handles construction of the executor" but doesn't specify how these heavyweight dependencies flow into the TUI, which currently has no access to the manifest or adapter resolution. `RunTUI()` in `app.go:121` currently passes `nil` for providers.

**Resolution**: Introduce a `LaunchDependencies` struct containing the pre-loaded `*manifest.Manifest`, the `state.StateStore` (already available to the data providers), and the pipelines directory path. Pass this struct through `NewAppModel()` → `NewContentModel()` → `PipelineLauncher`. The launcher resolves the adapter runner on demand at launch time (using `adapter.ResolveAdapter()` from the manifest, or `adapter.NewMockAdapter()` when the `--mock` flag is toggled). It constructs the workspace manager and audit logger per-launch as the CLI does in `runRun()`. This keeps TUI startup lightweight — no adapter or workspace manager is created until a pipeline is actually launched. The `RunTUI()` function signature changes to accept `LaunchDependencies` (or its constituent parts).

### C8: huh.Form completion detection in embedded Bubble Tea mode

**Ambiguity**: The spec says "The form's completion event triggers pipeline launch" but the `huh.Form` in the CLI uses blocking `form.Run()`. When embedded in Bubble Tea's non-blocking event loop, the completion mechanism is different.

**Resolution**: The `huh.Form` type implements `tea.Model` (Init/Update/View). When embedded, the `PipelineDetailModel` holds a `*huh.Form` field that is non-nil when in `configuring` state. Key events are forwarded to `form.Update(msg)`. After each update, the model checks `form.State`: if `huh.StateCompleted`, it extracts the bound values (input, model override, flags) from the form's value bindings, constructs a `LaunchConfig`, and returns a `tea.Cmd` that emits a `LaunchRequestMsg`. If `huh.StateAborted` (user pressed Esc within the form), the form field is set to nil and the right pane reverts to detail preview. This is the standard huh embedding pattern documented in the charmbracelet/huh README.

### C9: `q`-to-quit conflict with form text input

**Ambiguity**: `app.go:57` checks `msg.String() == "q" && !m.content.list.filtering` before forwarding key events to content. When the argument form is active in the right pane and the user types `q` in a text input field, this handler fires first and quits the application instead of inserting the character `q`.

**Resolution**: Gate the `q`-to-quit check on the content pane's focus state: `msg.String() == "q" && !m.content.list.filtering && m.content.focus == FocusPaneLeft`. When the right pane has focus (including during form input), `q` is treated as a regular character and forwarded to the focused child component. This is consistent with how Esc is already focus-gated in `content.go:62`. The same guard applies to any future single-character shortcuts.

### C10: PipelineLauncher component hierarchy position

**Ambiguity**: The spec introduces `PipelineLauncher` as a component but doesn't specify where it sits in the `AppModel → ContentModel → (PipelineListModel, PipelineDetailModel)` hierarchy. This affects message routing and access to the cancel function map.

**Resolution**: `PipelineLauncher` is a field on `ContentModel`. This is the natural position because `ContentModel` already mediates all interactions between the left pane (list) and right pane (detail) and manages focus transitions. The flow is: (1) `PipelineDetailModel` detects form completion and returns a `LaunchRequestMsg` via `tea.Cmd`, (2) `ContentModel.Update()` handles `LaunchRequestMsg` by calling `launcher.Launch(config)` which returns a `tea.Cmd` wrapping the executor goroutine, (3) the goroutine emits `PipelineLaunchedMsg` (on start) and `PipelineLaunchResultMsg` (on finish), (4) `ContentModel` routes these to both `PipelineListModel` (for Running section insertion) and the launcher (for cancel function map management). `AppModel` remains unchanged except for the `q`-quit guard and exit cleanup call.

### C11: Executor goroutine cleanup on TUI exit

**Ambiguity**: The spec requires "all TUI-launched pipeline contexts MUST be cancelled" on quit (`q` or `Ctrl+C`). Currently `AppModel.Update()` returns `tea.Quit` immediately. But cancel functions are stored on `ContentModel.launcher`, and there's no hook for `AppModel` to trigger cleanup before the program exits.

**Resolution**: `ContentModel` exposes a `CancelAll()` method that iterates over the launcher's cancel function map and calls each one. `AppModel.Update()` calls `m.content.CancelAll()` before returning `tea.Quit` on `q` or `Ctrl+C`. Since context cancellation is non-blocking (it just sets a flag), this completes instantly. The executor goroutines receive context cancellation and will update the state store with "cancelled" status best-effort — the TUI does not wait for goroutine completion since the process is exiting. For `Ctrl+C`, the existing `shuttingDown` flag ensures a second `Ctrl+C` force-exits via `os.Exit(0)`.

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Launch a Pipeline from the TUI (Priority: P1)

A developer navigates to an available pipeline in the left pane and presses Enter. The right pane transitions from the detail preview to an argument configuration form. The form shows an input text field (with a placeholder example from the pipeline), optional flag toggles (verbose, debug, dry-run, mock, output format), and an optional model override field. The developer fills in the input, toggles verbose mode, and submits the form. The pipeline starts executing in the background. The left pane gains focus, and the newly running pipeline appears at the top of the Running section, auto-selected by the cursor. The detail pane shows the running pipeline's status.

**Why this priority**: This is the core feature — the entire issue exists to enable pipeline launching from the TUI. Without this, users must exit the TUI and use `wave run` from the CLI.

**Independent Test**: Can be tested by selecting an available pipeline, pressing Enter, verifying the form renders with correct fields, submitting the form, and verifying the pipeline appears in the Running section with the left pane focused and cursor on the new entry.

**Acceptance Scenarios**:

1. **Given** the left pane is focused with an available pipeline selected, **When** the user presses Enter, **Then** the right pane transitions from the detail preview to an argument configuration form with input, model, and flag fields.
2. **Given** the argument form is visible in the right pane, **When** the form renders, **Then** the input field shows the pipeline's `InputExample` as placeholder text.
3. **Given** the argument form is filled out, **When** the user submits the form (Enter on the submit button), **Then** the pipeline executor starts in a background goroutine with the specified arguments.
4. **Given** the executor has started successfully, **When** the state updates, **Then** the pipeline appears at the top of the Running section in the left pane with an elapsed time indicator, the left pane gains focus, and the cursor auto-selects the new running pipeline.
5. **Given** the pipeline has been launched, **When** the status bar updates, **Then** it shows left-pane hints (navigate, view, filter, quit).

---

### User Story 2 - Cancel Pipeline Launch Before Starting (Priority: P1)

A developer opens the argument form for a pipeline but decides not to launch it. They press Esc at any point during the form. The right pane reverts to the pipeline detail preview. Focus returns to the left pane. No pipeline is started.

**Why this priority**: Equally critical to US-1. Users must be able to back out of a launch without consequences. Without cancel support, accidentally pressing Enter on a pipeline forces an unwanted execution.

**Independent Test**: Can be tested by opening the argument form, pressing Esc, and verifying the right pane reverts to the detail preview with no state changes (no new run record, no running pipeline).

**Acceptance Scenarios**:

1. **Given** the argument form is active in the right pane, **When** the user presses Esc, **Then** the form is dismissed, the right pane reverts to the available pipeline detail preview, and focus returns to the left pane.
2. **Given** the user pressed Esc during the form, **When** the left pane re-gains focus, **Then** the cursor remains on the same available pipeline that was previously selected.
3. **Given** the user pressed Esc during the form, **When** checking the state store, **Then** no new run record has been created.

---

### User Story 3 - Cancel a Running Pipeline (Priority: P2)

A developer selects a running pipeline in the left pane and presses `c`. The running pipeline's context is cancelled, causing the executor to stop after the current step completes (or immediately if the adapter supports it). The pipeline transitions from Running to Finished with a "cancelled" status.

**Why this priority**: Cancel support is important for long-running pipelines, but users can also close the TUI and send SIGINT as a workaround. This provides a more targeted cancellation mechanism.

**Independent Test**: Can be tested by launching a pipeline, selecting it in the Running section, pressing `c`, and verifying it transitions to Finished with "cancelled" status.

**Acceptance Scenarios**:

1. **Given** the left pane is focused with a running pipeline selected (launched from this TUI session), **When** the user presses `c`, **Then** the pipeline's cancellation function is invoked, stopping the executor.
2. **Given** a pipeline has been cancelled, **When** the next data refresh occurs, **Then** the pipeline moves from the Running section to the Finished section with "cancelled" status.
3. **Given** a running pipeline was started from outside the TUI (e.g., via CLI in another terminal), **When** the user selects it and presses `c`, **Then** no action is taken (the TUI only cancels pipelines it launched).
4. **Given** no running pipeline is selected (cursor is on a section header, finished, or available item), **When** the user presses `c`, **Then** nothing happens.

---

### User Story 4 - Handle Pipeline Launch Errors (Priority: P2)

A developer submits the argument form, but the pipeline fails to start (e.g., manifest loading error, missing adapter, preflight failure). The right pane displays an actionable error message explaining what went wrong. Focus returns to the left pane. The user can try again or select a different pipeline.

**Why this priority**: Error handling ensures the TUI doesn't silently fail or crash when launch conditions aren't met. However, most pipelines will start successfully in normal operation, making this secondary to the happy path.

**Independent Test**: Can be tested by attempting to launch a pipeline with invalid configuration (e.g., missing adapter) and verifying the error message appears in the right pane.

**Acceptance Scenarios**:

1. **Given** the user submits the argument form, **When** the executor fails to start (e.g., adapter resolution fails), **Then** the right pane displays an error message with the failure reason.
2. **Given** a launch error is displayed, **When** the user presses Esc or navigates away, **Then** the error is dismissed and the right pane returns to normal detail view behavior.
3. **Given** a launch error occurred, **When** the user re-selects the same available pipeline and presses Enter, **Then** the argument form appears again (retry is possible).

---

### User Story 5 - Dry-Run Preview (Priority: P3)

A developer toggles the `--dry-run` flag in the argument form and submits. The system shows the execution plan (step order, personas, artifacts) without actually running the pipeline. The preview appears in the right pane. No pipeline run record is created, and no workspaces are set up.

**Why this priority**: Dry-run is a convenience feature. Users can inspect pipeline details in the existing detail view and use `wave run --dry-run` from the CLI. TUI integration is nice-to-have.

**Independent Test**: Can be tested by selecting dry-run in the form, submitting, and verifying no run record is created while the execution plan is displayed.

**Acceptance Scenarios**:

1. **Given** the argument form is visible with the `--dry-run` flag toggled on, **When** the user submits the form, **Then** the right pane shows the execution plan without starting a real pipeline run.
2. **Given** a dry-run is in progress, **When** it completes, **Then** no entry appears in the Running or Finished sections of the left pane.

---

### Edge Cases

- What happens when the user presses Enter on an available pipeline that is already running (launched previously)? The argument form still appears — launching a second instance is allowed (each gets a unique run ID). The Running section shows both instances.
- What happens when the argument form is open and the terminal is resized? The form re-renders to fit the new right pane dimensions. `huh.Form` supports width updates via `.WithWidth()`.
- What happens when the user submits the form with empty input for a pipeline that requires it? The executor starts regardless — input validation is the executor's responsibility (pipelines that need input will fail at the appropriate step). The TUI does not pre-validate input requirements.
- What happens when the user presses `c` on a pipeline that has already finished between key press and processing? No action — the cancel function map lookup returns nil (already cleaned up).
- What happens when multiple pipelines are launched simultaneously from the TUI? Each runs in its own goroutine with its own context and cancel function. The Running section shows all of them. The cancel function map tracks them independently by run ID.
- What happens when the TUI is closed (q or Ctrl+C) while pipelines are running? All TUI-launched pipelines receive context cancellation via `ContentModel.CancelAll()` called from `AppModel` before `tea.Quit`. The executor goroutines clean up and mark runs as cancelled in the state store best-effort.
- What happens when the state store is unavailable? The pipeline can still be launched (executor handles missing state store gracefully), but the launched pipeline won't appear in the Running section since the list polls the state store. An error indicator could be shown.
- What happens when the user presses `q` while the argument form is active? The `q` key is forwarded to the form as a regular character (not quit) because the `q`-to-quit handler is gated on `m.content.focus == FocusPaneLeft`.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: When the user presses Enter on an available pipeline item in the left pane, the right pane MUST transition from the detail preview to an argument configuration form. Focus MUST transfer to the right pane.
- **FR-002**: The argument form MUST contain: (a) a text input field for pipeline input with the pipeline's `InputExample` as placeholder, (b) a text input field for optional model override, and (c) a multi-select of flags from `DefaultFlags()` (verbose, output json, output text, dry-run, mock, debug).
- **FR-003**: The argument form MUST use the existing `WaveTheme()` for visual consistency with the standalone pipeline selector.
- **FR-004**: Pressing Esc at any point during the argument form MUST dismiss the form, revert the right pane to the detail preview, and return focus to the left pane without starting a pipeline. Form abort is detected via `form.State == huh.StateAborted` after each `Update()` call.
- **FR-005**: Submitting the argument form MUST start the pipeline executor in a background goroutine via a `tea.Cmd`, passing the selected input, model override, and flags. Form completion is detected via `form.State == huh.StateCompleted` after each `Update()` call.
- **FR-006**: The executor MUST be constructed with the same options as the CLI path: `WithEmitter`, `WithRunID`, `WithWorkspaceManager`, `WithStateStore`, `WithAuditLogger`, `WithStepTimeout`, `WithModelOverride`, and `WithDebug` as applicable. Dependencies are supplied via a `LaunchDependencies` struct passed through `NewAppModel()` → `NewContentModel()` → `PipelineLauncher`.
- **FR-007**: After successful executor start, a `PipelineLaunchedMsg` MUST be emitted carrying the run ID, pipeline name, and cancel function. The left pane MUST immediately insert the pipeline at the top of the Running section and move the cursor to it.
- **FR-008**: Focus MUST return to the left pane after the pipeline is launched, with the newly running pipeline auto-selected.
- **FR-009**: The status bar MUST update its hints to reflect the current context: left-pane default hints when browsing, right-pane form hints when configuring (Tab/Shift+Tab: navigate fields, Enter: submit, Esc: cancel).
- **FR-010**: Pressing `c` on a running pipeline that was launched from the current TUI session MUST invoke the stored cancel function, sending context cancellation to the executor.
- **FR-011**: The cancel function map MUST be keyed by run ID. Entries MUST be cleaned up when the executor goroutine completes (via `PipelineLaunchResultMsg`).
- **FR-012**: When the executor goroutine finishes (success or failure), a `PipelineLaunchResultMsg` MUST be emitted with the run ID and optional error. The data refresh tick picks up the final state from the state store.
- **FR-013**: If the pipeline fails to start (executor construction or initial validation error), the right pane MUST display an error message. Focus MUST return to the left pane.
- **FR-014**: The argument form MUST be dismissable and re-openable — selecting a different available pipeline and pressing Enter MUST show a fresh form for the new pipeline.
- **FR-015**: When the `--dry-run` flag is selected, the executor SHOULD run in dry-run mode and the result SHOULD be displayed in the right pane without creating a persistent run record.
- **FR-016**: The right pane MUST handle the `configuring` state (form active), `launching` state (brief indicator), and `error` state (launch failure) in addition to existing states (placeholder, loading, detail preview).
- **FR-017**: When the TUI exits (q or Ctrl+C), all TUI-launched pipeline contexts MUST be cancelled. `AppModel.Update()` MUST call `m.content.CancelAll()` before returning `tea.Quit`.
- **FR-018**: Running pipelines started from outside the TUI (e.g., CLI) MUST NOT be cancellable via `c` — the key MUST only act on pipelines with a stored cancel function.
- **FR-019**: The `q`-to-quit handler in `AppModel.Update()` MUST be gated on `m.content.focus == FocusPaneLeft` to prevent quitting when the user types `q` in the argument form's text fields.
- **FR-020**: `PipelineLauncher` MUST be a field on `ContentModel`, which routes `LaunchRequestMsg` (from detail), `PipelineLaunchedMsg`, and `PipelineLaunchResultMsg` between the launcher, list, and detail components.

### Key Entities

- **LaunchDependencies**: Struct containing `*manifest.Manifest`, `state.StateStore`, and `PipelinesDir string`. Passed through `NewAppModel()` → `NewContentModel()` → `PipelineLauncher`. The launcher resolves adapter runner, workspace manager, and audit logger on demand at launch time, keeping TUI startup lightweight.
- **PipelineLauncher**: Component on `ContentModel` responsible for managing the lifecycle of TUI-launched pipelines. Stores cancel functions keyed by run ID. Constructs the executor with dependencies from `LaunchDependencies`. Exposes `Launch(config LaunchConfig) tea.Cmd`, `Cancel(runID string)`, and `CancelAll()`. Invoked by `ContentModel` when a `LaunchRequestMsg` arrives.
- **LaunchRequestMsg**: Message emitted by `PipelineDetailModel` when the huh.Form completes (`form.State == huh.StateCompleted`). Carries the `LaunchConfig`. Routed from detail → `ContentModel` → launcher.
- **PipelineLaunchedMsg**: Message emitted when a pipeline executor starts successfully. Carries `RunID string`, `PipelineName string`, and `CancelFunc context.CancelFunc`. Consumed by `PipelineListModel` to insert the running entry, and by the cancel function map for `c` key support.
- **PipelineLaunchResultMsg**: Message emitted when a pipeline executor goroutine completes. Carries `RunID string` and `Err error`. Used to clean up the cancel function map and optionally update UI state.
- **LaunchConfig**: Data structure assembled from the argument form submission. Contains `PipelineName string`, `Input string`, `ModelOverride string`, `Flags []string`, `DryRun bool`. Maps to the existing `Selection` type and `applySelection()` logic from `run_selector.go`.
- **DetailPaneState**: Enum (or const set) tracking the right pane's current rendering mode: `stateEmpty`, `stateLoading`, `stateAvailableDetail`, `stateFinishedDetail`, `stateRunningInfo`, `stateConfiguring`, `stateLaunching`, `stateError`. Replaces the implicit state inference from nil checks on `availableDetail`/`finishedDetail`.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Pressing Enter on an available pipeline transitions the right pane to an argument form with all specified fields (input, model, flags) — verified by unit tests rendering the form and checking field presence.
- **SC-002**: Submitting the argument form with valid input starts the executor in a background goroutine and the pipeline appears in the Running section within one render cycle — verified by tests with a mock adapter and state store.
- **SC-003**: Pressing Esc during the argument form reverts to the detail preview with no side effects (no run record, no executor started) — verified by unit tests checking state store and model state.
- **SC-004**: Pressing `c` on a TUI-launched running pipeline invokes the cancel function and the pipeline transitions to cancelled status — verified by tests with a mock adapter and context cancellation assertion.
- **SC-005**: Launch errors (adapter resolution, manifest loading, preflight failures) display an actionable error message in the right pane — verified by tests with mock components that return errors.
- **SC-006**: All existing tests (`go test ./internal/tui/...`) continue to pass after integration — the launch flow does not break pipeline list, detail, header, or status bar components.
- **SC-007**: The argument form renders correctly at terminal widths from 80 to 300 columns and heights from 24 to 100 rows — verified by rendering tests at boundary dimensions.
- **SC-008**: TUI exit (q/Ctrl+C) cancels all running TUI-launched pipelines and calls `CancelAll()` — verified by tests asserting cancel function invocation.
- **SC-009**: Pressing `q` while the argument form has focus does NOT quit the TUI — verified by unit test asserting the key is forwarded to the form component.
