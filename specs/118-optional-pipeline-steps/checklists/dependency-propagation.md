# Dependency Propagation Quality Review: Optional Pipeline Steps

**Feature**: 118-optional-pipeline-steps
**Date**: 2026-02-20
**Focus**: Quality of requirements for artifact injection skipping, transitive propagation, and dependency chain handling

---

## Skipping Logic Completeness

- [ ] CHK201 - Does the spec define the exact algorithm for checking artifact injection references before step execution — iterate all `inject_artifacts` entries and check each referenced step's state? [Completeness]
- [ ] CHK202 - Is it specified whether ALL artifact injection references must be satisfiable for a step to run, or whether a step with mixed references (some available, some from failed optional) is partially runnable? [Completeness]
- [ ] CHK203 - Does the spec address the case where a step's `inject_artifacts` references a step that hasn't executed yet (ordering error) — is this a separate concern or does the optional feature need to handle it? [Completeness]
- [ ] CHK204 - Are requirements defined for the log/event message content when a step is skipped — should it list which specific artifact reference(s) were unsatisfiable? [Completeness]
- [ ] CHK205 - Does the spec define whether skipped steps' workspaces are created at all, or whether workspace creation is bypassed entirely? [Completeness]

---

## Transitive Propagation Clarity

- [ ] CHK206 - Is the transitive propagation rule stated precisely: step C is skipped if ANY of its `inject_artifacts` reference a step in `failed_optional` or `skipped` state, regardless of how deep the chain is? [Clarity]
- [ ] CHK207 - Does the spec address potential circular dependency interactions — if steps form a diamond pattern where two paths lead to the same downstream step, one through a failed optional and one through a successful step, is the step skipped? [Clarity]
- [ ] CHK208 - Is it clear whether transitive propagation uses the SAME skipping state (`"skipped"`) regardless of depth in the chain? [Clarity]

---

## Dependency Chain Consistency

- [ ] CHK209 - Does T015's implementation description match CLR-002's resolution — specifically, does it check `inject_artifacts` references and NOT the `dependencies` field? [Consistency]
- [ ] CHK210 - Are the test cases in T016 consistent with the acceptance scenarios in User Story 3? [Consistency]
- [ ] CHK211 - Does the resume path (T024) replicate the same skipping logic as the normal execution path (T015) — are both specified to use the same algorithm? [Consistency]

---

## Dependency Edge Cases Coverage

- [ ] CHK212 - Does the spec address what happens when a step has `inject_artifacts` from multiple optional steps, and only SOME of them failed? [Coverage]
- [ ] CHK213 - Is there a requirement for handling `inject_artifacts` references where the `artifact` name doesn't match any output from the referenced step (independent of optional — but does optional introduce new edge cases here)? [Coverage]
- [ ] CHK214 - Does the spec define behavior when an optional step succeeds but produces EMPTY artifacts — are downstream steps that inject those artifacts affected? [Coverage]
- [ ] CHK215 - Is there a requirement for the depth limit of transitive propagation, or is it assumed unbounded (limited only by pipeline DAG depth)? [Coverage]
