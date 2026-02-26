# Infrastructure Parallels

Wave's architecture directly maps to proven Infrastructure-as-Code patterns. If you understand Docker Compose, Kubernetes, or Terraform, Wave's concepts will feel immediately familiar.

## Docker Compose and Wave

Docker Compose orchestrates container services. Wave orchestrates AI workflow steps.

### Service Definition

**Docker Compose:**
```yaml
services:
  web:
    build: .
    depends_on: [database]
    environment:
      DATABASE_URL: ${DB_URL}
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
```

**Wave Pipeline:**
```yaml
kind: WavePipeline
metadata:
  name: gh-pr-review

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
      source: "Analyze the codebase structure for {{ input }}"
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
    handover:
      contract:
        type: jsonschema
        schema_path: .wave/contracts/review.schema.json
```

### Concept Mapping

| Docker Compose | Wave | Purpose |
|---------------|------|---------|
| `services` | `steps` | Independent units of work |
| `depends_on` | `dependencies` | Execution ordering |
| `environment` | `inject_artifacts` | Data injection |
| `healthcheck` | `handover.contract` | Quality validation |
| `volumes` | `workspace.mount` | File system access |
| `image` | `persona` | Pre-configured execution unit |

### Lifecycle Commands

| Docker Compose | Wave | Purpose |
|----------------|------|---------|
| `docker-compose up` | `wave run <pipeline>` | Start execution |
| `docker-compose ps` | `wave status` | View running state |
| `docker-compose logs` | `wave logs` | View output logs |
| `docker-compose down` | `wave clean` | Cleanup resources |

## Kubernetes and Wave

Kubernetes orchestrates containerized applications with declarative state management. Wave applies similar patterns to AI workflows.

### Deployment Configuration

**Kubernetes Deployment:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: web-app
  template:
    spec:
      containers:
      - name: app
        image: myapp:v1.2.3
        resources:
          requests:
            memory: "64Mi"
          limits:
            memory: "128Mi"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          failureThreshold: 3
```

**Wave Pipeline Step:**
```yaml
kind: WavePipeline
metadata:
  name: feature-development

steps:
  - id: implement
    persona: craftsman
    memory:
      strategy: fresh
    workspace:
      mount:
        - source: ./src
          target: /workspace
          mode: readwrite
    exec:
      type: prompt
      source: "Implement the feature: {{ input }}"
    output_artifacts:
      - name: implementation
        path: .wave/output/code.patch
        type: patch
    handover:
      contract:
        type: testsuite
        command: "npm test"
        must_pass: true
      on_review_fail: retry
      max_retries: 3
```

### Concept Mapping

| Kubernetes | Wave | Purpose |
|-----------|------|---------|
| `kind: Deployment` | `kind: WavePipeline` | Resource type declaration |
| `metadata.name` | `metadata.name` | Resource identification |
| Container image | Persona | Execution environment |
| Resource limits | Persona permissions | Execution constraints |
| `livenessProbe` | `handover.contract` | Health/quality validation |
| `failureThreshold` | `max_retries` | Retry policy |
| ConfigMaps/Secrets | Artifacts | Data management |
| `replicas` | `strategy.max_concurrency` | Parallel execution |

### Resource Management

Like Kubernetes manages container resources, Wave manages AI execution:

- **Personas** define what tools an AI agent can use (similar to container security contexts)
- **Workspaces** provide isolated execution environments (similar to pods)
- **Artifacts** flow between steps (similar to ConfigMaps/Secrets)
- **Contracts** validate outputs (similar to liveness/readiness probes)

## Terraform and Wave

Terraform manages infrastructure state with plan/apply cycles. Wave manages pipeline state with similar patterns.

### State Management

**Terraform:**
```hcl
resource "aws_instance" "web" {
  ami           = "ami-0c02fb55956c7d316"
  instance_type = "t3.micro"

  tags = {
    Name = "WebServer"
  }
}

# Terraform tracks:
# - Current state vs desired state
# - Resource dependencies
# - Change planning
```

**Wave:**
```yaml
kind: WavePipeline
metadata:
  name: documentation

steps:
  - id: extract-api
    persona: navigator
    output_artifacts:
      - name: api-spec
        path: .wave/output/api.json
        type: json

  - id: generate-docs
    persona: documenter
    dependencies: [extract-api]
    memory:
      inject_artifacts:
        - step: extract-api
          artifact: api-spec
          as: spec
    output_artifacts:
      - name: documentation
        path: .wave/output/docs.md
        type: markdown

# Wave tracks:
# - Step execution state
# - Artifact availability
# - Pipeline resumption points
```

### Concept Mapping

| Terraform | Wave | Purpose |
|-----------|------|---------|
| Resources | Steps | Managed entities |
| State file | SQLite database | Execution tracking |
| `terraform plan` | `wave validate` | Validate configuration |
| `terraform apply` | `wave run` | Execute changes |
| `terraform destroy` | `wave clean` | Cleanup |
| State refresh | `wave status` | Check current state |
| Dependencies | `dependencies` | Ordering and relationships |

### Lifecycle Comparison

| Operation | Terraform | Wave |
|-----------|-----------|------|
| Validate config | `terraform validate` | `wave validate` |
| Preview changes | `terraform plan` | Configuration validation |
| Apply changes | `terraform apply` | `wave run` |
| Check state | `terraform show` | `wave status` |
| Resume | `terraform apply` (continues) | `wave resume` |
| Cleanup | `terraform destroy` | `wave clean` |

## Common Patterns

### Dependency Graphs

All three tools build execution graphs from dependencies:

```yaml
# Wave dependency resolution
steps:
  - id: analyze        # No dependencies - runs first

  - id: backend        # Depends on analyze
    dependencies: [analyze]

  - id: frontend       # Depends on analyze (parallel with backend)
    dependencies: [analyze]

  - id: integrate      # Waits for both
    dependencies: [backend, frontend]
```

Wave resolves this to:
```
analyze → backend  ↘
                    → integrate
analyze → frontend ↗
```

### Idempotency and Reproducibility

Like Terraform's `terraform plan` shows what would change, Wave's execution model provides:

- **Deterministic artifact flow**: Same inputs produce same outputs
- **Fresh memory at boundaries**: No hidden state between steps
- **Contract validation**: Outputs verified before use

### Configuration as Documentation

In all three paradigms, configuration files serve as living documentation:

```yaml
# This Wave pipeline IS the documentation for how code reviews work
kind: WavePipeline
metadata:
  name: gh-pr-review
  description: "Security-focused code review with quality gates"

steps:
  - id: security-scan
    persona: auditor
    # The configuration explains what happens at each step
```

## Why These Patterns Work

Infrastructure-as-Code succeeded because it solved fundamental problems:

| Problem | IaC Solution | Wave Implementation |
|---------|-------------|---------------------|
| Manual, error-prone processes | Declarative configuration | YAML pipeline definitions |
| Inconsistent environments | Reproducible execution | Fresh memory, isolated workspaces |
| Undocumented changes | Version-controlled configs | Git-tracked `.wave/` directory |
| No quality gates | Automated validation | Contract validation at handovers |
| Difficult collaboration | Shared configuration | Teams share pipeline definitions |

Wave applies these battle-tested patterns to AI workflows, inheriting decades of operational wisdom from the infrastructure community.

## Getting Started with IaC Background

If you're already comfortable with Infrastructure-as-Code:

1. **Think of personas as container images** - Pre-configured execution environments with specific capabilities
2. **Think of pipelines as compose files** - Multi-step workflows with dependencies
3. **Think of contracts as health checks** - Quality gates ensuring valid outputs
4. **Think of artifacts as volumes** - Data flow between execution units
5. **Think of workspaces as containers** - Isolated, ephemeral environments

Your IaC knowledge transfers directly to Wave.

## Next Steps

- [AI as Code](/paradigm/ai-as-code) - The foundational paradigm
- [Contracts](/paradigm/deliverables-contracts) - Quality guarantees in AI workflows
- [Pipeline Execution](/concepts/pipeline-execution) - Execution model details
