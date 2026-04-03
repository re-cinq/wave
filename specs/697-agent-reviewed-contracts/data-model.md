# Data Model: Agent-Reviewed Contracts

**Feature**: #697 Agent-Reviewed Contracts
**Date**: 2026-03-30

## Entity Relationship Overview

```
Step
 └── HandoverConfig
      ├── Contract (singular, backward-compat)
      └── Contracts[] (ordered list)
           └── ContractConfig
                ├── Type: "agent_review"
                ├── AgentReview (embedded config)
                │    ├── Persona (reviewer)
                │    ├── CriteriaPath
                │    ├── Context[] → ReviewContextSource
                │    ├── TokenBudget
                │    └── Timeout
                ├── OnFailure: "rework" | "fail" | "skip" | ...
                ├── ReworkStep
                └── MaxRetries

ReviewContextSource
 ├── Source: "git_diff" | "artifact"
 ├── Artifact: string (artifact name)
 └── MaxSize: int (bytes, for truncation)

ReviewFeedback (output)
 ├── Verdict: "pass" | "fail" | "warn"
 ├── Issues[] → ReviewIssue
 │    ├── Severity: "error" | "warning" | "info"
 │    └── Description: string
 ├── Suggestions[] → string
 └── Confidence: float64 (0.0–1.0)
```

## New Types

### ReviewFeedback

**Package**: `internal/contract`
**Purpose**: Structured output of an agent review. Extracted from reviewer stdout via JSON parsing. Written as artifact for rework injection, displayed in dashboard, tracked in retros.

```go
type ReviewFeedback struct {
    Verdict     string        `json:"verdict"`     // "pass", "fail", "warn"
    Issues      []ReviewIssue `json:"issues"`
    Suggestions []string      `json:"suggestions"`
    Confidence  float64       `json:"confidence"`  // 0.0–1.0
}

type ReviewIssue struct {
    Severity    string `json:"severity"`    // "error", "warning", "info"
    Description string `json:"description"`
}
```

**Invariants**:
- `Verdict` must be one of: "pass", "fail", "warn"
- `Confidence` must be in range [0.0, 1.0]
- When `Verdict` is "pass", `Issues` should be empty (validated but not enforced — reviewer may still note minor items)
- JSON serialization format matches what the reviewer agent is instructed to produce

### ReviewContextSource

**Package**: `internal/contract` (or inline in `agent_review.go`)
**Purpose**: Configures what context the reviewer receives. Each source is assembled at review time and concatenated into the reviewer's user prompt.

```go
type ReviewContextSource struct {
    Source   string `json:"source" yaml:"source"`     // "git_diff" or "artifact"
    Artifact string `json:"artifact" yaml:"artifact"` // artifact name (when source = "artifact")
    MaxSize  int    `json:"max_size" yaml:"max_size"` // truncation limit in bytes (default 50KB for git_diff)
}
```

### AgentReviewConfig

**Package**: `internal/pipeline` (embedded in `ContractConfig`)
**Purpose**: Agent-review-specific fields on `ContractConfig`.

```go
// New fields added to pipeline.ContractConfig:
type ContractConfig struct {
    // ... existing fields ...

    // Agent review settings
    Persona      string                `yaml:"persona,omitempty"`       // Reviewer persona name
    CriteriaPath string                `yaml:"criteria_path,omitempty"` // Path to review criteria markdown
    Context      []ReviewContextSource `yaml:"context,omitempty"`       // Context sources for reviewer
    TokenBudget  int                   `yaml:"token_budget,omitempty"`  // Max tokens for review
    Timeout      int                   `yaml:"timeout,omitempty"`       // Timeout in seconds (default 120)
    ReworkStep   string                `yaml:"rework_step,omitempty"`   // Step ID for rework on failure
}
```

**YAML example**:
```yaml
handover:
  contracts:
    - type: test_suite
      command: "{{ project.contract_test_command }}"
      dir: project_root
      must_pass: true
    - type: agent_review
      persona: navigator
      criteria_path: .wave/contracts/impl-review-criteria.md
      model: claude-haiku-4-5
      context:
        - source: git_diff
        - source: artifact
          artifact: assessment
      token_budget: 8000
      on_failure: rework
      rework_step: rework-impl
      max_retries: 1
```

## Modified Types

### HandoverConfig

**Package**: `internal/pipeline`
**Current** (`types.go:417-422`):
```go
type HandoverConfig struct {
    Contract     ContractConfig   `yaml:"contract,omitempty"`
    Compaction   CompactionConfig `yaml:"compaction,omitempty"`
    OnReviewFail string           `yaml:"on_review_fail,omitempty"`
    TargetStep   string           `yaml:"target_step,omitempty"`
}
```

**Modified**:
```go
type HandoverConfig struct {
    Contract     ContractConfig   `yaml:"contract,omitempty"`
    Contracts    []ContractConfig `yaml:"contracts,omitempty"`  // NEW: ordered list
    Compaction   CompactionConfig `yaml:"compaction,omitempty"`
    OnReviewFail string           `yaml:"on_review_fail,omitempty"`
    TargetStep   string           `yaml:"target_step,omitempty"`
}
```

**Normalization**: `EffectiveContracts()` method returns the canonical list:
```go
func (h HandoverConfig) EffectiveContracts() []ContractConfig {
    if len(h.Contracts) > 0 {
        return h.Contracts
    }
    if h.Contract.Type != "" {
        return []ContractConfig{h.Contract}
    }
    return nil
}
```

### ContractConfig (pipeline)

**Package**: `internal/pipeline`
**Current** (`types.go:424-440`): Type, Schema, Source, SchemaPath, Validate, Command, Dir, MustPass, OnFailure, MaxRetries, Model, Criteria, Threshold.

**Added fields**: `Persona`, `CriteriaPath`, `Context` ([]ReviewContextSource), `TokenBudget`, `Timeout`, `ReworkStep`.

### ContractConfig (contract package)

**Package**: `internal/contract`
**Current** (`contract.go:10-35`): Type, Source, Schema, SchemaPath, Command, CommandArgs, Dir, MustPass, MaxRetries, progressive/recovery/wrapper fields, Model, Criteria, Threshold.

**Added fields**: `Persona`, `CriteriaPath`, `Context` ([]ReviewContextSource), `TokenBudget`, `Timeout`, `ReworkStep`.

### AgentContractValidator (new interface)

**Package**: `internal/contract`
**Purpose**: Extension of `ContractValidator` for validators that need an adapter runner.

```go
type AgentContractValidator interface {
    ContractValidator
    ValidateWithRunner(cfg ContractConfig, workspacePath string, runner adapter.AdapterRunner, manifest interface{}) (*ReviewFeedback, error)
}
```

### FrictionType (retro)

**Package**: `internal/retro`
**Added constant**:
```go
FrictionReviewRework FrictionType = "review_rework"
```

### StepDetail (webui)

**Package**: `internal/webui`
**Added fields**:
```go
ReviewVerdict      string // "pass", "fail", "warn", or empty
ReviewIssueCount   int
ReviewerPersona    string
ReviewTokens       int
```

## State and Persistence

### Pipeline State (SQLite)

No schema changes required. Review verdicts are captured in:
- **Events**: `review_started`, `review_completed`, `review_failed` events stored via existing event persistence
- **Decision log**: Review pass/fail recorded via existing `recordDecision` mechanism
- **Performance metrics**: Review token spend recorded via existing `RecordPerformanceMetric`

### Artifact Storage

- **ReviewFeedback JSON**: Written to `.wave/artifacts/review_feedback.json` in the step's workspace when a review completes (pass or fail). Overwritten on re-review after rework.
- **Review criteria**: Referenced by path (not copied). Validated at pipeline load time.

## Validation Rules

1. **Self-review prevention**: `contract.Persona != step.Persona` — hard error at DAG validation
2. **Criteria path existence**: `contract.CriteriaPath` file must exist — hard error at pipeline load
3. **Token budget positive**: `contract.TokenBudget > 0` when set — hard error at pipeline load
4. **Rework step validity**: When `on_failure: rework`, `rework_step` must reference a valid `rework_only` step — reuses existing rework validation
5. **Reviewer persona exists**: `contract.Persona` must reference a persona defined in the manifest — hard error at pipeline load
6. **No mixed singular/plural**: If both `contract` and `contracts` are set on the same step, validation warns (contracts takes precedence)
