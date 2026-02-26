# fix(docs): CSS styles broken on GitHub Pages after repo URL change

**Feature Branch**: `165-docs-css-base-url`
**Issue**: [#165](https://github.com/re-cinq/wave/issues/165)
**Labels**: bug, documentation, priority: medium
**Author**: nextlevelshit
**Status**: Draft

## Summary

The documentation site experienced styling problems after the repository transitioned to public status. The URL changed to re-cinq.github.io/wave/, and this new path structure prevents CSS stylesheets from loading properly.

## Current Behavior

- Documentation pages at the new URL load without functional CSS styling
- GitHub Pages' folder structure prevents stylesheet paths from resolving correctly

## Expected Behavior

- All pages should display with proper styling applied
- Visual elements including navigation, typography, and layout should render as designed

## Root Cause

The VitePress configuration at `docs/.vitepress/config.ts` is missing the `base: '/wave/'` property. When a GitHub Pages site is served under a subpath (e.g., `re-cinq.github.io/wave/`), VitePress needs this `base` setting to correctly prefix all asset URLs (CSS, JS, images) with the subpath. Without it, assets are requested from the root (`/assets/...`) instead of the correct subpath (`/wave/assets/...`), resulting in 404s.

Additionally, the `head` config contains hardcoded absolute paths for OG image (`/og-image.png`) and favicon (`/favicon.svg`) that won't resolve under the `/wave/` subpath.

## Acceptance Criteria

- CSS renders correctly across all documentation pages
- Base URL/path prefix configuration updated for public repository (`base: '/wave/'`)
- Layout and navigation display as intended
- OG image and favicon paths resolve correctly under `/wave/` subpath

## Requirements

### Functional Requirements

- **FR-001**: VitePress config MUST include `base: '/wave/'` so all generated asset paths include the subpath prefix
- **FR-002**: OG image meta tag MUST reference the correct path under `/wave/`
- **FR-003**: Favicon link MUST reference the correct path under `/wave/`
- **FR-004**: All documentation pages MUST load CSS and JS assets correctly when served at `re-cinq.github.io/wave/`

## Success Criteria

- **SC-001**: Visiting `https://re-cinq.github.io/wave/` renders the documentation with full CSS styling
- **SC-002**: VitePress build completes without errors with the new base path
- **SC-003**: Browser DevTools shows no 404 errors for CSS, JS, or image assets
