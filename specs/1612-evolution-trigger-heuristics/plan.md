# Implementation Plan — #1612 Evolution Trigger Heuristics

## 1. Objective

Add `internal/evolution.ShouldEvolve(pipelineName)` that scores accumulated `pipeline_eval` rows against three heuristics (every-N judge drop, contract-pass drift, retry-rate spike) and emits an advisory `evolution_proposed` event from the post-run hook when any fires.

## 2. Approach

- New `internal/evolution` package owns the trigger logic and config.
- Config struct lives in `internal/evolution` with sensible defaults; loaded from a new `evolution:` block on `manifest.Manifest` (top-level, parallel to `runtime:`/`hooks:`).
- Trigger reads recent eval rows via existing `state.EvolutionStore.GetEvalsForPipeline`. To find "since last evolution" it needs a new accessor that returns the latest proposal timestamp for a pipeline → add `LastProposalAt(pipelineName) (time.Time, bool, error)` to `EvolutionStore`.
- Wire into `recordPipelineEval` in `internal/pipeline/executor_eval.go` after the row is persisted: if trigger fires, emit `event.Event{State: "evolution_proposed", ...}`. Best-effort, never blocks finalize.
- Tests use the existing in-process sqlite test store (see `internal/state` test helpers) to seed synthetic rows.

## 3. File Mapping

### Create

- `internal/evolution/doc.go` — package doc explaining the trigger surface and Phase 3.3 scope.
- `internal/evolution/config.go` — `Config` struct + `DefaultConfig()` + `Merge(yamlOverrides)` helpers.
- `internal/evolution/triggers.go` — `Service` with `ShouldEvolve(pipelineName) (bool, reason string, err error)`; private heuristics `everyNJudgeDrop`, `contractPassDrift`, `retryRateSpike`; helpers `medianJudgeScore`, `passRate`, `avgRetry`.
- `internal/evolution/triggers_test.go` — table-driven tests per heuristic + composition cases + insufficient-data cases + config override case.

### Modify

- `internal/manifest/types.go` — add `Evolution *EvolutionYAML \`yaml:"evolution,omitempty"\`` field on `Manifest`; mirror struct fields from `evolution.Config`.
- `internal/state/evolution.go` — add `LastProposalAt(pipelineName) (time.Time, bool, error)` to `EvolutionStore` interface and `stateStore` impl.
- `internal/pipeline/executor.go` — add `evolutionService *evolution.Service` field + `WithEvolutionService` option; thread through `New*` constructors.
- `internal/pipeline/executor_eval.go` — after `RecordEval` succeeds, call `evolutionService.ShouldEvolve(pipelineName)`; on `(true, reason)` emit `event.Event{State: "evolution_proposed", PipelineID: runID, Message: fmt.Sprintf("%s: %s", pipelineName, reason)}`. Skip silently when service is nil.
- `internal/pipeline/executor_eval_test.go` — add coverage for the emit path with a stub service that returns true/false.
- `cmd/wave/...` (constructor wiring) — instantiate `evolution.NewService(store, cfg)` from loaded manifest and pass via `WithEvolutionService`.
- `wave.yaml` — add commented `evolution:` block documenting defaults.

### Delete

None.

## 4. Architecture Decisions

1. **Package boundary**: `internal/evolution` depends on `internal/state` (EvolutionStore) and on its own `Config`. It does **not** import `internal/manifest` to avoid an import cycle with the executor wiring; instead `manifest.EvolutionYAML` is converted to `evolution.Config` at the call site.
2. **Trigger interface**: `Service` is a small struct so it can be stubbed in executor tests via a narrow interface (`type Trigger interface { ShouldEvolve(string) (bool, string, error) }`). Define interface in `internal/pipeline` (consumer-side) per Go convention.
3. **"Since last evolution" semantics**: defined as "evals with `recorded_at >` last `evolution_proposal.proposed_at` for this pipeline". When no prior proposal exists, the threshold falls back to time zero so all evals count.
4. **Median judge_score window comparison**: split the most recent `2 * every_n_window` rows that have non-nil `JudgeScore` into two halves; compare medians. Insufficient data (< `2 * every_n_window` scored rows) → heuristic does not fire.
5. **Drift / retry windows** are independent of "since last evolution" — they always look at the most recent N rows regardless of last proposal. This matches the issue language ("over last 20 evals", "over last 10 evals").
6. **Best-effort emission**: trigger failure (`err != nil`) is logged via `e.emit` warning and never blocks finalize, mirroring the existing `RecordEval` failure handling.
7. **No dedupe in this PR**: emitting `evolution_proposed` repeatedly across runs is acceptable for Phase 3.3. Phase 3.4 (UI / proposal queue) will own dedupe via the proposal lifecycle.

## 5. Risks

| Risk | Mitigation |
|------|------------|
| Median calc on unsorted data → wrong split | Use stable sort by `recorded_at DESC` (already returned by `GetEvalsForPipeline`); slice halves cleanly. |
| Nil `JudgeScore` rows skew average | Filter nil before median / average; track scored-row count separately. |
| Overfetch on hot pipelines | Cap query at `max(every_n_window*2, drift_window, retry_window)` rows. |
| Config-cycle import (`manifest` ↔ `evolution`) | Conversion happens at executor wiring layer; neither package imports the other. |
| Event spam after threshold met | Acceptable in Phase 3.3 (advisory only); dedupe in Phase 3.4. Document in commit + README. |
| Tests flaky on time-based comparisons | Inject a `Clock` interface (or pass `time.Time` explicitly) into trigger so tests use fixed timestamps. |

## 6. Testing Strategy

### Unit (`internal/evolution/triggers_test.go`)

- Table-driven cases: each of the three heuristics fires independently with synthetic rows.
- Composition: two heuristics fire → reason concatenated; precedence is every-N > drift > retry for first-match reason text.
- Insufficient data per heuristic returns `(false, "")`.
- Config override path: custom thresholds applied → fires/doesn't fire as expected.
- `enabled: false` short-circuits to `(false, "")`.

### Unit (`internal/state/evolution_test.go`)

- New `LastProposalAt`: empty store → `(zero, false, nil)`; one proposal → returns its `proposed_at`; multiple → returns most recent.

### Integration (`internal/pipeline/executor_eval_test.go`)

- Stub `Trigger` returns `(true, "drift")` → `recordPipelineEval` emits an `evolution_proposed` event with the reason in `Message`.
- Stub returns `(false, "")` → no advisory event emitted.
- Stub returns error → warning event emitted, no `evolution_proposed`.
- Nil service → no panic, no emission.

### Race

- `go test -race ./internal/evolution/... ./internal/pipeline/... ./internal/state/...`

## 7. Wave.yaml schema

```yaml
evolution:
  enabled: true            # master switch (default true)
  every_n_window: 10       # rows per half-window for judge-score median compare
  every_n_judge_drop: 0.1  # min median drop to fire every-N
  drift_window: 20         # rows for contract_pass drift
  drift_pass_drop: 0.15    # min absolute pass-rate drop to fire drift
  retry_window: 10         # rows for retry-rate
  retry_avg_threshold: 2.0 # avg retry_count over window to fire retry-rate
```
