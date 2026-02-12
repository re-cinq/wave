# Documentation Consistency

Pre-PR documentation consistency gate that scans code changes, cross-references docs, and creates a GitHub issue with inconsistencies.

## Prerequisites

- `gh` CLI authenticated with repo access

## Overview

```mermaid
graph TD
    S[Scan Changes] --> A[Analyze Consistency]
    A --> I[Create Issue]
```

The doc-loop pipeline catches documentation drift before it reaches main. It compares code changes against docs, CLI help text, and README content, then opens a GitHub issue listing every mismatch.

## Running

```bash
# Scan branch diff against main
wave run doc-loop

# Full scan of all documentation
wave run doc-loop "full"

# Scan against a specific ref
wave run doc-loop "v0.2.0"
```

## Expected Output

With `-o text`:

```
[10:00:01] → scan-changes (navigator)
[10:00:01]   scan-changes: Executing agent
[10:01:12] ✓ scan-changes completed (71.0s, 4.2k tokens)
[10:01:13] → analyze-consistency (reviewer)
[10:03:45] ✓ analyze-consistency completed (152.0s, 6.1k tokens)
[10:03:46] → create-issue (github-analyst)
[10:04:22] ✓ create-issue completed (36.0s, 1.4k tokens)

  ✓ Pipeline 'doc-loop' completed successfully (4m 21s)
```

## Steps

| Step | Persona | Description |
|------|---------|-------------|
| `scan-changes` | navigator | Categorize changed files, snapshot documentation state |
| `analyze-consistency` | reviewer | Cross-reference code against docs, rate severity |
| `create-issue` | github-analyst | Create GitHub issue with task list of inconsistencies |

## Artifacts

| Artifact | Path | Description |
|----------|------|-------------|
| `scan-results` | `output/scan-results.json` | Changed files by category, doc snapshots |
| `consistency-report` | `output/consistency-report.json` | Inconsistencies with severity ratings |
| `issue-result` | `output/issue-result.json` | Created issue number, URL, or skip status |
