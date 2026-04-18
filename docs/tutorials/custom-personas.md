# Creating Custom Personas

This tutorial walks through creating a custom security scanner persona.

## Prerequisites

- Wave initialized (`wave init`)
- Working `wave.yaml` manifest

## Step 1: Plan the Persona

- **Purpose:** Security vulnerability scanning
- **Can read:** All source code
- **Can write:** Only security reports
- **Commands:** Security tools only
- **Temperature:** Low (deterministic)

## Step 2: Create the System Prompt

Create `.agents/personas/security-scanner.md`:

```markdown
# Security Scanner

You are a security-focused code analyst identifying vulnerabilities.

## Output Format

Produce JSON: {"findings": [{"severity": "high", "file": "path", "description": "..."}]}

## Constraints

- Never modify source code
- Report findings without fixing
```

## Step 3: Configure Permissions

Add to `wave.yaml`:

```yaml
personas:
  security-scanner:
    adapter: claude
    description: "Security vulnerability scanner"
    system_prompt_file: .agents/personas/security-scanner.md
    temperature: 0.1
    permissions:
      allowed_tools:
        - Read
        - Glob
        - Grep
        - Bash(npm audit*)
        - Write(.agents/reports/*)
      deny:
        - Edit(*)
        - Bash(rm *)
```

### Permission Patterns

| Pattern | Effect |
|---------|--------|
| `Read` | Allow all file reads |
| `Write(path/*)` | Write only under `path/` |
| `Bash(cmd*)` | Commands starting with `cmd` |

## Step 4: Add Hooks (Optional)

Create `.agents/hooks/security-pre-scan.sh`:

```bash
#!/bin/bash
mkdir -p .agents/reports
```

Add to persona:

```yaml
hooks:
  PreToolUse:
    - matcher: "Bash(npm audit*)"
      command: ".agents/hooks/security-pre-scan.sh"
```

## Step 5: Create a Pipeline

Create `.agents/pipelines/security-audit.yaml`:

```yaml
kind: WavePipeline
metadata:
  name: security-audit
steps:
  - id: scan
    persona: security-scanner
    memory:
      strategy: fresh
    workspace:
      mount:
        - source: ./
          target: /src
          mode: readonly
    exec:
      type: prompt
      source: Perform a security audit focusing on: {{ input }}
    output_artifacts:
      - name: report
        path: .agents/output/security-report.json
```

## Step 6: Validate and Test

```bash
wave validate --verbose
wave run security-audit \
  --input "authentication"
```

## Next Steps

- [Pipeline design tutorial](/tutorials/pipeline-design)
- [Built-in persona archetypes](/reference/manifest-schema#built-in-persona-archetypes)
