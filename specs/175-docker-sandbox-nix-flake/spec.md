# Feature Specification: Nix Flake Packaging and Docker-Based Sandbox

**Feature Branch**: `175-docker-sandbox-nix-flake`
**Created**: 2026-03-16
**Status**: Draft
**Input**: https://github.com/re-cinq/wave/issues/175

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Install Wave via Nix Flake (Priority: P1)

A user wants to install and run Wave without cloning the repository. They use Nix flakes to consume Wave as a dependency or run it directly from the GitHub repository.

**Why this priority**: Nix flake packaging is the foundation for distribution. Without it, Wave cannot be consumed as a standard Nix package, limiting adoption for the Nix ecosystem. This is a prerequisite — it unblocks all other packaging and distribution improvements.

**Independent Test**: Can be fully tested by running `nix run github:re-cinq/wave -- --version` from any machine with Nix installed. Delivers a working Wave binary without requiring manual build steps.

**Acceptance Scenarios**:

1. **Given** a machine with Nix (flakes enabled), **When** the user runs `nix run github:re-cinq/wave -- --version`, **Then** Wave prints its current version and exits successfully.
2. **Given** a Nix flake project that lists Wave as a flake input, **When** the user runs `nix build`, **Then** the Wave binary is available in the output derivation.
3. **Given** a contributor cloning the Wave repo, **When** they run `nix develop`, **Then** the existing development shell continues to function identically (Go toolchain, bubblewrap sandbox on Linux, linters, etc.).
4. **Given** the Wave repository, **When** a user runs `nix flake check`, **Then** all checks pass (builds succeed, basic tests run).

---

### User Story 2 - Run Pipeline Steps in Docker Sandbox (Priority: P2)

An operator wants to execute Wave pipeline steps inside Docker containers to achieve portable isolation across Linux, macOS, and Windows WSL2 — without depending on NixOS-specific bubblewrap.

**Why this priority**: Docker sandbox is the core portability feature. The current bubblewrap sandbox only works on Linux with unprivileged user namespaces, excluding macOS and Windows users. Docker provides equivalent isolation guarantees on all major platforms.

**Independent Test**: Can be fully tested by configuring `runtime.sandbox.backend: docker` in `wave.yaml` and running a pipeline. The step executes inside a Docker container with read-only root filesystem, isolated HOME, curated environment, and domain allowlisting.

**Acceptance Scenarios**:

1. **Given** a `wave.yaml` with `runtime.sandbox.backend: docker`, **When** `wave run <pipeline> -- "<task>"` executes a step, **Then** the adapter spawns a Docker container with read-only root filesystem, tmpfs at `/tmp`, and an isolated `$HOME`.
2. **Given** a manifest with `runtime.sandbox.env_passthrough: [ANTHROPIC_API_KEY, GH_TOKEN]`, **When** a step runs in the Docker sandbox, **Then** only those environment variables are available inside the container, plus standard system variables (`HOME`, `PATH`, `TERM`).
3. **Given** a persona with `sandbox.allowed_domains: [api.anthropic.com, github.com]`, **When** the step attempts to access `evil.com`, **Then** the request is blocked; requests to `api.anthropic.com` succeed.
4. **Given** a step with `workspace.type: worktree`, **When** the step runs in Docker sandbox, **Then** the worktree is bind-mounted into the container at the configured workspace path, with the correct read/write mode.

---

### User Story 3 - Configure Sandbox Backend (Priority: P3)

An administrator wants to choose between Docker and bubblewrap sandbox backends, or disable sandboxing entirely, through manifest configuration.

**Why this priority**: Provides flexibility for different deployment environments. NixOS users keep their existing bubblewrap setup. Docker users get portable isolation. Development/testing environments can disable sandboxing.

**Independent Test**: Can be tested by setting `runtime.sandbox.backend` to `docker`, `bubblewrap`, or `none` in `wave.yaml` and verifying that the correct isolation mechanism activates (or none).

**Acceptance Scenarios**:

1. **Given** `runtime.sandbox.backend: docker` in `wave.yaml`, **When** Wave starts a pipeline, **Then** it uses Docker containers for step isolation.
2. **Given** `runtime.sandbox.backend: bubblewrap` in `wave.yaml`, **When** Wave starts a pipeline on Linux, **Then** it uses the existing bubblewrap-based sandbox (current behavior preserved).
3. **Given** `runtime.sandbox.backend: docker` but Docker daemon is not running, **When** Wave attempts to start a pipeline, **Then** preflight validation fails with a clear error message explaining Docker is required and how to start it.
4. **Given** no `runtime.sandbox.backend` configured (default), **When** Wave starts a pipeline, **Then** it uses the previous default behavior (sandbox disabled unless explicitly enabled).

---

### User Story 4 - Cross-Platform Pipeline Execution (Priority: P4)

A developer on macOS or Windows WSL2 wants to run Wave pipelines with the same isolation guarantees as Linux users, using Docker as the sandbox backend.

**Why this priority**: Expands Wave's usable platform surface from Linux-only (with isolation) to all major platforms. Without this, non-Linux users must run pipelines without any sandbox.

**Independent Test**: Can be tested by running `wave run <pipeline> -- "<task>"` on macOS with Docker Desktop and on Windows WSL2 with Docker, verifying identical isolation behavior.

**Acceptance Scenarios**:

1. **Given** macOS with Docker Desktop, **When** `wave run` executes a step with Docker sandbox, **Then** the step runs in a container with the same isolation guarantees as on Linux.
2. **Given** Windows WSL2 with Docker, **When** `wave run` executes a step with Docker sandbox, **Then** the step runs in a container with the same isolation guarantees as on Linux.
3. **Given** a platform where Docker is available but bubblewrap is not, **When** `runtime.sandbox.backend: bubblewrap` is configured, **Then** preflight validation fails with a message suggesting Docker as an alternative.

---

### Edge Cases

- What happens when Docker daemon is not running? Preflight check MUST fail with actionable error before any step executes.
- What happens when the Docker base image is not available locally? Wave MUST pull it automatically or provide a clear error with pull instructions.
- What happens when the Docker container runs out of memory/disk? The adapter MUST capture the OOM/disk-full exit code and report it as a step failure with a recovery hint.
- What happens when network domain filtering is configured but the proxy/DNS approach fails? The step MUST fail rather than silently running without network restrictions.
- What happens when concurrent pipeline steps share the same Docker network? Each container MUST have its own isolated network namespace — no cross-step communication.
- What happens when the user's UID/GID doesn't match the container's? Workspace bind mounts MUST use correct UID/GID mapping to prevent permission errors.
- What happens when `nix run` is used with a version of Nix that doesn't support flakes? The flake MUST fail with a clear error, not silently degrade.
- What happens when `nix develop` is run on macOS where bubblewrap is unavailable? The dev shell MUST still function without bubblewrap (current Darwin behavior preserved).

## Requirements _(mandatory)_

### Functional Requirements

#### Nix Flake Packaging

- **FR-001**: Repository MUST contain a `flake.nix` at the root that exposes `packages.default` with the Wave binary built from source.
- **FR-002**: `nix run github:re-cinq/wave -- --version` MUST produce a working binary that outputs the current version.
- **FR-003**: `nix develop` MUST continue to provide the existing development shell with Go toolchain, linters, and bubblewrap sandbox (on Linux).
- **FR-004**: `nix flake check` MUST pass, including build validation.
- **FR-005**: The flake MUST use `buildGoModule` with vendored or hashed dependencies for reproducibility.
- **FR-006**: The flake MUST declare `flake.lock` for pinning nixpkgs and any other inputs.

#### Docker Sandbox Backend

- **FR-007**: System MUST implement a Docker-based sandbox backend that executes adapter subprocesses inside containers.
- **FR-008**: Docker sandbox MUST enforce read-only root filesystem with writable tmpfs mounts at `/tmp` and `/var/run`.
- **FR-009**: Docker sandbox MUST provide an isolated `$HOME` directory (tmpfs or empty bind mount) — the host HOME MUST NOT be accessible.
- **FR-010**: Docker sandbox MUST pass through only environment variables listed in `runtime.sandbox.env_passthrough`, plus standard system variables (`HOME`, `PATH`, `TERM`, `TMPDIR`).
- **FR-011**: Docker sandbox MUST enforce network domain allowlisting as configured in persona or runtime sandbox settings.
- **FR-012**: Docker sandbox MUST bind-mount the step workspace directory with the correct read/write mode.
- **FR-013**: Docker sandbox MUST bind-mount artifact directories (`.wave/artifacts/`, `.wave/output/`) for input/output.
- **FR-014**: Docker sandbox MUST drop all Linux capabilities (`CAP_DROP=ALL`) and set `no-new-privileges`.
- **FR-015**: Docker sandbox MUST use `--network=none` when no allowed domains are configured, and a filtered network when domains are specified.
- **FR-016**: Docker sandbox MUST support concurrent step execution — each step gets its own container with no shared state.
- **FR-016a**: Docker containers MUST run as the host user's UID/GID (`--user $(id -u):$(id -g)`) to ensure correct permissions on bind-mounted workspace directories (see Clarification C4).
- **FR-016b**: Docker sandbox MUST enforce network domain allowlisting via an HTTP/HTTPS proxy sidecar when `allowed_domains` are configured. Non-HTTP traffic is blocked. Proxy failure MUST cause the step to fail (see Clarification C3).

#### Configuration

- **FR-017**: `wave.yaml` MUST support `runtime.sandbox.backend` field with values `docker`, `bubblewrap`, or `none`.
- **FR-017a**: `wave.yaml` MUST support `runtime.sandbox.docker_image` field (string, optional) to specify the base Docker image. Default: `ubuntu:24.04`.
- **FR-018**: When `runtime.sandbox.backend` is not specified, system MUST default to the current behavior (`none` unless `runtime.sandbox.enabled: true` with implicit bubblewrap). When `backend` is set, it takes precedence over the `enabled` boolean (see Clarification C1).
- **FR-019**: Per-persona `sandbox.allowed_domains` MUST override `runtime.sandbox.default_allowed_domains` when using the Docker backend.

#### Preflight Validation

- **FR-020**: System MUST validate Docker daemon availability during preflight when `runtime.sandbox.backend: docker` is configured.
- **FR-021**: System MUST validate bubblewrap binary availability during preflight when `runtime.sandbox.backend: bubblewrap` is configured.
- **FR-022**: Preflight failures MUST produce actionable error messages with recovery hints (e.g., "Docker daemon not running. Start it with: systemctl start docker").

#### Backward Compatibility

- **FR-023**: The existing bubblewrap sandbox (via `flake.nix` dev shell) MUST continue to function unchanged for NixOS users.
- **FR-024**: Existing `wave.yaml` files without `runtime.sandbox.backend` MUST continue to work without modification.
- **FR-025**: The `nix develop .#yolo` unsandboxed shell MUST be preserved.

### Key Entities

- **SandboxBackend**: Represents a sandbox execution strategy (Docker, bubblewrap, none). Determines how adapter subprocesses are isolated.
- **DockerContainer**: An ephemeral container created per pipeline step. Attributes: image, mounts, environment, network config, capabilities, lifecycle (create → start → wait → remove).
- **SandboxConfig**: Merged configuration from manifest runtime settings and persona-level overrides. Attributes: backend type, allowed domains, env passthrough list, image reference.
- **PreflightCheck**: A validation that runs before pipeline execution to verify sandbox backend availability.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: `nix run github:re-cinq/wave -- --version` succeeds and outputs a valid semver version string.
- **SC-002**: `nix develop` produces a shell with `go`, `golangci-lint`, and `wave` available on `$PATH`.
- **SC-003**: `nix flake check` passes with zero errors.
- **SC-004**: A pipeline step configured with `runtime.sandbox.backend: docker` executes inside a Docker container — verifiable by the absence of host filesystem access outside mounted paths.
- **SC-005**: Docker sandbox enforces read-only root, isolated HOME, and env passthrough — verifiable by running `touch /test`, `ls $HOME`, and `env` inside the container.
- **SC-006**: Docker sandbox blocks network access to non-allowlisted domains — verifiable by attempting a connection to a blocked domain from within the container.
- **SC-007**: Existing `wave.yaml` configurations without `runtime.sandbox.backend` continue to work without modification — verified by running the full test suite (`go test ./...`).
- **SC-008**: Docker sandbox works on Linux (native Docker), macOS (Docker Desktop), and Windows WSL2 (Docker Desktop) — verified by manual testing on each platform.
- **SC-009**: Preflight validation catches missing Docker daemon and reports an actionable error — verified by stopping Docker and running `wave run`.
- **SC-010**: Concurrent pipeline steps each get isolated containers with no shared state — verified by running 3+ concurrent steps and confirming independent container IDs.

## Clarifications

### C1: Relationship between `runtime.sandbox.backend` and existing `runtime.sandbox.enabled`

**Ambiguity**: The spec introduces `runtime.sandbox.backend: docker|bubblewrap|none` but the codebase already has `runtime.sandbox.enabled` (boolean) in `RuntimeSandbox`. How do these fields interact?

**Resolution**: The `backend` field supersedes the `enabled` boolean as the primary sandbox control. When `backend` is set to `docker` or `bubblewrap`, sandboxing is implicitly enabled. When `backend` is `none` or omitted, sandboxing is disabled (matching current default behavior). The `enabled` field is retained for backward compatibility: existing manifests with `enabled: true` (without `backend`) continue to use bubblewrap on Linux, equivalent to `backend: bubblewrap`. If both `backend` and `enabled` are set, `backend` takes precedence.

**Rationale**: This preserves backward compatibility (FR-024) while providing a clean migration path. Existing `wave.yaml` files with `runtime.sandbox.enabled: true` continue working unchanged.

### C2: Docker base image specification

**Ambiguity**: The spec requires Docker sandbox execution (FR-007) but does not specify what Docker image is used or how it's configured.

**Resolution**: Add a `runtime.sandbox.docker_image` configuration field (string, optional). When omitted, Wave defaults to a minimal image: `ubuntu:24.04`. The image must have the adapter binary (e.g., `claude`, `node`) available — Wave bind-mounts the host's adapter binary and required tools (from `$PATH`) into the container at runtime, similar to how bubblewrap uses `--ro-bind / /`. This avoids requiring a custom Wave Docker image. The image primarily provides a root filesystem and libc.

**Rationale**: Bind-mounting host binaries mirrors the bubblewrap approach (`--ro-bind / /`) and avoids maintaining a separate Docker image. A configurable image field allows operators to use custom images with additional dependencies if needed.

### C3: Network domain filtering mechanism in Docker

**Ambiguity**: FR-011 and FR-015 require domain allowlisting in Docker but don't specify the enforcement mechanism.

**Resolution**: Use Docker's `--network=none` for steps with no allowed domains (FR-015). For steps with allowed domains, use a lightweight HTTP/HTTPS proxy (e.g., `squid` or a simple Go-based forward proxy) running as a sidecar container on a dedicated Docker network. The proxy enforces domain allowlisting via `CONNECT` tunneling for HTTPS and `Host` header matching for HTTP. The step container's `HTTP_PROXY`/`HTTPS_PROXY` environment variables are set to point to the proxy. Non-HTTP traffic is blocked by default (no direct network access).

**Rationale**: Proxy-based filtering is the most portable approach across Linux, macOS Docker Desktop, and WSL2. iptables-based approaches require `CAP_NET_ADMIN` (contradicts FR-014's `CAP_DROP=ALL`) and don't work on Docker Desktop's Linux VM. DNS-based filtering can be bypassed via IP addresses. A proxy provides application-layer filtering that works consistently across platforms. If proxy setup fails, the step MUST fail rather than running without restrictions (per edge case #4).

### C4: UID/GID mapping for workspace bind mounts

**Ambiguity**: Edge case #6 mentions UID/GID mapping but no functional requirement specifies the strategy.

**Resolution**: Docker containers run as the host user's UID/GID via `--user $(id -u):$(id -g)`. This ensures bind-mounted workspace directories have correct permissions without requiring `userns-remap` or `chown` inside the container. The container does not need a matching `/etc/passwd` entry — Wave sets `HOME` explicitly via `--env HOME=/home/wave` with a tmpfs mount at that path.

**Rationale**: Running as host UID/GID is the simplest approach that works across Linux (native Docker), macOS (Docker Desktop with gRPC FUSE), and WSL2. It avoids the complexity of user namespace remapping and matches how most CI/CD systems handle bind mounts.

### C5: Relationship between Nix flake package and Docker sandbox

**Ambiguity**: The spec covers both Nix flake packaging (FR-001–FR-006) and Docker sandbox (FR-007–FR-016) but doesn't clarify whether the Docker sandbox uses the Nix-built binary.

**Resolution**: The Nix flake package output and Docker sandbox are independent features. The Docker sandbox executes whatever `wave` binary the user is running — it does not require or use the Nix-built binary. The flake's `packages.default` produces a standalone Wave binary for distribution. The Docker sandbox is a runtime isolation mechanism that wraps adapter subprocess execution regardless of how Wave itself was installed (Nix, `go install`, pre-built release binary). The flake's dev shell continues to provide the bubblewrap sandbox for development.

**Rationale**: Keeping these features orthogonal follows the single-responsibility principle. Users who install via `go install` or download a release binary should still be able to use Docker sandboxing. The Nix flake is a packaging/distribution concern; Docker sandbox is a runtime isolation concern.
