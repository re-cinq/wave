# Tasks: Nix Flake Packaging and Docker-Based Sandbox

**Branch**: `175-docker-sandbox-nix-flake` | **Date**: 2026-03-16 | **Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)

## Phase 1: Setup

- [ ] T001 [P1] Setup — Create `internal/sandbox/` package directory and `types.go` with `SandboxBackendType` enum (`none`, `docker`, `bubblewrap`) and `Config` struct matching `contracts/sandbox-interface.go` — `internal/sandbox/types.go`

## Phase 2: Foundational — Manifest Schema Extension (US3: Configure Sandbox Backend)

- [ ] T002 [P1] [US3] Add `Backend string` and `DockerImage string` fields to `RuntimeSandbox` struct — `internal/manifest/types.go`
- [ ] T003 [P1] [US3] Add `ResolveBackend() SandboxBackendType` method to `RuntimeSandbox` (backend field supersedes enabled boolean per C1) — `internal/manifest/types.go`
- [ ] T004 [P1] [US3] Add `GetDockerImage() string` method to `RuntimeSandbox` (default `ubuntu:24.04` per C2) — `internal/manifest/types.go`
- [ ] T005 [P1] [US3] Add unit tests for `ResolveBackend()` covering: backend set, enabled-only legacy, neither set, both set (backend wins) — `internal/manifest/types_test.go`
- [ ] T006 [P] [P1] [US3] Add manifest parser tests for new `backend` and `docker_image` YAML fields, including backward-compat test (no backend field parses correctly) — `internal/manifest/parser_test.go`

## Phase 3: Sandbox Interface & Implementations (US2: Docker Sandbox)

- [ ] T007 [P2] [US2] Create `internal/sandbox/sandbox.go` with `Sandbox` interface (`Wrap`, `Validate`, `Cleanup` methods) matching contract — `internal/sandbox/sandbox.go`
- [ ] T008 [P] [P2] [US2] Create `internal/sandbox/none.go` — `NoneSandbox` passthrough implementation (Wrap returns cmd unchanged, Validate/Cleanup are no-ops) — `internal/sandbox/none.go`
- [ ] T009 [P] [P2] [US2] Create `internal/sandbox/none_test.go` — tests for `NoneSandbox` — `internal/sandbox/none_test.go`
- [ ] T010 [P2] [US2] Create `internal/sandbox/docker.go` — `DockerSandbox` with `NewDockerSandbox()` constructor (resolves docker binary), `Validate()` runs `docker info`, `Wrap()` builds `docker run` command with: `--rm`, `--read-only`, tmpfs mounts (`/tmp`, `/var/run`, `/home/wave`), `-e HOME=/home/wave`, env passthrough, workspace bind mount, artifact/output bind mounts, `--cap-drop=ALL`, `--security-opt=no-new-privileges`, `--user UID:GID`, `--network=none` — `internal/sandbox/docker.go`
- [ ] T011 [P2] [US2] Create `internal/sandbox/docker_test.go` — unit tests for `DockerSandbox`: Wrap produces correct docker args, Validate behavior, env passthrough filtering, mount construction, UID/GID mapping — `internal/sandbox/docker_test.go`
- [ ] T012 [P2] [US2] Create `internal/sandbox/factory.go` — `NewSandbox(backendType SandboxBackendType) (Sandbox, error)` factory function that returns the correct implementation — `internal/sandbox/factory.go`
- [ ] T013 [P2] [US2] Create `internal/sandbox/factory_test.go` — tests for factory with all three backend types and invalid input — `internal/sandbox/factory_test.go`

## Phase 4: Pipeline Integration (US2, US3)

- [ ] T014 [P2] [US2] Add `SandboxBackend string` and `DockerImage string` fields to `AdapterRunConfig` struct — `internal/adapter/adapter.go`
- [ ] T015 [P2] [US2] Update executor `buildAdapterConfig()` to resolve sandbox backend via `RuntimeSandbox.ResolveBackend()` instead of just `Enabled` boolean, populate `SandboxBackend` and `DockerImage` on `AdapterRunConfig` — `internal/pipeline/executor.go`
- [ ] T016 [P2] [US2] Modify `ProcessGroupRunner.Run()` (or adapter layer) to call `sandbox.NewSandbox()` and `Wrap()` when `SandboxBackend` is `docker`, wrapping the adapter subprocess command — `internal/adapter/adapter.go`
- [ ] T017 [P2] [US2] Wire sandbox `Cleanup()` into step completion/failure paths in executor — `internal/pipeline/executor.go`
- [ ] T018 [P2] [US2] Update executor tests to verify sandbox backend resolution and config propagation — `internal/pipeline/executor_test.go`

## Phase 5: Preflight Validation (US3)

- [ ] T019 [P3] [US3] Add `CheckDockerDaemon() Result` method to `preflight.Checker` — checks `docker` on PATH, runs `docker info`, returns platform-specific recovery hints on failure — `internal/preflight/preflight.go`
- [ ] T020 [P3] [US3] Add `CheckBubblewrap() Result` method to `preflight.Checker` — checks `bwrap` on PATH when backend is bubblewrap — `internal/preflight/preflight.go`
- [ ] T021 [P3] [US3] Wire Docker/bubblewrap preflight checks into pipeline startup when the corresponding backend is configured — `internal/pipeline/executor.go`
- [ ] T022 [P] [P3] [US3] Add preflight tests for `CheckDockerDaemon()` (found/not-found, daemon-up/daemon-down) and `CheckBubblewrap()` — `internal/preflight/preflight_test.go`

## Phase 6: Nix Flake Packaging (US1: Install Wave via Nix Flake)

- [ ] T023 [P1] [US1] Compute correct `vendorHash` for `buildGoModule` by running `nix build .#wave`, replace `pkgs.lib.fakeHash` with the computed `sha256-...` hash — `flake.nix`
- [ ] T024 [P1] [US1] Verify `nix build .#wave` succeeds and produces a working binary — `flake.nix`
- [ ] T025 [P1] [US1] Verify `nix flake check` passes — `flake.nix`
- [ ] T026 [P1] [US1] Verify `nix develop` still provides existing dev shell (Go toolchain, linters, bubblewrap on Linux) — `flake.nix`

## Phase 7: Polish & Cross-Cutting Concerns

- [ ] T027 [P] Ensure `go test -race ./...` passes with all new code — project root
- [ ] T028 [P] Ensure backward compatibility: existing `wave.yaml` without `runtime.sandbox.backend` continues working (FR-024) — `internal/manifest/parser_test.go`
- [ ] T029 [P] Ensure `nix develop .#yolo` unsandboxed shell is preserved (FR-025) — `flake.nix`
- [ ] T030 [P] Verify concurrent Docker sandbox step isolation (FR-016) — each step gets its own container with unique container name/ID — `internal/sandbox/docker_test.go`
