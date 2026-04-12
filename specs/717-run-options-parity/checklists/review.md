# Quality Review Checklist: 717-run-options-parity

**Generated**: 2026-04-11 | **Scope**: Cross-artifact requirements quality

## Completeness

- [x] CHK001 - Does every RunOptions field from the canonical struct (run.go:36-61) have at least one FR requiring its exposure? [Completeness]
- [x] CHK002 - Are all 5 surfaces (CLI, TUI, WebUI pipeline detail, WebUI issues/PRs, API) covered by at least one user story? [Completeness]
- [x] CHK003 - Does each collapsible section (Advanced, Continuous) have its member options enumerated in the spec, not just named? [Completeness]
- [x] CHK004 - Are default values specified for every option that has one (on-failure=halt, timeout=0, delay=0s, max_iterations=0)? [Completeness]
- [x] CHK005 - Does the spec define what happens on form submission for each navigation outcome (detach → navigate, dry-run → inline, normal → navigate+stream)? [Completeness]
- [x] CHK006 - Is the Tier 4 exclusion boundary explicitly stated for every surface that omits it (WebUI forms, TUI forms, Issue/PR requests)? [Completeness]
- [ ] CHK007 - Are error message requirements specified for each validation rule (mutual exclusion, overlap, missing required fields), or only the fact that errors must occur? [Completeness]
- [x] CHK008 - Does the spec define the population source for dynamic selectors (from-step picker → manifest steps, adapter selector → available adapters)? [Completeness]
- [ ] CHK009 - Are loading/disabled states defined for selectors that depend on async data (adapter list, step list for from-step picker)? [Completeness]
- [x] CHK010 - Does every acceptance scenario have a Then clause that is concretely verifiable (no "works correctly" or "is handled")? [Completeness]

## Clarity

- [x] CHK011 - Is the timeout unit (integer minutes) stated unambiguously and consistently across all surfaces in the spec? [Clarity]
- [x] CHK012 - Is the delay format (duration string e.g. "5s") defined with enough precision for implementers to validate input? [Clarity]
- [x] CHK013 - Are the on-failure enum values exhaustively listed with their behavioral definitions? [Clarity]
- [x] CHK014 - Is "inline" form placement defined relative to existing page structure (replaces modal vs. added alongside existing content)? [Clarity]
- [x] CHK015 - Does the Force field conditional visibility rule specify the exact trigger (from-step has ANY value vs. a VALID value)? [Clarity]
- [ ] CHK016 - Is the "runs indefinitely" warning for max_iterations=0 specified as a requirement (FR/acceptance scenario) or only mentioned in edge cases? [Clarity]
- [x] CHK017 - Is "detach + dry-run precedence" defined clearly enough that implementers know which field to check first? [Clarity]
- [x] CHK018 - Are the 0 NEEDS CLARIFICATION markers confirmed by the clarify step resolving all ambiguities? [Clarity]

## Consistency

- [x] CHK019 - Do the data model field names match the JSON contract field names match the spec terminology (e.g., on_failure vs on-failure vs OnFailure)? [Consistency]
- [x] CHK020 - Does the plan's entity list match the spec's Key Entities section (same count, same names)? [Consistency]
- [x] CHK021 - Do the task phases map 1:1 to user stories in the spec (each US has a corresponding phase)? [Consistency]
- [x] CHK022 - Are tier numbers used consistently (never "Tier A" or "Basic tier" — always "Tier 1" through "Tier 4")? [Consistency]
- [x] CHK023 - Do contract JSON schemas (contracts/*.json) match the data model target state field sets exactly? [Consistency]
- [x] CHK024 - Does every FR referenced in a success criterion have a corresponding task in tasks.md? [Consistency]
- [x] CHK025 - Are validation rules in data-model.md reflected as edge cases AND as tasks in tasks.md? [Consistency]
- [x] CHK026 - Is the mutual exclusion (continuous + from-step) specified identically across spec edge cases, data model validation rules, and task descriptions? [Consistency]

## Coverage

- [x] CHK027 - Does the task list include verification tasks for each edge case in the spec (9 edge cases → at least 9 verification tasks or grouped equivalents)? [Coverage]
- [x] CHK028 - Are there test tasks for both client-side (form) and server-side (handler) validation of the same rules? [Coverage]
- [x] CHK029 - Does the spec cover backward compatibility for API consumers sending requests without the new optional fields? [Coverage]
- [x] CHK030 - Are accessibility requirements addressed for the new WebUI form elements (collapsible sections, conditional fields)? [Coverage]
- [ ] CHK031 - Is the behavior of the from-step picker when the pipeline has zero steps defined as a requirement (not just a TUI edge case)? [Coverage]
- [x] CHK032 - Does the spec define behavior for the "source" field in continuous mode when left empty? [Coverage]
- [x] CHK033 - Are the PR handler requirements (new endpoint POST /api/prs/start) covered by both a user story AND explicit FRs? [Coverage]
- [ ] CHK034 - Is the steps/exclude input format defined (comma-separated) with validation rules for the WebUI and TUI, not just referenced from CLI behavior? [Coverage]
