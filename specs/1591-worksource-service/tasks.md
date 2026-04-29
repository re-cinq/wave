# Work Items — #1591 WorkSourceService

## Phase 1: Setup

- [X] 1.1: Create `internal/worksource/` directory + `doc.go` package comment
- [X] 1.2: Add `internal/worksource/types.go` skeleton — `BindingID`, `Trigger` enum (with dashed↔underscored converters), `BindingSpec`, `BindingRecord`, `BindingFilter`, `WorkItemRef`, `selectorPayload` (private)

## Phase 2: Core Implementation

- [X] 2.1: `internal/worksource/validate.go` — `validateSpec`: empty forge / empty pipeline / unknown trigger / empty repo pattern / `path.Match` syntax check / no-`**` rule [P]
- [X] 2.2: `internal/worksource/match.go` — `matches(BindingRecord, WorkItemRef) bool`: forge equal, `path.Match` repo, label any-of, state subset, kind subset [P]
- [X] 2.3: `internal/worksource/service.go` — `Service` interface (six methods, ctx-first) + `service` struct + `NewService(state.WorksourceStore) Service`
- [X] 2.4: Wire `CreateBinding` — `validateSpec` → marshal selector JSON → `store.CreateBinding` → return `BindingID`
- [X] 2.5: Wire `GetBinding`/`ListBindings` — call store, translate `WorksourceBindingRecord`→`BindingRecord` (selector JSON unmarshal, trigger denormalise)
- [X] 2.6: Wire `UpdateBinding` — re-validate spec, preserve `CreatedAt`, call `store.UpdateBinding`
- [X] 2.7: Wire `DeleteBinding` → `store.DeactivateBinding` (soft-delete; document on interface)
- [X] 2.8: Wire `MatchBindings` — `store.ListActiveBindings` → in-memory `matches` filter

## Phase 3: Testing

- [X] 3.1: `internal/worksource/service_test.go` — CRUD round-trip with real SQLite state store [P]
- [X] 3.2: `internal/worksource/service_test.go` — invalid-spec rejection table (empty forge, empty pipeline, bad trigger, bad glob, empty repo) [P]
- [X] 3.3: `internal/worksource/service_test.go` — Get/Update/Delete missing-ID error paths [P]
- [X] 3.4: `internal/worksource/match_test.go` — exact repo, glob, label any-of, state, kind, inactive-excluded, multi-mismatch [P]
- [X] 3.5: `internal/worksource/types_test.go` — trigger dashed↔underscored round-trip; selector JSON wire form pinned [P]
- [X] 3.6: Run `go test ./internal/worksource/... -race` — green
- [X] 3.7: Run `go test ./...` (project contract) — green

## Phase 4: Polish

- [X] 4.1: Doc-string every exported symbol; note glob syntax limits and soft-delete semantics
- [ ] 4.2: Run `golangci-lint run ./internal/worksource/...` — clean (lint binary not present in this sandbox; `go vet ./internal/worksource/...` clean)
- [X] 4.3: Cross-check acceptance criteria checklist on the issue, mark each
- [X] 4.4: PR description references epic #1565, lists files, calls out #2.1 schema follow-up
