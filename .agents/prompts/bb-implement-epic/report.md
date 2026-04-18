You are posting a summary report on a parent epic issue after its subissues have been implemented.

Input: {{ input }}

The input format is `workspace/repo number` (e.g. `re-cinq/wave 184`).

## Available Artifacts

- `scope_plan`: The epic scope plan with subissue list and dependencies (from fetch-scope step)

## Working Directory

You are running in an isolated Wave workspace. Bitbucket API calls via `curl`
work from any directory, so no directory change is needed.

## Instructions

### Step 1: Parse Input and Read Scope Plan

Extract the workspace, repository slug, and epic number from the input. Read the scope plan artifact to get the list of subissues that were targeted for implementation.

### Step 2: Check Implementation Status

For each subissue in the scope plan, check whether a pull request was created. Search for PRs that reference the subissue:

```bash
curl -s "https://api.bitbucket.org/2.0/repositories/<WORKSPACE>/<REPO>/pullrequests?q=title+%7E+%22%23<SUBISSUE_NUMBER>%22+OR+description+%7E+%22closes+%23<SUBISSUE_NUMBER>%22+OR+description+%7E+%22related+to+%23<SUBISSUE_NUMBER>%22&pagelen=5" \
  -H "Authorization: Bearer $BB_TOKEN" | jq '.values[] | {id: .id, title: .title, state: .state, links: .links}'
```

Also check if the subissue itself was resolved:

```bash
curl -s "https://api.bitbucket.org/2.0/repositories/<WORKSPACE>/<REPO>/issues/<SUBISSUE_NUMBER>" \
  -H "Authorization: Bearer $BB_TOKEN" | jq '{state: .state}'
```

Classify each subissue as:
- **implemented**: A PR exists (open or merged) that closes the subissue
- **failed**: No PR found and the subissue is still open (implementation was attempted but failed)
- **skipped**: Subissue was skipped due to dependency failure

### Step 3: Build Summary

Compile the results into a structured summary with:
- Total subissues targeted
- Count of implemented, failed, and skipped
- PR URLs for successful implementations

### Step 4: Post Comment on Epic

Post a summary comment on the parent epic issue:

```bash
curl -s -X POST \
  "https://api.bitbucket.org/2.0/repositories/<WORKSPACE>/<REPO>/issues/<EPIC_NUMBER>/comments" \
  -H "Authorization: Bearer $BB_TOKEN" \
  -H "Content-Type: application/json" \
  -d "$(cat <<'COMMENT'
{
  "content": {
    "raw": "## Implementation Summary\n\n**Pipeline**: bb-implement-epic\n**Status**: X/Y subissues implemented\n\n### Results\n\n| Subissue | Title | Status | PR |\n|----------|-------|--------|-----|\n| #206 | Title | ✅ Implemented | !250 |\n| #207 | Title | ❌ Failed | - |\n| #208 | Title | ⏭ Skipped | - |\n\n### Details\n- Implemented: X\n- Failed: Y\n- Skipped: Z"
  }
}
COMMENT
)"
```

Note: Bitbucket uses `!<NUMBER>` notation for pull request references in markup.

### Step 5: Output

Write the result to `.wave/output/epic-report.json`:

```json
{
  "parent_issue": {
    "owner": "re-cinq",
    "repo": "wave",
    "number": 184,
    "url": "https://bitbucket.org/re-cinq/wave/issues/184"
  },
  "results": [
    {
      "number": 206,
      "title": "Subissue title",
      "status": "implemented",
      "pr_url": "https://bitbucket.org/re-cinq/wave/pull-requests/250",
      "pr_number": 250
    }
  ],
  "summary": {
    "total_subissues": 6,
    "implemented": 4,
    "failed": 1,
    "skipped": 1,
    "comment_posted": true,
    "comment_url": "https://bitbucket.org/re-cinq/wave/issues/184#comment-12345"
  }
}
```

## CRITICAL

- Always post a comment on the epic, even if all subissues failed
- Include PR links for all successfully implemented subissues
- Do NOT close the epic issue — leave that for manual review
