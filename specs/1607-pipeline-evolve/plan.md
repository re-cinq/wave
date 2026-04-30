# Implementation Plan — pipeline-evolve

## 1. Objective

Ship a meta-pipeline `pipeline-evolve.yaml` that reads `pipeline_eval` history for a target pipeline, asks an LLM to propose an improved v+1 (YAML + persona prompt diffs), and records the result as a `proposed` row in `evolution_proposal`.

## 2. Approach

Four sequential steps in one pipeline file, modeled on `audit-db-trace.yaml` (sqlite3 query) and `ops-issue-quality.yaml` (LLM persona step + structured artifact contract):

1. **gather-eval** (`type: command`) — `sqlite3 .agents/state.db` against `pipeline_eval`, write JSON rollup artifact.
2. **analyze** (`persona: navigator`, `model: cheapest`) — read rollup, classify recurring failures, write structured findings JSON.
3. **propose** (`persona: craftsman`) — read findings + active YAML/persona files, write candidate `<pipeline>.next.yaml` + a unified `prompt.diff`, plus a JSON `proposal_summary` describing what changed and why.
4. **record** (`type: command`) — copy diff/yaml to a stable on-disk diff dir under `.agents/output/evolution/<run_id>/`, then `INSERT INTO evolution_proposal (...) VALUES (..., 'proposed')` via `sqlite3`.

Pipeline input is the **target pipeline name** (a string). Empty input fails fast.

DB access pattern: pipeline reads/writes the SQLite state DB directly via `sqlite3` CLI. No new Go CLI subcommand is required — keeps blast radius minimal and aligns with how `audit-db-trace` reads `pipeline_run`.

## 3. File Mapping

### Created

| Path | Why |
|------|-----|
| `internal/defaults/embedfs/pipelines/pipeline-evolve.yaml` | The pipeline itself (embedded into binary). |
| `.agents/pipelines/pipeline-evolve.yaml` | Project-local copy synced from defaults so the `wave` repo can self-evolve. |
| `.agents/contracts/evolution-findings.schema.json` | JSON schema for `analyze` step output (failure classes, recurring symptoms). |
| `.agents/contracts/evolution-proposal.schema.json` | JSON schema for `propose` step output (paths to yaml/diff, reason, signal_summary). |
| `internal/defaults/embedfs/pipelines/pipeline_evolve_test.go` | Integration test: seed synthetic `pipeline_eval` rows, run pipeline, assert `evolution_proposal` row exists with `status='proposed'`. |

### Modified

| Path | Why |
|------|-----|
| `internal/defaults/embedfs/pipelines/pipeline-evolve.yaml` registry — verify `loader_test.go` picks it up automatically (no list to update). | Validation. |

### Not Modified

- `internal/state/evolution.go` — already exposes everything we need.
- Personas — reuse `navigator` + `craftsman` (default-agnostic).
- `cmd/wave/commands/` — no new CLI subcommand. Pipeline talks to `.agents/state.db` directly via `sqlite3`.

## 4. Architecture Decisions

- **Direct `sqlite3` access over Go CLI shim.** Issue scope is one pipeline; a CLI shim is out of scope for Phase 3.2 and would add API surface that locks us in before the evolution loop is proven. Mirrors the existing `audit-db-trace` precedent.
- **Diff format = unified diff text in a file**, not structured JSON. The diff is opaque to the DB (`diff_path TEXT`) and human-readable; `git apply --check` can verify it. The persona writes both the candidate yaml *and* the diff so reviewers can see the result either way.
- **`signal_summary` = compact JSON**, the same artifact written by the `analyze` step (failure class counts, judge score median, sample size, recurring symptoms). Stored verbatim into the column so the human gate UI has structured context.
- **`version_before` lookup**: query `pipeline_version` for active version; if none exists, default to `0` (the embedded default).
- **`version_after = version_before + 1`** — proposal-only. Activation is a separate step (out of scope, handled by Phase 3.3).
- **Persona choice**: `navigator` for analysis (read-only, low temp) and `craftsman` for proposal generation (creative, writes files). Both are default-agnostic and ship in `internal/defaults/embedfs/personas/`.
- **No `output_artifacts` for the `record` step** — it produces a side-effect (DB row) plus a tiny JSON status. We emit `proposal_id` and `status` so downstream pipelines can chain.
- **Empty-history short-circuit**: if `gather-eval` finds zero `pipeline_eval` rows, write a sentinel artifact and let `analyze` early-exit with `status='insufficient_data'`. The `record` step skips DB insert in that case (`on_failure: warn`).

## 5. Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| `.agents/state.db` not present in mounted workspace | Same fallback as `audit-db-trace` — emit empty artifact and let downstream steps no-op gracefully. |
| LLM proposes a YAML that doesn't parse | Test step uses `wave-cli` validation if available; otherwise schema check via `yq`. Failure → `status='proposed'` with diff still recorded; reviewer rejects. |
| Embedded vs project-local pipeline drift | Add the `.agents/pipelines/` copy to keep both in sync; tests run against embedded. |
| `signal_summary` exceeds reasonable size | `analyze` step caps the summary to top 10 failure classes; raw eval JSON is truncated to last 50 rows. |
| Concurrent runs producing duplicate proposals | Acceptable for Phase 3.2 — `evolution_proposal` allows multiple `proposed` rows per pipeline; human gate dedupes. |
| Default-agnostic violation (e.g. Go-only assumptions) | Step scripts use `sqlite3`/`jq` only. No Go test runner, no language-specific commands. Pipeline does not call `go test` or any project tooling. |

## 6. Testing Strategy

### Unit / integration

- `pipeline_evolve_test.go` (new): builds an in-memory state store via `internal/testutil/statestore.go`, seeds 5 synthetic `PipelineEvalRecord` rows for a fake pipeline name (mix of pass/fail/judge_score), invokes the pipeline via `executor.Run` with a mock adapter that returns canned LLM output (a stub yaml + diff), then asserts:
  1. One row in `evolution_proposal` with `status='proposed'`, matching `pipeline_name`, non-empty `diff_path`, valid `signal_summary` JSON.
  2. `version_before == 0` (no active version), `version_after == 1`.
  3. `gather-eval` artifact contains the seeded rows (sanity).

- Mock adapter (`internal/adapter/mock.go` — already exists) returns the canned proposal artifact paths so `craftsman` step is deterministic.

### Manual smoke

After merge, run on a real pipeline with eval data:

```
wave run pipeline-evolve --input "impl-issue" --adapter claude --model balanced
sqlite3 .agents/state.db "SELECT id, pipeline_name, status FROM evolution_proposal ORDER BY id DESC LIMIT 1;"
```

### CI

- `go test -race ./internal/defaults/embedfs/pipelines/...` covers the new test file.
- `golangci-lint` passes.
- Existing pipeline-loader tests (e.g. `internal/pipeline/loader_test.go`) auto-discover the new YAML via `embed.FS` walk — no list to update.

## 7. Out of Scope (Phase 3.3 territory)

- Activation logic (flipping `pipeline_version.active`)
- Human gate UI for approving/rejecting proposals
- Auto-application of approved diffs
- WebUI surface for the proposal queue
