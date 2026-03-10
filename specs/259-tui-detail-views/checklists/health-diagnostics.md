# Health Diagnostics Quality Checklist

**Feature**: #259 — TUI Alternative Master-Detail Views  
**Date**: 2026-03-06  
**Scope**: Quality of Health view requirements — 6 diagnostic checks, async execution, re-run behavior

## Completeness

- [ ] CHK048 - Are all 6 health check specifications detailed enough to implement — does each check define its inputs, success criteria, warning criteria, and failure criteria? [Completeness]
- [ ] CHK049 - Does the spec define timeout behavior for individual health checks — what happens if `git status` or `exec.LookPath` hangs? [Completeness]
- [ ] CHK050 - Is the "Adapter Binary" check fully specified — does it check all adapters or only those referenced by pipelines, and what version detection method is used? [Completeness]
- [ ] CHK051 - Does the spec define what "SQLite Database" connectivity means — is a successful `ListRuns(Limit:1)` sufficient, or should it verify schema version? [Completeness]
- [ ] CHK052 - Does the spec define what constitutes a WARN vs FAIL for each check — are thresholds or decision criteria documented? [Completeness]
- [ ] CHK053 - Are the health check detail key-value pairs defined for each check — what specific diagnostic fields appear in the right pane? [Completeness]

## Clarity

- [ ] CHK054 - Is the async execution model unambiguous — are all 6 checks launched in parallel, or do some depend on others (e.g., config check before tool check)? [Clarity]
- [ ] CHK055 - Is the `r` re-run behavior clear — does it re-run all 6 checks unconditionally, or only failed/warn checks? [Clarity]
- [ ] CHK056 - Is it clear whether the "last checked" timestamp is per-check or global — if per-check, can different checks show different timestamps after a partial re-run? [Clarity]

## Consistency

- [ ] CHK057 - Are the health check status icons (●, ▲, ✗) and colors consistent with any existing status indicators in the TUI (header health, pipeline status)? [Consistency]
- [ ] CHK058 - Does the health check "Git Repository" check reuse or conflict with the header's existing git state fetching (`GitStateMsg`)? [Consistency]
- [ ] CHK059 - Is the "Wave Configuration" check consistent with the existing manifest loading in `RunTUI` — does it re-parse the manifest or use the already-loaded one? [Consistency]

## Coverage

- [ ] CHK060 - Does the spec address how health checks behave when run in a Wave pipeline workspace (ephemeral worktree) vs the main repository? [Coverage]
- [ ] CHK061 - Are health check results actionable — does the spec define remediation hints (e.g., "run `wave install` to fix missing skills") for each failure mode? [Coverage]
- [ ] CHK062 - Does the spec address the case where the user presses `r` while a previous check batch is still in-flight — are results deduplicated or can they race? [Coverage]
