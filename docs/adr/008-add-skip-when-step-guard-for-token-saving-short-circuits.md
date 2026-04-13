# ADR-008: Add skip_when Step Guard for Token-Saving Short-Circuits

## Status
Proposed

## Date
2026-04-13

## Context

Wave pipelines execute steps in a directed acyclic graph (DAG) via a findReadySteps/executeStepBatch polling loop in executor.go, which contains approximately 5500 lines of code. Steps are dispatched to LLM adapters (Claude, Gemini, Codex) as subprocesses, each running in an ephemeral workspace with fresh memory—a constitutional requirement ensuring no chat history leaks between steps.

The current architecture provides no mechanism to conditionally skip a step based on the output of a prior step. The only skip mechanisms are: (1) transitive dependency failure propagation via skipDependentSteps, (2) ReworkOnly steps excluded from normal DAG scheduling, and (3) optional steps marked skipped on failure. Pipeline authors cannot express "run this step only if upstream analysis produced actionable output."

This gap causes significant token waste. Analysis of 78 pipeline runs across the impl-refactor, audit-dead-code variants, and doc-fix pipelines shows 70 runs (90%) produced zero diff—meaning the downstream steps had nothing to work on. impl-refactor alone wastes approximately 112,000 tokens per zero-diff run on test-baseline, refactor, and verify steps that have nothing to refactor. With a fleet of 50+ pipeline definitions, the cumulative waste scales linearly.

The constraint guiding implementation: Go 1.25.5 with no external expression library dependencies (single-static-binary philosophy: cobra, yaml.v3, sqlite, jsonschema only). Two separate template systems exist—PipelineContext.ResolvePlaceholders for pipeline variables and TemplateContext.ResolveTemplate for composition-level step output access via {{step_id.output.field}}. Bridging these systems at the scheduling level is required.

## Decision

Implement the Template-then-Truthy skip_when pattern: add a SkipWhen string field to the Step struct. At scheduling time (after findReadySteps resolves dependencies, before adapter dispatch), resolve the skip_when template using TemplateContext.ResolveTemplate, which already supports {{step_id.output.field}} expressions with JSON path extraction. The resolved string undergoes a truthiness check: empty string, 'false', '0', 'none', or 'null' evaluates to falsy (step runs); anything else is truthy (step skipped).

This option was chosen because it reuses four existing subsystems end-to-end—TemplateContext.ResolveTemplate for resolution, resolveStepOutput/ExtractJSONPath for step output access, stateSkipped for skip lifecycle, and event.StateSkipped for event emission—adding only approximately 200 lines of new Go code. It saves 150-200k tokens per batch on clean codebases while honoring the single-static-binary constraint. The truthiness approach covers the 90% of real-world skip cases (upstream output was empty/none/false), which the audit data demonstrates is the dominant pattern. It follows the proven ReworkOnly precedent of a field checked at scheduling time and sits cleanly at the scheduling/execution boundary, making it compatible with future GraphScheduler (ADR-005) and StepExecutor (ADR-002) extractions.

## Options Considered

### Option 1: Status Quo — No Conditional Skip
Keep the current architecture where all non-ReworkOnly steps execute unconditionally once dependencies complete. Pipeline authors accept the token cost of running downstream steps even when upstream produces no useful output.

Pros: Zero implementation effort, no schema updates, no coupling between PipelineContext/TemplateContext, no cascading-skip complexity for authors, avoids adding scheduling logic before ADR-002/ADR-005 refactors land.

Cons: Continues wasting 150-200k tokens per batch on zero-value work across 70/78 zero-diff runs (90%); impl-refactor alone wastes ~112k tokens per zero-diff run; no way to express "this step is pointless if upstream found nothing"; cumulative waste scales linearly as pipeline fleet grows; leaves battle-tested stateSkipped infrastructure unused.

Effort: Trivial. Risk: Low. Reversibility: Easy.

### Option 2: Template-then-Truthy skip_when (Chosen)
Add SkipWhen field to Step struct. At scheduling time, resolve template via TemplateContext.ResolveTemplate, then apply truthiness check (empty/false/0/none/null = falsy, step runs; anything else = truthy, step skips). Approximately 200 lines of Go across types.go, executor.go, template.go, schema, and pipeline YAMLs.

Pros: Reuses existing infrastructure end-to-end (TemplateContext, stateSkipped, events); saves 150-200k tokens per batch; no expression parser or new dependencies; follows ReworkOnly precedent; sits at scheduling/execution boundary for future extraction; fully additive, no breaking changes; covers 90% of real-world cases per audit data.

Cons: Requires bridging TemplateContext into DAG scheduling loop via LoadStepArtifact; truthiness semantics are implicit (authors must know falsy values); skipped steps produce no artifacts, so downstream artifact injection fails (cascading-skip must be manual); adds ~20 lines to executor.go; 15+ pipeline YAMLs need manual guard annotation.

Effort: Small. Risk: Low. Reversibility: Easy.

### Option 3: Full Expression Language skip_when
Same SkipWhen field but implement or integrate a full expression evaluator supporting operators (==, !=, >, <, >=, <=), functions (length, contains, isEmpty), and boolean logic. Example: '{{ analyze.output.findings | length == 0 }}'.

Pros: Covers 100% of skip conditions including numeric comparisons and array length checks; explicit operators make intent unambiguous; future-proof for complex conditions.

Cons: Violates single-static-binary constraint (no external expression library); custom parser adds 500-800 lines; common source of bugs (operator precedence, type coercion); overpowered for 90% truthiness case; introduces third evaluation system fragmenting the evaluation story; longer implementation delays token savings; still has cascading-skip footgun.

Effort: Medium. Risk: Medium. Reversibility: Moderate.

### Option 4: Extend Graph Edge Conditions
Extend condition.go EvaluateCondition to support output namespace (e.g., output.step_id.field=VALUE), then use graph-mode edge routing to conditionally bypass steps. Edge condition determines whether to traverse an incoming edge; unsatisfied edges mean step not scheduled.

Pros: Builds on existing condition pattern (outcome=success, context.KEY=VALUE); graph edge conditions are a graph scheduling primitive; aligns with ADR-005 GraphScheduler direction.

Cons: Requires converting 15+ linear DAG pipelines to graph mode with explicit edges; significant restructuring burden; condition.go coupling with outcomes/template layer; graph pipelines more complex to author/debug; edge semantics differ from step skipping conceptually; higher effort than simple skip_when annotation.

Effort: Large. Risk: Medium. Reversibility: Moderate.

### Option 5: Pre-step Script Guard
Use existing Step.Script/Step.Exec for lightweight pre-check before adapter dispatch. Script reads prior artifacts from disk, exits with conventional code (exit 42 = skip). Executor interprets exit code and sets stateSkipped.

Pros: No expression language needed; maximally flexible (shell/jq can check anything); clean separation for future StepExecutor extraction.

Cons: Spawns subprocess for every guarded step (adds latency); scripts are opaque to pipeline engine, no validation; requires jq available (conflicts with single-static-binary); fragile shell scripts in YAML; reinvents wheel when Go building blocks exist; 15+ pipelines need bespoke scripts, far more lines than Option B.

Effort: Medium. Risk: Medium. Reversibility: Easy.

## Consequences

### Positive
- Saves 150-200k tokens per batch on clean codebases (70/78 zero-diff runs no longer waste tokens)
- impl-refactor pipeline saves ~112k tokens per zero-diff run on test-baseline/refactor/verify steps
- Reuses stateSkipped infrastructure (persistence, events, UI badges, cost tracking) that was already battle-tested
- Allows pipeline authors to express "skip if upstream found nothing" as a first-class primitive

### Negative
- Pipeline authors must manually cascade skip_when guards to downstream steps that depend on skipped step artifacts (those steps will fail at resolveStepResources)
- Truthiness semantics require documentation: authors must understand that 'none', '0', 'false', '' are falsy
- Approximately 15+ pipeline YAML files need manual skip_when annotation (one-time effort)
- Requires bridging TemplateContext into DAG scheduling loop—LoadStepArtifact must load completed step outputs before skip_when evaluation

### Neutral
- wave-pipeline.schema.json must add skip_when to Step definition (adds new property)
- types.go: add SkipWhen field to Step struct (~5 lines)
- executor.go: add ~20 lines in scheduling loop for skip guard evaluation
- template.go: add ~30 lines bridging StepOutputs via LoadStepArtifact for DAG mode
- internal/event: may want distinct skip message (e.g., "skipped: skip_when condition met")
- Compatible with future ADR-005 GraphScheduler and ADR-002 StepExecutor extraction (skip_when sits at scheduling/execution boundary)

## Implementation Notes

1. Add SkipWhen field to Step struct in project/internal/pipeline/types.go with yaml tag `skip_when,omitempty`

2. Add skip_when guard in executor.go scheduling loop (around line 820-948 where findReadySteps results are processed, before adapter dispatch):
   - Load completed step artifacts via LoadStepArtifact into temporary TemplateContext
   - Resolve skip_when template via TemplateContext.ResolveTemplate
   - Apply truthiness check: empty string, 'false', '0', 'none', 'null' = falsy
   - If truthy, set step state to stateSkipped, emit event.StateSkipped, continue

3. Add StepOutputs bridging in project/internal/pipeline/template.go: extend LoadStepArtifact for DAG scheduling path (not just composition primitives)

4. Update project/internal/defaults/schemas/wave-pipeline.schema.json: add skip_when property to Step definition

5. Add skip_when guards to 15+ pipeline YAMLs: impl-refactor, audit-dead-code, audit-dead-code-issue, audit-dead-code-review, audit-junk-code, audit-closed, audit-consolidate, audit-security, doc-fix, doc-changelog, impl-hotfix, impl-improve, impl-issue, impl-feature, test-gen

6. Document truthiness semantics in pipeline authoring guide: falsy values are empty string, 'false', '0', 'none', 'null'; everything else is truthy

7. Testing: unit tests in executor_test.go and template_test.go covering truthiness evaluation; integration tests with sample pipelines having skip_when guards
