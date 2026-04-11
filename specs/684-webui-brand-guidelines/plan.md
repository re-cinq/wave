# Implementation Plan — #684 webui brand guidelines

## Objective

Replace the generic indigo/purple palette, system fonts, and custom Wave SVG logo in the `wave serve` web dashboard with the re:cinq brand guidelines v1.0 color system, Neue Montreal typography, and leaf logomark.

## Approach

This is a CSS-heavy theming change with minimal structural HTML changes. The strategy:

1. **Define brand tokens as CSS custom properties** — add `--brand-*` tokens at `:root` level, then map existing `--wave-*` and `--color-*` variables to brand values
2. **Update three color blocks** — `:root` (dark default), `[data-theme="light"]`, and `@media (prefers-color-scheme: light)` all need parallel updates
3. **Replace logo SVG** — swap the inline `<svg viewBox="0 0 200 200">` in layout.html with a re:cinq leaf logomark SVG
4. **Add Neue Montreal font** — add `@font-face` declarations or CDN import; update `--font-sans`
5. **Audit hardcoded colors** — a few CSS rules use hardcoded hex values (e.g. `.btn-fork`, ANSI colors, status-hook badges); semantic/functional colors stay, but shadow/glow rgba values referencing old brand hex need updating
6. **Add aurora gradient** — create a utility class or update `.nav-brand` gradient to use the Aurora gradient stops

## File Mapping

### Modified files
| File | Changes |
|------|---------|
| `internal/webui/static/style.css` | Replace `:root`, `[data-theme="light"]`, and `@media (prefers-color-scheme: light)` color blocks with brand palette; add `@font-face` for Neue Montreal; update `--font-sans`; update navbar gradient; update button shadow rgba values; update `.progress-bar` gradient |
| `internal/webui/templates/layout.html` | Replace inline SVG logo with re:cinq leaf logomark; optionally update `<title>` from "Wave" to "re:cinq Wave"; add font preload `<link>` if using self-hosted fonts |

### No changes needed
| File | Reason |
|------|--------|
| `internal/webui/templates/*.html` (non-layout) | Audit confirmed: no hardcoded hex colors in templates — all use CSS variables |
| `internal/webui/static/app.js` | No color references |

## Architecture Decisions

1. **Keep existing variable names** — map `--wave-primary`, `--color-bg`, etc. to new brand values rather than renaming. This avoids touching every CSS rule that references them.

2. **Add `--brand-*` token layer** — define brand tokens once at `:root`, then use them in both dark/light theme blocks. This makes future brand updates a single-location change.

3. **Dark mode secondary/tertiary derivation** — Midnight Navy `#0F1F49` is the base. Derive secondary (`#162254`) and tertiary (`#1D2B60`) by lightening slightly rather than picking arbitrary values.

4. **Neue Montreal font loading** — use `@font-face` with self-hosted WOFF2 files for reliability. If font files aren't available, use a CDN import. Fallback chain: `"Neue Montreal", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif`.

5. **Functional/semantic colors preserved** — green (completed), red (failed), amber (pending), and ANSI terminal colors are NOT brand colors. They stay as-is. Only adjust if contrast ratio against new backgrounds drops below WCAG AA (4.5:1).

6. **Logo approach** — the re:cinq leaf logomark SVG is not provided in the issue. Options:
   - Check repo for existing brand assets
   - Use a minimal placeholder leaf SVG matching brand style
   - The implementer will need to source or create the SVG

## Risks

| Risk | Mitigation |
|------|-----------|
| Neue Montreal font files not available | Provide sensible fallback stack; font renders system sans-serif if unavailable |
| re:cinq leaf logomark SVG not in repo | Create a clean minimal leaf SVG matching brand guidelines aesthetic |
| Contrast issues with new dark background | Verify status colors and text remain legible against Midnight Navy; adjust if needed |
| `@media (prefers-color-scheme: light)` block falls out of sync | Structure CSS so light theme values are defined once and reused in both `[data-theme="light"]` and the media query |
| Button/card shadow rgba values still reference old brand hex | Audit all `rgba(79, 70, 229, ...)` and update to new brand blue |

## Testing Strategy

1. **Visual inspection** — `wave serve` in browser, toggle dark/light mode, verify all pages
2. **No Go tests affected** — this is a pure CSS/HTML change; webui handler tests don't validate colors
3. **Acceptance criteria checklist** — verify each criterion from the issue
4. **Contrast check** — spot-check text/background contrast ratios on key elements
5. **Build verification** — `go build ./cmd/wave` to confirm embedded assets still compile
