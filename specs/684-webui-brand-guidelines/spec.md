# feat(webui): align wave serve dashboard with re:cinq brand guidelines v1.0

**Issue**: [#684](https://github.com/re-cinq/wave/issues/684)
**Author**: nextlevelshit
**State**: OPEN

## Context

The re:cinq brand guidelines v1.0 have been finalised. The `wave serve` web dashboard currently uses a generic indigo/purple palette (`#4f46e5`), system fonts, and a custom Wave SVG logo that don't match the official brand identity.

## Brand Specifications

### Color Palette (Primary)
| Token | Name | Hex | Usage |
|-------|------|-----|-------|
| `--brand-midnight-navy` | Midnight Navy | `#0F1F49` | Dark backgrounds, navbar, sidebar |
| `--brand-crystal-white` | Crystal White | `#FFFFFF` | Light mode backgrounds, text on dark |
| `--brand-quantum-blue` | Quantum Blue | `#0014EB` | Buttons, links, active states |
| `--brand-aurora-gradient` | Aurora | `#E4E6FD -> #5664F4 -> #8F96F6` | Hero, header accents |

### Color Palette (Secondary)
| Token | Name | Hex | Usage |
|-------|------|-----|-------|
| `--brand-pulse-blue` | Pulse Blue | `#5664F4` | Running status, active nav |
| `--brand-soft-indigo` | Soft Indigo | `#8F96F6` | Hover states, secondary actions |
| `--brand-nebula-light` | Nebula Light | `#E6E8FD` | Card backgrounds (light mode) |
| `--brand-neutral-fog` | Neutral Fog | `#F2F2F7` | Page background (light mode) |

### Typography
- **Font family:** Neue Montreal (sans-serif) — all UI text
- **Mono (code/logs):** keep existing monospace stack
- **Body:** Regular 16-18px
- **Captions/labels:** Light/Regular 12-14px

### Visual Elements
- Navbar background: Midnight Navy (dark) / Crystal White (light)
- Callout/highlight blocks: Aurora gradient background, 8-12px rounded corners
- Table headers: brand primary type style, lightweight body
- Icons: clean, minimal, geometric (matching brand icon style)

## Current State vs Target

| Element | Current | Target |
|---------|---------|--------|
| `--wave-primary` | `#818cf8` (dark) / `#4f46e5` (light) | Quantum Blue `#0014EB` or Pulse Blue `#5664F4` |
| `--wave-accent` | `#8b5cf6` | Soft Indigo `#8F96F6` |
| `--color-bg` (dark) | `#0d1117` (GitHub-like) | Midnight Navy `#0F1F49` |
| `--color-bg` (light) | `#ffffff` | Crystal White `#FFFFFF` (same) |
| `--color-bg-secondary` (light) | `#f8fafc` | Neutral Fog `#F2F2F7` |
| `--color-btn-primary` | `#4f46e5` | Quantum Blue `#0014EB` |
| Font | System sans-serif | Neue Montreal |
| Logo | Custom Wave SVG (`<svg viewBox="0 0 200 200">`) | re:cinq leaf logomark |
| Title | "Wave" text in navbar | "re:cinq Wave" or "Wave" with re:cinq logomark |

## Scope

### Files to update
- `internal/webui/static/style.css` — replace all colour variables with brand tokens, add Neue Montreal `@font-face`/import, update table/callout styles
- `internal/webui/templates/layout.html` — replace inline SVG logo with re:cinq leaf logomark, update `<title>` if needed, add font link
- `internal/webui/templates/*.html` — audit for any hardcoded colour values or inline styles

### Changes
1. **CSS custom properties** — replace `:root` and `[data-theme="light"]` colour blocks with brand palette values
2. **Dark mode** — backgrounds shift from GitHub-dark (`#0d1117`) to Midnight Navy (`#0F1F49`), secondary surfaces use slightly lighter navy tones
3. **Light mode** — backgrounds use Crystal White/Neutral Fog, accents use Quantum Blue
4. **Navbar** — replace inline SVG wave logo with re:cinq leaf logomark SVG; keep "Wave" text or change to "re:cinq Wave"
5. **Buttons** — primary buttons use Quantum Blue; hover uses Pulse Blue
6. **Status colours** — keep functional green/amber/red semantic colours, but adjust tonal balance against new backgrounds
7. **Typography** — load Neue Montreal; update `--font-sans` to prioritise it
8. **Tables** — headers use brand primary type style per guidelines; body text lightweight
9. **Cards/callouts** — aurora gradient accent on status cards, 8-12px radius (already close at `--radius-md: 8px`)

### Out of scope
- VitePress documentation site (tracked in #683)
- TUI/terminal display colours (`internal/display/`)
- Functional behaviour changes

## Acceptance Criteria
- [ ] `internal/webui/static/style.css` uses brand hex values exclusively (no legacy `#4f46e5`, `#0d1117` etc.)
- [ ] Neue Montreal renders in dashboard UI (with sensible fallback)
- [ ] Navbar shows re:cinq leaf logomark instead of wave SVG
- [ ] Dark mode backgrounds are Midnight Navy (`#0F1F49`)
- [ ] Light mode backgrounds are Crystal White / Neutral Fog
- [ ] Buttons and links use Quantum Blue (`#0014EB`)
- [ ] `wave serve` loads cleanly in browser with no visual regressions
- [ ] Status colours (completed/running/failed) remain distinguishable against new backgrounds
