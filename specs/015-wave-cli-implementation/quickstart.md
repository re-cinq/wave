# Quickstart: Testing the Hardened Wave CLI

**Branch**: `015-wave-cli-implementation`
**Date**: 2026-02-02

## Prerequisites

- Go 1.22+
- Claude Code CLI (`claude`) on PATH (optional, for integration tests)

## Building

```bash
# Build the binary
go build -o wave ./cmd/wave

# Verify it runs
./wave --help
```

## Running Tests

```bash
# Run all tests
go test ./...

# Run with race detection (important for concurrent code)
go test -race ./...

# Run specific package tests
go test ./internal/state/...
go test ./internal/pipeline/...
go test ./cmd/wave/commands/...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Test Categories

### Unit Tests

Fast, isolated tests for individual functions:

```bash
# Manifest parsing
go test ./internal/manifest/...

# DAG operations
go test ./internal/pipeline/... -run TestDAG

# Contract validation
go test ./internal/contract/...
```

### Integration Tests

Tests that exercise multiple packages together:

```bash
# Pipeline execution (uses mock adapter)
go test ./internal/pipeline/... -run TestExecutor

# CLI commands
go test ./cmd/wave/commands/...
```

### Concurrency Tests

Tests for race conditions and parallel execution:

```bash
# Matrix worker concurrency
go test -race ./internal/pipeline/... -run TestMatrix

# State store concurrency
go test -race ./internal/state/...
```

## Manual Testing

### Initialize a Project

```bash
./wave init
./wave validate
```

### Run a Pipeline

```bash
# Dry run (preview execution plan)
./wave run --pipeline hotfix --input "test bug" --dry-run

# Actual execution
./wave run --pipeline hotfix --input "test bug"
```

### Ad-Hoc Execution

```bash
# Quick task
./wave do "add a comment to main.go"

# With persona override
./wave do "find security issues" --persona auditor

# Save generated pipeline
./wave do "fix the bug" --save generated.yaml
```

### List Configuration

```bash
./wave list pipelines
./wave list personas
./wave list adapters
```

### Cleanup

```bash
# Preview what would be deleted
./wave clean --dry-run

# Keep last 3 workspaces
./wave clean --keep-last 3

# Delete all
./wave clean
```

## Test Fixtures

Test fixtures are located in `testdata/` directories:

```
internal/manifest/testdata/
├── valid.yaml           # Valid manifest
├── invalid-adapter.yaml # Missing adapter reference
├── invalid-yaml.yaml    # Malformed YAML
└── ...

internal/pipeline/testdata/
├── simple.yaml          # Linear pipeline
├── parallel.yaml        # Parallel steps
├── cycle.yaml           # Circular dependency (should fail)
└── ...
```

## Debugging Tests

```bash
# Print verbose test output
go test -v ./internal/pipeline/... -run TestExecutor

# Run specific test
go test ./internal/pipeline/... -run TestExecutor/step_timeout

# Debug with delve
dlv test ./internal/pipeline/... -- -test.run TestExecutor
```

## Coverage Targets

Target coverage by package:

| Package | Target | Notes |
|---------|--------|-------|
| internal/manifest | 90% | Critical path |
| internal/pipeline | 85% | Complex logic |
| internal/adapter | 80% | Subprocess boundaries |
| internal/workspace | 85% | File operations |
| internal/contract | 90% | Validation logic |
| internal/relay | 80% | Token monitoring |
| internal/state | 90% | Data persistence |
| internal/event | 85% | Event emission |
| internal/audit | 90% | Security-critical |
| cmd/wave/commands | 80% | CLI integration |
