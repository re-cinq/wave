# Implementation Plan: --model flag override

## Objective

Add a `--model` CLI flag to `wave run` (and `wave do`, `wave meta`) that overrides the persona-level model configuration from `wave.yaml` for a single invocation, without modifying the manifest.

## Approach

The model override needs to flow from CLI flag → RunOptions → executor → adapter config. The existing codebase already has model plumbing: `Persona.Model` in the manifest, `AdapterRunConfig.Model` in the adapter, and the `chat` command already has a `--model` flag. The implementation threads a new `ModelOverride` string through `ExecutorOption` into the executor, which applies it at step execution time when building `AdapterRunConfig`, respecting per-persona model pinning precedence.

### Precedence Rules (highest → lowest)

1. **Per-persona model** (`personas.<name>.model` in `wave.yaml`) — explicit pinning takes priority
2. **CLI `--model` flag** — runtime override for all unpinned steps
3. **Adapter default** (`"opus"` hardcoded in `ClaudeAdapter.buildArgs` and `prepareWorkspace`)

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `cmd/wave/commands/run.go` | modify | Add `Model` field to `RunOptions`, register `--model` flag, pass to executor |
| `cmd/wave/commands/do.go` | modify | Add `Model` field to `DoOptions`, register `--model` flag, pass to executor |
| `cmd/wave/commands/meta.go` | modify | Add `Model` field to `MetaOptions`, register `--model` flag, pass to executor |
| `internal/pipeline/executor.go` | modify | Add `modelOverride` field to `DefaultPipelineExecutor`, add `WithModelOverride` option, apply override in `runStepExecution` |
| `internal/pipeline/executor_test.go` | modify | Add tests for model override precedence |
| `cmd/wave/commands/run_test.go` | modify | Add test for `--model` flag registration |

## Architecture Decisions

### 1. Override at executor level, not manifest mutation

The override is stored on the executor (`DefaultPipelineExecutor.modelOverride`) and applied when building `AdapterRunConfig` in `runStepExecution`. This avoids mutating the shared `manifest.Manifest` or `manifest.Persona` structs, which would be error-prone for concurrent access and testing.

### 2. Per-persona pinning takes precedence

When a persona has an explicit `model` field in `wave.yaml`, that pinning is respected. The CLI `--model` flag only overrides steps whose persona does **not** have a model set. This matches the issue's acceptance criteria: "Per-step model pinning (if supported) takes precedence over the flag."

### 3. Pass-through model validation

Model identifiers are passed through to the adapter without validation against a hardcoded list. The adapter (Claude Code CLI) will produce a clear error if the model ID is invalid. This avoids maintaining a stale allowlist and works with new models automatically. However, an empty string check is added to reject `--model ""`.

### 4. Apply to `wave do` and `wave meta` as well

The issue mentions "any other relevant subcommands." Both `wave do` and `wave meta` create pipeline executors and would benefit from model override. The `wave chat` command already has its own `--model` flag with different semantics (interactive session), so it's left as-is.

## Risks

| Risk | Mitigation |
|------|------------|
| Model override silently ignored for pinned personas | Document precedence in help text; emit debug log when override is skipped |
| Invalid model strings cause confusing adapter errors | The adapter (Claude CLI) already produces clear error messages for invalid models |
| Breaking change if someone expects `--model` to override pinned personas | Document precedence clearly; this matches the issue's acceptance criteria |

## Testing Strategy

1. **Unit tests for flag registration**: Verify `--model` flag exists on `run`, `do`, and `meta` commands
2. **Unit tests for `WithModelOverride`**: Verify the executor option sets the field correctly
3. **Unit tests for precedence logic**: Test that:
   - When persona has no model set and override is provided → override is used
   - When persona has model pinned and override is provided → persona model is used
   - When no override is provided → existing default behavior is preserved
4. **Integration test**: End-to-end test with mock adapter verifying the model string reaches `AdapterRunConfig.Model`
