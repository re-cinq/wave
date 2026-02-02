# Implementation Plan: Prototype-Driven Development Pipelines

**Branch**: `017-prototype-driven-development` | **Date**: 2026-02-02 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/017-prototype-driven-development/spec.md`

## Summary

This feature adds a new `prototype` pipeline type to Wave that orchestrates greenfield development through four sequential phases: **spec** (requirements capture with speckit integration), **docs** (stakeholder documentation), **dummy** (working prototype with stub implementations), and **implement** (full production code). The pipeline enforces artifact contracts at each phase boundary and supports re-running individual phases for iterative refinement.

## Technical Context

**Language/Version**: Go 1.22+
**Primary Dependencies**: gopkg.in/yaml.v3, github.com/spf13/cobra (existing Wave dependencies)
**Storage**: SQLite for pipeline state, filesystem for workspaces and artifacts
**Testing**: go test with race detector
**Target Platform**: Linux, macOS, Windows (single static binary)
**Project Type**: Single project (CLI extension)
**Performance Goals**: Pipeline initialization < 1s, phase transitions < 100ms
**Constraints**: No runtime dependencies, graceful degradation when speckit unavailable
**Scale/Scope**: Single pipeline definition, 4 personas, 4 contracts

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary, Zero Dependencies | PASS | No new runtime dependencies; speckit is external prerequisite like Claude |
| P2: Manifest as Single Source of Truth | PASS | Pipeline defined in `.wave/pipelines/prototype.yaml`, referenced from manifest |
| P3: Persona-Scoped Execution Boundaries | PASS | Each phase uses existing personas (navigator, philosopher, craftsman, auditor) |
| P4: Fresh Memory at Every Step Boundary | PASS | Each phase starts fresh, artifacts flow via `inject_artifacts` |
| P5: Navigator-First Architecture | PASS | Spec phase begins with navigator to analyze existing codebase context |
| P6: Contracts at Every Handover | PASS | JSON schema contracts defined for each phase transition |
| P7: Relay via Dedicated Summarizer | PASS | Uses existing summarizer persona for compaction |
| P8: Ephemeral Workspaces for Safety | PASS | Each phase runs in isolated workspace per existing infrastructure |
| P9: Credentials Never Touch Disk | PASS | No credential handling in pipeline definition |
| P10: Observable Progress, Auditable Operations | PASS | Uses existing event emission and audit infrastructure |
| P11: Bounded Recursion and Resource Limits | PASS | Uses manifest-level meta_pipeline limits |
| P12: Minimal Step State Machine | PASS | Steps use existing 5-state machine |

**All gates pass. No constitution violations.**

## Project Structure

### Documentation (this feature)

```
specs/017-prototype-driven-development/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```
.wave/
├── pipelines/
│   └── prototype.yaml           # NEW: Main prototype pipeline definition
├── contracts/
│   ├── spec-phase.schema.json   # NEW: Spec phase output contract
│   ├── docs-phase.schema.json   # NEW: Docs phase output contract
│   └── dummy-phase.schema.json  # NEW: Dummy phase output contract
└── personas/
    ├── navigator.md             # EXISTING: Used in spec phase
    ├── philosopher.md           # EXISTING: Used in spec and docs phases
    ├── craftsman.md             # EXISTING: Used in dummy and implement phases
    └── auditor.md               # EXISTING: Used in review sub-phase
```

**Structure Decision**: This feature extends the existing Wave configuration structure. All new artifacts are configuration files (YAML, JSON schemas) that integrate with existing infrastructure. No changes to Go source code are required for the MVP.

---

## Technical Approach

### 1. Pipeline Architecture

The prototype pipeline is a linear DAG with four main phases and optional validation sub-steps:

```
┌─────────┐    ┌──────────┐    ┌─────────┐    ┌─────────────┐
│  spec   │───▶│   docs   │───▶│  dummy  │───▶│  implement  │
└─────────┘    └──────────┘    └─────────┘    └─────────────┘
     │              │               │               │
     ▼              ▼               ▼               ▼
 [contract]    [contract]      [contract]      [test_suite]
```

Each phase:
1. Starts with fresh memory
2. Receives artifacts from previous phases via `inject_artifacts`
3. Produces typed artifacts with defined schemas
4. Validates output against handover contract before proceeding

### 2. Speckit Integration Strategy

The spec phase integrates with speckit as an external tool, similar to how Wave integrates with Claude:

1. **Detection**: Check if `.specify/` directory exists in workspace
2. **Execution**: Run speckit commands via craftsman persona with Bash permissions
3. **Fallback**: If speckit unavailable, use philosopher persona to generate spec manually
4. **Output**: Unified spec artifact regardless of generation method

### 3. Artifact Flow Design

```
Phase       Inputs                          Outputs
─────────   ──────────────────────────────  ────────────────────────────
spec        user_description, codebase      specification.json
docs        specification.json              documentation.md, api-docs.md
dummy       specification.json, docs/*.md   prototype/, interfaces.json
implement   all prior artifacts             source/, tests/
```

### 4. Stale Artifact Detection

Wave's existing state persistence tracks step completion timestamps. To detect stale artifacts:

1. Each artifact records the timestamp of the step that produced it
2. When a phase is re-run, compare upstream artifact timestamps
3. If any input artifact is newer than the step's last completion, mark downstream steps as stale
4. Prompt user to re-run downstream phases

---

## Phase Definitions

### Phase 1: Specification (spec)

**Purpose**: Capture requirements using speckit or manual specification writing.

**Persona**: `philosopher` (with navigator sub-step for codebase analysis)

**Steps**:
```yaml
- id: spec-navigate
  persona: navigator
  exec:
    type: prompt
    source: |
      Analyze the codebase context for the new feature: {{ input }}

      Identify:
      1. Existing patterns and conventions
      2. Related functionality to integrate with
      3. Technology stack and dependencies
      4. Testing infrastructure
  output_artifacts:
    - name: codebase_context
      path: output/codebase-context.json
      type: json

- id: spec-define
  persona: philosopher
  dependencies: [spec-navigate]
  memory:
    inject_artifacts:
      - step: spec-navigate
        artifact: codebase_context
        as: context
  exec:
    type: prompt
    source: |
      Create a feature specification for: {{ input }}

      Based on the codebase context, define:
      1. User stories with acceptance criteria
      2. Key entities and data model
      3. Interface contracts (API/CLI/UI)
      4. Edge cases and error handling
      5. Success metrics

      Output as JSON matching the spec-phase schema.
  output_artifacts:
    - name: specification
      path: output/specification.json
      type: json
  handover:
    contract:
      type: json_schema
      schema_path: .wave/contracts/spec-phase.schema.json
      source: output/specification.json
      on_failure: retry
      max_retries: 2
```

### Phase 2: Documentation (docs)

**Purpose**: Generate stakeholder-readable documentation from specification.

**Persona**: `philosopher`

**Steps**:
```yaml
- id: docs-generate
  persona: philosopher
  dependencies: [spec-define]
  memory:
    inject_artifacts:
      - step: spec-define
        artifact: specification
        as: spec
  exec:
    type: prompt
    source: |
      Generate documentation from the specification.

      Create:
      1. Feature overview (non-technical stakeholders)
      2. User guide with workflows
      3. Technical architecture summary
      4. API/CLI reference (if applicable)

      Output documentation that can be understood without code.
  output_artifacts:
    - name: feature_docs
      path: output/feature-docs.md
      type: markdown
    - name: api_docs
      path: output/api-docs.md
      type: markdown
  handover:
    contract:
      type: json_schema
      schema_path: .wave/contracts/docs-phase.schema.json
      source: output/docs-manifest.json
      on_failure: retry
      max_retries: 2
```

### Phase 3: Dummy Implementation (dummy)

**Purpose**: Create working prototype with stub implementations.

**Persona**: `craftsman`

**Steps**:
```yaml
- id: dummy-scaffold
  persona: craftsman
  dependencies: [docs-generate]
  memory:
    inject_artifacts:
      - step: spec-define
        artifact: specification
        as: spec
      - step: docs-generate
        artifact: feature_docs
        as: docs
  workspace:
    mount:
      - source: ./
        target: /src
        mode: readwrite
  exec:
    type: prompt
    source: |
      Create a dummy/prototype implementation:

      1. Scaffold file structure following codebase patterns
      2. Implement interfaces with stub responses
      3. Create runnable entry points
      4. Add TODO markers for real implementation

      The prototype must be runnable and demonstrate the user flow.
  output_artifacts:
    - name: prototype
      path: prototype/
      type: directory
    - name: interfaces
      path: output/interfaces.json
      type: json
  handover:
    contract:
      type: json_schema
      schema_path: .wave/contracts/dummy-phase.schema.json
      source: output/dummy-manifest.json
      on_failure: retry
      max_retries: 2

- id: dummy-verify
  persona: auditor
  dependencies: [dummy-scaffold]
  memory:
    inject_artifacts:
      - step: spec-define
        artifact: specification
        as: spec
      - step: dummy-scaffold
        artifact: interfaces
        as: interfaces
  exec:
    type: prompt
    source: |
      Verify the dummy implementation:

      1. All specified interfaces are present
      2. Prototype is runnable
      3. User flows match specification
      4. No real business logic (only stubs)

      Report any gaps between spec and prototype.
  output_artifacts:
    - name: verification
      path: output/dummy-verification.md
      type: markdown
```

### Phase 4: Implementation (implement)

**Purpose**: Full production implementation guided by all prior artifacts.

**Persona**: `craftsman`

**Steps**:
```yaml
- id: implement-plan
  persona: planner
  dependencies: [dummy-verify]
  memory:
    inject_artifacts:
      - step: spec-define
        artifact: specification
        as: spec
      - step: dummy-scaffold
        artifact: interfaces
        as: interfaces
      - step: dummy-verify
        artifact: verification
        as: verification
  exec:
    type: prompt
    source: |
      Create implementation task breakdown:

      1. Order tasks by dependency
      2. Identify stub replacements from dummy
      3. Define test coverage requirements
      4. Estimate complexity per task
  output_artifacts:
    - name: implementation_plan
      path: output/implementation-plan.md
      type: markdown

- id: implement-code
  persona: craftsman
  dependencies: [implement-plan]
  memory:
    inject_artifacts:
      - step: spec-define
        artifact: specification
        as: spec
      - step: dummy-scaffold
        artifact: prototype
        as: prototype
      - step: implement-plan
        artifact: implementation_plan
        as: plan
  workspace:
    mount:
      - source: ./
        target: /src
        mode: readwrite
  exec:
    type: prompt
    source: |
      Implement the feature following the plan:

      1. Replace stubs with real implementations
      2. Write unit tests for each component
      3. Write integration tests for user flows
      4. Document public APIs

      Commit atomic changes per task in the plan.
  handover:
    contract:
      type: test_suite
      command: "go test ./..."
      must_pass: true
      on_failure: retry
      max_retries: 3
    compaction:
      trigger: "token_limit_80%"
      persona: summarizer

- id: implement-review
  persona: auditor
  dependencies: [implement-code]
  memory:
    inject_artifacts:
      - step: spec-define
        artifact: specification
        as: spec
  exec:
    type: prompt
    source: |
      Final implementation review:

      1. Security audit
      2. Test coverage analysis
      3. Code style compliance
      4. Documentation completeness

      Generate final report with sign-off or blocking issues.
  output_artifacts:
    - name: final_review
      path: output/final-review.md
      type: markdown
```

---

## Persona Assignments

| Phase | Step | Persona | Rationale |
|-------|------|---------|-----------|
| spec | spec-navigate | navigator | Read-only codebase analysis |
| spec | spec-define | philosopher | Specification writing, no code execution |
| docs | docs-generate | philosopher | Documentation writing, no code execution |
| dummy | dummy-scaffold | craftsman | Code generation with filesystem access |
| dummy | dummy-verify | auditor | Read-only verification |
| implement | implement-plan | planner | Task breakdown, read-only analysis |
| implement | implement-code | craftsman | Full implementation with tests |
| implement | implement-review | auditor | Final security and quality review |

---

## Contract Definitions

### spec-phase.schema.json

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["title", "description", "user_stories", "entities"],
  "properties": {
    "title": { "type": "string", "minLength": 5 },
    "description": { "type": "string", "minLength": 50 },
    "user_stories": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["id", "as_a", "i_want", "so_that", "acceptance_criteria"],
        "properties": {
          "id": { "type": "string", "pattern": "^US-[0-9]+$" },
          "as_a": { "type": "string" },
          "i_want": { "type": "string" },
          "so_that": { "type": "string" },
          "acceptance_criteria": {
            "type": "array",
            "items": { "type": "string" },
            "minItems": 1
          },
          "priority": { "type": "string", "enum": ["P1", "P2", "P3", "P4"] }
        }
      },
      "minItems": 1
    },
    "entities": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["name", "fields"],
        "properties": {
          "name": { "type": "string" },
          "fields": { "type": "array", "items": { "type": "object" } },
          "relationships": { "type": "array", "items": { "type": "string" } }
        }
      }
    },
    "interfaces": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["type", "name"],
        "properties": {
          "type": { "type": "string", "enum": ["cli", "api", "ui", "library"] },
          "name": { "type": "string" },
          "operations": { "type": "array", "items": { "type": "object" } }
        }
      }
    },
    "edge_cases": { "type": "array", "items": { "type": "string" } },
    "success_metrics": { "type": "array", "items": { "type": "string" } }
  }
}
```

### docs-phase.schema.json

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["generated_files", "spec_coverage"],
  "properties": {
    "generated_files": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["path", "type", "description"],
        "properties": {
          "path": { "type": "string" },
          "type": { "type": "string", "enum": ["overview", "user_guide", "api_reference", "architecture"] },
          "description": { "type": "string" }
        }
      },
      "minItems": 1
    },
    "spec_coverage": {
      "type": "object",
      "required": ["user_stories_documented", "entities_documented"],
      "properties": {
        "user_stories_documented": { "type": "array", "items": { "type": "string" } },
        "entities_documented": { "type": "array", "items": { "type": "string" } }
      }
    }
  }
}
```

### dummy-phase.schema.json

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["prototype_path", "interfaces_implemented", "runnable"],
  "properties": {
    "prototype_path": { "type": "string" },
    "interfaces_implemented": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["name", "stub_type"],
        "properties": {
          "name": { "type": "string" },
          "stub_type": { "type": "string", "enum": ["hardcoded", "mock", "noop", "echo"] },
          "file_path": { "type": "string" }
        }
      }
    },
    "runnable": { "type": "boolean" },
    "entry_point": { "type": "string" },
    "todo_markers": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "file": { "type": "string" },
          "line": { "type": "integer" },
          "description": { "type": "string" }
        }
      }
    }
  }
}
```

---

## Integration Points

### 1. CLI Integration

Add `--prototype` flag to `wave do` or create dedicated `wave prototype` command:

```bash
# Option A: Flag on wave do
wave do "implement user authentication" --prototype

# Option B: Dedicated command
wave prototype init "implement user authentication"
wave prototype status
wave prototype resume --from-phase docs
```

**Recommendation**: Option A leverages existing `wave do` infrastructure with routing.

### 2. Routing Integration

Add routing rule to match prototype-style requests:

```yaml
# In wave.yaml runtime.routing
routing:
  rules:
    - pattern: "*prototype*"
      pipeline: prototype
      priority: 10
    - pattern: "*greenfield*"
      pipeline: prototype
      priority: 10
    - pattern: "*from scratch*"
      pipeline: prototype
      priority: 10
```

### 3. State Persistence Integration

Leverage existing SQLite state for:
- Phase completion tracking
- Artifact timestamp recording
- Resume-from-phase support

Required queries:
```sql
-- Check phase completion
SELECT completed_at FROM step_state
WHERE pipeline_run_id = ? AND step_id = ?;

-- Get artifact timestamp
SELECT created_at FROM artifacts
WHERE pipeline_run_id = ? AND step_id = ? AND name = ?;

-- Find stale downstream phases
SELECT step_id FROM step_state
WHERE pipeline_run_id = ?
AND completed_at < (SELECT MAX(created_at) FROM artifacts WHERE step_id IN (?));
```

### 4. Speckit Integration

**Detection**:
```go
func hasSpeckit(workspacePath string) bool {
    _, err := os.Stat(filepath.Join(workspacePath, ".specify"))
    return err == nil
}
```

**Invocation** (via craftsman persona):
```yaml
exec:
  type: prompt
  source: |
    {{ if .HasSpeckit }}
    Run speckit to generate specification:
    /speckit.spec "{{ input }}"
    {{ else }}
    Generate specification manually following the spec-phase schema.
    {{ end }}
```

### 5. Event Emission

Use existing event infrastructure to emit phase events:

```json
{"event": "phase_started", "phase": "spec", "timestamp": "2026-02-02T10:00:00Z"}
{"event": "phase_completed", "phase": "spec", "artifacts": ["specification.json"], "duration_ms": 45000}
{"event": "contract_validated", "phase": "spec", "schema": "spec-phase.schema.json", "result": "pass"}
{"event": "phase_started", "phase": "docs", "timestamp": "2026-02-02T10:00:45Z"}
```

---

## Implementation Tasks (High-Level)

1. **Create Pipeline Definition**
   - Write `.wave/pipelines/prototype.yaml` with all phases
   - Test pipeline parsing with `wave validate`

2. **Create Contract Schemas**
   - Write spec-phase.schema.json
   - Write docs-phase.schema.json
   - Write dummy-phase.schema.json
   - Test schema validation with sample artifacts

3. **Add Routing Rule**
   - Update wave.yaml with prototype routing
   - Test routing with `wave do --dry-run`

4. **Integration Testing**
   - Create test project in `.wave/test-fixtures/prototype/`
   - Run full pipeline with mock adapter
   - Verify artifact flow between phases

5. **Documentation**
   - Update CLAUDE.md with prototype pipeline usage
   - Add examples to wave.yaml comments

---

## Complexity Tracking

_No constitution violations. No complexity tracking required._

---

## Next Steps

1. Run `/speckit.plan` Phase 0 (research.md) to resolve any remaining questions
2. Run `/speckit.plan` Phase 1 (data-model.md, contracts/, quickstart.md) for detailed design
3. Run `/speckit.tasks` to generate implementation task breakdown
