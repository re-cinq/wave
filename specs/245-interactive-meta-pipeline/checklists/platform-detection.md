# Platform Detection Requirements Quality: 245-interactive-meta-pipeline

**Domain**: Platform Detection & Routing (FR-005, FR-014, US3)
**Generated**: 2026-03-04

---

## Completeness

- [ ] CHK301 - Does FR-005 specify the URL pattern matching rules for each platform (SSH, HTTPS, custom domains), or just the expected outcome? [Completeness]
- [ ] CHK302 - Are self-hosted instance detection requirements specified — how does the system detect a self-hosted GitLab vs. a self-hosted Gitea? [Completeness]
- [ ] CHK303 - Does US3 scenario 3 (multiple remotes) specify the criteria for determining the "primary" remote beyond just "origin"? [Completeness]
- [ ] CHK304 - Does FR-014 specify the fallback behavior when a detected platform has no platform-specific pipeline variants available in the manifest? [Completeness]

## Clarity

- [ ] CHK305 - Is the "pipeline family" concept (gh/gl/bb/gt) clearly defined as a naming convention, a manifest field, or a runtime derivation? [Clarity]
- [ ] CHK306 - Does US3 scenario 4 clearly distinguish between "self-hosted platform" (detectable) and "unrecognized platform" (unknown)? [Clarity]

## Consistency

- [ ] CHK307 - Are the four platforms listed consistently across all artifacts — spec mentions GitHub/GitLab/Bitbucket/Gitea, do the plan and tasks match? [Consistency]
- [ ] CHK308 - Does the `PlatformProfile.CLITool` field in the data model match the actual CLI tool names referenced in the spec (gh, glab, etc.)? [Consistency]

## Coverage

- [ ] CHK309 - Does the spec address platform detection for repositories with remotes using non-standard SSH port configurations? [Coverage]
- [ ] CHK310 - Are requirements defined for repositories using SSH config aliases (e.g., `Host ghub` aliasing `github.com`)? [Coverage]
- [ ] CHK311 - Does the spec address GitHub Enterprise (non-github.com domains) vs. github.com detection? [Coverage]

---

## Summary

| Dimension | Items |
|-----------|-------|
| Completeness | CHK301–CHK304 (4) |
| Clarity | CHK305–CHK306 (2) |
| Consistency | CHK307–CHK308 (2) |
| Coverage | CHK309–CHK311 (3) |
| **Total** | **11** |
