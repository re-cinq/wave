# Tasks

## Phase 1: Foundation — CSS Variables & Typography

- [ ] Task 1.1: Replace `--wave-*` color variables in `custom.css` with brand palette values (Quantum Blue, Pulse Blue, Soft Indigo) for both light and dark mode
- [ ] Task 1.2: Map VitePress `--vp-c-brand-1/2/3` to Quantum Blue / Pulse Blue / Soft Indigo in `custom.css`
- [ ] Task 1.3: Update dark mode overrides to use Midnight Navy (`#0F1F49`) backgrounds
- [ ] Task 1.4: Add Neue Montreal `@font-face` declarations (self-hosted WOFF2 in `docs/public/fonts/`) and update `--vp-font-family-base` to use Neue Montreal
- [ ] Task 1.5: Update `config.ts` font preconnect links — remove Google Fonts preconnect for Inter (no longer needed), keep JetBrains Mono

## Phase 2: Component Styles

- [ ] Task 2.1: Update `components.css` — hero gradient to Aurora (`#E4E6FD` → `#5664F4` → `#8F96F6`), feature card border-radius to 8-12px, card backgrounds to Nebula Light [P]
- [ ] Task 2.2: Update `HeroSection.vue` terminal colors — adjust terminal background/text to harmonize with Midnight Navy palette [P]
- [ ] Task 2.3: Update `TerminalOutput.vue` and `TerminalPreview.vue` terminal backgrounds to align with brand dark colors [P]
- [ ] Task 2.4: Verify trust/status color fallbacks in `TrustSignals.vue`, `PermissionMatrix.vue`, `YamlPlayground.vue`, `UseCaseGallery.vue` — keep functional green/amber/red, ensure contrast against new backgrounds [P]
- [ ] Task 2.5: Update `CopyButton.vue` hardcoded success green to use CSS variable [P]
- [ ] Task 2.6: Update `PipelineVisualizer.vue` persona colors to harmonize with brand palette [P]

## Phase 3: Assets

- [ ] Task 3.1: Replace `docs/public/logo.svg` with re:cinq leaf logomark (blue, brackets/slashes inside, Quantum Blue background)
- [ ] Task 3.2: Replace `docs/public/favicon.ico` with simplified re:cinq leaf icon
- [ ] Task 3.3: Update `docs/index.md` hero section — remove any hardcoded color references that conflict with brand

## Phase 4: Validation

- [ ] Task 4.1: Grep for legacy hex values (`#4f46e5`, `#4338ca`, `#06b6d4`, `#8b5cf6`) — confirm zero matches in CSS/Vue files
- [ ] Task 4.2: Run `npm run docs:build` to confirm no build errors
- [ ] Task 4.3: Visual spot-check — light mode, dark mode, hero gradient, feature cards, terminal preview
