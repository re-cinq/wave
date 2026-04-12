# Implementation Plan: Infinite Scroll Fix for Large Screens

## Objective

Fix infinite scroll on large/high-resolution screens where initial content fits the viewport without scrolling. The current implementation relies solely on scroll events, which never fire when content fits without scrolling.

## Approach

Modify the infinite scroll logic in the three affected overview pages (runs, issues, PRs) to:
1. After initial load, check if content fits in viewport (`scrollHeight <= clientHeight`)
2. After each batch append, re-check if content still fits and trigger fetch if needed
3. Keep existing scroll-based trigger as fallback for normal viewport sizes

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/webui/templates/runs.html` | Modify | Update infinite scroll JS to check for viewport overflow after load and batch append |
| `internal/webui/templates/issues.html` | Modify | Same infinite scroll fix pattern |
| `internal/webui/templates/prs.html` | Modify | Same infinite scroll fix pattern |

## Architecture Decisions

1. **Non-breaking change**: Keep existing scroll listener and initial timeout check intact
2. **Recursive check after append**: After loading a batch, re-check `scrollHeight <= clientHeight` and trigger another fetch if conditions are met
3. **Throttle protection**: Existing `loading` flag prevents concurrent fetches
4. **Loader element removal**: When no more pages exist, loader is removed; this breaks the infinite loop naturally

## Risks

| Risk | Mitigation |
|------|------------|
| Infinite loading loop | `nextUrl` becomes falsy when no more pages, breaking the loop |
| Performance on very large datasets | Backend pagination limits page size (typically 20-50 items) |
| Race conditions | `loading` flag guards against concurrent fetches |

## Testing Strategy

1. **Manual testing**: Open runs/issues/PRs page on large monitor without scrollbar
2. **Automated testing**: Consider adding integration test with mocked DOM for scroll behavior
3. **Regression testing**: Existing scroll behavior remains unchanged
