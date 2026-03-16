# Requirements Quality Checklist: Continuous Pipeline Execution

**Feature**: #201 — Continuous Pipeline Execution
**Generated**: 2026-03-16

## Completeness

- [ ] CHK001 - Does the spec define behavior when `--continuous` is used without `--source`? (FR-001 implies `--source` is separate, but no requirement states it is mandatory with `--continuous`) [Completeness]
- [ ] CHK002 - Does the spec define what happens when a `file:` source file is modified externally while the continuous loop is running? (FileSource loads all lines at construction — is this stated clearly enough?) [Completeness]
- [ ] CHK003 - Are retry semantics for transient source failures (e.g., `gh` CLI network errors) fully specified beyond the edge case mention of rate limits? [Completeness]
- [ ] CHK004 - Does FR-012 (dedup tracking) specify how the work item ID is derived for each source type? (GitHub = issue number, file = line content hash — but only in plan.md, not spec.md) [Completeness]
- [ ] CHK005 - Is the behavior of `--delay` with value `0s` explicitly stated as "no delay" in FR-013, or could it be interpreted as "default system delay"? [Completeness]
- [ ] CHK006 - Does the spec define the output format of the summary (FR-014)? Is it structured JSON, plain text, or both? [Completeness]
- [ ] CHK007 - Are all five CLI flags (`--continuous`, `--source`, `--max-iterations`, `--delay`, `--on-failure`) covered by at least one acceptance scenario? [Completeness]
- [ ] CHK008 - Does the spec define behavior when `--max-iterations 0` is passed? Is 0 treated as "unlimited" or as an error? [Completeness]

## Clarity

- [ ] CHK009 - Is the `--source` URI grammar (`<provider>:<key=value,...>`) formally defined with a BNF or clear syntax description, or only by example? [Clarity]
- [ ] CHK010 - Is "current iteration completes" (US2, SC-002) unambiguous — does it mean the current step finishes, the current pipeline run finishes, or the current iteration's cleanup finishes? [Clarity]
- [ ] CHK011 - Is the distinction between "iteration" (one pipeline run) and "step" (one step within a pipeline) consistently used throughout all requirements and scenarios? [Clarity]
- [ ] CHK012 - Does the edge case about `--continuous` + `--from-step` mutual exclusion explain the rationale clearly enough for users to understand why it's rejected? [Clarity]
- [ ] CHK013 - Is `FailurePolicy` naming consistent across all artifacts? (spec uses `on_failure: halt|skip`, plan uses `FailurePolicyHalt|FailurePolicySkip`, CLI uses `--on-failure`) [Clarity]

## Consistency

- [ ] CHK014 - Does the in-memory-only ContinuousRun decision (C3) create a gap in observability — can `wave list runs` show the continuous session as a group, or only individual runs? [Consistency]
- [ ] CHK015 - Is the exit code semantics (C4: exit 1 on any failure even with `skip`) consistent with US5 scenario 3 which expects non-zero exit for partial failures? [Consistency]
- [ ] CHK016 - Are the event state names (`loop_start`, `loop_iteration_start`, etc.) consistent with Wave's existing event naming conventions in `internal/event/emitter.go`? [Consistency]
- [ ] CHK017 - Does the spec's "fresh workspace per iteration" (FR-004) align with the existing workspace lifecycle in `internal/workspace/` — will workspace cleanup happen between iterations? [Consistency]
- [ ] CHK018 - Is the `--on-failure` flag default (`halt`) consistent between spec (FR-009), plan (Phase A), and data model? [Consistency]
- [ ] CHK019 - Does the plan's executor factory pattern (`ExecutorFunc`) align with how `DefaultPipelineExecutor` is currently instantiated in `internal/pipeline/executor.go`? [Consistency]

## Coverage

- [ ] CHK020 - Are negative test scenarios defined for all mutual exclusion rules (e.g., `--continuous` + `--from-step`, `--source` without `--continuous`)? [Coverage]
- [ ] CHK021 - Do the acceptance scenarios cover the case where a work item source returns an error (not empty, but an actual error) on the first call? [Coverage]
- [ ] CHK022 - Is there a scenario testing the interaction between `--max-iterations` and `on_failure: skip` (e.g., does a skipped iteration count toward the max)? [Coverage]
- [ ] CHK023 - Do the test scenarios cover concurrent signal delivery (e.g., multiple rapid SIGINT signals)? [Coverage]
- [ ] CHK024 - Is there coverage for the `file:` source with edge cases: empty lines, whitespace-only lines, lines with special characters? [Coverage]
- [ ] CHK025 - Do the scenarios validate that `wave logs <run-id>` works for individual iteration run IDs after a continuous session? [Coverage]
- [ ] CHK026 - Is there a scenario verifying SC-005 (no overhead on single-run execution) — specifically that running without `--continuous` is functionally identical to current behavior? [Coverage]
