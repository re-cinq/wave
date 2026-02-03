# Wave Community Library

The Wave community has created a rich ecosystem of reusable workflows for common development tasks. This guide helps you discover, use, and contribute to the growing library of AI automation patterns.

## Ecosystem Overview

### Official Collections

**Wave Core Workflows** - `wave-core`
- Maintained by Wave team
- Battle-tested, production-ready
- Comprehensive documentation
- Semantic versioning
- Long-term support

**Wave Extensions** - `wave-extensions`
- Community-maintained workflows
- Experimental and specialized use cases
- Faster innovation cycle
- Contributor-supported

### Community Collections

**Language-Specific Libraries**
- `wave-javascript` - Frontend and Node.js workflows
- `wave-python` - Data science and web development
- `wave-go` - System and API development
- `wave-rust` - Performance and systems programming
- `wave-java` - Enterprise and Android development

**Domain-Specific Libraries**
- `wave-security` - Security analysis and auditing
- `wave-testing` - Test generation and validation
- `wave-docs` - Documentation automation
- `wave-devops` - Deployment and infrastructure
- `wave-data` - Data analysis and processing

## Discovering Workflows

### Browse the Registry

```bash
# List all available workflow collections
wave registry list

# Search for specific workflows
wave registry search "code review"
wave registry search --tag security
wave registry search --language python

# Get details about a workflow
wave registry info security/owasp-audit

# See workflow ratings and reviews
wave registry reviews security/owasp-audit
```

### Workflow Marketplace

**Featured Workflows:**

| Workflow | Collection | Description | Downloads |
|----------|------------|-------------|-----------|
| `comprehensive-code-review` | wave-core | Multi-language code review with security | 50K+ |
| `api-documentation-generator` | wave-core | OpenAPI docs from codebase | 35K+ |
| `security-vulnerability-scan` | wave-security | OWASP Top 10 security analysis | 28K+ |
| `test-suite-generator` | wave-testing | Automated test creation | 22K+ |
| `refactoring-assistant` | wave-core | Safe code refactoring with validation | 18K+ |
| `documentation-updater` | wave-docs | Sync code changes to documentation | 15K+ |

**Trending This Week:**
- `typescript-migration-helper` (wave-javascript)
- `kubernetes-config-audit` (wave-devops)
- `accessibility-checker` (wave-extensions)

### Discovery Commands

```bash
# Browse by category
wave browse --category security
wave browse --category testing
wave browse --category documentation

# See what's popular
wave trending --period week
wave trending --language javascript

# Find workflows similar to one you're using
wave similar code-review

# Get recommendations based on your project
wave recommend --project-type web-api
wave recommend --language python --framework django
```

## Installing and Using Community Workflows

### Installation

```bash
# Install a specific workflow
wave install security/owasp-audit

# Install entire collection
wave install wave-security

# Install specific version
wave install security/owasp-audit@1.2.0

# Install from git repository
wave install git+https://github.com/company/custom-workflows.git

# Install with dependencies
wave install testing/e2e-suite --with-deps
```

### Usage Examples

**Security Audit Workflow:**
```bash
# Basic security scan
wave run security/owasp-audit --input ./src

# Custom configuration
wave run security/owasp-audit \
  --input ./src \
  --config security-level=high \
  --config report-format=sarif

# Integration with CI/CD
wave run security/owasp-audit \
  --input ./src \
  --output security-report.json \
  --fail-on critical
```

**API Documentation Generator:**
```bash
# Generate OpenAPI docs
wave run docs/api-generator --input ./src/routes

# With custom templates
wave run docs/api-generator \
  --input ./src/routes \
  --config template=company-standard \
  --config output-format=redoc

# Include examples
wave run docs/api-generator \
  --input ./src/routes \
  --config include-examples=true \
  --config test-cases=./test/integration/
```

**Test Suite Generator:**
```bash
# Generate unit tests
wave run testing/unit-test-gen --input ./src/utils

# Generate integration tests
wave run testing/integration-test-gen \
  --input ./src/api \
  --config framework=jest \
  --config coverage-target=90

# Generate property-based tests
wave run testing/property-test-gen \
  --input ./src/core \
  --config test-library=fast-check
```

### Configuration Management

**Global Configuration:**
```yaml
# ~/.wave/config.yaml
registry:
  default_source: wave-registry
  trusted_sources:
    - wave-core
    - wave-extensions
    - company-internal

preferences:
  auto_update: false
  security_scan: true
  telemetry: opt-in

collections:
  security:
    default_security_level: high
    require_approval: true

  testing:
    default_framework: jest
    coverage_threshold: 80
```

**Project Configuration:**
```yaml
# .wave/wave.yaml
dependencies:
  security/owasp-audit: "^1.2.0"
  testing/unit-test-gen: "1.0.5"
  docs/api-generator: "latest"

settings:
  auto_install: true
  update_policy: minor

overrides:
  security/owasp-audit:
    config:
      security-level: critical
      compliance-standards: [sox, pci-dss]
```

## Quality Standards and Trust

### Workflow Certification

**Wave Certified** workflows meet strict standards:
- ✅ Comprehensive test coverage (>90%)
- ✅ Security review completed
- ✅ Documentation quality score >8/10
- ✅ Backwards compatibility guaranteed
- ✅ Performance benchmarked
- ✅ Community feedback score >4.5/5

**Community Verified** workflows are:
- ✅ Peer-reviewed code
- ✅ Basic test coverage (>70%)
- ✅ Documentation present
- ✅ No security vulnerabilities detected

### Security and Trust

**Security Scanning:**
```bash
# Scan workflow before installation
wave security-scan security/owasp-audit

# Review workflow permissions
wave permissions security/owasp-audit

# Check workflow source code
wave inspect security/owasp-audit

# Verify cryptographic signatures
wave verify security/owasp-audit
```

**Trust Levels:**
- **Official**: Maintained by Wave team
- **Verified**: Community-verified, security-scanned
- **Community**: User-contributed, use with caution
- **Experimental**: Bleeding-edge, development only

### Workflow Reviews

**Rating System:**
```bash
# View workflow reviews
wave reviews security/owasp-audit

# Add your review
wave review security/owasp-audit \
  --rating 5 \
  --comment "Excellent security coverage, found 3 critical issues"

# Report issues
wave report security/owasp-audit \
  --issue "Contract validation fails with TypeScript"
  --severity medium
```

**Review Criteria:**
- **Accuracy**: Does it produce correct results?
- **Completeness**: Does it handle edge cases?
- **Performance**: Is execution time reasonable?
- **Documentation**: Is usage clear?
- **Maintenance**: Is it actively updated?

## Contributing to the Community

### Creating Your First Workflow

**Contribution Checklist:**
- [ ] Solves a real problem
- [ ] Has comprehensive contracts
- [ ] Includes test cases
- [ ] Documented with examples
- [ ] Follows naming conventions
- [ ] Security-scanned
- [ ] Performance benchmarked

**Workflow Template:**
```yaml
# workflows/my-new-workflow.yaml
apiVersion: v1
kind: WavePipeline
metadata:
  name: my-new-workflow
  description: "Brief description of what this accomplishes"
  version: "1.0.0"
  author: "your-name <your-email>"
  license: "MIT"
  tags: [tag1, tag2, tag3]

  # Community metadata
  category: "security"  # security, testing, docs, devops, etc.
  difficulty: "beginner"  # beginner, intermediate, advanced
  estimated_runtime: "2-5 minutes"
  supported_languages: [javascript, typescript, python]

documentation:
  readme: README.md
  examples: examples/
  changelog: CHANGELOG.md

# ... workflow definition
```

### Submission Process

**1. Prepare Your Workflow:**
```bash
# Create new workflow repository
git init my-wave-workflow
cd my-wave-workflow

# Create workflow structure
mkdir workflows contracts docs examples tests
touch workflows/my-workflow.yaml
touch README.md CHANGELOG.md LICENSE
```

**2. Development and Testing:**
```bash
# Test your workflow thoroughly
wave test workflow workflows/my-workflow.yaml \
  --test-cases tests/ \
  --validate-contracts

# Security scan
wave security-scan workflows/my-workflow.yaml

# Performance benchmark
wave benchmark workflows/my-workflow.yaml \
  --test-inputs tests/performance/
```

**3. Documentation:**
```markdown
# README.md Template

# My Workflow

Brief description of what this workflow accomplishes.

## Purpose
Detailed explanation of the problem this solves.

## Usage

### Basic Usage
```bash
wave run my-workflow --input ./src
```

### Advanced Usage
```bash
wave run my-workflow \
  --input ./src \
  --config option1=value1 \
  --config option2=value2
```

## Input Requirements
- **Type**: file_path | git_diff | json
- **Format**: Description of expected input format
- **Size Limits**: Any size restrictions

## Output Guarantees
- **Format**: JSON conforming to schema
- **Contents**: Detailed description of outputs
- **Quality**: Quality metrics or standards

## Configuration Options
| Option | Type | Default | Description |
|--------|------|---------|-------------|
| option1 | string | "default" | What this option does |

## Examples
Link to examples/ directory with real usage scenarios.

## Changelog
See CHANGELOG.md for version history.

## Contributing
How others can contribute improvements.

## License
MIT License - see LICENSE file.
```

**4. Submit to Community:**
```bash
# Submit to Wave registry
wave submit my-wave-workflow

# Or submit via GitHub
gh repo create wave-community/my-workflow --public
git remote add origin git@github.com:wave-community/my-workflow.git
git push -u origin main

# Create pull request to community index
gh pr create --repo wave-community/workflow-index \
  --title "Add my-workflow" \
  --body "New workflow for [purpose]"
```

### Contribution Guidelines

**Code Quality Standards:**
```yaml
# .wave/quality-standards.yaml
code_quality:
  test_coverage: ">= 85%"
  documentation_score: ">= 8/10"
  security_scan: "must_pass"
  performance_benchmark: "required"

contract_standards:
  validation: "strict"
  error_handling: "comprehensive"
  retry_logic: "required_for_external_deps"

documentation:
  readme: "required"
  examples: "minimum_3"
  api_docs: "complete"
  changelog: "semantic_versioning"
```

**Community Guidelines:**
- Use descriptive, action-oriented names
- Include comprehensive error handling
- Provide multiple usage examples
- Write clear documentation
- Respond to user issues promptly
- Follow semantic versioning
- Maintain backwards compatibility

### Maintenance and Updates

**Maintaining Your Workflow:**
```bash
# Update to new Wave version
wave update-workflow my-workflow --wave-version 2.0.0

# Respond to user feedback
wave issues my-workflow --status open
wave reply-issue 123 "Thanks for the feedback, fixed in v1.1.0"

# Release new version
git tag v1.1.0
git push --tags
wave publish my-workflow v1.1.0
```

**Community Support:**
```bash
# Monitor usage statistics
wave stats my-workflow --period month

# View download trends
wave analytics my-workflow --downloads --period quarter

# Check user satisfaction
wave reviews my-workflow --summary
```

## Ecosystem Governance

### Community Standards

**Workflow Naming Conventions:**
- Use descriptive, action-oriented names
- Include primary purpose: `security-audit`, `test-generator`
- Avoid version numbers in names
- Use kebab-case: `api-documentation-generator`

**Collection Organization:**
```
wave-security/
├── workflows/
│   ├── vulnerability-scan.yaml
│   ├── penetration-test.yaml
│   ├── compliance-audit.yaml
│   └── security-code-review.yaml
├── contracts/
│   ├── security-findings.schema.json
│   └── vulnerability-report.schema.json
├── docs/
│   ├── README.md
│   ├── security-guide.md
│   └── examples/
└── tests/
    ├── integration/
    └── fixtures/
```

### Quality Assurance

**Automated Testing:**
```yaml
# .github/workflows/quality-check.yml
name: Community Workflow QA

on: [push, pull_request]

jobs:
  quality-check:
    runs-on: ubuntu-latest
    steps:
      - name: Security Scan
        run: wave security-scan workflows/

      - name: Validate Contracts
        run: wave validate contracts contracts/

      - name: Test Workflows
        run: wave test workflows workflows/ --test-fixtures tests/

      - name: Documentation Check
        run: wave lint-docs docs/

      - name: Performance Benchmark
        run: wave benchmark workflows/ --compare-baseline
```

**Community Review Process:**
1. **Automated checks** - Security, syntax, performance
2. **Peer review** - Code quality, documentation
3. **Testing period** - Community feedback (1 week)
4. **Approval** - Maintainer approval for inclusion
5. **Publication** - Available in registry

### Conflict Resolution

**Duplicate Workflows:**
- Community vote on preferred implementation
- Merge workflows if possible
- Deprecate less popular version with migration guide

**Quality Issues:**
- Issue tracking on GitHub
- Community discussion
- Maintainer intervention if needed
- Removal from registry as last resort

**License Compliance:**
- All workflows must have clear license
- MIT, Apache 2.0, or BSD preferred
- Commercial licenses clearly marked
- No viral licenses (GPL) for core collection

## Advanced Usage Patterns

### Workflow Composition

**Combining Community Workflows:**
```yaml
# my-complete-pipeline.yaml
metadata:
  name: complete-development-pipeline
  dependencies:
    - security/owasp-audit@1.2.0
    - testing/unit-test-gen@2.1.0
    - docs/api-generator@1.0.5

steps:
  - id: security-check
    workflow: security/owasp-audit
    input: "{{ pipeline.input.codebase }}"

  - id: generate-tests
    workflow: testing/unit-test-gen
    dependencies: [security-check]
    input: "{{ pipeline.input.codebase }}"

  - id: generate-docs
    workflow: docs/api-generator
    dependencies: [security-check]
    input: "{{ pipeline.input.codebase }}"

  - id: final-report
    persona: reviewer
    dependencies: [security-check, generate-tests, generate-docs]
    # Synthesize results from all community workflows
```

### Custom Workflow Collections

**Building Organization Collections:**
```yaml
# my-company-workflows/collection.yaml
metadata:
  name: "acme-engineering-workflows"
  description: "ACME Corp standard development workflows"
  maintainer: "engineering@acme.corp"
  private: true

dependencies:
  # Use community workflows as building blocks
  security: "wave-security@2.0.0"
  testing: "wave-testing@1.5.0"

  # Add company-specific customizations
  acme-security:
    path: "./workflows/security/"
    description: "ACME-specific security checks"

  acme-compliance:
    path: "./workflows/compliance/"
    description: "SOX and PCI-DSS compliance workflows"

overrides:
  # Customize community workflows
  security/owasp-audit:
    config:
      compliance_standards: [sox, pci-dss, iso27001]
      custom_rules: ./rules/acme-security-rules.yaml
```

### Analytics and Insights

**Community Analytics:**
```bash
# Track ecosystem health
wave ecosystem-stats

# Find trending workflows
wave trending --category security --period month

# Usage analytics for your workflows
wave analytics my-workflow \
  --downloads \
  --user-feedback \
  --performance-metrics

# Community contribution stats
wave contributor-stats --user your-username
```

The Wave community library transforms AI automation from isolated scripts into a collaborative ecosystem. By contributing and using community workflows, teams can leverage collective intelligence and build on proven patterns.

## Getting Started

1. **Explore**: Browse the registry to see what's available
2. **Install**: Try a few workflows that match your needs
3. **Customize**: Modify workflows for your specific requirements
4. **Contribute**: Share your improvements back to the community
5. **Maintain**: Keep your workflows updated and respond to feedback

The community grows stronger when everyone contributes. Start small, think big, and help build the future of AI-powered development workflows.

## Next Steps

- [Creating Workflows](/workflows/creating-workflows) - Build your first workflow
- [Sharing Workflows](/workflows/sharing-workflows) - Team collaboration patterns
- [Examples](/workflows/examples/) - Complete workflow specimens
- [Wave Registry](https://registry.wave.dev) - Browse available workflows