# Manifest Reference

The `wave.yaml` manifest is the single source of truth for Wave configuration.

## Minimal Manifest

```yaml
apiVersion: v1
kind: WaveManifest
metadata:
  name: my-project
adapters:
  claude:
    binary: claude
    mode: headless
personas:
  navigator:
    adapter: claude
    system_prompt_file: .wave/personas/navigator.md
runtime:
  workspace_root: /tmp/wave
```

Copy this to `wave.yaml` and run `wave validate` to verify.

---

## Complete Example

```yaml
apiVersion: v1
kind: WaveManifest
metadata:
  name: acme-backend
  description: "Backend API service"
  repo: https://github.com/acme/backend

adapters:
  claude:
    binary: claude
    mode: headless
    output_format: json
    project_files:
      - CLAUDE.md
    default_permissions:
      allowed_tools: ["Read", "Write", "Bash"]
      deny: []

personas:
  navigator:
    adapter: claude
    description: "Read-only codebase exploration"
    system_prompt_file: .wave/personas/navigator.md
    temperature: 0.1
    permissions:
      allowed_tools: ["Read", "Glob", "Grep"]
      deny: ["Write(*)", "Edit(*)"]

  craftsman:
    adapter: claude
    description: "Implementation and testing"
    system_prompt_file: .wave/personas/craftsman.md
    temperature: 0.7
    permissions:
      allowed_tools: ["Read", "Write", "Edit", "Bash"]
      deny: ["Bash(rm -rf /*)"]

  reviewer:
    adapter: claude
    description: "Security and code review"
    system_prompt_file: .wave/personas/reviewer.md
    temperature: 0.1
    permissions:
      allowed_tools: ["Read", "Grep", "Glob", "Bash(npm audit*)"]
      deny: ["Write(*)", "Edit(*)"]

runtime:
  workspace_root: /tmp/wave
  max_concurrent_workers: 5
  default_timeout_minutes: 30
  relay:
    token_threshold_percent: 80
    strategy: summarize_to_checkpoint
  audit:
    log_dir: .wave/traces/
    log_all_tool_calls: true
```

---

## Top-Level Fields

| Field | Required | Description |
|-------|----------|-------------|
| `apiVersion` | yes | Schema version (`v1`) |
| `kind` | yes | Must be `WaveManifest` |
| `metadata` | yes | Project identification |
| `adapters` | yes | LLM CLI configurations |
| `personas` | yes | Agent configurations |
| `runtime` | yes | Execution settings |
| `skill_mounts` | no | External skill paths |

---

## Metadata

```yaml
metadata:
  name: my-project
  description: "Project description"
  repo: https://github.com/org/repo
```

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Project identifier |
| `description` | no | Human-readable description |
| `repo` | no | Repository URL |

---

## Adapters

An adapter wraps an LLM CLI for subprocess execution.

```yaml
adapters:
  claude:
    binary: claude
    mode: headless
    output_format: json
    project_files:
      - CLAUDE.md
      - .claude/settings.json
    default_permissions:
      allowed_tools: ["Read", "Write"]
      deny: ["Bash(rm *)"]
```

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `binary` | yes | - | CLI binary name (must be on PATH) |
| `mode` | yes | - | Execution mode (`headless`) |
| `output_format` | no | `json` | Expected output format |
| `project_files` | no | `[]` | Files to copy into workspaces |
| `default_permissions` | no | allow all | Default tool permissions |

---

## Personas

A persona defines an AI agent with specific permissions and behavior.

### Read-Only Persona

```yaml
personas:
  navigator:
    adapter: claude
    description: "Read-only codebase exploration"
    system_prompt_file: .wave/personas/navigator.md
    temperature: 0.1
    permissions:
      allowed_tools: ["Read", "Glob", "Grep"]
      deny: ["Write(*)", "Edit(*)", "Bash(*)"]
```

### Full-Access Persona

```yaml
personas:
  craftsman:
    adapter: claude
    description: "Implementation and testing"
    system_prompt_file: .wave/personas/craftsman.md
    temperature: 0.7
    permissions:
      allowed_tools: ["Read", "Write", "Edit", "Bash"]
      deny: ["Bash(rm -rf /*)"]
```

### Persona Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `adapter` | yes | - | References adapter key |
| `system_prompt_file` | yes | - | Path to system prompt |
| `description` | no | `""` | Human-readable purpose |
| `temperature` | no | adapter default | LLM temperature (0.0-1.0) |
| `permissions` | no | inherit adapter | Tool access control |
| `hooks` | no | `{}` | Pre/post tool hooks |

### Temperature Guidelines

| Range | Use Case |
|-------|----------|
| 0.0-0.2 | Deterministic: analysis, review |
| 0.3-0.5 | Balanced: specification, planning |
| 0.6-0.8 | Creative: implementation |

---

## Permissions

Tool access control using glob patterns.

```yaml
permissions:
  allowed_tools:
    - "Read"                # All Read calls
    - "Write(src/*.ts)"     # Write to TypeScript in src/
    - "Bash(npm test*)"     # Only npm test commands
  deny:
    - "Write(*.env)"        # Never write env files
    - "Bash(rm -rf *)"      # Block destructive commands
```

**Evaluation order:**
1. Check `deny` - any match blocks the call
2. Check `allowed_tools` - any match permits the call
3. No match - blocked (implicit deny)

### Common Patterns

| Pattern | Matches |
|---------|---------|
| `Read` | All Read calls |
| `Write(*)` | All Write calls |
| `Write(src/*.ts)` | Write to .ts files in src/ |
| `Bash(git *)` | Git commands only |
| `Bash(npm test*)` | npm test commands |
| `*` | All tool calls |

---

## Hooks

Execute shell commands at tool call boundaries.

```yaml
personas:
  craftsman:
    adapter: claude
    system_prompt_file: .wave/personas/craftsman.md
    hooks:
      PreToolUse:
        - matcher: "Bash(git commit*)"
          command: ".wave/hooks/pre-commit-lint.sh"
      PostToolUse:
        - matcher: "Write(src/**)"
          command: "npm test --silent"
```

**PreToolUse**: Non-zero exit blocks the tool call.

**PostToolUse**: Informational only, does not block.

---

## Runtime

Global execution settings.

```yaml
runtime:
  workspace_root: /tmp/wave
  max_concurrent_workers: 5
  default_timeout_minutes: 30
  relay:
    token_threshold_percent: 80
    strategy: summarize_to_checkpoint
  audit:
    log_dir: .wave/traces/
    log_all_tool_calls: true
    log_all_file_operations: false
  meta_pipeline:
    max_depth: 2
    max_total_steps: 20
    timeout_minutes: 60
```

### Runtime Fields

| Field | Default | Description |
|-------|---------|-------------|
| `workspace_root` | `/tmp/wave` | Workspace directory |
| `max_concurrent_workers` | `5` | Parallel matrix workers |
| `default_timeout_minutes` | `30` | Per-step timeout |

### Relay Settings

| Field | Default | Description |
|-------|---------|-------------|
| `token_threshold_percent` | `80` | Context limit trigger |
| `strategy` | `summarize_to_checkpoint` | Compaction strategy |

### Audit Settings

| Field | Default | Description |
|-------|---------|-------------|
| `log_dir` | `.wave/traces/` | Audit log directory |
| `log_all_tool_calls` | `false` | Log every tool call |
| `log_all_file_operations` | `false` | Log file operations |

### Meta-Pipeline Limits

| Field | Default | Description |
|-------|---------|-------------|
| `max_depth` | `2` | Recursion limit |
| `max_total_steps` | `20` | Total steps across levels |
| `max_total_tokens` | `500000` | Token consumption limit |
| `timeout_minutes` | `60` | Hard timeout |

---

## Skill Mounts

External skill discovery paths.

```yaml
skill_mounts:
  - path: .wave/skills/       # Project-local
  - path: ~/.wave/skills/     # User-global
  - path: /opt/wave/skills/   # System-wide
```

Skills are discovered in order. Project-local takes precedence.

---

## Validation

```bash
wave validate
```

**Output:**
```
Validating wave.yaml...
  Adapters: 1 defined
  Personas: 3 defined
  Pipelines: 2 discovered

All validation checks passed.
```

### Validation Checks

| Check | Severity |
|-------|----------|
| Adapter references valid | error |
| System prompt files exist | error |
| Hook scripts exist | error |
| Binary on PATH | warning |
| Required fields present | error |
| Value ranges valid | error |

---

## Next Steps

- [Pipelines](/concepts/pipelines) - Define multi-step workflows
- [Personas](/concepts/personas) - Configure AI agents
- [Pipeline Schema](/reference/pipeline-schema) - Pipeline configuration
