# feat: automatic run retrospectives with structured friction analysis

**Issue**: [#578](https://github.com/re-cinq/wave/issues/578)
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
- Smoothness rating (5-point scale: Effortless → Smooth → Bumpy → Struggled → Failed)
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
  retros:
    enabled: true           # default: true
    narrate: true           # LLM narrative (default: true, disable for cost saving)
    narrate_model: claude-haiku-4-5  # cheap model for narration
```

Per-run override: `wave run --no-retro impl-issue -- "..."`

## Acceptance Criteria

1. Quantitative retrospective generated automatically after every pipeline run (success or failure)
2. Quantitative data sourced from existing state store (RunRecord, PerformanceMetricRecord, StepAttemptRecord)
3. Optional LLM narrative generation using a configurable cheap model
4. Retrospectives stored as JSON files in `.wave/retros/<run-id>.json`
5. `wave retro <run-id>` CLI command to view a retrospective
6. `wave retro <run-id> --narrate` to (re)generate narrative
7. `wave retro list` with `--pipeline` and `--since` filters
8. `wave retro stats` for aggregate statistics
9. Web UI retrospective page and API endpoints
10. Configuration in `wave.yaml` under `runtime.retros`
11. `--no-retro` flag on `wave run` to skip generation
12. All new code has unit tests
