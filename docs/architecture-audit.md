# Wave Architecture Audit

**Date**: 2026-03-16
**Scope**: Full inventory of `internal/` packages, dependency graph, design patterns, and structural concerns.
**Purpose**: Baseline reference for [ADR-003](adr/003-layered-architecture.md) layered architecture transition.

## Package Inventory

Wave's `internal/` directory contains **25 Go packages** totaling ~48,655 lines of production code and ~62,552 lines of test code.

| Package | Prod Lines | Test Lines | Responsibility |
|---------|-----------|------------|----------------|
| `adapter` | 2,914 | 4,301 | Subprocess execution for Claude Code and other LLM CLIs |
| `audit` | 133 | 709 | Audit logging with credential scrubbing |
| `contract` | 4,030 | 3,498 | Output validation (JSON schema, TypeScript, test suites, markdown spec) |
| `defaults` | 209 | 529 | Embedded default personas, pipelines, and contracts |
| `display` | 5,294 | 5,605 | Terminal progress display and formatting |
| `doctor` | 2,474 | 2,500 | Project health checking and optimization |
| `event` | 200 | 642 | Progress event emission and monitoring (producer/consumer) |
| `forge` | 209 | 341 | Git forge/hosting platform detection (GitHub, GitLab, etc.) |
| `github` | 1,188 | 722 | GitHub API integration (issue enhancement, PR operations) |
| `manifest` | 555 | 1,816 | Configuration loading and validation (`wave.yaml`) |
| `onboarding` | 1,498 | 1,056 | Interactive `wave init` flow |
| `pathfmt` | 22 | 66 | Path formatting and normalization utilities |
| `pipeline` | 11,050 | 23,742 | Pipeline execution, DAG traversal, step management |
| `preflight` | 288 | 493 | Pipeline dependency validation and auto-install |
| `recovery` | 295 | 677 | Pipeline recovery hints and error guidance |
| `relay` | 422 | 2,780 | Context compaction and summarization |
| `security` | 955 | 1,195 | Security validation, path sanitization, permission enforcement |
| `skill` | 1,841 | 4,030 | Skill discovery, provisioning, and command management |
| `state` | 2,828 | 2,815 | SQLite persistence and state management |
| `suggest` | 362 | 405 | Pipeline suggestion engine |
| `tui` | 13,092 | 11,403 | Bubble Tea terminal UI (pipeline list, detail, live output) |
| `webui` | 2,111 | 907 | Web operations dashboard (behind `//go:build webui` tag) |
| `workspace` | 289 | 940 | Ephemeral workspace management |
| `worktree` | 134 | 335 | Git worktree lifecycle for isolated workspaces |

## Internal Dependency Graph

Each row shows which internal packages are imported.

| Package | Internal Imports | Fan-out |
|---------|-----------------|---------|
| `adapter` | `github` | 1 |
| `audit` | *(none)* | 0 |
| `contract` | `pathfmt` | 1 |
| `defaults` | `manifest`, `pipeline` | 2 |
| `display` | `event`, `pathfmt`, `state` | 3 |
| `doctor` | `forge`, `github`, `manifest`, `onboarding`, `pipeline` | 5 |
| `event` | *(none)* | 0 |
| `forge` | *(none)* | 0 |
| `github` | *(none)* | 0 |
| `manifest` | `skill` | 1 |
| `onboarding` | `manifest`, `skill`, `tui` | 3 |
| `pathfmt` | *(none)* | 0 |
| `pipeline` | `adapter`, `audit`, `contract`, `event`, `forge`, `manifest`, `preflight`, `recovery`, `relay`, `security`, `skill`, `state`, `workspace`, `worktree` | 14 |
| `preflight` | `skill` | 1 |
| `recovery` | `contract`, `pathfmt`, `preflight`, `security` | 4 |
| `relay` | *(none)* | 0 |
| `security` | *(none)* | 0 |
| `skill` | *(none)* | 0 |
| `state` | *(none)* | 0 |
| `suggest` | `doctor`, `forge` | 2 |
| `tui` | `display`, `event`, `forge`, `github`, `manifest`, `pathfmt`, `pipeline`, `state` | 8 |
| `webui` | `adapter`, `audit`, `display`, `event`, `manifest`, `pipeline`, `state`, `workspace` (behind `webui` build tag) | 8 |
| `workspace` | *(none)* | 0 |
| `worktree` | *(none)* | 0 |

### Fan-in (packages that depend on each package)

| Package | Depended On By | Fan-in |
|---------|---------------|--------|
| `manifest` | `defaults`, `doctor`, `onboarding`, `pipeline`, `tui`, `webui` | 6 |
| `skill` | `manifest`, `onboarding`, `pipeline`, `preflight` | 4 |
| `event` | `display`, `pipeline`, `tui`, `webui` | 4 |
| `pathfmt` | `contract`, `display`, `recovery`, `state`, `tui` | 5 |
| `pipeline` | `defaults`, `doctor`, `tui`, `webui` | 4 |
| `forge` | `doctor`, `pipeline`, `suggest`, `tui` | 4 |
| `state` | `pipeline`, `tui`, `webui` | 3 |
| `adapter` | `pipeline`, `webui` | 2 |
| `security` | `pipeline`, `recovery` | 2 |
| `display` | `tui`, `webui` | 2 |
| `github` | `adapter`, `doctor`, `tui` | 3 |
| `contract` | `pipeline`, `recovery` | 2 |
| `workspace` | `pipeline`, `webui` | 2 |
| `audit` | `pipeline`, `webui` | 2 |
| `preflight` | `pipeline`, `recovery` | 2 |
| `doctor` | `suggest` | 1 |
| `onboarding` | `doctor` | 1 |
| `tui` | `onboarding` | 1 |
| `worktree` | `pipeline` | 1 |
| `relay` | `pipeline` | 1 |
| `recovery` | `pipeline` | 1 |
| `suggest` | *(none internal)* | 0 |
| `defaults` | *(none internal)* | 0 |

### CLI Boundary

`cmd/wave/commands/` is the entry point connecting CLI to internal packages. It imports:
`adapter`, `audit`, `defaults`, `display`, `doctor`, `event`, `forge`, `manifest`, `onboarding`, `pipeline`, `preflight`, `recovery`, `skill`, `state`, `suggest`, `tui`, `workspace`

`cmd/wave/main.go` imports: `commands`, `doctor`, `manifest`, `state`, `suggest`, `tui`

The CLI layer acts as the composition root, wiring together all internal packages.

## Key Design Patterns

### 1. Event System (Producer/Consumer Decoupling)

The `event` package provides a publish-subscribe mechanism that decouples pipeline execution from display rendering. The `pipeline` package emits structured progress events (`StepStarted`, `StepCompleted`, `ContractValidating`, etc.) and `display`/`tui` packages consume them without direct coupling to the pipeline internals.

- **Producer**: `pipeline/executor.go` emits events via `event.Emitter`
- **Consumers**: `display/progress.go` renders terminal output, `tui/` renders Bubble Tea UI
- **Benefit**: Adding new display modes (e.g., `webui`) requires no pipeline changes

### 2. Adapter Pattern (Subprocess Execution)

The `adapter` package abstracts LLM CLI execution behind an `Adapter` interface. The primary implementation (`ClaudeAdapter`) manages subprocess lifecycle, I/O streaming, and settings.json generation.

- **Interface**: `Adapter` with `Run(ctx, config) (Result, error)`
- **Config**: `AdapterRunConfig` encapsulates workspace path, system prompt, permissions, timeout
- **Testability**: `MockAdapter` supports deterministic testing without subprocess execution

### 3. Contract Validation

The `contract` package validates step outputs against declarative schemas before marking steps successful. Supported validators: `json_schema`, `typescript_interface`, `test_suite`, `markdown_spec`, `format`.

- **Hard failures**: Block step completion, prevent downstream execution
- **Soft failures**: Log warnings, allow step to proceed
- **Recovery**: `recovery` package provides hints for common contract failures

### 4. Workspace Isolation

The `workspace` package creates ephemeral directories for step execution. The `worktree` package manages git worktrees for full repository isolation.

- **Mount modes**: `readonly`, `readwrite` for shared workspace access
- **Artifact injection**: Outputs from prior steps are injected into `.agents/artifacts/`
- **Cleanup**: Workspaces are cleaned up after pipeline completion (configurable retention)

### 5. State Persistence

The `state` package provides SQLite-backed persistence for pipeline runs, step status, and resumption data.

- **Run tracking**: Pipeline run metadata, step status, timestamps
- **Resume support**: Failed pipelines can resume from the last failed step
- **Migration system**: Versioned schema migrations in `internal/state/`

### 6. CLAUDE.md Assembly

At each step boundary, a per-step CLAUDE.md is generated from four layers:
1. Base protocol preamble (`.agents/personas/base-protocol.md`)
2. Persona system prompt (role, responsibilities, constraints)
3. Contract compliance section (auto-generated from step contract schema)
4. Restriction section (denied/allowed tools, network domains)

This ensures fresh memory at every step boundary — no chat history inheritance.

## Structural Concerns

### 1. God Object: `executor.go` (3,104 lines)

`internal/pipeline/executor.go` is the largest single file, handling:
- DAG traversal and topological sorting
- Step lifecycle management
- Workspace creation and cleanup
- Artifact injection and extraction
- CLAUDE.md assembly
- Adapter invocation
- Contract validation orchestration
- Event emission
- State persistence
- Error recovery
- Concurrent step execution

This is tracked in [ADR-002](adr/002-extract-step-executor.md) which proposes extracting a `StepExecutor` component.

### 2. High Fan-out: `pipeline` (15 internal imports)

The `pipeline` package imports 15 of the 25 internal packages (60%), making it a coupling hotspot. Any change to its dependencies risks cascading effects. This fan-out is partially inherent to its role as the orchestration core, but ADR-002's `StepExecutor` extraction would distribute some of these dependencies.

### 3. High Fan-out: `tui` and `webui` (8 internal imports each)

Both presentation packages import 8 internal packages. `tui` imports `pipeline` directly for type information and execution, coupling the terminal UI to orchestration internals. The `webui` package (behind a build tag) has similar coupling.

### 4. Layer Violations

Using ADR-003's four-layer model, the following cross-layer violations exist:

| Violation | Direction | Severity |
|-----------|-----------|----------|
| `doctor` → `onboarding` | Domain → Presentation | Medium |
| `webui` → `adapter`, `workspace` | Presentation → Domain/Infrastructure | Medium |
| `defaults` → `pipeline` | Domain → Domain (circular risk) | Low |
| `manifest` → `skill` | Cross-cutting → Domain | Low |

The `manifest` → `skill` violation was not previously documented in ADR-003. The `manifest` package (cross-cutting) imports `skill` (domain) for skill configuration types.

### 5. Build Tag Isolation

The `webui` package is gated behind `//go:build webui`, meaning its dependency violations only materialize when building with that tag. This provides effective isolation in the default build but means standard `go list` analysis misses its imports.

### 6. Leaf Packages (Zero Internal Dependencies)

Nine packages have no internal imports: `audit`, `event`, `forge`, `github`, `pathfmt`, `relay`, `security`, `skill`, `state`, `workspace`, `worktree`. These are healthy leaf nodes that can be tested and reasoned about in isolation.

### 7. Test Coverage Distribution

Test-to-production ratio varies significantly:

| Package | Ratio (test:prod) | Notes |
|---------|--------------------|-------|
| `relay` | 6.6:1 | Well-tested relative to size |
| `audit` | 5.3:1 | Well-tested |
| `pipeline` | 2.1:1 | Adequate given complexity |
| `manifest` | 3.3:1 | Well-tested |
| `workspace` | 3.3:1 | Well-tested |
| `tui` | 0.9:1 | Could use more test coverage |
| `webui` | 0.4:1 | Low test coverage |

## ADR-003 Discrepancies Found

During this audit, the following discrepancies with [ADR-003](adr/003-layered-architecture.md) were identified and corrected:

1. **`manifest` → `skill`**: Not listed in ADR-003's dependency table. `manifest` was shown with no internal imports, but it imports `skill` for skill configuration types. This is a cross-cutting → domain violation.

2. **`pipeline` → `forge`, `recovery`**: ADR-003's dependency table listed 13 internal imports for `pipeline`, but the actual count is 15. Missing: `forge` (infrastructure) and `recovery` (domain).

3. **`onboarding` → `skill`**: ADR-003 listed `onboarding` as importing `manifest`, `tui` but it also imports `skill`. This is an allowed import (presentation → domain) but was missing from the table.

4. **`executor.go` line count**: ADR-003 referenced "2,493+ lines" but the current count is 3,104 lines.
