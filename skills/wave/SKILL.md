---
name: wave
description: Expert Wave multi-agent pipeline orchestrator development including manifest configuration, pipeline authoring, persona management, and CLI operations
---

# Wave — Multi-Agent Pipeline Orchestrator

Go CLI orchestrating multi-step AI workflows. Each pipeline step runs in an isolated workspace under a named persona with enforced permissions, artifact chaining, and contract validation.

## Manifest (`wave.yaml`)

```yaml
apiVersion: v1
kind: WaveManifest
metadata:
  name: my-project

adapters:
  claude:
    binary: claude
    mode: headless
    output_format: json
    project_files: [CLAUDE.md, .claude/settings.json]
    default_permissions:
      allowed_tools: [Read, Write, Edit, Bash]

personas:
  navigator:
    adapter: claude
    system_prompt_file: .wave/personas/navigator.md
    model: sonnet                        # opus | sonnet | haiku
    permissions:
      allowed_tools: [Read, Glob, Grep, "Bash(git log*)", "Bash(git status*)"]
      deny: ["Write(*)", "Edit(*)", "Bash(git commit*)", "Bash(git push*)"]

runtime:
  workspace_root: .wave/workspaces
  max_concurrent_workers: 5
  default_timeout_minutes: 30
  relay:
    token_threshold_percent: 80
    strategy: summarize_to_checkpoint
    summarizer_persona: summarizer

skill_mounts:
  - path: .wave/skills/
```

## Pipeline (`.wave/pipelines/<name>.yaml`)

```yaml
kind: WavePipeline
metadata:
  name: gh-pr-review
  release: true

steps:
  - id: analyze
    persona: navigator
    dependencies: []
    memory:
      strategy: fresh
      inject_artifacts:
        - step: prior-step
          artifact: artifact-name
          as: local-filename            # available at artifacts/<as>
    exec:
      type: prompt
      source: |
        Analyze the code for: {{ input }}
    output_artifacts:
      - name: analysis
        path: output/analysis.json
        type: json
    handover:
      contract:
        type: json_schema               # or: command
        schema_path: .wave/contracts/analysis.schema.json
        source: output/analysis.json
        must_pass: true
        on_failure: retry
        max_retries: 2
```

## Built-in Personas

| Persona | Role |
|---------|------|
| `navigator` | Read-only codebase exploration |
| `craftsman` | Code implementation & testing |
| `implementer` | Full execution specialist |
| `philosopher` | Architecture & specification |
| `auditor` | Security review & QA |
| `summarizer` | Context compaction for relay |
| `reviewer` | Quality review & validation |
| `github-commenter` | Post GitHub issue comments |

Permission syntax: `Read`, `Write(.wave/specs/*)`, `Bash(git log*)`, `Bash(*)` in deny

## Artifacts

```yaml
output_artifacts:             # produced by this step
  - name: analysis
    path: output/data.json
    type: json

memory:
  inject_artifacts:           # consumed from prior step
    - step: analyze
      artifact: analysis
      as: analysis_data       # available at artifacts/analysis_data
```

## Key CLI Commands

```bash
wave run gh-pr-review "Review auth module"
wave run hotfix --dry-run
wave run <pipeline> --from-step <step> --force   # resume
wave do "fix the login bug"                       # ad-hoc task
wave meta "implement user auth"                   # dynamic pipeline
wave status --all
wave logs --step investigate --errors --follow
wave artifacts --step analyze --export ./out
wave list pipelines
wave list runs --limit 20 --run-status completed
wave validate --pipeline gh-pr-review
wave clean --all --keep-last 5 --dry-run
wave cancel --force
```

## Key Patterns

- **Template vars**: `{{ input }}`, `{{ timestamp }}`
- **Parallel steps**: Steps with independent `dependencies` run concurrently
- **Contract retry**: Failed contracts feed errors back to persona (up to `max_retries`)
- **Output formats**: `auto` (TUI on TTY), `json` (NDJSON), `text`, `quiet`

## Development

```bash
go test ./...
go test -race ./...
go run ./cmd/wave run hello-world "test"
wave run hello-world "test" --mock
```

## Complete Reference

For exhaustive patterns, examples, and advanced usage see:

**[`references/full-reference.md`](references/full-reference.md)**
