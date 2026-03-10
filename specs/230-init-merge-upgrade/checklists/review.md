# Review Checklist: Init Merge & Upgrade Workflow

**Feature**: #230 | **Date**: 2026-03-04
**Artifacts**: spec.md, plan.md, data-model.md, research.md, tasks.md

---

## Completeness

- [ ] CHK001 - Are all four flag combinations (`init`, `--force`, `--merge`, `--merge --force`) explicitly specified with distinct behavior? [Completeness]
- [ ] CHK002 - Is the behavior of `wave init` on a brand-new project (no existing wave.yaml) specified, or only the existing-project case? [Completeness]
- [ ] CHK003 - Are error recovery paths defined for each step of the pre-mutation flow (compute → display → confirm → apply)? [Completeness]
- [ ] CHK004 - Is the output format of the change summary (table columns, status indicators, grouping) specified precisely enough to implement without ambiguity? [Completeness]
- [ ] CHK005 - Are all asset categories that should be compared explicitly enumerated (personas, pipelines, contracts, prompts)? Are there other categories (e.g., templates, skills) that should be included? [Completeness]
- [ ] CHK006 - Is the behavior specified when `wave init --merge` is run and the embedded defaults contain fewer files than the user's project (user has extra files not in defaults)? [Completeness]
- [ ] CHK007 - Does the spec define what happens when `wave init --merge --yes` and `wave init --merge --force` are combined simultaneously? [Completeness]
- [ ] CHK008 - Is the atomic write behavior for wave.yaml specified (e.g., write-to-temp-then-rename to prevent corruption on failure)? [Completeness]

## Clarity

- [ ] CHK009 - Is the distinction between "preserved" and "up to date" clear enough that a developer can implement the categorization without referring to clarifications? [Clarity]
- [ ] CHK010 - Is the deep-merge behavior for nested map keys unambiguous — specifically what happens when a user has a key at a different nesting depth than the default? [Clarity]
- [ ] CHK011 - Is "non-interactive terminal" detection clearly defined (which exact condition: no stdin TTY, no stdout TTY, or both)? [Clarity]
- [ ] CHK012 - Are the ANSI color choices for status indicators specified, or left to implementation? If specified, are they accessible for colorblind users? [Clarity]
- [ ] CHK013 - Is the manifest diff display format (dot-path notation) clearly defined with examples covering nested keys, array values, and new top-level sections? [Clarity]

## Consistency

- [ ] CHK014 - Are FR-007 (`--merge --force`) and FR-008 (`--merge --yes`) truly identical in all described behaviors, or are there edge cases where they diverge? [Consistency]
- [ ] CHK015 - Does clarification C-3 align with all acceptance scenarios in User Story 3? Specifically, US3-AS3 says `--merge --force` prints summary to stderr — does this match FR-007? [Consistency]
- [ ] CHK016 - Is the terminology consistent between spec.md and data-model.md for file statuses (new/preserved/up_to_date vs new/preserved/up-to-date)? [Consistency]
- [ ] CHK017 - Does the plan's Phase ordering (A→G) align with the task dependency ordering (T001→T019) in tasks.md? [Consistency]
- [ ] CHK018 - Are the success criteria (SC-001 through SC-007) each traceable to at least one functional requirement and one test task? [Consistency]
- [ ] CHK019 - Is the "Next steps" guidance in C-4 consistent with the actual `wave migrate` subcommand names (up/status/validate)? [Consistency]

## Coverage

- [ ] CHK020 - Are all 7 edge cases from the spec covered by at least one test task in tasks.md? [Coverage]
- [ ] CHK021 - Do the integration tests (T014) cover both the "abort at confirmation" path and the "confirm and proceed" path? [Coverage]
- [ ] CHK022 - Is there a test scenario for the case where embedded defaults add a completely new asset category not previously present in the project? [Coverage]
- [ ] CHK023 - Are concurrent execution scenarios considered (e.g., two `wave init --merge` processes running simultaneously on the same project)? [Coverage]
- [ ] CHK024 - Is the upgrade path from every supported prior version tested, or only from the immediately previous version? [Coverage]
- [ ] CHK025 - Does the documentation task (T017) cover troubleshooting guidance for common failure modes (permission errors, malformed YAML, missing database)? [Coverage]
