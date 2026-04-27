# Onboarding-as-Session: Plan + Prior Research

**Status:** Draft (proposal — pre-implementation)
**Date:** 2026-04-27
**Author/Driver:** michael.c@re-cinq.com
**Related issues:** (none yet — to file before phase 1)
**Related ADRs:** ADR-003 (layered), ADR-007 (StateStore), ADR-009 (ontology bounded context), ADR-010 (pipeline IO), ADR-011 (Wave Lego), ADR-013 (failure taxonomy), ADR-014 (composition/graph boundary)

---

## 1. Vision

End-state experiment:

1. **Reproducible Docker VM**, always identical: clones a configured private repo (e.g. `code-crispies` from `git.librete.ch` via `tea` CLI), runs setup, installs Wave, runs onboarding.
2. **New entry page**: branches on onboarded vs. not.
   - Onboarded → dashboard.
   - Not onboarded → onboarding session.
3. **Onboarding = Claude-Code session driving Wave**, that:
   - Inspects project, generates project-tailored `.agents/pipelines/*.yaml`, `.agents/personas/*.md`, `.agents/prompts/*.md`, `.agents/contracts/*.json`.
   - Wires work-sources (GitHub Issues phase 1 → Gitea phase 2).
   - Proposes scheduled and on-demand triggers (e.g. "run X on new issue with label Y").
4. **Hide primitives**: user never sees pipelines or runs in normal operation. They see work items (issues/PRs/tasks). Action = "Run on this issue" → JIT param-fill, durable pipeline gets invoked behind scenes.
5. **Self-evolving pipelines**: every ~N runs, judge + eval signals trigger an evolution proposal. Diff gated behind human approve in webui (CLI parity for headless).
6. **CLI parity**: every webui action drivable from CLI. Webui = thin shell over a service layer.

Non-goals (explicit):

- Not building "yet another wizard". Existing `internal/onboarding` wizard treated as throwaway.
- Not making pipelines truly JIT-generated per task — they're durable + reused, evolved on schedule.
- Not multi-tenant. Single-project scope per VM/install.

---

## 2. Prior research summary

### 2.1 ADRs that constrain the design

| ADR | Constraint | Implication |
|---|---|---|
| **003 layered architecture** | depguard-enforced split: Presentation (`webui`, `tui`, `display`, `onboarding`) → Domain → Infrastructure → Cross-cutting. Reverse imports forbidden. | Onboarding orchestrator must NOT live in `internal/onboarding/` (presentation). Move to Domain layer (e.g. `internal/onboarding-svc/` or just `internal/onboarding/` after promotion) and treat current presentation `onboarding` as a thin CLI driver. |
| **007 StateStore consolidation** | All DB access goes through `state.StateStore`. depguard denies `database/sql` outside `internal/state/`. | Any new tables (eval ledger, evolution proposals, pipeline versions, work-source bindings, schedules) extend `StateStore` interface — no new direct DB clients. |
| **009 ontology bounded context** | Pattern: bounded-context package with `Service` interface + `NoOp` + `New(cfg, deps)`. | Reuse exact pattern for new bounded contexts: `internal/onboarding`, `internal/evolution`, `internal/worksource`, `internal/scheduler`. |
| **010 pipeline IO protocol** | Typed `input_ref`, shared schema registry. | New work-source dispatch must use `input_ref: { from: ... }` or `{ literal: ... }`. New canonical schemas needed: `work_item_ref` (generalization of `issue_ref`/`pr_ref`), `schedule_ref`. |
| **011 Wave Lego protocol** | One-step-one-output, canonical artifact path, `on_failure` deterministic, no path templating in contracts. | Generated `.agents/pipelines/*` must be WLP-clean from day one (we're authoring fresh; no migration debt). |
| **013 failure taxonomy** | 6-class enum + `CircuitBreaker` + fingerprinting. | Eval ledger consumes `ClassifiedFailure` directly — don't invent parallel taxonomy. Evolution trigger reads circuit-breaker state. |
| **014 composition/graph boundary** | Composition = pipeline-of-pipelines. Graph = step-level routing. | "Run on issue" = sub-pipeline invocation, NOT a graph dispatcher. Stay in composition layer. |
| **015 persona-agent migration** | Personas compile to Claude Code agent `.md` files. | Generated `.agents/personas/*.md` use frontmatter format. No CLAUDE.md/settings.json assembly. |
| **006 cost** | `BudgetCeiling` + Iron Rule + tier_models. | Evolution loop must respect budget — evolution itself is a pipeline run with cost. |
| **004 multi-adapter** | `AdapterRegistry` w/ resolve hierarchy: CLI > step > persona > default. | Onboarding session = adapter:claude (Claude Code subprocess). Evolution can use cheaper tier. |

### 2.2 Existing components reusable

**Webui** (`internal/webui/`):
- `http.ServeMux` + `html/template` + `//go:embed`. Routes registered via `registerRoutes()` at `routes.go:8`. Entry redirects `/` → `/runs` at `routes.go:13–15`.
- Detached subprocess pattern: `spawnDetachedRun()` at `handlers_control.go:108`. Spawns `wave run --pipeline X --run Y`. Server shutdown does not kill detached runs. SSE for live updates.
- Server boots without `wave.yaml` present (`server.go:90`) — entry-page branching is feasible with no manifest.

**Onboarding** (`internal/onboarding/`):
- `RunWizard()` at `onboarding.go:103` is interactive-only, CLI-coupled, monolithic. Uses `WizardStep` interface but with no UI driver seam.
- Files: `flavour.go`, `metadata.go`, `steps.go`, `wave_command_step.go`, `skill_step.go`, `ontology_step.go`, `state.go`. Memory says "shitty and unused" — treat as **rewrite candidate**, not refactor.
- Sentinel: `.agents/.onboarding-done` gates `wave run` at `cmd/wave/commands/run.go:236`. Preserve gate semantics in new flow.

**ops-bootstrap** (`internal/defaults/pipelines/ops-bootstrap.yaml`):
- 3 steps: `assess` (read-only, navigator) → `scaffold` (worktree, craftsman) → `commit` (worktree, craftsman, no `git add -A`).
- Detects flavour, generates per-language scaffold, commits + optional push.
- **Pattern proven** for greenfield-style bootstrap. Onboarding session pipeline can mirror this shape.

**LLM-as-judge** (`internal/contract/llm_judge.go`):
- `JudgeResponse{ CriteriaResults []CriterionResult, OverallPass bool, Score float64, Summary string }`.
- Already emits criterion-level pass/fail + numeric score. Persisted as step output artifact + `event_log` record. **Reuse as-is** for evolution signal.

**Failure taxonomy** (`internal/pipeline/failure.go:15–81`):
- 6 classes; `step_attempt.failure_class` column persisted. **Query-ready** for "last N runs of pipeline X classified as Y".

**Forge** (`internal/forge/`):
- Clean `Client` interface at `client.go:12–24`. `ForgeGitea` + `ForgeForgejo` enums exist (`detect.go:14–22`). GitHub adapter implemented; Gitea is `ErrNotSupported` stub.
- `classifyHost()` matches `git.librete.ch` (contains "lib"... actually no, host classifier matches literal "gitea"/"forgejo" substrings). **Action:** add explicit allowlist for `git.librete.ch` host or detect via `tea` presence.

**Gates** (`internal/pipeline/gate_handler.go`):
- `GateHandler` interface; `CLIGateHandler`, `AutoApproveHandler`, webui handler.
- Gate types: `approval`, `timer`, `pr_merge`, `ci_pass`. **Reuse for evolution approval.**

**Webhooks** (`internal/state/types.go:273–287`):
- `Webhook{ Events, Matcher, Secret, Active }`, `WebhookDelivery` table. Event-driven only — NOT a scheduler.

**Scheduler / cron**: does not exist. **Net-new** infrastructure needed.

### 2.3 Architecture-audit findings (current state)

Critical anti-patterns in current codebase blocking the feature:

1. **No service layer.** `cmd/wave/commands/run.go:234–350` and `internal/webui/handlers_control.go:346–430` independently load manifest, instantiate executor, set up adapter, open state store. Duplication, not parity.
2. **Pipeline loading scattered.** `loadPipeline()` in run.go:925–951, `loadPipelineYAML()` in handlers_control.go:723–752, `LoadPipelineByName()` in tui/pipelines.go:79–111 — three implementations, all filesystem-scan-per-call.
3. **In-process executor surface is private.** `webui.launchInProcess()` is a fallback only. No public API for "run pipeline in-process from non-CLI caller."
4. **Onboarding has no UI-driver seam.** `WizardStep` interface exists but each step is monolithic; webui can't intercept prompts.
5. **No eval-signal aggregation.** `pipeline_outcome` table exists but no "last N runs" or "judge score trend" queries.
6. **No pipeline versioning.** Pipelines are files. Evolution implies versioning. Schema gap.
7. **Forge: Gitea is stubbed.** Implementation needed for `git.librete.ch`.
8. **No scheduler.** Cron / recurring trigger infrastructure absent.

---

## 3. Pre-conditions (must land before feature work)

Ordered by dependency. Each is a separable PR. None block existing functionality.

### PRE-1. Service layer extraction

**Goal:** Single seam both CLI and webui call into. Required for parity claim to be real.

**New package:** `internal/service/` (Domain layer per ADR-003).

```
internal/service/
  pipeline.go    — PipelineService: Load, Validate, List, Discover
  executor.go    — ExecutorService: Run(ctx, name, input, opts) -> (runID, error)
  onboarding.go  — OnboardingService (post PRE-2)
  evolution.go   — EvolutionService (phase 3)
```

**Sites to migrate:**
- `cmd/wave/commands/run.go:234–350` → call `ExecutorService.Run`.
- `internal/webui/handlers_control.go:346–430` → same.
- `cmd/wave/commands/run.go:925–951`, `handlers_control.go:723–752`, `internal/tui/pipelines.go:79–111` → all call `PipelineService.Load`.

**Layer compliance:** Service in Domain. Both Presentation packages (`cmd`, `webui`, `tui`) consume Domain. depguard happy.

**Risk:** Touches load-bearing CLI run path. Migrate behind feature flag in CLI initially, gate on test passes.

### PRE-2. Onboarding rewrite (delete old wizard)

**Goal:** Replace `internal/onboarding` with thin Domain service + per-driver UI seam.

**Delete:** existing `RunWizard()` machinery (per memory: "unused and shitty"). Keep `flavour.go` and `metadata.go` (detection logic is reusable).

**New shape:**
```go
// internal/onboarding/service.go (Domain)
type Service interface {
  IsOnboarded(projectDir string) bool
  StartSession(ctx context.Context, projectDir string, opts StartOptions) (*Session, error)
  Resume(ctx context.Context, sessionID string) (*Session, error)
  Status(sessionID string) (*Status, error)
}

type UI interface { // implemented per-driver
  PromptString(question Question) (string, error)
  PromptChoice(question Question) (string, error)
  Notify(event Event) error
}
```

**Drivers:**
- `cmd/wave/commands/init.go` — `CLIOnboardingUI` (terminal prompts).
- `internal/webui/handlers_onboard.go` — `WebUIOnboardingUI` (HTTP form + SSE).

**Sentinel preserved:** `.agents/.onboarding-done` continues gating `wave run` (used by both drivers).

### PRE-3. Forge: Gitea adapter + tea CLI

**Goal:** `git.librete.ch` works.

**Implement:** `internal/forge/gitea.go` with `GiteaClient` implementing `forge.Client`. Backed by:
- Either `tea` CLI subprocess (mirror gh-CLI pattern), or
- HTTP client against Gitea API (`/api/v1/...`).

**Decision pending:** subprocess-via-tea is faster to ship but adds binary dep. HTTP is cleaner and tea-token-compatible. **Recommend HTTP** with `tea` only as auth-helper fallback.

**Detect:** Add explicit hostname `git.librete.ch` → `ForgeGitea` mapping in `detect.go:classifyHost()`. Generic hostname patterns (substring "gitea") already work for self-hosted.

### PRE-4. Pipeline registry + asset cache

**Goal:** Eliminate filesystem scan per request. Hot-reload during dev.

**New:** `pkg/registry/assets.go` — in-memory cache keyed by name, watched via `fsnotify`. Used by `PipelineService`.

**Scope:** pipelines, personas, contracts, prompts under `.agents/`.

**Deferrable:** can ship without it; quality-of-life only. Defer if PRE-1..3 are slow.

### PRE-5. StateStore extensions for new tables

**Goal:** All new persistence goes through `state.StateStore` per ADR-007. Don't bypass.

**New tables (additive only, no existing column changes):**

```sql
-- evolution signal aggregation
CREATE TABLE pipeline_eval (
  pipeline_name TEXT NOT NULL,
  run_id TEXT NOT NULL,
  judge_score REAL,
  contract_pass BOOLEAN,
  retry_count INTEGER,
  failure_class TEXT,
  human_override BOOLEAN,
  duration_ms INTEGER,
  cost_dollars REAL,
  recorded_at INTEGER NOT NULL,
  PRIMARY KEY (pipeline_name, run_id)
);

-- pipeline versioning (sha256 of yaml + signature)
CREATE TABLE pipeline_version (
  pipeline_name TEXT NOT NULL,
  version INTEGER NOT NULL,
  sha256 TEXT NOT NULL,
  yaml_path TEXT NOT NULL,  -- e.g. .agents/pipelines/impl-issue.v3.yaml
  active BOOLEAN NOT NULL,
  created_at INTEGER NOT NULL,
  PRIMARY KEY (pipeline_name, version)
);

-- evolution proposals awaiting human approve
CREATE TABLE evolution_proposal (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  pipeline_name TEXT NOT NULL,
  version_before INTEGER NOT NULL,
  version_after INTEGER NOT NULL,
  diff_path TEXT NOT NULL,  -- .agents/proposals/<id>.diff
  reason TEXT NOT NULL,     -- human-readable trigger summary
  signal_summary TEXT NOT NULL, -- JSON
  status TEXT NOT NULL,     -- proposed | approved | rejected | superseded
  proposed_at INTEGER NOT NULL,
  decided_at INTEGER,
  decided_by TEXT
);

-- work-source bindings (issue → pipeline mapping)
CREATE TABLE worksource_binding (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  forge TEXT NOT NULL,      -- github | gitea | ...
  repo TEXT NOT NULL,       -- owner/name
  selector TEXT NOT NULL,   -- JSON: { labels:[], state:'open', ... }
  pipeline_name TEXT NOT NULL,
  trigger TEXT NOT NULL,    -- on_demand | on_label | on_open | scheduled
  config TEXT,              -- JSON: cron, debounce, etc
  active BOOLEAN NOT NULL,
  created_at INTEGER NOT NULL
);

-- recurring schedules
CREATE TABLE schedule (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  pipeline_name TEXT NOT NULL,
  cron_expr TEXT NOT NULL,
  input_ref TEXT,           -- JSON input or work-source query
  active BOOLEAN NOT NULL,
  next_fire_at INTEGER,
  last_run_id TEXT,
  created_at INTEGER NOT NULL
);
```

**Methods on `StateStore`:** `RecordEval`, `GetEvalsForPipeline(name, limit)`, `CreatePipelineVersion`, `ActivateVersion`, `CreateProposal`, `DecideProposal`, `ListBindings`, `CreateBinding`, `ListSchedules`, `UpdateScheduleNextFire`.

### PRE-6. Scheduler infrastructure

**Goal:** Cron-like recurring runs, plus debounced work-source polling.

**Two options:**
- (a) In-process: goroutine in webui server tick loop. Dies on server stop. Cheap.
- (b) External: separate `wave scheduler` long-running process. Survives webui restart.

**Recommend (a) with --detach option** — webui already supports detached subprocess, so launching scheduler from webui boot satisfies both modes. Reuse `spawnDetachedRun` pattern.

**New:** `internal/scheduler/` (Domain layer, ADR-009 pattern).

```go
type Scheduler interface {
  Tick(ctx context.Context) error  // called every minute
  Schedule(spec ScheduleSpec) error
  Cancel(id int64) error
  List() ([]ScheduleSpec, error)
}
```

Uses `cron` expression parser (single-binary constraint: implement small parser or vendor `robfig/cron/v3` if dep budget allows; check existing go.mod).

---

## 4. Architecture (target)

```
┌─────────────────────────────────────────────────────────────┐
│ Presentation                                                │
│   cmd/wave/commands  ──┐                                    │
│   internal/webui     ──┤                                    │
│   internal/tui       ──┤                                    │
└────────────────────────┼────────────────────────────────────┘
                         │
                         ▼ ALL go through service layer
┌─────────────────────────────────────────────────────────────┐
│ Domain                                                      │
│   internal/service                                          │
│     ├ PipelineService    (load/validate/list)               │
│     ├ ExecutorService    (run pipelines, in-proc + detach)  │
│     ├ OnboardingService  (session orchestration + UI seam)  │
│     ├ EvolutionService   (signal aggregate → proposal)      │
│     ├ WorkSourceService  (forge → work items, bindings)     │
│     └ SchedulerService   (cron, debounced polls)            │
│                                                             │
│   internal/onboarding   (Service iface + NoOp + Real)       │
│   internal/evolution    (signal types, judge aggregator)    │
│   internal/worksource   (work_item_ref, dispatch)           │
│   internal/scheduler    (cron + tick loop)                  │
│   internal/pipeline     (executor — unchanged structurally) │
│   internal/contract     (judge — unchanged)                 │
│   internal/forge        (+ gitea.go)                        │
└─────────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│ Infrastructure                                              │
│   internal/state      (extended schema, all SQL here)       │
│   internal/workspace, worktree, github                      │
└─────────────────────────────────────────────────────────────┘
```

### 4.1 New entry-page flow

```
GET /
 ├ ProjectStatus.IsOnboarded(projectDir)?
 │   yes → render dashboard
 │   no  → redirect /onboard
GET /onboard
 ├ OnboardingService.StartSession()
 │   spawns subprocess: claude-code --agent .agents/agents/onboarder.md
 │   subprocess writes .agents/pipelines/, .agents/personas/, etc
 │   webui streams stdout via SSE to user
 │   user answers questions via web form (POST /onboard/answer)
 │     → persisted to session state
 │     → next prompt injected into subprocess via stdin or sentinel file
 ├ on completion: write .agents/.onboarding-done
 └ redirect /
```

### 4.2 Work-source dispatch flow

```
GET /work
 ├ WorkSourceService.List() — federates github + gitea forges
 │   shows unified list of issues / PRs / tasks
GET /work/<forge>/<repo>/<id>
 ├ render item + applicable bindings + "Run pipeline" button
POST /work/<forge>/<repo>/<id>/run
 ├ binding lookup: which pipeline for this item shape?
 ├ ExecutorService.Run(pipelineName, work_item_ref{...}, opts)
 ├ redirect to /runs/<runID>
```

### 4.3 Evolution loop

```
After every pipeline run completes:
  EvolutionService.RecordRun(pipelineName, runID, signals)
    → INSERT pipeline_eval row
    → if (count_since_last_evolution >= N) OR drift_detected:
        EvolutionService.ProposeEvolution(pipelineName)
        → spawn `wave run pipeline-evolve --pipeline <name>`
          (a meta-pipeline that reads last N evals + current yaml,
           emits proposed yaml diff)
        → INSERT evolution_proposal row, status=proposed
        → notify webui (badge on pipeline page)
Human in webui clicks Approve:
  EvolutionService.Decide(id, approve=true)
    → write .agents/pipelines/<name>.v(N+1).yaml
    → flip pipeline_version.active
    → archive old version
```

### 4.4 Hide primitives

Default routes for normal users:
- `/` — work board (issues/PRs/tasks)
- `/runs/<id>` — accessible when user clicks active run
- `/proposals` — pending evolution approvals (badge-driven)

Power-user routes (kept, but not promoted):
- `/pipelines`, `/personas`, `/contracts`, `/skills`, `/compose` — admin section

---

## 5. Data model summary (additive only)

See PRE-5. Five new tables; ~10 new `StateStore` methods. Zero existing column changes. SQLite migration via `internal/state/migration_definitions.go` pattern (per ADR-007).

New canonical schema (per ADR-010): `internal/contract/schemas/shared/work_item_ref.json` — generalizes `issue_ref` / `pr_ref` for forge-agnostic work items.

```json
{
  "type": "object",
  "required": ["forge", "repo", "kind", "id"],
  "properties": {
    "forge": { "enum": ["github", "gitea", "gitlab", "bitbucket"] },
    "repo": { "type": "string" },        // owner/name
    "kind": { "enum": ["issue", "pr", "task"] },
    "id": { "type": "string" },
    "title": { "type": "string" },
    "url": { "type": "string", "format": "uri" }
  }
}
```

---

## 6. Phased delivery

### Phase 0 — Pre-conditions (~3 sprints)

| # | Title | Files | Test gate |
|---|---|---|---|
| 0.1 | PRE-1 service layer | new `internal/service/{pipeline,executor}.go`; migrate `cmd/wave/commands/run.go` + `internal/webui/handlers_control.go` | run + webui smoke pipelines green |
| 0.2 | PRE-3 forge/Gitea | new `internal/forge/gitea.go`; `detect.go` host map | unit tests + `git.librete.ch` integration |
| 0.3 | PRE-5 schema + StateStore methods | `internal/state/migration_definitions.go`, `state/store.go` | migration up/down, method tests |
| 0.4 | PRE-2 onboarding rewrite (skeleton) | new `internal/onboarding/service.go`; delete old wizard; CLI driver | `wave init --yes` works (non-interactive baseline) |
| 0.5 | PRE-6 scheduler | `internal/scheduler/`, `service/scheduler.go` | cron tick test |
| 0.6 | PRE-4 asset registry | optional / deferrable | — |

### Phase 1 — Onboarding-as-session (~2 sprints)

| # | Title | Files |
|---|---|---|
| 1.1 | Onboarder agent + meta-pipeline | new `.agents/personas/onboarder.md`, new `internal/defaults/pipelines/onboard-project.yaml` (assess → propose → write `.agents/*` → smoke-test) |
| 1.2 | Webui driver | new `internal/webui/handlers_onboard.go`, new templates `onboard/start.html`, `onboard/step.html`, SSE stream |
| 1.3 | CLI driver | `cmd/wave/commands/init.go` — call `OnboardingService.StartSession` via CLI UI |
| 1.4 | Entry-page branch | `internal/webui/routes.go` — `/` checks `IsOnboarded`, redirects appropriately |

### Phase 2 — Work sources + dispatch (~2 sprints)

| # | Title | Files |
|---|---|---|
| 2.1 | `work_item_ref` shared schema | `internal/contract/schemas/shared/work_item_ref.json` + registry |
| 2.2 | WorkSourceService + bindings | `internal/service/worksource.go`, `internal/worksource/`, table CRUD |
| 2.3 | Webui `/work` board + detail | new templates, handlers; replaces dashboard as default landing |
| 2.4 | "Run on this issue" button | binding lookup → `ExecutorService.Run` |

### Phase 3 — Evolution loop (~2 sprints)

| # | Title | Files |
|---|---|---|
| 3.1 | EvalSignal types + recording hook | `internal/evolution/signal.go`; hook into executor on run-complete |
| 3.2 | `pipeline-evolve` meta-pipeline | `internal/defaults/pipelines/pipeline-evolve.yaml` (read evals → generate diff via judge persona) |
| 3.3 | Trigger heuristics | EvolutionService: every N runs OR judge-score drift OR retry-rate spike |
| 3.4 | Approval webui + CLI | `/proposals` route; `wave proposals list/approve/reject` |

### Phase 4 — Reproducible Docker VM (~1 sprint)

| # | Title | Files |
|---|---|---|
| 4.1 | Dockerfile (thin base + wave + claude-code + tea) | new `docker/Dockerfile.experiment` |
| 4.2 | Entrypoint script | clones repo via tea, runs `wave init`, boots `wave webui` |
| 4.3 | Volume layout | `.agents/` + `.wave/` mounted persistent |
| 4.4 | Smoke-test compose | `docker-compose.experiment.yml` for local repro |

---

## 7. Open questions for user

1. **Webui auth.** Single-user assumed (Docker VM is per-user). Confirm? Or need basic auth / token?
2. **Onboarder persona scope.** Should onboarder be allowed to install dependencies (`npm install`, `go mod download`) or only write config? Implications for sandbox.
3. **Evolution N.** Confirm `N=10` as default trigger threshold, or smaller for faster iteration during experiment.
4. **Schedule storage.** Approve in-process scheduler (option a) or want external `wave scheduler` daemon (option b)?
5. **`tea` vs HTTP for Gitea.** Recommend HTTP. Confirm or override.
6. **Docker base.** Alpine or Debian? Pre-bake claude-code or `npm i -g` at boot?
7. **Existing pipelines.** Do we keep them on existing names (`impl-issue`, `impl-speckit`, etc.) or generate fresh from onboarder per-project? Recommend: keep defaults shipped in binary; onboarder writes overlay copies in `.agents/pipelines/` only when project demands customization.

---

## 8. Risks

| Risk | Mitigation |
|---|---|
| Service-layer refactor destabilizes CLI run path | Land PRE-1 behind feature flag; keep old code path until smoke-tests green |
| Wave Lego protocol violations in generated pipelines | Onboarder writes pipelines, then runs `wave doctor` + `wave validate` before persisting; reject invalid YAML |
| Evolution proposes regressive changes | Mandatory human approve gate; signed diff in `.agents/proposals/`; rollback = re-activate prior version row |
| Subprocess-driven onboarding hangs | Hard timeout per session; SSE heartbeat; user can abort + retry |
| Webhook-style work-source events miss items | Polling fallback every 5min; idempotent dispatch (binding `last_seen_at`) |
| Gitea API differs from documented spec at `git.librete.ch` | Smoke-test against real instance during PRE-3, before phase 2 |
| User's "JIT pipeline" framing → over-design | Decision logged: pipelines durable, evolved on schedule. Not regenerated per task. |

---

## 9. References

**ADRs read:** 002, 003, 004, 005, 006, 007, 008, 009, 010, 011, 013, 014, 015.

**Codebase entry points indexed:**
- Webui server: `internal/webui/server.go:90`
- Webui routes: `internal/webui/routes.go:8`
- Detached run: `internal/webui/handlers_control.go:108`
- In-process run: `internal/webui/handlers_control.go:235`
- CLI run: `cmd/wave/commands/run.go:234`
- CLI init: `cmd/wave/commands/init.go:36`
- Onboarding wizard: `internal/onboarding/onboarding.go:103` (slated for delete)
- Pipeline executor: `internal/pipeline/executor.go` (~6700 LOC; unchanged structurally)
- Judge: `internal/contract/llm_judge.go:56-69`
- Failure classifier: `internal/pipeline/failure.go:15-81`
- Forge client: `internal/forge/client.go:12-24`
- ops-bootstrap pipeline: `internal/defaults/pipelines/ops-bootstrap.yaml`

**Memory keys consulted:** versioning policy, .agents convention, claude-code tool availability, workspace path resolution, pipeline unification, frictionless factory feedback, validation feedback, no-emojis feedback.
