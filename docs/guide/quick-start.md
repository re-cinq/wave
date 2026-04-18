# Quick Start

Get Wave running on your project in 5 minutes.

## 1. Initialize Project

```bash
cd your-project
wave init
```

The interactive wizard guides you through adapter selection and pipeline configuration. On success, you'll see:

```
  ╦ ╦╔═╗╦  ╦╔═╗
  ║║║╠═╣╚╗╔╝║╣
  ╚╩╝╩ ╩ ╚╝ ╚═╝
  Multi-Agent Pipeline Orchestrator

  Project initialized successfully!

  Created:
    wave.yaml                Main manifest
    .agents/personas/          5 persona archetypes
    .agents/pipelines/         12 pipelines
    .agents/contracts/         4 JSON schema validators
    .agents/prompts/           8 prompt templates
    .agents/workspaces/        Ephemeral workspace root
    .agents/traces/            Audit log directory

  Next steps:
    1. Run 'wave validate' to check configuration
    2. Run 'wave run ops-hello-world "test"' to verify setup
    3. Run 'wave run plan-task "your feature"' to plan a task
```

## 2. Review Configuration

```yaml
# wave.yaml
apiVersion: v1
kind: WaveManifest
metadata:
  name: my-project
  description: "Example project"

adapters:
  claude:
    binary: claude
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

Expected output:
```
✓ Manifest validation passed
✓ All persona system prompt files exist
✓ Adapter binaries found on PATH
```

## 4. Run First Pipeline

```bash
wave run ops-hello-world "testing Wave"
```

The default TUI shows a progress bar and spinners. For text output, use `-o text`:

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

Artifacts are saved in `.agents/workspaces/<pipeline-id>/<step-id>/`. Each step produces its own workspace.

## Quick Commands

```bash
# Ad-hoc task (no full pipeline)
wave do "fix typo in README"

# Resume from a specific step
wave run impl-speckit --from-step implement

# Clean up workspaces
wave clean
```

## What's Next?

- [Understand personas](/concepts/personas)
- [Design pipelines](/concepts/pipelines)
- [Add contracts](/concepts/contracts)
- [Explore examples](/examples/)