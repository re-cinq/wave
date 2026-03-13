# Implementation Plan: ADR-003 Layered Architecture

## Objective

Write ADR-003 (`docs/adr/003-layered-architecture.md`) documenting formal layer boundaries for Wave's 25 internal packages, establishing dependency rules between layers, and defining a migration strategy to enforce those boundaries. This complements ADR-002 (Extract StepExecutor) and delivers on #298's architecture audit.

## Approach

This is a documentation-only change. The deliverable is a single ADR file following the existing template (`docs/adr/000-template.md`), plus an update to the ADR README index. No code changes are required.

The ADR will be grounded in the actual dependency graph observed in the codebase (not hypothetical), classifying each of the 25 `internal/` packages into one of four layers and documenting the actual import relationships between them.

### Verified Dependency Graph (from codebase analysis)

| Package | Imports |
|---------|---------|
| `display` | `event`, `pathfmt`, `deliverable` |
| `tui` | `display`, `event`, `forge`, `github`, `manifest`, `pathfmt`, `pipeline`, `state` |
| `webui` | `adapter`, `audit`, `display`, `event`, `manifest`, `pipeline`, `state`, `workspace` |
| `pipeline` | `adapter`, `audit`, `contract`, `deliverable`, `event`, `manifest`, `preflight`, `relay`, `security`, `skill`, `state`, `workspace`, `worktree` |
| `adapter` | `github` |
| `contract` | `pathfmt` |
| `state` | (none) |
| `workspace` | (none) |
| `worktree` | (none) |
| `manifest` | (none) |
| `security` | (none) |
| `audit` | (none) |
| `event` | (none) |
| `defaults` | `manifest`, `pipeline` |
| `relay` | (none) |
| `deliverable` | `pathfmt` |
| `pathfmt` | (none) |
| `onboarding` | `manifest`, `tui` |
| `preflight` | `skill` |
| `recovery` | `contract`, `pathfmt`, `preflight`, `security` |
| `skill` | (none) |
| `doctor` | `forge`, `github`, `manifest`, `onboarding`, `pipeline` |
| `forge` | (none) |
| `suggest` | `doctor`, `forge` |
| `github` | (none) |

### Proposed Layer Classification

1. **Presentation** — `display`, `tui`, `webui`, `onboarding`
2. **Domain/Orchestration** — `pipeline`, `adapter`, `contract`, `relay`, `deliverable`, `preflight`, `recovery`, `skill`, `defaults`, `suggest`, `doctor`
3. **Infrastructure** — `state`, `workspace`, `worktree`, `forge`, `github`
4. **Cross-cutting** — `manifest`, `security`, `audit`, `event`, `pathfmt`

### Key Violations to Document

1. **`webui` → `adapter`, `workspace`**: Presentation layer importing infrastructure/domain directly
2. **`onboarding` → `tui`**: Cross-layer coupling within presentation (minor — both are presentation)
3. **`defaults` → `pipeline`**: Infrastructure-like package importing domain (circular risk)
4. **`doctor` → `onboarding`**: Domain package importing presentation

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `docs/adr/003-layered-architecture.md` | create | The ADR document |
| `docs/adr/README.md` | modify | Add ADR-003 to the index table |

## Architecture Decisions

- **Four-layer model** (Presentation / Domain / Infrastructure / Cross-cutting) rather than three-layer (Presentation / Business / Data) because Wave's cross-cutting concerns (manifest, security, audit, event, pathfmt) are genuinely shared and don't fit neatly into any single layer.
- **Options Considered** will include: (1) Convention-only with documentation, (2) Go build constraints / `internal/` sub-packages, (3) CI linting with depguard/go-cleanarch, (4) Go Modules per layer. The recommendation will be Option 3 (CI linting) as it provides automated enforcement without restructuring the package tree.
- **ADR-002 relationship**: This ADR defines the inter-package rules; ADR-002 addresses intra-package decomposition within `pipeline/`. They are complementary.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Layer classification disagreement | Medium | Low | ADR is "Proposed" status — open for review |
| Stale dependency data | Low | Low | Based on actual codebase analysis at time of writing |
| Over-prescription of rules | Medium | Medium | Focus on documenting current state and recommended direction, not mandating immediate restructuring |

## Testing Strategy

No code changes, so no unit/integration tests needed. Validation:
- ADR follows the template structure from `000-template.md`
- All 25 packages are classified
- Dependency rules are consistent with the observed import graph
- ADR-002 and #298 are properly referenced
- README index is updated
