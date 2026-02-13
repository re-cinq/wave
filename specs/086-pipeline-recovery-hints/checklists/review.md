# Requirements Quality Review Checklist

**Feature**: Pipeline Recovery Hints on Failure
**Spec**: `specs/086-pipeline-recovery-hints/spec.md`
**Date**: 2026-02-13

## Completeness

- [ ] CHK001 - Are all four user stories (resume, force, workspace, debug) fully specified with acceptance scenarios that cover both the positive and negative paths? [Completeness]
- [ ] CHK002 - Does the spec define what happens when multiple error types apply simultaneously (e.g., a contract validation error that also has an empty message)? [Completeness]
- [ ] CHK003 - Is the behavior of recovery hints when a pipeline is invoked programmatically (not via CLI) specified or explicitly scoped out? [Completeness]
- [ ] CHK004 - Does the spec define the exact ordering of hints within the recovery block, or only that they are "ordered"? [Completeness]
- [ ] CHK005 - Are success criteria (SC-001 through SC-006) each traceable to at least one functional requirement and one acceptance scenario? [Completeness]
- [ ] CHK006 - Does the spec address the behavior when the executor returns multiple step failures (e.g., in a parallel execution model)? [Completeness]
- [ ] CHK007 - Is the workspace path pattern (`.wave/workspaces/<runID>/<stepID>/`) defined consistently between the spec, data model, and research documents? [Completeness]

## Clarity

- [ ] CHK008 - Is the distinction between "recovery hints" (CLI-layer, this feature) and "error messages" (executor-layer, `ErrorMessageProvider`) clearly articulated in the spec? [Clarity]
- [ ] CHK009 - Does FR-006 ("no more than 8 lines") define what constitutes a "line" — is a two-line hint (label + command) counted as 1 or 2 lines? [Clarity]
- [ ] CHK010 - Is the `--force` hint label ("Resume and skip validation checks") unambiguous about which validations are skipped (contract, phase, stale-artifact)? [Clarity]
- [ ] CHK011 - Does FR-008 clearly specify which event type (e.g., `step_failed`, `pipeline_failed`) carries the `recovery_hints` field in JSON mode? [Clarity]
- [ ] CHK012 - Is the shell escaping algorithm (POSIX single-quote wrapping) described with enough detail for an implementer to produce the correct output for all edge cases? [Clarity]
- [ ] CHK013 - Is the term "ambiguous error" in User Story 4 defined precisely enough to distinguish it from a "runtime error" in the classification scheme? [Clarity]

## Consistency

- [ ] CHK014 - Does the error classification scheme (contract_validation, security_violation, runtime_error, unknown) in the spec match the classification logic described in the plan and data model? [Consistency]
- [ ] CHK015 - Is the `--from-step` flag name used consistently across all user stories, edge cases, and functional requirements? [Consistency]
- [ ] CHK016 - Does the spec's statement that hints are generated "from context already available at the call site" (FR-010) align with the plan's decision to build hints in `run.go`? [Consistency]
- [ ] CHK017 - Is the `RecoveryHint` struct shape (Label, Command, Type) consistent between the spec's Key Entities section, the data model, and the plan's Event extension? [Consistency]
- [ ] CHK018 - Does the plan's dependency graph (T001-T015) correctly reflect the implementation order described in the plan's Step 1-8 narrative? [Consistency]
- [ ] CHK019 - Are the edge cases listed in the spec (empty input, flag-style invocation, re-resume, cleaned workspace, JSON mode, quiet mode, single-step) all covered by at least one task in tasks.md? [Consistency]

## Coverage

- [ ] CHK020 - Does the spec define requirements for accessibility of recovery hints (e.g., screen reader compatibility, color usage)? [Coverage]
- [ ] CHK021 - Are there requirements for internationalization or localization of recovery hint labels? [Coverage]
- [ ] CHK022 - Does the spec address performance impact of recovery hint generation on the error path (latency, memory allocation)? [Coverage]
- [ ] CHK023 - Is the behavior defined for when the `wave` binary is not in the user's PATH and the recovery command uses a bare `wave` invocation? [Coverage]
- [ ] CHK024 - Does the spec define how recovery hints interact with the `--no-color` flag or terminals that don't support ANSI formatting? [Coverage]
- [ ] CHK025 - Are there requirements for testing recovery hints in CI/CD environments where stderr may be redirected or suppressed? [Coverage]
