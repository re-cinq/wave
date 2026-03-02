# Tasks

## Phase 1: Core Extraction Engine

- [X] Task 1.1: Add `JSONPathLabel` field to `OutcomeDef` struct in `internal/pipeline/types.go`
  - Add `JSONPathLabel string \`yaml:"json_path_label,omitempty"\`` field
  - No changes to `Validate()` (field is optional)

- [X] Task 1.2: Add `ContainsWildcard` helper to `internal/pipeline/outcomes.go`
  - Returns `true` if a json_path string contains `[*]`
  - Simple `strings.Contains` check

- [X] Task 1.3: Implement `ExtractJSONPathAll` in `internal/pipeline/outcomes.go`
  - Split the path at `[*]` into prefix and suffix
  - Use prefix to navigate to the array field
  - Iterate each element, applying suffix sub-path extraction
  - Return `[]string` of extracted values
  - Empty arrays return `[]string{}`, not an error
  - Non-array values at the wildcard position return an error

## Phase 2: Outcome Processing Integration

- [X] Task 2.1: Update `processStepOutcomes` in `internal/pipeline/executor.go`
  - Detect `[*]` in `outcome.JSONPath` using `ContainsWildcard`
  - When wildcard detected: call `ExtractJSONPathAll` instead of `ExtractJSONPath`
  - For each extracted value, register a deliverable with the tracker
  - When `json_path_label` is set: extract label per item, format as `"<label>: <extracted_label>"`
  - When `json_path_label` is not set: format labels as `"<base_label> (1/N)"`, `"<base_label> (2/N)"`, etc.
  - Empty array result: log friendly message and skip (no error, no warning event)
  - Non-wildcard paths: existing scalar logic unchanged

## Phase 3: Testing

- [X] Task 3.1: Add `ExtractJSONPathAll` unit tests to `internal/pipeline/outcomes_test.go` [P]
  - Array of strings: `.urls[*]`
  - Array of objects with sub-path: `.enhanced_issues[*].url`
  - Nested prefix: `.result.items[*].name`
  - Empty array: `.items[*]` on `{"items": []}`
  - Non-array at wildcard: `.name[*]` on `{"name": "string"}`
  - Missing field: `.missing[*]` returns error
  - Invalid JSON: returns error
  - Path with only prefix, no suffix: `.items[*]` on string array

- [X] Task 3.2: Add `ContainsWildcard` tests to `internal/pipeline/outcomes_test.go` [P]
  - `.items[*].url` → true
  - `.items[0].url` → false
  - `.simple_field` → false
  - `[*]` alone → true

- [X] Task 3.3: Add `OutcomeDef` YAML parsing test for `json_path_label` in `internal/pipeline/types_test.go` [P]
  - Parse YAML with `json_path_label` field set
  - Parse YAML without `json_path_label` field (defaults to empty)

- [X] Task 3.4: Verify backward compatibility — run existing `ExtractJSONPath` tests unchanged [P]
  - All existing scalar extraction tests must pass without modification

## Phase 4: Polish

- [X] Task 4.1: Run full test suite (`go test ./...`) and fix any regressions
- [X] Task 4.2: Verify `go vet ./...` passes with no warnings
