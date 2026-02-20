# Data Model: Typed Artifact Composition

**Feature Branch**: `109-typed-artifact-composition`
**Created**: 2026-02-20
**Status**: Complete

## Entity Overview

This feature extends existing Wave data structures to support stdout artifact capture, typed artifact consumption, and bidirectional contract validation.

## Entities

### 1. ArtifactDef (Extended)

**Location**: `internal/pipeline/types.go:97-102`

**Purpose**: Defines an artifact produced by a pipeline step. Extended to support stdout as a source.

**Current Definition**:
```go
type ArtifactDef struct {
    Name     string `yaml:"name"`
    Path     string `yaml:"path"`
    Type     string `yaml:"type,omitempty"`
    Required bool   `yaml:"required,omitempty"`
}
```

**Extended Definition**:
```go
type ArtifactDef struct {
    Name     string `yaml:"name"`
    Path     string `yaml:"path,omitempty"`          // Optional when Source is "stdout"
    Type     string `yaml:"type,omitempty"`          // "json", "text", "markdown", "binary"
    Required bool   `yaml:"required,omitempty"`
    Source   string `yaml:"source,omitempty"`        // NEW: "file" (default) or "stdout"
}
```

**Field Details**:

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `Name` | string | Yes | - | Artifact identifier, referenced by downstream `inject_artifacts` |
| `Path` | string | When Source=file | - | Filesystem path relative to workspace. Auto-generated for stdout artifacts. |
| `Type` | string | No | - | Content type: `json`, `text`, `markdown`, `binary` |
| `Required` | bool | No | false | If true, step fails when artifact is missing |
| `Source` | string | No | `file` | Where content comes from: `file` (persona writes file) or `stdout` (captured from process output) |

**Validation Rules**:
- When `Source == "stdout"`: `Path` is ignored; system generates `.wave/artifacts/<step-id>/<name>`
- When `Source == "file"` or empty: `Path` is required
- `Type` must be one of: `json`, `text`, `markdown`, `binary`

**YAML Example**:
```yaml
output_artifacts:
  - name: analysis-report
    source: stdout
    type: json
  - name: summary
    path: .wave/output/summary.md
    type: markdown
```

---

### 2. ArtifactRef (Extended)

**Location**: `internal/pipeline/types.go:68-72`

**Purpose**: References an artifact from a prior step for injection into current step. Extended to support type validation and optional schema.

**Current Definition**:
```go
type ArtifactRef struct {
    Step     string `yaml:"step"`
    Artifact string `yaml:"artifact"`
    As       string `yaml:"as"`
}
```

**Extended Definition**:
```go
type ArtifactRef struct {
    Step       string `yaml:"step"`
    Artifact   string `yaml:"artifact"`
    As         string `yaml:"as"`
    Type       string `yaml:"type,omitempty"`        // NEW: Expected artifact type
    SchemaPath string `yaml:"schema_path,omitempty"` // NEW: JSON schema for validation
    Optional   bool   `yaml:"optional,omitempty"`    // NEW: If true, missing artifact doesn't fail
}
```

**Field Details**:

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `Step` | string | Yes | - | ID of the step that produced the artifact |
| `Artifact` | string | Yes | - | Name of the artifact (matches `ArtifactDef.Name`) |
| `As` | string | Yes | - | Name to mount under `.wave/artifacts/` in workspace |
| `Type` | string | No | - | Expected type; fails if mismatched |
| `SchemaPath` | string | No | - | Path to JSON schema file for content validation |
| `Optional` | bool | No | false | If true, missing artifact is allowed |

**Validation Rules**:
- When `Type` is specified: Artifact's declared type must match
- When `SchemaPath` is specified: Content is validated against JSON schema
- When `Optional == true`: Missing artifact results in empty `{{artifacts.<name>}}` substitution
- `SchemaPath` must point to a valid JSON Schema draft-07 file

**YAML Example**:
```yaml
memory:
  inject_artifacts:
    - step: analyze
      artifact: analysis-report
      as: report
      type: json
      schema_path: ./schemas/analysis-report.json
      optional: false
```

---

### 3. RuntimeArtifactsConfig (New)

**Location**: `internal/manifest/types.go` (new struct within `RuntimeConfig`)

**Purpose**: Global configuration for artifact handling, including size limits.

**Definition**:
```go
type RuntimeArtifactsConfig struct {
    MaxStdoutSize      int64  `yaml:"max_stdout_size,omitempty"` // Bytes, default 10MB
    DefaultArtifactDir string `yaml:"default_artifact_dir,omitempty"` // Default: ".wave/artifacts"
}
```

**Field Details**:

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `MaxStdoutSize` | int64 | No | 10485760 (10MB) | Maximum bytes to capture from stdout |
| `DefaultArtifactDir` | string | No | `.wave/artifacts` | Base directory for stdout artifacts |

**YAML Example**:
```yaml
runtime:
  artifacts:
    max_stdout_size: 5242880  # 5MB
    default_artifact_dir: .wave/artifacts
```

---

### 4. StdoutArtifact (Runtime Entity)

**Location**: Not persisted; exists only during execution in `PipelineExecution`

**Purpose**: In-memory representation of captured stdout before writing to disk.

**Definition**:
```go
type StdoutArtifact struct {
    Name      string
    Content   []byte
    Type      string
    StepID    string
    Size      int64
    CapturedAt time.Time
}
```

**Field Details**:

| Field | Type | Description |
|-------|------|-------------|
| `Name` | string | Artifact name from `ArtifactDef.Name` |
| `Content` | []byte | Raw stdout bytes (buffered) |
| `Type` | string | Declared type from `ArtifactDef.Type` |
| `StepID` | string | Step that produced this artifact |
| `Size` | int64 | Content size in bytes |
| `CapturedAt` | time.Time | Timestamp when capture completed |

**Lifecycle**:
1. Created during `runStepExecution()` if step has stdout artifacts
2. Populated as adapter streams output
3. On step success: Written to filesystem, registered in `ArtifactPaths`
4. On step failure: Discarded (atomicity guarantee)

---

### 5. InputValidationResult (New)

**Location**: `internal/pipeline/executor.go` or `internal/contract/`

**Purpose**: Result of input artifact validation before step execution.

**Definition**:
```go
type InputValidationResult struct {
    Passed     bool
    ArtifactRef ArtifactRef
    Error      error
    TypeMatch  bool
    SchemaValid bool
}
```

**Field Details**:

| Field | Type | Description |
|-------|------|-------------|
| `Passed` | bool | Overall validation result |
| `ArtifactRef` | ArtifactRef | The reference being validated |
| `Error` | error | Validation error if failed |
| `TypeMatch` | bool | Whether declared type matched |
| `SchemaValid` | bool | Whether schema validation passed (if applicable) |

---

## Entity Relationships

```
┌─────────────────────────────────────────────────────────────────────┐
│                           Pipeline                                   │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                         Step A                               │   │
│  │  ┌─────────────────────────────────────────────────────┐    │   │
│  │  │               output_artifacts                        │    │   │
│  │  │  ┌─────────────────────────────────────────────┐     │    │   │
│  │  │  │ ArtifactDef                                  │     │    │   │
│  │  │  │   name: "report"                             │     │    │   │
│  │  │  │   source: "stdout" ───┐                      │     │    │   │
│  │  │  │   type: "json"        │                      │     │    │   │
│  │  │  └───────────────────────│──────────────────────┘     │    │   │
│  │  └──────────────────────────│────────────────────────────┘    │   │
│  └─────────────────────────────│────────────────────────────────┘   │
│                                │                                     │
│                                ▼                                     │
│                    ┌───────────────────────┐                        │
│                    │  StdoutArtifact       │                        │
│                    │  (runtime only)       │                        │
│                    │  content: [...]       │                        │
│                    └───────────┬───────────┘                        │
│                                │ write on success                    │
│                                ▼                                     │
│                    ┌───────────────────────┐                        │
│                    │ .wave/artifacts/      │                        │
│                    │   step-a/report       │◄────────────────────┐  │
│                    └───────────────────────┘                     │  │
│                                                                   │  │
│  ┌─────────────────────────────────────────────────────────────┐ │  │
│  │                         Step B                               │ │  │
│  │  ┌─────────────────────────────────────────────────────┐    │ │  │
│  │  │               memory.inject_artifacts                 │    │ │  │
│  │  │  ┌─────────────────────────────────────────────┐     │    │ │  │
│  │  │  │ ArtifactRef                                  │     │    │ │  │
│  │  │  │   step: "step-a"                             │     │    │ │  │
│  │  │  │   artifact: "report" ────────────────────────┼─────┼────┘  │
│  │  │  │   as: "analysis"                             │     │    │   │
│  │  │  │   type: "json" ─────┐                        │     │    │   │
│  │  │  │   schema_path: ...  │                        │     │    │   │
│  │  │  └─────────────────────┼────────────────────────┘     │    │   │
│  │  └────────────────────────│──────────────────────────────┘    │   │
│  │                           │                                   │   │
│  │                           ▼                                   │   │
│  │              ┌───────────────────────────┐                    │   │
│  │              │  InputValidationResult     │                    │   │
│  │              │  (validates before exec)   │                    │   │
│  │              └───────────────────────────┘                    │   │
│  └───────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Schema Evolution

### Backward Compatibility

All changes are additive:

| Change | Compatibility | Notes |
|--------|---------------|-------|
| `ArtifactDef.Source` | Full | Defaults to "file"; existing pipelines unchanged |
| `ArtifactRef.Type` | Full | Optional; no validation if omitted |
| `ArtifactRef.SchemaPath` | Full | Optional; no schema check if omitted |
| `ArtifactRef.Optional` | Full | Defaults to false; existing fail-on-missing behavior |
| `RuntimeArtifactsConfig` | Full | Uses defaults if section absent |

### Migration Path

No migration required. Existing pipelines continue to work. New features are opt-in via new YAML fields.

---

## Persistence

| Entity | Persisted | Location | Notes |
|--------|-----------|----------|-------|
| ArtifactDef | Yes | Pipeline YAML | Part of step definition |
| ArtifactRef | Yes | Pipeline YAML | Part of step memory config |
| RuntimeArtifactsConfig | Yes | `wave.yaml` | Global runtime config |
| StdoutArtifact | No | Memory only | Transient during execution |
| Written artifact | Yes | `.wave/artifacts/<step-id>/<name>` | Filesystem |
| ArtifactPaths registry | Yes | `PipelineExecution` struct | In-memory during run |

---

## Validation Constraints

### Type Validation Matrix

| Declared Type | Allowed Content |
|---------------|-----------------|
| `json` | Valid JSON (UTF-8, parseable) |
| `text` | Valid UTF-8 text |
| `markdown` | Valid UTF-8 text (markdown syntax not validated) |
| `binary` | Any bytes (no validation) |

### Size Constraints

| Constraint | Default | Configurable | Location |
|------------|---------|--------------|----------|
| Max stdout size | 10MB | Yes | `runtime.artifacts.max_stdout_size` |
| Max artifact path length | 4096 | No | OS filesystem limit |

### Schema Constraints

- Schema files must be valid JSON Schema draft-07
- Schema path must be relative to workspace or absolute
- Schema validation timeout: 5 seconds (hardcoded)
