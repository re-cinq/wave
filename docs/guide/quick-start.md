# Quick Start

Get Wave running on your project in 5 minutes.

## 1. Initialize Project

```bash
cd your-project
wave init
```

In a terminal, init asks for the adapter (`claude` or `codex`; the first one found on `PATH` is the default), the model (pinned for every tier: `sonnet` for claude, `gpt-6.2-sol` for codex) and any test/lint/build command it could not detect. Use `--yes` to accept the defaults, or pass `--adapter`, `--model`, `--test-command`, `--lint-command` and `--build-command`. On success, you'll see something like:

```
  Project initialized successfully!

  Adapter: claude (model sonnet for every tier)

  Created:
    wave.yaml                  Main manifest
    .agents/personas/          12 personas
    .agents/pipelines/         9 pipelines
    .agents/contracts/         17 contracts
    .agents/prompts/           5 prompt templates
    .agents/workspaces/        Ephemeral workspace root
    .agents/traces/            Audit log directory
    AGENTS.md                  Project instructions copied into every agent workspace
    CLAUDE.md                  Imports AGENTS.md for Claude Code
    .gitignore                 Ignores Wave runtime state

  Pipelines: audit-architecture, audit-security, impl-issue, ops-hello-world, ops-parallel-audit, ops-pr-review, plan-research, plan-scope, plan-task

  Next steps:
    1. Run 'wave validate' to check configuration
    2. Run 'wave run ops-hello-world "test"' to verify setup
    3. Run 'wave run plan-task "your feature"' to plan a task
```

If init cannot detect your test, lint or build command it says so under "Missing project commands". Set them in the `project:` block of `wave.yaml` or run `wave doctor --optimize`.

## 2. Review Configuration

```yaml
# wave.yaml
apiVersion: v1
kind: WaveManifest
metadata:
  name: my-project
  description: "Example project"

project:
  language: go
  test_command: go test ./...
  lint_command: go vet ./...
  build_command: go build ./...

adapters:
  claude:
    binary: claude
    default_model: sonnet
    tier_models:
      cheapest: sonnet
      balanced: sonnet
      strongest: sonnet
    mode: headless
    output_format: json

personas:
  navigator:
    adapter: claude
    system_prompt_file: .agents/personas/navigator.md
    temperature: 0.1
    permissions:
      allowed_tools: ["Read", "Bash(git *)"]
      deny: ["Write(*)"]

runtime:
  workspace_root: .agents/workspaces
  max_concurrent_workers: 3
```

## 3. Validate Configuration

```bash
wave validate
```

It prints `Validation successful`, preceded by any warnings.

For a broader health check (git, adapter binaries, forge, tools) run:

```bash
wave doctor
```

## 4. Run First Pipeline

```bash
wave run ops-hello-world "testing Wave"
```

The default output shows a progress display. For text output, use `-o text`:

```bash
wave run ops-hello-world "testing Wave" -o text
```

```
[10:00:01] → greet (craftsman)
[10:00:01]   greet: Executing agent
[10:00:05] ✓ greet completed (4.0s, 0k tokens)
[10:00:05] → verify (navigator)
[10:00:12] ✓ verify completed (6.9s, 0k tokens)
```

## 5. Check Results

Artifacts are saved in `.agents/workspaces/<pipeline>-<run-id>/<step-id>/`. Each step produces its own workspace.

## Quick Commands

```bash
# Generate a pipeline for a one-off task
wave meta "fix typo in README" --dry-run

# Resume from a specific step
wave run impl-issue --from-step implement

# Clean up workspaces
wave clean
```

## What's Next?

- [Understand personas](/concepts/personas)
- [Design pipelines](/concepts/pipelines)
- [Add contracts](/concepts/contracts)
- [Explore examples](/examples/)