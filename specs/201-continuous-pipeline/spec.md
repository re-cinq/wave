# feat: support continuous/long-running pipeline execution with automatic issue iteration

**Issue**: [#201](https://github.com/re-cinq/wave/issues/201)
**Labels**: `enhancement`, `pipeline`
**Author**: nextlevelshit

## Summary

Enable pipelines to run continuously, automatically picking up and processing new issues one after another without manual re-invocation.

## Problem Statement

Currently, pipelines execute once and exit. To process multiple issues (e.g., with `gh-implement`), the user must manually re-run the pipeline for each issue. This is tedious for batch processing workflows.

## Proposed Behavior

- A pipeline can be invoked in **continuous mode** (e.g., `wave run --continuous gh-implement`)
- In continuous mode, after completing one iteration the pipeline:
  1. Polls for the next available issue (using configured filters/labels)
  2. Creates a fresh workspace and executes the pipeline for that issue
  3. Repeats until no more matching issues exist or the user interrupts (Ctrl+C)
- Each iteration is fully isolated (fresh memory, fresh workspace) per Wave's existing contract
- Graceful shutdown on interrupt — completes current step, then exits

## Acceptance Criteria

- [ ] Pipeline manifest supports a `continuous` or `loop` execution mode
- [ ] `wave run --continuous <pipeline>` iterates over matching issues automatically
- [ ] Each iteration gets a clean workspace and fresh agent memory
- [ ] Graceful shutdown on SIGINT/SIGTERM (finish current step, then stop)
- [ ] Progress events emitted for each iteration (issue number, status)
- [ ] Pipeline exits cleanly when no more matching issues are found
- [ ] Documentation updated with continuous mode usage

## Technical Considerations

- How does the pipeline determine which issue to process next? (label filter, oldest first, priority?)
- Should there be a configurable delay between iterations?
- How to handle failures — skip and continue, or halt?
- Rate limiting considerations for GitHub API calls
- State tracking to avoid re-processing already-completed issues
