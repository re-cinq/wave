# Implementation Plan

## Objective

Add a "Logs vs Progress Output" comparison section to `docs/reference/cli.md` explaining the difference between `wave logs` (post-hoc event history from the state DB) and `--output` modes (real-time progress rendering during execution), with 3 use-case examples.

## Approach

Insert a new section into the CLI reference document between the `wave logs` section (ends at line ~268) and the `wave cancel` section (starts at line ~272). This placement is logical because it directly follows the `wave logs` documentation and provides context before moving on to other commands.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `docs/reference/cli.md` | modify | Add "Logs vs Progress Output" section after line ~268 |

## Architecture Decisions

- **Placement**: After `wave logs` section, before `wave cancel`. This keeps the comparison close to the `wave logs` docs where users are most likely looking for it.
- **Format**: Use a comparison table followed by 3 concrete use-case examples with command snippets. Consistent with the rest of the CLI reference style.
- **No code changes**: Pure documentation — no Go files modified.

## Risks

- **Line numbers may drift**: The issue references specific line numbers which may have shifted. Mitigated by using section headers for anchoring instead.
- **Minimal risk**: Single-file docs change with no code impact.

## Testing Strategy

- No automated tests needed (documentation-only change)
- Manual validation: ensure the markdown renders correctly and the section flows logically within the document
- Verify `go test ./...` still passes (no code changes, but confirms no regressions)
