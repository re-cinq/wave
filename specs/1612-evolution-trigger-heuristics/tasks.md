# Work Items — #1612

## Phase 1: Setup
- [X] 1.1 Create `internal/evolution/` directory with `doc.go` package comment
- [X] 1.2 Add `EvolutionYAML` struct on `manifest.Manifest` (`yaml:"evolution,omitempty"`)
- [X] 1.3 Extend `state.EvolutionStore` interface with `LastProposalAt(name) (time.Time, bool, error)` and implement on `stateStore`

## Phase 2: Core Implementation
- [X] 2.1 `internal/evolution/config.go`: `Config` struct, `DefaultConfig()`, `YAMLOverrides.Apply()` converter
- [X] 2.2 `internal/evolution/triggers.go`: `Service` struct + `NewService(store, cfg)` + `ShouldEvolve(name)` orchestrator
- [X] 2.3 `internal/evolution/triggers.go`: heuristic helpers `everyNJudgeDrop`, `contractPassDrift`, `retryRateSpike`
- [X] 2.4 `internal/evolution/triggers.go`: stat helpers `medianFloats`, `passRate`, `avgRetry`
- [X] 2.5 Define consumer-side `EvolutionTrigger` interface in `internal/pipeline/executor_eval.go`
- [X] 2.6 Add `evolutionTrigger EvolutionTrigger` field + `WithEvolutionTrigger` option on `DefaultPipelineExecutor`
- [X] 2.7 Wire trigger call into `recordPipelineEval` after successful `RecordEval`; emit `evolution_proposed` event
- [X] 2.8 Construct `evolution.Service` from manifest in `cmd/wave` startup and pass via option

## Phase 3: Testing
- [X] 3.1 `internal/evolution/triggers_test.go`: every-N fires + below threshold no-fire
- [X] 3.2 `internal/evolution/triggers_test.go`: drift fires + below threshold no-fire
- [X] 3.3 `internal/evolution/triggers_test.go`: retry-rate fires + below threshold no-fire
- [X] 3.4 `internal/evolution/triggers_test.go`: composition (multi-fire), insufficient-data, disabled, custom-config cases
- [X] 3.5 `internal/state/evolution_test.go`: `LastProposalAt` empty / single / multi cases
- [X] 3.6 `internal/pipeline/executor_eval_test.go`: stub trigger emit / no-emit / error / nil-service paths
- [X] 3.7 `go test -race ./internal/evolution/... ./internal/pipeline/... ./internal/state/...`

## Phase 4: Polish
- [X] 4.1 Add commented `evolution:` block to `wave.yaml` (no `internal/defaults/embedfs/wave.yaml` present)
- [X] 4.2 Lint / vet clean
- [ ] 4.3 (deferred) Update `docs/scope/onboarding-as-session-plan.md` Phase 3 row 3.3 — out of scope for this PR
- [ ] 4.4 (deferred) Final manual run with synthetic eval rows — covered by integration tests
