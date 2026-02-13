# Documentation Consistency Report — 6 Inconsistencies Found

**Issue**: [#89](https://github.com/re-cinq/wave/issues/89)
**Labels**: documentation
**Author**: nextlevelshit
**Scan date**: 2026-02-13T13:23:08Z

## Summary

Machine-generated documentation consistency report identifying 6 inconsistencies across Wave's documentation. All changes are documentation-only — no code logic changes required.

## Inconsistencies

### DOC-001 (CRITICAL): Missing `wave serve` command documentation

**Affected file**: `docs/reference/cli.md`

The `wave serve` command exists in `cmd/wave/commands/serve.go` (behind `webui` build tag) but is not documented in the CLI reference. Documentation should cover:
- Command purpose (web dashboard server)
- Options: `--port` (default 8080), `--bind` (default 127.0.0.1), `--token`, `--db`
- `WAVE_SERVE_TOKEN` environment variable
- Authentication behavior (required for non-localhost binding)
- Examples

### DOC-002 (HIGH): Incorrect persona count

**Affected files**: `docs/concepts/personas.md`, `docs/guide/personas.md`

- `docs/concepts/personas.md` states "four core personas" (line 26) but only lists 4 in its table. The actual count of built-in persona files is 14.
- `docs/guide/personas.md` correctly states 14 personas and lists all including `github-pr-creator`.
- The issue asks to update to 13 and remove `github-pr-creator` references, but `github-pr-creator.md` exists in `.wave/personas/`. The actual persona file count is 14, so docs stating 14 are already correct.
- **Resolution**: Update `docs/concepts/personas.md` from "four core personas" to reflect the actual count (14). Keep `github-pr-creator` references since the persona file exists.

### DOC-003 (HIGH): Inconsistent flag names

**Affected file**: `docs/reference/cli.md`

Line 455 shows `--status` for `wave list runs` but the actual CLI flag is `--run-status` (confirmed in `cmd/wave/commands/list.go:139`).

### DOC-004 (HIGH): Undocumented environment variables

**Affected files**: `docs/reference/environment.md`, `docs/reference/cli.md`

- `WAVE_SERVE_TOKEN`: Used in `cmd/wave/commands/serve.go` for dashboard authentication but not documented in environment reference.
- `WAVE_FORCE_TTY`: Already documented in `docs/reference/environment.md` — no action needed for this variable.

### DOC-005 (MEDIUM): Outdated persona count in concepts

**Affected file**: `docs/concepts/personas.md`

States "four core personas" but there are 14 built-in personas. The concepts page only shows Navigator, Auditor, Implementer, and Craftsman in its table — this should be updated to acknowledge all 14 while still highlighting the core ones.

### DOC-006 (MEDIUM): Stale pipeline references

**Affected file**: `README.md`

- Line 324 references a `docs` pipeline but the actual pipeline is `doc-sync`.
- Line 325 references a `migrate` pipeline but no such pipeline exists in `.wave/pipelines/`.
- Line 334 references `docs-to-impl`, `gh-poor-issues`, and `umami` pipelines that don't exist.

## Acceptance Criteria

- [ ] `wave serve` command is documented in CLI reference
- [ ] `WAVE_SERVE_TOKEN` is documented in environment reference
- [ ] Persona counts are consistent across all documentation pages
- [ ] `--status` is corrected to `--run-status` for `wave list runs` in CLI reference
- [ ] `docs/concepts/personas.md` no longer says "four core personas"
- [ ] Stale pipeline names in README are corrected to match actual pipeline files
