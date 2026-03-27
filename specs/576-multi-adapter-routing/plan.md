# Implementation Plan: Multi-Adapter Model Routing

## 1. Objective

Enable per-step adapter and model selection in Wave pipelines, add Codex and Gemini CLI adapters, and implement fallback chains for provider resilience. The core change is moving from a single shared `AdapterRunner` to per-step adapter resolution.

## 2. Approach

### Core Architecture Change

Currently, `cmd/wave/commands/run.go` picks the **first** adapter from the manifest and creates a **single** `AdapterRunner` that the executor uses for all steps. The executor stores this as `e.runner`.

The change introduces an `AdapterResolver` -- a function or interface that the executor calls per-step to get the correct `AdapterRunner`. The resolution chain is:

```
step.Adapter > persona.Adapter > first manifest adapter
```

Similarly, model resolution becomes:

```
CLI --model > step.Model > persona.Model > empty
```

### New Adapter Implementations

Add `codex.go` and `gemini.go` following the same pattern as `claude.go` and `opencode.go`:
- Implement `AdapterRunner` interface (single `Run()` method)
- Handle workspace preparation (write adapter-specific config files)
- Parse NDJSON or structured output for stream events
- Support timeout/graceful termination

### Fallback Chains

Add `runtime.fallbacks` mapping adapter names to ordered fallback lists. When an adapter returns a transient/quota error (rate limit, 503, etc.), the executor retries with the next adapter in the fallback chain. Permanent errors (bad model name, auth failure) fail immediately.

## 3. File Mapping

### Create

| File | Purpose |
|------|---------|
| `internal/adapter/registry.go` | `AdapterRegistry` type with `Resolve(name)` method; move `ResolveAdapter()` here |
| `internal/adapter/codex.go` | Codex CLI adapter implementing `AdapterRunner` |
| `internal/adapter/gemini.go` | Gemini CLI adapter implementing `AdapterRunner` |
| `internal/adapter/codex_test.go` | Unit tests for Codex adapter |
| `internal/adapter/gemini_test.go` | Unit tests for Gemini adapter |
| `internal/adapter/registry_test.go` | Unit tests for adapter registry |
| `internal/pipeline/executor_routing_test.go` | Tests for per-step adapter routing |

### Modify

| File | Changes |
|------|---------|
| `internal/pipeline/types.go` | Add `Adapter` and `Model` fields to `Step` struct |
| `internal/manifest/types.go` | Add `Fallbacks` to `Runtime` struct |
| `internal/manifest/parser.go` | Add validation for `fallbacks` config and step adapter references |
| `internal/adapter/opencode.go` | Move `ResolveAdapter()` to `registry.go`, keep adapter implementation |
| `internal/pipeline/executor.go` | Change `runner` field to `AdapterResolver` or resolver func; update `runStepExecution()` to resolve per-step; update `resolveModel()` for step-level model; implement fallback retry logic |
| `cmd/wave/commands/run.go` | Create `AdapterRegistry` instead of single runner; pass to executor |
| `internal/preflight/preflight.go` | Add validation for adapter binary availability per step |
| `internal/adapter/mock.go` | Add `MockAdapterRegistry` support for multi-adapter testing |

### Delete

None.

## 4. Architecture Decisions

### AD-1: Resolver function vs. registry object

**Decision**: Use a concrete `AdapterRegistry` struct with a `Resolve(name string) AdapterRunner` method. This is simpler than an interface and the registry can cache adapter instances.

**Rationale**: Adapters are stateless (they create subprocess per call), so caching by name is safe. A registry object also makes testing straightforward -- inject a mock registry.

### AD-2: Step.Adapter field type

**Decision**: `string` field matching adapter names in `manifest.Adapters` map. Empty string means "use persona default".

**Rationale**: Consistent with how `Step.Persona` references `manifest.Personas`. No new types needed.

### AD-3: Fallback scope

**Decision**: Fallback chains map adapter names (not provider names). When adapter `claude` fails with a transient error, try `openai`, then `gemini` per the configured chain.

**Rationale**: Adapter names are the resolution unit in Wave. Mapping by adapter name avoids introducing a new "provider" concept.

### AD-4: API adapter out of scope

**Decision**: The `api` adapter (direct API calls where Wave manages the agent loop) is deferred. This issue focuses on CLI-based adapters only.

**Rationale**: The issue explicitly marks it as "future". CLI adapters follow the established subprocess pattern.

### AD-5: Model validation

**Decision**: Pass-through model names to adapters without validation against known lists. Each adapter is responsible for model name interpretation.

**Rationale**: Model names change frequently. Validating against a hardcoded list creates maintenance burden. The adapter binary will report invalid model errors.

### AD-6: Executor backward compatibility

**Decision**: Keep `NewDefaultPipelineExecutor(runner, opts...)` signature working for tests that pass a single runner. Add `WithAdapterRegistry(reg)` option. If registry is set, it takes precedence over the single runner.

**Rationale**: Avoids breaking 50+ test files that construct executors with mock adapters.

## 5. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Codex/Gemini CLIs have different output formats | Medium | Medium | Implement minimal adapters with NDJSON parsing; degrade gracefully to raw stdout |
| Fallback chains create confusing debugging experience | Low | Medium | Log which adapter is being tried; include adapter name in progress events |
| Per-step adapter resolution breaks existing tests | Medium | High | Keep single-runner path working via AD-6; registry is additive |
| Binary availability varies across environments | High | Low | Preflight catches missing binaries; steps using unavailable adapters fail clearly |

## 6. Testing Strategy

### Unit Tests
- `registry_test.go`: Registry resolves known adapters, returns ProcessGroupRunner for unknown
- `codex_test.go`: Workspace preparation, argument building, output parsing
- `gemini_test.go`: Workspace preparation, argument building, output parsing
- `executor_routing_test.go`: Per-step adapter resolution with step > persona > manifest fallback
- `executor_routing_test.go`: Fallback chain triggered on transient errors, skipped on permanent errors
- `executor_routing_test.go`: Model resolution with step > persona priority

### Integration Tests
- Existing `go test ./...` must pass (no regressions)
- Pipeline execution with mixed adapters (mock registry)

### Validation
- Preflight rejects pipelines referencing undefined adapters
- Manifest parser rejects invalid fallback config (self-referencing, unknown adapter names)
