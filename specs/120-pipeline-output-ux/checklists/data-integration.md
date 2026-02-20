# Data Model & Integration Requirements Checklist

**Feature**: Pipeline Output UX — Surface Key Outcomes
**Spec**: `specs/120-pipeline-output-ux/spec.md`
**Date**: 2026-02-20

This checklist validates the quality of data model definitions, type extensions, and integration point requirements.

---

## Deliverable Type Extensions

- [ ] CHK201 - Is the behavior of GetByType() for the new TypeBranch and TypeIssue constants specified in terms of what it returns when no matching deliverables exist? [Completeness]
- [ ] CHK202 - Are the exact Metadata keys for branch deliverables ("pushed", "remote_ref", "push_error") defined with value types and default values? [Clarity]
- [ ] CHK203 - Is the deduplication behavior for AddBranch() specified when the same branch name is added multiple times from different steps? [Completeness]
- [ ] CHK204 - Is the thread-safety requirement for UpdateMetadata() explicitly stated, including behavior when called from concurrent goroutines? [Clarity]
- [ ] CHK205 - Is the error handling for UpdateMetadata() when the target deliverable doesn't exist defined (silent no-op vs error return)? [Clarity]

## Executor Instrumentation

- [ ] CHK206 - Are the exact code locations for branch deliverable recording specified with enough context to avoid instrumenting wrong paths (e.g., worktree reuse vs new creation)? [Clarity]
- [ ] CHK207 - Is the issue URL detection regex pattern specified with test cases covering valid and invalid URL patterns? [Completeness]
- [ ] CHK208 - Is the mechanism for detecting "publish steps" defined — by step ID naming convention, step metadata, or explicit pipeline configuration? [Clarity]
- [ ] CHK209 - Are the conditions under which push status metadata gets updated defined for all scenarios (push succeeds, push fails, push not attempted)? [Completeness]
- [ ] CHK210 - Is the relationship between existing PR detection in trackCommonDeliverables() and the new issue detection specified to avoid duplicate URL recording? [Consistency]

## JSON Output Schema

- [ ] CHK211 - Is the JSON schema for OutcomesJSON formally defined (or describable as JSON Schema) for downstream consumer validation? [Completeness]
- [ ] CHK212 - Is the behavior for the `issues` field specified as empty array `[]` (not `null`) per US3-AC3, and is this enforced by the struct definition or serialization config? [Clarity]
- [ ] CHK213 - Is backward compatibility for existing JSON consumers explicitly addressed — specifically, what happens when an old consumer encounters the new `outcomes` field? [Completeness]
- [ ] CHK214 - Is the JSON output for failed pipelines specified (does outcomes still appear, are partial results included)? [Completeness]

## Cross-Layer Integration

- [ ] CHK215 - Is the data flow from Tracker → PipelineOutcome → OutcomeSummary/OutcomesJSON fully traceable with no data transformations left unspecified? [Completeness]
- [ ] CHK216 - Is the BuildOutcome() function's contract (preconditions, postconditions, invariants) specified beyond just its signature? [Clarity]
- [ ] CHK217 - Are the import relationships between packages (display → deliverable, display → event, run → display) specified to avoid circular dependencies? [Consistency]
- [ ] CHK218 - Is the AllDeliverables field on PipelineOutcome specified in terms of ordering (creation time, priority, or unspecified)? [Clarity]
