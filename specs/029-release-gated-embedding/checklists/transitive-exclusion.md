# Transitive Exclusion Requirements Checklist

**Feature**: Release-Gated Pipeline Embedding
**Domain**: Transitive dependency resolution for contracts, schemas, and prompts
**Date**: 2026-02-11

---

## Reference Resolution

- [ ] CHK101 - Are all reference path formats documented? The spec identifies `schema_path` with `.wave/contracts/` prefix and `source_path` with `.wave/prompts/` prefix — are there other reference patterns in existing pipelines that could be missed? [Completeness]
- [ ] CHK102 - Does the spec define behavior when a `schema_path` uses a different prefix (e.g., `contracts/` without `.wave/`, or an absolute path)? Is prefix stripping robust to path variations? [Completeness]
- [ ] CHK103 - Does the spec account for `source_path` references that point outside `.wave/prompts/` (e.g., a project-relative path)? Should these be treated as external dependencies with no transitive exclusion impact? [Completeness]
- [ ] CHK104 - Is the normalization rule for contract keys (strip `.wave/contracts/` to get bare filename) correct for nested contract directories? Or are all contracts guaranteed to be flat? [Clarity]

## Inclusion Logic

- [ ] CHK105 - Does the spec clearly define the "referenced by at least one release pipeline" rule as set membership rather than reference counting? Is the intent unambiguous between counting references and checking presence? [Clarity]
- [ ] CHK106 - Is the persona exemption (FR-005, US2-3) requirement justified with sufficient rationale? Would an implementer understand WHY personas are always included while prompts are not? [Clarity]
- [ ] CHK107 - Does the spec define what happens to contracts/prompts that are referenced by zero pipelines (neither release nor non-release)? Are completely unreferenced assets included or excluded during init? [Coverage]
- [ ] CHK108 - Does the spec address contracts referenced via mechanisms other than `schema_path` (e.g., dynamically constructed paths at runtime)? [Coverage]

## Shared Resources

- [ ] CHK109 - Does US2-2 (shared contract between release and non-release pipeline) have a corresponding test requirement in the success criteria? [Consistency]
- [ ] CHK110 - Does the edge case for shared prompts (spec line 97) have a corresponding acceptance scenario or success criterion? [Consistency]
- [ ] CHK111 - Is the shared resource rule stated consistently across all asset types? Contracts say "shared contracts are preserved" (US2-2), prompts say "shared resources are preserved" (edge case) — but do the FRs capture this shared-resource principle explicitly? [Consistency]

## Algorithm Specification

- [ ] CHK112 - Is the two-phase algorithm (1: partition pipelines, 2: collect references) specified with enough detail for a single correct implementation, or could different interpreters produce different filtering results? [Clarity]
- [ ] CHK113 - Does the spec define whether the filtering algorithm should fail-fast or collect all errors when encountering unparseable pipeline YAML? [Completeness]
- [ ] CHK114 - Is the prefix-stripping approach robust to pipeline YAMLs that use Windows-style backslash paths or trailing slashes? [Coverage]
