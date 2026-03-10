# Requirements Quality Review: Pipeline Composition UI (#261)

**Feature**: `261-tui-compose-ui` | **Date**: 2026-03-07
**Scope**: Cross-cutting quality validation of spec.md, plan.md, tasks.md, and data-model.md

## Completeness

- [ ] CHK001 - Are all keyboard interactions documented with exact key bindings (modifier + key) for every compose mode action? [Completeness]
- [ ] CHK002 - Are error states defined for every user action that can fail (e.g., loading a pipeline for compose, malformed pipeline YAML)? [Completeness]
- [ ] CHK003 - Is the behavior specified for when the manifest changes while compose mode is open (e.g., pipelines added/removed externally)? [Completeness]
- [ ] CHK004 - Are focus transitions between left pane, right pane, and picker explicitly defined with all state transitions? [Completeness]
- [ ] CHK005 - Is the behavior specified for the `s` key when compose mode is already open (re-entry or no-op)? [Completeness]
- [ ] CHK006 - Are the visual styling requirements defined for compose mode elements (colors, borders, indicators) or is this delegated to existing patterns? [Completeness]
- [ ] CHK007 - Is the maximum sequence length defined or explicitly stated as unbounded? [Completeness]
- [ ] CHK008 - Are all acceptance scenarios from User Stories 1-5 traceable to at least one functional requirement (FR-001 through FR-015)? [Completeness]
- [ ] CHK009 - Is the loading/resolution behavior specified for the full pipeline picker (how available pipelines are enumerated and loaded)? [Completeness]
- [ ] CHK010 - Are accessibility requirements beyond terminal width (e.g., screen reader, high contrast) explicitly scoped in or out? [Completeness]

## Clarity

- [ ] CHK011 - Is the artifact matching algorithm unambiguous — does it specify what happens when multiple outputs share the same name? [Clarity]
- [ ] CHK012 - Is the "last step" and "first step" heuristic clearly defined — do multi-branch pipelines with parallel final steps have a deterministic "last step"? [Clarity]
- [ ] CHK013 - Is the distinction between `inject_artifacts` and `output_artifacts` naming clear enough for implementers to locate the correct pipeline types? [Clarity]
- [ ] CHK014 - Is the phrase "modal state within the Pipelines view" sufficiently precise — does it define which ContentModel fields change and which are preserved? [Clarity]
- [ ] CHK015 - Is the confirmation prompt behavior (FR-008) specified with enough detail — what text, what options, what happens on timeout/resize? [Clarity]
- [ ] CHK016 - Are the status indicator symbols (✓, ⚠, ✗) specified as requirements or as implementation suggestions? [Clarity]
- [ ] CHK017 - Is the "informational message" for #249 dependency (FR-013) specified with content requirements or left to implementer discretion? [Clarity]
- [ ] CHK018 - Does the spec clearly define whether `Enter` on a sequence item (focus detail) vs `Enter` in compose mode (start sequence) are distinguishable by context? [Clarity]

## Consistency

- [ ] CHK019 - Is the compose mode state machine consistent with existing modal states (stateConfiguring, stateRunningLive) in terms of Tab-blocking, Esc behavior, and focus handling? [Consistency]
- [ ] CHK020 - Is the `s` key binding consistent with the existing keybinding scheme — does it conflict with any current action in the Pipelines view? [Consistency]
- [ ] CHK021 - Is the `wave compose` CLI command's output format consistent with other Wave CLI commands (e.g., `wave run`, `wave init`)? [Consistency]
- [ ] CHK022 - Does the spec reference the correct type names from `internal/pipeline/types.go` (ArtifactDef, ArtifactRef, Memory.InjectArtifacts)? [Consistency]
- [ ] CHK023 - Is the "grouped running display" (FR-010) specification consistent with the existing Running section rendering in `PipelineListModel`? [Consistency]
- [ ] CHK024 - Are the clarification resolutions (C1-C5) all reflected back into the main requirements section, or do some exist only in the clarifications section? [Consistency]
- [ ] CHK025 - Is the plan's file list consistent with the task assignments (every task references a file that exists in the plan's project structure)? [Consistency]

## Coverage

- [ ] CHK026 - Are there test scenarios for compose mode interaction with concurrent pipeline events (pipeline finishes while composing, pipeline starts externally)? [Coverage]
- [ ] CHK027 - Are there edge case specifications for pipelines with zero steps or steps with both inject and output artifacts on the same step? [Coverage]
- [ ] CHK028 - Is the behavior specified for very long pipeline names that exceed the left pane width in compose mode? [Coverage]
- [ ] CHK029 - Is the behavior specified for sequences where the same boundary has both compatible and incompatible artifacts? [Coverage]
- [ ] CHK030 - Are there specifications for the compose mode's interaction with the existing filter/search feature in the pipeline list? [Coverage]
- [ ] CHK031 - Is the `--json` output flag (from #260 CLI compliance) accounted for in the `wave compose` command? [Coverage]
- [ ] CHK032 - Are there requirements for compose mode behavior during terminal resize events? [Coverage]
- [ ] CHK033 - Is the behavior specified when the user presses `s` rapidly or spams key inputs during compose mode transitions? [Coverage]
