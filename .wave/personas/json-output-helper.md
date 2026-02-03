# JSON Output Helper Guide

This guide helps personas produce valid, deterministic JSON output that passes Wave pipeline validation.

## Quick Reference

### Common JSON Errors and Fixes

| Problem | Example | Fix |
|---------|---------|-----|
| Trailing commas | `{"key": "value",}` | Remove the comma: `{"key": "value"}` |
| Single quotes | `{'key': 'value'}` | Use double quotes: `{"key": "value"}` |
| Unquoted keys | `{key: "value"}` | Quote the keys: `{"key": "value"}` |
| Newlines in strings | Actual newline in string | Escape as `\n`: `"Line 1\nLine 2"` |
| Comments | `{"key": "value"} // comment` | Remove: `{"key": "value"}` |

## JSON Structure Checklist

Before finalizing your JSON output, verify:

- [x] **Syntax Valid**: No trailing commas, all brackets matched
- [x] **Proper Quoting**: All keys and string values use `"` (double quotes)
- [x] **Escaped Specials**: Quotes inside strings are `\"`, newlines are `\n`
- [x] **No Comments**: Remove all `//`, `/* */`, and `#` comments
- [x] **Number Types**: Numbers are unquoted, booleans are `true`/`false` (lowercase)
- [x] **Null Handling**: Use `null` (lowercase) not `undefined` or empty string
- [x] **Array Format**: `[item1, item2]` with commas between items
- [x] **Object Format**: `{"key": "value", "key2": "value2"}` with commas between pairs

## Validation Tools

### Test Your JSON

**Using jq (command line)**:
```bash
echo '{"key":"value"}' | jq .
```

If it outputs without error, your JSON is valid.

**Using jsonlint (online)**:
Visit https://jsonlint.com and paste your JSON to validate.

**Using Python**:
```bash
python3 -c "import json; json.load(open('artifact.json'))"
```

### Multi-line String Handling

When including descriptions or long text:

**Wrong** (actual newlines break JSON):
```json
{
  "description": "Line 1
Line 2
Line 3"
}
```

**Correct** (escape newlines):
```json
{
  "description": "Line 1\nLine 2\nLine 3"
}
```

The Wave pipeline will properly interpret `\n` as newlines when processing the JSON.

## Step-by-Step JSON Output Creation

### 1. Plan Your Structure
```
Object root
├── string fields
├── number fields
├── array fields
│   └── array items (objects or primitives)
└── nested object fields
```

### 2. Write the JSON Skeleton
```json
{
  "field1": "value",
  "field2": 0,
  "field3": [],
  "field4": {}
}
```

### 3. Fill in Values
Replace placeholders with actual content:
- Use `\"` for quotes inside strings
- Use `\n` for line breaks
- Don't add comments or trailing commas

### 4. Verify Structure
- Count opening `{` and `[` equals closing `}` and `]`
- Every item except the last in arrays/objects has a comma
- Last item has no comma

### 5. Validate
```bash
jq . artifact.json  # or equivalent validation command
```

## Real-World Examples

### GitHub Issue Analysis

```json
{
  "repository": {
    "owner": "owner-name",
    "name": "repo-name"
  },
  "total_issues": 10,
  "analyzed_count": 10,
  "poor_quality_issues": [
    {
      "number": 123,
      "title": "Issue Title",
      "body": "Description with\nNewlines properly escaped",
      "quality_score": 45,
      "problems": [
        "Problem 1",
        "Problem 2"
      ],
      "recommendations": [
        "Recommendation 1",
        "Recommendation 2"
      ],
      "labels": ["bug", "needs-info"],
      "url": "https://github.com/owner/repo/issues/123"
    }
  ],
  "quality_threshold": 70,
  "timestamp": "2026-02-03T10:00:00Z"
}
```

### Code Analysis with Multiline Content

```json
{
  "files_analyzed": 5,
  "findings": [
    {
      "file": "src/main.go",
      "line": 42,
      "issue": "Potential nil pointer dereference",
      "context": "Original code:\nif obj != nil {\n  doSomething(obj.field)\n}",
      "recommendation": "Add additional null check for obj.field"
    }
  ],
  "summary": "Found 3 potential issues\n- 2 critical\n- 1 medium severity"
}
```

## Error Recovery

If the Wave pipeline rejects your JSON as malformed:

1. **Get the error message**: Wave will show what's wrong
2. **Common causes**:
   - Trailing comma: Remove `,}` → `}`
   - Unescaped characters: `"` → `\"`, newline → `\n`
   - Comments: Remove `//` and `/* */`
3. **Test locally**: Use jq or jsonlint before submitting
4. **Submit cleaned version**: Fix and re-run the step

## Advanced: Conditional JSON Generation

If you need to conditionally include fields:

**Include field if needed**:
```json
{
  "required_field": "value",
  "optional_field": "value if applicable",
  "conditional_array": []
}
```

**Note**: Keep empty arrays `[]` rather than omitting, as schema validation expects the structure.

## References

- [JSON Specification (RFC 7158)](https://tools.ietf.org/html/rfc7158)
- [jq Manual](https://stedolan.github.io/jq/)
- [Wave Contract Validation Guide](../docs/contract-validation.md)
