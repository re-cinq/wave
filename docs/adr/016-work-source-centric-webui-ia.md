# ADR-016: Work-Source-Centric Webui IA and Onboarding-as-Session UX

## Status
Proposed

## Date
2026-04-27

## Context

Wave's current webui is **pipeline-centric**. The default route redirects `/` → `/runs`
(`internal/webui/routes.go:13–15`); the primary nav exposes `/runs`, `/pipelines`,
`/personas`, `/contracts`, `/skills`, `/compose`, `/issues`, `/prs`, `/health`. Pipelines
and runs are surfaced as the primary objects users interact with.

The proposed onboarding-as-session experiment (see
[`docs/scope/onboarding-as-session-plan.md`](../scope/onboarding-as-session-plan.md))
flips this model: the user thinks in **work items** (issues, PRs, scheduled tasks).
Pipelines are an implementation detail — durable, evolved on a schedule, picked just-in-time
by binding rules. UI mockups in [`docs/scope/mockups/`](../scope/mockups/) explored the new
shape. Walking the mockups through the `code-crispies` example walkthrough surfaced a set of
substantive UX decisions that should be captured before any code lands, because the
service-layer refactor in PRE-1..6 of the plan is shaped by these decisions.

The current webui has additional UX gaps documented in
[`docs/webui-ux-audit.md`](../webui-ux-audit.md) (P3/P4 polish items remain). Those are
**not** the subject of this ADR — this ADR is about an information-architecture (IA) shift
that subsumes most of them by re-grounding the UI on a different primary object.

Adjacent decisions already accepted constrain the design:

- **ADR-003** (Layered Architecture) places `webui` in Presentation. The IA change must not
  introduce reverse imports.
- **ADR-007** (StateStore consolidation) requires all DB access to flow through
  `state.StateStore`. New tables (`worksource_binding`, `evolution_proposal`,
  `pipeline_eval`, `schedule`, `pipeline_version`) extend the interface — no direct SQL.
- **ADR-009** (bounded-context pattern) — every new domain concern (`worksource`,
  `evolution`, `scheduler`) follows the `Service` interface + `NoOp` + `New(cfg, deps)`
  shape.
- **ADR-010 / ADR-011** — generated pipelines from onboarding must be Wave-Lego clean from
  day 1.
- **ADR-013** (failure taxonomy) — the eval signals surfaced in the evolution-proposal UI
  consume the canonical 6-class taxonomy and per-step `failure_class` column directly.
- **ADR-015** (persona-as-agent-md) — the onboarder agent compiles to a Claude Code agent
  `.md` file via `PersonaToAgentMarkdown()`, no special webui plumbing needed.

Existing wizard machinery in `internal/onboarding/RunWizard()` is CLI-bound, has no UI
driver seam, and per project memory is "unused and shitty" — slated for replacement, not
preservation.

## Decision

Adopt a **work-source-centric information architecture** for the webui, with onboarding
delivered as a **streaming Claude-Code session**, pipelines hidden behind a per-item "Run"
affordance, and pipeline evolution surfaced as a first-class approval queue.

This ADR captures **nine inter-locking UX decisions** that together define the new shape.
Each is a binding constraint on the implementation work tracked in
[`onboarding-as-session-plan.md`](../scope/onboarding-as-session-plan.md) and
[`preview-route-plan.md`](../scope/preview-route-plan.md).

### D1. Default landing route is `/work`, not `/runs` or `/pipelines`

Onboarded projects redirect `/` → `/work`. The work-board lists issues, PRs, and scheduled
tasks across all connected forges in a single unified list. `/runs` and `/pipelines` remain
reachable but are demoted to an "Admin" section in the nav.

### D2. Onboarding is a chat-style streaming session, not a stepwise wizard

Not-onboarded projects redirect `/` → `/onboard`. The page renders a chat-like log of
agent activity, with structured **inline gate prompts** for user input (radio choices,
checkbox lists, freeform fields) interleaved with agent messages and a live stream block of
tool invocations. Forms submit answers back to the running session via SSE-paired POST,
and the agent continues. There is no separate "next step" / "back" wizard navigation. A
condensed stepper at the top (`detect → inspect → propose → scaffold → smoke-test → commit`)
gives the user a sense of progress without forcing linear stepping.

### D3. Pipelines are hidden by default; the work item is the primary surface

The user's primary interaction loop is:

1. See work item (issue / PR / scheduled task) on `/work`.
2. Click into `/work/<forge>/<repo>/<id>`.
3. Click **Run** — the binding-resolved default pipeline runs.

Pipelines are **never picked from a separate page** in the default flow. Power users can
override via an "advanced options" disclosure that exposes pipeline picker, adapter
override, model tier, budget, and detached mode.

### D4. Bindings are the IA glue between work items and pipelines

A `worksource_binding` row maps `(forge, repo, selector)` → `pipeline_name + trigger`. The
work-item detail page shows applicable bindings as radio choices ("usual pick", "research
only", "decompose if epic-shaped") with cost + duration estimates next to each. Onboarding
proposes an initial binding set; users can add/edit bindings inline from the detail page
("Auto-run on label X?").

### D5. Cost is surfaced before dispatch, not only after the fact

Every dispatch affordance (work-item Run button, scheduled-task list, evolution
re-trigger) shows an estimated cost (`~$0.18`) and budget cap. The estimate is derived
from the active pipeline version's average cost over the last N runs of that pipeline (per
ADR-006 cost ledger). On detail pages, a "Live cost forecast" side-card shows estimated /
budget-cap / adapter / tier.

### D6. Evolution proposals are a first-class nav item with a badge

A `Proposals` link in the primary nav shows pending `evolution_proposal` rows, with a
yellow badge counter. The proposal-detail page is an **approval review** UI: unified diff
view of the YAML changes, side-card listing the trigger signals (judge score delta, retry
rate, top failure class, avg cost), a replay forecast ("if activated, last 10 runs would
have re-classified as: 7 still-pass / 2 newly-fail / 1 newly-pass"), and prominent
**Approve & activate** / **Reject** buttons. No partial-approval / line-by-line editing —
proposals are atomic.

### D7. Auto-rollback is configurable but defaults to off

`wave.yaml` may declare `runtime.evolution.auto_rollback.{enabled,window,threshold}`.
Default is **off** — humans decide rollbacks for the first release. This decision is
reviewable per major version; if telemetry shows humans rubber-stamping rollbacks we
default it on.

### D8. SVG icons only — no emojis anywhere in the UI

Reaffirms the existing project policy ("never use emojis in Wave UI, only SVG icons or
HTML entities" — feedback memory). All status badges, action icons, and decorations use
inline SVG. Status states (open / running / failed / scoped) use coloured pills with an
inline SVG glyph. This ADR captures the policy for new surfaces; legacy surfaces are
already compliant.

### D9. Multi-forge federation is built into the work-board, not a separate switcher

The work-board filter bar exposes a `All forges / gitea: code-crispies / github:
re-cinq/mp` selector, but the default view aggregates across all connected forges in one
list. Each row tags its forge inline (`gitea / code-crispies`). There is no per-forge
landing page. The nav meta-bar shows the **current project** (`code-crispies` ·
`git.librete.ch`) but a project may federate across multiple forges (e.g. its own gitea
repo + a downstream github mirror).

## Options Considered

### Option 1: Polish current pipeline-centric webui (status quo + UX polish)

Keep `/runs` as default, address P3/P4 items in `webui-ux-audit.md`, optionally add a
`/work` page later as a secondary surface alongside `/runs` and `/pipelines`.

**Pros:**
- Zero blast radius. No service-layer refactor required.
- Existing webui muscle memory preserved for current users.
- Polish work is already inventoried.

**Cons:**
- Does not address the actual product question — current users (small) report that the
  pipeline-centric UI is opaque to non-Wave-experts. The mockup walkthrough on
  `code-crispies` showed how much friction it adds to dispatch a pipeline against a
  GitHub/Gitea issue today.
- Onboarding stays a CLI-only stepwise wizard, blocking the headless-Docker-VM use case.
- Evolution still has no surface; pipelines remain manually curated, contradicting the
  self-evolving direction.
- "Hide complexity" is half-implemented at best.

### Option 2: Work-source-centric IA + onboarding-as-session (chosen)

The nine decisions above. Replaces the default landing, hides pipelines, makes onboarding
a streaming session, surfaces evolution.

**Pros:**
- Aligns the UI with how a per-project deployment is actually used: user opens a forge
  issue, picks a binding, gets a PR.
- Onboarding becomes runnable from the webui (not just CLI), which unblocks the
  reproducible-Docker-VM target.
- Pipeline evolution is finally visible — without a UI, a self-evolving pipeline product
  is just unobservable behaviour.
- Bindings are an explicit object users can reason about; cost is surfaced; gating is
  explicit.
- Mockups exist (`docs/scope/mockups/`) as concrete review artifacts.

**Cons:**
- Service-layer refactor is non-negotiable to deliver this — see PRE-1..6.
- Existing users (devs who built Wave with the current UI in mind) lose familiar
  surfaces; admin pages cover the gap but require nav demotion.
- Multi-forge federation requires Gitea adapter (`internal/forge/gitea.go`) and
  generalised `work_item_ref` schema (per ADR-010).

### Option 3: Two parallel UIs ("classic" + "work-board")

Ship work-board as a secondary IA at `/work` while keeping `/runs` as the default.

**Pros:**
- Reduces switching friction for current users.
- Can A/B / opt-in.

**Cons:**
- Two IAs means two surfaces to maintain; coupling between them creates more code, not
  less.
- Defeats the "hide pipelines" point of decision D3 — if `/pipelines` and `/runs` are
  still primary nav, users still mental-model on pipelines.
- Decision deferred is decision avoided: makes ratchet-forward harder.

### Option 4: Move webui to a SPA framework (React, Vue, htmx)

Treat the IA shift as a chance to migrate from `html/template` + light JS to a richer SPA.

**Pros:**
- Some interactions in the mockups (chat-style streaming, inline form replies, live
  stream block) are easier in a SPA.

**Cons:**
- Out of scope for a UX/IA decision. SPA migration is its own ADR with its own
  cost/benefit analysis.
- Adds dependencies, build pipeline, hydration concerns, and accessibility risk that the
  current `html/template` + small JS stack does not have.
- The mockups were authored as plain HTML and remained legible with no JS — this is
  evidence that the IA shift does not require a SPA.
- Wave is a single-static-binary product; embedding a SPA bundle adds fragility.

## Consequences

### Positive

- **Single coherent IA**: every new feature surfaces under a clear primary object (work
  item, run, proposal, schedule, binding) — no more "where does this go?" debates.
- **Onboarding becomes a product surface**: reproducible Docker VM target unblocked;
  CLI users get the same flow with a CLI driver of the same `OnboardingService`.
- **Evolution is observable**: proposals queue is a first-class operator surface, not
  buried in logs.
- **Cost-aware dispatch by default**: every Run button is annotated with an estimate;
  surprise costs become rare.
- **Mockups are review artifacts**: nine decisions are visible in
  `docs/scope/mockups/*.html` and can be reviewed by clicking through, not by reading
  prose.
- **CLI parity is enforced architecturally**: every webui action goes through the same
  service interface CLI uses; drift is caught at the test boundary.

### Negative

- **Refactor cost upfront**: PRE-1..6 must land before any of this ships. The
  service-layer extraction touches `cmd/wave/commands/run.go`, `internal/webui/handlers_*`,
  pipeline loading, and onboarding.
- **Two-IA transition window**: during phase A/B/C of the preview-route plan, current and
  new IA coexist. Surface drift is a risk; mitigated by build-tag gating and a hard
  "PREVIEW" banner.
- **Bindings UX is novel**: users have to learn what a binding is. Mitigation: the
  onboarder proposes the initial set with sensible defaults, so users encounter bindings
  as something already configured rather than something they must author.
- **Cost estimates rely on history**: a freshly onboarded project has no run history, so
  the estimate is "—" or a coarse default. Acceptable; gets better with use.

### Neutral

- **Old webui pages remain reachable**: `/runs`, `/pipelines`, `/personas`, etc. are
  demoted to admin nav. They are not deleted.
- **Existing event/SSE infrastructure is reused**: the chat-style streaming session in D2
  rides the existing event broker.
- **No DB schema impact from this ADR alone**: the new tables required for D4–D7 are
  motivated by the main feature plan, not this ADR. This ADR locks in their *meaning*,
  not their *introduction*.
- **Theme stays dark for first release**: light theme is non-blocking; can land as a
  follow-up.
- **Mobile is non-blocking**: the webui's primary use is desktop / VM-internal browser.
  Mobile responsiveness is a P3 item, not a launch gate.

## Implementation Notes

This ADR is binding on the implementation tracked in:

- [`docs/scope/onboarding-as-session-plan.md`](../scope/onboarding-as-session-plan.md) §6
  (phased delivery)
- [`docs/scope/preview-route-plan.md`](../scope/preview-route-plan.md) (preview-route
  migration phases A → D)
- [`docs/scope/codecrispies-walkthrough.md`](../scope/codecrispies-walkthrough.md) (config
  surfaces affected by each decision)

### Decision-to-code mapping

| Decision | Where it lands |
|---|---|
| D1 default `/work` | `internal/webui/routes.go` — change `/` redirect after onboarded sentinel check |
| D2 onboarding session | `internal/webui/handlers_onboard.go` (NEW) + `internal/service/onboarding.go` (NEW per PRE-2) + `internal/onboarding/service.go` (rewrite) |
| D3 pipelines hidden | `internal/webui/templates/layout.html` — nav reorganization; admin section gated |
| D4 bindings | new `worksource_binding` table (see plan PRE-5) + `internal/service/worksource.go` (NEW) + `internal/webui/handlers_work.go` (NEW) |
| D5 cost forecast | side-card partial reading from existing `pipeline_eval` aggregations + ADR-006 cost ledger |
| D6 proposals queue | `internal/webui/handlers_proposals.go` (NEW) + `internal/service/evolution.go` (NEW per PRE-1.5) |
| D7 auto-rollback | `wave.yaml` schema extension; `internal/evolution` consults config on activation |
| D8 SVG icons | already policy; new templates inline SVG only |
| D9 multi-forge | `internal/forge/gitea.go` (NEW per PRE-3) + `work_item_ref` shared schema (per ADR-010 phase 2) |

### Migration path

The decisions land **incrementally** through the preview-route plan:

1. **Phase A** (static fixtures under `/preview/*`, build-tagged): all nine decisions are
   visible in markup form. Reviewers click through and validate.
2. **Phase B** (stub services): D2 / D4 / D6 wired to stub service implementations
   returning fixture data. Validates that service interfaces match UI needs.
3. **Phase C** (real data): each route per work item / proposal / binding switches to live
   data as underlying domain services land.
4. **Phase D** (promote): preview build tag dropped, `/preview/*` routes replace the
   default IA. Old `/runs`-rooted routes move to `/admin/*`.

### Reversibility

Per phase:

- Phase A → B: drop preview templates; no impact on default build.
- Phase B → C: roll back per-route by switching back to fixtures.
- Phase C → D: requires re-deploying the prior binary if reverting (no DB schema rollback
  needed since new tables are additive).

After phase D, partial reversal is possible by re-routing `/work` to redirect back to
`/runs` and re-promoting old nav items. The new tables remain (additive only).

### Out of scope for this ADR

- Detailed visual design tokens (colors, spacing). Mockups establish a working palette;
  refinement is design polish, not architecture.
- Light-theme support.
- Mobile responsive breakpoints below 480px.
- Internationalization.
- Authentication model (single-user assumed for first release; multi-user is a future
  ADR).

### Validation

- Five trusted reviewers (project memory says only `nextlevelshit` admin is trusted on
  Wave; one reviewer is sufficient) click through `docs/scope/mockups/index.html` and
  confirm each of D1–D9 is visible / makes sense in context.
- Open questions in
  [`onboarding-as-session-plan.md`](../scope/onboarding-as-session-plan.md) §7,
  [`preview-route-plan.md`](../scope/preview-route-plan.md) §10, and
  [`codecrispies-walkthrough.md`](../scope/codecrispies-walkthrough.md) §11 are resolved
  before phase A starts.
- Once accepted, this ADR's status flips to **Accepted** and the implementation issues are
  filed.
