# scope: split internal/pipeline/executor.go god-struct into execution, security, persistence, and delivery layers

**Issue:** [re-cinq/wave#1158](https://github.com/re-cinq/wave/issues/1158)
**Author:** nextlevelshit
**Labels:** `scope-audit`
**State:** OPEN

## Context

From wave-scope-audit run wave-scope-audit-20260422-075324-5b6c.

Package inventory flags `internal/pipeline` as `active (bloated)`. The executor has accreted responsibilities for execution, security, persistence, and delivery into a single god-struct, making the pipeline runner hard to change safely. Scope document calls this out as a `keep (split executor god-struct)` action.

## Acceptance Criteria

- [ ] `internal/pipeline/executor.go` god-struct decomposed into distinct layers: execution, security, persistence, and delivery
- [ ] Each layer is independently testable
- [ ] Existing pipeline runner behavior preserved (no behavioral regression in integration tests)
- [ ] Contract validation still routed through `internal/contract`

## Current State

- `internal/pipeline/executor.go` is **6718 lines**, one giant struct `DefaultPipelineExecutor` with ~80 methods
- Struct holds: emitter, runner, registry, store, logger, wsManager, relayMonitor, securityConfig, pathValidator, inputSanitizer, securityLogger, deliverableTracker, etaCalculator, stepFilter, skillStore, debugTracer, hookRunner, gateHandler, retroGenerator, costLedger, webhookRunner, ontology — 25+ collaborators
- Methods conflate four concerns:
  - **Execution**: scheduling loop, DAG walk, step run, contracts, composition primitives, rework
  - **Security**: schema sanitization, path validation, secure schema load, skill ref validation
  - **Persistence**: state store, run records, decisions, ontology usage, status, cleanup, cost ledger
  - **Delivery**: artifact writes, outcomes, deliverables, webhooks, terminal hooks

## Constraints

- No behavioral regression — existing integration tests must pass unchanged
- Contract validation must still route through `internal/contract`
- `PipelineExecutor` interface and `DefaultPipelineExecutor` exported API stay stable (used by CLI, TUI, WebUI)
- Pre-1.0: no backward-compat shims required for removed internals
