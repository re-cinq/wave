# ADR-003: Layered Architecture Separation

## Status
Accepted (Phase 1 — depguard rules live; package count drifted upward)

## Date
2026-03-13 (proposed) — 2026-04-26 (accepted)

## Implementation Status

Landed:
- `depguard` rules in `.golangci.yml` enforce two layer constraints:
  - `no-presentation-reverse` blocks `display`, `tui`, `webui`, `onboarding` from being imported by domain or infrastructure code.
  - `infrastructure-no-domain` blocks `pipeline`, `adapter`, `contract`, `relay`, `deliverable`, `preflight`, `recovery`, `skill`, `defaults`, `suggest`, `doctor` from being imported by infrastructure.
- Verified import constraints: `display` → `event`/`pathfmt`/`deliverable` only; `pipeline` does not import `display`/`tui`.

Drift since the original ADR was drafted:
- `internal/` package count grew from 25 to 39 (additions include `attention`, `bench`, `classify`, `continuous`, `cost`, `fileutil`, `hooks`, `humanize`, `ontology`, `retro`, `sandbox`, `scope`, `testutil`, `timeouts`, `tools`).
- `cmd/` layer has no depguard rule blocking `database/sql` imports — see ADR-007 for the consequences.
- New packages still need explicit layer assignments; the next iteration is ADR-007 enforcement plus a `cmd/`-layer rule.

## Context

Wave's `internal/` directory contains 25 packages that have grown organically during rapid prototyping (v0.15.0 → v0.84.1). While the separation between presentation and backend is already reasonably clean — `display/` imports only `event/`, `pathfmt/`, and `deliverable/`; `pipeline/` does not import `display/` or `tui/` — some areas have accumulated cross-layer coupling that warrants formal documentation and enforcement.

### Current State

An audit of the codebase reveals the following import relationships among internal packages:

| Package | Internal Imports |
|---------|-----------------|
| `adapter` | `github` |
| `audit` | (none) |
| `contract` | `pathfmt` |
| `defaults` | `manifest`, `pipeline` |
| `deliverable` | `pathfmt` |
| `display` | `deliverable`, `event`, `pathfmt` |
| `doctor` | `forge`, `github`, `manifest`, `onboarding`, `pipeline` |
| `event` | (none) |
| `forge` | (none) |
| `github` | (none) |
| `manifest` | `skill` |
| `onboarding` | `manifest`, `skill`, `tui` |
| `pathfmt` | (none) |
| `pipeline` | `adapter`, `audit`, `contract`, `deliverable`, `event`, `forge`, `manifest`, `preflight`, `recovery`, `relay`, `security`, `skill`, `state`, `workspace`, `worktree` |
| `preflight` | `skill` |
| `recovery` | `contract`, `pathfmt`, `preflight`, `security` |
| `relay` | (none) |
| `security` | (none) |
| `skill` | (none) |
| `state` | (none) |
| `suggest` | `doctor`, `forge` |
| `tui` | `display`, `event`, `forge`, `github`, `manifest`, `pathfmt`, `pipeline`, `state` |
| `webui` | `adapter`, `audit`, `display`, `event`, `manifest`, `pipeline`, `state`, `workspace` (behind `//go:build webui` tag) |
| `workspace` | (none) |
| `worktree` | (none) |

The presentation/backend separation is mostly intact — the event system properly decouples display consumers from pipeline producers — but several violations exist:

- **`webui` → `adapter`, `workspace`**: Presentation layer directly importing domain and infrastructure packages
- **`doctor` → `onboarding`**: Domain package importing a presentation package
- **`defaults` → `pipeline`**: Cross-cutting package importing domain, creating a circular-risk dependency

The `pipeline/executor.go` god-object (3,104 lines, 11+ responsibilities) is the primary structural concern, but that is addressed separately by [ADR-002](002-extract-step-executor.md). This ADR focuses on the inter-package layer boundaries, not intra-package decomposition.

This work builds on the [architecture audit](../architecture-audit.md) from [#298](https://github.com/re-cinq/wave/issues/298).

## Decision

Adopt a four-layer architectural model for Wave's internal packages and enforce dependency rules between layers using CI-integrated linting.

### Layer Definitions

**1. Presentation** — User-facing interfaces (terminal, web, onboarding flows).

| Package | Responsibility |
|---------|---------------|
| `display` | Terminal progress display and formatting |
| `tui` | Bubble Tea terminal UI |
| `webui` | Web operations dashboard |
| `onboarding` | Interactive `wave init` flow |

**2. Domain / Orchestration** — Core business logic for pipeline execution, validation, and coordination.

| Package | Responsibility |
|---------|---------------|
| `pipeline` | Pipeline execution, DAG traversal, step management |
| `adapter` | Subprocess execution and adapter management |
| `contract` | Output validation (JSON schema, TypeScript, test suites) |
| `relay` | Context compaction and summarization |
| `deliverable` | Pipeline deliverable tracking and output |
| `preflight` | Pipeline dependency validation and auto-install |
| `recovery` | Pipeline recovery hints and error guidance |
| `skill` | Skill discovery, provisioning, and command management |
| `defaults` | Embedded default personas, pipelines, and contracts |
| `suggest` | Pipeline suggestion engine |
| `doctor` | Project health checking and optimization |

**3. Infrastructure** — External system integrations and persistence.

| Package | Responsibility |
|---------|---------------|
| `state` | SQLite persistence and state management |
| `workspace` | Ephemeral workspace management |
| `worktree` | Git worktree lifecycle for isolated workspaces |
| `forge` | Git forge/hosting platform detection |
| `github` | GitHub API integration |

**4. Cross-cutting** — Shared utilities and concerns used across all layers.

| Package | Responsibility |
|---------|---------------|
| `manifest` | Configuration loading and validation |
| `security` | Security validation and sanitization |
| `audit` | Audit logging and credential scrubbing |
| `event` | Progress event emission and monitoring |
| `pathfmt` | Path formatting and normalization utilities |

### Dependency Rules

```
Presentation → Domain → Infrastructure
     ↓             ↓           ↓
     └─────── Cross-cutting ───┘
```

1. **Presentation** may import from **Domain**, **Infrastructure**, and **Cross-cutting**
2. **Domain** may import from **Infrastructure** and **Cross-cutting**
3. **Infrastructure** may import from **Cross-cutting** only
4. **Cross-cutting** must not import from any other layer
5. **No layer may import from Presentation** except Presentation itself
6. **No reverse dependencies** — Domain must not import Presentation; Infrastructure must not import Domain or Presentation

### Current Violations

| Violation | Packages | Severity | Remediation |
|-----------|----------|----------|-------------|
| Presentation → Domain (direct adapter/workspace use) | `webui` → `adapter`, `workspace` | Medium | Refactor `webui` to use domain interfaces rather than concrete infrastructure types |
| Domain → Presentation | `doctor` → `onboarding` | Medium | Extract the shared logic into a domain-level interface or move `onboarding` interaction behind an interface |
| Cross-cutting → Domain | `defaults` → `pipeline` | Low | `defaults` embeds pipeline definitions — consider moving the pipeline type definitions it needs into `manifest` |
| Cross-cutting → Domain | `manifest` → `skill` | Low | `manifest` imports `skill` for skill configuration types — consider extracting shared types into `manifest` or a shared types package |

## Options Considered

### Option 1: Convention-Only Documentation

Document the layer model and dependency rules in this ADR and in `CLAUDE.md`. Rely on code review to enforce boundaries. No tooling changes.

**Pros:**
- Zero implementation effort beyond writing the ADR
- No CI pipeline changes or new dependencies
- Flexible — allows pragmatic violations during rapid prototyping
- Immediately actionable — contributors can reference the rules today

**Cons:**
- Enforcement depends entirely on reviewer diligence
- Violations accumulate silently over time
- AI personas generating code have no automated guardrails against cross-layer imports
- Historical evidence shows documentation-only rules are frequently ignored under deadline pressure

### Option 2: Go Build Constraints / `internal/` Sub-packages

Restructure `internal/` into sub-directories per layer (e.g., `internal/presentation/tui/`, `internal/domain/pipeline/`). Go's `internal/` visibility rules would enforce some boundaries at compile time.

**Pros:**
- Compile-time enforcement — violations are build errors
- Package paths make layer membership self-documenting
- Strongest possible enforcement mechanism

**Cons:**
- Requires renaming every package and updating every import path across the entire codebase
- Massive, risky refactoring with high merge-conflict potential
- Breaks all existing test setups, CI scripts, and documentation references
- Overly disruptive for the current prototype phase
- Go's `internal/` rules enforce visibility but not dependency direction — a sub-package can still import siblings

### Option 3: CI Linting with depguard / go-cleanarch (Recommended)

Add a CI linting step using [depguard](https://github.com/OpenPeeDeeP/depguard) (via golangci-lint) or [go-cleanarch](https://github.com/roblaszczak/go-clean-arch) to enforce layer dependency rules. Violations fail the CI build.

**Pros:**
- Automated enforcement without restructuring the package tree
- Incremental adoption — start with error-level rules for the most critical boundaries, warn on known violations
- Integrates into the existing `golangci-lint` CI step
- Configuration is declarative and version-controlled
- AI personas see lint failures just like human developers, providing a natural guardrail

**Cons:**
- Requires maintaining a linter configuration that maps packages to layers
- New packages must be classified in the config when added
- Known violations must be explicitly allowed until remediated
- depguard configuration can become complex for nuanced rules

### Option 4: Go Modules Per Layer

Split `internal/` into separate Go modules (e.g., `go.recinq.io/wave/presentation`, `go.recinq.io/wave/domain`). Module boundaries enforce dependency direction at the `go.mod` level.

**Pros:**
- Strongest enforcement — module boundaries are absolute
- Independent versioning and release cycles per layer
- Clear ownership boundaries for teams

**Cons:**
- Extreme restructuring effort — the most disruptive option
- Go multi-module repositories are complex to manage (replace directives, version coordination)
- Overkill for a single-team project in prototype phase
- Significantly increases build and CI complexity
- No community precedent for this pattern at Wave's scale

## Consequences

### Positive
- Layer boundaries are formally documented and enforceable via CI, preventing further coupling drift
- AI personas operating within Wave pipelines benefit from bounded contexts — each layer has a clear responsibility boundary that aligns with step isolation (fresh memory per step, artifact-based communication)
- New contributors can orient themselves by layer rather than memorizing 25 individual package relationships
- Security audit surface is clearer — infrastructure and cross-cutting packages can be reviewed independently from presentation logic
- Future decomposition (e.g., ADR-002's StepExecutor extraction) has a documented architectural context to reference

### Negative
- Linter configuration must be maintained as packages are added or reclassified
- Known violations (the three documented above) must be explicitly allowed until remediated, creating a "tech debt ledger" that requires attention
- Layer classification edge cases will generate discussion (e.g., is `defaults` truly domain or cross-cutting?)

### Neutral
- No code restructuring is required — the layer model describes the current package layout with minimal violations
- The four-layer model is a documentation overlay on the existing structure, not a mandate to reorganize
- Existing pipeline definitions, personas, contracts, and manifests are unaffected

## Implementation Notes

### Phase 1: Linter Configuration

Add depguard rules to the existing `.golangci.yml` configuration:

```yaml
linters-settings:
  depguard:
    rules:
      presentation-no-reverse:
        deny:
          - pkg: "github.com/recinq/wave/internal/display"
            desc: "Domain/Infrastructure must not import presentation packages"
          - pkg: "github.com/recinq/wave/internal/tui"
            desc: "Domain/Infrastructure must not import presentation packages"
          - pkg: "github.com/recinq/wave/internal/webui"
            desc: "Domain/Infrastructure must not import presentation packages"
          - pkg: "github.com/recinq/wave/internal/onboarding"
            desc: "Domain/Infrastructure must not import presentation packages"
        files:
          - "internal/pipeline/**"
          - "internal/adapter/**"
          - "internal/contract/**"
          - "internal/state/**"
          - "internal/workspace/**"
          - "internal/worktree/**"
      infrastructure-no-domain:
        deny:
          - pkg: "github.com/recinq/wave/internal/pipeline"
            desc: "Infrastructure must not import domain packages"
          - pkg: "github.com/recinq/wave/internal/adapter"
            desc: "Infrastructure must not import domain packages"
        files:
          - "internal/state/**"
          - "internal/workspace/**"
          - "internal/worktree/**"
          - "internal/forge/**"
          - "internal/github/**"
```

### Phase 2: Violation Remediation

Address the three documented violations in priority order:

1. **`doctor` → `onboarding`** — Extract the shared interaction pattern into a domain-level interface
2. **`webui` → `adapter`, `workspace`** — Introduce domain-level interfaces that `webui` depends on instead of concrete types
3. **`defaults` → `pipeline`** — Evaluate whether the pipeline type definitions can be moved to `manifest`

### Phase 3: New Package Classification

When adding a new `internal/` package, classify it into one of the four layers and add it to the depguard configuration. Update this ADR's layer table if needed.

### Agent / LLM Impact

Clean layer separation directly supports Wave's multi-agent pipeline execution model:

- **Fresh memory per step**: Each persona starts with no chat history. When a persona operates within a single layer (e.g., a security auditor reviewing only cross-cutting and infrastructure packages), the bounded context reduces the codebase surface the persona must comprehend, improving accuracy within the token window.
- **Artifact-based communication**: Layer boundaries align with natural artifact boundaries — a domain step produces validated output (contracts), and a presentation step consumes it (display formatting). The layer model formalizes these handover points.
- **Persona scoping**: Personas can be constrained to operate within specific layers. A "UI specialist" persona needs presentation + cross-cutting access but not domain internals. Layer boundaries make these permission rules precise and auditable.
- **Workspace isolation**: Each pipeline step runs in an ephemeral worktree. Layer boundaries help determine which packages a step needs access to, enabling more targeted workspace mounts and reducing the attack surface of each step.
