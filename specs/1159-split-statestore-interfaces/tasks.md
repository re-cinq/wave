# Work Items

## Phase 1: Setup

- [X] 1.1: Inventory every method on `StateStore` and confirm domain assignment matrix matches plan.md §2.
- [X] 1.2: Confirm no caller mutates an interface receiver type (would block narrowing).

## Phase 2: Define Narrow Interfaces

- [X] 2.1: Create `internal/state/runstore.go` with `RunStore` interface [P]
- [X] 2.2: Create `internal/state/eventstore.go` with `EventStore` interface [P]
- [X] 2.3: Create `internal/state/ontologystore.go` with `OntologyStore` interface [P]
- [X] 2.4: Create `internal/state/webhookstore.go` with `WebhookStore` interface [P]
- [X] 2.5: Create `internal/state/chatstore.go` with `ChatStore` interface [P]
- [X] 2.6: Rewrite `StateStore` in `internal/state/store.go` as composite embedding the five interfaces + `Close() error`
- [X] 2.7: Add `internal/state/interfaces_test.go` with compile-time `var _` assertions for each interface against `*stateStore`
- [X] 2.8: `go build ./...` and `go vet ./...` — gate before any caller changes

## Phase 3: Migrate Consumers (narrow signatures)

- [X] 3.1: `internal/retro/{storage,generator,collector}.go` → narrowed (`Generator` takes `state.RunStore`; collector/storage already use even narrower local interfaces) [P]
- [X] 3.2: `internal/ontology/{real,service}.go` → `state.OntologyStore` [P]
- [X] 3.3: `internal/pipeline/gate.go` → `state.RunStore` [P]
- [X] 3.4: `internal/pipeline/eta.go` → `state.RunStore` [P]
- [X] 3.5: `internal/pipeline/checkpoint.go` → `state.RunStore` [P]
- [X] 3.6: `internal/pipeline/sequence.go` — kept aggregate (forwards to executor via `WithStateStore`)
- [X] 3.7: `internal/pipeline/fork.go` → `state.RunStore` [P]
- [X] 3.8: `internal/pipeline/chatcontext.go` → composite `ChatContextStore` (RunStore + EventStore) for `BuildChatContext`; `state.RunStore` for `MostRecentCompletedRunID` [P]
- [X] 3.9: `internal/pipeline/composition.go` → `state.RunStore` [P]
- [X] 3.10: `internal/pipeline/stepcontroller.go` → `state.EventStore` [P]
- [X] 3.11: `internal/pipeline/executor.go` — kept aggregate (multi-domain orchestrator)
- [X] 3.12: `internal/webui/handlers_control.go` → `loggingEmitter.store` narrowed to `state.EventStore`
- [X] 3.13: `internal/webui/run_stats.go` → `state.RunStore`
- [X] 3.14: `internal/webui/server.go` — kept aggregate at top (root constructor)
- [X] 3.15: `internal/tui/health_provider.go` → `state.RunStore` [P]
- [X] 3.16: `internal/tui/pipeline_provider.go` → `state.RunStore` [P]
- [X] 3.17: `internal/tui/pipeline_detail_provider.go` → composite `detailStore` (RunStore + EventStore) [P]
- [X] 3.18: `internal/tui/persona_provider.go` → `state.RunStore` [P]
- [X] 3.19: `internal/tui/ontology_provider.go` → `state.OntologyStore` [P]
- [X] 3.20: `internal/tui/pipeline_messages.go` — kept aggregate on `LaunchDependencies` (top-level deps spanning RunStore + EventStore)
- [X] 3.21: `cmd/wave/commands/{rewind,run,status,chat,postmortem,resume,do}.go` — kept aggregate at command entry (root construction edge)

## Phase 4: Test Mocks + Test Updates

- [X] 4.1: `internal/testutil/statestore.go` — added compile-time interface-satisfaction assertions for each narrow interface and the aggregate
- [X] 4.2: Tests pass unchanged because aggregate `StateStore` still satisfies every narrowed parameter via interface embedding

## Phase 5: Validation

- [X] 5.1: `go build ./...`
- [X] 5.2: `go vet ./...`
- [X] 5.3: `go test -race ./internal/state/... ./internal/retro/... ./internal/ontology/... ./internal/pipeline/... ./internal/webui/... ./internal/tui/...`
- [ ] 5.4: `golangci-lint run ./...` (lint binary not installed in this workspace; CI will run it)
- [X] 5.5: Manual scan: `state.StateStore` references reduced from 41 → 24 files; remaining are root constructors, executor, sequence forwarder, top-level TUI deps, *_test.go fixtures, and doc comments.

## Phase 6: Polish

- [X] 6.1: Update `internal/state/doc.go` with one-line description of the split
- [ ] 6.2: PR description summarises the five interfaces and the aggregate retention rationale (PR step)
