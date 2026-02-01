# Quickstart: Wave

## Prerequisites

- Go 1.22+ (for building from source)
- Claude Code CLI (`claude`) on PATH
- Git

## Install

```bash
# From source
go install github.com/recinq/wave/cmd/wave@latest

# Or build locally
git clone https://github.com/recinq/wave.git
cd wave
go build -o wave ./cmd/wave/
```

## Initialize a Project

```bash
cd your-project
wave init
```

This creates:
- `wave.yaml` — manifest with default claude adapter and example personas
- `.wave/personas/` — system prompt files for each persona
- `.wave/pipelines/` — example pipeline definitions

## Validate Configuration

```bash
wave validate
```

Checks that:
- `wave.yaml` parses correctly
- All persona system prompt files exist
- All adapter binaries are on PATH
- All hook scripts exist
- Pipeline YAML files have valid DAGs (no cycles)

## Run a Pipeline

```bash
# Run a named pipeline with input
wave run --pipeline speckit-flow --input "add user authentication"

# Resume an interrupted pipeline
wave resume --pipeline-id <uuid>

# Resume from a specific step
wave resume --pipeline-id <uuid> --from-step "speckit.plan"
```

## Ad-Hoc Execution

```bash
# Quick task — navigator + craftsman, no full pipeline
wave do "fix the auth bug in src/auth/"

# With persona override
wave do "check for SQL injection" --persona auditor

# Save the generated pipeline for inspection
wave do "add dark mode" --save .wave/pipelines/adhoc-dark-mode.yaml
```

## Clean Up Workspaces

```bash
# Remove all ephemeral workspaces
wave clean

# Remove workspaces for a specific pipeline
wave clean --pipeline-id <uuid>
```

## Verify Installation

```bash
# Create a test manifest and validate
wave init
wave validate

# Run a dry-run (no LLM calls, validates pipeline structure)
wave run --pipeline speckit-flow --dry-run --input "test"
```

Expected output: structured JSON events showing each step transition
(Pending → Running → Completed) without actually invoking adapters.
