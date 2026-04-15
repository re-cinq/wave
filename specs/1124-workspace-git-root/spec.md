# bug: CreateWorkspace resolves paths against CWD instead of git root

**Issue**: https://github.com/re-cinq/wave/issues/1124
**Labels**: bug
**State**: OPEN
**Author**: nextlevelshit

## Issue Body

## Bug

`CreateWorkspace` resolves `.wave/workspaces/` relative to the process CWD instead of the git root. When tests run from `internal/pipeline/`, workspaces get created at `internal/pipeline/.wave/workspaces/` instead of `.wave/workspaces/`.

## Evidence

Found ~90+ misplaced `.wave/workspaces/` directories across three categories:

1. **Source packages** — `internal/pipeline/.wave/workspaces/`, `internal/webui/.wave/workspaces/`, `tests/integration/.wave/workspaces/`
2. **Nested inside worktree workspaces** — ~30 pipeline runs where sub-workspaces were created inside worktree CWD subdirectories
3. **Claude Code agent worktrees** — `.claude/worktrees/agent-*/internal/pipeline/.wave/workspaces/`

Each misplaced workspace is a full project worktree (git worktree with `.git` pointer), wasting disk and polluting the tree.

## Root Cause

`CreateWorkspace` computes the workspace base path from `os.Getwd()` (or step CWD) rather than always resolving to the git root.

## Fix (two parts)

1. **Code**: `CreateWorkspace` must resolve workspace paths against git root, never CWD
2. **`.gitignore`**: Add `**/.wave/workspaces/` wildcard to catch any future strays

## Verification

Check if the fix already landed — memory notes `git init -q` in mount/basic workspaces as a prior fix for path resolution. If git root resolution is already wired up, this may just need the `.gitignore` wildcard.

## Acceptance Criteria

- `NewWorkspaceManager` with a relative `baseDir` resolves that path against the git root, not process CWD
- Tests running from `internal/pipeline/` or any subdirectory do NOT create workspaces under those subdirectories
- `.gitignore` includes `**/.wave/workspaces/` to prevent future stray workspaces from polluting git index
- Unit tests cover the git-root resolution path in `NewWorkspaceManager`
