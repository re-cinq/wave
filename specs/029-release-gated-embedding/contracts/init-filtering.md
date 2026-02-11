# API Contract: Init Command Release Filtering

**Package**: `cmd/wave/commands`
**File**: `init.go`

## Modified Struct

### `InitOptions`

```go
type InitOptions struct {
    Force      bool
    Merge      bool
    All        bool   // NEW — bypass release filtering
    Adapter    string
    Workspace  string
    OutputPath string
    Yes        bool
}
```

**New flag registration**:
```go
cmd.Flags().BoolVar(&opts.All, "all", false, "Include all pipelines regardless of release status")
```

## Modified Behavior: `runInit()`

### Without `--all` (default)

1. Get release pipelines: `defaults.GetReleasePipelines()`
2. Extract contract references from release pipelines (normalized `schema_path` values)
3. Extract prompt references from release pipelines (normalized `source_path` values)
4. Filter `defaults.GetContracts()` to only those in the contract reference set
5. Filter `defaults.GetPrompts()` to only those in the prompt reference set
6. Get all personas: `defaults.GetPersonas()` (unfiltered)
7. Write filtered assets to `.wave/` directories
8. If zero release pipelines, warn on stderr, succeed with empty `.wave/pipelines/`

### With `--all`

1. Get all pipelines: `defaults.GetPipelines()`
2. Get all contracts: `defaults.GetContracts()`
3. Get all prompts: `defaults.GetPrompts()`
4. Get all personas: `defaults.GetPersonas()`
5. Write all assets to `.wave/` directories
6. Behavior identical to current (pre-feature) `wave init`

## Modified Behavior: `runMerge()`

### Without `--all`

- Uses `defaults.GetReleasePipelines()` instead of `defaults.GetPipelines()`
- Applies same transitive exclusion to contracts and prompts
- Only adds missing release-flagged pipelines and their dependencies
- Existing files are preserved (never deleted)

### With `--all` and `--merge`

- Both flags compose: uses `defaults.GetPipelines()` (all) for merge
- Adds all missing pipelines and their dependencies
- Existing files preserved

## New Internal Function: `filterTransitiveDeps()`

```go
// filterTransitiveDeps computes the transitive dependency sets for contracts and prompts
// based on the pipeline set. Returns filtered contract and prompt maps.
func filterTransitiveDeps(
    pipelines map[string]string,
    allContracts map[string]string,
    allPrompts map[string]string,
) (contracts map[string]string, prompts map[string]string, err error)
```

**Algorithm**:
1. Parse each pipeline YAML into `pipeline.Pipeline`
2. Walk all steps, collect `Handover.Contract.SchemaPath` values
3. Walk all steps, collect `Exec.SourcePath` values
4. Normalize contract refs: strip `.wave/contracts/` prefix
5. Normalize prompt refs: strip `.wave/prompts/` prefix
6. Filter `allContracts` to keys present in contract ref set
7. Filter `allPrompts` to keys present in prompt ref set
8. Return filtered maps

**Contract**:
- Missing schema files (referenced but not in embedded FS) emit a warning, not an error
- Inline `source:` blocks (no `source_path`) contribute nothing to the prompt ref set
- Empty `schema_path` or `source_path` values are ignored

## Modified Behavior: `printInitSuccess()`

- Accepts extracted asset counts/names rather than querying `defaults.Get*()` directly
- Displays post-filtering counts (e.g., "3 pipelines" not "18 pipelines")
- Pipeline names list reflects only the extracted set
