# Tasks: Manifest & Pipeline Design

**Input**: Design documents from `/specs/014-manifest-pipeline-design/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: Tests are included as the spec involves safety-critical behavior (permission enforcement, contract validation, state persistence).

**Organization**: Tasks grouped by user story for independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup

**Purpose**: Go project initialization and shared infrastructure

- [ ] T001 Initialize Go module and install dependencies: `go mod init github.com/recinq/muzzle` then add cobra, yaml.v3, modernc.org/sqlite, jsonschema/v6 in go.mod
- [ ] T002 Create CLI entry point with cobra root command and subcommand stubs (init, validate, run, do, resume, clean) in cmd/muzzle/main.go
- [ ] T003 [P] Create structured event emitter (NDJSON to stdout) in internal/event/emitter.go implementing EventEmitter interface from contracts
- [ ] T004 [P] Create audit logger with credential scrubbing in internal/audit/logger.go implementing AuditLogger interface from contracts

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core types and packages that ALL user stories depend on

**CRITICAL**: No user story work can begin until this phase is complete

- [ ] T005 Define manifest Go types (Manifest, Adapter, Persona, Permissions, HookConfig, HookRule, Runtime, RelayConfig, AuditConfig, MetaConfig, SkillMount) in internal/manifest/types.go matching data-model.md
- [ ] T006 Define pipeline Go types (Pipeline, Step, MemoryConfig, WorkspaceConfig, Mount, ExecConfig, ArtifactDef, ArtifactRef, HandoverConfig, ContractConfig, CompactionConfig, MatrixStrategy, ValidationRule, StepState constants) in internal/pipeline/types.go matching data-model.md
- [ ] T007 [P] Define adapter interface and AdapterRunConfig/AdapterResult types in internal/adapter/adapter.go matching contracts/interfaces.go
- [ ] T008 [P] Create mock adapter for testing (returns configurable stdout JSON, exit codes, token counts) in internal/adapter/mock.go
- [ ] T009 [P] Create workspace manager (create dirs, set permissions, copy files, inject artifacts, cleanup) in internal/workspace/workspace.go implementing WorkspaceManager interface
- [ ] T010 [P] Create SQLite state store with schema (pipeline_state, step_state tables) in internal/state/store.go and internal/state/schema.sql implementing StateStore interface
- [ ] T011 [P] Write table-driven tests for workspace manager (create, inject, clean) in internal/workspace/workspace_test.go
- [ ] T012 [P] Write table-driven tests for state store (save/get pipeline, save/get steps, resume queries) in internal/state/store_test.go

**Checkpoint**: Foundation ready — all core types defined, workspace and state packages tested

---

## Phase 3: User Story 1 — Define Project Manifest (Priority: P1)

**Goal**: Developer can create, parse, and validate `muzzle.yaml`

**Independent Test**: Run `muzzle init` then `muzzle validate` and confirm all references resolve

### Tests for User Story 1

- [ ] T013 [P] [US1] Write tests for manifest YAML parsing (valid manifest, missing fields, unknown adapters, invalid YAML) in internal/manifest/parser_test.go
- [ ] T014 [P] [US1] Write tests for manifest validation (missing prompt files, missing adapter refs, missing hook scripts, binary-on-PATH check) in internal/manifest/parser_test.go

### Implementation for User Story 1

- [ ] T015 [US1] Implement manifest YAML parser using yaml.v3 with line-number error reporting in internal/manifest/parser.go
- [ ] T016 [US1] Implement manifest validator (resolve all persona→adapter refs, check file existence for system_prompt_file and hook commands, warn on missing binaries) in internal/manifest/parser.go
- [ ] T017 [US1] Implement `muzzle init` subcommand: generate scaffold muzzle.yaml with default claude adapter, navigator+craftsman personas, runtime defaults, and create .muzzle/personas/ directory with example prompt files in cmd/muzzle/main.go
- [ ] T018 [US1] Implement `muzzle validate` subcommand: load manifest, run validator, print errors/warnings with file paths and line numbers in cmd/muzzle/main.go

**Checkpoint**: `muzzle init && muzzle validate` works end-to-end

---

## Phase 4: User Story 2 — Run a Pipeline End-to-End (Priority: P1)

**Goal**: Developer can trigger a named pipeline that executes a DAG of steps in dependency order

**Independent Test**: Run `muzzle run --pipeline test.yaml --input "test"` with mock adapter and observe steps execute in order with artifact flow

### Tests for User Story 2

- [ ] T019 [P] [US2] Write tests for DAG resolution (topological sort, cycle detection, dependency validation) in internal/pipeline/dag_test.go
- [ ] T020 [P] [US2] Write tests for pipeline executor (sequential steps, retry on failure, halt on max retries, state transitions Pending→Running→Completed/Failed/Retrying) in internal/pipeline/executor_test.go

### Implementation for User Story 2

- [ ] T021 [US2] Implement pipeline YAML parser in internal/pipeline/types.go (reuse yaml.v3, validate step IDs unique, persona refs valid)
- [ ] T022 [US2] Implement DAG resolver: topological sort of steps by dependencies, cycle detection via DFS in internal/pipeline/dag.go
- [ ] T023 [US2] Implement pipeline executor: iterate topologically sorted steps, for each step create workspace → inject artifacts → invoke adapter → validate contract → persist state in internal/pipeline/executor.go
- [ ] T024 [US2] Implement step retry logic: on failure or contract violation, transition to Retrying, re-execute up to max_retries, then transition to Failed in internal/pipeline/executor.go
- [ ] T025 [US2] Implement artifact injection: copy output artifacts from completed step workspaces into dependent step workspaces at configured paths in internal/pipeline/executor.go
- [ ] T026 [US2] Implement `muzzle run` subcommand: parse --pipeline and --input flags, load manifest + pipeline YAML, execute pipeline, emit progress events in cmd/muzzle/main.go
- [ ] T027 [US2] Implement `muzzle resume` subcommand: load state from SQLite, find last completed step, re-execute from next pending step in cmd/muzzle/main.go

**Checkpoint**: A multi-step pipeline with mock adapter runs to completion, artifacts flow between steps, retries work

---

## Phase 5: User Story 3 — Persona-Scoped Agent Execution (Priority: P1)

**Goal**: Each step runs an agent configured by its persona with enforced permissions, hooks, and temperature

**Independent Test**: Run a navigator persona and confirm it can read but not write; run a craftsman with a PreToolUse hook that blocks commits when tests fail

### Tests for User Story 3

- [ ] T028 [P] [US3] Write tests for Claude adapter subprocess invocation (build command args, set env vars, stream stdout, handle exit codes, timeout kill) in internal/adapter/adapter_test.go
- [ ] T029 [P] [US3] Write tests for permission enforcement (deny patterns block matching tools, allowed patterns permit, deny takes precedence) in internal/adapter/adapter_test.go

### Implementation for User Story 3

- [ ] T030 [US3] Implement Claude Code adapter: build `claude -p` command with --allowedTools, --output-format json, persona temperature; set up process group for clean timeout kill in internal/adapter/claude.go
- [ ] T031 [US3] Implement permission enforcement: before invoking subprocess, generate .claude/settings.json in workspace with allowed_tools and deny patterns from persona config in internal/adapter/claude.go
- [ ] T032 [US3] Implement system prompt projection: copy persona's system_prompt_file as CLAUDE.md into workspace root before subprocess invocation in internal/adapter/claude.go
- [ ] T033 [US3] Implement hook configuration: generate hooks config in .claude/settings.json from persona's PreToolUse/PostToolUse rules in internal/adapter/claude.go
- [ ] T034 [US3] Implement per-step timeout: use context.WithTimeout on subprocess, kill process group on timeout, transition step to Retrying in internal/adapter/adapter.go
- [ ] T035 [US3] Integrate persona binding into pipeline executor: for each step, look up persona in manifest, configure adapter with persona settings before invocation in internal/pipeline/executor.go

**Checkpoint**: Pipeline steps run with correct persona permissions, hooks fire, timeouts kill subprocesses

---

## Phase 6: User Story 4 — Context Relay and Compaction (Priority: P2)

**Goal**: When an agent nears its context limit, a summarizer compacts the session and a fresh instance resumes

**Independent Test**: Run an agent with a small context window, observe relay trigger, verify checkpoint is produced and resumed agent continues without repeating work

### Tests for User Story 4

- [ ] T036 [P] [US4] Write tests for relay monitor (threshold detection, compact call, checkpoint structure validation) in internal/relay/relay_test.go

### Implementation for User Story 4

- [ ] T037 [US4] Implement token usage parser: extract token counts from Claude Code JSON stdout stream in internal/adapter/claude.go
- [ ] T038 [US4] Implement relay monitor: compare cumulative tokens against threshold percentage of model context window, signal compaction needed in internal/relay/relay.go
- [ ] T039 [US4] Implement compaction: spawn summarizer persona subprocess with chat history as input, capture checkpoint.md output in internal/relay/relay.go
- [ ] T040 [US4] Implement checkpoint injection: when resuming after compaction, inject checkpoint.md into workspace and prepend "Read checkpoint.md first" to the prompt in internal/relay/checkpoint.go
- [ ] T041 [US4] Integrate relay into pipeline executor: after each adapter response, check relay monitor; if triggered, compact and restart step with checkpoint in internal/pipeline/executor.go

**Checkpoint**: Relay triggers at threshold, summarizer produces checkpoint, resumed agent continues from checkpoint

---

## Phase 7: User Story 5 — Handover Contracts Between Steps (Priority: P2)

**Goal**: Step boundaries validate output artifacts against typed contracts before allowing progression

**Independent Test**: Define a step with a JSON schema contract, produce invalid output, confirm retry

### Tests for User Story 5

- [ ] T042 [P] [US5] Write tests for JSON schema validator (valid passes, missing required field fails, type mismatch fails) in internal/contract/contract_test.go
- [ ] T043 [P] [US5] Write tests for TypeScript interface validator (valid .ts compiles, syntax error fails, missing field fails) in internal/contract/contract_test.go
- [ ] T044 [P] [US5] Write tests for test suite validator (all pass succeeds, any failure fails, command not found fails) in internal/contract/contract_test.go

### Implementation for User Story 5

- [ ] T045 [US5] Implement contract interface and dispatcher (route to json_schema, typescript_interface, or test_suite validator by type) in internal/contract/contract.go
- [ ] T046 [US5] Implement JSON schema validator using jsonschema/v6: load schema from inline string or file, validate artifact file against it in internal/contract/jsonschema.go
- [ ] T047 [US5] Implement TypeScript interface validator: run `tsc --noEmit` on the contract file, parse exit code and stderr for errors; degrade gracefully if tsc not on PATH in internal/contract/typescript.go
- [ ] T048 [US5] Implement test suite validator: run configured command, check exit code, capture stderr for error reporting in internal/contract/testsuite.go
- [ ] T049 [US5] Integrate contract validation into pipeline executor: after step completes, run handover contract validator; on failure trigger retry in internal/pipeline/executor.go

**Checkpoint**: All three contract types validate correctly, invalid artifacts trigger retries

---

## Phase 8: User Story 6 — Ad-Hoc Task Execution (Priority: P2)

**Goal**: `muzzle do "task"` generates and runs a minimal navigate→execute pipeline

**Independent Test**: Run `muzzle do "fix typo"` and confirm navigator runs first, then craftsman, both in ephemeral workspaces

### Implementation for User Story 6

- [ ] T050 [US6] Implement ad-hoc pipeline generator: given an input string, persona override, and manifest, produce an in-memory Pipeline with navigate + execute steps in internal/pipeline/adhoc.go
- [ ] T051 [US6] Implement `muzzle do` subcommand: parse input string, --persona flag, --save flag; generate pipeline, execute it, optionally write YAML to --save path in cmd/muzzle/main.go
- [ ] T052 [US6] Write tests for ad-hoc pipeline generation (default personas, persona override, save to file) in internal/pipeline/adhoc_test.go

**Checkpoint**: `muzzle do` generates correct 2-step pipeline and executes it

---

## Phase 9: User Story 7 — Meta-Pipeline (Priority: P3)

**Goal**: Philosopher persona designs a custom pipeline at runtime; runtime validates and executes it with bounded recursion

**Independent Test**: Route a novel task to meta pipeline, confirm philosopher generates valid pipeline that executes

### Implementation for User Story 7

- [ ] T053 [US7] Implement meta-pipeline executor: load meta.yaml template, execute philosopher step, validate generated pipeline YAML (schema + semantic checks), execute child pipeline with incremented depth in internal/pipeline/meta.go
- [ ] T054 [US7] Implement recursion depth tracking: pass --parent-pipeline flag to child pipelines, check depth against max_depth in manifest, block meta steps at limit in internal/pipeline/meta.go
- [ ] T055 [US7] Implement semantic validation for generated pipelines: step[0] must use navigator, all steps must have handover contracts, all steps must use fresh memory in internal/pipeline/meta.go
- [ ] T056 [US7] Write tests for meta-pipeline (valid generation, recursion depth limit, semantic validation failure) in internal/pipeline/meta_test.go

**Checkpoint**: Meta-pipeline generates, validates, and executes child pipelines with bounded recursion

---

## Phase 10: User Story 2 (continued) — Matrix Strategy (Priority: P1)

**Goal**: Pipeline steps with matrix strategy fan out into parallel workers

**Independent Test**: Define a step with matrix strategy over 3 tasks, confirm 3 parallel workers launch

### Implementation for Matrix Strategy

- [ ] T057 [US2] Implement matrix strategy: parse items_source file (JSON array), spawn one goroutine per item (up to max_concurrency), each with isolated workspace and injected task context in internal/pipeline/matrix.go
- [ ] T058 [US2] Implement matrix worker coordination: use errgroup.Group for parallel execution, collect all results, fail pipeline if any worker fails after retries in internal/pipeline/matrix.go
- [ ] T059 [US2] Write tests for matrix strategy (fan-out to N workers, concurrency limit respected, one failure fails pipeline, all success proceeds) in internal/pipeline/matrix_test.go

**Checkpoint**: Matrix steps spawn parallel workers, respect concurrency limits, coordinate results

---

## Phase 11: VitePress Documentation (P1)

**Goal**: Comprehensive documentation site that serves as authoritative guide for Muzzle

**Independent Test**: Documentation builds successfully, all links resolve, all examples validate

### Documentation Setup

- [ ] T060 [P] Initialize VitePress site in docs/ directory with TypeScript config and basic theme
- [ ] T061 [P] Create custom Vue components: MuzzleConfig (manifest editor), PipelineVisualizer (DAG), TerminalOutput (styled output), PersonaCard in docs/.vitepress/theme/components/
- [ ] T062 [P] Set up Mermaid plugin for diagrams and search plugin with custom index
- [ ] T063 Create site configuration (nav, sidebar, social links) in docs/.vitepress/config.ts
- [ ] T064 [P] Create logo.svg, favicon.ico, and og-image.png in docs/public/

### Core Documentation Pages

- [ ] T065 Write landing page (index.md) with hero section and feature highlights
- [ ] T066 Write installation guide (docs/guide/installation.md) with binary install and from-source instructions
- [ ] T067 Write quick-start guide (docs/guide/quick-start.md) with 5-minute first-run walkthrough
- [ ] T068 [P] Write configuration guide (docs/guide/configuration.md) with complete manifest reference and examples
- [ ] T069 Write personas guide (docs/guide/personas.md) explaining persona system, permissions, and hooks
- [ ] T070 [P] Write pipelines guide (docs/guide/pipelines.md) with DAG concepts, step types, and patterns
- [ ] T071 [P] Write contracts guide (docs/guide/contracts.md) explaining handover contracts and validation types
- [ ] T072 [P] Write relay guide (docs/guide/relay.md) explaining context compaction and checkpoint mechanism

### Reference Documentation

- [ ] T073 [P] Generate CLI reference (docs/reference/cli.md) from Cobra commands with examples
- [ ] T074 [P] Generate manifest schema (docs/reference/manifest-schema.md) from Go struct tags
- [ ] T075 [P] Generate pipeline schema (docs/reference/pipeline-schema.md) from pipeline types
- [ ] T076 [P] Write adapters reference (docs/reference/adapters.md) with Claude Code and future adapter configs
- [ ] T077 [P] Write troubleshooting guide (docs/reference/troubleshooting.md) with common issues and solutions

### Tutorials

- [ ] T078 Write first project tutorial (docs/tutorials/first-project.md) with step-by-step walkthrough
- [ ] T079 Write custom personas tutorial (docs/tutorials/custom-personas.md) with real-world examples
- [ ] T080 Write pipeline design tutorial (docs/tutorials/pipeline-design.md) with effective patterns
- [ ] T081 Write meta-pipelines tutorial (docs/tutorials/meta-pipelines.md) with self-designing examples
- [ ] T082 Write CI integration tutorial (docs/tutorials/ci-integration.md) for GitHub Actions setup

### Examples and Concepts

- [ ] T083 Write examples index and 4 detailed examples (simple-feature, bug-fix, refactoring, multi-persona) in docs/examples/
- [ ] T084 Write architecture concepts (docs/concepts/architecture.md) with system overview and diagrams
- [ ] T085 [P] Write isolation concepts (docs/concepts/isolation.md) explaining workspace security model
- [ ] T086 [P] Write state management concepts (docs/concepts/state-management.md) with SQLite persistence details
- [ ] T087 [P] Write security concepts (docs/concepts/security.md) covering permissions and credential handling
- [ ] T088 [P] Write performance concepts (docs/concepts/performance.md) with optimization guidelines

### Development Documentation

- [ ] T089 Write contributing guide (docs/development/contributing.md) with PR process and standards
- [ ] T090 Write architecture decisions (docs/development/architecture-decisions.md) documenting ADRs
- [ ] T091 Write building guide (docs/development/building.md) for building from source
- [ ] T092 Write release process (docs/development/release-process.md) with versioning and release steps

### Diagrams and Assets

- [ ] T093 [P] Create Mermaid diagrams: manifest-flow, pipeline-execution, persona-binding, relay-flow in docs/assets/diagrams/
- [ ] T094 [P] Create screenshots: CLI output, manifest example, pipeline progress in docs/assets/images/
- [ ] T095 Add interactive examples with runnable code blocks where appropriate

### Automation and Quality

- [ ] T096 Set up doc generation scripts for CLI reference and schemas from Go code
- [ ] T097 Set up CI/CD for docs: build, validate links, deploy to GitHub Pages
- [ ] T098 Add pre-commit hooks: spell check, validate examples, check internal links
- [ ] T099 Create example validation script to ensure all code examples work with current implementation

---

## Phase 12: Polish & Cross-Cutting Concerns

**Purpose**: Integration, cleanup, and validation across all stories

- [ ] T100 Implement `muzzle clean` subcommand: delete workspace directories by pipeline ID or all in cmd/muzzle/main.go
- [ ] T101 Implement `muzzle run --dry-run` mode: walk pipeline DAG emitting step transitions without invoking adapters in internal/pipeline/executor.go
- [ ] T102 [P] Implement input routing: match work item labels against routing rules in manifest to select pipeline in internal/pipeline/router.go
- [ ] T103 [P] Implement structured progress events throughout executor: emit Event on every state transition with timestamp, pipeline_id, step_id, state, duration in internal/pipeline/executor.go
- [ ] T104 Wire audit logger into adapter invocations: log tool calls and file operations when audit.log_all_tool_calls is enabled in internal/adapter/adapter.go
- [ ] T105 Run `go vet ./...` and `go test ./...` across all packages, fix any issues
- [ ] T106 Run quickstart.md validation: execute `muzzle init && muzzle validate && muzzle run --dry-run` end-to-end
- [ ] T107 Build documentation site and validate all links resolve
- [ ] T108 Verify all documentation examples work with current implementation

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS all user stories
- **US1 Manifest (Phase 3)**: Depends on Foundational
- **US2 Pipeline (Phase 4)**: Depends on Foundational + US1 (needs manifest parser)
- **US3 Persona (Phase 5)**: Depends on US2 (needs pipeline executor)
- **US4 Relay (Phase 6)**: Depends on US3 (needs adapter integration)
- **US5 Contracts (Phase 7)**: Depends on US2 (needs pipeline executor for integration)
- **US6 Ad-Hoc (Phase 8)**: Depends on US2 + US3 (needs executor + persona binding)
- **US7 Meta (Phase 9)**: Depends on US2 + US5 (needs executor + contract validation)
- **Matrix (Phase 10)**: Depends on US2 (extends executor)
- **VitePress Docs (Phase 11)**: Can start after Phase 1, runs in parallel with development
- **Polish (Phase 12)**: Depends on all user stories + documentation

### Parallel Opportunities

After Foundational completes:
- US1 (Manifest) can start immediately
- Documentation (Phase 11) can start immediately after Setup
- After US1: US2 (Pipeline) starts
- After US2: US3, US5, US6, Matrix can start in parallel
- After US3: US4 starts
- After US2 + US5: US7 starts

Within each phase, tasks marked [P] can run in parallel.

### Within Each User Story

- Tests written FIRST, verify they FAIL before implementation
- Types/models before services
- Services before CLI integration
- Core implementation before integration with executor
- Story complete before moving to next priority

---

## Implementation Strategy

### MVP First (User Story 1 + 2 + 3)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: US1 — Manifest parsing and validation
4. Complete Phase 4: US2 — Pipeline DAG execution
5. Complete Phase 5: US3 — Persona-scoped adapter invocation
6. **STOP and VALIDATE**: `muzzle init && muzzle validate && muzzle run --pipeline test.yaml` works end-to-end with mock adapter
7. This gives you: manifest config → pipeline execution → persona-scoped agents. The core loop works.

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. Add US1 (Manifest) → `muzzle init && muzzle validate` works
3. Add US2 (Pipeline) → `muzzle run` executes DAG with mock adapter
4. Add US3 (Persona) → Real adapter invocation with permissions/hooks
5. Add US4 (Relay) → Long-running tasks survive context limits
6. Add US5 (Contracts) → Step boundaries enforce quality
7. Add US6 (Ad-hoc) → `muzzle do` for quick tasks
8. Add US7 (Meta) → Self-designing pipelines
9. Add Matrix → Parallel task execution
10. Polish → Production-ready

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story
- Each user story is independently completable and testable after its dependencies
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Pure Go SQLite (`modernc.org/sqlite`) — no CGo, single binary
- Mock adapter used for all tests until US3 introduces real Claude adapter
