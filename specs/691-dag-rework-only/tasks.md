# Tasks

## Phase 1: Core Implementation

- [X] Task 1.1: Filter rework_only steps in `handlePipelineDetailPage` — add `if step.ReworkOnly { continue }` before appending to `dagSteps`
- [X] Task 1.2: Filter rework_only steps in run detail DAG builder in `handlers_runs.go` — same `continue` guard [P]
- [X] Task 1.3: Strip dangling dependency references — after filtering, remove any `Dependencies` entries that reference a filtered-out step ID [P]

## Phase 2: Testing

- [X] Task 2.1: Add test case in `dag_test.go` — pipeline with a rework-only step verifying it's excluded from the layout
- [X] Task 2.2: Run `go test ./internal/webui/...` to confirm no regressions

## Phase 3: Validation

- [X] Task 3.1: Run `go test ./...` full suite
- [X] Task 3.2: Verify `impl-issue` pipeline DAG no longer shows `fix-implement` at layer 0
