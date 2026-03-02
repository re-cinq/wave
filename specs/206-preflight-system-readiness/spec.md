# feat(preflight): extend system readiness checks for wave run wave pre-flight

**Issue**: [re-cinq/wave#206](https://github.com/re-cinq/wave/issues/206)
**Parent**: #184
**Labels**: enhancement, pipeline, priority: high
**Author**: nextlevelshit

## Summary

Extend the existing `internal/preflight/` and `internal/onboarding/` systems to serve as the system readiness gate for `wave run wave`. This includes verifying adapter health (binary reachability + authentication), CLI availability for all configured forge types, skill dependencies, and Wave initialization status with last-update reporting. The existing preflight checks validate per-pipeline dependencies; this issue extends them to validate the full system state required for interactive orchestration mode.

## Acceptance Criteria

- [ ] `wave run wave` pre-flight validates adapter binary is reachable and authenticated (not just present)
- [ ] Pre-flight checks verify CLI availability for all configured forge types (gh, glab, tea, bb) based on detected forge
- [ ] Skill dependency validation reports missing skills with actionable install guidance
- [ ] Wave initialization status is checked with last-update date displayed
- [ ] Pre-flight results are emitted as a structured artifact consumable by downstream pipeline steps
- [ ] Clear pass/fail reporting — each check has a status, message, and remediation hint
- [ ] Extends existing `internal/preflight/preflight.go` rather than creating a parallel system
- [ ] All existing preflight tests continue to pass (`go test ./internal/preflight/...`)

## Dependencies

None — this is foundational work that other sub-issues depend on.

## Scope Notes

- **In scope**: Extending existing preflight and onboarding systems, adapter health validation, forge CLI checks, skill dependency validation, structured artifact output
- **Out of scope**: Auto-installation of missing dependencies on the fly (tracked separately in #97), changes to the onboarding wizard UX flow, CI/CD pipeline preflight (#173 covers that independently)
- **Overlap note**: #173 covers CI/CD preflight and onboarding guidance broadly; this issue is specifically about the `wave run wave` runtime readiness gate
