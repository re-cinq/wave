## PR-scope guard (mandatory first check)

Before scanning, check whether `.agents/artifacts/pr-context` exists. If it does, read
its `changed_files` (JSON array) and `diff_path` fields. From this point on:

- ONLY flag issues whose file path is in `changed_files`. Do not walk the whole
  repository — scan exclusively the listed files.
- Treat `diff_path` (the unified diff blob on disk) as the authoritative scope. If a
  finding is not visible as a hunk in that diff, it is out of scope.
- Findings on files outside `changed_files` will be dropped by the downstream
  `filter-scope` step regardless. Producing them wastes tokens.

If `.agents/artifacts/pr-context` does NOT exist, this is a standalone audit run —
proceed with the whole-repo scan described below.

## Objective

Perform an architecture audit of the implementation, analyzing package structure, import
graph, coupling issues, misplaced packages, and pattern violations. Your job is to verify
that the implementation follows the project's established architectural conventions and
does not introduce structural problems. This is the reconnaissance phase — breadth over
depth.

## Context

This is the first step of a two-step architecture audit pipeline. You have read-only
access to the entire project. Your output feeds into an aggregation step that merges
findings across all audit dimensions. The more precise your file path and line number
references are, the more efficiently downstream steps can work.

## Requirements

1. **Analyze package placement**: Check whether new code is placed in the correct package
   according to the project's established structure. Flag:
   - Business logic in transport/handler packages
   - Database queries outside repository/state packages
   - Configuration logic scattered across packages instead of centralized
   - Utility code that belongs in a shared package but is duplicated locally

2. **Check import graph health**: Examine import patterns for:
   - Circular dependencies (direct or transitive)
   - Internal packages importing from higher-level packages (dependency inversion violation)
   - Unnecessary coupling between unrelated packages
   - God packages that import everything
   - Test-only imports leaking into production code

3. **Verify pattern consistency**: Does the implementation follow established patterns?
   - Interface usage for dependency injection where the project uses DI
   - Error handling patterns matching existing conventions
   - Constructor patterns (NewXxx functions) matching project style
   - Configuration patterns (viper, env vars, flags) matching existing approach

4. **Assess coupling and cohesion**: For each new type, function, or module:
   - Does it have a single, clear responsibility?
   - Are its dependencies explicit and minimal?
   - Could it be tested in isolation?
   - Does it duplicate responsibility already owned by another package?

5. **Check API surface**: For new exported symbols:
   - Are they necessary? Could they be unexported?
   - Do they follow the project's naming conventions?
   - Are they placed in the right package for discoverability?
   - Do they have appropriate documentation?

## Constraints and Anti-patterns

- Do NOT report correctness or security issues. This is an architecture audit only.
- Do NOT flag patterns as wrong just because you would do it differently. The project's
  established patterns are the standard, not your preferences.
- Do NOT modify any files. This is a read-only scan.
- Do NOT flag test helper code for architectural purity — test packages have more flexibility.

## Output Format

Write your findings to the output artifact path matching the contract schema. Each finding
must include: type="architecture", severity (critical/high/medium/low/info), the affected
file path and line range, a description of the architectural issue, evidence from the code,
and a recommendation (refactor/investigate/fix).

## Quality Bar

A passing architecture scan identifies structural problems that will cause maintenance
headaches — misplaced code, tight coupling, pattern violations. Missing an obvious
circular dependency or a god package constitutes a failure.
