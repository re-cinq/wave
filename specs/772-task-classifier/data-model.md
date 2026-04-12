# Data Model: Wave Task Classifier

**Date**: 2026-04-12 | **Branch**: `772-task-classifier`

## Entities

### TaskProfile

The central classification output. Represents a structured assessment of a task's characteristics along five dimensions.

```go
package classify

import "github.com/recinq/wave/internal/suggest"

// Complexity enumerates task complexity levels.
type Complexity string

const (
    ComplexitySimple        Complexity = "simple"
    ComplexityMedium        Complexity = "medium"
    ComplexityComplex       Complexity = "complex"
    ComplexityArchitectural Complexity = "architectural"
)

// Domain enumerates task domain categories.
type Domain string

const (
    DomainSecurity    Domain = "security"
    DomainPerformance Domain = "performance"
    DomainBug         Domain = "bug"
    DomainRefactor    Domain = "refactor"
    DomainFeature     Domain = "feature"
    DomainDocs        Domain = "docs"
    DomainResearch    Domain = "research"
)

// VerificationDepth enumerates verification levels.
type VerificationDepth string

const (
    VerificationStructuralOnly VerificationDepth = "structural_only"
    VerificationBehavioral     VerificationDepth = "behavioral"
    VerificationFullSemantic   VerificationDepth = "full_semantic"
)

// TaskProfile is the structured output of task classification.
type TaskProfile struct {
    BlastRadius       float64           // 0.0-1.0, risk/impact score
    Complexity        Complexity        // simple/medium/complex/architectural
    Domain            Domain            // security/performance/bug/refactor/feature/docs/research
    VerificationDepth VerificationDepth // structural_only/behavioral/full_semantic
    InputType         suggest.InputType // reused from internal/suggest
}
```

**Field derivation rules** (FR-009, FR-010):
- `BlastRadius`: base from complexity (simple=0.1, medium=0.3, complex=0.6, architectural=0.8) + domain modifier (security=+0.2, performance=+0.1, docs=-0.1), clamped [0.0, 1.0]
- `VerificationDepth`: derived from complexity (simple→structural_only, medium→behavioral, complex/architectural→full_semantic)
- `InputType`: set by calling `suggest.ClassifyInput(input)` before keyword analysis

### PipelineConfig

The output of pipeline selection. Minimal struct with pipeline name and routing rationale.

```go
// PipelineConfig is the result of pipeline selection.
type PipelineConfig struct {
    Name   string // pipeline name, e.g. "impl-issue"
    Reason string // human-readable routing explanation
}
```

## Relationships

```
User Input (string, string)
    │
    ▼
┌─────────────┐     uses      ┌──────────────────┐
│  Classify()  │──────────────▶│ suggest.ClassifyInput() │
│  analyzer.go │               │ internal/suggest       │
└──────┬──────┘               └──────────────────┘
       │ returns
       ▼
┌──────────────┐
│  TaskProfile  │
│  profile.go   │
└──────┬───────┘
       │ consumed by
       ▼
┌────────────────┐
│ SelectPipeline()│
│ selector.go     │
└───────┬────────┘
        │ returns
        ▼
┌────────────────┐
│ PipelineConfig  │
└────────────────┘
```

## Defaults (Edge Cases)

| Condition | Default Values |
|-----------|---------------|
| Empty/whitespace input | complexity=simple, domain=feature, blast_radius=0.1, verification_depth=structural_only |
| No recognizable keywords | complexity=medium, domain=feature, blast_radius=0.3 |
| Undetermined blast_radius | 0.5 (moderate risk) |
| PR URL input | Short-circuits to ops-pr-review regardless of content analysis |

## Domain Priority Ordering (FR-011)

When multiple domain signals detected: security > performance > bug > refactor > feature > docs > research
