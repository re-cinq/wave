# Requirements Quality Review: Nix Flake Packaging and Docker-Based Sandbox

**Feature**: `175-docker-sandbox-nix-flake` | **Date**: 2026-03-16

## Completeness

- [ ] CHK001 - Are resource limits (CPU, memory, disk) specified for Docker containers, or is unbounded resource usage acceptable? [Completeness]
- [ ] CHK002 - Is the Docker container lifecycle (create → start → wait → remove) fully specified for error paths (e.g., container start fails, wait times out)? [Completeness]
- [ ] CHK003 - Are timeout requirements defined for Docker operations (pull, create, start)? What happens if Docker hangs? [Completeness]
- [ ] CHK004 - Is the behavior specified when `runtime.sandbox.docker_image` references a non-existent or invalid image? [Completeness]
- [ ] CHK005 - Are logging and observability requirements defined for Docker sandbox execution (container ID, exit codes, stderr capture)? [Completeness]
- [ ] CHK006 - Is the `flake.lock` update strategy specified? Who or what updates the lock file when nixpkgs moves? [Completeness]
- [ ] CHK007 - Are requirements defined for what happens when the Docker socket path differs from the default (`/var/run/docker.sock`)? [Completeness]
- [ ] CHK008 - Is the proxy sidecar container lifecycle defined — when is it created, shared across steps, or per-step? [Completeness]

## Clarity

- [ ] CHK009 - Does FR-011 ("enforce network domain allowlisting") clearly define what "enforce" means — block vs. log-and-allow vs. fail-step? [Clarity]
- [ ] CHK010 - Does FR-008 ("read-only root filesystem") clearly specify which paths are writable (tmpfs at `/tmp`, `/var/run`, `/home/wave`) and whether the list is exhaustive? [Clarity]
- [ ] CHK011 - Does FR-010 specify the exact list of "standard system variables" beyond `HOME`, `PATH`, `TERM`, `TMPDIR`? Are there platform-specific additions? [Clarity]
- [ ] CHK012 - Does FR-018 unambiguously define the precedence when both `enabled: true` and `backend: none` are set? [Clarity]
- [ ] CHK013 - Does C2 ("bind-mounts the host's adapter binary") specify which host binaries are mounted and how they are discovered? [Clarity]
- [ ] CHK014 - Does FR-015 distinguish between "no allowed domains configured" and "empty allowed domains list"? Are these semantically different? [Clarity]

## Consistency

- [ ] CHK015 - Is the default backend behavior consistent across FR-018 (default is `none`) and US3-AS4 (default is "previous behavior")? Are these the same? [Consistency]
- [ ] CHK016 - Are FR-017a (`docker_image`) and C2 (bind-mount host binaries into base image) consistent? If binaries are mounted, does the image choice matter beyond providing libc? [Consistency]
- [ ] CHK017 - Is the network isolation model consistent between FR-015 (`--network=none`) and FR-016b (HTTP proxy sidecar)? Are these two separate modes clearly delineated? [Consistency]
- [ ] CHK018 - Are Edge Case #4 (proxy failure must fail step) and C3 (proxy enforcement) expressed identically in the requirements section, or only in clarifications? [Consistency]
- [ ] CHK019 - Is FR-003 (nix develop preserves existing shell) consistent with any flake.nix structural changes needed for FR-001 (packages.default)? [Consistency]
- [ ] CHK020 - Do success criteria SC-004 through SC-006 test the same requirements as FR-007 through FR-015, or are there gaps? [Consistency]

## Coverage

- [ ] CHK021 - Are non-HTTP/HTTPS protocols (SSH, git://) addressed? FR-016b blocks non-HTTP traffic — is this acceptable for git operations inside containers? [Coverage]
- [ ] CHK022 - Are signal forwarding requirements specified? How does SIGTERM/SIGINT propagate from Wave to the Docker container and the adapter process inside it? [Coverage]
- [ ] CHK023 - Are volume cleanup requirements specified? What happens to tmpfs data and any residual container state after step completion or failure? [Coverage]
- [ ] CHK024 - Are DNS resolution requirements specified for the Docker container? Does `--network=none` prevent DNS, and how does the proxy sidecar handle DNS for allowed domains? [Coverage]
- [ ] CHK025 - Are requirements defined for Docker Desktop-specific behaviors (gRPC FUSE file sharing performance, VirtioFS mode, resource limits)? [Coverage]
- [ ] CHK026 - Is the interaction between Docker sandbox and the existing bubblewrap dev shell sandbox specified? Can they coexist? What if both are configured? [Coverage]
- [ ] CHK027 - Are adapter binary compatibility requirements specified? What if the host binary is dynamically linked against a different libc than the container provides? [Coverage]
- [ ] CHK028 - Is the behavior for `nix run` on systems without flakes enabled specified beyond "clear error"? Should the flake provide a compatibility shim? [Coverage]
