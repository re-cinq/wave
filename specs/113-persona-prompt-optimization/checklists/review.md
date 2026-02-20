# Requirements Quality Review Checklist

**Feature**: Persona Prompt Optimization (Issue #96)
**Spec**: `specs/113-persona-prompt-optimization/spec.md`
**Date**: 2026-02-20

## Completeness

- [ ] CHK001 - Does the spec define the exact content of the five base protocol sections (fresh memory, artifact I/O, workspace isolation, contract compliance, permission enforcement) with enough detail to write the file without interpretation? [Completeness]
- [ ] CHK002 - Does FR-002 specify the separator format unambiguously (e.g., `\n\n---\n\n` vs `---`)? Are both the plan and spec aligned on the exact separator string? [Completeness]
- [ ] CHK003 - Does the spec define how `wave init` should handle `base-protocol.md` — should it be listed in the personas section of the default manifest, or explicitly excluded? [Completeness]
- [ ] CHK004 - Does the spec define what happens when a persona prompt is loaded via `cfg.SystemPrompt` (inline) vs from a file — is the base protocol prepended in both cases? [Completeness]
- [ ] CHK005 - Are all 17 persona names listed explicitly and consistently across FR-010, the task list, and the data model? [Completeness]
- [ ] CHK006 - Does the spec define the behavioral constraints that SHOULD remain in persona prompts (per CLR-005/FR-009) with enough specificity to distinguish "keep" from "remove" during optimization? [Completeness]
- [ ] CHK007 - Does the spec define what "zero content duplication" means operationally — is it verbatim substring matching, semantic overlap, or structural similarity? [Completeness]
- [ ] CHK008 - Does the spec address the parity enforcement mechanism — is it a CI test, a Go test, or a manual check? Is the test location specified? [Completeness]
- [ ] CHK009 - Does the spec define how the token count heuristic (words x 100/75) should be applied — inclusive or exclusive of markdown syntax (headings, bullets, separators)? [Completeness]

## Clarity

- [ ] CHK010 - Is the distinction between "persona prompt" (the .md file content) and "persona" (the manifest configuration + prompt + adapter + model) clear and consistently used throughout the spec? [Clarity]
- [ ] CHK011 - Does FR-015 clearly delineate what changes are permitted to `prepareWorkspace` vs what is off-limits? Is the boundary precise enough to prevent scope creep or overly conservative implementation? [Clarity]
- [ ] CHK012 - Is the term "identity statement" defined clearly enough to distinguish it from a generic H1 heading? Does the spec provide criteria for what makes a good identity statement? [Clarity]
- [ ] CHK013 - Is the "output contract section" requirement (FR-005) clear about whether this refers to a markdown section heading or any content describing output format? [Clarity]
- [ ] CHK014 - Does the spec clearly distinguish between "generic process descriptions" (to remove) and "role-specific process descriptions" (to keep)? Are examples provided for borderline cases? [Clarity]
- [ ] CHK015 - Is the error handling requirement in FR-002 (missing base-protocol.md) specified clearly enough — what error type, what message, where is it surfaced? [Clarity]

## Consistency

- [ ] CHK016 - Does the token range in FR-008 (100-400) align with the ranges specified per-persona in the plan and data model? Are there any personas whose plan range exceeds 400? [Consistency]
- [ ] CHK017 - Does FR-012 (base protocol parity) align with FR-011 (persona parity) — are the same enforcement mechanisms specified for both? [Consistency]
- [ ] CHK018 - Are the acceptance scenarios in US1-US4 testable against the requirements as written, or do they introduce new requirements not captured in the FR list? [Consistency]
- [ ] CHK019 - Does the research finding (Unknown 3) that `GetPersonas()` will include `base-protocol.md` conflict with FR-015's statement about excluding it from the persona list? [Consistency]
- [ ] CHK020 - Are the anti-patterns listed in FR-006 ("Communication Style", "Domain Expertise", process sections) consistent with what the research identified as actually present in current persona files? [Consistency]
- [ ] CHK021 - Does the plan's D1 (injection point) description match the spec's FR-002 and CLR-003 exactly — same method name, same line numbers, same behavior? [Consistency]

## Coverage

- [ ] CHK022 - Does the spec address backward compatibility of the CLAUDE.md format — will tools or tests that parse CLAUDE.md break with the new base protocol preamble? [Coverage]
- [ ] CHK023 - Does the spec address what happens during `wave init` when upgrading from a pre-optimization version — will the new `base-protocol.md` be created alongside existing persona files? [Coverage]
- [ ] CHK024 - Does the spec address the interaction between the base protocol and the restriction section — could the concatenation produce ambiguous or conflicting instructions? [Coverage]
- [ ] CHK025 - Does the spec address how the base protocol interacts with custom/user-defined personas that may exist in `.wave/personas/`? [Coverage]
- [ ] CHK026 - Does the spec address whether the `base-protocol.md` content should be version-controlled or whether it can evolve independently of persona prompts? [Coverage]
- [ ] CHK027 - Does the spec address testing strategy for SC-008 (base protocol injection at runtime) — is it a unit test, integration test, or both? [Coverage]
- [ ] CHK028 - Does the spec cover the edge case where the base protocol file exists but is empty or malformed? [Coverage]
- [ ] CHK029 - Does the spec address the impact on existing adapter tests — will adding the base protocol prepend cause existing test assertions on CLAUDE.md content to fail? [Coverage]
- [ ] CHK030 - Does the spec address rollback strategy if the optimization causes behavioral regressions in pipeline execution that aren't caught by unit tests? [Coverage]
