# Implementation Plan: Fix CreateWorkspace git-root path resolution

## Objective

`NewWorkspaceManager` must resolve relative `baseDir` paths against the git repository root rather than the process CWD. This prevents test runs from subdirectories creating stray `.wave/workspaces/` trees inside source packages.

## Approach

### Current Behaviour

`NewWorkspaceManager` in `internal/workspace/workspace.go` takes `baseDir` (default `.wave/workspaces`) and calls `os.MkdirAll(baseDir, …)` directly. When the caller's CWD is `internal/pipeline/`, the directory is created at `internal/pipeline/.wave/workspaces/` — a misplaced tree that registers as a git worktree if the project uses git worktrees.

### Fix Strategy

1. **Add `resolveGitRoot()` helper** to `internal/workspace/workspace.go`  
   Uses `git rev-parse --show-toplevel` from process CWD. Falls back to process CWD on error (preserves current behaviour in environments without git).

2. **Update `NewWorkspaceManager`**  
   If `baseDir` is relative, join it with the resolved git root before `os.MkdirAll`. If `baseDir` is already absolute, leave it unchanged (callers that pass `/tmp/...` in tests are unaffected).

3. **Update `.gitignore`**  
   Add `**/.wave/workspaces/` to catch any stray workspace directories that might exist nested inside source subdirectories.

## File Mapping

| File | Action | Change |
|------|--------|--------|
| `internal/workspace/workspace.go` | modify | Add `resolveGitRoot()`, update `NewWorkspaceManager` to resolve relative baseDir against git root |
| `.gitignore` | modify | Add `**/.wave/workspaces/` wildcard after existing `.wave/workspaces/` entry |
| `internal/workspace/workspace_test.go` | modify | Add test for git-root resolution in `NewWorkspaceManager` |

## Architecture Decisions

- **No new package**: the helper belongs in `internal/workspace` alongside `NewWorkspaceManager`; it is package-private.
- **Fallback on error**: if `git rev-parse` fails (no repo, git not found), fall back to `os.Getwd()` so the function still works in non-git test environments that pass an absolute `tmpDir`.
- **Absolute baseDir is pass-through**: callers that already pass an absolute path (e.g., `workspace.NewWorkspaceManager(t.TempDir())` in tests) are unaffected.
- **No behaviour change for callers that use `workspace_root` from manifest**: the CLI (run.go, compose.go, etc.) passes `m.Runtime.WorkspaceRoot` which is typically `.wave/workspaces` — a relative path. After the fix it resolves to `<git-root>/.wave/workspaces`, which is the correct location.

## Risks

| Risk | Mitigation |
|------|-----------|
| `git` not on PATH in CI | Fall back to CWD — existing tests already mock the directory |
| Tests that pass `t.TempDir()` (absolute) are accidentally re-rooted | Absolute path check prevents this |
| `go test` called from a non-git working directory | Fallback to CWD preserves old behaviour; unit tests use `t.TempDir()` (absolute) so they're unaffected |

## Testing Strategy

- **Unit test in `workspace_test.go`**: create a temp git repo (`git init`), change to a subdirectory, call `NewWorkspaceManager(".wave/workspaces")`, verify the returned manager's `baseDir` points inside the git root, not the subdirectory.
- **Existing tests**: all existing `NewWorkspaceManager(tmpDir)` calls use absolute paths — unchanged.
- **`.gitignore` test**: not needed; `.gitignore` changes are verified by inspection and `git check-ignore`.
