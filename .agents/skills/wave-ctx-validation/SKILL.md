---
name: wave-ctx-validation
description: Domain context for Multi-layer output validation system — contract enforcement with six validator types (json_schema, typescript_interface, test_suite, markdown_spec, format, non_empty_file), progressive JSON recovery with three aggressiveness tiers, adaptive retry with failure classification and repair prompts, error wrapper detection for AI-generated envelopes, input artifact schema validation, preflight dependency checks with auto-install, and pipeline-level phase/concurrency/staleness validation.
---

# Validation Context

Multi-layer output validation system — contract enforcement with six validator types (json_schema, typescript_interface, test_suite, markdown_spec, format, non_empty_file), progressive JSON recovery with three aggressiveness tiers, adaptive retry with failure classification and repair prompts, error wrapper detection for AI-generated envelopes, input artifact schema validation, preflight dependency checks with auto-install, and pipeline-level phase/concurrency/staleness validation.

## Invariants

- Unknown contract types silently pass — NewValidator returns nil, Validate returns nil
- must_pass controls whether contract failure blocks the pipeline — true=hard error, false=soft failure event
- maxRetries<=0 defaults to 1 attempt (no retry) in basic mode, but 3 in adaptive mode — intentional asymmetry
- After exhausting all retries, Retryable is set to false on the ValidationError
- JSON schema validation requires either Schema (inline) or SchemaPath (file) — neither produces non-retryable error
- Test suite validator rejects unresolved template variables — '{{ ' or '{{' in command produces non-retryable error
- Test suite defaults Dir to 'project_root' resolved via git rev-parse --show-toplevel
- TypeScript validator degrades gracefully when tsc absent — must_pass:true = hard error with install instructions, must_pass:false = silently skips
- JSON recovery defaults to progressive when not explicitly configured and must_pass is false — conservative for must_pass contracts
- Recovered JSON must be structurally complete (balanced braces/brackets) — incomplete recovery is retryable error
- Error wrapper detection requires error_type AND raw_output AND at least 3 indicator fields
- Input artifact validation fails fast — first invalid artifact stops all further checks
- Non-empty file validator requires file exists AND has size > 0
- Missing tools return ToolError with list of missing tool names
- Missing skills return SkillError with list of missing skill names
- PreflightError wraps both ToolError and SkillError, supporting errors.As() for either
- Skills with Install command are auto-installed — post-install verification retries with $HOME/.local/bin in PATH
- Init commands NOT run in preflight — deferred to worktree creation
- Phase skip validation only applies to impl-prototype or prototype pipelines
- Generic step sequence validation requires prior run state to exist before resuming from a step
- on_failure:rework requires rework_step to be set — validated at parse time
- Workspace locks prevent concurrent access — in-memory lock returns error if workspace in use
- Optional steps default to on_failure:continue; non-optional default to on_failure:fail
- Format type inferred from schema filename — github-pr-draft.schema.json maps to github_pr

## Key Decisions

- Six contract types with dedicated validators — json_schema, typescript_interface, test_suite, markdown_spec, format, non_empty_file — each plugged in via factory pattern
- Three-tier progressive JSON recovery (conservative, progressive, aggressive) — must_pass gets conservative (data integrity), soft contracts get progressive (AI compatibility)
- Two independent retry layers — contract-level retries re-validate same data, step-level retries re-run entire adapter with optional prompt adaptation
- Adaptive retry classifies failures into 6 types (schema_mismatch, missing_content, format_error, quality_gate, structure, unknown) with targeted repair prompts
- Exponential backoff with jitter for adaptive retry — base 1s, max 30s, factor 2.0
- Error wrapper detection for AI-generated error envelopes — extracts raw_output field content for validation instead of failing on the wrapper
- Input vs output validation separation — input validation at artifact injection (before step), output validation after adapter returns
- Preflight runs before any step — tool/skill existence checks with auto-install for skills
- Format validator provides production-readiness rules per output type — placeholder detection, required section validation, content length checks
- ValidationErrorFormatter provides actionable guidance with error categorization (missing_required_fields, type_mismatch, enum_violation, additional_properties, array_issues, format_violation), common pitfalls, and fix examples

## Domain Vocabulary

| Term | Meaning |
|------|--------|
| ContractConfig | Configuration for contract validation — type, source, schema, retries, recovery level, must_pass, on_failure |
| ContractValidator | Interface: Validate(cfg ContractConfig, workspacePath string) error — pluggable per contract type |
| ValidationError | Structured error with ContractType, Message, Details, Retryable, Attempt, MaxRetries |
| RetryStrategy | Interface: ShouldRetry, GetRetryDelay, GenerateRepairPrompt — pluggable retry behavior |
| AdaptiveRetryStrategy | Concrete strategy with failure classification, exponential backoff, and targeted repair prompts |
| FailureClassifier | Analyzes validation errors to determine FailureType for targeted repair |
| ClassifiedFailure | Contains Type, Message, Details, Retryable, Confidence, Suggestions |
| RetryResult | Outcome of retry sequence — Success, Attempts, FailureTypes, TotalDuration, FinalError |
| JSONRecoveryParser | Progressive JSON parser with tiered recovery strategies (conservative/progressive/aggressive) |
| RecoveryResult | Contains OriginalInput, RecoveredJSON, IsValid, AppliedFixes, Warnings, ParsedData |
| RecoveryLevel | Enum: ConservativeRecovery (safe fixes only), ProgressiveRecovery (structural fixes), AggressiveRecovery (inference-based reconstruction) |
| ErrorWrapper | AI-generated error envelope structure detected by matching indicator fields |
| WrapperDetectionResult | Detection outcome with IsWrapper, RawContent, Confidence, FieldsMatched |
| FormatValidator | Production-readiness validator for structured outputs like GitHub issues, PRs, implementation results |
| InputValidationResult | Input artifact check outcome — Passed, ArtifactRef, Error, TypeMatch, SchemaValid |
| SchemaErrorAnalysis | Categorized schema error with MainMessage, ErrorType, Suggestions, CommonPitfalls, Example, Retryable |
| Checker | Preflight dependency validator — checks tools on PATH and skills installed |
| ToolError | Error wrapping list of missing CLI tools |
| SkillError | Error wrapping list of missing skills with auto-install support |
| PreflightError | Composite error wrapping both ToolError and SkillError, supporting errors.As() |
| PhaseSkipValidator | Validates phase prerequisites when resuming prototype pipelines |
| StaleArtifactDetector | Detects stale upstream artifacts by comparing modification times |
| ConcurrencyValidator | In-memory workspace locks preventing concurrent pipeline execution on same workspace |
| FailureTypeSchemaMismatch | Wrong types or missing fields in validated output |
| FailureTypeMissingContent | Required content absent from output |
| FailureTypeFormatError | Invalid JSON syntax in output |
| FailureTypeQualityGate | Output fails quality check thresholds |
| FailureTypeStructure | Incorrect document structure in output |
| FailureTypeUnknown | Unclassifiable error — NOT retryable |

## Neighboring Contexts

- **execution**
- **security**
- **configuration**

## Key Files

- `internal/contract/contract.go`
- `internal/contract/jsonschema.go`
- `internal/contract/jsonschema_recovery.go`
- `internal/contract/retry_strategy.go`
- `internal/contract/testsuite.go`
- `internal/contract/format.go`
- `internal/contract/jsonschema_wrapper.go`
- `internal/contract/jsonschema_input.go`
- `internal/contract/validation_error_formatter.go`
- `internal/contract/non_empty_file.go`
- `internal/preflight/preflight.go`
- `internal/pipeline/validation.go`

