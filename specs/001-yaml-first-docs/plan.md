# Implementation Plan: YAML-First Documentation Paradigm

**Branch**: `001-yaml-first-docs` | **Date**: 2026-02-03 | **Spec**: [spec.md](spec.md)
**Design Reference**: [ngrok documentation](https://ngrok.com/docs)

## Summary

Restructure Wave documentation following ngrok's enterprise-grade patterns:
- **Task-oriented navigation** instead of feature-oriented
- **60-second quickstart** as primary entry point
- **1-2 sentences + code block** pattern for all concept pages
- **Progressive complexity** from simple → contracts → multi-step → team → enterprise
- **Escape routes** at every potential blocker

## Technical Context

**Documentation Format**: Markdown with VitePress/similar static site generator
**Primary Dependencies**: None (static documentation site)
**Storage**: File-based markdown + assets
**Testing**: Documentation validation via user testing + example validation
**Target Platform**: Static site hosting (Netlify, Vercel, GitHub Pages)
**Performance Goals**: Page load < 2s, first pipeline success in < 60s

## Constitution Check

**No Constitutional Violations**: Documentation restructure aligns with existing principles:
- YAML-first approach reinforces Principle 2 (Manifest as Single Source of Truth)
- Contract emphasis supports Principle 6 (Contracts at Every Handover)
- Task-oriented docs highlight observable progress (Principle 10)

## Project Structure

### Current Documentation (to be restructured)

```
docs/
├── index.md                     # Landing (needs rewrite)
├── paradigm/                    # TO BE REMOVED - integrate into index
│   ├── ai-as-code.md
│   ├── infrastructure-parallels.md
│   └── deliverables-contracts.md
├── workflows/                   # RENAME to use-cases/
│   ├── creating-workflows.md
│   ├── sharing-workflows.md
│   └── community-library.md
├── concepts/                    # REWRITE with 1-2 sentences + code
│   └── pipeline-execution.md
├── reference/                   # ENHANCE with command+output pairs
└── migration/                   # RENAME to guides/
    ├── from-personas-to-workflows.md
    ├── team-adoption.md
    └── enterprise-patterns.md
```

### Target Documentation (ngrok-inspired)

```
docs/
├── index.md                    # "What is Wave" - ONE paragraph, ONE diagram, quickstart CTA
├── quickstart.md               # 60-second first pipeline (NEW - CRITICAL)
│
├── use-cases/                  # Task-oriented (PRIMARY navigation)
│   ├── index.md               # Card-based overview
│   ├── code-review.md         # Complete runnable example
│   ├── security-audit.md      # Complete runnable example
│   ├── docs-generation.md     # Complete runnable example
│   └── test-generation.md     # Complete runnable example
│
├── concepts/                   # 1-2 sentences + code (SECONDARY)
│   ├── index.md               # Overview
│   ├── pipelines.md           # Core concept
│   ├── personas.md            # Supporting concept
│   ├── contracts.md           # Progressive examples
│   ├── artifacts.md           # Output handling
│   └── execution.md           # How Wave runs
│
├── reference/                  # Command + output pairs (TERTIARY)
│   ├── cli.md                 # All commands
│   ├── manifest.md            # wave.yaml reference
│   ├── pipeline-schema.md     # Pipeline YAML schema
│   └── contract-types.md      # Contract type reference
│
└── guides/                     # Advanced patterns (ADOPTION)
    ├── ci-cd.md               # CI/CD integration
    ├── team-adoption.md       # Team rollout
    └── enterprise.md          # Enterprise patterns
```

## Content Patterns (from ngrok analysis)

### Pattern 1: Concept Pages

```markdown
# Concept Name

One sentence explaining what it is.

​```yaml
# Minimal working example (5-10 lines max)
​```

One sentence explaining when to use it.

## Next Steps

- [Related concept](/concepts/related) - brief description
- [Use case](/use-cases/example) - brief description
```

### Pattern 2: Use-Case Pages

```markdown
# Task Name (e.g., "Automate Code Review")

One paragraph: problem → solution → outcome.

## Quick Start

​```bash
wave run code-review "Review my PR"
​```

Expected output shown.

## Complete Pipeline

​```yaml
# Full runnable example
​```

## Customization

Progressive examples adding complexity.

## Next Steps

- Related use cases
- Relevant concepts
```

### Pattern 3: Quickstart Page

```markdown
# Quickstart

Get your first pipeline running in 60 seconds.

## 1. Install Wave

​```bash
# Installation command
​```

> **Don't have X?** [Escape route link]

## 2. Initialize

​```bash
wave init
​```

## 3. Run Your First Pipeline

​```bash
wave run code-review "Hello Wave"
​```

## What Just Happened?

Brief explanation of what executed.

## Next Steps

- [Code Review](/use-cases/code-review) - deeper dive
- [Create Custom Pipeline](/concepts/pipelines) - build your own
- [Add Contracts](/concepts/contracts) - guarantee outputs
```

## Migration Strategy

### Phase 1: Core Infrastructure
1. Create quickstart.md (CRITICAL - enables 60-second success)
2. Rewrite index.md (one paragraph + diagram + CTA)
3. Create use-cases/ directory structure

### Phase 2: Use-Case Documentation
1. Create use-cases/index.md with card layout
2. Create code-review.md with complete example
3. Create security-audit.md, docs-generation.md, test-generation.md

### Phase 3: Concept Rewrite
1. Rewrite each concept page to 1-2 sentences + code pattern
2. Add "Next Steps" to every page
3. Ensure progressive complexity

### Phase 4: Reference Enhancement
1. Add command + output pairs to CLI reference
2. Create pipeline-schema.md with clear required/optional
3. Create contract-types.md reference

### Phase 5: Guides Consolidation
1. Rename migration/ to guides/
2. Consolidate content into ci-cd.md, team-adoption.md, enterprise.md
3. Remove paradigm/ section (integrate key points into index.md)

### Phase 6: Cleanup
1. Remove deprecated files
2. Update navigation config
3. Validate all examples are runnable
4. Test 60-second quickstart flow

## File Mapping

| Current File | Action | Target File |
|-------------|--------|-------------|
| index.md | Rewrite | index.md |
| - | Create | quickstart.md |
| paradigm/ai-as-code.md | Delete | (integrate into index.md) |
| paradigm/infrastructure-parallels.md | Delete | (integrate into index.md) |
| paradigm/deliverables-contracts.md | Delete | (integrate into concepts/contracts.md) |
| workflows/creating-workflows.md | Rewrite | concepts/pipelines.md |
| workflows/sharing-workflows.md | Rewrite | guides/team-adoption.md |
| workflows/community-library.md | Delete | (patterns in use-cases/) |
| concepts/pipeline-execution.md | Rewrite | concepts/execution.md |
| - | Create | use-cases/index.md |
| - | Create | use-cases/code-review.md |
| - | Create | use-cases/security-audit.md |
| - | Create | use-cases/docs-generation.md |
| - | Create | use-cases/test-generation.md |
| - | Create | concepts/index.md |
| - | Create | concepts/artifacts.md |
| - | Create | reference/pipeline-schema.md |
| - | Create | reference/contract-types.md |
| migration/from-personas-to-workflows.md | Delete | (concepts explain this) |
| migration/team-adoption.md | Rewrite | guides/team-adoption.md |
| migration/enterprise-patterns.md | Rewrite | guides/enterprise.md |
| - | Create | guides/ci-cd.md |

## Key Metrics

- **60 seconds**: Time to first successful pipeline run
- **5 seconds**: Time to find relevant use-case documentation
- **10 lines**: Maximum YAML in first example on any page
- **100%**: Pages with "Next Steps" section
- **100%**: Code examples that are copy-paste runnable

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Quickstart fails due to missing Claude CLI | Add escape route with installation link |
| API key not configured | Add escape route with setup link |
| No codebase to analyze | Provide sample repository or self-analysis |
| Examples become outdated | Validation script to test all YAML examples |
