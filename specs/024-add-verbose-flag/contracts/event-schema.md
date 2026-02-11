# Contract: Event Struct Verbose Fields

**Type**: Go struct field contract
**Package**: `internal/event`
**Struct**: `Event`

## New Fields

The following fields are added to the `Event` struct in `internal/event/emitter.go`. All fields use `omitempty` JSON tags to ensure zero output change when verbose mode is not active.

### WorkspacePath

```go
WorkspacePath string `json:"workspace_path,omitempty"`
```

- **Type**: `string`
- **Populated when**: Executor `verbose` is true and a step workspace is created or used
- **Content**: Absolute filesystem path to the step's workspace directory
- **Empty when**: Verbose mode is off, or the event is not workspace-related

### InjectedArtifacts

```go
InjectedArtifacts []string `json:"injected_artifacts,omitempty"`
```

- **Type**: `[]string`
- **Populated when**: Executor `verbose` is true and artifacts are injected into a step
- **Content**: List of artifact filenames (not full paths) injected into the workspace
- **Empty when**: Verbose mode is off, no artifacts injected, or event is not artifact-related

### ContractResult

```go
ContractResult string `json:"contract_result,omitempty"`
```

- **Type**: `string`
- **Populated when**: Executor `verbose` is true and contract validation completes
- **Content**: Human-readable contract validation result (e.g., "json_schema: passed", "test_suite: 3/3 assertions passed")
- **Empty when**: Verbose mode is off, step has no contract, or event is not validation-related

### VerboseDetail

```go
VerboseDetail string `json:"verbose_detail,omitempty"`
```

- **Type**: `string`
- **Populated when**: Executor `verbose` is true and additional context is available
- **Content**: General-purpose verbose message providing operational insight
- **Empty when**: Verbose mode is off or no additional context is available

## Backward Compatibility

All new fields use `omitempty` JSON tags. When verbose mode is not active:
- Fields are zero-valued (empty string or nil slice)
- `omitempty` ensures they are omitted from JSON serialization
- NDJSON output is byte-identical to current behavior
- Existing consumers parsing NDJSON are not affected by new optional fields

## Validation Rules

- `WorkspacePath` must be an absolute path when populated (starts with `/`)
- `InjectedArtifacts` must contain only filenames (no path separators)
- `ContractResult` must be non-empty when populated
- `VerboseDetail` has no format constraints

## WithVerbose ExecutorOption

```go
func WithVerbose(verbose bool) ExecutorOption {
    return func(ex *DefaultPipelineExecutor) { ex.verbose = verbose }
}
```

- **Pattern**: Identical to `WithDebug` at `executor.go:74-76`
- **Default**: `false` (verbose mode off)
- **Effect**: When `true`, executor populates verbose fields on emitted events
