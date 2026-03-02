# feat(pipeline): auto-detect forge type and propose correct pipeline family

**Issue**: [#211](https://github.com/re-cinq/wave/issues/211)
**Parent**: [#184](https://github.com/re-cinq/wave/issues/184)
**Labels**: enhancement, pipeline, priority: high
**Author**: nextlevelshit
**State**: OPEN

## Summary

Implement automatic forge type detection and pipeline family selection for `wave run wave`. When the orchestration mode starts, it should detect whether the repository uses GitHub, Bitbucket, GitLab, or Gitea based on git remote URLs and available configuration, then automatically filter and propose pipelines from the correct family (`gh-*`, `bb-*`, `gl-*`, `gt-*`). The 43+ existing pipelines across 4 forge backends should be transparently matched to the repository's forge without manual user configuration.

## Acceptance Criteria

- [ ] Forge type is auto-detected from git remote URLs (github.com → GitHub, bitbucket.org → Bitbucket, gitlab.com → GitLab, gitea instances → Gitea)
- [ ] Detection supports self-hosted instances (e.g., GitHub Enterprise, self-hosted GitLab) via configurable domain mapping in `wave.yaml`
- [ ] Detected forge type is available as a structured value for downstream consumers (health analysis, proposal engine)
- [ ] Pipeline catalog is filtered to the correct family — only `gh-*` pipelines are proposed for GitHub repos, etc.
- [ ] Multi-forge repositories (e.g., mirrored repos) are handled — user is prompted to select primary forge if ambiguous
- [ ] Forge detection result is part of the system readiness artifact (complements #206)
- [ ] Detection works without requiring forge CLI to be installed (falls back to git remote parsing)
- [ ] Existing forge-specific pipelines (43+ across gh/bb/gl/gt) work unchanged — no pipeline modifications needed

## Dependencies

- #206 — System readiness checks (forge detection integrates into the pre-flight phase)

## Scope Notes

- **In scope**: Git remote URL parsing, forge type classification, configurable domain mapping for self-hosted instances, pipeline family filtering, multi-forge disambiguation
- **Out of scope**: Creating new forge-specific pipelines (they already exist), modifying existing pipeline definitions, implementing new forge API integrations (handled per-forge in #207)
- **Design note**: Forge detection should be a lightweight, early-phase operation that runs before health analysis — it determines which forge CLI to validate and which API to call for health data
- **Implementation hint**: Could live in `internal/manifest/` (configuration-adjacent) or a new `internal/forge/` package; should be a small, focused module
