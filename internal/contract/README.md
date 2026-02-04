# Wave Contract System

## Overview

The Wave contract system provides deterministic output guarantees for AI-powered pipelines through multi-layer validation, adaptive retry strategies, and rollback mechanisms.

## Architecture

```
contract/
├── contract.go           # Core validation infrastructure
├── quality_gate.go       # Quality gate validation framework
├── retry_strategy.go     # Adaptive retry with failure classification
├── format_validator.go   # Production-ready format enforcement
├── template.go           # Structured template validation
├── verification.go       # Automated verification checks
├── rollback.go           # State management and rollback
├── jsonschema.go         # JSON Schema validator
├── typescript.go         # TypeScript interface validator
├── testsuite.go          # Test suite executor
└── markdownspec.go       # Markdown structure validator
```

## Key Components

### 1. Contract Validators

**Interface:**
```go
type ContractValidator interface {
    Validate(cfg ContractConfig, workspacePath string) error
}
```

**Implementations:**
- `jsonSchemaValidator` - JSON Schema (RFC 7159) validation
- `typeScriptValidator` - TypeScript interface validation
- `testSuiteValidator` - Test execution validation
- `markdownSpecValidator` - Markdown structure validation
- `TemplateValidator` - Structured template validation (NEW)
- `FormatValidator` - Production format validation (NEW)

### 2. Quality Gates

Quality gates ensure output quality beyond schema compliance:

```go
type QualityGate interface {
    Check(workspacePath string, config QualityGateConfig) ([]QualityViolation, error)
    Name() string
}
```

**Standard Gates:**
- `LinkValidationGate` - Validates markdown links
- `MarkdownStructureGate` - Checks heading hierarchy
- `JSONStructureGate` - Validates JSON formatting
- `RequiredFieldsGate` - Ensures required fields present
- `ContentCompletenessGate` - Checks content quality
- `VerificationGate` - Runs automated verification (NEW)

### 3. Retry Strategy

Adaptive retry with intelligent failure classification:

```go
type RetryStrategy interface {
    ShouldRetry(attempt int, err error) bool
    GetRetryDelay(attempt int) time.Duration
    GenerateRepairPrompt(err error, attempt int) string
}
```

**Features:**
- Automatic failure classification
- Targeted repair prompts
- Exponential backoff with jitter
- Failure type tracking

**Failure Types:**
- `schema_mismatch` - Wrong types or missing fields
- `format_error` - Invalid syntax
- `missing_content` - Incomplete content
- `quality_gate` - Failed quality checks
- `structure` - Incorrect document structure

### 4. Format Validation (NEW)

Ensures outputs are production-ready:

```go
validator := &FormatValidator{}
err := validator.Validate(cfg, workspacePath)
```

**Format Types:**
- `github_issue` - GitHub issue format (title, body, labels)
- `github_pr` - Pull request format (conventional commits, testing, references)
- `implementation_results` - Code implementation (compilation, tests, completeness)
- `analysis` - Analysis outputs (findings, recommendations)
- `generic` - Generic placeholder detection

**Validation Rules:**
- Minimum/maximum length constraints
- Required section presence
- Placeholder detection (TODO, FIXME, TBD)
- Format compliance (conventional commits, etc.)
- Cross-reference validation

### 5. Verification Framework (NEW)

Automated verification of generated outputs:

```go
gate := &VerificationGate{}
violations, err := gate.Check(workspacePath, config)
```

**Verification Rules:**
- `code_compilation` - Verify code compiles
- `test_execution` - Run test suite
- `link_validation` - Check URLs are valid
- `cross_reference` - Validate issue/PR references
- `file_existence` - Ensure required files exist

### 6. Rollback Manager (NEW)

State management for failed pipelines:

```go
manager := NewRollbackManager(stateDir)

// Create checkpoint
checkpoint, _ := manager.CreateCheckpoint(pipelineID, stepID, workspace, artifacts)

// Log operations
manager.LogOperation(pipelineID, RollbackOperation{
    Type: "file_created",
    Target: "/path/to/file",
    CanRevert: true,
})

// Rollback on failure
manager.Rollback(pipelineID, checkpoint)
```

**Operation Types:**
- `file_created` - Delete created file
- `file_modified` - Restore from backup
- `file_deleted` - Restore from backup
- `git_commit` - Manual revert instructions

## Usage

### Basic Validation

```go
cfg := contract.ContractConfig{
    Type:       "json_schema",
    Source:     "output.json",
    SchemaPath: "schema.json",
}

err := contract.Validate(cfg, workspacePath)
```

### With Retry

```go
cfg.MaxRetries = 3
result, err := contract.ValidateWithAdaptiveRetry(cfg, workspacePath)

if result.Success {
    fmt.Printf("Passed after %d attempts\n", result.Attempts)
} else {
    fmt.Printf("Failed: %v\n", result.FinalError)
}
```

### With Quality Gates

```go
cfg := contract.ContractConfig{
    Type:       "json_schema",
    SchemaPath: "schema.json",
    QualityGates: []contract.QualityGateConfig{
        {
            Type:     "required_fields",
            Required: true,
            Parameters: map[string]interface{}{
                "fields": []string{"title", "body", "labels"},
            },
        },
        {
            Type:      "content_completeness",
            Threshold: 80,
            Parameters: map[string]interface{}{
                "min_words": 50,
            },
        },
    },
}

err := contract.Validate(cfg, workspacePath)
```

### Format Validation

```go
cfg := contract.ContractConfig{
    Type:       "format",
    Source:     "output.json",
    SchemaPath: "github-pr-draft.schema.json", // Infers format type
}

validator := contract.NewValidator(cfg)
err := validator.Validate(cfg, workspacePath)
```

### With Rollback

```go
manager := contract.NewRollbackManager(".wave/rollback")

// Initialize
log, _ := manager.InitRollbackLog("my-pipeline")

// Create checkpoint before risky operation
checkpoint, _ := manager.CreateCheckpoint("my-pipeline", "step-1", workspace, artifacts)

// Log operations
manager.LogOperation("my-pipeline", contract.RollbackOperation{
    Type:      "file_created",
    Target:    newFile,
    CanRevert: true,
})

// On failure, rollback
if err != nil {
    manager.Rollback("my-pipeline", checkpoint)
}
```

## Configuration

### Pipeline Configuration

```yaml
handover:
  contract:
    type: json_schema
    schema_path: .wave/contracts/output.schema.json
    validate: true
    must_pass: true      # Block pipeline if validation fails
    max_retries: 3       # Retry up to 3 times
  quality_gates:
    - type: required_fields
      required: true
      parameters:
        fields: ["title", "body"]
    - type: content_completeness
      threshold: 80
      parameters:
        min_words: 50
    - type: verification
      required: true
      parameters:
        rules:
          - type: code_compilation
            command: "go build ./..."
          - type: test_execution
            command: "go test ./..."
```

## Testing

```bash
# Run all contract tests
go test ./internal/contract/...

# Run with race detector
go test -race ./internal/contract/...

# Run specific test
go test -v ./internal/contract -run TestFormatValidator

# Run with coverage
go test -cover ./internal/contract/...
```

## Examples

See:
- `format_validator_test.go` - Format validation examples
- `rollback_test.go` - Rollback mechanism examples

## Integration Points

### Pipeline Executor

The executor automatically runs contract validation:

```go
// In pipeline/executor.go
if step.Handover.Contract.Type != "" {
    contractCfg := contract.ContractConfig{
        Type:         step.Handover.Contract.Type,
        Source:       resolvedSource,
        SchemaPath:   step.Handover.Contract.SchemaPath,
        MaxRetries:   step.Handover.Contract.MaxRetries,
        QualityGates: step.Handover.Contract.QualityGates,
    }

    err := contract.Validate(contractCfg, workspacePath)
    if err != nil && contractCfg.MustPass {
        return fmt.Errorf("contract validation failed: %w", err)
    }
}
```

### Event Emission

Validation events are emitted for monitoring:

```go
e.emit(event.Event{
    PipelineID:      pipelineID,
    StepID:          stepID,
    State:           "validating",
    ValidationPhase: "contract",
})
```

## Performance

- Schema validation: ~10-20ms
- Format validation: ~5-10ms
- Quality gates: ~20-50ms total
- Retry with backoff: 1s, 2s, 4s, ...
- Checkpoint creation: < 100ms
- Rollback: < 1s for typical operations

## Future Enhancements

1. **Machine Learning Integration**
   - Predict likely failure types
   - Suggest schema improvements
   - Auto-tune retry parameters

2. **Visual Validation**
   - HTML/Markdown validation reports
   - Diff views for retry attempts
   - Interactive validation results

3. **Advanced Verification**
   - Container-based test execution
   - External service integration
   - Parallel verification for speed

4. **Schema Evolution**
   - Automatic schema versioning
   - Migration tools
   - Backward compatibility checks

## Contributing

When adding new validators or gates:

1. Implement the appropriate interface
2. Add to validator/gate registry
3. Write comprehensive tests
4. Update documentation
5. Add example usage

## License

Part of the Wave project - see root LICENSE file.
