---
title: Migration
description: Plan and execute codebase migrations with dependency analysis and verification
---

# Migration

<div class="use-case-meta">
  <span class="complexity-badge advanced">Advanced</span>
  <span class="category-badge">DevOps</span>
</div>

Plan and execute codebase migrations with dependency analysis, implementation planning, and verification. This pipeline handles framework upgrades, language version migrations, and architectural transformations.

## Prerequisites

- Wave installed and initialized (`wave init`)
- Git repository with version control
- Experience with [refactoring](/use-cases/refactoring) pipeline
- Understanding of [test-generation](/use-cases/test-generation) for verification
- Comprehensive test coverage (highly recommended)

## Quick Start

```bash
wave run migrate "upgrade from Go 1.21 to Go 1.25 with generics adoption"
```

Expected output:

```
[10:00:01] started   analyze           (navigator)              Starting step
[10:00:45] completed analyze           (navigator)   44s   4.8k Analysis complete
[10:00:46] started   plan              (philosopher)            Starting step
[10:01:32] completed plan              (philosopher)  46s   5.2k Plan complete
[10:01:33] started   implement         (craftsman)              Starting step
[10:03:15] completed implement         (craftsman)  102s   9.5k Implementation complete
[10:03:16] started   verify            (auditor)                Starting step
[10:03:52] completed verify            (auditor)     36s   3.1k Verification complete
[10:03:53] started   document          (summarizer)             Starting step
[10:04:18] completed document          (summarizer)  25s   2.8k Documentation complete

Pipeline migrate completed in 257s
Artifacts: output/migration-report.md
```

## Complete Pipeline

Save the following YAML to `.wave/pipelines/migrate.yaml`:

```yaml
kind: WavePipeline
metadata:
  name: migrate
  description: "Plan and execute codebase migrations"

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
        Analyze the codebase for migration: {{ input }}

        Identify:
        1. Current versions (language, frameworks, dependencies)
        2. Target versions and changes needed
        3. Breaking changes to address
        4. Deprecated APIs to replace
        5. New features to adopt
        6. Dependencies requiring updates
        7. Test coverage of affected areas
        8. Risk assessment by component

        Output as JSON:
        {
          "current_state": {},
          "target_state": {},
          "breaking_changes": [],
          "deprecations": [],
          "new_features": [],
          "dependency_updates": [],
          "affected_files": [],
          "risk_areas": []
        }
    output_artifacts:
      - name: analysis
        path: output/migration-analysis.json
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
        Create a migration plan for: {{ input }}

        Structure the plan in phases:
        1. **Preparation**: Environment setup, dependency updates
        2. **Migration**: Code changes, API updates
        3. **Verification**: Testing, validation
        4. **Cleanup**: Remove deprecated code, update docs

        For each phase:
        - List specific tasks
        - Order by dependencies
        - Estimate effort
        - Identify rollback points
        - Define success criteria

        Prioritize:
        - Safety over speed
        - Incremental changes over big bang
        - Tests before changes
    output_artifacts:
      - name: plan
        path: output/migration-plan.md
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
        Execute the migration plan.

        Guidelines:
        1. Follow the plan phases in order
        2. Make atomic, reversible changes
        3. Run tests after each significant change
        4. Document any deviations from plan
        5. Create checkpoint commits

        For each change:
        - What was changed
        - Why it was needed
        - How to verify
        - How to rollback if needed
    handover:
      contract:
        type: test_suite
        command: "go test ./... -v"
        must_pass: true
        on_failure: retry
        max_retries: 3
    output_artifacts:
      - name: changes
        path: output/migration-changes.md
        type: markdown

  - id: verify
    persona: auditor
    dependencies: [implement]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: analyze
          artifact: analysis
          as: original_analysis
        - step: plan
          artifact: plan
          as: original_plan
        - step: implement
          artifact: changes
          as: actual_changes
    workspace:
      mount:
        - source: ./
          target: /src
          mode: readonly
    exec:
      type: prompt
      source: |
        Verify the migration was successful.

        Check:
        1. All planned changes implemented?
        2. All tests passing?
        3. No regressions introduced?
        4. Performance acceptable?
        5. No deprecated APIs remaining?
        6. Dependencies correctly updated?
        7. Build and deployment work?

        Run verification commands:
        - go build ./...
        - go test ./... -race
        - go vet ./...

        Output: verification report with pass/fail status
    output_artifacts:
      - name: verification
        path: output/migration-verification.md
        type: markdown

  - id: document
    persona: summarizer
    dependencies: [verify]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: analyze
          artifact: analysis
          as: analysis
        - step: plan
          artifact: plan
          as: plan
        - step: implement
          artifact: changes
          as: changes
        - step: verify
          artifact: verification
          as: verification
    exec:
      type: prompt
      source: |
        Create comprehensive migration documentation.

        Include:
        1. Executive Summary
        2. Migration Scope (what changed)
        3. Breaking Changes (for consumers)
        4. Upgrade Guide (step-by-step)
        5. New Features Available
        6. Known Issues and Workarounds
        7. Rollback Procedure
        8. Changelog

        Write for both developers and operators.
    output_artifacts:
      - name: report
        path: output/migration-report.md
        type: markdown
```

## Expected Outputs

The pipeline produces five artifacts:

| Artifact | Path | Description |
|----------|------|-------------|
| `analysis` | `output/migration-analysis.json` | Current state and migration scope |
| `plan` | `output/migration-plan.md` | Phased migration plan |
| `changes` | `output/migration-changes.md` | Log of implemented changes |
| `verification` | `output/migration-verification.md` | Verification results |
| `report` | `output/migration-report.md` | Complete migration documentation |

### Example Output

The pipeline produces `output/migration-report.md`:

```markdown
# Migration Report: Go 1.21 to Go 1.25

**Date**: 2026-02-04
**Status**: COMPLETED
**Duration**: 4 hours (automated) + 1 hour (review)

## Executive Summary

Successfully migrated from Go 1.21 to Go 1.25, adopting generics in
collection utilities and improving type safety across the codebase.
All tests pass, performance improved by 12% on average.

## Migration Scope

### Version Changes

| Component | Before | After |
|-----------|--------|-------|
| Go | 1.21.0 | 1.25.0 |
| go.mod | go 1.21 | go 1.25 |
| Docker base | golang:1.21 | golang:1.25 |

### Files Changed

- **Modified**: 47 files
- **Added**: 3 files (generic utilities)
- **Deleted**: 5 files (type-specific utilities)

### Dependencies Updated

| Package | Before | After | Notes |
|---------|--------|-------|-------|
| golang.org/x/sync | v0.3.0 | v0.6.0 | Required for Go 1.25 |
| github.com/stretchr/testify | v1.8.0 | v1.9.0 | Optional update |

## Breaking Changes

### For Library Consumers

1. **Collection functions now use generics**
   ```go
   // Before
   func Map(items []interface{}, fn func(interface{}) interface{}) []interface{}

   // After
   func Map[T, U any](items []T, fn func(T) U) []U
   ```

2. **Removed deprecated APIs**
   - `utils.IntSliceContains` - Use `slices.Contains` instead
   - `utils.StringSet` - Use `map[string]struct{}` or generic `Set[T]`

## Upgrade Guide

### Step 1: Update Go Version

```bash
# Update go.mod
go mod edit -go=1.25

# Update Docker images
sed -i 's/golang:1.21/golang:1.25/g' Dockerfile
```

### Step 2: Update Dependencies

```bash
go get -u ./...
go mod tidy
```

### Step 3: Update Code

```go
// Replace type-specific functions with generics
// Before
result := utils.MapStrings(items, transform)
// After
result := utils.Map(items, transform)
```

### Step 4: Verify

```bash
go build ./...
go test ./... -race
go vet ./...
```

## New Features Available

With Go 1.25, you can now use:

1. **Generic collections** - Type-safe Map, Filter, Reduce
2. **Range over integers** - `for i := range 10`
3. **Improved error handling** - Enhanced error wrapping
4. **Performance improvements** - Faster compilation, smaller binaries

## Known Issues

1. **IDE support** - Ensure your IDE is updated for Go 1.25 generics
2. **CI caching** - Clear Go module cache after upgrade

## Rollback Procedure

If issues arise, rollback with:

```bash
git revert HEAD~5..HEAD  # Revert migration commits
go mod edit -go=1.21
go mod tidy
```

## Changelog

### Added
- Generic collection utilities (Map, Filter, Reduce, Find)
- Generic Set[T] implementation
- Comprehensive type constraints

### Changed
- Updated all collection operations to use generics
- Improved error messages with type information
- Updated CI/CD for Go 1.25

### Removed
- Type-specific utility functions (replaced by generics)
- Deprecated StringSet and IntSet types

### Performance
- Binary size: -8% (28MB -> 25.7MB)
- Build time: -15% (45s -> 38s)
- Test execution: -12% (120s -> 105s)
```

## Customization

### Framework migration

```bash
wave run migrate "migrate from Echo to Chi router framework"
```

### Database migration

```bash
wave run migrate "migrate from PostgreSQL 12 to PostgreSQL 16"
```

### Dependency upgrade

```bash
wave run migrate "upgrade all dependencies to latest stable versions"
```

### Dry run mode

Create a planning-only variant:

```yaml
kind: WavePipeline
metadata:
  name: migrate-plan
  description: "Plan migration without implementing"

steps:
  - id: analyze
    # ... analysis step

  - id: plan
    # ... planning step
    # No implement step - just output the plan
```

### Add rollback testing

```yaml
- id: rollback-test
  persona: auditor
  dependencies: [implement]
  exec:
    source: |
      Test rollback procedure:
      1. Create checkpoint
      2. Apply rollback commands
      3. Verify system returns to previous state
      4. Re-apply migration
      5. Verify migration succeeds again
```

## Migration Best Practices

### Before Migration

1. **Ensure test coverage** - At least 70% for critical paths
2. **Create baseline** - Document current behavior
3. **Plan rollback** - Know how to undo changes
4. **Communicate** - Inform stakeholders of timeline

### During Migration

1. **Incremental changes** - Small, testable steps
2. **Run tests frequently** - After each change
3. **Document deviations** - Note any plan changes
4. **Create checkpoints** - Regular commits

### After Migration

1. **Full test suite** - Run all tests including integration
2. **Performance testing** - Compare before/after
3. **Documentation** - Update all affected docs
4. **Monitoring** - Watch for issues post-deployment

## Common Migration Types

| Type | Complexity | Example |
|------|------------|---------|
| Language version | Medium | Go 1.21 -> 1.25 |
| Framework upgrade | Medium-High | Echo v3 -> v4 |
| Database migration | High | PostgreSQL -> MySQL |
| Architecture change | Very High | Monolith -> Microservices |
| Dependency replacement | Low-Medium | One library to another |

## Related Use Cases

- [Refactoring](/use-cases/refactoring) - Incremental code improvements
- [Test Generation](/use-cases/test-generation) - Ensure coverage before migration
- [Multi-Agent Review](/use-cases/multi-agent-review) - Review migration changes

## Next Steps

- [Concepts: Contracts](/concepts/contracts) - Validate migration steps
- [Concepts: Pipelines](/concepts/pipelines) - Understand multi-step execution

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
