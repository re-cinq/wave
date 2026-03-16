# Tasks

## Phase 1: Architecture Audit Document

- [X] Task 1.1: Create `docs/architecture-audit.md` with package inventory — enumerate all 25 internal packages with line counts, responsibilities, and public API surface
- [X] Task 1.2: Document the full internal dependency graph with import relationships (verify/update ADR-003's table against actual `go list` output)
- [X] Task 1.3: Document key design patterns — event system (producer/consumer decoupling), adapter pattern (subprocess execution), contract validation, workspace isolation, state persistence
- [X] Task 1.4: Identify structural concerns — god objects (executor.go at 3,104 lines), high fan-in packages (pipeline imports 15 internal packages), layer violations, and cyclic-risk dependencies
- [X] Task 1.5: Document the CLI→internal boundary — how `cmd/wave/commands/` connects to internal packages

## Phase 2: ADR-003 Refinement

- [X] Task 2.1: Cross-reference audit findings with ADR-003 layer classifications — verify every package is correctly classified [P]
- [X] Task 2.2: Update ADR-003 violation inventory if audit reveals additional cross-layer imports [P]
- [X] Task 2.3: Update ADR README index if ADR-003 status changes

## Phase 3: Depguard Configuration

- [X] Task 3.1: Read existing `.golangci.yml` to understand current linter configuration
- [X] Task 3.2: Add depguard rules for the two most critical boundaries: (a) no reverse imports into Presentation, (b) Infrastructure must not import Domain
- [X] Task 3.3: Allow-list the three known violations documented in ADR-003
- [X] Task 3.4: Run `golangci-lint run ./...` to verify rules pass with current codebase

## Phase 4: Validation

- [X] Task 4.1: Run `go test ./...` to confirm no regressions
- [X] Task 4.2: Run `golangci-lint run ./...` with new depguard config
- [X] Task 4.3: Final review — ensure audit document, ADR-003, and depguard config are consistent
