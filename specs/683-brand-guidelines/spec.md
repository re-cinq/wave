# docs: align VitePress documentation site with re:cinq brand guidelines v1.0

**Issue:** [#683](https://github.com/re-cinq/wave/issues/683)
**Repository:** re-cinq/wave
**Author:** nextlevelshit
**State:** OPEN
**Labels:** none

## Context

The re:cinq brand guidelines v1.0 have been finalised. The Wave documentation site (VitePress) currently uses a generic indigo/purple palette and system fonts that don't match the official brand identity.

## Brand Specifications

### Color Palette (Primary)
| Token | Name | Hex | Usage |
|-------|------|-----|-------|
| `--brand-midnight-navy` | Midnight Navy | `#0F1F49` | Main neutral, dark backgrounds, typography |
| `--brand-crystal-white` | Crystal White | `#FFFFFF` | Light backgrounds, text on dark |
| `--brand-quantum-blue` | Quantum Blue | `#0014EB` | Accent: buttons, links, highlights |
| `--brand-aurora-start` | Aurora gradient start | `#E4E6FD` | Gradient backgrounds |
| `--brand-aurora-mid` | Aurora gradient mid | `#5664F4` | Gradient backgrounds |
| `--brand-aurora-end` | Aurora gradient end | `#8F96F6` | Gradient backgrounds |

### Color Palette (Secondary)
| Token | Name | Hex | Usage |
|-------|------|-----|-------|
| `--brand-pulse-blue` | Pulse Blue | `#5664F4` | Supporting accent |
| `--brand-soft-indigo` | Soft Indigo | `#8F96F6` | Supporting accent |
| `--brand-nebula-light` | Nebula Light | `#E6E8FD` | Light backgrounds, cards |
| `--brand-neutral-fog` | Neutral Fog | `#F2F2F7` | Subtle backgrounds |

### Typography
- **Font family:** Neue Montreal (sans-serif) -- headlines and body
- **H1:** Bold, 64px/48pt
- **H2:** Medium/Bold, 40px/30pt
- **H3:** Medium/Regular, 24px/18pt
- **Body:** Regular, 16-18px/12-14pt
- **Captions:** Light/Regular, 12-14px/9-10pt
- **Mono (code):** keep JetBrains Mono

### Visual Elements
- Aurora gradient for hero/banner sections
- Rounded corners: 8-12px (callout/highlight blocks)
- Brand Pattern 1.0: white at 20-30% opacity on Aurora/Pulse Blue backgrounds
- Leaf logomark (blue, brackets/slashes inside)
- Clean, minimal icon style with soft gradients

## Scope

### Files to update
- `docs/.vitepress/theme/styles/custom.css` -- replace `--wave-*` colour variables with brand tokens, add Neue Montreal `@font-face` or Google Fonts import
- `docs/.vitepress/theme/styles/components.css` -- update component colours
- `docs/.vitepress/config.ts` -- update font preconnect, og metadata, site title/description if needed
- `docs/public/logo.svg` -- replace with re:cinq logomark (blue leaf)
- `docs/public/favicon.ico` -- replace with re:cinq favicon (simplified leaf icon)
- `docs/index.md` -- update hero section if it uses hardcoded brand elements

### Changes
1. **Colour tokens** -- map all `--wave-primary`, `--wave-accent`, `--wave-secondary` to brand palette. Dark mode: use Midnight Navy backgrounds. Light mode: use Crystal White/Neutral Fog backgrounds with Quantum Blue accents
2. **VitePress brand vars** -- `--vp-c-brand-1/2/3` should map to Quantum Blue / Pulse Blue / Soft Indigo
3. **Hero gradient** -- replace current `linear-gradient(135deg, ...)` with Aurora gradient
4. **Typography** -- add Neue Montreal as primary sans-serif font; keep JetBrains Mono for code
5. **Logo & favicon** -- replace Wave SVG with re:cinq leaf logomark
6. **Feature cards** -- update hover shadow, border, and background to match brand callout style (Aurora gradient bg, 8-12px radius)
7. **Trust badges / status colours** -- keep functional green/amber/red but ensure they work against new backgrounds

### Out of scope
- Content rewrites (that's a separate pass)
- `wave serve` web dashboard (separate issue)

## Acceptance Criteria
- [ ] `docs/.vitepress/theme/styles/custom.css` uses brand hex values exclusively (no legacy `#4f46e5` etc.)
- [ ] Neue Montreal loads correctly and renders in headings + body
- [ ] Logo and favicon match re:cinq logomark
- [ ] Dark mode uses Midnight Navy (`#0F1F49`) backgrounds
- [ ] Light mode uses Crystal White / Neutral Fog backgrounds
- [ ] Aurora gradient visible in hero section
- [ ] `npm run docs:dev` renders without visual regressions

## Missing Assets
- Neue Montreal font files or CDN URL not provided -- implementer will need to source the font
- Actual logo SVG and favicon files not attached to the issue
