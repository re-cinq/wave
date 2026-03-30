# Tasks

## Phase 1: Brand Token Foundation
- [ ] Task 1.1: Add brand color tokens to `:root` in style.css — define `--brand-midnight-navy`, `--brand-crystal-white`, `--brand-quantum-blue`, `--brand-aurora-gradient`, `--brand-pulse-blue`, `--brand-soft-indigo`, `--brand-nebula-light`, `--brand-neutral-fog` as CSS custom properties
- [ ] Task 1.2: Add Neue Montreal font loading — add `@font-face` declarations or CDN `@import` at top of style.css; update `--font-sans` to prioritize "Neue Montreal"

## Phase 2: Dark Mode Color Migration
- [ ] Task 2.1: Update `:root` (dark mode default) color block — replace `--wave-primary` (#818cf8 -> #5664F4), `--wave-primary-dark` (#6366f1 -> #0014EB), `--wave-accent` (#8b5cf6 -> #8F96F6), `--color-bg` (#0d1117 -> #0F1F49), `--color-bg-secondary`, `--color-bg-tertiary`, `--color-border`, `--color-border-light`, `--color-btn-primary` (#4f46e5 -> #0014EB), `--color-btn-primary-hover`, `--color-link-hover` with brand-derived values
- [ ] Task 2.2: Verify/adjust dark mode text colors and semantic status colors against Midnight Navy background for WCAG AA contrast

## Phase 3: Light Mode Color Migration [P]
- [ ] Task 3.1: Update `[data-theme="light"]` color block — `--wave-primary` (#4f46e5 -> #0014EB), `--color-bg-secondary` (#f8fafc -> #F2F2F7), `--color-bg-tertiary` (#f1f5f9 -> #E6E8FD), `--color-btn-primary` (#4f46e5 -> #0014EB), `--color-running` (#4f46e5 -> #5664F4) with brand values
- [ ] Task 3.2: Update `@media (prefers-color-scheme: light)` block — mirror the same values as `[data-theme="light"]` to keep system-preference and explicit-toggle in sync

## Phase 4: Hardcoded Color Cleanup [P]
- [ ] Task 4.1: Update button shadow rgba values — replace `rgba(79, 70, 229, 0.25)` and `rgba(79, 70, 229, 0.35)` in `.btn-primary` with brand-appropriate rgba values based on Quantum Blue
- [ ] Task 4.2: Update `.log-search input:focus` box-shadow — replace `rgba(79, 70, 229, 0.15)` with brand blue rgba
- [ ] Task 4.3: Update `.nav-brand a` gradient — replace `var(--wave-primary) -> var(--wave-accent)` gradient with Aurora gradient (`#E4E6FD -> #5664F4 -> #8F96F6`)

## Phase 5: Logo and Layout
- [ ] Task 5.1: Replace inline SVG logo in `layout.html` — swap the `<svg class="nav-logo" viewBox="0 0 200 200" ...>` with re:cinq leaf logomark SVG
- [ ] Task 5.2: Update navbar brand text — change "Wave" to include re:cinq branding if appropriate
- [ ] Task 5.3: Add font preload link to layout.html `<head>` if using self-hosted Neue Montreal

## Phase 6: Polish and Gradient Accents
- [ ] Task 6.1: Update `.progress-bar` fill gradient — replace `var(--wave-primary) -> var(--wave-accent)` with Aurora gradient
- [ ] Task 6.2: Update `.btn-primary` gradient — replace `var(--color-btn-primary) -> var(--wave-accent)` with brand gradient
- [ ] Task 6.3: Audit remaining hardcoded hex colors in CSS — verify functional colors (fork green, rewind amber, ANSI terminal, status-hook badges) still have sufficient contrast against new backgrounds

## Phase 7: Validation
- [ ] Task 7.1: Build verification — run `go build ./cmd/wave` to confirm embedded static assets compile
- [ ] Task 7.2: Visual verification — confirm `wave serve` renders correctly in browser with dark/light mode toggle
- [ ] Task 7.3: Acceptance criteria checklist — verify all 8 acceptance criteria from the issue
