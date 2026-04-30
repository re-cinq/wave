# Implementation Plan — work_item_ref shared schema

## 1. Objective

Add a canonical `work_item_ref` JSON schema (draft-07) to the embedded shared-schemas registry so future Phase 2 work (`WorkSourceService`, webui board, dispatch wiring) has a single, validated lingua franca for forge issues, PRs, scheduled jobs, and manual triggers.

## 2. Approach

Mirror the existing `issue_ref.json` / `pr_ref.json` pattern in `internal/contract/schemas/shared/`. The registry auto-loads any `*.json` file in that directory via `go:embed`, so the only code change is updating the canonical-types list in `registry_test.go` and adding a new validation test that exercises one fixture per `source` enum value.

Conditional-required fields (`forge_host`, `owner`, `repo`, `number` only required when `source` is a forge) are expressed via JSON Schema `allOf` + `if/then`. This keeps schedule/manual entries valid without giving forge entries a soft contract.

## 3. File Mapping

### Created

| Path | Purpose |
|---|---|
| `internal/contract/schemas/shared/work_item_ref.json` | Canonical schema (draft-07, embedded) |
| `internal/contract/schemas/shared/work_item_ref_test.go` | Loads schema + validates one sample doc per `source` enum value (github, gitea, gitlab, bitbucket, schedule, manual) |
| `specs/1590-work-item-ref-schema/spec.md` | Issue + acceptance criteria capture (this PR) |
| `specs/1590-work-item-ref-schema/plan.md` | This file |
| `specs/1590-work-item-ref-schema/tasks.md` | Phased work breakdown |

### Modified

| Path | Change |
|---|---|
| `internal/contract/schemas/shared/registry_test.go` | Add `"work_item_ref"` to the canonical-types list in `TestRegistryContainsCanonicalTypes`; bump the `len(names) < 7` floor to 8 in `TestNamesSorted` |
| `docs/adr/010-pipeline-io-protocol.md` | Append `work_item_ref` to the registered shared types table (if such table exists; otherwise add a short "Update history" entry) |
| `docs/scope/onboarding-as-session-plan.md` | Tick Phase 2.1 row; replace inline draft snippet (lines 392–404) with a link to the canonical file or a note that the shipped shape differs |

### Not modified

- `.agents/contracts/` and `internal/defaults/embedfs/contracts/` — these hold step-output schemas (validated by `sync_test.go`), not pipeline-I/O typed schemas. Shared-schemas use a separate registry (`internal/contract/schemas/shared/registry.go`). Skipping this avoids creating a second source of truth.

## 4. Architecture Decisions

### D1 — Use `source` discriminator (not `forge`)

`issue_ref.json` uses `forge` because every entry *is* a forge entry. `work_item_ref` must also represent `schedule` and `manual` triggers, where "forge" is a category error. Using `source` (per acceptance criteria) cleanly accommodates both.

### D2 — Conditional required fields via `if/then/else`

Acceptance criteria lists `forge_host`, `owner`, `repo`, `number` under "required fields" but those only make sense for forge sources. Plan: declare them in `properties`, mark `source`/`url`/`title`/`state`/`created_at` unconditionally required, then use:

```json
"allOf": [{
  "if": { "properties": { "source": { "enum": ["github","gitea","gitlab","bitbucket"] } } },
  "then": { "required": ["forge_host","owner","repo"] }
}]
```

`number` is *not* required even for forge sources — generic forge "tasks" (Gitea task lists, GitLab work items) may not carry one. We define it as `integer, minimum: 1` when present.

### D3 — Reject the draft-snippet shape from `onboarding-as-session-plan.md:392`

Earlier draft used `{ forge, repo, kind, id }`. Acceptance criteria is the authoritative shape. We replace the draft snippet rather than carry both.

### D4 — `state` enum: `open | closed | merged`

Per acceptance criteria. `merged` is PR-only but we keep it on the parent enum (no per-source narrowing) — webui filters can layer on top.

### D5 — `additionalProperties: false`

Match `issue_ref.json` strictness. Future extensions (`assignees`, `milestone`, `body`, `priority`) get added explicitly via subsequent PRs, not silently accepted.

### D6 — `created_at` is RFC 3339 string

`"type": "string", "format": "date-time"`. Aligns with how StateStore stores ISO timestamps. No native JSON Date type.

### D7 — `labels` is `array of string`

Each entry `minLength: 1`. No requirement that the array itself be non-empty (manual/schedule sources may have none).

### D8 — Schema `$id` and title

`"$id": "wave://shared/work_item_ref"`, `"title": "WorkItemRef"`. Matches existing conventions.

## 5. Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Acceptance criteria's `if/then` requirement isn't what stakeholder expects ("required fields" interpreted strictly) | Med | Plan text + ADR update explicitly call out the conditional shape; reviewer can request strict-required if D2 is unwanted |
| `.agents/contracts/` copy is wanted after all | Low | Trivial follow-up; `sync_test.go` will guide. Note in PR description |
| `TestNamesSorted` floor (currently `< 7`) breaks if future PRs lower it before this lands | Low | Bump to 8 in same PR |
| ADR-010 doesn't have a "registered types" table to update | Low | Plan B is a "Schema additions" footer; verified at edit time |
| Draft snippet in onboarding-plan.md is referenced elsewhere | Low | grep before edit; replace with a link to canonical file |

## 6. Testing Strategy

### Unit (mandatory)

- `internal/contract/schemas/shared/work_item_ref_test.go` — for each of the six `source` enum values, marshal a representative fixture and call `gojsonschema.Validate` (or whatever validator is already used by other shared-schema tests; check `internal/contract/schemas/shared/` neighbouring tests if any) against the embedded schema. Both the forge and non-forge branches of the conditional must pass.
- Negative test: a doc with `source: "github"` but missing `forge_host` must fail validation.
- Negative test: unknown `source` value rejected.
- Negative test: extra property rejected (`additionalProperties: false`).

### Registry (mandatory)

- `registry_test.go` already iterates a canonical list — add `work_item_ref` so a missing/renamed schema fails CI.

### Manual / contract validation

- `make test` (full suite) — confirms no upstream consumer broke.
- `golangci-lint run ./...` — repo gate.
- No need to drive a pipeline run; this PR ships only a schema + test. Phase 2.2+ will validate end-to-end.

### Out of scope

- Schema-versioning story (no `$schema` migration plan needed; we ship draft-07 and that's it).
- Pipeline I/O protocol changes — none required.
