# ADR-011: Wave Lego Protocol

## Status
Accepted

## Date
2026-04-21

## Context

[ADR-010](010-pipeline-io-protocol.md) introduced typed pipeline inputs and
outputs and a canonical shared-schema registry. Typed wiring (`input_ref:
{from|literal}`) is validated at load time. That ADR framed pipelines as
composable Lego blocks at the *boundary*: what flows in, what flows out.

Phase-4 validation of the shipped pipelines under ADR-010 surfaced a second
class of defect — not in the typed boundary, but in the mechanics every
pipeline uses to declare step outputs, address them in templates, and hand
them over to child pipelines. The same free-form latitude that allowed
ADR-010's untyped baseline also enables a familiar family of bugs:

1. **Artifact-as-directory collision (`impl-recinq`).** A step declared an
   output artifact whose canonical path was a directory the agent was
   simultaneously asked to write files into. `writeOutputArtifacts`
   attempted to overwrite the directory node and failed silently; downstream
   steps saw a stale path. Root cause: no convention enforcing
   `.agents/artifacts/<step-id>/<output-name>` as the only valid artifact
   location, so authors reused `.agents/<artifact>.json` and other ad-hoc
   paths that happened to shadow directories.

2. **Non-deterministic `{{ step.output.field }}` (`ops-epic-runner`).** The
   `scope` step produces three named outputs — `scope`, `child_issues`,
   `plan`. All three are registered under keys with the prefix `scope:` in
   `execution.ArtifactPaths`. `resolveStepOutputRef("{{ scope.output.field
   }}")` walks the map looking for any key starting with `scope:` and
   returns the first JSON file that contains `field`. Map iteration order
   is non-deterministic, so the same pipeline sometimes resolved the
   template against `scope:child_issues` and sometimes against
   `scope:plan`, with silently different content. (Mitigated by
   [8c2de05c](https://github.com/re-cinq/wave/commit/8c2de05c) which probes all candidates, but that masks rather
   than fixes the ambiguity.)

3. **`on_failure: retry` without `rework_step` (`impl-issue`, many others).**
   Most shipped contracts use `on_failure: retry`. The executor historically
   accepted "retry" as a contract outcome and re-ran the step from scratch.
   This duplicates (with different semantics) the step-level retry policy,
   makes it impossible to answer "what is this contract's deterministic
   outcome on failure?" from the YAML, and dead-ended when pipelines wanted
   `rework` semantics (repair via a different step). `retry` is accepted by
   `dryrun.go` but only partially wired through the executor, so the
   behavior differs between contract types.

4. **Path drift in `plan-scope`.** Three separate pipeline_outputs
   (`scope`, `child_issues`, `plan`) all declared `path:
   .agents/artifact.json`. Each step wrote to the same relative path and
   then the artifact archiver copied to
   `.agents/artifacts/<step>/<output>`, but agents reading
   `.agents/artifact.json` in later steps saw whichever step wrote it
   last. The template system hid the collision because path templating
   (`{{ step.id }}` in the path) was nominally allowed.

5. **Sub-pipeline handover via free-text.** Legacy `input: "{{ scope.output
   }}"` on a sub-pipeline step hands over a blob of text whose shape is
   whatever the producer step happens to write today. `impl-issue` consumes
   a JSON issue reference, but `scope` produced three different JSON
   shapes across its outputs; if the executor picked the wrong one the
   child failed with a confusing "missing issue number" at runtime rather
   than a load-time wiring error.

Relevant code and files:
- `internal/pipeline/executor.go` — `runNamedSubPipeline`,
  `writeOutputArtifacts`, `resolveStepOutputRef`.
- `internal/pipeline/iotypes.go`, `internal/pipeline/dag.go` — load-time
  validation seams.
- `internal/pipeline/types.go` — `PipelineOutput`, `StepInput`,
  `RetryConfig`.
- `internal/contract/contract.go` — `ContractConfig.OnFailure`.
- `.agents/pipelines/*.yaml` — 33 pipelines affected by at least one of
  the five bugs above.
- ADR-010, `docs/adr/010-pipeline-io-protocol.md`.

These bugs share a common shape: Wave lets authors get the mechanics
right-ish but never says **"the mechanics are these exact seven things and
anything else is wrong."** Each pipeline author rediscovered the
conventions, and each rediscovery produced a slightly different wrong
answer. The goal of this ADR is to name the conventions and enforce them
at load time.

## Decision

Adopt the **Wave Lego Protocol (WLP)** — seven rules that govern how steps
declare outputs, how authors address them, how contracts handle failures,
and how sub-pipelines hand data to each other. Load-time enforcement lands
in this PR (softened to warnings where shipped pipelines would break);
hard enforcement lands in the phase-5 migration PR.

**The seven rules:**

1. **One step = one primary output.** A step declares exactly one canonical
   output artifact; any additional artifacts are marked `secondary: true`.
   *Justification:* the ambiguity that drove bugs (2) and (4) disappears
   once "what does this step produce?" has a single unambiguous answer.

2. **Canonical artifact path.** Every artifact lives at
   `.agents/artifacts/<step-id>/<output-name>.<ext>`. No free-form `path:`
   overrides in pipeline YAML.
   *Justification:* eliminates bugs (1) and (4). The convention also gives
   the executor, the archiver, and the retention policy a single path
   grammar to understand.

3. **Typed output required.** Every `pipeline_outputs[*]` entry sets
   `type:` explicitly. `string` is valid but must be spelled out.
   *Justification:* ADR-010's "default to string" saved legacy pipelines
   but blocked `TypedWiringCheck` from catching mis-wired compositions
   in many cases. Explicit types make type-check coverage the rule.

4. **Named-output template addressing.** The canonical template syntax is
   `{{ step.out.<name> }}` for the raw artifact content and
   `{{ step.out.<name>.field }}` for JSON-path extraction. Legacy
   `{{ step.output }}` / `{{ step.output.field }}` continues to resolve
   for backward compatibility but emits a one-shot deprecation warning
   per step per run.
   *Justification:* deterministic key lookup (`<stepID>:<name>` in
   `ArtifactPaths`) kills bug (2). No more prefix-scan guessing.

5. **Deterministic `on_failure` values.** Contracts must use one of `fail`,
   `skip`, `continue`, `rework` (with `rework_step`), or `warn`. `retry`
   is **not** a valid contract outcome; step-level retries go on the
   step's `retry:` block.
   *Justification:* contract outcomes become answerable at load time. Bug
   (3) goes away. Separation of concerns: contracts assert; retries
   reschedule. The step-level `retry.max_attempts` already covers
   retries with exponential backoff.

6. **No path templating in contracts.** Contract `schema_path`,
   `criteria_path`, `source`, `spec_artifact`, and related path fields
   must be literal strings. No `{{ step.id }}` or `{{ run_id }}` in
   contract paths.
   *Justification:* contracts are validated at load time. Path templating
   defers path resolution to runtime and makes it impossible to confirm
   at load that a required schema file exists or is readable. Bug (4) is
   partly a consequence of this.

7. **Typed sub-pipeline handover.** Composition steps (`pipeline:`,
   `iterate:`, `branch:`) use `input_ref: {from: "<step>.<output>"}` or
   `input_ref: {literal: "<value>"}`. Legacy `input: "<template>"` still
   works for string-typed children but emits a deprecation warning.
   *Justification:* ADR-010 introduced this seam and this rule makes it
   mandatory in practice. Bug (5) is closed when every handover carries
   a declared type that `TypedWiringCheck` can verify.

### Enforcement

Two code changes in this PR implement WLP enforcement without breaking
the ~33 pipelines that currently violate rule 5 or rule 3:

- `internal/pipeline/iotypes.go` gains `CollectWLPLoadWarnings(p)`, which
  the YAML loader calls after `ValidatePipelineIOTypes`. It returns
  non-fatal deprecation notices for:
  - contracts using `on_failure: retry` (rule 5)
  - `pipeline_outputs` entries without explicit `type:` (rule 3)
- `internal/pipeline/executor.go` `resolveStepOutputRef` learns the
  typed addressing form `{{ step.out.<name> }}` (rule 4) and emits a
  one-shot ADR-011 deprecation warning each time the legacy
  `{{ step.output }}` form resolves successfully inside a run.

Rules 1, 2, 6, and 7 are stylistic / structural and land alongside the
phase-5 migration PR, where the shipped pipelines are rewritten to comply.
Once compliant, `CollectWLPLoadWarnings` is promoted to
`ValidatePipelineIOTypes` as hard errors.

## Options Considered

### Option 1: Runtime-only guards

Keep the status quo of "load accepts anything, executor papers over
ambiguity." Harden `resolveStepOutputRef` and `writeOutputArtifacts` to
be more tolerant (e.g., probe all candidates — already done in
[8c2de05c](https://github.com/re-cinq/wave/commit/8c2de05c)).

**Pros:**
- Zero breaking changes.
- No pipeline migration needed.

**Cons:**
- Doesn't fix the *author-experience* bug: writing a WLP-violating pipeline
  still "works" until it doesn't, at which point the debugging path is a
  runtime stack trace in the wrong step.
- Every future bug of the same class compounds.
- "Papering over" is how we arrived at three different resolver strategies
  in `resolveStepOutputRef`, each hiding a different author mistake.

### Option 2: Hard-error on WLP violations in this PR

Promote every rule to a load-time error immediately.

**Pros:**
- No soft/hard transition window.
- Forces immediate migration.

**Cons:**
- Breaks 33+ shipped pipelines and their load-tests in a single commit,
  blocking every other in-flight refactor.
- No migration guide yet; authors would have to reverse-engineer the
  rules from failing tests.

### Option 3: Warn-then-error with ADR (**chosen**)

Soft-warn in this PR with explicit ADR reference; hard-error after the
migration PR rewrites shipped pipelines. Pair with the additive
`{{ step.out.<name> }}` template so migrations can land incrementally.

**Pros:**
- Backward-compatible.
- Migration is visible (deprecation warnings) but non-blocking.
- The ADR exists before the migration commits, so every PR that edits a
  pipeline has a canonical reference.
- New pipelines written post-ADR can be WLP-clean from day one.

**Cons:**
- Short-lived "warn but pass" phase where authors might ignore warnings.
  Mitigated by hard-erroring in the next PR.

## Consequences

### Positive
- Five known bug classes are structurally prevented, not patched.
- Pipeline authors have a single page that answers "how do I declare an
  output / address it / hand it to a child?"
- `resolveStepOutputRef` becomes deterministic when authors migrate —
  same inputs, same output, no map-order surprise.
- Contract outcomes become load-time answerable: pipeline readers can
  tell what happens on failure without reading executor source.
- The addressing rule (rule 4) pairs with the already-deterministic
  `<stepID>:<name>` key in `ArtifactPaths`, so the executor's own
  invariants become expressible in the template language.

### Negative
- Phase-5 migration touches ~33 pipeline YAMLs. This is bounded and
  mechanical.
- Two template forms (`{{ step.output }}` and `{{ step.out.<name> }}`)
  coexist temporarily. The loader and executor both emit deprecation
  warnings for the legacy form to drive migration.
- Docs under `docs/guide/contracts.md` and `docs/guide/pipelines.md` show
  `on_failure: retry` in examples — they will need a follow-up pass.

### Neutral
- No schema changes to `ContractConfig`, `PipelineOutput`, or `Step`.
- No changes to the artifact DB, sandbox, adapter, or webui.
- ADR-010's `input_ref` seam is reused as-is (rule 7 codifies its
  mandatory use rather than inventing a new mechanism).
- `CollectWLPLoadWarnings` is decoupled from `ValidatePipelineIOTypes`
  so the phase-5 PR can swap the warn/error distinction by moving
  checks between the two functions.

## Implementation Notes

Files modified in this PR:
- `internal/pipeline/iotypes.go` — `CollectWLPLoadWarnings` helper; rule 5
  and rule 3 soft-checks.
- `internal/pipeline/iotypes_test.go` — coverage for the new checks and
  loader wiring.
- `internal/pipeline/dag.go` — `Unmarshal` populates `Pipeline.Warnings`
  after type-check.
- `internal/pipeline/types.go` — `Pipeline.Warnings` runtime-only field.
- `internal/pipeline/executor.go` — `resolveStepOutputRef` supports
  `{{ stepID.out.<name> }}`; `validatePipelineAndCreateContext` emits
  warnings at preflight; `warnLegacyStepOutputOnce` guards against
  per-run warning spam.
- `internal/pipeline/resolve_step_output_ref_test.go` — new test file
  covering typed and legacy addressing.
- `docs/adr/011-wave-lego-protocol.md` — this file.

Files **not** modified in this PR:
- `.agents/pipelines/*.yaml` — migration is phase 5.
- `internal/defaults/pipelines/*.yaml` — same.
- `internal/contract/*` — no contract schema changes.
- `docs/guide/*` — doc refresh is phase 5.

### Migration plan (phase 5, separate PR)

1. Rewrite all 33 pipelines with `on_failure: retry` to the appropriate
   deterministic outcome (`rework` with `rework_step`, `continue`, or
   `fail`). Keep step-level `retry.max_attempts` where retries are
   genuinely wanted.
2. Add `type:` to every `pipeline_outputs[*]` entry. Use `string` for
   free-text outputs rather than omitting the field.
3. Rewrite every step's `output_artifacts` to use one primary output plus
   zero or more `secondary: true` artifacts.
4. Replace every `{{ stepID.output }}` and `{{ stepID.output.field }}`
   with `{{ stepID.out.<name> }}` / `{{ stepID.out.<name>.field }}`.
5. Replace every `input: "<template>"` on sub-pipeline steps with
   `input_ref: {from: "<step>.<output>"}`.
6. Remove any `{{ ... }}` templating from contract path fields.
7. Promote `CollectWLPLoadWarnings` checks to `ValidatePipelineIOTypes`
   hard errors.
8. Refresh `docs/guide/contracts.md`, `docs/guide/pipelines.md`, and the
   archived migration / workflow docs that still show `on_failure:
   retry` examples.

### Non-goals for this PR

- No content-addressable artifact store (still phase-2 of ADR-010).
- No rewrite of the shipped pipelines — strictly code + ADR.
- No changes to runtime contract validation semantics — only the
  accepted set of `on_failure` values at load time.
- No renaming of `pipeline_outputs`, `input_ref`, or existing keys.
