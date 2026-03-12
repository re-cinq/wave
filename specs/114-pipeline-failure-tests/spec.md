# Add Integration Tests Covering Pipeline Failure Modes and False-Positive Detection

**Issue**: [#114](https://github.com/re-cinq/wave/issues/114)
**Labels**: enhancement, ci, pipeline
**Author**: nextlevelshit
**State**: OPEN

## Problem

Pipelines currently lack sufficient test coverage for failure scenarios, creating a risk of false-positives — where a pipeline reports success despite producing incorrect results or encountering errors during execution.

## Goal

Achieve full introspectability of pipeline execution outcomes. A pipeline must **never** report success when:
- A step failed or produced unexpected output
- Contract validation failed
- An artifact was missing or malformed
- A permission denial occurred
- A workspace error was encountered

## Unhappy Cases to Cover

The following failure scenarios must be tested with both unit and integration tests:

- [ ] **Contract schema mismatch** — step output does not match the expected JSON schema
- [ ] **Step timeout** — a step exceeds its configured timeout and the pipeline aborts cleanly
- [ ] **Missing artifact** — a downstream step requires an artifact that was not produced
- [ ] **Permission denial** — a step attempts a disallowed tool call and is correctly rejected
- [ ] **Workspace corruption** — the workspace directory is in an invalid state mid-run
- [ ] **Non-zero adapter exit code** — the underlying CLI (e.g. Claude Code) exits with an error
- [ ] **Contract validation returns false-positive** — validate that the validator does not pass malformed output

## Affected Pipelines

At minimum, the following pipelines should have integration test coverage:

- `gh-issue-rewrite`
- `doc-sync`
- `dead-code`
- `speckit-flow`
- `gh-issue-implement`

## Acceptance Criteria

- [ ] All listed unhappy cases have corresponding tests in `tests/`
- [ ] Pipeline execution returns a non-zero exit code when contract validation fails
- [ ] Pipeline execution returns a non-zero exit code when any step exits with an error
- [ ] No existing passing tests are broken
- [ ] Tests run cleanly under `go test -race ./...`
- [ ] "Real pipeline runs" are defined as CI integration tests (not manual verification)

## References

- Existing test structure: `tests/` directory
- Contract validation: `internal/contract/`
- Pipeline execution: `internal/pipeline/`
