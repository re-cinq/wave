# Requirements Quality Review Checklist

**Feature**: Restore and Stabilize `wave meta` Dynamic Pipeline Generation
**Date**: 2026-03-16
**Spec**: `specs/095-restore-meta-pipeline/spec.md`

---

## Completeness

- [ ] CHK001 - Are all five user stories traceable to at least one functional requirement? [Completeness]
- [ ] CHK002 - Does every functional requirement (FR-001–FR-013) have at least one acceptance scenario or test task that exercises it? [Completeness]
- [ ] CHK003 - Are error recovery behaviors defined for every failure mode (malformed YAML, circular deps, invalid JSON schemas, missing persona, timeout)? [Completeness]
- [ ] CHK004 - Is the default value for every resource limit explicitly stated in the spec (max_depth, max_total_steps, max_total_tokens, timeout)? [Completeness]
- [ ] CHK005 - Does the spec define the philosopher persona's expected input format (prompt template, context injection) or delegate it to an existing mechanism? [Completeness]
- [ ] CHK006 - Are output artifact auto-generation rules (FR-011) specified for all contract types, or only `json_schema`? Is this gap intentional? [Completeness]
- [ ] CHK007 - Does the spec define what happens when `--save` and `--dry-run` are combined? [Completeness]
- [ ] CHK008 - Are all six edge cases covered by at least one acceptance scenario or explicit error behavior? [Completeness]

## Clarity

- [ ] CHK009 - Is the distinction between "semantic validation" (FR-003) and "contract validation" (FR-004) clearly defined with non-overlapping scope? [Clarity]
- [ ] CHK010 - Is the `--- PIPELINE --- / --- SCHEMAS ---` delimiter format unambiguously specified (exact string, line boundaries, whitespace rules)? [Clarity]
- [ ] CHK011 - Is "navigator-first step" clearly defined as the topologically-first step in the DAG, not the positionally-first step in YAML? [Clarity]
- [ ] CHK012 - Are the `--save <name>` path resolution rules clear for all input variations (bare name, relative path, absolute path, with/without extension)? [Clarity]
- [ ] CHK013 - Is the auto-repair scope for JSON schemas (FR-012) bounded — which specific errors are repaired vs. rejected? [Clarity]
- [ ] CHK014 - Is "fresh memory strategy" precisely defined — does it mean `memory.strategy: fresh` in YAML, or a behavioral guarantee? [Clarity]

## Consistency

- [ ] CHK015 - Do the default resource limits in the spec (CL-001) match the constants in `internal/pipeline/meta.go`? [Consistency]
- [ ] CHK016 - Does FR-003's "all steps have contracts" align with the plan's `normalizeGeneratedPipeline()` which only handles `json_schema` contracts? [Consistency]
- [ ] CHK017 - Do the success criteria timeouts (SC-001: 120s for dry-run) conflict with the manifest timeout (30 min default)? Are these measuring different things? [Consistency]
- [ ] CHK018 - Does the plan's task count (23 tasks) and file scope (4 files) align with the functional requirements and user stories in the spec? [Consistency]
- [ ] CHK019 - Are the event names in FR-009 (meta_generate_started, meta_generate_completed, philosopher_invoking, schema_saved) consistent with existing event naming conventions in the codebase? [Consistency]
- [ ] CHK020 - Does the plan's `ValidationOption` pattern (Task 2) align with existing validation patterns in `internal/pipeline/validation.go`? [Consistency]

## Coverage

- [ ] CHK021 - Are concurrent/parallel execution scenarios addressed — what happens if two `wave meta` invocations run simultaneously? [Coverage]
- [ ] CHK022 - Is the interaction between `--mock` and `--save` defined — should mock-generated pipelines be saveable? [Coverage]
- [ ] CHK023 - Are token counting semantics defined — does `max_total_tokens` count philosopher tokens, child pipeline tokens, or both? [Coverage]
- [ ] CHK024 - Is the behavior defined when the philosopher generates a pipeline that itself contains a `meta` step (recursive meta)? [Coverage]
- [ ] CHK025 - Are filesystem permission errors handled — what if `.wave/contracts/` or `.wave/pipelines/` is not writable? [Coverage]
- [ ] CHK026 - Is cancellation behavior defined — what happens to child pipeline execution if the parent meta pipeline is cancelled mid-execution? [Coverage]
- [ ] CHK027 - Are backwards-compatibility implications addressed — do changes to `ValidateGeneratedPipeline()` signature break existing callers? [Coverage]
