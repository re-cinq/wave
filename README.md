# Wave: Multi-Agent Pipeline Orchestrator

Wave coordinates multiple AI personas through structured pipelines for AI-assisted development workflows, enforcing permissions, contracts, and workspace isolation at every step.

**Note**: Wave requires AI adapter binaries to be installed and available in PATH. Currently supports `claude`, `opencode`, and `dummy` adapters.

## Quick Start

### 1. Build and Install

```bash
# Build the CLI
go build -o wave ./cmd/wave

# (Optional) Install to PATH
sudo mv wave /usr/local/bin/
```

### 2. Set Up Adapter (Required)

**Option A: Use Real AI Adapter**
```bash
# Install Claude CLI or OpenCode CLI to PATH
# Then use:
wave init --adapter claude
```

**Option B: Create Dummy Adapter for Testing**
```bash
# Create dummy adapter script
echo '#!/bin/bash
echo "Dummy adapter response for: $*"' > /usr/local/bin/dummy
chmod +x /usr/local/bin/dummy

# Initialize with dummy adapter
wave init --adapter dummy
```

### 3. Initialize Project

```bash
cd your-project
wave init --adapter <claude|opencode|dummy>
```

This creates:
- `wave.yaml` - Project manifest with adapters and personas
- `.wave/personas/` - AI persona system prompts
- `.wave/pipelines/` - Example pipeline definitions

### 4. Configure Personas

The dummy adapter initialization creates a minimal manifest. You need to add required personas:

```yaml
# wave.yaml
apiVersion: v1
kind: Wave
metadata:
  name: my-project
  description: "My Wave project"

adapters:
  dummy:
    binary: dummy
    mode: agent
    outputFormat: markdown

personas:
  navigator:
    adapter: dummy
    systemPromptFile: .wave/personas/navigator.md
    temperature: 0.1
    description: "Codebase navigator and analyzer"
  
  craftsman:
    adapter: dummy
    systemPromptFile: .wave/personas/craftsman.md
    temperature: 0.7
    description: "Implementation specialist"

runtime:
  workspaceRoot: ./workspace
  defaultTimeoutMin: 10
  maxConcurrentWorkers: 4
```

Create the persona files:
```bash
mkdir -p .wave/personas
echo "You are a code navigator. Analyze and understand codebases." > .wave/personas/navigator.md
echo "You are a craftsman developer. Implement high-quality code." > .wave/personas/craftsman.md
```

### 5. Validate Configuration

```bash
wave validate
```

### 6. Run Your First Pipeline

```bash
# Quick task without creating a pipeline file
wave do "fix the typo in README.md line 42"

# Or run a full pipeline (requires pipeline file)
wave run --pipeline speckit-flow --input "add user authentication"
```

## Command Reference

### `wave init`

Initialize a new Wave project with default configuration.

```bash
wave init [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--adapter` | `claude` | Default adapter to use (`claude`, `opencode`, `dummy`) |
| `--force` | `false` | Overwrite existing files |
| `--workspace` | `.wave/workspaces` | Workspace directory path |
| `--output` | `wave.yaml` | Output path for manifest file |

**Examples:**
```bash
wave init                           # Initialize with Claude adapter
wave init --adapter opencode        # Use OpenCode adapter
wave init --force                   # Overwrite existing configuration
```

**Creates:**
- `wave.yaml` — Project manifest with adapters and personas
- `.wave/personas/` — AI persona system prompts (navigator, philosopher, craftsman, auditor, summarizer)
- `.wave/pipelines/` — Example pipelines (speckit-flow, hotfix)
- `.wave/contracts/` — JSON schema contracts for validation
- `.wave/workspaces/` — Ephemeral workspace root
- `.wave/traces/` — Audit log directory

---

### `wave do`

Execute an ad-hoc task with a minimal navigate→execute pipeline.

```bash
wave do <task description> [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--persona` | `craftsman` | Override execute persona |
| `--save` | — | Save generated pipeline YAML to path |
| `--manifest` | `wave.yaml` | Path to manifest file |
| `--mock` | `false` | Use mock adapter (for testing) |
| `--dry-run` | `false` | Show execution plan without running |

**Examples:**
```bash
# Quick fixes
wave do "fix the typo in README.md"
wave do "fix the broken import in auth.go"
wave do "update the copyright year in LICENSE"

# Small features
wave do "add a health check endpoint at /healthz"
wave do "add input validation to the signup form"
wave do "add retry logic to the HTTP client"

# Code reviews with auditor persona
wave do "audit auth middleware for SQL injection" --persona auditor
wave do "review the payment module for security issues" --persona auditor
wave do "check error handling in the API layer" --persona auditor

# Save pipeline for reuse
wave do "add dark mode toggle" --save dark-mode.yaml
wave do "implement caching layer" --save caching.yaml

# Preview without executing
wave do "refactor user service" --dry-run
```

---

### `wave run`

Execute a defined pipeline.

```bash
wave run --pipeline <name> [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--pipeline` | — | Pipeline name to run (required) |
| `--input` | — | Input data for the pipeline |
| `--dry-run` | `false` | Show execution plan without running |
| `--from-step` | — | Start execution from specific step |
| `--timeout` | manifest default | Timeout in minutes |
| `--manifest` | `wave.yaml` | Path to manifest file |
| `--mock` | `false` | Use mock adapter (for testing) |

**Examples:**
```bash
wave run --pipeline speckit-flow --input "add user authentication"
wave run --pipeline hotfix --input "fix memory leak in cache.go"
wave run --pipeline speckit-flow --dry-run
wave run --pipeline speckit-flow --from-step implement
wave run --pipeline speckit-flow --timeout 60
```

---

### `wave list`

List available pipelines, personas, and adapters.

```bash
wave list [pipelines|personas|adapters] [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--manifest` | `wave.yaml` | Path to manifest file |
| `--format` | `table` | Output format (`table`, `json`) |

**Examples:**
```bash
wave list                   # List everything
wave list pipelines         # List available pipelines
wave list personas          # List configured personas
wave list adapters          # List configured adapters
```

---

### `wave resume`

Resume a paused or failed pipeline execution.

```bash
wave resume --pipeline <id> [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--pipeline` | — | Pipeline ID to resume (required) |
| `--from-step` | — | Resume from specific step |
| `--manifest` | `wave.yaml` | Path to manifest file |

**Examples:**
```bash
wave resume --pipeline abc123
wave resume --pipeline abc123 --from-step implement
```

---

### `wave clean`

Remove generated workspaces, state files, and cached artifacts.

```bash
wave clean [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--all` | `false` | Clean all workspaces and state |
| `--pipeline` | — | Clean specific pipeline workspace |
| `--force` | `false` | Skip confirmation |

**Examples:**
```bash
wave clean --all                    # Remove all workspaces and state
wave clean --pipeline speckit-flow  # Clean specific pipeline
wave clean --all --force            # Clean without confirmation
```

**Removes:**
- `.wave/state.db` — Pipeline state database
- `.wave/traces/` — Audit logs
- `.wave/workspaces/` — All ephemeral workspaces

---

### `wave validate`

Validate Wave configuration and project structure.

```bash
wave validate [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--manifest` | `wave.yaml` | Path to manifest file |
| `--pipeline` | — | Specific pipeline to validate |
| `-v, --verbose` | `false` | Verbose output |

**Examples:**
```bash
wave validate                              # Validate everything
wave validate --verbose                    # Show detailed validation steps
wave validate --pipeline speckit-flow      # Validate specific pipeline
```

**Checks:**
- Manifest syntax and structure
- Required fields (apiVersion, kind, metadata.name)
- Adapter binary availability in PATH
- Persona system prompt file existence
- Pipeline step dependencies and persona references

---

## Built-in Personas

Wave ships with 7 specialized personas, each with role-specific permissions:

| Persona | Temperature | Description | Permissions |
|---------|-------------|-------------|-------------|
| **navigator** | 0.1 | Read-only codebase exploration and analysis | `Read`, `Glob`, `Grep`, `Bash(git log*)`, `Bash(git status*)` |
| **philosopher** | 0.3 | Architecture design and specification | `Read`, `Write(.wave/specs/*)` |
| **planner** | 0.3 | Task breakdown and project planning | `Read`, `Glob`, `Grep` |
| **craftsman** | 0.7 | Code implementation and testing | `Read`, `Write`, `Edit`, `Bash` |
| **debugger** | 0.2 | Systematic issue diagnosis and root cause analysis | `Read`, `Grep`, `Bash(git bisect*)`, `Bash(go test*)` |
| **auditor** | 0.1 | Security review and quality assurance | `Read`, `Grep`, `Bash(go vet*)`, `Bash(npm audit*)` |
| **summarizer** | 0.0 | Context compaction for relay handoffs | `Read` only |

### Persona Capabilities

**navigator** — Explores the codebase without modifications
- Searches and reads source files
- Maps dependencies between modules
- Identifies patterns and conventions
- Outputs structured JSON analysis

**philosopher** — Designs architecture and specifications
- Creates feature specifications with user stories
- Designs data models and API schemas
- Identifies edge cases and security considerations
- Cannot execute shell commands

**planner** — Breaks down complex tasks
- Decomposes features into atomic tasks
- Identifies dependencies between tasks
- Estimates complexity (S/M/L/XL)
- Defines acceptance criteria

**craftsman** — Implements features and fixes
- Writes production code following specifications
- Creates unit and integration tests
- Runs test suites to verify correctness
- Full read/write access to codebase

**debugger** — Diagnoses issues systematically
- Forms and tests hypotheses about root causes
- Uses git bisect to find regressions
- Traces execution paths and data flow
- Documents findings for fixes

**auditor** — Reviews for security and quality
- Checks for OWASP Top 10 vulnerabilities
- Verifies authentication and authorization
- Assesses test coverage
- Read-only access (cannot modify code)

**summarizer** — Compacts conversation context
- Distills long conversations into checkpoints
- Preserves key decisions and technical details
- Used automatically when token limits approached
- Read-only access

---

## Built-in Pipelines

### `speckit-flow` — Specification-Driven Development

Full feature development workflow with 5 steps:

```
navigate → specify → plan → implement → review
```

| Step | Persona | Description |
|------|---------|-------------|
| `navigate` | navigator | Analyze codebase for relevant files and patterns |
| `specify` | philosopher | Create feature specification with user stories |
| `plan` | philosopher | Generate ordered implementation plan |
| `implement` | craftsman | Write code and tests following the plan |
| `review` | auditor | Security and quality review of implementation |

**Examples:**
```bash
# Feature development
wave run --pipeline speckit-flow --input "add user authentication with JWT"
wave run --pipeline speckit-flow --input "implement pagination for the /users endpoint"
wave run --pipeline speckit-flow --input "add dark mode support to the dashboard"
wave run --pipeline speckit-flow --input "refactor database layer to use connection pooling"

# Preview execution plan
wave run --pipeline speckit-flow --input "add webhook notifications" --dry-run

# Resume from implementation after fixing spec issues
wave run --pipeline speckit-flow --input "add rate limiting" --from-step implement

# With longer timeout for complex features
wave run --pipeline speckit-flow --input "migrate to microservices" --timeout 120
```

**Artifacts produced:**
- `output/analysis.json` — Navigation findings
- `output/spec.md` — Feature specification
- `output/plan.md` — Implementation plan
- `output/review.md` — Security/quality review

---

### `hotfix` — Production Issue Resolution

Fast-track pipeline for urgent fixes with 3 steps:

```
investigate → fix → verify
```

| Step | Persona | Description |
|------|---------|-------------|
| `investigate` | navigator | Root cause analysis with blast radius assessment |
| `fix` | craftsman | Minimal fix with regression test |
| `verify` | auditor | Go/no-go recommendation for deployment |

**Examples:**
```bash
# Production bugs
wave run --pipeline hotfix --input "fix memory leak in cache.go"
wave run --pipeline hotfix --input "users getting 500 errors on login since last deploy"
wave run --pipeline hotfix --input "race condition in order processing causing duplicate charges"
wave run --pipeline hotfix --input "API returning stale data after cache invalidation"

# Security incidents
wave run --pipeline hotfix --input "SQL injection vulnerability in search endpoint"
wave run --pipeline hotfix --input "authentication bypass when session token is malformed"

# Performance issues
wave run --pipeline hotfix --input "database queries timing out under load"
wave run --pipeline hotfix --input "memory usage growing unbounded in worker process"

# Skip investigation if root cause is known
wave run --pipeline hotfix --input "null pointer in user.GetProfile()" --from-step fix
```

**Artifacts produced:**
- `output/findings.json` — Root cause analysis
- `output/verdict.md` — Deployment recommendation

---

### `code-review` — Pull Request Review

Comprehensive code review with parallel security and quality checks:

```
diff-analysis → security-review ─┬→ summary
                quality-review ──┘
```

| Step | Persona | Description |
|------|---------|-------------|
| `diff-analysis` | navigator | Map changed files and breaking changes |
| `security-review` | auditor | Check for vulnerabilities and secrets |
| `quality-review` | auditor | Check for errors, duplication, coverage |
| `summary` | summarizer | Synthesize into PR review comment |

**Examples:**
```bash
wave run --pipeline code-review --input "review changes in auth module"
wave run --pipeline code-review --input "review PR #123"
wave run --pipeline code-review --input "review last 3 commits"
```

---

### `refactor` — Safe Refactoring

Refactoring with test baseline and verification:

```
analyze → test-baseline → refactor → verify
```

| Step | Persona | Description |
|------|---------|-------------|
| `analyze` | navigator | Map refactoring scope and affected callers |
| `test-baseline` | craftsman | Ensure test coverage before changes |
| `refactor` | craftsman | Perform atomic refactoring changes |
| `verify` | auditor | Verify behavior preserved |

**Examples:**
```bash
wave run --pipeline refactor --input "extract UserService from monolith"
wave run --pipeline refactor --input "rename package auth to authentication"
wave run --pipeline refactor --input "convert callbacks to async/await"
```

---

### `debug` — Systematic Debugging

Hypothesis-driven debugging with root cause analysis:

```
reproduce → hypothesize → investigate → fix
```

| Step | Persona | Description |
|------|---------|-------------|
| `reproduce` | debugger | Create minimal reproduction case |
| `hypothesize` | debugger | Form ranked hypotheses about cause |
| `investigate` | debugger | Test hypotheses systematically |
| `fix` | craftsman | Implement fix with regression test |

**Examples:**
```bash
wave run --pipeline debug --input "intermittent 500 errors on /api/users"
wave run --pipeline debug --input "memory grows unbounded after 1000 requests"
wave run --pipeline debug --input "race condition in order processing"
```

---

### `test-gen` — Test Generation

Generate comprehensive test coverage:

```
analyze-coverage → generate-tests → verify-coverage
```

| Step | Persona | Description |
|------|---------|-------------|
| `analyze-coverage` | navigator | Find uncovered code paths |
| `generate-tests` | craftsman | Write tests for gaps |
| `verify-coverage` | auditor | Verify meaningful coverage increase |

**Examples:**
```bash
wave run --pipeline test-gen --input "improve coverage for internal/auth"
wave run --pipeline test-gen --input "add edge case tests for payment module"
wave run --pipeline test-gen --input "generate integration tests for API"
```

---

### `docs` — Documentation Generation

Generate or update documentation:

```
discover → generate → review
```

| Step | Persona | Description |
|------|---------|-------------|
| `discover` | navigator | Find public APIs and existing docs |
| `generate` | philosopher | Write documentation with examples |
| `review` | auditor | Check accuracy and completeness |

**Examples:**
```bash
wave run --pipeline docs --input "document the REST API"
wave run --pipeline docs --input "write README for internal/cache package"
wave run --pipeline docs --input "update API reference after v2 changes"
```

---

### `plan` — Task Planning

Break down a feature into actionable tasks:

```
explore → breakdown → review
```

| Step | Persona | Description |
|------|---------|-------------|
| `explore` | navigator | Understand codebase context |
| `breakdown` | planner | Create ordered task list |
| `review` | philosopher | Review for completeness |

**Examples:**
```bash
wave run --pipeline plan --input "implement user authentication with OAuth"
wave run --pipeline plan --input "add real-time notifications"
wave run --pipeline plan --input "migrate from REST to GraphQL"
```

---

### `migrate` — Database/API Migration

Safe migrations with rollback plans:

```
impact-analysis → migration-plan → implement → review
```

| Step | Persona | Description |
|------|---------|-------------|
| `impact-analysis` | navigator | Map affected code and breaking changes |
| `migration-plan` | philosopher | Design zero-downtime migration |
| `implement` | craftsman | Write migration scripts |
| `review` | auditor | Verify rollback works |

**Examples:**
```bash
wave run --pipeline migrate --input "add email_verified column to users table"
wave run --pipeline migrate --input "split users table into users and profiles"
wave run --pipeline migrate --input "migrate from v1 to v2 API"
```

---

## Configuration

### Manifest (wave.yaml)

```yaml
apiVersion: v1
kind: WaveManifest
metadata:
  name: my-project
  description: "Example project using Wave"

adapters:
  claude:
    binary: claude
    mode: headless
    output_format: json

personas:
  navigator:
    adapter: claude
    system_prompt_file: .wave/personas/navigator.md
    temperature: 0.1
    permissions:
      allowed_tools: ["Read", "Bash(git *)"]
      deny: ["Write(*)"]
  
  craftsman:
    adapter: claude
    system_prompt_file: .wave/personas/craftsman.md
    temperature: 0.7
    permissions:
      allowed: ["*"]

runtime:
  workspace_root: ./workspace
  max_concurrent_workers: 3
  default_timeout_minutes: 30
```

### Pipeline Example (.wave/pipelines/speckit-flow.yaml)

```yaml
kind: WavePipeline
metadata:
  name: speckit-flow
  description: "Specification-driven development workflow"

steps:
  - id: navigate
    persona: navigator
    memory:
      strategy: fresh
    exec:
      type: prompt
      source: "Analyze the codebase for: {{ input }}"
  
  - id: specify
    persona: philosopher
    dependencies: [navigate]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: navigate
          artifact: analysis
          as: context
    exec:
      type: prompt
      source: "Create detailed specifications for: {{ input }} using context from navigation analysis"
  
  - id: implement
    persona: craftsman
    dependencies: [specify]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: specify
          artifact: specification
          as: spec
    exec:
      type: prompt
      source: "Implement the feature according to the specification"
```

## Output and Events

Wave emits structured NDJSON events during pipeline execution:

```json
{"timestamp":"2026-02-01T10:00:00Z","pipeline_id":"123","step_id":"navigate","state":"running","duration_ms":0}
{"timestamp":"2026-02-01T10:01:30Z","pipeline_id":"123","step_id":"navigate","state":"completed","duration_ms":90000}
{"timestamp":"2026-02-01T10:01:31Z","pipeline_id":"123","step_id":"specify","state":"running","duration_ms":0}
```

Pipe events to `jq` for filtering:
```bash
wave run --pipeline flow.yaml --input "task" | jq 'select(.state == "completed")'
```

## Workspace Structure

Each pipeline step gets an isolated workspace:

```
./workspace/<pipeline-id>/<step-id>/
├── artifacts/           # Step outputs
│   ├── analysis.json   # Navigation results
│   ├── spec.md         # Generated specifications
│   └── implementation/ # Code changes
├── workspace/          # Working directory for the step
└── state.db           # SQLite state for resumption
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General Error |
| 2 | Usage Error |
| 3 | Pipeline Failed |
| 4 | Validation Error |
| 5 | Timeout |
| 130 | Interrupted (Ctrl+C) - state persisted for resumption |

## Advanced Features

### Shell Completion

```bash
# Bash
wave completion bash > /etc/bash_completion.d/wave

# Zsh  
wave completion zsh > "${fpath[1]}/_wave"

# Fish
wave completion fish > ~/.config/fish/completions/wave.fish
```

### Pipeline Features

- **DAG-based execution** with dependency resolution
- **Workspace isolation** for each step
- **State persistence** for resumption
- **Memory strategies** (fresh, cumulative, checkpoint)
- **Artifact injection** between steps
- **Permission enforcement** per persona
- **Contract validation** for structured outputs

### Development Commands

```bash
# Build locally
go build -o wave ./cmd/wave

# Run tests
go test ./...

# Lint (if configured)
golangci-lint run
```

## Examples

### Quick Bug Fix
```bash
wave do "fix the memory leak in user_service.go"
```

### Feature Development
```bash
wave run --pipeline speckit-flow --input "add OAuth2 authentication with Google provider"
```

### Code Review
```bash
wave do "review the recent authentication changes for security issues" --persona auditor
```

### Cleanup
```bash
# Clean old workspaces
wave clean --older-than 7d

# Clean specific failed pipeline
wave clean --pipeline-id a1b2c3d4
```

## Architecture

Wave implements a **pipeline orchestrator** pattern:

1. **Manifests** define available adapters and personas
2. **Pipelines** define multi-step workflows as DAGs  
3. **Steps** execute single persona tasks with isolated workspaces
4. **Events** provide real-time progress tracking
5. **State** enables resumption and debugging

The system enforces **least privilege** through persona-specific permissions and **auditability** through structured logging and artifact preservation.

For detailed documentation, see the `/docs` directory or run `wave <command> --help`.