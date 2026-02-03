# Quickstart: Prototype-Driven Development Pipeline

**Feature**: 017-prototype-driven-development
**Date**: 2026-02-02

## Overview

The prototype pipeline guides greenfield development through four sequential phases: specification, documentation, dummy implementation, and full implementation. Each phase produces artifacts that flow to the next, with contract validation at every boundary.

---

## Prerequisites

1. **Wave installed and configured**
   ```bash
   wave --version
   # wave v0.x.x
   ```

2. **Manifest file present**
   ```bash
   ls wave.yaml
   # wave.yaml should exist
   ```

3. **Personas configured**
   Required personas: `navigator`, `philosopher`, `craftsman`, `auditor`, `planner`, `summarizer`
   ```bash
   wave list personas
   ```

4. **Pipeline deployed** (after implementation)
   ```bash
   ls .wave/pipelines/prototype.yaml
   ```

---

## Usage

### Basic Usage

Run the prototype pipeline for a new feature:

```bash
wave run --pipeline prototype --input "implement user authentication with JWT tokens"
```

### With Routing (Automatic Selection)

Configure routing in `wave.yaml`:

```yaml
runtime:
  routing:
    rules:
      - pattern: "*prototype*"
        pipeline: prototype
        priority: 10
      - pattern: "*greenfield*"
        pipeline: prototype
        priority: 10
```

Then use `wave do`:

```bash
wave do "prototype: implement user authentication with JWT tokens"
```

### Dry Run (Preview)

See what will be executed without running:

```bash
wave run --pipeline prototype --input "implement user auth" --dry-run
```

---

## Pipeline Phases

### Phase 1: Specification

**What it does**: Analyzes the codebase and generates a feature specification.

**Steps**:
1. `spec-navigate` - Navigator analyzes codebase patterns
2. `spec-define` - Philosopher creates specification document

**Output**: `specification.json` containing:
- User stories with acceptance criteria
- Entity definitions
- Interface contracts
- Edge cases

**Contract**: `spec-phase.schema.json`

**Duration**: ~2-5 minutes

---

### Phase 2: Documentation

**What it does**: Generates stakeholder-readable documentation from the specification.

**Steps**:
1. `docs-generate` - Philosopher creates documentation

**Output**:
- `feature-docs.md` - Overview and user guide
- `api-docs.md` - Technical API reference
- `docs-manifest.json` - Documentation tracking

**Contract**: `docs-phase.schema.json`

**Duration**: ~1-3 minutes

---

### Phase 3: Dummy Implementation

**What it does**: Creates a working prototype with stub implementations.

**Steps**:
1. `dummy-scaffold` - Craftsman creates prototype code
2. `dummy-verify` - Auditor verifies prototype completeness

**Output**:
- `prototype/` - Working stub implementation
- `interfaces.json` - Interface definitions
- `dummy-manifest.json` - Prototype tracking
- `dummy-verification.md` - Verification report

**Contract**: `dummy-phase.schema.json`

**Duration**: ~5-10 minutes

---

### Phase 4: Implementation

**What it does**: Full production implementation with tests and review.

**Steps**:
1. `implement-plan` - Planner creates task breakdown
2. `implement-code` - Craftsman implements with tests
3. `implement-review` - Auditor performs final review

**Output**:
- Source code changes
- Unit and integration tests
- `implementation-plan.md`
- `final-review.md`

**Contract**: Test suite must pass (`go test ./...`)

**Duration**: ~15-60 minutes (depends on complexity)

---

## Common Operations

### Resume from a Phase

If the pipeline is interrupted, resume from the last completed step:

```bash
wave resume
```

Or resume from a specific phase:

```bash
# Resume from docs phase
wave run --pipeline prototype --from-step docs-generate

# Resume from implementation
wave run --pipeline prototype --from-step implement-plan
```

### Re-run a Phase

To re-generate artifacts from a specific phase:

```bash
# Re-run spec phase with updated input
wave run --pipeline prototype --from-step spec-navigate --input "updated requirements"
```

**Note**: Downstream phases will be marked as stale and should be re-run.

### Check Pipeline Status

View the current state of a pipeline run:

```bash
wave status --pipeline prototype
```

### View Artifacts

Artifacts are stored in the workspace:

```bash
ls .wave/workspaces/prototype-<run-id>/
```

---

## Troubleshooting

### Contract Validation Failed

**Symptom**: Pipeline stops with "contract validation failed"

**Cause**: Phase output doesn't match expected schema

**Solution**:
1. Check the error message for specific field failures
2. Review the phase output in the workspace
3. Re-run the phase with more specific input

```bash
# Example: spec phase failed validation
wave run --pipeline prototype --from-step spec-define --input "more detailed requirements"
```

### Prototype Not Runnable

**Symptom**: `dummy-verify` fails with "prototype not runnable"

**Cause**: Missing entry point or broken dependencies

**Solution**:
1. Check `dummy-manifest.json` for `runnable: false`
2. Review `dummy-verification.md` for specific issues
3. Re-run scaffold with clearer interface requirements

### Implementation Tests Failing

**Symptom**: `implement-code` retries exhausted

**Cause**: Generated code doesn't pass test suite

**Solution**:
1. Review test failures in workspace logs
2. Check if specification is clear enough
3. Resume from `implement-code` after manual fixes

### Stale Artifacts Warning

**Symptom**: "Downstream artifacts may be stale"

**Cause**: Upstream phase was re-run after downstream completed

**Solution**:
```bash
# Re-run affected downstream phases
wave run --pipeline prototype --from-step docs-generate
```

---

## Example: Complete Workflow

```bash
# 1. Initialize new feature development
wave run --pipeline prototype --input "implement rate limiting middleware with configurable limits per endpoint"

# 2. Review specification (in workspace)
cat .wave/workspaces/prototype-*/output/specification.json | jq .

# 3. Review documentation
cat .wave/workspaces/prototype-*/output/feature-docs.md

# 4. Test the dummy implementation
cd .wave/workspaces/prototype-*/prototype/
./run.sh  # or equivalent entry point

# 5. Review final implementation
git diff main
cat .wave/workspaces/prototype-*/output/final-review.md

# 6. Clean up (optional)
wave clean --pipeline prototype --keep-last 1
```

---

## Integration with Speckit

If your project uses speckit (`.specify/` directory present), the spec phase will automatically use speckit commands for richer specification generation:

```bash
# Speckit will be detected and used automatically
wave run --pipeline prototype --input "implement feature X"

# The spec phase will run:
# /speckit.spec "implement feature X"
```

Without speckit, the philosopher persona generates specifications manually following the same schema.

---

## Configuration Reference

### Pipeline File Location

```
.wave/pipelines/prototype.yaml
```

### Contract Schema Locations

```
.wave/contracts/spec-phase.schema.json
.wave/contracts/docs-phase.schema.json
.wave/contracts/dummy-phase.schema.json
```

### Workspace Location

```
.wave/workspaces/prototype-<timestamp>/
```

### Runtime Configuration

In `wave.yaml`:

```yaml
runtime:
  default_timeout_minutes: 30
  meta_pipeline:
    max_total_steps: 20
    max_total_tokens: 500000
  relay:
    token_threshold_percent: 80
    strategy: summarize_to_checkpoint
```
