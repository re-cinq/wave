You are parsing the scope output from a previously-run `bb-scope` pipeline to extract the list of subissues and their dependency graph.

Input: {{ input }}

The input format is `workspace/repo number` (e.g. `re-cinq/wave 184`) where the number is the **parent epic** issue.

## Working Directory

You are running in an isolated Wave workspace. Bitbucket API calls via `curl`
work from any directory, so no directory change is needed.

## Instructions

### Step 1: Parse Input

Extract the workspace, repository slug, and epic issue number from the input string. The format is `<WORKSPACE>/<REPO> <NUMBER>`.

### Step 2: Fetch Epic Issue and Comments

Use `curl` with the Bitbucket REST API v2.0 to fetch the epic issue:

```bash
curl -s "https://api.bitbucket.org/2.0/repositories/<WORKSPACE>/<REPO>/issues/<NUMBER>" \
  -H "Authorization: Bearer $BB_TOKEN" | jq .
```

Then fetch its comments:

```bash
curl -s "https://api.bitbucket.org/2.0/repositories/<WORKSPACE>/<REPO>/issues/<NUMBER>/comments?pagelen=100" \
  -H "Authorization: Bearer $BB_TOKEN" | jq .
```

### Step 3: Find Scope Summary Comment

Search through the issue comments for the scope summary posted by `bb-scope`. The scope comment contains a structured list of subissues with:
- Issue numbers and titles
- Dependency relationships (e.g., "depends on #206")
- Complexity ratings (S/M/L/XL)

Look for a comment containing a section like "## Scope Summary" or a table/list of created subissues with issue numbers, URLs, and dependency information.

### Step 4: Fetch Each Subissue

For each subissue referenced in the scope comment:

```bash
curl -s "https://api.bitbucket.org/2.0/repositories/<WORKSPACE>/<REPO>/issues/<SUBISSUE_NUMBER>" \
  -H "Authorization: Bearer $BB_TOKEN" | jq '{id: .id, title: .title, state: .state, content: .content.raw, links: .links}'
```

Verify each subissue:
- Exists and is accessible
- Record its current state (`new`, `open`, `resolved`, `closed`, etc.)
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
    "url": "https://bitbucket.org/re-cinq/wave/issues/184"
  },
  "subissues": [
    {
      "number": 206,
      "repository": "re-cinq/wave",
      "title": "Subissue title",
      "url": "https://bitbucket.org/re-cinq/wave/issues/206",
      "state": "open",
      "complexity": "M",
      "dependencies": []
    },
    {
      "number": 207,
      "repository": "re-cinq/wave",
      "title": "Another subissue",
      "url": "https://bitbucket.org/re-cinq/wave/issues/207",
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

- Only include subissues in `new` or `open` state in the output
- Dependencies MUST be integer issue numbers, not strings
- Every dependency must reference a subissue that exists in the `subissues` array
- If no scope comment is found, write a JSON output with `"error": "no_scope_comment"`, `"subissues": []`, and `"total_subissues": 0` — do NOT guess or fabricate subissues. This will fail contract validation, which is the correct behavior.
- If the scope comment is malformed (missing dependencies, partial subissue list), parse what is available and include `"partial": true` in the output
