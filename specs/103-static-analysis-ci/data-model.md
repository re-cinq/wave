# Data Model: Static Analysis for Unused/Redundant Go Code

**Feature**: 103-static-analysis-ci
**Date**: 2026-02-18

## Entity Overview

This feature introduces no Go types, database schemas, or runtime data structures.
All entities are declarative configuration files consumed by external tools
(golangci-lint, GitHub Actions). The "data model" describes the structure and
relationships of these configuration artifacts.

## Entities

### Entity 1: Linter Configuration (`.golangci.yml`)

**Location**: Repository root
**Format**: YAML (golangci-lint v2 schema)
**Consumed by**: golangci-lint binary (local + CI)

```yaml
# Structural schema
version: "2"                    # Required: v2 format marker

linters:
  default: standard             # Baseline preset (FR-002)
  enable:                       # Additional linters (FR-003)
    - unparam
    - wastedassign
    - gocritic
    - nolintlint
  settings:                     # Per-linter config
    nolintlint:
      require-explanation: true  # FR-004
      require-specific: true     # FR-004

exclusions:
  presets:                      # Built-in exclusion presets (FR-013)
    - std-error-handling
    - comments
```

**Constraints**:
- `version` MUST be `"2"` (string, not integer)
- `linters.default` MUST be `standard`
- `revive` MUST NOT appear in `linters.enable` (FR-014)
- `gocritic` uses default stable checks, no explicit per-check config (FR-015)

**Relationships**:
- Referenced by CI workflow via golangci-lint-action (implicit — action reads `.golangci.yml` from repo root)
- Referenced by `make lint` target (implicit — `golangci-lint run` reads `.golangci.yml` from repo root)

---

### Entity 2: CI Lint Workflow (`.github/workflows/lint.yml`)

**Location**: `.github/workflows/lint.yml`
**Format**: GitHub Actions YAML
**Consumed by**: GitHub Actions runner

```yaml
# Structural schema
name: Lint
on:
  pull_request:
    branches: [main]           # FR-005: PRs targeting main
  push:
    branches: [main]           # FR-005: pushes to main

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod    # FR-008
      - uses: golangci/golangci-lint-action@<version>
        with:
          version: "<pinned-version>"  # FR-007: specific v2.x version
          only-new-issues: true        # FR-006: incremental mode
```

**Constraints**:
- MUST be a separate file from `release.yml` (FR-012)
- MUST NOT duplicate test/build/release jobs
- Go version MUST come from `go.mod` (FR-008)
- golangci-lint version MUST be pinned (not range or "latest") (FR-007)

**Relationships**:
- Depends on `.golangci.yml` existing in repo root
- Independent from `release.yml` and `docs.yml`
- Uses same Go version as `release.yml` (via `go-version-file: go.mod`)

---

### Entity 3: Makefile Lint Target

**Location**: `Makefile` (line 24)
**Format**: GNU Make
**Consumed by**: Developer CLI

```makefile
# Before (current)
lint:
	go vet ./...

# After (FR-010)
lint:
	golangci-lint run ./...
```

**Constraints**:
- Single command replacement, no installation detection logic (C-003)
- Must exit non-zero when violations found (SC-007)

**Relationships**:
- Depends on `golangci-lint` binary being on `$PATH`
- Uses `.golangci.yml` configuration implicitly

---

### Entity 4: CLAUDE.md Documentation Updates

**Location**: `CLAUDE.md` (Testing section + Code Style section)
**Format**: Markdown
**Consumed by**: AI agents and developers

Changes required:
1. **Testing section**: Add lint command (`golangci-lint run ./...`) and auto-fix command (`golangci-lint run --fix ./...`) (FR-011)
2. **Code Style section**: Update "Run `go vet` for static analysis" to reference golangci-lint

**Relationships**:
- References `.golangci.yml` configuration
- Guides both human and AI agent behavior

## Dependency Graph

```
.golangci.yml
    ↑ (reads config)           ↑ (reads config)
    |                          |
lint.yml                   Makefile (make lint)
    |
    ↓ (reports to)
GitHub PR Checks
```

All four entities are independent in creation order — they can be implemented in
any sequence. However, testing any entity requires `.golangci.yml` to exist first.

## Migration Impact

- **No database changes**: No SQLite migrations needed
- **No Go code changes**: No new packages, types, or functions
- **No dependency additions**: golangci-lint is an external tool, not a Go module dependency
- **Backward compatibility**: Not a concern per constitution (Rapid Prototype phase)
