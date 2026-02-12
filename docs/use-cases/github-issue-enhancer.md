# Issue Enhancement

Scan GitHub issues for poor documentation, generate improvements, and apply them automatically.

## Prerequisites

- `gh` CLI authenticated with write access to the target repository

## Overview

```mermaid
graph TD
    S[Scan Issues] --> P[Plan Enhancements]
    P --> A[Apply Enhancements]
    A --> V[Verify]
```

The github-issue-enhancer pipeline finds poorly documented issues, rewrites titles and bodies with proper structure, adds labels, and verifies the changes were applied.

## Running

```bash
# Enhance a single issue
wave run github-issue-enhancer "re-cinq/wave 42"

# Batch mode — scan and enhance up to 10 issues
wave run github-issue-enhancer "re-cinq/wave"
```

## Expected Output

```
[10:00:01] started   scan-issues         (github-analyst)         Starting step
[10:00:38] completed scan-issues         (github-analyst)  37s   2.8k Scan complete
[10:00:39] started   plan-enhancements   (github-analyst)         Starting step
[10:01:52] completed plan-enhancements   (github-analyst)  73s   3.5k Plan complete
[10:01:53] started   apply-enhancements  (github-enhancer)        Starting step
[10:02:41] completed apply-enhancements  (github-enhancer) 48s   1.9k Applied
[10:02:42] started   verify-enhancements (github-analyst)         Starting step
[10:03:15] completed verify-enhancements (github-analyst)  33s   1.2k Verified

Pipeline github-issue-enhancer completed in 3m 14s
Artifacts: artifact.json
```

## Steps

| Step | Persona | Description |
|------|---------|-------------|
| `scan-issues` | github-analyst | Fetch and score issue quality |
| `plan-enhancements` | github-analyst | Draft improved titles, bodies, labels |
| `apply-enhancements` | github-enhancer | Apply changes via `gh issue edit` |
| `verify-enhancements` | github-analyst | Re-fetch issues and verify changes stuck |

## Artifacts

| Artifact | Path | Description |
|----------|------|-------------|
| `issue_analysis` | `artifact.json` | Quality scores and poor-quality issue list |
| `enhancement_plan` | `artifact.json` | Suggested titles, bodies, labels per issue |
| `enhancement_results` | `artifact.json` | Applied changes and success/failure counts |
| `verification_report` | `artifact.json` | Post-apply verification results |
