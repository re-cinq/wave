# Requirements Quality Review: Add --verbose Flag to Wave CLI

**Feature Branch**: `024-add-verbose-flag`
**Date**: 2026-02-06
**Artifacts Reviewed**: spec.md, plan.md, tasks.md, research.md, data-model.md

## Completeness

- [ ] CHK001 - Does FR-003 define the exact format or structure of verbose output lines (e.g., prefix, indentation, color), or is the rendering left unspecified? [Completeness]
- [ ] CHK002 - Is there a requirement specifying what verbose output the `status` and `clean` commands must display, comparable to the detail FR-003 provides for pipeline execution? [Completeness]
- [ ] CHK003 - Does the spec acknowledge that `list` and `artifacts` (mentioned in User Story 2's preamble) are explicitly excluded from FR-004's verbose scope, or is this an unstated gap between the story and the requirement? [Completeness]
- [ ] CHK004 - Is there a requirement specifying the behavior of `--verbose` when no pipeline steps produce verbose-eligible output (e.g., an empty pipeline or a single no-op step)? [Completeness]
- [ ] CHK005 - Does the plan address the `--verbose` combined with `--no-logs` edge case specified in the spec, or is it left unimplemented despite being a current (non-future) edge case? [Completeness]
- [ ] CHK006 - Is there a requirement defining how verbose output behaves during error conditions (e.g., a step fails mid-pipeline)? Does verbose mode show additional error context? [Completeness]
- [ ] CHK007 - Does the spec define whether `WAVE_VERBOSE=true` or an equivalent environment variable should activate verbose mode, consistent with how `--debug` may be activated via environment? [Completeness]
- [ ] CHK008 - Is there a requirement for verbose output during `wave run` dry-run or plan-only modes, if such modes exist? [Completeness]

## Clarity

- [ ] CHK009 - FR-001 states Cobra resolves the `-v` shorthand conflict by "letting the local flag shadow the persistent flag." Is there a test scenario that validates this shadowing behavior works as assumed for the project's Cobra version? [Clarity]
- [ ] CHK010 - FR-005 says "the system MUST use the higher detail level (debug)" when both flags are active. Is it clear whether verbose output is a strict subset of debug output, or could verbose emit information that debug does not? [Clarity]
- [ ] CHK011 - The spec says verbose output is emitted "by extending the existing event system" (FR-008). Is the mechanism sufficiently constrained — new event types, new fields on existing events, or additional rendering logic? [Clarity]
- [ ] CHK012 - User Story 2, Scenario 2 says verbose status shows "last state transition timestamps." Is the timestamp format defined or referenced (RFC 3339, Unix epoch, local time)? [Clarity]
- [ ] CHK013 - The edge cases for `--quiet` and `--format json` are marked "Future behavior." Is it clear to implementers that they should NOT implement these interactions now? [Clarity]
- [ ] CHK014 - Design Decision D-001 (plan) chooses "Bool Threading over VerbosityLevel Enum," but the spec defines `VerbosityLevel` as a key entity with three values. Is it clear which is authoritative — the spec's entity model or the plan's design decision? [Clarity]

## Consistency

- [ ] CHK015 - The spec defines `VerbosityLevel` as a key entity (normal, verbose, debug), but the plan uses a simple `verbose bool` field (D-001). Does the plan's simplification conflict with the spec's entity model or FR-005's composable hierarchy? [Consistency]
- [ ] CHK016 - FR-004 scopes non-pipeline verbose output to "validate, status, and clean," but User Story 2 explicitly names five commands in its preamble: "validate, status, list, clean, artifacts." Is this discrepancy between story and requirement documented? [Consistency]
- [ ] CHK017 - SC-001 says "any Wave command with --verbose produces additional output lines," but FR-004 only requires verbose output for three non-pipeline commands. Would commands outside that scope (e.g., `wave init --verbose`) fail SC-001? [Consistency]
- [ ] CHK018 - The plan lists four phases (A through D) but does not include any work for the non-TTY edge case specified in the spec. Is this an intentional omission or a gap between spec and plan? [Consistency]
- [ ] CHK019 - FR-002 says verbose propagation follows "the existing --debug flag pattern," but the data model introduces a new `WithVerbose` executor option. Is the propagation mechanism actually identical to debug, or is "follows the pattern" imprecise? [Consistency]

## Coverage

- [ ] CHK020 - Is there an acceptance scenario that tests the `-v` shorthand specifically (not just `--verbose`), to validate FR-001's shorthand requirement? [Coverage]
- [ ] CHK021 - Is there an acceptance scenario for the non-TTY/piped environment edge case, or is this only an edge case statement without a testable scenario? [Coverage]
- [ ] CHK022 - Is there an acceptance scenario verifying that verbose output goes to stderr and not stdout (FR-008's dual-stream requirement)? [Coverage]
- [ ] CHK023 - Do the acceptance scenarios for User Story 3 cover the full interaction matrix (verbose-only, debug-only, both, neither), or only two of the four combinations? [Coverage]
- [ ] CHK024 - Is there an acceptance scenario that validates SC-003 (help documentation), or is this success criterion untested by any scenario? [Coverage]
- [ ] CHK025 - Does the testing plan cover the edge case where global `--verbose` and local `--verbose` on `validate` are both specified simultaneously? [Coverage]
- [ ] CHK026 - Is there an acceptance scenario for verbose output during a multi-step pipeline where some steps succeed and some fail, to verify partial verbose output is still useful? [Coverage]
