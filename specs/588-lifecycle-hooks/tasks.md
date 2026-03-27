# Tasks

## Phase 1: Core Types and Manifest Integration

- [X] Task 1.1: Create `internal/hooks/hooks.go` — define `LifecycleHookDef`, `EventType` constants (10 events), `HookType` constants (command/http/llm_judge/script), `HookEvent`, `HookResult`, `HookDecision`
- [X] Task 1.2: Create `internal/hooks/matcher.go` — regex-based step name matcher with compile-time validation
- [X] Task 1.3: Add `Hooks []hooks.LifecycleHookDef` field to `manifest.Manifest` struct in `internal/manifest/types.go`
- [X] Task 1.4: Add hook validation in `internal/manifest/parser.go` — validate event names, hook types, compile regex matchers at parse time

## Phase 2: Hook Executors

- [X] Task 2.1: Create `internal/hooks/command.go` — command hook executor (shell exec, exit code 0=pass/2=block-with-reason/other=block) [P]
- [X] Task 2.2: Create `internal/hooks/http.go` — HTTP hook executor (POST JSON event context, parse `{"ok": bool}` response) [P]
- [X] Task 2.3: Create `internal/hooks/llm.go` — LLM judge hook executor (single-turn `claude` CLI call, parse JSON response) [P]
- [X] Task 2.4: Create `internal/hooks/script.go` — script hook executor (temp file creation, execution, cleanup) [P]

## Phase 3: Hook Runner and Orchestration

- [X] Task 3.1: Create `internal/hooks/runner.go` — `HookRunner` interface and `DefaultHookRunner` with hook filtering (event type + matcher), sequential execution, blocking/non-blocking semantics, fail-open/fail-closed defaults
- [X] Task 3.2: Add `hookRunner` field to `DefaultPipelineExecutor` and wire through `ExecutorOption`
- [X] Task 3.3: Integrate hook calls into executor lifecycle — `run_start` (before first step), `run_completed`/`run_failed` (after pipeline completes)
- [X] Task 3.4: Integrate hook calls into step lifecycle — `step_start`, `step_completed`, `step_failed`, `step_retrying`
- [X] Task 3.5: Integrate hook calls for contract/artifact/workspace events — `contract_validated`, `artifact_created`, `workspace_created`
- [X] Task 3.6: Add hook event emission — emit `hook_started`, `hook_passed`, `hook_failed` events via EventEmitter for observability

## Phase 4: Testing

- [X] Task 4.1: Create `internal/hooks/hooks_test.go` — unit tests for types, defaults, validation
- [X] Task 4.2: Create `internal/hooks/matcher_test.go` — table-driven tests for regex matching [P]
- [X] Task 4.3: Create `internal/hooks/command_test.go` — tests for command execution and exit code interpretation [P]
- [X] Task 4.4: Create `internal/hooks/http_test.go` — tests with httptest.Server for HTTP hooks [P]
- [X] Task 4.5: Create `internal/hooks/llm_test.go` — tests for LLM judge response parsing [P]
- [X] Task 4.6: Create `internal/hooks/script_test.go` — tests for script execution [P]
- [X] Task 4.7: Create `internal/hooks/runner_test.go` — tests for runner orchestration, filtering, blocking semantics
- [X] Task 4.8: Add executor integration tests in `internal/pipeline/hooks_integration_test.go` — verify hooks fire at correct lifecycle points

## Phase 5: Validation

- [X] Task 5.1: Run `go test ./...` and `go vet ./...` — ensure all tests pass
- [X] Task 5.2: Run `golangci-lint run ./...` — fix any lint issues
- [X] Task 5.3: Verify no import cycles between `hooks` and `pipeline` packages
