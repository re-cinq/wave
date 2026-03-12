# Tasks

## Phase 1: Prerequisites & Installation Docs

- [X] Task 1.1: Expand `README.md` Requirements section with a dependency table listing all tools (wave, claude/LLM CLI, gh, git), their purposes, and whether they are required or optional
- [X] Task 1.2: Expand `docs/guide/installation.md` prerequisites section with install instructions per tool and platform (macOS, Linux, Nix), and add a `wave doctor` verification step

## Phase 2: CI/CD Guide Consolidation

- [X] Task 2.1: Merge useful content from `docs/future/integrations/github-actions.md` into `docs/guides/ci-cd.md` — specifically the workflow permissions section, adapter caching, and troubleshooting details not already present [P]
- [X] Task 2.2: Add a "Headless / No-TTY Mode" section to `docs/guides/ci-cd.md` documenting `--no-tui`, `WAVE_FORCE_TTY=0`, `TERM=dumb`, `-o json`, and CI auto-detection [P]
- [X] Task 2.3: Add a "Health Checks in CI" section to `docs/guides/ci-cd.md` showing `wave doctor --json` as a CI gate step with exit code handling [P]
- [X] Task 2.4: Add a "Secret Injection" section to `docs/guides/ci-cd.md` covering `runtime.sandbox.env_passthrough`, which vars each adapter needs, and CI-specific examples [P]
- [X] Task 2.5: Add deprecation/redirect notice to `docs/future/integrations/github-actions.md` pointing to `docs/guides/ci-cd.md`

## Phase 3: Example Workflow

- [X] Task 3.1: Create `.github/workflows/wave-ci-example.yml` — a production-ready GitHub Actions workflow that installs Wave + Claude Code, runs `wave doctor`, executes a pipeline, uploads artifacts, and handles failures

## Phase 4: Validation

- [X] Task 4.1: Run `go test ./...` to verify no regressions
- [X] Task 4.2: Validate YAML syntax of the example workflow
