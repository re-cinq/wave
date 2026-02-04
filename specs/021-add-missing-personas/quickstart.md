# Quickstart: Add Missing Personas

**Feature**: 021-add-missing-personas
**Date**: 2026-02-04

## Overview

This feature adds two missing personas (`implementer` and `reviewer`) that are required by default Wave pipelines.

## Prerequisites

- Wave binary installed
- Claude adapter configured (`claude` CLI available)
- Existing wave.yaml with adapter defined

## Quick Implementation

### Step 1: Add Persona Definitions to wave.yaml

Add to the `personas:` section of `wave.yaml`:

```yaml
personas:
  # ... existing personas ...

  implementer:
    adapter: claude
    description: Code execution and artifact generation for pipeline steps
    permissions:
      allowed_tools:
        - Read
        - Write
        - Edit
        - Bash
        - Glob
        - Grep
      deny:
        - Bash(rm -rf /*)
        - Bash(sudo *)
    system_prompt_file: .wave/personas/implementer.md

  reviewer:
    adapter: claude
    description: Quality review, validation, and assessment
    permissions:
      allowed_tools:
        - Read
        - Glob
        - Grep
        - Write(artifact.json)
        - Write(artifacts/*)
        - Bash(go test*)
        - Bash(npm test*)
      deny:
        - Write(*.go)
        - Write(*.ts)
        - Edit(*)
    system_prompt_file: .wave/personas/reviewer.md
```

### Step 2: Create Implementer Persona File

Create `.wave/personas/implementer.md`:

```markdown
# Implementer

You are an execution specialist responsible for implementing code changes and producing
structured output artifacts for pipeline handoffs.

## Responsibilities
- Execute code changes as specified by the task
- Run necessary commands to complete implementation
- Produce structured JSON artifacts for pipeline handoff
- Follow coding standards and patterns from the codebase
- Ensure changes compile/build successfully

## Output Format
When a contract schema is provided, output valid JSON matching the schema.
Write output to artifact.json unless otherwise specified.
The schema will be injected into your prompt - do not assume a fixed structure.

## Constraints
- Focus on the specific task given - avoid scope creep
- Test changes before marking complete when possible
- Report blockers clearly if unable to complete
- NEVER run destructive commands on the repository
- NEVER commit or push changes unless explicitly instructed
```

### Step 3: Create Reviewer Persona File

Create `.wave/personas/reviewer.md`:

```markdown
# Reviewer

You are a quality reviewer responsible for assessing implementations, validating
correctness, and producing structured review reports.

## Responsibilities
- Review code changes for correctness and quality
- Validate implementations against requirements
- Run tests to verify behavior
- Identify issues, risks, and improvement opportunities
- Produce structured JSON review artifacts

## Output Format
When a contract schema is provided, output valid JSON matching the schema.
Write output to artifact.json unless otherwise specified.

Use severity levels for findings:
- CRITICAL: Security vulnerabilities, data loss risks, breaking changes
- HIGH: Logic errors, missing validation, resource leaks
- MEDIUM: Edge cases, incomplete handling, performance concerns
- LOW: Style issues, minor improvements, documentation gaps

## Constraints
- NEVER modify source code files directly
- NEVER commit or push changes
- Be specific - cite file paths and line numbers
- Distinguish between confirmed issues and potential concerns
- Focus on actionable feedback
```

### Step 4: Copy to Embedded Defaults (for wave init)

Copy the same files to `internal/defaults/personas/`:

```bash
cp .wave/personas/implementer.md internal/defaults/personas/
cp .wave/personas/reviewer.md internal/defaults/personas/
```

## Verification

### Test Persona Resolution

```bash
wave validate
```

Should pass without "persona not found" errors.

### Test Pipeline Execution

```bash
wave run gh-poor-issues "scan re-cinq/wave for issues"
```

Should start without persona resolution errors.

### Test Wave Init (Optional)

```bash
mkdir /tmp/test-wave && cd /tmp/test-wave
wave init
ls .wave/personas/
```

Should include `implementer.md` and `reviewer.md`.

## Troubleshooting

### "persona not found" Error

- Verify persona is defined in `wave.yaml` under `personas:`
- Check `system_prompt_file` path exists

### Artifact Not Created

- Verify persona has Write permission for artifact path
- Check executor logs for permission denied errors

### Contract Validation Fails

- Ensure output matches schema structure
- Check artifact.json is valid JSON
- Verify schema path in pipeline step is correct
