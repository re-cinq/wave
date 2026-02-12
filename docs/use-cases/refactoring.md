---
title: Refactoring
description: Systematic code refactoring with analysis, implementation, and verification steps
---

# Refactoring

<div class="use-case-meta">
  <span class="complexity-badge intermediate">Intermediate</span>
  <span class="category-badge">Code Quality</span>
</div>

Perform systematic code refactoring with analysis, safe implementation, and verification. Wave's refactoring pipeline identifies improvement opportunities, applies changes incrementally, and verifies behavior is preserved.

## Prerequisites

- Wave installed and initialized (`wave init`)
- Git repository with tests (highly recommended)
- Understanding of [code-review](/use-cases/code-review) pipeline
- Familiarity with common refactoring patterns

## Quick Start

```bash
wave run refactor "refactor the database connection handling for better error management"
```

Expected output:

```
[10:00:01] started   analyze           (navigator)              Starting step
[10:00:35] completed analyze           (navigator)   34s   3.1k Analysis complete
[10:00:36] started   plan              (philosopher)            Starting step
[10:01:05] completed plan              (philosopher)  29s   2.4k Plan complete
[10:01:06] started   implement         (craftsman)              Starting step
[10:02:15] completed implement         (craftsman)   69s   6.8k Implementation complete
[10:02:16] started   verify            (auditor)                Starting step
[10:02:45] completed verify            (auditor)     29s   2.1k Verification complete

Pipeline refactor completed in 164s
Artifacts: output/refactoring-report.md
```

## Complete Pipeline

Save the following YAML to `.wave/pipelines/refactor.yaml`:

<div v-pre>

```yaml
kind: WavePipeline
metadata:
  name: refactor
  description: "Systematic code refactoring with verification"

input:
  source: cli

steps:
  - id: analyze
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
        Analyze the codebase for refactoring: {{ input }}

        Identify:
        1. Code smells (duplication, long methods, large classes)
        2. Coupling issues (tight coupling, circular dependencies)
        3. Naming inconsistencies
        4. Dead code or unused exports
        5. Existing test coverage for affected areas
        6. Risk assessment for each area

        Output as JSON:
        {
          "target_areas": [{"file": "", "issue": "", "risk": "low|medium|high"}],
          "dependencies": [],
          "test_coverage": {},
          "recommended_order": []
        }
    output_artifacts:
      - name: analysis
        path: output/refactor-analysis.json
        type: json

  - id: plan
    persona: philosopher
    dependencies: [analyze]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: analyze
          artifact: analysis
          as: analysis
    exec:
      type: prompt
      source: |
        Create a refactoring plan based on the analysis.

        For each change:
        1. Describe the transformation (rename, extract, inline, move)
        2. List affected files
        3. Identify breaking changes
        4. Define verification criteria
        5. Order changes to minimize risk

        Prioritize:
        - Changes with high test coverage first
        - Independent changes before dependent ones
        - Smaller, safer changes before larger ones
    output_artifacts:
      - name: plan
        path: output/refactor-plan.md
        type: markdown

  - id: implement
    persona: craftsman
    dependencies: [plan]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: analyze
          artifact: analysis
          as: analysis
        - step: plan
          artifact: plan
          as: plan
    workspace:
      mount:
        - source: ./
          target: /src
          mode: readwrite
    exec:
      type: prompt
      source: |
        Implement the refactoring plan.

        Guidelines:
        1. Make one logical change at a time
        2. Preserve behavior - this is not a feature change
        3. Update tests to match new structure
        4. Keep commits atomic and revertible
        5. Add comments explaining non-obvious changes

        After each major change, verify tests still pass.
    handover:
      contract:
        type: test_suite
        command: "go test ./... -v"
        must_pass: true
        on_failure: retry
        max_retries: 3
    output_artifacts:
      - name: changes
        path: output/refactor-changes.md
        type: markdown

  - id: verify
    persona: auditor
    dependencies: [implement]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: plan
          artifact: plan
          as: original_plan
        - step: implement
          artifact: changes
          as: actual_changes
    exec:
      type: prompt
      source: |
        Verify the refactoring was successful:

        1. All planned changes implemented correctly?
        2. Tests still passing?
        3. No behavioral changes introduced?
        4. Code quality improved?
        5. Any remaining issues?

        Output: verification report with pass/fail status
    output_artifacts:
      - name: report
        path: output/refactoring-report.md
        type: markdown
```

</div>

## Expected Outputs

The pipeline produces four artifacts:

| Artifact | Path | Description |
|----------|------|-------------|
| `analysis` | `output/refactor-analysis.json` | Analysis of refactoring targets |
| `plan` | `output/refactor-plan.md` | Detailed refactoring plan |
| `changes` | `output/refactor-changes.md` | Log of implemented changes |
| `report` | `output/refactoring-report.md` | Verification report |

### Example Output

The pipeline produces `output/refactoring-report.md`:

```markdown
# Refactoring Report

**Target**: Database connection handling
**Status**: PASSED

## Summary

Successfully refactored the database connection handling to improve error
management. All tests pass, and the changes are backward compatible.

## Changes Implemented

| Change | Files | Status |
|--------|-------|--------|
| Extract `ConnectionPool` interface | db/pool.go | DONE |
| Add retry logic with exponential backoff | db/connect.go | DONE |
| Wrap errors with context | db/errors.go | DONE |
| Update tests for new error types | db/pool_test.go | DONE |

## Metrics

| Metric | Before | After | Change |
|--------|--------|-------|--------|
| Cyclomatic Complexity | 15 | 8 | -47% |
| Lines of Code | 450 | 380 | -16% |
| Test Coverage | 72% | 78% | +6% |

## Verification

- [x] All unit tests pass
- [x] Integration tests pass
- [x] No behavioral changes (API compatible)
- [x] Error messages improved
- [x] Retry logic handles transient failures

## Remaining Items

None. Refactoring complete.
```

## Customization

### Extract methods

```bash
wave run refactor "extract common validation logic into reusable functions"
```

### Rename for consistency

```bash
wave run refactor "rename all handler functions to follow HandleXxx pattern"
```

### Reduce coupling

```bash
wave run refactor "decouple the notification service from email implementation"
```

### Add dependency injection

```bash
wave run refactor "add dependency injection to the user service for testability"
```

### Safe mode (dry run)

Create a variant that only produces a plan:

<div v-pre>

```yaml
kind: WavePipeline
metadata:
  name: refactor-plan
  description: "Plan refactoring without implementing"

steps:
  - id: analyze
    # ... same as above

  - id: plan
    # ... same as above
    # No implement step - just output the plan
```

</div>

## Best Practices

### Before Refactoring

1. **Ensure test coverage** - Run `go test -cover` to verify coverage
2. **Commit current state** - Create a clean baseline
3. **Review the plan** - Check the generated plan before implementing

### During Refactoring

1. **Small steps** - Make incremental changes
2. **Run tests frequently** - Verify after each change
3. **Keep commits atomic** - One logical change per commit

### After Refactoring

1. **Full test run** - Verify all tests pass
2. **Code review** - Use [code-review](/use-cases/code-review) pipeline
3. **Documentation** - Update any affected docs

## Related Use Cases

- [Code Review](/use-cases/code-review) - Review refactoring changes
- [Test Generation](/use-cases/test-generation) - Add tests before refactoring

## Next Steps

- [Concepts: Contracts](/concepts/contracts) - Ensure tests pass during refactoring
- [Concepts: Personas](/concepts/personas) - Understanding the craftsman persona

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
