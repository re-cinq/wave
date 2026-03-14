# Data Model: Wave Skills CLI

**Feature**: `wave skills` CLI command
**Date**: 2026-03-14

## Existing Entities (from `internal/skill/`)

### Skill (parse.go:13)
The core domain entity — a parsed SKILL.md file.
```go
type Skill struct {
    Name          string
    Description   string
    Body          string
    License       string
    Compatibility string
    Metadata      map[string]string
    AllowedTools  []string
    SourcePath    string            // populated by Store.Read/List
    ResourcePaths []string          // populated by Store.Read
}
```

### DirectoryStore (store.go:72)
Multi-source CRUD store backed by filesystem directories.
```go
type DirectoryStore struct {
    sources []SkillSource
}
// Methods: Read, Write, List, Delete
```

### SkillSource (store.go:66)
A directory root with precedence for multi-source resolution.
```go
type SkillSource struct {
    Root       string
    Precedence int
}
```

### SourceRouter (source.go:54)
Dispatches source strings to adapters by prefix.
```go
type SourceRouter struct {
    adapters map[string]SourceAdapter
}
// Methods: Parse, Install, Prefixes
```

### InstallResult (source.go:25)
Outcome of a source adapter installation.
```go
type InstallResult struct {
    Skills   []Skill
    Warnings []string
}
```

### DependencyError (source.go:37)
CLI tool not found error.
```go
type DependencyError struct {
    Binary       string
    Instructions string
}
```

## New Entities (in `cmd/wave/commands/`)

### SkillListItem (CLI output struct)
Represents one skill in `wave skills list` output.
```go
type SkillListItem struct {
    Name        string   `json:"name"`
    Description string   `json:"description"`
    Source      string   `json:"source"`
    UsedBy      []string `json:"used_by,omitempty"`
}
```

### SkillListOutput (CLI output struct)
Top-level output for `wave skills list`.
```go
type SkillListOutput struct {
    Skills   []SkillListItem `json:"skills"`
    Warnings []string        `json:"warnings,omitempty"`
}
```

### SkillInstallOutput (CLI output struct)
Output for `wave skills install`.
```go
type SkillInstallOutput struct {
    InstalledSkills []string `json:"installed_skills"`
    Source          string   `json:"source"`
    Warnings        []string `json:"warnings,omitempty"`
}
```

### SkillRemoveOutput (CLI output struct)
Output for `wave skills remove`.
```go
type SkillRemoveOutput struct {
    Removed string `json:"removed"`
    Source  string `json:"source"`
}
```

### SkillSearchResult (CLI output struct)
One search result item.
```go
type SkillSearchResult struct {
    Name        string `json:"name"`
    Rating      string `json:"rating,omitempty"`
    Description string `json:"description"`
}
```

### SkillSyncOutput (CLI output struct)
Output for `wave skills sync`.
```go
type SkillSyncOutput struct {
    SyncedSkills []string `json:"synced_skills"`
    Warnings     []string `json:"warnings,omitempty"`
    Status       string   `json:"status"`
}
```

## Entity Relationships

```
SourceRouter ──dispatch──> SourceAdapter ──install──> DirectoryStore ──write──> Skill
                                                      DirectoryStore ──read───> Skill
                                                      DirectoryStore ──list───> []Skill
                                                      DirectoryStore ──delete─> (removes)

wave skills list    → DirectoryStore.List() + collectSkillPipelineUsage() → SkillListOutput
wave skills install → SourceRouter.Install()                              → SkillInstallOutput
wave skills remove  → DirectoryStore.Delete()                             → SkillRemoveOutput
wave skills search  → exec tessl search                                   → []SkillSearchResult
wave skills sync    → exec tessl install --project-dependencies           → SkillSyncOutput
```

## Error Codes (new constants in errors.go)

| Code                       | Trigger                                    | Suggestion                                    |
|----------------------------|--------------------------------------------|-----------------------------------------------|
| `skill_not_found`          | `DirectoryStore.Delete` returns ErrNotFound | List installed skills, check name              |
| `skill_source_error`       | `SourceRouter.Parse` fails, adapter error  | Show recognized prefixes                       |
| `skill_dependency_missing` | `*DependencyError` from adapter            | Show install instructions for missing binary   |
