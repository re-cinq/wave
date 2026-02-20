# AI as Code

Wave brings Infrastructure-as-Code principles to AI development. Like Terraform for infrastructure or Docker Compose for containers, Wave lets you define AI workflows as declarative configuration files that are version-controlled, reproducible, and shareable.

## The Problem with Traditional AI Development

When developers use AI assistants today, they face fundamental challenges:

- **Ephemeral sessions**: Conversations disappear after use
- **Copy-paste workflows**: Prompts shared via Slack or scattered docs
- **Inconsistent outputs**: Same task produces different results each time
- **No quality gates**: Outputs accepted on faith without validation
- **Individual knowledge**: What works for one developer stays with them

This mirrors where infrastructure was before Infrastructure-as-Code: manual, undocumented, and unreproducible.

## Wave's Solution

Wave treats AI workflows like infrastructure code. You declare what you want in YAML files, commit them to git, and Wave handles execution with built-in quality guarantees.

### Your First Pipeline

```yaml
kind: WavePipeline
metadata:
  name: code-review
  description: "Automated code review with security analysis"

input:
  source: cli

steps:
  - id: analyze
    persona: navigator
    memory:
      strategy: fresh
    workspace:
      mount:
        - source: ./
          target: /src
          mode: readonly
    exec:
      type: prompt
      source: |
        Analyze the code changes: {{ input }}

        Identify:
        1. Files changed and their purposes
        2. Potential security concerns
        3. Test coverage gaps
    output_artifacts:
      - name: analysis
        path: .wave/output/analysis.json
        type: json

  - id: review
    persona: auditor
    dependencies: [analyze]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: analyze
          artifact: analysis
          as: context
    exec:
      type: prompt
      source: |
        Based on the analysis, generate a code review.

        Context: {{ artifacts.context }}
    output_artifacts:
      - name: review
        path: .wave/output/review.md
        type: markdown
```

Run it:

```bash
wave run code-review "Review changes in src/auth"
```

### Key Concepts

**Pipelines** define multi-step AI workflows. Each step uses a specific persona (an AI agent with defined capabilities and constraints) and can depend on outputs from previous steps.

**Personas** are configured in your `wave.yaml` manifest with specific permissions and system prompts:

```yaml
# wave.yaml
apiVersion: v1
kind: WaveManifest
metadata:
  name: my-project

personas:
  navigator:
    adapter: claude
    description: "Read-only codebase analysis"
    system_prompt_file: .wave/personas/navigator.md
    permissions:
      allowed_tools: [Read, Glob, Grep]
      deny: [Write, Edit, Bash]

  auditor:
    adapter: claude
    description: "Security and quality review"
    system_prompt_file: .wave/personas/auditor.md
    permissions:
      allowed_tools: [Read, Grep]
      deny: [Write, Edit]
```

**Fresh memory** at step boundaries ensures reproducibility. Each step starts with a clean context, receiving only explicitly declared artifacts from previous steps.

**Contracts** validate outputs before handover to the next step:

```yaml
handover:
  contract:
    type: jsonschema
    schema_path: .wave/contracts/analysis.schema.json
    on_failure: retry
    max_retries: 2
```

## Infrastructure-as-Code Parallels

If you've used Terraform, Kubernetes, or Docker Compose, Wave's model will feel familiar:

| IaC Pattern | Wave Equivalent |
|-------------|-----------------|
| Terraform resources | Pipeline steps |
| Container images | Personas |
| Health checks | Contract validation |
| Volume mounts | Artifact injection |
| State management | SQLite execution tracking |

Like infrastructure code, Wave configurations are:

- **Declarative**: Describe what, not how
- **Version-controlled**: Track changes in git
- **Reproducible**: Same config produces same results
- **Reviewable**: Pull requests for workflow changes

## Core Execution Model

Wave executes pipelines with these guarantees:

1. **Dependency resolution**: Steps run in correct order based on declared dependencies
2. **Workspace isolation**: Each step gets a fresh, ephemeral workspace
3. **Fresh context**: No conversation history inherited between steps
4. **Artifact flow**: Outputs explicitly passed via `inject_artifacts`
5. **Contract validation**: Outputs validated before proceeding
6. **State persistence**: Execution state stored in SQLite for resumption

```bash
# Run a pipeline
wave run code-review "Review authentication changes"

# Check status
wave status

# View logs
wave logs

# Resume interrupted pipeline
wave resume <run-id>
```

## Benefits

**For Individual Developers**
- Capture working prompts as reusable pipelines
- Consistent results across sessions
- Quality guarantees through contracts

**For Teams**
- Share workflows via git
- Review AI workflow changes in PRs
- Standardize on proven patterns

**For Organizations**
- Audit trails for AI usage
- Permission controls via personas
- Observable, measurable AI workflows

## Next Steps

- [Infrastructure Parallels](/paradigm/infrastructure-parallels) - Detailed comparisons with Docker, Kubernetes, and Terraform
- [Contracts](/paradigm/deliverables-contracts) - How Wave guarantees output quality
- [Pipeline Execution](/concepts/pipeline-execution) - Execution model in depth
- [Creating Workflows](/workflows/creating-workflows) - Build your first pipeline
