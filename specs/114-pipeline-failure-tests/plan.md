# Implementation Plan: Pipeline Failure Mode Tests

## Objective

Add comprehensive integration and unit tests covering 7 pipeline failure modes to ensure pipelines never report success on failures (false-positive detection). Tests must use the existing mock adapter infrastructure and run under `go test -race ./...`.

## Approach

### Strategy: Mock-Based Integration Tests

Use the existing `MockAdapter` and `stepAwareAdapter` patterns (from `executor_test.go`) to simulate each failure mode without requiring real subprocess execution. This approach:

1. Tests the full `DefaultPipelineExecutor.Execute()` flow (DAG resolution, workspace creation, artifact injection, adapter execution, contract validation)
2. Runs fast (no real LLM calls)
3. Is deterministic and race-free
4. Follows existing patterns in `internal/pipeline/executor_test.go` and `internal/pipeline/contract_integration_test.go`

### Two Test Files

1. **`internal/pipeline/failure_modes_test.go`** — Pipeline-level integration tests for all 7 failure scenarios using `DefaultPipelineExecutor.Execute()` with mock adapters
2. **`internal/contract/false_positive_test.go`** — Unit tests specifically targeting contract validator false-positive detection (malformed output that should be rejected)

### Pipeline-Specific Tests (Coverage for Named Pipelines)

Rather than testing each named pipeline (gh-rewrite, dead-code, etc.) end-to-end (which would require real adapter execution), we test the **failure patterns** that apply across all pipelines:
- Contract schema validation with pipeline-representative schemas
- Multi-step artifact dependency chains
- Timeout propagation with step-level overrides

This is the same approach used in the existing `prototype_*_test.go` files.

## File Mapping

| File | Action | Purpose |
|------|--------|---------|
| `internal/pipeline/failure_modes_test.go` | **create** | 7 integration tests for pipeline failure modes |
| `internal/contract/false_positive_test.go` | **create** | Unit tests for contract false-positive detection |

## Architecture Decisions

1. **No new production code** — this issue is purely additive tests. The production behavior already exists; we're validating it.

2. **Test placement** — tests go in the same package as the code they test (`pipeline` and `contract` packages) to access internal types (`stepAwareAdapter`, `contractTestPromptCapturingAdapter`, etc.)

3. **Mock adapter patterns** — reuse existing `adapter.WithFailure()`, `adapter.WithExitCode()`, `adapter.WithSimulatedDelay()` options. Create `stepAwareAdapter` variants where needed to simulate per-step behavior.

4. **Table-driven tests** — use table-driven patterns for contract false-positive scenarios (consistent with existing `contract_test.go` style).

5. **No real pipeline YAML loading** — construct `Pipeline` structs in Go directly (matching existing test patterns in `executor_test.go`). This avoids fragile file dependencies and tests the execution engine directly.

## Risks

| Risk | Mitigation |
|------|------------|
| Tests could be flaky due to timing (timeout tests) | Use short timeouts (50-200ms) with generous context timeouts (30s), matching existing patterns |
| Workspace permission tests may behave differently on CI vs local | Test permission semantics through error propagation, not raw filesystem permissions |
| Contract false-positive tests may need maintenance as validators evolve | Keep test cases focused on obvious malformed input (type mismatches, missing fields, truncated JSON) |
| Tests might duplicate existing coverage | Audit existing tests first; focus on gaps (workspace corruption, permission denial, non-zero exit code without adapter error) |

## Testing Strategy

### Coverage Gaps to Fill (Based on Codebase Analysis)

| Failure Mode | Existing Coverage | Gap |
|---|---|---|
| Contract schema mismatch | `contract_integration_test.go:227` — covered for must_pass=true | Need: pipeline returns non-zero exit, event has "failed" state |
| Step timeout | `executor_test.go:3508-3668` — timeout config tests | Need: actual timeout via `context.DeadlineExceeded` propagation |
| Missing artifact | `executor_test.go:2110` — covered for non-existent step | Need: artifact exists but is malformed/empty |
| Permission denial | `prototype_e2e_test.go:331` — persona permissions test | Need: test that denied tool results in step failure |
| Workspace corruption | No existing tests | Need: workspace dir removed/unwritable mid-run |
| Non-zero exit code | No dedicated test | Need: adapter returns non-zero exit code, verify pipeline behavior |
| False-positive detection | `contract_test.go:64-165` — validation failure tests | Need: edge cases (truncated JSON, wrong types masquerading as correct, empty objects passing required field checks) |

### Test Design

Each test follows this pattern:
1. Create mock adapter with the specific failure mode
2. Construct pipeline with appropriate steps and contracts
3. Execute via `DefaultPipelineExecutor.Execute()`
4. Assert: error returned, error contains expected message, events contain expected states
5. Assert: no false-positive success events after failure
