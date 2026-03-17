# Data Model: Polish Log Streaming UX

**Feature**: #464 — Polish Log Streaming UX
**Date**: 2026-03-17

## Overview

All entities are **client-side only** — no backend schema changes. These are JavaScript objects/classes used within the log viewer module.

## Entities

### LogLine

A single rendered log entry derived from a `stream_activity` SSE event.

```
LogLine {
    lineNumber: number          // Sequential per-step, starting at 1
    timestamp:  string          // ISO 8601 from event.timestamp, displayed as HH:MM:SS.mmm
    stepId:     string          // Which step produced this line
    toolName:   string          // "Read", "Write", "Bash", "Grep", etc.
    toolTarget: string          // File path, command, pattern
    message:    string          // Human-readable activity description
    rawContent: string          // Original message (with any ANSI codes)
    htmlContent: string         // ANSI-rendered HTML string
    element:    HTMLElement|null // Cached DOM reference (for search highlighting)
}
```

**Source mapping**: Each `stream_activity` SSE event maps 1:1 to a LogLine. Fields are populated from `event.Event`:
- `lineNumber` — assigned incrementally per step section
- `timestamp` — `event.timestamp` (server-side emission time)
- `stepId` — `event.step_id`
- `toolName` — `event.tool_name`
- `toolTarget` — `event.tool_target`
- `message` — `event.message`
- `rawContent` — `event.message` (preserved for download/copy)
- `htmlContent` — result of ANSI-to-HTML conversion of `message`

### LogSection

A collapsible container for a step's log output. Extends the existing step card concept.

```
LogSection {
    stepId:     string          // Step identifier (matches step card)
    stepName:   string          // Display name (same as step_id)
    status:     string          // "pending" | "running" | "completed" | "failed" | "cancelled"
    expanded:   boolean         // Whether log content is visible
    lines:      LogLine[]       // Ordered log lines for this step
    lineCount:  number          // Total lines received (= lines.length)
    element:    HTMLElement|null // Reference to the DOM container for this section
    autoScroll: boolean         // Whether auto-scroll is active for this section
}
```

**Default expand rules**:
- `status === "running"` → `expanded = true`
- `status === "failed"` → `expanded = true`
- `status === "completed"` → `expanded = false`
- `status === "pending"` → `expanded = false`

**Lifecycle**: Created when the first `stream_activity` event arrives for a step, or when the page loads with existing step cards. Updated as SSE events stream in.

### SearchState

Client-side search context, global across all visible log sections.

```
SearchState {
    query:        string        // Current search term (empty = no search)
    matches:      SearchMatch[] // Ordered list of all matches across sections
    currentIndex: number        // Index into matches[] for current highlight (-1 = none)
    totalCount:   number        // Total match count (= matches.length)
    debounceTimer: number|null  // Timeout ID for debounced search execution
}
```

### SearchMatch

A single search match reference.

```
SearchMatch {
    stepId:     string          // Which step section contains this match
    lineIndex:  number          // Index into LogSection.lines[]
    charStart:  number          // Character offset within the line text
    charEnd:    number          // End character offset
}
```

### ConnectionState

SSE connection status for the reconnection indicator.

```
ConnectionState {
    status:     string          // "connected" | "reconnecting" | "disconnected"
    retryCount: number          // Number of reconnection attempts
    lastEventId: string|null    // Last received event ID for backfill
}
```

## State Management

All state is managed in a single `LogViewer` object attached to the run detail page:

```
LogViewer {
    sections:    Map<string, LogSection>  // stepId → LogSection
    search:      SearchState
    connection:  ConnectionState
    batchBuffer: LogLine[]                // Buffer for requestAnimationFrame batching
    batchTimer:  number|null              // rAF callback ID
}
```

No global mutable state outside this object. The `LogViewer` is initialized when the run detail page loads and destroyed on page unload.

## Relationships

```
LogViewer 1 ──── * LogSection (one per pipeline step)
LogSection 1 ──── * LogLine (ordered by lineNumber)
LogViewer 1 ──── 1 SearchState
SearchState 1 ──── * SearchMatch (ordered by section, then line)
LogViewer 1 ──── 1 ConnectionState
```

## Event Flow

```
SSE stream_activity event
  → sse.js handleSSEEvent()
    → logViewer.addLine(stepId, eventData)
      → Create LogLine from event data
      → ANSI-to-HTML conversion
      → Append to LogSection.lines[]
      → Buffer in batchBuffer[]
      → requestAnimationFrame → flushBatch()
        → DOM insertion (chunked)
        → If search active, check new line for matches
        → If autoScroll active, scroll to bottom
```
