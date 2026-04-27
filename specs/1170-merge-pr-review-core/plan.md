# Implementation Plan: Merge ops-pr-review-core into ops-pr-review

## Objective

Eliminate the `ops-pr-review-core.yaml` fork by inlining its 5 review steps into `ops-pr-review.yaml`, and gate the trailing `publish` step behind a profile flag so callers can opt out of forge interaction.

## Approach

The chosen mechanism is a **per-step `branch` gate** driven by a profile value supplied by the parent (sub-pipeline caller) or defaulted to `full` for direct CLI invocation. Two implementation tracks were considered:

- **Track A (chosen): SubPipelineConfig env passthrough + template `{{ env.* }}`**
  Adds `Env map[string]string` to `SubPipelineConfig` (already partially started in prior run), plumbs it into `TemplateContext`, and resolves `{{ env.profile }}` with a default fallback. Lets composition pipelines (`inception-bugfix`) request `profile: core` while CLI invocation defaults to `full`. Branch step uses `cases.core: skip, cases.default: ops-pr-review-publish`.

- **Track B (rejected): input-encoded profile (e.g. `core:<pr_url>`).**
  Avoids any code change but pollutes the public CLI surface with magic prefixes, breaks the typed `pr_ref` input contract declared in ADR-010, and forces every step that templates `{{ input }}` to strip the prefix. Bad ergonomics, contract violation.

Track A also requires a tiny new pipeline file (`ops-pr-review-publish.yaml`) holding only the publish step, because the existing `branch` primitive dispatches to pipeline names — not to inline blocks. Net file delta: **-1** (`-core` deleted, `-publish` added, original kept and inlined).

## File Mapping

### Create

- `internal/defaults/pipelines/ops-pr-review-publish.yaml` — single-step pipeline holding the publish prompt (extracted verbatim from current `ops-pr-review.yaml`'s `publish` step). `release: false`. Input type `pr_ref`. Reads `.agents/output/review-verdict.json` and `.agents/output/diff-analysis.json` from parent workspace.
- `.agents/pipelines/ops-pr-review-publish.yaml` — mirror.
- `specs/1170-merge-pr-review-core/{spec.md,plan.md,tasks.md}` — already created.

### Modify

- `internal/pipeline/types.go` — extend `SubPipelineConfig` with `Env map[string]string`. Update `Validate()` if any constraints (none expected; arbitrary string map).
- `internal/pipeline/template.go` — extend `TemplateContext` with `Env map[string]string`; add `case strings.HasPrefix(expr, "env."):` arm in `resolveExpression`. Missing keys resolve to empty string (template default behavior consistent with `{{ input }}` when empty).
- `internal/pipeline/composition.go` — in `executeSubPipeline`, propagate `step.Config.Env` into the child's `TemplateContext`. Inherit parent env first, then overlay child-specific overrides.
- `internal/pipeline/subpipeline.go` — wire env into the child runner construction.
- `internal/defaults/schemas/wave-pipeline.schema.json` — add `env` property to the `SubPipelineConfig` definition (object, additionalProperties string).
- `internal/defaults/pipelines/ops-pr-review.yaml` — replace the `core-review: { pipeline: ops-pr-review-core }` step with the 5 inlined steps from `ops-pr-review-core.yaml` (diff-analysis → security-review → quality-review → slop-review → summary). Replace the inline `publish` prompt step with a `branch` step:
  ```yaml
  - id: publish
    branch:
      on: "{{ env.profile }}"
      cases:
        core: skip
        default: ops-pr-review-publish
    dependencies: [summary]
  ```
  Update `pipeline_outputs.verdict` to point at `step: summary` (was `step: publish`). Also expose `diff` artifact like the core variant declared.
- `.agents/pipelines/ops-pr-review.yaml` — mirror.
- `internal/defaults/pipelines/inception-bugfix.yaml` — change the `review` step from `pipeline: ops-pr-review-core` to:
  ```yaml
  - id: review
    pipeline: ops-pr-review
    config:
      env:
        profile: core
    dependencies: [fix]
    input: "{{ input }}"
  ```
- `.agents/pipelines/inception-bugfix.yaml` — mirror.
- `docs/adr/010-pipeline-io-protocol.md` — update the I/O table row that lists `ops-pr-review-core` (line ~233) to remove `-core`. Also strike its mention in the "6 of 26 pipelines" line at ~18 and update the count.
- `docs/guides/contract-chaining.md` — replace any `ops-pr-review-core` reference with the new profile-flag example.
- `AGENTS.md` — replace `ops-pr-review-core` references with the unified pipeline + profile guidance.

### Delete

- `internal/defaults/pipelines/ops-pr-review-core.yaml`
- `.agents/pipelines/ops-pr-review-core.yaml`
- `specs/761-pipeline-consolidation/tasks.md` and `specs/1401-ops-pr-respond/{spec,plan,tasks}.md` are historical artifacts — leave untouched (they document past planning, not live config).

## Architecture Decisions

1. **Env passthrough scope**: `Env` is a flat `map[string]string`. No nesting, no typed values. The only consumer for now is profile selection. Avoid premature generalization.
2. **Default value semantics**: missing env keys resolve to empty string. The `branch` step's `cases.default: ops-pr-review-publish` arm handles this — `{{ env.profile }}` → `""` → no `cases[""]` match → fall back to `default`. This means CLI direct invocation (no parent, no env) gets `full` behavior automatically.
3. **No new top-level pipeline `env:` block**. Defaults belong in the calling pipeline or the consumer's `cases.default`. Adds zero schema surface.
4. **Inline vs sub-pipeline for publish**: `branch` requires a pipeline name as the case value. Extracting publish into its own pipeline file is the lightest path. The new `ops-pr-review-publish.yaml` is `release: false` (internal building block, not user-facing).
5. **Artifact path consistency**: the inlined steps keep their existing `.agents/output/*.json` paths. The publish pipeline reads those same paths from its sub-pipeline workspace — verify path resolution works across the parent→child boundary (sub-pipelines share workspace root by default; confirm via the existing `ops-pr-review-core` invocation pattern that already works today).
6. **No `release: true` change**: ops-pr-review remains user-facing (`release: true`); the new internal `-publish` is `release: false` like the deleted `-core` was.
7. **Mirror both pipeline copies**: per memory, `internal/defaults/pipelines/` (shipped defaults) and `.agents/pipelines/` (project-tailored) are independent but for built-in pipelines must stay in sync. Diff confirmed both copies are identical today.

## Risks

| Risk | Mitigation |
|------|-----------|
| Sub-pipeline workspace doesn't share artifact paths with parent — `ops-pr-review-publish` can't read `.agents/output/review-verdict.json` written by the `summary` step. | Confirm by testing; if the boundary breaks, use `config.extract: [verdict, diff]` on the inlined summary/diff-analysis steps and `config.inject` on the branch dispatch — but `branch` doesn't have a config. Fallback: wire publish step inline using a `command` step + small inline scripting. Prefer keeping it as a sub-pipeline if workspace sharing works (it does today between ops-pr-review and ops-pr-review-core). |
| `branch` step's `cases.default` keyword conflicts with literal value `default`. | Verified in `internal/pipeline/composition.go:332` — `default` is the explicit fallback key; safe as long as no profile is literally named `default`. |
| `Env` template var collides with shell environment expectations. | Distinct: `{{ env.X }}` reads from sub-pipeline-config-supplied vars only, NOT process env. Document in template.go comment to prevent confusion. |
| Tests for `SubPipelineConfig.Env` and `TemplateContext.Env` may not exist; prior partial commit may have added them. | Check git log on prior branch (if any); rewrite from scratch if not present. Cover: env propagation, default-arm dispatch, missing-key empty resolution. |
| Pipeline schema validation may reject the `env` property until schema is updated. | Update `wave-pipeline.schema.json` in the same PR. |
| Removing `ops-pr-review-core` breaks any external consumer (user-defined pipelines under `.wave/pipelines/`). | Out of scope per memory: pre-1.0.0, no compat shims. Document in PR description and release notes. |
| Inception-bugfix mirror copies might diverge during edit. | Diff `internal/defaults/pipelines/inception-bugfix.yaml` against `.agents/pipelines/inception-bugfix.yaml` before and after edits. |

## Testing Strategy

### Unit tests

- `internal/pipeline/types_test.go` (or existing equivalent): `SubPipelineConfig.Validate` happy path with non-nil `Env`.
- `internal/pipeline/template_test.go`: add cases for `{{ env.foo }}` (set, missing, empty key handling).
- `internal/pipeline/composition_test.go` or `subpipeline_test.go`: parent → child env propagation, child override semantics, branch `default` arm dispatch when env value missing.

### Pipeline integration

- Load both modified pipelines via the existing pipeline loader test (whichever file enumerates and validates all defaults).
- Existing `composition_test.go::TestCompositionExecutor_BranchDispatch` covers `skip` semantics — confirm it still passes.

### End-to-end smoke test (manual / dev-loop)

- `wave run ops-pr-review <pr_url>` against a real PR — confirms full profile (publish posts comment).
- `wave run inception-bugfix <input>` end-to-end — confirms core profile (no comment posted).
- Compare verdict artifact shape between old `ops-pr-review-core` (via git blame baseline) and new inlined steps to confirm parity.

### Lint / build

- `go test -race ./internal/pipeline/...`
- `go build ./...`
- Pipeline schema validation if a CI hook runs against `wave-pipeline.schema.json`.

## Out of Scope

- Migrating all other composition pipelines to the new `Env` mechanism. Only `inception-bugfix` is touched here (it's the only consumer of `-core`).
- Adding CLI `--var key=value` support. The `Env` channel is sub-pipeline-internal for now; CLI invocation just gets defaults.
- Reworking `ops-pr-review-publish` to support multiple posting targets (Slack, email). Existing `gh pr comment` shape preserved.
