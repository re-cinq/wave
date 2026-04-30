# Work Items

## Phase 1: Setup
- [X] 1.1: Confirm branch `1582-reject-test-deletions` checked out from clean `main`
- [X] 1.2: Re-read `.agents/pipelines/impl-issue.yaml` and `internal/defaults/embedfs/pipelines/impl-issue.yaml` for current `implement` and `create-pr` step shapes

## Phase 2: Core Implementation
- [X] 2.1 [P]: Create `.agents/prompts/implement/judge-test-deletion.md`. Prompt instructs navigator to:
  1. Run `git diff --no-color main...HEAD -- '*_test.go'` from workspace root
  2. Write the captured output (or a clear "no test files changed" header if empty) to `.agents/output/test-diff.md`
  3. Emit no other side effects
- [X] 2.2 [P]: Edit `.agents/pipelines/impl-issue.yaml`:
  - Insert new step `judge-test-deletion` between `implement` and `create-pr`
    - `persona: navigator`, `model: cheapest`, `dependencies: [implement]`
    - `workspace.type: worktree`, same `branch: "{{ pipeline_id }}"`
    - `exec.source_path: .agents/prompts/implement/judge-test-deletion.md`
    - `output_artifacts: [{name: test-diff, path: .agents/output/test-diff.md, type: text}]`
    - `handover.contract`: `type: llm_judge`, `source: .agents/output/test-diff.md`, `model: cheapest`, single criterion (see plan.md), `threshold: 1.0`, `on_failure: rework`, `rework_step: fix-implement`
    - `retry.policy: standard`, `max_attempts: 2`
  - Update `create-pr.dependencies` to `[judge-test-deletion]`
- [X] 2.3 [P]: Mirror the YAML change in `internal/defaults/embedfs/pipelines/impl-issue.yaml`

## Phase 3: Testing
- [X] 3.1: Add a regression case in `internal/contract/llm_judge_test.go` covering: deletion-only diff fails, addition-only diff passes, replacement (delete + add) passes. Use the same stubbing pattern existing tests use (HTTP server stub).
- [X] 3.2: Run `go test ./internal/contract/... ./internal/pipeline/... -race` locally — pass.
- [X] 3.3: Run `go vet ./...` — pass. (`golangci-lint` not installed in this sandbox; skipped — CI runs it on the PR.)

## Phase 4: Polish
- [ ] 4.1 [P]: Document the new step in `docs/reference/pipeline-schema.md` — no new fields introduced; skipped.
- [ ] 4.2 [P]: Add a one-line note to `docs/guide/contracts.md` — deferred (file scope of this issue is pipeline + contract only).
- [ ] 4.3: Manual reproducer (acceptance #1) — out of scope for this commit; acceptance verified by unit test (`TestLLMJudge_TestDeletionGate/deletion-only diff fails the gate`) which proves the contract wiring rejects deletion-only diffs. Live `wave run` reproducer is a follow-up validation step.
- [ ] 4.4: Manual regression (acceptance #2) — covered by addition-only and replacement subtests; live regression run is a follow-up.
- [X] 4.5: Stage only the modified files (no `git add -A`) and commit with `feat(pipeline): reject net test deletions in impl-issue via llm_judge`.
