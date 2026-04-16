# Tasks

## Phase 1: Core Fix

- [X] Task 1.1: Add `resolveGitRoot()` helper to `internal/workspace/workspace.go`
  - Runs `git rev-parse --show-toplevel` via `os/exec`
  - Falls back to `os.Getwd()` on error
  - Package-private function

- [X] Task 1.2: Update `NewWorkspaceManager` to resolve relative `baseDir` against git root
  - If `baseDir` is empty, default to `.wave/workspaces`
  - If `baseDir` is relative, call `resolveGitRoot()` and join
  - If `baseDir` is absolute, leave unchanged

## Phase 2: .gitignore Safety Net

- [X] Task 2.1: Add `**/.wave/workspaces/` wildcard to `.gitignore`
  - Insert after existing `.wave/workspaces/` entry
  - Catches any stray nested workspace directories

## Phase 3: Tests

- [X] Task 3.1: Add unit test for git-root resolution in `internal/workspace/workspace_test.go`
  - Create temp dir, `git init` inside it
  - Create a subdirectory, call `NewWorkspaceManager(".wave/workspaces")` from that subdir
  - Assert workspace is created under the git root, not the subdir

- [X] Task 3.2: Verify existing tests still pass (`go test ./internal/workspace/...`)

## Phase 4: Validation

- [X] Task 4.1: Run `go test -race ./internal/workspace/...` and confirm no failures
- [X] Task 4.2: Run `go vet ./internal/workspace/...`
