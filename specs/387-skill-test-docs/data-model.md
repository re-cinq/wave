# Data Model: Skill Test Coverage & Documentation (Issue #387)

**Date**: 2026-03-14
**Feature Branch**: `387-skill-test-docs`

## Entities (Existing — No New Types)

This feature does not introduce new data types. All entities already exist. This document maps them for implementation reference.

### Skill (parse.go)
```go
type Skill struct {
    Name          string            // Required, ^[a-z0-9]([a-z0-9-]*[a-z0-9])?$, max 64
    Description   string            // Required, max 1024
    Body          string            // Markdown body, may be empty
    License       string            // Optional
    Compatibility string            // Optional, max 500
    Metadata      map[string]string // Optional key-value pairs
    AllowedTools  []string          // Parsed from space-separated string
    SourcePath    string            // Set by store.Read(), filesystem path
    ResourcePaths []string          // Discovered by discoverResources()
}
```

### DirectoryStore (store.go)
```go
type DirectoryStore struct {
    sources []SkillSource // Sorted by Precedence descending
}

type SkillSource struct {
    Root       string // Filesystem path
    Precedence int    // Higher = checked first
}
```
**CLI config**: project `.wave/skills` (precedence 2), user `~/.claude/skills` (precedence 1).

### SourceAdapter Interface (source.go)
```go
type SourceAdapter interface {
    Install(ctx context.Context, ref string, store Store) (*InstallResult, error)
    Prefix() string
}
```
**Implementations**: TesslAdapter, BMADAdapter, OpenSpecAdapter, SpecKitAdapter, GitHubAdapter, FileAdapter, URLAdapter

### SkillInfo (provision.go)
```go
type SkillInfo struct {
    Name        string
    Description string
    SourcePath  string
}
```
Returned by `ProvisionFromStore()` for each successfully provisioned skill.

### SkillConfig (types.go)
```go
type SkillConfig struct {
    Install      string `yaml:"install,omitempty"`
    Init         string `yaml:"init,omitempty"`
    Check        string `yaml:"check,omitempty"`
    CommandsGlob string `yaml:"commands_glob,omitempty"`
}
```
Declared in wave.yaml at global, persona, and pipeline scopes.

### Error Types (store.go, source.go)
```go
type ParseError struct { Field, Constraint, Value string }
type SkillError struct { SkillName, Path string; Err error }
type DiscoveryError struct { Errors []SkillError }
type DependencyError struct { Binary, Instructions string }
var ErrNotFound = errors.New("skill not found")
```

### CLI Output Types (cmd/wave/commands/skills.go)
```go
type SkillListOutput struct { Skills []SkillListItem; Warnings []string }
type SkillListItem struct { Name, Description, Source string; UsedBy []string }
type SkillInstallOutput struct { InstalledSkills []string; Source string; Warnings []string }
type SkillRemoveOutput struct { Removed, Source string }
type SkillSearchResult struct { Name, Rating, Description string }
type SkillSyncOutput struct { SyncedSkills []string; Warnings []string; Status string }
```

## Relationships

```
wave.yaml ─── declares ───> SkillConfig (at global/persona/pipeline scope)
                                │
                                ▼
ResolveSkills(global, persona, pipeline) ──> deduplicated, sorted []string
                                │
                                ▼
DirectoryStore ─── Read(name) ──> Skill (with Body, ResourcePaths)
       │                              │
       │                              ▼
       │            ProvisionFromStore(store, workspace, names) ──> []SkillInfo
       │
       ├── List() ──> []Skill (metadata-only, via ParseMetadata)
       ├── Write(skill) ──> highest-precedence source
       └── Delete(name) ──> first source containing it

SourceRouter ─── Parse(source) ──> (SourceAdapter, ref, error)
    │
    └── Install(ctx, source, store) ──> *InstallResult
          │
          └── delegates to matched SourceAdapter.Install()
```

## Test File Mapping

| Test Area | Source File | Test File | Status |
|-----------|------------|-----------|--------|
| Parse/Serialize | parse.go | store_test.go | Good (80%+), needs CRLF e2e |
| Store CRUD | store.go | store_test.go | Good (85%+), needs concurrency |
| ResolveSkills | resolve.go | resolve_test.go | Complete (100%) |
| ProvisionFromStore | provision.go (function) | provision_test.go (bottom half) | Needs resource verification, content match |
| Provisioner (commands) | skill.go | skill_test.go | Complete |
| CLI adapters | source_cli.go | source_cli_test.go | Needs mocked success path, stderr |
| Source router | source.go | source_test.go | Complete (90%+) |
| File adapter | source_file.go | source_file_test.go | Good (79%) |
| GitHub adapter | source_github.go | source_github_test.go | Good (60%), gap-fill only |
| URL adapter | source_url.go | source_url_test.go | Good (70%), gap-fill only |
| Validation | validate.go | validate_test.go | Complete (100%) |
| CLI commands | skills.go | skills_test.go | Needs help output, search/sync parse tests |
| Documentation | N/A | N/A | All 3 guides needed |
