# feat(pipeline): codebase health analysis step — forge-aware repository state artifact

**Issue**: [#207](https://github.com/re-cinq/wave/issues/207)
**Parent**: #184
**Labels**: enhancement, pipeline, priority: high
**Author**: nextlevelshit
**State**: OPEN

## Summary

Implement a codebase health analysis pipeline step that produces a structured artifact describing the current state of a repository. This step performs forge-aware analysis (GitHub/Bitbucket/GitLab/Gitea) covering recent commit history, open PR status (including review state, comments, triage), open issues with actionable categorization, and code health metrics. The output artifact follows the existing `inject_artifacts` pattern and is the primary input for the pipeline proposal engine.

## Acceptance Criteria

- [ ] Health analysis step produces a structured JSON artifact with well-defined schema
- [ ] Artifact includes recent commit history analysis (frequency, authors, areas of activity)
- [ ] Artifact includes open PR status: count, review state, comment activity, staleness
- [ ] Artifact includes open issue summary: count, categories, actionable items, priorities
- [ ] Artifact includes basic code health signals (test pass rate if available, recent CI status)
- [ ] Analysis is forge-aware — uses correct API (gh, glab, tea, or Bitbucket REST) based on detected forge type
- [ ] GitHub (`gh`) forge path is fully implemented as the primary target
- [ ] Other forge paths (bb, gl, gt) have stub implementations with clear TODO markers
- [ ] Artifact schema is documented and validated via contract
- [ ] Step integrates with the existing artifact injection system (`inject_artifacts`)

## Dependencies

- #206 — System readiness checks (forge CLI must be validated before health analysis runs)

## Scope Notes

- **In scope**: Forge-aware API calls to gather repository state, structured artifact output, contract schema for the artifact, GitHub as primary implementation target
- **Out of scope**: Deep static analysis of source code (e.g., complexity metrics, dependency audits), performance profiling, security scanning — these are separate pipeline concerns
- **Design note**: The artifact must be machine-parseable (JSON) so the proposal engine can consume it programmatically; human-readable markdown summaries are optional additions
