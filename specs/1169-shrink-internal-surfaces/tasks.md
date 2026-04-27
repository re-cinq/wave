# Work Items

## Phase 1: Setup
- [ ] Item 1.1: Confirm baseline `go build ./...` and `go test ./...` are green on current branch
- [ ] Item 1.2: Re-survey each package's exported symbols against current `cmd/` + `internal/` callers (final cross-check before edits)

## Phase 2: Core Implementation (per-package, sequential)
- [ ] Item 2.1: `internal/classify` — delete `lore.go`, `lore_test.go`; remove `applyLoreHints` from `analyzer.go` (drop call at L77 + func at L132+); update `analyzer_test.go` if it references lore symbols
- [ ] Item 2.2: `internal/bench` — un-export `RepoCache` → `repoCache`; audit other symbols (`BenchTask`, `BenchResult`, `CompareReport`, `ReportRef`, `CompareSummary`, `TaskDiff`, `BenchStatus`, `Status*`, `PipelineRunner`, `SubprocessRunner`, `ListDatasets`) and un-export those without external callers [P]
- [ ] Item 2.3: `internal/continuous` — un-export `FileSource` → `fileSource`, `GitHubSource` → `githubSource`; ensure `NewFileSource` / `NewGitHubSource` return `WorkItemSource` interface [P]
- [ ] Item 2.4: `internal/cost` — audit each exported symbol against `internal/pipeline/executor.go` callers; un-export helpers used only internally (likely `EstimateTokens`, `LookupContextWindow`, `LookupPricing`, `ComputeCost`, `ModelContextWindow`, `DefaultContextWindow`) [P]
- [ ] Item 2.5: `internal/sandbox` — un-export `DockerSandbox` → `dockerSandbox`, `NoneSandbox` → `noneSandbox`; delete `SandboxBackendBubblewrap` constant + factory branch (return error for unknown) [P]
- [ ] Item 2.6: `internal/attention` — audit `Classify`, `RunAttention`, `State` against webui callers; un-export those without external callers [P]

## Phase 3: Verification
- [ ] Item 3.1: `go build ./...` after each package edit (must stay green continuously)
- [ ] Item 3.2: `go test ./internal/<pkg>/...` for each modified package
- [ ] Item 3.3: Full `go test -race ./...`
- [ ] Item 3.4: `go vet ./...` + project linter
- [ ] Item 3.5: CLI smoke: `./wave bench --help`, `./wave run --help`, `./wave do --help`

## Phase 4: Polish
- [ ] Item 4.1: Update package doc comments where lore is mentioned (e.g. `internal/classify/doc.go` if present)
- [ ] Item 4.2: Search defaults / `.wave/` / `wave.yaml` for `bubblewrap` references; remove if any
- [ ] Item 4.3: Per-package commits with `refactor(<pkg>):` prefix; final commit message references issue #1169
- [ ] Item 4.4: Open PR with summary listing un-exported / deleted symbols per package
