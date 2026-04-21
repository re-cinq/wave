# Tasks: Farewell Function

**Branch**: `1107-farewell-function`
**Spec**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md) | **Contract**: [contracts/farewell.md](./contracts/farewell.md)

Legend: `[P]` = parallelizable (different files, no dep); `[US1|US2|US3]` = user story tag.

## Phase 1: Setup

- [X] T001 Create package directory `internal/farewell/` at repo root.
- [X] T002 Add package doc comment in `internal/farewell/doc.go` describing purpose (single-source farewell string for CLI/TUI/embedders).

## Phase 2: Foundational

- [X] T003 [US2] Define package constants in `internal/farewell/farewell.go`: generic template `"Farewell — see you next wave."` and named template `"Farewell, %s — see you next wave."` as unexported consts.

## Phase 3: User Story 2 — Programmatic Farewell (P2, foundational for US1/US3)

- [X] T004 [US2] Implement `func Farewell(name string) string` in `internal/farewell/farewell.go`: trim whitespace; empty → generic; else interpolate named template (FR-001, FR-002, FR-003, FR-009).
- [X] T005 [P] [US2] Implement `func WriteFarewell(w io.Writer, name string, suppress bool) error` in `internal/farewell/farewell.go`: no-op when `suppress`, else write `Farewell(name) + "\n"` (FR-005, FR-006, FR-011).
- [X] T006 [US2] Add table-driven tests in `internal/farewell/farewell_test.go` covering contract cases 1–6 (empty, named, whitespace-trim, determinism, write, suppress).

**Checkpoint US2**: `go test ./internal/farewell/...` green; public API stable.

## Phase 4: User Story 1 — Visible Farewell on Session End (P1)

- [X] T007 [US1] Locate existing quiet/TTY suppression helper in `cmd/wave/commands/output.go`; expose (or reuse as-is) a predicate `shouldSuppressOutput()` usable from post-run hook.
- [X] T008 [US1] Add post-run hook in `cmd/wave/commands/root.go` (cobra `PersistentPostRunE` or equivalent) that, on nil error, resolves `os.Getenv("USER")` and calls `farewell.WriteFarewell(os.Stdout, name, suppress)` (FR-004, FR-006, FR-007, FR-010).
- [X] T009 [P] [US1] Wire identical call into TUI teardown path (search `internal/tui/` for shutdown/exit; add single call to `farewell.WriteFarewell` after UI clears) (FR-008, AS-1.2).
- [X] T010 [US1] Add CLI integration test (or command test using existing test harness in `cmd/wave/commands/`) asserting successful command stdout ends with farewell line when TTY + not quiet (C1, SC-001).
- [X] T011 [P] [US1] Add test: `$USER=alice` yields line containing `alice`; unset `$USER` yields generic line (C5, C6, FR-010).
- [X] T012 [P] [US1] Add test: failing command produces no farewell line; error output unchanged (C4, FR-007).

**Checkpoint US1**: interactive `wave` successful command prints farewell; failures don't.

## Phase 5: User Story 3 — Silent / Scripting Mode (P3)

- [X] T013 [US3] Add test asserting non-TTY stdout (piped) produces no farewell line (C3, SC-002).
- [X] T014 [P] [US3] Add test asserting `--quiet` flag set suppresses farewell regardless of TTY (C2, FR-005, FR-011).

**Checkpoint US3**: suppression honored in all documented modes.

## Phase 6: Polish & Cross-Cutting

- [X] T015 [P] Run `go test ./...` and `go vet ./...` from repo root; ensure no regressions.
- [X] T016 [P] Run `golangci-lint run ./internal/farewell/... ./cmd/wave/commands/...` and fix findings.
- [X] T017 Manual smoke: `wave list` (TTY) shows farewell; `wave list | cat` does not; `wave list --quiet` does not; failing command does not (C1–C4 manual validation).
- [X] T018 [P] Update `docs/` if a user-facing CLI doc enumerates output behavior (only if such doc exists; otherwise skip).

## Dependency Notes

- T003 blocks T004, T005.
- T004 blocks T005 (T005 calls `Farewell`).
- T004, T005 block T006 and all Phase 4/5 tasks.
- T007 blocks T008, T009.
- T008 blocks T010–T012.
- Phase 5 tests depend on Phase 4 wiring (T008).
- Polish tasks (Phase 6) run last.

## Parallel Opportunities

Tagged `[P]`: T005, T009, T011, T012, T014, T015, T016, T018.
