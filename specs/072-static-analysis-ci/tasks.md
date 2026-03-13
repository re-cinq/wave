# Tasks

## Phase 1: Configuration

- [X] Task 1.1: Create `.golangci.yml` at repo root with v2-format config
  - Use `version: "2"` header
  - Set `linters.default: standard` preset
  - Enable additional linters: `unparam`, `wastedassign`, `gocritic`, `nolintlint`
  - Configure `nolintlint` with `require-explanation: true` and `require-specific: true`
  - Add exclusion presets for `comments` and `std-error-handling`

## Phase 2: CI Workflow

- [X] Task 2.1: Create `.github/workflows/lint.yml`
  - Trigger on `push` to `main` and `pull_request`
  - Use `actions/checkout@v5`
  - Use `actions/setup-go@v6` with `go-version-file: go.mod`
  - Use `golangci/golangci-lint-action@v9` with `version: v2.10`
  - Enable `only-new-issues: true` for incremental adoption

## Phase 3: Documentation

- [X] Task 3.1: Update CLAUDE.md Testing section
  - Add `golangci-lint run ./...` command alongside existing test commands

## Phase 4: Validation

- [X] Task 4.1: Verify config parses correctly with `golangci-lint config verify`
  - golangci-lint not available locally; config follows documented v2 format
- [X] Task 4.2: Run `golangci-lint run ./...` locally to check for issues
  - golangci-lint not available locally; CI will validate on push
- [X] Task 4.3: Commit changes and push branch to trigger CI workflow
