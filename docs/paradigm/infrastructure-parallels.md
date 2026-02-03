# Infrastructure Parallels: Wave and IaC Tools

Wave's architecture directly mirrors proven Infrastructure-as-Code patterns. If you understand Docker Compose, Kubernetes, or Terraform, you already understand Wave's mental model.

## Declarative Configuration Comparison

### Docker Compose ↔ Wave Pipelines

**Docker Compose** orchestrates containers:
```yaml
services:
  web:
    build: .
    depends_on: [database, cache]
    environment:
      - DATABASE_URL=${DB_URL}
      - CACHE_URL=${CACHE_URL}
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]

  database:
    image: postgres:13
    environment:
      - POSTGRES_DB=app
    volumes:
      - db_data:/var/lib/postgresql/data

  cache:
    image: redis:alpine
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
```

**Wave** orchestrates AI steps:
```yaml
pipeline:
  name: feature-development
  steps:
    - id: analyze
      persona: navigator
      task: "Analyze codebase structure"
      artifacts: ["analysis-report"]

    - id: implement
      persona: craftsman
      dependencies: [analyze]
      inputs: ["analysis-report"]
      task: "Implement feature based on analysis"
      handover:
        contract:
          type: json_schema
          schema: .wave/contracts/implementation.schema.json

    - id: test
      persona: tester
      dependencies: [implement]
      inputs: ["implementation"]
      task: "Generate comprehensive tests"
      handover:
        contract:
          type: test_suite
          command: "npm test"
```

**Key Parallels:**
- **Services ↔ Steps**: Individual units of work
- **depends_on ↔ dependencies**: Execution ordering
- **environment ↔ inputs**: Data injection
- **healthcheck ↔ contract**: Validation gates

### Kubernetes ↔ Wave Execution

**Kubernetes Deployment**:
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
    metadata:
      labels:
        app: web-app
    spec:
      containers:
      - name: app
        image: myapp:v1.2.3
        resources:
          requests:
            memory: "64Mi"
            cpu: "250m"
          limits:
            memory: "128Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
```

**Wave Step Configuration**:
```yaml
steps:
  - id: code-review
    persona: reviewer
    resources:
      memory_limit: "2GB"
      timeout: "10m"
    task: "Review implementation for quality"
    handover:
      contract:
        type: test_suite
        command: "npm run lint && npm test"
        on_failure: retry
        max_retries: 2
    permissions:
      allow:
        - "file_read:**/*.js"
        - "file_write:output/**"
      deny:
        - "network_access:*"
```

**Key Parallels:**
- **Containers ↔ Personas**: Execution units with specific capabilities
- **Resource limits ↔ Persona permissions**: Constrained execution
- **Probes ↔ Contracts**: Health/quality validation
- **Labels ↔ Artifacts**: Metadata and identification

### Terraform ↔ Wave State Management

**Terraform State**:
```hcl
# main.tf
resource "aws_instance" "web" {
  ami           = "ami-0c02fb55956c7d316"
  instance_type = "t3.micro"

  tags = {
    Name = "WebServer"
  }
}

resource "aws_s3_bucket" "data" {
  bucket = "my-app-data-bucket"
}

# Terraform tracks:
# - Resource dependencies
# - Current state vs desired state
# - Change planning and application
```

**Wave State**:
```yaml
# wave.yaml
pipeline:
  name: documentation-generation
  steps:
    - id: analyze-api
      persona: navigator
      artifacts: ["api-schema"]

    - id: generate-docs
      persona: documenter
      dependencies: [analyze-api]
      inputs: ["api-schema"]
      artifacts: ["documentation"]

# Wave tracks:
# - Step dependencies
# - Execution state vs desired state
# - Pipeline resumption and rollback
```

**Key Parallels:**
- **Resources ↔ Steps**: Managed entities with state
- **State file ↔ SQLite database**: Persistent execution tracking
- **Plan/Apply ↔ Run/Resume**: State transition management
- **Dependencies ↔ Dependencies**: Ordering and relationships

## Operational Patterns

### Configuration Management

| Pattern | Docker/K8s | Terraform | Wave |
|---------|------------|-----------|------|
| **Environments** | Multiple compose files | Workspaces/vars | Multiple wave.yaml files |
| **Secrets** | Environment variables | Variable files | Environment injection |
| **Validation** | Schema validation | `terraform validate` | `wave validate` |
| **Dry Run** | `docker-compose config` | `terraform plan` | `wave plan` |

### Lifecycle Management

| Operation | Docker Compose | Terraform | Wave |
|-----------|----------------|-----------|------|
| **Deploy** | `docker-compose up` | `terraform apply` | `wave run` |
| **Status** | `docker-compose ps` | `terraform show` | `wave status` |
| **Logs** | `docker-compose logs` | Provider logs | `wave logs` |
| **Cleanup** | `docker-compose down` | `terraform destroy` | `wave cleanup` |
| **Resume** | `docker-compose restart` | `terraform apply` | `wave resume` |

### Debugging and Troubleshooting

| Issue Type | Docker/K8s Approach | Wave Approach |
|------------|---------------------|---------------|
| **Container failures** | `kubectl describe pod`, `docker logs` | `wave trace pipeline-id`, `wave logs step-id` |
| **Network issues** | `kubectl port-forward`, `docker network ls` | Workspace inspection, artifact flow validation |
| **Resource constraints** | `kubectl top`, resource monitoring | Persona permission analysis, timeout debugging |
| **Configuration errors** | YAML validation, schema checking | `wave validate`, contract schema validation |

## Advanced Patterns

### Blue-Green Deployments ↔ Pipeline Versioning

**Kubernetes Blue-Green**:
```yaml
# Blue deployment (current)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app-blue
  labels:
    version: blue

# Green deployment (new)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app-green
  labels:
    version: green

# Switch traffic via service selector
```

**Wave Pipeline Versioning**:
```yaml
# Production pipeline (current)
# .wave/pipelines/feature-dev-v1.yaml
pipeline:
  name: feature-development
  version: v1
  steps: [...]

# Experimental pipeline (new)
# .wave/pipelines/feature-dev-v2.yaml
pipeline:
  name: feature-development
  version: v2
  steps: [...]

# Switch via configuration reference
```

### Rolling Updates ↔ Incremental Pipeline Migration

**Kubernetes Rolling Update**:
```yaml
spec:
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
```

**Wave Incremental Migration**:
```yaml
# Gradually migrate steps to new personas
steps:
  - id: analyze
    persona: navigator-v2  # Upgraded

  - id: implement
    persona: craftsman-v1  # Still old version

  - id: test
    persona: tester-v2     # Upgraded
```

### Circuit Breakers ↔ Contract Failure Handling

**Microservices Circuit Breaker**:
```yaml
resilience4j:
  circuitbreaker:
    failure-rate-threshold: 50
    wait-duration-in-open-state: 60s
    sliding-window-size: 10
```

**Wave Contract Circuit Breaker**:
```yaml
handover:
  contract:
    type: json_schema
    on_failure: retry
    max_retries: 3
    backoff_strategy: exponential
    failure_threshold: 0.5  # Halt pipeline if 50% of recent attempts fail
```

## Migration Strategies

### From Manual to Infrastructure-as-Code

**Traditional Infrastructure Migration**:
1. Document current manual processes
2. Create IaC configurations for existing resources
3. Test configurations in staging
4. Gradually migrate production workloads
5. Decommission manual processes

**AI Workflow Migration**:
1. Document current manual AI interactions
2. Create Wave configurations for existing workflows
3. Test pipelines with same inputs/outputs
4. Gradually migrate team workflows
5. Decommission copy-paste AI processes

### Team Adoption Patterns

**Infrastructure Teams**:
- Start with simple, single-environment configurations
- Add complexity gradually (multi-environment, advanced features)
- Establish conventions and standards
- Build reusable modules/templates

**AI Workflow Teams**:
- Start with simple, single-step pipelines
- Add complexity gradually (multi-step, contracts, parallelism)
- Establish persona and pipeline conventions
- Build reusable workflow libraries

## Tool Ecosystem Parallels

### Validation and Testing

| Category | Infrastructure | AI Workflows |
|----------|----------------|--------------|
| **Syntax** | YAML/JSON linters | `wave validate` |
| **Security** | Container scanning, RBAC | Persona permissions, credential scrubbing |
| **Testing** | Infrastructure tests | Contract validation, test suite execution |
| **Policy** | OPA, Falco | Contract schemas, permission policies |

### Monitoring and Observability

| Category | Infrastructure | AI Workflows |
|----------|----------------|--------------|
| **Metrics** | Prometheus, CloudWatch | Pipeline execution metrics |
| **Logs** | ELK stack, Fluentd | Structured audit logs |
| **Tracing** | Jaeger, Zipkin | Execution traces, artifact lineage |
| **Alerts** | AlertManager, PagerDuty | Contract failures, step timeouts |

### Development Workflow

| Stage | Infrastructure | AI Workflows |
|-------|----------------|--------------|
| **Local** | docker-compose, minikube | `wave run` with local personas |
| **CI/CD** | GitHub Actions, Jenkins | Pipeline execution in CI |
| **Staging** | Pre-prod environments | Test pipeline configurations |
| **Production** | Production deployment | Production AI workflow execution |

## Why These Patterns Work

Infrastructure-as-Code succeeded because it addressed fundamental problems:

1. **Reproducibility**: Same configuration produces same results
2. **Version Control**: Track changes over time
3. **Collaboration**: Teams can share and modify configurations
4. **Automation**: Eliminate manual, error-prone processes
5. **Scaling**: Patterns that work for one resource work for thousands

Wave applies these same solutions to AI workflows, inheriting decades of proven operational wisdom from the infrastructure community.

## Getting Started with IaC Background

If you're already comfortable with Infrastructure-as-Code tools:

1. **Think of personas as container images** - pre-configured execution environments
2. **Think of pipelines as docker-compose files** - orchestrated multi-step workflows
3. **Think of contracts as health checks** - validation gates ensuring quality
4. **Think of artifacts as mounted volumes** - data flow between steps
5. **Think of workspaces as ephemeral containers** - isolated execution environments

Your existing IaC knowledge directly transfers to Wave - you're not learning a new paradigm, you're applying a familiar one to AI development.

## Next Steps

- [AI as Code](/paradigm/ai-as-code) - The foundational paradigm explanation
- [Deliverables and Contracts](/paradigm/deliverables-contracts) - Quality guarantees in AI workflows
- [Pipeline Execution](/concepts/pipeline-execution) - How Wave orchestrates your configurations