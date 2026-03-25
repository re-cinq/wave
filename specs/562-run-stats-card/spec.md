# feat(webui): run details header stats card

**Issue**: [#562](https://github.com/re-cinq/wave/issues/562)
**Labels**: enhancement, ux, frontend
**Author**: nextlevelshit
**Parent**: Extracted from #550 — Feature 1

## Problem

The run details page has a minimal summary bar (duration, step count, total tokens, start time). It lacks key context that GitHub Actions shows at a glance.

## Changes Required

### Backend
- Pass full `RunRecord.Input` to the template (currently only `InputPreview` at 80 chars)
- Parse `Input` field to detect GitHub issue/PR URLs and expose as a structured `LinkedURL` field
- Expose `CompletedAt` timestamp (already in `RunRecord`, not passed to template)

### Frontend
- Replace the summary bar with a stats card grid showing:
  - Run ID (copyable), pipeline name, full input text (expandable if long)
  - Start time, duration, finish time
  - Total tokens (prompt/completion breakdown is a stretch goal — requires schema change)
  - Branch name (clickable if GitHub URL derivable)
  - Linked issue/PR (clickable link, parsed from input)
- Card should render for both completed and in-progress runs

### Out of Scope (for now)
- Prompt/completion token split (requires DB schema migration — `total_tokens` is a single int)
- LOC changed/added (requires diff infrastructure from the diff browser issue)

## Acceptance Criteria

- [ ] Stats card renders for completed runs with all available fields
- [ ] Stats card renders for in-progress runs (duration ticks live)
- [ ] Full input text displayed (expandable for long inputs)
- [ ] Linked issue/PR URL is clickable when input contains a GitHub URL
- [ ] Branch name displayed
- [ ] Start time and finish time shown with human-readable formatting
