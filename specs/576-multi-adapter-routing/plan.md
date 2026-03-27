# Implementation Plan: Multi-Adapter Model Routing

## Objective

Enable per-step adapter and model selection in Wave pipelines, replacing the current single-adapter-per-execution model with a per-step resolution system that supports multiple LLM CLI backends (Claude Code, Codex, Gemini CLI) and provider fallback chains.

## Approach

The current architecture passes a single `AdapterRunner` into the executor at construction time. All steps use this runner. The change introduces an **adapter registry** that the executor queries per-step, resolving the adapter from a 3-tier precedence chain (step > persona > manifest default). The existing `ResolveAdapter()` factory already maps names to implementations — it becomes the registry's core resolution function.

### Key Insight: Minimal Executor Refactor

The executor's `runStepExecution()` already builds per-step `AdapterRunConfig` with adapter-specific fields (binary, model, output format). The change is:
1. Replace `e.runner` (single `AdapterRunner`) with `e.adapterResolver` (func that returns runner per adapter name)
2. Call resolver inside `runStepExecution()` instead of using the fixed runner
3. Add `Adapter` and `Model` fields to the `Step` struct for step-level overrides

## File Mapping

### New Files
| Path | Purpose |
|------|---------|
| `internal/adapter/codex.go` | OpenAI Codex CLI adapter implementation |
| `internal/adapter/codex_test.go` | Codex adapter unit tests |
| `internal/adapter/gemini.go` | Gemini CLI adapter implementation |
| `internal/adapter/gemini_test.go` | Gemini CLI adapter unit tests |
| `internal/adapter/registry.go` | AdapterRegistry type and resolution logic |
| `internal/adapter/registry_test.go` | Registry unit tests |
| `internal/adapter/fallback.go` | FallbackRunner wrapping adapter with provider fallback chain |
| `internal/adapter/fallback_test.go` | Fallback chain unit tests |

### Modified Files
| Path | Change |
|------|--------|
| `internal/pipeline/types.go` | Add `Adapter` and `Model` fields to `Step` struct |
| `internal/manifest/types.go` | Add `Fallbacks` field to `Runtime` struct |
| `internal/manifest/parser.go` | Validate step-level adapter references and fallback config |
| `internal/pipeline/executor.go` | Replace single `runner` with `AdapterResolver`; update `runStepExecution()` and `resolveModel()` |
| `internal/pipeline/executor_test.go` | Update tests for registry-based adapter resolution |
| `internal/adapter/opencode.go` | Move `ResolveAdapter()` to `registry.go` |
| `internal/adapter/environment.go` | Extend `knownModelPrefixes` if new providers need prefix inference |
| `internal/preflight/checker.go` | Add adapter binary validation per pipeline |
| `internal/preflight/checker_test.go` | Test adapter binary validation |
| `cmd/wave/commands/run.go` | Pass adapter registry instead of single runner |
| `cmd/wave/commands/resume.go` | Same registry change |
| `cmd/wave/commands/do.go` | Same registry change |
| `cmd/wave/commands/compose.go` | Same registry change |
| `cmd/wave/commands/meta.go` | Same registry change |

## Architecture Decisions

### 1. AdapterRegistry vs AdapterResolverFunc

**Decision**: Use a concrete `AdapterRegistry` struct, not a plain function.

**Rationale**: The registry needs to hold fallback config, cache adapter instances, and support mock injection for tests. A function is too opaque.

```go
type AdapterRegistry struct {
    fallbacks map[string][]string // provider → fallback providers
}

func (r *AdapterRegistry) Resolve(adapterName string) AdapterRunner
func (r *AdapterRegistry) ResolveWithFallback(adapterName string, model string) AdapterRunner
```

### 2. Model Resolution Precedence (4-tier)

```
CLI --model > step.Model > persona.Model > adapter default (empty)
```

The `resolveModel()` function gains one tier: step-level model.

### 3. Adapter Resolution Precedence (3-tier)

```
step.Adapter > persona.Adapter > "claude" (hardcoded default)
```

Step-level adapter lookup references the manifest `adapters` map, same as persona-level.

### 4. Fallback Chain Triggering

Fallback triggers on these `FailureReason` values:
- `rate_limit` — provider quota exhausted
- `timeout` — only if classified as provider-side (not step logic)

Does NOT trigger on:
- `context_exhaustion` — model-specific, fallback won't help
- `general_error` — unknown cause, risky to retry

### 5. Codex/Gemini Adapters: Workspace Prep Strategy

Both new adapters follow the Claude adapter pattern:
- Workspace prep (write config files, system prompt)
- NDJSON or JSON stream parsing
- Token extraction from output
- Failure classification

Codex uses `codex` CLI with `--full-auto` mode. Gemini uses `gemini` CLI. Both support `--model` flag.

### 6. Backward Compatibility

The single-runner constructor `NewDefaultPipelineExecutor(runner, opts...)` is replaced with `NewDefaultPipelineExecutor(registry, opts...)`. All CLI commands that call `ResolveAdapter()` now create a registry instead. Existing manifests without step-level `adapter`/`model` fields work unchanged — persona defaults apply.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Codex/Gemini CLI output format changes | Medium | High | Abstract output parsing, version-pin CLI expectations |
| Fallback chain triggers infinite retry loops | Low | High | Max fallback attempts = len(chain), no retrying same provider |
| Step-level adapter field breaks YAML parsing | Low | Medium | Fields are optional with zero-value defaults |
| Mock adapter tests break due to registry change | High | Low | MockAdapterRegistry already exists — wire it through |
| Adapter binary missing at runtime | Medium | Medium | Preflight check validates all referenced adapters have binaries |

## Testing Strategy

### Unit Tests
- `registry_test.go`: Resolution precedence, unknown adapter fallback, registry caching
- `fallback_test.go`: Fallback chain execution, trigger conditions, max attempts
- `codex_test.go`: Workspace prep, output parsing, command building
- `gemini_test.go`: Workspace prep, output parsing, command building
- `executor_test.go`: Per-step adapter resolution, model resolution with step override
- `parser_test.go`: Manifest validation of step-level adapter/model, fallback config

### Integration Tests
- Pipeline execution with mixed adapters (claude + mock)
- Fallback chain triggered by rate limit error
- Preflight validation catching missing adapter binary

### Existing Test Compatibility
- All `executor_test.go` tests must pass — `MockAdapterRegistry` already satisfies the interface pattern
- `go test -race ./...` required before PR
