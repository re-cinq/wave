# Implementation Plan: Brand Guidelines Alignment

## Objective

Replace the Wave documentation site's generic indigo/purple palette and system fonts with the re:cinq brand guidelines v1.0 color scheme (Midnight Navy, Quantum Blue, Aurora gradients) and Neue Montreal typography.

## Approach

This is a CSS-first theming change. The VitePress theme uses CSS custom properties (`--wave-*` and `--vp-c-brand-*`) extensively, making the color swap straightforward. Typography requires adding `@font-face` declarations for Neue Montreal and updating the font stack. Logo/favicon are asset replacements.

**Strategy:** Work bottom-up from CSS variables (foundation) through component styles, then config/assets, finishing with the hero section.

## File Mapping

### Modified Files
| File | Change |
|------|--------|
| `docs/.vitepress/theme/styles/custom.css` | Replace all `--wave-*` color variables with brand palette values; add `@font-face` for Neue Montreal; update `--vp-c-brand-*` mappings; update dark mode overrides to Midnight Navy |
| `docs/.vitepress/theme/styles/components.css` | Update hardcoded hex values in component styles to use brand CSS variables; update gradients to Aurora; adjust border-radius to 8-12px per brand spec |
| `docs/.vitepress/config.ts` | Update font preconnect links if using CDN; update favicon path if format changes |
| `docs/public/logo.svg` | Replace wave motif SVG with re:cinq blue leaf logomark with brackets/slashes |
| `docs/public/favicon.ico` | Replace with simplified leaf icon favicon |
| `docs/index.md` | Update any hardcoded color references in hero section inline styles |

### No Changes Expected
| File | Reason |
|------|--------|
| `docs/.vitepress/theme/index.ts` | Component registrations unchanged |
| `docs/.vitepress/theme/types.ts` | Type definitions unchanged |
| Vue components | They consume CSS variables; no source changes needed unless they have hardcoded hex values |

### Possible New Files
| File | Condition |
|------|-----------|
| `docs/public/fonts/NeueMontreal-*.woff2` | If self-hosting the font (preferred for performance) |

## Architecture Decisions

1. **Self-hosted fonts over CDN**: Neue Montreal is not on Google Fonts. Self-host WOFF2 files in `docs/public/fonts/` with `@font-face` declarations. This avoids third-party dependencies and GDPR concerns.

2. **Preserve `--wave-*` variable naming**: Keep the `--wave-` prefix for project-specific variables but update their values to brand colors. This avoids a mass rename across all component files. Add `--brand-*` aliases for direct brand token access.

3. **Keep functional status colors**: Trust badges use green/amber/red for semantic meaning. These stay but are verified against new brand backgrounds for contrast.

4. **Logo as SVG**: Create a clean SVG leaf logomark matching the brand description (blue leaf with brackets/slashes). The favicon can be generated from the same SVG source.

5. **Aurora gradient as reusable variable**: Define `--brand-aurora-gradient` as a CSS custom property so it can be reused across hero, feature cards, and other sections.

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Neue Montreal font licensing | Cannot distribute proprietary font | Check if font files are available in project assets or use a suitable open-source alternative (e.g., PP Neue Montreal if licensed, or Montreal as fallback) |
| Hardcoded hex values in Vue components | Brand colors not applied everywhere | Grep for all hex color literals (`#4f46e5`, `#4338ca`, `#06b6d4`, `#8b5cf6`) across Vue/CSS files |
| Dark mode contrast | Text unreadable against Midnight Navy | Test WCAG AA contrast ratios for all text/background combinations |
| Terminal preview colors | Syntax highlighting may clash with new palette | Terminal uses its own isolated color scheme; verify it works against Midnight Navy |
| Missing asset files | Logo/favicon not attached to issue | Create placeholder SVGs matching the brand description; flag for design review |

## Testing Strategy

1. **Visual regression**: Run `npm run docs:dev` and manually verify:
   - Home page hero with Aurora gradient
   - Light mode: Crystal White/Neutral Fog backgrounds
   - Dark mode: Midnight Navy backgrounds
   - Feature cards with brand colors
   - Trust badges remain readable
   - Terminal preview contrast

2. **Build verification**: Run `npm run docs:build` to confirm no build errors

3. **Grep validation**: Verify no legacy hex values remain:
   - `#4f46e5` (old primary)
   - `#4338ca` (old primary-dark)
   - `#06b6d4` (old secondary)
   - `#8b5cf6` (old accent)

4. **Font loading**: Verify Neue Montreal renders in headings and body text in browser DevTools

5. **Cross-browser**: Verify `@font-face` loads in Chrome, Firefox, Safari
