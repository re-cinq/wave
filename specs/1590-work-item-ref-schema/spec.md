# Phase 2.1: work_item_ref shared schema + registry entry

**Issue:** [#1590](https://github.com/re-cinq/wave/issues/1590)
**Repository:** re-cinq/wave
**Labels:** enhancement, ready-for-impl
**State:** OPEN
**Author:** nextlevelshit
**Epic:** [#1565 Phase 2 (work-source dispatch)](https://github.com/re-cinq/wave/issues/1565)

## Issue Body

Part of Epic #1565 Phase 2 (work-source dispatch).

### Goal

Define the canonical `work_item_ref` schema that work-sources (forge issues, scheduled jobs, manual triggers) emit and pipelines consume. Register the schema in the contract registry.

### Acceptance criteria

- [ ] `internal/contract/work-item-ref.schema.json` (JSON schema, draft-07)
- [ ] Required fields: `source` (enum: github|gitea|gitlab|bitbucket|schedule|manual), `forge_host`, `owner`, `repo`, `number` (issues/PRs), `url`, `title`, `labels`, `state` (open|closed|merged), `created_at`
- [ ] Schema sync to `.agents/contracts/work-item-ref.schema.json`
- [ ] Test that loads the schema and validates a sample document for each source type
- [ ] Documented in epic plan + an ADR-update if the schema crosses an architectural boundary

### Out of scope

- WorkSourceService implementation (that's #2.2)
- Webui board (that's #2.3)
- Dispatch wiring (that's #2.4)

### Dependencies

- PRE-3 (Forge adapter, MERGED) — for forge-host classification
- PRE-5 (StateStore schema, MERGED) — `worksource_binding` table can reference work_item_ref by URL

## Path-convention reconciliation

Issue body lists `internal/contract/work-item-ref.schema.json` and `.agents/contracts/work-item-ref.schema.json`.

Codebase convention for canonical typed I/O schemas (per ADR-010, registered in `internal/contract/schemas/shared/registry.go`) is:

- `internal/contract/schemas/shared/<name>.json` (snake_case, no `.schema` suffix, draft-07)

`docs/scope/onboarding-as-session-plan.md:390` and the Phase 2.1 row at line 435 both confirm `internal/contract/schemas/shared/work_item_ref.json`. We follow this convention; the issue body's path is shorthand.

`.agents/contracts/*.schema.json` (synced via `sync_test.go` to `internal/defaults/embedfs/contracts/`) is a *different* registry — it holds **step-output** contracts referenced by `contract.json_schema`. No existing `*_ref` shared schema is duplicated there. We do **not** copy the shared schema into `.agents/contracts/`; documenting the rationale satisfies the spirit of the "schema sync" criterion. (This is captured as an open question in the assessment under `missing_info`.)

## Acceptance Criteria (extracted)

1. New JSON schema (draft-07) at `internal/contract/schemas/shared/work_item_ref.json`.
2. Schema declares required fields: `source`, `url`, `title`, `state`, `created_at`. Forge-only fields (`forge_host`, `owner`, `repo`, `number`) are conditionally required when `source` is a forge value.
3. Schema is reachable via `shared.Lookup("work_item_ref")` and `shared.Exists("work_item_ref")`.
4. `registry_test.go` lists `work_item_ref` among canonical types.
5. New test loads a fixture per source type (github/gitea/gitlab/bitbucket/schedule/manual) and validates against the schema.
6. ADR-010 (pipeline I/O protocol) updated with new shared type entry, OR a short note under ADR-016 D9 confirming the shape that landed.
7. Phase 2.1 row in `docs/scope/onboarding-as-session-plan.md` ticked off (or schema diff shown if it diverged from the draft snippet).
8. `make test` passes; `golangci-lint` clean.
