# Implementation Plan — Split Executor God-Struct

## 1. Objective

Decompose the 6718-line `internal/pipeline/executor.go` god-struct into four
independently testable layers (execution, security, persistence, delivery)
without changing the public `PipelineExecutor` API or behavior.

## 2. Approach

**Strangler-fig refactor inside `internal/pipeline`** — no new sub-packages
(import-cycle risk with shared `Pipeline`/`Step`/`PipelineExecution` types is
high). Extract collaborator structs that own their state and methods. The
`DefaultPipelineExecutor` becomes a thin coordinator that wires layers
together and dispatches to them.

Why same package, not subpackages:
- `PipelineExecution`, `Step`, `Pipeline`, `PipelineContext`, `OutcomeDef` are
  package-level types referenced by every layer
- Splitting subpackages now would require either pulling those types up into a
  new `internal/pipeline/types` package (large blast radius) or duplicating
  them (forbidden)
- Same-package extraction still satisfies the "independently testable" AC —
  each layer is a struct with its own constructor, can be tested in isolation
  with stub dependencies. We do NOT need package boundaries for testability.
- Sub-package split is a follow-up once the layer boundaries are proven stable

### Layer boundaries

| Layer | Owns | Methods (sample) |
|---|---|---|
| **execution** | scheduling, DAG walk, step run, contracts, composition | `executeGraphPipeline`, `runSchedulingLoop`, `executeStep`, `runStepExecution`, `executeMatrixStep`, `executeConcurrentStep`, `executeReworkStep`, `executeCompositionStep`, `executeIterateInDAG`, `executeAggregateInDAG`, `executeBranchInDAG`, `executeLoopInDAG`, `executeGateInDAG`, `findReadySteps`, `skipDependentSteps`, `validateStepContracts`, `applyContractOnFailure`, `runSingleContract`, `triggerContractRework`, `processAdapterResult`, `resolveStepResources`, `buildStepAdapterConfig`, `buildStepPrompt`, `buildContractPrompt` |
| **security** | path validation, input sanitization, schema sanitization, skill ref validation | `loadSchemaContent`, `loadSecureSchemaContent`, `sanitizeSchemaContent`, `validateSkillRefs`, plus owns `pathValidator` / `inputSanitizer` / `securityLogger` / `securityConfig` |
| **persistence** | state store ops, run records, decisions, status, ontology usage, costs, ETA, hooks | `recordDecision`, `recordStepOntologyUsage`, `cleanupCompletedPipeline`, `cleanupWorktrees`, `GetStatus`, `GetCostSummary`, plus owns `store` / `costLedger` / `etaCalculator` / `hookRunner` / `webhookRunner` / `retroGenerator` / `ontology` |
| **delivery** | output artifacts, outcomes, deliverables, webhooks, terminal hooks | `writeOutputArtifacts`, `warnOnUnexpectedArtifacts`, `processStepOutcomes`, `processWildcardOutcome`, `registerOutcomeDeliverable`, `trackStepDeliverables`, `injectArtifacts`, `buildArtifactTypeMap`, `fireWebhooks`, `runTerminalHooks`, plus owns `deliverableTracker` |

Coordinator (`DefaultPipelineExecutor`) keeps:
- The `PipelineExecutor` interface impl: `Execute`, `Resume`, `ResumeWithValidation`, `GetStatus`, `LastExecution`, getters
- `pipelines` map, `mu`, `runID`, `lastExecution`, CLI overrides (`modelOverride`, `forceModel`, `adapterOverride`, `stepFilter`, `taskComplexity`, `preserveWorkspace`, `stackedBaseBranch`, `autoApprove`, `gateHandler`, `parentArtifactPaths`, `parentWorkspacePath`, `crossPipelineArtifacts`, `debug`, `debugTracer`, `stepTimeoutOverride`, `totalTokens`)
- The four layer fields: `exec *executionLayer`, `sec *securityLayer`, `persist *persistenceLayer`, `delivery *deliveryLayer`
- Construction (`NewDefaultPipelineExecutor`) wires all four layers from options

### Cross-layer coupling

Layers reference each other through the coordinator. Each layer holds a back-
pointer to `*DefaultPipelineExecutor` for now (avoids defining N narrow
interfaces in step 1). After extraction stabilizes, narrow interfaces can be
introduced — that's a follow-up, not in scope here.

## 3. File Mapping

### New files (`internal/pipeline/`)

- `executor_execution.go` — `executionLayer` struct, scheduling/DAG/step/contract/composition methods
- `executor_security.go` — `securityLayer` struct, schema + path + sanitization + skill-ref methods
- `executor_persistence.go` — `persistenceLayer` struct, store/decisions/status/ontology/cost/ETA/hooks
- `executor_delivery.go` — `deliveryLayer` struct, artifact/outcome/deliverable/webhook methods

### New tests

- `executor_execution_test.go` — scheduling order, ready-step calc, rework triggering, composition dispatch
- `executor_security_test.go` — schema sanitization, path validation, skill ref errors
- `executor_persistence_test.go` — decision recording, status, cleanup, ontology usage
- `executor_delivery_test.go` — artifact writes, outcome processing, deliverable tracking, webhook fire

### Modified files

- `internal/pipeline/executor.go` — shrinks to coordinator (~600-800 lines): struct, options, `New*`, `Execute`, `Resume*`, `GetStatus`, getters, dispatch helpers
- `internal/pipeline/executor_test.go` — keep existing integration coverage; rename only if helpers move

### Untouched (no API changes)

- `internal/contract/*` — contract validation routing preserved
- `internal/pipeline/types.go`, `dag.go`, `graph.go`, `composition.go`, `gate.go`, etc. — same package, layers reference these
- CLI/TUI/WebUI callers — exported API stable

### No deletions in this PR

Old code is moved, not removed (besides the methods being relocated). The end
result is: `executor.go` shrinks, four new files appear.

## 4. Architecture Decisions

1. **Same package over subpackages.** Reason: shared types create cycles. Subpackage split is a possible follow-up after boundaries prove out.
2. **Layers as structs with back-pointer to coordinator.** Reason: minimizes risk of missing edge dependencies during extraction. Narrow interfaces are a follow-up refinement.
3. **Move, don't rewrite.** Method bodies are translated verbatim with receiver swap (`(e *DefaultPipelineExecutor)` → `(s *securityLayer)`). Behavior is byte-identical; tests prove it.
4. **Preserve `DefaultPipelineExecutor` exported API.** All public methods on the coordinator remain. Internally, public methods delegate (e.g. `e.GetStatus(...)` → `e.persist.getStatus(...)`).
5. **Coordinator keeps CLI override fields.** These are configuration scalars, not collaborators. Layers that need them read through `e` back-pointer.
6. **Tests per layer.** Each layer gets a `_test.go` with at least one focused test that constructs only that layer. This satisfies "independently testable" without replacing the existing integration tests.
7. **No subpipeline / sequence / resume / fork extraction in this PR.** Those have their own files already (`subpipeline.go`, `sequence.go`, `resume.go`, `fork.go`) and call into the coordinator's API. Out of scope.

## 5. Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Behavioral regression from method relocation | All moves are pure receiver swaps. `go test ./... -race` and full integration suite run before merge. |
| Hidden field access from a method that's now in another struct | Layers hold back-pointer `e *DefaultPipelineExecutor`; any field access still works. |
| `go vet` shadow / unused warnings on extraction | Compile after each layer extraction; fix immediately. |
| Test files referencing private methods | Tests in same package can still call moved methods. If a test directly calls `e.runSingleContract(...)`, it now becomes `e.exec.runSingleContract(...)`. Update test call sites. |
| Diff size scares reviewers | Land in 4 PRs (one per layer) OR one PR with clear commit-per-layer history. PR strategy: one PR, four commits. |
| Composition primitives have subtle DAG state coupling | Keep all of `executeIterate*`/`executeAggregate*`/`executeBranch*`/`executeLoop*`/`executeGate*` together in `executor_execution.go`. Don't split mid-feature. |
| Contract routing accidentally bypasses `internal/contract` | `runSingleContract`, `validateStepContracts`, `applyContractOnFailure`, `triggerContractRework` all stay together in execution layer. Acceptance test asserts contract package still imported. |
| `emitterMixin` field collision | Coordinator keeps `emitterMixin` embed. Layers that emit do so through `e.emit*` accessors. |

## 6. Testing Strategy

### Pre-refactor baseline
1. `go build ./...` clean
2. `go test ./internal/pipeline/... -race -count=1` clean — capture coverage
3. Record `go test ./... -run Integration` passing list

### During refactor (per layer)
1. After each layer extraction commit, run `go build ./...`
2. Run `go test ./internal/pipeline/... -race`
3. Diff coverage — must not drop on existing tests

### New per-layer tests (one file each, minimal but real)
- `executor_security_test.go` — `TestSecurityLayer_SanitizeSchemaContent_RejectsCRLFInjection`, `TestSecurityLayer_LoadSecureSchemaContent_PathTraversal`
- `executor_persistence_test.go` — `TestPersistenceLayer_RecordDecision_PersistsToStore`, `TestPersistenceLayer_GetStatus_ReportsPipeline`
- `executor_delivery_test.go` — `TestDeliveryLayer_WriteOutputArtifacts_RespectsExisting`, `TestDeliveryLayer_ProcessStepOutcomes_RegistersDeliverables`
- `executor_execution_test.go` — `TestExecutionLayer_FindReadySteps_RespectsDependencies`, `TestExecutionLayer_SkipDependents_PropagatesFailure`

### Post-refactor validation
1. `go test ./... -race -count=1` — full suite
2. Run a real pipeline end-to-end via `wave run scope --input <test-issue> --detach` against a sample repo
3. Confirm contract validation still flows through `internal/contract` (grep import in execution layer)
4. Run `internal/pipeline/contract_integration_test.go`, `forge_integration_test.go`, `hooks_integration_test.go`, `failure_modes_test.go`, `stress_test.go` explicitly

### Out of scope for this PR
- Performance tuning
- Sub-package split
- Narrow interface introduction between layers
- Resume/fork/sequence refactoring
