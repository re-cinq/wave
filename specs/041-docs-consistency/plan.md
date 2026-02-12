# Implementation Plan — #41 Documentation Consistency

## Objective

Fix the 3 remaining documentation inconsistencies identified in the documentation consistency report: update pipeline count in README.md, document the GitHub adapter in the adapters reference, and document `GITHUB_TOKEN`/`GH_TOKEN` in the environment reference.

## Approach

Straightforward documentation edits across 3 files. No code changes, no architectural decisions. Each fix is precisely scoped with exact file locations and content requirements.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `README.md` | modify | Update "18 built-in pipelines" to "19" (lines 288 and 300) |
| `docs/reference/adapters.md` | modify | Add GitHub adapter section between OpenCode and Multiple Adapters sections |
| `docs/reference/environment.md` | modify | Add `GITHUB_TOKEN`/`GH_TOKEN` to the Required Environment Variables table |

## Architecture Decisions

None — this is a documentation-only change with no design implications.

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Pipeline count changes again before merge | Low | The count is derived from `internal/defaults/pipelines/` — verify at implementation time |
| GitHub adapter API changes | Low | Document the current interface as found in `internal/adapter/github.go` |

## Testing Strategy

- **No automated tests needed** — purely documentation changes
- **Manual verification**: Confirm pipeline count matches `ls internal/defaults/pipelines/*.yaml | wc -l`
- **Link validation**: Ensure any cross-references in new docs are valid
- **Build check**: Run `go test ./...` to confirm no test references to old pipeline counts
