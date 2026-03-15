You are posting a summary report on a parent epic issue after its subissues have been implemented.

Input: {{ input }}

The input format is `owner/repo number` (e.g. `re-cinq/wave 184`).

## Available Artifacts

- `scope_plan`: The epic scope plan with subissue list and dependencies (from fetch-scope step)

## Working Directory

You are running in an isolated Wave workspace. The `glab` CLI works from any
directory when using the `--repo` flag, so no directory change is needed.

## Instructions

### Step 1: Parse Input and Read Scope Plan

Extract the repository and epic number from the input. Read the scope plan artifact to get the list of subissues that were targeted for implementation.

### Step 2: Check Implementation Status

For each subissue in the scope plan, check whether a merge request was created:

```bash
glab mr list --repo <OWNER/REPO> --search "Closes #<SUBISSUE_NUMBER> OR Related to #<SUBISSUE_NUMBER>" --output json --per-page 5
```

Also check if the subissue itself was closed:

```bash
glab issue view <SUBISSUE_NUMBER> --repo <OWNER/REPO> --output json
```

Classify each subissue as:
- **implemented**: An MR exists (open or merged) that closes the subissue
- **failed**: No MR found and the subissue is still open (implementation was attempted but failed)
- **skipped**: Subissue was skipped due to dependency failure

### Step 3: Build Summary

Compile the results into a structured summary with:
- Total subissues targeted
- Count of implemented, failed, and skipped
- MR URLs for successful implementations

### Step 4: Post Comment on Epic

Post a summary comment on the parent epic:

```bash
glab issue comment <EPIC_NUMBER> --repo <OWNER/REPO> --message "$(cat <<'COMMENT'
## Implementation Summary

**Pipeline**: gl-implement-epic
**Status**: X/Y subissues implemented

### Results

| Subissue | Title | Status | MR |
|----------|-------|--------|-----|
| #206 | Title | ✅ Implemented | !250 |
| #207 | Title | ❌ Failed | - |
| #208 | Title | ⏭ Skipped | - |

### Details
- Implemented: X
- Failed: Y
- Skipped: Z
COMMENT
)"
```

### Step 5: Output

Write the result to `.wave/output/epic-report.json`:

```json
{
  "parent_issue": {
    "owner": "re-cinq",
    "repo": "wave",
    "number": 184,
    "url": "https://gitlab.com/re-cinq/wave/-/issues/184"
  },
  "results": [
    {
      "number": 206,
      "title": "Subissue title",
      "status": "implemented",
      "pr_url": "https://gitlab.com/re-cinq/wave/-/merge_requests/250",
      "pr_number": 250
    }
  ],
  "summary": {
    "total_subissues": 6,
    "implemented": 4,
    "failed": 1,
    "skipped": 1,
    "comment_posted": true,
    "comment_url": "https://gitlab.com/re-cinq/wave/-/issues/184#note_12345"
  }
}
```

## CRITICAL

- Always post a comment on the epic, even if all subissues failed
- Include MR links for all successfully implemented subissues
- Do NOT close the epic issue — leave that for manual review
