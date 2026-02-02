---
description: "Task list for Critical Security Vulnerability Fixes"
---

# Tasks: Critical Security Vulnerability Fixes

**Input**: Design documents from `/specs/017-fix-critical-security/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Security-focused tests are included to validate vulnerability fixes

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each security fix

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

Based on plan.md: Wave project structure with `internal/` packages

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Security infrastructure initialization

- [x] T001 Create internal/security package structure
- [x] T002 [P] Initialize security logging framework in internal/security/logging.go
- [x] T003 [P] Setup security testing framework in tests/security/

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core security infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T004 Create SecurityViolationEvent struct in internal/security/events.go
- [x] T005 [P] Implement base path validation utilities in internal/security/path.go
- [x] T006 [P] Implement base input sanitization in internal/security/sanitize.go
- [x] T007 [P] Create security configuration structs in internal/security/config.go
- [x] T008 Setup security error types in internal/security/errors.go
- [x] T009 [P] Create security test utilities in tests/security/utils.go

**Checkpoint**: Security foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Secure Schema Processing (Priority: P1) 🎯 MVP

**Goal**: Prevent path traversal vulnerabilities and sanitize schema content

**Independent Test**: Create pipelines with malicious schema paths and verify they are blocked with proper logging

### Security Tests for User Story 1

**NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T010 [P] [US1] Path traversal attack tests in tests/security/path_test.go
- [ ] T011 [P] [US1] Schema content sanitization tests in tests/security/schema_test.go
- [ ] T012 [P] [US1] Security logging validation tests in tests/security/logging_test.go

### Implementation for User Story 1

- [x] T013 [P] [US1] Implement path validation functions in internal/security/path.go
- [x] T014 [P] [US1] Create SchemaValidationResult struct in internal/security/types.go
- [x] T015 [US1] Integrate path validation into buildStepPrompt in internal/pipeline/executor.go
- [x] T016 [US1] Add schema content sanitization to buildStepPrompt in internal/pipeline/executor.go
- [x] T017 [US1] Implement security violation logging in schema processing
- [ ] T018 [US1] Add approved directory configuration to manifest types in internal/manifest/types.go

**Checkpoint**: Path traversal prevention should be fully functional and testable independently

---

## Phase 4: User Story 2 - Validated User Input Processing (Priority: P1)

**Goal**: Sanitize user input to prevent prompt injection and AI manipulation

**Independent Test**: Submit various malicious payloads and verify proper sanitization without breaking legitimate use

### Security Tests for User Story 2

- [ ] T019 [P] [US2] Prompt injection detection tests in tests/security/injection_test.go
- [ ] T020 [P] [US2] Input length validation tests in tests/security/input_test.go
- [ ] T021 [P] [US2] Sanitization effectiveness tests in tests/security/sanitize_test.go

### Implementation for User Story 2

- [ ] T022 [P] [US2] Create InputSanitizationRecord struct in internal/security/types.go
- [ ] T023 [P] [US2] Implement prompt injection detection in internal/security/sanitize.go
- [ ] T024 [US2] Add input length validation to sanitization functions
- [ ] T025 [US2] Integrate user input sanitization into task processing pipeline
- [ ] T026 [US2] Add sanitization to meta-pipeline input handling in internal/pipeline/meta.go
- [ ] T027 [US2] Implement risk scoring algorithm for input assessment
- [ ] T028 [US2] Add sanitization configuration to manifest structure

**Checkpoint**: User input sanitization should be fully functional across all input vectors

---

## Phase 5: User Story 3 - Robust Meta-Pipeline Generation (Priority: P2)

**Goal**: Ensure meta-pipeline generation only references valid personas and produces valid JSON

**Independent Test**: Generate meta-pipelines and verify they execute with valid personas and proper JSON

### Tests for User Story 3

- [ ] T029 [P] [US3] Persona validation tests in tests/security/persona_test.go
- [ ] T030 [P] [US3] Meta-pipeline generation tests in tests/pipeline/meta_test.go
- [ ] T031 [P] [US3] JSON output validation tests in tests/security/json_test.go

### Implementation for User Story 3

- [ ] T032 [P] [US3] Create PersonaReference struct in internal/security/types.go
- [ ] T033 [P] [US3] Implement persona enumeration in internal/manifest/loader.go
- [ ] T034 [US3] Add persona validation to buildPhilosopherPrompt in internal/pipeline/meta.go
- [ ] T035 [US3] Inject available personas into philosopher prompt generation
- [ ] T036 [US3] Add pipeline YAML validation against manifest personas
- [ ] T037 [US3] Enhance JSON comment prevention in persona prompts
- [ ] T038 [US3] Add persona validation to pipeline execution startup

**Checkpoint**: Meta-pipeline generation should work reliably with valid persona references

---

## Phase 6: User Story 4 - Reliable Contract Validation (Priority: P2)

**Goal**: Fix contract validation to handle malformed JSON and respect must_pass settings

**Independent Test**: Run pipelines with JSON contracts and verify validation behavior in strict and soft modes

### Tests for User Story 4

- [ ] T039 [P] [US4] JSON comment cleaning tests in tests/contract/json_test.go
- [ ] T040 [P] [US4] must_pass handling tests in tests/contract/validation_test.go
- [ ] T041 [P] [US4] Contract retry logic tests in tests/contract/retry_test.go

### Implementation for User Story 4

- [x] T042 [P] [US4] Implement JSON comment stripping in internal/contract/jsonschema.go
- [x] T043 [P] [US4] Fix must_pass validation logic in internal/contract/jsonschema.go
- [x] T044 [US4] Add JSON linting with error recovery to contract validation
- [ ] T045 [US4] Enhance contract failure event emission
- [x] T046 [US4] Add strict/soft mode configuration to contract types
- [x] T047 [US4] Improve contract error messages and logging

**Checkpoint**: Contract validation should handle all JSON formats and respect configuration properly

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Security improvements that affect multiple user stories

- [ ] T048 [P] Comprehensive security documentation in docs/security/
- [ ] T049 [P] Security configuration examples in docs/examples/
- [ ] T050 [P] Performance optimization for security validation overhead
- [ ] T051 [P] Security audit logging configuration documentation
- [ ] T052 Integration testing across all security features
- [ ] T053 Run quickstart.md security validation scenarios
- [ ] T054 [P] Security compliance validation checklist
- [ ] T055 Final security penetration testing

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-6)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P1 → P2 → P2)
- **Polish (Phase 7)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational - No dependencies on other stories
- **User Story 2 (P1)**: Can start after Foundational - No dependencies on other stories
- **User Story 3 (P2)**: Can start after Foundational - No dependencies on other stories
- **User Story 4 (P2)**: Can start after Foundational - No dependencies on other stories

### Within Each User Story

- Security tests MUST be written and FAIL before implementation
- Core security functions before integration points
- Configuration before implementation
- Logging integration after core functionality
- Story validation before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- Once Foundational phase completes, all user stories can start in parallel (if team capacity allows)
- All security tests for a user story marked [P] can run in parallel
- Security utility functions within a story marked [P] can run in parallel
- Different user stories can be worked on in parallel by different team members

---

## Parallel Example: User Story 1

```bash
# Launch all security tests for User Story 1 together:
Task: "Path traversal attack tests in tests/security/path_test.go"
Task: "Schema content sanitization tests in tests/security/schema_test.go"
Task: "Security logging validation tests in tests/security/logging_test.go"

# Launch all security utilities for User Story 1 together:
Task: "Implement path validation functions in internal/security/path.go"
Task: "Create SchemaValidationResult struct in internal/security/types.go"
```

---

## Implementation Strategy

### MVP First (Critical Security - User Stories 1 & 2)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (Path traversal fixes)
4. Complete Phase 4: User Story 2 (Input sanitization)
5. **STOP and VALIDATE**: Test critical security fixes independently
6. Deploy security patches immediately

### Full Security Enhancement

1. Complete Setup + Foundational → Security foundation ready
2. Add User Story 1 → Test path traversal prevention → Validate independently
3. Add User Story 2 → Test input sanitization → Validate independently
4. Add User Story 3 → Test meta-pipeline fixes → Validate independently
5. Add User Story 4 → Test contract validation → Validate independently
6. Each story adds security without breaking previous functionality

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (Path traversal)
   - Developer B: User Story 2 (Input validation)
   - Developer C: User Story 3 (Meta-pipeline)
   - Developer D: User Story 4 (Contract validation)
3. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific security vulnerability for traceability
- Each user story should be independently completable and testable
- Verify security tests fail before implementing protections
- Commit after each security fix or logical group
- Stop at any checkpoint to validate security improvements independently
- Focus: critical vulnerabilities first (US1, US2), then functionality fixes (US3, US4)
- All security fixes must maintain backward compatibility