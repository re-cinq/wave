---
title: Developer Onboarding
description: Generate onboarding materials and codebase exploration guides for new team members
---

# Developer Onboarding

<div class="use-case-meta">
  <span class="complexity-badge beginner">Beginner</span>
  <span class="category-badge">Onboarding</span>
</div>

Generate comprehensive onboarding materials for new team members. This pipeline analyzes your codebase and produces architecture guides, getting started documentation, and learning paths.

## Prerequisites

- Wave installed and initialized (`wave init`)
- Codebase with some existing structure (README, comments, etc.)
- Basic understanding of YAML configuration

## Quick Start

```bash
wave run onboarding "create onboarding guide for new backend developers"
```

Expected output:

```
[10:00:01] started   explore         (navigator)              Starting step
[10:00:38] completed explore         (navigator)   37s   4.2k Exploration complete
[10:00:39] started   document        (philosopher)            Starting step
[10:01:25] completed document        (philosopher)  46s   6.8k Documentation complete

Pipeline onboarding completed in 83s
Artifacts: output/onboarding-guide.md
```

## Complete Pipeline

Save the following YAML to `.wave/pipelines/onboarding.yaml`:

<div v-pre>

```yaml
kind: WavePipeline
metadata:
  name: onboarding
  description: "Generate developer onboarding materials"

input:
  source: cli

steps:
  - id: explore
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
        Explore the codebase for onboarding: {{ input }}

        Discover:
        1. Project structure and organization
        2. Main entry points and core modules
        3. Key abstractions and patterns used
        4. External dependencies and their purposes
        5. Configuration and environment setup
        6. Build, test, and deployment processes
        7. Existing documentation and READMEs

        Output as JSON:
        {
          "project_type": "",
          "language": "",
          "framework": "",
          "structure": {},
          "entry_points": [],
          "core_modules": [],
          "patterns": [],
          "dependencies": [],
          "build_commands": {},
          "existing_docs": []
        }
    output_artifacts:
      - name: exploration
        path: output/codebase-exploration.json
        type: json

  - id: document
    persona: philosopher
    dependencies: [explore]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: explore
          artifact: exploration
          as: codebase
    exec:
      type: prompt
      source: |
        Create comprehensive onboarding documentation for: {{ input }}

        Include:
        1. Welcome and project overview
        2. Development environment setup
        3. Project architecture explanation
        4. Key concepts and terminology
        5. Common workflows (develop, test, deploy)
        6. Important files and where to find things
        7. Coding conventions and standards
        8. Recommended learning path
        9. FAQ and troubleshooting
        10. Who to ask for help

        Write for a developer who is new to this codebase but
        experienced in the technology stack.
    output_artifacts:
      - name: guide
        path: output/onboarding-guide.md
        type: markdown
```

</div>

## Expected Outputs

The pipeline produces two artifacts:

| Artifact | Path | Description |
|----------|------|-------------|
| `exploration` | `output/codebase-exploration.json` | Structured codebase analysis |
| `guide` | `output/onboarding-guide.md` | Complete onboarding guide |

### Example Output

The pipeline produces `output/onboarding-guide.md`:

```markdown
# Wave Developer Onboarding Guide

Welcome to the Wave project! This guide will help you get productive quickly.

## Project Overview

Wave is a multi-agent pipeline orchestrator written in Go. It coordinates
AI personas to execute complex development workflows like code review,
testing, and documentation generation.

## Getting Started

### Prerequisites

- Go 1.25 or later
- Git
- Claude API key (for AI features)

### Setup

1. Clone the repository:
   ` ` `bash
   git clone https://github.com/recinq/wave.git
   cd wave
   ` ` `

2. Install dependencies:
   ` ` `bash
   go mod download
   ` ` `

3. Configure environment:
   ` ` `bash
   export ANTHROPIC_API_KEY="your-key-here"
   ` ` `

4. Build and test:
   ` ` `bash
   go build ./cmd/wave
   go test ./...
   ` ` `

## Project Architecture

` ` `
wave/
├── cmd/wave/           # CLI entry point
├── internal/           # Core packages (not exported)
│   ├── adapter/        # Subprocess execution
│   ├── contract/       # Output validation
│   ├── manifest/       # Configuration loading
│   ├── pipeline/       # Pipeline execution
│   ├── persona/        # AI persona management
│   ├── state/          # SQLite persistence
│   └── workspace/      # Isolated execution environments
├── .wave/              # Default configurations
│   ├── personas/       # Persona definitions
│   └── pipelines/      # Built-in pipelines
└── docs/               # Documentation
` ` `

## Key Concepts

### Personas
AI agents with specific roles and permissions. Each persona has a system
prompt that defines its capabilities and constraints.

**Built-in personas:**
- `navigator` - Analyzes codebases and plans tasks
- `craftsman` - Implements code changes
- `auditor` - Reviews and validates
- `philosopher` - Designs and documents
- `summarizer` - Synthesizes information

### Pipelines
Multi-step workflows where each step is executed by a persona. Steps can
run in parallel and pass artifacts to downstream steps.

### Contracts
Validation rules that ensure step outputs meet requirements (JSON schema,
test suites, TypeScript compilation).

## Common Workflows

### Running a Pipeline
` ` `bash
wave run code-review "review the authentication module"
` ` `

### Creating a Custom Pipeline
` ` `bash
wave do "refactor the error handling" --save .wave/pipelines/refactor.yaml
` ` `

### Checking Pipeline Status
` ` `bash
wave status
` ` `

## Important Files

| File | Purpose |
|------|---------|
| `wave.yaml` | Project manifest and configuration |
| `cmd/wave/main.go` | CLI entry point |
| `internal/pipeline/executor.go` | Core execution engine |
| `internal/manifest/loader.go` | Configuration parsing |

## Coding Conventions

- Follow standard Go conventions (`gofmt`, `go vet`)
- Use table-driven tests
- Document exported functions with godoc
- Keep packages focused (single responsibility)
- Use interfaces for testability

## Learning Path

### Week 1: Basics
1. Read this guide and run basic pipelines
2. Explore the CLI commands
3. Try the built-in pipelines

### Week 2: Understanding
1. Read the concepts documentation
2. Study the pipeline executor code
3. Create a simple custom pipeline

### Week 3: Contributing
1. Pick a "good first issue"
2. Follow the contribution guide
3. Submit your first PR

## FAQ

**Q: How do I debug a failing pipeline?**
A: Use `wave run --debug` for detailed logging, or check `.wave/traces/`.

**Q: Where are pipeline outputs stored?**
A: In the `output/` directory by default.

**Q: How do I add a new persona?**
A: Create a YAML file in `.wave/personas/` following the existing format.

## Getting Help

- **Slack**: #wave-dev channel
- **Documentation**: docs.wave.dev
- **Issues**: github.com/recinq/wave/issues
- **Team Lead**: @alice (architecture), @bob (pipelines)

Welcome to the team!
```

## Customization

### Role-specific onboarding

```bash
wave run onboarding "create onboarding guide for frontend developers"
```

```bash
wave run onboarding "create onboarding guide for QA engineers"
```

### Project-specific focus

```bash
wave run onboarding "onboarding guide focused on the API layer"
```

### Include architecture diagrams

Add a diagram generation step:

<div v-pre>

```yaml
- id: diagrams
  persona: philosopher
  dependencies: [explore]
  exec:
    source: |
      Create ASCII architecture diagrams showing:
      1. High-level system architecture
      2. Data flow between components
      3. Deployment topology
```

</div>

### Add interactive exercises

<div v-pre>

```yaml
- id: exercises
  persona: philosopher
  dependencies: [document]
  exec:
    source: |
      Create hands-on exercises for new developers:

      1. "Hello World" task (modify and run)
      2. Add a simple feature
      3. Write a test
      4. Debug a (seeded) bug
      5. Review a sample PR
```

</div>

## Extended Pipeline

For comprehensive onboarding, create an extended version:

<div v-pre>

```yaml
kind: WavePipeline
metadata:
  name: onboarding-full
  description: "Comprehensive onboarding with exercises"

steps:
  - id: explore
    # ... same as above

  - id: architecture
    persona: philosopher
    dependencies: [explore]
    exec:
      source: |
        Create detailed architecture documentation:
        - Component diagrams
        - Sequence diagrams for key flows
        - Data model documentation
    output_artifacts:
      - name: architecture
        path: output/architecture.md
        type: markdown

  - id: setup-guide
    persona: navigator
    dependencies: [explore]
    exec:
      source: |
        Create detailed environment setup guide:
        - Step-by-step instructions
        - Common issues and solutions
        - IDE configuration tips
    output_artifacts:
      - name: setup
        path: output/setup-guide.md
        type: markdown

  - id: exercises
    persona: philosopher
    dependencies: [explore]
    exec:
      source: |
        Create progressive learning exercises:
        - Day 1: Build and run
        - Week 1: First contribution
        - Month 1: Feature development
    output_artifacts:
      - name: exercises
        path: output/exercises.md
        type: markdown

  - id: compile
    persona: summarizer
    dependencies: [architecture, setup-guide, exercises]
    exec:
      source: |
        Compile all materials into a complete onboarding package
        with table of contents and cross-references.
    output_artifacts:
      - name: package
        path: output/onboarding-package.md
        type: markdown
```

</div>

## Related Use Cases

- [Documentation Generation](/use-cases/documentation-generation) - Generate API docs
- [Code Review](/use-cases/code-review) - Learn review standards

## Next Steps

- [Concepts: Personas](/concepts/personas) - Understanding navigator and philosopher
- [Concepts: Pipelines](/concepts/pipelines) - Build custom onboarding flows

<style>
.use-case-meta {
  display: flex;
  gap: 8px;
  margin-bottom: 24px;
}
.complexity-badge {
  padding: 4px 12px;
  font-size: 12px;
  font-weight: 600;
  border-radius: 12px;
  text-transform: uppercase;
}
.complexity-badge.beginner {
  background: #dcfce7;
  color: #166534;
}
.complexity-badge.intermediate {
  background: #fef3c7;
  color: #92400e;
}
.complexity-badge.advanced {
  background: #fee2e2;
  color: #991b1b;
}
.category-badge {
  padding: 4px 12px;
  font-size: 12px;
  font-weight: 500;
  border-radius: 12px;
  background: var(--vp-c-brand-soft);
  color: var(--vp-c-brand-1);
}
</style>
