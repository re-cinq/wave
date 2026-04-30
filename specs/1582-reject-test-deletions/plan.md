# Implementation Plan — Reject Net Test Deletions

## Objective

Block the `impl-issue` pipeline from accepting an implementation that net-removes `func Test*` declarations without a demonstrable replacement (rename, migration, or replacement test). Failure mode reproduced in run `impl-issue-20260429-221616-d8e4` on issue #1580.

## Approach

Add a small navigator step `judge-test-deletion` between `implement` and `create-pr` in `.agents/pipelines/impl-issue.yaml` (and the embedfs mirror). The step:

1. Captures the diff of `_test.go` files (vs. base branch) into an artifact file the LLM judge can read — `llm_judge` reads `cfg.Source` from disk; it does not (yet) auto-inject `git_diff` like `agent_review` does.
2. Runs an `llm_judge` contract with criteria framed as a binary question on net-removed `func Test*` declarations.
3. On failure, reworks via the existing `fix-implement` step.

This avoids modifying core contract code (`internal/contract/llm_judge.go`) and keeps the change isolated to YAML + a tiny prompt.

### Why a separate step (not just an additional contract on `implement`)

`implement` runs the craftsman with `strongest` model and large token budget. Re-running it on every judge-failure (the contract retry path) wastes budget and risks unrelated diff churn. A small navigator step is cheaper, focused, and lets `fix-implement` handle the rework (matches the existing `implement → fix-implement` rework wiring).

### Why `llm_judge` and not `agent_review`

The issue text explicitly proposes `llm_judge`. `llm_judge` returns a structured pass/fail per criterion with a numeric threshold — well-suited for a single binary check. `agent_review` is criteria-doc + prose review; less appropriate for a yes/no gate.

## File Mapping

### Created
- `.agents/prompts/implement/judge-test-deletion.md` — prompt that runs `git diff --no-color {{ base_branch }}...HEAD -- '*_test.go'` and writes the captured diff to `.agents/output/test-diff.md` with a short header. (Persona: navigator; tools needed: `Bash(git diff:*)`, `Write`.)
- `specs/1582-reject-test-deletions/spec.md` — already created, mirrors issue body.
- `specs/1582-reject-test-deletions/plan.md` — this file.
- `specs/1582-reject-test-deletions/tasks.md` — work breakdown.

### Modified
- `.agents/pipelines/impl-issue.yaml` — insert `judge-test-deletion` step between `implement` and `create-pr`; update `create-pr` `dependencies: [implement]` → `dependencies: [judge-test-deletion]`.
- `internal/defaults/embedfs/pipelines/impl-issue.yaml` — same change for the shipped default.

### Test additions
- `internal/contract/llm_judge_test.go` — extend with a regression case mirroring the issue scenario: a diff that removes a `func Test*` block returns a failing judge result, while a diff that adds tests returns pass. Uses the existing test harness for the validator (no real API calls — see existing `Test_llmJudgeValidator_*` patterns).

### Not changed
- Core contract code (`internal/contract/llm_judge.go`) — no new fields needed.
- `fix-implement` step — already wired as the rework target for `implement`; we set `rework_step: fix-implement` on the new step too so a deletion gate triggers the same surgical rework.

## Architecture Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Where to gate | Separate navigator step after `implement` | Cheap, focused, doesn't bloat craftsman prompt budget |
| Contract type | `llm_judge` (one criterion, threshold 1.0) | Matches issue request; binary gate semantics |
| Diff scope | `*_test.go` only | Issue scope; reduces token use, avoids false positives on unrelated changes |
| Diff base | `{{ base_branch }}...HEAD` (three-dot) | Same range used elsewhere; ignores upstream churn on base |
| Rework target | `fix-implement` | Existing step already handles surgical fixes for craftsman output |
| Threshold | `1.0` | Single criterion; either it passes or it doesn't. No partial credit |
| `on_failure` | `rework` | Per issue spec |
| Model tier | `cheapest` | Single-question yes/no judge — strongest tier wasted here |

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| **False positive: legitimate test rename trips the gate** | Criterion explicitly allows "renamed-and-edited or migrated to a new file" — judge is told to inspect for matching additions in the same diff. If FP rate is high in practice, follow-up issue can swap to a structural source-diff check (companion issue mentioned in #1582). |
| **False negative: judge LLM hallucinates a non-existent replacement** | Threshold 1.0 + one criterion keeps the prompt tight; cheapest-tier model bias is to over-flag, which is the safer side for this gate. |
| **Diff exceeds the 50000-char truncation in `buildUserPrompt`** | Scoped to `*_test.go` keeps payload small in normal cases. If truncation does occur, judge errs on the side of pass — acceptable; companion source-diff contract (siblings of #1582) catches large structural deletions independently. |
| **Empty diff (no `_test.go` files touched)** | Prompt produces a header-only output that the judge classifies as pass; no Test* declarations means no deletions to flag. |
| **`git diff` fails in unusual workspace state** | Navigator prompt writes a clearly-marked "diff capture failed" sentinel. Judge sees no `func Test*` lines → passes. Pipeline does not block on infrastructure errors. |
| **Pipeline duration creeps up** | Adds ~10–30 s for one cheap-model call. Acceptable given the failure cost demonstrated in issue. |

## Testing Strategy

### Unit
- Extend `internal/contract/llm_judge_test.go`: stub the API/CLI call to return a controlled `JudgeResponse` and verify (a) deletion-only diff → contract returns `ValidationError`, (b) addition-only diff → `nil`, (c) replacement (delete + add) → `nil`.

### Pipeline reproducer (per issue acceptance #1)
- Manual: craft a synthetic issue body that asks the implementer to "remove the obsolete test `TestFoo` from `pkg/bar/foo_test.go`" with no replacement. Run `impl-issue` against it locally. Expect `judge-test-deletion` step to fail and rework to fire.
- Captured as a documented manual repro in `specs/1582-reject-test-deletions/tasks.md` Phase 4 — automating this end-to-end requires a live LLM call and is out of CI scope.

### Regression (per issue acceptance #2)
- Run `impl-issue` against an existing green-path issue (pick any recently-merged simple issue) — must complete without rework loop on the new step.

### YAML validation
- `wave doctor` and any pipeline schema validation should still pass.
- `go test ./internal/pipeline/...` covers parser-level regressions.
