# Tasks

## Phase 1: Research & Analysis

- [X] Task 1.1: Fetch and analyze the Claude Code Teams gist content in detail
- [X] Task 1.2: Audit all 18 Wave persona `.md` files for role clarity and overlap [P]
- [X] Task 1.3: Audit all pipeline YAML files to build a persona usage matrix [P]
- [X] Task 1.4: Review `internal/adapter/claude.go` and `internal/manifest/types.go` for persona integration points

## Phase 2: Comparison Document

- [X] Task 2.1: Write comprehensive comparison document (`docs/persona-architecture-evaluation.md`)
  - Claude Code Teams architecture summary
  - Wave persona architecture summary
  - Pattern comparison (Pipeline, Leader-Worker, Swarm, Council, Watchdog)
  - Permission model comparison
  - Coordination mechanism comparison
- [X] Task 2.2: Document at least 3 actionable proposals with rationale

## Phase 3: Persona Consolidation

- [X] Task 3.1: Consolidate `craftsman` + `implementer` — merge implementer into craftsman [P]
  - Update `.wave/personas/craftsman.md` with implementer capabilities
  - Update `wave.yaml` persona entries
  - Update all pipeline YAML files referencing `implementer` to use `craftsman`
  - Remove or archive `.wave/personas/implementer.md`
- [X] Task 3.2: Consolidate `auditor` + `reviewer` — merge into unified `reviewer` [P]
  - Update `.wave/personas/reviewer.md` with security audit capabilities
  - Update `wave.yaml` persona entries
  - Update all pipeline YAML files referencing `auditor` to use `reviewer`
  - Remove or archive `.wave/personas/auditor.md`
- [X] Task 3.3: Clarify `planner` vs `philosopher` boundaries (or consolidate into `architect`)
  - Evaluate whether consolidation or clear differentiation is better
  - Update persona prompts with explicit scope boundaries
  - Update `wave.yaml` if consolidating

## Phase 4: Persona Prompt Enhancement

- [X] Task 4.1: Enhance remaining persona prompts with anti-patterns and output checklists [P]
- [X] Task 4.2: Add cross-persona awareness to base-protocol.md [P]
- [X] Task 4.3: Update `wave.yaml` permission models for any consolidated personas

## Phase 5: Testing & Validation

- [X] Task 5.1: Run `go test ./...` to verify no test regressions
- [X] Task 5.2: Validate all pipeline YAML files parse correctly (grep for removed persona names)
- [X] Task 5.3: Run `go vet ./...` for static analysis
- [X] Task 5.4: Final review of all changes for constitutional compliance
