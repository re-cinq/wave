# Implementation Plan: Lifecycle Hooks

## Objective

Add a pipeline-level lifecycle hooks system that enables pipeline authors to define custom validation, notification, and guardrail logic at 10 lifecycle events. Hooks support 4 execution types (command, HTTP, LLM judge, script), regex-based step matching, blocking/non-blocking semantics, and fail-open/fail-closed behavior.

## Approach

### Strategy: New Package + Executor Integration

Create a new `internal/hooks/` package with clean separation of concerns:
- **Types** define the hook configuration schema (parsed from `wave.yaml`)
- **Executor** runs hooks with type-specific logic (command, HTTP, LLM, script)
- **Matcher** evaluates regex patterns against step IDs
- **Integration points** in `internal/pipeline/executor.go` call hook execution at lifecycle boundaries

The existing `manifest.HookConfig` (PreToolUse/PostToolUse) remains untouched — it serves Claude Code adapter-level hooks, which are a different concern.

### Design Decisions

1. **New package `internal/hooks/`** rather than adding to `internal/pipeline/` — hooks are a cross-cutting concern with HTTP/LLM dependencies that don't belong in the pipeline package
2. **Hooks defined at manifest level** (`wave.yaml` `hooks:` key) rather than per-pipeline — this matches the issue's design and allows global hooks (lint on every pipeline)
3. **HookRunner interface** for testability — mock the runner in executor tests
4. **Sequential hook execution** within an event — hooks for the same event fire in definition order, not concurrently, to ensure deterministic behavior
5. **Timeout support** with configurable per-hook timeout and a global default (30s for commands, 10s for HTTP, 60s for LLM)
6. **Environment variable expansion** in hook config values (URLs, commands) using `os.ExpandEnv`

## File Mapping

### New Files

| File | Purpose |
|------|---------|
| `internal/hooks/hooks.go` | Core types: `LifecycleHook`, `HookEvent`, `HookResult`, `HookDecision` |
| `internal/hooks/runner.go` | `HookRunner` interface and `DefaultHookRunner` implementation |
| `internal/hooks/command.go` | Command hook executor (shell execution, exit code interpretation) |
| `internal/hooks/http.go` | HTTP hook executor (POST event context, parse response) |
| `internal/hooks/llm.go` | LLM Judge hook executor (single-turn evaluation) |
| `internal/hooks/script.go` | Script hook executor (inline multi-line script via temp file) |
| `internal/hooks/matcher.go` | Regex-based step name matching |
| `internal/hooks/hooks_test.go` | Unit tests for types, matcher, and hook decision logic |
| `internal/hooks/runner_test.go` | Unit tests for runner orchestration |
| `internal/hooks/command_test.go` | Unit tests for command hook execution |
| `internal/hooks/http_test.go` | Unit tests for HTTP hook execution |
| `internal/hooks/llm_test.go` | Unit tests for LLM judge hook execution |
| `internal/hooks/script_test.go` | Unit tests for script hook execution |
| `internal/hooks/matcher_test.go` | Unit tests for regex matching |

### Modified Files

| File | Change |
|------|--------|
| `internal/manifest/types.go` | Add `Hooks []LifecycleHookDef` field to `Manifest` struct |
| `internal/manifest/parser.go` | Add validation for hooks section (valid events, types, matchers) |
| `internal/pipeline/executor.go` | Add `hookRunner` field; call hooks at 10 lifecycle points |
| `internal/pipeline/executor_test.go` | Add tests for hook integration in executor |

## Architecture Decisions

### AD-1: Manifest-Level Hooks (Not Pipeline-Level)

Hooks are defined in `wave.yaml` at the manifest root, not per-pipeline. This matches the issue's design:

```yaml
# wave.yaml
hooks:
  - name: lint-on-write
    event: step_completed
    ...
```

Pipeline-specific hooks could be added later by allowing a `hooks:` section in pipeline YAML files, but the initial scope is manifest-level only.

### AD-2: HookRunner Interface

```go
type HookRunner interface {
    RunHooks(ctx context.Context, event HookEvent) ([]HookResult, error)
}
```

The executor holds a `HookRunner` and calls it at lifecycle points. This allows:
- Easy testing with mock runners
- Clean separation between hook orchestration and execution
- Future extensibility (e.g., per-pipeline hook overrides)

### AD-3: Hook Event Context

Each hook receives a `HookEvent` with contextual information:

```go
type HookEvent struct {
    Type       EventType    // "run_start", "step_completed", etc.
    PipelineID string
    StepID     string       // Empty for run-level events
    Input      string       // Pipeline input
    Workspace  string       // Step workspace path (step-level events only)
    Artifacts  []string     // Artifact paths (artifact_created only)
    Error      string       // Error message (failed events only)
}
```

This context is serialized as JSON for HTTP hooks and made available as environment variables for command/script hooks.

### AD-4: Blocking Semantics

- **Blocking hooks**: If the hook returns a non-OK result, the step/pipeline fails. Multiple blocking hooks must ALL pass.
- **Non-blocking hooks**: Fire-and-forget. Failures are logged but don't affect pipeline flow.
- **Default blocking behavior** per event type:
  - `run_start`, `step_start`, `step_completed`: blocking=true
  - All others: blocking=false

### AD-5: Separate from Claude Code Hooks

The existing `manifest.HookConfig` (PreToolUse/PostToolUse) on `Persona` is for Claude Code's internal hook system. Pipeline lifecycle hooks are a completely separate system:

- **Claude Code hooks**: Run inside the adapter subprocess, configured via `settings.json`
- **Pipeline hooks**: Run by the Wave executor, between steps, at pipeline lifecycle boundaries

No naming collision since the manifest field is `Manifest.Hooks` (new, `[]LifecycleHookDef`) vs `Persona.Hooks` (existing, `HookConfig`).

## Risks

| Risk | Mitigation |
|------|-----------|
| Hook timeouts blocking pipeline indefinitely | Configurable per-hook timeout with sensible defaults; context cancellation |
| LLM Judge hooks requiring adapter infrastructure | Use simple `exec.Command` with `claude` CLI for LLM calls, avoiding adapter package dependency |
| Multiple hooks on same event creating confusing behavior | Execute in definition order, document clearly, emit events for observability |
| Hook failures masking contract validation results | Hooks run BEFORE contract validation on `step_completed`; contract validation is independent |
| Regex matcher compilation errors at runtime | Validate matchers during manifest parsing (fail fast at load time) |
| HTTP hooks leaking sensitive data | Sanitize event context through existing `security.InputSanitizer` before HTTP POST |
| Environment variable expansion in URLs | Use `os.ExpandEnv` which is safe; document that `$VAR` or `${VAR}` syntax works |

## Testing Strategy

### Unit Tests
- **Types**: Hook config parsing, default resolution, validation
- **Matcher**: Regex compilation, step name matching, empty/wildcard patterns
- **Command executor**: Exit code interpretation (0=pass, 2=block-with-reason, other=block)
- **HTTP executor**: JSON serialization, response parsing, timeout handling (use httptest.Server)
- **LLM executor**: Response parsing, ok/not-ok handling, fail-open behavior
- **Script executor**: Temp file creation, execution, cleanup
- **Runner**: Hook filtering by event type, matcher evaluation, blocking/non-blocking orchestration

### Integration Tests
- **Executor integration**: Verify hooks fire at correct lifecycle points using mock runner
- **End-to-end with command hooks**: Real shell execution in test environment
- **Blocking behavior**: Verify blocking hooks prevent step completion
- **Non-blocking behavior**: Verify non-blocking hooks don't affect pipeline flow

### Test Patterns
- Table-driven tests for matcher patterns and exit code interpretation
- `httptest.Server` for HTTP hook tests (no real network calls)
- `configCapturingAdapter` pattern for verifying executor calls hook runner at correct points
