# Work Items — #1304 audit source_cli

## Phase 1: Setup
- [X] Item 1.1: Confirm branch `1304-audit-source-cli` is checked out from clean `main`.
- [X] Item 1.2: Confirm `internal/skill/source_cli.go` is still absent at HEAD (sanity).

## Phase 2: Core Implementation
- [X] Item 2.1: Create `docs/audit/` directory.
- [X] Item 2.2: Write `docs/audit/1304-source-cli-removal.md` with: PR #1080 origin, commit `6e0fc562` removal reference, link to #1113 skills overhaul, audit-trail closing note.

## Phase 3: Testing
- [X] Item 3.1: Run `go test ./...` to confirm no regression (contract test).
- [X] Item 3.2: Verify `docs/audit/1304-source-cli-removal.md` renders as Markdown (no syntax errors).

## Phase 4: Polish
- [X] Item 4.1: Stage spec/plan/tasks under `specs/1304-audit-source-cli/` and the new audit doc; reset `.wave/`, `.agents/`, `.claude/`, `CLAUDE.md` per CLAUDE.md commit hygiene.
- [X] Item 4.2: Commit with `docs(audit):` prefix; open PR referencing issue #1304 and PR #1080.
