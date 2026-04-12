# Pipelines Guide

Pipelines are DAGs (Directed Acyclic Graphs) that orchestrate multi-step agent workflows. Each step executes one persona in an isolated workspace, passing artifacts to dependent steps.

## Built-in Pipelines

Wave ships with 51 pipelines organized by use case:

### Development

| Pipeline | Steps | Use Case |
|----------|-------|----------|
| `impl-speckit` | specify → clarify → plan → tasks → checklist → analyze → implement → create-pr | Feature development |
| `impl-feature` | explore → plan → implement → publish | Feature planning and implementation |
| `impl-hotfix` | investigate → fix → verify | Production bugs |
| `impl-refactor` | analyze → test-baseline → refactor → verify | Safe refactoring |
| `impl-prototype` | spec → docs → dummy → implement → pr-create → ops-pr-review → pr-respond → pr-fix → pr-merge | Prototype-driven development |
| `impl-improve` | assess → implement → verify | Targeted code improvements |

### Quality & Debugging

| Pipeline | Steps | Use Case |
|----------|-------|----------|
| `ops-pr-review` | diff-analysis → security-review + quality-review → summary → publish | PR reviews |
| `test-gen` | analyze-coverage → generate-tests → verify-coverage | Test coverage |
| `ops-debug` | reproduce → hypothesize → investigate → fix | Root cause analysis |
| `audit-security` | scan → deep-dive → report | Security vulnerability audit |
| `audit-dead-code` | scan → clean → verify → create-pr | Dead code removal |
| `ops-supervise` | gather → evaluate → verdict | Work and process quality review |
| `test-smoke` | analyze → summarize | Configuration validation |

### Planning & Documentation

| Pipeline | Steps | Use Case |
|----------|-------|----------|
| `plan-task` | explore → breakdown → review | Task planning |
| `audit-doc` | scan-changes → analyze-consistency → compose-report → publish | Documentation consistency gate |
| `doc-fix` | scan-changes → analyze → fix-docs → create-pr | Documentation fix and commit |
| `doc-explain` | explore → analyze → document | Code explanation deep-dive |
| `plan-adr` | explore-context → analyze-options → draft-record → publish | Architecture Decision Records |
| `doc-changelog` | analyze-commits → categorize → format | Changelog generation |
| `doc-onboard` | survey → guide | New contributor onboarding |

### Issue Automation

| Pipeline | Steps | Use Case |
|----------|-------|----------|
| `impl-issue` | fetch-assess → plan → implement → create-pr | Implement GitHub issue end-to-end |
| `ops-implement-epic` | fetch-scope → implement-subissues → report | Implement all subissues from an epic |
| `plan-research` | fetch-issue → analyze-topics → research-topics → synthesize-report → post-comment | Research and report on issues |
| `ops-rewrite` | scan-and-score → apply-enhancements | Rewrite poorly documented issues |
| `ops-refresh` | gather-context → draft-update → apply-update | Refresh stale issues |
| `plan-scope` | fetch-epic → scope-and-create → verify-report | Decompose epics into child issues |

### Code Quality & Analysis

| Pipeline | Steps | Use Case |
|----------|-------|----------|
| `audit-consolidate` | scan | Detect redundant implementations and architectural drift |
| `audit-dead-code-issue` | scan → compose-report → create-issue | Scan for dead code and create a GitHub issue |
| `audit-dead-code-review` | scan → compose → publish | Scan PR-changed files for dead code and post a review comment |
| `audit-dual` | quality-scan + quality-detail, audit-security + security-detail → merge | Parallel code-quality and security analysis |
| `audit-dx` | audit | Evaluate developer experience for contributors and integrators |
| `audit-junk-code` | scan | Identify accidental complexity, conceptual misalignment, and technical debt |
| `audit-quality-loop` | quality-check | Supervise work, loop improvements until quality passes |
| `audit-ux` | audit | Evaluate user experience across CLI, TUI, docs, or workflows |

### Multi-Pipeline Orchestration

| Pipeline | Steps | Use Case |
|----------|-------|----------|
| `ops-epic-runner` | scope → implement-all | Scope an epic, implement each child issue sequentially |
| `ops-release-harden` | scan → triage → gate | Security scan, branch on severity, apply hotfixes, generate changelog |
| `impl-research` | research → implement → review | Research a GitHub issue, implement the solution, then review the PR |

### Wave Self-Evolution (wave-*)

| Pipeline | Steps | Use Case |
|----------|-------|----------|
| `wave-audit` | collect-inventory → audit-items → compose-triage → publish | Zero-trust implementation fidelity audit |
| `wave-bugfix` | investigate → fix | Investigate and fix a bug in Wave |
| `wave-evolve` | analyze → evolve → verify | Evolve Wave pipelines, personas, and prompts based on execution analysis |
| `wave-review` | review | Code review of Wave changes |
| `wave-security-audit` | threat-model → verify | Security audit of Wave's own codebase |
| `wave-test-hardening` | analyze-coverage → harden | Harden Wave's test suite — find gaps, add edge cases |

### Utility

| Pipeline | Steps | Use Case |
|----------|-------|----------|
| `ops-hello-world` | greet → verify | Smoke test / example |
| `wave-land` | commit → ship | Branch, commit, push, PR, merge |
| `impl-recinq` | gather → diverge → converge → probe → distill → simplify → report → publish | Double Diamond code simplification |

## Running Pipelines

```bash
# Run with input
wave run impl-speckit "add user authentication"

# Preview execution plan
wave run impl-hotfix --dry-run

# Start from specific step
wave run impl-speckit --from-step implement

# Custom timeout
wave run migrate --timeout 120
```

## Pipeline Structure

<div v-pre>

```yaml
kind: WavePipeline
metadata:
  name: my-pipeline
  description: "What this pipeline does"

steps:
  - id: first-step
    persona: navigator
    memory:
      strategy: fresh
    exec:
      type: prompt
      source: "Analyze: {{ input }}"
    output_artifacts:
      - name: analysis
        path: .wave/output/analysis.json
        type: json

  - id: second-step
    persona: craftsman
    dependencies: [first-step]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: first-step
          artifact: analysis
          as: context
    exec:
      type: prompt
      source: "Implement based on the analysis."
```

</div>

## Step Configuration

| Field | Required | Description |
|-------|----------|-------------|
| `id` | yes | Unique step identifier |
| `persona` | yes | References persona in wave.yaml |
| `memory.strategy` | yes | Always `fresh` (clean context) |
| `exec.type` | yes | `prompt` or `command` |
| `exec.source` | yes | Prompt template or shell command |
| `dependencies` | no | Step IDs that must complete first |
| `output_artifacts` | no | Files produced by this step |
| `handover.contract` | no | Validation for step output |

## Dependencies and DAG

Steps execute in dependency order. Independent steps run in parallel:

```yaml
steps:
  - id: navigate        # Runs first
  - id: specify
    dependencies: [navigate]
  - id: implement
    dependencies: [specify]
  - id: test            # Parallel with review
    dependencies: [implement]
  - id: review          # Parallel with test
    dependencies: [implement]
```

## Artifacts

### Declaring Output

```yaml
output_artifacts:
  - name: analysis
    path: .wave/output/analysis.json
    type: json
```

### Injecting into Steps

```yaml
memory:
  strategy: fresh
  inject_artifacts:
    - step: navigate
      artifact: analysis
      as: codebase_analysis
```

## Contracts

Validate step output before proceeding:

```yaml
handover:
  contract:
    type: json_schema
    schema_path: .wave/contracts/analysis.schema.json
    source: .wave/output/analysis.json
    on_failure: retry
    max_retries: 2
```

Contract types:
- `json_schema` — Validate against JSON Schema
- `typescript_interface` — Validate against TypeScript interface
- `test_suite` — Run test command, must pass
- `markdown_spec` — Validate Markdown structure

## Template Variables

| Variable | Description |
|----------|-------------|
| <code v-pre>{{ input }}</code> | Pipeline input from `--input` flag |
| <code v-pre>{{ task }}</code> | Current task in matrix strategy |

## Workspace Configuration

Each step gets an isolated workspace:

```yaml
workspace:
  mount:
    - source: ./
      target: /src
      mode: readonly    # or readwrite
```

Default structure:
```
.wave/workspaces/<pipeline-id>/<step-id>/
├── src/               # Mounted source
├── .wave/artifacts/   # Injected from dependencies
└── .wave/output/      # Step output
```

## Matrix Strategy (Parallel Fan-Out)

Spawn parallel instances from a task list:

```yaml
- id: plan
  output_artifacts:
    - name: tasks
      path: .wave/output/tasks.json

- id: execute
  dependencies: [plan]
  strategy:
    type: matrix
    items_source: plan/tasks.json
    item_key: task
    max_concurrency: 4
  exec:
    type: prompt
    source: "Execute: {{ task }}"
```

## Pipeline Examples

### impl-speckit

Full feature development workflow (8 steps):

```yaml
steps:
  - id: specify
    persona: implementer
    exec:
      source: "Create specification for: {{ input }}"

  - id: clarify
    persona: implementer
    dependencies: [specify]
    exec:
      source: "Clarify specification details"

  - id: plan
    persona: implementer
    dependencies: [clarify]
    exec:
      source: "Create implementation plan"

  - id: tasks
    persona: implementer
    dependencies: [plan]
    exec:
      source: "Break down into tasks"

  - id: checklist
    persona: implementer
    dependencies: [tasks]
    exec:
      source: "Create implementation checklist"

  - id: analyze
    persona: implementer
    dependencies: [checklist]
    exec:
      source: "Analyze codebase for implementation"

  - id: implement
    persona: craftsman
    dependencies: [analyze]
    exec:
      source: "Implement according to plan"

  - id: create-pr
    persona: craftsman
    dependencies: [implement]
    exec:
      source: "Create pull request"
```

### impl-hotfix

Fast-track bug fix:

```yaml
steps:
  - id: investigate
    persona: navigator
    exec:
      source: "Investigate: {{ input }}"

  - id: fix
    persona: craftsman
    dependencies: [investigate]
    exec:
      source: "Fix the issue with regression test"

  - id: verify
    persona: reviewer
    dependencies: [fix]
    exec:
      source: "Verify fix is safe for production"
```

### ops-debug

Systematic debugging:

```yaml
steps:
  - id: reproduce
    persona: debugger
    exec:
      source: "Reproduce: {{ input }}"

  - id: hypothesize
    persona: debugger
    dependencies: [reproduce]
    exec:
      source: "Form hypotheses about root cause"

  - id: investigate
    persona: debugger
    dependencies: [hypothesize]
    exec:
      source: "Test each hypothesis"

  - id: fix
    persona: craftsman
    dependencies: [investigate]
    exec:
      source: "Implement fix with regression test"
```

## Ad-Hoc Execution

For quick tasks without pipeline files:

```bash
wave do "fix the bug" --persona craftsman
```

Generates a 2-step pipeline (navigate → execute) automatically.

## Related Topics

- [Pipeline Schema Reference](/reference/pipeline-schema)
- [Contracts Guide](/guide/contracts)
- [Personas Guide](/guide/personas)
- [Relay Guide](/guide/relay)
