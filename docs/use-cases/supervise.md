---
title: Work Supervision
description: Review work quality and process quality, including AI session transcripts
---

# Work Supervision

<div class="use-case-meta">
  <span class="complexity-badge intermediate">Intermediate</span>
  <span class="category-badge">Quality Assurance</span>
</div>

Supervise completed work by evaluating both the **output quality** (correctness, completeness, test coverage) and the **process quality** (efficiency, scope discipline, tool usage). Reads claudit session transcripts stored as git notes to understand not just *what* was done, but *how* it was done.

## Prerequisites

- Wave installed and initialized (`wave init`)
- Git repository with recent work to review
- Optional: [claudit](https://github.com/re-cinq/claudit) for session transcript storage via git notes

## Quick Start

```bash
# Auto-detect last pipeline run
wave run supervise

# Review a specific pipeline run
wave run supervise "last pipeline run"

# Review a specific branch
wave run supervise "feature/add-auth"

# Review current PR
wave run supervise "current pr"
```

With `-o text`:

```
[10:00:01] -> gather (supervisor)
[10:00:01]   gather: Executing agent
[10:04:30] + gather completed (269s, 8.2k tokens)
[10:04:31] -> evaluate (supervisor)
[10:08:45] + evaluate completed (254s, 6.1k tokens)
[10:08:46] -> verdict (reviewer)
[10:12:30] + verdict completed (224s, 3.8k tokens)

  + Pipeline 'supervise' completed successfully (748s)
```

## Pipeline Structure

```
gather (supervisor) -> evaluate (supervisor) -> verdict (reviewer)
```

All three steps use **readonly** workspace mounts -- this is a purely analytical pipeline that never modifies code.

### Step 1: Gather Evidence

The `supervisor` persona parses input heuristically to determine what to inspect:

| Input | Detection Strategy |
|-------|-------------------|
| *(empty)* | Most recent pipeline run from `.wave/workspaces/` |
| `"last pipeline run"` | Same as empty |
| `"current pr"` or `"PR #42"` | Current or specified pull request |
| `"feature/auth"` | All commits on that branch vs main |
| Free-form text | Search via grep/git log |

Evidence collected includes:
- Recent commits with diffs and stats
- Claudit session transcripts from git notes
- Pipeline workspace artifacts
- Test results and coverage
- Branch and PR state

### Step 2: Evaluate Quality

The `supervisor` scores each dimension as **excellent / good / adequate / poor**:

**Output Quality:**
- Correctness, completeness, test coverage, code quality

**Process Quality:**
- Efficiency, scope discipline, tool usage, token economy

### Step 3: Final Verdict

The `reviewer` independently verifies claims, runs the test suite, and issues a verdict:

- **APPROVE** -- work is good quality, process was efficient
- **PARTIAL_APPROVE** -- output acceptable but process had notable issues
- **REWORK** -- significant issues requiring attention

## Expected Outputs

| Artifact | Path | Description |
|----------|------|-------------|
| `evidence` | `.wave/output/supervision-evidence.json` | Raw evidence bundle with commits, artifacts, transcripts |
| `evaluation` | `.wave/output/supervision-evaluation.json` | Scored evaluation across all quality dimensions |
| `verdict` | `.wave/output/supervision-verdict.md` | Final verdict with action items and lessons learned |

### Example Output

The pipeline produces `.wave/output/supervision-verdict.md`:

```markdown
## Verdict: PARTIAL_APPROVE

## Output Quality
The implementation is correct and complete. All 47 tests pass,
including 12 new tests added for the feature. Code follows
existing project conventions.

## Process Quality
The agent took 3 unnecessary detours:
1. Read 14 unrelated files before finding the target module
2. Attempted a refactor that was reverted after 200 lines of changes
3. Re-ran the full test suite 5 times when targeted tests would suffice

Estimated 30% of tokens were spent on non-productive exploration.

## Action Items
- should-fix: Consider using targeted `go test ./internal/pipeline/...`
  instead of full suite during iterative development

## Lessons Learned
- Scope the initial exploration phase more tightly
- Use Glob/Grep before reading files to narrow candidates
```

## Customization

### Focus on process quality only

```bash
wave run supervise "focus on process efficiency of the last pipeline run"
```

### Review a specific PR

```bash
wave run supervise "PR #42"
```

## Related Use Cases

- [Code Review](/use-cases/code-review) - Automated PR review for code quality
- [Simplify](/use-cases/simplify) - Rethink and simplify code

## Next Steps

- [Concepts: Contracts](/concepts/contracts) - Understand contract validation
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
