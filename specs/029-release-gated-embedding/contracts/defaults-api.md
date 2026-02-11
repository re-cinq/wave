# API Contract: defaults Package Extensions

**Package**: `internal/defaults`
**File**: `embed.go`

## New Functions

### `GetReleasePipelines() (map[string]string, error)`

Returns only embedded pipelines where `metadata.release: true`.

**Behavior**:
- Calls `GetPipelines()` to get all embedded pipeline YAML strings
- Unmarshals each YAML into `pipeline.Pipeline` struct
- Returns only entries where `pipeline.Metadata.Release == true`
- Returns the same `map[string]string` format (filename → YAML content)

**Contract**:
- Result is a strict subset of `GetPipelines()` — every key in the result also exists in `GetPipelines()`
- If no pipelines have `release: true`, returns an empty map (not nil), no error
- If YAML unmarshalling fails for a pipeline, that pipeline is excluded (logged as warning, not a hard error)
- Keys are bare filenames (e.g., `doc-loop.yaml`)

### `ReleasePipelineNames() []string`

Returns filenames of release-flagged pipelines.

**Behavior**:
- Calls `GetReleasePipelines()`
- Extracts and returns keys

**Contract**:
- Result is a strict subset of `PipelineNames()`
- Order is non-deterministic (map iteration)

## Unchanged Functions

The following existing functions remain unchanged:

- `GetPipelines()` — returns all pipelines (unfiltered)
- `GetPersonas()` — returns all personas (never filtered)
- `GetContracts()` — returns all contracts (filtering applied by caller)
- `GetPrompts()` — returns all prompts (filtering applied by caller)
- `PipelineNames()` — returns all pipeline names
- `PersonaNames()` — unchanged
- `ContractNames()` — unchanged
- `PromptNames()` — unchanged
