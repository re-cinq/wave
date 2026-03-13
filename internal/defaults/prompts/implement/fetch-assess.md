# Fetch and Assess Issue

You are working on a **{{ forge.type }}** repository. Use the **{{ forge.cli_tool }}** CLI tool.

## Input

{{ input }}

## Parse Input

Accepted input formats:
- Full URL (e.g., https://{{ forge.host }}/owner/repo/issues/123)
- Short form: owner/repo 123

Extract OWNER/REPO and ISSUE_NUMBER from the input.

## Fetch Issue

### For GitHub (forge.type = github):
```
gh issue view <NUMBER> --repo <OWNER/REPO> --json number,title,body,labels,state,author,createdAt,url,comments
```

### For GitLab (forge.type = gitlab):
```
glab issue view <NUMBER> --repo <OWNER/REPO> --output json
```

### For Bitbucket (forge.type = bitbucket):
```
curl -s -H "Authorization: Bearer $BB_TOKEN" \
  "https://api.bitbucket.org/2.0/repositories/<OWNER>/<REPO>/issues/<NUMBER>"
```
Also fetch comments:
```
curl -s -H "Authorization: Bearer $BB_TOKEN" \
  "https://api.bitbucket.org/2.0/repositories/<OWNER>/<REPO>/issues/<NUMBER>/comments"
```

### For Gitea (forge.type = gitea):
```
tea issues view <NUMBER> --repo <OWNER/REPO> --output json
```

## Assess Implementability

After fetching, assess the issue:

1. **Quality Score** (0-100): Rate on title clarity, description completeness, acceptance criteria, and testability.
   - 80-100: Well-specified, proceed with implementation
   - 60-79: Adequate, may need minor assumptions
   - 40-59: Marginal, significant assumptions needed
   - 0-39: Insufficient, set implementable to false

2. **Implementability Decision**:
   - Set `implementable: true` if quality_score >= 40 AND the issue has clear intent
   - Set `implementable: false` if quality_score < 40 OR missing critical information
   - If not implementable, provide `missing_info` listing what's needed

3. **Branch Name**: Generate as `<NNN>-<short-name>` where NNN is the issue number (zero-padded to 3 digits) and short-name is a kebab-case summary (max 40 chars).

4. **Complexity**: Assess as one of: trivial, simple, medium, complex

5. **Skip Steps**: Determine which pipeline steps to skip (if any) based on the issue.

## CRITICAL: Implementability Gate

If the issue does NOT have enough detail to implement:
- Set `"implementable": false` in the output
- This will cause the contract validation to fail, aborting the pipeline
- Include `missing_info` listing what specific information is needed
- Include a `summary` explaining why the issue cannot be implemented as-is

If the issue IS implementable:
- Set `"implementable": true`

## CONSTRAINTS

- Do NOT spawn Task subagents -- work directly in the main context
- Do NOT modify the issue -- this is read-only assessment

## Output

Write a JSON file to `.wave/output/issue-assessment.json` matching the contract schema with fields:
- issue_number, title, body_summary, repository (owner, name)
- quality_score, implementable, missing_info (if applicable)
- branch_name, complexity, labels, skip_steps
- url (the issue URL)
