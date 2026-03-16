# Implementation Plan: Nix Flake Packaging and Docker-Based Sandbox

**Branch**: `175-docker-sandbox-nix-flake` | **Date**: 2026-03-16 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `specs/175-docker-sandbox-nix-flake/spec.md`

## Summary

Add two orthogonal capabilities to Wave: (1) fix the existing `flake.nix` package definition so `nix run github:re-cinq/wave` produces a working binary, and (2) implement a Docker-based sandbox backend that wraps adapter subprocess execution in Docker containers with read-only root, isolated HOME, curated environment, capability dropping, and network isolation. The Docker backend is configured via `runtime.sandbox.backend: docker` in `wave.yaml` and validated during preflight.

## Technical Context

**Language/Version**: Go 1.25+ (matches `go.mod`)
**Primary Dependencies**: `os/exec` (Docker CLI invocation), `gopkg.in/yaml.v3` (manifest parsing), Nix flakes (`buildGoModule`)
**Storage**: N/A (no database changes)
**Testing**: `go test -race ./...`, manual Nix flake verification
**Target Platform**: Linux (native Docker), macOS (Docker Desktop), Windows WSL2 (Docker Desktop)
**Project Type**: Single Go binary (CLI)
**Performance Goals**: Docker sandbox adds <2s overhead per step (container create+start)
**Constraints**: Single static binary (Constitution P1), no new runtime dependencies beyond Docker CLI
**Scale/Scope**: ~10 new/modified files, ~500-800 lines of new Go code

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | PASS | Wave binary remains a single Go binary. Docker is an external runtime dependency (like adapter binaries), not bundled. |
| P2: Manifest as SSOT | PASS | `runtime.sandbox.backend` and `runtime.sandbox.docker_image` added to `wave.yaml`. All config traces to manifest. |
| P3: Persona-Scoped Execution | PASS | Per-persona `sandbox.allowed_domains` overrides runtime defaults. Sandbox enforces persona permissions. |
| P4: Fresh Memory | PASS | Each Docker container is ephemeral — no state leaks between steps. |
| P5: Navigator-First | N/A | No change to pipeline step ordering. |
| P6: Contracts at Handover | N/A | No change to contract validation. Sandbox is transparent to contract system. |
| P7: Relay via Summarizer | N/A | No change to relay/compaction. |
| P8: Ephemeral Workspaces | PASS | Docker containers are ephemeral. Workspace bind mounts mirror existing worktree isolation. |
| P9: Credentials Never Touch Disk | PASS | Docker sandbox passes credentials only via `-e` flags (environment variables). No disk persistence. |
| P10: Observable Progress | PASS | Docker sandbox is transparent to event system — adapter streams through Docker stdout/stderr. |
| P11: Bounded Recursion | N/A | No change to meta-pipeline limits. |
| P12: Minimal Step State Machine | N/A | No new states. Docker failures map to existing Failed state. |
| P13: Test Ownership | PASS | New package `internal/sandbox/` has full test coverage. Modified packages update existing tests. |

## Project Structure

### Documentation (this feature)

```
specs/175-docker-sandbox-nix-flake/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── contracts/
│   ├── sandbox-config.yaml    # Configuration schema contract
│   └── sandbox-interface.go   # Go interface contract
└── tasks.md             # Phase 2 output (NOT created by plan)
```

### Source Code (repository root)

```
internal/
├── sandbox/              # NEW package
│   ├── types.go          # SandboxBackendType enum, Config struct
│   ├── sandbox.go        # Sandbox interface definition
│   ├── docker.go         # DockerSandbox implementation
│   ├── docker_test.go    # DockerSandbox unit tests
│   ├── none.go           # NoneSandbox (passthrough) implementation
│   ├── none_test.go      # NoneSandbox tests
│   ├── factory.go        # NewSandbox(backendType) factory
│   └── factory_test.go   # Factory tests
├── manifest/
│   └── types.go          # MODIFY: add Backend, DockerImage to RuntimeSandbox
├── adapter/
│   └── adapter.go        # MODIFY: add SandboxBackend, DockerImage to AdapterRunConfig
├── pipeline/
│   └── executor.go       # MODIFY: resolve sandbox backend, wire to adapter config
└── preflight/
    └── preflight.go      # MODIFY: add CheckDockerDaemon() method

flake.nix                 # MODIFY: fix vendorHash for packages.wave
```

**Structure Decision**: All new code lives in `internal/sandbox/` as a new package. Existing packages receive minimal modifications (field additions, config wiring). The flake.nix change is a one-line hash fix.

## Implementation Phases

### Phase A: Nix Flake Fix (FR-001 through FR-006)

1. Compute the correct `vendorHash` for `buildGoModule` by running `nix build .#wave`
2. Replace `pkgs.lib.fakeHash` with the computed hash
3. Verify `nix build .#wave` produces a working binary
4. Verify `nix run .#wave -- --version` outputs version info
5. Verify `nix flake check` passes
6. Verify `nix develop` still provides the existing dev shell (FR-003, FR-023, FR-025)

### Phase B: Manifest Schema Extension (FR-017, FR-017a, FR-018, FR-024)

1. Add `Backend string` and `DockerImage string` to `RuntimeSandbox` in `internal/manifest/types.go`
2. Add `ResolveBackend()` and `GetDockerImage()` methods
3. Update manifest parser tests to cover new fields
4. Verify backward compatibility: manifests without `backend` field parse correctly

### Phase C: Sandbox Package (FR-007 through FR-016b)

1. Create `internal/sandbox/types.go` — enum + config struct
2. Create `internal/sandbox/sandbox.go` — interface definition
3. Create `internal/sandbox/none.go` — passthrough implementation
4. Create `internal/sandbox/docker.go` — Docker implementation:
   - `Wrap()`: construct `docker run` command with all isolation flags
   - `Validate()`: check Docker daemon via `docker info`
   - `Cleanup()`: remove Docker network if created
5. Create `internal/sandbox/factory.go` — `NewSandbox(backendType)` factory
6. Full test coverage for each implementation

### Phase D: Pipeline Integration (FR-017 through FR-019)

1. Modify `AdapterRunConfig` to carry `SandboxBackend` and `DockerImage`
2. Modify `executor.go` `buildAdapterConfig()` to resolve backend via `ResolveBackend()`
3. Modify adapter `Run()` methods to apply sandbox wrapping
4. Wire sandbox cleanup into step completion/failure paths

### Phase E: Preflight Validation (FR-020 through FR-022)

1. Add `CheckDockerDaemon()` to `preflight.Checker`
2. Wire Docker preflight check when `backend: docker` is configured
3. Provide platform-specific recovery hints
4. Add bubblewrap preflight check when `backend: bubblewrap` is configured

### Phase F: Testing and Verification (SC-001 through SC-010)

1. Unit tests for all new code
2. Integration tests with mock Docker binary
3. `go test -race ./...` pass
4. Manual verification of Nix flake (SC-001 through SC-003)
5. Manual verification of Docker sandbox (SC-004 through SC-006)

## Complexity Tracking

_No constitution violations. No complexity justifications needed._

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|-----------|--------------------------------------|
| None | — | — |
