# Quick Start

Get Wave running on your project in 5 minutes.

## 1. Initialize Project

```bash
cd your-project
wave init
```

This creates:
- `wave.yaml` - Manifest with default adapter and personas
- `.wave/personas/` - System prompt files
- `.wave/pipelines/` - Example pipeline definitions

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
    system_prompt_file: .wave/personas/navigator.md
    temperature: 0.1
    permissions:
      allowed_tools: ["Read", "Bash(git *)"]
      deny: ["Write(*)"]

runtime:
  workspace_root: /tmp/wave
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
wave run speckit-flow "add user authentication"
```

The default TUI shows a progress bar and spinners. For text output, use `-o text`:

```bash
wave run speckit-flow "add user authentication" -o text
```

```
[10:00:01] → navigate (navigator)
[10:00:01]   navigate: Executing agent
[10:01:30] ✓ navigate completed (90.0s, 2.0k tokens)
[10:01:31] → implement (craftsman)
...
```

## 5. Check Results

Artifacts are saved in `/tmp/wave/<pipeline-id>/<step-id>/`. Each step produces its own workspace.

## Quick Commands

```bash
# Ad-hoc task (no full pipeline)
wave do "fix typo in README"

# Resume from a specific step
wave run speckit-flow --from-step implement

# Clean up workspaces
wave clean
```

## What's Next?

- [Understand personas](/concepts/personas)
- [Design pipelines](/concepts/pipelines)
- [Add contracts](/concepts/contracts)
- [Explore examples](/examples/)