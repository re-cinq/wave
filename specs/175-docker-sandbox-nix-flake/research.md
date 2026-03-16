# Research: Nix Flake Packaging and Docker-Based Sandbox

**Branch**: `175-docker-sandbox-nix-flake` | **Date**: 2026-03-16

## R1: Nix Flake `buildGoModule` Packaging

### Decision
Use `pkgs.buildGoModule` with `vendorHash` to build the Wave binary as a Nix package.

### Rationale
- `buildGoModule` is the standard Nix builder for Go projects — mature, well-documented, and handles Go module vendoring automatically.
- The existing `flake.nix` already has a `packages.wave` using `buildGoModule` with `vendorHash = pkgs.lib.fakeHash` (placeholder). The real hash must be computed by running `nix build` and capturing the expected hash from the error output.
- `subPackages = [ "cmd/wave" ]` is correctly scoped — only builds the Wave binary, not test binaries.
- `ldflags` inject version info via `main.version`, `main.commit`, `main.date` — matches the existing `cmd/wave/main.go` pattern.

### Alternatives Rejected
- **`buildGoPackage`**: Deprecated in favor of `buildGoModule`. Not suitable.
- **`gomod2nix`**: Adds a separate tool dependency for lock file generation. `buildGoModule` with `vendorHash` is simpler and sufficient.
- **Pre-built binary download**: Would skip reproducible builds, defeating the purpose of Nix.

### Current State
The `flake.nix` already has the package definition but uses `fakeHash`. The fix is mechanical:
1. Run `nix build .#wave 2>&1` to trigger the hash mismatch error
2. Replace `fakeHash` with the computed `sha256-...` hash
3. Verify `nix build .#wave` succeeds
4. Verify `nix flake check` passes

### Risks
- **Go version pinning**: `flake.nix` uses `pkgs.go` which tracks nixpkgs-unstable. If the Go version in nixpkgs diverges from `go.mod`'s `go 1.25.5`, builds may break. Mitigation: pin nixpkgs to a commit that provides Go 1.25+.
- **Vendor hash churn**: Every dependency change requires updating `vendorHash`. This is standard Nix Go workflow — CI should catch stale hashes.

## R2: Docker Sandbox Backend Architecture

### Decision
Implement a `SandboxBackend` interface with three implementations: `DockerSandbox`, `BubblewrapSandbox` (future, wrapping existing), and `NoneSandbox`. The Docker backend wraps the existing adapter subprocess execution with `docker run`.

### Rationale
- Wave adapters (`ClaudeAdapter`, `OpenCodeAdapter`, etc.) execute subprocesses via `os/exec`. The Docker sandbox intercepts this by prepending `docker run ...` to the command execution.
- The cleanest integration point is a new `internal/sandbox/` package that provides a `Sandbox` interface with a `Wrap(cmd *exec.Cmd) *exec.Cmd` method (or equivalent). The adapter's `Run()` method calls the sandbox wrapper before executing.
- Alternative: wrap at the `AdapterRunner` level with a decorator pattern. This is cleaner because it doesn't require modifying each adapter individually.

### Architecture

```
AdapterRunner.Run(ctx, cfg)
  → SandboxRunner.Run(ctx, cfg)      // decorator wrapping the real adapter
    → builds docker run command
    → exec.CommandContext("docker", "run", ..., originalBinary, originalArgs...)
    → returns AdapterResult
```

### Key Design Decisions

1. **Decorator pattern over interface modification**: Wrap `AdapterRunner` rather than adding sandbox logic inside each adapter. This keeps adapters focused on their protocol (Claude NDJSON, OpenCode, etc.) while sandbox is orthogonal.

2. **Docker command construction**: Build `docker run` args from `AdapterRunConfig`:
   - `--rm` — ephemeral containers
   - `--read-only` — read-only root (FR-008)
   - `--tmpfs /tmp:rw,nosuid,nodev` — writable tmp (FR-008)
   - `--tmpfs /var/run:rw,nosuid,nodev` — writable var/run (FR-008)
   - `--tmpfs /home/wave:rw,nosuid,nodev` — isolated HOME (FR-009)
   - `-e HOME=/home/wave` — explicit HOME (FR-009)
   - `-e` for each passthrough var (FR-010)
   - `-v workspace:workspace:rw` — workspace bind mount (FR-012)
   - `-v .wave/artifacts:.wave/artifacts:ro` — input artifacts (FR-013)
   - `-v .wave/output:.wave/output:rw` — output artifacts (FR-013)
   - `--cap-drop=ALL --security-opt=no-new-privileges` — capabilities (FR-014)
   - `--network=none` or custom network (FR-015)
   - `--user $(id -u):$(id -g)` — UID/GID mapping (FR-016a)

3. **Network domain filtering**: Use proxy sidecar (FR-016b):
   - When `allowed_domains` is non-empty, create a Docker network + proxy container
   - Step container gets `--network=<custom>` + `HTTP_PROXY`/`HTTPS_PROXY` env vars
   - Proxy enforces domain allowlisting
   - When `allowed_domains` is empty, use `--network=none`

4. **Host binary bind-mounting**: Mount the adapter binary (e.g., `claude`) and its runtime dependencies (Node.js for Claude Code) from the host into the container. This avoids building custom Docker images.

### Alternatives Rejected
- **Custom Docker image per adapter**: High maintenance burden. Bind-mounting host binaries is simpler and mirrors the bubblewrap approach.
- **iptables-based network filtering**: Requires `CAP_NET_ADMIN`, contradicts FR-014. Doesn't work on Docker Desktop (macOS/WSL2).
- **DNS-based filtering**: Bypassable via IP addresses. Not reliable for security.
- **gVisor/Kata containers**: Over-engineered for this use case. Standard Docker containers with capability dropping provide sufficient isolation.

### Risks
- **Host binary compatibility**: Bind-mounted binaries may have glibc/musl mismatches between host and container. Mitigation: use `ubuntu:24.04` as base image (matches most host environments), document that the container base image must be compatible with host binaries.
- **Docker Desktop overhead**: Docker Desktop on macOS uses a Linux VM, adding latency. This is inherent to Docker on macOS and acceptable.
- **Proxy sidecar complexity**: The network filtering proxy adds operational complexity. Mitigation: start with `--network=none` for the initial implementation, add proxy sidecar as a follow-up.

## R3: Configuration Schema Design

### Decision
Add `Backend` and `DockerImage` fields to `RuntimeSandbox`. `Backend` supersedes the existing `Enabled` boolean.

### Rationale
- The existing `RuntimeSandbox` struct has `Enabled bool`, `DefaultAllowedDomains []string`, `EnvPassthrough []string`.
- Adding `Backend string` (values: `docker`, `bubblewrap`, `none`) provides a clean enum for sandbox selection.
- The `Enabled` field is retained for backward compatibility — `Enabled: true` without `Backend` implies `bubblewrap` (current behavior).
- `DockerImage string` defaults to `ubuntu:24.04`.

### Migration Logic
```
if Backend != "" {
    // New behavior: backend takes precedence
    sandboxEnabled = (Backend != "none")
} else if Enabled {
    // Legacy behavior: enabled implies bubblewrap
    Backend = "bubblewrap"
    sandboxEnabled = true
} else {
    // Default: no sandbox
    sandboxEnabled = false
}
```

## R4: Preflight Validation Extension

### Decision
Extend `preflight.Checker` to validate Docker daemon availability when `runtime.sandbox.backend: docker` is configured.

### Rationale
- The current `Checker` validates tools and skills. Docker is a tool dependency when the Docker backend is selected.
- Validation: run `docker info` (not just `docker --version`) to verify the daemon is running, not just the CLI installed.
- On failure, provide actionable recovery hints per platform:
  - Linux: `systemctl start docker`
  - macOS: "Start Docker Desktop"
  - WSL2: "Start Docker Desktop for Windows"

### Implementation
Add a `CheckDockerDaemon() Result` method to `Checker` that:
1. Checks `docker` is on PATH
2. Runs `docker info` to verify daemon connectivity
3. Returns a `Result` with platform-specific recovery hints on failure

## R5: Proxy Sidecar for Network Filtering

### Decision
Defer proxy sidecar to a follow-up. Initial implementation uses `--network=none` for all Docker sandbox steps, with `allowed_domains` configuration parsed but not enforced via Docker (only via Claude Code's `settings.json` network settings).

### Rationale
- The proxy sidecar is the most complex component and can be delivered independently.
- `--network=none` provides strong isolation for steps that don't need network access.
- Claude Code's built-in `settings.json` sandbox network settings already provide some domain filtering for steps that need network access (the adapter binary respects these settings).
- For the Docker backend, steps needing network access (e.g., `WebSearch`, `WebFetch`) can run with `--network=host` initially, with the understanding that domain filtering is advisory (via settings.json) not enforced (via proxy).

### Follow-up Scope
- Go-based forward proxy (lightweight, no external dependency)
- Docker Compose-style network creation per step
- `CONNECT` method handling for HTTPS tunneling
- Proxy health check before step execution
