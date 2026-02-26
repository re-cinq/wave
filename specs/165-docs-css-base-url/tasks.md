# Tasks

## Phase 1: Core Fix

- [X] Task 1.1: Add `base: '/wave/'` property to `defineConfig()` in `docs/.vitepress/config.ts`
- [X] Task 1.2: Update OG image path in `head` array from `'/og-image.png'` to `'/wave/og-image.png'`
- [X] Task 1.3: Update favicon href in `head` array from `'/favicon.svg'` to `'/wave/favicon.svg'`

## Phase 2: Validation

- [X] Task 2.1: Run VitePress build (`npm run build` in `docs/`) and verify it completes without errors
- [X] Task 2.2: Inspect generated HTML in `docs/.vitepress/dist/index.html` to confirm CSS/JS paths include `/wave/` prefix

## Phase 3: Polish

- [X] Task 3.1: Commit changes with conventional commit message `fix(docs): add base URL for GitHub Pages subpath hosting`
