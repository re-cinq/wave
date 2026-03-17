# feat(webui): responsive layout and styling consistency across all pages

**Issue**: [#463](https://github.com/re-cinq/wave/issues/463)
**Parent**: #455
**Labels**: enhancement, ux, frontend
**Author**: nextlevelshit

## Summary

Audit and fix responsive layout and styling consistency across all 15 webui pages. Ensure consistent spacing, typography, color usage, and responsive behavior so the UI feels cohesive rather than a collection of individually-built pages.

## Acceptance Criteria

- [ ] Consistent spacing system applied across all pages (margins, padding, gaps)
- [ ] Typography hierarchy is uniform (heading sizes, body text, monospace code)
- [ ] Color palette usage is consistent (status colors, backgrounds, borders, text colors in both light and dark mode)
- [ ] Button styles are consistent across all pages (primary, secondary, danger variants)
- [ ] Tables and lists use the same styling patterns across all pages
- [ ] Layout is responsive: usable at 1024px, 1440px, and 1920px viewport widths
- [ ] Navigation sidebar/header is consistent and highlights the active page
- [ ] Dark mode and light mode both look polished (no contrast issues, no missing theme variables)

## Scope Notes

- **In scope**: CSS/styling consistency and responsive layout fixes
- **Out of scope**: New UI components or pages
- **Out of scope**: JavaScript behavior changes (covered by other issues)

## Dependencies

- #459 (UX audit) — audit findings identify specific inconsistencies
