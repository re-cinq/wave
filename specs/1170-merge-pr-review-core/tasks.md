# Work Items

## Phase 1: Code Plumbing (env passthrough)

- [X] 1.1: Add `Env map[string]string \`yaml:"env,omitempty"\`` field to `SubPipelineConfig` in `internal/pipeline/types.go`. Update inline doc comment.
- [X] 1.2: Extend `TemplateContext` in `internal/pipeline/template.go` with `Env map[string]string` field; update `NewTemplateContext` signature or add setter.
- [X] 1.3: Add `case strings.HasPrefix(expr, "env."):` branch in `resolveExpression`. Missing keys return empty string. Update doc-comment in `ResolveTemplate`.
- [X] 1.4: In `internal/pipeline/composition.go::executeSubPipeline` (or `subpipeline.go` runner construction), copy parent env into child context, then overlay `step.Config.Env`.
- [X] 1.5: Update `internal/defaults/schemas/wave-pipeline.schema.json` `SubPipelineConfig` definition to include `env` (object, additionalProperties: {type: string}).

## Phase 2: Tests for env passthrough

- [X] 2.1: Add template_test cases for `{{ env.foo }}` resolution (set / unset / multi-key) [P].
- [X] 2.2: Add composition or subpipeline test for parent→child env propagation and override [P].
- [X] 2.3: Add branch-dispatch test covering `cases.default` arm when env value missing [P].

## Phase 3: New publish pipeline

- [X] 3.1: Create `internal/defaults/pipelines/ops-pr-review-publish.yaml`. Single-step pipeline. Persona `{{ forge.type }}-commenter`. Lift the publish prompt verbatim from current `ops-pr-review.yaml`. `release: false`. Input `pr_ref`, output `findings_report`.
- [X] 3.2: Mirror to `.agents/pipelines/ops-pr-review-publish.yaml` (verify byte-identical with diff).

## Phase 4: Merge core into ops-pr-review

- [X] 4.1: In `internal/defaults/pipelines/ops-pr-review.yaml`: replace the `core-review: { pipeline: ops-pr-review-core }` step with the 5 inlined steps (diff-analysis → security-review → quality-review → slop-review → summary) lifted verbatim from `ops-pr-review-core.yaml`. Adjust step IDs only if needed for uniqueness; preserve dependencies as in core.
- [X] 4.2: Replace the inline `publish` prompt step with a `branch` step keyed on `{{ env.profile }}`: `core: skip`, `default: ops-pr-review-publish`. Keep `dependencies: [summary]`.
- [X] 4.3: Update `pipeline_outputs` block: `verdict` references `step: summary, artifact: verdict`; expose `diff` from `step: diff-analysis` to match the deleted core's outputs.
- [X] 4.4: Update top-level `chat_context.artifact_summaries` and `metadata.description` to reflect the unified shape.
- [X] 4.5: Mirror to `.agents/pipelines/ops-pr-review.yaml`.

## Phase 5: Update consumers

- [X] 5.1: Edit `internal/defaults/pipelines/inception-bugfix.yaml` `review` step: change `pipeline: ops-pr-review-core` → `pipeline: ops-pr-review` + add `config: { env: { profile: core } }`.
- [X] 5.2: Mirror to `.agents/pipelines/inception-bugfix.yaml`.

## Phase 6: Delete the fork

- [X] 6.1: `git rm internal/defaults/pipelines/ops-pr-review-core.yaml`
- [X] 6.2: `git rm .agents/pipelines/ops-pr-review-core.yaml`

## Phase 7: Documentation

- [X] 7.1: Update `docs/adr/010-pipeline-io-protocol.md` — strike `ops-pr-review-core` from the I/O table (~line 233) and the typed-pipelines list (~line 18). Adjust the count.
- [X] 7.2: Update `docs/guides/contract-chaining.md` — replace `ops-pr-review-core` reference(s) with the unified pipeline + profile-flag example [P].
- [X] 7.3: Update `AGENTS.md` — replace `ops-pr-review-core` mentions with profile-flag guidance [P].

## Phase 8: Validation

- [X] 8.1: `go build ./...` — confirms type/wiring compiles.
- [X] 8.2: `go test -race ./internal/pipeline/...` — unit tests pass.
- [X] 8.3: `go test ./...` — full suite green (or scoped if too slow).
- [X] 8.4: Load both modified pipelines through the loader test (or a quick `wave list` if available) to confirm schema validation passes.
- [X] 8.5: Manual smoke: `wave run ops-pr-review <test_pr_url>` against a non-prod PR — confirm publish step posts comment.
- [X] 8.6: Manual smoke: `wave run inception-bugfix <finding_input>` — confirm core path completes without posting comment.
- [X] 8.7: Verify `git diff` between `internal/defaults/pipelines/` and `.agents/pipelines/` mirrors stays in sync.

## Phase 9: Polish

- [X] 9.1: Update PR description with profile-flag usage example and migration note for any external consumers of `-core`.
- [X] 9.2: Confirm spec/plan/tasks markdown committed under `specs/1170-merge-pr-review-core/`.
