# View Architecture Quality Checklist

**Feature**: #259 — TUI Alternative Master-Detail Views  
**Date**: 2026-03-06  
**Scope**: Architectural quality of the multi-view system, data providers, and message routing

## Completeness

- [ ] CHK030 - Are the data provider interface boundaries fully specified — does each provider define all methods needed for both initial load and incremental updates? [Completeness]
- [ ] CHK031 - Is the ContentModel's message routing exhaustively defined — are all possible message types mapped to their target view models without ambiguity? [Completeness]
- [ ] CHK032 - Does the spec define the lifecycle of view models — creation, size propagation, data loading, destruction (if any), and memory cleanup? [Completeness]
- [ ] CHK033 - Are the dependencies for DefaultPersonaDataProvider fully enumerated — manifest, state store, pipelines dir — with behavior defined for each nullable dependency? [Completeness]
- [ ] CHK034 - Does the spec define how `SetSize()` propagates to all view models, including those not yet lazy-initialized (deferred dimensions)? [Completeness]

## Clarity

- [ ] CHK035 - Is it clear which messages are "global" (routed to all views) vs "view-scoped" (routed only to the active view)? [Clarity]
- [ ] CHK036 - Is the lazy initialization contract unambiguous — exactly which operations happen in `cycleView()` and which are deferred to `Init()`? [Clarity]
- [ ] CHK037 - Is the distinction between `PersonaDataProvider.FetchPersonas()` (manifest scan) and `PersonaDataProvider.FetchPersonaStats()` (DB query) clearly motivated in the spec? [Clarity]
- [ ] CHK038 - Is the scope of "pipeline YAML scanning" clearly defined — does it scan `pipelinesDir` files, or does it use the already-parsed `manifest.Pipelines` map? [Clarity]

## Consistency

- [ ] CHK039 - Do all 4 alternative view data providers use the same pipeline-scanning approach (manifest object vs YAML files) for computing pipeline usage? [Consistency]
- [ ] CHK040 - Is the error handling strategy consistent — do all `Fetch*` methods return `(result, error)` tuples and are errors surfaced the same way in all list models? [Consistency]
- [ ] CHK041 - Are the model field naming conventions consistent across all list models (cursor, navigable, filtering, filterInput, scrollOffset, focused)? [Consistency]
- [ ] CHK042 - Does the spec ensure that `ContentProviders` struct injection follows the same dependency-injection pattern used by `RunTUI`'s existing `TUIDeps`? [Consistency]

## Coverage

- [ ] CHK043 - Does the spec address thread safety for health check results arriving concurrently via tea.Cmd — is Bubble Tea's single-goroutine Update() sufficient? [Coverage]
- [ ] CHK044 - Are the integration points between alternative views and existing pipeline-specific features (launch, cancel, live output) clearly scoped as non-applicable? [Coverage]
- [ ] CHK045 - Does the spec address how `WindowSizeMsg` is handled during view transitions — can a resize during `cycleView()` cause dimension mismatch? [Coverage]
- [ ] CHK046 - Is there a requirement for the order in which async data messages arrive — can a `PersonaStatsMsg` arrive before `PersonaDataMsg` and if so what happens? [Coverage]
- [ ] CHK047 - Does the spec address what happens if `filepath.Glob()` for skills returns an error (permission denied, broken symlink) vs simply no matches? [Coverage]
