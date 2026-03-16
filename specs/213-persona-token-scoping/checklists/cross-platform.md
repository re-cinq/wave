# Cross-Platform Requirements Quality: Persona Token Scoping

**Feature**: #213 — Persona Token Scoping
**Date**: 2026-03-16

## Platform Parity

- [ ] CHK037 - Are the scope mapping tables complete for all 5 canonical resources × 3 permission levels × 3 supported forges? [Platform Parity]
- [ ] CHK038 - Does the spec account for GitHub classic PATs vs. fine-grained PATs having fundamentally different scope models? [Platform Parity]
- [ ] CHK039 - Is the Gitea introspection approach specified with the same level of detail as GitHub and GitLab? [Platform Parity]
- [ ] CHK040 - Are requirements defined for self-hosted forge instances that may have non-standard API endpoints? [Platform Parity]

## Graceful Degradation

- [ ] CHK041 - Is the degradation hierarchy clearly specified: full validation → partial validation → skip with warning → skip silently? [Graceful Degradation]
- [ ] CHK042 - Does the spec define the warning message format for each degradation scenario (unknown forge, introspection failure, missing CLI tool)? [Graceful Degradation]
- [ ] CHK043 - Are the conditions for each degradation level exhaustively enumerated, or could edge cases fall through without any behavior defined? [Graceful Degradation]
- [ ] CHK044 - Does the spec address network timeout behavior during token introspection (how long to wait, what to report)? [Graceful Degradation]
