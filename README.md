# Wave - declare your agents, validate their output

Wave is a small Go binary that runs AI coding agents from declarative YAML, the way Kubernetes
runs services from manifests. Every handoff between agents is checked by a typed contract.

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/re-cinq/wave)](https://github.com/re-cinq/wave/releases)

## Why

- **Agents as YAML.** A `wave.yaml` manifest declares adapters and personas; pipelines are DAGs of
  steps. Diffable, reviewable, versioned in git like any other infrastructure.
- **Typed contracts between steps.** JSON schema, test suite, test diff, non-empty file, LLM judge.
  Wave validates output outside the model before the next step starts, and retries or reworks the
  step with the validation error fed back.
- **Pipelines on demand.** `wave meta "<task>"` drafts a new pipeline plus its contracts when none
  of the shipped ones fit. Failed runs resume from the failed step instead of starting over.
- **Adapter-agnostic.** Wraps the Claude Code, Codex, Gemini and OpenCode CLIs as subprocesses.
  Switching a persona to another adapter is one field.

Wave is not a chat UI and not a CI system. Each step runs headless in its own git worktree with
fresh context.

## 60-second start

```bash
curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/scripts/install.sh | sh
cd your-project
wave init                              # writes wave.yaml and .agents/
wave run ops-hello-world "testing Wave"
wave status                            # last runs and their state
```

## The three objects

A persona in `wave.yaml`:

```yaml
personas:
  craftsman:
    adapter: claude
    description: Code implementation and testing
    system_prompt_file: .agents/personas/craftsman.md
```

A pipeline (`.agents/pipelines/ops-hello-world.yaml`, shortened):

```yaml
kind: WavePipeline
metadata:
  name: ops-hello-world
steps:
  - id: greet
    persona: craftsman
    exec: { type: prompt, source: "Write greeting.txt with: Hello from Wave! {{ input }}" }
    output_artifacts: [{ name: greeting, path: greeting.txt, type: text }]
    handover:
      contract: { type: non_empty_file, source: greeting.txt, on_failure: fail }
  - id: verify
    persona: craftsman
    dependencies: [greet]
    memory:
      inject_artifacts: [{ step: greet, artifact: greeting, as: greeting_file }]
    exec: { type: prompt, source: "Verify greeting_file. Produce output matching the contract schema." }
    output_artifacts: [{ name: result, path: .agents/output/result.json, type: json }]
    handover:
      contract:
        type: json_schema
        source: .agents/output/result.json
        schema_path: .agents/contracts/hello-world-result.schema.json
        on_failure: fail
    retry: { policy: standard, max_attempts: 2 }
```

The contract (`.agents/contracts/hello-world-result.schema.json`):

```json
{
  "type": "object",
  "required": ["success", "message"],
  "properties": {
    "success": { "type": "boolean" },
    "message": { "type": "string" }
  }
}
```

## When a contract fails

`on_failure` decides what happens: `fail`, `retry`, `rework` (run a named `rework_step` with the
validation error), `skip`, `continue` or `warn`. `impl-issue` uses `rework` on its test suite,
`test_diff` and `test_count_baseline` contracts. A run that still fails can be resumed:

```bash
wave resume <run-id>                     # restarts at the failed step
wave resume <run-id> --from-step verify  # or at a step you pick
```

## Make a pipeline on demand

```bash
wave meta "add a caching layer" --dry-run           # show the generated pipeline
wave meta "add a caching layer" --save cache.yaml   # keep it for reuse
wave pipeline create --template ops-hello-world     # scaffold from a shipped pipeline
```

## Shipped pipelines

| Pipeline | What it does |
|---|---|
| `ops-hello-world` | Two-step smoke test of the contract loop |
| `impl-issue` | GitHub issue -> plan -> code in a worktree -> PR, with rework on failing contracts |
| `ops-pr-review` | Security, quality and slop review of a PR, posts a verdict |
| `plan-task` | Break a feature into ordered tasks with affected files |
| `plan-research` | Research an issue and post findings |
| `plan-scope` | Split an epic into scoped issues |
| `audit-security` | Security scan with typed findings |
| `audit-architecture` | Architecture review with typed findings |
| `ops-parallel-audit` | Composition: runs both audits in parallel and merges the findings |

## Compared with scripting subagents in Claude Code

Where Wave helps: workflows live as declarative YAML in git; contracts are enforced by the
orchestrator, not by the model; runs persist in SQLite and resume, fork or rewind; the same
pipeline runs on another adapter; every step gets an isolated worktree.

Where it does not: one more binary and a manifest to set up; per-step subprocess startup; steps are
headless, so you cannot steer mid-step; agents share files (artifacts), not in-memory context.

## Status and limits

- Pre-1.0. Expect breaking changes between releases.
- The Claude Code adapter is exercised heavily; Codex, Gemini and OpenCode adapters exist but are
  lightly tested.
- Shipped personas run with full tool access (`allowed_tools: ["*"]`). Permission scoping is
  available in the manifest but not preconfigured.
- GitHub is the verified forge. Other forges are detected from git remotes but unverified.

## Docs, contributing, license

- Documentation: https://re-cinq.github.io/wave/
- Contributing: [CONTRIBUTING.md](CONTRIBUTING.md)
- License: [MIT](LICENSE)
