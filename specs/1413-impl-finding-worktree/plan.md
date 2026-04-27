# Implementation Plan: 1413-impl-finding-worktree

## Objective

Replace `impl-finding`'s mount workspace with a per-child worktree workspace anchored on the PR head branch so commits actually push to `origin`. Restore `resolve-each` to `mode: parallel` once children no longer race on a shared working tree.

## Approach

1. Switch `internal/defaults/pipelines/impl-finding.yaml` from `workspace.mount` to `workspace.type: worktree`.
2. Use the proven `impl-issue-core` pattern: `branch: "{{ pipeline_id }}"` (per-child unique throwaway branch). The worktree inherits the parent repo's `origin` and full ref database.
3. Inject the parent's `pr-context` artifact into the child via `config.inject` on the parent's `resolve-each` step so the child can read the PR head branch name.
4. Rewrite the prompt's git steps: `git fetch origin <pr-branch>` then `git checkout -B <pr-branch> origin/<pr-branch>` inside the worktree. Drop the `cd /project` indirection — the worktree IS a real checkout.
5. Keep the `git ls-remote` push verification (already correct, just relocate to operate on the worktree's origin).
6. Flip `ops-pr-respond.yaml` `resolve-each.iterate.mode` from `serial` back to `parallel`. Remove the now-stale comment that pinned it to serial.

## File Mapping

### Modify

- `internal/defaults/pipelines/impl-finding.yaml`
  - Replace `workspace.mount` block with `workspace.type: worktree` + `branch: "{{ pipeline_id }}"`.
  - Rewrite Step 5 prompt section: drop `cd /project`, use worktree CWD, fetch + checkout PR branch, commit, push, verify.
  - Update Step 4 prompt note about CWD (worktree root, not workspace root).
  - Update doc comments in prompt header (Context section) about the working directory.

- `internal/defaults/pipelines/ops-pr-respond.yaml`
  - Add `config.inject: ["pr-context"]` to the `resolve-each` step so each `impl-finding` child receives the PR context artifact.
  - Flip `resolve-each.iterate.mode` from `serial` to `parallel`. Keep `max_concurrent` if present, or set a reasonable cap (e.g. 6) consistent with `parallel-review`.
  - Remove the `# Serial until impl-finding gets per-child workspace isolation (#1413)` comment.

### Add (tests only)

- `internal/pipeline/impl_finding_workspace_test.go` (or extend existing pipeline contract tests)
  - Assert `impl-finding.yaml` declares `workspace.type: worktree`.
  - Assert no `workspace.mount` block exists on the `apply-fix` step.
  - Assert `ops-pr-respond.yaml` `resolve-each` declares `config.inject` containing `pr-context`.
  - Assert `resolve-each.iterate.mode == parallel`.

### Delete

- None.

## Architecture Decisions

### A1. Reuse `branch: "{{ pipeline_id }}"` instead of templating PR branch into workspace.branch

**Why:** Step-output template references resolve only against the *current* pipeline's completed steps. `impl-finding` is invoked as an iterate child whose own pipeline has no `fetch-pr` step. Templating `{{ steps.fetch-pr.artifacts.pr-context.branch }}` in the child's workspace config would resolve against the child's empty step graph and fail.

`{{ pipeline_id }}` produces a unique branch name per child run (e.g. `impl-finding-20260427-...`), guaranteeing worktree isolation. The prompt then explicitly fetches and checks out the real PR head branch on top of that worktree's working copy. This matches the impl-issue-core pattern and keeps the failure modes well-understood.

### A2. Rely on `config.inject` for cross-pipeline artifact handoff

**Why:** With the mount gone, the child no longer sees `.agents/output/pr-context.json` via filesystem mount. `SubPipelineConfig.Inject` is the supported mechanism for parent-to-child artifact propagation. Wave's executor copies named parent artifacts into the child workspace before the child runs.

### A3. Keep `git ls-remote` verification

**Why:** Defense in depth. The worktree fix removes the silent-no-op-push class of bug, but pre-receive hooks, network glitches, or branch protection can still reject a push without the local commit knowing. Cross-checking the remote SHA is cheap and turns any future failure mode loud.

### A4. Restore parallel mode in same PR

**Why:** Issue acceptance criterion: "`resolve-each` can return to `mode: parallel`." Per-child worktrees fully isolate working trees, so parallel children no longer race. Splitting parallelism into a follow-up PR would leave the user-visible regression (serial findings = ~6× slower) unfixed despite the underlying race being solved.

## Risks

### R1. Worktree creation overhead per child

20 parallel `git worktree add` calls hit the parent repo's lockfile briefly. Git serializes `worktree add` against the index lock but each call is sub-second. **Mitigation:** keep `max_concurrent` at 6 (matches `parallel-review`) so worktree creation is bounded.

### R2. `config.inject` compatibility with iterate steps

The explore probe noted iterate-step children may not honour `step.Config.Inject`. **Mitigation:** verify by running `ops-pr-respond` end-to-end against a small representative PR before declaring done. If inject is silently ignored, fallback: have the child read PR branch from `{{ input }}` (the iterate item itself includes the branch context if we extend the input shape) — but this is the contingency, not the primary path.

### R3. Worktree cleanup on parallel failure

If a child errors mid-run, its worktree may stick around. Wave's worktree manager already handles cleanup on pipeline completion, but parallel partial-failure paths are less battle-tested. **Mitigation:** rely on existing `worktree.NewManager()` cleanup; document the manual `git worktree prune` recovery path in the resolution record on failure.

### R4. PR branch fetch races

Multiple children fetching the same `origin <pr-branch>` simultaneously is safe (git handles concurrent fetches), but if branch protection or auth drops a fetch, the child sees a stale local ref. **Mitigation:** the existing push verification step catches this — if local SHA differs from remote after push, the child fails loudly.

## Testing Strategy

### Unit / Pipeline Schema Tests

- Assert `impl-finding.yaml` schema: `workspace.type == worktree`, no `mount` block.
- Assert `ops-pr-respond.yaml` schema: `resolve-each.config.inject` contains `pr-context`, `iterate.mode == parallel`.
- Existing pipeline lint / contract validation (`go test ./internal/pipeline/...`) covers structural validity.

### Integration / Manual Validation

The spec's acceptance criterion is empirical: "A second run of `ops-pr-respond` on a representative PR produces commits that show up on `origin/<pr-branch>` without manual salvage." This requires an end-to-end run, not a Go test.

Plan:
1. Build the binary locally.
2. Pick a low-stakes test PR with ≥3 plausible findings.
3. Run `wave run ops-pr-respond <PR>`.
4. Verify on `origin/<pr-branch>`: every actionable resolution record's commit SHA appears in `git log origin/<pr-branch>` *and* matches `git ls-remote origin <pr-branch>` HEAD progression.
5. Repeat with `iterate.mode: parallel` to confirm no race regressions (concurrent file edits don't clobber each other).

Document the validation run id in the PR description.

### Regression Coverage

- `go test -race ./internal/pipeline/...` — guards against any data race introduced by the inject path.
- `golangci-lint run` — catches yaml-loader regressions if the schema changes propagate.
