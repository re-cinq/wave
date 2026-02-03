# Deliverables and Contracts: Guaranteed AI Outputs

Traditional AI interactions are unpredictable - you get whatever the AI decides to give you. Wave fundamentally changes this by providing **guaranteed deliverables** through enforced contracts, bringing Infrastructure-as-Code reliability to AI workflows.

## The Reliability Problem in AI

### Traditional AI Development

**The Problem:**
```
User: "Generate API documentation in JSON format"
AI: Returns markdown instead of JSON
User: "Actually, I need JSON"
AI: Returns invalid JSON structure
User: "Please follow the exact schema"
AI: Returns valid JSON but missing required fields
```

**Why This Happens:**
- No validation of AI outputs
- No enforcement of format requirements
- No quality gates between workflow steps
- Manual checking and re-prompting

### Infrastructure Reliability for Comparison

Infrastructure teams solved similar problems decades ago:

**Before Infrastructure-as-Code:**
- Manual deployments with unknown outcomes
- "Hope it works" deployment strategies
- Inconsistent environments
- Manual validation and fixes

**After Infrastructure-as-Code:**
- Declarative configurations with guaranteed outcomes
- Automated validation and rollback
- Consistent, reproducible environments
- Built-in quality gates and health checks

## Wave's Solution: Deliverables + Contracts

Wave brings the same reliability guarantees to AI workflows through two key concepts:

### 1. Deliverables: What You Will Get

Deliverables are **explicit declarations** of what each step must produce:

```yaml
steps:
  - id: api-analysis
    persona: navigator
    task: "Analyze API endpoints and generate schema"
    deliverables:
      - name: "api-schema"
        format: "json"
        description: "Complete OpenAPI 3.0 specification"
      - name: "endpoint-list"
        format: "csv"
        description: "All endpoints with HTTP methods and descriptions"
```

**Key Benefits:**
- **Explicit expectations**: Everyone knows exactly what will be produced
- **Format guarantees**: No surprises about output format
- **Artifact tracking**: Wave manages and tracks all deliverables
- **Dependency clarity**: Downstream steps know exactly what inputs they'll receive

### 2. Contracts: How Quality is Enforced

Contracts are **automated validation gates** that ensure deliverables meet requirements:

```yaml
steps:
  - id: api-documentation
    persona: documenter
    task: "Generate comprehensive API documentation"
    deliverables:
      - name: "api-docs"
        format: "json"
    handover:
      contract:
        type: json_schema
        schema: .wave/contracts/api-docs.schema.json
        source: output/api-docs.json
        on_failure: retry
        max_retries: 3
```

**Contract Validation:**
- **Automatic**: No manual checking required
- **Enforced**: Step cannot complete until contract passes
- **Retry logic**: Failed outputs trigger automatic retry
- **Quality guaranteed**: Downstream steps receive validated inputs

## Types of Guarantees

### Format Guarantees

Ensure outputs match expected structure and syntax:

```yaml
# JSON Schema Contract
contract:
  type: json_schema
  schema: .wave/contracts/user-profile.schema.json

# Schema ensures exact structure
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["id", "name", "email", "created_at"],
  "properties": {
    "id": {"type": "string", "pattern": "^[0-9a-f-]{36}$"},
    "name": {"type": "string", "minLength": 1},
    "email": {"type": "string", "format": "email"},
    "created_at": {"type": "string", "format": "date-time"}
  }
}
```

### Compilation Guarantees

Ensure generated code compiles and is syntactically valid:

```yaml
# TypeScript Interface Contract
contract:
  type: typescript_interface
  source: output/types.ts
  validate: true

# Example generated TypeScript that MUST compile
interface UserProfile {
  id: string;
  name: string;
  email: string;
  created_at: Date;
}
```

### Functional Guarantees

Ensure outputs work correctly through test execution:

```yaml
# Test Suite Contract
contract:
  type: test_suite
  command: "npm test -- --testPathPattern=generated"
  must_pass: true

# Tests that MUST pass
describe('Generated API Client', () => {
  test('connects to API successfully', async () => {
    const client = new GeneratedAPIClient();
    expect(await client.health()).toBe('ok');
  });

  test('handles authentication correctly', async () => {
    const client = new GeneratedAPIClient();
    expect(await client.authenticate(token)).toBe(true);
  });
});
```

## Contract Design Patterns

### Progressive Validation

Start with basic structure checks, add complexity as workflow progresses:

```yaml
# Early step: Basic structure
- id: data-extraction
  handover:
    contract:
      type: json_schema
      schema: .wave/contracts/basic-data.schema.json  # Just required fields

# Middle step: Richer validation
- id: data-enrichment
  handover:
    contract:
      type: json_schema
      schema: .wave/contracts/enriched-data.schema.json  # More constraints

# Final step: Functional validation
- id: data-integration
  handover:
    contract:
      type: test_suite
      command: "python -m pytest tests/integration/"  # End-to-end tests
```

### Layered Contracts

Multiple validation levels for comprehensive quality assurance:

```yaml
- id: code-generation
  handover:
    contracts:  # Multiple contracts for the same step
      - type: json_schema
        schema: .wave/contracts/code-metadata.schema.json
        source: output/metadata.json
      - type: typescript_interface
        source: output/generated-types.ts
        validate: true
      - type: test_suite
        command: "npm run test:generated"
        must_pass: true
```

### Conditional Contracts

Different validation based on context or configuration:

```yaml
- id: deployment-config
  handover:
    contract:
      type: json_schema
      schema: |
        {% if environment == "production" %}
        .wave/contracts/production-config.schema.json
        {% else %}
        .wave/contracts/dev-config.schema.json
        {% endif %}
```

## Failure Handling and Quality Assurance

### Automatic Retry with Fresh Context

When contracts fail, Wave automatically retries with clean state:

```yaml
handover:
  contract:
    type: json_schema
    schema: .wave/contracts/api-spec.schema.json
    on_failure: retry
    max_retries: 3
    backoff: exponential
```

**Retry Process:**
1. Contract validation fails (e.g., missing required field)
2. Wave creates fresh workspace (no contamination from failed attempt)
3. Persona re-executes step with same inputs
4. Contract validation re-runs on new output
5. Process repeats until success or max_retries exceeded

### Detailed Failure Information

Wave provides specific feedback about what failed:

```json
{
  "contract_failure": {
    "type": "json_schema",
    "schema": ".wave/contracts/api-spec.schema.json",
    "validation_errors": [
      {
        "path": "$.paths./users.get.responses",
        "message": "Required property '200' is missing",
        "expected": "Response object for successful operation"
      },
      {
        "path": "$.info.version",
        "message": "Expected string, got number",
        "value": 1.0
      }
    ],
    "retry_count": 1,
    "max_retries": 3
  }
}
```

### Escalation Strategies

When automatic retry isn't enough:

```yaml
handover:
  contract:
    on_failure: retry
    max_retries: 2
    escalation:
      # After max retries, switch to different persona
      fallback_persona: senior-developer
      # Or halt for manual intervention
      manual_review: true
      # Or proceed with warnings
      allow_partial: true
```

## Infrastructure Parallels

### Health Checks ↔ Contracts

**Kubernetes Health Checks:**
```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  failureThreshold: 3
  periodSeconds: 10
```

**Wave Contracts:**
```yaml
contract:
  type: test_suite
  command: "npm run health-check"
  max_retries: 3
  backoff: exponential
```

### Rolling Deployments ↔ Contract Validation

**Infrastructure:** Gradual rollout with automatic rollback on health check failure

**Wave:** Gradual step execution with automatic retry on contract failure

### Circuit Breakers ↔ Contract Failure Policies

**Infrastructure:** Stop routing traffic to failing services

**Wave:** Stop pipeline execution on repeated contract failures

## Real-World Examples

### Documentation Generation Pipeline

```yaml
pipeline:
  name: api-documentation
  steps:
    - id: extract-endpoints
      persona: api-navigator
      task: "Extract all API endpoints from codebase"
      deliverables: ["endpoint-list"]
      handover:
        contract:
          type: json_schema
          schema: .wave/contracts/endpoints.schema.json

    - id: generate-openapi
      persona: api-documenter
      dependencies: [extract-endpoints]
      task: "Generate OpenAPI specification"
      deliverables: ["openapi-spec"]
      handover:
        contract:
          type: json_schema
          schema: .wave/contracts/openapi-3.0.schema.json

    - id: generate-client
      persona: code-generator
      dependencies: [generate-openapi]
      task: "Generate TypeScript API client"
      deliverables: ["api-client"]
      handover:
        contracts:
          - type: typescript_interface
            source: output/api-client.ts
          - type: test_suite
            command: "npm test -- --testNamePattern='API Client'"
```

**Guarantees:**
- Endpoint extraction produces valid JSON list
- OpenAPI spec complies with OpenAPI 3.0 standard
- Generated client compiles and passes all tests

### Code Review Pipeline

```yaml
pipeline:
  name: automated-code-review
  steps:
    - id: analyze-changes
      persona: code-analyst
      task: "Analyze code changes for review"
      deliverables: ["change-analysis"]
      handover:
        contract:
          type: json_schema
          schema: .wave/contracts/change-analysis.schema.json

    - id: security-review
      persona: security-reviewer
      dependencies: [analyze-changes]
      task: "Review changes for security issues"
      deliverables: ["security-report"]
      handover:
        contract:
          type: json_schema
          schema: .wave/contracts/security-report.schema.json

    - id: generate-feedback
      persona: reviewer
      dependencies: [analyze-changes, security-review]
      task: "Generate comprehensive code review"
      deliverables: ["review-comments"]
      handover:
        contract:
          type: test_suite
          command: "python validate_review.py output/review-comments.json"
```

**Guarantees:**
- Change analysis includes all required fields (files, complexity, risk)
- Security report follows standard vulnerability format
- Review comments pass validation for completeness and tone

## Business Value

### Predictable Outcomes

Traditional AI workflows:
- **Unpredictable**: "The AI might generate what we need"
- **Manual QA**: Humans must check every output
- **Rework cycles**: Failed outputs require complete restart

Wave workflows:
- **Guaranteed**: "The AI will generate exactly what we specified"
- **Automated QA**: Contracts ensure quality without human intervention
- **Automatic retry**: Failed outputs retry automatically until they pass

### Team Collaboration

Traditional AI workflows:
- **Individual**: Each developer has their own prompts and patterns
- **Inconsistent**: Different people get different quality outputs
- **Undocumented**: AI interactions lost in chat history

Wave workflows:
- **Collaborative**: Shared workflow definitions and contract standards
- **Consistent**: Same workflow produces same quality results for everyone
- **Documented**: Contracts explicitly specify quality requirements

### Enterprise Adoption

Traditional AI workflows:
- **Unpredictable quality**: Can't rely on AI outputs for production
- **Manual oversight**: Requires human validation at every step
- **Limited scalability**: Doesn't scale beyond individual productivity

Wave workflows:
- **Production-ready**: Contract validation ensures enterprise-quality outputs
- **Automated oversight**: Quality gates built into the workflow
- **Scalable adoption**: Teams can standardize on reliable AI workflows

## Getting Started with Contracts

### 1. Identify Your Quality Requirements

What format do you need?
```yaml
deliverables:
  - name: "user-data"
    format: "json"  # Not markdown, not CSV - exactly JSON
```

### 2. Create Contract Schemas

Define exactly what "good" looks like:
```json
{
  "type": "object",
  "required": ["users", "total_count"],
  "properties": {
    "users": {"type": "array"},
    "total_count": {"type": "integer", "minimum": 0}
  }
}
```

### 3. Add Contracts to Your Pipeline

```yaml
handover:
  contract:
    type: json_schema
    schema: .wave/contracts/user-data.schema.json
    on_failure: retry
    max_retries: 2
```

### 4. Test and Iterate

Run your pipeline and refine contracts based on actual AI output patterns.

Wave's deliverables and contracts transform AI from "hope it works" to "guaranteed to work" - bringing the same reliability revolution that Infrastructure-as-Code brought to system deployments.

## Next Steps

- [AI as Code](/paradigm/ai-as-code) - The foundational paradigm
- [Infrastructure Parallels](/paradigm/infrastructure-parallels) - Detailed IaC comparisons
- [Contracts](/concepts/contracts) - Complete contract implementation guide
- [Pipeline Execution](/concepts/pipeline-execution) - How contracts integrate with workflow execution