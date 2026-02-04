# Tasks: Enterprise Documentation Enhancement

**Input**: Design documents from `/specs/001-enterprise-docs/`
**Prerequisites**: plan.md (required), spec.md (required for user stories)

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **VitePress docs**: `docs/` at repository root
- **Vue components**: `docs/.vitepress/theme/components/`
- **Styles**: `docs/.vitepress/theme/styles/`
- **Plugins**: `docs/.vitepress/plugins/`
- **Content**: `docs/[section]/[page].md`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: VitePress project initialization and custom theme structure

- [ ] T001 Initialize VitePress project structure in docs/ directory
- [ ] T002 Configure VitePress in docs/.vitepress/config.ts with site metadata, navigation, and sidebar
- [ ] T003 [P] Create custom theme entry point in docs/.vitepress/theme/index.ts
- [ ] T004 [P] Create base styles in docs/.vitepress/theme/styles/custom.css
- [ ] T005 [P] Create component styles in docs/.vitepress/theme/styles/components.css
- [ ] T006 [P] Setup TypeScript configuration for Vue components in docs/.vitepress/

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared components and utilities that ALL user stories depend on

**CRITICAL**: No user story work can begin until this phase is complete

- [ ] T007 Create CopyButton.vue component in docs/.vitepress/theme/components/CopyButton.vue with clipboard API integration
- [ ] T008 [P] Create copy-code.ts plugin in docs/.vitepress/plugins/copy-code.ts to inject copy buttons into code blocks
- [ ] T009 [P] Register CopyButton component globally in docs/.vitepress/theme/index.ts
- [ ] T010 Create shared type definitions in docs/.vitepress/theme/types.ts for component props
- [ ] T011 [P] Configure Mermaid.js integration in docs/.vitepress/config.ts for diagram rendering
- [ ] T012 [P] Setup navigation structure with Trust Center, Quickstart, Use Cases sections in docs/.vitepress/config.ts

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Enterprise Security Review (Priority: P0)

**Goal**: Security team leads can review Wave's security model, compliance posture, and audit capabilities to approve for production use

**Independent Test**: Navigate to Trust Center section, find downloadable security documentation, compliance status, and threat model documentation within 2 clicks from homepage

### Implementation for User Story 1

- [ ] T013 [P] [US1] Create Trust Center index page in docs/trust-center/index.md with overview of security resources
- [ ] T014 [P] [US1] Create TrustSignals.vue component in docs/.vitepress/theme/components/TrustSignals.vue displaying compliance badges (SOC 2, HIPAA, GDPR status)
- [ ] T015 [P] [US1] Create security-model.md in docs/trust-center/security-model.md documenting credential handling, workspace isolation, and permission enforcement
- [ ] T016 [P] [US1] Create compliance.md in docs/trust-center/compliance.md with compliance roadmap table showing certification status
- [ ] T017 [P] [US1] Create audit-logging.md in docs/trust-center/audit-logging.md with audit log specification and examples
- [ ] T018 [US1] Create downloads directory docs/trust-center/downloads/ with audit-log-schema.json
- [ ] T019 [US1] Generate security-whitepaper.pdf covering Wave's security model for docs/trust-center/downloads/
- [ ] T020 [US1] Add vulnerability disclosure section to docs/trust-center/index.md with security contact and process
- [ ] T021 [US1] Add Trust Center to main navigation sidebar in docs/.vitepress/config.ts

**Checkpoint**: User Story 1 (Enterprise Security Review) is fully functional and testable independently

---

## Phase 4: User Story 2 - Landing Page First Impression (Priority: P0)

**Goal**: First-time visitors immediately understand what Wave does, why it matters, and how to get started within 10 seconds

**Independent Test**: Measure time-to-comprehension and verify click-through paths to Get Started and View Examples CTAs

### Implementation for User Story 2

- [ ] T022 [P] [US2] Create HeroSection.vue component in docs/.vitepress/theme/components/HeroSection.vue with value proposition headline, tagline, and CTA buttons
- [ ] T023 [P] [US2] Create FeatureCards.vue component in docs/.vitepress/theme/components/FeatureCards.vue with grid layout for personas, pipelines, contracts, security features
- [ ] T024 [US2] Design and add feature icons to docs/.vitepress/theme/assets/ for each capability card
- [ ] T025 [US2] Redesign docs/index.md landing page integrating HeroSection, FeatureCards, and TrustSignals components
- [ ] T026 [US2] Add "Get Started" CTA linking to quickstart in docs/index.md hero section
- [ ] T027 [US2] Add "View Examples" CTA linking to use cases in docs/index.md hero section
- [ ] T028 [US2] Add trust signals section to docs/index.md with security highlights and Trust Center link
- [ ] T029 [US2] Add responsive styles for landing page components in docs/.vitepress/theme/styles/components.css

**Checkpoint**: User Story 2 (Landing Page First Impression) is fully functional and testable independently

---

## Phase 5: User Story 3 - Developer Quickstart Experience (Priority: P0)

**Goal**: Developers can install Wave and run their first pipeline within 15 minutes on any OS (macOS, Linux, Windows)

**Independent Test**: Time a new user from documentation landing to first successful pipeline run across all three platforms

### Implementation for User Story 3

- [ ] T030 [P] [US3] Create PlatformTabs.vue component in docs/.vitepress/theme/components/PlatformTabs.vue with macOS, Linux, Windows tabs
- [ ] T031 [P] [US3] Add platform detection logic to PlatformTabs.vue to auto-select user's OS
- [ ] T032 [US3] Enhance docs/quickstart.md with PlatformTabs for installation commands
- [ ] T033 [US3] Add macOS installation content to docs/quickstart.md with Homebrew and binary options
- [ ] T034 [US3] Add Linux installation content to docs/quickstart.md with package manager and binary options
- [ ] T035 [US3] Add Windows installation content to docs/quickstart.md with Scoop, Chocolatey, and binary options
- [ ] T036 [US3] Add troubleshooting callout boxes for common errors in docs/quickstart.md
- [ ] T037 [P] [US3] Create adapter selection section in docs/quickstart.md with tabs for Claude Code, OpenCode, etc.
- [ ] T038 [US3] Add first pipeline tutorial to docs/quickstart.md with expected output examples
- [ ] T039 [US3] Ensure all code blocks in docs/quickstart.md have CopyButton integration via plugin

**Checkpoint**: User Story 3 (Developer Quickstart) is fully functional and testable independently

---

## Phase 6: User Story 4 - Pipeline Configuration Learning (Priority: P1)

**Goal**: Developers can configure complex pipelines with multiple personas, dependencies, and contracts using only documentation

**Independent Test**: Developer creates a multi-step pipeline with contracts using only documentation, no source code access

### Implementation for User Story 4

- [ ] T040 [P] [US4] Create PipelineVisualizer.vue component in docs/.vitepress/theme/components/PipelineVisualizer.vue rendering DAG from YAML
- [ ] T041 [P] [US4] Add Mermaid.js integration to PipelineVisualizer.vue for step dependencies and artifact flow
- [ ] T042 [P] [US4] Create YamlPlayground.vue component in docs/.vitepress/theme/components/YamlPlayground.vue with editor pane
- [ ] T043 [US4] Add YAML parsing library (js-yaml) to YamlPlayground.vue for client-side validation
- [ ] T044 [US4] Add schema validation to YamlPlayground.vue with real-time error feedback
- [ ] T045 [US4] Create pipeline configuration guide in docs/guides/pipeline-configuration.md
- [ ] T046 [US4] Add interactive PipelineVisualizer examples to docs/guides/pipeline-configuration.md
- [ ] T047 [US4] Create sample pipeline YAML files in docs/examples/ for use in playground
- [ ] T048 [US4] Add YamlPlayground sandbox section to docs/guides/pipeline-configuration.md
- [ ] T049 [US4] Document pipeline dependencies, artifacts, and contract patterns in docs/concepts/pipelines.md

**Checkpoint**: User Story 4 (Pipeline Configuration Learning) is fully functional and testable independently

---

## Phase 7: User Story 5 - Persona and Permission Understanding (Priority: P1)

**Goal**: Team architects understand Wave's persona system, permission model, and security boundaries to design workflows

**Independent Test**: Architect can explain the permission model to their security team using only documentation

### Implementation for User Story 5

- [ ] T050 [P] [US5] Create PermissionMatrix.vue component in docs/.vitepress/theme/components/PermissionMatrix.vue with interactive table
- [ ] T051 [US5] Add persona data structure to PermissionMatrix.vue supporting filtering and sorting
- [ ] T052 [US5] Style PermissionMatrix.vue with deny (red), allow (green), conditional (yellow) indicators
- [ ] T053 [US5] Create persona documentation page in docs/concepts/personas.md with PermissionMatrix integration
- [ ] T054 [US5] Document deny-first evaluation, allow patterns, and permission inheritance in docs/concepts/personas.md
- [ ] T055 [US5] Add workspace isolation visualization to docs/concepts/personas.md using Mermaid diagrams
- [ ] T056 [US5] Document fresh memory boundaries with visual explanation in docs/concepts/personas.md
- [ ] T057 [US5] Create custom persona guide in docs/guides/custom-personas.md with validation examples
- [ ] T058 [US5] Add security boundary documentation to docs/trust-center/security-model.md referencing permission matrix

**Checkpoint**: User Story 5 (Persona and Permission Understanding) is fully functional and testable independently

---

## Phase 8: User Story 6 - Integration with Existing Workflows (Priority: P2)

**Goal**: DevOps engineers can integrate Wave into CI/CD pipelines and GitHub workflows

**Independent Test**: Follow integration guides to set up Wave in a CI/CD environment within 30 minutes

### Implementation for User Story 6

- [ ] T059 [P] [US6] Create integrations directory docs/integrations/
- [ ] T060 [P] [US6] Create GitHub Actions integration guide in docs/integrations/github-actions.md with copy-ready YAML
- [ ] T061 [P] [US6] Create GitLab CI integration guide in docs/integrations/gitlab-ci.md with copy-ready YAML
- [ ] T062 [US6] Add CI/CD pipeline examples with environment variable handling in docs/integrations/
- [ ] T063 [US6] Create error codes reference in docs/reference/error-codes.md with resolution steps
- [ ] T064 [US6] Add troubleshooting section to each integration guide with common CI/CD issues
- [ ] T065 [US6] Add integrations section to main navigation in docs/.vitepress/config.ts

**Checkpoint**: User Story 6 (Integration with Existing Workflows) is fully functional and testable independently

---

## Phase 9: User Story 7 - Use Case Discovery (Priority: P2)

**Goal**: Users can discover pre-built pipeline patterns for common tasks like code review, security auditing, documentation generation

**Independent Test**: Navigate to use cases and find a complete, runnable pipeline for a common task

### Implementation for User Story 7

- [ ] T066 [P] [US7] Create UseCaseGallery.vue component in docs/.vitepress/theme/components/UseCaseGallery.vue with card grid
- [ ] T067 [US7] Add filtering logic to UseCaseGallery.vue supporting category, complexity, and persona type filters
- [ ] T068 [US7] Add search/filter UI to UseCaseGallery.vue with filter chips
- [ ] T069 [US7] Create use cases index page in docs/use-cases/index.md with UseCaseGallery integration
- [ ] T070 [P] [US7] Create code-review use case in docs/use-cases/code-review.md with complete pipeline YAML
- [ ] T071 [P] [US7] Create security-audit use case in docs/use-cases/security-audit.md with complete pipeline YAML
- [ ] T072 [P] [US7] Create documentation-generation use case in docs/use-cases/documentation-generation.md
- [ ] T073 [P] [US7] Create test-generation use case in docs/use-cases/test-generation.md with complete pipeline YAML
- [ ] T074 [P] [US7] Create refactoring use case in docs/use-cases/refactoring.md with complete pipeline YAML
- [ ] T075 [P] [US7] Create multi-agent-review use case in docs/use-cases/multi-agent-review.md
- [ ] T076 [P] [US7] Create incident-response use case in docs/use-cases/incident-response.md
- [ ] T077 [P] [US7] Create onboarding use case in docs/use-cases/onboarding.md
- [ ] T078 [P] [US7] Create api-design use case in docs/use-cases/api-design.md
- [ ] T079 [P] [US7] Create migration use case in docs/use-cases/migration.md
- [ ] T080 [US7] Add complexity badges and prerequisite tags to all use case pages
- [ ] T081 [US7] Add Use Cases to main navigation in docs/.vitepress/config.ts

**Checkpoint**: User Story 7 (Use Case Discovery) is fully functional and testable independently

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T082 [P] Add breadcrumb navigation component in docs/.vitepress/theme/components/Breadcrumb.vue
- [ ] T083 [P] Create changelog page in docs/changelog.md with version history
- [ ] T084 [P] Add visual card navigation to major section index pages
- [ ] T085 Enhance existing docs/concepts/ pages with improved formatting and copy buttons
- [ ] T086 Enhance existing docs/guides/ pages with improved formatting and copy buttons
- [ ] T087 Update docs/reference/cli.md with improved structure and copy buttons
- [ ] T088 Update docs/reference/manifest-schema.md with interactive examples
- [ ] T089 Add Go interface documentation to docs/reference/ (when programmatic usage available)
- [ ] T090 Run VitePress build validation and fix any errors
- [ ] T091 Run link checker and fix broken links
- [ ] T092 Verify WCAG 2.1 AA accessibility compliance across all components
- [ ] T093 Performance optimization: Verify page load < 2s and component response < 100ms
- [ ] T094 Add meta descriptions and Open Graph tags for SEO in docs/.vitepress/config.ts

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-9)**: All depend on Foundational phase completion
  - P0 stories (US1, US2, US3) should be prioritized
  - P1 stories (US4, US5) can follow
  - P2 stories (US6, US7) complete the feature set
- **Polish (Phase 10)**: Depends on all desired user stories being complete

### User Story Dependencies

| Story | Priority | Dependencies | Can Parallel With |
|-------|----------|--------------|-------------------|
| US1 (Trust Center) | P0 | Phase 2 only | US2, US3 |
| US2 (Landing Page) | P0 | Phase 2, US1 (for TrustSignals) | US3 |
| US3 (Quickstart) | P0 | Phase 2 only | US1, US2 |
| US4 (Pipeline Config) | P1 | Phase 2 only | US5 |
| US5 (Personas) | P1 | Phase 2 only | US4 |
| US6 (Integrations) | P2 | Phase 2 only | US7 |
| US7 (Use Cases) | P2 | Phase 2 only | US6 |

### Within Each User Story

- Components before content pages that use them
- Base pages before enhanced sections
- Core implementation before polish

### Parallel Opportunities

**Phase 1 (Setup)**:
- T003, T004, T005, T006 can run in parallel

**Phase 2 (Foundational)**:
- T008, T009, T011, T012 can run in parallel after T007

**Phase 3 (US1 - Trust Center)**:
- T013, T014, T015, T016, T017 can all run in parallel

**Phase 4 (US2 - Landing Page)**:
- T022, T023 can run in parallel

**Phase 5 (US3 - Quickstart)**:
- T030, T031, T037 can run in parallel

**Phase 6 (US4 - Pipeline Config)**:
- T040, T041, T042 can run in parallel

**Phase 9 (US7 - Use Cases)**:
- T070-T079 (all use case pages) can run in parallel

---

## Parallel Example: User Story 1 (Trust Center)

```bash
# Launch all Trust Center content pages in parallel:
Task: "Create Trust Center index page in docs/trust-center/index.md"
Task: "Create TrustSignals.vue component in docs/.vitepress/theme/components/TrustSignals.vue"
Task: "Create security-model.md in docs/trust-center/security-model.md"
Task: "Create compliance.md in docs/trust-center/compliance.md"
Task: "Create audit-logging.md in docs/trust-center/audit-logging.md"
```

---

## Parallel Example: User Story 7 (Use Cases)

```bash
# Launch all use case pages in parallel:
Task: "Create code-review use case in docs/use-cases/code-review.md"
Task: "Create security-audit use case in docs/use-cases/security-audit.md"
Task: "Create documentation-generation use case in docs/use-cases/documentation-generation.md"
Task: "Create test-generation use case in docs/use-cases/test-generation.md"
Task: "Create refactoring use case in docs/use-cases/refactoring.md"
Task: "Create multi-agent-review use case in docs/use-cases/multi-agent-review.md"
Task: "Create incident-response use case in docs/use-cases/incident-response.md"
Task: "Create onboarding use case in docs/use-cases/onboarding.md"
Task: "Create api-design use case in docs/use-cases/api-design.md"
Task: "Create migration use case in docs/use-cases/migration.md"
```

---

## Implementation Strategy

### MVP First (P0 User Stories)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (Trust Center)
4. Complete Phase 4: User Story 2 (Landing Page)
5. Complete Phase 5: User Story 3 (Quickstart)
6. **STOP and VALIDATE**: Test all P0 stories independently
7. Deploy/demo MVP

### Incremental Delivery

1. Complete Setup + Foundational -> Foundation ready
2. Add US1 (Trust Center) -> Test independently -> Enterprise security reviews possible
3. Add US2 (Landing Page) -> Test independently -> Professional first impression
4. Add US3 (Quickstart) -> Test independently -> Developer onboarding works (MVP!)
5. Add US4 (Pipeline Config) -> Test independently -> Advanced configuration documented
6. Add US5 (Personas) -> Test independently -> Security model fully documented
7. Add US6 (Integrations) -> Test independently -> CI/CD integration ready
8. Add US7 (Use Cases) -> Test independently -> Complete feature set
9. Polish phase -> Final quality pass

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (Trust Center)
   - Developer B: User Story 2 (Landing Page)
   - Developer C: User Story 3 (Quickstart)
3. P1 stories can start while P0 stories complete:
   - Developer D: User Story 4 (Pipeline Config)
   - Developer E: User Story 5 (Personas)
4. Stories complete and integrate independently

---

## Summary

| Metric | Count |
|--------|-------|
| **Total Tasks** | 94 |
| **Setup Tasks** | 6 |
| **Foundational Tasks** | 6 |
| **US1 Tasks (Trust Center)** | 9 |
| **US2 Tasks (Landing Page)** | 8 |
| **US3 Tasks (Quickstart)** | 10 |
| **US4 Tasks (Pipeline Config)** | 10 |
| **US5 Tasks (Personas)** | 9 |
| **US6 Tasks (Integrations)** | 7 |
| **US7 Tasks (Use Cases)** | 16 |
| **Polish Tasks** | 13 |
| **Parallelizable Tasks** | 42 (45%) |

### MVP Scope (P0 Stories Only)

- **Tasks**: T001-T039 (39 tasks)
- **Deliverables**: Setup + Foundational + Trust Center + Landing Page + Quickstart
- **User Value**: Enterprise security reviews, professional first impression, developer onboarding

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Tests not included per spec (no explicit test requirement)
