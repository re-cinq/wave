# refactor: merge internal/deliverable into internal/state.OutcomeRecord (single outcome model)

**Issue:** [#1286](https://github.com/re-cinq/wave/issues/1286)
**Repository:** re-cinq/wave
**Labels:** scope-audit
**Author:** nextlevelshit
**State:** OPEN

## Context

From wave-scope-audit run `wave-scope-audit-20260422-223006-df0b`.

`internal/deliverable` is an in-memory tracker that duplicates the responsibilities of `state.OutcomeRecord`. The scope audit marks `deliverable` as the only package-level **merge** verdict: there should be a single outcome model backed by `internal/state` rather than a parallel in-memory abstraction.

Scope package verdict: *"`deliverable` — In-memory tracker of pipeline deliverables. partial → **merge** into `state.OutcomeRecord`."*

Scope governance rule: *"One source of truth per concern. Skill probing, docker probing, pipeline discovery, outcome tracking, truncation helpers — each gets one implementation."*

## Acceptance Criteria

- [ ] Map every field and method of `internal/deliverable` onto `state.OutcomeRecord` (or its neighbors in `internal/state`); add any missing fields to `state`.
- [ ] Migrate all callers of `internal/deliverable` to the `state` equivalents.
- [ ] Delete `internal/deliverable` package and its tests.
- [ ] `go build ./...` and `go test ./...` pass; `go vet` clean.
- [ ] Update any ADRs or docs that mention `deliverable` to point at `state.OutcomeRecord`.
