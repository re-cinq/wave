# Requirements Quality Review Checklist

**Feature**: Release-Gated Pipeline Embedding
**Spec**: `specs/029-release-gated-embedding/spec.md`
**Date**: 2026-02-11

---

## Completeness

- [ ] CHK001 - Does the spec define behavior for every combination of `release` and `disabled` field values (true/true, true/false, false/true, false/false, absent/absent)? [Completeness]
- [ ] CHK002 - Does FR-003 specify what happens when a contract is referenced by a release pipeline via an inline schema (not `schema_path`)? Is transitive exclusion only for file-based references? [Completeness]
- [ ] CHK003 - Does the spec define what `wave init --all --merge` does when a release pipeline already exists but has been locally modified by the user? [Completeness]
- [ ] CHK004 - Is the expected warning message for "zero release pipelines" (FR-011) specified with enough detail for implementation (message text, output stream, exit code)? [Completeness]
- [ ] CHK005 - Does the spec define whether `GetReleasePipelines()` returns an error or empty map when YAML unmarshalling fails for a single pipeline? [Completeness]
- [ ] CHK006 - Does the spec address what happens when `schema_path` contains a path that doesn't use the `.wave/contracts/` prefix (e.g., a relative path or absolute path)? [Completeness]
- [ ] CHK007 - Are all artifact types that could be transitively excluded explicitly enumerated? The spec covers contracts, schemas, and prompts — are there other embedded asset types (e.g., templates) that need consideration? [Completeness]
- [ ] CHK008 - Does the spec define the expected `--all` flag help text content, or only that it must exist (SC-007)? [Completeness]

## Clarity

- [ ] CHK009 - Is the distinction between "contracts" and "schemas" consistently used? The spec sometimes says "contracts/schemas" as if they are different, and sometimes uses them interchangeably. Are these the same asset type? [Clarity]
- [ ] CHK010 - Is the term "transitive exclusion" well-defined? The dependencies are one level deep (pipeline → contract), not truly transitive. Could this terminology confuse implementers into building a recursive graph resolver? [Clarity]
- [ ] CHK011 - Is it clear whether `GetReleasePipelines()` should be a standalone function or a method, and what package it belongs to? The research says `embed.go` but the spec says "defaults subsystem" (FR-007). [Clarity]
- [ ] CHK012 - Does the spec clearly distinguish between "pipeline is not written" vs "pipeline file is deleted" for merge scenarios? The edge case says "existing files should be preserved" — does "preserved" mean untouched or re-validated? [Clarity]
- [ ] CHK013 - Is the acceptance scenario for US3-4 (invalid boolean `"yes"`) clear about what "validation error" means — a Go YAML unmarshal error, a custom validation error, or a user-facing CLI error? [Clarity]
- [ ] CHK014 - Does the spec clearly state where the transitive exclusion logic lives (init.go vs embed.go vs a new package)? Research says init.go, but is this reflected in the spec? [Clarity]

## Consistency

- [ ] CHK015 - Are FR-003 and FR-004 consistent in how they describe the exclusion mechanism? FR-003 says "transitively exclude contracts/schemas" while FR-004 says "transitively exclude prompt files" — does the transitive exclusion algorithm apply identically to both? [Consistency]
- [ ] CHK016 - Is the `omitempty` tag on `Release` consistent with the requirement that `release: false` should be distinguishable from "no release field"? With `omitempty`, both marshal to the same YAML. Does this matter for any use case? [Consistency]
- [ ] CHK017 - Does US2-3 (personas never excluded) align with FR-005? Both say personas are always included, but does the spec explain WHY personas are exempt while prompts are not? Is the rationale sufficient to prevent future confusion? [Consistency]
- [ ] CHK018 - Is CLR-005 (filtered counts in `printInitSuccess`) reflected in the functional requirements or success criteria, or is it only a clarification? Should it be promoted to an FR? [Consistency]
- [ ] CHK019 - Does the spec consistently treat `--merge` behavior? The edge case section discusses it, CLR-004 resolves it, but no FR explicitly covers merge + release filtering. [Consistency]
- [ ] CHK020 - Are the success criteria (SC-001 through SC-008) fully traceable to functional requirements (FR-001 through FR-011)? Is every FR covered by at least one SC? [Consistency]

## Coverage

- [ ] CHK021 - Does the spec cover error handling for malformed pipeline YAML during release filtering? What if one pipeline fails to parse — should it be skipped, treated as non-release, or should the entire init fail? [Coverage]
- [ ] CHK022 - Does the spec address backward compatibility of the `metadata.release` field for pipeline YAMLs that exist outside the embedded defaults (user-authored pipelines in `.wave/pipelines/`)? [Coverage]
- [ ] CHK023 - Does the spec cover the interaction between release filtering and the pipeline executor? If a non-release pipeline is somehow present in `.wave/pipelines/`, can it still be executed via `wave run`? [Coverage]
- [ ] CHK024 - Are test requirements specific enough? FR-010 says "existing tests MUST be updated" but doesn't specify what assertions should be added. SC-005 and SC-006 describe minimal test coverage — is this sufficient? [Coverage]
- [ ] CHK025 - Does the spec address logging/observability for the filtering process? Should filtered-out pipelines be logged at debug level for troubleshooting? [Coverage]
- [ ] CHK026 - Does the spec cover what happens when embedded contract files exist that are NOT referenced by any pipeline (neither release nor non-release)? Are orphaned contracts included or excluded? [Coverage]
- [ ] CHK027 - Does the spec address performance considerations for the filtering step? With N pipelines each having M steps, is the O(N*M) parsing overhead acceptable? Are there constraints on embedded pipeline count? [Coverage]
