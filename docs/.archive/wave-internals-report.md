# Wave Internals Report

This document provides a comprehensive technical analysis of Wave's internal architecture based on codebase exploration.

---

## 1. Pipeline Execution System

### Pipeline Definition & Loading

**Pipeline Structure (YAML)**
Located in: `internal/pipeline/types.go`

A Wave pipeline is defined in YAML with these key components:

```yaml
kind: WavePipeline
metadata:
  name: pipeline-id
  description: "description"

input:
  source: cli | github | ...
  label_filter: optional
  batch_size: optional

steps:
  - id: step-id
    persona: persona-name
    dependencies: [other-step-ids]
    memory:
      strategy: fresh | inherit
      inject_artifacts: [...]
    workspace:
      root: path
      mount: [source, target, mode]
    exec:
      type: prompt
      source: task-description
    output_artifacts: [name, path, type]
    handover:
      contract: {...}
      compaction: {...}
    strategy: matrix {...}
    validation: [...]
```

**Loading Process**
- File: `internal/pipeline/dag.go`
- `YAMLPipelineLoader.Load(path)` → reads YAML → unmarshals into `Pipeline` struct
- `DAGValidator.ValidateDAG(p)` → checks dependencies exist and detects cycles
- Defaults: `kind` defaults to "WavePipeline"

**Manifest Definition (wave.yaml)**
Located in: `internal/manifest/types.go`

```yaml
apiVersion: v1
kind: WaveManifest
metadata:
  name: project-name
adapters:
  claude:
    binary: claude
    mode: headless
    output_format: json
personas:
  craftsman:
    adapter: claude
    system_prompt_file: .wave/personas/craftsman.md
    permissions: {allowed_tools, deny}
runtime:
  workspace_root: .wave/workspaces
  max_concurrent_workers: 5
  relay: {token_threshold_percent, strategy}
  audit: {log_dir, log_all_tool_calls}
  routing: {default, rules}
```

### Dependency Resolution & Execution Order

**Graph Structure**
- File: `internal/pipeline/dag.go`
- Uses Directed Acyclic Graph (DAG) to model step dependencies
- Validates:
  - All referenced dependencies exist
  - No circular dependencies (cycle detection via DFS)

**Topological Sort Algorithm**
```go
TopologicalSort(p *Pipeline) → []*Step
```
- Performs depth-first traversal
- Returns steps in execution order
- Dependencies always execute before dependent steps
- Steps without mutual dependencies can run in parallel

**Execution Flow**
File: `internal/pipeline/executor.go`

```
DefaultPipelineExecutor.Execute()
├─ Validate DAG
├─ Topologically sort steps
├─ Create pipeline execution tracking
├─ For each step in order:
│  ├─ executeStep()
│  ├─ Handle retries (max_retries)
│  ├─ Track deliverables
│  └─ Check relay/compaction threshold
└─ Emit completion event
```

**State Tracking**
States: `pending` → `running` → `completed` | `failed` | `retrying`

Stored in:
- In-memory: `DefaultPipelineExecutor.pipelines[pipelineID]`
- Persistent: `state.StateStore` (SQLite)

### Artifact Flow Between Steps

**Output Artifacts**
Each step can declare output artifacts:

```yaml
output_artifacts:
  - name: result
    path: artifact.json
    type: json
    required: true
```

**Artifact Injection**
File: `internal/pipeline/executor.go` line 763

```go
injectArtifacts(execution, step, workspacePath)
```

Memory config specifies what artifacts to inject:
```yaml
memory:
  inject_artifacts:
    - step: previous-step
      artifact: result
      as: injected-name
```

Flow:
1. Step writes output artifacts to its workspace (e.g., `artifact.json`)
2. Artifact path registered: `execution.ArtifactPaths["step-id:artifact-name"]` → filesystem path
3. Dependent step's memory config injects artifacts to `artifacts/` subdirectory in its workspace
4. Step reads from `artifacts/injected-name`

### Step Execution Mechanics

**Workspace Creation**
- Location: `.wave/workspaces/<pipeline>/<step>/`
- Isolation: Each step gets isolated filesystem
- Mounts: Can mount source directories (readonly/readwrite)

**Persona Assignment**
- Each step specifies `persona: craftsman|navigator|auditor|...`
- Persona defines:
  - Adapter (e.g., Claude CLI)
  - System prompt file
  - Allowed/denied tools (permissions)
  - Model & temperature settings

**Prompt Building**
File: `internal/pipeline/executor.go` line 608

```go
buildStepPrompt(execution, step)
```

Process:
1. Get step's exec.source template
2. Inject `{{ input }}` with sanitized user input
3. Inject `{{ task }}` for matrix workers
4. Inject json_schema contract requirements
5. Resolve template variables ({{pipeline_context.branch_name}}, etc.)
6. Return final prompt for adapter

**Execution via Adapter**
```go
adapter.AdapterRunConfig{
  Adapter: binary-name
  Persona: persona-name
  WorkspacePath: isolated-dir
  Prompt: full-prompt
  SystemPrompt: persona-system-prompt
  Timeout: default-timeout
  AllowedTools: [Read, Write, Bash]
  DenyTools: [dangerous-operations]
}

result := runner.Run(ctx, cfg)
```

### Key Data Structures

**PipelineExecution** (tracks running pipeline state)
```go
type PipelineExecution struct {
  Pipeline       *Pipeline
  Manifest       *manifest.Manifest
  States         map[string]string          // step-id → state
  Results        map[string]map[string]interface{} // step outputs
  ArtifactPaths  map[string]string          // "step:artifact" → filepath
  WorkspacePaths map[string]string          // step-id → workspace-path
  Input          string                     // original user input
  Status         *PipelineStatus            // aggregated status
  Context        *PipelineContext           // template variables
}
```

**PipelineContext** (template variable resolution)
```go
type PipelineContext struct {
  BranchName      string            // git branch (auto-detected)
  FeatureNum      string            // extracted from branch (### format)
  SpeckitMode     bool              // Speckit protocol detection
  PipelineID      string
  StepID          string
  CustomVariables map[string]string
}
```

### Key Files Reference

| File | Purpose |
|------|---------|
| `internal/pipeline/types.go` | Pipeline, Step, artifact type definitions |
| `internal/pipeline/executor.go` | Main execution engine (1068 lines) |
| `internal/pipeline/dag.go` | DAG validation & topological sort |
| `internal/pipeline/matrix.go` | Parallel matrix strategy execution |
| `internal/pipeline/router.go` | Work item routing to pipelines |
| `internal/pipeline/context.go` | Template variable resolution |
| `internal/manifest/types.go` | Manifest, Adapter, Persona definitions |
| `internal/manifest/parser.go` | YAML parsing & validation |

---

## 2. Contract Validation System

### Types of Contracts Supported

Wave supports multiple contract types defined in `internal/contract/contract.go`:

| Contract Type | Purpose | Implementation |
|---|---|---|
| **json_schema** | JSON Schema (RFC 7159) validation | `jsonSchemaValidator` - validates artifact.json against schema |
| **typescript_interface** | TypeScript interface validation | `typeScriptValidator` - runs `tsc --noEmit --strict` compiler checks |
| **test_suite** | Test execution validation | `testSuiteValidator` - executes arbitrary commands (e.g., `go test ./...`) |
| **markdown_spec** | Markdown document structure | `markdownSpecValidator` - validates heading hierarchy, links, sections |
| **template** | Structured template matching | `TemplateValidator` - validates template compliance |
| **format** | Production-ready format validation | `FormatValidator` - checks for GitHub issues, PRs, code completeness |

**Key Features:**
- **Progressive Validation** - Can warn instead of fail using `WarnOnRecovery`
- **Recovery Levels** - Conservative (strict), Progressive, Aggressive JSON parsing
- **Error Wrapper Detection** - Automatically extracts valid JSON from error messages
- **Quality Gates** - Additional validation checks beyond schema

### How Contracts Are Defined in Pipelines

Contracts are configured in the pipeline YAML in the `handover` section of each step:

```yaml
handover:
  contract:
    type: "json_schema"                    # Contract type
    schema_path: ".wave/contracts/spec-phase.schema.json"  # Schema file
    # OR inline schema:
    schema: '{"type": "object", ...}'

    source: "artifact.json"                # Default: artifact.json
    must_pass: true                        # Block pipeline on failure (default: false)
    max_retries: 2                         # Retry attempts on failure

    # Advanced options:
    allow_recovery: true                   # Enable JSON recovery
    recovery_level: "progressive"          # "conservative", "progressive", "aggressive"
    progressive_validation: true           # Warnings instead of errors
    disable_wrapper_detection: false       # Auto-detect JSON in error messages

  # Optional quality gates:
  quality_gates:
    - type: "required_fields"
      required: true
      parameters:
        fields: ["title", "body", "labels"]
    - type: "content_completeness"
      threshold: 80
      parameters:
        min_words: 50
```

### Validation Flow

Validation happens **after step execution, before returning to the next step**:

```
Step Executes → Adapter Generates Output → Write Artifacts → Relay Compaction Check
→ CONTRACT VALIDATION ← Artifact is validated here
  → Quality Gates Check
  → On Success: Step marked completed
  → On Failure:
    - If must_pass=true: Block pipeline (error)
    - If must_pass=false: Soft failure (warning, continue)
    - If max_retries > 0: Re-execute step with repair guidance
```

**Validation Code** (`internal/pipeline/executor.go` lines 495-549):

```go
// After adapter execution
if step.Handover.Contract.Type != "" {
    contractCfg := contract.ContractConfig{
        Type:       step.Handover.Contract.Type,
        Source:     resolvedSource,
        SchemaPath: step.Handover.Contract.SchemaPath,
        MustPass:   step.Handover.Contract.MustPass,
        MaxRetries: step.Handover.Contract.MaxRetries,
    }

    if err := contract.Validate(contractCfg, workspacePath); err != nil {
        // Emit contract_failed event
        if contractCfg.StrictMode {
            return fmt.Errorf("contract validation failed: %w", err)  // Hard fail
        } else {
            // Soft failure - emit contract_soft_failure event
        }
    }
    // Emit contract_passed event on success
}
```

### What Happens When Validation Fails

**A. Hard Failure (`must_pass: true`)**
```
Validation Failure → Emit "contract_failed" event → Block Pipeline
  → Return error to executor
  → Pipeline terminates with error
```

**B. Soft Failure (`must_pass: false`)**
```
Validation Failure → Emit "contract_soft_failure" event → Continue Pipeline
  → Log error for audit trail
  → Next step proceeds with previous output
```

**C. Retry on Failure (`max_retries > 0`)**
```
Validation Failure (Attempt 1)
  ↓
Classify Failure Type:
  - schema_mismatch: Missing fields, wrong types
  - format_error: Invalid syntax
  - missing_content: Incomplete content
  - quality_gate: Failed quality checks
  - structure: Document structure issues
  ↓
Generate Repair Prompt with targeted guidance
  ↓
Re-execute Step with Repair Context (Attempt 2-N)
  ↓
Success or All Retries Exhausted
```

**Retry Strategy** (`internal/contract/retry_strategy.go`):
- **Adaptive Retry** - Analyzes failure type and suggests fixes
- **Exponential Backoff** - Delays between retries (1s, 2s, 4s...)
- **Failure Classification** - Maps errors to fix strategies
- **Repair Prompts** - Targeted guidance injected into prompt for retries

---

## 3. Workspace Management System

### Ephemeral Workspace Creation

**Location**: `internal/workspace/workspace.go` (lines 61-110)

**Workspace Directory Structure**:
```
.wave/workspaces/
├── <pipeline_id>/
│   ├── <step_id>/
│   │   ├── artifacts/          (injected artifacts)
│   │   ├── <mount_target>/     (mounted source directories)
│   │   └── (other step outputs)
```

**Creation Process** (`Create()` method):
1. **Initialization**: Takes a `WorkspaceConfig` with mounts and template variables
2. **Path Construction**: Builds workspace path as `.wave/workspaces/<pipeline_id>/<step_id>`
3. **Mount Processing**: For each mount configuration:
   - Validates source exists (prevents non-existent source errors)
   - Performs variable substitution (e.g., `{{ pipeline_id }}`)
   - Copies source to target recursively via `copyRecursive()`
   - Sets permissions based on mount mode (readonly=0555, readwrite=0755)
4. **Fresh State**: Pipeline runs clean workspace each time (previous state removed)

### Workspace Isolation Mechanisms

**Isolation Strategy**:
1. **Path-Based Isolation**: Each step gets dedicated isolated directory
2. **Copy-on-Mount**: Sources are **copied, not linked**
3. **Filesystem Permissions**: Readonly mounts get 0555, readwrite get 0755
4. **Concurrent Safety**: Unique paths per concurrent execution prevent collisions

### Artifact Injection System

**Injection Process** (`InjectArtifacts()` method - lines 112-154):

1. **Resolution**: Maps artifact references to actual filesystem paths
2. **Naming Convention**: Format `<step_id>_<artifact_name_or_as>`
3. **Artifact Injection Flow**:
   ```
   Pipeline Execution
   ├── Step A (spec phase)
   │   └── Creates: spec.md (output_artifacts)
   │       Registered in: ArtifactPaths["spec:spec"] = <path>
   │
   ├── Step B (docs phase, depends on A)
   │   └── Injects artifacts from step A
   │       Result: artifacts/spec_input-spec.md in workspace B
   ```

### File Copying and Optimization

**Smart Copy Mechanism** (`copyRecursive()` - lines 171-206):

**Directory Skipping** (lines 157-169):
```go
var skipDirs = map[string]bool{
    "node_modules": true,  // Large dependency folders
    ".git":         true,  // Version control
    ".wave":        true,  // Wave internal state
    ".claude":      true,  // Claude internal state
    "vendor":       true,  // Go vendor directory
    "__pycache__":  true,  // Python cache
    ".venv":        true,  // Python virtual env
    "dist":         true,  // Build output
    "build":        true,  // Build artifacts
    ".next":        true,  // Next.js build
    ".cache":       true,  // Cache directories
}
```

**File Size Limit**: Skips files > 10MB (line 197)

### Key Design Properties

| Property | Implementation |
|----------|-----------------|
| **Ephemeral** | Fresh workspace per pipeline run, cleaned before start |
| **Isolated** | Independent copies per step, no cross-pollution |
| **Efficient** | Smart copying with skipDirs, file size limits |
| **Observable** | Workspace paths tracked in execution, available for inspection |
| **Persistent** | Workspaces retained in `.wave/workspaces/` for post-mortem analysis |
| **Async-Safe** | Unique paths per concurrent execution prevent collisions |
| **Artifact-Driven** | Explicit injection mechanism for inter-step communication |
| **Resumable** | Workspace paths stored in persistent state for pipeline resumption |

---

## 4. State Management System

### Database Schema & Tables

#### Core Infrastructure
- **Database Engine**: SQLite (modernc.org/sqlite) with WAL mode enabled for concurrent access
- **Location**: `.wave/state.db`
- **Concurrency**: Single connection pool (Max 1 open, 1 idle) due to SQLite's locking model
- **Configuration**: 5-second busy timeout, foreign keys enabled

#### Base State Tables (Migration 1)
- **pipeline_state**: Pipeline-level execution state
  - `pipeline_id` (TEXT, PK), `pipeline_name`, `status`, `input`, `created_at`, `updated_at`

- **step_state**: Step-level execution state for resumption
  - `step_id` (TEXT, PK), `pipeline_id` (FK), `state` (pending/running/completed/failed/retrying)
  - `retry_count`, `started_at`, `completed_at`, `workspace_path`, `error_message`

#### Run Tracking Tables (Migration 2)
- **pipeline_run**: Individual pipeline execution runs
  - `run_id` (TEXT, PK): Format `{pipeline_name}-{timestamp}-{random}`
  - `pipeline_name`, `status`, `input`, `current_step`, `total_tokens`
  - `tags_json`: JSON array for run categorization/filtering

- **event_log**: Step-level event logging
  - `id` (AUTOINCREMENT), `run_id` (FK), `timestamp`, `step_id`, `state`
  - `persona`, `message`, `tokens_used`, `duration_ms`

- **artifact**: Artifact metadata and tracking
  - `id` (AUTOINCREMENT), `run_id` (FK), `step_id`, `name`, `path`, `type`, `size_bytes`

- **cancellation**: Cancellation request flags
  - `run_id` (TEXT, PK), `requested_at`, `force` (boolean)

#### Performance Metrics (Migration 3)
- **performance_metric**: Step execution metrics
  - `run_id`, `step_id`, `pipeline_name`, `persona`
  - `duration_ms`, `tokens_used`, `files_modified`, `artifacts_generated`

#### Progress Tracking (Migration 4)
- **progress_snapshot**: Point-in-time progress records
- **step_progress**: Real-time step progress (UPSERT on step_id)
- **pipeline_progress**: Pipeline-level aggregation (UPSERT on run_id)

### Resumption Mechanism

The **ResumeManager** enables pause/resume at step granularity:

1. **ValidateResumePoint(pipeline, fromStep)**:
   - Confirms step exists in pipeline
   - Validates phase sequence
   - Checks workspace not in use

2. **LoadResumeState(pipeline, fromStep)**:
   - Scans workspace filesystem
   - Marks completed predecessors
   - Indexes artifact paths for injection

3. **CreateResumeSubpipeline(pipeline, fromStep)**:
   - Creates new pipeline with steps[fromStep:]
   - Maintains original dependencies

4. **ExecuteResumedPipeline(execution, fromStep)**:
   - Validates DAG of subpipeline
   - Injects preserved artifacts
   - Records new execution history

### Summary Diagram

```
PERSISTENCE LAYER
├─ SQLite Database (.wave/state.db)
│  ├─ Pipeline Execution (pipeline_state, pipeline_run, event_log)
│  ├─ Step Execution (step_state, step_progress, progress_snapshot)
│  ├─ Artifact Tracking (artifact, artifact_metadata)
│  ├─ Performance Metrics (performance_metric)
│  ├─ Run Management (cancellation, tags_json)
│  └─ Schema Tracking (schema_migrations)
│
├─ Filesystem State
│  └─ .wave/workspaces/{pipeline}/{step}/
│     ├─ src/ (injected sources)
│     ├─ artifacts/ (outputs)
│     └─ workspace files
│
└─ In-Memory State (DefaultPipelineExecutor)
   └─ pipelines map[string]*PipelineExecution
```

---

## 5. Security and Audit System

### Security Validation Architecture

#### Path Sanitization (`internal/security/path.go`)
- **Path Traversal Prevention**: Comprehensive detection of path traversal attempts
  - Blocks common patterns: `..`, `./`, `../`, `..\\`, encoded variants like `%2e%2e`
  - Validates against approved directory whitelist
  - Maximum path length enforcement (default: 255 chars)
  - Symbolic link detection (disabled by default)

- **Approved Directories Model**: Allowlist-based path validation
  - Default approved paths: `.wave/contracts/`, `.wave/schemas/`, `contracts/`, `schemas/`
  - Absolute path resolution for safety checks

#### Input Sanitization (`internal/security/sanitize.go`)
- **Prompt Injection Detection**: Regex-based pattern matching
  - Default patterns detect: ignore instructions, system prompts, you are now, disregard above
  - Two modes: strict (reject) vs. sanitize (remove/neutralize patterns)
  - Input hash tracking (SHA-256) for audit purposes

- **Content Sanitization**: Schema and payload cleaning
  - Script tag removal: `<script>...</script>`
  - Event handler removal: `on* = '...'`
  - JavaScript URL removal: `javascript:...`
  - Size limits enforcement (default: 1MB content limit)

- **Risk Scoring**: Quantitative assessment of input danger
  - Base score: 20 if any sanitization applied
  - Prompt injection: +50
  - Suspicious content removal: +30
  - Length truncation: +10
  - Credential-like keywords: +5 per keyword found
  - Capped at 100

### Credential Scrubbing in Logs (`internal/audit/logger.go`)

#### Pattern-Based Redaction
- **Credential Keywords** detected (case-insensitive):
  - `API_KEY`, `API-KEY`, `APIKEY`
  - `TOKEN`
  - `SECRET`, including prefixed: `client_secret`, `signing_secret`
  - `PASSWORD`, including prefixed: `db_password`, `MYSQL_PASSWORD`
  - `CREDENTIAL`, `AUTH`
  - `PRIVATE_KEY`, `PRIVATEKEY`
  - `ACCESS_KEY`, `ACCESSKEY`

- **Pattern Format**: `(KEYWORD)[=:]?\s*[\w\-]+`
  - Matches: `API_KEY=value`, `token:value`, `PASSWORD=pass123`
  - All matches replaced with `[REDACTED]` marker

### Permission Enforcement Model (`internal/adapter/permissions.go`)

#### Persona Permission Structure
```yaml
personas:
  implementer:
    adapter: claude
    permissions:
      allowed_tools:
        - Read
        - Write
        - Edit
        - Bash
      deny:
        - Bash(rm -rf /*)
        - Bash(sudo *)
```

#### Permission Pattern Format
- **Simple**: `Read`, `Write`, `Edit`, `Bash` (no argument constraint)
- **With argument patterns**: `Write(artifact.json)`, `Bash(git log*)`
- **Glob patterns**: `Write(*.go)`, `Write(.wave/specs/*)`, `Bash(go test*)`
- **Wildcards**: `Write(*)` = all write operations

#### Deny-First Precedence (Fail-Safe)
```go
// CheckPermission algorithm:
// 1. Check ALL deny patterns first - if ANY match, DENY (even if allowed)
// 2. If no allow patterns defined, ALLOW by default
// 3. Check allow patterns - if ANY match, ALLOW
// 4. No allow pattern matched -> DENY
```

#### Built-in Personas

| Persona | Can Write | Can Edit | Can Read | Can Bash | Special Restrictions |
|---------|-----------|----------|----------|----------|----------------------|
| **implementer** | Yes (all) | Yes | Yes | Yes | Deny: `rm -rf /*`, `sudo *` |
| **reviewer** | Limited | No | Yes | Test only | Deny: `*.go`, `*.ts`, `Edit(*)` |
| **navigator** | No | No | Yes | Git only | Deny: `Write(*)`, `Edit(*)` |
| **auditor** | No | No | Yes | Audit only | Deny: `Write(*)`, `Edit(*)` |
| **craftsman** | Yes (all) | Yes | Yes | Yes | Deny: `rm -rf /*` |
| **philosopher** | `.wave/specs/` only | No | Yes | No | Deny: `Bash(*)` |
| **planner** | No | No | Yes | No | Deny: `Write(*)`, `Edit(*)`, `Bash(*)` |

### Constitutional Compliance

The security model enforces:
- ✓ **Fresh memory at boundaries**: Each step starts with clean context
- ✓ **Permission enforcement**: Deny/allow patterns strictly enforced
- ✓ **Contract validation**: Output validated before step completion
- ✓ **Ephemeral workspaces**: Isolated filesystem per execution
- ✓ **Observable execution**: Structured events for monitoring
- ✓ **Credential scrubbing**: Logs safe for storage/transmission
- ✓ **Security-first defaults**: Strict mode enabled, symlinks disabled

---

## Key File Locations

| Component | File | Purpose |
|-----------|------|---------|
| Path Validation | `/internal/security/path.go` | Traversal prevention, approved directories |
| Input Sanitization | `/internal/security/sanitize.go` | Prompt injection, content cleaning |
| Configuration | `/internal/security/config.go` | Security defaults and settings |
| Credential Scrubbing | `/internal/audit/logger.go` | Redaction patterns and trace logging |
| Event Emission | `/internal/event/emitter.go` | Progress tracking, NDJSON output |
| Permissions | `/internal/adapter/permissions.go` | Deny/allow enforcement engine |
| Manifest Types | `/internal/manifest/types.go` | Persona permission definitions |
| Error Types | `/internal/security/errors.go` | Structured security errors |
| Workspace | `/internal/workspace/workspace.go` | Ephemeral workspace isolation |
| Pipeline Executor | `/internal/pipeline/executor.go` | Main execution engine |
| DAG Validation | `/internal/pipeline/dag.go` | Dependency resolution, cycle detection |
| Matrix Execution | `/internal/pipeline/matrix.go` | Parallel fan-out execution |
| Contract Validation | `/internal/contract/contract.go` | Schema validation, quality gates |
| State Store | `/internal/state/store.go` | SQLite persistence layer |
| Resume Manager | `/internal/pipeline/resume.go` | Pipeline resumption logic |

---

*Report generated from codebase analysis - February 2026*
