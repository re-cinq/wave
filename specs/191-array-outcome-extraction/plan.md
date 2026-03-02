# Implementation Plan: Array Outcome Extraction

## Objective

Enable `OutcomeDef.JSONPath` to resolve arrays via `[*]` glob syntax, registering one deliverable per array element in `processStepOutcomes`, so pipeline outcome summaries display all extracted links instead of only the first.

## Approach

The implementation follows a layered strategy: first extend the JSON path extraction engine to understand `[*]` syntax, then update the outcome processing loop to handle multi-value results, and finally add a `json_path_label` field for labeling individual array items. Display code requires no structural changes — it already renders lists of deliverables. The key constraint is full backward compatibility: existing scalar paths must continue to produce identical behavior.

### Design: Two-Function Extraction

1. **`ExtractJSONPath`** (existing) — continues to return `(string, error)` for scalar paths. No signature change.
2. **`ExtractJSONPathAll`** (new) — returns `([]string, error)` for paths containing `[*]`. Called by `processStepOutcomes` when the path contains `[*]`.

This avoids breaking the existing `ExtractJSONPath` API while cleanly separating scalar vs. array extraction.

### `[*]` Semantics

- `[*]` means "iterate all elements of the array field"
- `.enhanced_issues[*].url` → for each element in `enhanced_issues`, extract `.url`
- `.urls[*]` → each element of `urls` treated as a scalar string
- Only one `[*]` per path is supported (no nested wildcards)
- Empty arrays return an empty slice (not an error)

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/pipeline/types.go` | modify | Add `JSONPathLabel` field to `OutcomeDef` struct |
| `internal/pipeline/outcomes.go` | modify | Add `ExtractJSONPathAll` function, add `ContainsWildcard` helper |
| `internal/pipeline/executor.go` | modify | Update `processStepOutcomes` to detect `[*]` and call array extraction |
| `internal/pipeline/outcomes_test.go` | modify | Add tests for `ExtractJSONPathAll`, wildcard paths, empty arrays, mixed paths |
| `internal/pipeline/types_test.go` | modify | Add YAML parsing test for `json_path_label` field |

### Files NOT Changed

- `internal/display/outcome.go` — already renders lists of `OutcomeLink` items; no changes needed
- `internal/deliverable/tracker.go` — already supports adding multiple deliverables per step; no changes needed

## Architecture Decisions

1. **New function rather than modifying `ExtractJSONPath` signature**: Keeps the existing single-value API intact. Callers that only need one value don't have to handle slices. `processStepOutcomes` is the only caller that needs multi-value extraction.

2. **`[*]` as the only wildcard syntax**: Matches common JSONPath convention. No need for more complex query syntax (e.g., `[?(@.status == "open")]`). Keeps the implementation simple.

3. **Single `[*]` per path**: Nested wildcards (e.g., `.a[*].b[*].c`) add significant complexity for minimal value. One wildcard per path covers the primary use case (iterating a top-level array of results).

4. **`json_path_label` as an optional field**: When present, it extracts a label per array item (e.g., issue number) to produce descriptive deliverable names like "Issue #42" instead of generic "issue (1/14)". When absent, items are labeled with their index.

5. **No max limit on array extraction**: The issue assessment noted this as missing info. Given that pipeline artifacts are typically small (tens of items, not thousands), no artificial limit is needed. The deliverable tracker and display already handle variable-length lists.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| `[*]` in paths that users intend as literal characters | Low | Low | `[*]` is not valid JSON key syntax; no ambiguity |
| Large arrays producing excessive deliverables | Low | Low | Pipeline artifacts are typically small; display already truncates |
| Parsing edge cases in `[*]` combined with nested dots | Medium | Medium | Thorough table-driven tests covering edge cases |
| Breaking existing `ExtractJSONPath` callers | Low | High | New function, no signature changes to existing one |

## Testing Strategy

### Unit Tests (`outcomes_test.go`)
- `ExtractJSONPathAll` with array of strings (`[*]`)
- `ExtractJSONPathAll` with array of objects and sub-path (`[*].url`)
- `ExtractJSONPathAll` with nested prefix path (`.result.items[*].url`)
- `ExtractJSONPathAll` with empty array → returns empty slice, no error
- `ExtractJSONPathAll` on non-array field → returns error
- `ExtractJSONPathAll` with invalid path → returns error
- `ContainsWildcard` helper function
- Existing `ExtractJSONPath` tests pass unchanged (backward compat)

### Unit Tests (`types_test.go`)
- `OutcomeDef` YAML parsing with `json_path_label` field
- `OutcomeDef.Validate` still works (no new required fields)

### Integration-Level Coverage
- `processStepOutcomes` with wildcard path registers multiple deliverables (covered by existing executor test patterns)
