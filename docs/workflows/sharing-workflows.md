# Sharing Wave Workflows

Wave workflows are designed to be shared, versioned, and reused across teams just like infrastructure code. This guide covers the git-based workflow distribution patterns that make AI automation as collaborative as your infrastructure deployments.

## Git-Based Workflow Collaboration

### Repository Structure

Organize workflows in your repository for maximum shareability:

```
my-project/
├── .wave/
│   ├── wave.yaml                 # Main manifest
│   ├── workflows/
│   │   ├── development/
│   │   │   ├── feature-branch.yaml
│   │   │   ├── code-review.yaml
│   │   │   └── hotfix.yaml
│   │   ├── testing/
│   │   │   ├── unit-test-gen.yaml
│   │   │   ├── integration-tests.yaml
│   │   │   └── e2e-validation.yaml
│   │   ├── deployment/
│   │   │   ├── staging-deploy.yaml
│   │   │   ├── prod-deploy.yaml
│   │   │   └── rollback.yaml
│   │   └── documentation/
│   │       ├── api-docs.yaml
│   │       ├── changelog.yaml
│   │       └── release-notes.yaml
│   ├── contracts/
│   │   ├── schemas/
│   │   │   ├── code-analysis.json
│   │   │   ├── test-results.json
│   │   │   └── security-report.json
│   │   └── validators/
│   │       ├── quality-gates.py
│   │       └── security-checks.js
│   └── personas/
│       ├── senior-navigator.md
│       ├── security-auditor.md
│       └── documentation-writer.md
├── README.md
└── docs/
    └── workflows/
        ├── README.md             # How to use our workflows
        ├── contributing.md       # How to contribute new workflows
        └── examples/
            ├── simple-review.md  # Usage examples
            └── complex-pipeline.md
```

### Team Workflow Manifest

Create a central manifest that defines your team's standard workflows:

```yaml
# .wave/wave.yaml
apiVersion: v1
kind: WaveManifest
metadata:
  name: "team-engineering-workflows"
  description: "Standard AI workflows for the engineering team"
  version: "2.1.0"
  maintainer: "engineering-leads@company.com"

workflows:
  # Development workflows
  feature-development:
    path: workflows/development/feature-branch.yaml
    description: "End-to-end feature development with testing"
    tags: [development, testing, review]

  code-review:
    path: workflows/development/code-review.yaml
    description: "Automated code review with security analysis"
    tags: [review, security, quality]

  hotfix:
    path: workflows/development/hotfix.yaml
    description: "Emergency fix workflow with accelerated review"
    tags: [hotfix, emergency, fast-track]

  # Testing workflows
  test-generation:
    path: workflows/testing/unit-test-gen.yaml
    description: "Generate comprehensive unit tests"
    tags: [testing, unit-tests, automation]

  # Documentation workflows
  api-documentation:
    path: workflows/documentation/api-docs.yaml
    description: "Generate API documentation from codebase"
    tags: [documentation, api, openapi]

personas:
  senior-navigator:
    system_prompt_file: personas/senior-navigator.md
    temperature: 0.1
    permissions:
      allowed_tools: ["Read", "Bash(git *)", "Grep", "Glob"]
      deny: ["Write(*)", "Edit(*)"]

  security-auditor:
    system_prompt_file: personas/security-auditor.md
    temperature: 0.2
    permissions:
      allowed_tools: ["Read", "Grep", "Bash(security-tools *)"]
      deny: ["Write(*)", "Edit(*)", "Bash(rm *)"]

runtime:
  workspace_root: /tmp/wave-team
  max_concurrent_workers: 4
  default_timeout: 1800

settings:
  team_standards:
    contract_validation: required
    security_review: required
    test_coverage_minimum: 80

  notifications:
    slack_channel: "#engineering-ai"
    on_failure: true
    on_completion: false
```

## Reproducibility Guarantees

Wave ensures identical workflows produce identical results across different environments and team members.

### Environmental Consistency

**Problem with Traditional AI:**
```bash
# Developer A
$ claude "review this code: $(cat feature.js)"
> Detailed review with 12 suggestions

# Developer B
$ claude "review this code: $(cat feature.js)"
> Brief review with 3 suggestions

# Why different? Chat history, prompt variations, different Claude model versions
```

**Wave Solution:**
```bash
# Developer A
$ wave run code-review --input feature.js
> Standardized review following team contracts

# Developer B
$ wave run code-review --input feature.js
> Identical standardized review following same contracts

# Same workflow + same input = guaranteed identical output
```

### Fresh Context Guarantee

Each workflow step starts with a clean slate:

```yaml
steps:
  - id: analyze-security
    persona: security-auditor
    memory:
      strategy: fresh  # No contamination from previous runs
    exec:
      source: "Perform security analysis of: {{ input }}"
    handover:
      contract:
        type: json_schema
        schema: security-findings.schema.json
        # Contract ensures output format is identical every time
```

**Reproducibility Benefits:**
- **Same team member, different times**: Identical results
- **Different team members**: Identical results
- **Different environments**: Identical results
- **CI/CD pipelines**: Reliable automation

### Contract-Enforced Consistency

Contracts guarantee not just format, but quality consistency:

```yaml
# Every team member gets same quality level
handover:
  contract:
    type: json_schema
    schema: |
      {
        "type": "object",
        "required": ["summary", "issues", "recommendations"],
        "properties": {
          "summary": {"type": "string", "minLength": 100},
          "issues": {
            "type": "array",
            "minItems": 1,
            "items": {
              "type": "object",
              "required": ["severity", "description", "location"],
              "properties": {
                "severity": {"enum": ["low", "medium", "high", "critical"]},
                "description": {"type": "string", "minLength": 50},
                "location": {"type": "string"}
              }
            }
          },
          "recommendations": {
            "type": "array",
            "minItems": 3,
            "items": {"type": "string", "minLength": 30}
          }
        }
      }
    on_failure: retry
    max_retries: 3
```

## Git Workflow Integration

### Workflow Versioning

Version workflows like infrastructure code:

```yaml
# workflows/code-review-v2.yaml
metadata:
  name: code-review
  version: "2.0.0"
  changelog:
    - "2.0.0: Added TypeScript-specific analysis"
    - "1.2.0: Enhanced security checks"
    - "1.1.0: Improved contract validation"
    - "1.0.0: Initial release"
  compatibility:
    min_wave_version: "1.5.0"
    breaking_changes:
      - "Output format changed to include type analysis"
      - "Contract schema updated for TypeScript support"
```

### Branching Strategies

**Feature Branch Workflow:**
```bash
# Create feature branch for workflow changes
git checkout -b feature/enhanced-security-review

# Modify workflow
vim .wave/workflows/development/code-review.yaml

# Test workflow changes
wave run code-review --input sample-code.js

# Update contracts if needed
vim .wave/contracts/security-findings.schema.json

# Test with contract validation
wave validate workflow .wave/workflows/development/code-review.yaml

# Commit changes
git add .wave/workflows/development/code-review.yaml
git commit -m "enhance: Add SQL injection detection to code review

- Added database query analysis step
- Enhanced security contract with injection checks
- Updated test cases for validation
- Maintains backward compatibility"

# Open pull request
gh pr create --title "Enhanced security review workflow" \
  --body "Adds SQL injection detection to standard code review"
```

**Workflow Review Process:**
```bash
# Team reviews workflow changes in PR
# Test the new workflow against known code samples
wave run code-review --input test-fixtures/sql-injection-sample.js

# Verify contract compliance
wave test contracts .wave/contracts/security-findings.schema.json

# Merge after approval
git checkout main
git merge feature/enhanced-security-review
```

### Release Management

**Tagging Workflow Releases:**
```bash
# Create workflow release
git tag -a workflows-v2.1.0 -m "Release v2.1.0: Enhanced security workflows

- Added SQL injection detection
- Improved TypeScript analysis
- Updated security contracts
- Better error messages"

git push origin workflows-v2.1.0
```

**Release Notes Example:**
```markdown
# Workflows v2.1.0 Release Notes

## New Features
- **SQL Injection Detection**: Code review now includes database security analysis
- **TypeScript Support**: Enhanced type checking and interface analysis
- **Improved Error Messages**: More specific contract failure descriptions

## Breaking Changes
- Security findings contract now requires `injection_risks` field
- Minimum Wave version: 1.5.0

## Migration Guide
Update existing security-findings.schema.json:
```json
{
  "properties": {
    "injection_risks": {
      "type": "array",
      "items": {"type": "object"}
    }
  }
}
```

## Backward Compatibility
- All v2.0.x workflows continue to work
- Optional migration to new features
```

### Deployment Strategies

**Environment-Specific Workflows:**
```bash
# Development environment
.wave/
├── environments/
│   ├── dev/
│   │   ├── wave.yaml          # Relaxed contracts for iteration
│   │   └── workflows/
│   │       └── code-review-dev.yaml
│   ├── staging/
│   │   ├── wave.yaml          # Standard contracts
│   │   └── workflows/
│   │       └── code-review-staging.yaml
│   └── production/
│       ├── wave.yaml          # Strict contracts, full validation
│       └── workflows/
│           └── code-review-prod.yaml
```

**Environment Promotion:**
```bash
# Test in dev
wave --config .wave/environments/dev/wave.yaml run code-review

# Promote to staging
cp .wave/environments/dev/workflows/new-feature.yaml \
   .wave/environments/staging/workflows/

# Test in staging
wave --config .wave/environments/staging/wave.yaml run new-feature

# Promote to production
cp .wave/environments/staging/workflows/new-feature.yaml \
   .wave/environments/production/workflows/
```

## Team Collaboration Patterns

### Workflow Ownership

**CODEOWNERS Integration:**
```bash
# .github/CODEOWNERS
.wave/workflows/security/*     @security-team
.wave/workflows/testing/*      @qa-team
.wave/workflows/deployment/*   @platform-team
.wave/personas/                @ai-workflows-team
.wave/contracts/               @architecture-team
```

### Shared Workflow Libraries

**Organization-Wide Workflows:**
```yaml
# .wave/remote-workflows.yaml
remotes:
  company-standards:
    url: "git@github.com:company/ai-workflows-standard.git"
    version: "v1.2.0"
    workflows:
      - security-audit
      - compliance-check
      - documentation-review

  team-specific:
    url: "git@github.com:company/backend-ai-workflows.git"
    version: "main"
    workflows:
      - api-testing
      - database-migration-review
      - service-integration-check
```

**Using Remote Workflows:**
```bash
# Install organization workflows
wave install company-standards

# Use remote workflow
wave run company-standards/security-audit --input ./src

# Update to latest version
wave update company-standards
```

### Contribution Workflow

**Contributing to Shared Workflows:**
```bash
# Fork the team workflows repository
gh repo fork company/ai-workflows-standard

# Clone and create branch
git clone git@github.com:yourname/ai-workflows-standard.git
cd ai-workflows-standard
git checkout -b improve-security-detection

# Enhance workflow
vim workflows/security-audit.yaml

# Test thoroughly
wave test workflow workflows/security-audit.yaml \
  --test-cases test-fixtures/security/*

# Document changes
vim docs/security-audit.md

# Commit and push
git commit -am "improve: Enhanced SQL injection detection in security audit"
git push origin improve-security-detection

# Create pull request
gh pr create --title "Enhanced security detection" \
  --body "Adds advanced SQL injection pattern detection"
```

## Workflow Distribution Patterns

### Monorepo Pattern

Single repository with all team workflows:

```
company-ai-workflows/
├── teams/
│   ├── backend/
│   │   ├── api-review.yaml
│   │   ├── database-migration.yaml
│   │   └── service-testing.yaml
│   ├── frontend/
│   │   ├── component-review.yaml
│   │   ├── accessibility-audit.yaml
│   │   └── performance-check.yaml
│   └── mobile/
│       ├── ios-review.yaml
│       ├── android-review.yaml
│       └── cross-platform-test.yaml
├── shared/
│   ├── security-audit.yaml
│   ├── documentation-gen.yaml
│   └── compliance-check.yaml
├── contracts/
│   └── shared-schemas/
└── docs/
    ├── usage-guide.md
    └── contributing.md
```

### Multi-Repo Pattern

Separate repositories for different concerns:

```bash
# Core workflows repository
git clone company/ai-workflows-core
# Contains: security-audit, code-review, documentation

# Team-specific workflows
git clone company/ai-workflows-backend
# Contains: api-testing, database-review, service-integration

# Experimental workflows
git clone company/ai-workflows-experimental
# Contains: bleeding-edge workflows for testing
```

### Package Manager Pattern

Distribute workflows like software packages:

```yaml
# package.yaml
name: "security-workflows"
version: "1.3.0"
description: "Standard security review workflows"
author: "security-team@company.com"

dependencies:
  core-workflows: ">=2.0.0"

workflows:
  security-audit:
    entry: workflows/security-audit.yaml
    description: "Comprehensive security analysis"

  penetration-test:
    entry: workflows/pentest.yaml
    description: "Automated penetration testing"

contracts:
  - security-findings.schema.json
  - vulnerability-report.schema.json
```

**Installation and Usage:**
```bash
# Install workflow package
wave install security-workflows@1.3.0

# Use packaged workflow
wave run security-workflows/security-audit --input ./src

# Update to latest
wave update security-workflows
```

## Quality Assurance for Shared Workflows

### Automated Testing

**Workflow CI/CD Pipeline:**
```yaml
# .github/workflows/test-workflows.yml
name: Test AI Workflows

on: [push, pull_request]

jobs:
  test-workflows:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Install Wave
        run: |
          curl -sSL https://install.wave.dev | sh
          echo "$HOME/.wave/bin" >> $GITHUB_PATH

      - name: Validate Workflow Syntax
        run: |
          for workflow in .wave/workflows/**/*.yaml; do
            wave validate workflow "$workflow"
          done

      - name: Test Contract Schemas
        run: |
          for schema in .wave/contracts/*.json; do
            wave validate schema "$schema"
          done

      - name: Run Workflow Test Suite
        run: |
          wave test workflows .wave/workflows/ \
            --test-fixtures test-fixtures/ \
            --output test-results.json

      - name: Upload Test Results
        uses: actions/upload-artifact@v3
        with:
          name: workflow-test-results
          path: test-results.json
```

### Contract Compatibility Testing

**Version Compatibility Matrix:**
```yaml
# .wave/compatibility-tests.yaml
compatibility_matrix:
  workflows:
    code-review:
      versions: ["1.0.0", "1.1.0", "2.0.0"]
      test_cases:
        - input: test-fixtures/javascript-sample.js
          expected_schema: contracts/code-review-v1.schema.json
        - input: test-fixtures/typescript-sample.ts
          expected_schema: contracts/code-review-v2.schema.json

    security-audit:
      versions: ["1.0.0", "1.2.0"]
      test_cases:
        - input: test-fixtures/vulnerable-code.js
          expected_findings: test-fixtures/expected-vulnerabilities.json
```

**Run Compatibility Tests:**
```bash
# Test all workflow versions
wave test compatibility .wave/compatibility-tests.yaml

# Test specific version upgrade path
wave test migration \
  --from code-review@1.1.0 \
  --to code-review@2.0.0 \
  --test-cases test-fixtures/migration/
```

### Documentation Standards

**Workflow Documentation Template:**
```markdown
# Workflow: Code Review

## Purpose
Automated code review with security analysis and quality assessment.

## Input Requirements
- **Type**: `git_diff` or `file_path`
- **Format**: Git diff output or path to file/directory
- **Size Limits**: Maximum 10MB per file

## Output Guarantees
- **Format**: JSON conforming to `code-review.schema.json`
- **Contents**: Analysis, security findings, recommendations
- **Quality**: Minimum 5 actionable recommendations

## Usage Examples

### Basic Usage
```bash
wave run code-review --input "$(git diff HEAD~1)"
```

### With Custom Configuration
```bash
wave run code-review --input ./src/feature.js \
  --config security-level=high \
  --config language=typescript
```

## Configuration Options
- `security-level`: `standard` | `high` | `critical`
- `language`: Auto-detected or specify explicitly
- `focus-areas`: Array of areas to emphasize

## Contract Schema
```json
{
  "type": "object",
  "required": ["summary", "security_findings", "recommendations"],
  "properties": {
    "summary": {
      "type": "object",
      "properties": {
        "complexity_score": {"type": "integer", "min": 1, "max": 10},
        "risk_level": {"enum": ["low", "medium", "high", "critical"]},
        "lines_analyzed": {"type": "integer"}
      }
    }
  }
}
```

## Troubleshooting
- **Contract validation fails**: Check input format matches schema
- **Analysis incomplete**: Increase timeout for large files
- **Security findings missing**: Verify security-level configuration

## Changelog
- v2.0.0: Added TypeScript support, breaking contract changes
- v1.1.0: Enhanced security detection
- v1.0.0: Initial release
```

## Team Adoption Best Practices

### Gradual Migration Strategy

**Phase 1: Pilot Team**
```bash
# Start with one team, one workflow
team-backend/
├── pilot-workflow.yaml     # Simple code review
├── README.md              # How to use
└── test-cases/            # Known good examples
```

**Phase 2: Team Standardization**
```bash
# Expand to full team workflows
team-backend/
├── workflows/
│   ├── code-review.yaml
│   ├── api-testing.yaml
│   └── security-audit.yaml
├── shared-contracts/
└── team-guide.md
```

**Phase 3: Organization Adoption**
```bash
# Company-wide workflow library
company-workflows/
├── teams/                 # Team-specific workflows
├── shared/               # Cross-team workflows
├── standards/            # Organization contracts
└── governance/           # Usage policies
```

### Training and Onboarding

**New Team Member Workflow:**
```bash
# 1. Clone team workflows
git clone company/backend-ai-workflows
cd backend-ai-workflows

# 2. Install Wave and workflows
wave install

# 3. Run first workflow
wave run code-review --input sample-code.js
# Expected output: Detailed review with 5+ suggestions

# 4. Try interactive tutorial
wave tutorial team-workflows

# 5. Read team guide
cat docs/team-workflow-guide.md
```

**Team Workshop Template:**
```markdown
# Team AI Workflows Workshop

## Session 1: Introduction (1 hour)
- What are Wave workflows?
- Demo: Traditional prompting vs Wave workflows
- Hands-on: Run existing code-review workflow

## Session 2: Creating Workflows (2 hours)
- Workflow anatomy
- Writing your first workflow
- Contract design
- Testing and validation

## Session 3: Team Integration (1 hour)
- Git workflow integration
- Code review process
- Team standards and governance
```

### Measuring Adoption Success

**Metrics to Track:**
```yaml
# .wave/analytics.yaml
metrics:
  usage:
    - workflow_runs_per_week
    - unique_users_per_month
    - workflows_created_by_team

  quality:
    - contract_validation_success_rate
    - workflow_completion_rate
    - average_output_quality_score

  productivity:
    - time_saved_vs_manual_review
    - code_review_cycle_time
    - deployment_frequency_improvement

  adoption:
    - teams_using_workflows
    - workflows_shared_across_teams
    - community_contributions
```

**Monthly Review Template:**
```bash
# Generate adoption report
wave analytics report \
  --period "last-month" \
  --teams backend,frontend,mobile \
  --output adoption-report.json

# Key questions for team review:
# 1. Which workflows are most/least used?
# 2. What contract failures are common?
# 3. How can we improve workflow quality?
# 4. What new workflows would be valuable?
```

Wave workflows become more valuable when shared. Treat them as infrastructure code - version them, test them, review them, and evolve them collaboratively. The reproducibility guarantees ensure that what works for one team member works for everyone.

## Next Steps

- [Community Library](/workflows/community-library) - Discover public workflows
- [Creating Workflows](/workflows/creating-workflows) - Build your own workflows
- [Examples](/workflows/examples/) - Complete workflow specimens
- [Team Adoption](/migration/team-adoption) - Organizational implementation patterns