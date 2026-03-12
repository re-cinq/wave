# Implementation Plan: Restore `wave meta`

## Objective

Fix the `wave meta` command so it works end-to-end in both live and mock modes: philosopher generates valid pipeline YAML, the executor parses and runs it, and `--dry-run`/`--save`/`--mock` flags all function correctly.

## Approach

The investigation revealed four specific issues (see spec.md § Diagnosed Issues). The fixes are targeted and localized — no architectural redesign needed.

1. **Fix mock adapter** to produce `--- PIPELINE ---` / `--- SCHEMAS ---` formatted output when invoked for the meta-philosopher workspace
2. **Fix unsafe io.Reader** consumption to use `io.ReadAll()` instead of manual buffer read
3. **Add onboarding check** to the meta command
4. **Add integration test** for `wave meta --mock --dry-run` to verify the full flow

## File Mapping

| File | Action | Purpose |
|------|--------|---------|
| `internal/adapter/mock.go` | modify | Add `generateMetaPhilosopherOutput()` for meta-pipeline workspace path |
| `internal/pipeline/meta.go` | modify | Replace manual `Read(buf)` with `io.ReadAll()` in `invokePhilosopherWithSchemas` |
| `cmd/wave/commands/meta.go` | modify | Add `checkOnboarding()` call |
| `cmd/wave/commands/meta_test.go` | modify | Add integration test for `--mock --dry-run` flow |
| `internal/pipeline/meta_test.go` | modify | Add test for mock adapter philosopher output parsing |

## Architecture Decisions

### Mock output format
The mock adapter's `generateRealisticOutput` routes based on workspace path, then persona. The meta executor creates workspace at `.wave/workspaces/meta-philosopher`. We add a workspace-path check for `meta-philosopher` that returns properly delimited output with a valid 2-step pipeline (navigate + implement) and matching schema files.

### io.ReadAll vs buffered Read
`io.ReadAll()` is the idiomatic Go approach for consuming an entire reader. The existing 1MB manual buffer is both fragile and unnecessarily complex. The mock adapter returns `bytes.NewReader()` which supports full reads, and the real adapter collects all stdout before returning, so `io.ReadAll()` is safe in both cases.

### Onboarding check
Mirrors the pattern in `do.go` and `run.go`. Placed at the top of `runMeta()` before manifest loading.

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Mock output doesn't match real philosopher output format | Mock tests pass but real mode fails | The format (`--- PIPELINE ---` / `--- SCHEMAS ---`) is enforced by `extractPipelineAndSchemas` — both mock and real must produce it |
| `io.ReadAll` on very large output | Memory spike | Philosopher output is bounded by token limits (max 500K tokens ≈ 2MB text). Acceptable. |
| Philosopher persona system prompt conflicts with meta instructions | LLM confusion in real mode | The `buildPhilosopherPrompt` overrides the system prompt with pipeline-specific instructions. Not addressing persona file changes in this PR — that's a design improvement for a follow-up. |

## Testing Strategy

1. **Unit test** (`internal/pipeline/meta_test.go`): Verify `extractPipelineAndSchemas` correctly parses the mock adapter's philosopher output
2. **Unit test** (`internal/adapter/mock.go`): Verify `generateMetaPhilosopherOutput` produces valid delimited format with parseable YAML and JSON schemas
3. **Command test** (`cmd/wave/commands/meta_test.go`): Add `TestMetaCommand_MockDryRun` that exercises `runMeta` with `--mock --dry-run` end-to-end
4. **Existing tests**: All existing meta tests continue to pass (verified: 19 pass currently)
5. **Race detection**: `go test -race ./...` must pass
