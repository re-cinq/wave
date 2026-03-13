# Implementation Plan — Documentation Consistency (#358)

## Objective

Fix 14 documentation inconsistencies between the docs and actual codebase, spanning manifest schema, CLI reference, pipeline documentation, and persona counts.

## Approach

Documentation-only changes across 4-5 files. No Go code changes required. Each DOC item is a targeted text edit to bring docs in line with the actual code.

## File Mapping

| File | Action | DOC Items |
|------|--------|-----------|
| `docs/reference/manifest-schema.md` | modify | DOC-001, DOC-006, DOC-007 |
| `docs/reference/contract-types.md` | modify | DOC-002 (verify, possibly no-op) |
| `docs/reference/cli.md` | modify | DOC-004, DOC-005, DOC-010 |
| `docs/guide/pipelines.md` | modify | DOC-003, DOC-008, DOC-009 |
| `README.md` | modify | DOC-008, DOC-012 |
| `docs/reference/environment.md` | modify | DOC-011 |

## Architecture Decisions

1. **DOC-001 — Remove `skills` from manifest top-level**: The `Manifest` struct has no `skills` field. Skills are declared per-pipeline via `requires.skills`. The `SkillConfig` section should be moved/reframed as pipeline-level documentation, and removed from the top-level fields table. The complete example at the bottom should also remove the `skills:` block.

2. **DOC-002 — Verify, likely no-op**: `on_failure` IS present on `ContractConfig` in the code (`internal/pipeline/types.go:256`). The docs appear correct. Mark as verified-correct. If the issue refers to something more specific (e.g., `on_failure` on a contract type that doesn't support it), that would need closer look, but based on code the field exists.

3. **DOC-003 — Remove phantom platform pipelines**: The gl-*, gt-*, bb-* pipeline sections should be removed from `docs/guide/pipelines.md`. These platform variants don't exist as files. Add a note that these are planned/future if appropriate, or simply remove them.

4. **DOC-004 — Add `--preserve-workspace`**: Add to the `wave run` options section in CLI reference.

5. **DOC-005 — Add `wave migrate validate`**: Add to the migrate section in CLI reference.

6. **DOC-006 — Add `Persona.sandbox`**: Add `sandbox` field to the Persona table in manifest-schema.md with its sub-fields.

7. **DOC-007 — Add missing fields**: Add `match_labels` to `RoutingRule` table. Add `context_window` and `summarizer_persona` to `RelayConfig` table.

8. **DOC-008 — Fix pipeline count**: Update "47 built-in" to the actual count. After removing phantom pipelines, the documented count should match actual `.wave/pipelines/` file count (46).

9. **DOC-009 — Document missing pipelines**: Add the 17 undocumented pipelines to appropriate sections in `docs/guide/pipelines.md`.

10. **DOC-010 — Likely no-op**: `--manifest` is already documented as a global flag. Verify and close.

11. **DOC-011 — Document token precedence**: Add a note about `GH_TOKEN` taking precedence over `GITHUB_TOKEN` in `docs/reference/environment.md`.

12. **DOC-012 — Clarify persona count**: The count "30" is technically correct (31 files minus base-protocol.md). Add a note clarifying base-protocol.md is not a persona, or adjust wording.

13. **DOC-013 — Out of scope**: Comprehensive persona docs is a separate effort. Note in PR.

14. **DOC-014 — Investigate reviewer.yaml**: Check if the `.yaml` format is intentionally supported alongside `.md`. If consistent with other persona configs (which also have `.yaml` files), it's not anomalous.

## Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Phantom pipeline removal may surprise users expecting them | Low | They never existed as files; removing docs removes confusion |
| Pipeline count may change between PR creation and merge | Low | Use exact count at time of edit |
| DOC-002 may have a subtlety not captured | Low | Verify `on_failure` behavior in contract code before marking as resolved |

## Testing Strategy

- No Go code changes, so `go test ./...` should pass unchanged
- Manual review: verify all doc links still resolve
- Verify markdown formatting is valid
- Cross-reference each DOC item against the spec checklist
