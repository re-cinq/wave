# ADR-002: Adopt Interface-Driven Dependency Injection for Pipeline Executor

## Status
Proposed

## Date
2026-03-12

## Context

Wave's pipeline executor (`internal/pipeline/executor.go`) is the central orchestration hub for all pipeline execution. It directly imports and creates instances from 10+ packages: adapter, contract, workspace, state, relay, security, skill, event, and deliverable. This tight coupling means that changes to any component's internals can require corresponding changes in the executor, and testing the executor requires real instances of all dependencies — including SQLite databases, filesystem workspaces, and subprocess-spawning adapters.

The current architecture is functional and correct for Wave's scale (2-8 step pipelines, single-machine execution, 30+ personas, 42+ pipelines). The subprocess-based adapter model provides natural memory isolation, the synchronous DAG loop is predictable and debuggable, and SQLite with WAL mode gives ACID guarantees with zero operational overhead. These fundamentals are sound and should be preserved.

However, several pain points are emerging:

- **Testing friction**: Executor tests must either spawn real subprocesses or use integration-level setups. Unit testing individual orchestration logic (retry, deadlock detection, artifact injection) in isolation is difficult.
- **Coupling creep**: The `ClaudeAdapter` reads `base-protocol.md` by hardcoded relative path. `PipelineContext` calls git directly. These concrete dependencies make it harder to test, substitute, or reconfigure components.
- **Future extensibility blocked**: Any future evolution — whether adding new adapter backends, alternative state stores, plugin systems, or event-driven execution — requires decoupled interfaces as a prerequisite. Without interface boundaries, each of these directions demands a large, risky refactoring as a first step.

The system is in prototype phase with no backward compatibility constraints, making this the ideal time for structural improvements that establish clean boundaries before the interfaces ossify.

## Decision

Refactor the pipeline executor to use **explicit dependency injection via constructor parameters** and **narrow, executor-defined interfaces** (Interface Segregation Principle). Introduce a **composition root** in `cmd/wave/` that wires all concrete implementations together.

This preserves the existing execution model (single binary, subprocess adapters, synchronous DAG, SQLite state, ephemeral workspaces) while decoupling the executor from concrete implementations. The refactoring is purely structural — no behavioral changes.

## Options Considered

### Option 1: Monolithic Subprocess Orchestrator (Current Architecture)

Maintain the current architecture unchanged. The single static binary with embedded defaults, subprocess-based adapters, tight-loop DAG execution, and direct package imports in the executor.

**Pros:**
- Zero effort — no changes required
- Zero risk of introducing regressions
- Architecture is proven correct at current scale
- Simple mental model: one binary, one process, direct function calls

**Cons:**
- Testing friction persists and worsens as the codebase grows
- Coupling between executor and 10+ packages makes isolated changes increasingly difficult
- Every future architectural evolution requires decoupling as a prerequisite anyway
- Hardcoded paths and direct git calls in components reduce portability and testability

### Option 2: API-First Adapter Architecture

Replace subprocess-based adapter execution with direct HTTP API calls to LLM providers (Anthropic Messages API, OpenAI API). Wave would construct API requests, manage tool execution, and handle streaming responses directly.

**Pros:**
- Eliminates subprocess overhead (1-3 seconds per step for process spawning and workspace preparation)
- Full control over token management and context window optimization
- Removes dependency on CLI binary availability
- Enables fine-grained tool execution auditing at the API level

**Cons:**
- Massive implementation effort: must reimplement tool execution (Read, Write, Edit, Bash, Glob, Grep) — thousands of lines of carefully tested code currently handled by Claude Code CLI
- Loses Claude Code's built-in permission system, sandbox enforcement, hook execution, and session management
- API key management becomes Wave's responsibility
- Security model regresses from three layers (Nix sandbox + CLI sandbox + CLAUDE.md restrictions) to one
- Breaks the AdapterRunner interface which assumes subprocess semantics (ExitCode, Stdout)
- Vendor lock-in: each LLM provider has different API semantics for tool use and streaming

### Option 3: Plugin-Based Extensibility Architecture

Introduce a plugin system using hashicorp/go-plugin (gRPC over stdin/stdout) for adapters, contract validators, and workspace strategies. Core engine remains a single binary but discovers and loads plugins at runtime.

**Pros:**
- Runtime extensibility without forking Wave or waiting for releases
- Process-level plugin boundaries enforce interface contracts
- hashicorp/go-plugin is production-proven (Terraform, Vault, Nomad)
- Language-agnostic plugin authoring via gRPC

**Cons:**
- Violates the single-binary constraint: plugins are external files requiring discovery and management
- gRPC serialization overhead on every adapter call and contract validation
- Go's native plugin package is Linux/macOS only with no unloading support
- Plugin distribution, version compatibility, and failure handling add significant complexity
- Current interface surface is large (AdapterRunConfig has 20+ fields) — gRPC schema would be complex

### Option 4: Event-Driven Reactive Pipeline Engine

Replace the synchronous DAG execution loop with an event-driven architecture using channels and a central event bus. Steps emit completion events that trigger dependency resolution asynchronously.

**Pros:**
- Natural fit for DAG execution where steps become ready upon dependency completion
- Enables dynamic pipeline modification at runtime
- Decouples execution from observation (TUI/WebUI become event subscribers)
- Supports external event sources (CI/CD webhooks, manual approvals)

**Cons:**
- Event-driven systems are harder to debug — race conditions and ordering bugs are subtle
- Overkill for current scale (2-8 step pipelines with simple linear dependencies)
- Existing test suite assumes synchronous execution — substantial test infrastructure rewrite needed
- Loss of execution predictability: synchronous loop is deterministic, event-driven introduces non-determinism
- Channel lifecycle management (leaked goroutines, deadlocks, backpressure) adds complexity

### Option 5: Hybrid Interface-Driven Architecture with Dependency Injection (Recommended)

Refactor the executor to receive all dependencies via constructor injection. Define minimal interfaces in the executor package for each dependency. Wire concrete implementations in a composition root in `cmd/wave/`.

**Pros:**
- Minimal disruption: preserves all existing behavior, subprocess model, and execution semantics
- Dramatically improves testability: executor tests use mock implementations without subprocesses or SQLite
- Reduces coupling: executor defines the interfaces it needs rather than importing concrete packages
- Prepares for future extensibility: swapping implementations becomes a single-line change at the composition root
- Eliminates hardcoded filesystem paths: base-protocol.md path and persona prompts become injected configuration
- Composition root makes the dependency graph explicit and auditable
- Compatible with existing test infrastructure: table-driven tests work with mock injection
- Can be executed incrementally — one interface at a time without disrupting feature work

**Cons:**
- Does not solve horizontal scaling, dynamic extensibility, or runtime plugin loading
- Interface proliferation risk if too many narrow interfaces are defined
- Constructor parameter lists may grow large without parameter objects or functional options
- Requires touching test files to switch from concrete types to interfaces — large but mechanical change
- No immediate user-visible benefit: internal quality improvement that enables future work

## Consequences

### Positive
- Executor unit tests can validate orchestration logic (retry, deadlock detection, artifact injection, dependency resolution) without spawning subprocesses, touching SQLite, or creating filesystem workspaces
- Changes to adapter internals, contract validation logic, or state store implementation no longer require executor modifications — only the interface contract must be preserved
- The composition root in `cmd/wave/` provides a single location where the full dependency graph is visible and auditable
- Future architectural evolution (new adapters, alternative state stores, plugin systems, event-driven execution) can be introduced by implementing existing interfaces rather than requiring a decoupling refactoring first
- Hardcoded filesystem paths in `ClaudeAdapter` become injected configuration, improving portability and test isolation

### Negative
- Large mechanical PR touching executor, adapter, contract, workspace, state, and their test files — review burden is high even though changes are structural rather than behavioral
- Interface proliferation risk: defining too many narrow interfaces can fragment the codebase and make navigation harder — discipline is needed to keep interfaces minimal
- No immediate user-visible improvement — this is an investment in internal quality that pays off over time
- Constructor parameter lists for the executor may become unwieldy without Go's named parameter support — may need functional options or a config struct

### Neutral
- The fundamental execution model is unchanged: single binary, subprocess adapters, synchronous DAG, SQLite state, ephemeral workspaces, fresh memory at step boundaries
- All existing integration tests continue to work unchanged — they exercise the full stack through the composition root
- The single-binary constraint is preserved: no runtime plugin loading, no external dependencies added
- This is a prerequisite for Options 2, 3, and 4 — if any of those directions are chosen later, the interface boundaries established here will be the foundation

## Implementation Notes

1. **Define executor-local interfaces**: Create minimal interfaces in `internal/pipeline/` for each dependency the executor uses. Start with the highest-value targets:
   - `StepRunner` (wraps `AdapterRunner` — `Run(ctx, cfg) (Result, error)`)
   - `ContractValidator` (wraps contract validation — `Validate(ctx, step, output) (ValidationResult, error)`)
   - `WorkspaceManager` (wraps workspace creation/teardown — `Create(ctx, cfg) (Workspace, error)`, `Cleanup(ctx, ws) error`)
   - `StateStore` (wraps SQLite persistence — step status updates, artifact recording, performance tracking)
   - `EventEmitter` (wraps progress event emission)

2. **Refactor executor constructor**: Change `NewExecutor()` to accept interfaces via constructor parameters or a config struct. Remove internal construction of concrete dependencies.

3. **Create composition root**: In `cmd/wave/`, add a `wire.go` or similar file that constructs all concrete implementations and passes them to `NewExecutor()`. This becomes the single place where the dependency graph is assembled.

4. **Migrate incrementally**: Introduce one interface at a time in separate PRs to keep reviews manageable. Suggested order: `StepRunner` → `ContractValidator` → `WorkspaceManager` → `StateStore` → `EventEmitter`. Each PR includes the interface definition, executor refactoring, mock implementation, and updated tests.

5. **Extract hardcoded paths**: Move `base-protocol.md` path resolution and persona prompt file lookup from `ClaudeAdapter` into injected configuration. This eliminates the adapter's coupling to the embedded filesystem layout.

6. **Key files requiring changes**:
   - `internal/pipeline/executor.go` — primary refactoring target
   - `internal/pipeline/types.go` — new interface definitions
   - `internal/adapter/adapter.go` — ensure `AdapterRunner` satisfies the new `StepRunner` interface
   - `internal/contract/contract.go` — ensure validators satisfy the new `ContractValidator` interface
   - `internal/workspace/workspace.go` — ensure manager satisfies `WorkspaceManager` interface
   - `internal/state/store.go` — ensure store satisfies `StateStore` interface
   - `cmd/wave/` — new composition root wiring

7. **Validation**: Run `go test ./...` and `go test -race ./...` after each incremental PR to ensure no behavioral regressions. The refactoring is structural — all existing tests must continue to pass.
