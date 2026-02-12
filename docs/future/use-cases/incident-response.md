---
title: Incident Response
description: Rapid investigation and remediation workflows for production incidents
---

# Incident Response

<div class="use-case-meta">
  <span class="complexity-badge advanced">Advanced</span>
  <span class="category-badge">DevOps</span>
</div>

Rapid investigation and remediation workflows for production incidents. This pipeline helps identify root causes, propose fixes, and generate post-incident documentation.

## Prerequisites

- Wave installed and initialized (`wave init`)
- Access to logs, metrics, or error traces
- Experience with [code-review](/use-cases/code-review) and [security-audit](./security-audit) pipelines
- Understanding of your system's architecture and monitoring stack

## Quick Start

```bash
wave run incident-response "500 errors spiking on /api/orders endpoint since 14:30 UTC"
```

Expected output:

```
[10:00:01] started   triage            (navigator)              Starting step
[10:00:25] completed triage            (navigator)   24s   2.1k Triage complete
[10:00:26] started   investigate       (auditor)                Starting step
[10:00:58] completed investigate       (auditor)     32s   3.8k Investigation complete
[10:00:59] started   root-cause        (philosopher)            Starting step
[10:01:28] completed root-cause        (philosopher)  29s   2.5k Analysis complete
[10:01:29] started   remediate         (craftsman)              Starting step
[10:02:15] completed remediate         (craftsman)   46s   4.2k Fix ready
[10:02:16] started   postmortem        (summarizer)             Starting step
[10:02:42] completed postmortem        (summarizer)  26s   3.1k Report complete

Pipeline incident-response completed in 161s
Artifacts: output/incident-report.md
```

## Complete Pipeline

Save the following YAML to `.wave/pipelines/incident-response.yaml`:

<div v-pre>

```yaml
kind: WavePipeline
metadata:
  name: incident-response
  description: "Rapid incident investigation and remediation"

input:
  source: cli

steps:
  - id: triage
    persona: navigator
    memory:
      strategy: fresh
    workspace:
      mount:
        - source: ./
          target: /src
          mode: readonly
    exec:
      type: prompt
      source: |
        Triage the incident: {{ input }}

        Gather initial information:
        1. What is the symptom? (errors, latency, failures)
        2. When did it start? (timestamp, deployment, event)
        3. What is the blast radius? (users, services, regions)
        4. What changed recently? (deployments, configs, dependencies)
        5. What are the immediate mitigation options?

        Output as JSON:
        {
          "symptom": "",
          "severity": "P1|P2|P3|P4",
          "started_at": "",
          "blast_radius": {},
          "recent_changes": [],
          "mitigation_options": []
        }
    output_artifacts:
      - name: triage
        path: output/triage.json
        type: json

  - id: investigate
    persona: auditor
    dependencies: [triage]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: triage
          artifact: triage
          as: triage
    workspace:
      mount:
        - source: ./
          target: /src
          mode: readonly
    exec:
      type: prompt
      source: |
        Deep investigation of the incident.

        Analyze:
        1. Error logs and stack traces
        2. Recent code changes in affected areas
        3. Configuration changes
        4. Dependency updates
        5. Infrastructure changes
        6. Similar past incidents

        For each potential cause:
        - Likelihood: HIGH / MEDIUM / LOW
        - Evidence supporting this theory
        - Evidence against this theory
        - Tests to confirm or rule out
    output_artifacts:
      - name: investigation
        path: output/investigation.md
        type: markdown

  - id: root-cause
    persona: philosopher
    dependencies: [investigate]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: triage
          artifact: triage
          as: triage
        - step: investigate
          artifact: investigation
          as: evidence
    exec:
      type: prompt
      source: |
        Determine the root cause of the incident.

        Apply the 5 Whys technique:
        1. Why is [symptom] happening?
        2. Why did that happen?
        3. Why did that happen?
        4. Why did that happen?
        5. Why did that happen?

        Identify:
        - Immediate cause (what broke)
        - Contributing factors (what enabled it)
        - Root cause (why it could happen)
        - Systemic issues (patterns to address)
    output_artifacts:
      - name: root-cause
        path: output/root-cause.md
        type: markdown

  - id: remediate
    persona: craftsman
    dependencies: [root-cause]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: triage
          artifact: triage
          as: triage
        - step: root-cause
          artifact: root-cause
          as: analysis
    workspace:
      mount:
        - source: ./
          target: /src
          mode: readwrite
    exec:
      type: prompt
      source: |
        Develop remediation for the incident.

        Create:
        1. Immediate fix (stop the bleeding)
        2. Short-term fix (proper solution)
        3. Long-term improvements (prevent recurrence)

        For each fix:
        - Code changes needed
        - Configuration changes
        - Rollback procedure
        - Verification steps
        - Risk assessment
    output_artifacts:
      - name: fix
        path: output/remediation.md
        type: markdown

  - id: postmortem
    persona: summarizer
    dependencies: [remediate]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: triage
          artifact: triage
          as: triage
        - step: root-cause
          artifact: root-cause
          as: analysis
        - step: remediate
          artifact: fix
          as: remediation
    exec:
      type: prompt
      source: |
        Generate a blameless post-incident report.

        Include:
        1. Executive Summary
        2. Timeline of Events
        3. Impact Assessment
        4. Root Cause Analysis
        5. Resolution and Recovery
        6. Action Items (with owners and due dates)
        7. Lessons Learned
        8. Appendix (logs, graphs, evidence)

        Focus on systemic improvements, not individual blame.
    output_artifacts:
      - name: report
        path: output/incident-report.md
        type: markdown
```

</div>

## Expected Outputs

The pipeline produces five artifacts:

| Artifact | Path | Description |
|----------|------|-------------|
| `triage` | `output/triage.json` | Initial triage and severity assessment |
| `investigation` | `output/investigation.md` | Detailed investigation findings |
| `root-cause` | `output/root-cause.md` | Root cause analysis |
| `fix` | `output/remediation.md` | Remediation plan and fixes |
| `report` | `output/incident-report.md` | Complete post-incident report |

### Example Output

The pipeline produces `output/incident-report.md`:

```markdown
# Incident Report: Order API 500 Errors

**Incident ID**: INC-2026-0204-001
**Severity**: P1 (Customer-facing service degradation)
**Duration**: 14:30 - 15:45 UTC (75 minutes)
**Status**: Resolved

## Executive Summary

The /api/orders endpoint experienced elevated 500 errors (35% error rate) due
to a database connection pool exhaustion caused by a missing connection timeout
in a new query introduced in deployment v2.45.0.

## Timeline

| Time (UTC) | Event |
|------------|-------|
| 14:15 | Deployment v2.45.0 released |
| 14:30 | First alerts fired (error rate > 5%) |
| 14:35 | On-call engineer paged |
| 14:45 | Initial triage complete, rollback considered |
| 15:00 | Root cause identified (connection pool exhaustion) |
| 15:15 | Hotfix deployed (connection timeout added) |
| 15:30 | Error rate normalized |
| 15:45 | Incident resolved, monitoring continued |

## Impact

- **Users Affected**: ~12,000 (15% of active users)
- **Failed Requests**: ~45,000
- **Revenue Impact**: Estimated $8,500 in failed orders
- **SLA Impact**: 99.9% SLA breached for 75 minutes

## Root Cause Analysis

### 5 Whys

1. **Why were 500 errors occurring?**
   Database queries were timing out.

2. **Why were queries timing out?**
   Connection pool was exhausted (0 available connections).

3. **Why was the pool exhausted?**
   New order history query was holding connections for 30+ seconds.

4. **Why was the query so slow?**
   Missing index on `orders.created_at` column.

5. **Why was the missing index not caught?**
   Query was tested only with small datasets; no load testing on production-like data.

### Contributing Factors

- No query timeout configured (defaulted to infinite)
- Connection pool monitoring alerts not configured
- Load testing skipped for "minor" change

## Resolution

### Immediate Fix
- Added 5-second query timeout to the database driver config
- Increased connection pool size from 20 to 50

### Deployed Fix
- Added index on `orders.created_at` (query time: 30s -> 50ms)
- Refactored query to use pagination

## Action Items

| ID | Action | Owner | Due Date | Status |
|----|--------|-------|----------|--------|
| 1 | Add connection pool monitoring | @sre-team | 2026-02-07 | Open |
| 2 | Implement query timeout defaults | @backend-team | 2026-02-11 | Open |
| 3 | Add load testing to CI pipeline | @qa-team | 2026-02-18 | Open |
| 4 | Review all queries for missing indexes | @dba-team | 2026-02-14 | Open |

## Lessons Learned

### What Went Well
- Quick detection via error rate alerts
- Effective collaboration between teams
- Clear communication in incident channel

### What Could Improve
- Load testing should be mandatory for database changes
- Connection pool metrics should be monitored
- Rollback decision tree should be documented
```

## Customization

### Focus on specific symptoms

```bash
wave run incident-response "high latency on payment processing, p99 > 5s"
```

### Include external context

```bash
wave run incident-response "502 errors after AWS us-east-1 degradation announcement"
```

### Quick triage only

Create a minimal pipeline for initial assessment:

<div v-pre>

```yaml
kind: WavePipeline
metadata:
  name: incident-triage
  description: "Quick incident triage"

steps:
  - id: triage
    # Only the triage step for rapid assessment
```

</div>

### Add automated rollback check

<div v-pre>

```yaml
- id: rollback-assessment
  persona: auditor
  dependencies: [triage]
  exec:
    source: |
      Assess if rollback is the best immediate action:

      1. Is there a recent deployment?
      2. Can we identify the change that caused it?
      3. What is the rollback risk?
      4. How long would rollback take?

      Recommendation: ROLLBACK / HOTFIX / INVESTIGATE
```

</div>

## Incident Response Checklist

1. **Detect** - Alert fires, customer report, monitoring
2. **Triage** - Assess severity, blast radius, urgency
3. **Communicate** - Status page, stakeholders, customers
4. **Investigate** - Logs, metrics, recent changes
5. **Mitigate** - Rollback, feature flag, scaling
6. **Resolve** - Deploy fix, verify recovery
7. **Document** - Postmortem, action items, learnings

## Related Use Cases

- [Security Audit](./security-audit) - Investigate security incidents
- [Code Review](/use-cases/code-review) - Review incident fixes
- [Multi-Agent Review](./multi-agent-review) - Comprehensive fix review

## Next Steps

- [Concepts: Pipelines](/concepts/pipelines) - Understand pipeline execution
- [Concepts: Contracts](/concepts/contracts) - Validate incident reports

<style>
.use-case-meta {
  display: flex;
  gap: 8px;
  margin-bottom: 24px;
}
.complexity-badge {
  padding: 4px 12px;
  font-size: 12px;
  font-weight: 600;
  border-radius: 12px;
  text-transform: uppercase;
}
.complexity-badge.beginner {
  background: #dcfce7;
  color: #166534;
}
.complexity-badge.intermediate {
  background: #fef3c7;
  color: #92400e;
}
.complexity-badge.advanced {
  background: #fee2e2;
  color: #991b1b;
}
.category-badge {
  padding: 4px 12px;
  font-size: 12px;
  font-weight: 500;
  border-radius: 12px;
  background: var(--vp-c-brand-soft);
  color: var(--vp-c-brand-1);
}
</style>
