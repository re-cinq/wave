# Implementation Plan — #1304 audit source_cli

## Objective

Resolve audit finding from issue #1304: document that
`internal/skill/source_cli.go` was intentionally removed in commit
`6e0fc562`, closing the partial-fidelity audit on PR #1080.

## Approach

This is a documentation-only resolution. The audit pipeline flagged a
missing file that turns out to have been intentionally removed during a
later refactor. The plan is to drop a minimal audit-trail note under
`docs/audit/` that future audits and humans can grep, and update the
docs index so the trail is discoverable.

No source-code change is required. No tests need to change.

## File Mapping

- **Create** `docs/audit/1304-source-cli-removal.md`
  - Brief note: PR #1080 introduced `source_cli.go`; commit `6e0fc562`
    removed it as part of the skills overhaul (#1113); audit closed.
- **Modify** none — the existing audit doc surface is just a directory.

## Architecture Decisions

- **Doc location**: place under `docs/audit/` (new dir). Rationale:
  audit findings deserve a dedicated trail separate from ADRs (decision
  records) and changelog (release notes). Future wave-audit runs can
  consult or grep this directory.
- **No code comment in `internal/skill`**: dropping a "see audit
  #1304" comment in production source would be noise — the git
  history already explains the deletion.
- **No issue-only close**: pipeline produces a PR; a documentation
  note is the smallest change that both creates a PR and resolves the
  audit trail.

## Risks

- **Risk**: introducing `docs/audit/` as a new directory might
  conflict with future audit conventions.
  **Mitigation**: name the file with the issue number prefix so the
  format scales; the directory is empty otherwise.
- **Risk**: reviewer may prefer issue-close-only resolution.
  **Mitigation**: doc is minimal (≤30 lines) and easily reverted.

## Testing Strategy

- No unit/integration tests — documentation-only change.
- Contract test (`go test ./...`) must still pass unchanged.
- Verify `docs/audit/1304-source-cli-removal.md` renders as plain
  Markdown.
