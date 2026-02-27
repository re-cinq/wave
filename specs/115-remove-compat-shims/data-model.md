# Data Model: Remove Backwards-Compatibility Shims

**Feature**: 115-remove-compat-shims
**Date**: 2026-02-20

## Overview

This feature does not add new entities. It simplifies existing data models by removing deprecated fields, dead code paths, and legacy fallbacks. The changes below document the **before** and **after** state of each affected entity.

---

## Entity: ContractConfig

**Package**: `internal/contract`
**File**: `contract.go`

### Before
```go
type ContractConfig struct {
    Type        string   `json:"type"`
    Source      string   `json:"source,omitempty"`
    Schema      string   `json:"schema,omitempty"`
    SchemaPath  string   `json:"schemaPath,omitempty"`
    Command     string   `json:"command,omitempty"`
    CommandArgs []string `json:"commandArgs,omitempty"`
    Dir         string   `json:"dir,omitempty"`
    StrictMode  bool     `json:"strictMode,omitempty"`  // <-- REMOVED
    MustPass    bool     `json:"must_pass,omitempty"`
    MaxRetries  int      `json:"maxRetries,omitempty"`
    // ... remaining fields unchanged
}
```

### After
```go
type ContractConfig struct {
    Type        string   `json:"type"`
    Source      string   `json:"source,omitempty"`
    Schema      string   `json:"schema,omitempty"`
    SchemaPath  string   `json:"schemaPath,omitempty"`
    Command     string   `json:"command,omitempty"`
    CommandArgs []string `json:"commandArgs,omitempty"`
    Dir         string   `json:"dir,omitempty"`
    MustPass    bool     `json:"must_pass,omitempty"`    // <-- Only field for strictness
    MaxRetries  int      `json:"maxRetries,omitempty"`
    // ... remaining fields unchanged
}
```

### Impact
- All consumers that set `StrictMode` must be updated to set `MustPass` instead
- All consumers that read `StrictMode` must read `MustPass` instead
- The `json:"strictMode"` tag is removed, so any JSON/YAML with `strictMode` key will be silently ignored (no runtime error)

---

## Entity: Migration

**Package**: `internal/state`
**File**: `migration_definitions.go`

### Before
```go
Migration{
    Version: 1,
    Description: "...",
    Up: `CREATE TABLE ...`,
    Down: `DROP TABLE ...`,   // <-- Contains rollback SQL
}
```

### After
```go
Migration{
    Version: 1,
    Description: "...",
    Up: `CREATE TABLE ...`,
    Down: "",                 // <-- Empty string, no rollback
}
```

### Impact
- The `Down` field remains on the `Migration` struct (no type change)
- All 6 migration `Down` values become `""`
- `MigrateDown()` at `migrations.go:269` returns `"migration N has no rollback script"` for empty Down — this is the desired behavior
- Migration checksums will change (they include both Up and Down content)

---

## Entity: StateStore initialization

**Package**: `internal/state`
**File**: `store.go`

### Before
- Dual-path initialization: migration system OR legacy `schema.sql` via `go:embed`
- `WAVE_MIGRATION_ENABLED=false` triggers `schema.sql` fallback

### After
- Single-path initialization: migration system only
- `WAVE_MIGRATION_ENABLED=false` returns an error
- `schema.sql` file deleted
- `go:embed` directive and `schemaFS` variable removed
- `"embed"` import removed

---

## Entity: PipelineGenerationResult extraction

**Package**: `internal/pipeline`
**File**: `meta.go`

### Before
- `extractPipelineAndSchemas()` falls back to `extractYAMLLegacy()` when `--- PIPELINE ---` marker is absent

### After
- `extractPipelineAndSchemas()` returns error when `--- PIPELINE ---` marker is absent
- `extractYAMLLegacy()` function deleted

---

## Entity: JSONCleaner extraction

**Package**: `internal/contract`
**File**: `json_cleaner.go`

### Before
- `ExtractJSONFromText()` falls back to `extractJSONFromTextLegacy()` when recovery parser fails

### After
- `ExtractJSONFromText()` returns the recovery parser error directly
- `extractJSONFromTextLegacy()` method deleted

---

## Entity: Resume workspace lookup

**Package**: `internal/pipeline`
**File**: `resume.go`

### Before
- `loadResumeState()` checks for exact-name directory (no hash suffix) as legacy fallback

### After
- Only checks for `<name>-<timestamp>-<hash>` pattern directories

---

## Removed Functions

| Function | File | Reason |
|----------|------|--------|
| `IsTypeScriptAvailable()` | `internal/contract/typescript.go:95-100` | Backwards-compat wrapper for `CheckTypeScriptAvailability()` |
| `extractJSONFromTextLegacy()` | `internal/contract/json_cleaner.go:83-148` | Legacy fallback for JSON extraction |
| `extractYAMLLegacy()` | `internal/pipeline/meta.go:604-630` | Legacy fallback for YAML extraction |

## Removed Files

| File | Reason |
|------|--------|
| `internal/state/schema.sql` | Legacy schema initialization replaced by migration system |
