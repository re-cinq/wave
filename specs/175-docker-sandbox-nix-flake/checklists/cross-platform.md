# Cross-Platform Requirements Quality: Nix Flake Packaging and Docker-Based Sandbox

**Feature**: `175-docker-sandbox-nix-flake` | **Date**: 2026-03-16

## Platform Parity

- [ ] CHK041 - Are acceptance criteria for "same isolation guarantees" (US4-AS1, US4-AS2) measurably defined? What specific properties must be identical across platforms? [Clarity]
- [ ] CHK042 - Are platform-specific Docker limitations documented as requirements? (e.g., macOS Docker Desktop file sharing performance, WSL2 memory management) [Completeness]
- [ ] CHK043 - Is the behavior specified when `--user UID:GID` produces different results on macOS Docker Desktop (where UID mapping differs from Linux)? [Coverage]
- [ ] CHK044 - Are requirements for Nix flake behavior on non-NixOS Linux distributions specified? (e.g., Ubuntu with Nix installed via Determinate Systems installer) [Coverage]
- [ ] CHK045 - Does the spec define which Docker engine versions and API versions are supported? [Completeness]
- [ ] CHK046 - Are requirements defined for Podman compatibility, or is Docker the only supported container runtime? [Completeness]

## Nix Ecosystem

- [ ] CHK047 - Are requirements defined for the minimum Nix version needed for flake support? [Completeness]
- [ ] CHK048 - Is the flake's `systems` attribute specified? Does it declare supported architectures (x86_64-linux, aarch64-linux, x86_64-darwin, aarch64-darwin)? [Completeness]
- [ ] CHK049 - Are requirements defined for NixOS module integration, or is the flake limited to `packages` and `devShells` outputs? [Coverage]
- [ ] CHK050 - Does FR-005 ("vendored or hashed dependencies") specify whether `gomod2nix` or `buildGoModule`'s built-in vendoring is preferred? [Clarity]
