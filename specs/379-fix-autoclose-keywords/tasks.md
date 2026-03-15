# Tasks

## Phase 1: Core Fix — Replace Closing Keywords in Unified Prompts

- [X] Task 1.1: Update `internal/defaults/prompts/implement/create-pr.md` — replace all `Closes #<NUMBER>` / `Closes #<ISSUE_NUMBER>` with `Related to #<NUMBER>` / `Related to #<ISSUE_NUMBER>` in all four forge template examples (GitHub, GitLab, Bitbucket, Gitea) and in the CONSTRAINTS section
- [X] Task 1.2: Update `internal/contract/format_validator.go` — change the issue reference regex at line 168 to also accept `Related to #N` pattern: `(?i)(closes|fixes|resolves|related\s+to)\s+#\d+`; update violation message at line 169 to say `"PR body should reference related issues (Related to #123 or Closes #123)"`
- [X] Task 1.3: Update `internal/contract/format_validator_test.go` — change the test case at line 144 from `Closes #123` to `Related to #123` to verify the new default pattern works

## Phase 2: Deprecated Forge-Specific Prompts [P]

- [X] Task 2.1: Update `.wave/prompts/gh-implement/create-pr.md` — replace `Closes #<NUMBER>` with `Related to #<NUMBER>` in PR body template and CONSTRAINTS section [P]
- [X] Task 2.2: Update `.wave/prompts/gl-implement/create-mr.md` — replace `Closes #<NUMBER>` / `Closes #<ISSUE_NUMBER>` with `Related to` equivalents in MR body template and CONSTRAINTS section [P]
- [X] Task 2.3: Update `.wave/prompts/gt-implement/create-pr.md` — replace `Closes #<NUMBER>` / `Closes #<ISSUE_NUMBER>` with `Related to` equivalents in PR body template and CONSTRAINTS section [P]
- [X] Task 2.4: Update `.wave/prompts/bb-implement/create-pr.md` — replace `Closes #NUMBER` / `Closes #<NUMBER>` with `Related to` equivalents in PR body template and CONSTRAINTS section [P]

## Phase 3: Epic Report Search Patterns [P]

- [X] Task 3.1: Update `.wave/prompts/gh-implement-epic/report.md` — change `gh pr list --search "Closes #<SUBISSUE_NUMBER>"` to search for both old and new patterns: `"Closes #<N> OR Related to #<N>"` [P]
- [X] Task 3.2: Update `.wave/prompts/gl-implement-epic/report.md` — change `glab mr list --search "Closes #<SUBISSUE_NUMBER>"` to search for both patterns [P]
- [X] Task 3.3: Update `.wave/prompts/gt-implement-epic/report.md` — update `tea` search/jq filter to match both `Closes` and `Related to` patterns [P]
- [X] Task 3.4: Update `.wave/prompts/bb-implement-epic/report.md` — update Bitbucket API search query to match both `closes` and `related+to` patterns [P]

## Phase 4: Audit Pipeline Backward Compatibility [P]

- [X] Task 4.1: Update `internal/defaults/pipelines/wave-audit.yaml` — add `"Related to #N"` alongside existing `"Fixes #N", "Closes #N"` in the linked_prs search pattern description [P]
- [X] Task 4.2: Update `.wave/pipelines/wave-audit.yaml` — same change as above for the local override copy [P]

## Phase 5: Testing and Validation

- [X] Task 5.1: Run `go test -race ./...` to verify no test regressions
- [X] Task 5.2: Run grep audit to confirm no remaining closing keywords in generation contexts (only in search/validator contexts)
