# Implementation Plan: file:// URI Scheme for Display Paths

## Objective

Add `file://` URI prefix to all absolute file paths in Wave's user-facing CLI output (error messages, recovery hints, outcome summaries, artifact references) to enable clickable terminal links. Internal processing paths remain unmodified.

## Approach

Create a single shared utility function `FileURI(path string) string` in a new file `internal/display/fileuri.go`. This function converts absolute paths to `file://` URIs and is idempotent (no double-prefixing). Then apply this function at each display boundary where paths are rendered for human consumption.

The utility lives in `internal/display` because all consumers are display/formatting code. This avoids creating a new package and keeps the change self-contained.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/display/fileuri.go` | **create** | New utility: `FileURI(path) string` — prefixes absolute paths with `file://`, skips relative paths and paths already containing a URI scheme |
| `internal/display/fileuri_test.go` | **create** | Table-driven tests for `FileURI`: absolute paths, relative paths, already-prefixed paths, empty strings, paths with special characters |
| `internal/contract/validation_error_formatter.go` | **modify** | Line 23: wrap `artifactPath` with `display.FileURI()` in the `File:` format string |
| `internal/contract/jsonschema.go` | **modify** | Lines 156, 213, 242: wrap `artifactPath` with `display.FileURI()` in error detail strings |
| `internal/recovery/recovery.go` | **modify** | Line 129: wrap `block.WorkspacePath` with `display.FileURI()` in the workspace hint `ls` command |
| `internal/display/outcome.go` | **modify** | Lines 275-276: wrap `outcome.WorkspacePath` with `FileURI()` in `GenerateNextSteps` |
| `internal/deliverable/types.go` | **modify** | Line 117: wrap `absPath` with `display.FileURI()` in `Deliverable.String()` |
| `internal/display/progress.go` | **modify** | Line 681: wrap `path` with `FileURI()` in handover artifact line |
| `internal/display/bubbletea_model.go` | **modify** | Line 311: wrap `path` with `FileURI()` in handover artifact line |
| `internal/contract/validation_error_formatter_test.go` | **modify** | Update expected output strings to include `file://` prefix |
| `internal/recovery/recovery_test.go` | **modify** | Update expected workspace paths with `file://` prefix for absolute path test cases |
| `internal/display/outcome_test.go` | **modify** | Update expected output strings to include `file://` prefix |

## Architecture Decisions

1. **Single utility function, not an interface**: This is a pure formatting function with no side effects or dependencies. An interface would be over-engineering.

2. **Placed in `internal/display`**: The `display` package is the natural home since it owns all formatting concerns. `internal/contract` and `internal/recovery` will import `display` — this is acceptable since contracts and recovery already produce display-ready strings. Check for import cycles; if any exist, the function can be placed in a tiny `internal/pathfmt` package instead.

3. **Applied at display boundaries only**: The function is called where strings are formatted for human output, not where paths are used internally. This ensures no side effects on file operations.

4. **Idempotent by design**: `FileURI` checks for existing URI schemes before prepending, so calling it multiple times on the same path is safe.

5. **Unix-only scope**: The `file://` prefix uses POSIX path conventions (`/` root). Windows support is out of scope per the issue.

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Import cycle `contract` → `display` | Build failure | If cycle exists, extract `FileURI` to `internal/pathfmt` or `internal/display/pathfmt` |
| Recovery hint `ls` command breaks with `file://` prefix | `ls file:///path` doesn't work in shell | The `ls` command path should remain a raw path; only the display label/inspect command gets the prefix. Need to distinguish between the executable command and the display path |
| Test assertion breakage | Many tests check exact string output | Update all affected test assertions; use grep to find all instances |

## Testing Strategy

1. **Unit tests for `FileURI`** (`internal/display/fileuri_test.go`):
   - Absolute path → `file:///absolute/path`
   - Relative path → unchanged
   - Already `file://` prefixed → unchanged
   - `https://` prefixed → unchanged
   - Empty string → empty string
   - Path with spaces → properly prefixed
   - Path with special characters → properly prefixed

2. **Updated unit tests** in affected packages:
   - `internal/contract/validation_error_formatter_test.go` — verify `File:` line contains `file://`
   - `internal/recovery/recovery_test.go` — verify workspace paths with absolute roots get `file://`
   - `internal/display/outcome_test.go` — verify next steps workspace inspection uses `file://`

3. **Run full test suite**: `go test ./...` to ensure no regressions
