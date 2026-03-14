# Implementation Plan: Ecosystem Adapters for Skill Sources

**Branch**: `383-skill-source-adapters` | **Date**: 2026-03-14 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/383-skill-source-adapters/spec.md`

## Summary

Implement a source adapter system for the skill package that enables installing skills from 7 source prefixes: `tessl:`, `bmad:`, `openspec:`, `speckit:`, `github:`, `file:`, `https://`. Each prefix maps to a concrete adapter implementing a common `SourceAdapter` interface. A `SourceRouter` parses source strings and dispatches to the correct adapter. External CLI tools (tessl, npx, openspec, specify, git) are soft dependencies — missing tools produce actionable error messages with install instructions. All code lives within `internal/skill/` alongside the existing store and parser.

## Technical Context

**Language/Version**: Go 1.25+ (matches project)
**Primary Dependencies**: Standard library only (`os/exec`, `net/http`, `archive/tar`, `archive/zip`, `compress/gzip`, `context`, `path/filepath`)
**Storage**: Filesystem via existing `skill.Store` interface / `DirectoryStore`
**Testing**: `go test -race ./internal/skill/...` — table-driven tests with mocked `exec.LookPath` and mocked store
**Target Platform**: Linux (primary), macOS (secondary)
**Project Type**: Single Go module — extends existing `internal/skill/` package
**Performance Goals**: N/A — adapter operations are I/O-bound (subprocess, network, filesystem)
**Constraints**: 2-minute timeout on subprocess/network ops, 30-second HTTP header timeout (FR-016, C-004)
**Scale/Scope**: 7 adapters, ~8 new source files, ~1200 lines of Go code + tests

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | PASS | No new runtime dependencies. CLI tools (tessl, git, npx) are soft dependencies — Wave functions without them. Standard library only for archive handling. |
| P2: Manifest as Source of Truth | PASS | Source adapters are infrastructure. A future manifest `source:` field may reference them, but this feature doesn't modify `wave.yaml` schema. |
| P3: Persona-Scoped Execution | N/A | Source adapters are not persona-related. |
| P4: Fresh Memory at Step Boundary | N/A | Not a pipeline step change. |
| P5: Navigator-First Architecture | N/A | Not a pipeline change. |
| P6: Contracts at Every Handover | N/A | Not a pipeline change. |
| P7: Relay via Dedicated Summarizer | N/A | Not a pipeline change. |
| P8: Ephemeral Workspaces | PASS | Adapters use `os.MkdirTemp` for scratch work, cleaned up via `defer os.RemoveAll`. |
| P9: Credentials Never Touch Disk | PASS | Git uses system credentials via environment. No credentials stored. |
| P10: Observable Progress | N/A | Source adapters are library code, not pipeline steps. Callers can log as needed. |
| P11: Bounded Recursion | N/A | No recursion in adapter design. |
| P12: Minimal Step State Machine | N/A | Not a pipeline step change. |
| P13: Test Ownership | PASS | All new code includes comprehensive tests. `go test -race ./internal/skill/...` must pass. |

**Post-Phase 1 Re-check**: All principles remain PASS/N/A. No violations introduced.

## Project Structure

### Documentation (this feature)

```
specs/383-skill-source-adapters/
├── plan.md              # This file
├── research.md          # Phase 0 output — technology decisions
├── data-model.md        # Phase 1 output — entity definitions
├── contracts/           # Phase 1 output — API contracts
│   └── source-adapter.go
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```
internal/skill/
├── source.go              # SourceAdapter interface, SourceRouter, SourceReference, InstallResult, error types
├── source_test.go         # Router unit tests, parsing tests, error formatting tests
├── source_cli.go          # CLI-based adapters: TesslAdapter, BMADAdapter, OpenSpecAdapter, SpecKitAdapter
├── source_cli_test.go     # CLI adapter tests (mocked exec, mocked store)
├── source_github.go       # GitHubAdapter — git clone, path parsing, SKILL.md discovery
├── source_github_test.go  # GitHub adapter tests (mocked git, mocked store)
├── source_file.go         # FileAdapter — local path resolution, containment, copy
├── source_file_test.go    # File adapter tests (path traversal, symlinks, containment)
├── source_url.go          # URLAdapter — HTTP download, archive extraction (tar.gz, zip)
├── source_url_test.go     # URL adapter tests (mocked HTTP, archive formats)
├── parse.go               # (existing) SKILL.md parser — used by adapters
├── store.go               # (existing) Store interface and DirectoryStore — used by adapters
├── skill.go               # (existing) Provisioner — unchanged
└── types.go               # (existing) SkillConfig — unchanged
```

**Structure Decision**: All source adapter code is added to the existing `internal/skill/` package as new files. This avoids circular dependencies and provides direct access to `Store`, `Skill`, `Parse()`, `ValidateName()`, and `containedPath()`. No new packages or subpackages.

## Implementation Approach

### Layer 1: Core Types and Router (`source.go`)

1. Define `SourceAdapter` interface with `Install(ctx, ref, store)` and `Prefix()` methods
2. Define `SourceReference` struct (prefix, reference, raw source string)
3. Define `InstallResult` struct (skills, warnings)
4. Define `DependencyError` and `CLIDependency` types
5. Implement `SourceRouter` with:
   - `NewSourceRouter(adapters ...SourceAdapter)` constructor
   - `Register(adapter)` for dynamic registration
   - `Parse(source)` with URL-scheme-first parsing (check `https://` / `http://` before colon split)
   - `Install(ctx, source, store)` convenience method
   - `Prefixes()` for error messages

### Layer 2: CLI Adapters (`source_cli.go`)

Shared pattern for all 4 CLI adapters (tessl, bmad, openspec, speckit):
1. Check `exec.LookPath(dep.Binary)` → return `DependencyError` if missing
2. Create temp dir via `os.MkdirTemp`
3. Run CLI command via `exec.CommandContext(ctx, binary, args...)` in temp dir
4. Walk temp dir to discover `SKILL.md` files
5. Parse each via `skill.Parse()`, validate, write to store
6. Clean up temp dir

Each adapter has its specific command and arguments:
- **TesslAdapter**: `tessl install <ref>`
- **BMADAdapter**: `npx bmad-method install --tools claude-code --yes`
- **OpenSpecAdapter**: `openspec init`
- **SpecKitAdapter**: `specify init`

### Layer 3: GitHub Adapter (`source_github.go`)

1. Check `exec.LookPath("git")`
2. Parse reference: `owner/repo[/path/to/skill]`
3. Construct clone URL: `https://github.com/<owner>/<repo>.git`
4. `git clone --depth 1` into temp dir
5. If path suffix: navigate to subdirectory, find SKILL.md
6. If no path: check root for SKILL.md; if not found, walk subdirectories
7. Parse, validate, write to store

### Layer 4: File Adapter (`source_file.go`)

1. Resolve path relative to project root (or absolute)
2. Validate path containment using `containedPath()` pattern from store.go
3. Reject symlinks via `os.Lstat`
4. Read SKILL.md from the directory
5. Parse, validate, write to store
6. No temp dir needed — reads directly from source

### Layer 5: URL Adapter (`source_url.go`)

1. HTTP GET with configured timeouts (30s header, 2min overall)
2. Detect archive format by URL extension (`.tar.gz`, `.tgz`, `.zip`)
3. Extract archive to temp dir
4. Walk extracted contents to discover SKILL.md files
5. Parse, validate, write to store
6. Clean up temp dir

### Testing Strategy

All adapters are tested via dependency injection:
- **Mocked store**: In-memory `Store` implementation that records `Write` calls
- **Mocked exec**: `exec.LookPath` is abstracted via a function field for testing CLI presence/absence
- **Mocked HTTP**: `httptest.NewServer` for URL adapter tests
- **Temp directories**: `t.TempDir()` for filesystem operations
- **Table-driven tests**: Cover happy path, missing dependency, invalid content, timeout, path traversal

## Complexity Tracking

No constitution violations to track. All design decisions align with existing patterns.
