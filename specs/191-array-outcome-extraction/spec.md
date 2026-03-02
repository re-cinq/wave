# feat(pipeline): support array extraction in outcome json_path for multi-link results

**Issue**: [#191](https://github.com/re-cinq/wave/issues/191)
**Labels**: enhancement, pipeline
**Author**: nextlevelshit
**State**: OPEN

## Summary

Pipeline outcome definitions (`OutcomeDef`) currently extract a single URL via `json_path`, but steps can produce artifacts containing arrays of URLs. For example, the `gt-rewrite` pipeline enhances multiple issues but can only display one in the outcome summary because `json_path: .enhanced_issues[0].url` is hardcoded to index `[0]`.

This feature adds support for extracting multiple links from a JSON array in a single outcome definition, so all produced URLs appear in the pipeline output summary.

## Problem

In `internal/pipeline/types.go`, `OutcomeDef.JSONPath` resolves to a single string value. When a step produces an artifact with an array of results (e.g., `enhanced_issues` with 14 entries), only one can be extracted per outcome declaration. Pipeline authors must either:
- Hardcode `[0]` and lose visibility into other results
- Declare N separate outcome entries (not feasible when count is dynamic)

## Proposed Solution

Add array-aware extraction to `processStepOutcomes` in `internal/pipeline/executor.go`:
1. When `json_path` resolves to an array, iterate and register each element as a separate deliverable
2. Add an optional `json_path_label` field to `OutcomeDef` for labeling array items (e.g., `.enhanced_issues[*].issue_number`)
3. Preserve backward compatibility — scalar paths continue to work as before

## Affected Components

- `internal/pipeline/types.go` — `OutcomeDef` struct
- `internal/pipeline/outcomes.go` — `ExtractJSONPath` function
- `internal/pipeline/executor.go` — `processStepOutcomes` method
- `internal/display/outcome.go` — rendering of multiple links

## Acceptance Criteria

- [ ] `json_path` resolving to a JSON array of strings registers one deliverable per element
- [ ] `json_path` resolving to a JSON array of objects with a sub-path extracts the URL from each
- [ ] Scalar `json_path` values continue to work identically (backward compatible)
- [ ] Pipeline YAML `outcomes` section supports `[*]` glob syntax in `json_path`
- [ ] Outcome summary displays all extracted links, not just the first
- [ ] Unit tests cover array extraction, empty arrays, and mixed scalar/array paths

## Context

Observed during `gt-rewrite` pipeline run where 14 issues were enhanced but only 1 appeared in the outcome summary.
