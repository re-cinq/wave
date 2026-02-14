# Security & Isolation Requirements Quality Checklist

**Feature**: Skill Dependency Installation in Pipeline Steps
**Branch**: `102-skill-deps-pipeline`
**Date**: 2026-02-14

This checklist validates that security and workspace isolation requirements are fully and clearly specified. Wave's security model is a constitutional constraint — gaps here are high-risk.

---

## Command Execution Security

- [ ] CHK-S01 - Are the `check`, `install`, and `init` commands executed via `sh -c` clearly documented, and are the implications for shell injection addressed? [Completeness]
- [ ] CHK-S02 - Does the spec define who authors skill definitions in `wave.yaml` and whether they are treated as trusted input (no sanitization) or untrusted input (must be sanitized)? [Clarity]
- [ ] CHK-S03 - Are there requirements for sanitizing or validating the `commands_glob` pattern to prevent path traversal (e.g., `../../../etc/passwd`)? [Completeness]
- [ ] CHK-S04 - Does the spec address whether install commands inherit the pipeline's environment variables, and if so, whether the curated env passthrough (P9) applies? [Completeness]
- [ ] CHK-S05 - Are there requirements for logging or auditing which skill commands are executed during preflight (aligned with Wave's audit logging principle)? [Coverage]

## Workspace Isolation

- [ ] CHK-S06 - Does FR-012 (independent skill commands per step) define what "independent" means — are files copied or symlinked, and does this matter for workspace isolation? [Clarity]
- [ ] CHK-S07 - Is there a requirement that skill command provisioning does not modify files outside the step workspace directory? [Completeness]
- [ ] CHK-S08 - Does the spec address whether the `repoRoot` path used for glob resolution is validated against the allowed directory list from the security model? [Completeness]
- [ ] CHK-S09 - Are there requirements for cleanup of the staging directory (`.wave-skill-commands/`) after the adapter copies files? [Completeness]

## Failure Mode Security

- [ ] CHK-S10 - Does the spec define whether a failed preflight phase leaves any side effects (partially installed skills, partial file copies) that could affect subsequent runs? [Completeness]
- [ ] CHK-S11 - Are error messages from failed install/init commands required to be scrubbed for credentials before being included in preflight results or events? [Coverage]
- [ ] CHK-S12 - Does the spec address the case where an install command succeeds but installs something unexpected (malicious skill binary)? Is trust model for install commands documented? [Clarity]
