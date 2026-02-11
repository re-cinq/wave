# Cross-Platform Installation Method

**Feature Branch**: `050-cross-platform-install`
**Issue**: [re-cinq/wave#50](https://github.com/re-cinq/wave/issues/50)
**Author**: nextlevelshit
**Status**: Draft
**Complexity**: Complex

## Summary

Wave currently lacks a standardized installation method. We need a proper installation pipeline that provides maximum compatibility across Linux distributions and macOS.

## Target Platforms

### Linux
- **Debian/Ubuntu** — `.deb` package via APT repository
- **Arch Linux** — AUR package or PKGBUILD
- **Nix/NixOS** — Flake with `nix profile install` support

### macOS
- **Homebrew** — tap or core formula
- **Direct binary** — universal binary (amd64 + arm64)

## Requirements

- Single static binary (already a constraint per constitution)
- Versioned releases with checksums (SHA256)
- Automated release pipeline (GitHub Actions)
- `wave --version` reports build version/commit
- Install script (`curl | sh` style) as fallback for unsupported platforms

## Proposed Approach

1. **GoReleaser** for cross-compilation and packaging (deb, rpm, tar.gz, zip)
2. **GitHub Releases** as the primary distribution channel
3. **Homebrew tap** (`re-cinq/homebrew-tap`) for macOS
4. **AUR package** for Arch Linux
5. **Nix flake** output with `packages.${system}.wave` (already partially in place)
6. **Install script** that detects OS/arch and downloads the right binary

## Acceptance Criteria

- [ ] `goreleaser.yaml` configured for all target platforms
- [ ] GitHub Actions workflow triggers on tag push (`v*`)
- [ ] Debian/Ubuntu users can install via `.deb` or install script
- [ ] Arch users can install via AUR
- [ ] Nix users can install via `nix profile install`
- [ ] macOS users can install via Homebrew tap
- [ ] Install script works on all supported platforms
- [ ] `wave --version` shows version, commit, and build date

## Labels

None assigned.
