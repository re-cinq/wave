# Data Model: Release-Gated Pipeline Embedding

**Feature Branch**: `029-release-gated-embedding`
**Date**: 2026-02-11

## Entity Diagram

```
┌──────────────────────────────┐
│      PipelineMetadata        │
│  (internal/pipeline/types.go)│
├──────────────────────────────┤
│  Name        string          │
│  Description string          │
│  Release     bool  (NEW)     │ ← defaults to false (Go zero value)
│  Disabled    bool  (NEW)     │ ← independent of Release
└──────────┬───────────────────┘
           │ parsed from
           │
┌──────────▼───────────────────┐
│        Pipeline              │
│  (internal/pipeline/types.go)│
├──────────────────────────────┤
│  Kind     string             │
│  Metadata PipelineMetadata   │
│  Input    InputConfig        │
│  Steps    []Step             │
└──────────┬───────────────────┘
           │ steps reference
           ▼
┌──────────────────────────────┐     ┌────────────────────────────────┐
│          Step                │     │     ContractConfig             │
├──────────────────────────────┤     ├────────────────────────────────┤
│  ID        string            │     │  SchemaPath string             │
│  Exec      ExecConfig ───────┼──┐  │  (e.g. .wave/contracts/x.json)│
│  Handover  HandoverConfig ───┼──┼──▶  normalized → "x.json"        │
│  ...                         │  │  └────────────────────────────────┘
└──────────────────────────────┘  │
                                  │  ┌────────────────────────────────┐
                                  │  │     ExecConfig                 │
                                  │  ├────────────────────────────────┤
                                  └──▶  SourcePath string             │
                                     │  (e.g. .wave/prompts/a/b.md)  │
                                     │  normalized → "a/b.md"         │
                                     └────────────────────────────────┘
```

## Entities

### PipelineMetadata (Modified)

**Location**: `internal/pipeline/types.go:18-21`
**Change**: Add two new boolean fields

```go
type PipelineMetadata struct {
    Name        string `yaml:"name"`
    Description string `yaml:"description,omitempty"`
    Release     bool   `yaml:"release,omitempty"`   // NEW
    Disabled    bool   `yaml:"disabled,omitempty"`   // NEW
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Release` | `bool` | `false` | Controls whether pipeline is distributed via `wave init`. Explicit opt-in. |
| `Disabled` | `bool` | `false` | Controls whether pipeline can be executed at runtime. Independent of `Release`. |

**Invariants**:
- `Release` and `Disabled` are independent (a pipeline can be `release: true, disabled: true`)
- Go's `bool` zero value (`false`) provides the correct default behavior
- `omitempty` ensures `false` values don't appear in marshalled YAML

### Release Pipeline Set (Computed, not persisted)

**Computed at**: `wave init` time in `internal/defaults/embed.go`
**Input**: All embedded pipelines (`GetPipelines()`)
**Output**: Subset where `pipeline.Metadata.Release == true`

```go
// GetReleasePipelines returns only pipelines with metadata.release: true
func GetReleasePipelines() (map[string]string, error)
```

The function:
1. Calls `GetPipelines()` to get all embedded pipeline YAML strings
2. Unmarshals each into `pipeline.Pipeline` struct
3. Filters to those where `Metadata.Release == true`
4. Returns the filtered map (same `map[string]string` format as `GetPipelines()`)

### Transitive Dependency Set (Computed, not persisted)

**Computed at**: `wave init` time in `cmd/wave/commands/init.go`
**Input**: Release pipeline set + all embedded contracts + all embedded prompts
**Output**: Filtered contracts and prompts maps

#### Contract References
- **Source field**: `Step.Handover.Contract.SchemaPath`
- **Reference format**: `.wave/contracts/plan-exploration.schema.json`
- **Embedded key format**: `plan-exploration.schema.json` (bare filename)
- **Normalization**: Strip `.wave/contracts/` prefix

#### Prompt References
- **Source field**: `Step.Exec.SourcePath`
- **Reference format**: `.wave/prompts/speckit-flow/specify.md`
- **Embedded key format**: `speckit-flow/specify.md` (relative path)
- **Normalization**: Strip `.wave/prompts/` prefix
- **Note**: Only `source_path` references count. Inline `source:` blocks have no file dependency.

#### Personas
- **Always included** — never transitively excluded (FR-005)
- Personas may be shared across multiple pipelines

### InitOptions (Modified)

**Location**: `cmd/wave/commands/init.go:16-23`
**Change**: Add `All bool` field

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

## Reference Normalization Rules

| Asset Type | YAML Reference Path | Embedded Map Key | Strip Prefix |
|------------|-------------------|------------------|-------------|
| Contract | `.wave/contracts/foo.schema.json` | `foo.schema.json` | `.wave/contracts/` |
| Prompt | `.wave/prompts/speckit-flow/specify.md` | `speckit-flow/specify.md` | `.wave/prompts/` |
| Persona | (referenced by name in `step.persona`) | `navigator.md` | N/A (always included) |

## Data Flow

```
wave init [--all]
    │
    ├── --all? ──YES──▶ GetPipelines()     → all pipelines
    │                    GetContracts()     → all contracts
    │                    GetPrompts()       → all prompts
    │                    GetPersonas()      → all personas
    │
    └── --all? ──NO───▶ GetReleasePipelines() → release pipelines only
                         │
                         ├─ Extract schema_path refs → normalize → contract include set
                         ├─ Extract source_path refs → normalize → prompt include set
                         │
                         ├─ Filter GetContracts() by include set → release contracts
                         ├─ Filter GetPrompts() by include set → release prompts
                         └─ GetPersonas() → all personas (always)
```
