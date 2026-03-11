# Data Model: Closed-Issue/PR Audit Pipeline (#305)

**Branch**: `305-audit-pipeline` | **Date**: 2026-03-11

## Domain Entities

### InventoryItem

A closed issue or merged PR with extracted metadata for auditing.

```json
{
  "number": 42,
  "type": "issue",
  "title": "Add verbose flag for pipeline execution",
  "url": "https://github.com/re-cinq/wave/issues/42",
  "body": "Full issue body text...",
  "labels": ["enhancement", "cli"],
  "close_reason": "completed",
  "closed_at": "2026-02-15T10:30:00Z",
  "linked_prs": [55],
  "linked_commits": ["abc1234"],
  "acceptance_criteria": [
    "wave run --verbose shows step-level output",
    "verbose flag documented in help text"
  ]
}
```

**Fields**:
- `number` (integer, required): GitHub issue or PR number
- `type` (string, required): `"issue"` or `"pr"`
- `title` (string, required): Issue/PR title
- `url` (string, required): Full GitHub URL
- `body` (string, required): Full body text for analysis
- `labels` (string[], required): Associated labels
- `close_reason` (string, required): `"completed"`, `"not_planned"`, or `"merged"` (for PRs)
- `closed_at` (string, required): ISO 8601 timestamp
- `linked_prs` (integer[]): PR numbers linked to issues
- `linked_commits` (string[]): Commit SHAs associated with the item
- `acceptance_criteria` (string[]): Extracted acceptance criteria from body (if parseable)

**Extraction rules**:
- Issues closed as `"not_planned"` are excluded from inventory (FR-011)
- Acceptance criteria are extracted by the persona by looking for checklist patterns (`- [ ]`, `- [x]`) or sections titled "Acceptance Criteria" in the body
- For PRs, `linked_commits` includes the merge commit SHA; `linked_prs` is empty

### AuditFinding

The result of auditing one inventory item against the current codebase.

```json
{
  "item_number": 42,
  "item_type": "issue",
  "item_url": "https://github.com/re-cinq/wave/issues/42",
  "title": "Add verbose flag for pipeline execution",
  "category": "partial",
  "evidence": [
    "cmd/wave/commands/run.go:45 — --verbose flag defined in cobra command",
    "internal/pipeline/executor.go — no verbose output handling found"
  ],
  "unmet_criteria": [
    "verbose flag documented in help text — not found in README.md or docs/"
  ],
  "remediation": "Add verbose output handling in executor.go and document the flag in README.md"
}
```

**Fields**:
- `item_number` (integer, required): Source issue/PR number
- `item_type` (string, required): `"issue"` or `"pr"`
- `item_url` (string, required): Full GitHub URL for reference
- `title` (string, required): Item title for readability
- `category` (string, required): One of the five fidelity categories
- `evidence` (string[], required): File paths, code references, commit SHAs supporting the classification
- `unmet_criteria` (string[]): Specific acceptance criteria that were not satisfied (for partial/regressed)
- `remediation` (string): Actionable description of what needs to change (empty for verified/obsolete)

### FidelityCategory

Enumeration of the five classification states.

| Category | Meaning | Evidence Pattern |
|----------|---------|-----------------|
| `verified` | Fully implemented and intact at HEAD | All referenced files exist, key functions present, tests exist |
| `partial` | Some acceptance criteria unmet or incomplete | Some but not all criteria have matching code evidence |
| `regressed` | Was implemented but later broken or reverted | Files modified/deleted after implementing PR, revert commits found |
| `obsolete` | Codebase diverged enough that the item no longer applies | Referenced files deleted, architecture has changed |
| `unverifiable` | No linked PRs, commits, or traceable code changes | Issue body has no file references, no linked PRs |

### TriageReport

Aggregated output grouping findings by category with prioritized actions.

```json
{
  "metadata": {
    "scope": "full",
    "timestamp": "2026-03-11T10:00:00Z",
    "repository": "re-cinq/wave",
    "total_items_audited": 150
  },
  "summary": {
    "verified": 120,
    "partial": 15,
    "regressed": 5,
    "obsolete": 8,
    "unverifiable": 2
  },
  "findings": [],
  "prioritized_actions": []
}
```

**Priority ordering** for `prioritized_actions`:
1. `regressed` items (highest priority — was working, now broken)
2. `partial` items with most unmet criteria
3. `partial` items with fewer unmet criteria
4. `unverifiable` items (lowest actionable priority)
5. `obsolete` items are excluded from actions (intentionally non-applicable)

### PublishResult

Status of GitHub issue creation for actionable findings.

```json
{
  "success": true,
  "repository": "re-cinq/wave",
  "issues_created": [
    {
      "number": 310,
      "url": "https://github.com/re-cinq/wave/issues/310",
      "source_item": 42,
      "category": "partial"
    }
  ],
  "issues_skipped": 0,
  "timestamp": "2026-03-11T10:30:00Z"
}
```

## Contract Schemas

Four JSON Schema contracts, one per step output:

| Contract File | Step | Validates |
|--------------|------|-----------|
| `audit-inventory.schema.json` | `collect-inventory` | Inventory with items array |
| `audit-findings.schema.json` | `audit-items` | Findings with per-item classifications |
| `audit-triage-report.schema.json` | `compose-triage` | Triage report with summary + prioritized actions |
| `audit-publish-result.schema.json` | `publish` | Issue creation status |

See `specs/305-audit-pipeline/contracts/` for full schemas.

## Artifact Flow

```
CLI Input ("last 30 days" / "label:X" / empty)
    │
    ▼
┌──────────────────────┐
│  collect-inventory    │ github-analyst
│  Output: inventory    │
└──────────┬───────────┘
           │ inventory.json
           ▼
┌──────────────────────┐
│  audit-items          │ navigator
│  Input: inventory     │
│  Output: findings     │
└──────────┬───────────┘
           │ audit-findings.json
           ▼
┌──────────────────────┐
│  compose-triage       │ navigator
│  Input: findings      │
│  Output: triage-report│
└──────────┬───────────┘
           │ triage-report.json
           ▼
┌──────────────────────┐
│  publish (optional)   │ craftsman
│  Input: triage-report │
│  Output: publish-result│
└──────────────────────┘
```

## File Organization

New files (all in `.wave/` directory):

```
.wave/
├── pipelines/
│   └── wave-audit.yaml              # NEW: Pipeline definition
├── contracts/
│   ├── audit-inventory.schema.json   # NEW: Inventory step contract
│   ├── audit-findings.schema.json    # NEW: Audit step contract
│   ├── audit-triage-report.schema.json # NEW: Triage step contract
│   └── audit-publish-result.schema.json # NEW: Publish step contract
```

No Go code changes required — this is a pipeline-only feature using existing personas and Wave infrastructure.
