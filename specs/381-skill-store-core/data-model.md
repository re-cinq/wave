# Data Model: Skill Store Core

**Feature**: #381 — Skill Store Core
**Date**: 2026-03-13

## Entity Diagram

```
┌─────────────────────────────────────┐
│              Store                  │  ← Interface
│─────────────────────────────────────│
│  Read(name string) (Skill, error)   │
│  Write(skill Skill) error           │
│  List() ([]Skill, error)            │
│  Delete(name string) error          │
└───────────────┬─────────────────────┘
                │ implements
┌───────────────▼─────────────────────┐
│         DirectoryStore              │  ← Concrete struct
│─────────────────────────────────────│
│  sources  []SkillSource             │  ordered by precedence (highest first)
│─────────────────────────────────────│
│  Read(name string) (Skill, error)   │  checks sources in order, returns first match
│  Write(skill Skill) error           │  writes to first (highest-precedence) source
│  List() ([]Skill, error)            │  merges all sources, first-name-wins dedup
│  Delete(name string) error          │  deletes from first source containing name
└─────────────────────────────────────┘

┌─────────────────────────────────────┐
│           SkillSource               │  ← Value type
│─────────────────────────────────────│
│  Root       string                  │  absolute or relative directory path
│  Precedence int                     │  higher = wins on conflict
└─────────────────────────────────────┘

┌─────────────────────────────────────┐
│              Skill                  │  ← Domain entity
│─────────────────────────────────────│
│  Name          string               │  ^[a-z0-9]([a-z0-9-]*[a-z0-9])?$, max 64
│  Description   string               │  non-empty, max 1024 chars
│  Body          string               │  markdown content (may be empty)
│  License       string               │  optional
│  Compatibility string               │  optional, max 500 chars
│  Metadata      map[string]string    │  optional key-value pairs
│  AllowedTools  []string             │  parsed from space-delimited string
│  SourcePath    string               │  directory path where loaded from
│  ResourcePaths []string             │  paths to files in scripts/, references/, assets/
└─────────────────────────────────────┘

┌─────────────────────────────────────┐
│           ParseError                │  ← Error type
│─────────────────────────────────────│
│  Field      string                  │  which field failed ("name", "description", etc.)
│  Constraint string                  │  what constraint was violated
│  Value      string                  │  the actual value that failed
│─────────────────────────────────────│
│  Error() string                     │
│  Unwrap() error                     │
└─────────────────────────────────────┘

┌─────────────────────────────────────┐
│        DiscoveryError               │  ← Aggregate error type
│─────────────────────────────────────│
│  Errors   []SkillError              │  per-skill parse/validation failures
│─────────────────────────────────────│
│  Error() string                     │  summary of all failures
│  Unwrap() error                     │  nil (aggregate, use Errors field)
└─────────────────────────────────────┘

┌─────────────────────────────────────┐
│          SkillError                 │  ← Per-skill error wrapper
│─────────────────────────────────────│
│  SkillName string                   │  which skill failed
│  Path      string                   │  filesystem path to the skill directory
│  Err       error                    │  underlying error (ParseError or os error)
│─────────────────────────────────────│
│  Error() string                     │
│  Unwrap() error                     │
└─────────────────────────────────────┘
```

## Type Definitions (Go)

### Skill (domain entity)

```go
// Skill represents a parsed SKILL.md file per the Agent Skills Specification.
type Skill struct {
    Name          string            `yaml:"name"`
    Description   string            `yaml:"description"`
    License       string            `yaml:"license,omitempty"`
    Compatibility string            `yaml:"compatibility,omitempty"`
    Metadata      map[string]string `yaml:"metadata,omitempty"`
    AllowedTools  []string          `yaml:"-"` // parsed from space-delimited "allowed-tools" string
    Body          string            `yaml:"-"` // markdown content after frontmatter
    SourcePath    string            `yaml:"-"` // directory where loaded from
    ResourcePaths []string          `yaml:"-"` // discovered resource files
}
```

**YAML mapping note**: `AllowedTools` uses `yaml:"-"` because it needs custom parsing from the `allowed-tools` YAML string field. A raw `AllowedToolsRaw` field handles the YAML deserialization, and a post-parse step splits it into the slice.

### Frontmatter (internal parse helper)

```go
// frontmatter is the raw YAML structure for SKILL.md frontmatter.
// It maps YAML field names (with hyphens) to Go struct fields.
type frontmatter struct {
    Name          string            `yaml:"name"`
    Description   string            `yaml:"description"`
    License       string            `yaml:"license,omitempty"`
    Compatibility string            `yaml:"compatibility,omitempty"`
    Metadata      map[string]string `yaml:"metadata,omitempty"`
    AllowedTools  string            `yaml:"allowed-tools,omitempty"`
}
```

### Store (interface)

```go
// Store defines CRUD operations for skill management.
type Store interface {
    // Read returns a skill by name with full content (body + resources).
    Read(name string) (Skill, error)

    // Write persists a skill to the filesystem. Creates directory if needed.
    Write(skill Skill) error

    // List returns all skills with metadata-only loading (name + description).
    // Returns valid skills and a non-nil error if some skills failed to parse.
    List() ([]Skill, error)

    // Delete removes a skill by name from the store.
    Delete(name string) error
}
```

### Error Types

```go
// ParseError represents a SKILL.md parsing or validation failure.
type ParseError struct {
    Field      string // field name that failed validation
    Constraint string // constraint that was violated
    Value      string // actual value (sanitized for security)
}

// DiscoveryError is returned when List encounters per-skill failures.
// It contains the individual errors; valid skills are returned alongside.
type DiscoveryError struct {
    Errors []SkillError
}

// SkillError wraps an error with the skill name and path context.
type SkillError struct {
    SkillName string
    Path      string
    Err       error
}
```

**Note**: `SkillError` here is a different type from `preflight.SkillError`. The preflight type represents missing skill dependencies for pipeline execution; this type represents per-skill parse/validation failures in the store. They live in separate packages and serve different domains.

## Filesystem Layout

```
.wave/skills/                    # Project-local skill source (highest precedence)
├── my-skill/
│   ├── SKILL.md                 # Required: frontmatter + body
│   ├── scripts/                 # Optional: executable scripts
│   │   └── setup.sh
│   ├── references/              # Optional: reference documents
│   │   └── api-spec.json
│   └── assets/                  # Optional: static assets
│       └── template.txt
└── another-skill/
    └── SKILL.md

.claude/skills/                  # User-level skill source (lower precedence)
├── golang/
│   └── SKILL.md
└── speckit/
    └── SKILL.md
```

## Validation Rules

| Field | Constraint | Error |
|-------|-----------|-------|
| `name` | Required, matches `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`, max 64 chars | `ParseError{Field:"name", Constraint:"..."}` |
| `name` | Must match parent directory name (when loaded from disk) | `ParseError{Field:"name", Constraint:"must match directory name"}` |
| `description` | Required, non-empty, max 1024 chars | `ParseError{Field:"description", Constraint:"..."}` |
| `compatibility` | Optional, max 500 chars if present | `ParseError{Field:"compatibility", Constraint:"..."}` |
| frontmatter | Must have opening and closing `---` delimiters | `ParseError{Field:"frontmatter", Constraint:"..."}` |
| `name` (CRUD ops) | Must not contain `/`, `\`, `..`, path separators | `ParseError{Field:"name", Constraint:"invalid characters"}` |

## Relationships to Existing Types

```
internal/skill/
├── types.go          # EXISTING: SkillConfig (legacy provisioning config)
├── skill.go          # EXISTING: Provisioner (legacy command-file copying)
├── skill_test.go     # EXISTING: Provisioner tests (must remain unchanged)
├── store.go          # NEW: Store interface + DirectoryStore
├── parse.go          # NEW: SKILL.md parser (frontmatter + body)
├── errors.go         # NEW: ParseError, DiscoveryError, SkillError
└── store_test.go     # NEW: Store and parser tests
```

No modifications to existing files. The new `Skill` type is distinct from `SkillConfig`:
- `SkillConfig` = how to install/check/glob a skill (pipeline dependency)
- `Skill` = parsed SKILL.md content (skill definition itself)
