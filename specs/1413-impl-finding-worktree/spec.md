# fix(pipeline): impl-finding mount workspace silently drops commits — git push has no remote

**Issue:** [re-cinq/wave#1413](https://github.com/re-cinq/wave/issues/1413)
**Labels:** bug
**State:** OPEN
**Author:** nextlevelshit

## Body

### Root cause: `impl-finding` mount workspace has no remote, so `git push` silently no-ops

#### Symptom

When `ops-pr-respond` was run on PR #1407 (run `ops-pr-respond-20260426-203623-b73e`), all 19 actionable findings were classified, all 19 `impl-finding` sub-pipelines ran to completion, and the comment-back step posted a resolution table claiming SHAs `8243ae93`, `e7a34a84`, etc. for each `f1`–`f19` fix.

But the PR head branch (`origin/1401-ops-pr-respond-v2`) saw none of those commits. Only the original PR commit + the unrelated `f6` AGENTS.md fix landed remotely.

The 19 commits did exist locally — each in its own isolated child workspace under `.agents/workspaces/impl-finding-*/apply-fix/`. They never reached `origin`.

#### Root cause

`internal/defaults/pipelines/impl-finding.yaml` declares the workspace as:

```yaml
workspace:
  mount:
    - source: ./
      target: /project
      mode: readwrite
```

A mount-based workspace creates a fresh `git init` at the workspace root and mounts the project filesystem at `/project`. The workspace's own git has:

- **No `origin` remote.**
- **No tracking branch.**
- A default branch name (`master` / `main`), not the PR head branch.

The pipeline's Step 5 then runs:

```bash
BRANCH=$(jq -r .branch .agents/output/pr-context.json)
CURRENT=$(git rev-parse --abbrev-ref HEAD)
if [ "$CURRENT" != "$BRANCH" ]; then
  echo "refusing to push..." >&2
  exit 1
fi
git add ...
git commit -m "fix(<id>): ..."
git push origin HEAD:"$BRANCH"
```

What actually happened:

1. The precondition check should have fired `exit 1` (workspace HEAD is `master`, PR branch is `1401-ops-pr-respond-v2`). Several children apparently bypassed it (the LLM rewrote or skipped the guard).
2. Even when the guard passed, `git push origin HEAD:<branch>` runs against the workspace's own git, which has no `origin` remote. The push silently no-ops with a non-error.
3. The local `git commit` succeeds (workspace git writes the commit), so the resolution record reports the commit SHA as if it landed.
4. The comment-back step reads the resolution records and posts the table — every entry shows `applied` because the local commit succeeded.

#### Why the mount layout exists

Mount workspaces are intentional for read-only steps where the agent inspects the project tree. They predate the worktree workspace mode. `fetch-pr` and several audit-* pipelines use mount appropriately.

`impl-finding` is the only pipeline that needs to **commit and push to the PR head branch** but uses mount.

#### Proposed fix

Change `impl-finding.yaml` to a worktree workspace that checks out the PR head branch. Existing pipelines (`impl-issue-core`, `impl-recinq`) already use this pattern.

A worktree gives a real git checkout with the same `origin` as the parent repo. `git push origin HEAD:branch` will then actually push.

If artifact templating is not available at workspace-config time, use the impl-issue pattern (`branch: "{{ pipeline_id }}"`) and have the prompt's first step run `git fetch origin <pr-branch>` + `git checkout <pr-branch>` explicitly.

#### Secondary issues (file separately if confirmed)

- **Race on shared mount**: 19 parallel `impl-finding` children all share `/project` read-write. They can clobber each other's working-tree state during the LLM-driven file edit phase.
- **Resolution record trusts local commit**: Step 6 records `Status: applied` whenever `git commit` succeeded, without checking that the push actually landed.
- **comment-back step does not verify remote**: it reads resolution.md per child and trusts the SHAs listed.

#### Validation

Hand-salvage of the 19 lost commits from this run: see commits `76d1264e`..`14ffa8ea` on `1401-ops-pr-respond-v2` (force-pushed 2026-04-26).

#### Source

Empirical baseline: `ops-pr-respond-20260426-203623-b73e` on PR #1407 (lost 19 commits, hand-recovered from child workspaces).

## Comments

**nextlevelshit:**
> Partial mitigation landed in PR #1407 (commit c30870ef): push step now runs against /project mount, push is verified via git ls-remote, origin-remote precondition added, resolve-each switched to mode: serial. Full fix (worktree-based isolated checkout per child to restore mode: parallel) still owned by this issue. Acceptance criteria unchanged.

## Acceptance Criteria

- `impl-finding` workspace is a worktree (or equivalent isolated git checkout) of the PR head branch.
- A second run of `ops-pr-respond` on a representative PR produces commits that show up on `origin/<pr-branch>` without manual salvage.
- Resolution records cross-check `git ls-remote origin <branch>` to confirm the commit pushed before reporting `applied`.
- `resolve-each` can return to `mode: parallel` without races on shared `/project` working tree.
