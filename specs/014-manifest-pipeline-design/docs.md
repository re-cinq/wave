# VitePress Documentation Specification

**Branch**: `014-manifest-pipeline-design`
**Date**: 2026-02-01
**Status**: Draft
**Purpose**: Define the complete VitePress documentation site for Muzzle

## Overview

Muzzle needs comprehensive, documentation-driven development with a VitePress site that serves as both developer documentation and user guide. The documentation should be authoritative - the spec defines exactly what should exist, and implementation should match 1:1.

## Documentation Structure

```
docs/
├── .vitepress/
│   ├── config.ts              # Site configuration
│   ├── theme/
│   │   ├── index.ts          # Theme customization
│   │   └── components/       # Custom Vue components
│   └── sidebar/             # Auto-generated sidebar config
├── guide/
│   ├── index.md             # Getting started overview
│   ├── installation.md      # Installation instructions
│   ├── quick-start.md       # 5-minute first run
│   ├── configuration.md     # muzzle.yaml manifest reference
│   ├── personas.md          # Persona system explained
│   ├── pipelines.md         # Pipeline DAG concepts
│   ├── contracts.md         # Handover contracts
│   └── relay.md            # Context compaction
├── reference/
│   ├── cli.md              # Command reference (init, validate, run, etc.)
│   ├── manifest-schema.md   # Complete manifest YAML schema
│   ├── pipeline-schema.md   # Complete pipeline YAML schema
│   ├── adapters.md         # Adapter configurations (Claude, OpenCode, etc.)
│   └── troubleshooting.md   # Common issues and solutions
├── tutorials/
│   ├── first-project.md    # Tutorial: Your first Muzzle project
│   ├── custom-personas.md   # Tutorial: Creating custom personas
│   ├── pipeline-design.md   # Tutorial: Designing effective pipelines
│   ├── meta-pipelines.md    # Tutorial: Self-designing pipelines
│   └── ci-integration.md   # Tutorial: GitHub Actions setup
├── examples/
│   ├── index.md            # Examples overview
│   ├── simple-feature.md    # Example: Adding a simple feature
│   ├── bug-fix.md         # Example: Debugging and fixing a bug
│   ├── refactoring.md      # Example: Code refactoring workflow
│   └── multi-persona.md   # Example: Complex multi-persona workflow
├── concepts/
│   ├── architecture.md     # System architecture overview
│   ├── isolation.md        # Workspace isolation model
│   ├── state-management.md # Pipeline state persistence
│   ├── security.md         # Permission model and credential handling
│   └── performance.md      # Performance considerations
└── development/
    ├── contributing.md     # Contributing guidelines
    ├── architecture-decisions.md # ADRs
    ├── building.md         # Building from source
    └── release-process.md  # Release process

Static assets:
├── public/
│   ├── logo.svg           # Muzzle logo
│   ├── og-image.png       # Social sharing image
│   └── favicon.ico
└── assets/
    ├── diagrams/          # Architecture diagrams
    │   ├── manifest-flow.svg
    │   ├── pipeline-execution.svg
    │   ├── persona-binding.svg
    │   └── relay-flow.svg
    └── images/           # Screenshots and illustrations
        ├── cli-output.png
        ├── manifest-example.png
        └── pipeline-progress.png
```

## Detailed Page Specifications

### 1. Landing Page (`docs/index.md`)

```markdown
---
layout: home
hero:
  name: Muzzle
  text: Multi-Agent Orchestrator for Claude Code
  tagline: Wrap Claude Code and other LLM CLIs with personas, pipelines, and contracts
  image:
    src: /logo.svg
    alt: Muzzle
  actions:
    - theme: brand
      text: Get Started
      link: /guide/quick-start
    - theme: alt
      text: View on GitHub
      link: https://github.com/recinq/muzzle
features:
  - icon: 🛡️
    title: Persona-Scoped Safety
    details: Each agent runs with explicit permissions, hooks, and tool restrictions
  - icon: 🔄
    title: Pipeline DAGs
    details: Compose multi-step workflows with handover contracts between agents
  - icon: 🧠
    title: Context Relay
    details: Automatic context compaction preserves continuity across long tasks
  - icon: ⚡
    title: Ad-Hoc Execution
    details: Quick single commands with full safety model, no pipeline needed
---
```

### 2. Configuration Guide (`docs/guide/configuration.md`)

Complete manifest reference with:
- Full YAML schema documentation
- Each field with type, required status, default, example
- Cross-references between related fields
- Best practices for common patterns
- Validation error explanations
- Migration guide for different versions

### 3. CLI Reference (`docs/reference/cli.md`)

Auto-generated from Cobra CLI:
```markdown
# muzzle

Muzzle CLI - Multi-agent orchestrator

## Commands

### init
Initialize a new Muzzle project

```bash
muzzle init [flags]
```

**Flags**:
- `--adapter string`   Default adapter to use (default "claude")
- `--persona string`   Initial persona to create (default "craftsman")

**Examples**:
```bash
# Initialize with defaults
muzzle init

# Initialize for specific use case
muzzle init --adapter claude --persona fullstack
```

### validate
Validate Muzzle configuration

```bash
muzzle validate [flags]
```

**Flags**:
- `--manifest string`   Path to manifest (default "muzzle.yaml")
- `--verbose`          Show detailed validation errors

**Exit Codes**:
- 0: Validation passed
- 1: Validation failed
- 2: Manifest not found
```

### 4. Manifest Schema (`docs/reference/manifest-schema.md`)

```yaml
# Generated from Go structs
title: Muzzle Manifest Schema
$schema: http://json-schema.org/draft-07/schema#

type: object
required: [apiVersion, kind, metadata, adapters, personas, runtime]
properties:
  apiVersion:
    type: string
    enum: [v1]
    description: Schema version
  kind:
    type: string
    enum: [MuzzleManifest]
    description: Document type
  metadata:
    $ref: "#/$defs/Metadata"
  adapters:
    type: object
    patternProperties:
      "^[a-z][a-z0-9-]*$":
        $ref: "#/$defs/Adapter"
    description: Named adapter configurations

definitions:
  Metadata:
    type: object
    required: [name]
    properties:
      name:
        type: string
        description: Project name
      description:
        type: string
        description: Project description
      repo:
        type: string
        format: uri
        description: Repository URL
```

### 5. Pipeline Schema (`docs/reference/pipeline-schema.md`)

Complete pipeline YAML schema with:
- Step configuration options
- Dependency patterns
- Contract types and schemas
- Matrix strategy configuration
- Memory strategy options

### 6. Tutorial: First Project (`docs/tutorials/first-project.md`)

Step-by-step walkthrough:
1. Install Muzzle
2. Initialize a new project
3. Examine generated files
4. Run first pipeline
5. Check results
6. Modify a persona
7. Create custom pipeline

Include:
- Expected outputs at each step
- Common pitfalls and how to avoid them
- Screenshots of terminal output
- Link to example repository

### 7. Tutorial: Custom Personas (`docs/tutorials/custom-personas.md`)

Create different personas:
- Code Reviewer (strict, focuses on security)
- Documentation Writer (focuses on clarity)
- Performance Optimizer (focuses on benchmarks)
- QA Engineer (focuses on test coverage)

For each persona:
- System prompt template
- Permission recommendations
- Hook examples
- Use case scenarios

### 8. Examples Directory (`docs/examples/`)

Each example includes:
- Problem statement
- Muzzle configuration
- Step-by-step execution
- Expected results
- Variations and extensions

### 9. Architecture Concepts (`docs/concepts/`)

Explain core concepts:
- How Muzzle wraps Claude Code subprocess
- Workspace isolation mechanism
- Pipeline state machine
- Token monitoring and relay triggers
- Permission enforcement model

Include Mermaid diagrams for visual clarity.

### 10. Troubleshooting (`docs/reference/troubleshooting.md`)

Common issues with solutions:
- Adapter not found on PATH
- Permission denied errors
- Pipeline hangs on step
- Relay not triggering
- Workspace cleanup issues
- Performance tuning

## VitePress Configuration

### Theme Customization

Custom Vue components:
- `<MuzzleConfig>` - Interactive manifest editor
- `<PipelineVisualizer>` - DAG visualization
- `<TerminalOutput>` - Styled terminal output
- `<PersonaCard>` - Persona overview card

### Plugins

- `@vuepress/plugin-search` with custom search index
- Custom plugin for schema validation examples
- Mermaid plugin for diagrams
- Copy code button plugin
- Auto-linking internal references

### Build Configuration

```typescript
// .vitepress/config.ts
export default {
  title: 'Muzzle',
  description: 'Multi-Agent Orchestrator for Claude Code',
  
  themeConfig: {
    nav: [
      { text: 'Guide', link: '/guide/' },
      { text: 'Reference', link: '/reference/' },
      { text: 'Tutorials', link: '/tutorials/' },
      { text: 'Examples', link: '/examples/' },
    ],
    
    sidebar: {
      '/guide/': [
        {
          text: 'Getting Started',
          items: [
            { text: 'Installation', link: '/guide/installation' },
            { text: 'Quick Start', link: '/guide/quick-start' },
          ]
        },
        // ... auto-generated
      ],
      // ... other sections
    },
    
    socialLinks: [
      { icon: 'github', link: 'https://github.com/recinq/muzzle' }
    ]
  },
  
  markdown: {
    config: (md) => {
      md.use(highlightLines)
      md.use(preWrapper)
    }
  }
}
```

## Documentation Quality Standards

### Writing Guidelines

1. **Always show, don't just tell**: Include code examples for every concept
2. **Provide copy-paste ready examples**: All code blocks should be complete and runnable
3. **Explain the "why"**: Not just what, but why the design decision was made
4. **Include common patterns**: Show idiomatic ways to solve problems
5. **Cross-reference extensively**: Link between related concepts

### Code Example Standards

```yaml
# Always include the file path
# File: muzzle.yaml

apiVersion: v1
kind: MuzzleManifest
metadata:
  name: my-project
  description: "Example project configuration"

adapters:
  claude:
    binary: claude
    mode: headless
    output_format: json
```

### Diagram Standards

Use Mermaid for all diagrams:
- Flowcharts for processes
- Sequence diagrams for interactions
- Graph diagrams for relationships
- Class diagrams for data models

### Versioning

- Document matches released version
- Include version selector in UI
- Mark features with version requirements
- Migration guides for breaking changes

## Automation

### Doc Generation

1. **CLI Reference**: Auto-generate from Cobra commands
2. **Schema Documentation**: Auto-generate from Go struct tags
3. **API Docs**: Auto-generate from interface definitions
4. **Examples**: Validate against current schema

### CI/CD Integration

```yaml
# .github/workflows/docs.yml
name: Documentation

on:
  push:
    branches: [main]
    paths: ['docs/**', 'cmd/**/*.go']

jobs:
  docs:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: 20
      - run: npm ci
      - run: npm run build
      - run: npm run validate-links
      - name: Deploy to GitHub Pages
        uses: peaceiris/actions-gh-pages@v3
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          publish_dir: docs/.vitepress/dist
```

### Pre-commit Hooks

- Spell check all markdown files
- Validate all YAML/JSON examples
- Check all internal links resolve
- Ensure code examples are up-to-date

## Success Metrics

1. **Completeness**: Every CLI command, config option, and feature documented
2. **Accuracy**: All examples validated against current implementation
3. **Usability**: Users can accomplish tasks with only the documentation
4. **Discoverability**: Related content easily found through links and search
5. **Maintainability**: Documentation updates automated where possible

## Implementation Plan

The documentation should be implemented alongside the code:

1. **Phase 1**: Basic structure and Getting Started guide (P1 features)
2. **Phase 2**: Complete reference documentation for core features
3. **Phase 3**: Tutorials and examples
4. **Phase 4**: Advanced concepts and integration guides
5. **Phase 5**: Automation and CI/CD

Each phase's documentation must be complete and accurate before the corresponding code feature is considered done.