# Implementation Plan

## Objective

Update the README.md command tables and CLI Reference block to document all 22 registered CLI commands, ensuring comprehensive coverage.

## Approach

Single-file documentation update to `README.md`. Two sections need changes:

1. **CLI Reference block** (lines ~107-148): Add the 7 missing commands (resume, compose, doctor, suggest, skills, postmortem, agent) to the text-art command listing
2. **Commands tables** (lines ~152-188): Add the 10 missing commands to the appropriate table sections, creating new sections where needed

## File Mapping

| File | Action | Details |
|------|--------|---------|
| `README.md` | modify | Update CLI Reference block and Commands tables |

## Architecture Decisions

1. **Grouping**: New commands will be organized into existing or new table sections:
   - **Pipeline Execution**: Add `resume`, `compose` (pipeline-related)
   - **Monitoring & Inspection**: Add `chat`, `postmortem` (analysis tools)
   - **Maintenance**: Add `doctor`, `suggest`, `serve`, `migrate`, `skills`, `agent`
2. **Description source**: Use the `Short` field from each command's Go source as the canonical description
3. **CLI Reference block**: Add commands in alphabetical order to match existing convention

## Risks

- **Stale descriptions**: If command descriptions change in Go source, README will drift. Mitigation: descriptions are pulled directly from source for this PR.
- **No risk of code breakage**: Documentation-only change.

## Testing Strategy

- Visual review that all 22 commands appear in both sections
- No automated tests needed (docs-only change)
