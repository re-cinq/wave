# feat(webui): add SVG icons for forges and adapters on landing page

**Issue**: [#703](https://github.com/re-cinq/wave/issues/703)
**Labels**: enhancement, frontend
**Author**: nextlevelshit

## Description

The landing page currently shows forge and adapter names as plain text. Add SVG icons for each supported forge (GitHub, GitLab, Bitbucket) and adapter (claude-code, browser, mock) to improve visual clarity and make the dashboard more approachable for first-time users.

## Scope

- **Forges**: GitHub, GitLab, Bitbucket
- **Adapters**: claude-code, browser, mock
- **Location**: Landing page dashboard (`internal/webui/`)

## Acceptance Criteria

- [ ] SVG icons for each forge type displayed alongside forge names
- [ ] SVG icons for each adapter type displayed alongside adapter names
- [ ] Icons are consistent in style and size (recommend 20x20px inline SVGs)
- [ ] Fallback to text-only display if icons fail to load
- [ ] Icons work in both light and dark themes (use `currentColor` or CSS variables)
- [ ] Icons are embedded as inline SVG in templates (no external asset loading)

## Implementation Notes

- Inline SVG preferred over external files since webui uses Go `html/template` with embedded assets
- Use `currentColor` fill so icons inherit text color across themes
- Consider a shared `icons.go` or template partial for reuse across pages
