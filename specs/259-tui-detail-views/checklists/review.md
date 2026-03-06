# Requirements Quality Review Checklist

**Feature**: #259 — TUI Alternative Master-Detail Views  
**Date**: 2026-03-06  
**Scope**: Overall requirements quality validation across spec.md, plan.md, tasks.md

## Completeness

- [ ] CHK001 - Are all 5 views (Pipelines, Personas, Contracts, Skills, Health) fully specified with both left-pane and right-pane content requirements? [Completeness]
- [ ] CHK002 - Does the spec define what happens when view-specific data sources are unavailable (nil store, missing files, parse errors) for each of the 4 alternative views? [Completeness]
- [ ] CHK003 - Are empty-state requirements defined for all 4 alternative views (no personas, no contracts, no skills, no health data)? [Completeness]
- [ ] CHK004 - Does the spec cover terminal resize behavior for all alternative views, not just the pipeline view? [Completeness]
- [ ] CHK005 - Are accessibility requirements addressed — does the spec define minimum contrast, screen-reader hints, or color-blind-safe status indicators beyond the color codes? [Completeness]
- [ ] CHK006 - Is the initial loading state defined for all lazy-initialized views (what does the user see between Tab press and data arrival)? [Completeness]
- [ ] CHK007 - Are keyboard shortcut conflicts fully mapped — does a conflict matrix exist for all key bindings across all 5 views and all focus states? [Completeness]
- [ ] CHK008 - Does the spec define the maximum number of items each list view can reasonably handle without performance degradation? [Completeness]

## Clarity

- [ ] CHK009 - Is the `ViewType` cycle order (Pipelines → Personas → Contracts → Skills → Health) explicitly justified, or could an alternative ordering better match user workflows? [Clarity]
- [ ] CHK010 - Is the "lazy initialization" trigger clearly defined — does it happen on first Tab press to the view, or on first rendering of the view? [Clarity]
- [ ] CHK011 - Is the persona stats aggregation boundary clear — does "all performance_metric rows" mean all-time or a bounded window (last N days, last M runs)? [Clarity]
- [ ] CHK012 - Are the schema preview truncation rules for Contracts view clearly specified — is "first ~30 lines" a hard limit, a soft guideline, or configurable? [Clarity]
- [ ] CHK013 - Is it clear how `ContentModel.cycleView()` handles the transition atomically — can a partial state (new view selected but old focus/dimming) flash on screen? [Clarity]
- [ ] CHK014 - Does the spec clearly distinguish between "no data available" (loading) and "data loaded but empty" (no items) for each view's list pane? [Clarity]

## Consistency

- [ ] CHK015 - Is the navigation model (↑/↓, Enter, Esc, /) consistent across all 4 alternative views and the existing pipeline view? [Consistency]
- [ ] CHK016 - Do all alternative views use the same cursor indicator (`▶ `) and selection highlighting as the pipeline list? [Consistency]
- [ ] CHK017 - Are the status bar hint formats consistent across all views — same separator, same key label style, same ordering? [Consistency]
- [ ] CHK018 - Is the data provider interface pattern consistent — do all 4 providers follow the same error-returning convention and naming scheme? [Consistency]
- [ ] CHK019 - Is the message naming convention consistent — do all views follow the same `<Entity>DataMsg`, `<Entity>SelectedMsg` pattern? [Consistency]
- [ ] CHK020 - Are the right-pane rendering styles (bold cyan titles, dimmed previews, section headers) consistent across all detail views? [Consistency]
- [ ] CHK021 - Is the pipeline usage display format ("pipeline / step" pairs) consistent between Personas, Contracts, and Skills views? [Consistency]

## Coverage

- [ ] CHK022 - Does the spec cover the interaction between health checks and the header's existing `HealthStatus` aggregation — can they conflict or show different statuses? [Coverage]
- [ ] CHK023 - Are error recovery paths defined for each async operation (persona stats fetch failure, health check timeout, filesystem glob error)? [Coverage]
- [ ] CHK024 - Does the spec address how Tab cycling interacts with the live output streaming state (`liveOutputActive`) added in #257? [Coverage]
- [ ] CHK025 - Does the spec address how Tab cycling interacts with the finished pipeline actions (chat, branch checkout, diff) added in #258? [Coverage]
- [ ] CHK026 - Are the `q` and `Ctrl-C` quit behaviors specified for all combinations of view × focus state × active mode? [Coverage]
- [ ] CHK027 - Does the spec define whether background data refresh tickers (if any) in alternative views should stop when the view is not active, or continue? [Coverage]
- [ ] CHK028 - Is there a requirement for how stale health check results are indicated — does the "last checked" timestamp alone suffice, or should results age out? [Coverage]
- [ ] CHK029 - Does the spec cover what happens when the manifest changes on disk while the TUI is running — do alternative views reflect updates or require restart? [Coverage]
