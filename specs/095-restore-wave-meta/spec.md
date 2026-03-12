# feat: restore and stabilize `wave meta` dynamic pipeline generation

**Issue**: [#95](https://github.com/re-cinq/wave/issues/95)
**Labels**: enhancement, needs-design, priority: medium
**Author**: nextlevelshit

## Summary

The `wave meta` command — which dynamically generates and executes multi-step pipelines via the philosopher persona — needs to be restored to working condition. The command exists in the codebase (`cmd/wave/commands/meta.go`) with full implementation including dry-run, save, and execution modes, but is currently non-functional or degraded.

## Background

`wave meta` was introduced in commit `164a20f` (\"implement meta-pipeline for self-designing pipelines\") and received subsequent improvements (progress events, schema generation, contract validation, prompt cleanup). It allows users to describe a task in natural language and have the philosopher persona design an appropriate pipeline with steps, personas, and contracts, then execute it.

### Current Implementation

- **Command**: `wave meta [task description]`
- **Flags**: `--dry-run`, `--save <path>`, `--manifest <path>`, `--mock`
- **Flow**: User input → philosopher persona generates pipeline YAML → pipeline executor runs generated steps
- **Key files**: `cmd/wave/commands/meta.go`, `internal/pipeline/meta.go`

## Problem

The `wave meta` command is not functioning correctly. This issue tracks diagnosing the failure mode and restoring full functionality.

## Acceptance Criteria

- [ ] `wave meta "<task>" --dry-run` generates a valid pipeline and displays the step plan
- [ ] `wave meta "<task>"` executes the generated pipeline end-to-end
- [ ] `wave meta "<task>" --save <name>` persists the generated pipeline YAML for reuse
- [ ] `wave meta` with `--mock` adapter works for testing without live LLM calls
- [ ] All existing tests in `cmd/wave/commands/meta_test.go` pass
- [ ] Philosopher persona is properly configured in the default manifest
- [ ] Error messages are clear when prerequisites are missing (e.g., no philosopher persona)

## Investigation Checklist

- [ ] Verify the philosopher persona exists and is correctly defined in the default manifest
- [ ] Check that `MetaPipelineExecutor` in `internal/pipeline/` is properly wired
- [ ] Test with `--mock` adapter to isolate adapter vs. pipeline issues
- [ ] Review recent refactors that may have broken the meta command integration
- [ ] Confirm contract validation works for dynamically generated pipelines

## Related History

- `164a20f` — feat: implement meta-pipeline for self-designing pipelines
- `c6e0870` — feat(do): add --meta flag for dynamic pipeline generation
- `5b9ab1d` — fix(meta): remove redundant schema instructions from generated prompts
- `5e7b2af` — feat: add standalone wave meta command
- `6d24cc9` — refactor(pipeline): default memory.strategy to fresh

## Diagnosed Issues

### 1. Mock Adapter Does Not Produce Meta-Pipeline Format (Critical)

The mock adapter (`internal/adapter/mock.go`) handles the philosopher persona by falling back to `generateDocsPhaseOutput()`, which returns JSON. However, `invokePhilosopherWithSchemas()` in `meta.go` expects output in the `--- PIPELINE ---` / `--- SCHEMAS ---` delimited format parsed by `extractPipelineAndSchemas()`. This means `wave meta --mock` always fails with "missing --- PIPELINE --- marker".

**Root cause**: No `meta-philosopher` workspace path or philosopher-meta output generator in the mock adapter.

### 2. Unsafe io.Reader Consumption (Bug)

In `invokePhilosopherWithSchemas()` (`internal/pipeline/meta.go:283-285`), stdout is read with a single `result.Stdout.Read(buf)` call using a 1MB buffer. The `io.Reader` contract does NOT guarantee a single `Read()` returns all available data — it may return fewer bytes. For large generated pipeline YAML + schemas, this silently truncates output.

**Root cause**: Should use `io.ReadAll()` instead of a manual `Read()` call.

### 3. Missing Onboarding Check (Minor)

The `meta` command does not call `checkOnboarding()` like other commands (`do`, `run`). This means `wave meta` can run in an uninitialized project without the expected guardrail error.

### 4. Philosopher Persona System Prompt Mismatch (Design)

The philosopher persona (`philosopher.md`) is configured as a spec/architecture writer, not a meta-pipeline architect. The `buildPhilosopherPrompt()` function injects pipeline-generation instructions, but the persona's system prompt may conflict or confuse the LLM when both are active.
