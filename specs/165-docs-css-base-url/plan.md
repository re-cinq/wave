# Implementation Plan: fix(docs) CSS Base URL

## Objective

Add the `base: '/wave/'` property to the VitePress configuration so that all generated asset paths (CSS, JS, images) include the `/wave/` subpath prefix required for GitHub Pages hosting under `re-cinq.github.io/wave/`.

## Approach

This is a trivial configuration fix. The primary change is adding `base: '/wave/'` to the `defineConfig()` call in `docs/.vitepress/config.ts`. VitePress automatically rewrites all internal links and asset URLs when `base` is set, so most paths will be handled automatically.

However, the `head` array contains hardcoded absolute paths for the OG image and favicon that need manual updating since they are raw HTML meta/link tags injected directly into the page head — VitePress does not rewrite these automatically.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `docs/.vitepress/config.ts` | modify | Add `base: '/wave/'` to `defineConfig()`, update hardcoded paths in `head` array |

## Architecture Decisions

1. **Use VitePress `base` property** rather than modifying build scripts or the GitHub Actions workflow. This is the idiomatic VitePress approach for subpath deployment.
2. **Update `head` paths manually** for OG image and favicon since VitePress does not auto-rewrite raw HTML injected via the `head` config array.
3. **No workflow changes needed** — the existing `docs.yml` workflow uses `actions/configure-pages@v4` which handles GitHub Pages deployment correctly regardless of subpath. The VitePress `base` property is sufficient.

## Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Local dev broken by base path | Low | VitePress `dev` server handles `base` correctly; `npm run dev` will still work |
| Missed hardcoded paths elsewhere | Low | Grep for absolute `/` paths in docs config; VitePress handles internal links automatically |

## Testing Strategy

1. **Build verification**: Run `npm run build` in `docs/` to confirm VitePress builds successfully with the new `base` setting
2. **Manual inspection**: Inspect the generated HTML in `docs/.vitepress/dist/` to verify asset paths include `/wave/` prefix
3. **No Go tests affected**: This change is docs-only (TypeScript config) — `go test ./...` is not required
