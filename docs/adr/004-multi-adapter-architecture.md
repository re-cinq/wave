# ADR-004: Multi-Adapter Architecture

## Status
Accepted

## Date
2026-03-27

## Context

Wave's pipeline executor currently holds a single `AdapterRunner` instance, injected at construction time via `NewDefaultPipelineExecutor(runner)`. All steps in a pipeline share this single runner regardless of their computational complexity or the persona's preferred provider. This creates three problems:

1. **No per-step adapter selection.** A pipeline that mixes simple formatting tasks with complex architectural reasoning must use the same adapter (and model) for both — overpaying for trivial steps or under-powering complex ones.

2. **No provider resilience.** If the single adapter's provider experiences an outage or rate-limits the request, the entire pipeline fails. There is no mechanism to fall back to an alternative provider.

3. **Static resolution at CLI level.** `ResolveAdapter()` in `opencode.go` is a hardcoded switch statement called once during CLI startup (`run.go`, `do.go`, `resume.go`). The resolved adapter is fixed for the pipeline's lifetime.

### Current Architecture

The `AdapterRunner` interface (`Run(ctx, cfg) -> AdapterResult`) has five implementations: `ClaudeAdapter`, `OpenCodeAdapter`, `BrowserAdapter`, `ProcessGroupRunner`, and `MockAdapter`. Model selection already supports a three-tier precedence hierarchy via `resolveModel()` — CLI `--model` flag > `persona.Model` > adapter default — but adapter selection has no equivalent hierarchy.

`ParseProviderModel()` in `environment.go` already handles multi-provider model identifiers (e.g., `gpt-4o` infers OpenAI, `claude-3.5-sonnet` infers Anthropic), providing a foundation for automatic adapter inference from model names.

The persona configuration includes an `adapter` field referencing a manifest adapter definition, but this field is only used to look up the adapter binary name and output format — not to select a different `AdapterRunner` per step.

### Architectural Context

- [ADR-002](002-extract-step-executor.md) proposes extracting a `StepExecutor` from the monolithic `executor.go`. Multi-adapter selection is a natural responsibility of per-step lifecycle management and should align with that decomposition.
- [ADR-003](003-layered-architecture.md) establishes a four-layer model (Presentation / Domain / Infrastructure / Cross-cutting). The adapter package sits in the Infrastructure layer; the executor sits in the Domain layer. Any registry interface must respect this boundary — the domain layer should depend on an abstraction, not concrete adapter implementations.

### Key Constraints

- **Single static binary** — all adapter implementations are compiled in; no dynamic plugin loading.
- **Security isolation** — fresh memory at step boundaries, per-persona permissions, per-adapter sandbox configuration. A registry must preserve these isolation guarantees.
- **Child pipeline propagation** — sub-pipelines create child executors via `NewDefaultPipelineExecutor(e.runner)`. The chosen approach must propagate correctly to child executors.
- **Heterogeneous capabilities** — adapters differ significantly: `ClaudeAdapter` supports agent markdown compilation, NDJSON streaming, and skill provisioning; `OpenCodeAdapter` has a different config format; `BrowserAdapter` is non-LLM. The architecture must handle these differences gracefully.
- **No backward compatibility constraint** during prototype phase — breaking changes are acceptable provided tests pass.

## Decision

Introduce an `AdapterRegistry` interface at the domain/infrastructure boundary. The executor receives an `AdapterRegistry` instead of a single `AdapterRunner`. Per-step adapter resolution happens inside `runStepExecution()` using a hierarchical precedence: **step-level adapter > persona-level adapter > manifest default**. The registry supports optional fallback chains per adapter, triggered only on provider-level failures (rate limiting, timeout, context exhaustion), not on contract or validation failures.

### Registry Interface

```go
// AdapterRegistry resolves adapter runners by name.
// Lives at the domain/infrastructure boundary per ADR-003.
type AdapterRegistry interface {
    // Resolve returns the AdapterRunner for the given adapter name.
    Resolve(name string) (AdapterRunner, error)

    // Available returns the names of all registered adapters.
    Available() []string

    // FallbackChain returns the ordered fallback adapter names
    // for the given primary adapter. Returns nil if no fallbacks configured.
    FallbackChain(primary string) []string
}
```

### Resolution Hierarchy

Adapter selection follows a four-tier hierarchy (extended from the original three-tier design to include the CLI flag):

```
CLI --adapter flag > step.Adapter > persona.Adapter > manifest.Defaults.Adapter
```

If a step specifies `adapter: opencode`, that takes precedence over the persona's default adapter. If neither step nor persona specifies an adapter, the manifest-level default is used. If the resolved adapter name includes a model identifier (e.g., `gpt-4o`), `ParseProviderModel()` infers the correct adapter automatically.

### Fallback Chain Behavior

Fallback chains are configured per adapter in `wave.yaml`:

```yaml
adapters:
  claude:
    binary: claude
    fallback:
      - opencode
    fallback_on:
      - rate_limit
      - timeout
      - context_exhaustion
```

When an adapter returns a provider-level failure matching `fallback_on`, the executor queries `FallbackChain()` and retries with the next adapter. Contract failures, validation errors, and application-level errors do **not** trigger fallback — these indicate problems with the step's logic, not the provider.

## Options Considered

### Option 1: AdapterRegistry with Fallback Chains (Recommended)

Introduce a formal `AdapterRegistry` interface at the domain/infrastructure boundary. The registry maps adapter names to `AdapterRunner` implementations, supports hierarchical resolution, and provides fallback chain configuration. The executor receives an `AdapterRegistry` instead of a single `AdapterRunner`, resolving the correct adapter in `runStepExecution()`. The registry is initialized at CLI startup, replacing the current `ResolveAdapter()` switch.

**Pros:**
- Clean separation of concerns: registry interface at the domain boundary per ADR-003, concrete implementations in infrastructure
- Hierarchical resolution (step > persona > manifest default) mirrors the proven `resolveModel()` pattern
- Fallback chains provide provider resilience against rate limits, timeouts, and outages
- Clear extension point for new adapters — register once, available everywhere — replacing the fragile switch in `opencode.go`
- Enables cost optimization: assign cheap models (Haiku) for formatting steps and expensive models (Opus) for architecture steps
- Forward-compatible with ADR-002: registry becomes a `StepExecutorFactory` dependency when decomposition happens
- `ParseProviderModel()` integrates into resolution for automatic adapter selection from model name
- Child executor propagation is straightforward: pass registry reference
- Testable: `MockRegistry` can be injected without changing executor logic

**Cons:**
- Requires changing the executor constructor signature (`NewDefaultPipelineExecutor` takes registry instead of runner)
- All three CLI entry points (`run.go`, `do.go`, `resume.go`) must be updated
- Fallback chain logic adds complexity to error handling — must distinguish provider failures from contract failures
- Registry lifecycle: adapters may need per-invocation state, so registry must produce fresh runners or handle stateless dispatch
- Manifest schema change required: `wave.yaml` needs fallback chain configuration and optional step-level adapter override fields

### Option 2: Composite Multiplexing AdapterRunner

Create a single `CompositeAdapter` implementing the existing `AdapterRunner` interface that internally dispatches to concrete adapters based on `AdapterRunConfig.Adapter`. The executor's `e.runner` field type does not change. Fallback chains are encapsulated inside the composite.

**Pros:**
- Zero changes to executor interface — `e.runner` stays as `AdapterRunner`
- Zero changes to CLI entry points
- Minimal blast radius: only adapter package changes
- Existing executor tests pass unchanged

**Cons:**
- Violates single responsibility: routing, fallback, lifecycle, and capability management in one type
- Per-step resolution logic in the adapter layer inverts the dependency direction per ADR-003
- The composite must understand the resolution hierarchy, leaking domain knowledge into infrastructure
- Harder to test fallback chains in isolation
- Does not align with ADR-002: when the executor is decomposed, the composite's routing responsibility must move anyway
- Hides complexity: the executor appears to use one adapter but actually uses many, complicating debugging
- Errors in adapter name resolution are caught late (at `Run` time) instead of early (at resolution time)

### Option 3: Resolver Function Injection

Replace the single `AdapterRunner` field with a resolver function: `type AdapterResolverFn func(adapterName string) (AdapterRunner, error)`. The executor calls this function per step. Fallback chains are implemented as wrapper resolvers.

**Pros:**
- Minimal new abstractions: one function type replaces the single runner field
- Idiomatic Go — function injection is a common pattern
- Simple testing: inject a mock function
- Does not preclude evolving to a full registry later

**Cons:**
- No introspection — cannot list available adapters or validate configuration at startup
- Fallback chain configuration has no natural home
- Resolver signature may need to evolve (needing context, manifest, or persona info), causing churn
- No lifecycle management hooks for adapter initialization or cleanup
- Hierarchical resolution must be implemented in the executor rather than the resolver, since the resolver only receives a name
- Less discoverable for new developers

### Option 4: StepExecutor-Bound Adapter (ADR-002 Aligned)

Defer multi-adapter support to the ADR-002 `StepExecutor` extraction. Each `StepExecutor` is constructed with its own `AdapterRunner` already bound. A `StepExecutorFactory` resolves the correct adapter per step.

**Pros:**
- Cleanest architecture: adapter selection is a natural per-step lifecycle responsibility
- Perfect alignment with ADR-002 — both initiatives reinforce each other
- Each `StepExecutor` is fully self-contained with its adapter, sandbox config, and permissions
- Enables parallel step execution with different adapters

**Cons:**
- Blocked on ADR-002 which is still in Proposed status
- Largest implementation effort: requires both executor decomposition and multi-adapter simultaneously
- High coupling risk: if ADR-002 changes direction, this design must be reworked
- Delays multi-adapter support until the full refactoring is complete
- Over-engineers the immediate need

## Consequences

### Positive
- Pipeline authors can assign different adapters and models per step, enabling cost optimization (e.g., Haiku for linting, Opus for architecture)
- Provider outages no longer cause full pipeline failures — fallback chains provide automatic resilience
- New adapters are added by registering with the registry, replacing the brittle `ResolveAdapter()` switch
- The registry interface creates a clean domain/infrastructure boundary per ADR-003
- Design is forward-compatible with ADR-002's `StepExecutor` extraction — the registry migrates cleanly to a `StepExecutorFactory` dependency

### Negative
- Executor constructor changes propagate to all CLI entry points and test setups
- Fallback chain logic adds a new failure-classification responsibility to the execution path
- Manifest schema grows: adapter fallback configuration and step-level adapter overrides add new fields that must be validated
- Heterogeneous adapter capabilities mean the registry cannot guarantee uniform behavior across adapters — callers must handle capability differences

### Neutral
- Existing adapter implementations (`ClaudeAdapter`, `OpenCodeAdapter`, etc.) are unchanged — they are registered with the registry but their internal logic is unaffected
- Contract validation remains independent of adapter selection — contracts validate outputs regardless of which adapter produced them
- The security model (fresh memory, permission enforcement, sandbox configuration) continues to operate per-step; the registry adds adapter-specific sandbox resolution but does not weaken isolation
- Existing `wave.yaml` manifests without adapter overrides continue to work — the manifest-level default adapter serves as the fallback

## Implementation Notes

### Phase 1: Registry Interface and Core Implementation

1. Define the `AdapterRegistry` interface in `internal/adapter/adapter.go` alongside the existing `AdapterRunner` interface
2. Implement `DefaultAdapterRegistry` in a new file `internal/adapter/registry.go` — a map-backed registry with fallback chain support
3. Relocate `ResolveAdapter()` logic from `opencode.go` into registry initialization

**Files changed:** `internal/adapter/adapter.go`, `internal/adapter/registry.go` (new), `internal/adapter/opencode.go`

### Phase 2: Executor Integration

1. Change `DefaultPipelineExecutor` to accept `AdapterRegistry` instead of `AdapterRunner`
2. Add `resolveAdapter()` method to the executor (parallel to existing `resolveModel()`) implementing the step > persona > manifest default hierarchy
3. Update `runStepExecution()` to call `resolveAdapter()` and then `registry.Resolve()` per step
4. Update child executor creation to propagate the registry

**Files changed:** `internal/pipeline/executor.go`

### Phase 3: CLI and Manifest Updates

1. Update `run.go`, `do.go`, `resume.go` to construct a registry and inject it into the executor
2. Extend manifest types to support per-step adapter overrides and fallback chain configuration
3. Update manifest validation to validate adapter references and fallback chains

**Files changed:** `cmd/wave/commands/run.go`, `cmd/wave/commands/do.go`, `cmd/wave/commands/resume.go`, `internal/manifest/types.go`, `internal/manifest/parser.go`

### Phase 4: Fallback Chain Implementation

1. Implement fallback dispatch in the executor: on provider-level failure (matching `fallback_on` error types from `errors.go`), retry with next adapter in the fallback chain
2. Ensure contract failures and validation errors do **not** trigger fallback
3. Add structured event emission for fallback events (adapter switch, fallback exhausted)

**Files changed:** `internal/pipeline/executor.go`, `internal/adapter/errors.go`, `internal/event/`

### Phase 5: Documentation and Testing

1. Update `docs/guides/adapter-development.md` to document registry-based registration
2. Add table-driven tests for registry resolution, hierarchical precedence, and fallback chain behavior
3. Verify all existing tests pass with registry-backed executor

### Migration Path to ADR-002

When `StepExecutor` extraction proceeds, the `AdapterRegistry` migrates from being a direct executor dependency to a `StepExecutorFactory` dependency. The registry interface itself remains stable — only its consumer changes. No throwaway work.

## Implementation Record

This ADR has been implemented. Key details:

- **AdapterRegistry** in `internal/adapter/registry.go` — map-backed registry with fallback chain support, resolving adapter runners by name.
- **Per-step adapter resolution** in `executor.go` `runStepExecution()` — the executor calls `resolveAdapter()` per step using the four-tier hierarchy: CLI `--adapter` flag > `step.adapter` (pipeline YAML) > `persona.adapter` > manifest default.
- **`--adapter` CLI flag** added to `wave run` — allows runtime override of all adapter selection.
- **Step-level `adapter:`** in pipeline YAML — per-step override in the pipeline manifest.
- **Supported adapters**: claude, opencode, gemini, codex.
- **Fallback chains**: infrastructure in place via `FallbackRunner`, triggered on provider-level failures (rate limiting, timeout, context exhaustion) but not on contract or validation failures.
- **Tier Models**: Each adapter can define `tier_models` mapping (`cheapest`, `fastest`, `strongest`) for automatic model selection based on step complexity.
- **Complexity Classification**: Steps are classified into tiers based on persona keywords and step type. `cheapest` personas (navigator, summarizer, auditor, planner) use cost-optimized models. `strongest` personas (craftsman, implementer, debugger, researcher) use capability-optimized models.
