# CI Workflow Requirements Quality Checklist

**Feature**: 103-static-analysis-ci — Static Analysis for Unused/Redundant Go Code
**Date**: 2026-02-18
**Domain**: GitHub Actions CI workflow configuration and integration

---

## Trigger & Event Completeness

- [ ] CHK-CI-001 - Are all trigger events (pull_request, push) specified with their branch filters, or could additional events (workflow_dispatch, schedule) be needed? [Completeness]
- [ ] CHK-CI-002 - Does the spec define path filters for the workflow triggers (e.g., should the lint workflow skip runs when only docs or non-Go files change)? [Completeness]
- [ ] CHK-CI-003 - Is the concurrency behavior defined for the lint workflow (e.g., cancel in-progress runs when a new commit is pushed to the same PR)? [Completeness]

## Action Version & Pinning Clarity

- [ ] CHK-CI-004 - Is the version pinning strategy clearly defined — semver tag (e.g., `v7`) vs. exact tag (e.g., `v7.0.0`) vs. commit SHA? [Clarity]
- [ ] CHK-CI-005 - Does the spec define a process for updating pinned versions when new releases of golangci-lint or its action are published? [Completeness]
- [ ] CHK-CI-006 - Is it specified whether `actions/checkout` should use shallow clone (`fetch-depth: 1`) or full history (needed for `only-new-issues` diff computation)? [Completeness]

## Caching & Performance

- [ ] CHK-CI-007 - Are the caching requirements for the lint workflow explicitly stated, or does the spec rely entirely on action defaults without documenting what those defaults provide? [Clarity]
- [ ] CHK-CI-008 - Is the expected cold-cache vs. warm-cache performance difference documented as context for the timeout decision (C-004)? [Completeness]

## Integration & Isolation

- [ ] CHK-CI-009 - Does the spec define whether the lint check should be a required status check for PR merges, or is it advisory? [Completeness]
- [ ] CHK-CI-010 - Is non-interference with `release.yml` defined in terms of specific properties (no shared job names, no conflicting triggers, no shared caches)? [Clarity]
- [ ] CHK-CI-011 - Does the spec address whether the lint workflow should share the same runner labels and resource constraints as existing workflows? [Completeness]

## Error Reporting & Developer Experience

- [ ] CHK-CI-012 - Are the requirements for how lint violations are surfaced to the developer defined (PR annotations, check run summary, workflow logs)? [Completeness]
- [ ] CHK-CI-013 - Does the spec define what information each violation report must include (linter name, file path, line number, message, severity)? [Completeness]
- [ ] CHK-CI-014 - Is the expected developer workflow for resolving a lint failure documented (fix locally, push, re-run CI)? [Completeness]
