# Implementation Plan: Adopt Epic #589 Features

## Objective

Integrate the engine capabilities delivered by epic #589 (graph loops, gates, hooks, retro, multi-adapter routing, LLM-as-judge, failure taxonomy) into all shipped pipelines, TUI health checks, TUI run detail rendering, and documentation guides. The user-facing product must visibly surface these capabilities.

## Approach

Three-phase rollout aligned with the issue structure:

1. **Pipeline Adoption** (highest impact, most files): Modify all 51 shipped pipeline YAMLs to use named retry policies, haiku model routing for analysis steps, and add graph loop to `impl-issue`. Add LLM judge to `ops-pr-review` quality-review. Sync `.wave/pipelines/` → `internal/defaults/pipelines/`.
2. **TUI + WebUI Enhancements**: Add step type indicators and retro viewer to TUI run detail. Expand doctor health checks for hooks/retro. WebUI already has gate/loop/retro rendering — no WebUI changes needed.
3. **Documentation**: Four new guides covering the adopted features.

## File Mapping

### Phase 1: Pipeline YAML Changes

#### `impl-issue.yaml` — Add Graph Loop
| File | Action | Description |
|------|--------|-------------|
| `.wave/pipelines/impl-issue.yaml` | modify | Restructure implement step into graph loop: implement→test→fix with edges, conditions, max_visits |
| `internal/defaults/pipelines/impl-issue.yaml` | modify | Mirror changes |

The graph loop design for `impl-issue`:
- **implement** step: runs implementation, thread continuity preserved
- **test** step (new): runs `{{ project.contract_test_command }}`, captures test results
- **fix** step (new): conditional — only entered on test failure, feeds back to test via edge
- **create-pr**: entered when test passes

Uses `edges` with `condition: "outcome=success"` / `condition: "outcome=failure"` for routing. `max_visits: 3` on the fix→test loop to prevent infinite cycling.

#### `ops-pr-review.yaml` — Add LLM Judge to Quality Review
| File | Action | Description |
|------|--------|-------------|
| `.wave/pipelines/ops-pr-review.yaml` | modify | Add `llm_judge` section to quality-review handover, matching security-review pattern |
| `internal/defaults/pipelines/ops-pr-review.yaml` | modify | Mirror changes |

#### ALL Pipelines — Remove Redundant `max_attempts`
| File | Action | Description |
|------|--------|-------------|
| `.wave/pipelines/*.yaml` (51 files) | modify | Remove `max_attempts` lines where a named `policy` already exists |
| `internal/defaults/pipelines/*.yaml` (51 files) | modify | Mirror changes |

Every shipped pipeline already specifies both `policy:` and `max_attempts:`. The named policy defines attempts (standard=3, aggressive=5, patient=3, none=1). Removing `max_attempts` lets the policy be the single source of truth.

#### ALL Pipelines — Haiku Routing for Analysis Steps
| File | Action | Description |
|------|--------|-------------|
| 36 pipeline files in `.wave/pipelines/` | modify | Add `model: claude-haiku-4-5` to 52 steps using navigator/reviewer/summarizer/analyst personas |
| 36 pipeline files in `internal/defaults/pipelines/` | modify | Mirror changes |

### Phase 2: TUI + Doctor Changes

#### Doctor Health Checks
| File | Action | Description |
|------|--------|-------------|
| `internal/doctor/doctor.go` | modify | Add `checkHooksConfig()` and `checkRetroStore()` health checks |
| `internal/doctor/doctor_test.go` | modify | Add tests for new health checks |

New checks:
- **Hooks Config**: Scan pipelines for `hooks:` declarations, verify referenced commands/URLs are accessible
- **Retro Store**: Verify retro store is writable (check `.wave/retros/` or SQLite state)

#### TUI Step Type Rendering
| File | Action | Description |
|------|--------|-------------|
| `internal/tui/pipeline_detail.go` | modify | Add step type indicators (gate/command/conditional/pipeline) to `renderFinishedDetail()` step list |

Currently TUI renders: `✓ step-name  1m 23s  (persona)`. Add type badge: `✓ [gate] approve-plan  1m 23s  (navigator)`.

#### TUI Retro Viewer
| File | Action | Description |
|------|--------|-------------|
| `internal/tui/retro_view.go` | create | New Bubble Tea component for retrospective display |
| `internal/tui/pipeline_detail.go` | modify | Add retro section to finished run detail, lazily loaded |

Display quantitative metrics (duration, steps, retries, tokens) and narrative (smoothness, friction points, learnings, recommendations) matching WebUI retro display.

### Phase 3: Documentation

| File | Action | Description |
|------|--------|-------------|
| `docs/guides/graph-loops.md` | create | Guide: graph loops and conditional routing |
| `docs/guides/approval-gates.md` | create | Guide: human approval gates |
| `docs/guides/retry-policies.md` | create | Guide: retry policies |
| `docs/guides/model-routing.md` | create | Guide: multi-adapter model routing |

## Architecture Decisions

1. **Graph loop in impl-issue uses edge-based routing, not LoopConfig**: The `edges` + `condition` mechanism in `graph.go` is the proven pattern for conditional routing. `LoopConfig` is designed for sub-pipeline iteration. Edge-based loops match the implement→test→fix semantic better.

2. **Remove max_attempts rather than keep both**: Named policies are self-documenting. Having both is confusing and the `max_attempts` override defeats the purpose of named policies. Some steps currently override policy defaults (e.g., `policy: patient, max_attempts: 2` where patient defaults to 3) — we accept the policy default as the correct value.

3. **TUI retro as separate component, not inline**: Retro data can be large (narrative, friction points, learnings). A separate view component with lazy loading avoids bloating the pipeline detail view.

4. **WebUI already complete**: The exploration confirmed WebUI already has gate hexagons, loop backward arrows (dashed orange), conditional diamonds, command rounded-dashed, retro viewer with quantitative + narrative. No WebUI changes needed.

5. **Health checks for hooks/retro are lightweight**: They scan pipeline YAML and check file system / state store accessibility, not deep validation.

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Removing `max_attempts` changes retry behavior for steps that override policy defaults | Medium — some steps may retry more/fewer times | Accept policy defaults as correct; they were designed for these use cases |
| Graph loop in `impl-issue` may cause infinite cycling on persistent test failures | High — stuck pipeline | `max_visits: 3` circuit breaker + graph walker's existing 3-identical-error circuit breaker |
| 102 YAML files modified across both directories | Medium — merge conflict risk | Phase 1 changes are mechanical (grep-replaceable); commit in small batches |
| TUI retro view requires retro store integration | Low — retro store already exists | Use existing `internal/retro/` package APIs |

## Testing Strategy

1. **Pipeline YAML validation**: `go test ./internal/pipeline/...` — existing pipeline loading tests validate YAML structure
2. **Doctor health checks**: Table-driven tests in `doctor_test.go` for new `checkHooksConfig()` and `checkRetroStore()`
3. **TUI rendering**: Verify step type indicators render correctly for each type
4. **Regression**: Full `go test ./...` must pass
5. **Manual validation**: `wave list pipelines` shows pipelines, `wave doctor` shows new health checks
