You are parsing the scope output from a previously-run `gt-scope` pipeline to extract the list of subissues and their dependency graph.

Input: {{ input }}

The input format is `owner/repo number` (e.g. `re-cinq/wave 184`) where the number is the **parent epic** issue.

## Working Directory

You are running in an isolated Wave workspace. The `tea` CLI works from any
directory when using the `--repo` flag, so no directory change is needed.

## Instructions

### Step 1: Parse Input

Extract the repository (`owner/repo`) and epic issue number from the input string.

### Step 2: Fetch Epic Issue and Comments

Use the `tea` CLI to fetch the epic issue with its comments:

```bash
tea issues view <NUMBER> --repo <OWNER/REPO> --output json --comments
```

### Step 3: Find Scope Summary Comment

Search through the issue comments for the scope summary posted by `gt-scope`. The scope comment contains a structured list of subissues with:
- Issue numbers and titles
- Dependency relationships (e.g., "depends on #206")
- Complexity ratings (S/M/L/XL)

Look for a comment containing a section like "## Scope Summary" or a table/list of created subissues with issue numbers, URLs, and dependency information.

### Step 4: Fetch Each Subissue

For each subissue referenced in the scope comment:

```bash
tea issues view <SUBISSUE_NUMBER> --repo <OWNER/REPO> --output json
```

Verify each subissue:
- Exists and is accessible
- Record its current state (open or closed)
- Extract any dependency references from its body (look for "depends on #X" or "blocked by #X" patterns)

### Step 5: Build Dependency Graph

Construct the dependency graph using integer issue numbers:
- Parse explicit dependency declarations from issue bodies and the scope comment
- Ensure all referenced dependencies exist in the subissue set
- Dependencies should reference issue numbers (integers), not titles

### Step 6: Output

Write the result to `.wave/output/epic-scope-plan.json` with this structure:

```json
{
  "parent_issue": {
    "owner": "re-cinq",
    "repo": "wave",
    "number": 184,
    "title": "Epic title",
    "url": "https://gitea.example.com/re-cinq/wave/issues/184"
  },
  "subissues": [
    {
      "number": 206,
      "repository": "re-cinq/wave",
      "title": "Subissue title",
      "url": "https://gitea.example.com/re-cinq/wave/issues/206",
      "state": "open",
      "complexity": "M",
      "dependencies": []
    },
    {
      "number": 207,
      "repository": "re-cinq/wave",
      "title": "Another subissue",
      "url": "https://gitea.example.com/re-cinq/wave/issues/207",
      "state": "open",
      "complexity": "L",
      "dependencies": [206]
    }
  ],
  "total_subissues": 2,
  "open_subissues": 2,
  "dependency_tiers": [[206], [207]]
}
```

## CRITICAL

- Only include subissues in open state in the output
- Dependencies MUST be integer issue numbers, not strings
- Every dependency must reference a subissue that exists in the `subissues` array
- If no scope comment is found, write a JSON output with `"error": "no_scope_comment"`, `"subissues": []`, and `"total_subissues": 0` — do NOT guess or fabricate subissues. This will fail contract validation, which is the correct behavior.
- If the scope comment is malformed (missing dependencies, partial subissue list), parse what is available and include `"partial": true` in the output
