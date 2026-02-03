# Implementation Plan: YAML-First Documentation Paradigm

**Branch**: `001-yaml-first-docs` | **Date**: 2026-02-03 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-yaml-first-docs/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Refresh Wave documentation to position it as "Infrastructure as Code for AI" rather than an AI persona tool. Primary requirement is to lead with the YAML-first paradigm where developers can create reproducible, shareable, version-controlled AI workflows. Technical approach focuses on restructuring content hierarchy, creating Infrastructure-as-Code parallels, and emphasizing deliverables + contracts as guaranteed output mechanism.

## Technical Context

**Documentation Format**: Markdown with VitePress/similar static site generator
**Primary Dependencies**: None (static documentation site)
**Storage**: File-based markdown + assets (no database)
**Testing**: Documentation validation via manual review + user testing
**Target Platform**: Static site hosting (Netlify, Vercel, GitHub Pages)
**Project Type**: Documentation restructure and content refresh
**Performance Goals**: Page load < 2s, content discoverability within 30s of landing
**Constraints**: Must maintain existing technical accuracy while shifting paradigm presentation
**Scale/Scope**: ~40 existing docs files, new content structure for 3 primary user journeys

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

### Relevant Principles Assessment

**✅ Principle 2: Manifest as Single Source of Truth**
- Documentation MUST emphasize `wave.yaml` as the primary interface
- YAML-first approach aligns perfectly with this principle

**✅ Principle 4: Fresh Memory at Every Step Boundary**
- Documentation MUST explain this architecture clearly as a key differentiator
- Essential for Infrastructure-as-Code positioning

**✅ Principle 6: Contracts at Every Handover**
- Documentation MUST prominently feature contracts as output guarantees
- Core to the deliverables + contracts value proposition

**✅ Principle 10: Observable Progress, Auditable Operations**
- Documentation should present structured events as monitoring/GitOps integration

**No Constitutional Violations**: This is a documentation refresh that better aligns with existing architectural principles rather than changing them.

### Post-Design Constitution Re-Check ✅

After Phase 1 design completion:

**✅ All principles maintained**: Documentation reorganization supports constitutional architecture
- YAML-first approach reinforces Principle 2 (Manifest as Single Source of Truth)
- Contract validation emphasis supports Principle 6 (Contracts at Every Handover)
- Infrastructure-as-Code positioning highlights Principle 10 (Observable Progress)
- Fresh memory architecture (Principle 4) prominently explained as differentiator

**✅ No new violations introduced**: Documentation changes only improve principle presentation

## Project Structure

### Documentation (this feature)

```
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Documentation Structure (repository root)

```
docs/
├── index.md                     # YAML-first landing page
├── paradigm/                    # NEW: AI-as-Code positioning
│   ├── ai-as-code.md           # Core paradigm explanation
│   ├── infrastructure-parallels.md  # K8s, Docker Compose comparisons
│   └── deliverables-contracts.md    # Guaranteed outputs concept
├── workflows/                   # RESTRUCTURED: YAML-first organization
│   ├── creating-workflows.md    # Primary how-to content
│   ├── sharing-workflows.md     # Git-based workflow sharing
│   ├── community-library.md     # Ecosystem and discovery
│   └── examples/               # Complete YAML workflow examples
├── concepts/                    # UPDATED: Supporting concepts
│   ├── personas.md             # Demoted from primary focus
│   ├── contracts.md            # Elevated as core concept
│   ├── workspaces.md          # Technical implementation detail
│   └── pipeline-execution.md   # How YAML becomes execution
├── reference/                   # MAINTAINED: Technical specs
│   ├── yaml-schema.md          # Comprehensive YAML reference
│   ├── cli-commands.md         # Command reference
│   └── troubleshooting.md      # Common issues and solutions
└── migration/                   # NEW: Adoption guides
    ├── from-personas-to-workflows.md
    ├── team-adoption.md
    └── enterprise-patterns.md
```

**Structure Decision**: Paradigm-first organization where users encounter the "AI as Code" concept immediately, then progress through workflow creation (YAML-first) to supporting concepts. Traditional persona-focused content becomes supporting reference material.

## Complexity Tracking

_No constitutional violations identified - section not needed._
