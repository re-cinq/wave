You are creating a feature specification for the following request:

{{ input }}

## Working Directory

You are running in an **isolated git worktree** checked out at `main` (detached HEAD).
Your working directory IS the project root. All git operations here are isolated
from the main working tree and will not affect it.

Use `create-new-feature.sh` to create the feature branch from this clean starting point.

## Issue / PR Resolution

**Before doing anything else**, check whether the input is a forge issue or PR URL:

```
https://<host>/<owner>/<repo>/issues/<N>       (GitHub, Gitea)
https://<host>/<owner>/<repo>/-/issues/<N>     (GitLab)
https://<host>/<owner>/<repo>/pull/<N>         (GitHub PR)
https://<host>/<owner>/<repo>/pulls/<N>        (Gitea PR)
https://<host>/<owner>/<repo>/-/merge_requests/<N> (GitLab MR)
```

If matched, fetch the full issue/PR **including all comments** using `{{ forge.cli_tool }}`
(current forge: `{{ forge.type }}`):

**For GitHub** (`gh`):
```bash
gh issue view <N> --repo <owner>/<repo> --json number,title,body,url,state,author,comments
# for PRs:
gh pr view <N> --repo <owner>/<repo> --json number,title,body,url,state,author,comments
```

**For GitLab** (`glab`):
```bash
glab issue view <N> --repo <owner>/<repo>
glab issue note list <N> --repo <owner>/<repo>
```

**For Gitea** (`tea`):
```bash
tea issues view <N> --repo <owner>/<repo>
tea issues comments <N> --repo <owner>/<repo>
```

Then scan the body and all comments for referenced issues/PRs:
- Inline: `#123`, `owner/repo#123`
- Full URLs matching the same forge host
- Closing keywords: `closes #N`, `fixes #N`, `resolves #N`

Fetch each unique referenced issue/PR (1 level deep, max 5) with the same tool.

Treat the **combined content** — issue body + all comments + all referenced issue bodies — as
the feature description for all subsequent steps.

If the input is **not** a forge URL, proceed with the raw input text as the feature description.

## Instructions

Follow the `/speckit.specify` workflow to generate a complete feature specification:

1. Generate a concise short name (2-4 words) for the feature branch
2. Check existing branches to determine the next available number:
   ```bash
   git fetch --all --prune
   git ls-remote --heads origin | grep -E 'refs/heads/[0-9]+-'
   git branch | grep -E '^[* ]*[0-9]+-'
   ```
3. Run the feature creation script:
   ```bash
   .specify/scripts/bash/create-new-feature.sh --json --number <N> --short-name "<name>" "{{ input }}"
   ```
4. Load `.specify/templates/spec-template.md` for the required structure
5. Write the specification to the SPEC_FILE returned by the script
6. Create the quality checklist at `FEATURE_DIR/checklists/requirements.md`
7. Run self-validation against the checklist (up to 3 iterations)

## Parallel Exploration

Sequence these activities to build the spec efficiently:
- Step 1: Analyze the codebase to understand existing patterns and architecture
- Step 2: Survey domain-specific best practices for the feature
- Step 3: Draft specification sections grounded in the gathered context

## Quality Standards

- Focus on WHAT and WHY, not HOW (no implementation details)
- Every requirement must be testable and unambiguous
- Maximum 3 `[NEEDS CLARIFICATION]` markers — make informed guesses for the rest
- Include user stories with acceptance criteria, data model, edge cases
- Success criteria must be measurable and technology-agnostic

## Output

Produce a JSON status report matching the injected output schema.
