# Quickstart: Muzzle

## Prerequisites

- Go 1.22+ (for building from source)
- Claude Code CLI (`claude`) on PATH
- Git

## Install

```bash
# From source
go install github.com/recinq/muzzle/cmd/muzzle@latest

# Or build locally
git clone https://github.com/recinq/muzzle.git
cd muzzle
go build -o muzzle ./cmd/muzzle/
```

## Initialize a Project

```bash
cd your-project
muzzle init
```

This creates:
- `muzzle.yaml` — manifest with default claude adapter and example personas
- `.muzzle/personas/` — system prompt files for each persona
- `.muzzle/pipelines/` — example pipeline definitions

## Validate Configuration

```bash
muzzle validate
```

Checks that:
- `muzzle.yaml` parses correctly
- All persona system prompt files exist
- All adapter binaries are on PATH
- All hook scripts exist
- Pipeline YAML files have valid DAGs (no cycles)

## Run a Pipeline

```bash
# Run a named pipeline with input
muzzle run --pipeline speckit-flow --input "add user authentication"

# Resume an interrupted pipeline
muzzle resume --pipeline-id <uuid>

# Resume from a specific step
muzzle resume --pipeline-id <uuid> --from-step "speckit.plan"
```

## Ad-Hoc Execution

```bash
# Quick task — navigator + craftsman, no full pipeline
muzzle do "fix the auth bug in src/auth/"

# With persona override
muzzle do "check for SQL injection" --persona auditor

# Save the generated pipeline for inspection
muzzle do "add dark mode" --save .muzzle/pipelines/adhoc-dark-mode.yaml
```

## Clean Up Workspaces

```bash
# Remove all ephemeral workspaces
muzzle clean

# Remove workspaces for a specific pipeline
muzzle clean --pipeline-id <uuid>
```

## Verify Installation

```bash
# Create a test manifest and validate
muzzle init
muzzle validate

# Run a dry-run (no LLM calls, validates pipeline structure)
muzzle run --pipeline speckit-flow --dry-run --input "test"
```

Expected output: structured JSON events showing each step transition
(Pending → Running → Completed) without actually invoking adapters.
