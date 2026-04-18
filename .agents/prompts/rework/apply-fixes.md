Apply review fixes for PR: {{ input }}

## Context

The `triage` artifact contains the triaged review verdict — only **accepted** findings that passed triage.
The `raw_findings` artifact contains the original review data including the PR head branch.

## Step 1: Checkout the PR branch

```bash
HEAD_BRANCH=$(cat .wave/artifacts/triage | jq -r '.head_branch')
git fetch origin "$HEAD_BRANCH"
git checkout "$HEAD_BRANCH"
```

## Step 2: Review the triage verdict

Read `.wave/artifacts/triage` and understand:
- How many fixes are accepted (`fixes` array)
- Their severity and priority order
- The `action_detail` for each — this is your instruction
- The `blast_radius` and `confidence` — proceed carefully on high-blast/low-confidence fixes

## Step 3: Apply fixes in priority order

1. **Critical findings first** — these block the merge
2. **Major findings second** — these should all be resolved
3. **Minor findings last** — fix if straightforward

For each fix:
- Read the current file at the specified path and line
- Apply the **minimal** change that addresses the `action_detail`
- Don't refactor or "improve" surrounding code
- Don't fix things that aren't in the triage verdict

## Step 4: Run tests

```bash
{{ project.contract_test_command }}
```

Fix any test failures introduced by your changes.

## Step 5: Run linter

```bash
golangci-lint run ./... 2>&1 | head -50
```

Fix any lint issues in files you modified.

## Step 6: Commit and push

```bash
git add -A
git diff --cached --stat
git commit -m "fix: address review findings

$(cat .wave/artifacts/triage | jq -r '.fixes[] | "- [\(.severity)] \(.summary)"' | head -10)"

git push origin "$HEAD_BRANCH"
```

## Constraints

- Do NOT force push
- Do NOT rebase or squash — add a new commit
- Do NOT modify files outside the triage verdict
- If a fix would break something, skip it and note why in the commit message
