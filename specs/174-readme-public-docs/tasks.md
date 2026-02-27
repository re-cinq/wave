# Tasks

## Phase 1: Audit and Preparation

- [X] Task 1.1: Audit README.md for internal-only references (private registries, internal URLs, team-specific tooling, credentials)
- [X] Task 1.2: Audit `docs/guide/installation.md` for stale private-repo warnings and internal references
- [X] Task 1.3: Determine correct license (MIT vs Apache-2.0) — check with maintainer or resolve the inconsistency between README badge and `.goreleaser.yaml`
- [X] Task 1.4: Check `go.mod` module path vs GitHub URL (`github.com/recinq/wave` vs `github.com/re-cinq/wave`) to determine if `go install` is viable

## Phase 2: Core Documentation Updates

- [X] Task 2.1: Rewrite README.md Installation section with public install methods (install script, GitHub Releases, build from source, Nix) [P]
- [X] Task 2.2: Revise README.md Quick Start section to be self-contained for new users (clone → build → init → hello-world) [P]
- [X] Task 2.3: Remove `::: warning Private Repository` block from `docs/guide/installation.md` and update language to reflect public repo status [P]

## Phase 3: New Files

- [X] Task 3.1: Create `LICENSE` file with the resolved license text (MIT or Apache-2.0)
- [X] Task 3.2: Create `CONTRIBUTING.md` with prerequisites, build instructions, test commands, commit conventions, and PR workflow
- [X] Task 3.3: Add Contributing section to README.md linking to `CONTRIBUTING.md` [P]
- [X] Task 3.4: Fix license badge in README.md if license was changed from what the badge currently says [P]

## Phase 4: Consistency and Cleanup

- [X] Task 4.1: Update `.goreleaser.yaml` license field to match the chosen license if it changed
- [X] Task 4.2: Search entire repo for remaining internal-only references and fix any found
- [X] Task 4.3: Validate all markdown links in README.md point to existing files

## Phase 5: Verification

- [X] Task 5.1: Run `make build` to confirm build-from-source instructions work
- [X] Task 5.2: Run `go test ./...` to confirm no tests were broken
- [X] Task 5.3: Walk through the README Quick Start flow manually (or document the expected flow)
- [X] Task 5.4: Grep for license string mismatches across the repo
