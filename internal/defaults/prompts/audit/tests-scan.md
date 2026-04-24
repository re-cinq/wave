## Objective

Perform a test quality audit of the implementation, analyzing test coverage, test quality,
missing tests, untested code paths, and gaps in error case coverage. Your job is to verify
that the test suite adequately validates the implementation and will catch regressions.
This is the reconnaissance phase — breadth over depth.

## Context

This is the first step of a two-step test audit pipeline. You have read-only access to
the entire project. Your output feeds into an aggregation step that merges findings across
all audit dimensions. The more precise your file path and line number references are, the
more efficiently downstream steps can work.

## Requirements

1. **Identify untested code**: For each new or modified source file, check whether
   corresponding test files exist. For each exported function and method, check whether
   at least one test exercises it. Flag:
   - Functions with zero test coverage
   - Methods that are only tested through integration tests, not unit tests
   - Code paths reachable only through error conditions that have no test

2. **Assess test quality**: For existing tests, evaluate:
   - **Happy-path only**: Tests that only exercise the success path without any error
     cases. Flag functions that handle errors but whose tests never trigger those paths.
   - **Assertion strength**: Tests that check only for "no error" without validating
     the actual output. Weak assertions miss behavioral regressions.
   - **Table-driven coverage**: For functions with multiple input variations, are
     table-driven tests used? Are edge cases represented in the test table?
   - **Test isolation**: Do tests depend on external state (files, network, databases)
     without cleanup? Do tests depend on execution order?

3. **Check edge case coverage**: For each function, consider:
   - Nil/zero-value inputs
   - Empty collections (slices, maps)
   - Boundary values (max int, empty string, single element)
   - Concurrent access patterns (if applicable)
   - Context cancellation and timeout handling
   - Malformed or invalid inputs

4. **Verify test naming and organization**: Do tests follow the project's conventions?
   - Test function names that describe the scenario (TestXxx_WhenYyy_ReturnsZzz)
   - Subtest names in table-driven tests that are descriptive
   - Test helpers that are reusable and well-documented

5. **Check for test anti-patterns**:
   - Tests that test implementation details (asserting on internal state)
   - Excessive mocking that makes tests pass regardless of behavior
   - Tests that sleep for arbitrary durations instead of using synchronization
   - Tests with commented-out assertions or TODO markers

## Constraints and Anti-patterns

- Do NOT report code quality or security issues. This is a test audit only.
- Do NOT flag test files for style or formatting issues.
- Do NOT modify any files. This is a read-only scan.
- Do NOT flag pre-existing test gaps unless the implementation made them worse.

## Output Format

Write your findings to the output artifact path matching the contract schema. Each finding
must include: type="test", severity (critical/high/medium/low/info), the affected file path
and line range, a description of the test gap, evidence from the code, and a recommendation
(fix/investigate).

## Quality Bar

A passing test scan identifies all significant test gaps — untested functions, happy-path-only
tests, and missing edge cases. Missing an obvious untested code path or flagging well-tested
code as untested constitutes a failure.
