---
title: Simplify
description: Rethink and simplify code using divergent-convergent thinking (Double Diamond)
---

# Simplify

<div class="use-case-meta">
  <span class="complexity-badge advanced">Advanced</span>
  <span class="category-badge">Code Quality</span>
</div>

Rethink and simplify code by systematically challenging assumptions and reducing accidental complexity. Structured around the **Double Diamond** model -- Guilford's divergent/convergent thinking oscillation -- where agents alternate between casting a wide net and narrowing to actionable proposals.

## Prerequisites

- Wave installed and initialized (`wave init`)
- Git repository with code to analyze
- Passing test suite (the pipeline commits to a worktree branch)

## Quick Start

```bash
# Analyze the whole project
wave run simplify

# Target a specific module
wave run simplify "internal/pipeline"

# Target a specific concern
wave run simplify "internal/adapter internal/manifest"
```

With `-o text`:

```
[10:00:01] -> diverge (provocateur)
[10:00:01]   diverge: Executing agent
[10:09:00] + diverge completed (539s, 12.4k tokens)
[10:09:01] -> distill (planner)
[10:14:41] + distill completed (340s, 8.7k tokens)
[10:14:42] -> simplify (craftsman)
[10:19:22] + simplify completed (280s, 6.2k tokens)

  + Pipeline 'simplify' completed successfully (1161s)
```

## The Double Diamond

The pipeline embodies the divergent/convergent thinking oscillation:

```
Step 1 (DIVERGE)           Step 2 (CONVERGE->DIVERGE->CONVERGE)       Step 3 (CONVERGE)
-----------------          -----------------------------------        -----------------
Cast widest net     ->     Validate -> Probe deeper -> Synthesize ->  Apply & stabilize
   provocateur                        planner                           craftsman
```

### Step 1: Diverge (provocateur, temp 0.8)

The highest-temperature persona in the system. Casts the widest possible net:

- **Premature abstractions** -- interfaces with one implementation, generics used once
- **Unnecessary indirection** -- layers that pass through without adding value
- **Overengineering** -- configuration for things that never change
- **YAGNI violations** -- code for hypothetical futures that never arrived
- **Accidental complexity** -- things that are hard because of how they're built
- **Copy-paste drift** -- similar code that diverged accidentally
- **Dead weight** -- unused exports, unreachable code, stale TODOs
- **Naming lies** -- names that don't match actual behavior
- **Dependency gravity** -- modules that pull in too much

Every finding gets a **DVG-xxx** ID and must include concrete metrics (line counts, grep counts, change frequency).

### Step 2: Distill (planner, temp 0.3)

The pivot step -- explicitly oscillates between opening up and narrowing down:

1. **CONVERGE**: Validate each DVG finding against actual code. Confirmed / partially confirmed / rejected.
2. **DIVERGE AGAIN**: For confirmed findings, probe deeper. What else connects? What are second-order effects? What patterns emerge across findings?
3. **CONVERGE AGAIN**: Synthesize into prioritized **SMP-xxx** proposals with impact/effort/risk matrix and 80/20 analysis.

### Step 3: Simplify (craftsman, temp 0.3)

Final convergence on a worktree branch `refactor/<pipeline-id>`:

- Applies **tier-1 proposals only**, in dependency order
- Each proposal: apply -> build -> test -> commit (or revert if tests fail)
- Every proposal gets its own atomic commit
- Full test suite verification at the end

## Expected Outputs

| Artifact | Path | Description |
|----------|------|-------------|
| `findings` | `.wave/output/divergent-findings.json` | DVG-xxx findings with evidence and metrics |
| `proposals` | `.wave/output/convergent-proposals.json` | SMP-xxx prioritized proposals with 80/20 analysis |
| `result` | `.wave/output/result.md` | Summary of applied changes on worktree branch |

Plus: **committed changes** on branch `refactor/<pipeline-id>` with atomic commits per proposal.

### Example Divergent Finding

```json
{
  "id": "DVG-003",
  "category": "premature_abstraction",
  "title": "WorkspaceProvider interface has single implementation",
  "description": "The WorkspaceProvider interface in workspace/provider.go is only implemented by LocalProvider. The abstraction adds indirection without flexibility.",
  "evidence": {
    "files": ["internal/workspace/provider.go", "internal/workspace/local.go"],
    "line_count": 45,
    "reference_count": 3,
    "change_frequency": 2,
    "metrics": "Interface: 8 methods, 1 implementation, 3 call sites"
  },
  "severity": "medium",
  "confidence": "high"
}
```

### Example Convergent Proposal

```json
{
  "id": "SMP-001",
  "title": "Inline WorkspaceProvider into concrete type",
  "description": "Remove the WorkspaceProvider interface and use LocalProvider directly. If a second implementation is needed later, extract the interface then.",
  "source_findings": ["DVG-003"],
  "impact": "medium",
  "effort": "small",
  "risk": "low",
  "tier": 1,
  "files": ["internal/workspace/provider.go", "internal/workspace/local.go", "internal/pipeline/executor.go"],
  "lines_removed_estimate": 35,
  "second_order_effects": ["Simplifies executor.go constructor by removing interface binding"]
}
```

## Customization

### Analysis only (skip implementation)

Run just the first two steps to get proposals without applying them:

```bash
wave run simplify "internal/pipeline" --to-step distill
```

### Review proposals before applying

```bash
# Run diverge + distill
wave run simplify "internal/pipeline" --to-step distill

# Review the proposals
cat .wave/workspaces/simplify/*/.wave/output/convergent-proposals.json | jq .

# If satisfied, resume from simplify
wave run simplify --from-step simplify
```

## Related Use Cases

- [Refactoring](/use-cases/refactoring) - Targeted refactoring with test safety net
- [Work Supervision](/use-cases/supervise) - Review quality of completed work

## Next Steps

- [Concepts: Pipelines](/concepts/pipelines) - Understand pipeline execution
- [Concepts: Personas](/concepts/personas) - Learn about persona capabilities

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
