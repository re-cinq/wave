# Data Model: Hierarchical Skill Configuration

**Feature**: #385 — Skill Hierarchy Config
**Date**: 2026-03-14

## Entity Changes

### Modified: `manifest.Manifest` (internal/manifest/types.go)

```go
type Manifest struct {
    APIVersion  string              `yaml:"apiVersion"`
    Kind        string              `yaml:"kind"`
    Metadata    Metadata            `yaml:"metadata"`
    Project     *Project            `yaml:"project,omitempty"`
    Adapters    map[string]Adapter  `yaml:"adapters,omitempty"`
    Personas    map[string]Persona  `yaml:"personas,omitempty"`
    Runtime     Runtime             `yaml:"runtime"`
    Skills      []string            `yaml:"skills,omitempty"`       // NEW: global skill references
}
```

**Change**: Add `Skills []string` field. YAML tag `skills,omitempty` ensures absent/null/empty
all parse cleanly to `nil` slice with no error (FR-010).

### Modified: `manifest.Persona` (internal/manifest/types.go)

```go
type Persona struct {
    Adapter          string          `yaml:"adapter"`
    Description      string          `yaml:"description,omitempty"`
    SystemPromptFile string          `yaml:"system_prompt_file"`
    Temperature      float64         `yaml:"temperature,omitempty"`
    Model            string          `yaml:"model,omitempty"`
    Permissions      Permissions     `yaml:"permissions,omitempty"`
    Hooks            HookConfig      `yaml:"hooks,omitempty"`
    Sandbox          *PersonaSandbox `yaml:"sandbox,omitempty"`
    Skills           []string        `yaml:"skills,omitempty"`      // NEW: persona skill references
}
```

**Change**: Add `Skills []string` field. Same `omitempty` semantics.

### Modified: `pipeline.Pipeline` (internal/pipeline/types.go)

```go
type Pipeline struct {
    Kind     string           `yaml:"kind"`
    Metadata PipelineMetadata `yaml:"metadata"`
    Requires *Requires        `yaml:"requires,omitempty"`
    Input    InputConfig      `yaml:"input"`
    Steps           []Step                       `yaml:"steps"`
    PipelineOutputs map[string]PipelineOutput    `yaml:"pipeline_outputs,omitempty"`
    ChatContext     *ChatContextConfig           `yaml:"chat_context,omitempty"`
    Skills          []string                     `yaml:"skills,omitempty"`        // NEW: pipeline skill references
}
```

**Change**: Add `Skills []string` field at top level of `Pipeline` struct (not under
`Requires`). Per C1 resolution: `Requires` holds operational metadata, `Skills` is
declarative intent.

## New: Skill Resolution Function

**File**: `internal/skill/resolve.go`

```go
// ResolveSkills merges skill references from global, persona, and pipeline scopes
// into a single deduplicated, sorted list. Pipeline > Persona > Global precedence
// (higher-precedence entries appear first in the dedup pass, but final output is
// alphabetically sorted).
func ResolveSkills(global, persona, pipeline []string) []string
```

**Behavior**:
1. Iterate pipeline skills first (highest precedence), then persona, then global.
2. Track seen names in a `map[string]bool` for deduplication.
3. Collect unique names into a result slice.
4. Sort result alphabetically for determinism (SC-005).
5. Return the sorted slice.

**Note**: `requires.skills` keys are included by the caller — the executor extracts
them from `Requires.SkillNames()` and appends to the pipeline slice before calling
`ResolveSkills`.

## New: Skill Validation Functions

**File**: `internal/skill/validate.go`

```go
// ValidateSkillRefs validates a list of skill name references:
// 1. Name format validation via ValidateName()
// 2. Existence check against the provided Store (if non-nil)
// Returns all errors aggregated (FR-011), each annotated with scope context.
func ValidateSkillRefs(names []string, scope string, store Store) []error
```

**Parameters**:
- `names`: The skill name strings to validate.
- `scope`: Human-readable scope label for error messages (e.g., "global", "persona:planner", "pipeline:implement").
- `store`: The DirectoryStore for existence checks. If `nil`, only format validation runs.

**Returns**: Aggregated error slice — all invalid names across the input, not fail-fast.

## YAML Examples

### wave.yaml (global + persona)

```yaml
apiVersion: v1
kind: WaveManifest
metadata:
  name: my-project
skills:                      # Global defaults
  - speckit
  - lint-rules
personas:
  planner:
    adapter: claude
    system_prompt_file: .wave/personas/planner.md
    skills:                  # Persona-specific
      - speckit              # Deduplicated with global
      - architecture-guide
  craftsman:
    adapter: claude
    system_prompt_file: .wave/personas/craftsman.md
    # No skills: field — inherits only global skills
```

### Pipeline YAML (.wave/pipelines/implement.yaml)

```yaml
kind: WavePipeline
metadata:
  name: implement
skills:                      # Pipeline-level
  - golang
  - testing
requires:
  skills:
    speckit:                 # SkillConfig with install/check/init
      check: specify check
      install: uv tool install specify-cli
```

**Resolution for a step using `planner` persona**:
- Global: `[speckit, lint-rules]`
- Persona (planner): `[speckit, architecture-guide]`
- Pipeline: `[golang, testing]` + requires.skills keys: `[speckit]`
- Resolved: `[architecture-guide, golang, lint-rules, speckit, testing]` (sorted, deduplicated)

## Validation Flow

```
┌─────────────────────────────────────────────────────────────┐
│ Manifest Load (manifest.ValidateWithFile)                   │
│   ├── ValidateSkillRefs(manifest.Skills, "global", store)   │
│   └── for name, persona := range manifest.Personas:         │
│         ValidateSkillRefs(persona.Skills,                    │
│                           "persona:"+name, store)           │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ Pipeline Load (pipeline.ValidateDAG or new validation)      │
│   └── ValidateSkillRefs(pipeline.Skills, "pipeline:"+name,  │
│                          store)                             │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ Step Execution (executor.buildAdapterRunConfig)             │
│   └── ResolveSkills(                                        │
│           manifest.Skills,                                  │
│           persona.Skills,                                   │
│           append(pipeline.Skills, requires.SkillNames()...) │
│       )                                                     │
│   └── Provision resolved skills into workspace              │
└─────────────────────────────────────────────────────────────┘
```
