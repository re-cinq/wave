# ADR-006: Cost Infrastructure — Token Split, Pricing Matrix, and Budget Enforcement

## Status

Accepted (Partial — pricing + budget + Iron Rule live; schema migration pending)

## Date

2026-03-28 (proposed) — 2026-04-26 (accepted)

## Implementation Status

Landed:
- Pricing matrix in `internal/cost/ledger.go` (`DefaultPricing` map covers Claude / OpenAI / Gemini models with per-million input/output rates).
- Budget enforcement: `manifest.Runtime.Cost.BudgetCeiling` (USD); per-step check in `executor.go` (~line 3435) calls `cost.CheckIronRule()` and budget guard.
- Token split: `adapter.AdapterResult` exposes `TokensIn` / `TokensOut` alongside legacy `TokensUsed`.
- Iron Rule: `internal/cost/ironrule.go` enforces deployment-time prompt-size limits (alternative safety guardrail in lieu of a decision-log table).

Pending:
- `schema.sql` not yet extended with `tokens_input`, `tokens_output`, `estimated_cost_dollars` columns on `step_attempt`; aggregate cost reporting is approximate.
- Decision log table deferred indefinitely; Iron Rule covers the highest-priority guardrail.

## Context

Wave tracks token usage at five levels (pipeline_run, event_log, performance_metric, step_progress, step_attempt) but stores only aggregate integer counts. It cannot calculate costs, enforce budgets, or inform model selection decisions with financial data. This gap is significant for autonomous pipeline execution, where detached runs can accumulate unbounded costs without operator visibility or control.

An ontology mapping analysis between GSD-2 (a single-agent TypeScript coding framework) and Wave identified six candidate features for adoption: token optimization profiles, dynamic model routing, cost ledger, decision log, lock file crash detection, and Iron Rule enforcement. While the two systems differ fundamentally — Wave is a multi-agent DAG orchestrator with SQLite persistence, GSD-2 is a single-agent linear state machine with file-driven state — the analysis surfaced real gaps in Wave's cost infrastructure that exist independently of the GSD-2 comparison.

Wave's multi-adapter architecture (Claude, OpenCode, Codex, Gemini, Browser) means costs vary dramatically by provider and model. The existing four-tier model resolution hierarchy (CLI > step > persona > adapter default) selects models without cost awareness. Operators running 3-6 concurrent pipelines in autonomous mode have no mechanism to cap spend.

Several in-flight ADRs constrain the solution space:

- **ADR-002** (Extract StepExecutor) targets a 40-50% reduction in executor.go — new features should target the StepExecutor boundary, not deepen the monolith.
- **ADR-003** (Layered Architecture) defines four layers with dependency rules — cost tracking belongs in the infrastructure layer, budget enforcement is cross-cutting.
- **ADR-013** (Failure Taxonomy, formerly ADR-004) proposes `budget_exhausted` as a canonical failure class — budget enforcement directly implements this.
- **ADR-005** (Graph Execution Model) replaces TopologicalSort with NextSteps — cost tracking must work with both the current DAG executor and the future graph scheduler.

## Decision

Implement **cost infrastructure first**: input/output token split, an embedded pricing matrix, per-step cost calculation, and budget enforcement with hard ceilings. Defer dynamic model routing, decision logging, and Iron Rule enforcement until real cost data informs whether they justify their complexity.

## Options Considered

### Option 1: Comprehensive Adoption — All Six GSD-2 Features

Adopt all six candidates simultaneously: token optimization profiles, dynamic model routing, cost ledger, decision log, lock file crash detection, and Iron Rule enforcement. Implement as a coordinated effort across adapter, state, relay, pipeline, and contract packages.

**Pros:**
- Maximizes value extraction — every viable pattern is adopted in one pass
- Features are mutually reinforcing: cost ledger enables cost-aware routing, Iron Rule prevents budget overruns, decision log provides audit trail
- Single coordinated migration avoids repeated schema changes

**Cons:**
- Large blast radius across 8+ packages — high risk of destabilizing in-flight ADRs (002, 003, 004)
- Lock file crash detection is low-value since SQLite state persistence already enables pipeline resumption
- Conflicts with ADR-002's goal of reducing executor.go complexity — adding six features before extraction compounds the problem
- No feature ships until all are ready, delaying time-to-first-value

### Option 2: Cost Infrastructure First — Cost Ledger + Token Split + Budget Enforcement (Recommended)

Focus on the financial infrastructure layer: extend the state schema with input/output token split, add an embedded pricing matrix to wave.yaml, compute per-step and per-pipeline costs, and implement budget ceilings that trigger the `budget_exhausted` failure class.

**Pros:**
- Addresses the highest-value gap: Wave has zero cost visibility despite tracking tokens at five levels
- Input/output token split is a prerequisite for accurate cost calculation and for any future dynamic routing — avoids rework
- Budget enforcement directly implements ADR-013's `budget_exhausted` failure class
- Contained scope: primarily touches state (schema.sql, types.go, store.go) and manifest (pricing config)
- Aligns with ADR-003: cost tracking is infrastructure layer, budget enforcement is cross-cutting
- Enables operators to set cost ceilings on autonomous pipeline runs — the most urgent operational need

**Cons:**
- Defers dynamic model routing — operators still assign models manually
- Pricing matrix requires manual maintenance when providers change prices; no auto-update in a static binary
- Schema migration adds columns to high-traffic tables (step_attempt, performance_metric)
- Cost calculation accuracy depends on adapters reporting input/output token counts correctly

### Option 3: Observability Bundle — Cost Ledger + Decision Log + Iron Rule

Adopt three features that enhance visibility and safety without changing execution behavior: cost ledger, decision log as a structured artifact type, and Iron Rule enforcement as a hard ceiling layered on the existing RelayMonitor.ShouldCompact infrastructure.

**Pros:**
- Pure observability: adds cost visibility, decision audit trail, and safety enforcement without changing execution semantics
- Iron Rule layers directly on existing RelayMonitor infrastructure
- Decision log fills a genuine gap in post-mortem analysis

**Cons:**
- Decision log as a new artifact/contract type adds conceptual complexity to the already rich contract system
- Iron Rule at 90% creates a second threshold alongside the 80% relay compaction trigger — two overlapping mechanisms
- Still defers dynamic model routing — the most impactful cost-reduction feature
- Three features across different subsystems still require coordination

### Option 4: Incremental Cherry-Pick — Cost Ledger Only, Then Evaluate

Adopt only the cost ledger as the first deliverable. Evaluate each remaining feature independently based on observed cost data and operator feedback, with separate lightweight ADR evaluations.

**Pros:**
- Lowest risk: single feature with well-understood scope
- Real cost data informs whether other features are actually needed
- Each subsequent feature gets fresh evaluation against the codebase after prior ADRs land
- Easiest to course-correct

**Cons:**
- Slowest path to full value — each feature requires its own evaluation cycle
- Repeated schema migrations instead of a batched change
- Operators running autonomous pipelines lack budget enforcement until a later iteration
- Misses synergies: cost ledger + budget enforcement are more valuable together

### Option 5: Wave-Native Evolution — Skip GSD-2 Mapping Entirely

Evolve Wave's existing infrastructure toward cost visibility and budget control using Wave-native concepts, without adopting GSD-2 terminology or patterns. Extend token tracking with cost calculation. Use the existing model resolution hierarchy for cost hints. Leverage the failure taxonomy for budget enforcement.

**Pros:**
- No conceptual impedance mismatch — Wave's multi-agent DAG is fundamentally different from GSD-2's single-agent linear state machine
- Avoids importing foreign terminology (Iron Rule, fidelity profiles) that may confuse contributors
- May arrive at better designs unconstrained by GSD-2's approach

**Cons:**
- Loses the analytical leverage of the GSD-2 comparison that identified real gaps
- Risk of reinventing solutions GSD-2 already validated
- No concrete deliverable — this is a philosophical stance, not an implementation plan
- Prioritization becomes open-ended without the GSD-2 feature list as a roadmap

## Consequences

### Positive

- Wave gains cost visibility for the first time — operators can see per-step and per-pipeline cost estimates
- Budget enforcement enables safe autonomous execution by capping spend with hard ceilings
- Input/output token split creates the data foundation for future dynamic model routing and optimization profiles without pre-committing to those features
- Budget ceilings integrate with ADR-013's failure taxonomy via the `budget_exhausted` class, maintaining architectural consistency
- Real cost data from production pipelines will replace speculation when evaluating whether to adopt dynamic routing, decision logging, or Iron Rule enforcement

### Negative

- Embedded pricing matrix becomes a maintenance burden — prices must be updated in source when providers change rates
- Adapters that do not report input/output token splits separately will produce less accurate cost estimates (graceful degradation to aggregate-based estimation)
- Schema migration touches high-traffic tables, requiring testing with realistic data volumes
- Operators who want cost-aware model selection must wait for a future phase

### Neutral

- Adapter interface may need an optional `TokenDetail` struct alongside the existing aggregate `TokensUsed` field
- TUI and WebUI progress displays will need updates to surface cost data
- `wave.yaml` schema expands to include a `pricing` section under `runtime` or `adapters`
- Documentation needs to cover budget configuration, pricing matrix format, and how cost estimates are calculated

## Implementation Notes

### Schema Changes

1. Add `tokens_input` and `tokens_output` columns to `step_attempt` and `performance_metric` tables alongside the existing `tokens_used` (which becomes the sum for backward compatibility)
2. Add `estimated_cost_dollars REAL` to `step_attempt`, `performance_metric`, and `pipeline_run` tables
3. Add `budget_max_dollars REAL` columns to pipeline and step configuration in state

### Pricing Matrix

Add an embedded pricing configuration to `wave.yaml`:

```yaml
runtime:
  pricing:
    claude:
      claude-sonnet-4-6:
        input_per_million: 3.00
        output_per_million: 15.00
      claude-opus-4-6:
        input_per_million: 15.00
        output_per_million: 75.00
    # ... other adapters/models
```

Pricing defaults are embedded in the binary; `wave.yaml` overrides allow operators to update without rebuilding.

### Budget Enforcement

1. Add `budget` fields to pipeline and step manifest configuration:
   ```yaml
   pipelines:
     impl-issue:
       budget:
         max_dollars: 5.00
       steps:
         implement:
           budget:
             max_dollars: 2.00
   ```
2. After each step completion, calculate cost and check against the pipeline-level ceiling
3. When exceeded, emit `budget_exhausted` failure (per ADR-013 taxonomy) and halt the pipeline

### Adapter Interface

Extend the adapter result type to optionally include `TokensInput` and `TokensOutput` alongside the existing `TokensUsed`. Adapters that cannot split tokens continue reporting the aggregate — cost calculation falls back to a blended rate.

### Affected Files

- `internal/state/schema.sql` — new columns and migration
- `internal/state/types.go` — extended token and cost fields
- `internal/state/store.go` — cost accumulation queries
- `internal/manifest/types.go` — pricing and budget configuration types
- `internal/manifest/parser.go` — parsing pricing and budget from wave.yaml
- `internal/adapter/adapter.go` — extended result type with token split
- `internal/pipeline/executor.go` — cost calculation and budget checks at step boundaries
- `wave.yaml` — pricing matrix and budget defaults

### Migration Plan

1. Schema migration adds new columns with NULL defaults — no data loss, existing runs continue to work
2. Adapters are updated incrementally to report token splits — those not yet updated report NULL for input/output, and cost calculation uses a blended rate
3. Budget enforcement is opt-in: pipelines without `budget.max_dollars` have no ceiling
4. After real cost data accumulates, a follow-up ADR evaluates whether dynamic model routing or Iron Rule enforcement should be adopted
