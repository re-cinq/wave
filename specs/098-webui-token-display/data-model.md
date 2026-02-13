# Data Model: Web UI Token Display

**Feature**: 098-webui-token-display
**Date**: 2026-02-13

## Entity Overview

This feature does **not** introduce new entities or database tables. It corrects and extends the display of existing token data that already flows through the system. The key entities below are all pre-existing; this document maps how they relate to the feature's requirements.

## Existing Entities (No Changes Required)

### `state.RunRecord`

**Location**: `internal/state/types.go:6-18`
**Database table**: `pipeline_run`

| Field | Type | DB Column | Feature Role |
|-------|------|-----------|-------------|
| RunID | string | run_id | Primary key |
| PipelineName | string | pipeline_name | Pipeline identifier |
| Status | string | status | Run lifecycle state |
| TotalTokens | int | total_tokens | **FR-002, FR-004**: Aggregate token count displayed in run list and run detail header |
| StartedAt | time.Time | started_at | Timing display |
| CompletedAt | *time.Time | completed_at | Duration calculation |

**Note**: `TotalTokens` is already written by `UpdateRunStatus()` in `cmd/wave/commands/run.go:261-266`. The executor calls `GetTotalTokens()` which sums per-step results.

### `state.LogRecord`

**Location**: `internal/state/types.go:30-40`
**Database table**: `event_log`

| Field | Type | DB Column | Feature Role |
|-------|------|-----------|-------------|
| StepID | string | step_id | Associates tokens with a step |
| State | string | state | Event type (running/completed/failed) |
| TokensUsed | int | tokens_used | **FR-001**: Per-step token count, source for `buildStepDetails()` |
| DurationMs | int64 | duration_ms | Step duration |

**Note**: `buildStepDetails()` in `handlers_runs.go:290-403` already reconstructs per-step tokens from events using `if ev.TokensUsed > si.tokens { si.tokens = ev.TokensUsed }` (takes the max, which corresponds to the `completed` event's authoritative value).

### `event.Event`

**Location**: `internal/event/emitter.go:11-45`

| Field | Type | JSON Key | Feature Role |
|-------|------|----------|-------------|
| TokensUsed | int | tokens_used | **FR-005**: Carried in SSE events for real-time updates |
| State | string | state | Determines event type in SSE stream |
| StepID | string | step_id | Associates SSE token updates with step cards |

### `webui.RunSummary`

**Location**: `internal/webui/types.go:13-25`

| Field | Type | JSON Key | Feature Role |
|-------|------|----------|-------------|
| TotalTokens | int | total_tokens | **FR-002**: Displayed in run list; **FR-004**: Displayed in run detail header |

**Note**: Already populated by `runToSummary()` from `RunRecord.TotalTokens`.

### `webui.StepDetail`

**Location**: `internal/webui/types.go:37-49`

| Field | Type | JSON Key | Feature Role |
|-------|------|----------|-------------|
| TokensUsed | int | tokens_used | **FR-001**: Per-step tokens in step cards |
| State | string | state | **FR-008**: Determines display logic (show for completed/failed, omit for pending) |

## Modified Entities

### `display.FormatTokenCount` (Function, Not Entity)

**Location**: `internal/display/formatter.go:500-505`
**Change**: Extend threshold logic for M (million) and B (billion).

**Current**:
```
< 1000: raw integer
≥ 1000: "%.1fk"
```

**New**:
```
< 1000:         raw integer  (e.g., "842")
< 1_000_000:   "%.1fk"      (e.g., "1.5k")
< 1_000_000_000: "%.1fM"    (e.g., "2.3M")
≥ 1_000_000_000: "%.1fB"    (e.g., "1.2B")
```

## Data Flow Diagram

```
Adapter NDJSON → executor.TokensUsed → event.Event.TokensUsed
                                              │
                           ┌──────────────────┼───────────────────┐
                           ▼                  ▼                   ▼
                     event_log DB        NDJSON stdout        SSE broker
                    (per-step tokens)   (JSON consumers)    (real-time)
                           │                                      │
                           ▼                                      ▼
                    buildStepDetails()                     sse.js handlers
                    → StepDetail.TokensUsed               → updateStepCard
                           │
                           ▼
                    step_card.html
                    {{formatTokens .TokensUsed}}

executor.GetTotalTokens() → store.UpdateRunStatus(tokens)
                                      │
                                      ▼
                               pipeline_run.total_tokens
                                      │
                           ┌──────────┴──────────┐
                           ▼                     ▼
                    run_row.html            run_detail.html
                    {{formatTokens          run-meta header
                     .TotalTokens}}        {{formatTokens .Run.TotalTokens}}
```

## Template Function Registry

**New template function**: `formatTokens`

Registered in the Go template `FuncMap` alongside existing `formatTime` and `statusClass`:

```go
template.FuncMap{
    "formatTime":   formatTimeFunc,
    "statusClass":  statusClassFunc,
    "formatTokens": display.FormatTokenCount,  // NEW
}
```

This function is used in:
- `run_row.html` — run list total tokens column
- `step_card.html` — per-step token display
- `run_detail.html` — summary header total tokens

## JavaScript Formatting (SSE Client)

A small JS function mirrors the Go logic for dynamically-created DOM elements:

```javascript
function formatTokens(n) {
    if (n < 1000) return String(n);
    if (n < 1000000) return (n / 1000).toFixed(1) + 'k';
    if (n < 1000000000) return (n / 1000000).toFixed(1) + 'M';
    return (n / 1000000000).toFixed(1) + 'B';
}
```

Used in `createStepCard()` and `updatePageFromAPI()` for SSE/polling-driven updates.
