You are fetching an issue and assessing whether it has enough detail to implement.

Input: {{ input }}

The input format is `owner/repo number` (e.g. `re-cinq/wave 42`).

## Working Directory

You are running in an isolated Wave workspace. The `{{ forge.cli_tool }}` CLI works from any
directory when using the `--repo` flag, so no directory change is needed.

## Instructions

### Step 1: Parse Input

Extract the repository (`owner/repo`) and issue number from the input string.

### Step 2: Fetch Issue

Use the appropriate CLI for your platform ({{ forge.type }}) to fetch the issue with full details.

**For GitHub** (`gh`):
```bash
gh issue view <NUMBER> --repo <OWNER/REPO> --json number,title,body,url,labels,state,author,comments
```

**For GitLab** (`glab`):
```bash
glab issue view <NUMBER> --repo <OWNER/REPO> --output json
```

**For Bitbucket** (REST API):
```bash
curl -s -H "Authorization: Bearer $BB_TOKEN" \
  "https://api.bitbucket.org/2.0/repositories/WORKSPACE/REPO/issues/NUMBER" \
  | jq '{id, title, content: .content.raw, state, kind, reporter: .reporter.display_name, created_on, url: .links.html.href}'
```
Also fetch comments:
```bash
curl -s -H "Authorization: Bearer $BB_TOKEN" \
  "https://api.bitbucket.org/2.0/repositories/WORKSPACE/REPO/issues/NUMBER/comments" \
  | jq '[.values[] | {id, content: .content.raw, user: .user.display_name, created_on}]'
```

**For Gitea** (`tea`):
```bash
tea issues view <NUMBER> --repo <OWNER/REPO> --output json
```

### Step 3: Assess Implementability

Evaluate the issue against these criteria:

1. **Clear description**: Does the issue describe what needs to change? (not just "X is broken")
2. **Sufficient context**: Can you identify which code/files are affected?
3. **Testable outcome**: Are there acceptance criteria, or can you infer them from the description?

Score the issue 0-100:
- **80-100**: Well-specified, clear requirements, acceptance criteria present
- **60-79**: Adequate detail, some inference needed but feasible
- **40-59**: Marginal — missing key details but core intent is clear
- **0-39**: Too vague to implement — set `implementable` to `false`

### Step 4: Determine Skip Steps

Based on the issue quality, decide which speckit steps can be skipped:
- Issues with detailed specs can skip `specify`, `clarify`, `checklist`, `analyze`
- Issues with moderate detail might skip `specify` and `clarify` only
- Vague issues should skip nothing (but those should fail the assessment)

### Step 5: Generate Branch Name

Create a branch name using the pattern `<NNN>-<short-name>` where:
- `<NNN>` is the issue number zero-padded to 3 digits
- `<short-name>` is 2-3 words from the issue title, kebab-case

### Step 6: Assess Complexity

Estimate implementation complexity:
- **trivial**: Single file change, obvious fix (typo, config tweak)
- **simple**: 1-3 files, straightforward logic change
- **medium**: 3-10 files, new feature with tests
- **complex**: 10+ files, architectural changes, cross-cutting concerns

## CRITICAL: Implementability Gate

If the issue does NOT have enough detail to implement:
- Set `"implementable": false` in the output
- This will cause the contract validation to fail, aborting the pipeline
- Include `missing_info` listing what specific information is needed
- Include a `summary` explaining why the issue cannot be implemented as-is

If the issue IS implementable:
- Set `"implementable": true`

## CONSTRAINTS

- Do NOT spawn Task subagents — work directly in the main context
- Do NOT modify the issue — this is read-only assessment

## OUTPUT

Write your assessment to `.wave/output/issue-assessment.json` in this format:

```json
{
  "implementable": true/false,
  "issue": {
    "number": 0,
    "title": "",
    "body": "",
    "repository": "",
    "url": "",
    "labels": [],
    "state": ""
  },
  "assessment": {
    "score": 0,
    "clarity": 0,
    "context": 0,
    "testability": 0,
    "missing_info": [],
    "summary": ""
  },
  "complexity": "trivial/simple/medium/complex",
  "skip_steps": [],
  "branch_name": ""
}
```

Create the `.wave/output/` directory if it doesn't exist.
