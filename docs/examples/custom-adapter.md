# Example: Custom Adapter

How to wrap a custom LLM CLI as a Muzzle adapter, enabling it to participate in pipelines alongside Claude Code.

## Prerequisites

Your LLM CLI must support:

1. **Prompt input** — accept a prompt via command-line argument or stdin.
2. **Headless mode** — run non-interactively as a subprocess.
3. **Structured output** — produce JSON output (preferred) or parseable text.

## Step 1: Define the Adapter

Add the adapter to your `muzzle.yaml`:

```yaml
adapters:
  claude:
    binary: claude
    mode: headless
    output_format: json

  # Custom adapter for a local LLM
  local-llm:
    binary: ollama
    mode: headless
    output_format: json
    default_permissions:
      allowed_tools: ["Read", "Write"]
      deny: ["Bash(rm *)", "Bash(curl *)"]
```

## Step 2: Create Personas Using the Adapter

```yaml
personas:
  # Fast navigator using local model
  local-navigator:
    adapter: local-llm
    description: "Quick codebase analysis with local model"
    system_prompt_file: .muzzle/personas/navigator.md
    temperature: 0.1
    permissions:
      allowed_tools: ["Read", "Glob", "Grep"]
      deny: ["Write(*)", "Bash(*)"]

  # Cloud-powered implementation
  craftsman:
    adapter: claude
    description: "Implementation with Claude"
    system_prompt_file: .muzzle/personas/craftsman.md
    temperature: 0.7
```

## Step 3: Mix Adapters in a Pipeline

Use different adapters for different steps based on their requirements:

```yaml
kind: MuzzlePipeline
metadata:
  name: hybrid-flow
  description: "Local model for analysis, cloud model for implementation"

steps:
  - id: navigate
    persona: local-navigator    # Uses local LLM — fast, free
    memory:
      strategy: fresh
    exec:
      type: prompt
      source: "Analyze the codebase structure for: {{ input }}"
    output_artifacts:
      - name: analysis
        path: output/analysis.json

  - id: implement
    persona: craftsman          # Uses Claude — higher quality
    dependencies: [navigate]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: navigate
          artifact: analysis
          as: context
    exec:
      type: prompt
      source: "Implement based on the analysis: {{ input }}"
    handover:
      contract:
        type: test_suite
        command: "npm test"
        must_pass: true
```

## Step 4: Validate

```bash
$ muzzle validate --verbose
✓ Adapter 'claude' binary found on PATH
✓ Adapter 'local-llm' binary found on PATH
✓ Persona 'local-navigator' references adapter 'local-llm'
✓ Persona 'craftsman' references adapter 'claude'
✓ Pipeline 'hybrid-flow' DAG is valid
```

## Adapter Wrapper Script

If your LLM CLI doesn't natively support headless JSON output, write a wrapper:

```bash
#!/bin/bash
# .muzzle/bin/my-llm-wrapper
# Wraps a CLI to produce JSON output compatible with Muzzle

PROMPT="$1"
WORKSPACE="$(pwd)"

# Invoke the actual CLI
RESULT=$(my-llm-cli --prompt "$PROMPT" --no-interactive 2>/dev/null)

# Output as JSON
echo "{\"output\": $(echo "$RESULT" | jq -Rs .), \"status\": \"completed\"}"
```

Then reference the wrapper:

```yaml
adapters:
  custom:
    binary: .muzzle/bin/my-llm-wrapper
    mode: headless
    output_format: json
```

## Environment Variables

Each adapter can use different credentials. They're all inherited from the parent process:

```bash
# Set credentials for both adapters
export ANTHROPIC_API_KEY="sk-ant-..."    # For Claude
export OLLAMA_HOST="http://localhost:11434"  # For local Ollama

muzzle run --pipeline .muzzle/pipelines/hybrid-flow.yaml \
  --input "add feature"
```

## When to Use Multiple Adapters

| Scenario | Strategy |
|----------|----------|
| Cost optimization | Local model for analysis, cloud for implementation |
| Speed | Fast local model for navigation, thorough cloud model for review |
| Compliance | On-premise model for sensitive code, cloud for public code |
| Evaluation | Run same pipeline with different models to compare output |

## Further Reading

- [Manifest Schema — Adapter Fields](/reference/manifest-schema#adapter) — complete field reference
- [Adapters Concept](/concepts/adapters) — how adapters work
- [Environment & Credentials](/reference/environment) — credential handling
