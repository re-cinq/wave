# Requirements Quality Review: Persona Token Scoping

**Feature**: #213 — Persona Token Scoping
**Date**: 2026-03-16

## Completeness

- [ ] CHK001 - Does the spec define how `token_scopes` interacts with persona inheritance or composition if personas are ever reused across pipelines? [Completeness]
- [ ] CHK002 - Are error message formats for preflight scope violations fully specified (structured vs. free text, machine-parseable fields)? [Completeness]
- [ ] CHK003 - Is the caching strategy for token introspection results defined (per-pipeline-run, TTL, invalidation)? [Completeness]
- [ ] CHK004 - Does the spec define behavior when the same persona appears in multiple pipeline steps with potentially different runtime contexts? [Completeness]
- [ ] CHK005 - Are the `env_passthrough` configuration error messages and remediation hints specified with enough detail for implementation? [Completeness]
- [ ] CHK006 - Is the behavior defined when a token has MORE permissions than declared (overprivileged) — should there be a warning? [Completeness]
- [ ] CHK007 - Does the spec define the order of preflight checks (tools/skills first, then scopes) and what happens if both fail simultaneously? [Completeness]

## Clarity

- [ ] CHK008 - Is the distinction between "lint warning for unknown resources" (manifest load) vs. "warning for unknown forge" (runtime) clearly differentiated in requirements? [Clarity]
- [ ] CHK009 - Is the `PermissionSatisfies` hierarchy (`admin` ⊇ `write` ⊇ `read`) clearly defined for cross-resource scenarios (e.g., GitHub's `repo` scope granting both issues and pulls)? [Clarity]
- [ ] CHK010 - Are the fine-grained PAT "targeted API probes" approach sufficiently specified, or is the fallback behavior ambiguous? [Clarity]
- [ ] CHK011 - Is it clear whether `token_scopes: []` (empty list) is semantically different from omitting `token_scopes` entirely? [Clarity]
- [ ] CHK012 - Does FR-002 clearly define whether the canonical resource set is case-sensitive? [Clarity]

## Consistency

- [ ] CHK013 - Are the mapping tables in research.md consistent with the canonical resources defined in FR-002 and C4? [Consistency]
- [ ] CHK014 - Does the data model's `Validator` struct match the plan's Phase D description of constructor parameters? [Consistency]
- [ ] CHK015 - Is the `ValidationResult.Warnings` field in the data model consistent with FR-007's "emit a warning" language and the plan's event emission? [Consistency]
- [ ] CHK016 - Are the user story acceptance scenarios consistent with the functional requirements (e.g., US2-AC4 opt-in matches FR-010)? [Consistency]
- [ ] CHK017 - Is the task ordering in tasks.md consistent with the dependency graph implied by the plan's phases? [Consistency]
- [ ] CHK018 - Does the Gitea introspection approach in research.md (U5) align with T019's description in tasks.md? [Consistency]

## Coverage

- [ ] CHK019 - Are negative test scenarios specified for every acceptance criterion (not just happy paths)? [Coverage]
- [ ] CHK020 - Does the spec cover concurrent pipeline runs sharing the same token — any race conditions in introspection? [Coverage]
- [ ] CHK021 - Are accessibility/UX requirements defined for error output (color coding, terminal width, structured logging format)? [Coverage]
- [ ] CHK022 - Does the spec address performance impact of network-based token introspection on preflight latency (SC-001 says 5 seconds)? [Coverage]
- [ ] CHK023 - Are rollback/migration requirements defined for manifests that might have invalid `token_scopes` after schema changes? [Coverage]
- [ ] CHK024 - Does the spec cover the `--debug` flag behavior for token scope validation (what additional info is logged)? [Coverage]
- [ ] CHK025 - Are the success criteria (SC-001 through SC-006) all independently measurable without manual inspection? [Coverage]
