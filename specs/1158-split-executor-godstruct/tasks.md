# Work Items

## Phase 1: Setup & Baseline

- [ ] 1.1: Confirm clean checkout on branch `1158-split-executor-godstruct`
- [ ] 1.2: Run `go build ./...` — must be clean baseline
- [ ] 1.3: Run `go test ./internal/pipeline/... -race -count=1` — capture passing baseline
- [ ] 1.4: Capture line counts for `executor.go` and any related files (post-refactor diff target)
- [ ] 1.5: Add design notes to plan.md for any surprises found while reading current code (cross-references between methods we missed)

## Phase 2: Security Layer Extraction (smallest, lowest risk — extract first)

- [ ] 2.1: Create `internal/pipeline/executor_security.go` with `securityLayer` struct
- [ ] 2.2: Move fields `securityConfig`, `pathValidator`, `inputSanitizer`, `securityLogger` from `DefaultPipelineExecutor` to `securityLayer`
- [ ] 2.3: Move methods: `loadSchemaContent`, `loadSecureSchemaContent`, `sanitizeSchemaContent`, `schemaFieldPlaceholder`, `validateSkillRefs`
- [ ] 2.4: Update coordinator: add `sec *securityLayer` field, wire in `NewDefaultPipelineExecutor`
- [ ] 2.5: Update all call sites: `e.loadSecureSchemaContent(...)` → `e.sec.loadSecureSchemaContent(...)` etc.
- [ ] 2.6: `go build ./...`, `go test ./internal/pipeline/... -race`
- [ ] 2.7: Add `executor_security_test.go` with two focused tests
- [ ] 2.8: Commit: `refactor(pipeline): extract securityLayer from executor god-struct`

## Phase 3: Delivery Layer Extraction [P with Phase 4]

- [ ] 3.1: Create `internal/pipeline/executor_delivery.go` with `deliveryLayer` struct
- [ ] 3.2: Move fields: `deliverableTracker`
- [ ] 3.3: Move methods: `writeOutputArtifacts`, `warnOnUnexpectedArtifacts`, `processStepOutcomes`, `processWildcardOutcome`, `registerOutcomeDeliverable`, `trackStepDeliverables`, `injectArtifacts`, `buildArtifactTypeMap`, `fireWebhooks`, `runTerminalHooks`
- [ ] 3.4: Update coordinator: `delivery *deliveryLayer` field; getter methods (`GetDeliverables`, `GetDeliverableTracker`) delegate
- [ ] 3.5: Update call sites
- [ ] 3.6: `go build ./...`, `go test ./internal/pipeline/... -race`
- [ ] 3.7: Add `executor_delivery_test.go` with two focused tests
- [ ] 3.8: Commit: `refactor(pipeline): extract deliveryLayer from executor god-struct`

## Phase 4: Persistence Layer Extraction [P with Phase 3]

- [ ] 4.1: Create `internal/pipeline/executor_persistence.go` with `persistenceLayer` struct
- [ ] 4.2: Move fields: `store`, `costLedger`, `etaCalculator`, `hookRunner`, `webhookRunner`, `retroGenerator`, `ontology`
- [ ] 4.3: Move methods: `recordDecision`, `recordStepOntologyUsage`, `cleanupCompletedPipeline`, `cleanupWorktrees`, `GetStatus` (delegated), `GetCostSummary` (delegated), `GetTotalCost` (delegated), `GetTotalTokens` (delegated)
- [ ] 4.4: `webhookStoreAdapter` moves with persistence
- [ ] 4.5: Update coordinator: `persist *persistenceLayer` field; public getters delegate
- [ ] 4.6: Update call sites
- [ ] 4.7: `go build ./...`, `go test ./internal/pipeline/... -race`
- [ ] 4.8: Add `executor_persistence_test.go` with two focused tests
- [ ] 4.9: Commit: `refactor(pipeline): extract persistenceLayer from executor god-struct`

## Phase 5: Execution Layer Extraction (largest — last)

- [ ] 5.1: Create `internal/pipeline/executor_execution.go` with `executionLayer` struct
- [ ] 5.2: Move scheduling/DAG methods: `executeGraphPipeline`, `runSchedulingLoop`, `findReadySteps`, `skipDependentSteps`, `hasRequiredFailures`, `executeStepBatch`
- [ ] 5.3: Move step methods: `executeStep`, `executeReworkStep`, `executeMatrixStep`, `executeConcurrentStep`, `runStepExecution`, `resolveStepResources`, `buildStepAdapterConfig`, `processAdapterResult`, `executeCommandStep`
- [ ] 5.4: Move contract methods: `validateStepContracts`, `applyContractOnFailure`, `runSingleContract`, `triggerContractRework` (+ `errContractSkip`)
- [ ] 5.5: Move composition methods: `executeCompositionStep`, `resolveSubPipelineInput`, `runNamedSubPipeline`, `executeIterateInDAG`, `executeIterateParallelInDAG`, `collectIterateOutputs`, `executeAggregateInDAG`, `executeBranchInDAG`, `executeLoopInDAG`, `executeGateInDAG`, `reQueueStep` (+ `reQueueError`)
- [ ] 5.6: Move prompt building: `buildStepPrompt`, `buildContractPrompt`, `resolveModel`, `resolveStepOutputRef`, `resolveWorkspaceStepRefs`, `warnLegacyStepOutputOnce`
- [ ] 5.7: Move workspace helpers: `createStepWorkspace`, `checkRelayCompaction`
- [ ] 5.8: Move misc helpers: `pollCancellation`, `startProgressTicker`, `trace` (move to whichever layer fits)
- [ ] 5.9: Update coordinator: `exec *executionLayer` field; `Execute` orchestration delegates
- [ ] 5.10: Update all call sites
- [ ] 5.11: `go build ./...`, `go test ./internal/pipeline/... -race`
- [ ] 5.12: Add `executor_execution_test.go` with two focused tests (`findReadySteps`, `skipDependents`)
- [ ] 5.13: Commit: `refactor(pipeline): extract executionLayer from executor god-struct`

## Phase 6: Coordinator Cleanup

- [ ] 6.1: Verify `executor.go` is now ~600-800 lines (interface, status, options, `New*`, `Execute`, `Resume`, getters, dispatch only)
- [ ] 6.2: Confirm `DefaultPipelineExecutor` struct only retains: layer fields, `pipelines`, `mu`, `runID`, `lastExecution`, CLI overrides, `emitterMixin`, `runner`, `registry`, `wsManager`, `relayMonitor`, `logger`, `debug`, `debugTracer`, `stepFilter`, `skillStore`, `gateHandler`, `parentArtifactPaths`, `parentWorkspacePath`, `crossPipelineArtifacts`, `taskComplexity`, `stackedBaseBranch`, `preserveWorkspace`, `autoApprove`, `stepTimeoutOverride`, `modelOverride`, `forceModel`, `adapterOverride`, `totalTokens`
- [ ] 6.3: Audit `NewChildExecutor` — must propagate all four layers correctly
- [ ] 6.4: Run `go vet ./...` clean
- [ ] 6.5: Run `golangci-lint run ./internal/pipeline/...` clean
- [ ] 6.6: Commit: `refactor(pipeline): slim DefaultPipelineExecutor to coordinator role`

## Phase 7: Integration & Behavior Validation

- [ ] 7.1: Run full `go test ./... -race -count=1` — must pass
- [ ] 7.2: Run targeted integration tests: `contract_integration_test.go`, `forge_integration_test.go`, `hooks_integration_test.go`, `failure_modes_test.go`, `stress_test.go`
- [ ] 7.3: Build wave binary: `go build -o /tmp/wave ./cmd/wave`
- [ ] 7.4: Run a real pipeline end-to-end (e.g. `wave run scope --adapter mock --input "test"` against a sample fixture or noop pipeline) — verify no regressions in event stream / state / artifacts
- [ ] 7.5: Confirm `internal/contract` import still present in execution layer (grep)
- [ ] 7.6: Run `wave-evolve` or `wave-bugfix` smoke if available

## Phase 8: PR Polish

- [ ] 8.1: Update commit messages to be coherent across the 5-6 commits
- [ ] 8.2: Open PR with title `refactor(pipeline): split executor god-struct into 4 layers (closes #1158)`
- [ ] 8.3: PR body lists each layer's responsibilities, file mapping, and AC checkboxes
- [ ] 8.4: Tag PR with `scope-audit`, `refactor`
- [ ] 8.5: Note in PR: subpackage split is intentional follow-up, not in scope here
