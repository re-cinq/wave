# Work Items

## Phase 1: Setup
- [X] Item 1.1: Re-read `internal/defaults/pipelines/impl-issue-core.yaml` worktree pattern + `worktree.Manager.Create()` semantics so the new `impl-finding.yaml` matches existing conventions exactly.
- [X] Item 1.2: Confirm `config.inject` semantics on iterate-step sub-pipelines via a focused unit test or executor read-through.

## Phase 2: Core Implementation
- [X] Item 2.1: Rewrite `internal/defaults/pipelines/impl-finding.yaml` workspace block from `mount` to `type: worktree` + `branch: "{{ pipeline_id }}"`. [P]
- [X] Item 2.2: Rewrite Step 5 of the impl-finding prompt: drop `cd /project`, fetch + checkout PR branch into the worktree, commit, push, verify via `git ls-remote`. Update Step 4 CWD note. [P]
- [X] Item 2.3: Update `internal/defaults/pipelines/ops-pr-respond.yaml` `resolve-each` step: add `config.inject: ["pr-context"]`, flip `iterate.mode` from `serial` to `parallel`, set `max_concurrent: 6`, remove the stale `# Serial until ... #1413` comment. [P]

## Phase 3: Testing
- [X] Item 3.1: Add or extend a pipeline schema test asserting `impl-finding` declares worktree workspace and `ops-pr-respond` `resolve-each` injects `pr-context` + runs parallel.
- [X] Item 3.2: Run `go test -race ./internal/pipeline/...` and `go test ./...` locally; fix any breakage.
- [ ] Item 3.3: End-to-end manual validation: build binary, run `ops-pr-respond` against a small test PR, verify resolution-record commit SHAs land on `origin/<pr-branch>` and `git ls-remote` matches. (Deferred: requires live PR + admin approval — record run id in implementation PR description before merge.)

## Phase 4: Polish
- [X] Item 4.1: Update inline comments in both yaml files to reflect the new model (no more "shared mount race" caveats).
- [X] Item 4.2: Run `golangci-lint run` and `go vet ./...`.
- [ ] Item 4.3: Reference the validation run id (and resulting PR's `origin/<pr-branch>` log diff) in the implementation PR description. (Performed at PR-creation time, not in this commit.)
