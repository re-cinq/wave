# Infinite scroll doesn't trigger on large screens

## Issue Metadata

- **Issue Number**: 1042
- **Title**: Infinite scroll doesn't trigger on large screens
- **Repository**: re-cinq/wave
- **URL**: https://github.com/re-cinq/wave/issues/1042
- **Labels**: []
- **State**: OPEN
- **Author**: nextlevelshit

## Summary

On large/high-resolution screens, the overview pages (runs, pipelines) don't load additional content because the initial viewport is tall enough that no scrollbar appears. The infinite scroll loader only triggers on scroll events, so if all visible content fits without scrolling, no new data is ever fetched.

## Steps to Reproduce

1. Open a runs/pipelines overview page on a large or high-res monitor
2. Observe that only the initial batch of items loads
3. No scroll event fires because the content fits the viewport — the infinite scroll trigger is never reached

## Expected Behavior

The page should detect that the scroll container is not overflowing and automatically load more content until the container overflows or all data is loaded.

## Suggested Fix

After initial load (and after each batch append), check if `scrollHeight <= clientHeight`. If so, trigger the next fetch immediately instead of waiting for a scroll event.

## Acceptance Criteria

1. On large screens where initial content fits viewport, infinite scroll triggers automatically
2. After loading a batch, if content still fits, another batch is fetched
3. The behavior works consistently across runs, issues, and PRs overview pages
4. No infinite loading loops occur when all data has been loaded
5. The existing scroll-based trigger continues to work for smaller viewports
