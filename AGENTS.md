# Wave Development Guidelines

**Wave** is a multi-agent pipeline orchestrator written in Go that wraps Claude Code and other LLM CLIs via subprocess execution. It composes personas, pipelines, contracts, and relay/compaction into a continuous development system.

## ACCOUNTABILITY — YOU FOUND IT, YOU FIX IT

> **URGENT — NON-NEGOTIABLE**: This is the single most important rule in this file. It survives context compaction. Re-read it if you are unsure.

If you discover a problem — any problem — you own it. Fix it immediately.

There is NO concept of "pre-existing issue" in this project.
If you touched it or saw it break — fix it.

If a validation step reveals issues in files you didn't modify — fix those too.

Never argue about whether something is your responsibility.

## Critical Constraints

1. **Single static binary** — no runtime dependencies except adapter binaries
2. **Test ownership** — every failing test is YOUR concern. Fix or delete (with justification), never ignore. Changes to personas, pipelines, contracts, or meta-pipelines require `go test ./...`
3. **Security first** — all inputs validated, paths sanitized, permissions enforced
4. **Constitutional compliance** — navigator-first architecture, fresh memory at step boundaries, contract validation at handovers, ephemeral workspace isolation, observable progress events
5. **Observable execution** — structured progress events for monitoring
6. **No backward compatibility constraint** during prototype phase — move fast, let tests catch regressions
7. **No `t.Skip()`** without a linked issue. Delete tests only with clear justification

## How Wave Works at Runtime

Each pipeline is a **topologically-sorted DAG** of steps. For every step:

1. **Workspace creation** — an ephemeral worktree is created under `.wave/workspaces/<pipeline>/<step>/`. Steps can share workspaces via `workspace.ref`. Mounts support readonly/readwrite modes
2. **Artifact injection** — outputs from prior steps are injected into `.wave/artifacts/` before execution begins. The system validates existence, enforces optional/required semantics, and checks schemas if `ref.SchemaPath` is specified
3. **Runtime CLAUDE.md assembly** — a per-step CLAUDE.md is generated from four layers:
   - Base protocol preamble (`.wave/personas/base-protocol.md`)
   - Persona system prompt (role, responsibilities, constraints)
   - Contract compliance section (auto-generated from step contract schema)
   - Restriction section (denied/allowed tools, network domains from manifest permissions)
4. **Adapter execution** — the persona runs in isolated context with fresh memory (no chat history inheritance)
5. **Contract validation** — step output is validated against its contract (json_schema, typescript_interface, test_suite, markdown_spec, format) **before** marking the step successful. Hard failures block; soft failures log warnings

Key source files: `internal/pipeline/executor.go`, `internal/adapter/claude.go`, `internal/contract/`, `internal/workspace/`

## Architecture

### Active Technologies
- Go 1.25+ with `gopkg.in/yaml.v3`, `github.com/spf13/cobra`
- SQLite for pipeline state, filesystem for workspaces and artifacts

### Core Components
- **Manifests** (`wave.yaml`) — single source of truth for configuration
- **Personas** — AI agents with specific roles, permissions, and system prompts
- **Pipelines** — multi-step workflows with dependency resolution
- **Contracts** — output validation (JSON schema, TypeScript, test suites)
- **Workspaces** — ephemeral isolated execution environments
- **State Management** — SQLite-backed persistence and resumption

### Security Model
- Fresh memory at every step boundary — no chat history inheritance
- Permission enforcement with deny/allow patterns — strictly enforced
- Ephemeral workspaces — isolated filesystem execution
- Contract validation — all outputs validated before step completion
- Audit logging — credential scrubbing and tool call tracking

## File Structure
```
internal/
├── adapter/      # Subprocess execution and adapter management
├── audit/        # Audit logging and credential scrubbing
├── bench/        # SWE-bench benchmarking and comparison
├── continuous/   # Continuous pipeline execution
├── contract/     # Output validation (JSON, TypeScript, test suites)
├── cost/         # Cost ledger, iron rule enforcement, model pricing
├── defaults/     # Embedded default personas, pipelines, and contracts
├── deliverable/  # Pipeline deliverable tracking and output
├── display/      # Terminal progress display and formatting
├── doctor/       # Project health checking and optimization
├── event/        # Progress event emission and monitoring
├── forge/        # Git forge/hosting platform detection (GitHub, GitLab, Gitea, Forgejo, Codeberg, Bitbucket, local)
├── github/       # GitHub API integration for issue enhancement
├── hooks/        # Lifecycle hooks and webhook delivery runner
├── manifest/     # Configuration loading and validation
├── onboarding/   # Interactive wave init flow (monorepo-aware, Docker compose, flavour detection)
├── pathfmt/      # Path formatting and normalization utilities
├── pipeline/     # Pipeline execution, step management, model routing, decision logging
├── preflight/    # Pipeline dependency validation and auto-install
├── recovery/     # Pipeline recovery hints and error guidance
├── relay/        # Context compaction and summarization
├── retro/        # Run retrospective generation
├── sandbox/      # Docker and bubblewrap sandbox backends
├── scope/        # Persona token scope parsing and validation
├── security/     # Security validation and sanitization
├── skill/        # Skill discovery, provisioning, and command management
├── state/        # SQLite persistence, webhooks, decision log, ontology usage
├── suggest/      # Pipeline suggestion engine
├── tui/          # Bubble Tea terminal UI
├── webui/        # Web operations dashboard (runs, pipelines, webhooks, admin, analytics)
├── worktree/     # Git worktree lifecycle for isolated workspaces
└── workspace/    # Ephemeral workspace management

cmd/wave/         # CLI command structure
tests/            # Test coverage
.wave/            # Default personas, pipelines, contracts
```

## Active Adapters

| Adapter | Binary | Model Format | Instruction File |
|---------|--------|-------------|-----------------|
| Claude | `claude` | Short names or full IDs (`sonnet`, `haiku`) | `CLAUDE.md` |
| OpenCode | `opencode` | `provider/model` (`zai-coding-plan/glm-5-turbo`) | `AGENTS.md` |
| Gemini | `gemini` | Plain names (`gemini-2.0-pro`) | `GEMINI.md` |
| Codex | `codex` | `provider/model` or plain names | `AGENTS.md` |

### Override Hierarchy (strongest to weakest)

| Tier | Source | Adapter | Model |
|------|--------|---------|-------|
| 1 | CLI flags | `--adapter <name>` | `--model <model>` |
| 2 | Pipeline step YAML | `adapter:` | `model:` |
| 3 | Persona in wave.yaml | `adapter:` | `model:` |
| 4 | Adapter defaults | binary default | empty / auto-route |

```bash
wave run my-pipeline --adapter opencode --model "zai-coding-plan/glm-5-turbo"
wave run my-pipeline --adapter gemini --model "gemini-2.0-pro"
```

## Security

- All user input sanitized for prompt injection; file paths validated against approved directories
- Persona permissions strictly enforced at runtime; deny rules projected into `settings.json` AND runtime `CLAUDE.md`
- **Outer sandbox**: Nix dev shell with bubblewrap (read-only FS, hidden `$HOME`, curated env)
- **Adapter sandbox**: `settings.json` sandbox settings with network domain allowlisting
- **Prompt restrictions**: runtime `CLAUDE.md` restriction section generated from manifest
- **Environment hygiene**: only `runtime.sandbox.env_passthrough` vars reach subprocesses
- No credentials on disk; sanitized logging; workspace isolation prevents data leakage

## Development

### Code Standards
- Follow effective Go practices (`gofmt`, `go vet`), single responsibility per package
- Use interfaces for testability and dependency injection
- Comprehensive error types with structured details
- Table-driven tests with edge case coverage

### Testing
```bash
go test ./...            # Run all tests
go test -race ./...      # Run with race detector (required for PR)
golangci-lint run ./...  # Run static analysis linters
```

See `docs/migrations.md` for database migration documentation.

## Git Commits

- **No Co-Authored-By** — never include Co-Authored-By lines in commit messages
- **No AI attribution** — do not add "Generated with Claude Code" or similar
- Use conventional commit prefixes: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`

## Versioning

Automated semantic versioning from conventional commits. Every merge to `main` produces a release.

| Commit prefix | Bump | Example |
|---------------|------|---------|
| `fix:`, `docs:`, `refactor:`, `test:`, `chore:` | **patch** (0.0.X) | v0.1.0 → v0.1.1 |
| `feat:` | **minor** (0.X.0) | v0.1.1 → v0.2.0 |
| `BREAKING CHANGE:` or `!:` (e.g. `feat!:`) | **major** (X.0.0) | v0.2.0 → v1.0.0 |

## Debugging
- Use `--debug` flag for detailed execution logging
- Check `.wave/traces/` for audit logs
- Workspace contents preserved for post-mortem analysis

## Wave Swarm Orchestration

When acting as the **core orchestrator** (the Claude instance steering Wave pipelines), follow these patterns:

### Pipeline Selection

| Issue complexity | Pipeline | When to use |
|-----------------|----------|-------------|
| Bug fix, small tweak | `impl-issue` | Single-file or few-file changes, clear scope |
| Medium feature | `impl-issue` | Well-scoped feature with clear acceptance criteria |
| Complex feature | `impl-speckit` | Multi-component changes, needs spec → plan → tasks → impl |
| Architecture change | `impl-speckit` | Touches 5+ files, needs design discussion |
| Research then implement | `impl-research` | External integrations, unfamiliar APIs, need web research first |
| Code quality | `audit-junk-code`, `audit-dx`, `audit-dual` | Analysis and improvement |
| Security | `audit-security`, `wave-security-audit` | Security scanning (any project / Wave itself) |
| PR review | `ops-pr-review` | **Always** run before merging any PR |
| Wave bug fix | `wave-bugfix` | Fix bugs in Wave's own codebase |
| Wave evolution | `wave-evolve` | Evolve Wave pipelines, personas, and prompts |
| Wave test hardening | `wave-test-hardening` | Harden Wave's test suite — find gaps, add edge cases |
| Wave audit | `wave-audit` | Zero-trust implementation fidelity audit of Wave |
| Wave PR review | `wave-review` | Review Wave's own PRs |
| Epic decomposition | `plan-scope` | Decompose an epic into child issues |
| Issue research | `plan-research` | Research an issue and post findings |
| Stale issues | `ops-refresh` | Refresh a stale issue against recent codebase changes |
| Dead code | `audit-dead-code` | Detect and report unused code |
| Code simplification | `impl-recinq` | Divergent-convergent code simplification (Double Diamond) |
| SWE-bench benchmark | `bench-solve` | Solve a single SWE-bench task (used by `wave bench run`) |

### PR Review-Then-Merge Protocol

**MANDATORY**: Never merge a PR without a review. Before launching any pipeline, check existing state first to avoid redundant work.

1. **Check PR state first**: `gh pr view <N> --json reviews,comments` — if a completed `ops-pr-review` already posted a review, do NOT re-run the review pipeline
2. **Check for existing pipeline runs**: `wave list runs --limit 20` — if a review run already completed for this PR, skip re-running
3. Only launch review if no prior review exists: `wave run -v ops-pr-review -- "<PR-URL>" &`
4. Wait for review completion and check results
5. Only merge after review passes
6. Check for leaked files: `gh pr diff <N> --name-only | grep -E "^\.claude/|^\.wave/artifacts/|^\.wave/output/"`

### Concurrency Rules

- **Maximum 6 concurrent pipelines** — beyond this, API rate limits and CPU contention degrade quality
- **Optimal: 3-5 concurrent pipelines** — best throughput-to-quality ratio
- Launch pipelines via `wave run --detach -v <pipeline> -- "<input>"` (detached — survives shell exit)
- Monitor with `wave list runs --limit N` and `wave logs <run-id>`

### Issue Triage

Before launching a pipeline for an issue:
1. **Check if already implemented** — search codebase for the feature
2. **Check for duplicates/superseded** — compare with other open issues
3. **Assess implementability** — `implementable: false` is correct behavior, not a failure
4. **Close non-actionable issues** with a comment explaining why

| Category | Action | Pipeline |
|----------|--------|----------|
| **close** | Already implemented, superseded, or duplicate | Close with comment |
| **implement** | Well-scoped, single-PR implementation | `impl-issue` |
| **impl-speckit** | Complex, needs spec → plan → tasks → implement | `impl-speckit` |
| **defer** | Needs design discussion, experiment, or blocked | Leave open |

### Monitoring

```bash
wave list runs --limit N                                          # Status overview
wave logs <run-id> | grep -E "completed|running|validating"       # Step transitions
wave logs <run-id> | grep "stream_activity" | tail -3             # Latest activity
```

### Post-Pipeline PR Validation

1. **Check if review already exists**: `gh pr view <N> --json reviews,comments` — skip `ops-pr-review` if one already completed
2. If no review exists, run `ops-pr-review`: `wave run -v ops-pr-review -- "<PR-URL>" &`
3. Check for leaked files: `gh pr diff <N> --name-only | grep -E "^\.claude/|^\.wave/artifacts/|^\.wave/output/"`
4. After review passes: `gh pr merge <N> --merge`

### Failure Recovery

- **Merge conflicts**: Close old PR, `git pull origin main`, re-run pipeline
- **No-op PRs**: Close PR, re-run — review pipeline catches these
- **Contract failures**: Pipeline retries automatically
- **Rate limits**: Reduce concurrency, wait, re-run

### Batch PR Merge Protocol

1. Merge oldest/smallest PRs first to minimize cascading conflicts
2. Check remaining PRs for conflicts after each merge
3. Close CONFLICTING PRs and re-run from updated main
4. Pull main after batch: `git pull origin main`

## Ontology Context Injection

Ontology contexts (`wave.yaml` → `ontology.contexts`) are injected into step prompts to encode invariants, key decisions, and domain vocabulary. Understanding how context selection works is important for writing and debugging pipelines.

### Inherit-All vs Explicit Context Selection

**Explicit contexts** — a step with a `contexts:` list receives only the named contexts:

```yaml
- id: implement
  contexts: [execution, delivery]   # injects only execution + delivery invariants
```

**Inherit-all** — a step with **no** `contexts:` field automatically receives **all** defined contexts from `wave.yaml`:

```yaml
- id: plan
  # no contexts: field → injects ALL ontology contexts
```

This is intentional: the plan step typically needs the full domain picture, while implementation steps are narrowed to execution/delivery constraints. The trace log reveals the difference:

```
[ONTOLOGY_INJECT] step=plan    contexts=[quality,execution,delivery] invariants=11
[ONTOLOGY_INJECT] step=implement contexts=[execution,delivery]       invariants=7
```

### Undefined Context Warning

If a step's `contexts:` list references a context name that does **not** exist in `wave.yaml`, the runtime emits an `[ONTOLOGY_WARN]` log line and continues — the step runs unconstrained rather than failing:

```
[ONTOLOGY_WARN] pipeline=impl-issue step=fetch-assess undefined_contexts=[configuration]
```

This warning means the step received zero invariants from that context. Fix it by either adding the missing context to `wave.yaml` or correcting the context name in the pipeline step.

## Custom Pipeline Tips

- **Rapid prototyping**: Use `on_failure: skip` in contract blocks when creating new custom pipelines. This lets the pipeline complete even without schema files, making iteration fast. Graduate to `on_failure: retry` once the pipeline stabilizes.
- **Scaffolding**: Use `wave pipeline create --name my-pipeline --template impl-hotfix` to scaffold from an existing template.
- **Persona scaffolding**: Use `wave persona create --name my-persona --template researcher` to scaffold a custom persona.

## Constraints

1. NEVER write contract or artifact schemas in prompts. Wave has to parse, validate and inject them properly into the proper pipeline step. **Exception**: `gh pr create --body-file .wave/artifacts/<name>` and similar CLI commands that require a literal file path are acceptable — the persona needs the path to pass to external tools.
4. NEVER duplicate information in step prompts that Wave already injects at runtime. `buildContractPrompt()` automatically injects: (a) output artifact file paths and write instructions, (b) full contract schema content for `json_schema` contracts, (c) injected input artifact paths and read instructions (`## Available Artifacts` section). Therefore prompts must NOT specify: where to write output files, where to read injected artifacts, JSON field enumerations, or output structure that matches a contract schema. Reference artifacts by their `as:` alias name only (e.g., "Read the `greeting_file` artifact"), never by path. Prompts should focus on reasoning, constraints, quality bar, and domain-specific guidance. For markdown outputs without a schema contract, describing the expected section structure is acceptable since Wave only injects the file path, not the format.
2. NEVER pass validations silently. If a validation fails, it must be reported as an error and the step should not complete successfully.
3. NEVER make bulk-edits to the codebase except it is not functional code; BEWARE personas, pipelines etc. are all functional code and should be edited with the same care as any other code: Individually!

<!-- MANUAL ADDITIONS END -->
