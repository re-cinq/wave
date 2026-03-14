# Data Model: Ecosystem Adapters for Skill Sources

**Feature**: #383 — Skill source adapters with prefix routing
**Date**: 2026-03-14

## Core Entities

### SourceAdapter (Interface)

The primary extension point. Each adapter handles one source prefix and implements the full lifecycle: dependency check → fetch → parse → validate → write to store.

```go
// SourceAdapter handles installation from a specific source type.
type SourceAdapter interface {
    // Install fetches and installs skills from the given reference into the store.
    // ctx carries timeout deadlines. ref is the adapter-specific locator
    // (everything after the prefix). store is the target for installed skills.
    Install(ctx context.Context, ref string, store Store) (*InstallResult, error)

    // Prefix returns the source prefix this adapter handles (e.g., "tessl", "github", "https://").
    Prefix() string
}
```

**Relationships**: Registered in `SourceRouter`. Uses `Store` for writing. Returns `InstallResult`.

### SourceRouter

Registry and dispatcher. Parses source strings, selects the correct adapter, delegates installation.

```go
// SourceRouter dispatches source strings to the appropriate adapter.
type SourceRouter struct {
    adapters map[string]SourceAdapter
}

// NewSourceRouter creates a router with the given adapters registered.
func NewSourceRouter(adapters ...SourceAdapter) *SourceRouter

// Register adds an adapter to the router.
func (r *SourceRouter) Register(adapter SourceAdapter)

// Parse splits a source string into prefix and reference.
// Returns the matched adapter and the reference string.
func (r *SourceRouter) Parse(source string) (SourceAdapter, string, error)

// Install parses the source string and delegates to the matched adapter.
func (r *SourceRouter) Install(ctx context.Context, source string, store Store) (*InstallResult, error)

// Prefixes returns all registered prefix strings (for error messages).
func (r *SourceRouter) Prefixes() []string
```

**Relationships**: Owns `map[string]SourceAdapter`. Created with pre-registered adapters.

### SourceReference

A parsed source string — the result of splitting `prefix:reference`.

```go
// SourceReference represents a parsed source string.
type SourceReference struct {
    Prefix    string // e.g., "tessl", "github", "file", "https://"
    Reference string // e.g., "github/spec-kit", "owner/repo", "./local/path"
    Raw       string // Original source string
}
```

**Relationships**: Produced by `SourceRouter.Parse()`. Consumed by adapters.

### InstallResult

The outcome of an adapter invocation. Contains successfully installed skills and any warnings.

```go
// InstallResult represents the outcome of a source adapter installation.
type InstallResult struct {
    Skills   []Skill  // Successfully installed skills
    Warnings []string // Non-fatal warnings (e.g., "skipped duplicate skill")
}
```

**Relationships**: Returned by `SourceAdapter.Install()`. Contains existing `Skill` type.

### DependencyError

A structured error for missing CLI dependencies. Provides actionable guidance.

```go
// DependencyError indicates a required CLI tool is not installed.
type DependencyError struct {
    Binary       string // Tool binary name (e.g., "tessl", "git")
    Instructions string // Install instructions (e.g., "npm i -g @tessl/cli")
}

func (e *DependencyError) Error() string
```

**Relationships**: Returned by CLI adapters when `exec.LookPath` fails.

### CLIDependency

Describes an external CLI tool required by an adapter.

```go
// CLIDependency describes an external CLI tool required by an adapter.
type CLIDependency struct {
    Binary       string // Binary name to look up on PATH
    Instructions string // Human-readable install instructions
}
```

**Relationships**: Embedded in each CLI-based adapter.

## Concrete Adapters

### TesslAdapter

```go
type TesslAdapter struct {
    dep CLIDependency // {Binary: "tessl", Instructions: "npm i -g @tessl/cli"}
}
```

**Behavior**: `tessl install <ref>` → discover SKILL.md files → parse → write to store.

### BMADAdapter

```go
type BMADAdapter struct {
    dep CLIDependency // {Binary: "npx", Instructions: "npm i -g npx (comes with npm)"}
}
```

**Behavior**: `npx bmad-method install --tools claude-code --yes` → discover SKILL.md files → write to store.

### OpenSpecAdapter

```go
type OpenSpecAdapter struct {
    dep CLIDependency // {Binary: "openspec", Instructions: "npm i -g @openspec/cli"}
}
```

**Behavior**: `openspec init` → discover skill files → write to store.

### SpecKitAdapter

```go
type SpecKitAdapter struct {
    dep CLIDependency // {Binary: "specify", Instructions: "npm i -g @speckit/cli"}
}
```

**Behavior**: `specify init` → discover skill files → write to store.

### GitHubAdapter

```go
type GitHubAdapter struct {
    dep CLIDependency // {Binary: "git", Instructions: "install git from https://git-scm.com"}
}
```

**Behavior**: `git clone --depth 1 <url>` → navigate to optional path → discover SKILL.md files → write to store.

**Reference parsing**: `owner/repo[/path/to/skill]` — split on `/`, first two components form the GitHub URL, remainder is the subdirectory path.

### FileAdapter

```go
type FileAdapter struct {
    projectRoot string // Base directory for path containment validation
}
```

**Behavior**: Resolve path (relative to projectRoot or absolute) → validate containment → validate no symlinks → copy SKILL.md → write to store.

**Security**: Uses same `containedPath` pattern as `DirectoryStore`. Rejects symlinks, path traversal, escape beyond project root.

### URLAdapter

```go
type URLAdapter struct {
    client *http.Client // Configured with 30s header timeout
}
```

**Behavior**: HTTP GET → detect archive format by extension → extract to temp dir → discover SKILL.md files → write to store.

**Supported formats**: `.tar.gz`, `.tgz`, `.zip`.

## Existing Entities (Unchanged)

These entities from the current codebase are used by the new system without modification:

- **`Store` (interface)**: `Read`, `Write`, `List`, `Delete` operations. Source adapters use `Write` to persist installed skills.
- **`DirectoryStore`**: Filesystem-backed store implementation. No changes needed.
- **`Skill`**: Parsed SKILL.md representation. Source adapters produce these via `Parse()`.
- **`SkillConfig`**: Manifest-level skill declarations. Coexists with source adapters (see spec C-002).
- **`Parse()` / `ParseMetadata()`**: SKILL.md parser. Used by adapters to validate extracted content.
- **`Serialize()`**: SKILL.md serializer. Not directly used by adapters (they use `store.Write()` which calls it internally).

## Entity Relationship Diagram

```
                    ┌──────────────┐
                    │ SourceRouter │
                    │              │
                    │ adapters map │
                    └──────┬───────┘
                           │ dispatches to
          ┌────────────────┼────────────────┐
          │                │                │
          ▼                ▼                ▼
  ┌───────────────┐ ┌────────────┐ ┌──────────────┐
  │ TesslAdapter  │ │ FileAdapter│ │ GitHubAdapter │  ... (7 adapters)
  │               │ │            │ │              │
  │ CLIDependency │ │ projectRoot│ │ CLIDependency│
  └───────┬───────┘ └─────┬──────┘ └──────┬───────┘
          │               │               │
          └───────────┬───┘───────────────┘
                      │ Install() returns
                      ▼
              ┌───────────────┐
              │ InstallResult │
              │               │
              │ Skills []Skill│
              │ Warnings []   │
              └───────┬───────┘
                      │ writes via
                      ▼
               ┌────────────┐
               │   Store    │ (interface)
               │            │
               │ Write()    │
               └────────────┘
```

## Timeout Configuration

All timeouts are constants (not configurable in v1):

| Context | Timeout | Implementation |
|---------|---------|---------------|
| CLI subprocess | 2 min | `context.WithTimeout` on `exec.CommandContext` |
| Git clone | 2 min | `context.WithTimeout` on `exec.CommandContext` |
| HTTP download | 2 min overall | `context.WithTimeout` on request context |
| HTTP response headers | 30 sec | `http.Transport.ResponseHeaderTimeout` |
