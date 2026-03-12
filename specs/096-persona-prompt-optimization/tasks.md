# Tasks

## Phase 1: Audit & Baseline

- [X] Task 1.1: Run existing tests to establish baseline (`go test ./internal/defaults/... -v`)
- [X] Task 1.2: Audit base-protocol.md for completeness -- does it cover all shared concerns? Document any gaps
- [X] Task 1.3: Audit all 30 personas to catalog low-signal content types present (generic process, duplicated Wave protocol, Communication Style, Domain Expertise sections)

## Phase 2: Core Persona Optimization (17 personas)

Each task edits ONE persona in BOTH `internal/defaults/personas/` AND `.wave/personas/`.

- [X] Task 2.1: Optimize navigator.md -- verify Anti-Patterns and Quality Checklist earn their tokens, remove any base-protocol duplication [P]
- [X] Task 2.2: Optimize implementer.md -- already lean, verify meets 100-token minimum [P]
- [X] Task 2.3: Optimize reviewer.md -- preserve Ontology-vs-Code section, remove any generic process content [P]
- [X] Task 2.4: Optimize planner.md -- preserve Scope Boundary, check for low-signal content [P]
- [X] Task 2.5: Optimize researcher.md -- preserve Composition Pipeline section, check for generic content [P]
- [X] Task 2.6: Optimize debugger.md -- remove any base-protocol duplication, tighten Anti-Patterns [P]
- [X] Task 2.7: Optimize auditor.md -- ensure clear differentiation from reviewer persona [P]
- [X] Task 2.8: Optimize craftsman.md -- tighten Guidelines/Anti-Patterns, remove any generic advice [P]
- [X] Task 2.9: Optimize summarizer.md -- verify Anti-Patterns add value beyond Constraints [P]
- [X] Task 2.10: Optimize github-analyst.md -- already lean, verify consistency [P]
- [X] Task 2.11: Optimize github-commenter.md -- preserve CLI syntax examples (high-signal) [P]
- [X] Task 2.12: Optimize github-enhancer.md -- already lean, verify consistency [P]
- [X] Task 2.13: Optimize github-scoper.md -- preserve Decomposition Guidelines and Template [P]
- [X] Task 2.14: Optimize philosopher.md -- preserve Ontology Extraction, check Scope Boundary against planner [P]
- [X] Task 2.15: Optimize provocateur.md -- preserve Thinking Style and Evidence Gathering [P]
- [X] Task 2.16: Optimize validator.md -- already well-optimized, minor verification [P]
- [X] Task 2.17: Optimize synthesizer.md -- preserve JSON output emphasis and Ontology Evolution [P]
- [X] Task 2.18: Optimize supervisor.md -- verify Evidence Gathering and Evaluation Criteria [P]

## Phase 3: Forge-Specific Persona Optimization (13 personas)

Each task optimizes ONE forge family (analyst + commenter + enhancer + scoper for that forge).

- [X] Task 3.1: Optimize gitea-analyst.md, gitea-commenter.md, gitea-enhancer.md, gitea-scoper.md -- ensure consistency with github-* family [P]
- [X] Task 3.2: Optimize gitlab-analyst.md, gitlab-commenter.md, gitlab-enhancer.md, gitlab-scoper.md -- ensure consistency with github-* family [P]
- [X] Task 3.3: Optimize bitbucket-analyst.md, bitbucket-commenter.md, bitbucket-enhancer.md, bitbucket-scoper.md -- preserve API-specific content [P]

## Phase 4: Testing & Validation

- [X] Task 4.1: Run `go test ./internal/defaults/... -v` and verify all tests pass
- [X] Task 4.2: Run `go test -race ./...` to catch any regressions across the full test suite
- [X] Task 4.3: Spot-check signal density on 3-4 optimized personas (navigator, craftsman, provocateur, bitbucket-commenter)
- [X] Task 4.4: Verify no base-protocol.md content is duplicated in any persona file
- [X] Task 4.5: Final parity check -- diff `internal/defaults/personas/` vs `.wave/personas/` to confirm byte-identical
