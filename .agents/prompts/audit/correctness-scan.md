## Objective

Perform a correctness audit of the implementation against the original issue requirements.
Your job is to verify that the implementation actually addresses what was requested, identify
logic errors, missing features, and incomplete implementations. This is the reconnaissance
phase — breadth over depth. The downstream aggregate step will merge your findings with
other audit dimensions.

## Context

This is the first step of a two-step correctness audit pipeline. The issue assessment
artifact from the implementation step describes what was requested. You have read-only
access to the entire project. Your output feeds into an aggregation step that merges
findings across all audit dimensions. The more precise your file path and line number
references are, the more efficiently downstream steps can work.

## Requirements

1. **Read the issue assessment**: Understand what was requested — acceptance criteria,
   functional requirements, and expected behavior. This is your ground truth for
   correctness evaluation.

2. **Verify feature completeness**: For each acceptance criterion and functional requirement,
   check whether the implementation addresses it. Flag requirements that are:
   - Completely missing (not implemented at all)
   - Partially implemented (some aspects addressed, others not)
   - Incorrectly implemented (code exists but does the wrong thing)

3. **Check for logic errors**: Examine the implementation for:
   - Off-by-one errors in loops and index calculations
   - Incorrect conditional logic (wrong operator, inverted conditions)
   - Missing nil/zero-value checks that could cause panics
   - Race conditions in concurrent code paths
   - Incorrect error propagation (swallowed errors, wrong error types)
   - Edge cases not handled (empty inputs, boundary values)

4. **Verify data flow correctness**: Trace key data paths through the implementation:
   - Are inputs validated before use?
   - Are transformations correct (type conversions, format changes)?
   - Are outputs in the expected format and structure?
   - Are side effects (file writes, API calls) happening at the right time?

5. **Check test alignment**: Do the tests actually test what the requirements specify?
   Flag tests that:
   - Test implementation details rather than behavior
   - Use mocked data that doesn't reflect real-world inputs
   - Only cover the happy path without error cases
   - Assert on wrong values or conditions

## Constraints and Anti-patterns

- Do NOT report style or formatting issues. This is a correctness audit only.
- Do NOT flag theoretical issues without citing an actual file path and the specific
  code pattern that is incorrect.
- Do NOT modify any files. This is a read-only scan.
- Do NOT flag test files as incorrect unless the test expectations contradict the
  documented requirements.

## Output Format

Write your findings to the output artifact path matching the contract schema. Each finding
must include: type="correctness", severity (critical/high/medium/low/info), the affected
file path and line range, a description of the correctness issue, evidence from the code,
and a recommendation (fix/investigate).

## Quality Bar

A passing correctness scan identifies all requirements that are unaddressed or incorrectly
implemented, with precise file references. Missing an obvious logic error or unimplemented
requirement constitutes a failure.
