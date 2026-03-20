# audit: partial — README pipeline count stale (#41)

**Issue**: [#503](https://github.com/re-cinq/wave/issues/503)
**Labels**: audit
**Author**: nextlevelshit
**Source**: #41 — docs: documentation consistency report

## Problem

README.md references "46 built-in pipelines" in two locations but the actual count of `.wave/pipelines/*.yaml` files is 51.

## Evidence

- Line 382: `**46 built-in pipelines** for development, debugging, documentation, and GitHub automation.`
- Line 410: `A selection of the 46 built-in pipelines:`

## Acceptance Criteria

- [ ] Both occurrences of "46" pipeline count in README.md updated to reflect actual count (51)
