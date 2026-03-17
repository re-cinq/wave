# feat(webui): polish run list view with status indicators, filtering, and sorting

**Issue**: [#460](https://github.com/re-cinq/wave/issues/460)
**Parent**: #455
**Labels**: enhancement, ux, frontend
**Author**: nextlevelshit

## Summary

Polish the run list view with better status indicators, filtering UX, and sorting controls to match GitHub Actions workflow run list quality. The current implementation has cursor pagination and status/pipeline/since filters but needs visual refinement.

## Acceptance Criteria

- [ ] Status indicators use clear iconography and color coding (running=animated, completed=green, failed=red, cancelled=grey) — matching GitHub Actions visual language
- [ ] Filter controls are intuitive: status dropdown, pipeline dropdown, date range — with clear active-filter indication
- [ ] Sorting controls for start time, duration, and status with visual sort-direction indicators
- [ ] Run rows show: pipeline name, branch/trigger, status badge, duration, relative timestamp
- [ ] Hover states and row click affordances are clear
- [ ] Pagination controls are polished (current cursor pagination works but needs visual refinement)
- [ ] Empty state when no runs match filters ("No runs found matching your filters")

## Dependencies

- #459 (UX audit) — audit findings inform specific improvements needed

## Scope Notes

- **In scope**: Visual polish and UX improvements to the existing run list page
- **Out of scope**: New API endpoints or backend filter capabilities
- **Out of scope**: Adding new pages or views
