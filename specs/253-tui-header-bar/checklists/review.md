# Requirements Quality Review: TUI Header Bar

**Feature**: 253-tui-header-bar | **Date**: 2026-03-05

## Completeness

- [ ] CHK001 - Are all metadata fields listed in FR-005 (health, repo, branch, remote, dirty, issues, commit) traced to a specific data source in FR-007? [Completeness]
- [ ] CHK002 - Does the spec define what "abbreviated commit hash" means numerically (e.g., 7 characters) or is this left ambiguous? [Completeness]
- [ ] CHK003 - Does FR-006 specify where the `BranchName` field is populated in the pipeline lifecycle, or only where it is consumed? [Completeness]
- [ ] CHK004 - Are fallback/error states defined for every metadata column (not just branch and project)? [Completeness]
- [ ] CHK005 - Does the spec define the visual appearance of the health status indicators (symbols, colors) or only the semantic states? [Completeness]
- [ ] CHK006 - Are the placeholder values (pre-load state) defined for every metadata field, not just generically as "…"? [Completeness]
- [ ] CHK007 - Does the spec define the header's fixed height (3 lines) independently of the logo height, or does it assume they are always equal? [Completeness]
- [ ] CHK008 - Is the git refresh timer interval (30s in FR-013) justified with a rationale, or is it an arbitrary choice that may need tuning? [Completeness]

## Clarity

- [ ] CHK009 - Does FR-003 unambiguously define the animation palette? Are color values specified concretely (e.g., ANSI codes or lipgloss values) or only by name (cyan, blue, magenta)? [Clarity]
- [ ] CHK010 - Is the term "running pipeline" consistently defined across all requirements? Does `runningCount > 0` refer to pipelines in "running" state in the state DB, or pipelines with active adapter subprocesses? [Clarity]
- [ ] CHK011 - Does "progressively hidden" in FR-009 mean columns are hidden one at a time as width decreases, or all at once below a threshold? [Clarity]
- [ ] CHK012 - Is the column priority order in FR-009 a strict order (always hide lowest priority first) or a guideline? Does the spec define specific width breakpoints? [Clarity]
- [ ] CHK013 - Is "clean/dirty state" a binary indicator or does it distinguish between staged, unstaged, and untracked changes? [Clarity]
- [ ] CHK014 - Does "repository name" mean the full "owner/repo" format or just the repo name? Is this consistently specified across FR-005 and FR-007? [Clarity]
- [ ] CHK015 - Does FR-012's "placeholder values" requirement specify the visual appearance of placeholders (e.g., "…" vs spinner vs empty)? [Clarity]

## Consistency

- [ ] CHK016 - Are the 200ms animation tick (FR-003) and 16ms render budget (SC-001) compatible? Does the spec address whether animation ticks trigger re-renders? [Consistency]
- [ ] CHK017 - Does the column priority list in FR-009 match the column priority table in the data model? Are there any ordering conflicts? [Consistency]
- [ ] CHK018 - Is the NO_COLOR behavior in FR-008 consistent with the logo animation requirement in FR-003? When NO_COLOR is set, is the animation effectively invisible (no color change) or is it explicitly disabled? [Consistency]
- [ ] CHK019 - Do US1 acceptance scenarios and FR-005 list the same metadata fields? Is "open issues count" in FR-005 reflected in the US1 scenarios? [Consistency]
- [ ] CHK020 - Is the health status definition in FR-011 ("no pipelines have failed") consistent with the HealthStatus enum (OK/Warn/Err)? What constitutes a "soft failure" vs "hard failure"? [Consistency]
- [ ] CHK021 - Does C-004's resolution (BranchName on RunRecord) align with FR-006's requirement for dynamic branch display? Is the data flow from executor→state→header fully specified? [Consistency]

## Coverage

- [ ] CHK022 - Is there a requirement addressing what happens when multiple finished pipelines are selected in sequence? Does the override branch update correctly each time? [Coverage]
- [ ] CHK023 - Does the spec define behavior when the terminal width drops below 80 columns (the stated minimum)? [Coverage]
- [ ] CHK024 - Is there a requirement for header behavior during TUI shutdown or cleanup? (Edge case 5 mentions animation tick, but what about metadata refresh timers?) [Coverage]
- [ ] CHK025 - Does the spec address the case where `wave.yaml` has no `metadata.repo` field AND git remote parsing also fails? [Coverage]
- [ ] CHK026 - Is there a defined behavior for when the state DB is unavailable or corrupted? How does health status degrade? [Coverage]
- [ ] CHK027 - Does the spec address concurrent metadata updates (e.g., periodic git refresh and event-driven refresh arriving simultaneously)? [Coverage]
- [ ] CHK028 - Is there a requirement for keyboard/focus interaction with the header itself, or is it explicitly non-interactive? [Coverage]
- [ ] CHK029 - Does the spec address how the header integrates with the existing TUI layout (3-row scaffold from #252)? Is the integration boundary defined? [Coverage]
