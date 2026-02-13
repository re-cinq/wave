# Data Model: Expand Persona Definitions

**Feature**: 096-expand-persona-prompts
**Date**: 2026-02-13

## Overview

This feature is a **content-only change** — no new data types, database schemas, or API contracts are introduced. The "data model" describes the structure of persona definition files (Markdown) and the file-system layout they occupy.

## Entity: Persona Definition File

A persona definition is a Markdown file that serves as the complete behavioral specification for an AI agent within a Wave pipeline step.

### File Locations (Dual-Location Model)

```
.wave/personas/{name}.md              # Runtime / user-facing copy
internal/defaults/personas/{name}.md  # Embedded copy (compiled into binary via //go:embed)
```

**Invariant (FR-010)**: Contents of both locations MUST be byte-identical for every persona file.

### Structural Template

Every persona definition file follows this consistent Markdown structure. Section headings may be adapted for role clarity (e.g., "Debugging Process" instead of "Process"), but all 7 concepts must be present.

```
# {Persona Name}                           ← REQUIRED: Concept 1 — Identity
                                            ← "You are..." opening paragraph
You are {identity statement}.

## Domain Expertise                        ← REQUIRED: Concept 2 — Domain Expertise
- {Knowledge area 1}
- {Knowledge area 2}
- {Knowledge area N}

## Responsibilities                        ← REQUIRED: Concept 3 — Responsibilities
- {Duty 1}
- {Duty N}

## [Communication Style]                   ← OPTIONAL: Additional section
- {Style attributes}

## Process                                 ← REQUIRED: Concept 4 — Process/Methodology
1. {Step 1}
N. {Step N}

## Tools and Permissions                   ← REQUIRED: Concept 5 — Tools
- {Tool}: {description}
- Note: Actual permissions enforced by pipeline orchestrator

## Output Format                           ← REQUIRED: Concept 6 — Output Format
{Default output structure}
Note: Contract schemas override when provided.

## Constraints                             ← REQUIRED: Concept 7 — Constraints
- {Hard boundary 1}
- {Hard boundary N}
```

### Validation Rules

| Rule | Source | Check |
|------|--------|-------|
| Min 30 lines | FR-009 | `wc -l >= 30` |
| Max 200 lines | FR-013 | `wc -l <= 200` |
| Identity statement present | FR-001 | "You are" within first 3 lines of body |
| All 7 concepts present | FR-002–FR-007 | Section headings or equivalent content |
| Language-agnostic | FR-008 | No hardcoded language toolchain references |
| Parity | FR-010 | `diff -r .wave/personas/ internal/defaults/personas/` = 0 |

### File Inventory

13 persona files exist in both locations:

| # | File Name | Role |
|---|-----------|------|
| 1 | `navigator.md` | Codebase exploration and analysis |
| 2 | `philosopher.md` | Architecture and specification writing |
| 3 | `planner.md` | Task planning and decomposition |
| 4 | `craftsman.md` | Production-quality implementation |
| 5 | `implementer.md` | Task execution and artifact production |
| 6 | `reviewer.md` | Quality review and validation |
| 7 | `auditor.md` | Security and compliance review |
| 8 | `debugger.md` | Bug investigation and root cause analysis |
| 9 | `researcher.md` | Web research and information synthesis |
| 10 | `summarizer.md` | Context compaction and summarization |
| 11 | `github-analyst.md` | GitHub issue analysis and scoring |
| 12 | `github-commenter.md` | GitHub issue commenting |
| 13 | `github-enhancer.md` | GitHub issue enhancement |

## Relationship to Existing Code

### Loading Mechanism (NOT modified by this feature)

```
wave.yaml persona config
  └── system_prompt_file: "personas/craftsman.md"
        └── resolved by manifest parser against .wave/ directory
              └── file content loaded as system prompt string
                    └── injected into adapter subprocess
```

### Embedding Mechanism (NOT modified by this feature)

```
internal/defaults/embed.go
  └── //go:embed personas/*.md
        └── Go compiler embeds file contents at build time
              └── wave init extracts to .wave/personas/ in new projects
```

## No API Contracts

This feature does not introduce or modify any API contracts. The persona files are consumed as opaque string content by the manifest parser. There is no schema to validate against other than the structural template described above, which is a human-readable convention, not a machine-enforced schema.

## Change Impact

| Artifact | Modified | Reason |
|----------|----------|--------|
| `.wave/personas/*.md` (4 files) | YES | Fix FR-008 violations |
| `internal/defaults/personas/*.md` (13 files) | YES | Sync parity (FR-010) |
| `.go` files | NO | FR-011 prohibits |
| `wave.yaml` | NO | FR-011 prohibits |
| JSON schemas | NO | FR-011 prohibits |
| Test files | NO | Content-only change; no behavioral changes |
