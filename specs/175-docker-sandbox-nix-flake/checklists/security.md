# Security Requirements Quality: Nix Flake Packaging and Docker-Based Sandbox

**Feature**: `175-docker-sandbox-nix-flake` | **Date**: 2026-03-16

## Container Isolation

- [ ] CHK029 - Are seccomp profile requirements defined for Docker containers, or does `CAP_DROP=ALL` alone provide sufficient system call restriction? [Completeness]
- [ ] CHK030 - Is `--security-opt=no-new-privileges` explicitly listed as a functional requirement, or only implied by FR-014? [Clarity]
- [ ] CHK031 - Are requirements defined for preventing container escape via mounted volumes (e.g., mounting Docker socket, `/proc`, `/sys`)? [Coverage]
- [ ] CHK032 - Is the proxy sidecar's own security posture specified? Can a compromised step container attack the proxy? [Coverage]
- [ ] CHK033 - Are requirements defined for preventing the step from discovering or connecting to other containers on the Docker host? [Coverage]
- [ ] CHK034 - Does FR-010 (env passthrough) specify validation of environment variable names to prevent injection (e.g., `LD_PRELOAD`, `LD_LIBRARY_PATH`)? [Completeness]
- [ ] CHK035 - Are requirements defined for Docker image provenance? Should Wave verify image signatures or restrict to known registries? [Completeness]
- [ ] CHK036 - Is the threat model for the proxy sidecar specified? Can the step bypass the proxy by using raw IP addresses or non-standard ports? [Coverage]
- [ ] CHK037 - Are audit/logging requirements defined for sandbox security events (blocked network requests, capability denials, mount violations)? [Completeness]
- [ ] CHK038 - Does the spec address the risk of `--user UID:GID` allowing the container process to access other host files readable by that UID? [Coverage]

## Credential Handling

- [ ] CHK039 - Are requirements defined for how credentials in passthrough env vars are protected from container introspection (`/proc/self/environ`)? [Coverage]
- [ ] CHK040 - Does the spec address credential leakage through Docker's inspect command (`docker inspect` shows env vars)? [Coverage]
