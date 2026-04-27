# End-to-end flow — `code-crispies` walkthrough

**Status:** Reference walkthrough for [`onboarding-as-session-plan.md`](onboarding-as-session-plan.md)
**Date:** 2026-04-27
**Subject:** `code-crispies` (Bun + TypeScript, hosted on `git.librete.ch` via Gitea, owner `mwc`)

Every config artifact, env var, file, DB row, and schedule entry that appears in the loop. Trace top to bottom.

---

## Phase 0 — VM boot

User runs:

```bash
docker run --rm -it \
  -e WAVE_PROJECT_REPO=mwc/code-crispies \
  -e WAVE_PROJECT_FORGE=gitea \
  -e WAVE_PROJECT_HOST=git.librete.ch \
  -e TEA_TOKEN=$TEA_TOKEN \
  -e ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY \
  -p 8080:8080 \
  -v code-crispies-state:/work \
  recinq/wave-experiment:latest
```

### Env vars consumed

| Var | Purpose | Required |
|---|---|---|
| `WAVE_PROJECT_REPO` | `<owner>/<name>` to clone | yes |
| `WAVE_PROJECT_FORGE` | `gitea` \| `github` \| `gitlab` | yes |
| `WAVE_PROJECT_HOST` | Forge hostname (e.g. `git.librete.ch`) | yes for non-github |
| `TEA_TOKEN` | Gitea API token (also valid as `GITEA_TOKEN`) | yes for gitea |
| `GH_TOKEN` | GitHub token | yes for github |
| `ANTHROPIC_API_KEY` | Claude API | yes |
| `WAVE_BUDGET_DEFAULT` | Default budget USD per pipeline run | optional, default `1.00` |
| `WAVE_PREVIEW` | Enable `/preview/*` routes | optional |

### Container layout

```
/work/                          (volume — survives container restart)
  code-crispies/                (clone target)
    .agents/                    (created by onboarding)
    .wave/                      (state DB, runs)
    src/, tests/, package.json, …
/etc/wave/
  defaults/                     (binary-shipped pipelines/personas/contracts)
/usr/local/bin/
  wave                          (binary)
  claude-code                   (binary)
  tea                           (binary)
```

### Entrypoint script (`/entrypoint.sh`)

```bash
#!/usr/bin/env bash
set -euo pipefail

cd /work
if [ ! -d "$(basename "$WAVE_PROJECT_REPO")" ]; then
  tea login add --name librete --url "https://$WAVE_PROJECT_HOST" --token "$TEA_TOKEN"
  tea repos clone "$WAVE_PROJECT_REPO"
fi
cd "$(basename "$WAVE_PROJECT_REPO")"

# fresh repos need git identity
git config --local user.email "wave@$(hostname)"
git config --local user.name  "Wave (autonomous)"

# launch webui
exec wave webui --addr 0.0.0.0:8080 --project-dir .
```

---

## Phase 1 — Webui boot, entry-page branch

`wave webui` opens state DB at `.wave/state.db` (creates if missing) + scans for `.agents/.onboarding-done` sentinel.

`GET /`:
- sentinel missing → `302 /onboard`
- sentinel present → `302 /work` (default landing post-feature)

No `wave.yaml` required at this point. The webui boots even with empty project dir (current behaviour: `internal/webui/server.go:90`).

---

## Phase 2 — Auto-detection (no user input yet)

Onboarding service runs a read-only sweep. Same logic that already lives in `internal/onboarding/flavour.go` + `metadata.go` — preserved, ported under new service.

Files inspected:

| Path | Read | What it tells us |
|---|---|---|
| `package.json` | `name`, `scripts`, `type`, `dependencies` | Bun + TS, has `vitest`, `biome`, `tsc` |
| `bun.lockb` | exists | Force Bun runtime over Node |
| `tsconfig.json` | `strict`, `target` | TypeScript strictness level |
| `biome.json` | exists | Lint/format command = `bun biome check` |
| `.gitignore` | exists | Avoid duplicating |
| `README.md` | first 40 lines | Project intent extraction |
| `.gitea/workflows/*.yml` or `.github/workflows/*.yml` | exists | CI provider detection |
| `git remote -v` | first remote | Forge classification (`gitea` matched via host pattern in `internal/forge/detect.go`) |

### Auto-detected struct (in-memory, presented to user)

```yaml
project:
  flavour: typescript
  runtime: bun
  language: typescript
  build_command: bun run build
  test_command: bun test
  contract_test_command: bun test
  typecheck_command: bun run typecheck
  lint_command: bun biome check .
  source_glob: "src/**/*.ts"
  test_glob: "tests/**/*.test.ts"

forge:
  type: gitea
  host: git.librete.ch
  owner: mwc
  repo: code-crispies
  default_branch: main
  cli_tool: tea
  pr_term: pull request
  pr_command: tea pr create

ci:
  provider: gitea-actions
  workflow_path: .gitea/workflows/ci.yml
```

User sees this struct on `/onboard` page as confirmable defaults. They can override any field.

---

## Phase 3 — Onboarding session (Claude Code drives the writes)

### 3.1 Onboarder agent

New shipped default at `internal/defaults/personas/onboarder.md`. Compiled to Claude Code agent file via `PersonaToAgentMarkdown()` (per ADR-015).

```yaml
---
model: sonnet
tools:
  - Read
  - Glob
  - Grep
  - Write
  - Edit
  - Bash
disallowedTools: []
permissionMode: acceptEdits
---

You are the Wave Onboarder. You inspect a project and write project-tailored
.agents/ artifacts. You never modify project source code. You only write under:

- .agents/pipelines/*.yaml
- .agents/personas/*.md
- .agents/prompts/*.md
- .agents/contracts/*.json
- wave.yaml (if absent or user-confirmed)

Workflow: detect → propose → ask user gates → write files → run wave validate → run dry-run smoke
test → mark .agents/.onboarding-done.

NEVER: install dependencies, run lifecycle scripts (npm install, bun install),
modify CI configs, push to remote.
```

### 3.2 Onboarder pipeline

New shipped default at `internal/defaults/pipelines/onboard-project.yaml`. Stub:

```yaml
kind: WavePipeline
metadata:
  name: onboard-project
  description: Tailor Wave to a specific project. Generates .agents/* per detected flavour.
  release: true

input:
  source: cli
  type: string
  schema: { type: string }
  example: "interactive"

pipeline_outputs:
  manifest:
    step: write
    artifact: manifest_summary
    type: string

steps:
  - id: detect
    persona: navigator
    model: cheapest
    workspace: { mount: [{ source: ./, target: /project, mode: readonly }] }
    exec: { type: prompt, source: <prompt-detect.md> }
    output_artifacts: [{ name: detection, path: .agents/output/detection.json, type: json }]
    handover: { contract: { type: json_schema, schema_path: .agents/contracts/detection.schema.json } }

  - id: propose
    persona: onboarder
    model: balanced
    dependencies: [detect]
    memory: { inject_artifacts: [{ step: detect, artifact: detection, as: detection }] }
    workspace: { mount: [{ source: ./, target: /project, mode: readonly }] }
    exec: { type: prompt, source: <prompt-propose.md> }
    output_artifacts: [{ name: proposal, path: .agents/output/proposal.json, type: json }]

  - id: gate-confirm
    type: gate
    dependencies: [propose]
    gate:
      type: approval
      message: "Confirm proposed pipelines, personas, schedules"
      choices: [{ id: approve, target: write }, { id: reject, target: __abort }]

  - id: write
    persona: onboarder
    model: balanced
    dependencies: [gate-confirm]
    memory: { inject_artifacts: [{ step: propose, artifact: proposal, as: proposal }] }
    workspace: { type: worktree, branch: wave/onboard }
    exec: { type: prompt, source: <prompt-write.md> }

  - id: validate
    type: command
    dependencies: [write]
    exec: { type: command, command: "wave validate" }

  - id: smoke
    type: command
    dependencies: [validate]
    exec: { type: command, command: "wave run impl-issue --dry-run --input '{\"forge\":\"gitea\",\"repo\":\"mwc/code-crispies\",\"kind\":\"issue\",\"id\":\"1\"}'" }

  - id: mark-done
    type: command
    dependencies: [smoke]
    exec: { type: command, command: "touch .agents/.onboarding-done" }
```

### 3.3 Inputs the agent gets per step

| Step | Reads | Writes | Contract |
|---|---|---|---|
| `detect` | project files (read-only) | `.agents/output/detection.json` | json schema `detection.schema.json` |
| `propose` | `detection.json` | `.agents/output/proposal.json` | json schema |
| `gate-confirm` | proposal | (gate decision row) | — |
| `write` | proposal | `.agents/pipelines/*.yaml`, `.agents/personas/*.md`, `.agents/prompts/*.md`, `.agents/contracts/*.json`, possibly `wave.yaml` | — |
| `validate` | written `.agents/*` | (none) | exit 0 |
| `smoke` | written `.agents/*` | dry-run trace | exit 0 |
| `mark-done` | (none) | `.agents/.onboarding-done` | — |

### 3.4 User-answerable gates (asked inline as prompt-step `gate` types)

| Gate | Default | Stored in |
|---|---|---|
| Lint failure policy | `rework` (max 2) | proposal artifact |
| Test failure policy | `fail` | proposal artifact |
| Recurring schedules | `[scope on epic, pr-review on new PR]` | proposal artifact |
| Default budget cap per run | `$0.50` | `wave.yaml` |
| Adapter / model tier | `claude / balanced` | `wave.yaml` |
| Onboarder may push branches | `false` | proposal artifact |

---

## Phase 4 — Files written into the repo

### 4.1 `wave.yaml` (top-level manifest, not under `.agents/`)

```yaml
version: 1
project:
  name: code-crispies
  language: typescript
  runtime: bun
  build_command: bun run build
  test_command: bun test
  contract_test_command: bun test
  source_glob: "src/**/*.ts"

forge:
  type: gitea
  host: git.librete.ch
  cli_tool: tea
  pr_command: tea pr create
  pr_term: pull request

runtime:
  default_adapter: claude
  default_tier: balanced
  cost:
    budget_ceiling: 0.50

adapters:
  claude:
    binary: claude
    tier_models:
      cheapest: claude-haiku-4-5
      balanced: claude-sonnet-4-6
      strongest: claude-opus-4-7
```

### 4.2 `.agents/` tree after onboarding

```
.agents/
  .onboarding-done                       (sentinel)
  pipelines/
    impl-issue.yaml                      (Bun-flavoured)
    pr-review.yaml
    scope.yaml
    research.yaml
    pipeline-evolve.yaml                 (meta — copied verbatim from defaults)
  personas/
    craftsman.md                         (Bun + biome aware)
    reviewer.md
    navigator.md
    planner.md                           (read-only)
    onboarder.md                         (copied from defaults — used for re-runs)
  prompts/
    impl-issue/
      implement.md
      verify.md
    pr-review/
      review.md
    scope/
      decompose.md
  contracts/
    impl-issue.json                      (test_suite + json schema)
    pr-review.json
    scope.json
    detection.schema.json
  output/                                (runtime-only, gitignored)
  artifacts/                             (runtime-only, gitignored)
  workspaces/                            (runtime-only, gitignored)
```

### 4.3 Sample generated pipeline — `.agents/pipelines/impl-issue.yaml`

```yaml
kind: WavePipeline
metadata:
  name: impl-issue
  description: Implement a Gitea issue end-to-end on code-crispies (Bun + TS).
  release: true

input:
  source: cli
  type: work_item_ref
  schema: { $ref: "shared:work_item_ref" }
  example: '{"forge":"gitea","repo":"mwc/code-crispies","kind":"issue","id":"142"}'

pipeline_outputs:
  pr:
    step: open-pr
    artifact: pr_ref
    type: pr_ref

steps:

  - id: fetch
    persona: navigator
    model: cheapest
    exec:
      type: prompt
      source_file: prompts/impl-issue/fetch.md
    output_artifacts:
      - { name: issue, path: .agents/artifacts/fetch/issue.json, type: json }

  - id: plan
    persona: planner
    model: balanced
    dependencies: [fetch]
    memory: { inject_artifacts: [{ step: fetch, artifact: issue, as: issue }] }
    exec:
      type: prompt
      source_file: prompts/impl-issue/plan.md
    output_artifacts:
      - { name: plan, path: .agents/artifacts/plan/plan.md, type: markdown }

  - id: implement
    persona: craftsman
    model: balanced
    dependencies: [plan]
    workspace: { type: worktree, branch: "wave/{{ input.id }}" }
    memory:
      inject_artifacts:
        - { step: fetch, artifact: issue, as: issue }
        - { step: plan,  artifact: plan,  as: plan }
    exec:
      type: prompt
      source_file: prompts/impl-issue/implement.md
    handover:
      contract:
        type: test_suite
        command: "bun test && bun run typecheck"
        must_pass: true
        on_failure: rework
        rework_step: implement
        retry: { max_attempts: 2 }

  - id: open-pr
    persona: craftsman
    model: cheapest
    dependencies: [implement]
    workspace: { type: worktree, branch: "wave/{{ input.id }}" }
    exec:
      type: prompt
      source_file: prompts/impl-issue/open-pr.md
    output_artifacts:
      - { name: pr_ref, path: .agents/artifacts/open-pr/pr_ref.json, type: json }
    handover:
      contract:
        type: json_schema
        schema_ref: "shared:pr_ref"
        must_pass: true
        on_failure: fail
```

### 4.4 Sample generated prompt — `.agents/prompts/impl-issue/implement.md`

```markdown
# Implement issue {{ issue.id }}

You are the craftsman. Implement the plan in {{ plan }}.

Context: code-crispies is a Bun + TypeScript CSV processor.

## Required behaviour
- Edit `src/**/*.ts` as needed.
- Add tests under `tests/` mirroring source layout.
- Maintain `bun run typecheck` cleanliness — no `any`, no `// @ts-ignore`.
- Run `bun test` and `bun run typecheck` before reporting done.
- Follow biome formatting (`bun biome check --write` if necessary).

## Forbidden
- Do not modify `package.json`, `bun.lockb`, `tsconfig.json`, `biome.json` unless
  the plan explicitly requires it.
- Do not commit or push.
- Do not use `git add -A`.

## Output
Working tree contains the implementation. The contract step runs the tests.
```

### 4.5 Sample generated contract — `.agents/contracts/impl-issue.json`

```json
{
  "name": "impl-issue",
  "criteria": [
    { "id": "tests_pass",   "weight": 0.5, "description": "bun test passes" },
    { "id": "types_clean",  "weight": 0.3, "description": "bun typecheck reports zero errors" },
    { "id": "lint_clean",   "weight": 0.2, "description": "biome reports zero errors on changed files" }
  ]
}
```

### 4.6 `.gitignore` entries appended

```
# wave runtime
.agents/output/
.agents/artifacts/
.agents/workspaces/
.wave/
```

`.agents/pipelines/`, `.agents/personas/`, `.agents/prompts/`, `.agents/contracts/`, `.agents/.onboarding-done` are **committed**.

---

## Phase 5 — DB rows after onboarding

State DB at `.wave/state.db`. New tables (per main plan PRE-5).

### `pipeline_version`

| pipeline_name | version | sha256 | yaml_path | active | created_at |
|---|---|---|---|---|---|
| `impl-issue` | 1 | `8f3a…` | `.agents/pipelines/impl-issue.yaml` | true | 1714220400 |
| `pr-review`  | 1 | `2c91…` | `.agents/pipelines/pr-review.yaml`  | true | 1714220400 |
| `scope`      | 1 | `b0d7…` | `.agents/pipelines/scope.yaml`      | true | 1714220400 |
| `research`   | 1 | `5ee2…` | `.agents/pipelines/research.yaml`   | true | 1714220400 |

### `worksource_binding`

| id | forge | repo | selector (json) | pipeline_name | trigger | config (json) | active |
|---|---|---|---|---|---|---|---|
| 1 | `gitea` | `mwc/code-crispies` | `{"kind":"issue","label_excludes":["epic"]}` | `impl-issue` | `on_demand` | `{}` | true |
| 2 | `gitea` | `mwc/code-crispies` | `{"kind":"issue","labels":["epic"]}` | `scope` | `on_label` | `{"label":"epic"}` | true |
| 3 | `gitea` | `mwc/code-crispies` | `{"kind":"pr"}` | `pr-review` | `on_open` | `{}` | true |
| 4 | `gitea` | `mwc/code-crispies` | `{"kind":"issue","labels":["auto-impl"]}` | `impl-issue` | `scheduled` | `{"cron":"0 2 * * *"}` | false (opt-in, off by default) |

### `schedule`

| id | pipeline_name | cron_expr | input_ref (json) | active | next_fire_at |
|---|---|---|---|---|---|
| 1 | `impl-issue` | `0 2 * * *` | `{"binding_id":4}` | false | null |

(Empty unless user enabled "nightly auto-impl sweep".)

### `pipeline_eval` — empty

(Populated as runs complete in phase 7.)

### `evolution_proposal` — empty

(Populated as eval signals trigger evolution in phase 8.)

---

## Phase 6 — Dispatch flow: "Run on issue #142"

User clicks `Run impl-issue` on `/work/gitea/code-crispies/142`. Webui:

1. Resolves binding (`worksource_binding.id=1`) → `pipeline_name="impl-issue"`, active version (`pipeline_version` v1).
2. Builds `work_item_ref` payload from forge fetch:

   ```json
   {
     "forge": "gitea",
     "repo":  "mwc/code-crispies",
     "kind":  "issue",
     "id":    "142",
     "title": "Add CSV column-type inference for numeric vs string fields",
     "url":   "https://git.librete.ch/mwc/code-crispies/issues/142"
   }
   ```

3. Calls `service.ExecutorService.Run(ctx, "impl-issue", payload, ExecutionOptions{ Detached: true, Budget: 0.50 })`.
4. Behind the scenes: `wave run impl-issue --input '<json>' --run r/3a8c91 --detach` subprocess (existing pattern, `internal/webui/handlers_control.go:108`).
5. SSE on `/runs/r/3a8c91` streams progress.

### Per-step config resolution (existing ADR-004 hierarchy)

For each step, the executor resolves:

```
adapter:  CLI flag > step.adapter > persona.adapter > wave.yaml runtime.default_adapter   (claude)
model:    CLI flag > step.model   > persona.model   > tier_models[step.tier]              (sonnet-4-6 for "balanced")
budget:   step.budget > pipeline.budget > wave.yaml runtime.cost.budget_ceiling           ($0.50)
```

---

## Phase 7 — After-run signal recording

When run `r/3a8c91` finishes, executor calls `EvolutionService.RecordRun(...)`. Hook lives in pipeline executor on terminal state transitions.

### `pipeline_eval` row written

| pipeline_name | run_id | judge_score | contract_pass | retry_count | failure_class | human_override | duration_ms | cost_dollars | recorded_at |
|---|---|---|---|---|---|---|---|---|---|
| `impl-issue` | `r/3a8c91` | 0.92 | true | 0 | null | false | 348 712 | 0.16 | 1714250000 |

### Sources of each column

| Column | Source |
|---|---|
| `judge_score` | `internal/contract/llm_judge.go` — `JudgeResponse.Score` |
| `contract_pass` | `step.handover.contract` outcome |
| `retry_count` | sum of step-level retries from `step_attempt` |
| `failure_class` | `internal/pipeline/failure.go` 6-class (or null on success) |
| `human_override` | true if any gate decision overrode default |
| `duration_ms` | `pipeline_run.completed_at - started_at` |
| `cost_dollars` | sum of `step_attempt.estimated_cost_dollars` (per ADR-006) |

---

## Phase 8 — Evolution proposal (after N runs OR drift)

`EvolutionService.MaybePropose("impl-issue")` triggers when ANY of:

- `count(*) since last_evolution >= 10`
- `avg(judge_score) over last 10 < 0.80`
- `count(retry_count > 0) over last 10 / 10 >= 0.30`
- `count(failure_class='contract_failure') over last 10 >= 5`

If triggered → spawn `wave run pipeline-evolve --pipeline impl-issue` (reusing existing self-evolution pattern from `wave-test-hardening.yaml`).

### `pipeline-evolve` reads

- Last 10 rows of `pipeline_eval` for `impl-issue`
- Active version yaml: `.agents/pipelines/impl-issue.yaml`
- Last 10 step attempts (failure messages, judge reasoning)

### `pipeline-evolve` writes

- `.agents/proposals/<id>.diff` (unified diff)
- `evolution_proposal` row:

| id | pipeline_name | version_before | version_after | diff_path | reason | signal_summary (json) | status | proposed_at |
|---|---|---|---|---|---|---|---|---|
| 7 | `impl-issue` | 3 | 4 | `.agents/proposals/7.diff` | "judge dropped to 0.78; contract_failure dominant" | `{...}` | `proposed` | 1714280000 |

### Human approves on `/proposals/7`

Webui calls `EvolutionService.Decide(7, approve=true)`:

1. Generate new yaml file: `.agents/pipelines/impl-issue.yaml` (overwrites; full content embedded in diff).
2. Insert `pipeline_version` row (version=4, active=true, sha256 of new yaml).
3. Update prior `pipeline_version` row (version=3, active=false).
4. Update `evolution_proposal.status='approved'`, `decided_at=now`, `decided_by=mwc`.
5. Optional: emit forge event ("Wave evolved impl-issue v3 → v4: …") as repo issue/comment if `forge.evolution_announcements: true`.

### Auto-rollback (config option)

`wave.yaml` may declare:

```yaml
runtime:
  evolution:
    auto_rollback:
      enabled: true
      window: 3      # next 3 runs after activation
      threshold: 0.70  # if avg judge < this, revert
```

If triggered, version 4 is marked inactive, version 3 re-activated, `evolution_proposal.status='superseded'`.

---

## 9. Master config index

Every config touchpoint, single table for skim.

| Layer | File / table | Purpose | Owned by |
|---|---|---|---|
| Container | env vars (`WAVE_*`, tokens) | runtime injection | Docker run |
| Container | `/entrypoint.sh` | clone + git config + boot webui | image |
| Container | `recinq/wave-experiment:latest` | image identity | release |
| Repo | `wave.yaml` | manifest, runtime, forge, budgets, pricing | onboarder writes, user owns |
| Repo | `.agents/.onboarding-done` | sentinel gate | service writes, never edited by user |
| Repo | `.agents/pipelines/*.yaml` | per-project pipelines | onboarder + evolution writes |
| Repo | `.agents/personas/*.md` | per-project personas (tool list, prompt) | onboarder writes |
| Repo | `.agents/prompts/**/*.md` | extracted prompt bodies | onboarder writes |
| Repo | `.agents/contracts/*.json` | contract criteria + json schemas | onboarder + evolution writes |
| Repo | `.agents/proposals/*.diff` | pending evolution diffs | evolution writes |
| Repo | `.gitignore` | `.agents/output/`, `.agents/artifacts/`, `.agents/workspaces/`, `.wave/` | onboarder appends |
| State | `.wave/state.db` | runs, events, evals, proposals, bindings, schedules, versions | StateStore |
| State | table `pipeline_run` | one row per run | executor |
| State | table `step_attempt` | per-step attempt + cost + failure class | executor |
| State | table `pipeline_eval` | per-run signals (NEW) | EvolutionService |
| State | table `pipeline_version` | per-pipeline version history (NEW) | OnboardingService + EvolutionService |
| State | table `evolution_proposal` | pending diffs (NEW) | EvolutionService |
| State | table `worksource_binding` | issue/PR → pipeline mapping (NEW) | OnboardingService + user |
| State | table `schedule` | cron entries (NEW) | OnboardingService + user |
| State | table `webhook` (existing) | optional outgoing notifications | webui admin |
| Forge | `tea login add` token | api auth | env var |
| Forge | git remote `origin` | push target | git config |
| Wave defaults (binary) | `internal/defaults/personas/onboarder.md` | onboarding agent | Wave release |
| Wave defaults (binary) | `internal/defaults/pipelines/onboard-project.yaml` | onboarder pipeline | Wave release |
| Wave defaults (binary) | `internal/defaults/pipelines/pipeline-evolve.yaml` | evolution meta-pipeline | Wave release |
| Wave defaults (binary) | `internal/contract/schemas/shared/work_item_ref.json` | shared schema (NEW) | Wave release |
| Wave defaults (binary) | `internal/defaults/contracts/detection.schema.json` | onboarding detection schema (NEW) | Wave release |
| Adapter | `~/.claude/agents/<persona>.md` | compiled-at-runtime agent file | adapter (ADR-015) |

---

## 10. What user can edit vs. tool-owned

| Editable by user | Edits acceptable | Tool-owned (don't hand-edit) |
|---|---|---|
| `wave.yaml` | yes — language overrides, budgets, schedule cron | — |
| `.agents/pipelines/*.yaml` | yes — but evolution may overwrite. Use `# wave:lock` comment to skip evolution | versions tracked in DB |
| `.agents/personas/*.md` | yes | — |
| `.agents/prompts/**` | yes | — |
| `.agents/contracts/*.json` | yes — but tied to pipeline criteria | — |
| `.agents/.onboarding-done` | no | onboarding service |
| `.wave/state.db` | no | StateStore |
| `.agents/proposals/*.diff` | no | evolution service |
| `.agents/output/`, `.agents/artifacts/` | no — gitignored, transient | runtime |

`# wave:lock` magic comment on top of a pipeline yaml exempts it from evolution. Useful for hand-tuned pipelines the user owns.

---

## 11. Remaining design Qs (specific to this walkthrough)

1. **`work_item_ref` shape.** Add `body` field for full issue text? Or fetch lazily via forge client per step? Recommend lazy — keeps payload small, avoids stale text.
2. **Sentinel format.** Plain `touch`-empty-file or JSON with `{onboarded_at, version}`? JSON wins for future-proofing.
3. **`.agents/pipelines/` lock comments.** Where is the lock signal — top-of-file comment or a manifest list (`runtime.evolution.locked: [impl-issue]`)? Manifest list is queryable; comment is local. Both? Manifest wins.
4. **Auto-rollback on regression.** Default off (let humans decide) or default on with conservative threshold (0.50)? Recommend default off for first release.
5. **Forge events on evolution.** Post a comment on the repo when activating new version? Toggle-able. Off by default.
6. **`wave.yaml` schema migration.** When binary upgrades and adds new fields, how is `wave.yaml` migrated? `wave init --migrate` subcommand? Or auto-migrate on `wave webui` boot?
7. **Multi-repo per VM.** Out of scope for first iteration. Single project per `/work` mount.
