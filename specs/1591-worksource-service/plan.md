# Implementation Plan — #1591 WorkSourceService + bindings CRUD

## 1. Objective

Add a domain service in `internal/worksource/` that wraps the existing `state.WorksourceStore` (PRE-5) with validated CRUD plus a `MatchBindings` query that selects bindings applicable to a given `WorkItemRef` (forge + repo + labels + state). No HTTP/webui yet — pure service + tests.

## 2. Approach

Thin domain layer on top of `internal/state`:

- Define value types (`BindingSpec`, `BindingRecord`, `BindingID`, `BindingFilter`, `WorkItemRef`) in `internal/worksource/types.go`.
- Define the `Service` interface and a `service` struct backed by `state.WorksourceStore` in `internal/worksource/service.go`.
- Translate between domain `BindingSpec` (typed: repo glob, label filter slice, trigger enum) and the storage shape (`WorksourceBindingRecord` with opaque JSON `selector`/`config`). Selector JSON shape: `{ "labels": [], "state": "open|closed|any", "kinds": [] }`.
- `MatchBindings` filters via `ListActiveBindings()` then in-Go: forge equality, `path.Match` glob on `repo`, label intersection, kind/state subset. Index already exists for forge+repo (active=1).
- Reject invalid specs at write time: empty forge, empty pipeline name, invalid trigger value, malformed glob (`path.Match("", "test")` returns error on bad pattern), non-positive ID on update/get/delete.
- Map "delete" to `DeactivateBinding` (state layer never hard-deletes; preserves run history per existing comment).

### Trigger naming

Issue uses dashed form (`on-demand`); state layer uses underscored form (`on_demand`). Service accepts the dashed external form, normalises to underscored before persisting, and renders dashed on the way out. One conversion table in `types.go`.

## 3. File Mapping

| Action | Path | Purpose |
|---|---|---|
| CREATE | `internal/worksource/doc.go` | Package doc — domain service for forge↔pipeline bindings |
| CREATE | `internal/worksource/types.go` | `BindingSpec`, `BindingRecord`, `BindingID`, `BindingFilter`, `WorkItemRef`, `Trigger` enum + dashed↔underscored converters, selector marshal/unmarshal |
| CREATE | `internal/worksource/service.go` | `Service` interface + `NewService(state.WorksourceStore) Service` + struct impl of all six methods |
| CREATE | `internal/worksource/validate.go` | `validateSpec(BindingSpec) error` — central rejection rules (forge, pipeline, trigger, repo glob, label values) |
| CREATE | `internal/worksource/match.go` | `matches(rec BindingRecord, ref WorkItemRef) bool` — the in-memory predicate used by `MatchBindings` |
| CREATE | `internal/worksource/service_test.go` | CRUD round-trip + invalid-spec rejection cases (uses `state` test helper to stand up SQLite) |
| CREATE | `internal/worksource/match_test.go` | match-by-label, match-by-glob, match-state, no-match cases (table driven) |
| MODIFY | (none in state package) | PRE-5 already complete |

No `internal/service/worksource.go` — issue scopes only `internal/worksource/`. The thicker service-layer wrapper (`internal/service/`) does not exist yet and is out of scope for #2.2; #2.4 ("Run on this issue" button) will introduce it if needed.

## 4. Architecture Decisions

1. **Domain types separate from storage record.** `BindingRecord` exposes typed `LabelFilter []string`, `RepoPattern string`, `Trigger Trigger`; storage `WorksourceBindingRecord` keeps opaque JSON `Selector`/`Config`. Service is the only translator. Keeps state layer schema-stable as the dispatch matrix grows (per the comment in `state/worksource.go:23`).
2. **Soft-delete via `DeactivateBinding`.** `DeleteBinding` does not hard-delete; it flips `active=0`. Matches PRE-5 design and preserves run-history references. Documented on the interface method.
3. **`MatchBindings` is in-memory filter, not SQL.** Glob and label intersection do not map cleanly onto SQLite. `ListActiveBindings()` is bounded by binding count (small N — one row per project/forge/pipeline tuple), so an in-memory filter is fine and avoids encoding glob semantics in SQL. Revisit if N exceeds ~10⁴.
4. **`WorkItemRef` lives in this package, not in `internal/contract/schemas/shared/`.** #2.1 ships the JSON Schema. The Go struct is a plain value object here, mirroring the schema shape so that when #2.1 lands, the validator can be wired in via a one-line registry call. `WorkItemRef` fields: `Forge`, `Repo`, `Kind`, `ID`, `Title`, `URL`, `Labels []string`, `State string`.
5. **Trigger enum mirrored, not re-exported.** Define `worksource.Trigger` separately from `state.WorksourceTrigger` so service callers don't import `state`. Conversion in `types.go`.
6. **No context plumbing into the state store yet.** `state.WorksourceStore` methods don't take `context.Context`. Service interface accepts ctx (matches issue signature), checks `ctx.Err()` at entry, then calls store. State layer can add ctx in a follow-up without breaking the service contract.

## 5. Risks

| Risk | Mitigation |
|---|---|
| #2.1 `work_item_ref` schema not yet merged | Define Go struct locally with the schema shape from `onboarding-as-session-plan.md` §5. When schema lands, swap to schema-bound validation; field names already aligned. |
| Glob semantics surprise (`path.Match` is not full glob — no `**`) | Doc-string the supported syntax (`*`, `?`, `[abc]`); reject double-star at validate time with clear error. Real-world repo patterns don't need `**`. |
| Trigger string mismatch (dashed vs underscored) breaks round-trip | Conversion in one place (`types.go`); table-test both directions. |
| Selector JSON shape drift between writers | Single private `selectorPayload` struct with `json` tags; only marshalled by the service. Tests pin the wire form. |
| Test isolation across packages (state migrations need temp SQLite) | Reuse the existing test helper in `internal/state` (e.g. `newTestStore(t)` pattern); `t.TempDir()` for the DB file. |

## 6. Testing Strategy

Unit tests live in `internal/worksource/`. Each test spins up a fresh SQLite-backed `state.StateStore` via the existing test helper, then a `worksource.Service` over it.

| File | Cases |
|---|---|
| `service_test.go` | Create→Get round-trip, Create→List filter (forge, repo), Update mutates fields, Delete deactivates (List active excludes), invalid spec rejection (empty forge / empty pipeline / bad trigger / bad glob / empty repo pattern), Get/Update/Delete on missing ID returns error. |
| `match_test.go` | Table-driven: exact repo match, glob `*` match, glob `owner/*` match, label-filter (any-of semantics) match, state filter match, kind filter match, no match (forge mismatch / repo mismatch / label mismatch / state mismatch / inactive bindings excluded). |

Run gate: `go test ./internal/worksource/... -race`. Project contract: `go test ./...` (per `wave.yaml` `contract_test_command`).

No integration tests against real forges in this phase — that's #2.3/#2.4.

## 7. Out of Scope

- HTTP handlers / webui (#2.3)
- Dispatch wiring to `ExecutorService.Run` (#2.4)
- Schedule / cron evaluation (PRE-6 / phase 3)
- `work_item_ref` schema file (#2.1) — consumed as a Go struct here
