# Implementation Plan: Architecture Audit & Layered Transition

## Objective

Create a comprehensive architecture audit document (`docs/architecture-audit.md`) capturing Wave's current package structure, dependency graph, design patterns, and structural concerns. Update ADR-003 with any new findings and ensure the transition plan is actionable. Add depguard linter configuration to enforce layer boundaries via CI.

## Approach

This is primarily a **documentation and tooling task** — no production code is modified. The work produces three deliverables:

1. **Architecture audit document** — standalone markdown capturing current state
2. **ADR-003 updates** — incorporate audit findings, verify accuracy of layer classifications and violation inventory
3. **Depguard configuration** — `.golangci.yml` rules enforcing layer boundaries (as specified in ADR-003's implementation plan)

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `docs/architecture-audit.md` | create | Comprehensive audit of current architecture |
| `docs/adr/003-layered-architecture.md` | modify | Update status, refine violations based on audit |
| `docs/adr/README.md` | modify | Update index if ADR-003 status changes |
| `.golangci.yml` | modify | Add depguard rules for layer boundary enforcement |

## Architecture Decisions

1. **Audit as standalone document, not an ADR**: The audit is a reference document describing current state, not a decision record. It lives at `docs/architecture-audit.md` and ADR-003 references it.

2. **Depguard over go-cleanarch**: ADR-003 already recommends depguard via golangci-lint. This integrates into the existing CI pipeline with no new tooling.

3. **No code restructuring**: Per ADR-003's decision, the layer model is an overlay on the existing package structure. No packages are moved or renamed.

4. **Known violations are allow-listed**: The three documented violations (`doctor→onboarding`, `webui→adapter/workspace`, `defaults→pipeline`) are tracked but not fixed in this PR — they get separate remediation issues.

## Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| Depguard rules too strict, breaking CI | Medium | Start with the most critical boundaries only; allow-list known violations |
| Audit becomes stale quickly | Low | Link audit to CI — depguard enforces boundaries automatically |
| ADR-003 layer classifications debatable | Low | Document rationale for edge cases (e.g., `defaults`, `manifest`) |

## Testing Strategy

- `golangci-lint run ./...` must pass with new depguard rules
- `go test ./...` unaffected (no production code changes)
- Manual review of audit document for accuracy against actual import graph
- Verify depguard catches intentional layer violations (add a test import, confirm lint failure, revert)
