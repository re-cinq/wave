# feat: automatic run retrospectives with structured friction analysis

**Issue**: [re-cinq/wave#578](https://github.com/re-cinq/wave/issues/578)
**Labels**: enhancement
**Author**: nextlevelshit

## Context

Fabro generates automatic retrospectives after every run — combining quantitative metrics (duration, cost, retries, files touched per step) with an LLM-narrated analysis that rates smoothness, identifies friction points, captures learnings, and flags open items. These feed into DuckDB for cross-run analytics.

Wave has the raw infrastructure (events, audit logs, state DB) but no automated analysis. This is a high-value addition that creates a feedback loop for improving pipelines and personas.

## Design Goals — Best of Both Worlds

Combine Fabro's retrospective concept with Wave's existing event/state infrastructure.

### Retrospective Generation

After each pipeline run completes (success or failure), generate a retrospective:

**Phase 1 — Quantitative (immediate, no LLM)**:
- Per-step: duration, retry count, adapter used, model used, exit code, files changed
- Aggregate: total duration, total steps, success/failure ratio, total retries

**Phase 2 — Narrative (LLM-powered, optional)**:
- Smoothness rating (5-point scale: Effortless -> Smooth -> Bumpy -> Struggled -> Failed)
- Friction points: `retry`, `timeout`, `wrong_approach`, `tool_failure`, `ambiguity`, `contract_failure`
- Learnings: `repo`, `code`, `workflow`, `tool` categories
- Open items: `tech_debt`, `follow_up`, `investigation`, `test_gap`
- Concrete recommendations for pipeline/persona improvements

### Storage

```
.wave/retros/<run-id>.json
```

Schema:
```json
{
  "run_id": "...",
  "pipeline": "impl-issue",
  "timestamp": "...",
  "quantitative": {
    "total_duration_ms": 120000,
    "steps": [
      {"name": "plan", "duration_ms": 30000, "retries": 0, "status": "success"},
      {"name": "implement", "duration_ms": 90000, "retries": 2, "status": "success"}
    ]
  },
  "narrative": {
    "smoothness": "bumpy",
    "intent": "Implement login validation",
    "outcome": "Completed with 2 retries on implementation step",
    "friction_points": [
      {"type": "retry", "step": "implement", "detail": "First attempt had wrong import path"}
    ],
    "learnings": [
      {"category": "code", "detail": "Auth module uses custom middleware, not stdlib"}
    ],
    "open_items": [
      {"type": "test_gap", "detail": "No integration test for OAuth flow"}
    ]
  }
}
```

### CLI Integration

```bash
wave retro <run-id>              # View retrospective
wave retro <run-id> --narrate    # Re-generate narrative
wave retro list --pipeline impl-issue --since 7d  # Cross-run view
wave retro stats                 # Aggregate statistics
```

### Web UI Dashboard

Add retrospective view to the existing web UI:
- Per-run retro display
- Aggregate smoothness trends
- Most common friction points
- Pipeline comparison (which pipelines struggle most?)

### Configuration

```yaml
runtime:
  retro:
    enabled: true
    narrate: true
    narrate_model: claude-haiku-4-5
```

Per-run override: `wave run --no-retro impl-issue -- "..."`

## What Wave Keeps

- Existing event/audit infrastructure (source data for retros)
- SQLite state management (retros stored alongside run state)
- Structured progress events (feed into quantitative layer)

## What Wave Gains

- Pipeline improvement feedback loop — friction data identifies where pipelines need work
- Cost visibility — per-step and per-run cost tracking
- Failure pattern detection — cross-run analysis reveals systemic issues
- Persona effectiveness — which personas/models succeed most on which tasks?

## Implementation Scope

1. `internal/retro/` package — retrospective generation and storage
2. Quantitative collector from existing event stream
3. LLM narrative generator (using adapter for cheap model)
4. CLI commands for viewing and aggregating retros
5. Web UI retrospective components
6. Configuration in manifest schema

## Acceptance Criteria

1. After every pipeline run (success or failure), a quantitative retrospective is automatically generated and saved to `.wave/retros/<run-id>.json`
2. When `runtime.retro.narrate: true`, an LLM-powered narrative is appended to the retro with smoothness rating, friction points, learnings, and open items
3. `wave retro <run-id>` displays the retrospective in human-readable format
4. `wave retro list` shows recent retrospectives with filtering options
5. `wave retro stats` shows aggregate statistics across runs
6. `wave run --no-retro` disables retro generation for a single run
7. Retrospective data is also persisted in SQLite for querying
8. Web UI shows retro data on the run detail page
9. Retro generation does not block or slow down pipeline execution
10. All new code has unit tests with >80% coverage
