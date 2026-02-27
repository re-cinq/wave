# docs: update README install/quickstart and contributor guidance for public repo

**Issue**: [#174](https://github.com/re-cinq/wave/issues/174)
**Feature Branch**: `174-readme-public-docs`
**Labels**: documentation, enhancement, good first issue
**Author**: nextlevelshit
**Status**: Open

## Context

Now that Wave is open-sourced, the README needs to be updated so external contributors and users can install, build, and get started without internal knowledge.

## Current State

The README currently references internal workflows and lacks public-facing install and quickstart instructions. Specifically:

- The installation docs (`docs/guide/installation.md`) still contain a `::: warning Private Repository` block advising users to build from source "while the repository is private"
- The README Quick Start references `./install.sh` but doesn't mention `go install`, GitHub Releases, or the install script at `scripts/install.sh`
- No `CONTRIBUTING.md` file exists in the repository
- No `LICENSE` file exists in the repository root (despite the README badge linking to one and the `.goreleaser.yaml` referencing Apache-2.0)
- The `.goreleaser.yaml` lists the license as "Apache-2.0" but the README badge says "MIT" — these are inconsistent

## Tasks

- [ ] Update **Installation** section with public install instructions (e.g. `go install`, binary releases, Nix flake)
- [ ] Add or revise **Quickstart** guide: first pipeline run from clone to output
- [ ] Remove or replace any internal-only references (private registries, internal URLs, team-specific tooling)
- [ ] Verify **Build from source** instructions work on a clean checkout
- [ ] Add **Contributing** section or link to CONTRIBUTING.md if it exists
- [ ] Confirm license badge and LICENSE file are present and correct
- [ ] Review for any leaked internal context (team names, private repos, credentials)

## Acceptance Criteria

- A new user can clone the repo and run their first pipeline by following README instructions alone
- No internal-only references remain in the README
- Install section covers at least one binary distribution method and build-from-source
- Contributing guidance is present or linked

## Codebase Observations

### Installation Methods Available

1. **Install script** (`install.sh` / `scripts/install.sh`): Downloads the correct binary from GitHub Releases with SHA256 checksum verification. Supports Linux (x86_64, ARM64) and macOS (Intel, Apple Silicon).
2. **Build from source**: `make build` compiles the binary. `make install` installs to `~/.local/bin` by default.
3. **GoReleaser + GitHub Actions**: On every merge to `main`, a tag is auto-created and GoReleaser produces `.tar.gz` (Linux), `.zip` (macOS), and `.deb` packages with checksums. Homebrew tap is configured but `skip_upload: true`.
4. **Nix flake**: `flake.nix` exists for sandboxed development environments.

### License Inconsistency

- README badge: MIT
- `.goreleaser.yaml` nfpm section: Apache-2.0
- No actual LICENSE file exists in the repository root

### Missing Files

- No `CONTRIBUTING.md`
- No `LICENSE` file
