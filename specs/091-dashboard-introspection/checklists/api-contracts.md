# API Contract Fidelity Checklist

**Feature**: Dashboard Inspection, Rendering, Statistics & Run Introspection
**Spec**: specs/091-dashboard-introspection/spec.md
**Generated**: 2026-02-14

This checklist validates that the 5 API contracts are well-specified and consistent with the functional requirements and data model.

---

## Schema-Requirement Alignment

- [ ] CHK-A01 - Does api-pipeline-detail.json's `steps[].contract` object include all fields specified in FR-003 (contract type, schema content, validation rules)? The current schema has `type`, `schema`, `schema_path`, `must_pass`, `max_retries` — is "validation rules" captured? [Completeness]
- [ ] CHK-A02 - Does api-persona-detail.json require the `system_prompt` field, given FR-002 mandates displaying system prompt content? Currently `required` only lists `["name", "adapter"]`. [Consistency]
- [ ] CHK-A03 - Does api-statistics.json's `trends` array specify ordering (ascending or descending by date)? The spec doesn't define this. [Clarity]
- [ ] CHK-A04 - Does api-enhanced-run-detail.json include token delta information per event, given FR-015 requires "token deltas" in the event timeline? The `events[].tokens_used` field exists but is it cumulative or delta? [Clarity]
- [ ] CHK-A05 - Does api-workspace.json's `WorkspaceTreeResponse.entries` specify a sort order (alphabetical, directories-first, or unsorted)? [Clarity]
- [ ] CHK-A06 - Is the `mime_type` field in WorkspaceFileResponse well-defined — what MIME types are expected for the supported file types (Go, YAML, JSON, etc.)? [Clarity]

## Contract-Data Model Alignment

- [ ] CHK-A07 - Does the PipelineDetailResponse Go type in data-model.md match the api-pipeline-detail.json contract exactly (same required fields, same nesting)? [Consistency]
- [ ] CHK-A08 - Does the StatisticsResponse Go type include `pending` and `running` in the aggregate, matching the contract but diverging from the FR text? Is this intentional? [Consistency]
- [ ] CHK-A09 - Does the EnhancedStepDetail embedding StepDetail create JSON that matches the flat structure in api-enhanced-run-detail.json, or does Go's embedding produce nested output? [Consistency]
- [ ] CHK-A10 - Are all `required` fields in every contract schema guaranteed to be populated by the corresponding handler — e.g., can `steps[].state` always be determined from event data? [Completeness]

## Contract Versioning & Evolution

- [ ] CHK-A11 - Is it specified whether these new API endpoints are versioned (e.g., /api/v1/statistics) or follow the existing unversioned pattern? [Completeness]
- [ ] CHK-A12 - Do the contracts define error response schemas for 404 (not found), 400 (bad request), and 500 (server error) cases, or only the success path? [Completeness]
- [ ] CHK-A13 - Is backward compatibility specified for the enhanced run detail endpoint — does it extend the existing /api/runs/{id} response or create a new endpoint? [Clarity]
