# Contract: Tool Target Extraction

## Function Signature
```go
func extractToolTarget(toolName string, input map[string]json.RawMessage) string
```

## Behavioral Contract

### Explicit Mappings (exact match on toolName)
| Tool Name    | Input Field   | Example Output             |
|-------------|---------------|----------------------------|
| Read        | file_path     | "src/main.go"              |
| Write       | file_path     | "output/result.json"       |
| Edit        | file_path     | "internal/adapter/claude.go"|
| NotebookEdit| notebook_path | "analysis.ipynb"           |
| Glob        | pattern       | "**/*.go"                  |
| Grep        | pattern       | "func.*Extract"            |
| Bash        | command       | "go test ./..." (truncated to 60 chars) |
| Task        | description   | "Research streaming patterns" |
| WebFetch    | url           | "https://example.com/api"  |
| WebSearch   | query         | "go ndjson streaming"      |

### Generic Heuristic (unrecognized tool names)
Check input fields in priority order:
1. file_path
2. url
3. pattern
4. command
5. query
6. notebook_path

Return the first non-empty value found. If none found, return empty string.

### Invariants
- MUST NOT panic on nil/empty input
- MUST NOT return error — returns empty string on failure
- Truncation: Bash command targets truncated to 60 characters with "..." suffix
- Unrecognized tools with no matching heuristic fields emit tool name alone (empty target)
