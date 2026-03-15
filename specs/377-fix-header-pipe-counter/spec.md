# fix(tui): rethink header bar — remove misleading pipe counter and audit remaining elements

**Issue**: [#377](https://github.com/re-cinq/wave/issues/377)
**Labels**: enhancement, ux, display
**Author**: nextlevelshit
**Complexity**: simple

## Summary

The TUI header bar displays a misleading "Pipes: 0/96" counter that reads as "used/capacity" but actually shows `running_count/total_entries` (running + available + finished pipeline items). Since the denominator is just a sum of all known pipeline entries — not an enforced limit — the `X/Y` format confuses users into thinking there is a hard pipeline cap. The remaining header elements should also be audited for usefulness.

## Problem

The "Pipes: 0/96" indicator in the header bar is misleading because:
- The `X/Y` format naturally reads as "used out of maximum capacity"
- The denominator (`TotalPipes`) is actually `len(running) + len(available) + len(finished)` — a count of all known pipeline entries, not a concurrency limit
- There is no enforced pipeline-level concurrency limit in Wave (the manifest's `max_concurrent_workers` and `max_concurrency` settings apply to step-level execution within a pipeline, not to how many pipelines can run simultaneously)
- Users can run the same pipeline multiple times concurrently with no cap

The rendering logic is in `renderPipesValue()` at `internal/tui/header.go:243-256`, which formats the value as `fmt.Sprintf("%d/%d", running, total)`.

A pipeline limit counter would only make sense if:
- A manifest setting existed to configure max concurrent pipeline runs
- Pipeline scheduling with queueing was implemented

## Requirements

- [ ] Remove or replace the misleading "Pipes: X/Y" counter from the header (specifically `renderPipesValue()` at `internal/tui/header.go:243`)
- [ ] Audit all remaining header elements for accuracy and usefulness (the current 3-row, 3-column metadata grid at lines 132-163)
- [ ] Replace removed elements with meaningful status information (e.g., just the active running count like "2 running", or system status)
- [ ] Ensure header content is consistent with actual system capabilities

## Header Audit: Current Elements

The header renders a 3-row x 3-column metadata grid (progressive disclosure by terminal width):

| Row | Col 1 (>=20 chars) | Col 2 (>=40 chars) | Col 3 (>=60 chars) |
|-----|---------------------|---------------------|---------------------|
| 1   | **Health**: OK/WARN/ERR | **GitHub/Project**: repo slug or project name | **Remote**: git remote name |
| 2   | **Pipes**: running/total | **Branch**: current or override branch | **Clean**: dirty indicator |
| 3   | **Steps**: step count | **Issues**: N open / [offline] | **Commit**: abbreviated hash |

**Accurate elements (STILL_VALID)**:
- Health, GitHub/Project, Remote, Branch, Clean, Steps, Issues, Commit — all display real, useful data fetched from git, manifest, and GitHub API.

**Misleading elements**:
- **Pipes: X/Y** — the `X/Y` format implies a capacity ratio. Should show just the running count or use a clearer format.

## Acceptance Criteria

- [ ] Header no longer displays a misleading pipeline limit counter
- [ ] All header elements display accurate, useful information
- [ ] Header audit findings are documented in the PR description
- [ ] TUI header tests updated to reflect new content (tests in `internal/tui/header_test.go`)

## Technical Notes

- **Header model**: `internal/tui/header.go` — `HeaderModel` struct, `View()` renders the 3-line metadata grid
- **Metadata types**: `internal/tui/header_metadata.go` — `HeaderMetadata` struct with `RunningCount`, `TotalPipes` fields
- **Messages**: `internal/tui/header_messages.go` — `RunningCountMsg{Count, TotalPipes}`
- **Provider**: `internal/tui/header_provider.go` — `DefaultMetadataProvider` fetches git/manifest/GitHub data
- **Logo animation**: `internal/tui/header_logo.go` — `LogoAnimator` with walking glow effect
- **Pipeline list emits count**: `internal/tui/pipeline_list.go:138-140` — computes `totalPipes` as sum of all entries
- **Tests**: `internal/tui/header_test.go` (56 tests), `internal/tui/header_provider_test.go` (11 tests)
- Related: TUI epic parent issue #251
- Step concurrency (PR #369, merged 2026-03-13) added `concurrency` property for parallel step agents — this is step-level, confirming there is still no pipeline-level concurrency limit
