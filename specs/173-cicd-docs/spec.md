# feat: dependency preflight checks, onboarding guidance, and CI/CD pipeline support

**Issue**: [re-cinq/wave#173](https://github.com/re-cinq/wave/issues/173)
**Author**: nextlevelshit
**Labels**: documentation, enhancement, ux, ci, priority: high
**State**: OPEN

## Problem

Wave currently assumes that all required CLI tools (Claude Code, `tea`, `gh`, speckit skills) are pre-installed on the host machine. This works on the maintainer's development environment but will break for any new user or CI/CD runner that does not have these dependencies.

## Status (March 2026)

Per maintainer comment, the core implementation work is **already complete**:

1. **Preflight dependency detection** — done: `internal/preflight/preflight.go` validates tools + skills before pipeline execution, supports auto-install
2. **`wave doctor` command** — done: checks Wave init status, adapter health, forge CLI, tools, skills. Supports `--json`, `--fix`, exit codes 0/1/2
3. **Onboarding wizard** — done: `internal/onboarding/` — interactive 5-step setup

### What Remains

The remaining work is **documentation-focused**:

- CI/CD documentation and examples (GitHub Actions) — needs hardening/completion
- Environment variable secret injection docs — needs completion
- Headless/no-TTY mode verification and documentation

## Existing Documentation State

Several docs already exist but need review, consolidation, and hardening:

- `docs/guides/ci-cd.md` — CI/CD integration guide with GitHub Actions and GitLab CI examples
- `docs/future/integrations/github-actions.md` — Detailed GitHub Actions examples (currently in `future/`)
- `docs/reference/environment.md` — Environment variables reference including CI/CD detection and credential handling
- `docs/guide/installation.md` — Installation guide with prerequisites
- `README.md` — Has Requirements section listing Go 1.25+, LLM CLI adapter, Nix

## Existing Code Support for CI/CD

- **CI detection**: Wave auto-detects CI environments via `GITHUB_ACTIONS`, `GITLAB_CI`, `CI`, etc. (documented in `docs/reference/environment.md`)
- **Headless mode**: `--no-tui` flag and `WAVE_FORCE_TTY=0` disable interactive TUI; `TERM=dumb` implies both `--no-color` and `--no-tui`
- **JSON output**: `-o json` for machine-parseable output
- **Credential flow**: `runtime.sandbox.env_passthrough` controls which env vars reach adapter subprocesses
- **Preflight checks**: `internal/preflight/` validates tools and skills before execution
- **Doctor command**: `wave doctor --json` for CI health checks with exit codes 0/1/2

## Acceptance Criteria

1. A new user cloning the repo gets clear guidance on missing dependencies before any pipeline step fails
2. `wave run` fails fast with actionable error messages when dependencies are missing
3. At least one CI/CD example (GitHub Actions) is documented and tested
4. All required CLIs and their minimum versions are listed in documentation
5. Existing CI/CD docs are consolidated (move `future/integrations/github-actions.md` content into main docs)
6. Headless/no-TTY mode is documented with examples

## Out of Scope

- Automatic installation of dependencies (users install their own tools)
- Supporting every possible CI/CD platform (start with GitHub Actions)
- New code features — this is documentation completion work
