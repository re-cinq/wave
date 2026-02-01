# Configuration Guide

Muzzle uses a single `muzzle.yaml` file as the source of truth for all configuration.

## Manifest Structure

```yaml
apiVersion: v1
kind: MuzzleManifest
metadata:
  name: string          # Required
  description: string     # Optional
  repo: string           # Optional

adapters:                 # Required
  <adapter-name>:
    binary: string           # Required - CLI binary name
    mode: string            # Required - "headless"
    output_format: string     # Optional - Default "json"
    project_files: []        # Optional - Files to project
    default_permissions: {}  # Optional - Default tool perms
    hooks_template: string    # Optional - Hook script dir

personas:                  # Required
  <persona-name>:
    adapter: string           # Required - References adapters.<name>
    description: string        # Optional
    system_prompt_file: string # Required - Path to prompt file
    temperature: float        # Optional - 0.0-1.0
    permissions: {}          # Optional - Override adapter perms
    hooks: {}               # Optional - Pre/Post hooks

runtime:                    # Required
  workspace_root: string         # Optional - Default "/tmp/muzzle"
  max_concurrent_workers: int   # Optional - Default 5
  default_timeout_minutes: int   # Optional - Default 30
  relay: {}                   # Optional - Relay config
  audit: {}                   # Optional - Audit config
  meta_pipeline: {}            # Optional - Meta pipeline limits

skill_mounts: []            # Optional - Skill discovery paths
```

## Adapters

Adapters wrap LLM CLIs for Muzzle to use.

### Claude Code Adapter

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
      allowed_tools: ["Read", "Write", "Bash"]
      deny: []
```

## Personas

Personas define how agents behave.

### Navigator Persona

```yaml
personas:
  navigator:
    adapter: claude
    system_prompt_file: .muzzle/personas/navigator.md
    temperature: 0.1
    permissions:
      allowed_tools: ["Read", "Bash(git log, git status)"]
      deny: ["Write(*)", "Bash(git commit, git push)"]
    hooks:
      PreToolUse:
        - matcher: "Bash(rm -rf)"
          command: "echo 'DANGER: Destructive command blocked'"
```

### Craftsman Persona

```yaml
personas:
  craftsman:
    adapter: claude
    system_prompt_file: .muzzle/personas/craftsman.md
    temperature: 0.7
    permissions:
      allowed_tools: ["Read", "Write", "Bash"]
    hooks:
      PostToolUse:
        - matcher: "Bash(git commit)"
          command: "npm test"
```

## Runtime Configuration

### Workspace Management

```yaml
runtime:
  workspace_root: /tmp/muzzle
  max_concurrent_workers: 3
  default_timeout_minutes: 30
```

### Context Relay

```yaml
runtime:
  relay:
    token_threshold_percent: 80
    strategy: "summarize_to_checkpoint"
```

### Auditing

```yaml
runtime:
  audit:
    log_dir: .muzzle/traces/
    log_all_tool_calls: true
    log_all_file_operations: false
```

### Meta Pipeline Limits

```yaml
runtime:
  meta_pipeline:
    max_depth: 2
    max_total_steps: 20
    max_total_tokens: 500000
    timeout_minutes: 60
```

## Skill Mounts

Mount external skill directories:

```yaml
skill_mounts:
  - path: ./my-muzzle-skills/
  - path: ~/.muzzle/skills/
```

## Validation Rules

Muzzle validates:

1. Every persona must reference a defined adapter
2. All `system_prompt_file` paths must exist
3. All hook script paths must exist
4. Adapter binaries must be on PATH (warning only)
5. No circular persona references
6. Required fields must be present
7. Types must be valid (string, int, float, array)

## Common Patterns

### Project-Specific Configuration

```yaml
# For a React TypeScript project
metadata:
  name: "my-react-app"
  description: "React TypeScript application"

adapters:
  claude:
    project_files:
      - "src/**/*.{ts,tsx}"
      - "package.json"
      - "tsconfig.json"

runtime:
  default_timeout_minutes: 45  # Longer timeout for build steps
```

### Environment-Specific Overrides

```yaml
# Development
runtime:
  workspace_root: ./muzzle-workspaces
  audit:
    log_all_tool_calls: false

# Production
runtime:
  workspace_root: /tmp/muzzle
  audit:
    log_all_tool_calls: true
    log_all_file_operations: true
```