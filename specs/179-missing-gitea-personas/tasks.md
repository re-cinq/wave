# Tasks

## Phase 1: Core Fix

- [X] Task 1.1: Add `gitea-analyst` persona entry to `createDefaultManifest()` in `cmd/wave/commands/init.go` — adapter, description "Gitea issue analysis and scanning", system_prompt_file `.wave/personas/gitea-analyst.md`, temperature 0.1, permissions `Read`, `Write`, `Bash(tea *)` [P]
- [X] Task 1.2: Add `gitea-enhancer` persona entry — description "Gitea issue enhancement and improvement", temperature 0.2, permissions `Read`, `Write`, `Bash(tea *)` [P]
- [X] Task 1.3: Add `gitea-commenter` persona entry — description "Posts comments on Gitea issues", temperature 0.2, permissions `Read`, `Bash(tea *)` [P]
- [X] Task 1.4: Add `gitlab-analyst` persona entry — description "GitLab issue analysis and scanning", temperature 0.1, permissions `Read`, `Write`, `Bash(glab *)` [P]
- [X] Task 1.5: Add `gitlab-enhancer` persona entry — description "GitLab issue enhancement and improvement", temperature 0.2, permissions `Read`, `Write`, `Bash(glab *)` [P]
- [X] Task 1.6: Add `gitlab-commenter` persona entry — description "Posts comments on GitLab issues", temperature 0.2, permissions `Read`, `Bash(glab *)` [P]
- [X] Task 1.7: Add `bitbucket-analyst` persona entry — description "Bitbucket issue analysis and scanning", temperature 0.1, permissions `Read`, `Write`, `Bash(bb *)` [P]
- [X] Task 1.8: Add `bitbucket-enhancer` persona entry — description "Bitbucket issue enhancement and improvement", temperature 0.2, permissions `Read`, `Write`, `Bash(bb *)` [P]
- [X] Task 1.9: Add `bitbucket-commenter` persona entry — description "Posts comments on Bitbucket issues", temperature 0.2, permissions `Read`, `Bash(bb *)` [P]
- [X] Task 1.10: Add `supervisor` persona entry — description "Work supervision and quality evaluation", temperature 0.1, permissions `Read`, `Glob`, `Grep`, `Bash(git *)`, `Bash(go test *)`, deny `Write(*)`, `Edit(*)`

## Phase 2: Testing

- [X] Task 2.1: Run `go test ./cmd/wave/commands/` to verify init tests pass
- [X] Task 2.2: Run `go test ./...` to verify no project-wide regressions
- [X] Task 2.3: Verify `TestInitPersonasNeverExcluded` passes with correct persona count
- [X] Task 2.4: Verify `TestInitOutputValidatesWithWaveValidate` passes

## Phase 3: Validation

- [X] Task 3.1: Cross-check every bundled pipeline's persona references against the manifest entries to confirm complete coverage
