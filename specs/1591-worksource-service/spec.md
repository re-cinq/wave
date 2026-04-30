# Phase 2.2: WorkSourceService + bindings CRUD

**Issue:** [re-cinq/wave#1591](https://github.com/re-cinq/wave/issues/1591)
**Epic:** #1565 Phase 2 (work-source dispatch)
**Labels:** enhancement, ready-for-impl
**State:** OPEN
**Author:** nextlevelshit

## Goal

Implement `WorkSourceService` in `internal/worksource/` that manages bindings between work-sources (forges, schedules) and the pipelines they should trigger. CRUD over the `worksource_binding` table from PRE-5.

## Acceptance Criteria

- [ ] `internal/worksource/service.go` with interface:
  - `CreateBinding(ctx, BindingSpec) (BindingID, error)`
  - `ListBindings(ctx, filter) ([]BindingRecord, error)`
  - `GetBinding(ctx, BindingID) (BindingRecord, error)`
  - `UpdateBinding(ctx, BindingID, BindingSpec) error`
  - `DeleteBinding(ctx, BindingID) error`
  - `MatchBindings(ctx, work_item_ref) ([]BindingRecord, error)` — returns bindings that match a given work-item
- [ ] BindingSpec validates: forge type, repo pattern (glob or exact), label filter, pipeline name, trigger mode (`on-demand|on-label|on-open|scheduled`)
- [ ] Tests: CRUD round-trip + match-by-label + match-by-glob + invalid-spec rejection
- [ ] No webui yet (that is #2.3)

## Dependencies

- **PRE-5** — `worksource_binding` table (MERGED in `internal/state/migration_definitions.go` v32, `internal/state/worksource.go`)
- **#2.1** — `work_item_ref` shared schema. Per `docs/scope/onboarding-as-session-plan.md` §6 Phase 2, this lives at `internal/contract/schemas/shared/work_item_ref.json` and generalises `issue_ref` / `pr_ref`. **NOT YET LANDED** at the time of this plan; see plan §Risks.

## Open Questions / Missing Info

1. **Exact `BindingSpec` / `BindingRecord` shape** — the issue lists fields by name only. Plan §Architecture infers a concrete shape compatible with the existing `WorksourceBindingRecord` (state layer) and the trigger enum already in code.
2. **`work_item_ref` consumer contract** — schema not yet checked in. Plan adopts the shape published in onboarding-as-session-plan §5 (`forge`, `repo`, `kind`, `id`, `title`, `url`, plus optional `labels`, `state`) and gates real schema-bound validation on #2.1.

## References

- Original: https://github.com/re-cinq/wave/issues/1591
- ADR-010 (pipeline IO protocol)
- ADR-016 (work-source-centric webui IA)
- `docs/scope/onboarding-as-session-plan.md` §4.2, §6 Phase 2
- Existing infra: `internal/state/worksource.go`, `internal/state/migration_definitions.go` (v32)
