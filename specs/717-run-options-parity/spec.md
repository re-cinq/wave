# Feature Specification: Run Options Parity Across All Surfaces

**Feature Branch**: `717-run-options-parity`  
**Created**: 2026-04-11  
**Status**: Draft  
**Input**: https://github.com/re-cinq/wave/issues/717

## User Scenarios & Testing _(mandatory)_

### User Story 1 - WebUI Inline Run Form on Pipeline Detail (Priority: P1)

A user navigates to a pipeline detail page in the WebUI and sees all run configuration options directly inline — no modal dialog. Tier 1 options (Input, Adapter, Model) are always visible. Tier 2 (Advanced) and Tier 3 (Continuous) are in collapsible sections. The user configures options and starts a run without leaving the page.

**Why this priority**: The pipeline detail page is the primary launch surface for WebUI users. The current modal limits discoverability and only exposes 6 options. Replacing it with an inline tiered form is the highest-impact change for option parity.

**Independent Test**: Navigate to any pipeline detail page. Verify the inline form renders with all Tier 1–3 options. Start a run with non-default options and confirm they are passed through to the pipeline execution.

**Acceptance Scenarios**:

1. **Given** the pipeline detail page is loaded, **When** the user views the page, **Then** Tier 1 options (Input, Adapter, Model) are visible without any interaction.
2. **Given** the inline form is visible, **When** the user expands "Advanced", **Then** Tier 2 options (from-step, force, dry-run, detach, steps, exclude, timeout, on-failure) are shown.
3. **Given** Tier 2 is expanded, **When** from-step has no value, **Then** the Force checkbox is hidden.
4. **Given** Tier 2 is expanded, **When** the user sets a from-step value, **Then** the Force checkbox appears.
5. **Given** the user selects Detach mode and submits, **When** the run starts, **Then** the browser navigates immediately to the run detail page (no live log streaming on the pipeline page).
6. **Given** the user selects Dry Run and submits, **When** the dry run completes, **Then** the dry-run report is rendered inline on the same pipeline detail page.
7. **Given** the user submits without Detach, **When** the run starts, **Then** the browser navigates to the run detail page and streams live logs.

---

### User Story 2 - API Request Types Carry Full Option Set (Priority: P1)

A developer or automated system sends a `StartPipelineRequest` to the API with any combination of Tier 1–4 options. The handler wires every field through to `RunOptions` so the subprocess receives the correct flags.

**Why this priority**: The API is the integration backbone — WebUI, TUI, and external tools all depend on it. Without full API coverage, no surface can reach parity.

**Independent Test**: Send a `StartPipelineRequest` with all Tier 1–4 fields populated. Verify the spawned subprocess command line includes the corresponding flags.

**Acceptance Scenarios**:

1. **Given** a `StartPipelineRequest` with `detach: true`, **When** the handler processes it, **Then** the subprocess is spawned with `--detach`.
2. **Given** a `StartPipelineRequest` with `continuous: true, source: "github:label=bug", max_iterations: 10, delay: "5s"`, **When** the handler processes it, **Then** all four continuous flags are passed to the subprocess.
3. **Given** a `StartPipelineRequest` with `on_failure: "skip"`, **When** the handler processes it, **Then** `--on-failure skip` is passed to the subprocess.
4. **Given** a `StartPipelineRequest` with `mock: true, auto_approve: true, no_retro: true, preserve_workspace: true`, **When** the handler processes it, **Then** all four Tier 4 flags are passed.

---

### User Story 3 - TUI Pipeline Launcher Options Expansion (Priority: P2)

A user launching a pipeline from the TUI sees additional fields beyond the current limited set: adapter selector, timeout, from-step picker, steps/exclude inputs, and a detach toggle.

**Why this priority**: The TUI is the second most-used interactive surface after the WebUI. Expanding its options removes the need to drop to CLI for common overrides.

**Independent Test**: Open the TUI pipeline launcher. Verify adapter, timeout, from-step, steps, exclude, and detach fields are present and functional. Launch a pipeline with non-default values and confirm the subprocess receives the correct flags.

**Acceptance Scenarios**:

1. **Given** the TUI pipeline launcher is open, **When** the user views the form, **Then** an adapter selector is visible alongside the existing model field.
2. **Given** the launcher form, **When** the user sets a timeout value, **Then** `--timeout <value>` is passed to the subprocess.
3. **Given** the launcher form, **When** the user selects a from-step from the picker, **Then** `--from-step <step>` is passed to the subprocess.
4. **Given** the launcher form, **When** the user toggles detach on, **Then** `--detach` is passed and the TUI returns to the pipeline list.
5. **Given** the launcher form, **When** the user enters comma-separated step names in the steps field, **Then** `--steps <value>` is passed to the subprocess.

---

### User Story 4 - Issues/PRs Pages Expose Run Overrides (Priority: P2)

A user on an issue or PR page in the WebUI can override Adapter and Model before starting a pipeline run. Expanding an "Advanced" section reveals Tier 2 options.

**Why this priority**: Issues/PRs pages currently pass zero overrides. Adding at least Tier 1–2 eliminates a common workflow gap where users must navigate to the pipeline detail page just to change the model.

**Independent Test**: Navigate to an issue page. Verify Adapter and Model selectors are visible. Expand "Advanced" and set timeout. Start the run and confirm the overrides are applied.

**Acceptance Scenarios**:

1. **Given** an issue detail page, **When** the user views the run action area, **Then** Adapter and Model selectors are visible.
2. **Given** the issue run action area, **When** the user expands "Advanced", **Then** Tier 2 options are accessible.
3. **Given** a PR detail page with overrides set, **When** the user starts the pipeline, **Then** the `StartPRRequest` carries all specified options to the handler.

---

### User Story 5 - CLI Help Groups Flags by Tier (Priority: P3)

A user running `wave run --help` sees flags organized into four groups: Essential, Execution, Continuous, and Dev/Debug. Each flag description uses consistent language matching the canonical tier model.

**Why this priority**: The CLI is already feature-complete. This is a documentation/UX polish pass to align its presentation with the tiered model used across all surfaces.

**Independent Test**: Run `wave run --help` and verify flags appear under four group headings with consistent descriptions.

**Acceptance Scenarios**:

1. **Given** the user runs `wave run --help`, **When** the output renders, **Then** flags are grouped under "Essential", "Execution", "Continuous", and "Dev/Debug" headings.
2. **Given** the help output, **When** the user reads flag descriptions, **Then** each description uses the same language as the canonical tier model in the documentation.

---

### User Story 6 - Documentation Reflects Tiered Model (Priority: P3)

A user reading the CLI reference docs or the running-pipelines guide sees the full tiered option model documented consistently across all surfaces.

**Why this priority**: Documentation alignment is the final consistency layer. It has lower urgency than code changes but is required for the feature to be considered complete.

**Independent Test**: Review `docs/reference/cli.md` and the running-pipelines guide. Verify all tiers are documented with consistent language, and WebUI/TUI options are covered.

**Acceptance Scenarios**:

1. **Given** `docs/reference/cli.md`, **When** the user reads it, **Then** all flags are listed with tier groupings.
2. **Given** a running-pipelines guide exists, **When** the user reads it, **Then** TUI and WebUI run options are documented.
3. **Given** `CHANGELOG.md`, **When** the user checks the relevant version, **Then** an entry describes the run options parity changes.

---

### Edge Cases

- **from-step references a step that no longer exists in the pipeline YAML**: The form must show an inline validation error and prevent submission.
- **Continuous mode with max_iterations=0**: The system treats this as unlimited iterations. The UI must display a warning that the pipeline will run indefinitely.
- **Detach + Dry Run selected simultaneously**: Dry run takes precedence — the system performs a dry run and renders the report inline; detach is ignored.
- **steps and exclude both specified with overlapping values**: The system must reject the request with a clear validation error.
- **API request with unknown fields**: Unknown fields are silently ignored for forward compatibility.
- **TUI from-step picker on a pipeline with no steps defined**: The picker must be disabled with explanatory text.
- **WebUI form submission with required field (pipeline name) missing**: Inline validation error prevents submission.
- **Timeout of 0**: Treated as no timeout (infinite). No warning displayed.
- **--continuous and --from-step combined**: Mutually exclusive — the system MUST reject with a clear error message.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: The WebUI pipeline detail page MUST replace the modal run dialog with an inline tiered run form.
- **FR-002**: The inline form MUST display Tier 1 options (Input, Adapter, Model) without any user interaction.
- **FR-003**: The inline form MUST provide a collapsible "Advanced" section containing all Tier 2 options (from-step, force, dry-run, detach, steps, exclude, timeout, on-failure).
- **FR-004**: The inline form MUST provide a collapsible "Continuous" section containing all Tier 3 options (continuous toggle, source, max-iterations, delay).
- **FR-005**: The Force field MUST only be visible when from-step has a value.
- **FR-006**: Dry Run submissions MUST render the report inline on the pipeline detail page without navigation.
- **FR-007**: Detach submissions MUST navigate to the run detail page immediately without streaming.
- **FR-008**: Non-detach, non-dry-run submissions MUST navigate to the run detail page and stream live logs.
- **FR-009**: `StartPipelineRequest` MUST include fields for all Tier 1–3 options and Tier 4 flags (mock, preserve_workspace, auto_approve, no_retro).
- **FR-010**: `handleStartPipeline` MUST wire every `StartPipelineRequest` field through to the subprocess command line via `RunOptions`.
- **FR-011**: `StartIssueRequest` and `StartPRRequest` MUST carry the full option set (Tier 1–3 at minimum).
- **FR-012**: The TUI pipeline launcher MUST add adapter, timeout, from-step, steps, exclude, and detach fields.
- **FR-013**: The TUI `DefaultFlags` MUST include a detach option.
- **FR-014**: The CLI `wave run --help` MUST group flags by tier (Essential, Execution, Continuous, Dev/Debug).
- **FR-015**: The from-step picker (WebUI and TUI) MUST be populated from the pipeline's manifest steps.
- **FR-016**: The adapter selector (WebUI and TUI) MUST be populated from available adapters.
- **FR-017**: Form validation MUST prevent submission when required fields are missing and show inline error messages.
- **FR-018**: `docs/reference/cli.md` MUST document all flags with tier groupings.
- **FR-019**: A running-pipelines guide MUST document TUI and WebUI run options.
- **FR-020**: `CHANGELOG.md` MUST include an entry for the run options parity changes.

### Key Entities

- **RunOptions**: The canonical set of all configuration options for a pipeline run. Defined authoritatively in `cmd/wave/commands/run.go:36-61`; all other surfaces map to this same set.
- **Tier**: A grouping level (1–4) that determines visibility and placement of run options across all surfaces. Tier 1 = always visible, Tier 2 = collapsible "Advanced", Tier 3 = collapsible "Continuous", Tier 4 = dev/debug-only (API and CLI only, not exposed in WebUI/TUI forms).
- **Timeout**: Integer field representing minutes. 0 means no timeout (infinite). Matches CLI `--timeout` semantics.
- **OnFailure**: String enum with valid values `halt` (default) | `skip`. Matches CLI `--on-failure` flag.
- **Delay**: Duration string (e.g., `5s`, `1m`, `30s`). Default `0s`. Used for continuous mode iteration spacing.
- **StartPipelineRequest**: The API contract for initiating a pipeline run from the WebUI. Defined in `internal/webui/types.go`. Must be extended to carry all Tier 1–4 options.
- **StartIssueRequest / StartPRRequest**: API contracts for initiating pipeline runs from issue/PR pages. Must carry Tier 1–3 options. Tier 4 is excluded — issue/PR workflows are user-facing and dev flags add complexity without value.
- **LaunchConfig**: The TUI's internal representation of run configuration (currently at `internal/tui/pipeline_messages.go:45-54`). Must be extended to carry Tier 1–3 options via typed fields rather than the current `Flags []string` approach.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Every Tier 1–3 option available in the CLI is also available in the WebUI pipeline detail form, the API request types, and the TUI launcher.
- **SC-002**: The WebUI pipeline detail page has zero modal dialogs for run initiation — all configuration is inline.
- **SC-003**: A user can start a detached continuous pipeline run with custom adapter, model, timeout, and on-failure settings from any of the four surfaces (CLI, TUI, WebUI pipeline detail, API).
- **SC-004**: The `wave run --help` output groups flags into exactly four named sections matching the canonical tier model.
- **SC-005**: Issues/PRs pages expose at minimum Adapter and Model overrides, plus Tier 2 via a collapsible section.
- **SC-006**: All form inputs validate before submission — no invalid requests reach the API handler.
- **SC-007**: Documentation (CLI reference, running-pipelines guide, CHANGELOG) reflects the full tiered model with consistent terminology.

## Clarifications

### C-001: Timeout unit and format
**Question**: The spec references "timeout" across all surfaces without specifying the unit or data type.  
**Resolution**: Timeout is an **integer representing minutes**, matching the CLI definition at `cmd/wave/commands/run.go:168`. Value `0` means no timeout. All surfaces (API, WebUI, TUI) must use the same integer-minutes convention.  
**Rationale**: The CLI is the canonical source; deviating would cause confusion when values are passed through the API to the subprocess.

### C-002: on-failure valid values
**Question**: The spec mentions `on_failure: "skip"` in one acceptance scenario but never enumerates all valid values.  
**Resolution**: Valid values are `halt` (default) and `skip`, matching the CLI flag description at `run.go:182`. The WebUI/TUI should present these as a dropdown/select with `halt` pre-selected.  
**Rationale**: The CLI only defines two values. Expanding the enum is out of scope for this feature.

### C-003: Tier 4 surface exposure
**Question**: FR-009 requires Tier 4 flags in the API, but no user story covers Tier 4 in WebUI/TUI. Is this intentional?  
**Resolution**: **Yes, intentional.** Tier 4 flags (mock, preserve_workspace, auto_approve, no_retro, force_model) are dev/debug tools. They are available via CLI flags and API request fields only. WebUI and TUI forms do not expose them — they add complexity to user-facing forms without matching user need.  
**Rationale**: Tier 4 is labeled "Dev/Debug" in the tier model. Exposing them in interactive forms would confuse non-developer users.

### C-004: StartIssueRequest / StartPRRequest — Tier 4 inclusion
**Question**: FR-011 says "Tier 1–3 at minimum" — does that mean Tier 4 should also be included?  
**Resolution**: **No.** Issue/PR request types carry Tier 1–3 only. The "at minimum" phrasing is clarified to mean "exactly Tier 1–3." Tier 4 flags are not applicable to issue/PR-triggered runs.  
**Rationale**: Issue/PR workflows are user-initiated from context pages where dev/debug flags have no use case. Keeping the types lean reduces API surface area.

### C-005: Mutual exclusion — continuous + from-step
**Question**: The spec does not address the interaction between `--continuous` and `--from-step`, but the CLI already validates they are mutually exclusive (`run.go:201-204`).  
**Resolution**: Added edge case: "continuous and from-step combined" is rejected with a clear error. All surfaces (WebUI, TUI, API) must enforce this mutual exclusion via form validation or server-side rejection.  
**Rationale**: Continuous mode iterates over work items; from-step resumes a single run at a specific point. Combining them is semantically meaningless, and the CLI already rejects it.
