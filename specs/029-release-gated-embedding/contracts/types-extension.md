# API Contract: Pipeline Types Extension

**Package**: `internal/pipeline`
**File**: `types.go`

## Modified Struct

### `PipelineMetadata`

```go
type PipelineMetadata struct {
    Name        string `yaml:"name"`
    Description string `yaml:"description,omitempty"`
    Release     bool   `yaml:"release,omitempty"`
    Disabled    bool   `yaml:"disabled,omitempty"`
}
```

## Field Semantics

### `Release` field

| YAML value | Go value | Meaning |
|-----------|----------|---------|
| `release: true` | `true` | Pipeline included in `wave init` output |
| `release: false` | `false` | Pipeline excluded from `wave init` output |
| (field absent) | `false` | Pipeline excluded (explicit opt-in required) |

### `Disabled` field

| YAML value | Go value | Meaning |
|-----------|----------|---------|
| `disabled: true` | `true` | Pipeline cannot be executed at runtime |
| `disabled: false` | `false` | Pipeline can be executed |
| (field absent) | `false` | Pipeline can be executed |

### Independence

`Release` and `Disabled` are orthogonal concerns:

| `release` | `disabled` | Behavior |
|-----------|-----------|----------|
| `true` | `false` | Distributed AND executable |
| `true` | `true` | Distributed but NOT executable |
| `false` | `false` | NOT distributed but executable (development only) |
| `false` | `true` | NOT distributed, NOT executable (fully internal) |

## Validation

- `release: "yes"` or other non-boolean values: YAML unmarshalling will fail with a clear error (Go's `yaml.v3` enforces type matching)
- No cross-field validation between `release` and `disabled` (they are independent)
