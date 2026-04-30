# Phase 1.5c: Tailwind vendored compile (replace CDN)

**Issue**: [re-cinq/wave#1601](https://github.com/re-cinq/wave/issues/1601)
**Labels**: enhancement, ready-for-impl, frontend
**State**: OPEN
**Author**: nextlevelshit

## Body

Part of Epic #1565 Phase 1.5.

### Goal

Replace Tailwind CDN `<script src="https://cdn.tailwindcss.com">` with a vendored compiled stylesheet. Avoids runtime CDN dep + offline-mode break before Phase D ship.

### Acceptance criteria

- [ ] `tailwindcss` CLI run at build time, output to `internal/webui/static/tailwind.css`
- [ ] Stylesheet embedded via `//go:embed`
- [ ] Templates reference embedded path, not CDN
- [ ] Build step documented in Makefile / docs/build.md
- [ ] No external CDN dependency at runtime

### Dependencies

- 1.5a Phase A (PR #1585 MERGED) — uses CDN currently
- Optionally after 1.5b
