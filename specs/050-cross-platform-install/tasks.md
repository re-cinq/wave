# Tasks

## Phase 1: Version Infrastructure

- [X] Task 1.1: Add commit and date build variables to `cmd/wave/main.go` and format `--version` output to show version, commit, and build date
- [X] Task 1.2: Update `Makefile` build target to inject version, commit, and date via `-ldflags`

## Phase 2: GoReleaser Configuration

- [X] Task 2.1: Create `.goreleaser.yaml` with builds (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64), archives (tar.gz for linux, zip for darwin), and checksums
- [X] Task 2.2: Add nfpms section to `.goreleaser.yaml` for `.deb` package generation
- [X] Task 2.3: Add brews section to `.goreleaser.yaml` for Homebrew tap auto-publish to `re-cinq/homebrew-tap`

## Phase 3: GitHub Actions Release Workflow

- [X] Task 3.1: Create `.github/workflows/release.yml` triggered on `v*` tag push, using `goreleaser/goreleaser-action`
- [X] Task 3.2: Add snapshot build step to an existing or new CI workflow for non-tag pushes (validates GoReleaser config) [P]

## Phase 4: Platform-Specific Packaging

- [X] Task 4.1: Create `packaging/aur/PKGBUILD` for Arch Linux AUR package [P]
- [X] Task 4.2: Extend `flake.nix` to add `packages.${system}.wave` output using `buildGoModule` [P]
- [X] Task 4.3: Create `scripts/install.sh` — cross-platform install script with OS/arch detection, GitHub Release download, SHA256 checksum verification, and installation [P]

## Phase 5: Testing & Validation

- [X] Task 5.1: Test GoReleaser config with `goreleaser release --snapshot --clean`
- [X] Task 5.2: Verify `make build && ./wave --version` outputs version, commit, and date
- [X] Task 5.3: Test `nix build .#wave` produces a working binary
- [X] Task 5.4: Validate install script logic (shellcheck, manual review)

## Phase 6: Documentation

- [X] Task 6.1: Update project README or docs with installation instructions for each platform
