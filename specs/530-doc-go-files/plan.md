# Implementation Plan

## Objective

Add `doc.go` files to 10 core internal packages that currently lack package-level documentation, following the existing convention established in `internal/continuous/doc.go`.

## Approach

Create one `doc.go` file per package with a multi-line package comment describing:
- What the package does (1-2 sentence overview)
- Key responsibilities and capabilities
- How it fits in the overall architecture

Follow the existing convention: `// Package <name> ...` comment block followed by `package <name>`.

## File Mapping

All files are **new creations** (no modifications):

| File | Package Purpose |
|------|----------------|
| `internal/pipeline/doc.go` | DAG pipeline orchestration, execution, resumption, and artifact management |
| `internal/adapter/doc.go` | Subprocess execution of LLM adapters with streaming and permission enforcement |
| `internal/contract/doc.go` | Output validation against structured contracts (JSON schema, TypeScript, test suites) |
| `internal/manifest/doc.go` | Wave YAML configuration parsing, validation, and typed structures |
| `internal/workspace/doc.go` | Ephemeral isolated execution environments with mount-based file mapping |
| `internal/state/doc.go` | SQLite-backed pipeline state persistence and run tracking |
| `internal/event/doc.go` | Structured progress event emission for real-time monitoring |
| `internal/audit/doc.go` | Execution trace logging with credential scrubbing |
| `internal/security/doc.go` | Input sanitization, path validation, and prompt injection prevention |
| `internal/relay/doc.go` | Token usage monitoring and context compaction triggering |

## Architecture Decisions

- **No code changes**: Only new doc.go files, no modifications to existing code
- **Convention alignment**: Match the style of `internal/continuous/doc.go`
- **Accurate descriptions**: Each comment derived from reading the actual package source

## Risks

- **Minimal risk**: Adding doc.go files cannot break compilation or tests
- Only risk is inaccurate descriptions — mitigated by reading source before writing

## Testing Strategy

- `go build ./...` to verify compilation
- `go vet ./...` to verify no vet issues
- `go test ./...` to verify no regressions
- `go doc ./internal/<pkg>` spot-check for each package
