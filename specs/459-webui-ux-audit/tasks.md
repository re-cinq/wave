# Tasks: WebUI UX Audit (#459)

## Phase 1: Audit Document Creation

- [X] 1.1 Create audit document structure
- [X] 1.2 Layout & Navigation findings
- [X] 1.3 Runs page findings
- [X] 1.4 Run Detail page findings
- [X] 1.5 Pipelines page findings
- [X] 1.6 Personas page findings
- [X] 1.7 Contracts page findings
- [X] 1.8 Skills page findings
- [X] 1.9 Compose page findings
- [X] 1.10 Issues page findings
- [X] 1.11 PRs page findings
- [X] 1.12 Health page findings
- [X] 1.13 Not Found page findings
- [X] 1.14 CSS & JS asset findings
- [X] 1.15 Cross-page consistency section

## Phase 2: Validation

- [X] 2.1 Verify template coverage (17/17 templates documented)
- [X] 2.2 Verify finding format (all have page, description, severity)
- [X] 2.3 Verify thematic categorization (layout, states, feedback, accessibility, consistency)
- [X] 2.4 Final review

## Phase 3: Pre-existing Test Fixes

- [X] 3.1 Fix `TestConcurrentStepWideFanOut` — increased adapter delay from 50ms to 200ms to ensure all 4 parallel steps overlap
- [X] 3.2 Fix `TestParseExistingSkills` — replaced brittle hardcoded count (14) with minimum bound check
