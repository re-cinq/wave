# Research: Ecosystem Adapters for Skill Sources

**Feature**: #383 — Skill source adapters with prefix routing
**Date**: 2026-03-14

## Decision 1: Interface Design Pattern for SourceAdapter

**Decision**: Strategy pattern — each adapter implements a common `SourceAdapter` interface with `Install(ctx, ref, store)` and `Prefix()` methods. The router holds a `map[string]SourceAdapter` and dispatches by prefix.

**Rationale**: This is the standard Go pattern for pluggable behavior (see `io.Reader`, `http.Handler`, `sort.Interface`). It satisfies SC-007 (extensibility) — new adapters are added by implementing the interface and calling `RegisterAdapter()`, with zero changes to existing code.

**Alternatives Rejected**:
- **Function map** (`map[string]func(...) error`): Less type-safe, harder to test, loses the ability to associate metadata (prefix) with behavior.
- **Switch statement**: Violates open/closed principle, every new adapter modifies the router.

## Decision 2: Source String Parsing Strategy

**Decision**: URL-scheme-first parsing. The router checks `strings.HasPrefix(src, "https://")` and `strings.HasPrefix(src, "http://")` before splitting on the first `:` for standard prefixes.

**Rationale**: The `https://` prefix contains `://` which conflicts with the simple `prefix:reference` split on first colon. Checking URL schemes first avoids ambiguity and matches user expectation — a URL looks like a URL, not `https` + `//example.com`.

**Alternatives Rejected**:
- **Split on `://` vs `:`**: Overly complex parsing, fragile for edge cases.
- **Require `url:` prefix for URLs**: Poor UX — users expect `https://` to just work.

## Decision 3: Subprocess Execution for CLI Adapters

**Decision**: Use `exec.CommandContext(ctx, binary, args...)` with `context.WithTimeout` for all CLI adapters (tessl, bmad, openspec, speckit). Capture stdout+stderr via pipes.

**Rationale**: This matches the existing `adapter.ProcessGroupRunner` pattern. Using `CommandContext` ensures timeout enforcement via context cancellation. The spec requires 2-minute timeouts (FR-016, C-004).

**Implementation Detail**: Each CLI adapter will:
1. Check `exec.LookPath(binary)` to verify the tool exists
2. Create a temp directory via `os.MkdirTemp`
3. Run the CLI command with the temp dir as working directory
4. Discover SKILL.md files in the temp dir
5. Parse and validate each SKILL.md via existing `skill.Parse()`
6. Write valid skills to the store via `store.Write()`
7. Clean up temp dir via `defer os.RemoveAll()`

**Alternatives Rejected**:
- **Shell execution** (`sh -c`): Injection risk, platform-dependent.
- **Reusing `adapter.ProcessGroupRunner`**: That runner is purpose-built for Claude Code NDJSON output, not for generic CLI invocation. Bringing in its process group management and stream event parsing would add unnecessary complexity.

## Decision 4: Soft Dependency Error Reporting

**Decision**: Each adapter defines its CLI dependency with a struct: `{Binary, InstallInstructions}`. When `exec.LookPath` fails, return a `*DependencyError` containing the binary name and install instructions.

**Rationale**: FR-011 requires actionable error messages naming the missing tool and install instructions. A typed error allows callers to present structured guidance.

**Implementation Detail**: Use `exec.LookPath` (already used by `preflight.Checker.CheckTools`). The error type will be:
```go
type DependencyError struct {
    Binary       string
    Instructions string
}
```

**Alternatives Rejected**:
- **Generic error string**: Loses structured information, harder for callers to extract.
- **Reusing `preflight.ToolError`**: That type is for batch preflight checks. Source adapter errors are per-adapter, single-tool errors with install instructions.

## Decision 5: File Adapter Path Security

**Decision**: Reuse the `containedPath()` function pattern from `internal/skill/store.go` for the `file:` adapter. Resolve relative paths against the project root. Reject symlinks and paths that escape the project root.

**Rationale**: The skill store already has battle-tested path containment validation. The `file:` adapter has the same security requirements. Spec scenarios explicitly require: rejecting symlinks (User Story 3, scenario 4), rejecting traversal (Edge Case 5).

**Implementation Detail**:
- Relative paths (`file:./foo`) resolve against project root
- Absolute paths (`file:/abs/path`) are validated against project root containment
- Symlinks are rejected via `os.Lstat` check
- The `file:` adapter copies the skill directory into the store — it doesn't create a symlink

## Decision 6: GitHub Adapter Cloning Strategy

**Decision**: Use `git clone --depth 1` for shallow clones into a temp directory. Parse the optional path suffix from the reference string to locate specific skill subdirectories.

**Rationale**: Shallow clones minimize bandwidth and disk usage. The spec only needs the latest files, not history. The `github:owner/repo/path` format requires parsing — split on `/` after the first two components (`owner/repo`), remainder is the path.

**Implementation Detail**:
- `github:owner/repo` → clone `https://github.com/owner/repo.git`, discover SKILL.md at root
- `github:owner/repo/path/to/skill` → clone, navigate to `path/to/skill`, validate SKILL.md
- When no path and no root SKILL.md: walk subdirectories to discover all SKILL.md files

## Decision 7: HTTPS Adapter Archive Handling

**Decision**: Use Go's `net/http` client with timeouts for downloading. Detect archive format by file extension (`.tar.gz`, `.tgz`, `.zip`). Use `archive/tar` + `compress/gzip` and `archive/zip` from the standard library.

**Rationale**: Go's standard library provides robust archive handling. No external dependencies needed. Extension-based detection is straightforward and matches user expectations.

**Implementation Detail**:
- HTTP client: 30s header timeout, 2min overall via context
- Supported formats: `.tar.gz`, `.tgz`, `.zip`
- After extraction: discover SKILL.md files, parse, validate, write to store
- Non-archive responses (HTML, etc.) return a clear error

## Decision 8: Package Layout

**Decision**: All source adapter code lives within `internal/skill/` as new files. No new package.

**Rationale**: The source adapter system is tightly coupled with the skill store and parser — it reads SKILL.md files and writes to the store. Creating a separate package would require exporting internal types or creating circular dependencies. Keeping it in `internal/skill/` provides direct access to `Store`, `Parse()`, `Skill`, etc.

**File layout**:
```
internal/skill/
├── source.go          # SourceAdapter interface, SourceRouter, SourceReference, InstallResult, error types
├── source_test.go     # Router and parsing tests
├── source_cli.go      # CLI-based adapters: tessl, bmad, openspec, speckit
├── source_cli_test.go # CLI adapter tests with mocked exec
├── source_github.go   # GitHub adapter
├── source_github_test.go
├── source_file.go     # Local file adapter
├── source_file_test.go
├── source_url.go      # HTTPS URL/archive adapter
├── source_url_test.go
```

**Alternatives Rejected**:
- **`internal/skill/source/` sub-package**: Would require exporting `Store`, `Skill`, `Parse` types from `internal/skill/` and importing them. Adds package complexity with no encapsulation benefit since all adapters are internal.
- **`internal/source/`**: Decouples from skill types, requiring inter-package imports for `skill.Store`, `skill.Skill`, etc.
