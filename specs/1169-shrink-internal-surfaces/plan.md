# Implementation Plan — Shrink Internal Package Surfaces

## Objective

Reduce the exported API surface of six `internal/*` packages so only symbols actually consumed by external callers remain capitalized. Delete the entire `classify` lore subsystem (Wave is not a memory/lore platform) and drop unused sandbox backend constants. External behavior must be unchanged.

## Approach

For each package: enumerate exported symbols, identify outside-package callers, rename unused-externally symbols to lower-case (Go un-export), update intra-package + test references, run `go build ./...` and `go test ./...` after each package. Lore removal is a wholesale delete — file plus call sites in `analyzer.go`. Commit per-package so bisect is clean if regressions appear.

## File Mapping

### 1. `internal/classify` (lore removal — delete + edit)

| File | Action | Notes |
|------|--------|-------|
| `internal/classify/lore.go` | DELETE | entire lore subsystem |
| `internal/classify/lore_test.go` | DELETE | tests of deleted code |
| `internal/classify/analyzer.go` | EDIT | remove `applyLoreHints` func + call at L77; drop `domain, complexity = applyLoreHints(...)` line |
| `internal/classify/analyzer_test.go` | EDIT (verify) | check for any LoreHint references |

External callers (`cmd/wave/commands/do.go`) only use `Classify`, `SelectPipeline`, `TaskProfile`, `PipelineConfig` — none are lore. Zero external impact.

### 2. `internal/bench` (un-export internals)

| Symbol | File | Action |
|--------|------|--------|
| `RepoCache` (type) | `repo.go` + refs | rename → `repoCache`; only used internally by `SubprocessRunner` |

External callers (`cmd/wave/commands/bench.go`) use: `LoadDataset`, `RunBenchmark`, `NewSubprocessRunner`, `ModeWave`, `ModeClaude`, `Compare`, `BenchReport`, `RunConfig`. Keep these exported.

Verify: `BenchTask`, `BenchResult`, `CompareReport`, `ReportRef`, `CompareSummary`, `TaskDiff`, `BenchStatus`, `StatusPass/Fail/Error`, `PipelineRunner`, `SubprocessRunner`, `ListDatasets` — confirm any not referenced externally during implementation; un-export accordingly.

### 3. `internal/continuous` (un-export concrete sources)

| Symbol | File | Action |
|--------|------|--------|
| `FileSource` (struct) | `source_file.go` + refs | rename → `fileSource`; expose only via `NewFileSource` factory |
| `GitHubSource` (struct) | `source_github.go` + refs | rename → `githubSource`; expose only via `NewGitHubSource` factory |

`NewFileSource` / `NewGitHubSource` return `WorkItemSource` interface — callers don't need concrete type. External caller (`cmd/wave/commands/run.go`) uses only the interface and factories.

### 4. `internal/cost` (audit helpers)

External caller (`internal/pipeline/executor.go`) uses: `NewLedger`, `CheckIronRule`, `IronRuleWarning`, `IronRuleFail`, `BudgetWarning`, `BudgetExceeded`. Audit other exported symbols (`LookupPricing`, `ComputeCost`, `Record`, `TotalCost`, `Entries`, `Summary`, `EstimateTokens`, `LookupContextWindow`, `DefaultPricing`, `DefaultContextWindow`, `ModelContextWindow`, `BudgetOK`, `BudgetStatus`, `IronRuleStatus`, `IronRuleOK`, `Entry`, `ModelPricing`, `Ledger`) for external reference. Un-export the ones with no outside-package callers.

Likely un-export candidates after audit: `EstimateTokens`, `LookupContextWindow`, `LookupPricing`, `ComputeCost`, `ModelContextWindow`, `DefaultContextWindow` (if only used internally by `Ledger.Record` / `CheckIronRule`).

### 5. `internal/sandbox` (un-export types + drop constants)

| Symbol | File | Action |
|--------|------|--------|
| `DockerSandbox` (type) | `docker.go` + refs | rename → `dockerSandbox` |
| `NoneSandbox` (type) | `none.go` + refs | rename → `noneSandbox` |
| `SandboxBackendBubblewrap` (const) | `types.go`, `factory.go`, tests | DELETE (returns NoneSandbox no-op; bubblewrap handled by Nix flake) |
| `SandboxBackendNone` (const) | keep | used as default empty-string match in factory |

Factory `NewSandbox` returns `Sandbox` interface — concrete types don't need export. `SandboxBackendDocker` stays (referenced in `internal/adapter/adapter.go`).

After dropping `SandboxBackendBubblewrap`, simplify factory switch: collapse the bubblewrap branch into default error or fold into none. Decision: return `unknown sandbox backend` error so any stale config surfaces clearly (no silent no-op).

### 6. `internal/attention` (audit helpers)

External callers (`internal/webui/handlers_attention.go`, `server.go`, `sse.go`) use: `Summary`, `Subscribe`, `Unsubscribe`, `NewBroker`, `UpdateWithName`, `Update`, `Broker`. Audit: `Classify` (free function), `RunAttention`, `State` enum + 4 constants. Un-export anything not externally consumed.

Likely un-export candidates: `Classify` if only `Broker` methods call it; `RunAttention` if only built internally and exposed via `Summary()` return.

## Architecture Decisions

1. **Wholesale lore deletion, not deprecation.** Per memory policy (no legacy/compat support pre-1.0.0) and acceptance criterion explicitly requesting removal.
2. **Factory-only access for sources/sandboxes.** Concrete struct types become package-private; `interface` types remain exported for callers' variable declarations. Keep public constructors.
3. **Drop `SandboxBackendBubblewrap` instead of un-exporting.** It's a no-op alias for `BackendNone` — dead code with confusing semantics. Removing is cleaner than `// Deprecated:`.
4. **Per-package commits.** One `refactor(<pkg>):` commit per package keeps bisect clean and review focused.
5. **No new tests.** This is pure visibility refactor — existing test suite confirms behavior parity. Tests that referenced un-exported symbols (e.g. `lore_test.go`) get deleted with the deleted symbols.

## Risks & Mitigations

| Risk | Mitigation |
|------|-----------|
| Hidden external caller in `cmd/`, `internal/webui/`, `internal/pipeline/` missed by search | After each rename, run `go build ./...` — compile errors surface every reference |
| Test file uses un-exported symbol via different package | `go test ./...` after each package; `_test.go` in the same package can use lower-case fine |
| Lore removal breaks `analyzer.go` logic | The `applyLoreHints` call only mutates domain/complexity from registered provider; default `NoOpLoreProvider` is the only registered impl across the codebase, so removal is observably a no-op |
| Sandbox factory returns error for bubblewrap config in someone's `wave.yaml` | Search for `bubblewrap` in `.wave/`, `wave.yaml`, default configs; if any reference exists, also remove from defaults |
| `SandboxBackendNone` default-empty-string match break | Keep `case "":` → `noneSandbox{}` path; constant either kept or inlined |

## Testing Strategy

1. **Build check after every rename:** `go build ./...` — must stay green.
2. **Full test suite per package:** `go test ./internal/<pkg>/...` after edits to that package.
3. **Race-detector full run before PR:** `go test -race ./...`.
4. **Vet + lint:** `go vet ./...` and project linter (typically `golangci-lint run`).
5. **Behavioral verification:** No new tests added — existing tests guarantee external behavior is identical, since un-exporting is rename-only and lore default was no-op.
6. **Manual smoke (CLI):** Run `wave bench --help`, `wave run --continuous`, `wave do "test task"` to confirm CLI surfaces still resolve.

## Out of Scope

- Behavior changes (factory return values, default policies, output formats) — strictly visibility.
- Documentation rewrites beyond updating doc.go / package comments where lore is mentioned.
- Re-organizing files within packages (split/merge).
- Adding new packages or moving symbols across packages.
