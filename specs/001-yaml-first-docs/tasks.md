# Tasks: ngrok-Inspired Documentation Restructure

**Input**: Design documents from `/specs/001-yaml-first-docs/`
**Prerequisites**: plan.md, spec.md
**Design Reference**: [ngrok documentation](https://ngrok.com/docs)

**Organization**: Tasks grouped by user story for independent implementation.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story (US1, US2, US3, US4)
- Include exact file paths

## Path Conventions

Repository root: `/home/libretech/Repos/wave/`

---

## Phase 1: Setup (Directory Structure)

**Purpose**: Create new directory structure, preserve existing content

- [ ] T001 Create docs/use-cases/ directory for task-oriented documentation
- [ ] T002 [P] Create docs/guides/ directory for advanced patterns
- [ ] T003 [P] Backup docs/paradigm/ to docs/.archive/paradigm/ before removal
- [ ] T004 [P] Backup docs/workflows/ to docs/.archive/workflows/ before removal
- [ ] T005 [P] Backup docs/migration/ to docs/.archive/migration/ before removal

---

## Phase 2: User Story 1 - First Pipeline in 60 Seconds (Priority: P1)

**Goal**: Developer runs first pipeline within 60 seconds of landing

**Independent Test**: Time new developer from landing to first `wave run`

### Implementation

- [ ] T006 [US1] Create docs/quickstart.md with 60-second pipeline flow (CRITICAL)
- [ ] T007 [US1] Add escape route for missing Claude CLI in docs/quickstart.md
- [ ] T008 [US1] Add escape route for missing API key in docs/quickstart.md
- [ ] T009 [US1] Add escape route for no codebase (self-analysis fallback) in docs/quickstart.md
- [ ] T010 [P] [US1] Rewrite docs/index.md hero: one paragraph explaining Wave
- [ ] T011 [P] [US1] Add "What is Wave" diagram to docs/index.md
- [ ] T012 [US1] Add single CTA "Get started in 60 seconds" linking to quickstart
- [ ] T013 [US1] Remove persona mentions from docs/index.md hero section

**Checkpoint**: Quickstart flow testable - time to first pipeline

---

## Phase 3: User Story 2 - Task-Based Discovery (Priority: P2)

**Goal**: Developer finds relevant use-case in under 5 seconds

**Independent Test**: Give developers tasks, measure time to find documentation

### Implementation

- [ ] T014 [US2] Create docs/use-cases/index.md with card-based overview
- [ ] T015 [P] [US2] Create docs/use-cases/code-review.md with complete runnable pipeline
- [ ] T016 [P] [US2] Create docs/use-cases/security-audit.md with complete runnable pipeline
- [ ] T017 [P] [US2] Create docs/use-cases/docs-generation.md with complete runnable pipeline
- [ ] T018 [P] [US2] Create docs/use-cases/test-generation.md with complete runnable pipeline
- [ ] T019 [US2] Add expected output examples to each use-case page
- [ ] T020 [US2] Add "Next Steps" section to each use-case page

**Checkpoint**: Task-based navigation testable

---

## Phase 4: User Story 3 - Progressive Complexity (Priority: P3)

**Goal**: Simple examples first, complexity added progressively

**Independent Test**: Verify first example on each page is under 10 lines

### Concept Pages (1-2 sentences + code pattern)

- [ ] T021 [US3] Create docs/concepts/index.md with concept overview
- [ ] T022 [P] [US3] Rewrite docs/concepts/pipelines.md: 1-2 sentences + minimal example
- [ ] T023 [P] [US3] Rewrite docs/concepts/personas.md: 1-2 sentences + minimal example
- [ ] T024 [P] [US3] Rewrite docs/concepts/contracts.md: progressive examples (simple → complex)
- [ ] T025 [P] [US3] Create docs/concepts/artifacts.md: 1-2 sentences + minimal example
- [ ] T026 [P] [US3] Rewrite docs/concepts/execution.md from pipeline-execution.md
- [ ] T027 [US3] Add "Next Steps" section to every concept page
- [ ] T028 [US3] Verify all first examples are under 10 lines YAML

### Reference Pages (command + output pairs)

- [ ] T029 [P] [US3] Rewrite docs/reference/cli.md with command + expected output pairs
- [ ] T030 [P] [US3] Create docs/reference/manifest.md with copy-paste examples
- [ ] T031 [P] [US3] Create docs/reference/pipeline-schema.md with required/optional fields
- [ ] T032 [P] [US3] Create docs/reference/contract-types.md with all contract types

**Checkpoint**: Progressive complexity testable

---

## Phase 5: User Story 4 - Team Adoption Path (Priority: P4)

**Goal**: Clear adoption path from individual to team to enterprise

**Independent Test**: Team lead can articulate adoption steps

### Implementation

- [ ] T033 [US4] Create docs/guides/ci-cd.md with GitHub Actions and GitLab CI examples
- [ ] T034 [P] [US4] Rewrite docs/guides/team-adoption.md from migration/team-adoption.md
- [ ] T035 [P] [US4] Rewrite docs/guides/enterprise.md from migration/enterprise-patterns.md
- [ ] T036 [US4] Add "Next Steps" section to all guide pages
- [ ] T037 [US4] Ensure progression: git sharing → team patterns → enterprise controls

**Checkpoint**: Adoption path clear and testable

---

## Phase 6: Cleanup & Validation

**Purpose**: Remove deprecated content, validate all examples

### Cleanup

- [ ] T038 [P] Remove docs/paradigm/ directory (content integrated elsewhere)
- [ ] T039 [P] Remove docs/workflows/ directory (replaced by use-cases/)
- [ ] T040 [P] Remove docs/migration/ directory (replaced by guides/)
- [ ] T041 Update docs navigation config for new structure

### Validation

- [ ] T042 [P] Validate all YAML examples are syntactically correct
- [ ] T043 [P] Validate all code examples are copy-paste runnable
- [ ] T044 [P] Verify every page has "Next Steps" section
- [ ] T045 Test quickstart flow end-to-end (target: 60 seconds)
- [ ] T046 Test task-based navigation (target: 5 seconds to find use-case)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies - start immediately
- **Phase 2 (US1)**: Depends on Phase 1
- **Phase 3 (US2)**: Depends on Phase 1, can run parallel to US1
- **Phase 4 (US3)**: Depends on Phase 1, can run parallel to US1/US2
- **Phase 5 (US4)**: Depends on Phase 1, can run parallel to US1/US2/US3
- **Phase 6 (Cleanup)**: Depends on all user stories complete

### Parallel Opportunities

**Phase 1**: All tasks can run in parallel (T001-T005)

**After Phase 1 completes**: All user stories can run in parallel:
- Agent A: User Story 1 (quickstart + landing)
- Agent B: User Story 2 (use-cases)
- Agent C: User Story 3 (concepts + reference)
- Agent D: User Story 4 (guides)

**Within each phase**: Tasks marked [P] can run in parallel

---

## Parallel Example: 6 Agents

```
Agent 1: T001-T005 (Setup - sequential)
         Then: T006-T013 (US1 - quickstart/landing)

Agent 2: Wait for T001
         Then: T014-T020 (US2 - use-cases)

Agent 3: Wait for T001
         Then: T021-T028 (US3 concepts)

Agent 4: Wait for T001
         Then: T029-T032 (US3 reference)

Agent 5: Wait for T001
         Then: T033-T037 (US4 - guides)

Agent 6: Wait for all user stories
         Then: T038-T046 (cleanup/validation)
```

---

## Content Standards

### Every Page Must Have

1. **Opening**: 1-2 sentences explaining the concept/task
2. **Code Example**: Minimal working example (first example < 10 lines)
3. **Next Steps**: 2-3 links to related content

### Code Examples Must Be

1. **Copy-paste runnable**: No modifications needed
2. **Progressive**: Simple → complex within each page
3. **Real**: Based on actual Wave syntax (no invented features)

### ngrok Patterns to Follow

1. **Problem-outcome framing**: Explain why before how
2. **Escape routes**: "Don't have X? Try this instead"
3. **Task-oriented**: Organize by what users want to do
4. **Minimal prose**: Let code examples speak

---

## Key Metrics

| Metric | Target | How to Test |
|--------|--------|-------------|
| Time to first pipeline | < 60 seconds | User testing |
| Time to find use-case | < 5 seconds | Navigation testing |
| First example size | < 10 lines | Manual review |
| Pages with Next Steps | 100% | Automated check |
| Runnable examples | 100% | YAML validation |

---

## Notes

- All tasks use checkbox format for trackability
- [P] tasks can run concurrently within same phase
- [Story] labels enable independent user story work
- Focus on ngrok patterns: task-oriented, minimal prose, progressive complexity
- Remove paradigm/ section entirely - integrate key points into index.md
