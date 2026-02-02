# Wave Development Guidelines

Last updated: 2026-02-02

## Overview

Wave is a multi-agent pipeline orchestrator for AI-assisted development. It coordinates multiple AI personas through structured pipelines, enforcing permissions, contracts, and workspace isolation at every step.

## Active Technologies

- Go 1.22+ (single static binary, goroutines for concurrency)
- SQLite for pipeline state persistence
- Filesystem for workspaces and artifacts
- `gopkg.in/yaml.v3` for YAML parsing
- `github.com/spf13/cobra` for CLI framework

## Project Structure

```
cmd/wave/              # CLI entry point and commands
internal/
  adapter/             # Adapter runners (Claude, mock, process group)
  audit/               # Audit logging with credential scrubbing
  contract/            # Contract validation for handover gates
  event/               # NDJSON event emission
  manifest/            # YAML manifest parsing
  pipeline/            # Pipeline execution, DAG, matrix workers
  relay/               # Token monitoring and checkpoints
  state/               # SQLite state persistence
  workspace/           # Workspace isolation and artifact injection
specs/                 # Feature specifications
```

## CLI Commands

### `wave init`
Initialize a new Wave project with default manifest.

### `wave validate`
Validate the manifest and pipeline YAML files.

### `wave run`
Execute a pipeline.

**Flags:**
- `--pipeline <name>` - Pipeline name to run (required)
- `--input <text>` - Input data for the pipeline
- `--dry-run` - Show what would be executed without running
- `--from-step <id>` - Start execution from specific step
- `--timeout <minutes>` - Timeout in minutes (overrides manifest)
- `--manifest <path>` - Path to manifest file (default: wave.yaml)
- `--mock` - Use mock adapter for testing

### `wave do`
Execute an ad-hoc task (generates navigate/execute pipeline).

**Flags:**
- `--persona <name>` - Override execute persona
- `--save <path>` - Save generated pipeline YAML to path
- `--manifest <path>` - Path to manifest file
- `--mock` - Use mock adapter for testing
- `--dry-run` - Show what would be executed without running

### `wave resume`
Resume a previously interrupted pipeline run.

### `wave clean`
Clean up project artifacts.

**Flags:**
- `--pipeline <name>` - Clean specific pipeline workspace
- `--all` - Clean all workspaces and state
- `--force` - Skip confirmation
- `--keep-last <n>` - Keep the N most recent workspaces
- `--dry-run` - Show what would be deleted without removing

### `wave list`
List pipelines, personas, and adapters.

**Subcommands:**
- `wave list pipelines` - List available pipelines
- `wave list personas` - List configured personas
- `wave list adapters` - List configured adapters

**Flags:**
- `--manifest <path>` - Path to manifest file
- `--format <type>` - Output format (table, json)

### Global Flags
- `--manifest, -m <path>` - Path to manifest file (default: wave.yaml)
- `--debug, -d` - Enable debug mode
- `--log-format <type>` - Log format (text, json)

## Testing

```bash
# Run all tests
go test ./...

# Run with race detector (required for PR)
go test -race ./...

# Run specific package
go test ./internal/pipeline/...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...
```

## Code Style

Follow standard Go conventions:
- Use `gofmt` for formatting
- Run `go vet` for static analysis
- Keep functions focused and testable
- Use interfaces for dependency injection

## Recent Changes

- Phase 15: Added thread-safe event emission with mutex protection
- Phase 15: Added comprehensive credential scrubbing tests
- Phase 15: Added workspace isolation tests
- Phase 15: Added subprocess timeout tests with process group kill
- Phase 15: All tests pass with race detector

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
