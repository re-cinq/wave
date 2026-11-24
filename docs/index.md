---
layout: home

hero:
  name: Wave
  text: Declare your agents. Validate their output.
  tagline: AI coding agents defined in YAML like Kubernetes resources, with typed contracts checked at every handoff.
  actions:
    - theme: brand
      text: Quickstart
      link: /quickstart
    - theme: alt
      text: Concepts
      link: /concepts/pipelines
    - theme: alt
      text: GitHub
      link: https://github.com/re-cinq/wave

features:
  - title: Agents as YAML
    details: A wave.yaml manifest declares adapters and personas. Pipelines are DAGs of steps, each run by one persona in its own git worktree. Diff, review and version them like infrastructure.
    link: /concepts/manifests
  - title: Typed contracts with rework and retry
    details: JSON schema, test suite, test diff, non-empty file, LLM judge. Wave validates output outside the model before the next step starts, then retries or reworks the step with the error fed back.
    link: /concepts/contracts
  - title: Pipelines on demand
    details: wave meta drafts a new pipeline and its contracts from a one-line task. Failed runs resume from the failed step instead of starting over.
    link: /guide/meta-pipelines
  - title: Adapter-agnostic
    details: Wraps the Claude Code, Codex, Gemini and OpenCode CLIs. Moving a persona to another adapter is one field in the manifest.
    link: /concepts/adapters
---

## Try it

```bash
curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/scripts/install.sh | sh
cd your-project
wave init
wave run ops-hello-world "testing Wave"
```

## A step with a contract

```yaml
- id: verify
  persona: craftsman
  dependencies: [greet]
  output_artifacts:
    - name: result
      path: .agents/output/result.json
      type: json
  handover:
    contract:
      type: json_schema
      source: .agents/output/result.json
      schema_path: .agents/contracts/hello-world-result.schema.json
      on_failure: fail
  retry:
    policy: standard
    max_attempts: 2
```

`on_failure` can be `fail`, `retry`, `rework` (run a named `rework_step` with the validation error),
`skip`, `continue` or `warn`. See [Contracts](/guide/contracts) and [Retry Policies](/guide/retry-policies).

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

## Status

Pre-1.0. The Claude Code adapter is exercised heavily; the others are lightly tested. Shipped
personas run with full tool access; permission scoping is available in the manifest but not
preconfigured. GitHub is the verified forge.
