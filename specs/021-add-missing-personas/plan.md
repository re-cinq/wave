# Implementation Plan: Add Missing Personas

**Branch**: `021-add-missing-personas` | **Date**: 2026-02-04 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/021-add-missing-personas/spec.md`

## Summary

Add the missing `implementer` and `reviewer` personas that are referenced by default pipelines (gh-poor-issues, umami, doc-loop) but not defined. The implementer persona requires broad execution permissions (Read, Write, Bash, Edit) for code changes and artifact output. The reviewer persona requires read-focused permissions with write access for artifact output. Both must be added to wave.yaml, .wave/personas/, and internal/defaults/personas/ for wave init scaffolding.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: gopkg.in/yaml.v3, github.com/spf13/cobra (existing Wave dependencies)
**Storage**: Filesystem (YAML config, Markdown persona files)
**Testing**: go test ./...
**Target Platform**: CLI (Linux/macOS/Windows)
**Project Type**: Single binary CLI
**Performance Goals**: N/A (configuration changes only)
**Constraints**: Single static binary (Principle 1), Manifest as single source of truth (Principle 2)
**Scale/Scope**: 2 new personas, 4 files to create/modify

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Verification |
|-----------|--------|--------------|
| P1: Single Binary, Minimal Dependencies | PASS | No new dependencies required - only config files |
| P2: Manifest as Single Source of Truth | PASS | Personas defined in wave.yaml, referenced by system_prompt_file |
| P3: Persona-Scoped Execution Boundaries | PASS | Each persona has distinct permissions; implementer has Write, reviewer has limited Write |
| P4: Fresh Memory at Every Step | N/A | No changes to memory behavior |
| P5: Navigator-First Architecture | N/A | Personas don't modify pipeline structure |
| P6: Contracts at Every Handover | PASS | Both personas designed to output artifact.json for contract validation |
| P7: Relay via Dedicated Summarizer | N/A | No relay changes |
| P8: Ephemeral Workspaces | N/A | Workspace behavior unchanged |
| P9: Credentials Never Touch Disk | PASS | No credential handling in personas |
| P10: Observable Progress | N/A | No progress emission changes |
| P11: Bounded Recursion | N/A | No recursion changes |
| P12: Minimal Step State Machine | N/A | No state machine changes |

**Constitution Check Result**: PASS - No violations detected

## Project Structure

### Documentation (this feature)

```
specs/021-add-missing-personas/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── checklists/          # Quality checklists
│   └── requirements.md  # Spec quality checklist
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```
wave.yaml                              # Add implementer and reviewer persona definitions
.wave/personas/
├── implementer.md                     # NEW: Implementer persona system prompt
└── reviewer.md                        # NEW: Reviewer persona system prompt

internal/defaults/personas/
├── implementer.md                     # NEW: Embedded default for wave init
└── reviewer.md                        # NEW: Embedded default for wave init
```

**Structure Decision**: Single project structure. This feature only adds configuration files (persona definitions in YAML and Markdown) with no code changes to the Go source.

## Complexity Tracking

_No violations - table not needed_

## Implementation Phases

### Phase 0: Research (Complete)

No external research needed - this is a configuration addition following existing patterns. See existing personas (craftsman, navigator, auditor) for templates.

### Phase 1: Design

#### Implementer Persona Design

**Purpose**: Execute code changes, run commands, and write artifacts for pipeline handoffs

**Permissions** (modeled after craftsman):
```yaml
implementer:
  adapter: claude
  description: Code execution and artifact generation
  permissions:
    allowed_tools:
      - Read
      - Write
      - Edit
      - Bash
    deny:
      - Bash(rm -rf /*)
  system_prompt_file: .wave/personas/implementer.md
```

**System Prompt Structure**:
- Role: Execution specialist for implementing changes and producing structured output
- Responsibilities: Execute code changes, run commands, produce JSON artifacts
- Output Format: Structured JSON when used with contracts (schema injected at runtime)
- Constraints: Focus on task completion, write artifact.json for handoff

#### Reviewer Persona Design

**Purpose**: Review and validate work from other steps, produce quality assessments

**Permissions** (read-focused with artifact write):
```yaml
reviewer:
  adapter: claude
  description: Quality review and validation
  permissions:
    allowed_tools:
      - Read
      - Glob
      - Grep
      - Write(artifact.json)
      - Bash(go test*)
      - Bash(npm test*)
    deny:
      - Write(*.go)
      - Write(*.ts)
      - Edit(*)
  system_prompt_file: .wave/personas/reviewer.md
```

**System Prompt Structure**:
- Role: Quality reviewer and validator
- Responsibilities: Review implementations, validate correctness, produce review reports
- Output Format: Structured JSON review results (schema injected at runtime)
- Constraints: Read-only for source code, can only write artifact.json

### Phase 2: Implementation Tasks

See `tasks.md` (generated by `/speckit.tasks`)

## Files to Create/Modify

| File | Action | Description |
|------|--------|-------------|
| `wave.yaml` | MODIFY | Add implementer and reviewer persona definitions |
| `.wave/personas/implementer.md` | CREATE | System prompt for implementer persona |
| `.wave/personas/reviewer.md` | CREATE | System prompt for reviewer persona |
| `internal/defaults/personas/implementer.md` | CREATE | Embedded default for wave init |
| `internal/defaults/personas/reviewer.md` | CREATE | Embedded default for wave init |

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Permission too broad for implementer | Low | Medium | Model after craftsman which is proven safe |
| Permission too narrow for reviewer | Medium | Low | Include test execution for validation workflows |
| Wave init doesn't pick up new defaults | Low | Medium | Verify embed.go includes new files automatically |

## Testing Strategy

1. **Unit Tests**: Verify persona resolution in manifest parser
2. **Integration Tests**: Run gh-poor-issues pipeline with new personas
3. **Regression Tests**: Ensure existing tests pass (`go test ./...`)
4. **Manual Verification**: Run `wave init` and verify new personas are scaffolded
