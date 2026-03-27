# Tasks

## Phase 1: Adapter Registry Foundation

- [X] Task 1.1: Create `AdapterRegistry` type in `internal/adapter/registry.go` with `Resolve(name string) AdapterRunner` method. Move `ResolveAdapter()` from `opencode.go` into the registry as its core resolution logic. Add `NewAdapterRegistry(fallbacks map[string][]string)` constructor.
- [X] Task 1.2: Add `Adapter` (string) and `Model` (string) optional fields to the `Step` struct in `internal/pipeline/types.go`.
- [X] Task 1.3: Add `Fallbacks` field (`map[string][]string`) to the `Runtime` struct in `internal/manifest/types.go`.
- [X] Task 1.4: Update manifest validation in `internal/manifest/parser.go` to validate step-level `adapter` references exist in the manifest's `adapters` map, and validate fallback provider names.

## Phase 2: Executor Per-Step Resolution

- [X] Task 2.1: Replace `runner adapter.AdapterRunner` field in `DefaultPipelineExecutor` with `registry *adapter.AdapterRegistry`. Update `NewDefaultPipelineExecutor` signature. Add `WithRegistry` executor option.
- [X] Task 2.2: Update `runStepExecution()` in `executor.go` to resolve adapter per-step: step.Adapter > persona.Adapter, then call `e.registry.Resolve(resolvedAdapterName)` instead of `e.runner`.
- [X] Task 2.3: Extend `resolveModel()` to 4-tier precedence: CLI --model > step.Model > persona.Model > empty. Update `AdapterRunConfig` construction to use step-level model.
- [X] Task 2.4: Update all CLI commands (`run.go`, `resume.go`, `do.go`, `compose.go`, `meta.go`) to create `AdapterRegistry` instead of calling `ResolveAdapter()` directly. Pass registry to executor.

## Phase 3: New Adapter Implementations

- [X] Task 3.1: Implement Codex CLI adapter in `internal/adapter/codex.go` — workspace prep, command building with `--full-auto --model <model>`, output parsing, token extraction, failure classification. [P]
- [X] Task 3.2: Implement Gemini CLI adapter in `internal/adapter/gemini.go` — workspace prep, command building with `--model <model>`, output parsing, token extraction, failure classification. [P]
- [X] Task 3.3: Register `codex` and `gemini` adapter names in the registry's `Resolve()` switch. Update `knownModelPrefixes` in `environment.go` if needed. [P]

## Phase 4: Fallback Chain Logic

- [X] Task 4.1: Implement `FallbackRunner` in `internal/adapter/fallback.go` — wraps an `AdapterRunner` with a provider fallback chain. On `rate_limit` failure, resolves next provider's adapter and retries. Max attempts = len(chain).
- [X] Task 4.2: Wire fallback config from `Runtime.Fallbacks` into `AdapterRegistry`. `ResolveWithFallback()` returns a `FallbackRunner` when fallbacks are configured for the resolved provider.

## Phase 5: Preflight & Validation

- [X] Task 5.1: Add adapter binary validation to `internal/preflight/checker.go` — given a pipeline's steps and personas, collect all referenced adapter binaries and check each via `exec.LookPath()`.
- [X] Task 5.2: Update preflight invocation in executor to pass pipeline-specific adapter requirements.

## Phase 6: Testing

- [X] Task 6.1: Write unit tests for `AdapterRegistry` — resolution precedence, unknown adapter fallback to ProcessGroupRunner, mock injection. [P]
- [X] Task 6.2: Write unit tests for `FallbackRunner` — chain execution on rate_limit, no fallback on context_exhaustion, max attempts respected. [P]
- [X] Task 6.3: Write unit tests for Codex adapter — command building, workspace prep, output parsing. [P]
- [X] Task 6.4: Write unit tests for Gemini adapter — command building, workspace prep, output parsing. [P]
- [X] Task 6.5: Update existing `executor_test.go` tests to use registry-based construction. Verify per-step adapter resolution.
- [X] Task 6.6: Add manifest validation tests for step-level adapter/model fields and fallback config.

## Phase 7: Final Validation

- [X] Task 7.1: Run `go test -race ./...` and fix any failures.
- [X] Task 7.2: Run `golangci-lint run ./...` and fix any findings.
