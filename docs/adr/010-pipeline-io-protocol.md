# ADR-010: Pipeline I/O Protocol

## Status
Accepted

## Date
2026-04-20

## Context

Wave ships 57 pipelines. A pre-1.0 audit revealed three symptoms of the
same underlying problem:

1. **Inputs are universally free-text.** Every pipeline accepts a
   `string` input, regardless of whether the semantic intent is a URL,
   an issue reference, a spec path, or an arbitrary prompt. This pushes
   the cost of parsing into each pipeline's first step and makes typos
   survive until runtime.
2. **Output shapes diverge.** `pipeline_outputs` is uniform in
   structure (`{ name: { step, artifact } }`), but the contents differ
   dramatically. There are 7 different shapes for "PR result" and 11 for
   "findings," none of which share a schema. Pipelines that want to
   consume another pipeline's output must know the producer's private
   schema by convention.
3. **Composition is brittle.** Only 3 of 57 pipelines compose today.
   Sub-pipelines exchange data via `SubPipelineConfig.Inject` /
   `Extract` — manual artifact-name gymnastics that break silently when
   a producer renames an artifact or changes a field name. The handover
   is text-shaped and unverified.

Together these make Wave pipelines individually useful but collectively
incapable of being treated as Lego blocks. The goal of this ADR is a
typed I/O protocol that lets a user write `A → B` and have Wave reject
the pipeline at load time if B cannot consume what A produces.

Relevant code and files:
- `internal/pipeline/types.go` — `Pipeline`, `Step`, `PipelineOutput`,
  `InputConfig`
- `internal/pipeline/composition.go` — iterate / branch / loop / aggregate
- `internal/pipeline/subpipeline.go` — inject / extract (legacy path)
- `internal/pipeline/dag.go` — `YAMLPipelineLoader`
- `.agents/pipelines/*.yaml` — pipeline definitions
- `internal/contract/` — JSON-schema, TypeScript, test-suite validators

## Decision

Introduce a **typed I/O protocol** with four pieces:

1. **Shared schema registry.** A fixed set of canonical types lives in
   `internal/contract/schemas/shared/*.json`, embedded into the Wave
   binary via `go:embed`. Each file's basename is the type name
   (`issue_ref`, `pr_ref`, `branch_ref`, `spec_ref`, `findings_report`,
   `plan_ref`, `workspace_ref`, `scope_result`). A pipeline's type
   annotation must resolve against this registry or the sentinel
   `string`.

2. **Typed inputs and outputs at the pipeline level.**

   ```yaml
   input:
     source: cli
     type: issue_ref            # or "string"
     example: "re-cinq/wave 42"

   pipeline_outputs:
     pr:
       step: create-pr
       artifact: pr-result
       type: pr_ref             # or "string"
   ```

   Both fields are optional; an absent `type` is treated as `string`,
   which lets the 50+ free-text pipelines keep working unchanged.

3. **Typed handover at composition boundaries.** Composition steps
   (`pipeline:`, `iterate:`, `branch:`) gain an optional `input_ref`
   block that replaces today's ad-hoc string template:

   ```yaml
   - id: implement-all
     pipeline: impl-issue
     iterate:
       over: "{{ scope.output.child_issues }}"
       mode: sequential
     input_ref:
       from: scope.scope        # parent_step.output_name
       # literal: "..."          # alternative: static value
   ```

   Exactly one of `from` or `literal` is set. `from` references a
   pipeline output of a prior step; at validation time, the declared
   output type must match the child pipeline's declared input type.

4. **Load-time type-checking.** `YAMLPipelineLoader.Unmarshal` calls
   `ValidatePipelineIOTypes` immediately after defaulting. Unknown type
   names, malformed `input_ref` blocks, and outputs pointing at
   non-existent steps all fail there, before any step has run.
   Cross-pipeline wiring (producer output type must match consumer
   input type) is enforced by `TypedWiringCheck`, which is invoked
   alongside `detectSubPipelineCycles` when the composition runtime
   resolves sub-pipeline references.

## Options Considered

### Option 1: Per-pipeline inline schemas (status quo + more JSON)

Let each pipeline keep defining its own bespoke schema in
`.agents/contracts/` and push harder on documentation conventions.

**Pros:**
- Zero code changes.
- Full flexibility for pipeline authors.

**Cons:**
- Does not solve the composition problem — two pipelines that both
  produce "a PR result" still have incompatible schemas.
- Documentation drift is inevitable and has already happened.
- Catches nothing at load time.

### Option 2: Content-addressable artifact store (phase 2)

Store every artifact by content hash, with the type name as metadata.
Composition wires `from: <hash>` and validates by schema on read.

**Pros:**
- Perfect traceability.
- Enables cache-style reuse of prior step outputs.

**Cons:**
- Substantial state-store changes.
- Breaks intuitive `step.output` references.
- Overkill for phase 1. Keep as a follow-up.

### Option 3: Shared registry + declared types (**chosen**)

Canonical shared schemas embedded in the binary, declared type names
on input/output, load-time validation.

**Pros:**
- Minimal surface area: two optional YAML fields, one Go package.
- Fails fast on typos and mis-wired compositions.
- Leaves free-text pipelines alone.
- Sets the stage for content-addressable storage later without
  constraining it.

**Cons:**
- Introduces a second schema location (shared vs per-pipeline). This
  is acceptable because the shared set is small and stable by design.
- Pipelines that want richer types than the canonical set still need
  per-pipeline schemas; shared types are the interoperable core, not
  an exhaustive catalog.

## Consequences

### Positive
- Pipeline authors declare semantic intent once; Wave enforces it.
- New pipelines compose by default — the loader tells you at parse
  time whether A → B is valid.
- Seven of the most common "shape churns" (PR, issue, branch, spec,
  plan, workspace, findings, scope) now have a single reference shape.
- Existing untyped pipelines continue to run without edit.

### Negative
- Shared-schema evolution must be handled with care. Any breaking
  change to a shared schema invalidates every pipeline that uses it.
  Before 1.0 this is fine (no external users); after 1.0, changes
  require a major-version bump per project policy.
- Two flavors of input wiring coexist temporarily: the legacy
  `input: "<string template>"` and the new `input_ref`. The legacy
  form remains valid for `string`-typed children indefinitely.

### Neutral
- `ArtifactRef.Type` (a pre-existing field on step-level
  `inject_artifacts`) is not affected — it remains a documentation
  hint for now; extending it to use the shared registry is a
  follow-up.
- No DB schema changes.
- No changes to sandbox, adapter, webui, audit, ontology.

## Implementation Notes

Files created:
- `internal/contract/schemas/shared/*.json` (8 canonical schemas)
- `internal/contract/schemas/shared/registry.go` (+ test)
- `internal/pipeline/iotypes.go` (+ test)
- `docs/adr/010-pipeline-io-protocol.md` (this file)

Files modified:
- `internal/pipeline/types.go` — added `InputConfig.Type`,
  `PipelineOutput.Type`, `StepInput`, `Step.InputRef`.
- `internal/pipeline/dag.go` — invoke `ValidatePipelineIOTypes` from
  `YAMLPipelineLoader.Unmarshal`.
- `internal/pipeline/composition.go` — `resolveStepInput` honors
  `InputRef.From` / `InputRef.Literal` ahead of the legacy `SubInput`.
- Reference pipelines migrated:
  - `.agents/pipelines/impl-issue.yaml` +
    `internal/defaults/pipelines/impl-issue.yaml`
    — input `issue_ref`, output `pr_ref`.
  - `.agents/pipelines/impl-speckit.yaml` +
    `internal/defaults/pipelines/impl-speckit.yaml`
    — input `string` (free-text is the right model for natural-language
    feature descriptions), output `pr_ref`.
  - `.agents/pipelines/plan-scope.yaml` +
    `internal/defaults/pipelines/plan-scope.yaml`
    — input `string`; added typed `scope` output (`scope_result`).
  - `.agents/pipelines/ops-epic-runner.yaml` +
    `internal/defaults/pipelines/ops-epic-runner.yaml`
    — shows scope → iterate over child_issues → impl-issue wiring.

### Phase 2 migration TODO

The following pipelines have obvious typed-I/O candidates and should
be migrated in a follow-up PR. None of them block this ADR.

| Pipeline | Suggested input | Suggested outputs |
|---|---|---|
| `impl-issue-core` | `issue_ref` | `pr_ref` |
| `impl-research` | `issue_ref` | `pr_ref` |
| `impl-hotfix`, `impl-feature`, `impl-improve`, `impl-refactor`, `impl-recinq`, `impl-prototype` | `issue_ref` or `string` | `pr_ref` |
| `ops-pr-review`, `ops-pr-review-core`, `ops-pr-fix-review` | `pr_ref` | `findings_report` |
| `ops-refresh` | `issue_ref` | `issue_ref` |
| `plan-research`, `plan-adr`, `plan-task`, `plan-approve-implement` | `string` or `issue_ref` | `plan_ref` |
| `audit-*` (14 pipelines) | `string` (scope hint) | `findings_report` |
| `audit-dead-code-issue`, `audit-dead-code-review` | `issue_ref` / `pr_ref` | `findings_report` |
| `doc-fix`, `doc-scan`, `doc-onboard` | `string` | `findings_report` |
| `test-gen` | `string` (target package/path) | `findings_report` |
| `bench-solve` | `string` (task id) | custom (keep per-pipeline) |
| `wave-bugfix`, `wave-evolve`, `wave-review`, `wave-audit`, `wave-security-audit`, `wave-test-hardening` | `string` | `findings_report` |
| `ops-implement-epic`, `ops-epic-runner` (partial) | `string` | aggregate |
| `ops-bootstrap` | `string` | `workspace_ref` |

The `wave-smoke-*` and `wave-ontology-*` test pipelines stay untyped
by design — they exercise specific executor paths and need free shape.

### Non-goals for phase 1

- No content-addressable artifact store (ADR candidate for phase 2).
- No automatic runtime schema-validation of produced outputs against
  declared types. Today's `json_schema` contract handles output
  validation per-step; the shared registry is for **load-time**
  compatibility, not runtime enforcement. A follow-up ADR should
  decide whether to wire shared types into the runtime contract path.
- No deep `field`-extraction on `pipeline_outputs` with type
  narrowing. The existing `field:` hint stays as-is.
