# Implementation Plan: Cross-Platform Installation

## Objective

Configure GoReleaser-based cross-compilation and packaging, a GitHub Actions release workflow, Homebrew tap support, AUR packaging, enhanced Nix flake output, and a universal install script so that Wave can be installed on Linux (Debian/Ubuntu, Arch, Nix) and macOS (Homebrew, direct binary) from versioned GitHub Releases.

## Approach

The implementation follows a layered strategy:

1. **Version injection** — Extend `cmd/wave/main.go` to expose `version`, `commit`, and `date` via `-ldflags`, and update the Makefile to pass them during `go build`.
2. **GoReleaser** — Add `.goreleaser.yaml` at the project root. Configure cross-compilation (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64), packaging (tar.gz, zip, deb), and checksum generation (SHA256).
3. **GitHub Actions release workflow** — Add `.github/workflows/release.yml` triggered on `v*` tag pushes. Uses `goreleaser/goreleaser-action` to build, package, and upload artifacts to GitHub Releases.
4. **Homebrew tap** — Add a GoReleaser `brews` section that auto-publishes a formula to `re-cinq/homebrew-tap` on release.
5. **AUR package** — Add a `PKGBUILD` template under `packaging/aur/` that sources the release tarball and builds from source, plus a GoReleaser `aurs` section if goreleaser-pro is available, otherwise the PKGBUILD is maintained manually.
6. **Nix flake package output** — Extend `flake.nix` to add `packages.${system}.wave` (a `buildGoModule` derivation) alongside the existing `devShells`, so `nix profile install` works.
7. **Install script** — Add `scripts/install.sh` that detects OS/arch, downloads the correct binary from the latest GitHub Release, verifies the SHA256 checksum, and installs to `/usr/local/bin` (or `~/.local/bin`).

## File Mapping

| File | Action | Purpose |
|------|--------|---------|
| `cmd/wave/main.go` | modify | Add `commit` and `date` build variables; display in `--version` |
| `Makefile` | modify | Add `VERSION`, `COMMIT`, `DATE` ldflags to build target |
| `.goreleaser.yaml` | create | GoReleaser configuration (builds, archives, nfpms, brews, checksum) |
| `.github/workflows/release.yml` | create | GitHub Actions workflow triggered on `v*` tags |
| `flake.nix` | modify | Add `packages.${system}.wave` and `packages.${system}.default` outputs |
| `scripts/install.sh` | create | Cross-platform install script (curl \| sh) |
| `packaging/aur/PKGBUILD` | create | Arch Linux AUR package definition |

## Architecture Decisions

1. **GoReleaser over manual cross-compilation** — GoReleaser is the de facto standard for Go binary distribution. It handles cross-compilation, packaging, checksums, and changelogs in a single declarative config. The free (OSS) edition covers all needed features except AUR automation.

2. **ldflags version injection** — Using `-ldflags -X main.version=...` is the standard Go pattern for build-time version injection. No runtime dependencies or file reads needed.

3. **Homebrew tap (not core)** — A dedicated tap (`re-cinq/homebrew-tap`) is appropriate for a project that is not yet in Homebrew core. GoReleaser can auto-update the formula on release.

4. **Manual AUR PKGBUILD** — GoReleaser's AUR publisher requires GoReleaser Pro. A manually maintained PKGBUILD that pulls the release tarball and builds via `go build` is the pragmatic choice.

5. **Nix buildGoModule** — The flake already exists with `devShells`. Adding a `packages` output using `buildGoModule` is the idiomatic Nix approach and enables `nix profile install github:re-cinq/wave`.

6. **Install script with checksum verification** — The script downloads from GitHub Releases API, verifies SHA256, and installs. This provides a fallback for platforms without a package manager integration.

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| GoReleaser config errors on first release | Builds fail, no artifacts | Test with `goreleaser release --snapshot --clean` locally |
| Homebrew tap repo doesn't exist yet | Formula publish fails | Create `re-cinq/homebrew-tap` repo before first release, or disable brews section initially |
| AUR PKGBUILD vendored hash mismatch | AUR build fails | Document the manual hash update process; consider a script to automate |
| Nix vendorHash changes with dependencies | Nix build fails | Use `lib.fakeHash` during dev, update on release |
| CGO_ENABLED=0 breaks SQLite | Runtime crash | Wave uses `modernc.org/sqlite` (pure Go) — no CGO needed. Verify in CI. |
| Install script edge cases (old curl, busybox) | Script fails on minimal systems | Test on Alpine, Ubuntu minimal; provide clear error messages |

## Testing Strategy

1. **GoReleaser snapshot build** — Run `goreleaser release --snapshot --clean` in CI (non-tag pushes) to catch config errors early.
2. **Version output test** — Unit test that `wave --version` output contains version, commit, and date when built with ldflags.
3. **Install script testing** — Test the install script in a Docker container (Ubuntu, Alpine) to verify detection, download, checksum verification, and installation.
4. **Nix build** — Run `nix build .#wave` to verify the flake package output builds correctly.
5. **Makefile integration** — Verify `make build && ./wave --version` shows injected version info.
