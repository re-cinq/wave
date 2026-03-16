# Source Abstraction Quality Checklist

**Feature**: #201 — Continuous Pipeline Execution
**Domain**: WorkItemSource interface and implementations
**Generated**: 2026-03-16

## Completeness

- [ ] CHK101 - Does the `WorkItemSource` interface define a `Close()` or cleanup method for sources that hold resources (e.g., file handles)? [Completeness]
- [ ] CHK102 - Are the supported filter keys for `github:` source (`label`, `state`, `sort`, `direction`, `limit`) all documented with their default values? [Completeness]
- [ ] CHK103 - Does the spec define how multiple labels are handled in `github:label=bug,enhancement` — is this "AND" or "OR" filtering? [Completeness]
- [ ] CHK104 - Is the file source line format specified? Are leading/trailing whitespace, empty lines, or comment lines (`#`) handled? [Completeness]
- [ ] CHK105 - Does the spec address what happens when the `file:` path is relative vs absolute? Is it resolved against CWD or project root? [Completeness]

## Clarity

- [ ] CHK106 - Is the one-shot fetch model for `GitHubSource` (fetch all, iterate locally) clearly stated as a design decision with rationale? [Clarity]
- [ ] CHK107 - Is it clear that `file:queue.txt` is read-only (items loaded at start) vs. the spec's mention of "reads and removes the first line"? The spec says "removes" but the plan says "load all lines". [Clarity]
- [ ] CHK108 - Is the `WorkItem.ID` derivation strategy (issue number for GitHub, content hash for file) specified clearly enough for implementers? [Clarity]

## Consistency

- [ ] CHK109 - Does the source URI scheme (`github:`, `file:`) conflict with or complement Wave's existing forge template variables (`{{ forge.cli_tool }}`)? [Consistency]
- [ ] CHK110 - Is the `GitHubSource`'s use of `gh issue list` consistent with Wave's existing `gh` CLI usage patterns in `internal/github/`? [Consistency]
- [ ] CHK111 - Does the plan's `Limit: 100` default for GitHub source match the spec's clarification C2 which also says 100? [Consistency]

## Coverage

- [ ] CHK112 - Are extensibility requirements for future forge providers (`gitlab:`, `bitbucket:`) captured as a design goal, even if not implemented in v1? [Coverage]
- [ ] CHK113 - Is there a scenario covering `github:` source when the `gh` CLI is not installed or not authenticated? [Coverage]
- [ ] CHK114 - Do the requirements specify behavior when `github:` returns issues that include PRs (GitHub's API treats PRs as issues)? [Coverage]
