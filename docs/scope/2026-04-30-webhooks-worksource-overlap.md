# Webhooks + WorkSource overlap analysis

Audited overlap between future webhooks (#638) and current worksource/dispatch surface (Phase 2). Conclusion: additive, no refactor needed.

## Outbound webhooks (lifecycle listeners)

Integration point: `internal/pipeline/executor_observers.go:112-116` via existing `HookRunner` / `FireWebhooks()`. Webhooks fire on lifecycle events (run-start, step-complete, run-finish) as additional observers alongside existing emitter, state-store, and contract hooks. No worksource changes required.

## Inbound webhook triggers (future phase)

Payload normalization → `WorkItemRef` → `MatchBindings()` → `launchPipelineExecution`. Plug-in point: `internal/webui/handlers_work_dispatch.go:83-91`. New endpoint (`POST /api/webhooks/{forge}`) would parse forge-specific payloads, extract issue/PR identifiers, and feed into the existing dispatch flow. No worksource schema changes needed — `WorkItemRef` already models the required fields.

## No action required

This document is informational. Link from Epic #1565 follow-ups and future #638 epic body.
