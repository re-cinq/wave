# Tasks

## Phase 1: Implementation

- [X] Task 1.1: Add "Logs vs Progress Output" section to `docs/reference/cli.md` after the `wave logs` section (~line 268), including:
  - A brief intro explaining the two mechanisms
  - A comparison table (mechanism, data source, timing, typical use)
  - 3 use-case examples with command snippets:
    1. Debugging a failed step (use `wave logs`)
    2. Watching a pipeline run live (use `--output text -v`)
    3. Scripting/CI integration (use `--output json` for real-time, `wave logs --format json` for post-hoc)

## Phase 2: Validation

- [X] Task 2.1: Run `go test ./...` to confirm no regressions
- [X] Task 2.2: Review markdown rendering for consistency with existing CLI reference style
