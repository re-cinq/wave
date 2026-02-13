# Implementation Plan: Documentation Consistency Fixes

## Objective

Fix 6 documentation inconsistencies identified by the doc-loop pipeline scan, ensuring all documentation accurately reflects the current codebase state (commands, flags, personas, pipelines, environment variables).

## Approach

This is a documentation-only change touching 4 files. Each fix is a targeted text edit — no code changes required. The fixes are independent and can be applied in any order, but we group them by file to minimize context switching.

## File Mapping

| File | Action | Fixes Applied |
|------|--------|---------------|
| `docs/reference/cli.md` | modify | DOC-001 (add `wave serve` section), DOC-003 (fix `--status` → `--run-status`) |
| `docs/reference/environment.md` | modify | DOC-004 (add `WAVE_SERVE_TOKEN` variable) |
| `docs/concepts/personas.md` | modify | DOC-002, DOC-005 (update persona count from "four" to actual count) |
| `README.md` | modify | DOC-006 (fix stale pipeline names: `docs` → `doc-sync`, remove `migrate`, clean up non-existent pipeline references) |

## Architecture Decisions

1. **DOC-001 — `wave serve` documentation style**: Follow the existing pattern in `cli.md` — each command gets a `## wave <command>` section with description, example usage, output block, and options table. Place it after `wave validate` and before `wave migrate` since `serve` is a server command distinct from pipeline operations.

2. **DOC-002/DOC-005 — Persona count**: The actual count of persona `.md` files in `.wave/personas/` is 14 (including `github-pr-creator`). The issue says to update to 13 and remove `github-pr-creator`, but since the file exists, we keep the count at 14. The concepts page should acknowledge all 14 while still highlighting the "core" ones in its detail section. Change "four core personas" to "14 built-in personas" with a note that the page highlights a representative subset.

3. **DOC-003 — Flag name**: Simple text replacement. The code uses `--run-status` for `wave list runs`. The `--status` flag on `wave clean` is correct and should not be changed.

4. **DOC-006 — Pipeline references**: Replace `docs` with `doc-sync`. Remove the `migrate` pipeline row entirely (no such pipeline exists). Clean up the "More pipelines" list to only reference pipelines that actually exist in `.wave/pipelines/`.

## Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Persona count changes again before merge | Low | Count is stable; PR description should note the count was verified against `.wave/personas/` |
| `wave serve` is behind build tag (`webui`) | Medium | Document that `wave serve` requires the `webui` build tag |
| Pipeline list changes before merge | Low | Changes would be caught by re-running doc-loop |

## Testing Strategy

- **Manual review**: Read each modified file to verify consistency with source code
- **Cross-reference check**: Verify persona count matches `ls .wave/personas/ | wc -l`
- **CLI flag check**: Verify `--run-status` against `cmd/wave/commands/list.go`
- **Pipeline check**: Verify referenced pipeline names against `ls .wave/pipelines/`
- **Link check**: Ensure any new internal links in cli.md point to valid anchors
- **Build verification**: `go build ./...` to confirm no code changes were inadvertently made (should be no-op for docs-only changes)
