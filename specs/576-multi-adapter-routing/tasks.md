# Tasks

## Phase 1: Schema & Registry Foundation

- [ ] Task 1.1: Add `Adapter` and `Model` fields to `Step` struct in `internal/pipeline/types.go`
- [ ] Task 1.2: Add `Fallbacks` field (`map[string][]string`) to `Runtime` struct in `internal/manifest/types.go`
- [ ] Task 1.3: Create `internal/adapter/registry.go` with `AdapterRegistry` struct and `Resolve(name string) AdapterRunner` method; move `ResolveAdapter()` from `opencode.go`
- [ ] Task 1.4: Remove `ResolveAdapter()` from `internal/adapter/opencode.go` and update all callers to use registry
- [ ] Task 1.5: Write `internal/adapter/registry_test.go` -- registry resolves known adapters, caching, unknown fallback

## Phase 2: New Adapter Implementations

- [ ] Task 2.1: Implement Codex CLI adapter in `internal/adapter/codex.go` [P]
- [ ] Task 2.2: Implement Gemini CLI adapter in `internal/adapter/gemini.go` [P]
- [ ] Task 2.3: Write `internal/adapter/codex_test.go` -- workspace prep, arg building, output parsing [P]
- [ ] Task 2.4: Write `internal/adapter/gemini_test.go` -- workspace prep, arg building, output parsing [P]

## Phase 3: Executor Per-Step Routing

- [ ] Task 3.1: Add `WithAdapterRegistry(reg *adapter.AdapterRegistry)` executor option in `internal/pipeline/executor.go`
- [ ] Task 3.2: Update `runStepExecution()` to resolve adapter per-step: step.Adapter > persona.Adapter > first manifest adapter; fall back to `e.runner` if no registry
- [ ] Task 3.3: Update `resolveModel()` to include step-level model: CLI --model > step.Model > persona.Model > empty
- [ ] Task 3.4: Update `cmd/wave/commands/run.go` to create `AdapterRegistry` and pass via `WithAdapterRegistry()`

## Phase 4: Fallback Chains

- [ ] Task 4.1: Add fallback config validation to `internal/manifest/parser.go` (no self-references, all adapter names must exist in manifest)
- [ ] Task 4.2: Implement fallback retry logic in executor -- on transient/quota `FailureReason`, try next adapter in chain
- [ ] Task 4.3: Add adapter name to progress events and audit logs for observability

## Phase 5: Preflight & Validation

- [ ] Task 5.1: Add step-level adapter reference validation in `internal/manifest/parser.go` (step.Adapter must reference existing adapter in manifest)
- [ ] Task 5.2: Extend preflight to check binary availability for all adapters referenced by pipeline steps (not just first adapter)

## Phase 6: Testing & Validation

- [ ] Task 6.1: Write `internal/pipeline/executor_routing_test.go` -- per-step adapter resolution, model resolution, fallback chains
- [ ] Task 6.2: Update `internal/adapter/mock.go` `MockAdapterRegistry` to support multi-adapter test scenarios
- [ ] Task 6.3: Run `go test ./...` and `go test -race ./...` -- fix any regressions
- [ ] Task 6.4: Run `golangci-lint run ./...` -- fix lint issues
