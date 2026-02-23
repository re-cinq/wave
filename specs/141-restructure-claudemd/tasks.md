# Tasks

## Phase 1: Audit and Analysis

- [X] Task 1.1: Record baseline line count of CLAUDE.md (285 lines)
- [X] Task 1.2: Verify no tests reference CLAUDE.md content directly (grep for CLAUDE.md in test files)

## Phase 2: Core Restructure

- [X] Task 2.1: Create new "Critical Constraints" section at top of file by extracting and consolidating from "Critical Constraints", "Test Ownership", and "Constitutional Compliance" sections
- [X] Task 2.2: Write new "How Wave Works at Runtime" section (~20 lines) covering pipeline execution, workspace isolation, contract validation, artifact injection, and runtime CLAUDE.md assembly [P]
- [X] Task 2.3: Condense "Security Considerations" section — collapse four sub-sections (Input Validation, Permission Enforcement, Sandbox Isolation, Data Protection) into a single compact list [P]
- [X] Task 2.4: Remove "Common Tasks" section entirely — replace with one-line reference to source directories [P]
- [X] Task 2.5: Remove "Performance Considerations" section — generic and not actionable [P]
- [X] Task 2.6: Condense "Database Migrations" section to single line referencing `docs/migrations.md` [P]
- [X] Task 2.7: Condense "Testing" section — keep only essential commands (`go test ./...`, `go test -race ./...`) [P]
- [X] Task 2.8: Condense "Versioning" section — keep the commit prefix table, remove surrounding prose [P]
- [X] Task 2.9: Merge "Key Implementation Patterns" into the new runtime section (Task 2.2) [P]

## Phase 3: Polish and Verify

- [X] Task 3.1: Verify `<!-- MANUAL ADDITIONS START -->` and `<!-- MANUAL ADDITIONS END -->` markers are preserved with their content
- [X] Task 3.2: Verify `Recent Changes` section is preserved as-is
- [X] Task 3.3: Verify final line count is ≤200 lines (30%+ reduction from 285)
- [X] Task 3.4: Verify all section headings are clear and scannable

## Phase 4: Validation

- [X] Task 4.1: Run `go test ./...` to confirm no test regressions
- [X] Task 4.2: Run `wc -l CLAUDE.md` to confirm line count target met
- [X] Task 4.3: Review final document for readability and completeness
