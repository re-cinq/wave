# Quickstart: Your First AI Workflow

**Purpose**: Get developers from zero to running their first YAML-defined AI workflow in under 5 minutes.
**Audience**: Infrastructure engineers familiar with IaC patterns
**Outcome**: Complete workflow YAML file that produces guaranteed deliverables

## The 2-Minute Concept

Wave brings **Infrastructure as Code** to AI workflows. Just like you define deployments in `kubernetes.yaml` or services in `docker-compose.yml`, you define AI workflows in `workflow.yaml`.

```yaml
# feature-development.yaml
apiVersion: v1
kind: Pipeline
metadata:
  name: feature-development
  version: "1.0.0"
spec:
  steps:
    - name: analyze
      persona: navigator
      deliverable: analysis.md
      contract: schemas/analysis.json
    - name: implement
      persona: craftsman
      depends_on: [analyze]
      deliverable: src/feature.go
      contract: tests/feature.test.go
    - name: verify
      persona: auditor
      depends_on: [implement]
      deliverable: security-report.json
      contract: schemas/security-review.json
```

## What This Gets You

- **Reproducible results**: Same workflow, same deliverables, every time
- **Version control**: Commit workflows to git, share with your team
- **Guaranteed outputs**: Contracts ensure you get exactly what you specify
- **Team standardization**: Everyone uses the same proven workflows

## 5-Minute Setup

### 1. Install Wave
```bash
go install github.com/recinq/wave/cmd/wave@latest
```

### 2. Initialize Your Project
```bash
cd your-project
wave init
```

This creates:
- `wave.yaml` - Your manifest (like `package.json` or `Cargo.toml`)
- `.wave/workflows/` - Where you store workflow definitions
- `.wave/personas/` - AI agent configurations
- `.wave/contracts/` - Output validation schemas

### 3. Run Your First Workflow
```bash
wave run feature-development.yaml --input "add rate limiting to the API"
```

Watch as Wave:
1. **Analyzes** your codebase (guaranteed structured output)
2. **Implements** the feature (guaranteed working code)
3. **Verifies** security (guaranteed compliance report)

### 4. Check Your Deliverables
```bash
# Analysis deliverable (validated by JSON schema)
cat /tmp/wave/[run-id]/analyze/analysis.md

# Implementation deliverable (validated by tests passing)
cat /tmp/wave/[run-id]/implement/src/feature.go

# Security deliverable (validated by schema)
cat /tmp/wave/[run-id]/verify/security-report.json
```

## What Just Happened?

1. **YAML Definition**: You declared what you wanted (not how to get it)
2. **Persona Execution**: Each step ran with appropriate AI agent permissions
3. **Contract Validation**: Every deliverable was validated before the next step
4. **Guaranteed Outputs**: You got exactly what you specified, or Wave retried until you did

## Next Steps

### Share With Your Team
```bash
# Commit workflow to git
git add feature-development.yaml
git commit -m "Add standardized feature development workflow"
git push

# Teammates can now use the same workflow
git clone your-repo
wave run feature-development.yaml --input "their feature idea"
```

### Explore Community Workflows
```bash
# Install popular community workflows
wave install security/owasp-audit.yaml
wave install testing/e2e-generator.yaml
wave install docs/api-documentation.yaml

# Use them like your own
wave run owasp-audit.yaml --input "audit my REST API"
```

### Create Custom Workflows
```bash
# Start from a template
wave new workflow --template security-review
# Edit the YAML to your needs
# Test and share with your team
```

## Why This Matters

Traditional AI tools require you to:
- Manually prompt for each step
- Hope the output is in the right format
- Manually validate quality
- Can't reproduce results
- Can't share workflows

Wave AI workflows are:
- **Declarative**: Define what you want, not how to get it
- **Reproducible**: Same input, same output, always
- **Shareable**: Version control and distribute like infrastructure code
- **Guaranteed**: Contracts ensure output quality
- **Composable**: Build complex workflows from simple components

You already know this pattern from Kubernetes, Docker, Terraform. Now apply it to AI.

## Troubleshooting

**"wave: command not found"**
```bash
# Ensure Go bin is in PATH
export PATH=$PATH:$(go env GOPATH)/bin
```

**"No such workflow"**
```bash
# Check available workflows
wave list workflows
# Create the workflow if it doesn't exist
wave new workflow feature-development
```

**"Contract validation failed"**
```bash
# Check the specific validation error
wave run --debug feature-development.yaml
# Fix the contract schema or retry
```

Ready to revolutionize your development workflow? Start with [Creating Your First Workflow](workflows/creating-workflows.md).